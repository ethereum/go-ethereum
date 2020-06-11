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

package params

import (
	"github.com/ethereum/go-ethereum/common"
	"math/big"
	"reflect"
	"testing"
)

func TestCheckCompatible(t *testing.T) {
	type test struct {
		stored, new *ChainConfig
		head        uint64
		wantErr     *ConfigCompatError
	}
	tests := []test{
		{stored: AllEthashProtocolChanges, new: AllEthashProtocolChanges, head: 0, wantErr: nil},
		{stored: AllEthashProtocolChanges, new: AllEthashProtocolChanges, head: 100, wantErr: nil},
		{
			stored:  &ChainConfig{EIP150Block: big.NewInt(10)},
			new:     &ChainConfig{EIP150Block: big.NewInt(20)},
			head:    9,
			wantErr: nil,
		},
		{
			stored: AllEthashProtocolChanges,
			new:    &ChainConfig{HomesteadBlock: nil},
			head:   3,
			wantErr: &ConfigCompatError{
				What:         "Homestead fork block",
				StoredConfig: big.NewInt(0),
				NewConfig:    nil,
				RewindTo:     0,
			},
		},
		{
			stored: AllEthashProtocolChanges,
			new:    &ChainConfig{HomesteadBlock: big.NewInt(1)},
			head:   3,
			wantErr: &ConfigCompatError{
				What:         "Homestead fork block",
				StoredConfig: big.NewInt(0),
				NewConfig:    big.NewInt(1),
				RewindTo:     0,
			},
		},
		{
			stored: &ChainConfig{HomesteadBlock: big.NewInt(30), EIP150Block: big.NewInt(10)},
			new:    &ChainConfig{HomesteadBlock: big.NewInt(25), EIP150Block: big.NewInt(20)},
			head:   25,
			wantErr: &ConfigCompatError{
				What:         "EIP150 fork block",
				StoredConfig: big.NewInt(10),
				NewConfig:    big.NewInt(20),
				RewindTo:     9,
			},
		},
	}

	for _, test := range tests {
		err := test.stored.CheckCompatible(test.new, test.head)
		if !reflect.DeepEqual(err, test.wantErr) {
			t.Errorf("error mismatch:\nstored: %v\nnew: %v\nhead: %v\nerr: %v\nwant: %v", test.stored, test.new, test.head, err, test.wantErr)
		}
	}
}

func TestSortedForkList(t *testing.T) {
	var cases = []struct{
		config *ChainConfig
		blocks []uint64
		hashes []common.Hash
	} {
		{
			MainnetChainConfig,
			[]uint64{1150000, 2463000, 2675000, 4370000, 7280000, 9069000, 9200000},
			[]common.Hash{
				common.HexToHash("0x584bdb5d4e74fe97f5a5222b533fe1322fd0b6ad3eb03f02c3221984e2c0b430"),
				common.HexToHash("0x2086799aeebeae135c246c65021c82b4e15a2c451340993aacfd2751886514f0"),
				common.HexToHash("0x58eff9265aedf8a54da8121de1324e1e0d9aac99f694d16c6a41afffe3817d73"),
				common.HexToHash("0xb1fcff633029ee18ab6482b58ff8b6e95dd7c82a954c852157152a7a6d32785e"),
				common.HexToHash("0xeddb0590e1095fbe51205a51a297daef7259e229af0432214ae6cb2c1f750750"),
				common.HexToHash("0x451226b98bf4f784314e9ca2daaa30dc664a387c342ef775ba2d88682a27c084"),
				common.HexToHash("0x6ba9486095de7d96a75b67954cfe2581234eae1ef2a92ab03b84fc2eae2deb8a"),
			},
		},
	}
	for _, c := range cases {
		blocks, hashes := c.config.SortedForkCheckList()
		if !reflect.DeepEqual(blocks, c.blocks) {
			t.Fatalf("The fork number list is different")
		}
		if !reflect.DeepEqual(hashes, c.hashes) {
			t.Fatalf("The fork hash list is different")
		}
	}
}