// Copyright 2017 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

// This file contains the implementation for interacting with the Trezor hardware
// wallets. The wire protocol spec can be found on the SatoshiLabs website:
// https://doc.satoshilabs.com/trezor-tech/api-protobuf.html

package usbwallet

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"math"
	"math/big"
	"reflect"

	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/accounts/usbwallet/trezor"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/signer/core/apitypes"
	pin "github.com/reserve-protocol/trezor"
	"google.golang.org/protobuf/proto"
)

// errTrezorReplyInvalidHeader is the error message returned by a Trezor data exchange
// if the device replies with a mismatching header. This usually means the device
// is in browser mode.
var errTrezorReplyInvalidHeader = errors.New("trezor: invalid reply header")

type TrezorFailure struct {
	*trezor.Failure
}

// Error implements the error interface for the TrezorFailure type, returning
// a formatted error message containing the failure reason.
func (f *TrezorFailure) Error() string {
	return fmt.Sprintf("trezor: %s", f.GetMessage())
}

// trezorDriver implements the communication with a Trezor hardware wallet.
type trezorDriver struct {
	device     io.ReadWriter // USB device connection to communicate through
	version    [3]uint32     // Current version of the Trezor firmware
	label      string        // Current textual label of the Trezor device
	passphrase string
	failure    error      // Any failure that would make the device unusable
	log        log.Logger // Contextual logger to tag the trezor with its id
}

// newTrezorDriver creates a new instance of a Trezor USB protocol driver.
func newTrezorDriver(logger log.Logger) driver {
	return &trezorDriver{
		log: logger,
	}
}

// Status implements accounts.Wallet, always whether the Trezor is opened, closed
// or whether the Ethereum app was not started on it.
func (w *trezorDriver) Status() (string, error) {
	if w.failure != nil {
		return fmt.Sprintf("Failed: %v", w.failure), w.failure
	}
	if w.device == nil {
		return "Closed", w.failure
	}
	return fmt.Sprintf("Trezor v%d.%d.%d '%s' online", w.version[0], w.version[1], w.version[2], w.label), w.failure
}

// Open implements usbwallet.driver, attempting to initialize the connection to
// the Trezor hardware wallet.
func (w *trezorDriver) Open(device io.ReadWriter, passphrase string) error {
	w.device, w.passphrase, w.failure = device, passphrase, nil

	if _, err := w.trezorExchange(&trezor.EndSession{}, new(trezor.Success)); err != nil {
		return err
	}

	features := new(trezor.Features)
	if _, err := w.trezorExchange(&trezor.Initialize{}, features); err != nil {
		return err
	}
	w.version = [3]uint32{features.GetMajorVersion(), features.GetMinorVersion(), features.GetPatchVersion()}
	w.label = features.GetLabel()

	return w.Heartbeat()
}

// Close implements usbwallet.driver, cleaning up and metadata maintained within
// the Trezor driver.
func (w *trezorDriver) Close() error {
	w.version, w.label = [3]uint32{}, ""
	return nil
}

// Heartbeat implements usbwallet.driver, performing a sanity check against the
// Trezor to see if it's still online.
func (w *trezorDriver) Heartbeat() error {
	if _, err := w.trezorExchange(&trezor.Ping{}, new(trezor.Success)); err != nil {
		w.failure = err
		return err
	}
	return nil
}

// Derive implements usbwallet.driver, sending a derivation request to the Trezor
// and returning the Ethereum address located on that derivation path.
func (w *trezorDriver) Derive(path accounts.DerivationPath) (common.Address, error) {
	return w.trezorDerive(path)
}

// SignTx implements usbwallet.driver, sending the transaction to the Trezor and
// waiting for the user to confirm or deny the transaction.
func (w *trezorDriver) SignTx(path accounts.DerivationPath, tx *types.Transaction, chainID *big.Int) (common.Address, *types.Transaction, error) {
	if w.device == nil {
		return common.Address{}, nil, accounts.ErrWalletClosed
	}
	return w.trezorSign(path, tx, chainID)
}

