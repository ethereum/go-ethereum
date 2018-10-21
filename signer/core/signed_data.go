package core

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto/sha3"
	"github.com/ethereum/go-ethereum/rlp"
	"math/big"
	"mime"
	"reflect"
	"sort"
	"strings"
	"unicode"

	"github.com/PaulRBerg/basics/helpers"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
)

type TypedData struct {
	Types       EIP712Types				`json:"types"`
	PrimaryType string 					`json:"primaryType"`
	Domain      EIP712Domain			`json:"domain"`
	Message     EIP712Data				`json:"message"`
}

type EIP712Type []map[string]string

type EIP712Types map[string]EIP712Type

type EIP712TypePriority struct {
	Type  string
	Value uint
}

type EIP712Data = map[string]interface{}

type EIP712Domain struct {
	Name              string         	`json:"name"`
	Version           string         	`json:"version"`
	ChainId           *big.Int       	`json:"chainId"`
	VerifyingContract common.Address 	`json:"verifyingContract"`
	Salt              hexutil.Bytes  	`json:"salt"`
}

const (
	TypeArray 		= "array"
	TypeAddress 	= "address"
	TypeBool 		= "bool"
	TypeBytes 		= "bytes"
	TypeInt 		= "int"
	TypeString		= "string"
)

// Sign receives a request and produces a signature

