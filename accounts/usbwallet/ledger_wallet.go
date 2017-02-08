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

// This file contains the implementation for interacting with the Ledger hardware
// wallets. The wire protocol spec can be found in the Ledger Blue GitHub repo:
// https://raw.githubusercontent.com/LedgerHQ/blue-app-eth/master/doc/ethapp.asc

// +build !ios

package usbwallet

import (
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"math/big"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/logger"
	"github.com/ethereum/go-ethereum/logger/glog"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/karalabe/gousb/usb"
)

// ledgerDerivationPath is the base derivation parameters used by the wallet.
var ledgerDerivationPath = []uint32{0x80000000 + 44, 0x80000000 + 60, 0x80000000 + 0, 0}

// ledgerOpcode is an enumeration encoding the supported Ledger opcodes.
type ledgerOpcode byte

// ledgerParam1 is an enumeration encoding the supported Ledger parameters for
// specific opcodes. The same parameter values may be reused between opcodes.
type ledgerParam1 byte

// ledgerParam2 is an enumeration encoding the supported Ledger parameters for
// specific opcodes. The same parameter values may be reused between opcodes.
type ledgerParam2 byte

const (
	ledgerOpRetrieveAddress  ledgerOpcode = 0x02 // Returns the public key and Ethereum address for a given BIP 32 path
	ledgerOpSignTransaction  ledgerOpcode = 0x04 // Signs an Ethereum transaction after having the user validate the parameters
	ledgerOpGetConfiguration ledgerOpcode = 0x06 // Returns specific wallet application configuration

	ledgerP1DirectlyFetchAddress    ledgerParam1 = 0x00 // Return address directly from the wallet
	ledgerP1ConfirmFetchAddress     ledgerParam1 = 0x01 // Require a user confirmation before returning the address
	ledgerP1InitTransactionData     ledgerParam1 = 0x00 // First transaction data block for signing
	ledgerP1ContTransactionData     ledgerParam1 = 0x80 // Subsequent transaction data block for signing
	ledgerP2DiscardAddressChainCode ledgerParam2 = 0x00 // Do not return the chain code along with the address
	ledgerP2ReturnAddressChainCode  ledgerParam2 = 0x01 // Require a user confirmation before returning the address
)

// ledgerWallet represents a live USB Ledger hardware wallet.
type ledgerWallet struct {
	context    *usb.Context  // USB context to interface libusb through
	hardwareID deviceID      // USB identifiers to identify this device type
	locationID uint16        // USB bus and address to identify this device instance
	url        *accounts.URL // Textual URL uniquely identifying this wallet

	device  *usb.Device  // USB device advertising itself as a Ledger wallet
	input   usb.Endpoint // Input endpoint to send data to this device
	output  usb.Endpoint // Output endpoint to receive data from this device
	failure error        // Any failure that would make the device unusable

	version  [3]byte                     // Current version of the Ledger Ethereum app (zero if app is offline)
	accounts []accounts.Account          // List of derive accounts pinned on the Ledger
	paths    map[common.Address][]uint32 // Known derivation paths for signing operations

	quit chan chan error
	lock sync.RWMutex
}

// URL implements accounts.Wallet, returning the URL of the Ledger device.
func (w *ledgerWallet) URL() accounts.URL {
	return *w.url
}

// Status implements accounts.Wallet, always whether the Ledger is opened, closed
// or whether the Ethereum app was not started on it.
func (w *ledgerWallet) Status() string {
	w.lock.RLock()
	defer w.lock.RUnlock()

	if w.failure != nil {
		return fmt.Sprintf("Failed: %v", w.failure)
	}
	if w.device == nil {
		return "Closed"
	}
	if w.version == [3]byte{0, 0, 0} {
		return "Ethereum app offline"
	}
	return fmt.Sprintf("Ethereum app v%d.%d.%d online", w.version[0], w.version[1], w.version[2])
}

