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

package clique

import (
	"bytes"
	"math/big"
	"strings"
	"testing"

	"github.com/scroll-tech/go-ethereum/common"
	"github.com/scroll-tech/go-ethereum/core"
	"github.com/scroll-tech/go-ethereum/core/rawdb"
	"github.com/scroll-tech/go-ethereum/core/types"
	"github.com/scroll-tech/go-ethereum/core/vm"
	"github.com/scroll-tech/go-ethereum/crypto"
	"github.com/scroll-tech/go-ethereum/params"
	"github.com/scroll-tech/go-ethereum/trie"
)

// This test case is a repro of an annoying bug that took us forever to catch.
// In Clique PoA networks (Görli, etc), consecutive blocks might have
// the same state root (no block subsidy, empty block). If a node crashes, the
// chain ends up losing the recent state and needs to regenerate it from blocks
// already in the database. The bug was that processing the block *prior* to an
// empty one **also completes** the empty one, ending up in a known-block error.
func TestReimportMirroredState(t *testing.T) {
	// Initialize a Clique chain with a single signer
	var (
		db     = rawdb.NewMemoryDatabase()
		key, _ = crypto.HexToECDSA("b71c71a67e1177ad4e901695e1b4b9ee17ae16c6668d313eac2f96dbcda3f291")
		addr   = crypto.PubkeyToAddress(key.PublicKey)
		engine = New(params.AllCliqueProtocolChanges.Clique, db)
		signer = new(types.HomesteadSigner)
	)
	genspec := &core.Genesis{
		Config:    params.AllCliqueProtocolChanges,
		ExtraData: make([]byte, extraVanity+common.AddressLength+extraSeal),
		Alloc: map[common.Address]core.GenesisAccount{
			addr: {Balance: big.NewInt(10000000000000000)},
		},
		BaseFee: big.NewInt(params.InitialBaseFee),
	}
	copy(genspec.ExtraData[extraVanity:], addr[:])

	// Generate a batch of blocks, each properly signed
	chain, _ := core.NewBlockChain(rawdb.NewMemoryDatabase(), nil, genspec, nil, engine, vm.Config{}, nil, nil)
	defer chain.Stop()

	_, blocks, _ := core.GenerateChainWithGenesis(genspec, engine, 3, func(i int, block *core.BlockGen) {
		// The chain maker doesn't have access to a chain, so the difficulty will be
		// lets unset (nil). Set it here to the correct value.
		block.SetDifficulty(diffInTurn)

		// We want to simulate an empty middle block, having the same state as the
		// first one. The last is needs a state change again to force a reorg.
		if i != 1 {
			tx, err := types.SignTx(types.NewTransaction(block.TxNonce(addr), common.Address{0x00}, new(big.Int), params.TxGas, block.BaseFee(), nil), signer, key)
			if err != nil {
				panic(err)
			}
			block.AddTxWithChain(chain, tx)
		}
	})
	for i, block := range blocks {
		header := block.Header()
		if i > 0 {
			header.ParentHash = blocks[i-1].Hash()
		}
		header.Extra = make([]byte, extraVanity+extraSeal)
		header.Difficulty = diffInTurn

		sig, _ := crypto.Sign(SealHash(header).Bytes(), key)
		copy(header.Extra[len(header.Extra)-extraSeal:], sig)
		blocks[i] = block.WithSeal(header)
	}
	// Insert the first two blocks and make sure the chain is valid
	db = rawdb.NewMemoryDatabase()
	chain, _ = core.NewBlockChain(db, nil, genspec, nil, engine, vm.Config{}, nil, nil)
	defer chain.Stop()

	if _, err := chain.InsertChain(blocks[:2]); err != nil {
		t.Fatalf("failed to insert initial blocks: %v", err)
	}
	if head := chain.CurrentBlock().Number.Uint64(); head != 2 {
		t.Fatalf("chain head mismatch: have %d, want %d", head, 2)
	}

	// Simulate a crash by creating a new chain on top of the database, without
	// flushing the dirty states out. Insert the last block, triggering a sidechain
	// reimport.
	chain, _ = core.NewBlockChain(db, nil, genspec, nil, engine, vm.Config{}, nil, nil)
	defer chain.Stop()

	if _, err := chain.InsertChain(blocks[2:]); err != nil {
		t.Fatalf("failed to insert final block: %v", err)
	}
	if head := chain.CurrentBlock().Number.Uint64(); head != 3 {
		t.Fatalf("chain head mismatch: have %d, want %d", head, 3)
	}
}

