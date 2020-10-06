// Copyright 2020 The go-ethereum Authors
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

package ethtest

import (
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"math/big"
	"os"
	"strings"

	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/forkid"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/rlp"
)

type Chain struct {
	blocks      []*types.Block
	chainConfig *params.ChainConfig
}

func (c *Chain) WriteTo(writer io.Writer) error {
	for _, block := range c.blocks {
		if err := rlp.Encode(writer, block); err != nil {
			return err
		}
	}

	return nil
}

// Len returns the length of the chain.
func (c *Chain) Len() int {
	return len(c.blocks)
}

// TD calculates the total difficulty of the chain.
func (c *Chain) TD(height int) *big.Int { // TODO later on channge scheme so that the height is included in range
	sum := big.NewInt(0)
	for _, block := range c.blocks[:height] {
		sum.Add(sum, block.Difficulty())
	}
	return sum
}

// ForkID gets the fork id of the chain.
func (c *Chain) ForkID() forkid.ID {
	return forkid.NewID(c.chainConfig, c.blocks[0].Hash(), uint64(c.Len()))
}

// Shorten returns a copy chain of a desired height from the imported
func (c *Chain) Shorten(height int) *Chain {
	blocks := make([]*types.Block, height)
	copy(blocks, c.blocks[:height])

	config := *c.chainConfig
	return &Chain{
		blocks:      blocks,
		chainConfig: &config,
	}
}

// Head returns the chain head.
func (c *Chain) Head() *types.Block {
	return c.blocks[c.Len()-1]
}

func (c *Chain) GetHeaders(req GetBlockHeaders) (BlockHeaders, error) {
	if req.Amount < 1 {
		return nil, fmt.Errorf("no block headers requested")
	}

	headers := make(BlockHeaders, req.Amount)
	var blockNumber uint64

	// range over blocks to check if our chain has the requested header
	for _, block := range c.blocks {
		if block.Hash() == req.Origin.Hash || block.Number().Uint64() == req.Origin.Number {
			headers[0] = block.Header()
			blockNumber = block.Number().Uint64()
		}
	}
	if headers[0] == nil {
		return nil, fmt.Errorf("no headers found for given origin number %v, hash %v", req.Origin.Number, req.Origin.Hash)
	}

	if req.Reverse {
		for i := 1; i < int(req.Amount); i++ {
			blockNumber -= (1 - req.Skip)
			headers[i] = c.blocks[blockNumber].Header()

		}

		return headers, nil
	}

	for i := 1; i < int(req.Amount); i++ {
		blockNumber += (1 + req.Skip)
		headers[i] = c.blocks[blockNumber].Header()
	}

	return headers, nil
}

// loadChain takes the given chain.rlp file, and decodes and returns
// the blocks from the file.
func loadChain(chainfile string, genesis string) (*Chain, error) {
	// Open the file handle and potentially unwrap the gzip stream
	fh, err := os.Open(chainfile)
	if err != nil {
		return nil, err
	}
	defer fh.Close()

	var reader io.Reader = fh
	if strings.HasSuffix(chainfile, ".gz") {
		if reader, err = gzip.NewReader(reader); err != nil {
			return nil, err
		}
	}
	stream := rlp.NewStream(reader, 0)
	var blocks []*types.Block
	for i := 0; ; i++ {
		var b types.Block
		if err := stream.Decode(&b); err == io.EOF {
			break
		} else if err != nil {
			return nil, fmt.Errorf("at block %d: %v", i, err)
		}
		blocks = append(blocks, &b)
	}

	// Open the file handle and potentially unwrap the gzip stream
	chainConfig, err := ioutil.ReadFile(genesis)
	if err != nil {
		return nil, err
	}
	var gen core.Genesis
	if err := json.Unmarshal(chainConfig, &gen); err != nil {
		return nil, err
	}

	return &Chain{
		blocks:      blocks,
		chainConfig: gen.Config,
	}, nil
}