func (w *trezorDriver) SignTypedHash(path accounts.DerivationPath, domainHash []byte, messageHash []byte) ([]byte, error) {
	if w.device == nil {
		return nil, accounts.ErrWalletClosed
	}
	response := new(trezor.EthereumTypedDataSignature)
	_, err := w.trezorExchange(&trezor.EthereumSignTypedHash{
		AddressN:            path,
		DomainSeparatorHash: domainHash,
		MessageHash:         messageHash,
	}, response)
	if err != nil {
		return nil, err
	}
	return response.Signature, nil
}

func (w *trezorDriver) SignText(path accounts.DerivationPath, text []byte) ([]byte, error) {
	if w.device == nil {
		return nil, accounts.ErrWalletClosed
	}
	response := new(trezor.EthereumMessageSignature)
	_, err := w.trezorExchange(&trezor.EthereumSignMessage{
		AddressN: path,
		Message:  text,
	}, response)
	if err != nil {
		return nil, err
	}
	return response.Signature, nil
}

func (w *trezorDriver) SignedTypedData(path accounts.DerivationPath, data apitypes.TypedData) ([]byte, error) {
	if w.device == nil {
		return nil, accounts.ErrWalletClosed
	}

	_, hashes, err := apitypes.TypedDataAndHash(data)
	if err != nil {
		return nil, fmt.Errorf("trezor: error hashing typed data: %w", err)
	}
	domainHash, messageHash := hashes[2:34], hashes[34:66]
	if w.version[0] == 1 {
		// legacy Trezor devices don't support typed data; fallback to hash signing:
		return w.SignTypedHash(path, []byte(domainHash), []byte(messageHash))
	}

	if w.version[0] == 2 && (w.version[1] < 9 || (w.version[1] == 9 && w.version[2] == 0)) {
		// ShowMessageHash was introduced in Trezor firmware v2.9.1
		return nil, fmt.Errorf("trezor: typed data signing requires firmware v2.9.1 or newer")
	}

	signature := new(trezor.EthereumTypedDataSignature)
	structRequest := new(trezor.EthereumTypedDataStructRequest)
	valueRequest := new(trezor.EthereumTypedDataValueRequest)
	var req proto.Message = &trezor.EthereumSignTypedData{
		AddressN:        path,
		PrimaryType:     &data.PrimaryType,
		ShowMessageHash: []byte(messageHash),
	}
	nestedArray := false
	for {
		n, err := w.trezorExchange(req, signature, structRequest, valueRequest)
		if err != nil {
			var trezorFailure *TrezorFailure
			if nestedArray && errors.As(err, &trezorFailure) &&
				trezorFailure.Code != nil && *trezorFailure.Code == trezor.Failure_Failure_FirmwareError {
				return nil, fmt.Errorf("trezor: nested arrays are not supported by this firmware version: %w", err)
			}
			return nil, err
		}
		nestedArray = false
		switch n {
		case 0:
			// No additional data needed, return the signature
			return signature.Signature, nil
		case 1:
			fields := data.Types[structRequest.GetName()]
			if len(fields) == 0 {
				return nil, fmt.Errorf("trezor: no fields for struct %s", structRequest.GetName())
			}
			ack := &trezor.EthereumTypedDataStructAck{
				Members: make([]*trezor.EthereumTypedDataStructAck_EthereumStructMember, len(fields)),
			}
			for i, field := range fields {
				dt, name, byteLength, arrays, err := parseType(data, field)
				if err != nil {
					return nil, err
				}
				ubyteLength := uint32(byteLength)
				t := &trezor.EthereumTypedDataStructAck_EthereumFieldType{}
				inner := t
				for i := len(arrays) - 1; i >= 0; i-- {
					dataType := trezor.EthereumTypedDataStructAck_ARRAY
					inner.DataType = &dataType
					if arrays[i] != nil {
						length := uint32(*arrays[i])
						inner.Size = &length
					}
					inner.EntryType = &trezor.EthereumTypedDataStructAck_EthereumFieldType{}
					inner = inner.EntryType
				}
				var dataType trezor.EthereumTypedDataStructAck_EthereumDataType
				switch dt {
				case CustomType:
					inner.StructName = &name
					dataType = trezor.EthereumTypedDataStructAck_STRUCT
					members := uint32(len(data.Types[name]))
					inner.Size = &members
				case IntType:
					dataType = trezor.EthereumTypedDataStructAck_INT
					inner.Size = &ubyteLength
				case UintType:
					dataType = trezor.EthereumTypedDataStructAck_UINT
					inner.Size = &ubyteLength
				case AddressType:
					dataType = trezor.EthereumTypedDataStructAck_ADDRESS
				case BoolType:
					dataType = trezor.EthereumTypedDataStructAck_BOOL
				case StringType:
					dataType = trezor.EthereumTypedDataStructAck_STRING
				case FixedBytesType:
					dataType = trezor.EthereumTypedDataStructAck_BYTES
					inner.Size = &ubyteLength
				case BytesType:
					dataType = trezor.EthereumTypedDataStructAck_BYTES
				}
				inner.DataType = &dataType
				ack.Members[i] = &trezor.EthereumTypedDataStructAck_EthereumStructMember{
					Name: &field.Name,
					Type: t,
				}
			}
			req = ack
		case 2:
			structType := data.Types[data.PrimaryType]
			structValue := data.Message
			if valueRequest.MemberPath[0] == 0 {
				// populate with domain info
				structType = data.Types["EIP712Domain"]
				structValue = data.Domain.Map()
			}
			var value []byte
			for i := 1; i < len(valueRequest.MemberPath); i++ {
				p := valueRequest.MemberPath[i]
				if structType == nil {
					return nil, fmt.Errorf("trezor: no struct type for path %v", path)
				}
				if int(p) >= len(structType) {
					return nil, fmt.Errorf("trezor: invalid field index %d for struct %s", p, structRequest.GetName())
				}
				field := structType[p]
				nextValue := structValue[field.Name]
				dt, name, byteLength, arrays, err := parseType(data, field)
				if err != nil {
					return nil, err
				}
				if len(arrays) > 1 {
					nestedArray = true
				}
				for j := 0; j < len(arrays) && i < len(valueRequest.MemberPath)-1; i, j = i+1, j+1 {
					k := reflect.TypeOf(nextValue).Kind()
					if !(k == reflect.Array || k == reflect.Slice) {
						return nil, fmt.Errorf("trezor: expected array at path %v, got %T", valueRequest.MemberPath[:i+1], nextValue)
					}
					a := reflect.ValueOf(nextValue)
					p = valueRequest.MemberPath[i+1]
					if int(p) >= a.Len() {
						return nil, fmt.Errorf("trezor: invalid array index %d for path %v", p, valueRequest.MemberPath[:i+1])
					}
					nextValue = a.Index(int(p)).Interface()
				}
				k := reflect.TypeOf(nextValue).Kind()
				if i < len(valueRequest.MemberPath)-1 {
					if reflect.TypeOf(nextValue).Kind() != reflect.Map {
						return nil, fmt.Errorf("trezor: expected map at path %v, got %T", valueRequest.MemberPath[:i+1], nextValue)
					}
					structType = data.Types[name]
					structValue = nextValue.(apitypes.TypedDataMessage)
				} else if k == reflect.Array || k == reflect.Slice {
					// Array value, return length as uint16
					value = binary.BigEndian.AppendUint16([]byte{}, uint16(reflect.ValueOf(nextValue).Len()))
				} else {
					// Last value, encode it as a primitive value
					switch dt {
					case CustomType:
						return nil, fmt.Errorf("trezor: cannot encode custom type %s at path %v", name, valueRequest.MemberPath[:i+1])
					case IntType, UintType, AddressType, FixedBytesType:
						if str, ok := nextValue.(string); ok {
							value = common.FromHex(str)
						} else if f, ok := nextValue.(float64); ok {
							value = new(big.Int).SetInt64(int64(f)).Bytes()
						}
						if len(value) > byteLength {
							return nil, fmt.Errorf("trezor: value at path %v is too long (%d bytes, expected %d)", valueRequest.MemberPath[:i+1], len(value), byteLength)
						}
						for len(value) < byteLength {
							value = append([]byte{0}, value...)
						}
					case BoolType:
						if b, ok := nextValue.(bool); ok {
							if b {
								value = []byte{1}
							} else {
								value = []byte{0}
							}
						} else {
							return nil, fmt.Errorf("trezor: expected bool at path %v, got %T", valueRequest.MemberPath[:i+1], nextValue)
						}
					case StringType:
						if str, ok := nextValue.(string); ok {
							value = []byte(str)
						} else {
							return nil, fmt.Errorf("trezor: expected string at path %v, got %T", valueRequest.MemberPath[:i+1], nextValue)
						}
					case BytesType:
						if str, ok := nextValue.(string); ok {
							value = common.FromHex(str)
						} else {
							return nil, fmt.Errorf("trezor: expected bytes at path %v, got %T", valueRequest.MemberPath[:i+1], nextValue)
						}
					}
				}
			}
			req = &trezor.EthereumTypedDataValueAck{
				Value: value,
			}
		default:
			return nil, fmt.Errorf("trezor: unexpected reply index %d", n)
		}
	}
}

