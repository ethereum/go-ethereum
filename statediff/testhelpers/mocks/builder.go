package mocks

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/statediff/builder"
)

type Builder struct {
	OldStateRoot common.Hash
	NewStateRoot common.Hash
	BlockNumber  int64
	BlockHash    common.Hash
	stateDiff    *builder.StateDiff
	builderError error
}

func (builder *Builder) BuildStateDiff(oldStateRoot, newStateRoot common.Hash, blockNumber int64, blockHash common.Hash) (*builder.StateDiff, error) {
	builder.OldStateRoot = oldStateRoot
	builder.NewStateRoot = newStateRoot
	builder.BlockNumber = blockNumber
	builder.BlockHash = blockHash

	return builder.stateDiff, builder.builderError
}

func (builder *Builder) SetStateDiffToBuild(stateDiff *builder.StateDiff) {
	builder.stateDiff = stateDiff
}

func (builder *Builder) SetBuilderError(err error) {
	builder.builderError = err
}
