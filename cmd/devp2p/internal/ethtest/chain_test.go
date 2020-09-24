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
	"path/filepath"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestChain_GetHeaders(t *testing.T) {
	chainFile, err := filepath.Abs("./testdata/chain.rlp.gz")
	if err != nil {
		t.Fatal(err)
	}
	genesisFile, err := filepath.Abs("./testdata/genesis.json")
	if err != nil {
		t.Fatal(err)
	}

	chain, err := loadChain(chainFile, genesisFile)
	if err != nil {
		t.Fatal(err)
	}

	var tests = []struct{
		req GetBlockHeaders
		expected BlockHeaders
	}{
		{
			req: GetBlockHeaders{
				Origin:  hashOrNumber{
					Number: uint64(2),
				},
				Amount:  uint64(5),
				Skip:    1,
				Reverse: false,
			},
			expected: BlockHeaders{
				chain.blocks[2].Header(),
				chain.blocks[4].Header(),
				chain.blocks[6].Header(),
				chain.blocks[8].Header(),
				chain.blocks[10].Header(),
			},
		},
		{
			req: GetBlockHeaders{
				Origin:  hashOrNumber{
					Number: uint64(chain.Len()-1),
				},
				Amount:  uint64(3),
				Skip:    0,
				Reverse: true,
			},
			expected: BlockHeaders{
				chain.blocks[chain.Len()-1].Header(),
				chain.blocks[chain.Len()-2].Header(),
				chain.blocks[chain.Len()-3].Header(),
			},
		},
		{
			req: GetBlockHeaders{
				Origin:  hashOrNumber{
					Hash: chain.Head().Hash(),
				},
				Amount:  uint64(1),
				Skip:    0,
				Reverse: false,
			},
			expected: BlockHeaders{
				chain.Head().Header(),
			},
		},
	}

	for i, tt := range tests {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			headers, err := chain.GetHeaders(tt.req)
			if err != nil {
				t.Fatal(err)
			}
			assert.Equal(t, headers, tt.expected)
		})
	}
}
