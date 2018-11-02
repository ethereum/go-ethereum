// Copyright 2018 The go-ethereum Authors
// This file is part of go-ethereum.
//
// go-ethereum is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// go-ethereum is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with go-ethereum. If not, see <http://www.gnu.org/licenses/>.
//
package core

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"math/big"
	"mime"
	"reflect"
	"strconv"
	"strings"
	"unicode"

	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/crypto/sha3"
	"github.com/ethereum/go-ethereum/rlp"
)

type SigFormat struct {
	Mime        string
	ByteVersion byte
}

var (
	TextValidator = SigFormat{
		"text/validator",
		0x00,
	}
	DataTyped = SigFormat{
		"data/typed",
		0x01,
	}
	ApplicationClique = SigFormat{
		"application/clique",
		0x02,
	}
	TextPlain = SigFormat{
		"text/plain",
		0x45,
	}
)

type ValidatorData struct {
	Address 	common.Address
	Message		hexutil.Bytes
}

type TypedData struct {
	Types       EIP712Types  `json:"types"`
	PrimaryType string       `json:"primaryType"`
	Domain      EIP712Domain `json:"domain"`
	Message     EIP712Data   `json:"message"`
}

type EIP712Type []map[string]string

type EIP712Types map[string]EIP712Type

type EIP712TypePriority struct {
	Type  string
	Value uint
}

type EIP712Data = map[string]interface{}

type EIP712Domain struct {
	Name              string        	`json:"name"`
	Version           string        	`json:"version"`
	ChainId           *big.Int      	`json:"chainId"`
	VerifyingContract string			`json:"verifyingContract"`
	Salt              string 			`json:"salt"`
}

const (
	TypeAddress = "address"
	TypeBool    = "bool"
	TypeBytes   = "bytes"
	TypeInt     = "int"
	TypeString  = "string"
	TypeUint	= "uint"
)

// Sign receives a request and produces a signature

// Note, the produced signature conforms to the secp256k1 curve R, S and V values,
// where the V value will be 27 or 28 for legacy reasons.
func (api *SignerAPI) Sign(ctx context.Context, addr common.MixedcaseAddress, req *SignDataRequest) (hexutil.Bytes, error) {
	req.Address = addr
	req.Meta = MetadataFromContext(ctx)

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
	// Sign the data with the wallet
	signature, err := wallet.SignHashWithPassphrase(account, res.Password, req.Hash)
	if err != nil {
		return nil, err
	}
	signature[64] += 27 // Transform V from 0/1 to 27/28 according to the yellow paper
	return signature, nil
}

// SignData signs the hash of the provided data, but does so differently
// depending on the content-type specified.
//
// Different types of validation occur.
func (api *SignerAPI) SignData(ctx context.Context, contentType string, addr common.MixedcaseAddress, data interface{}) (hexutil.Bytes, error) {
	var req, err= api.determineSignatureFormat(contentType, addr, data)
	if err != nil {
		return nil, err
	}

	signature, err := api.Sign(ctx, addr, req)
	if err != nil {
		api.UI.ShowError(err.Error())
		return nil, err
	}

	return signature, nil
}

