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
	"bytes"
	"context"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/consensus/beacon"
	"github.com/ethereum/go-ethereum/consensus/ethash"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/core/vm/program"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/params"
)

// TestStatelessContractCoinbaseMerkle is the merkle-patricia twin of
// TestPBTStatelessContractCoinbase, and it is the evidence for holding merkle
// to the witness completeness check in ExecuteStateless rather than only the
// binary tree.
//
// The check is a whole-execution latch, so widening it means a witness that is
// short of anything turns a block whose root is actually correct into a
// rejection. The one read known to bypass Witness.AddCode was the code size
// that updateStateObject used to ask for on every dirty contract account, and
// a contract fee recipient is the account that reaches it in every block while
// being executed by none of them. This is that shape, under merkle: without
// the fix it fails with "incomplete witness: code is not found".
//
// Every other ExecuteStateless test in the tree is PBT, and the blockchain
// fixtures in tests/ skip because testdata is an uninitialised submodule, so
// without this the widening rests on eth/catalyst's single witness test.
//
// It runs at Amsterdam as well as at Osaka because the two unwitnessed code
// reads TODO.md still lists are both Amsterdam-only - recordAccessListChanges
// needs a block access list, and ReaderWithBlockLevelAccessList is the EIP-7928
// layer. PBT is an optional fork on top of Amsterdam rather than an Amsterdam
// consequence, so merkle-at-Amsterdam is a real configuration.
//
// Be clear about what the Amsterdam arm does and does not establish. It does
// take the parallel processor, which is the path an Amsterdam node takes and
// which nothing else in the tree exercises - the assertion below makes sure of
// that, because the access list is easy to drop by accident. What it cannot
// show is that the completeness check would catch an unwitnessed read there:
// on that path every transaction runs against its own statedb whose error is
// never consulted, so only the root comparison is doing work. See TODO.md.
func TestStatelessContractCoinbaseMerkle(t *testing.T) {
	amsterdam := *testPBTChainConfig
	amsterdam.BinaryTrieTime = nil // same forks, merkle state
	amsterdam.Ethash = nil

	for _, tc := range []struct {
		name   string
		config params.ChainConfig
	}{
		{"osaka", *params.MergedTestChainConfig},
		{"amsterdam", amsterdam},
	} {
		t.Run(tc.name, func(t *testing.T) {
			statelessContractCoinbaseMerkle(t, tc.config)
		})
	}
}

func statelessContractCoinbaseMerkle(t *testing.T, config params.ChainConfig) {
	t.Helper()

	key, err := crypto.HexToECDSA("b71c71a67e1177ad4e901695e1b4b9ee17ae16c6668d313eac2f96dbcda3f291")
	if err != nil {
		t.Fatal(err)
	}
	var (
		sender    = crypto.PubkeyToAddress(key.PublicKey)
		recipient = common.Address{0x0f, 0xf1}
		coinbase  = common.Address{0xc0, 0x1b}
	)
	if config.IsPBT() {
		t.Fatal("this fixture is meant to be merkle-patricia; it proves nothing under the tree")
	}
	genesis := &Genesis{
		Config:     &config,
		Difficulty: big.NewInt(0),
		BaseFee:    big.NewInt(params.InitialBaseFee),
		Alloc: types.GenesisAlloc{
			sender: {Balance: new(big.Int).Mul(big.NewInt(1000), big.NewInt(params.Ether))},
			// A contract at the fee recipient that no transaction calls.
			coinbase: {Balance: big.NewInt(1), Code: bytes.Repeat([]byte{0x60, 0x01, 0x50}, 300)},
		},
	}
	engine := beacon.New(ethash.NewFaker())
	signer := types.LatestSigner(genesis.Config)

	db, blocks, _ := GenerateChainWithGenesis(genesis, engine, 1, func(i int, gen *BlockGen) {
		gen.SetCoinbase(coinbase)
		tx, err := types.SignTx(types.NewTransaction(
			gen.TxNonce(sender), recipient, big.NewInt(1000), 1_000_000,
			new(big.Int).Add(gen.BaseFee(), common.Big1), nil,
		), signer, key)
		if err != nil {
			t.Fatal(err)
		}
		gen.AddTx(tx)

		// A deployment as well, so some account is journalled with a code
		// change. recordAccessListChanges only reaches obj.Code() under
		// state.codeSet, which a block of plain transfers never sets.
		deploy, err := types.SignTx(types.NewContractCreation(
			gen.TxNonce(sender), big.NewInt(0), 1_000_000,
			new(big.Int).Add(gen.BaseFee(), common.Big1),
			program.New().ReturnViaCodeCopy([]byte{0x60, 0x01, 0x50}).Bytes(),
		), signer, key)
		if err != nil {
			t.Fatal(err)
		}
		gen.AddTx(deploy)
	})
	chain, err := NewBlockChain(db, genesis, engine, DefaultConfig())
	if err != nil {
		t.Fatal(err)
	}
	defer chain.Stop()

	parent := chain.GetHeaderByNumber(0)
	res, err := chain.ProcessBlock(context.Background(), parent.Root, blocks[0], ExecuteConfig{MakeWitness: true})
	if err != nil {
		t.Fatalf("processing the block: %v", err)
	}
	witness := res.Witness()
	if witness == nil {
		t.Fatal("no witness was gathered")
	}
	header := types.CopyHeader(blocks[0].Header())
	header.Root, header.ReceiptHash = common.Hash{}, common.Hash{}
	// Carry the access list across explicitly. types.Body has no field for it
	// and WithBody takes it from the receiver, so the obvious construction
	// hands ExecuteStateless a nil-BAL block, which silently routes it to the
	// sequential processor - not the path an Amsterdam node takes.
	task := types.NewBlockWithHeader(header).WithBody(*blocks[0].Body())
	if al := blocks[0].AccessList(); al != nil {
		task = task.WithAccessListUnsafe(al)
	}
	// Without the access list the replay quietly takes the sequential
	// processor, which is not what an Amsterdam node does and would make this
	// subtest a duplicate of the Osaka one.
	if config.IsAmsterdam(blocks[0].Number(), blocks[0].Time()) && task.AccessList() == nil {
		t.Fatal("the Amsterdam task carries no access list, so the replay would not take the parallel path")
	}

	stateRoot, _, err := ExecuteStateless(context.Background(), chain.Config(), vm.Config{}, task, witness)
	if err != nil {
		t.Fatalf("merkle stateless execution with a contract fee recipient failed: %v", err)
	}
	if stateRoot != blocks[0].Root() {
		t.Fatalf("stateless state root mismatch: got %x, want %x", stateRoot, blocks[0].Root())
	}
	// The fee recipient has to have actually been paid, or it was never dirty
	// and the account-write path was never reached for it.
	if _, err := chain.InsertChain([]*types.Block{blocks[0]}); err != nil {
		t.Fatal(err)
	}
	state, err := chain.State()
	if err != nil {
		t.Fatal(err)
	}
	if got := state.GetBalance(coinbase).Uint64(); got <= 1 {
		t.Fatalf("the fee recipient's balance is %d, so it was never credited and never dirty", got)
	}
}