// Open implements accounts.Wallet, attempting to open a USB connection to the
// Ledger hardware wallet. The Ledger does not require a user passphrase so that
// is silently discarded.
func (w *ledgerWallet) Open(passphrase string) error {
	w.lock.Lock()
	defer w.lock.Unlock()

	// If the wallet was already opened, don't try to open again
	if w.device != nil {
		return accounts.ErrWalletAlreadyOpen
	}
	// Otherwise iterate over all USB devices and find this again (no way to directly do this)
	// Iterate over all attached devices and fetch those seemingly Ledger
	devices, err := w.context.ListDevices(func(desc *usb.Descriptor) bool {
		// Only open this single specific device
		return desc.Vendor == w.hardwareID.Vendor && desc.Product == w.hardwareID.Product &&
			uint16(desc.Bus)<<8+uint16(desc.Address) == w.locationID
	})
	if err != nil {
		return err
	}
	// Device opened, attach to the input and output endpoints
	device := devices[0]

	var invalid string
	switch {
	case len(device.Descriptor.Configs) == 0:
		invalid = "no endpoint config available"
	case len(device.Descriptor.Configs[0].Interfaces) == 0:
		invalid = "no endpoint interface available"
	case len(device.Descriptor.Configs[0].Interfaces[0].Setups) == 0:
		invalid = "no endpoint setup available"
	case len(device.Descriptor.Configs[0].Interfaces[0].Setups[0].Endpoints) < 2:
		invalid = "not enough IO endpoints available"
	}
	if invalid != "" {
		device.Close()
		return fmt.Errorf("ledger wallet [%s] invalid: %s", w.url, invalid)
	}
	// Open the input and output endpoints to the device
	input, err := device.OpenEndpoint(
		device.Descriptor.Configs[0].Config,
		device.Descriptor.Configs[0].Interfaces[0].Number,
		device.Descriptor.Configs[0].Interfaces[0].Setups[0].Number,
		device.Descriptor.Configs[0].Interfaces[0].Setups[0].Endpoints[1].Address,
	)
	if err != nil {
		device.Close()
		return fmt.Errorf("ledger wallet [%s] input open failed: %v", w.url, err)
	}
	output, err := device.OpenEndpoint(
		device.Descriptor.Configs[0].Config,
		device.Descriptor.Configs[0].Interfaces[0].Number,
		device.Descriptor.Configs[0].Interfaces[0].Setups[0].Number,
		device.Descriptor.Configs[0].Interfaces[0].Setups[0].Endpoints[0].Address,
	)
	if err != nil {
		device.Close()
		return fmt.Errorf("ledger wallet [%s] output open failed: %v", w.url, err)
	}
	// Wallet seems to be successfully opened, guess if the Ethereum app is running
	w.device, w.input, w.output = device, input, output

	w.paths = make(map[common.Address][]uint32)
	w.quit = make(chan chan error)
	defer func() {
		go w.heartbeat()
	}()

	if _, err := w.deriveAddress(ledgerDerivationPath); err != nil {
		// Ethereum app is not running, nothing more to do, return
		return nil
	}
	// Try to resolve the Ethereum app's version, will fail prior to v1.0.2
	if w.resolveVersion() != nil {
		w.version = [3]byte{1, 0, 0} // Assume worst case, can't verify if v1.0.0 or v1.0.1
	}
	return nil
}

// heartbeat is a health check loop for the Ledger wallets to periodically verify
// whether they are still present or if they malfunctioned. It is needed because:
//  - libusb on Windows doesn't support hotplug, so we can't detect USB unplugs
//  - communication timeout on the Ledger requires a device power cycle to fix
func (w *ledgerWallet) heartbeat() {
	// Execute heartbeat checks until termination or error
	var (
		errc chan error
		fail error
	)
	for errc == nil && fail == nil {
		// Wait until termination is requested or the heartbeat cycle arrives
		select {
		case errc = <-w.quit:
			// Termination requested
			continue
		case <-time.After(time.Second):
			// Heartbeat time
		}
		// Execute a tiny data exchange to see responsiveness
		w.lock.Lock()
		if err := w.resolveVersion(); err == usb.ERROR_IO || err == usb.ERROR_NO_DEVICE {
			w.failure = err
			fail = err
		}
		w.lock.Unlock()
	}
	// In case of error, wait for termination
	if fail != nil {
		errc = <-w.quit
	}
	errc <- fail
}

