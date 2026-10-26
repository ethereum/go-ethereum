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
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/consensus/ethash"
	"github.com/ethereum/go-ethereum/core/rawdb"
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
	contract.Balance = big.NewInt(1000)
	contract.Storage = map[common.Hash]common.Hash{
		slot(locked, 2):   common.BigToHash(big.NewInt(123)),
		slot(locked, 3):   common.BytesToHash(unlocked[:]), // Root signer must not change shard stake ownership.
		slot(unlocked, 0): common.BigToHash(common.Big1),
		slot(unlocked, 2): common.BigToHash(big.NewInt(789)),
	}
	alloc[params.ShardStakingContractAddress] = contract
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
		got, err := chain.GetLockedStakes(parent, test.address)
		if err != nil || got.Uint64() != test.want {
			t.Fatalf("%s: %v, %v", test.address, got, err)
		}
	}
	statedb, err := chain.StateAt(parent.Root, parent.Number, parent.Time)
	if err != nil {
		t.Fatal(err)
	}
	statedb.SetState(params.ShardStakingContractAddress, slot(locked, 2), common.BigToHash(big.NewInt(456)))
	fork := types.CopyHeader(parent)
	fork.Root, err = statedb.Commit(chain.Config().Rules(parent.Number, true, parent.Time), 1)
	if err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct {
		parent *types.Header
		want   uint64
	}{{fork, 456}, {parent, 123}} {
		got, err := chain.GetLockedStakes(test.parent, locked)
		if err != nil || got.Uint64() != test.want {
			t.Fatalf("fork state: %v %v", got, err)
		}
	}
}
