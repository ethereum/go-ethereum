// Copyright 2015 The go-ethereum Authors
// This file is part of go-ethereum.
//
// go-ethereum is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// go-ethereum is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with go-ethereum.  If not, see <http://www.gnu.org/licenses/>.

package api

import (
	"encoding/json"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/rpc/shared"
)

type WhisperMessageArgs struct {
	Payload  string
	To       string
	From     string
	Topics   []string
	Priority uint32
	Ttl      uint32
}

func (args *WhisperMessageArgs) UnmarshalJSON(b []byte) (err error) {
	var obj []struct {
		Payload  string
		To       string
		From     string
		Topics   []string
		Priority interface{}
		Ttl      interface{}
	}

	if err = json.Unmarshal(b, &obj); err != nil {
		return shared.NewDecodeParamError(err.Error())
	}

	if len(obj) < 1 {
		return shared.NewInsufficientParamsError(len(obj), 1)
	}
	args.Payload = obj[0].Payload
	args.To = obj[0].To
	args.From = obj[0].From
	args.Topics = obj[0].Topics

	var num *big.Int
	if num, err = numString(obj[0].Priority); err != nil {
		return err
	}
	args.Priority = uint32(num.Int64())

	if num, err = numString(obj[0].Ttl); err != nil {
		return err
	}
	args.Ttl = uint32(num.Int64())

	return nil
}

type WhisperIdentityArgs struct {
	Identity string
}

func (args *WhisperIdentityArgs) UnmarshalJSON(b []byte) (err error) {
	var obj []interface{}
	if err := json.Unmarshal(b, &obj); err != nil {
		return shared.NewDecodeParamError(err.Error())
	}

	if len(obj) < 1 {
		return shared.NewInsufficientParamsError(len(obj), 1)
	}

	argstr, ok := obj[0].(string)
	if !ok {
		return shared.NewInvalidTypeError("arg0", "not a string")
	}

	args.Identity = argstr

	return nil
}

type WhisperFilterArgs struct {
	To     string
	From   string
	Topics [][]string
}

// UnmarshalJSON implements the json.Unmarshaler interface, invoked to convert a
// JSON message blob into a WhisperFilterArgs structure.
func (args *WhisperFilterArgs) UnmarshalJSON(b []byte) (err error) {
	// Unmarshal the JSON message and sanity check
	var obj []struct {
		To     interface{} `json:"to"`
		From   interface{} `json:"from"`
		Topics interface{} `json:"topics"`
	}
	if err := json.Unmarshal(b, &obj); err != nil {
		return shared.NewDecodeParamError(err.Error())
	}
	if len(obj) < 1 {
		return shared.NewInsufficientParamsError(len(obj), 1)
	}
	// Retrieve the simple data contents of the filter arguments
	if obj[0].To == nil {
		args.To = ""
	} else {
		argstr, ok := obj[0].To.(string)
		if !ok {
			return shared.NewInvalidTypeError("to", "is not a string")
		}
		args.To = argstr
	}
	if obj[0].From == nil {
		args.From = ""
	} else {
		argstr, ok := obj[0].From.(string)
		if !ok {
			return shared.NewInvalidTypeError("from", "is not a string")
		}
		args.From = argstr
	}
	// Construct the nested topic array
	if obj[0].Topics != nil {
		// Make sure we have an actual topic array
		list, ok := obj[0].Topics.([]interface{})
		if !ok {
			return shared.NewInvalidTypeError("topics", "is not an array")
		}
		// Iterate over each topic and handle nil, string or array
		topics := make([][]string, len(list))
		for idx, field := range list {
			switch value := field.(type) {
			case nil:
				topics[idx] = []string{}

			case string:
				topics[idx] = []string{value}

			case []interface{}:
				topics[idx] = make([]string, len(value))
				for i, nested := range value {
					switch value := nested.(type) {
					case nil:
						topics[idx][i] = ""

					case string:
						topics[idx][i] = value

					default:
						return shared.NewInvalidTypeError(fmt.Sprintf("topic[%d][%d]", idx, i), "is not a string")
					}
				}
			default:
				return shared.NewInvalidTypeError(fmt.Sprintf("topic[%d]", idx), "not a string or array")
			}
		}
		args.Topics = topics
	}
	return nil
}
