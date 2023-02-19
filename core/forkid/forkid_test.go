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
	"bytes"
	"math"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/rlp"
)

func u64(val uint64) *uint64 { return &val }

// TestCreation tests that different genesis and fork rule combinations result in
// the correct fork ID.
func TestCreation(t *testing.T) {
	// Temporary non-existent scenario TODO(karalabe): delete when Shanghai is enabled
	timestampedConfig := *params.MainnetChainConfig
	timestampedConfig.ShanghaiTime = u64(1668000000)

	type testcase struct {
		head uint64
		time uint64
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
				{0, 0, ID{Hash: checksumToBytes(0xfc64ec04), Next: 1150000}},         // Unsynced
				{1149999, 0, ID{Hash: checksumToBytes(0xfc64ec04), Next: 1150000}},   // Last Frontier block
				{1150000, 0, ID{Hash: checksumToBytes(0x97c2c34c), Next: 1920000}},   // First Homestead block
				{1919999, 0, ID{Hash: checksumToBytes(0x97c2c34c), Next: 1920000}},   // Last Homestead block
				{1920000, 0, ID{Hash: checksumToBytes(0x91d1f948), Next: 2463000}},   // First DAO block
				{2462999, 0, ID{Hash: checksumToBytes(0x91d1f948), Next: 2463000}},   // Last DAO block
				{2463000, 0, ID{Hash: checksumToBytes(0x7a64da13), Next: 2675000}},   // First Tangerine block
				{2674999, 0, ID{Hash: checksumToBytes(0x7a64da13), Next: 2675000}},   // Last Tangerine block
				{2675000, 0, ID{Hash: checksumToBytes(0x3edd5b10), Next: 4370000}},   // First Spurious block
				{4369999, 0, ID{Hash: checksumToBytes(0x3edd5b10), Next: 4370000}},   // Last Spurious block
				{4370000, 0, ID{Hash: checksumToBytes(0xa00bc324), Next: 7280000}},   // First Byzantium block
				{7279999, 0, ID{Hash: checksumToBytes(0xa00bc324), Next: 7280000}},   // Last Byzantium block
				{7280000, 0, ID{Hash: checksumToBytes(0x668db0af), Next: 9069000}},   // First and last Constantinople, first Petersburg block
				{9068999, 0, ID{Hash: checksumToBytes(0x668db0af), Next: 9069000}},   // Last Petersburg block
				{9069000, 0, ID{Hash: checksumToBytes(0x879d6e30), Next: 9200000}},   // First Istanbul and first Muir Glacier block
				{9199999, 0, ID{Hash: checksumToBytes(0x879d6e30), Next: 9200000}},   // Last Istanbul and first Muir Glacier block
				{9200000, 0, ID{Hash: checksumToBytes(0xe029e991), Next: 12244000}},  // First Muir Glacier block
				{12243999, 0, ID{Hash: checksumToBytes(0xe029e991), Next: 12244000}}, // Last Muir Glacier block
				{12244000, 0, ID{Hash: checksumToBytes(0x0eb440f6), Next: 12965000}}, // First Berlin block
				{12964999, 0, ID{Hash: checksumToBytes(0x0eb440f6), Next: 12965000}}, // Last Berlin block
				{12965000, 0, ID{Hash: checksumToBytes(0xb715077d), Next: 13773000}}, // First London block
				{13772999, 0, ID{Hash: checksumToBytes(0xb715077d), Next: 13773000}}, // Last London block
				{13773000, 0, ID{Hash: checksumToBytes(0x20c327fc), Next: 15050000}}, // First Arrow Glacier block
				{15049999, 0, ID{Hash: checksumToBytes(0x20c327fc), Next: 15050000}}, // Last Arrow Glacier block
				{15050000, 0, ID{Hash: checksumToBytes(0xf0afd0e3), Next: 0}},        // First Gray Glacier block
				{20000000, 0, ID{Hash: checksumToBytes(0xf0afd0e3), Next: 0}},        // Future Gray Glacier block
			},
		},
		// Rinkeby test cases
		{
			params.RinkebyChainConfig,
			params.RinkebyGenesisHash,
			[]testcase{
				{0, 0, ID{Hash: checksumToBytes(0x3b8e0691), Next: 1}},             // Unsynced, last Frontier block
				{1, 0, ID{Hash: checksumToBytes(0x60949295), Next: 2}},             // First and last Homestead block
				{2, 0, ID{Hash: checksumToBytes(0x8bde40dd), Next: 3}},             // First and last Tangerine block
				{3, 0, ID{Hash: checksumToBytes(0xcb3a64bb), Next: 1035301}},       // First Spurious block
				{1035300, 0, ID{Hash: checksumToBytes(0xcb3a64bb), Next: 1035301}}, // Last Spurious block
				{1035301, 0, ID{Hash: checksumToBytes(0x8d748b57), Next: 3660663}}, // First Byzantium block
				{3660662, 0, ID{Hash: checksumToBytes(0x8d748b57), Next: 3660663}}, // Last Byzantium block
				{3660663, 0, ID{Hash: checksumToBytes(0xe49cab14), Next: 4321234}}, // First Constantinople block
				{4321233, 0, ID{Hash: checksumToBytes(0xe49cab14), Next: 4321234}}, // Last Constantinople block
				{4321234, 0, ID{Hash: checksumToBytes(0xafec6b27), Next: 5435345}}, // First Petersburg block
				{5435344, 0, ID{Hash: checksumToBytes(0xafec6b27), Next: 5435345}}, // Last Petersburg block
				{5435345, 0, ID{Hash: checksumToBytes(0xcbdb8838), Next: 8290928}}, // First Istanbul block
				{8290927, 0, ID{Hash: checksumToBytes(0xcbdb8838), Next: 8290928}}, // Last Istanbul block
				{8290928, 0, ID{Hash: checksumToBytes(0x6910c8bd), Next: 8897988}}, // First Berlin block
				{8897987, 0, ID{Hash: checksumToBytes(0x6910c8bd), Next: 8897988}}, // Last Berlin block
				{8897988, 0, ID{Hash: checksumToBytes(0x8E29F2F3), Next: 0}},       // First London block
				{10000000, 0, ID{Hash: checksumToBytes(0x8E29F2F3), Next: 0}},      // Future London block
			},
		},
		// Goerli test cases
		{
			params.GoerliChainConfig,
			params.GoerliGenesisHash,
			[]testcase{
				{0, 0, ID{Hash: checksumToBytes(0xa3f5ab08), Next: 1561651}},       // Unsynced, last Frontier, Homestead, Tangerine, Spurious, Byzantium, Constantinople and first Petersburg block
				{1561650, 0, ID{Hash: checksumToBytes(0xa3f5ab08), Next: 1561651}}, // Last Petersburg block
				{1561651, 0, ID{Hash: checksumToBytes(0xc25efa5c), Next: 4460644}}, // First Istanbul block
				{4460643, 0, ID{Hash: checksumToBytes(0xc25efa5c), Next: 4460644}}, // Last Istanbul block
				{4460644, 0, ID{Hash: checksumToBytes(0x757a1c47), Next: 5062605}}, // First Berlin block
				{5000000, 0, ID{Hash: checksumToBytes(0x757a1c47), Next: 5062605}}, // Last Berlin block
				{5062605, 0, ID{Hash: checksumToBytes(0xB8C6299D), Next: 0}},       // First London block
				{6000000, 0, ID{Hash: checksumToBytes(0xB8C6299D), Next: 0}},       // Future London block
			},
		},
		// Sepolia test cases
		{
			params.SepoliaChainConfig,
			params.SepoliaGenesisHash,
			[]testcase{
				{0, 0, ID{Hash: checksumToBytes(0xfe3366e7), Next: 1735371}},                   // Unsynced, last Frontier, Homestead, Tangerine, Spurious, Byzantium, Constantinople, Petersburg, Istanbul, Berlin and first London block
				{1735370, 0, ID{Hash: checksumToBytes(0xfe3366e7), Next: 1735371}},             // Last London block
				{1735371, 0, ID{Hash: checksumToBytes(0xb96cbd13), Next: 1677557088}},          // First MergeNetsplit block
				{1735372, 1677557087, ID{Hash: checksumToBytes(0xb96cbd13), Next: 1677557088}}, // Last MergeNetsplit block
				{1735372, 1677557088, ID{Hash: checksumToBytes(0xf7f9bc08), Next: 0}},          // First Shanghai block
			},
		},
		// Temporary timestamped test cases
		{
			&timestampedConfig,
			params.MainnetGenesisHash,
			[]testcase{
				{0, 0, ID{Hash: checksumToBytes(0xfc64ec04), Next: 1150000}},                    // Unsynced
				{1149999, 0, ID{Hash: checksumToBytes(0xfc64ec04), Next: 1150000}},              // Last Frontier block
				{1150000, 0, ID{Hash: checksumToBytes(0x97c2c34c), Next: 1920000}},              // First Homestead block
				{1919999, 0, ID{Hash: checksumToBytes(0x97c2c34c), Next: 1920000}},              // Last Homestead block
				{1920000, 0, ID{Hash: checksumToBytes(0x91d1f948), Next: 2463000}},              // First DAO block
				{2462999, 0, ID{Hash: checksumToBytes(0x91d1f948), Next: 2463000}},              // Last DAO block
				{2463000, 0, ID{Hash: checksumToBytes(0x7a64da13), Next: 2675000}},              // First Tangerine block
				{2674999, 0, ID{Hash: checksumToBytes(0x7a64da13), Next: 2675000}},              // Last Tangerine block
				{2675000, 0, ID{Hash: checksumToBytes(0x3edd5b10), Next: 4370000}},              // First Spurious block
				{4369999, 0, ID{Hash: checksumToBytes(0x3edd5b10), Next: 4370000}},              // Last Spurious block
				{4370000, 0, ID{Hash: checksumToBytes(0xa00bc324), Next: 7280000}},              // First Byzantium block
				{7279999, 0, ID{Hash: checksumToBytes(0xa00bc324), Next: 7280000}},              // Last Byzantium block
				{7280000, 0, ID{Hash: checksumToBytes(0x668db0af), Next: 9069000}},              // First and last Constantinople, first Petersburg block
				{9068999, 0, ID{Hash: checksumToBytes(0x668db0af), Next: 9069000}},              // Last Petersburg block
				{9069000, 0, ID{Hash: checksumToBytes(0x879d6e30), Next: 9200000}},              // First Istanbul and first Muir Glacier block
				{9199999, 0, ID{Hash: checksumToBytes(0x879d6e30), Next: 9200000}},              // Last Istanbul and first Muir Glacier block
				{9200000, 0, ID{Hash: checksumToBytes(0xe029e991), Next: 12244000}},             // First Muir Glacier block
				{12243999, 0, ID{Hash: checksumToBytes(0xe029e991), Next: 12244000}},            // Last Muir Glacier block
				{12244000, 0, ID{Hash: checksumToBytes(0x0eb440f6), Next: 12965000}},            // First Berlin block
				{12964999, 0, ID{Hash: checksumToBytes(0x0eb440f6), Next: 12965000}},            // Last Berlin block
				{12965000, 0, ID{Hash: checksumToBytes(0xb715077d), Next: 13773000}},            // First London block
				{13772999, 0, ID{Hash: checksumToBytes(0xb715077d), Next: 13773000}},            // Last London block
				{13773000, 0, ID{Hash: checksumToBytes(0x20c327fc), Next: 15050000}},            // First Arrow Glacier block
				{15049999, 0, ID{Hash: checksumToBytes(0x20c327fc), Next: 15050000}},            // Last Arrow Glacier block
				{15050000, 0, ID{Hash: checksumToBytes(0xf0afd0e3), Next: 1668000000}},          // First Gray Glacier block
				{19999999, 1667999999, ID{Hash: checksumToBytes(0xf0afd0e3), Next: 1668000000}}, // Last Gray Glacier block
				{20000000, 1668000000, ID{Hash: checksumToBytes(0x71147644), Next: 0}},          // First Shanghai block
				{20000000, 2668000000, ID{Hash: checksumToBytes(0x71147644), Next: 0}},          // Future Shanghai block
			},
		},
	}
	for i, tt := range tests {
		for j, ttt := range tt.cases {
			if have := NewID(tt.config, tt.genesis, ttt.head, ttt.time); have != ttt.want {
				t.Errorf("test %d, case %d: fork ID mismatch: have %x, want %x", i, j, have, ttt.want)
			}
		}
	}
}

