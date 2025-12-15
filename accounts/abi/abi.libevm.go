// Copyright 2025 the libevm authors.
//
// The libevm additions to go-ethereum are free software: you can redistribute
// them and/or modify them under the terms of the GNU Lesser General Public License
// as published by the Free Software Foundation, either version 3 of the License,
// or (at your option) any later version.
//
// The libevm additions are distributed in the hope that they will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the GNU Lesser
// General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see
// <http://www.gnu.org/licenses/>.

package abi

import (
	"fmt"

	"github.com/ava-labs/libevm/common"
)

// PackEvent packs the given `args` to conform with the ABI for the specified
// event. Arguments MUST match the order specified in the event ABI. Indexed
// arguments are returned as topics (the slice of which MAY be nil),as described
// in the [Solidity docs], while the rest are packed into `data`.
//
// Struct, slice, and array arguments are not supported except for `[]byte`.
//
// [Solidity docs]:
// https://docs.soliditylang.org/en/latest/abi-spec.html#encoding-of-indexed-event-parameters
func (abi ABI) PackEvent(name string, args ...any) (topics []common.Hash, data []byte, _ error) {
	event, ok := abi.Events[name]
	if !ok {
		return nil, nil, fmt.Errorf("event %q not found", name)
	}
	if got, want := len(args), len(event.Inputs); got != want {
		return nil, nil, fmt.Errorf("event %q received %d inputs; expecting %d", name, got, want)
	}

	var (
		indexed  []any
		packed   []any
		packArgs Arguments
	)
	for i, arg := range args {
		if inp := event.Inputs[i]; inp.Indexed {
			indexed = append(indexed, arg)
		} else {
			packed = append(packed, arg)
			packArgs = append(packArgs, inp)
		}
	}

	topics, err := makeTopics1D(indexed)
	if err != nil {
		return nil, nil, err
	}
	if !event.Anonymous {
		topics = append([]common.Hash{event.ID}, topics...)
	}

	data, err = packArgs.Pack(packed...)
	if err != nil {
		return nil, nil, err
	}

	return topics, data, nil
}

func makeTopics1D(a []any) ([]common.Hash, error) {
	t, err := MakeTopics(a)
	if err != nil {
		return nil, err
	}
	return t[0], nil
}

// PackOutput packs the given `args` to conform with the ABI for the specified
// method's output.
func (abi ABI) PackOutput(method string, args ...any) ([]byte, error) {
	m, ok := abi.Methods[method]
	if !ok {
		return nil, fmt.Errorf("method %q not found", method)
	}
	return m.Outputs.Pack(args...)
}

// UnpackInputIntoInterface is equivalent to [ABI.UnpackIntoInterface], with all
// the same caveats, except that it treats `data` as:
//
//  1. Input when handling a method; or
//  2. Unindexed data when handling an event.
func (abi ABI) UnpackInputIntoInterface(v any, methodOrEventName string, data []byte) error {
	in, err := abi.methodOrEventInputs(methodOrEventName)
	if err != nil {
		return err
	}
	unpacked, err := in.Unpack(data)
	if err != nil {
		return err
	}
	return in.Copy(v, unpacked)
}

func (abi ABI) methodOrEventInputs(name string) (Arguments, error) {
	if m, ok := abi.Methods[name]; ok {
		return m.Inputs, nil
	}
	if ev, ok := abi.Events[name]; ok {
		return ev.Inputs, nil
	}
	return nil, fmt.Errorf("no method nor event %q", name)
}
