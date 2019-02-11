package service_test

import (
	"math/big"
	"math/rand"
	"reflect"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	s "github.com/ethereum/go-ethereum/statediff/service"
	"github.com/ethereum/go-ethereum/statediff/testhelpers/mocks"
)

func TestServiceLoop(t *testing.T) {
	testErrorInChainEventLoop(t)
	testErrorInBlockLoop(t)
}

var (
	eventsChannel = make(chan core.ChainEvent, 1)

	parentHeader1 = types.Header{Number: big.NewInt(rand.Int63())}
	parentHeader2 = types.Header{Number: big.NewInt(rand.Int63())}

	parentBlock1 = types.NewBlock(&parentHeader1, nil, nil, nil)
	parentBlock2 = types.NewBlock(&parentHeader2, nil, nil, nil)

	parentHash1 = parentBlock1.Hash()
	parentHash2 = parentBlock2.Hash()

	header1 = types.Header{ParentHash: parentHash1}
	header2 = types.Header{ParentHash: parentHash2}
	header3 = types.Header{ParentHash: common.HexToHash("parent hash")}

	block1 = types.NewBlock(&header1, nil, nil, nil)
	block2 = types.NewBlock(&header2, nil, nil, nil)
	block3 = types.NewBlock(&header3, nil, nil, nil)

	event1 = core.ChainEvent{Block: block1}
	event2 = core.ChainEvent{Block: block2}
	event3 = core.ChainEvent{Block: block3}
)

func testErrorInChainEventLoop(t *testing.T) {
	//the first chain event causes and error (in blockchain mock)
	extractor := mocks.Extractor{}

	blockChain := mocks.BlockChain{}
	service := s.StateDiffService{
		Builder:    nil,
		Extractor:  &extractor,
		BlockChain: &blockChain,
	}

	blockChain.SetParentBlocksToReturn([]*types.Block{parentBlock1, parentBlock2})
	blockChain.SetChainEvents([]core.ChainEvent{event1, event2, event3})
	service.Loop(eventsChannel)

	//parent and current blocks are passed to the extractor
	expectedCurrentBlocks := []types.Block{*block1, *block2}
	if !reflect.DeepEqual(extractor.CurrentBlocks, expectedCurrentBlocks) {
		t.Error("Test failure:", t.Name())
		t.Logf("Actual does not equal expected.\nactual:%+v\nexpected: %+v", extractor.CurrentBlocks, expectedCurrentBlocks)
	}
	expectedParentBlocks := []types.Block{*parentBlock1, *parentBlock2}
	if !reflect.DeepEqual(extractor.ParentBlocks, expectedParentBlocks) {
		t.Error("Test failure:", t.Name())
		t.Logf("Actual does not equal expected.\nactual:%+v\nexpected: %+v", extractor.CurrentBlocks, expectedParentBlocks)
	}

	//look up the parent block from its hash
	expectedHashes := []common.Hash{block1.ParentHash(), block2.ParentHash()}
	if !reflect.DeepEqual(blockChain.ParentHashesLookedUp, expectedHashes) {
		t.Error("Test failure:", t.Name())
		t.Logf("Actual does not equal expected.\nactual:%+v\nexpected: %+v", blockChain.ParentHashesLookedUp, expectedHashes)
	}
}

func testErrorInBlockLoop(t *testing.T) {
	//second block's parent block can't be found
	extractor := mocks.Extractor{}

	blockChain := mocks.BlockChain{}
	service := s.StateDiffService{
		Builder:    nil,
		Extractor:  &extractor,
		BlockChain: &blockChain,
	}

	blockChain.SetParentBlocksToReturn([]*types.Block{parentBlock1, nil})
	blockChain.SetChainEvents([]core.ChainEvent{event1, event2})
	service.Loop(eventsChannel)

	//only the first current block (and it's parent) are passed to the extractor
	expectedCurrentBlocks := []types.Block{*block1}
	if !reflect.DeepEqual(extractor.CurrentBlocks, expectedCurrentBlocks) {
		t.Error("Test failure:", t.Name())
		t.Logf("Actual does not equal expected.\nactual:%+v\nexpected: %+v", extractor.CurrentBlocks, expectedCurrentBlocks)
	}
	expectedParentBlocks := []types.Block{*parentBlock1}
	if !reflect.DeepEqual(extractor.ParentBlocks, expectedParentBlocks) {
		t.Error("Test failure:", t.Name())
		t.Logf("Actual does not equal expected.\nactual:%+v\nexpected: %+v", extractor.CurrentBlocks, expectedParentBlocks)
	}
}
