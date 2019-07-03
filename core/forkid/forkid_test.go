// Copyright 2019 The go-ethereum Authors
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

package forkid

import (
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/params"
)

// TestCreation tests that different genesis and fork rule combinations result in
// the correct fork ID.
func TestCreation(t *testing.T) {
	type testcase struct {
		head uint64
		want ID
	}
	tests := []struct {
		config  *params.ChainConfig
		genesis common.Hash
		cases   []testcase
	}{
		// Mainnet test cases
		{
			params.MainnetChainConfig,
			params.MainnetGenesisHash,
			[]testcase{
				{0, ID{0xfc, 0x64, 0xec, 0x04, 0xc9, 0x29, 0xf1, 0xc5}},       // Unsynced
				{1149999, ID{0xfc, 0x64, 0xec, 0x04, 0xc9, 0x29, 0xf1, 0xc5}}, // Last Frontier block
				{1150000, ID{0x97, 0xc2, 0xc3, 0x4c, 0x2d, 0x10, 0xef, 0x43}}, // First Homestead block
				{1919999, ID{0x97, 0xc2, 0xc3, 0x4c, 0x2d, 0x10, 0xef, 0x43}}, // Last Homestead block
				{1920000, ID{0x91, 0xd1, 0xf9, 0x48, 0x44, 0xfe, 0xbd, 0x6b}}, // First DAO block
				{2462999, ID{0x91, 0xd1, 0xf9, 0x48, 0x44, 0xfe, 0xbd, 0x6b}}, // Last DAO block
				{2463000, ID{0x7a, 0x64, 0xda, 0x13, 0xe3, 0x5d, 0x84, 0xf1}}, // First Tangerine block
				{2674999, ID{0x7a, 0x64, 0xda, 0x13, 0xe3, 0x5d, 0x84, 0xf1}}, // Last Tangerine block
				{2675000, ID{0x3e, 0xdd, 0x5b, 0x10, 0x4d, 0xd3, 0x46, 0x54}}, // First Spurious block
				{4369999, ID{0x3e, 0xdd, 0x5b, 0x10, 0x4d, 0xd3, 0x46, 0x54}}, // Last Spurious block
				{4370000, ID{0xa0, 0x0b, 0xc3, 0x24, 0xfc, 0xa4, 0x36, 0x40}}, // First Byzantium block
				{7279999, ID{0xa0, 0x0b, 0xc3, 0x24, 0xfc, 0xa4, 0x36, 0x40}}, // Last Byzantium block
				{7280000, ID{0x66, 0x8d, 0xb0, 0xaf, 0x00, 0x00, 0x00, 0x00}}, // First and last Constantinople, first Petersburg block
				{7987396, ID{0x66, 0x8d, 0xb0, 0xaf, 0x00, 0x00, 0x00, 0x00}}, // Today Petersburg block
			},
		},
		// Ropsten test cases
		{
			params.TestnetChainConfig,
			params.TestnetGenesisHash,
			[]testcase{
				{0, ID{0x30, 0xc7, 0xdd, 0xbc, 0x85, 0xf7, 0x36, 0x77}},       // Unsynced, last Frontier, Homestead and first Tangerine block
				{9, ID{0x30, 0xc7, 0xdd, 0xbc, 0x85, 0xf7, 0x36, 0x77}},       // Last Tangerine block
				{10, ID{0x63, 0x76, 0x01, 0x90, 0xb4, 0xbf, 0x05, 0xc3}},      // First Spurious block
				{1699999, ID{0x63, 0x76, 0x01, 0x90, 0xb4, 0xbf, 0x05, 0xc3}}, // Last Spurious block
				{1700000, ID{0x3e, 0xa1, 0x59, 0xc7, 0x9d, 0xca, 0x62, 0x15}}, // First Byzantium block
				{4229999, ID{0x3e, 0xa1, 0x59, 0xc7, 0x9d, 0xca, 0x62, 0x15}}, // Last Byzantium block
				{4230000, ID{0x97, 0xb5, 0x44, 0xf3, 0x3e, 0x63, 0x2f, 0x9e}}, // First Constantinople block
				{4939393, ID{0x97, 0xb5, 0x44, 0xf3, 0x3e, 0x63, 0x2f, 0x9e}}, // Last Constantinople block
				{4939394, ID{0xd6, 0xe2, 0x14, 0x9b, 0x00, 0x00, 0x00, 0x00}}, // First Petersburg block
				{5822692, ID{0xd6, 0xe2, 0x14, 0x9b, 0x00, 0x00, 0x00, 0x00}}, // Today Petersburg block
			},
		},
		// Rinkeby test cases
		{
			params.RinkebyChainConfig,
			params.RinkebyGenesisHash,
			[]testcase{
				{0, ID{0x3b, 0x8e, 0x06, 0x91, 0x12, 0x25, 0xef, 0xff}},       // Unsynced, last Frontier block
				{1, ID{0x60, 0x94, 0x92, 0x95, 0x8b, 0x2c, 0xbe, 0x45}},       // First and last Homestead block
				{2, ID{0x8b, 0xde, 0x40, 0xdd, 0xfc, 0x2b, 0x8e, 0xd3}},       // First and last Tangerine block
				{3, ID{0xcb, 0x3a, 0x64, 0xbb, 0x42, 0x35, 0xd4, 0x51}},       // First Spurious block
				{1035300, ID{0xcb, 0x3a, 0x64, 0xbb, 0x42, 0x35, 0xd4, 0x51}}, // Last Spurious block
				{1035301, ID{0x8d, 0x74, 0x8b, 0x57, 0xe8, 0xab, 0xd4, 0x37}}, // First Byzantium block
				{3660662, ID{0x8d, 0x74, 0x8b, 0x57, 0xe8, 0xab, 0xd4, 0x37}}, // Last Byzantium block
				{3660663, ID{0xe4, 0x9c, 0xab, 0x14, 0xa5, 0x41, 0x64, 0x45}}, // First Constantinople block
				{4321233, ID{0xe4, 0x9c, 0xab, 0x14, 0xa5, 0x41, 0x64, 0x45}}, // Last Constantinople block
				{4321234, ID{0xaf, 0xec, 0x6b, 0x27, 0x00, 0x00, 0x00, 0x00}}, // First Petersburg block
				{4586649, ID{0xaf, 0xec, 0x6b, 0x27, 0x00, 0x00, 0x00, 0x00}}, // Today Petersburg block
			},
		},
		// Goerli test cases
		{
			params.GoerliChainConfig,
			params.GoerliGenesisHash,
			[]testcase{
				{0, ID{0xa3, 0xf5, 0xab, 0x08, 0x00, 0x00, 0x00, 0x00}},      // Unsynced, last Frontier, Homestead, Tangerine, Spurious, Byzantium, Constantinople and first Petersburg block
				{795329, ID{0xa3, 0xf5, 0xab, 0x08, 0x00, 0x00, 0x00, 0x00}}, // Today Petersburg block
			},
		},
	}
	for i, tt := range tests {
		for j, ttt := range tt.cases {
			if have := newID(tt.config, tt.genesis, ttt.head); have != ttt.want {
				t.Errorf("test %d, case %d: fork ID mismatch: have %x, want %x", i, j, have, ttt.want)
			}
		}
	}
}