// Close implements accounts.Wallet, closing the USB connection to the Ledger.
func (w *ledgerWallet) Close() error {
	// Terminate the health checks
	errc := make(chan error)
	w.quit <- errc
	herr := <-errc // Save for later, we *must* close the USB

	// Terminate the device connection
	w.lock.Lock()
	defer w.lock.Unlock()

	if err := w.device.Close(); err != nil {
		return err
	}
	w.device, w.input, w.output, w.paths, w.quit = nil, nil, nil, nil, nil

	return herr // If all went well, return any health-check errors
}

// Accounts implements accounts.Wallet, returning the list of accounts pinned to
// the Ledger hardware wallet.
func (w *ledgerWallet) Accounts() []accounts.Account {
	w.lock.RLock()
	defer w.lock.RUnlock()

	cpy := make([]accounts.Account, len(w.accounts))
	copy(cpy, w.accounts)
	return cpy
}

// Contains implements accounts.Wallet, returning whether a particular account is
// or is not pinned into this Ledger instance. Although we could attempt to resolve
// unpinned accounts, that would be an non-negligible hardware operation.
func (w *ledgerWallet) Contains(account accounts.Account) bool {
	w.lock.RLock()
	defer w.lock.RUnlock()

	_, exists := w.paths[account.Address]
	return exists
}

// Derive implements accounts.Wallet, deriving a new account at the specific
// derivation path. If pin is set to true, the account will be added to the list
// of tracked accounts.
func (w *ledgerWallet) Derive(path string, pin bool) (accounts.Account, error) {
	w.lock.Lock()
	defer w.lock.Unlock()

	// If the wallet is closed, or the Ethereum app doesn't run, abort
	if w.device == nil || w.version == [3]byte{0, 0, 0} {
		return accounts.Account{}, accounts.ErrWalletClosed
	}
	// All seems fine, convert the user derivation path to Ledger representation
	path = strings.TrimPrefix(path, "/")

	parts := strings.Split(path, "/")
	lpath := make([]uint32, len(parts))
	for i, part := range parts {
		// Handle hardened paths
		if strings.HasSuffix(part, "'") {
			lpath[i] = 0x80000000
			part = strings.TrimSuffix(part, "'")
		}
		// Handle the non hardened component
		val, err := strconv.Atoi(part)
		if err != nil {
			return accounts.Account{}, fmt.Errorf("path element %d: %v", i, err)
		}
		lpath[i] += uint32(val)
	}
	// Try to derive the actual account and update it's URL if succeeful
	address, err := w.deriveAddress(lpath)
	if err != nil {
		return accounts.Account{}, err
	}
	account := accounts.Account{
		Address: address,
		URL:     accounts.URL{Scheme: w.url.Scheme, Path: fmt.Sprintf("%s/%s", w.url.Path, path)},
	}
	// If pinning was requested, track the account
	if pin {
		if _, ok := w.paths[address]; !ok {
			w.accounts = append(w.accounts, account)
			w.paths[address] = lpath
		}
	}
	return account, nil
}

// SignHash implements accounts.Wallet, however signing arbitrary data is not
// supported for Ledger wallets, so this method will always return an error.
func (w *ledgerWallet) SignHash(acc accounts.Account, hash []byte) ([]byte, error) {
	return nil, accounts.ErrNotSupported
}

// SignTx implements accounts.Wallet. It sends the transaction over to the Ledger
// wallet to request a confirmation from the user. It returns either the signed
// transaction or a failure if the user denied the transaction.
//
// Note, if the version of the Ethereum application running on the Ledger wallet is
// too old to sign EIP-155 transactions, but such is requested nonetheless, an error
// will be returned opposed to silently signing in Homestead mode.
func (w *ledgerWallet) SignTx(account accounts.Account, tx *types.Transaction, chainID *big.Int) (*types.Transaction, error) {
	w.lock.Lock()
	defer w.lock.Unlock()

	// Make sure the requested account is contained within
	path, ok := w.paths[account.Address]
	if !ok {
		return nil, accounts.ErrUnknownAccount
	}
	// Ensure the wallet is capable of signing the given transaction
	if chainID != nil && w.version[0] <= 1 && w.version[1] <= 0 && w.version[2] <= 2 {
		return nil, fmt.Errorf("Ledger v%d.%d.%d doesn't support signing this transaction, please update to v1.0.3 at least",
			w.version[0], w.version[1], w.version[2])
	}
	return w.sign(path, account.Address, tx, chainID)
}

