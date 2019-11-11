// Copyright 2019 The go-ethereum Authors
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

package core

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"math/big"
	"mime"
	"reflect"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"unicode"

	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/common/math"
	"github.com/ethereum/go-ethereum/consensus/clique"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/rlp"
)

type SigFormat struct {
	Mime        string
	ByteVersion byte
}

var (
	IntendedValidator = SigFormat{
		accounts.MimetypeDataWithValidator,
		0x00,
	}
	DataTyped = SigFormat{
		accounts.MimetypeTypedData,
		0x01,
	}
	ApplicationClique = SigFormat{
		accounts.MimetypeClique,
		0x02,
	}
	TextPlain = SigFormat{
		accounts.MimetypeTextPlain,
		0x45,
	}
)

type ValidatorData struct {
	Address common.Address
	Message hexutil.Bytes
}

type TypedData struct {
	Types       Types            `json:"types"`
	PrimaryType string           `json:"primaryType"`
	Domain      TypedDataDomain  `json:"domain"`
	Message     TypedDataMessage `json:"message"`
}

type Type struct {
	Name string `json:"name"`
	Type string `json:"type"`
}

func (t *Type) isArray() bool {
	return strings.HasSuffix(t.Type, "[]")
}

// typeName returns the canonical name of the type. If the type is 'Person[]', then
// this method returns 'Person'
func (t *Type) typeName() string {
	if strings.HasSuffix(t.Type, "[]") {
		return strings.TrimSuffix(t.Type, "[]")
	}
	return t.Type
}

func (t *Type) isReferenceType() bool {
	if len(t.Type) == 0 {
		return false
	}
	// Reference types must have a leading uppercase characer
	return unicode.IsUpper([]rune(t.Type)[0])
}

type Types map[string][]Type

type TypePriority struct {
	Type  string
	Value uint
}

type TypedDataMessage = map[string]interface{}

type TypedDataDomain struct {
	Name              string                `json:"name"`
	Version           string                `json:"version"`
	ChainId           *math.HexOrDecimal256 `json:"chainId"`
	VerifyingContract string                `json:"verifyingContract"`
	Salt              string                `json:"salt"`
}

var typedDataReferenceTypeRegexp = regexp.MustCompile(`^[A-Z](\w*)(\[\])?$`)

// sign receives a request and produces a signature
//
// Note, the produced signature conforms to the secp256k1 curve R, S and V values,
// where the V value will be 27 or 28 for legacy reasons, if legacyV==true.
func (api *SignerAPI) sign(addr common.MixedcaseAddress, req *SignDataRequest, legacyV bool) (hexutil.Bytes, error) {
	// We make the request prior to looking up if we actually have the account, to prevent
	// account-enumeration via the API
	res, err := api.UI.ApproveSignData(req)
	if err != nil {
		return nil, err
	}
	if !res.Approved {
		return nil, ErrRequestDenied
	}
	// Look up the wallet containing the requested signer
	account := accounts.Account{Address: addr.Address()}
	wallet, err := api.am.Find(account)
	if err != nil {
		return nil, err
	}
	pw, err := api.lookupOrQueryPassword(account.Address,
		"Password for signing",
		fmt.Sprintf("Please enter password for signing data with account %s", account.Address.Hex()))
	if err != nil {
		return nil, err
	}
	// Sign the data with the wallet
	signature, err := wallet.SignDataWithPassphrase(account, pw, req.ContentType, req.Rawdata)
	if err != nil {
		return nil, err
	}
	if legacyV {
		signature[64] += 27 // Transform V from 0/1 to 27/28 according to the yellow paper
	}
	return signature, nil
}

// SignData signs the hash of the provided data, but does so differently
// depending on the content-type specified.
//
// Different types of validation occur.
func (api *SignerAPI) SignData(ctx context.Context, contentType string, addr common.MixedcaseAddress, data interface{}) (hexutil.Bytes, error) {
	var req, transformV, err = api.determineSignatureFormat(ctx, contentType, addr, data)
	if err != nil {
		return nil, err
	}
	signature, err := api.sign(addr, req, transformV)
	if err != nil {
		api.UI.ShowError(err.Error())
		return nil, err
	}
	return signature, nil
}

