// Copyright 2018 The go-ethereum Authors
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

package bind

import (
	"encoding/binary"
	"errors"
	"fmt"
	"math/big"
	"reflect"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

// makeTopics converts a filter query argument list into a filter topic set.
func makeTopics(query ...[]interface{}) ([][]common.Hash, error) {
	topics := make([][]common.Hash, len(query))
	for i, filter := range query {
		for _, rule := range filter {
			var topic common.Hash

			// Try to generate the topic based on simple types
			switch rule := rule.(type) {
			case common.Hash:
				copy(topic[:], rule[:])
			case common.Address:
				copy(topic[common.HashLength-common.AddressLength:], rule[:])
			case *big.Int:
				blob := rule.Bytes()
				copy(topic[common.HashLength-len(blob):], blob)
			case bool:
				if rule {
					topic[common.HashLength-1] = 1
				}
			case int8:
				blob := big.NewInt(int64(rule)).Bytes()
				copy(topic[common.HashLength-len(blob):], blob)
			case int16:
				blob := big.NewInt(int64(rule)).Bytes()
				copy(topic[common.HashLength-len(blob):], blob)
			case int32:
				blob := big.NewInt(int64(rule)).Bytes()
				copy(topic[common.HashLength-len(blob):], blob)
			case int64:
				blob := big.NewInt(rule).Bytes()
				copy(topic[common.HashLength-len(blob):], blob)
			case uint8:
				blob := new(big.Int).SetUint64(uint64(rule)).Bytes()
				copy(topic[common.HashLength-len(blob):], blob)
			case uint16:
				blob := new(big.Int).SetUint64(uint64(rule)).Bytes()
				copy(topic[common.HashLength-len(blob):], blob)
			case uint32:
				blob := new(big.Int).SetUint64(uint64(rule)).Bytes()
				copy(topic[common.HashLength-len(blob):], blob)
			case uint64:
				blob := new(big.Int).SetUint64(rule).Bytes()
				copy(topic[common.HashLength-len(blob):], blob)
			case string:
				hash := crypto.Keccak256Hash([]byte(rule))
				copy(topic[:], hash[:])
			case []byte:
				hash := crypto.Keccak256Hash(rule)
				copy(topic[:], hash[:])

			default:
				// Attempt to generate the topic from funky types
				val := reflect.ValueOf(rule)

				switch {

				// static byte array
				case val.Kind() == reflect.Array && reflect.TypeOf(rule).Elem().Kind() == reflect.Uint8:
					reflect.Copy(reflect.ValueOf(topic[:val.Len()]), val)

				default:
					return nil, fmt.Errorf("unsupported indexed type: %T", rule)
				}
			}
			topics[i] = append(topics[i], topic)
		}
	}
	return topics, nil
}

// Big batch of reflect types for topic reconstruction.
var (
	reflectHash    = reflect.TypeOf(common.Hash{})
	reflectAddress = reflect.TypeOf(common.Address{})
	reflectBigInt  = reflect.TypeOf(new(big.Int))
)

// parseTopics converts the indexed topic fields into actual log field values.
//
// Note, dynamic types cannot be reconstructed since they get mapped to Keccak256
// hashes as the topic value!
func parseTopics(out interface{}, fields abi.Arguments, topics []common.Hash) error {
	// Sanity check that the fields and topics match up
	if len(fields) != len(topics) {
		return errors.New("topic/field count mismatch")
	}
	// Iterate over all the fields and reconstruct them from topics
	for _, arg := range fields {
		if !arg.Indexed {
			return errors.New("non-indexed field in topic reconstruction")
		}
		field := reflect.ValueOf(out).Elem().FieldByName(capitalise(arg.Name))

		// Try to parse the topic back into the fields based on primitive types
		switch field.Kind() {
		case reflect.Bool:
			if topics[0][common.HashLength-1] == 1 {
				field.Set(reflect.ValueOf(true))
			}
		case reflect.Int8:
			num := new(big.Int).SetBytes(topics[0][:])
			field.Set(reflect.ValueOf(int8(num.Int64())))

		case reflect.Int16:
			num := new(big.Int).SetBytes(topics[0][:])
			field.Set(reflect.ValueOf(int16(num.Int64())))

		case reflect.Int32:
			num := new(big.Int).SetBytes(topics[0][:])
			field.Set(reflect.ValueOf(int32(num.Int64())))

		case reflect.Int64:
			num := new(big.Int).SetBytes(topics[0][:])
			field.Set(reflect.ValueOf(num.Int64()))

		case reflect.Uint8:
			num := new(big.Int).SetBytes(topics[0][:])
			field.Set(reflect.ValueOf(uint8(num.Uint64())))

		case reflect.Uint16:
			num := new(big.Int).SetBytes(topics[0][:])
			field.Set(reflect.ValueOf(uint16(num.Uint64())))

		case reflect.Uint32:
			num := new(big.Int).SetBytes(topics[0][:])
			field.Set(reflect.ValueOf(uint32(num.Uint64())))

		case reflect.Uint64:
			num := new(big.Int).SetBytes(topics[0][:])
			field.Set(reflect.ValueOf(num.Uint64()))

		default:
			// Ran out of plain primitive types, try custom types
			switch field.Type() {
			case reflectHash: // Also covers all dynamic types
				field.Set(reflect.ValueOf(topics[0]))

			case reflectAddress:
				var addr common.Address
				copy(addr[:], topics[0][common.HashLength-common.AddressLength:])
				field.Set(reflect.ValueOf(addr))

			case reflectBigInt:
				num := new(big.Int).SetBytes(topics[0][:])
				field.Set(reflect.ValueOf(num))

			default:
				// Ran out of custom types, try the crazies
				switch {

				// static byte array
				case arg.Type.T == abi.FixedBytesTy:
					reflect.Copy(field, reflect.ValueOf(topics[0][:arg.Type.Size]))

				default:
					return fmt.Errorf("unsupported indexed type: %v", arg.Type)
				}
			}
		}
		topics = topics[1:]
	}
	return nil
}

// parseTopicsIntoMap converts the indexed topic field-value pairs into map key-value pairs
func parseTopicsIntoMap(out map[string]interface{}, fields abi.Arguments, topics []common.Hash) error {
	// Sanity check that the fields and topics match up
	if len(fields) != len(topics) {
		return errors.New("topic/field count mismatch")
	}
	// Iterate over all the fields and reconstruct them from topics
	for _, arg := range fields {
		if !arg.Indexed {
			return errors.New("non-indexed field in topic reconstruction")
		}

		switch arg.Type.T {
		case abi.BoolTy:
			out[arg.Name] = topics[0][common.HashLength-1] == 1
		case abi.IntTy, abi.UintTy:
			num := new(big.Int).SetBytes(topics[0][:])
			out[arg.Name] = num
		case abi.AddressTy:
			var addr common.Address
			copy(addr[:], topics[0][common.HashLength-common.AddressLength:])
			out[arg.Name] = addr
		case abi.HashTy:
			out[arg.Name] = topics[0]
		case abi.FixedBytesTy:
			out[arg.Name] = topics[0][:]
		case abi.StringTy, abi.BytesTy, abi.SliceTy, abi.ArrayTy:
			// Array types (including strings and bytes) have their keccak256 hashes stored in the topic- not a hash
			// whose bytes can be decoded to the actual value- so the best we can do is retrieve that hash
			out[arg.Name] = topics[0]
		case abi.FunctionTy:
			if garbage := binary.BigEndian.Uint64(topics[0][0:8]); garbage != 0 {
				return fmt.Errorf("bind: got improperly encoded function type, got %v", topics[0].Bytes())
			}
			var tmp [24]byte
			copy(tmp[:], topics[0][8:32])
			out[arg.Name] = tmp
		default: // Not handling tuples
			return fmt.Errorf("unsupported indexed type: %v", arg.Type)
		}

		topics = topics[1:]
	}

	return nil
}
