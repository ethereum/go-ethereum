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

func TestGenerate(t *testing.T) {
	for _, shardID := range []uint32{0, 1} {
		config := *params.AllEthashProtocolChanges
		config.QuarkChain = &params.QuarkChainConfig{ShardID: shardID}
		funded := common.HexToAddress("0xabcd")
		// Generation preserves configured accounts and fills missing staking accounts.
		base := &core.Genesis{
			Config: &config, GasLimit: 12_000_000, Difficulty: big.NewInt(1),
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
		if configured.Config.QuarkChain.ShardID != shardID || !reflect.DeepEqual(account, base.Alloc[params.ShardStakingContractAddress]) {
			t.Fatal("configured staking account changed")
		}
		if configured.Alloc[funded].Balance.Uint64() != 42 || !bytes.Equal(configured.Alloc[funded].Code, []byte{0}) {
			t.Fatal("ordinary allocation lost")
		}
		wantLen := 2
		if shardID == 0 {
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

func TestGenerateRejectsInvalidGenesis(t *testing.T) {
	config := *params.AllEthashProtocolChanges
	config.QuarkChain = &params.QuarkChainConfig{ShardID: 1}
	wrongShard, err := json.Marshal(&core.Genesis{Config: &config, GasLimit: 12_000_000, Difficulty: big.NewInt(1), Alloc: core.PoSWStakingContractAllocs(0)})
	if err != nil {
		t.Fatal(err)
	}
	occupied, err := json.Marshal(&core.Genesis{
		Config: &config, GasLimit: 12_000_000, Difficulty: big.NewInt(1),
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