// SignHashWithPassphrase implements accounts.Wallet, however signing arbitrary
// data is not supported for Ledger wallets, so this method will always return
// an error.
func (w *ledgerWallet) SignHashWithPassphrase(account accounts.Account, passphrase string, hash []byte) ([]byte, error) {
	return nil, accounts.ErrNotSupported
}

// SignTxWithPassphrase implements accounts.Wallet, attempting to sign the given
// transaction with the given account using passphrase as extra authentication.
// Since the Ledger does not support extra passphrases, it is silently ignored.
func (w *ledgerWallet) SignTxWithPassphrase(account accounts.Account, passphrase string, tx *types.Transaction, chainID *big.Int) (*types.Transaction, error) {
	return w.SignTx(account, tx, chainID)
}

// resolveVersion retrieves the current version of the Ethereum wallet app running
// on the Ledger wallet and caches it for future reference.
//
// The version retrieval protocol is defined as follows:
//
//   CLA | INS | P1 | P2 | Lc | Le
//   ----+-----+----+----+----+---
//    E0 | 06  | 00 | 00 | 00 | 04
//
// With no input data, and the output data being:
//
//   Description                                        | Length
//   ---------------------------------------------------+--------
//   Flags 01: arbitrary data signature enabled by user | 1 byte
//   Application major version                          | 1 byte
//   Application minor version                          | 1 byte
//   Application patch version                          | 1 byte
func (wallet *ledgerWallet) resolveVersion() error {
	// Send the request and wait for the response
	reply, err := wallet.exchange(ledgerOpGetConfiguration, 0, 0, nil)
	if err != nil {
		return err
	}
	if len(reply) != 4 {
		return errors.New("reply not of correct size")
	}
	// Cache the version for future reference
	copy(wallet.version[:], reply[1:])
	return nil
}

// deriveAddress retrieves the currently active Ethereum address from a Ledger
// wallet at the specified derivation path.
//
// The address derivation protocol is defined as follows:
//
//   CLA | INS | P1 | P2 | Lc  | Le
//   ----+-----+----+----+-----+---
//    E0 | 02  | 00 return address
//               01 display address and confirm before returning
//                  | 00: do not return the chain code
//                  | 01: return the chain code
//                       | var | 00
//
// Where the input data is:
//
//   Description                                      | Length
//   -------------------------------------------------+--------
//   Number of BIP 32 derivations to perform (max 10) | 1 byte
//   First derivation index (big endian)              | 4 bytes
//   ...                                              | 4 bytes
//   Last derivation index (big endian)               | 4 bytes
//
// And the output data is:
//
//   Description             | Length
//   ------------------------+-------------------
//   Public Key length       | 1 byte
//   Uncompressed Public Key | arbitrary
//   Ethereum address length | 1 byte
//   Ethereum address        | 40 bytes hex ascii
//   Chain code if requested | 32 bytes
func (w *ledgerWallet) deriveAddress(derivationPath []uint32) (common.Address, error) {
	// Flatten the derivation path into the Ledger request
	path := make([]byte, 1+4*len(derivationPath))
	path[0] = byte(len(derivationPath))
	for i, component := range derivationPath {
		binary.BigEndian.PutUint32(path[1+4*i:], component)
	}
	// Send the request and wait for the response
	reply, err := w.exchange(ledgerOpRetrieveAddress, ledgerP1DirectlyFetchAddress, ledgerP2DiscardAddressChainCode, path)
	if err != nil {
		return common.Address{}, err
	}
	// Discard the public key, we don't need that for now
	if len(reply) < 1 || len(reply) < 1+int(reply[0]) {
		return common.Address{}, errors.New("reply lacks public key entry")
	}
	reply = reply[1+int(reply[0]):]

	// Extract the Ethereum hex address string
	if len(reply) < 1 || len(reply) < 1+int(reply[0]) {
		return common.Address{}, errors.New("reply lacks address entry")
	}
	hexstr := reply[1 : 1+int(reply[0])]

	// Decode the hex sting into an Ethereum address and return
	var address common.Address
	hex.Decode(address[:], hexstr)
	return address, nil
}