// Determines which signature method should be used based upon the mime type
// In the cases where it matters ensure that the charset is handled. The charset
// resides in the 'params' returned as the second returnvalue from mime.ParseMediaType
// charset, ok := params["charset"]
// As it is now, we accept any charset and just treat it as 'raw'.
func (api *SignerAPI) determineSignatureFormat(contentType string, addr common.MixedcaseAddress, data interface{}) (*SignDataRequest, error) {
	var req *SignDataRequest
	mediaType, _, err := mime.ParseMediaType(contentType)
	if err != nil {
		return nil, err
	}

	switch mediaType {
	case TextValidator.Mime:
		// Data with an intended validator
		validatorData, err := UnmarshalValidatorData(data)
		if err != nil {
			return nil, err
		}
		sighash, msg := SignTextValidator(validatorData)
		fmt.Printf("sighash:%s", sighash)
		req = &SignDataRequest{Rawdata: validatorData, Message: msg, Hash: sighash, ContentType: mediaType}
		break
	case DataTyped.Mime:
		// Signs EIP-712 conformant typed data
		// hash = keccak256("\x19${byteVersion}${domainSeparator}${hashStruct(message)}")
		typedData, err := UnmarshalTypedData(data)
		if err != nil {
			return nil, err
		}
		sighash, msg := SignDataTyped(typedData)
		req = &SignDataRequest{Rawdata: typedData, Message: msg, Hash: sighash, ContentType: mediaType}
		break
	case ApplicationClique.Mime:
		// Clique is the Ethereum PoA standard
		cliqueData, err := hexutil.Decode(data.(string))
		if err != nil {
			return nil, err
		}
		header := &types.Header{}
		if err := rlp.DecodeBytes(cliqueData, header); err != nil {
			return nil, err
		}
		sighash, err := SignCliqueHeader(header)
		if err != nil {
			return nil, err
		}
		msg := fmt.Sprintf("clique block %d [0x%x]", header.Number, header.Hash())
		req = &SignDataRequest{Rawdata: cliqueData, Message: msg, Hash: sighash, ContentType: mediaType}
		break
	case TextPlain.Mime:
		// Calculates an Ethereum ECDSA signature for:
		// hash = keccak256("\x19${byteVersion}Ethereum Signed Message:\n${message length}${message}")
		plainData, err := hexutil.Decode(data.(string))
		if err != nil {
			return nil, err
		}
		sighash, msg := SignTextPlain(plainData)
		req = &SignDataRequest{Rawdata: plainData, Message: msg, Hash: sighash, ContentType: mediaType}
		break
	default:
		return nil, fmt.Errorf("content type '%s' not implemented for signing", contentType)
	}
	return req, nil

}

// SignTextWithValidator signs the given message which can be further recovered
// with the given validator.
//
// hash = keccak256("\x19\x00"${address}${data}).
func SignTextValidator(validatorData ValidatorData) (hexutil.Bytes, string) {
	msg := fmt.Sprintf("\x19\x00%s%s", string(validatorData.Address.Bytes()), string(validatorData.Message))
	fmt.Printf("SignTextValidator:%s\n", msg)
	return crypto.Keccak256([]byte(msg)), msg
}

// SignCliqueHeader returns the hash which is used as input for the proof-of-authority
// signing. It is the hash of the entire header apart from the 65 byte signature
// contained at the end of the extra data.
//
// The method requires the extra data to be at least 65 bytes -- the original implementation
// in clique.go panics if this is the case, thus it's been reimplemented here to avoid the panic
// and simply return an error instead
func SignCliqueHeader(header *types.Header) (hexutil.Bytes, error) {
	hash := common.Hash{}
	if len(header.Extra) < 65 {
		return hash.Bytes(), fmt.Errorf("clique header extradata too short, %d < 65", len(header.Extra))
	}
	hasher := sha3.NewKeccak256()
	rlp.Encode(hasher, []interface{}{
		header.ParentHash,
		header.UncleHash,
		header.Coinbase,
		header.Root,
		header.TxHash,
		header.ReceiptHash,
		header.Bloom,
		header.Difficulty,
		header.Number,
		header.GasLimit,
		header.GasUsed,
		header.Time,
		header.Extra[:len(header.Extra)-65],
		header.MixDigest,
		header.Nonce,
	})
	hasher.Sum(hash[:0])
	return hash.Bytes(), nil
}

// SignTextPlain is a helper function that calculates a hash for the given message that can be
// safely used to calculate a signature from. This gives context to the signed message and prevents
// signing of transactions.
//
// hash = keccak256("\x19$Ethereum Signed Message:\n"${message length}${message}).
func SignTextPlain(data hexutil.Bytes) (hexutil.Bytes, string) {
	// The letter `E` is \x45 in hex, retrofitting
	// https://github.com/ethereum/go-ethereum/pull/2940/commits
	msg := fmt.Sprintf("\x19Ethereum Signed Message:\n%d%s", len(data), data)
	return crypto.Keccak256([]byte(msg)), msg
}

// SignDataTyped signs EIP-712 conformant typed data
// hash = keccak256("\x19${byteVersion}${domainSeparator}${hashStruct(message)}")
func SignDataTyped(typedData TypedData) (hexutil.Bytes, string) {
	domainSeparator := typedData.HashStruct("EIP712Domain", typedData.Domain.Map())
	typedDataHash := typedData.HashStruct(typedData.PrimaryType, typedData.Message)
	msg := fmt.Sprintf("\x19\x01%s%s", common.Bytes2Hex(domainSeparator), common.Bytes2Hex(typedDataHash))
	return crypto.Keccak256(common.Hex2Bytes(msg)), msg
}

