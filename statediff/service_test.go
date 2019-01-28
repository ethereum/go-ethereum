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
	"sync"
	"testing"

	"github.com/ethereum/go-ethereum/trie"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/ethereum/go-ethereum/rpc"
	statediff "github.com/ethereum/go-ethereum/statediff"
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

	parentBlock1 = types.NewBlock(&parentHeader1, nil, nil, nil, new(trie.Trie))
	parentBlock2 = types.NewBlock(&parentHeader2, nil, nil, nil, new(trie.Trie))

	parentHash1 = parentBlock1.Hash()
	parentHash2 = parentBlock2.Hash()

	testRoot1 = common.HexToHash("0x03")
	testRoot2 = common.HexToHash("0x04")
	testRoot3 = common.HexToHash("0x04")
	header1   = types.Header{ParentHash: parentHash1, Root: testRoot1, Number: big.NewInt(1)}
	header2   = types.Header{ParentHash: parentHash2, Root: testRoot2, Number: big.NewInt(2)}
	header3   = types.Header{ParentHash: common.HexToHash("parent hash"), Root: testRoot3, Number: big.NewInt(3)}

	testBlock1 = types.NewBlock(&header1, nil, nil, nil, new(trie.Trie))
	testBlock2 = types.NewBlock(&header2, nil, nil, nil, new(trie.Trie))
	testBlock3 = types.NewBlock(&header3, nil, nil, nil, new(trie.Trie))

	receiptRoot1  = common.HexToHash("0x05")
	receiptRoot2  = common.HexToHash("0x06")
	receiptRoot3  = common.HexToHash("0x07")
	testReceipts1 = []*types.Receipt{types.NewReceipt(receiptRoot1.Bytes(), false, 1000), types.NewReceipt(receiptRoot2.Bytes(), false, 2000)}
	testReceipts2 = []*types.Receipt{types.NewReceipt(receiptRoot3.Bytes(), false, 3000)}

	event1 = core.ChainEvent{Block: testBlock1}
	event2 = core.ChainEvent{Block: testBlock2}
	event3 = core.ChainEvent{Block: testBlock3}

	defaultParams = statediff.Params{
		IncludeBlock:    true,
		IncludeReceipts: true,
		IncludeTD:       true,
	}
)

