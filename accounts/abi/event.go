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
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

// Event is an event potentially triggered by the EVM's LOG mechanism. The Event
// holds type information (inputs) about the yielded output. Anonymous events
// don't get the signature canonical representation as the first LOG topic.
type Event struct {
	// Name is the event name used for internal representation. It's derived from
	// the raw name and a suffix will be added in the case of a event overload.
	//
	// e.g.
	// There are two events have same name:
	// * foo(int,int)
	// * foo(uint,uint)
	// The event name of the first one wll be resolved as foo while the second one
	// will be resolved as foo0.
	Name string
	// RawName is the raw event name parsed from ABI.
	RawName   string
	Anonymous bool
	Inputs    Arguments
}

func (e Event) String() string {
	inputs := make([]string, len(e.Inputs))
	for i, input := range e.Inputs {
		inputs[i] = fmt.Sprintf("%v %v", input.Type, input.Name)
		if input.Indexed {
			inputs[i] = fmt.Sprintf("%v indexed %v", input.Type, input.Name)
		}
	}
	return fmt.Sprintf("event %v(%v)", e.RawName, strings.Join(inputs, ", "))
}

// Sig returns the event string signature according to the ABI spec.
//
// Example
//
//     event foo(uint32 a, int b) = "foo(uint32,int256)"
//
// Please note that "int" is substitute for its canonical representation "int256"
func (e Event) Sig() string {
	types := make([]string, len(e.Inputs))
	for i, input := range e.Inputs {
		types[i] = input.Type.String()
	}
	return fmt.Sprintf("%v(%v)", e.RawName, strings.Join(types, ","))
}

// ID returns the canonical representation of the event's signature used by the
// abi definition to identify event names and types.
func (e Event) ID() common.Hash {
	return common.BytesToHash(crypto.Keccak256([]byte(e.Sig())))
}