// determineSignatureFormat determines which signature method should be used based upon the mime type
// In the cases where it matters ensure that the charset is handled. The charset
// resides in the 'params' returned as the second returnvalue from mime.ParseMediaType
// charset, ok := params["charset"]
// As it is now, we accept any charset and just treat it as 'raw'.
// This method returns the mimetype for signing along with the request
func (api *SignerAPI) determineSignatureFormat(ctx context.Context, contentType string, addr common.MixedcaseAddress, data interface{}) (*SignDataRequest, bool, error) {
	var (
		req          *SignDataRequest
		useEthereumV = true // Default to use V = 27 or 28, the legacy Ethereum format
	)
	mediaType, _, err := mime.ParseMediaType(contentType)
	if err != nil {
		return nil, useEthereumV, err
	}

	switch mediaType {
	case IntendedValidator.Mime:
		// Data with an intended validator
		validatorData, err := UnmarshalValidatorData(data)
		if err != nil {
			return nil, useEthereumV, err
		}
		sighash, msg := SignTextValidator(validatorData)
		messages := []*NameValueType{
			{
				Name:  "This is a request to sign data intended for a particular validator (see EIP 191 version 0)",
				Typ:   "description",
				Value: "",
			},
			{
				Name:  "Intended validator address",
				Typ:   "address",
				Value: validatorData.Address.String(),
			},
			{
				Name:  "Application-specific data",
				Typ:   "hexdata",
				Value: validatorData.Message,
			},
			{
				Name:  "Full message for signing",
				Typ:   "hexdata",
				Value: fmt.Sprintf("0x%x", msg),
			},
		}
		req = &SignDataRequest{ContentType: mediaType, Rawdata: []byte(msg), Messages: messages, Hash: sighash}
	case ApplicationClique.Mime:
		// Clique is the Ethereum PoA standard
		stringData, ok := data.(string)
		if !ok {
			return nil, useEthereumV, fmt.Errorf("input for %v must be an hex-encoded string", ApplicationClique.Mime)
		}
		cliqueData, err := hexutil.Decode(stringData)
		if err != nil {
			return nil, useEthereumV, err
		}
		header := &types.Header{}
		if err := rlp.DecodeBytes(cliqueData, header); err != nil {
			return nil, useEthereumV, err
		}
		// The incoming clique header is already truncated, sent to us with a extradata already shortened
		if len(header.Extra) < 65 {
			// Need to add it back, to get a suitable length for hashing
			newExtra := make([]byte, len(header.Extra)+65)
			copy(newExtra, header.Extra)
			header.Extra = newExtra
		}
		// Get back the rlp data, encoded by us
		sighash, cliqueRlp, err := cliqueHeaderHashAndRlp(header)
		if err != nil {
			return nil, useEthereumV, err
		}
		messages := []*NameValueType{
			{
				Name:  "Clique header",
				Typ:   "clique",
				Value: fmt.Sprintf("clique header %d [0x%x]", header.Number, header.Hash()),
			},
		}
		// Clique uses V on the form 0 or 1
		useEthereumV = false
		req = &SignDataRequest{ContentType: mediaType, Rawdata: cliqueRlp, Messages: messages, Hash: sighash}
	default: // also case TextPlain.Mime:
		// Calculates an Ethereum ECDSA signature for:
		// hash = keccak256("\x19${byteVersion}Ethereum Signed Message:\n${message length}${message}")
		// We expect it to be a string
		if stringData, ok := data.(string); !ok {
			return nil, useEthereumV, fmt.Errorf("input for text/plain must be an hex-encoded string")
		} else {
			if textData, err := hexutil.Decode(stringData); err != nil {
				return nil, useEthereumV, err
			} else {
				sighash, msg := accounts.TextAndHash(textData)
				messages := []*NameValueType{
					{
						Name:  "message",
						Typ:   accounts.MimetypeTextPlain,
						Value: msg,
					},
				}
				req = &SignDataRequest{ContentType: mediaType, Rawdata: []byte(msg), Messages: messages, Hash: sighash}
			}
		}
	}
	req.Address = addr
	req.Meta = MetadataFromContext(ctx)
	return req, useEthereumV, nil
}

