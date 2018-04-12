// Copyright 2016 The go-etherfact Authors
// This file is part of the go-etherfact library.
//
// The go-etherfact library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-etherfact library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-etherfact library. If not, see <http://www.gnu.org/licenses/>.

package abi

import (
	"fmt"
	"strings"

	"github.com/EtherFact-Project/go-etherfact/common"
	"github.com/EtherFact-Project/go-etherfact/crypto"
)

// Event is an event potentially triggered by the EVM's LOG mechanism. The Event
// holds type information (inputs) about the yielded output. Anonymous events
// don't get the signature canonical representation as the first LOG topic.
type Event struct {
	Name      string
	Anonymous bool
	Inputs    Arguments
}

func (event Event) String() string {
	inputs := make([]string, len(event.Inputs))
	for i, input := range event.Inputs {
		inputs[i] = fmt.Sprintf("%v %v", input.Name, input.Type)
		if input.Indexed {
			inputs[i] = fmt.Sprintf("%v indexed %v", input.Name, input.Type)
		}
	}
	return fmt.Sprintf("event %v(%v)", event.Name, strings.Join(inputs, ", "))
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