func TestSealHash(t *testing.T) {
	have := SealHash(&types.Header{
		Difficulty: new(big.Int),
		Number:     new(big.Int),
		Extra:      make([]byte, 32+65),
		BaseFee:    new(big.Int),
	})
	want := common.HexToHash("0xbd3d1fa43fbc4c5bfcc91b179ec92e2861df3654de60468beb908ff805359e8f")
	if have != want {
		t.Errorf("have %x, want %x", have, want)
	}
}

func TestShadowFork(t *testing.T) {
	engineConf := *params.AllCliqueProtocolChanges.Clique
	engineConf.Epoch = 2
	forkedEngineConf := engineConf
	forkedEngineConf.ShadowForkHeight = 3
	shadowForkKey, _ := crypto.HexToECDSA(strings.Repeat("11", 32))
	shadowForkAddr := crypto.PubkeyToAddress(shadowForkKey.PublicKey)
	forkedEngineConf.ShadowForkSigner = shadowForkAddr

	// Initialize a Clique chain with a single signer
	var (
		db           = rawdb.NewMemoryDatabase()
		key, _       = crypto.HexToECDSA("b71c71a67e1177ad4e901695e1b4b9ee17ae16c6668d313eac2f96dbcda3f291")
		addr         = crypto.PubkeyToAddress(key.PublicKey)
		engine       = New(&engineConf, db)
		signer       = new(types.HomesteadSigner)
		forkedEngine = New(&forkedEngineConf, db)
	)
	genspec := &core.Genesis{
		Config:    params.AllCliqueProtocolChanges,
		ExtraData: make([]byte, extraVanity+common.AddressLength+extraSeal),
		Alloc: map[common.Address]core.GenesisAccount{
			addr: {Balance: big.NewInt(10000000000000000)},
		},
		BaseFee: big.NewInt(params.InitialBaseFee),
	}
	copy(genspec.ExtraData[extraVanity:], addr[:])
	genesis := genspec.MustCommit(db, trie.NewDatabase(db, trie.HashDefaults))

	// Generate a batch of blocks, each properly signed
	chain, _ := core.NewBlockChain(db, nil, genspec, nil, engine, vm.Config{}, nil, nil)
	defer chain.Stop()

	forkedChain, _ := core.NewBlockChain(db, nil, genspec, nil, forkedEngine, vm.Config{}, nil, nil)
	defer forkedChain.Stop()

	blocks, _ := core.GenerateChain(params.AllCliqueProtocolChanges, genesis, forkedEngine, db, 16, func(i int, block *core.BlockGen) {
		// The chain maker doesn't have access to a chain, so the difficulty will be
		// lets unset (nil). Set it here to the correct value.
		if block.Number().Uint64() > forkedEngineConf.ShadowForkHeight {
			block.SetDifficulty(diffShadowFork)
		} else {
			block.SetDifficulty(diffInTurn)
		}

		tx, err := types.SignTx(types.NewTransaction(block.TxNonce(addr), common.Address{0x00}, new(big.Int), params.TxGas, block.BaseFee(), nil), signer, key)
		if err != nil {
			panic(err)
		}
		block.AddTxWithChain(chain, tx)
	})
	for i, block := range blocks {
		header := block.Header()
		if i > 0 {
			header.ParentHash = blocks[i-1].Hash()
		}

		signingAddr, signingKey := addr, key
		if header.Number.Uint64() > forkedEngineConf.ShadowForkHeight {
			// start signing with shadow fork authority key
			signingAddr, signingKey = shadowForkAddr, shadowForkKey
		}

		header.Extra = make([]byte, extraVanity)
		if header.Number.Uint64()%engineConf.Epoch == 0 {
			header.Extra = append(header.Extra, signingAddr.Bytes()...)
		}
		header.Extra = append(header.Extra, bytes.Repeat([]byte{0}, extraSeal)...)

		sig, _ := crypto.Sign(SealHash(header).Bytes(), signingKey)
		copy(header.Extra[len(header.Extra)-extraSeal:], sig)
		blocks[i] = block.WithSeal(header)
	}

	if _, err := chain.InsertChain(blocks); err == nil {
		t.Fatalf("should've failed to insert some blocks to canonical chain")
	}
	if chain.CurrentHeader().Number.Uint64() != forkedEngineConf.ShadowForkHeight {
		t.Fatalf("unexpected canonical chain height")
	}
	if _, err := forkedChain.InsertChain(blocks); err != nil {
		t.Fatalf("failed to insert blocks to forked chain: %v %d", err, forkedChain.CurrentHeader().Number)
	}
	if forkedChain.CurrentHeader().Number.Uint64() != uint64(len(blocks)) {
		t.Fatalf("unexpected forked chain height")
	}
}