// SignTextWithValidator signs the given message which can be further recovered
// with the given validator.
// hash = keccak256("\x19\x00"${address}${data}).
func SignTextValidator(validatorData ValidatorData) (hexutil.Bytes, string) {
	msg := fmt.Sprintf("\x19\x00%s%s", string(validatorData.Address.Bytes()), string(validatorData.Message))
	return crypto.Keccak256([]byte(msg)), msg
}

// cliqueHeaderHashAndRlp returns the hash which is used as input for the proof-of-authority
// signing. It is the hash of the entire header apart from the 65 byte signature
// contained at the end of the extra data.
//
// The method requires the extra data to be at least 65 bytes -- the original implementation
// in clique.go panics if this is the case, thus it's been reimplemented here to avoid the panic
// and simply return an error instead
func cliqueHeaderHashAndRlp(header *types.Header) (hash, rlp []byte, err error) {
	if len(header.Extra) < 65 {
		err = fmt.Errorf("clique header extradata too short, %d < 65", len(header.Extra))
		return
	}
	rlp = clique.CliqueRLP(header)
	hash = clique.SealHash(header).Bytes()
	return hash, rlp, err
}

// SignTypedData signs EIP-712 conformant typed data
// hash = keccak256("\x19${byteVersion}${domainSeparator}${hashStruct(message)}")
func (api *SignerAPI) SignTypedData(ctx context.Context, addr common.MixedcaseAddress, typedData TypedData) (hexutil.Bytes, error) {
	domainSeparator, err := typedData.HashStruct("EIP712Domain", typedData.Domain.Map())
	if err != nil {
		return nil, err
	}
	typedDataHash, err := typedData.HashStruct(typedData.PrimaryType, typedData.Message)
	if err != nil {
		return nil, err
	}
	rawData := []byte(fmt.Sprintf("\x19\x01%s%s", string(domainSeparator), string(typedDataHash)))
	sighash := crypto.Keccak256(rawData)
	messages, err := typedData.Format()
	if err != nil {
		return nil, err
	}
	req := &SignDataRequest{ContentType: DataTyped.Mime, Rawdata: rawData, Messages: messages, Hash: sighash}
	signature, err := api.sign(addr, req, true)
	if err != nil {
		api.UI.ShowError(err.Error())
		return nil, err
	}
	return signature, nil
}

// HashStruct generates a keccak256 hash of the encoding of the provided data
func (typedData *TypedData) HashStruct(primaryType string, data TypedDataMessage) (hexutil.Bytes, error) {
	encodedData, err := typedData.EncodeData(primaryType, data, 1)
	if err != nil {
		return nil, err
	}
	return crypto.Keccak256(encodedData), nil
}

// Dependencies returns an array of custom types ordered by their hierarchical reference tree
func (typedData *TypedData) Dependencies(primaryType string, found []string) []string {
	includes := func(arr []string, str string) bool {
		for _, obj := range arr {
			if obj == str {
				return true
			}
		}
		return false
	}

	if includes(found, primaryType) {
		return found
	}
	if typedData.Types[primaryType] == nil {
		return found
	}
	found = append(found, primaryType)
	for _, field := range typedData.Types[primaryType] {
		for _, dep := range typedData.Dependencies(field.Type, found) {
			if !includes(found, dep) {
				found = append(found, dep)
			}
		}
	}
	return found
}

// EncodeType generates the following encoding:
// `name ‖ "(" ‖ member₁ ‖ "," ‖ member₂ ‖ "," ‖ … ‖ memberₙ ")"`
//
// each member is written as `type ‖ " " ‖ name` encodings cascade down and are sorted by name
func (typedData *TypedData) EncodeType(primaryType string) hexutil.Bytes {
	// Get dependencies primary first, then alphabetical
	deps := typedData.Dependencies(primaryType, []string{})
	if len(deps) > 0 {
		slicedDeps := deps[1:]
		sort.Strings(slicedDeps)
		deps = append([]string{primaryType}, slicedDeps...)
	}

	// Format as a string with fields
	var buffer bytes.Buffer
	for _, dep := range deps {
		buffer.WriteString(dep)
		buffer.WriteString("(")
		for _, obj := range typedData.Types[dep] {
			buffer.WriteString(obj.Type)
			buffer.WriteString(" ")
			buffer.WriteString(obj.Name)
			buffer.WriteString(",")
		}
		buffer.Truncate(buffer.Len() - 1)
		buffer.WriteString(")")
	}
	return buffer.Bytes()
}

