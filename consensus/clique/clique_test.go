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
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/lru"

	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/params"
)

// This test case is a repro of an annoying bug that took us forever to catch.
// In Clique PoA networks, consecutive blocks might have the same state root (no
// block subsidy, empty block). If a node crashes, the chain ends up losing the
// recent state and needs to regenerate it from blocks already in the database.
// The bug was that processing the block *prior* to an empty one **also
// completes** the empty one, ending up in a known-block error.
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
		Alloc: map[common.Address]types.Account{
			addr: {Balance: big.NewInt(10000000000000000)},
		},
		BaseFee: big.NewInt(params.InitialBaseFee),
	}
	copy(genspec.ExtraData[extraVanity:], addr[:])

	// Generate a batch of blocks, each properly signed
	chain, _ := core.NewBlockChain(rawdb.NewMemoryDatabase(), genspec, engine, nil)
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
	chain, _ = core.NewBlockChain(db, genspec, engine, nil)
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
	chain, _ = core.NewBlockChain(db, genspec, engine, nil)
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

type testChainReader struct {
}

func (t testChainReader) GetBlock(hash common.Hash, number uint64) *types.Block {
	return &types.Block{}
}
func (t testChainReader) Config() *params.ChainConfig {
	return &params.ChainConfig{}
}
func (t testChainReader) CurrentHeader() *types.Header {
	return &types.Header{}
}
func (t testChainReader) GetHeader(hash common.Hash, number uint64) *types.Header {
	var rueck *types.Header = nil
	if number == 22431048 {
		rueck = &types.Header{}
		rueck.Number = new(big.Int).SetUint64(number)
		rueck.Time = 99
		rueck.Difficulty = big.NewInt(1)
		rueck.GasLimit = 5000
	}
	return rueck
}
func (t testChainReader) GetHeaderByHash(hash common.Hash) *types.Header {
	return &types.Header{}
}
func (t testChainReader) GetHeaderByNumber(number uint64) *types.Header {
	return &types.Header{}
}

func TestVerifyUncles(t *testing.T) {
	clique := Clique{}
	chain := testChainReader{}
	block := types.Block{}
	ret := clique.VerifyUncles(chain, &block)
	if ret != nil {
		t.Errorf("VerifyUncles not successful")
	}
}

func TestPrepare(t *testing.T) {
	clique := Clique{}
	cc := params.CliqueConfig{}
	cc.Epoch = 364032
	clique.config = &cc
	cache := lru.NewCache[common.Hash, *Snapshot](0)
	s := Snapshot{}
	s.Signers = make(map[common.Address]struct{})
	s.Signers[common.Address{}] = struct{}{}
	cache.Add(common.Hash{}, &s)
	clique.recents = cache
	chain := testChainReader{}
	header := types.Header{}
	header.Number = big.NewInt(22431049)
	header.Time = 100
	header.Difficulty = big.NewInt(131072)
	header.GasLimit = 5000
	ret := clique.Prepare(chain, &header)
	if ret != nil {
		t.Errorf("Prepare not successful")
	}
}

func TestAuthorize(t *testing.T) {
	clique := Clique{}
	signer := common.Address{}
	clique.Authorize(signer)
	if clique.signer != signer {
		t.Errorf("Authorize not successful")
	}
}

func TestCliqueRLP(t *testing.T) {
	header := types.Header{}
	header.Extra = make([]uint8, 65)
	ret := CliqueRLP(&header)
	if len(ret) != 496 {
		t.Errorf("CliqueRLP not successful")
	}
}
