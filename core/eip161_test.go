// Copyright 2026 The go-ethereum Authors
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

package core

import (
	"crypto/ecdsa"
	"encoding/binary"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/consensus/beacon"
	"github.com/ethereum/go-ethereum/consensus/ethash"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/trie/bintrie"
)

// EIP-161 removes an account that is touched while empty. A touch changes no
// value, so it went unstated in the block access list - and the parallel
// processor derives the post-state root from that list alone, so the two
// executors disagreed. The fixtures here drive both.

// eip161Genesis returns a genesis holding a funded sender and an EIP-161-empty
// victim: zero nonce, zero balance, no code, and slots storage slots.
func eip161Genesis(t *testing.T, config params.ChainConfig, slots int) (*Genesis, *ecdsa.PrivateKey, common.Address, common.Address) {
	t.Helper()

	key, err := crypto.HexToECDSA("b71c71a67e1177ad4e901695e1b4b9ee17ae16c6668d313eac2f96dbcda3f291")
	if err != nil {
		t.Fatal(err)
	}
	config.Ethash = nil

	var (
		sender = crypto.PubkeyToAddress(key.PublicKey)
		victim = common.Address{0xde, 0xad}
		acct   = types.Account{Balance: new(big.Int)}
	)
	if slots > 0 {
		acct.Storage = pbtPrefilledStorage(pbtStorageSlots(slots))
	}
	genesis := &Genesis{
		Config:     &config,
		Difficulty: big.NewInt(0),
		BaseFee:    big.NewInt(params.InitialBaseFee),
		GasLimit:   pbtCodeBlockGas,
		Alloc: types.GenesisAlloc{
			sender: {Balance: new(big.Int).Mul(big.NewInt(1000), big.NewInt(params.Ether))},
			victim: acct,
		},
	}
	// Padding, so the deletion has branches to collapse rather than folding the
	// whole state into one record.
	pbtPadGenesis(genesis, 64)
	return genesis, key, sender, victim
}

// eip161ChainConfig returns a blockchain config whose state scheme follows the
// tree, optionally pinned to the sequential processor.
func eip161ChainConfig(config *params.ChainConfig, sequential bool) *BlockChainConfig {
	cfg := DefaultConfig()
	if config.IsPBT() {
		cfg = cfg.WithStateScheme(rawdb.PathScheme)
	}
	cfg.VmConfig.DisableParallelExecution = sequential
	return cfg
}

// TestEIP161ClearingAgreesAcrossExecutors pins that a block clearing a touched
// empty account imports the same way through both processors, on both trees -
// generation roots sequentially, so each arm passing is what proves agreement.
func TestEIP161ClearingAgreesAcrossExecutors(t *testing.T) {
	merkle := *testPBTChainConfig
	merkle.BinaryTrieTime = nil

	for _, tc := range []struct {
		name   string
		config params.ChainConfig
		slots  int
	}{
		{"pbt/victim owns storage", *testPBTChainConfig, 48},
		{"pbt/victim owns nothing", *testPBTChainConfig, 0},
		{"merkle/victim owns storage", merkle, 48},
		{"merkle/victim owns nothing", merkle, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			for _, sequential := range []bool{false, true} {
				name := "parallel"
				if sequential {
					name = "sequential"
				}
				t.Run(name, func(t *testing.T) {
					eip161Clearing(t, tc.config, tc.slots, sequential)
				})
			}
		})
	}
}

func eip161Clearing(t *testing.T, config params.ChainConfig, slots int, sequential bool) {
	t.Helper()

	genesis, key, sender, victim := eip161Genesis(t, config, slots)
	engine := beacon.New(ethash.NewFaker())

	// A zero-value call touches the victim without funding it, which is what
	// makes EIP-161 sweep it.
	db, blocks, _ := GenerateChainWithGenesis(genesis, engine, 1, pbtCallTx(t, key, sender, victim))

	chain, err := NewBlockChain(db, genesis, engine, eip161ChainConfig(genesis.Config, sequential))
	if err != nil {
		t.Fatal(err)
	}
	defer chain.Stop()

	// Pin which processor ran, or a fixture whose block lost its access list
	// would quietly test the sequential path twice.
	if got := chain.useBALExecution(blocks[0], false); got == sequential {
		t.Fatalf("expected BAL execution to be %v, got %v", !sequential, got)
	}
	if _, err := chain.InsertChain(blocks); err != nil {
		t.Fatalf("importing a block that clears a touched empty account: %v", err)
	}
	state, err := chain.State()
	if err != nil {
		t.Fatal(err)
	}
	if state.Exist(victim) {
		t.Fatal("the touched empty account survived the import")
	}
	// Its storage has to have gone with it, or the account is absent while the
	// slots it owned are still readable.
	for _, slot := range pbtStorageSlots(slots) {
		key := common.BigToHash(new(big.Int).SetUint64(slot))
		if got := state.GetState(victim, key); got != (common.Hash{}) {
			t.Fatalf("slot %d of a deleted account still holds %x", slot, got)
		}
	}
}

