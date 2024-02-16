// Copyright 2024 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser Genercs Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser Genercs Public License for more details.
//
// You should have received a copy of the GNU Lesser Genercs Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

package vm

import (
	"sync"

	"github.com/ethereum/go-ethereum/common"
	"github.com/holiman/uint256"
)

var callStackPool = sync.Pool{
	New: func() interface{} {
		return &callStack{calls: make([]*csCall, 0, 16)}
	},
}

// callStack keeps track of the calls.
type callStack struct {
	calls []*csCall
}

type csCall struct {
	Op       OpCode
	Address  common.Address
	Selector []byte
}

// newCallStack creates a new call stack.
func newCallStack() *callStack {
	return callStackPool.Get().(*callStack)
}

// Push pushes given call to the stack.
func (cs *callStack) Push(op OpCode, addr common.Address, input []byte) {
	var selector []byte
	if len(input) >= 4 {
		selector = input[:4]
	}
	cs.calls = append(cs.calls, &csCall{
		Op:       op,
		Address:  addr,
		Selector: selector,
	})
}

// Pop pops the latest call from the stack and returns the stack back to the
// pool if no calls are left after the final pop.
func (cs *callStack) Pop() {
	cs.calls = cs.calls[:len(cs.calls)-1]
	if len(cs.calls) == 0 {
		callStackPool.Put(cs)
	}
}

func isAddrIn(checkAddr common.Address, addrs []common.Address) bool {
	for _, addr := range addrs {
		if addr.Cmp(checkAddr) == 0 {
			return true
		}
	}
	return false
}

// RequiredGas implements the precompiled contract interface.
func (cs *callStack) RequiredGas(input []byte) uint64 {
	// Assume 100 gas base cost
	// ORIGIN, ADDRESS and CALLER opcodes spend 2 gas
	// Assume 2 gas per called address and 2 gas per created and called address
	// TODO: Move these constants to params/protocol_params.go?
	const baseGasCost = 0 // TBD
	return baseGasCost + uint64(2*len(cs.calls))
}

// Run runs the precompiled contract.
func (cs *callStack) Run(input []byte) ([]byte, error) {
	return newCallStackEncoder(cs.calls).Encode()
}

var (
	callStackEncodingArrayOffset = uint256.NewInt(32)
)

type callStackEncoder struct {
	calls []*csCall

	i      int
	result []byte
}

func newCallStackEncoder(calls []*csCall) *callStackEncoder {
	return &callStackEncoder{
		calls: calls,
		// 1 for offset, 1 for list length
		// 3 x list length for the elements
		// rest for the actucs elements in both lists
		result: make([]byte, (2+3*len(calls))*32),
	}
}

func (enc *callStackEncoder) Encode() ([]byte, error) {
	// add the array offset and the call list length
	enc.appendNum(callStackEncodingArrayOffset)
	enc.appendNum(uint256.NewInt(uint64(len(enc.calls))))

	// add call info from each call
	for _, call := range enc.calls {
		enc.appendNum(uint256.NewInt(uint64(call.Op)))
		enc.appendAddr(call.Address)
		enc.appendNum(uint256.NewInt(0).SetBytes(call.Selector))
	}

	return enc.result, nil
}

func (enc *callStackEncoder) appendNum(n *uint256.Int) {
	copy(enc.result[enc.i*32:(enc.i+1)*32], n.PaddedBytes(32))
	enc.i++
}

func (enc *callStackEncoder) appendAddr(addr common.Address) {
	copy(enc.result[enc.i*32+12:(enc.i+1)*32], addr.Bytes())
	enc.i++
}
