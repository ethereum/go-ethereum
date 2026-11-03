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

package core

import (
	"errors"
	"math/big"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/consensus"
	"github.com/ethereum/go-ethereum/consensus/ethash"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/tracing"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/params"
)

func TestLockedStakesParentState(t *testing.T) {
	locked, unlocked, funded := common.HexToAddress("0x1234"), common.HexToAddress("0x5678"), common.HexToAddress("0x9abc")
	// Solidity mapping(address => Stake) at slot zero, amount at offset two.
	slot := func(address common.Address, offset int64) common.Hash {
		key := crypto.Keccak256Hash(common.LeftPadBytes(address[:], 32), make([]byte, 32))
		return common.BigToHash(new(big.Int).Add(key.Big(), big.NewInt(offset)))
	}
	alloc := PoSWStakingContractAllocs(0)
	contract := alloc[params.ShardStakingContractAddress]
	contract.Storage = map[common.Hash]common.Hash{
		slot(locked, 2):   common.BigToHash(big.NewInt(123)),
		slot(locked, 3):   common.BytesToHash(unlocked[:]), // Root signer must not change shard stake ownership.
		slot(unlocked, 0): common.BigToHash(common.Big1),
		slot(unlocked, 2): common.BigToHash(big.NewInt(789)),
	}
	alloc[params.ShardStakingContractAddress] = contract
	rootContract := alloc[params.RootStakingContractAddress]
	rootContract.Storage = map[common.Hash]common.Hash{
		slot(locked, 2):   common.BigToHash(big.NewInt(321)),
		slot(locked, 3):   common.BytesToHash(funded[:]),
		slot(unlocked, 0): common.BigToHash(common.Big1),
		slot(unlocked, 2): common.BigToHash(big.NewInt(789)),
		slot(funded, 2):   common.BigToHash(big.NewInt(654)), // No explicit signer: use the staker.
	}
	alloc[params.RootStakingContractAddress] = rootContract
	alloc[funded] = types.Account{Balance: big.NewInt(10000)}
	genesis := &Genesis{Config: genesisConfig(0), GasLimit: 12000000, Difficulty: new(big.Int), Alloc: alloc}
	db := rawdb.NewMemoryDatabase()
	defer db.Close()
	chain, err := NewBlockChain(db, genesis, ethash.NewFaker(), DefaultConfig().WithStateScheme(rawdb.HashScheme))
	if err != nil {
		t.Fatal(err)
	}
	defer chain.Stop()
	parent := chain.Genesis().Header()
	for _, test := range []struct {
		address common.Address
		want    uint64
	}{{locked, 123}, {unlocked, 0}, {funded, 0}} {
		got, err := chain.GetShardLockedStakes(parent, test.address)
		if err != nil || got.Uint64() != test.want {
			t.Fatalf("%s: %v, %v", test.address, got, err)
		}
	}
	for _, test := range []struct {
		address common.Address
		want    uint64
		signer  common.Address
	}{{locked, 321, funded}, {unlocked, 0, common.Address{}}, {funded, 654, funded}} {
		got, signer, err := chain.GetRootChainStakes(parent, test.address)
		if err != nil || got.Uint64() != test.want || signer != test.signer {
			t.Fatalf("root stakes for %s: %v, signer %s, error %v", test.address, got, signer, err)
		}
	}
	statedb, err := chain.StateAt(parent.Root, parent.Number, parent.Time)
	if err != nil {
		t.Fatal(err)
	}
	statedb.SetState(params.ShardStakingContractAddress, slot(locked, 2), common.BigToHash(big.NewInt(456)))
	statedb.SetState(params.RootStakingContractAddress, slot(locked, 2), common.BigToHash(big.NewInt(987)))
	statedb.SetState(params.RootStakingContractAddress, slot(locked, 3), common.BytesToHash(unlocked[:]))
	fork := types.CopyHeader(parent)
	fork.Root, err = statedb.Commit(chain.Config().Rules(parent.Number, true, parent.Time), 1)
	if err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct {
		parent   *types.Header
		want     uint64
		rootWant uint64
		signer   common.Address
	}{{fork, 456, 987, unlocked}, {parent, 123, 321, funded}} {
		got, err := chain.GetShardLockedStakes(test.parent, locked)
		if err != nil || got.Uint64() != test.want {
			t.Fatalf("fork state: %v %v", got, err)
		}
		rootGot, signer, err := chain.GetRootChainStakes(test.parent, locked)
		if err != nil || rootGot.Uint64() != test.rootWant || signer != test.signer {
			t.Fatalf("root fork state: %v, signer %s, error %v", rootGot, signer, err)
		}
	}
	missingState := types.CopyHeader(parent)
	missingState.Root = common.HexToHash("0xdead")
	if _, err := chain.GetShardLockedStakes(missingState, locked); !errors.Is(err, consensus.ErrPrunedAncestor) {
		t.Fatalf("missing shard state: have %v, want ErrPrunedAncestor", err)
	}
	if _, _, err := chain.GetRootChainStakes(missingState, locked); !errors.Is(err, consensus.ErrPrunedAncestor) {
		t.Fatalf("missing root staking state: have %v, want ErrPrunedAncestor", err)
	}
}

func TestRootChainStakesShard0Only(t *testing.T) {
	for _, config := range []*params.QuarkChainConfig{nil, {ChainID: 1}} {
		chain := &BlockChain{chainConfig: &params.ChainConfig{QuarkChain: config}}
		if _, _, err := chain.GetRootChainStakes(&types.Header{}, common.Address{}); err == nil {
			t.Fatal("accepted root-chain stakes query outside shard 0")
		}
	}
}

func TestLockedStakesErrors(t *testing.T) {
	db := rawdb.NewMemoryDatabase()
	defer db.Close()
	genesis := &Genesis{Config: genesisConfig(0), GasLimit: 12000000, Difficulty: new(big.Int), Alloc: PoSWStakingContractAllocs(0)}
	chain, err := NewBlockChain(db, genesis, ethash.NewFaker(), DefaultConfig().WithStateScheme(rawdb.HashScheme))
	if err != nil {
		t.Fatal(err)
	}
	defer chain.Stop()
	for _, test := range []struct {
		name        string
		missingBlob bool
		wantError   string
	}{
		{name: "wrong runtime", wantError: "invalid staking contract runtime"},
		{name: "missing code blob", missingBlob: true, wantError: "code is not found"},
	} {
		t.Run(test.name, func(t *testing.T) {
			header := chain.Genesis().Header()
			if test.missingBlob {
				// Keep the account's valid code hash, but force a database read of
				// missing bytecode. This exercises the lazy StateDB error path.
				rawdb.DeleteCode(db, params.PoSWStakingRuntimeCodeHash)
				chain.codedb = state.NewCodeDB(db)
			} else {
				statedb, err := chain.StateAt(header.Root, header.Number, header.Time)
				if err != nil {
					t.Fatal(err)
				}
				statedb.SetCode(params.RootStakingContractAddress, []byte{0x00}, tracing.CodeChangeUnspecified)
				header.Root, err = statedb.Commit(chain.Config().Rules(header.Number, true, header.Time), 1)
				if err != nil {
					t.Fatal(err)
				}
			}
			if stakes, signer, err := chain.GetRootChainStakes(header, common.Address{}); err == nil || !strings.Contains(err.Error(), test.wantError) || stakes != nil || signer != (common.Address{}) {
				t.Fatalf("root stakes: got (%v, %s, %v), want %q error and no stakes or signer", stakes, signer, err, test.wantError)
			}
		})
	}
}