// SignTypedData signs EIP-712 conformant typed data
// hash = keccak256("\x19${byteVersion}${domainSeparator}${hashStruct(message)}")
//func (api *SignerAPI) SignTypedData(ctx context.Context, addr common.MixedcaseAddress, typedData TypedData) (hexutil.Bytes, error) {
//	domainSeparator := typedData.HashStruct("EIP712Domain", typedData.Domain.Map())
//	typedDataHash := typedData.HashStruct(typedData.PrimaryType, typedData.Message)
//	typedDataJson, err := json.Marshal(typedData.Map())
//	if err != nil {
//		return nil, err
//	}
//	buffer := bytes.Buffer{}
//	buffer.WriteString("\x19")
//	buffer.WriteString("\x01")
//	buffer.WriteString(common.Bytes2Hex(domainSeparator))
//	buffer.WriteString(common.Bytes2Hex(typedDataHash))
//	req := &SignDataRequest{
//		Rawdata: typedDataJson,
//		Message: buffer.String(),
//		Hash: crypto.Keccak256(buffer.Bytes()),
//		ContentType: DataTyped.Mime,
//	}
//	signature, err := api.Sign(ctx, addr, req)
//	if err != nil {
//		api.UI.ShowError(err.Error())
//		return nil, err
//	}
//	return signature, nil
//}

// HashStruct generates the following encoding for the given domain and message:
// `encode(domainSeparator : 𝔹²⁵⁶, message : 𝕊) = "\x19\x01" ‖ domainSeparator ‖ hashStruct(message)`
func (typedData *TypedData) HashStruct(primaryType string, data EIP712Data) hexutil.Bytes {
	return crypto.Keccak256(typedData.EncodeData(primaryType, data))
}

// dependencies returns an array of custom types ordered by their hierarchical reference tree
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
		for _, dep := range typedData.Dependencies(field["type"], found) {
			if !includes(found, dep) {
				found = append(found, dep)
			}
		}
	}
	return found
}

// encodeType generates the following encoding:
// `name ‖ "(" ‖ member₁ ‖ "," ‖ member₂ ‖ "," ‖ … ‖ memberₙ ")"`
//
// each member is written as `type ‖ " " ‖ name` encodings cascade down and are sorted by name
func (typedData *TypedData) EncodeType(primaryType string) hexutil.Bytes {
	// Get dependencies primary first, then alphabetical
	deps := typedData.Dependencies(primaryType, []string{})
	for i, dep := range deps {
		if dep == primaryType {
			deps = append(deps[:i], deps[i+1:]...)
			break
		}
	}
	deps = append([]string{primaryType}, deps...)

	// Format as a string with fields
	var buffer bytes.Buffer
	for _, dep := range deps {
		buffer.WriteString(dep)
		buffer.WriteString("(")
		for _, obj := range typedData.Types[dep] {
			buffer.WriteString(obj["type"])
			buffer.WriteString(" ")
			buffer.WriteString(obj["name"])
			buffer.WriteString(",")
		}
		buffer.Truncate(buffer.Len() - 1)
		buffer.WriteString(")")
	}
	return buffer.Bytes()
}

func (typedData *TypedData) TypeHash(primaryType string) hexutil.Bytes {
	return crypto.Keccak256(typedData.EncodeType(primaryType))
}

