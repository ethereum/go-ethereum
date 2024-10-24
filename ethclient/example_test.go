// Copyright 2024 The go-ethereum Authors
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

package ethclient

import (
	"context"
	"fmt"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/node"
)

var exampleNode *node.Node

// launch example server
func init() {
	config := &node.Config{
		HTTPHost: "127.0.0.1",
	}
	n, _, err := newTestBackend(config)
	if err != nil {
		panic("can't launch node: " + err.Error())
	}
	exampleNode = n
}

// Here we show how to get the error message of reverted contract call.
func ExampleRevertErrorData() {
	// First create an ethclient.Client instance.
	ctx := context.Background()
	ec, _ := DialContext(ctx, exampleNode.HTTPEndpoint())

	// Call the contract.
	// Note we expect the call to return an error.
	contract := common.HexToAddress("290f1b36649a61e369c6276f6d29463335b4400c")
	call := ethereum.CallMsg{To: &contract, Gas: 30000}
	result, err := ec.CallContract(ctx, call, nil)
	if len(result) > 0 {
		panic("got result")
	}
	if err == nil {
		panic("call did not return error")
	}

	// Extract the low-level revert data from the error.
	revertData, ok := RevertErrorData(err)
	if !ok {
		panic("unpacking revert failed")
	}
	fmt.Printf("revert: %x\n", revertData)

	// Parse the revert data to obtain the error message.
	message, err := abi.UnpackRevert(revertData)
	fmt.Println("message:", message)

	// Output:
	// revert: 08c379a00000000000000000000000000000000000000000000000000000000000000020000000000000000000000000000000000000000000000000000000000000000a75736572206572726f72
	// message: user error
}
