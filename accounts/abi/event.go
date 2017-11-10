// Copyright 2016 The go-ethereum Authors
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

package abi

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

// Event is an event potentially triggered by the EVM's LOG mechanism. The Event
// holds type information (inputs) about the yielded output. Anonymous events
// don't get the signature canonical representation as the first LOG topic.
type Event struct {
	Name      string
	Anonymous bool
	Inputs    []Argument
}

// Id returns the canonical representation of the event's signature used by the
// abi definition to identify event names and types.
func (e Event) Id() common.Hash {
	types := make([]string, len(e.Inputs))
	i := 0
	for _, input := range e.Inputs {
		types[i] = input.Type.String()
		i++
	}
	return common.BytesToHash(crypto.Keccak256([]byte(fmt.Sprintf("%v(%v)", e.Name, strings.Join(types, ",")))))
}

// unpacks an event return tuple into a struct of corresponding go types
//
// Unpacking can be done into a struct or a slice/array.
func (e Event) tupleUnpack(v interface{}, output []byte) error {
	// make sure the passed value is a pointer
	valueOf := reflect.ValueOf(v)
	if reflect.Ptr != valueOf.Kind() {
		return fmt.Errorf("abi: Unpack(non-pointer %T)", v)
	}

	var (
		value = valueOf.Elem()
		typ   = value.Type()
		kind  = value.Kind()
	)
	if err := requireUnpackKind(value, typ, kind, e.Inputs, true); err != nil {
		return err
	}

	// `i` counts the nonindexed arguments.
	// `j` counts the number of complex types.
	// both `i` and `j` are used to to correctly compute `data` offset.
	i, j := -1, 0
	for _, input := range e.Inputs {
		if input.Indexed {
			// Indexed arguments are not packed into data
			continue
		}
		i++
		marshalledValue, err := toGoType((i+j)*32, input.Type, output)
		if err != nil {
			return err
		}
		if input.Type.T == ArrayTy {
			// combined index ('i' + 'j') need to be adjusted only by size of array, thus
			// we need to decrement 'j' because 'i' was incremented
			j += input.Type.Size - 1
		}
		reflectValue := reflect.ValueOf(marshalledValue)

		switch kind {
		case reflect.Struct:
			for j := 0; j < typ.NumField(); j++ {
				field := typ.Field(j)
				// TODO read tags: `abi:"fieldName"`
				if field.Name == strings.ToUpper(input.Name[:1])+input.Name[1:] {
					if err := set(value.Field(j), reflectValue, input); err != nil {
						return err
					}
				}
			}
		case reflect.Slice, reflect.Array:
			v := value.Index(i)
			if err := requireAssignable(v, reflectValue); err != nil {
				return err
			}
			if err := set(v.Elem(), reflectValue, input); err != nil {
				return err
			}
		}
	}
	return nil
}

func (e Event) isTupleReturn() bool { return len(e.Inputs) > 1 }

func (e Event) singleUnpack(v interface{}, output []byte) error {
	// make sure the passed value is a pointer
	valueOf := reflect.ValueOf(v)
	if reflect.Ptr != valueOf.Kind() {
		return fmt.Errorf("abi: Unpack(non-pointer %T)", v)
	}

	if e.Inputs[0].Indexed {
		return fmt.Errorf("abi: attempting to unpack indexed variable into element")
	}

	value := valueOf.Elem()

	marshalledValue, err := toGoType(0, e.Inputs[0].Type, output)
	if err != nil {
		return err
	}
	return set(value, reflect.ValueOf(marshalledValue), e.Inputs[0])
}