// TestEIP161TouchOfAbsentAccountLeavesBALAlone pins the guard's other half: a
// zero-tip coinbase that never existed is touched, found empty and dropped in
// an ordinary block, and recording that would change BlockAccessListHash.
func TestEIP161TouchOfAbsentAccountLeavesBALAlone(t *testing.T) {
	genesis, key, sender, _ := eip161Genesis(t, *testPBTChainConfig, 0)
	delete(genesis.Alloc, common.Address{0xde, 0xad}) // no pre-existing empty account here
	engine := beacon.New(ethash.NewFaker())

	coinbase := common.Address{0xc0, 0x1b} // absent from the alloc above
	_, blocks, _ := GenerateChainWithGenesis(genesis, engine, 1, func(i int, gen *BlockGen) {
		gen.SetCoinbase(coinbase)
		// Exactly the base fee, so the effective tip - and the fee - is zero.
		tx, err := types.SignTx(types.NewTransaction(
			gen.TxNonce(sender), common.Address{0x0f, 0xf1}, big.NewInt(1),
			pbtTestTxGas, gen.BaseFee(), nil,
		), gen.Signer(), key)
		if err != nil {
			t.Fatal(err)
		}
		gen.AddTx(tx)
	})
	list := blocks[0].AccessList()
	if list == nil {
		t.Fatal("the generated block carries no access list")
	}
	for _, account := range *list {
		if account.Address != coinbase {
			continue
		}
		if len(account.BalanceChanges) != 0 {
			t.Fatalf("an account that never existed recorded %d balance changes when it was touched and dropped",
				len(account.BalanceChanges))
		}
	}
}

// pbtStorageSlots is a spread of slot numbers at and above HeaderStorageOffset
// (64), so they land in the overflow bucket rather than in the account's
// header stem. The spread is wide enough that they occupy several stems.
func pbtStorageSlots(n int) []uint64 {
	slots := make([]uint64, n)
	for i := range slots {
		slots[i] = bintrie.HeaderStorageOffset + uint64(i)*0x10001
	}
	return slots
}

// pbtPadGenesis adds accounts so the tree has branches for a deletion to walk
// through, rather than folding the whole state into a single group record.
func pbtPadGenesis(genesis *Genesis, n int) {
	for i := range n {
		var addr common.Address
		binary.BigEndian.PutUint64(addr[:8], uint64(i)+1)
		addr[19] = 0xad
		genesis.Alloc[addr] = types.Account{Balance: big.NewInt(int64(i) + 1)}
	}
}

// pbtPrefilledStorage turns slot numbers into a genesis storage map.
func pbtPrefilledStorage(slots []uint64) map[common.Hash]common.Hash {
	storage := make(map[common.Hash]common.Hash, len(slots))
	for i, slot := range slots {
		storage[common.BigToHash(new(big.Int).SetUint64(slot))] = common.BigToHash(big.NewInt(int64(i) + 1))
	}
	return storage
}

// pbtCallTx returns a generator sending a zero-value call to target, which is
// enough to touch it without changing its balance.
func pbtCallTx(t *testing.T, key *ecdsa.PrivateKey, sender, target common.Address) func(int, *BlockGen) {
	t.Helper()

	return func(i int, gen *BlockGen) {
		signer := gen.Signer()
		tx, err := types.SignTx(types.NewTransaction(
			gen.TxNonce(sender), target, new(big.Int), pbtCodeBlockGas/2,
			new(big.Int).Add(gen.BaseFee(), common.Big1), nil,
		), signer, key)
		if err != nil {
			t.Fatal(err)
		}
		gen.AddTx(tx)
	}
}
