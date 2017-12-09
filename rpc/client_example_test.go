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

package rpc_test

import (
	"context"
	"fmt"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/rpc"
)

// In this example, our client whishes to track the latest 'block number'
// known to the server. The server supports two methods:
//
// eth_getBlockByNumber("latest", {})
//    returns the latest block object.
//
// eth_subscribe("newBlocks")
//    creates a subscription which fires block objects when new blocks arrive.

type Block struct {
	Number *big.Int
}

func ExampleClientSubscription() {
	// Connect the client.
	client, _ := rpc.Dial("ws://127.0.0.1:8485")
	subch := make(chan Block)

	// Ensure that subch receives the latest block.
	go func() {
		for i := 0; ; i++ {
			if i > 0 {
				time.Sleep(2 * time.Second)
			}
			subscribeBlocks(client, subch)
		}
	}()

	// Print events from the subscription as they arrive.
	for block := range subch {
		fmt.Println("latest block:", block.Number)
	}
}

// subscribeBlocks runs in its own goroutine and maintains
// a subscription for new blocks.
func subscribeBlocks(client *rpc.Client, subch chan Block) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Subscribe to new blocks.
	sub, err := client.EthSubscribe(ctx, subch, "newBlocks")
	if err != nil {
		fmt.Println("subscribe error:", err)
		return
	}

	// The connection is established now.
	// Update the channel with the current block.
	var lastBlock Block
	if err := client.CallContext(ctx, &lastBlock, "eth_getBlockByNumber", "latest"); err != nil {
		fmt.Println("can't get latest block:", err)
		return
	}
	subch <- lastBlock

	// The subscription will deliver events to the channel. Wait for the
	// subscription to end for any reason, then loop around to re-establish
	// the connection.
	fmt.Println("connection lost: ", <-sub.Err())
}
