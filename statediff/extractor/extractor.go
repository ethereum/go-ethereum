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

// Contains a batch of utility type declarations used by the tests. As the node
// operates on unique types, a lot of them are needed to check various features.

package extractor

import (
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/statediff/builder"
	"github.com/ethereum/go-ethereum/statediff/publisher"
)

type Extractor interface {
	ExtractStateDiff(parent, current types.Block) (string, error)
}

type extractor struct {
	Builder   builder.Builder     // Interface for building state diff objects from two blocks
	Publisher publisher.Publisher // Interface for publishing state diff objects to a datastore (e.g. IPFS)
}

func NewExtractor(builder builder.Builder, publisher publisher.Publisher) *extractor {
	return &extractor{
		Builder:   builder,
		Publisher: publisher,
	}
}

func (e *extractor) ExtractStateDiff(parent, current types.Block) (string, error) {
	stateDiff, err := e.Builder.BuildStateDiff(parent.Root(), current.Root(), current.Number().Int64(), current.Hash())
	if err != nil {
		return "", err
	}

	return e.Publisher.PublishStateDiff(stateDiff)
}