// Note, the produced signature conforms to the secp256k1 curve R, S and V values,
// where the V value will be 27 or 28 for legacy reasons.
func (api *SignerAPI) Sign(ctx context.Context, addr common.MixedcaseAddress, req *SignDataRequest) ([]byte, error) {
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
func (api *SignerAPI) SignData(ctx context.Context, contentType string, addr common.MixedcaseAddress, data hexutil.Bytes) (hexutil.Bytes, error) {
	var req, err = api.determineSignatureFormat(contentType, data)
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
func (api *SignerAPI) determineSignatureFormat(contentType string, data hexutil.Bytes) (*SignDataRequest, error) {
	var req *SignDataRequest
	mediaType, _, err := mime.ParseMediaType(contentType)
	if err != nil {
		return nil, err
	}

	switch mediaType {
	case TextValidator.Mime:
		// Data with an intended validator
		sighash, msg := signTextWithValidator(data)
		req = &SignDataRequest{Rawdata: data, Message: msg, Hash: sighash, ContentType: mediaType}
		break
	case TextPlain.Mime:
		// Sign calculates an Ethereum ECDSA signature for:
		// hash = keccak256("\x19${byteVersion}Ethereum Signed Message:\n${message length}${message}")

		// In the cases where it matters ensure that the charset is handled. The charset
		// resides in the 'params' returned as the second returnvalue from mime.ParseMediaType
		// charset, ok := params["charset"]
		// As it is now, we accept any charset and just treat it as 'raw'.
		sighash, msg := signTextPlain(data)
		req = &SignDataRequest{Rawdata: data, Message: msg, Hash: sighash, ContentType: mediaType}
		break
	case ApplicationClique.Mime:
		// Clique is the Ethereum PoA standard
		header := &types.Header{}
		if err := rlp.DecodeBytes(data, header); err != nil {
			return nil, err
		}
		sighash, err := signCliqueHeader(header)
		if err != nil {
			return nil, err
		}
		msg := fmt.Sprintf("Clique block %d [0x%x]", header.Number, header.Hash())
		req = &SignDataRequest{Rawdata: data, Message: msg, Hash: sighash, ContentType: mediaType}
		break
	default:
		return nil, fmt.Errorf("content type '%s' not implemented for signing", contentType)
	}
	return req, nil

}

// signTextPlain is a helper function that calculates a hash for the given message that can be
// safely used to calculate a signature from.
//
// The hash is calculated as
//	keccak256("\x19${byteVersion}Ethereum Signed Message:\n"${message length}${message}).
//
// This gives context to the signed message and prevents signing of transactions.
func signTextPlain(data []byte) ([]byte, string) {
	msg := fmt.Sprintf("\x19\\x%xEthereum Signed Message:\n%d%s", TextPlain.ByteVersion, len(data), data)
	return crypto.Keccak256([]byte(msg)), msg
}

// signTextWithValidator signs the given message which can be further recovered
// with the given validator.
func signTextWithValidator(data []byte) ([]byte, string) {
	msg := "TODO"
	return crypto.Keccak256([]byte(msg)), msg
}

// signCliqueHeader returns the hash which is used as input for the proof-of-authority
// signing. It is the hash of the entire header apart from the 65 byte signature
// contained at the end of the extra data.
//
// The method requires the extra data to be at least 65 bytes -- the original implementation
// in clique.go panics if this is the case, thus it's been reimplemented here to avoid the panic
// and simply return an error instead
func signCliqueHeader(header *types.Header) (hexutil.Bytes, error) {
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

// SignTypedData signs EIP712 conformant typed data
// hash = keccak256("\x19${byteVersion}${domainSeparator}${hashStruct(message)}")
func (api *SignerAPI) SignTypedData(ctx context.Context, addr common.MixedcaseAddress, typedData TypedData) (hexutil.Bytes, error) {
	domainTypes := EIP712Types{
		"EIP712Domain": typedData.Types["EIP712Domain"],
	}
	domainSeparatorBytes := hashStruct(domainTypes, typedData.Domain.Map(), "EIP712Domain", 0)
	domainSeparator := common.Bytes2Hex(domainSeparatorBytes)
	//if err != nil {
	//	return nil, err
	//}
	delete(typedData.Types, "EIP712Domain")
	typedDataHashBytes := hashStruct(typedData.Types, typedData.Message, typedData.PrimaryType, 0)
	typedDataHash := common.Bytes2Hex(typedDataHashBytes)
	//if err != nil {
	//	return nil, err
	//}

	fmt.Println("domainSeparator", domainSeparator)
	fmt.Println("typedDataHash", typedDataHash)

	var buffer bytes.Buffer
	buffer.WriteString("\x19\x01")
	buffer.WriteString(domainSeparator)
	buffer.WriteString(typedDataHash)

	sighash := crypto.Keccak256(buffer.Bytes())
	msg := fmt.Sprintf("\x19\\x%xEthereum Signed Message\n:%s%s", DataTyped.ByteVersion, domainSeparator, typedDataHash)
	//req := &SignDataRequest{Rawdata: typedData, Message: msg, Hash: sighash, ContentType: DataTyped.Mime}

	var req, err = api.determineSignatureFormat(contentType, data)
	if err != nil {
		return nil, err
	}

	signature, err := api.Sign(ctx, addr, req)
	if err != nil {
		api.UI.ShowError(err.Error())
		return nil, err
	}

	return signature, nil

	return crypto.Keccak256(buffer.Bytes()), nil
}

// hashStruct generates the following encoding for the given domain and message:
// `encode(domainSeparator : 𝔹²⁵⁶, message : 𝕊) = "\x19\x01" ‖ domainSeparator ‖ hashStruct(message)`
func hashStruct(_types EIP712Types, data EIP712Data, dataType string, depth int) []byte {
	helpers.PrintJson("hashStruct", map[string]interface{}{
		"depth": depth,
	})

	typeEncoding := encodeType(_types)
	typeHash := hex.EncodeToString(crypto.Keccak256(typeEncoding))

	dataEncoding := encodeData(_types, data, dataType, depth)
	dataHash := hex.EncodeToString(crypto.Keccak256(dataEncoding))

	var buffer bytes.Buffer
	buffer.WriteString(typeHash)
	buffer.WriteString(dataHash)
	encoding := crypto.Keccak256(buffer.Bytes())

	if depth == 0 {
		fmt.Printf("typeEncoding %s\n", common.Bytes2Hex(typeEncoding))
		fmt.Printf("dataEncoding %s\n", common.Bytes2Hex(dataEncoding))
	}

	return encoding
}

// encodeType generates the followign encoding:
// `name ‖ "(" ‖ member₁ ‖ "," ‖ member₂ ‖ "," ‖ … ‖ memberₙ ")"`
//
// each member is written as `type ‖ " " ‖ name` encodings cascade down and are sorted by name
func encodeType(_types EIP712Types) []byte {
	helpers.PrintJson("encodeType", map[string]interface{}{
		"types": _types,
	})
	//fmt.Printf("encodeType: types %v\n\n", types)

	var priorities = make(map[string]uint)
	for key := range _types {
		priorities[key] = 0
	}

	// Updates the priority for every new custom type discovered
	update := func(typeKey string, typeVal string) {
		priorities[typeVal]++

		// Importantly, we also have to check for parent types to increment them too
		for _, typeObj := range _types[typeVal] {
			_typeVal := typeObj["type"]

			firstChar := []rune(_typeVal)[0]
			if unicode.IsUpper(firstChar) {
				priorities[_typeVal]++
			}
		}
	}

	// Checks if referenced type has already been visited to optimise algo
	visited := func(arr []string, val string) bool {
		for _, elem := range arr {
			if elem == val {
				return true
			}
		}
		return false
	}

	for typeKey, typeArr := range _types {
		var typeValArr []string

		for _, typeObj := range typeArr {
			typeVal := typeObj["type"]
			//if typeKey == typeVal {
			//	panic(fmt.Errorf("type %s cannot reference itself", typeVal))
			//}

			firstChar := []rune(typeVal)[0]
			if unicode.IsUpper(firstChar) {
				if _types[typeVal] != nil {
					if !visited(typeValArr, typeVal) {
						typeValArr = append(typeValArr, typeVal)
						update(typeKey, typeVal)
					}
				}
			}
		}

		typeValArr = []string{}
	}

	sortedPriorities := sortByPriorityAndName(priorities)
	var buffer bytes.Buffer
	for _, priority := range sortedPriorities {
		typeKey := priority.Type
		typeArr := _types[typeKey]

		buffer.WriteString(typeKey)
		buffer.WriteString("(")

		for _, typeObj := range typeArr {
			buffer.WriteString(typeObj["type"])
			buffer.WriteString(" ")
			buffer.WriteString(typeObj["name"])
			buffer.WriteString(",")
		}

		buffer.Truncate(buffer.Len() - 1)
		buffer.WriteString(")")
	}

	return buffer.Bytes()
}

// sortByPriorityAndName is a helper function to sort types by priority and name. Priority is calculated b
// based upon the number of references.
func sortByPriorityAndName(input map[string]uint) []EIP712TypePriority {
	var priorities []EIP712TypePriority
	for key, val := range input {
		priorities = append(priorities, EIP712TypePriority{key, val})
	}
	// Alphabetically
	sort.Slice(priorities, func(i, j int) bool {
		return priorities[i].Type < priorities[j].Type
	})
	// Priority
	sort.Slice(priorities, func(i, j int) bool {
		return priorities[i].Value > priorities[j].Value
	})

	//for _, priority := range priorities {
	//	fmt.Printf("%s, Value %d\n", priority.Type, priority.Value)
	//}
	//fmt.Printf("\n")

	return priorities
}

// encodeData generates the following encoding:
// `enc(value₁) ‖ enc(value₂) ‖ … ‖ enc(valueₙ)`
//
// each encoded member is 32-byte long
func encodeData(_types EIP712Types, data interface{}, dataType string, depth int) []byte {
	helpers.PrintJson("encodeData", map[string]interface{}{
		"dataType": dataType,
		"data":   data,
		"depth": depth,
	})

	var buffer bytes.Buffer

	// TODO regex
	// handle arrays
	if strings.Contains(dataType, "[]") {
		arrayVal := data.([]interface{})
		dataType := "TODO"

		var arrayBuffer bytes.Buffer
		for obj := range arrayVal {
			objEncoding := encodeData(_types, obj, dataType, depth+1)
			arrayBuffer.Write(objEncoding)
		}

		encoding := arrayBuffer.Bytes()
		buffer.Write(encoding)
		return buffer.Bytes()
	}

	// handle maps
	firstChar := []rune(dataType)[0]
	if unicode.IsUpper(firstChar) {
		for mapKey, mapVal := range data.(EIP712Data) {
			nextDataType := findNextDataType(_types, dataType, mapKey)
			if reflect.TypeOf(mapVal) == reflect.TypeOf(EIP712Data{}) {
				data := mapVal.(map[string]interface{})
				encoding := hashStruct(_types, data, nextDataType, depth+1)
				buffer.Write(encoding)
			} else {
				encoding := encodeData(_types, mapVal, nextDataType, depth+1)
				buffer.Write(encoding)
			}
		}
		return buffer.Bytes()
	}

	// TODO regex
	// handle bytes
	if strings.Contains(dataType, TypeBytes) {
		bytesVal := data.([]byte)
		encoding := crypto.Keccak256(bytesVal)
		buffer.Write(encoding)
	}

	// TODO regex
	// handle ints
	if strings.Contains(dataType, TypeInt) {
		encoding := abi.U256(data.(*big.Int)) // not sure if this is big endian order, but it's definitey sign extended to 256 bit because of using the U256 function
		buffer.Write(encoding)
		return buffer.Bytes()
	}

	// handle what's left
	switch dataType {
	case TypeAddress:
		addressVal, _ := data.(common.Address)
		encoding := addressVal.Bytes() // hopefully this means uint160 encoding?
		buffer.Write(encoding)
		break
	case TypeBool:
		boolVal, _ := data.(bool)
		var int64Val int64
		if boolVal {
			int64Val = 1
		}
		encoding := abi.U256(big.NewInt(int64Val))
		buffer.Write(encoding)
		break
	case TypeString:
		bytesVal := common.FromHex(data.(string))
		encoding := crypto.Keccak256(bytesVal)
		buffer.Write(encoding)
		break
	default:
		break
	}

	return buffer.Bytes()
}

// findNextDataType
// blah blah
func findNextDataType(_types EIP712Types, mapType string, mapKey string) string {
	eip712type := _types[mapType]

	for _, mapObj := range eip712type {
		if mapObj["name"] == mapKey {
			return mapObj["type"]
		}
	}

	return ""
}

// UnmarshalJSON validates the input data
func (typedData *TypedData) UnmarshalJSON(data []byte) error {
	type input struct {
		Types       EIP712Types 			`json:"types"`
		PrimaryType string                	`json:"primaryType"`
		Domain      EIP712Domain          	`json:"domain"`
		Message     EIP712Data         		`json:"message"`
	}

	var raw input
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}

	if raw.Types == nil {
		return errors.New("types are undefined")
	}
	if err := raw.Types.IsValid(); err != nil {
		return err
	}
	typedData.Types = raw.Types

	if raw.Types["EIP712Domain"] == nil {
		return errors.New("domain types are undefined")
	}
	if err := raw.Domain.IsValid(); err != nil {
		return err
	}
	typedData.Domain = raw.Domain

	if len(raw.PrimaryType) == 0 {
		return errors.New("primary type is undefined")
	}
	typedData.PrimaryType = raw.PrimaryType

	if raw.Message == nil {
		return errors.New("message is undefined")
	}
	typedData.Message = raw.Message

	return nil
}

// isStandardType checks if the given type is a EIP712 conformant type
func isStandardTypeStr(typeStr string) bool {
	standardTypes := []string{
		"array",
		"address",
		"boolean",
		"bytes",
		"string",
		"struct",
		"uint",
	}
	for _, val := range standardTypes {
		if strings.HasPrefix(typeStr, val) {
			return true
		}
	}
	return false
}


// IsValid checks if the given types object is conformant to the specs
func (types *EIP712Types) IsValid() error {
	for typeKey, typeArr := range (*types) {
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

// Values is a helper function to return the values of a domain as a map
// with arbitrary values
func (domain *EIP712Domain) Map() EIP712Data {
	dataMap := EIP712Data{
		"chainId":		domain.ChainId,
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

// Determines the content type and then recovers the address associated with the given sig
func (api *SignerAPI) EcRecover(ctx context.Context, contentType string, data hexutil.Bytes, sig hexutil.Bytes) (common.Address, error) {
	mediaType, _, err := mime.ParseMediaType(contentType)
	if err != nil {
		return common.Address{}, err
	}
	switch mediaType {
	case TextPlain.Mime:
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
		hash, _ := signTextPlain(data)
		rpk, err := crypto.SigToPub(hash, sig)
		if err != nil {
			return common.Address{}, err
		}
		return crypto.PubkeyToAddress(*rpk), nil
	default:
		return common.Address{}, fmt.Errorf("content type '%s' not implemented for ecRecover", contentType)
	}
}