package abi

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// normalizeArgument converts ArgumentMarshaling to a JSON-compatible map.
// Auto-generated parameter names like "param0", "param1" are converted to empty strings.
func normalizeArgument(arg ArgumentMarshaling, isEvent bool) map[string]interface{} {
	name := arg.Name
	if strings.HasPrefix(name, "param") && len(name) > 5 {
		isParamN := true
		for _, c := range name[5:] {
			if c < '0' || c > '9' {
				isParamN = false
				break
			}
		}
		if isParamN {
			name = ""
		}
	}

	result := map[string]interface{}{
		"name": name,
		"type": arg.Type,
	}
	if arg.InternalType != "" {
		result["internalType"] = arg.InternalType
	}
	if len(arg.Components) > 0 {
		components := make([]map[string]interface{}, len(arg.Components))
		for i, comp := range arg.Components {
			components[i] = normalizeArgument(comp, isEvent)
		}
		result["components"] = components
	}
	if isEvent {
		result["indexed"] = arg.Indexed
	}
	return result
}

// parseHumanReadableABIArray processes multiple human-readable ABI signatures
// and returns a JSON array. Comments and empty lines are skipped.
func parseHumanReadableABIArray(signatures []string) ([]byte, error) {
	var results []map[string]interface{}
	for _, sig := range signatures {
		sig = skipWhitespace(sig)
		if sig == "" || strings.HasPrefix(sig, "//") {
			continue
		}
		if strings.HasPrefix(sig, "struct ") {
			continue
		}
		result, err := ParseHumanReadableABI(sig)
		if err != nil {
			return nil, err
		}

		resultType := result["type"]
		normalized := map[string]interface{}{
			"type": resultType,
		}
		isEvent := resultType == "event"
		isFunction := resultType == "function"

		if name, ok := result["name"]; ok {
			normalized["name"] = name
		}
		if inputs, ok := result["inputs"].([]ArgumentMarshaling); ok {
			normInputs := make([]map[string]interface{}, len(inputs))
			for i, inp := range inputs {
				normInputs[i] = normalizeArgument(inp, isEvent)
			}
			normalized["inputs"] = normInputs
		}
		if outputs, ok := result["outputs"].([]ArgumentMarshaling); ok {
			normOutputs := make([]map[string]interface{}, len(outputs))
			for i, out := range outputs {
				normOutputs[i] = normalizeArgument(out, false)
			}
			normalized["outputs"] = normOutputs
		} else if isFunction {
			normalized["outputs"] = []map[string]interface{}{}
		}
		if stateMutability, ok := result["stateMutability"]; ok {
			normalized["stateMutability"] = stateMutability
		}
		if anonymous, ok := result["anonymous"]; ok {
			normalized["anonymous"] = anonymous
		}

		results = append(results, normalized)
	}
	return json.Marshal(results)
}

