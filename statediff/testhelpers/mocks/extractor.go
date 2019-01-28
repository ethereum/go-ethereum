package mocks

import "github.com/ethereum/go-ethereum/core/types"

type Extractor struct {
	ParentBlocks  []types.Block
	CurrentBlocks []types.Block
	extractError  error
}

func (me *Extractor) ExtractStateDiff(parent, current types.Block) (string, error) {
	me.ParentBlocks = append(me.ParentBlocks, parent)
	me.CurrentBlocks = append(me.CurrentBlocks, current)

	return "", me.extractError
}

func (me *Extractor) SetExtractError(err error) {
	me.extractError = err
}