// sign sends the transaction to the Ledger wallet, and waits for the user to
// confirm or deny the transaction.
//
// The transaction signing protocol is defined as follows:
//
//   CLA | INS | P1 | P2 | Lc  | Le
//   ----+-----+----+----+-----+---
//    E0 | 04  | 00: first transaction data block
//               80: subsequent transaction data block
//                  | 00 | variable | variable
//
// Where the input for the first transaction block (first 255 bytes) is:
//
//   Description                                      | Length
//   -------------------------------------------------+----------
//   Number of BIP 32 derivations to perform (max 10) | 1 byte
//   First derivation index (big endian)              | 4 bytes
//   ...                                              | 4 bytes
//   Last derivation index (big endian)               | 4 bytes
//   RLP transaction chunk                            | arbitrary
//
// And the input for subsequent transaction blocks (first 255 bytes) are:
//
//   Description           | Length
//   ----------------------+----------
//   RLP transaction chunk | arbitrary
//
// And the output data is:
//
//   Description | Length
//   ------------+---------
//   signature V | 1 byte
//   signature R | 32 bytes
//   signature S | 32 bytes
func (w *ledgerWallet) sign(derivationPath []uint32, address common.Address, tx *types.Transaction, chainID *big.Int) (*types.Transaction, error) {
	// We need to modify the timeouts to account for user feedback
	defer func(old time.Duration) { w.device.ReadTimeout = old }(w.device.ReadTimeout)
	w.device.ReadTimeout = time.Minute

	// Flatten the derivation path into the Ledger request
	path := make([]byte, 1+4*len(derivationPath))
	path[0] = byte(len(derivationPath))
	for i, component := range derivationPath {
		binary.BigEndian.PutUint32(path[1+4*i:], component)
	}
	// Create the transaction RLP based on whether legacy or EIP155 signing was requeste
	var (
		txrlp []byte
		err   error
	)
	if chainID == nil {
		if txrlp, err = rlp.EncodeToBytes([]interface{}{tx.Nonce(), tx.GasPrice(), tx.Gas(), tx.To(), tx.Value(), tx.Data()}); err != nil {
			return nil, err
		}
	} else {
		if txrlp, err = rlp.EncodeToBytes([]interface{}{tx.Nonce(), tx.GasPrice(), tx.Gas(), tx.To(), tx.Value(), tx.Data(), chainID, big.NewInt(0), big.NewInt(0)}); err != nil {
			return nil, err
		}
	}
	payload := append(path, txrlp...)

	// Send the request and wait for the response
	var (
		op    = ledgerP1InitTransactionData
		reply []byte
	)
	for len(payload) > 0 {
		// Calculate the size of the next data chunk
		chunk := 255
		if chunk > len(payload) {
			chunk = len(payload)
		}
		// Send the chunk over, ensuring it's processed correctly
		reply, err = w.exchange(ledgerOpSignTransaction, op, 0, payload[:chunk])
		if err != nil {
			return nil, err
		}
		// Shift the payload and ensure subsequent chunks are marked as such
		payload = payload[chunk:]
		op = ledgerP1ContTransactionData
	}
	// Extract the Ethereum signature and do a sanity validation
	if len(reply) != 65 {
		return nil, errors.New("reply lacks signature")
	}
	signature := append(reply[1:], reply[0])

	// Create the correct signer and signature transform based on the chain ID
	var signer types.Signer
	if chainID == nil {
		signer = new(types.HomesteadSigner)
	} else {
		signer = types.NewEIP155Signer(chainID)
		signature[64] = (signature[64]-34)/2 - byte(chainID.Uint64())
	}
	// Inject the final signature into the transaction and sanity check the sender
	signed, err := tx.WithSignature(signer, signature)
	if err != nil {
		return nil, err
	}
	sender, err := types.Sender(signer, signed)
	if err != nil {
		return nil, err
	}
	if sender != address {
		return nil, fmt.Errorf("signer mismatch: expected %s, got %s", address.Hex(), sender.Hex())
	}
	return signed, nil
}

