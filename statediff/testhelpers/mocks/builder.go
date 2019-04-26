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

package mocks

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/statediff"
)

// Builder is a mock state diff builder
type Builder struct {
	OldStateRoot common.Hash
	NewStateRoot common.Hash
	BlockNumber  int64
	BlockHash    common.Hash
	stateDiff    statediff.StateDiff
	builderError error
}

// BuildStateDiff mock method
func (builder *Builder) BuildStateDiff(oldStateRoot, newStateRoot common.Hash, blockNumber int64, blockHash common.Hash) (statediff.StateDiff, error) {
	builder.OldStateRoot = oldStateRoot
	builder.NewStateRoot = newStateRoot
	builder.BlockNumber = blockNumber
	builder.BlockHash = blockHash

	return builder.stateDiff, builder.builderError
}

// SetStateDiffToBuild mock method
func (builder *Builder) SetStateDiffToBuild(stateDiff statediff.StateDiff) {
	builder.stateDiff = stateDiff
}

// SetBuilderError mock method
func (builder *Builder) SetBuilderError(err error) {
	builder.builderError = err
}
