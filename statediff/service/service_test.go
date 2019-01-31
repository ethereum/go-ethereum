package service_test

import (
	"math/big"
	"math/rand"
	"reflect"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	service2 "github.com/ethereum/go-ethereum/statediff/service"
	"github.com/ethereum/go-ethereum/statediff/testhelpers/mocks"
)

func TestServiceLoop(t *testing.T) {
	testServiceLoop(t)
}

var (
	eventsChannel = make(chan core.ChainEvent, 10)

	parentHeader1 = types.Header{Number: big.NewInt(rand.Int63())}
	parentHeader2 = types.Header{Number: big.NewInt(rand.Int63())}

	parentBlock1 = types.NewBlock(&parentHeader1, nil, nil, nil)
	parentBlock2 = types.NewBlock(&parentHeader2, nil, nil, nil)

	parentHash1 = parentBlock1.Hash()
	parentHash2 = parentBlock2.Hash()

	header1 = types.Header{ParentHash: parentHash1}
	header2 = types.Header{ParentHash: parentHash2}

	block1 = types.NewBlock(&header1, nil, nil, nil)
	block2 = types.NewBlock(&header2, nil, nil, nil)

	event1 = core.ChainEvent{Block: block1}
	event2 = core.ChainEvent{Block: block2}
)

func testServiceLoop(t *testing.T) {
	eventsChannel <- event1
	eventsChannel <- event2

	extractor := mocks.Extractor{}
	close(eventsChannel)

	blockChain := mocks.BlockChain{}
	service := service2.StateDiffService{
		Builder:    nil,
		Extractor:  &extractor,
		BlockChain: &blockChain,
	}

	blockChain.SetParentBlockToReturn([]*types.Block{parentBlock1, parentBlock2})
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
