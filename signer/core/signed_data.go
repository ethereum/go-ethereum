package core

import (
	"bytes"
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"math/big"
	"math/rand"
	"reflect"
	"sort"
	"strings"
	"time"
	"unicode"

	"github.com/PaulRBerg/basics/helpers"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
)

type TypedData struct {
	Types       map[string]EIP712Type `json:"types"`
	PrimaryType string                `json:"primaryType"`
	Domain      EIP712Domain          `json:"domain"`
	Message     EIP712Message         `json:"message"`
}

type EIP712Type []map[string]string

type EIP712TypePriority struct {
	Type  string
	Value uint
}

type EIP712Data = map[string]interface{}

type EIP712Domain struct {
	Name              string         `json:"name"`
	Version           string         `json:"version"`
	ChainId           *big.Int       `json:"chainId"`
	VerifyingContract common.Address `json:"verifyingContract"`
	Salt              hexutil.Bytes  `json:"salt"`
}

type EIP712Message map[string]interface{}

// SignTypedData signs EIP712 conformant typed data
// hash = keccak256("\x19${byteVersion}${domainSeparator}${hashStruct(message)}")
func (api *SignerAPI) SignTypedData(ctx context.Context, addr common.MixedcaseAddress, data TypedData) (hexutil.Bytes, error) {
	if err := data.Domain.IsValid(); err != nil {
		return nil, err
	}
	if data.PrimaryType == "" {
		return nil, errors.New("primary type undefined")
	}

	domainTypes := map[string]EIP712Type{
		"EIP712Domain": data.Types["EIP712Domain"],
	}
	domainSeparator := hashStruct(domainTypes, data.PrimaryType, data.Domain.Values(), 0)
	//if err != nil {
	//	return nil, err
	//}
	delete(data.Types, "EIP712Domain")
	typedDataHash := hashStruct(data.Types, data.PrimaryType, data.Message, 0)
	//if err != nil {
	//	return nil, err
	//}

	fmt.Println("domainSeparator", domainSeparator.String())
	fmt.Println("typedDataHash", typedDataHash.String())
	return common.FromHex("0xdeadbeef"), nil
}

// hashStruct generates the following encoding for the given domain and message:
// `encode(domainSeparator : 𝔹²⁵⁶, message : 𝕊) = "\x19\x01" ‖ domainSeparator ‖ hashStruct(message)`
func hashStruct(types map[string]EIP712Type, key string, data EIP712Data, depth int) common.Hash {
	helpers.PrintJson("hashStruct", map[string]interface{}{
		"depth": depth,
	})

	typeEncoding := encodeType(types)
	typeHash := hex.EncodeToString(crypto.Keccak256([]byte(typeEncoding)))

	dataEncoding := encodeData(types, key, data, depth)
	dataHash := hex.EncodeToString(crypto.Keccak256([]byte(dataEncoding)))

	var buffer bytes.Buffer
	buffer.WriteString(typeHash)
	buffer.WriteString(dataHash)
	hash := common.BytesToHash(crypto.Keccak256(buffer.Bytes()))

	if depth == 0 {
		fmt.Printf("typeEncoding %s\n", typeEncoding)
		fmt.Printf("dataEncoding %s\n", dataEncoding)
	}
	return hash
}

// encodeType generates the followign encoding:
// `name ‖ "(" ‖ member₁ ‖ "," ‖ member₂ ‖ "," ‖ … ‖ memberₙ ")"`
//
// each member is written as `type ‖ " " ‖ name` encodings cascade down and are sorted by name
func encodeType(types map[string]EIP712Type) string {
	helpers.PrintJson("hashStruct", map[string]interface{}{
		"types": types,
	})

	var priorities = make(map[string]uint)
	for key := range types {
		priorities[key] = 0
	}

	// Updates the priority for every new custom type discovered
	update := func(typeKey string, typeVal string) {
		priorities[typeVal]++

		// Importantly, we also have to check for parent types to increment them too
		for _, typeObj := range types[typeVal] {
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

	for typeKey, typeArr := range types {
		var typeValArr []string

		for _, typeObj := range typeArr {
			typeVal := typeObj["type"]
			if typeKey == typeVal {
				panic(fmt.Errorf("type %s cannot reference itself", typeVal))
			}

			firstChar := []rune(typeVal)[0]
			if unicode.IsUpper(firstChar) {
				if types[typeVal] != nil {
					if !visited(typeValArr, typeVal) {
						typeValArr = append(typeValArr, typeVal)
						update(typeKey, typeVal)
					}
				} else {
					panic(fmt.Errorf("referenced type %s is undefined", typeVal))
				}
			} else {
				if !isStandardType(typeVal) {
					if types[typeVal] != nil {
						panic(fmt.Errorf("Custom type %s must be capitalized", typeVal))
					} else {
						panic(fmt.Errorf("Unknown type %s", typeVal))
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
		typeArr := types[typeKey]

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

	return buffer.String()
}

// encodeData generates the following encoding:
// `enc(value₁) ‖ enc(value₂) ‖ … ‖ enc(valueₙ)`
//
// each encoded member is 32-byte long
func encodeData(types map[string]EIP712Type, key string, val interface{}, depth int) string {
	helpers.PrintJson("hashStruct", map[string]interface{}{
		"key":   key,
		"val":   val,
		"depth": depth,
	})

	var buffer bytes.Buffer

	switch val.(type) {
	case EIP712Data:
		for mapKey, mapVal := range val.(EIP712Data) {
			if reflect.TypeOf(mapVal) == reflect.TypeOf(EIP712Data{}) {
				hash := hashStruct(types, mapKey, mapVal.(EIP712Data), depth+1)
				buffer.WriteString(hash.String())
			} else {
				str := encodeData(types, mapKey, mapVal, depth+1)
				buffer.WriteString(str)
			}
		}
		break

	case bool:
		boolVal, _ := val.(bool)
		var int64Val int64
		if boolVal {
			int64Val = 1
		}
		encodedVal := abi.U256(big.NewInt(int64Val))
		fmt.Printf("bool encoded value:", encodedVal)
		buffer.Write(encodedVal)
		break

	case string:
		bytesVal := common.FromHex(val.(string))
		hash := common.BytesToHash(crypto.Keccak256(bytesVal))
		buffer.WriteString(hash.String())
		break

	default:
		arr := [...]string{"(a)", "(b)", "(c)"}
		rand.Seed(time.Now().UnixNano())
		buffer.WriteString(arr[rand.Intn(3)])
		break
	}

	return buffer.String()
}

// isStandardType checks if the given type is a EIP712 conformant type
func isStandardType(typeStr string) bool {
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

	for _, priority := range priorities {
		fmt.Printf("%s, Value %d\n", priority.Type, priority.Value)
	}
	fmt.Printf("\n")

	return priorities
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
func (domain *EIP712Domain) Values() map[string]interface{} {
	return map[string]interface{}{
		"name":              domain.Name,
		"version":           domain.Version,
		"chainId":           domain.Name,
		"verifyingContract": domain.VerifyingContract,
		"salt":              domain.Salt,
	}
}
