// Copyright 2017 The go-ethereum Authors
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

package vm

import (
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/params"
	"github.com/holiman/uint256"
)

func TestOpChainID(t *testing.T) {
	pc := uint64(1)
	evm := EVM{}
	chainID := big.NewInt(1)
	chainConfig := params.ChainConfig{ChainID: chainID}
	evm.chainConfig = &chainConfig
	interpreter := EVMInterpreter{evm: &evm}
	stack := Stack{}
	scope := ScopeContext{Stack: &stack}
	_, ret2 := opChainID(&pc, &interpreter, &scope)
	if ret2 != nil || len(stack.data) != 1 || stack.data[0] != *uint256.NewInt(1) {
		t.Errorf("opChainID not successful")
	}
}

func TestOpBaseFee(t *testing.T) {
	pc := uint64(1)
	baseFee := big.NewInt(1)
	context := BlockContext{BaseFee: baseFee}
	evm := EVM{Context: context}
	interpreter := EVMInterpreter{evm: &evm}
	stack := Stack{}
	scope := ScopeContext{Stack: &stack}
	_, ret2 := opBaseFee(&pc, &interpreter, &scope)
	if ret2 != nil || len(stack.data) != 1 || stack.data[0] != *uint256.NewInt(1) {
		t.Errorf("opBaseFee not successful")
	}
}

func TestOpBlobBaseFee(t *testing.T) {
	pc := uint64(1)
	blobBaseFee := big.NewInt(1)
	context := BlockContext{BlobBaseFee: blobBaseFee}
	evm := EVM{Context: context}
	interpreter := EVMInterpreter{evm: &evm}
	stack := Stack{}
	scope := ScopeContext{Stack: &stack}
	_, ret2 := opBlobBaseFee(&pc, &interpreter, &scope)
	if ret2 != nil || len(stack.data) != 1 || stack.data[0] != *uint256.NewInt(1) {
		t.Errorf("opBlobBaseFee not successful")
	}
}

func TestOpPush1EIP4762(t *testing.T) {
	pc := uint64(1)
	context := BlockContext{}
	evm := EVM{Context: context}
	interpreter := EVMInterpreter{evm: &evm}
	contract := Contract{Code: []byte{0, 1, 2}}
	stack := Stack{}
	scope := ScopeContext{Contract: &contract, Stack: &stack}
	_, ret2 := opPush1EIP4762(&pc, &interpreter, &scope)
	if ret2 != nil || len(stack.data) != 1 || stack.data[0] != *uint256.NewInt(2) {
		t.Errorf("opPush1EIP4762 not successful")
	}
}

func TestMakePushEIP4762(t *testing.T) {
	ef := makePushEIP4762(4, 8)

	pc := uint64(1)
	context := BlockContext{}
	evm := EVM{Context: context}
	interpreter := EVMInterpreter{evm: &evm}
	contract := Contract{Code: []byte{0, 1, 2}, IsDeployment: true}
	stack := Stack{}
	scope := ScopeContext{Contract: &contract, Stack: &stack}

	_, ret2 := ef(&pc, &interpreter, &scope)
	if ret2 != nil || len(stack.data) != 1 || pc != uint64(5) {
		t.Errorf("makePushEIP4762 not successful")
	}
}