func testErrorInChainEventLoop(t *testing.T) {
	//the first chain event causes and error (in blockchain mock)
	builder := mocks.Builder{}
	blockChain := mocks.BlockChain{}
	serviceQuit := make(chan bool)
	service := statediff.Service{
		Mutex:             sync.Mutex{},
		Builder:           &builder,
		BlockChain:        &blockChain,
		QuitChan:          serviceQuit,
		Subscriptions:     make(map[common.Hash]map[rpc.ID]statediff.Subscription),
		SubscriptionTypes: make(map[common.Hash]statediff.Params),
	}
	payloadChan := make(chan statediff.Payload, 2)
	quitChan := make(chan bool)
	service.Subscribe(rpc.NewID(), payloadChan, quitChan, defaultParams)
	testRoot2 = common.HexToHash("0xTestRoot2")
	blockMapping := make(map[common.Hash]*types.Block)
	blockMapping[parentBlock1.Hash()] = parentBlock1
	blockMapping[parentBlock2.Hash()] = parentBlock2
	blockChain.SetBlocksForHashes(blockMapping)
	blockChain.SetChainEvents([]core.ChainEvent{event1, event2, event3})
	blockChain.SetReceiptsForHash(testBlock1.Hash(), testReceipts1)
	blockChain.SetReceiptsForHash(testBlock2.Hash(), testReceipts2)

	payloads := make([]statediff.Payload, 0, 2)
	wg := new(sync.WaitGroup)
	go func() {
		wg.Add(1)
		for i := 0; i < 2; i++ {
			select {
			case payload := <-payloadChan:
				payloads = append(payloads, payload)
			case <-quitChan:
			}
		}
		wg.Done()
	}()
	service.Loop(eventsChannel)
	wg.Wait()
	if len(payloads) != 2 {
		t.Error("Test failure:", t.Name())
		t.Logf("Actual number of payloads does not equal expected.\nactual: %+v\nexpected: 3", len(payloads))
	}

	testReceipts1Rlp, err := rlp.EncodeToBytes(testReceipts1)
	if err != nil {
		t.Error(err)
	}
	testReceipts2Rlp, err := rlp.EncodeToBytes(testReceipts2)
	if err != nil {
		t.Error(err)
	}
	expectedReceiptsRlp := [][]byte{testReceipts1Rlp, testReceipts2Rlp, nil}
	for i, payload := range payloads {
		if !bytes.Equal(payload.ReceiptsRlp, expectedReceiptsRlp[i]) {
			t.Error("Test failure:", t.Name())
			t.Logf("Actual receipt rlp for payload %d does not equal expected.\nactual: %+v\nexpected: %+v", i, payload.ReceiptsRlp, expectedReceiptsRlp[i])
		}
	}

	if !reflect.DeepEqual(builder.Params, defaultParams) {
		t.Error("Test failure:", t.Name())
		t.Logf("Actual params does not equal expected.\nactual:%+v\nexpected: %+v", builder.Params, defaultParams)
	}
	if !bytes.Equal(builder.Args.BlockHash.Bytes(), testBlock2.Hash().Bytes()) {
		t.Error("Test failure:", t.Name())
		t.Logf("Actual blockhash does not equal expected.\nactual:%x\nexpected: %x", builder.Args.BlockHash.Bytes(), testBlock2.Hash().Bytes())
	}
	if !bytes.Equal(builder.Args.OldStateRoot.Bytes(), parentBlock2.Root().Bytes()) {
		t.Error("Test failure:", t.Name())
		t.Logf("Actual root does not equal expected.\nactual:%x\nexpected: %x", builder.Args.OldStateRoot.Bytes(), parentBlock2.Root().Bytes())
	}
	if !bytes.Equal(builder.Args.NewStateRoot.Bytes(), testBlock2.Root().Bytes()) {
		t.Error("Test failure:", t.Name())
		t.Logf("Actual root does not equal expected.\nactual:%x\nexpected: %x", builder.Args.NewStateRoot.Bytes(), testBlock2.Root().Bytes())
	}
	//look up the parent block from its hash
	expectedHashes := []common.Hash{testBlock1.ParentHash(), testBlock2.ParentHash()}
	if !reflect.DeepEqual(blockChain.HashesLookedUp, expectedHashes) {
		t.Error("Test failure:", t.Name())
		t.Logf("Actual looked up parent hashes does not equal expected.\nactual:%+v\nexpected: %+v", blockChain.HashesLookedUp, expectedHashes)
	}
}

func testErrorInBlockLoop(t *testing.T) {
	//second block's parent block can't be found
	builder := mocks.Builder{}
	blockChain := mocks.BlockChain{}
	service := statediff.Service{
		Builder:           &builder,
		BlockChain:        &blockChain,
		QuitChan:          make(chan bool),
		Subscriptions:     make(map[common.Hash]map[rpc.ID]statediff.Subscription),
		SubscriptionTypes: make(map[common.Hash]statediff.Params),
	}
	payloadChan := make(chan statediff.Payload)
	quitChan := make(chan bool)
	service.Subscribe(rpc.NewID(), payloadChan, quitChan, defaultParams)
	blockMapping := make(map[common.Hash]*types.Block)
	blockMapping[parentBlock1.Hash()] = parentBlock1
	blockChain.SetBlocksForHashes(blockMapping)
	blockChain.SetChainEvents([]core.ChainEvent{event1, event2})
	// Need to have listeners on the channels or the subscription will be closed and the processing halted
	go func() {
		select {
		case <-payloadChan:
		case <-quitChan:
		}
	}()
	service.Loop(eventsChannel)
	if !reflect.DeepEqual(builder.Params, defaultParams) {
		t.Error("Test failure:", t.Name())
		t.Logf("Actual params does not equal expected.\nactual:%+v\nexpected: %+v", builder.Params, defaultParams)
	}
	if !bytes.Equal(builder.Args.BlockHash.Bytes(), testBlock1.Hash().Bytes()) {
		t.Error("Test failure:", t.Name())
		t.Logf("Actual blockhash does not equal expected.\nactual:%+v\nexpected: %x", builder.Args.BlockHash.Bytes(), testBlock1.Hash().Bytes())
	}
	if !bytes.Equal(builder.Args.OldStateRoot.Bytes(), parentBlock1.Root().Bytes()) {
		t.Error("Test failure:", t.Name())
		t.Logf("Actual old state root does not equal expected.\nactual:%+v\nexpected: %x", builder.Args.OldStateRoot.Bytes(), parentBlock1.Root().Bytes())
	}
	if !bytes.Equal(builder.Args.NewStateRoot.Bytes(), testBlock1.Root().Bytes()) {
		t.Error("Test failure:", t.Name())
		t.Logf("Actual new state root does not equal expected.\nactual:%+v\nexpected: %x", builder.Args.NewStateRoot.Bytes(), testBlock1.Root().Bytes())
	}
}