// TestValidation tests that a local peer correctly validates and accepts a remote
// fork ID.
func TestValidation(t *testing.T) {
	// Temporary non-existent scenario TODO(karalabe): delete when Shanghai is enabled
	timestampedConfig := *params.MainnetChainConfig
	timestampedConfig.ShanghaiTime = u64(1668000000)

	tests := []struct {
		config *params.ChainConfig
		head   uint64
		time   uint64
		id     ID
		err    error
	}{
		//------------------
		// Block based tests
		//------------------

		// Local is mainnet Gray Glacier, remote announces the same. No future fork is announced.
		{params.MainnetChainConfig, 15050000, 0, ID{Hash: checksumToBytes(0xf0afd0e3), Next: 0}, nil},

		// Local is mainnet Gray Glacier, remote announces the same. Remote also announces a next fork
		// at block 0xffffffff, but that is uncertain.
		{params.MainnetChainConfig, 15050000, 0, ID{Hash: checksumToBytes(0xf0afd0e3), Next: math.MaxUint64}, nil},

		// Local is mainnet currently in Byzantium only (so it's aware of Petersburg), remote announces
		// also Byzantium, but it's not yet aware of Petersburg (e.g. non updated node before the fork).
		// In this case we don't know if Petersburg passed yet or not.
		{params.MainnetChainConfig, 7279999, 0, ID{Hash: checksumToBytes(0xa00bc324), Next: 0}, nil},

		// Local is mainnet currently in Byzantium only (so it's aware of Petersburg), remote announces
		// also Byzantium, and it's also aware of Petersburg (e.g. updated node before the fork). We
		// don't know if Petersburg passed yet (will pass) or not.
		{params.MainnetChainConfig, 7279999, 0, ID{Hash: checksumToBytes(0xa00bc324), Next: 7280000}, nil},

		// Local is mainnet currently in Byzantium only (so it's aware of Petersburg), remote announces
		// also Byzantium, and it's also aware of some random fork (e.g. misconfigured Petersburg). As
		// neither forks passed at neither nodes, they may mismatch, but we still connect for now.
		{params.MainnetChainConfig, 7279999, 0, ID{Hash: checksumToBytes(0xa00bc324), Next: math.MaxUint64}, nil},

		// Local is mainnet exactly on Petersburg, remote announces Byzantium + knowledge about Petersburg. Remote
		// is simply out of sync, accept.
		{params.MainnetChainConfig, 7280000, 0, ID{Hash: checksumToBytes(0xa00bc324), Next: 7280000}, nil},

		// Local is mainnet Petersburg, remote announces Byzantium + knowledge about Petersburg. Remote
		// is simply out of sync, accept.
		{params.MainnetChainConfig, 7987396, 0, ID{Hash: checksumToBytes(0xa00bc324), Next: 7280000}, nil},

		// Local is mainnet Petersburg, remote announces Spurious + knowledge about Byzantium. Remote
		// is definitely out of sync. It may or may not need the Petersburg update, we don't know yet.
		{params.MainnetChainConfig, 7987396, 0, ID{Hash: checksumToBytes(0x3edd5b10), Next: 4370000}, nil},

		// Local is mainnet Byzantium, remote announces Petersburg. Local is out of sync, accept.
		{params.MainnetChainConfig, 7279999, 0, ID{Hash: checksumToBytes(0x668db0af), Next: 0}, nil},

		// Local is mainnet Spurious, remote announces Byzantium, but is not aware of Petersburg. Local
		// out of sync. Local also knows about a future fork, but that is uncertain yet.
		{params.MainnetChainConfig, 4369999, 0, ID{Hash: checksumToBytes(0xa00bc324), Next: 0}, nil},

		// Local is mainnet Petersburg. remote announces Byzantium but is not aware of further forks.
		// Remote needs software update.
		{params.MainnetChainConfig, 7987396, 0, ID{Hash: checksumToBytes(0xa00bc324), Next: 0}, ErrRemoteStale},

		// Local is mainnet Petersburg, and isn't aware of more forks. Remote announces Petersburg +
		// 0xffffffff. Local needs software update, reject.
		{params.MainnetChainConfig, 7987396, 0, ID{Hash: checksumToBytes(0x5cddc0e1), Next: 0}, ErrLocalIncompatibleOrStale},

		// Local is mainnet Byzantium, and is aware of Petersburg. Remote announces Petersburg +
		// 0xffffffff. Local needs software update, reject.
		{params.MainnetChainConfig, 7279999, 0, ID{Hash: checksumToBytes(0x5cddc0e1), Next: 0}, ErrLocalIncompatibleOrStale},

		// Local is mainnet Petersburg, remote is Rinkeby Petersburg.
		{params.MainnetChainConfig, 7987396, 0, ID{Hash: checksumToBytes(0xafec6b27), Next: 0}, ErrLocalIncompatibleOrStale},

		// Local is mainnet Gray Glacier, far in the future. Remote announces Gopherium (non existing fork)
		// at some future block 88888888, for itself, but past block for local. Local is incompatible.
		//
		// This case detects non-upgraded nodes with majority hash power (typical Ropsten mess).
		//
		// TODO(karalabe): This testcase will fail once mainnet gets timestamped forks, make legacy chain config
		{params.MainnetChainConfig, 88888888, 0, ID{Hash: checksumToBytes(0xf0afd0e3), Next: 88888888}, ErrLocalIncompatibleOrStale},

		// Local is mainnet Byzantium. Remote is also in Byzantium, but announces Gopherium (non existing
		// fork) at block 7279999, before Petersburg. Local is incompatible.
		//
		// TODO(karalabe): This testcase will fail once mainnet gets timestamped forks, make legacy chain config
		{params.MainnetChainConfig, 7279999, 0, ID{Hash: checksumToBytes(0xa00bc324), Next: 7279999}, ErrLocalIncompatibleOrStale},

		//------------------------------------
		// Block to timestamp transition tests
		//------------------------------------

		// Local is mainnet currently in Gray Glacier only (so it's aware of Shanghai), remote announces
		// also Gray Glacier, but it's not yet aware of Shanghai (e.g. non updated node before the fork).
		// In this case we don't know if Shanghai passed yet or not.
		{&timestampedConfig, 15050000, 0, ID{Hash: checksumToBytes(0xf0afd0e3), Next: 0}, nil},

		// Local is mainnet currently in Gray Glacier only (so it's aware of Shanghai), remote announces
		// also Gray Glacier, and it's also aware of Shanghai (e.g. updated node before the fork). We
		// don't know if Shanghai passed yet (will pass) or not.
		{&timestampedConfig, 15050000, 0, ID{Hash: checksumToBytes(0xf0afd0e3), Next: 1668000000}, nil},

		// Local is mainnet currently in Gray Glacier only (so it's aware of Shanghai), remote announces
		// also Gray Glacier, and it's also aware of some random fork (e.g. misconfigured Shanghai). As
		// neither forks passed at neither nodes, they may mismatch, but we still connect for now.
		{&timestampedConfig, 15050000, 0, ID{Hash: checksumToBytes(0xf0afd0e3), Next: math.MaxUint64}, nil},

		// Local is mainnet exactly on Shanghai, remote announces Gray Glacier + knowledge about Shanghai. Remote
		// is simply out of sync, accept.
		{&timestampedConfig, 20000000, 1668000000, ID{Hash: checksumToBytes(0xf0afd0e3), Next: 1668000000}, nil},

		// Local is mainnet Shanghai, remote announces Gray Glacier + knowledge about Shanghai. Remote
		// is simply out of sync, accept.
		{&timestampedConfig, 20123456, 1668123456, ID{Hash: checksumToBytes(0xf0afd0e3), Next: 1668000000}, nil},

		// Local is mainnet Shanghai, remote announces Arrow Glacier + knowledge about Gray Glacier. Remote
		// is definitely out of sync. It may or may not need the Shanghai update, we don't know yet.
		{&timestampedConfig, 20000000, 1668000000, ID{Hash: checksumToBytes(0x20c327fc), Next: 15050000}, nil},

		// Local is mainnet Gray Glacier, remote announces Shanghai. Local is out of sync, accept.
		{&timestampedConfig, 15050000, 0, ID{Hash: checksumToBytes(0x71147644), Next: 0}, nil},

		// Local is mainnet Arrow Glacier, remote announces Gray Glacier, but is not aware of Shanghai. Local
		// out of sync. Local also knows about a future fork, but that is uncertain yet.
		{&timestampedConfig, 13773000, 0, ID{Hash: checksumToBytes(0xf0afd0e3), Next: 0}, nil},

		// Local is mainnet Shanghai. remote announces Gray Glacier but is not aware of further forks.
		// Remote needs software update.
		{&timestampedConfig, 20000000, 1668000000, ID{Hash: checksumToBytes(0xf0afd0e3), Next: 0}, ErrRemoteStale},

		// Local is mainnet Gray Glacier, and isn't aware of more forks. Remote announces Gray Glacier +
		// 0xffffffff. Local needs software update, reject.
		{&timestampedConfig, 15050000, 0, ID{Hash: checksumToBytes(checksumUpdate(0xf0afd0e3, math.MaxUint64)), Next: 0}, ErrLocalIncompatibleOrStale},

		// Local is mainnet Gray Glacier, and is aware of Shanghai. Remote announces Shanghai +
		// 0xffffffff. Local needs software update, reject.
		{&timestampedConfig, 15050000, 0, ID{Hash: checksumToBytes(checksumUpdate(0x71147644, math.MaxUint64)), Next: 0}, ErrLocalIncompatibleOrStale},

		// Local is mainnet Gray Glacier, far in the future. Remote announces Gopherium (non existing fork)
		// at some future timestamp 8888888888, for itself, but past block for local. Local is incompatible.
		//
		// This case detects non-upgraded nodes with majority hash power (typical Ropsten mess).
		{params.MainnetChainConfig, 888888888, 1660000000, ID{Hash: checksumToBytes(0xf0afd0e3), Next: 1660000000}, ErrLocalIncompatibleOrStale},

		// Local is mainnet Gray Glacier. Remote is also in Gray Glacier, but announces Gopherium (non existing
		// fork) at block 7279999, before Shanghai. Local is incompatible.
		{&timestampedConfig, 19999999, 1667999999, ID{Hash: checksumToBytes(0xf0afd0e3), Next: 1667999999}, ErrLocalIncompatibleOrStale},

		//----------------------
		// Timestamp based tests
		//----------------------

		// Local is mainnet Shanghai, remote announces the same. No future fork is announced.
		{&timestampedConfig, 20000000, 1668000000, ID{Hash: checksumToBytes(0x71147644), Next: 0}, nil},

		// Local is mainnet Shanghai, remote announces the same. Remote also announces a next fork
		// at time 0xffffffff, but that is uncertain.
		{&timestampedConfig, 20000000, 1668000000, ID{Hash: checksumToBytes(0x71147644), Next: math.MaxUint64}, nil},

		// Local is mainnet currently in Shanghai only (so it's aware of Cancun), remote announces
		// also Shanghai, but it's not yet aware of Cancun (e.g. non updated node before the fork).
		// In this case we don't know if Cancun passed yet or not.
		//
		// TODO(karalabe): Enable this when Cancun is specced
		//{&timestampedConfig, 20000000, 1668000000, ID{Hash: checksumToBytes(0x71147644), Next: 0}, nil},

		// Local is mainnet currently in Shanghai only (so it's aware of Cancun), remote announces
		// also Shanghai, and it's also aware of Cancun (e.g. updated node before the fork). We
		// don't know if Cancun passed yet (will pass) or not.
		//
		// TODO(karalabe): Enable this when Cancun is specced and update next timestamp
		//{&timestampedConfig, 20000000, 1668000000, ID{Hash: checksumToBytes(0x71147644), Next: 1678000000}, nil},

		// Local is mainnet currently in Shanghai only (so it's aware of Cancun), remote announces
		// also Shanghai, and it's also aware of some random fork (e.g. misconfigured Cancun). As
		// neither forks passed at neither nodes, they may mismatch, but we still connect for now.
		//
		// TODO(karalabe): Enable this when Cancun is specced
		//{&timestampedConfig, 20000000, 1668000000, ID{Hash: checksumToBytes(0x71147644), Next: math.MaxUint64}, nil},

		// Local is mainnet exactly on Cancun, remote announces Shanghai + knowledge about Cancun. Remote
		// is simply out of sync, accept.
		//
		// TODO(karalabe): Enable this when Cancun is specced, update local head and time, next timestamp
		// {&timestampedConfig, 21000000, 1678000000, ID{Hash: checksumToBytes(0x71147644), Next: 1678000000}, nil},

		// Local is mainnet Cancun, remote announces Shanghai + knowledge about Cancun. Remote
		// is simply out of sync, accept.
		// TODO(karalabe): Enable this when Cancun is specced, update local head and time, next timestamp
		//{&timestampedConfig, 21123456, 1678123456, ID{Hash: checksumToBytes(0x71147644), Next: 1678000000}, nil},

		// Local is mainnet Prague, remote announces Shanghai + knowledge about Cancun. Remote
		// is definitely out of sync. It may or may not need the Prague update, we don't know yet.
		//
		// TODO(karalabe): Enable this when Cancun **and** Prague is specced, update all the numbers
		//{&timestampedConfig, 0, 0, ID{Hash: checksumToBytes(0x3edd5b10), Next: 4370000}, nil},

		// Local is mainnet Shanghai, remote announces Cancun. Local is out of sync, accept.
		//
		// TODO(karalabe): Enable this when Cancun is specced, update remote checksum
		//{&timestampedConfig, 21000000, 1678000000, ID{Hash: checksumToBytes(0x00000000), Next: 0}, nil},

		// Local is mainnet Shanghai, remote announces Cancun, but is not aware of Prague. Local
		// out of sync. Local also knows about a future fork, but that is uncertain yet.
		//
		// TODO(karalabe): Enable this when Cancun **and** Prague is specced, update remote checksum
		//{&timestampedConfig, 21000000, 1678000000, ID{Hash: checksumToBytes(0x00000000), Next: 0}, nil},

		// Local is mainnet Cancun. remote announces Shanghai but is not aware of further forks.
		// Remote needs software update.
		//
		// TODO(karalabe): Enable this when Cancun is specced, update local head and time
		//{&timestampedConfig, 21000000, 1678000000, ID{Hash: checksumToBytes(0x71147644), Next: 0}, ErrRemoteStale},

		// Local is mainnet Shanghai, and isn't aware of more forks. Remote announces Shanghai +
		// 0xffffffff. Local needs software update, reject.
		{&timestampedConfig, 20000000, 1668000000, ID{Hash: checksumToBytes(checksumUpdate(0x71147644, math.MaxUint64)), Next: 0}, ErrLocalIncompatibleOrStale},

		// Local is mainnet Shanghai, and is aware of Cancun. Remote announces Cancun +
		// 0xffffffff. Local needs software update, reject.
		//
		// TODO(karalabe): Enable this when Cancun is specced, update remote checksum
		//{&timestampedConfig, 20000000, 1668000000, ID{Hash: checksumToBytes(checksumUpdate(0x00000000, math.MaxUint64)), Next: 0}, ErrLocalIncompatibleOrStale},

		// Local is mainnet Shanghai, remote is random Shanghai.
		{&timestampedConfig, 20000000, 1668000000, ID{Hash: checksumToBytes(0x12345678), Next: 0}, ErrLocalIncompatibleOrStale},

		// Local is mainnet Shanghai, far in the future. Remote announces Gopherium (non existing fork)
		// at some future timestamp 8888888888, for itself, but past block for local. Local is incompatible.
		//
		// This case detects non-upgraded nodes with majority hash power (typical Ropsten mess).
		{&timestampedConfig, 88888888, 8888888888, ID{Hash: checksumToBytes(0x71147644), Next: 8888888888}, ErrLocalIncompatibleOrStale},

		// Local is mainnet Shanghai. Remote is also in Shanghai, but announces Gopherium (non existing
		// fork) at timestamp 1668000000, before Cancun. Local is incompatible.
		//
		// TODO(karalabe): Enable this when Cancun is specced
		//{params.MainnetChainConfig, 20999999, 1677999999, ID{Hash: checksumToBytes(0x71147644), Next: 1678000000}, ErrLocalIncompatibleOrStale},
	}
	for i, tt := range tests {
		filter := newFilter(tt.config, params.MainnetGenesisHash, func() (uint64, uint64) { return tt.head, tt.time })
		if err := filter(tt.id); err != tt.err {
			t.Errorf("test %d: validation error mismatch: have %v, want %v", i, err, tt.err)
		}
	}
}

// Tests that IDs are properly RLP encoded (specifically important because we
// use uint32 to store the hash, but we need to encode it as [4]byte).
func TestEncoding(t *testing.T) {
	tests := []struct {
		id   ID
		want []byte
	}{
		{ID{Hash: checksumToBytes(0), Next: 0}, common.Hex2Bytes("c6840000000080")},
		{ID{Hash: checksumToBytes(0xdeadbeef), Next: 0xBADDCAFE}, common.Hex2Bytes("ca84deadbeef84baddcafe,")},
		{ID{Hash: checksumToBytes(math.MaxUint32), Next: math.MaxUint64}, common.Hex2Bytes("ce84ffffffff88ffffffffffffffff")},
	}
	for i, tt := range tests {
		have, err := rlp.EncodeToBytes(tt.id)
		if err != nil {
			t.Errorf("test %d: failed to encode forkid: %v", i, err)
			continue
		}
		if !bytes.Equal(have, tt.want) {
			t.Errorf("test %d: RLP mismatch: have %x, want %x", i, have, tt.want)
		}
	}
}