// EncodeData generates the following encoding:
// `enc(value₁) ‖ enc(value₂) ‖ … ‖ enc(valueₙ)`
//
// each encoded member is 32-byte long
func (typedData *TypedData) EncodeData(primaryType string, data map[string]interface{}) hexutil.Bytes {
	encTypes := []string{}
	encValues := []interface{}{}

	// Add typehash
	encTypes = append(encTypes, "bytes32")
	encValues = append(encValues, typedData.TypeHash(primaryType))

	// Handle primitive values
	handlePrimitiveValue := func(encType string, encValue interface{}) (string, interface{}) {
		var primitiveEncType string
		var primitiveEncValue interface{}

		switch encType {
		case "address":
			primitiveEncType = "uint160"
			bytesValue := hexutil.Bytes{}
			for i := 0; i < 12; i++ {
				bytesValue = append(bytesValue, 0)
			}
			for _, _byte := range common.HexToAddress(encValue.(string)) {
				bytesValue = append(bytesValue, _byte)
			}
			primitiveEncValue = bytesValue
			break
		case "bool":
			primitiveEncType = "uint256"
			var int64Val int64
			if encValue.(bool) {
				int64Val = 1
			}
			primitiveEncValue = abi.U256(big.NewInt(int64Val))
			break
		case "bytes", "string":
			primitiveEncType = "bytes32"
			primitiveEncValue = crypto.Keccak256(bytesValueOf(encValue))
			break
		default:
			if strings.HasPrefix(encType, "bytes") {
				encTypes = append(encTypes, "bytes32")
				sizeStr := strings.TrimPrefix(encType, "bytes")
				size, _ := strconv.Atoi(sizeStr)
				bytesValue := hexutil.Bytes{}
				for i := 0; i < 32-size; i++ {
					bytesValue = append(bytesValue, 0)
				}
				for _, _byte := range encValue.(hexutil.Bytes) {
					bytesValue = append(bytesValue, _byte)
				}
				primitiveEncValue = bytesValue
			} else if strings.HasPrefix(encType, "uint") || strings.HasPrefix(encType, "int") {
				primitiveEncType = "uint256"
				primitiveEncValue = abi.U256(encValue.(*big.Int))
			}
			break
		}
		return primitiveEncType, primitiveEncValue
	}

	// Add field contents. Structs and arrays have special handlings.
	for _, field := range typedData.Types[primaryType] {
		encType := field["type"]
		encValue := data[field["name"]]
		if encType[len(encType)-1:] == "]" {
			encTypes = append(encTypes, "bytes32")
			parsedType := strings.Split(encType, "[")[0]
			arrayBuffer := bytes.Buffer{}
			for _, item := range encValue.([]interface{}) {
				if typedData.Types[parsedType] != nil {
					encoding := typedData.EncodeData(parsedType, item.(map[string]interface{}))
					arrayBuffer.Write(encoding)
				} else {
					_, encValue := handlePrimitiveValue(encType, encValue)
					arrayBuffer.Write(bytesValueOf(encValue))
				}
			}
			encValues = append(encValues, crypto.Keccak256(arrayBuffer.Bytes()))
		} else if typedData.Types[field["type"]] != nil {
			encTypes = append(encTypes, "bytes32")
			mapValue := encValue.(map[string]interface{})
			encValue = crypto.Keccak256(typedData.EncodeData(field["type"], mapValue))
			encValues = append(encValues, encValue)
		} else {
			primitiveEncType, primitiveEncValue := handlePrimitiveValue(encType, encValue)
			encTypes = append(encTypes, primitiveEncType)
			encValues = append(encValues, primitiveEncValue)
		}
	}

	buffer := bytes.Buffer{}
	for _, encValue := range encValues {
		buffer.Write(bytesValueOf(encValue))
	}

	return buffer.Bytes() // https://github.com/ethereumjs/ethereumjs-abi/blob/master/lib/index.js#L336
}

func bytesValueOf(_interface interface{}) hexutil.Bytes {
	bytesValue, ok := _interface.(hexutil.Bytes)
	if ok {
		return bytesValue
	}

	switch reflect.TypeOf(_interface) {
	case reflect.TypeOf(hexutil.Bytes{}):
		return _interface.(hexutil.Bytes)
	case reflect.TypeOf([]uint8{}):
		return _interface.([]uint8)
	case reflect.TypeOf(string("")):
		return common.Hex2Bytes(_interface.(string))
	default:
		break
	}

	panic(fmt.Errorf("unrecognized interface type %T", _interface))
	return hexutil.Bytes{}
}

// EcRecover recovers the address associated with the given sig.
// Only compatible with `text/plain`
func (api *SignerAPI) EcRecover(ctx context.Context, contentType string, data hexutil.Bytes, sig hexutil.Bytes) (common.Address, error) {
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
	hash, _ := SignTextPlain(data)
	rpk, err := crypto.SigToPub(hash, sig)
	if err != nil {
		return common.Address{}, err
	}
	return crypto.PubkeyToAddress(*rpk), nil
}