func TestParseHumanReadableABI(t *testing.T) {
	tests := []struct {
		name     string
		input    []string
		expected string
		hasError bool
	}{
		{
			name:  "simple function",
			input: []string{"function transfer(address to, uint256 amount)"},
			expected: `[
				{
					"type": "function",
					"name": "transfer",
					"inputs": [
						{"name": "to", "type": "address"},
						{"name": "amount", "type": "uint256"}
					],
					"outputs": [],
					"stateMutability": "nonpayable"
				}
			]`,
		},
		{
			name:  "function with view and returns",
			input: []string{"function balanceOf(address account) view returns (uint256)"},
			expected: `[
				{
					"type": "function",
					"name": "balanceOf",
					"inputs": [
						{"name": "account", "type": "address"}
					],
					"outputs": [
						{"name": "", "type": "uint256"}
					],
					"stateMutability": "view"
				}
			]`,
		},
		{
			name:  "function with payable",
			input: []string{"function deposit() payable"},
			expected: `[
				{
					"type": "function",
					"name": "deposit",
					"inputs": [],
					"outputs": [],
					"stateMutability": "payable"
				}
			]`,
		},
		{
			name:  "event with indexed parameters",
			input: []string{"event Transfer(address indexed from, address indexed to, uint256 value)"},
			expected: `[
				{
					"type": "event",
					"name": "Transfer",
					"inputs": [
						{"name": "from", "type": "address", "indexed": true},
						{"name": "to", "type": "address", "indexed": true},
						{"name": "value", "type": "uint256", "indexed": false}
					],
					"anonymous": false
				}
			]`,
		},
		{
			name: "multiple functions",
			input: []string{
				"function transfer(address to, uint256 amount)",
				"function balanceOf(address account) view returns (uint256)",
			},
			expected: `[
				{
					"type": "function",
					"name": "transfer",
					"inputs": [
						{"name": "to", "type": "address"},
						{"name": "amount", "type": "uint256"}
					],
					"outputs": [],
					"stateMutability": "nonpayable"
				},
				{
					"type": "function",
					"name": "balanceOf",
					"inputs": [
						{"name": "account", "type": "address"}
					],
					"outputs": [
						{"name": "", "type": "uint256"}
					],
					"stateMutability": "view"
				}
			]`,
		},
		{
			name:  "function with arrays",
			input: []string{"function batchTransfer(address[] recipients, uint256[] amounts)"},
			expected: `[
				{
					"type": "function",
					"name": "batchTransfer",
					"inputs": [
						{"name": "recipients", "type": "address[]"},
						{"name": "amounts", "type": "uint256[]"}
					],
					"outputs": [],
					"stateMutability": "nonpayable"
				}
			]`,
		},
		{
			name:  "function with fixed arrays",
			input: []string{"function getBalances(address[10] accounts) view returns (uint256[10])"},
			expected: `[
				{
					"type": "function",
					"name": "getBalances",
					"inputs": [
						{"name": "accounts", "type": "address[10]"}
					],
					"outputs": [
						{"name": "", "type": "uint256[10]"}
					],
					"stateMutability": "view"
				}
			]`,
		},
		{
			name:  "function with bytes types",
			input: []string{"function setData(bytes32 key, bytes value)"},
			expected: `[
				{
					"type": "function",
					"name": "setData",
					"inputs": [
						{"name": "key", "type": "bytes32"},
						{"name": "value", "type": "bytes"}
					],
					"outputs": [],
					"stateMutability": "nonpayable"
				}
			]`,
		},
		{
			name:  "function with small integers",
			input: []string{"function smallIntegers(uint8 u8, uint16 u16, uint32 u32, uint64 u64, int8 i8, int16 i16, int32 i32, int64 i64)"},
			expected: `[
				{
					"type": "function",
					"name": "smallIntegers",
					"inputs": [
						{"name": "u8", "type": "uint8"},
						{"name": "u16", "type": "uint16"},
						{"name": "u32", "type": "uint32"},
						{"name": "u64", "type": "uint64"},
						{"name": "i8", "type": "int8"},
						{"name": "i16", "type": "int16"},
						{"name": "i32", "type": "int32"},
						{"name": "i64", "type": "int64"}
					],
					"outputs": [],
					"stateMutability": "nonpayable"
				}
			]`,
		},
		{
			name:  "function with non-standard small integers",
			input: []string{"function nonStandardIntegers(uint24 u24, uint48 u48, uint72 u72, uint96 u96, uint120 u120, int24 i24, int36 i36, int48 i48, int72 i72, int96 i96, int120 i120)"},
			expected: `[
				{
					"type": "function",
					"name": "nonStandardIntegers",
					"inputs": [
						{"name": "u24", "type": "uint24"},
						{"name": "u48", "type": "uint48"},
						{"name": "u72", "type": "uint72"},
						{"name": "u96", "type": "uint96"},
						{"name": "u120", "type": "uint120"},
						{"name": "i24", "type": "int24"},
						{"name": "i36", "type": "int36"},
						{"name": "i48", "type": "int48"},
						{"name": "i72", "type": "int72"},
						{"name": "i96", "type": "int96"},
						{"name": "i120", "type": "int120"}
					],
					"outputs": [],
					"stateMutability": "nonpayable"
				}
			]`,
		},
		{
			name: "comments and empty lines",
			input: []string{
				"// This is a comment",
				"",
				"function transfer(address to, uint256 amount)",
				"",
				"// Another comment",
				"function balanceOf(address account) view returns (uint256)",
			},
			expected: `[
				{
					"type": "function",
					"name": "transfer",
					"inputs": [
						{"name": "to", "type": "address"},
						{"name": "amount", "type": "uint256"}
					],
					"outputs": [],
					"stateMutability": "nonpayable"
				},
				{
					"type": "function",
					"name": "balanceOf",
					"inputs": [
						{"name": "account", "type": "address"}
					],
					"outputs": [
						{"name": "", "type": "uint256"}
					],
					"stateMutability": "view"
				}
			]`,
		},
		{
			name: "function with nested dynamic arrays",
			input: []string{
				"function processNestedArrays(uint256[][] matrix, address[][2][] deepArray)",
			},
			expected: `[
				{
					"type": "function",
					"name": "processNestedArrays",
					"inputs": [
						{"name": "matrix", "type": "uint256[][]"},
						{"name": "deepArray", "type": "address[][2][]"}
					],
					"outputs": [],
					"stateMutability": "nonpayable"
				}
			]`,
		},
		{
			name: "function with mixed fixed and dynamic arrays",
			input: []string{
				"function processMixedArrays(uint256[5] fixedArray, address[] dynamicArray, bytes32[3][] fixedDynamicArray)",
			},
			expected: `[
				{
					"type": "function",
					"name": "processMixedArrays",
					"inputs": [
						{"name": "fixedArray", "type": "uint256[5]"},
						{"name": "dynamicArray", "type": "address[]"},
						{"name": "fixedDynamicArray", "type": "bytes32[3][]"}
					],
					"outputs": [],
					"stateMutability": "nonpayable"
				}
			]`,
		},
		{
			name: "function with deeply nested mixed arrays",
			input: []string{
				"function deepNestedArrays(uint256[][] complexArray, address[][] mixedArray)",
			},
			expected: `[
				{
					"type": "function",
					"name": "deepNestedArrays",
					"inputs": [
						{"name": "complexArray", "type": "uint256[][]"},
						{"name": "mixedArray", "type": "address[][]"}
					],
					"outputs": [],
					"stateMutability": "nonpayable"
				}
			]`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parseHumanReadableABIArray(tt.input)
			if tt.hasError {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)

			var expectedJSON interface{}
			err = json.Unmarshal([]byte(tt.expected), &expectedJSON)
			require.NoError(t, err)

			var actualJSON interface{}
			err = json.Unmarshal(result, &actualJSON)
			require.NoError(t, err)

			require.Equal(t, expectedJSON, actualJSON)
		})
	}
}

func TestParseHumanReadableABI_Errors(t *testing.T) {
	tests := []struct {
		name  string
		input []string
	}{
		{
			name:  "invalid function format",
			input: []string{"function invalid format"},
		},
		{
			name:  "invalid array size",
			input: []string{"function test(uint256[invalid] arr) returns (bool)"},
		},
		{
			name:  "unrecognized line",
			input: []string{"invalid line format"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := parseHumanReadableABIArray(tt.input)
			require.Error(t, err)
		})
	}
}
