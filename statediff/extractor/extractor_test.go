package extractor_test

import (
	"bytes"
	"github.com/ethereum/go-ethereum/core/types"
	b "github.com/ethereum/go-ethereum/statediff/builder"
	e "github.com/ethereum/go-ethereum/statediff/extractor"
	"github.com/ethereum/go-ethereum/statediff/testhelpers/mocks"
	"math/big"
	"math/rand"
	"reflect"
	"testing"
)

var publisher mocks.Publisher
var builder mocks.Builder
var currentBlockNumber *big.Int
var parentBlock, currentBlock *types.Block
var expectedStateDiff b.StateDiff
var extractor e.Extractor
var err error

func TestExtractor(t *testing.T) {
	publisher = mocks.Publisher{}
	builder = mocks.Builder{}
	extractor = e.NewExtractor(&builder, &publisher)
	if err != nil {
		t.Error(err)
	}

	blockNumber := rand.Int63()
	parentBlockNumber := big.NewInt(blockNumber - int64(1))
	currentBlockNumber = big.NewInt(blockNumber)
	parentBlock = types.NewBlock(&types.Header{Number: parentBlockNumber}, nil, nil, nil)
	currentBlock = types.NewBlock(&types.Header{Number: currentBlockNumber}, nil, nil, nil)

	expectedStateDiff = b.StateDiff{
		BlockNumber:     blockNumber,
		BlockHash:       currentBlock.Hash(),
		CreatedAccounts: nil,
		DeletedAccounts: nil,
		UpdatedAccounts: nil,
	}

	testBuildStateDiffStruct(t)
	testBuildStateDiffErrorHandling(t)
	testPublishingStateDiff(t)
	testPublisherErrorHandling(t)
}

func testBuildStateDiffStruct(t *testing.T) {
	builder.SetStateDiffToBuild(&expectedStateDiff)

	_, err = extractor.ExtractStateDiff(*parentBlock, *currentBlock)
	if err != nil {
		t.Error(err)
	}

	if !equals(builder.OldStateRoot, parentBlock.Root()) {
		t.Error()
	}
	if !equals(builder.NewStateRoot, currentBlock.Root()) {
		t.Error()
	}
	if !equals(builder.BlockNumber, currentBlockNumber.Int64()) {
		t.Error()
	}
	if !equals(builder.BlockHash, currentBlock.Hash()) {
		t.Error()
	}
}

func testBuildStateDiffErrorHandling(t *testing.T) {
	builder.SetBuilderError(mocks.Error)

	_, err = extractor.ExtractStateDiff(*parentBlock, *currentBlock)
	if err == nil {
		t.Error(err)
	}

	if !equals(err, mocks.Error) {
		t.Error()
	}
	builder.SetBuilderError(nil)
}

func testPublishingStateDiff(t *testing.T) {
	builder.SetStateDiffToBuild(&expectedStateDiff)

	_, err = extractor.ExtractStateDiff(*parentBlock, *currentBlock)
	if err != nil {
		t.Error(err)
	}

	if !equals(publisher.StateDiff, &expectedStateDiff) {
		t.Error()
	}
}

func testPublisherErrorHandling(t *testing.T) {
	publisher.SetPublisherError(mocks.Error)

	_, err = extractor.ExtractStateDiff(*parentBlock, *currentBlock)
	if err == nil {
		t.Error("Expected an error, but it didn't occur.")
	}
	if !equals(err, mocks.Error) {
		t.Error()
	}

	publisher.SetPublisherError(nil)
}

func equals(actual, expected interface{}) (success bool) {
	if actualByteSlice, ok := actual.([]byte); ok {
		if expectedByteSlice, ok := expected.([]byte); ok {
			return bytes.Equal(actualByteSlice, expectedByteSlice)
		}
	}

	return reflect.DeepEqual(actual, expected)
}