// trezorDerive sends a derivation request to the Trezor device and returns the
// Ethereum address located on that path.
func (w *trezorDriver) trezorDerive(derivationPath []uint32) (common.Address, error) {
	address := new(trezor.EthereumAddress)
	if _, err := w.trezorExchange(&trezor.EthereumGetAddress{AddressN: derivationPath}, address); err != nil {
		return common.Address{}, err
	}
	if addr := address.GetAddress(); len(addr) > 0 {
		return common.HexToAddress(addr), nil
	}
	return common.Address{}, errors.New("missing derived address")
}

// trezorSign sends the transaction to the Trezor wallet, and waits for the user
// to confirm or deny the transaction.
func (w *trezorDriver) trezorSign(derivationPath []uint32, tx *types.Transaction, chainID *big.Int) (common.Address, *types.Transaction, error) {
	// Create the transaction initiation message
	data := tx.Data()
	length := uint32(len(data))

	request := &trezor.EthereumSignTx{
		AddressN:   derivationPath,
		Nonce:      new(big.Int).SetUint64(tx.Nonce()).Bytes(),
		GasPrice:   tx.GasPrice().Bytes(),
		GasLimit:   new(big.Int).SetUint64(tx.Gas()).Bytes(),
		Value:      tx.Value().Bytes(),
		DataLength: &length,
	}
	if to := tx.To(); to != nil {
		// Non contract deploy, set recipient explicitly
		hex := to.Hex()
		request.To = &hex
	}
	if length > 1024 { // Send the data chunked if that was requested
		request.DataInitialChunk, data = data[:1024], data[1024:]
	} else {
		request.DataInitialChunk, data = data, nil
	}
	if chainID != nil { // EIP-155 transaction, set chain ID explicitly (only 32 bit is supported!?)
		id := chainID.Uint64()
		request.ChainId = &id
	}
	// Send the initiation message and stream content until a signature is returned
	response := new(trezor.EthereumTxRequest)
	if _, err := w.trezorExchange(request, response); err != nil {
		return common.Address{}, nil, err
	}
	for response.DataLength != nil && int(*response.DataLength) <= len(data) {
		chunk := data[:*response.DataLength]
		data = data[*response.DataLength:]

		if _, err := w.trezorExchange(&trezor.EthereumTxAck{DataChunk: chunk}, response); err != nil {
			return common.Address{}, nil, err
		}
	}
	// Extract the Ethereum signature and do a sanity validation
	if len(response.GetSignatureR()) == 0 || len(response.GetSignatureS()) == 0 || response.GetSignatureV() == 0 {
		return common.Address{}, nil, errors.New("reply lacks signature")
	}
	signature := append(append(response.GetSignatureR(), response.GetSignatureS()...), byte(response.GetSignatureV()))

	// Create the correct signer and signature transform based on the chain ID
	var signer types.Signer
	if chainID == nil {
		signer = new(types.HomesteadSigner)
	} else {
		// Trezor backend does not support typed transactions yet.
		signer = types.NewEIP155Signer(chainID)
		// if chainId is above (MaxUint32 - 36) / 2 then the final v values is returned
		// directly. Otherwise, the returned value is 35 + chainid * 2.
		if signature[64] > 1 && int(chainID.Int64()) <= (math.MaxUint32-36)/2 {
			signature[64] -= byte(chainID.Uint64()*2 + 35)
		}
	}

	// Inject the final signature into the transaction and sanity check the sender
	signed, err := tx.WithSignature(signer, signature)
	if err != nil {
		return common.Address{}, nil, err
	}
	sender, err := types.Sender(signer, signed)
	if err != nil {
		return common.Address{}, nil, err
	}
	return sender, signed, nil
}

