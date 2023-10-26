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

package vm

import (
	"context"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/rpc"

	"github.com/ethereum/go-ethereum/params"
)

// l1Call implemented as a native contract. it executes read-only code from
// an L1 Contract in the context of the L1 contract.
type l1Call struct{}

// RequiredGas returns the gas required to execute the pre-compiled contract.
func (c *l1Call) RequiredGas(input []byte) uint64 {
	return params.L1Call
}

// Run implements
// solidity call such as:
// l1call(abi.encodePacked(address(contractAddress), abiEncodedData))
func (c *l1Call) Run(opts *TaikoRunOpts, input []byte) ([]byte, error) {
	r, err := rpc.Dial(opts.L1RPCUrl)
	if err != nil {
		return nil, err
	}

	client := ethclient.NewClient(r)

	// get first 32 bytes which should be the address of the contract
	addr := common.BytesToAddress(input[:32])

	// rest of bytes should be msg data
	msgData := input[64:]

	// use latest blockNumber for now?
	// call contract, get the response, but dont execute it on L1.
	contractResponse, err := client.CallContract(
		context.Background(),
		ethereum.CallMsg{
			To:   &addr,
			Data: msgData,
		},
		nil,
	)

	if err != nil {
		return nil, err
	}

	return contractResponse, nil
}

// l1DelegateCall implemented as a native contract.
// it executes read-only code from an L1 contract in the context of the L2
// calling contract.
type l1DelegateCall struct{}

// RequiredGas returns the gas required to execute the pre-compiled contract.
func (c *l1DelegateCall) RequiredGas(input []byte) uint64 {
	return params.L1DelegateCall
}

// Run implements
// solidity call such as:
// l1delegetecall(abi.encodePacked(address(contractAddress), abiEncodedData))
// defaults to latest block. TODO: get latest block synced from
// TaikoL2 contract, dont allow usage of L1 blocks that havent been
// synced to L2.
func (c *l1DelegateCall) Run(opts *TaikoRunOpts, input []byte) ([]byte, error) {
	snapshot := opts.StateDB.Snapshot()

	defer func() {
		opts.StateDB.RevertToSnapshot(snapshot)
	}()

	// add the L1 bytecode gotten below to this stateDB, then execute contract, then REVERT.

	r, err := rpc.Dial(opts.L1RPCUrl)
	if err != nil {
		return nil, err
	}

	client := ethclient.NewClient(r)

	// get first 32 bytes which should be the address of the contract
	addr := common.BytesToAddress(input[:32])

	// rest of bytes should be msg data
	msgData := input[32:]

	// use latest blockNumber for now?
	// call contract, get the response, but dont execute it on L1.
	l1ContractBytecode, err := client.CodeAt(
		context.Background(),
		addr,
		nil,
	)

	// load this bytecode in at an address, and execute the rest of this msg.data
	// at that address.

	if err != nil {
		return nil, err
	}

	// overwrite L2 contract with the L1 contract, at the same address.
	// it will be reverted after
	contract := NewContract(opts.Caller, AccountRef(addr), nil, 0)
	contract.SetCallCode(&addr, opts.StateDB.GetCodeHash(addr), l1ContractBytecode)
	ret, err := opts.Interpreter.Run(contract, msgData, false)
	if err != nil {
		return nil, err
	}

	return ret, nil
}
