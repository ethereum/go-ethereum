// Copyright 2017 The go-ethereum Authors
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

// Contains some utility methods to allow creating hooked consensus structures
// mostly for test code where only a few methods are needed.

package consensus

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/params"
)

type CurrentHeaderFn func() *types.Header
type HeaderRetrievalFn func(hash common.Hash, number uint64) *types.Header
type HeaderByNumberRetrievalFn func(number uint64) *types.Header
type BlockRetrievalFn func(hash common.Hash, number uint64) *types.Block

// hookedChainReader is a tiny implementation of ChainReader based on callbacks
// and a preset chain configuration. It's useful to avoid defining various custom
// types where a cain reader isn't readily available.
type hookedChainReader struct {
	config            *params.ChainConfig
	currentHeader     CurrentHeaderFn
	getHeader         HeaderRetrievalFn
	getHeaderByNumber HeaderByNumberRetrievalFn
	getBlock          BlockRetrievalFn
}

// MakeChainReader creates a callback based chain reader.
func MakeChainReader(config *params.ChainConfig, currentHeader CurrentHeaderFn, getHeader HeaderRetrievalFn, getHeaderByNumber HeaderByNumberRetrievalFn, getBlock BlockRetrievalFn) ChainReader {
	return &hookedChainReader{
		config:            config,
		currentHeader:     currentHeader,
		getHeader:         getHeader,
		getHeaderByNumber: getHeaderByNumber,
		getBlock:          getBlock,
	}
}

// Config implements ChainReader, retrieving the blockchain's chain configuration.
func (hcr *hookedChainReader) Config() *params.ChainConfig { return hcr.config }

// CurrentHeader implements ChainReader, retrieving the current header.
func (hcr *hookedChainReader) CurrentHeader() *types.Header {
	return hcr.currentHeader()
}

// GetHeader implements ChainReader, retrieving a block header from the database
// by hash and number.
func (hcr *hookedChainReader) GetHeader(hash common.Hash, number uint64) *types.Header {
	return hcr.getHeader(hash, number)
}

// GetHeaderByNumber implements ChainReader, retrieving a block header from the
// database by number.
func (hcr *hookedChainReader) GetHeaderByNumber(number uint64) *types.Header {
	return hcr.getHeaderByNumber(number)
}

// GetBlock implements ChainReader, retrieving a block from the database by hash and number.
func (hcr *hookedChainReader) GetBlock(hash common.Hash, number uint64) *types.Block {
	return hcr.getBlock(hash, number)
}