// TypeHash creates the keccak256 hash  of the data
func (typedData *TypedData) TypeHash(primaryType string) hexutil.Bytes {
	return crypto.Keccak256(typedData.EncodeType(primaryType))
}

// EncodeData generates the following encoding:
// `enc(value₁) ‖ enc(value₂) ‖ … ‖ enc(valueₙ)`
//
// each encoded member is 32-byte long
func (typedData *TypedData) EncodeData(primaryType string, data map[string]interface{}, depth int) (hexutil.Bytes, error) {
	if err := typedData.validate(); err != nil {
		return nil, err
	}

	buffer := bytes.Buffer{}

	// Verify extra data
	if len(typedData.Types[primaryType]) < len(data) {
		return nil, errors.New("there is extra data provided in the message")
	}

	// Add typehash
	buffer.Write(typedData.TypeHash(primaryType))

	// Add field contents. Structs and arrays have special handlers.
	for _, field := range typedData.Types[primaryType] {
		encType := field.Type
		encValue := data[field.Name]
		if encType[len(encType)-1:] == "]" {
			arrayValue, ok := encValue.([]interface{})
			if !ok {
				return nil, dataMismatchError(encType, encValue)
			}

			arrayBuffer := bytes.Buffer{}
			parsedType := strings.Split(encType, "[")[0]
			for _, item := range arrayValue {
				if typedData.Types[parsedType] != nil {
					mapValue, ok := item.(map[string]interface{})
					if !ok {
						return nil, dataMismatchError(parsedType, item)
					}
					encodedData, err := typedData.EncodeData(parsedType, mapValue, depth+1)
					if err != nil {
						return nil, err
					}
					arrayBuffer.Write(encodedData)
				} else {
					bytesValue, err := typedData.EncodePrimitiveValue(parsedType, item, depth)
					if err != nil {
						return nil, err
					}
					arrayBuffer.Write(bytesValue)
				}
			}

			buffer.Write(crypto.Keccak256(arrayBuffer.Bytes()))
		} else if typedData.Types[field.Type] != nil {
			mapValue, ok := encValue.(map[string]interface{})
			if !ok {
				return nil, dataMismatchError(encType, encValue)
			}
			encodedData, err := typedData.EncodeData(field.Type, mapValue, depth+1)
			if err != nil {
				return nil, err
			}
			buffer.Write(crypto.Keccak256(encodedData))
		} else {
			byteValue, err := typedData.EncodePrimitiveValue(encType, encValue, depth)
			if err != nil {
				return nil, err
			}
			buffer.Write(byteValue)
		}
	}
	return buffer.Bytes(), nil
}

func parseInteger(encType string, encValue interface{}) (*big.Int, error) {
	var (
		length int
		signed = strings.HasPrefix(encType, "int")
		b      *big.Int
	)
	if encType == "int" || encType == "uint" {
		length = 256
	} else {
		lengthStr := ""
		if strings.HasPrefix(encType, "uint") {
			lengthStr = strings.TrimPrefix(encType, "uint")
		} else {
			lengthStr = strings.TrimPrefix(encType, "int")
		}
		atoiSize, err := strconv.Atoi(lengthStr)
		if err != nil {
			return nil, fmt.Errorf("invalid size on integer: %v", lengthStr)
		}
		length = atoiSize
	}
	switch v := encValue.(type) {
	case *math.HexOrDecimal256:
		b = (*big.Int)(v)
	case string:
		var hexIntValue math.HexOrDecimal256
		if err := hexIntValue.UnmarshalText([]byte(v)); err != nil {
			return nil, err
		}
		b = (*big.Int)(&hexIntValue)
	case float64:
		// JSON parses non-strings as float64. Fail if we cannot
		// convert it losslessly
		if float64(int64(v)) == v {
			b = big.NewInt(int64(v))
		} else {
			return nil, fmt.Errorf("invalid float value %v for type %v", v, encType)
		}
	}
	if b == nil {
		return nil, fmt.Errorf("invalid integer value %v/%v for type %v", encValue, reflect.TypeOf(encValue), encType)
	}
	if b.BitLen() > length {
		return nil, fmt.Errorf("integer larger than '%v'", encType)
	}
	if !signed && b.Sign() == -1 {
		return nil, fmt.Errorf("invalid negative value for unsigned type %v", encType)
	}
	return b, nil
}

