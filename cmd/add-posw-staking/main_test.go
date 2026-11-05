// Copyright 2026-2027 The quarkchain Authors
// This file is part of the quarkchain library.
//
// The quarkchain library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The quarkchain library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the quarkchain library. If not, see <http://www.gnu.org/licenses/>.

package main

import (
	"bytes"
	"encoding/json"
	"math/big"
	"reflect"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/params"
)

func stakingGenesisConfig(chainID int64) *params.ChainConfig {
	config := *params.AllEthashProtocolChanges
	config.ChainID = big.NewInt(chainID)
	config.QuarkChain = params.NewTestQuarkChainConfig(chainID).QuarkChain
	return &config
}

func TestGenerate(t *testing.T) {
	for _, chainID := range []int64{10001, 10002} {
		config := stakingGenesisConfig(chainID)
		funded := common.HexToAddress("0xabcd")
		// Generation preserves configured accounts and fills missing staking accounts.
		base := &core.Genesis{
			Config: config, GasLimit: 12_000_000, Difficulty: big.NewInt(1),
			Alloc: types.GenesisAlloc{
				funded: {Balance: big.NewInt(42), Code: []byte{0}},
				params.ShardStakingContractAddress: {
					Balance: big.NewInt(99), Nonce: 9, Code: params.PoSWStakingRuntimeCode,
					Storage: map[common.Hash]common.Hash{{1}: {2}},
				},
			},
		}
		input, err := json.Marshal(base)
		if err != nil {
			t.Fatal(err)
		}
		var output bytes.Buffer
		if err := generate(bytes.NewReader(input), &output); err != nil {
			t.Fatal(err)
		}
		var configured core.Genesis
		if err := json.Unmarshal(output.Bytes(), &configured); err != nil {
			t.Fatal(err)
		}
		if err := configured.ValidatePoSWStakingContracts(); err != nil {
			t.Fatal(err)
		}
		account := configured.Alloc[params.ShardStakingContractAddress]
		if configured.Config.ChainID.Int64() != chainID || !reflect.DeepEqual(account, base.Alloc[params.ShardStakingContractAddress]) {
			t.Fatal("configured staking account changed")
		}
		if configured.Alloc[funded].Balance.Uint64() != 42 || !bytes.Equal(configured.Alloc[funded].Code, []byte{0}) {
			t.Fatal("ordinary allocation lost")
		}
		wantLen := 2
		if chainID == 10001 {
			wantLen++
		}
		if len(configured.Alloc) != wantLen {
			t.Fatal("incorrect staking deployment scope")
		}
		var repeated bytes.Buffer
		if err := generate(bytes.NewReader(output.Bytes()), &repeated); err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(output.Bytes(), repeated.Bytes()) {
			t.Fatal("genesis generation is not idempotent")
		}
	}
}

func TestGenerateRootStakingOnFirstChain(t *testing.T) {
	id := new(big.Int).Add(new(big.Int).Lsh(big.NewInt(1), 128), big.NewInt(10002))
	for _, test := range []struct {
		name     string
		chainIDs []*big.Int
		wantRoot bool
	}{
		{"first chain", []*big.Int{new(big.Int).Set(id), big.NewInt(10001)}, true},
		{"other chain", []*big.Int{big.NewInt(10001), new(big.Int).Set(id)}, false},
	} {
		t.Run(test.name, func(t *testing.T) {
			config := stakingGenesisConfig(10002)
			config.ChainID = new(big.Int).Set(id)
			config.QuarkChain.ChainIDs = test.chainIDs
			input, err := json.Marshal(&core.Genesis{Config: config, GasLimit: 12_000_000, Difficulty: big.NewInt(1), Alloc: types.GenesisAlloc{}})
			if err != nil {
				t.Fatal(err)
			}
			var output bytes.Buffer
			if err := generate(bytes.NewReader(input), &output); err != nil {
				t.Fatal(err)
			}
			var genesis core.Genesis
			if err := json.Unmarshal(output.Bytes(), &genesis); err != nil {
				t.Fatal(err)
			}
			if genesis.Config.ChainID.Cmp(id) != 0 || !genesis.Config.QuarkChain.Equal(config.QuarkChain) {
				t.Fatal("large chain ID or configured chain order lost")
			}
			if gotRoot := len(genesis.Alloc[params.RootStakingContractAddress].Code) > 0; gotRoot != test.wantRoot {
				t.Fatalf("root staking deployment: have %v, want %v", gotRoot, test.wantRoot)
			}
		})
	}
}

func TestGenerateRejectsInvalidGenesis(t *testing.T) {
	config := stakingGenesisConfig(10002)
	wrongShard, err := json.Marshal(&core.Genesis{Config: config, GasLimit: 12_000_000, Difficulty: big.NewInt(1), Alloc: core.PoSWStakingContractAllocs(params.NewTestQuarkChainConfig(10001))})
	if err != nil {
		t.Fatal(err)
	}
	occupied, err := json.Marshal(&core.Genesis{
		Config: config, GasLimit: 12_000_000, Difficulty: big.NewInt(1),
		Alloc: types.GenesisAlloc{params.ShardStakingContractAddress: {Balance: big.NewInt(99)}},
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, input := range [][]byte{
		[]byte("invalid JSON"),
		[]byte(`{"gasLimit":"0xb71b00","difficulty":"0x1","alloc":{}}`),
		wrongShard,
		occupied,
	} {
		var output bytes.Buffer
		if err := generate(bytes.NewReader(input), &output); err == nil {
			t.Fatal("invalid genesis accepted")
		}
		if output.Len() != 0 {
			t.Fatal("generation emitted a partial configuration")
		}
	}
}