// TestValidation tests that a local peer correctly validates and accepts a remote
// fork ID.
func TestValidation(t *testing.T) {
	tests := []struct {
		head uint64
		id   ID
		err  error
	}{
		// Local is mainnet Petersburg, remote announces the same. No future fork is announced.
		{7987396, ID{0x66, 0x8d, 0xb0, 0xaf, 0x00, 0x00, 0x00, 0x00}, nil},

		// Local is mainnet Petersburg, remote announces the same. Remote also announces a next fork
		// at block 0xffffffff, but that is uncertain.
		{7987396, ID{0x66, 0x8d, 0xb0, 0xaf, 0xbb, 0x99, 0xff, 0x8a}, nil},

		// Local is mainnet currently in Byzantium only (so it's aware of Petersburg), remote announces
		// also Byzantium, but it's not yet aware of Petersburg (e.g. non updated node before the fork).
		// In this case we don't know if Petersburg passed yet or not.
		{7279999, ID{0xa0, 0x0b, 0xc3, 0x24, 0x00, 0x00, 0x00, 0x00}, nil},

		// Local is mainnet currently in Byzantium only (so it's aware of Petersburg), remote announces
		// also Byzantium, and it's also aware of Petersburg (e.g. updated node before the fork). We
		// don't know if Petersburg passed yet (will pass) or not.
		{7279999, ID{0xa0, 0x0b, 0xc3, 0x24, 0xfc, 0xa4, 0x36, 0x40}, nil},

		// Local is mainnet currently in Byzantium only (so it's aware of Petersburg), remote announces
		// also Byzantium, and it's also aware of some random fork (e.g. misconfigured Petersburg). As
		// neither forks passed at neither nodes, they may mismatch, but we still connect for now.
		{7279999, ID{0xa0, 0x0b, 0xc3, 0x24, 0xbb, 0x99, 0xff, 0x8a}, nil},

		// Local is mainnet Petersburg, remote announces Byzantium + knowledge about Petersburg. Remote
		// is simply out of sync, accept.
		{7987396, ID{0x66, 0x8d, 0xb0, 0xaf, 0xfc, 0xa4, 0x36, 0x40}, nil},

		// Local is mainnet Petersburg, remote announces Spurious + knowledge about Byzantium. Remote
		// is definitely out of sync. It may or may not need the Petersburg update, we don't know yet.
		{7987396, ID{0x3e, 0xdd, 0x5b, 0x10, 0x4d, 0xd3, 0x46, 0x54}, nil},

		// Local is mainnet Byzantium, remote announces Petersburg. Local is out of sync, accept.
		{7279999, ID{0x66, 0x8d, 0xb0, 0xaf, 0x00, 0x00, 0x00, 0x00}, nil},

		// Local is mainnet Spurious, remote announces Byzantium, but is not aware of Petersburg. Local
		// out of sync. Local also knows about a future fork, but that is uncertain yet.
		{4369999, ID{0xa0, 0x0b, 0xc3, 0x24, 0x00, 0x00, 0x00, 0x00}, nil},

		// Local is mainnet Petersburg. remote announces Byzantium but is not aware of further forks.
		// Remote needs software update.
		{7987396, ID{0xa0, 0x0b, 0xc3, 0x24, 0x00, 0x00, 0x00, 0x00}, ErrRemoteStale},

		// Local is mainnet Petersburg, and isn't aware of more forks. Remote announces Petersburg +
		// 0xffffffff. Local needs software update, reject.
		{7987396, ID{0x5c, 0xdd, 0xc0, 0xe1, 0x00, 0x00, 0x00, 0x00}, ErrLocalIncompatibleOrStale},

		// Local is mainnet Byzantium, and is aware of Petersburg. Remote announces Petersburg +
		// 0xffffffff. Local needs software update, reject.
		{7279999, ID{0x5c, 0xdd, 0xc0, 0xe1, 0x00, 0x00, 0x00, 0x00}, ErrLocalIncompatibleOrStale},

		// Local is mainnet Petersburg, remote is Rinkeby Petersburg.
		{7987396, ID{0xaf, 0xec, 0x6b, 0x27, 0x00, 0x00, 0x00, 0x00}, ErrLocalIncompatibleOrStale},
	}
	for i, tt := range tests {
		filter := newFilter(params.MainnetChainConfig, params.MainnetGenesisHash, func() uint64 { return tt.head })
		if err := filter(tt.id); err != tt.err {
			t.Errorf("test %d: validation error mismatch: have %v, want %v", i, err, tt.err)
		}
	}
}