// EncodePrimitiveValue deals with the primitive values found
// while searching through the typed data
func (typedData *TypedData) EncodePrimitiveValue(encType string, encValue interface{}, depth int) ([]byte, error) {
	switch encType {
	case "address":
		stringValue, ok := encValue.(string)
		if !ok || !common.IsHexAddress(stringValue) {
			return nil, dataMismatchError(encType, encValue)
		}
		retval := make([]byte, 32)
		copy(retval[12:], common.HexToAddress(stringValue).Bytes())
		return retval, nil
	case "bool":
		boolValue, ok := encValue.(bool)
		if !ok {
			return nil, dataMismatchError(encType, encValue)
		}
		if boolValue {
			return math.PaddedBigBytes(common.Big1, 32), nil
		}
		return math.PaddedBigBytes(common.Big0, 32), nil
	case "string":
		strVal, ok := encValue.(string)
		if !ok {
			return nil, dataMismatchError(encType, encValue)
		}
		return crypto.Keccak256([]byte(strVal)), nil
	case "bytes":
		bytesValue, ok := encValue.([]byte)
		if !ok {
			return nil, dataMismatchError(encType, encValue)
		}
		return crypto.Keccak256(bytesValue), nil
	}
	if strings.HasPrefix(encType, "bytes") {
		lengthStr := strings.TrimPrefix(encType, "bytes")
		length, err := strconv.Atoi(lengthStr)
		if err != nil {
			return nil, fmt.Errorf("invalid size on bytes: %v", lengthStr)
		}
		if length < 0 || length > 32 {
			return nil, fmt.Errorf("invalid size on bytes: %d", length)
		}
		if byteValue, ok := encValue.(hexutil.Bytes); !ok {
			return nil, dataMismatchError(encType, encValue)
		} else {
			return math.PaddedBigBytes(new(big.Int).SetBytes(byteValue), 32), nil
		}
	}
	if strings.HasPrefix(encType, "int") || strings.HasPrefix(encType, "uint") {
		b, err := parseInteger(encType, encValue)
		if err != nil {
			return nil, err
		}
		return abi.U256(b), nil
	}
	return nil, fmt.Errorf("unrecognized type '%s'", encType)

}

// dataMismatchError generates an error for a mismatch between
// the provided type and data
func dataMismatchError(encType string, encValue interface{}) error {
	return fmt.Errorf("provided data '%v' doesn't match type '%s'", encValue, encType)
}

// EcRecover recovers the address associated with the given sig.
// Only compatible with `text/plain`
func (api *SignerAPI) EcRecover(ctx context.Context, data hexutil.Bytes, sig hexutil.Bytes) (common.Address, error) {
	// Returns the address for the Account that was used to create the signature.
	//
	// Note, this function is compatible with eth_sign and personal_sign. As such it recovers
	// the address of:
	// hash = keccak256("\x19${byteVersion}Ethereum Signed Message:\n${message length}${message}")
	// addr = ecrecover(hash, signature)
	//
	// Note, the signature must conform to the secp256k1 curve R, S and V values, where
	// the V value must be be 27 or 28 for legacy reasons.
	//
	// https://github.com/ethereum/go-ethereum/wiki/Management-APIs#personal_ecRecover
	if len(sig) != 65 {
		return common.Address{}, fmt.Errorf("signature must be 65 bytes long")
	}
	if sig[64] != 27 && sig[64] != 28 {
		return common.Address{}, fmt.Errorf("invalid Ethereum signature (V is not 27 or 28)")
	}
	sig[64] -= 27 // Transform yellow paper V from 27/28 to 0/1
	hash := accounts.TextHash(data)
	rpk, err := crypto.SigToPub(hash, sig)
	if err != nil {
		return common.Address{}, err
	}
	return crypto.PubkeyToAddress(*rpk), nil
}

