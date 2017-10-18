// Copyright 2015 The go-ethereum Authors
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
	"encoding/json"
	"fmt"
	"io"
)

// The ABI holds information about a contract's context and available
// invokable methods. It will allow you to type check function calls and
// packs data accordingly.
type ABI struct {
	Constructor Method
	Methods     map[string]Method
	Events      map[string]Event
}

// JSON returns a parsed ABI interface and error if it failed.
func JSON(reader io.Reader) (ABI, error) {
	dec := json.NewDecoder(reader)

	var abi ABI
	if err := dec.Decode(&abi); err != nil {
		return ABI{}, err
	}

	return abi, nil
}

// Pack the given method name to conform the ABI. Method call's data
// will consist of method_id, args0, arg1, ... argN. Method id consists
// of 4 bytes and arguments are all 32 bytes.
// Method ids are created from the first 4 bytes of the hash of the
// methods string signature. (signature = baz(uint32,string32))
func (abi ABI) Pack(name string, args ...interface{}) ([]byte, error) {
	// Fetch the ABI of the requested method
	var method Method

	if name == "" {
		method = abi.Constructor
	} else {
		m, exist := abi.Methods[name]
		if !exist {
			return nil, fmt.Errorf("method '%s' not found", name)
		}
		method = m
	}
	arguments, err := method.pack(args...)
	if err != nil {
		return nil, err
	}
	// Pack up the method ID too if not a constructor and return
	if name == "" {
		return arguments, nil
	}
	return append(method.Id(), arguments...), nil
}

// Unpack output in v according to the abi specification
func (abi ABI) Unpack(v interface{}, name string, output []byte) (err error) {
	if err = bytesAreProper(output); err != nil {
		return err
	}
	// since there can't be naming collisions with contracts and events,
	// we need to decide whether we're calling a method or an event
	var unpack unpacker
	if method, ok := abi.Methods[name]; ok {
		unpack = method
	} else if event, ok := abi.Events[name]; ok {
		unpack = event
	} else {
		return fmt.Errorf("abi: could not locate named method or event.")
	}

	// requires a struct to unpack into for a tuple return...
	if unpack.isTupleReturn() {
		return unpack.tupleUnpack(v, output)
	}
	return unpack.singleUnpack(v, output)
}


// Unpack output in v according to the abi specification
func (abi ABI) UnpackEvent(v interface{}, name string, output []byte) error {
	var event = abi.Events[name]

	if len(output) == 0 {
		return fmt.Errorf("abi: unmarshalling empty output")
	}

	// make sure the passed value is a pointer
	valueOf := reflect.ValueOf(v)
	if reflect.Ptr != valueOf.Kind() {
		return fmt.Errorf("abi: Unpack(non-pointer %T)", v)
	}

	var (
		value = valueOf.Elem()
		typ   = value.Type()
	)

	if len(event.Inputs) > 1 {
		switch value.Kind() {
		// struct will match named return values to the struct's field
		// names
		case reflect.Struct:
			for i := 0; i < len(event.Inputs); i++ {
				marshalledValue, err := toGoType(i, event.Inputs[i], output)
				if err != nil {
					return err
				}
				reflectValue := reflect.ValueOf(marshalledValue)

				for j := 0; j < typ.NumField(); j++ {
					field := typ.Field(j)
					// TODO read tags: `abi:"fieldName"`
					if field.Name == strings.ToUpper(event.Inputs[i].Name[:1])+event.Inputs[i].Name[1:] {
						if err := set(value.Field(j), reflectValue, event.Inputs[i]); err != nil {
							return err
						}
					}
				}
			}
		case reflect.Slice:
			if !value.Type().AssignableTo(r_interSlice) {
				return fmt.Errorf("abi: cannot marshal tuple in to slice %T (only []interface{} is supported)", v)
			}

			// if the slice already contains values, set those instead of the interface slice itself.
			if value.Len() > 0 {
				if len(event.Inputs) > value.Len() {
					return fmt.Errorf("abi: cannot marshal in to slices of unequal size (require: %v, got: %v)", len(event.Inputs), value.Len())
				}

				for i := 0; i < len(event.Inputs); i++ {
					marshalledValue, err := toGoType(i, event.Inputs[i], output)
					if err != nil {
						return err
					}
					reflectValue := reflect.ValueOf(marshalledValue)
					if err := set(value.Index(i).Elem(), reflectValue, event.Inputs[i]); err != nil {
						return err
					}
				}
				return nil
			}

			// create a new slice and start appending the unmarshalled
			// values to the new interface slice.
			z := reflect.MakeSlice(typ, 0, len(event.Inputs))
			for i := 0; i < len(event.Inputs); i++ {
				marshalledValue, err := toGoType(i, event.Inputs[i], output)
				if err != nil {
					return err
				}
				z = reflect.Append(z, reflect.ValueOf(marshalledValue))
			}
			value.Set(z)
		default:
			return fmt.Errorf("abi: cannot unmarshal tuple in to %v", typ)
		}

	} else {
		marshalledValue, err := toGoType(0, event.Inputs[0], output)
		if err != nil {
			return err
		}
		if err := set(value, reflect.ValueOf(marshalledValue), event.Inputs[0]); err != nil {
			return err
		}
	}

	return nil
}

func (abi *ABI) UnmarshalJSON(data []byte) error {
	var fields []struct {
		Type      string
		Name      string
		Constant  bool
		Indexed   bool
		Anonymous bool
		Inputs    []Argument
		Outputs   []Argument
	}

	if err := json.Unmarshal(data, &fields); err != nil {
		return err
	}

	abi.Methods = make(map[string]Method)
	abi.Events = make(map[string]Event)
	for _, field := range fields {
		switch field.Type {
		case "constructor":
			abi.Constructor = Method{
				Inputs: field.Inputs,
			}
		// empty defaults to function according to the abi spec
		case "function", "":
			abi.Methods[field.Name] = Method{
				Name:    field.Name,
				Const:   field.Constant,
				Inputs:  field.Inputs,
				Outputs: field.Outputs,
			}
		case "event":
			abi.Events[field.Name] = Event{
				Name:      field.Name,
				Anonymous: field.Anonymous,
				Inputs:    field.Inputs,
			}
		}
	}

	return nil
}