// exchange performs a data exchange with the Ledger wallet, sending it a message
// and retrieving the response.
//
// The common transport header is defined as follows:
//
//  Description                           | Length
//  --------------------------------------+----------
//  Communication channel ID (big endian) | 2 bytes
//  Command tag                           | 1 byte
//  Packet sequence index (big endian)    | 2 bytes
//  Payload                               | arbitrary
//
// The Communication channel ID allows commands multiplexing over the same
// physical link. It is not used for the time being, and should be set to 0101
// to avoid compatibility issues with implementations ignoring a leading 00 byte.
//
// The Command tag describes the message content. Use TAG_APDU (0x05) for standard
// APDU payloads, or TAG_PING (0x02) for a simple link test.
//
// The Packet sequence index describes the current sequence for fragmented payloads.
// The first fragment index is 0x00.
//
// APDU Command payloads are encoded as follows:
//
//  Description              | Length
//  -----------------------------------
//  APDU length (big endian) | 2 bytes
//  APDU CLA                 | 1 byte
//  APDU INS                 | 1 byte
//  APDU P1                  | 1 byte
//  APDU P2                  | 1 byte
//  APDU length              | 1 byte
//  Optional APDU data       | arbitrary
func (w *ledgerWallet) exchange(opcode ledgerOpcode, p1 ledgerParam1, p2 ledgerParam2, data []byte) ([]byte, error) {
	// Construct the message payload, possibly split into multiple chunks
	var chunks [][]byte
	for left := data; len(left) > 0 || len(chunks) == 0; {
		// Create the chunk header
		var chunk []byte

		if len(chunks) == 0 {
			// The first chunk encodes the length and all the opcodes
			chunk = []byte{0x00, 0x00, 0xe0, byte(opcode), byte(p1), byte(p2), byte(len(data))}
			binary.BigEndian.PutUint16(chunk, uint16(5+len(data)))
		}
		// Append the data blob to the end of the chunk
		space := 64 - len(chunk) - 5 // 5 == header size
		if len(left) > space {
			chunks, left = append(chunks, append(chunk, left[:space]...)), left[space:]
			continue
		}
		chunks, left = append(chunks, append(chunk, left...)), nil
	}
	// Stream all the chunks to the device
	for i, chunk := range chunks {
		// Construct the new message to stream
		header := []byte{0x01, 0x01, 0x05, 0x00, 0x00} // Channel ID and command tag appended
		binary.BigEndian.PutUint16(header[3:], uint16(i))

		msg := append(header, chunk...)

		// Send over to the device
		if glog.V(logger.Core) {
			glog.Infof("-> %03d.%03d: %x", w.device.Bus, w.device.Address, msg)
		}
		if _, err := w.input.Write(msg); err != nil {
			return nil, err
		}
	}
	// Stream the reply back from the wallet in 64 byte chunks
	var reply []byte
	for {
		// Read the next chunk from the Ledger wallet
		chunk := make([]byte, 64)
		if _, err := io.ReadFull(w.output, chunk); err != nil {
			return nil, err
		}
		if glog.V(logger.Core) {
			glog.Infof("<- %03d.%03d: %x", w.device.Bus, w.device.Address, chunk)
		}
		// Make sure the transport header matches
		if chunk[0] != 0x01 || chunk[1] != 0x01 || chunk[2] != 0x05 {
			return nil, fmt.Errorf("invalid reply header: %x", chunk[:3])
		}
		// If it's the first chunk, retrieve the total message length
		if chunk[3] == 0x00 && chunk[4] == 0x00 {
			reply = make([]byte, 0, int(binary.BigEndian.Uint16(chunk[5:7])))
			chunk = chunk[7:]
		} else {
			chunk = chunk[5:]
		}
		// Append to the reply and stop when filled up
		if left := cap(reply) - len(reply); left > len(chunk) {
			reply = append(reply, chunk...)
		} else {
			reply = append(reply, chunk[:left]...)
			break
		}
	}
	return reply[:len(reply)-2], nil
}
