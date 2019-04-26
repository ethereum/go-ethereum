// Copyright 2019 The go-ethereum Authors
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

package statediff_test

import (
	"bytes"
	"math/big"
	"math/rand"
	"reflect"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/ethereum/go-ethereum/statediff"
	"github.com/ethereum/go-ethereum/statediff/testhelpers/mocks"
)

func TestServiceLoop(t *testing.T) {
	testErrorInChainEventLoop(t)
	testErrorInBlockLoop(t)
}

var (
	eventsChannel = make(chan core.ChainEvent, 1)

	parentRoot1   = common.HexToHash("0x01")
	parentRoot2   = common.HexToHash("0x02")
	parentHeader1 = types.Header{Number: big.NewInt(rand.Int63()), Root: parentRoot1}
	parentHeader2 = types.Header{Number: big.NewInt(rand.Int63()), Root: parentRoot2}

	parentBlock1 = types.NewBlock(&parentHeader1, nil, nil, nil)
	parentBlock2 = types.NewBlock(&parentHeader2, nil, nil, nil)

	parentHash1 = parentBlock1.Hash()
	parentHash2 = parentBlock2.Hash()

	testRoot1 = common.HexToHash("0x03")
	testRoot2 = common.HexToHash("0x04")
	testRoot3 = common.HexToHash("0x04")
	header1   = types.Header{ParentHash: parentHash1, Root: testRoot1}
	header2   = types.Header{ParentHash: parentHash2, Root: testRoot2}
	header3   = types.Header{ParentHash: common.HexToHash("parent hash"), Root: testRoot3}

	testBlock1 = types.NewBlock(&header1, nil, nil, nil)
	testBlock2 = types.NewBlock(&header2, nil, nil, nil)
	testBlock3 = types.NewBlock(&header3, nil, nil, nil)

	event1 = core.ChainEvent{Block: testBlock1}
	event2 = core.ChainEvent{Block: testBlock2}
	event3 = core.ChainEvent{Block: testBlock3}
)

func testErrorInChainEventLoop(t *testing.T) {
	//the first chain event causes and error (in blockchain mock)
	builder := mocks.Builder{}
	blockChain := mocks.BlockChain{}
	service := statediff.Service{
		Builder:       &builder,
		BlockChain:    &blockChain,
		QuitChan:      make(chan bool),
		Subscriptions: make(map[rpc.ID]statediff.Subscription),
	}
	testRoot2 = common.HexToHash("0xTestRoot2")
	blockChain.SetParentBlocksToReturn([]*types.Block{parentBlock1, parentBlock2})
	blockChain.SetChainEvents([]core.ChainEvent{event1, event2, event3})
	service.Loop(eventsChannel)
	if !reflect.DeepEqual(builder.BlockHash, testBlock2.Hash()) {
		t.Error("Test failure:", t.Name())
		t.Logf("Actual does not equal expected.\nactual:%+v\nexpected: %+v", builder.BlockHash, testBlock2.Hash())
	}
	if !bytes.Equal(builder.OldStateRoot.Bytes(), parentBlock2.Root().Bytes()) {
		t.Error("Test failure:", t.Name())
		t.Logf("Actual does not equal expected.\nactual:%+v\nexpected: %+v", builder.OldStateRoot, parentBlock2.Root())
	}
	if !bytes.Equal(builder.NewStateRoot.Bytes(), testBlock2.Root().Bytes()) {
		t.Error("Test failure:", t.Name())
		t.Logf("Actual does not equal expected.\nactual:%+v\nexpected: %+v", builder.NewStateRoot, testBlock2.Root())
	}
	//look up the parent block from its hash
	expectedHashes := []common.Hash{testBlock1.ParentHash(), testBlock2.ParentHash()}
	if !reflect.DeepEqual(blockChain.ParentHashesLookedUp, expectedHashes) {
		t.Error("Test failure:", t.Name())
		t.Logf("Actual does not equal expected.\nactual:%+v\nexpected: %+v", blockChain.ParentHashesLookedUp, expectedHashes)
	}
}

func testErrorInBlockLoop(t *testing.T) {
	//second block's parent block can't be found
	builder := mocks.Builder{}
	blockChain := mocks.BlockChain{}
	service := statediff.Service{
		Builder:       &builder,
		BlockChain:    &blockChain,
		QuitChan:      make(chan bool),
		Subscriptions: make(map[rpc.ID]statediff.Subscription),
	}

	blockChain.SetParentBlocksToReturn([]*types.Block{parentBlock1, nil})
	blockChain.SetChainEvents([]core.ChainEvent{event1, event2})
	service.Loop(eventsChannel)

	if !bytes.Equal(builder.BlockHash.Bytes(), testBlock1.Hash().Bytes()) {
		t.Error("Test failure:", t.Name())
		t.Logf("Actual does not equal expected.\nactual:%+v\nexpected: %+v", builder.BlockHash, testBlock1.Hash())
	}
	if !bytes.Equal(builder.OldStateRoot.Bytes(), parentBlock1.Root().Bytes()) {
		t.Error("Test failure:", t.Name())
		t.Logf("Actual does not equal expected.\nactual:%+v\nexpected: %+v", builder.OldStateRoot, parentBlock1.Root())
	}
	if !bytes.Equal(builder.NewStateRoot.Bytes(), testBlock1.Root().Bytes()) {
		t.Error("Test failure:", t.Name())
		t.Logf("Actual does not equal expected.\nactual:%+v\nexpected: %+v", builder.NewStateRoot, testBlock1.Root())
	}
}