// UnmarshalValidatorData converts the bytes input to typed data
func UnmarshalValidatorData(data interface{}) (ValidatorData, error) {
	raw, ok := data.(map[string]interface{})
	if !ok {
		return ValidatorData{}, errors.New("validator input is not a map[string]interface{}")
	}
	addr, ok := raw["address"].(string)
	if !ok {
		return ValidatorData{}, errors.New("validator address is not sent as a string")
	}
	addrBytes, err := hexutil.Decode(addr)
	if err != nil {
		return ValidatorData{}, err
	}
	if !ok || len(addrBytes) == 0 {
		return ValidatorData{}, errors.New("validator address is undefined")
	}

	message, ok := raw["message"].(string)
	if !ok {
		return ValidatorData{}, errors.New("message is not sent as a string")
	}
	messageBytes, err := hexutil.Decode(message)
	if err != nil {
		return ValidatorData{}, err
	}
	if !ok || len(messageBytes) == 0 {
		return ValidatorData{}, errors.New("message is undefined")
	}

	return ValidatorData{
		Address: common.BytesToAddress(addrBytes),
		Message: messageBytes,
	}, nil
}

// validate makes sure the types are sound
func (typedData *TypedData) validate() error {
	if err := typedData.Types.validate(); err != nil {
		return err
	}
	if err := typedData.Domain.validate(); err != nil {
		return err
	}
	return nil
}

// Map generates a map version of the typed data
func (typedData *TypedData) Map() map[string]interface{} {
	dataMap := map[string]interface{}{
		"types":       typedData.Types,
		"domain":      typedData.Domain.Map(),
		"primaryType": typedData.PrimaryType,
		"message":     typedData.Message,
	}
	return dataMap
}

// Format returns a representation of typedData, which can be easily displayed by a user-interface
// without in-depth knowledge about 712 rules
func (typedData *TypedData) Format() ([]*NameValueType, error) {
	domain, err := typedData.formatData("EIP712Domain", typedData.Domain.Map())
	if err != nil {
		return nil, err
	}
	ptype, err := typedData.formatData(typedData.PrimaryType, typedData.Message)
	if err != nil {
		return nil, err
	}
	var nvts []*NameValueType
	nvts = append(nvts, &NameValueType{
		Name:  "EIP712Domain",
		Value: domain,
		Typ:   "domain",
	})
	nvts = append(nvts, &NameValueType{
		Name:  typedData.PrimaryType,
		Value: ptype,
		Typ:   "primary type",
	})
	return nvts, nil
}

func (typedData *TypedData) formatData(primaryType string, data map[string]interface{}) ([]*NameValueType, error) {
	var output []*NameValueType

	// Add field contents. Structs and arrays have special handlers.
	for _, field := range typedData.Types[primaryType] {
		encName := field.Name
		encValue := data[encName]
		item := &NameValueType{
			Name: encName,
			Typ:  field.Type,
		}
		if field.isArray() {
			arrayValue, _ := encValue.([]interface{})
			parsedType := field.typeName()
			for _, v := range arrayValue {
				if typedData.Types[parsedType] != nil {
					mapValue, _ := v.(map[string]interface{})
					mapOutput, err := typedData.formatData(parsedType, mapValue)
					if err != nil {
						return nil, err
					}
					item.Value = mapOutput
				} else {
					primitiveOutput, err := formatPrimitiveValue(field.Type, encValue)
					if err != nil {
						return nil, err
					}
					item.Value = primitiveOutput
				}
			}
		} else if typedData.Types[field.Type] != nil {
			if mapValue, ok := encValue.(map[string]interface{}); ok {
				mapOutput, err := typedData.formatData(field.Type, mapValue)
				if err != nil {
					return nil, err
				}
				item.Value = mapOutput
			} else {
				item.Value = "<nil>"
			}
		} else {
			primitiveOutput, err := formatPrimitiveValue(field.Type, encValue)
			if err != nil {
				return nil, err
			}
			item.Value = primitiveOutput
		}
		output = append(output, item)
	}
	return output, nil
}