// trezorExchange performs a data exchange with the Trezor wallet, sending it a
// message and retrieving the response. If multiple responses are possible, the
// method will also return the index of the destination object used.
func (w *trezorDriver) trezorExchange(req proto.Message, results ...proto.Message) (int, error) {
	// Construct the original message payload to chunk up
	data, err := proto.Marshal(req)
	if err != nil {
		return 0, err
	}
	payload := make([]byte, 8+len(data))
	copy(payload, []byte{0x23, 0x23})
	binary.BigEndian.PutUint16(payload[2:], trezor.Type(req))
	binary.BigEndian.PutUint32(payload[4:], uint32(len(data)))
	copy(payload[8:], data)

	// Stream all the chunks to the device
	chunk := make([]byte, 64)
	chunk[0] = 0x3f // Report ID magic number

	for len(payload) > 0 {
		// Construct the new message to stream, padding with zeroes if needed
		if len(payload) > 63 {
			copy(chunk[1:], payload[:63])
			payload = payload[63:]
		} else {
			copy(chunk[1:], payload)
			copy(chunk[1+len(payload):], make([]byte, 63-len(payload)))
			payload = nil
		}
		// Send over to the device
		w.log.Trace("Data chunk sent to the Trezor", "chunk", hexutil.Bytes(chunk))
		if _, err := w.device.Write(chunk); err != nil {
			return 0, err
		}
	}
	// Stream the reply back from the wallet in 64 byte chunks
	var (
		kind  uint16
		reply []byte
	)
	for {
		// Read the next chunk from the Trezor wallet
		if _, err := io.ReadFull(w.device, chunk); err != nil {
			return 0, err
		}
		w.log.Trace("Data chunk received from the Trezor", "chunk", hexutil.Bytes(chunk))

		// Make sure the transport header matches
		if chunk[0] != 0x3f || (len(reply) == 0 && (chunk[1] != 0x23 || chunk[2] != 0x23)) {
			return 0, errTrezorReplyInvalidHeader
		}
		// If it's the first chunk, retrieve the reply message type and total message length
		var payload []byte

		if len(reply) == 0 {
			kind = binary.BigEndian.Uint16(chunk[3:5])
			reply = make([]byte, 0, int(binary.BigEndian.Uint32(chunk[5:9])))
			payload = chunk[9:]
		} else {
			payload = chunk[1:]
		}
		// Append to the reply and stop when filled up
		if left := cap(reply) - len(reply); left > len(payload) {
			reply = append(reply, payload...)
		} else {
			reply = append(reply, payload[:left]...)
			break
		}
	}
	// Try to parse the reply into the requested reply message
	if kind == uint16(trezor.MessageType_MessageType_Failure) {
		// Trezor returned a failure, extract and return the message
		failure := new(trezor.Failure)
		if err := proto.Unmarshal(reply, failure); err != nil {
			return 0, err
		}
		return 0, &TrezorFailure{Failure: failure}
	}
	if kind == uint16(trezor.MessageType_MessageType_ButtonRequest) {
		// Trezor is waiting for user confirmation, ack and wait for the next message
		return w.trezorExchange(&trezor.ButtonAck{}, results...)
	}
	if kind == uint16(trezor.MessageType_MessageType_PinMatrixRequest) {
		p, err := pin.GetPIN("Please enter your Trezor PIN")
		if err != nil {
			return 0, err
		}
		return w.trezorExchange(&trezor.PinMatrixAck{Pin: &p}, results...)
	}
	if kind == uint16(trezor.MessageType_MessageType_PassphraseRequest) {
		return w.trezorExchange(&trezor.PassphraseAck{Passphrase: &w.passphrase}, results...)
	}
	for i, res := range results {
		if trezor.Type(res) == kind {
			return i, proto.Unmarshal(reply, res)
		}
	}
	expected := make([]string, len(results))
	for i, res := range results {
		expected[i] = trezor.Name(trezor.Type(res))
	}
	return 0, fmt.Errorf("trezor: expected reply types %s, got %s", expected, trezor.Name(kind))
}
