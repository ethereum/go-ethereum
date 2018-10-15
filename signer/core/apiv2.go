package core

import (
	"bytes"
	"context"
	"encoding/hex"
	"fmt"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
	"math/big"
	"sort"
	"strings"
	"unicode"
)

type TypedData struct {
	Types       EIP712Types   `json:"types"`
	PrimaryType string        `json:"primaryType"`
	Domain      EIP712Domain  `json:"domain"`
	Message     EIP712Message `json:"message"`
}

type EIP712Types map[string][]map[string]string

type EIP712TypePriority struct {
	Type  string
	Value uint
}

type EIP712Domain struct {
	Name              string         `json:"name"`
	Version           string         `json:"version"`
	ChainId           *big.Int       `json:"chainId"`
	VerifyingContract common.Address `json:"verifyingContract"`
	Salt              hexutil.Bytes  `json:"salt"`
}

type EIP712Message map[string]interface{}

// Typed data according to EIP712
//
// hash = keccak256("\x19${byteVersion}${domainSeparator}${hashStruct(message)}")
func (api *SignerAPI) SignTypedData(ctx context.Context, addr common.MixedcaseAddress, data TypedData) (hexutil.Bytes, error) {
	if err := data.Domain.IsValid(); err != nil {
		return nil, err
	}
	if data.PrimaryType == "" {
		return nil, fmt.Errorf("primary type undefined")
	}

	domainTypes := EIP712Types{
		"EIP712Domain": data.Types["EIP712Domain"],
	}
	domainSeparator, err := hashStruct(domainTypes, data.Domain.Values(), "")
	if err != nil {
		return nil, err
	}

	delete(data.Types, "EIP712Domain")
	typedDataHash, err := hashStruct(data.Types, data.Message, data.PrimaryType)
	if err != nil {
		return nil, err
	}

	fmt.Println("domainSeparator", domainSeparator.String())
	fmt.Println("typedDataHash", typedDataHash.String())
	return common.FromHex("0xdeadbeef"), nil
}

// `encode(domainSeparator : 𝔹²⁵⁶, message : 𝕊) = "\x19\x01" ‖ domainSeparator ‖ hashStruct(message)`
func hashStruct(types EIP712Types, message EIP712Message, primaryType string) (common.Hash, error) {
	if primaryType != "" {
		if types[primaryType] == nil {
			return common.Hash{}, fmt.Errorf("primaryType specified but undefined")
		}
	}

	typeEncoding, err := encodeType(types, primaryType)
	if err != nil {
		return common.Hash{}, err
	}
	typeHash := hex.EncodeToString(crypto.Keccak256([]byte(typeEncoding)))

	dataEncoding, err := encodeData(message)
	if err != nil {
		return common.Hash{}, err
	}
	dataHash := hex.EncodeToString(crypto.Keccak256([]byte(dataEncoding)))

	var buffer bytes.Buffer
	buffer.WriteString(typeHash)
	buffer.WriteString(dataHash)
	hash := common.BytesToHash(crypto.Keccak256(buffer.Bytes()))

	return hash, nil
}

// encodeType transforms the given types into an encoding of the form
// `name ‖ "(" ‖ member₁ ‖ "," ‖ member₂ ‖ "," ‖ … ‖ memberₙ ")"`
//
// Each member is written as `type ‖ " " ‖ name` encodings cascade down and are sorted by name
func encodeType(types EIP712Types, primaryType string) (string, error) {
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
				return "", fmt.Errorf("type %s cannot reference itself", typeVal)
			}

			firstChar := []rune(typeVal)[0]
			if unicode.IsUpper(firstChar) {
				if types[typeVal] != nil {
					if !visited(typeValArr, typeVal) {
						typeValArr = append(typeValArr, typeVal)
						update(typeKey, typeVal)
					}
				} else {
					return "", fmt.Errorf("referenced type %s is undefined", typeVal)
				}
			} else {
				if !types.IsStandardType(typeVal) {
					if types[typeVal] != nil {
						return "", fmt.Errorf("Custom type %s must be capitalized", typeVal)
					} else {
						return "", fmt.Errorf("Unknown type %s", typeVal)
					}
				}
			}
		}

		typeValArr = []string{}
	}

	sortedPriorities := types.SortByPriorityAndName(priorities)
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

	return buffer.String(), nil
}

func encodeData(values EIP712Message) (string, error) {
	return "", nil
}

// Checks if the given type is a standard type accepted by EIP-712
func (types *EIP712Types) IsStandardType(typeStr string) bool {
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

// Helper function to sort types by priority and name. Priority is calculated b
// based upon the number of references.
func (types *EIP712Types) SortByPriorityAndName(input map[string]uint) []EIP712TypePriority {
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

// Check if the given domain is valid, i.e. contains at least the minimum viable keys and values
func (domain *EIP712Domain) IsValid() error {
	if domain.ChainId == big.NewInt(0) {
		return fmt.Errorf("chainId must be specified according to EIP-155")
	}

	if domain.Name == "" && domain.Version == "" && len(domain.VerifyingContract) == 0 && len(domain.Salt) == 0 {
		return fmt.Errorf("domain undefined")
	}

	return nil
}

// Helper function to return the values of a domain in the form of a golang map
func (domain *EIP712Domain) Values() map[string]interface{} {
	return map[string]interface{}{
		"name":              domain.Name,
		"version":           domain.Version,
		"chainId":           domain.Name,
		"verifyingContract": domain.VerifyingContract,
		"salt":              domain.Salt,
	}
}