func formatPrimitiveValue(encType string, encValue interface{}) (string, error) {
	switch encType {
	case "address":
		if stringValue, ok := encValue.(string); !ok {
			return "", fmt.Errorf("could not format value %v as address", encValue)
		} else {
			return common.HexToAddress(stringValue).String(), nil
		}
	case "bool":
		if boolValue, ok := encValue.(bool); !ok {
			return "", fmt.Errorf("could not format value %v as bool", encValue)
		} else {
			return fmt.Sprintf("%t", boolValue), nil
		}
	case "bytes", "string":
		return fmt.Sprintf("%s", encValue), nil
	}
	if strings.HasPrefix(encType, "bytes") {
		return fmt.Sprintf("%s", encValue), nil

	}
	if strings.HasPrefix(encType, "uint") || strings.HasPrefix(encType, "int") {
		if b, err := parseInteger(encType, encValue); err != nil {
			return "", err
		} else {
			return fmt.Sprintf("%d (0x%x)", b, b), nil
		}
	}
	return "", fmt.Errorf("unhandled type %v", encType)
}

// NameValueType is a very simple struct with Name, Value and Type. It's meant for simple
// json structures used to communicate signing-info about typed data with the UI
type NameValueType struct {
	Name  string      `json:"name"`
	Value interface{} `json:"value"`
	Typ   string      `json:"type"`
}

// Pprint returns a pretty-printed version of nvt
func (nvt *NameValueType) Pprint(depth int) string {
	output := bytes.Buffer{}
	output.WriteString(strings.Repeat("\u00a0", depth*2))
	output.WriteString(fmt.Sprintf("%s [%s]: ", nvt.Name, nvt.Typ))
	if nvts, ok := nvt.Value.([]*NameValueType); ok {
		output.WriteString("\n")
		for _, next := range nvts {
			sublevel := next.Pprint(depth + 1)
			output.WriteString(sublevel)
		}
	} else {
		output.WriteString(fmt.Sprintf("%q\n", nvt.Value))
	}
	return output.String()
}

// Validate checks if the types object is conformant to the specs
func (t Types) validate() error {
	for typeKey, typeArr := range t {
		if len(typeKey) == 0 {
			return fmt.Errorf("empty type key")
		}
		for i, typeObj := range typeArr {
			if len(typeObj.Type) == 0 {
				return fmt.Errorf("type %v:%d: empty Type", typeKey, i)
			}
			if len(typeObj.Name) == 0 {
				return fmt.Errorf("type %v:%d: empty Name", typeKey, i)
			}
			if typeKey == typeObj.Type {
				return fmt.Errorf("type '%s' cannot reference itself", typeObj.Type)
			}
			if typeObj.isReferenceType() {
				if _, exist := t[typeObj.typeName()]; !exist {
					return fmt.Errorf("reference type '%s' is undefined", typeObj.Type)
				}
				if !typedDataReferenceTypeRegexp.MatchString(typeObj.Type) {
					return fmt.Errorf("unknown reference type '%s", typeObj.Type)
				}
			} else if !isPrimitiveTypeValid(typeObj.Type) {
				return fmt.Errorf("unknown type '%s'", typeObj.Type)
			}
		}
	}
	return nil
}