// UnmarshalValidatorData converts the bytes input to typed data
func UnmarshalValidatorData(data interface{}) (ValidatorData, error) {
	raw := data.(map[string]interface{})

	addr, ok := raw["address"].(string)
	addrBytes, err := hexutil.Decode(addr)
	if err != nil {
		return ValidatorData{}, err
	}
	if !ok || len(addrBytes) == 0 {
		return ValidatorData{}, errors.New("validator address is undefined")
	}

	message, ok := raw["message"].(string)
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

// UnmarshalTypedData converts the bytes input to typed data
func UnmarshalTypedData(data interface{}) (TypedData, error) {
	raw := data.(map[string]interface{})

	var _types, ok = raw["types"].(EIP712Types)
	if !ok || _types == nil {
		return TypedData{}, errors.New("types are undefined")
	}
	if err := _types.IsValid(); err != nil {
		return TypedData{}, err
	}

	if _types["EIP712Domain"] == nil {
		return TypedData{}, errors.New("domain types are undefined")
	}

	domain, err := UnmarshalDomain(data)
	if err != nil {
		return TypedData{}, err
	}

	primaryType, ok := raw["primaryType"].(string)
	if !ok || len(primaryType) == 0 {
		return TypedData{}, errors.New("primary type is undefined")
	}

	message, ok := raw["message"].(EIP712Data)
	if !ok || message == nil {
		return TypedData{}, errors.New("message is undefined")
	}
	return TypedData{
		Types: _types,
		PrimaryType: primaryType,
		Domain: domain,
		Message: message,
	}, nil
}

// UnmarshalDomain converts the bytes input to a domain
func UnmarshalDomain(data interface{}) (EIP712Domain, error) {
	raw := data.(map[string]interface{})["domain"].(map[string]interface{})

	chainId := raw["chainId"].(*big.Int)
	if chainId == big.NewInt(0) {
		return EIP712Domain{}, errors.New("chainId must be specified according to EIP-155")
	}

	name, nameOk := raw["name"].(string)
	version, versionOk := raw["version"].(string)
	verifyingContract, verifyingContractOk := raw["verifyingContract"].(string)
	salt, saltOk := raw["salt"].(string)
	if (!nameOk || len(name) == 0) &&
		(!versionOk || len(version) == 0) &&
		(!verifyingContractOk || len(verifyingContract) == 0) &&
		(!saltOk || len(salt) == 0) {
		return EIP712Domain{}, errors.New("domain is undefined")
	}

	return EIP712Domain{
		Name: name,
		Version: version,
		ChainId: chainId,
		VerifyingContract: verifyingContract,
		Salt: salt,
	}, nil
}


// Map is a helper function to generate a map version of the typed data
func (typedData *TypedData) Map() map[string]interface{} {
	dataMap := map[string]interface{}{
		"types":       typedData.Types,
		"domain":      typedData.Domain.Map(),
		"primaryType": typedData.PrimaryType,
		"message":     typedData.Message,
	}

	return dataMap
}

// IsValid checks if the given types object is conformant to the specs
func (types *EIP712Types) IsValid() error {
	for typeKey, typeArr := range *types {
		for _, typeObj := range typeArr {
			typeVal := typeObj["type"]
			if typeKey == typeVal {
				panic(fmt.Errorf("type %s cannot reference itself", typeVal))
			}

			firstChar := []rune(typeVal)[0]
			if unicode.IsUpper(firstChar) {
				if (*types)[typeVal] == nil {
					return fmt.Errorf("referenced type %s is undefined", typeVal)
				}
			} else {
				// TODO: better type checking
				if !isStandardTypeStr(typeVal) {
					if (*types)[typeVal] != nil {
						return fmt.Errorf("custom type %s must be capitalized", typeVal)
					} else {
						return fmt.Errorf("unknown type %s", typeVal)
					}
				}
			}
		}
	}
	return nil
}

// isStandardType checks if the given type is a EIP712 conformant type
func isStandardTypeStr(encType string) bool {
	// Atomic types
	for _, standardType := range []string{
		TypeAddress,
		TypeBool,
		TypeBytes,
		TypeString,
	} {
		if standardType == encType {
			return true
		}
	}

	// Dynamic types
	for _, standardType := range []string {
		TypeBytes,
		TypeInt,
		TypeUint,
	} {
		if strings.HasPrefix(encType, standardType) {
			return true
		}
	}

	// Reference types
	if encType[len(encType)-1] == ']' {
		return true
	}

	return false
}

// IsValid checks if the given domain is valid, i.e. contains at least
// the minimum viable keys and values
func (domain *EIP712Domain) IsValid() error {
	if domain.ChainId == big.NewInt(0) {
		return errors.New("chainId must be specified according to EIP-155")
	}

	if len(domain.Name) == 0 && len(domain.Version) == 0 && len(domain.VerifyingContract) == 0 && len(domain.Salt) == 0 {
		return errors.New("domain undefined")
	}

	return nil
}

// Map is a helper function to generate a map version of the domain
func (domain *EIP712Domain) Map() map[string]interface{} {
	dataMap := map[string]interface{}{
		"chainId": domain.ChainId,
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