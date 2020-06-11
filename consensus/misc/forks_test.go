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

package misc

import (
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/params"
)

func TestVerifyForkHashes(t *testing.T) {
	var cases = []struct {
		config *params.ChainConfig
		number uint64
		hash   common.Hash
	}{
		// Non-fork point
		{params.MainnetChainConfig, 1, common.Hash{}},

		// Before the fork point
		{params.MainnetChainConfig, 1150000 - 1, common.Hash{}},

		// After the fork point
		{params.MainnetChainConfig, 1150000 + 1, common.Hash{}},

		// In the fork point
		{params.MainnetChainConfig, 1150000, common.HexToHash("0x584bdb5d4e74fe97f5a5222b533fe1322fd0b6ad3eb03f02c3221984e2c0b430")},

		// Last fork point
		{params.MainnetChainConfig, 9200000, common.HexToHash("0x6ba9486095de7d96a75b67954cfe2581234eae1ef2a92ab03b84fc2eae2deb8a")},

		// After the last fork point
		{params.MainnetChainConfig, 9200000 + 1, common.Hash{}},
	}
	for _, c := range cases {
		if err := verifyForkHashes(c.config, c.number, c.hash, false); err != nil {
			t.Fatalf("Failed to verify fork hashes, %v", err)
		}
	}
}