// Checks if the primitive value is valid
func isPrimitiveTypeValid(primitiveType string) bool {
	if primitiveType == "address" ||
		primitiveType == "address[]" ||
		primitiveType == "bool" ||
		primitiveType == "bool[]" ||
		primitiveType == "string" ||
		primitiveType == "string[]" {
		return true
	}
	if primitiveType == "bytes" ||
		primitiveType == "bytes[]" ||
		primitiveType == "bytes1" ||
		primitiveType == "bytes1[]" ||
		primitiveType == "bytes2" ||
		primitiveType == "bytes2[]" ||
		primitiveType == "bytes3" ||
		primitiveType == "bytes3[]" ||
		primitiveType == "bytes4" ||
		primitiveType == "bytes4[]" ||
		primitiveType == "bytes5" ||
		primitiveType == "bytes5[]" ||
		primitiveType == "bytes6" ||
		primitiveType == "bytes6[]" ||
		primitiveType == "bytes7" ||
		primitiveType == "bytes7[]" ||
		primitiveType == "bytes8" ||
		primitiveType == "bytes8[]" ||
		primitiveType == "bytes9" ||
		primitiveType == "bytes9[]" ||
		primitiveType == "bytes10" ||
		primitiveType == "bytes10[]" ||
		primitiveType == "bytes11" ||
		primitiveType == "bytes11[]" ||
		primitiveType == "bytes12" ||
		primitiveType == "bytes12[]" ||
		primitiveType == "bytes13" ||
		primitiveType == "bytes13[]" ||
		primitiveType == "bytes14" ||
		primitiveType == "bytes14[]" ||
		primitiveType == "bytes15" ||
		primitiveType == "bytes15[]" ||
		primitiveType == "bytes16" ||
		primitiveType == "bytes16[]" ||
		primitiveType == "bytes17" ||
		primitiveType == "bytes17[]" ||
		primitiveType == "bytes18" ||
		primitiveType == "bytes18[]" ||
		primitiveType == "bytes19" ||
		primitiveType == "bytes19[]" ||
		primitiveType == "bytes20" ||
		primitiveType == "bytes20[]" ||
		primitiveType == "bytes21" ||
		primitiveType == "bytes21[]" ||
		primitiveType == "bytes22" ||
		primitiveType == "bytes22[]" ||
		primitiveType == "bytes23" ||
		primitiveType == "bytes23[]" ||
		primitiveType == "bytes24" ||
		primitiveType == "bytes24[]" ||
		primitiveType == "bytes25" ||
		primitiveType == "bytes25[]" ||
		primitiveType == "bytes26" ||
		primitiveType == "bytes26[]" ||
		primitiveType == "bytes27" ||
		primitiveType == "bytes27[]" ||
		primitiveType == "bytes28" ||
		primitiveType == "bytes28[]" ||
		primitiveType == "bytes29" ||
		primitiveType == "bytes29[]" ||
		primitiveType == "bytes30" ||
		primitiveType == "bytes30[]" ||
		primitiveType == "bytes31" ||
		primitiveType == "bytes31[]" {
		return true
	}
	if primitiveType == "int" ||
		primitiveType == "int[]" ||
		primitiveType == "int8" ||
		primitiveType == "int8[]" ||
		primitiveType == "int16" ||
		primitiveType == "int16[]" ||
		primitiveType == "int32" ||
		primitiveType == "int32[]" ||
		primitiveType == "int64" ||
		primitiveType == "int64[]" ||
		primitiveType == "int128" ||
		primitiveType == "int128[]" ||
		primitiveType == "int256" ||
		primitiveType == "int256[]" {
		return true
	}
	if primitiveType == "uint" ||
		primitiveType == "uint[]" ||
		primitiveType == "uint8" ||
		primitiveType == "uint8[]" ||
		primitiveType == "uint16" ||
		primitiveType == "uint16[]" ||
		primitiveType == "uint32" ||
		primitiveType == "uint32[]" ||
		primitiveType == "uint64" ||
		primitiveType == "uint64[]" ||
		primitiveType == "uint128" ||
		primitiveType == "uint128[]" ||
		primitiveType == "uint256" ||
		primitiveType == "uint256[]" {
		return true
	}
	return false
}

// validate checks if the given domain is valid, i.e. contains at least
// the minimum viable keys and values
func (domain *TypedDataDomain) validate() error {
	if domain.ChainId == nil {
		return errors.New("chainId must be specified according to EIP-155")
	}

	if len(domain.Name) == 0 && len(domain.Version) == 0 && len(domain.VerifyingContract) == 0 && len(domain.Salt) == 0 {
		return errors.New("domain is undefined")
	}

	return nil
}

// Map is a helper function to generate a map version of the domain
func (domain *TypedDataDomain) Map() map[string]interface{} {
	dataMap := map[string]interface{}{}

	if domain.ChainId != nil {
		dataMap["chainId"] = domain.ChainId
	}

	if len(domain.Name) > 0 {
		dataMap["name"] = domain.Name
	}

	if len(domain.Version) > 0 {
		dataMap["version"] = domain.Version
	}

	if len(domain.VerifyingContract) > 0 {
		dataMap["verifyingContract"] = domain.VerifyingContract
	}

	if len(domain.Salt) > 0 {
		dataMap["salt"] = domain.Salt
	}
	return dataMap
}