func TestGetStateDiffAt(t *testing.T) {
	testErrorInStateDiffAt(t)
}

func testErrorInStateDiffAt(t *testing.T) {
	mockStateDiff := statediff.StateObject{
		BlockNumber: testBlock1.Number(),
		BlockHash:   testBlock1.Hash(),
	}
	expectedStateDiffRlp, err := rlp.EncodeToBytes(mockStateDiff)
	if err != nil {
		t.Error(err)
	}
	expectedReceiptsRlp, err := rlp.EncodeToBytes(testReceipts1)
	if err != nil {
		t.Error(err)
	}
	expectedBlockRlp, err := rlp.EncodeToBytes(testBlock1)
	if err != nil {
		t.Error(err)
	}
	expectedStateDiffPayload := statediff.Payload{
		StateObjectRlp: expectedStateDiffRlp,
		ReceiptsRlp:    expectedReceiptsRlp,
		BlockRlp:       expectedBlockRlp,
	}
	expectedStateDiffPayloadRlp, err := rlp.EncodeToBytes(expectedStateDiffPayload)
	if err != nil {
		t.Error(err)
	}
	builder := mocks.Builder{}
	builder.SetStateDiffToBuild(mockStateDiff)
	blockChain := mocks.BlockChain{}
	blockMapping := make(map[common.Hash]*types.Block)
	blockMapping[parentBlock1.Hash()] = parentBlock1
	blockChain.SetBlocksForHashes(blockMapping)
	blockChain.SetBlockForNumber(testBlock1, testBlock1.NumberU64())
	blockChain.SetReceiptsForHash(testBlock1.Hash(), testReceipts1)
	service := statediff.Service{
		Mutex:             sync.Mutex{},
		Builder:           &builder,
		BlockChain:        &blockChain,
		QuitChan:          make(chan bool),
		Subscriptions:     make(map[common.Hash]map[rpc.ID]statediff.Subscription),
		SubscriptionTypes: make(map[common.Hash]statediff.Params),
	}
	stateDiffPayload, err := service.StateDiffAt(testBlock1.NumberU64(), defaultParams)
	if err != nil {
		t.Error(err)
	}
	stateDiffPayloadRlp, err := rlp.EncodeToBytes(stateDiffPayload)
	if err != nil {
		t.Error(err)
	}
	if !reflect.DeepEqual(builder.Params, defaultParams) {
		t.Error("Test failure:", t.Name())
		t.Logf("Actual params does not equal expected.\nactual:%+v\nexpected: %+v", builder.Params, defaultParams)
	}
	if !bytes.Equal(builder.Args.BlockHash.Bytes(), testBlock1.Hash().Bytes()) {
		t.Error("Test failure:", t.Name())
		t.Logf("Actual blockhash does not equal expected.\nactual:%+v\nexpected: %x", builder.Args.BlockHash.Bytes(), testBlock1.Hash().Bytes())
	}
	if !bytes.Equal(builder.Args.OldStateRoot.Bytes(), parentBlock1.Root().Bytes()) {
		t.Error("Test failure:", t.Name())
		t.Logf("Actual old state root does not equal expected.\nactual:%+v\nexpected: %x", builder.Args.OldStateRoot.Bytes(), parentBlock1.Root().Bytes())
	}
	if !bytes.Equal(builder.Args.NewStateRoot.Bytes(), testBlock1.Root().Bytes()) {
		t.Error("Test failure:", t.Name())
		t.Logf("Actual new state root does not equal expected.\nactual:%+v\nexpected: %x", builder.Args.NewStateRoot.Bytes(), testBlock1.Root().Bytes())
	}
	if !bytes.Equal(expectedStateDiffPayloadRlp, stateDiffPayloadRlp) {
		t.Error("Test failure:", t.Name())
		t.Logf("Actual state diff payload does not equal expected.\nactual:%+v\nexpected: %+v", expectedStateDiffPayload, stateDiffPayload)
	}
}
