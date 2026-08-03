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
	"context"
	"math/big"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/consensus/beacon"
	"github.com/ethereum/go-ethereum/consensus/ethash"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
)

// The binary tree does not support everything the merkle-patricia trie does.
// Each unsupported operation is guarded in its own package, which makes the set
// of them hard to see and easy to regress one at a time. These tests cover the
// ones that live in this package; the index below is the whole contract, so it
// can be read without hunting for the guards.
//
// What is being pinned is not "this is unimplemented" but "this refuses, and
// says why". Silently returning a wrong answer is the failure mode that
// matters: a stateless run that rebuilds the wrong kind of database still
// produces a root, and a dump that gives up still returns bytes.
//
//	Operation                     Refused at                       Pinned by
//	---------------------------------------------------------------------------
//	stateless execution           core/stateless.go                this file
//	historic state                core/blockchain_reader.go        core/pbt_scheme_test.go
//	hash-scheme trie database     core/blockchain.go triedbConfig  core/pbt_scheme_test.go
//	account dumping               core/state/dump.go               core/state/pbt_capabilities_test.go
//	pathdb rollback (Recover)     triedb/pathdb/database.go        triedb/pathdb/pbt_rollback_test.go
//	state sync / AdoptSyncedState triedb/pathdb/database.go        triedb/pathdb/pbt_rollback_test.go
//	opening a tree datadir as MPT cmd/utils MakeTrieDatabase       cmd/geth/bintrie_convert_test.go
//	debug_storageRangeAt          eth/api_debug.go                 not pinned - guard read, no test
//
// Anything on this list failing a spec fixture is a known gap rather than a
// conformance defect, which is the distinction that makes the reference
// integration readable.

// TestPBTRefusesStatelessExecution pins that stateless execution is refused
// rather than attempted.
//
// The witness format is merkle-patricia shaped - MakeHashDB keys nodes by their
// keccak hash and accounts are RLP leaves - so a binary tree state cannot be
// rebuilt from it. Attempting it does not fail loudly: it yields a root that
// cannot match, which a node reports as its own valid block being invalid.
// Witness building is gated only on IsByzantium, and --stateless-self-validation
// is an ordinary flag, so this is reachable on a real node.
func TestPBTRefusesStatelessExecution(t *testing.T) {
	cfg := *testPBTChainConfig
	if !cfg.IsPBT() {
		t.Fatal("the fixture is not a binary tree configuration; this proves nothing")
	}
	block := types.NewBlockWithHeader(&types.Header{Number: big.NewInt(1)})

	_, _, err := ExecuteStateless(context.Background(), &cfg, vm.Config{}, block, nil)
	if err == nil {
		t.Fatal("stateless execution was attempted against a binary tree state")
	}
	if !strings.Contains(err.Error(), "binary tree") {
		t.Fatalf("the refusal does not say why: %v", err)
	}

	// The control: the same call on a merkle-patricia configuration must not be
	// refused by this guard. It fails later, on the nil witness, which is what
	// tells us the guard did not fire.
	plain := *testPBTChainConfig
	plain.PBT = false
	func() {
		defer func() { _ = recover() }() // a nil witness panics; only the guard matters here
		if _, _, err := ExecuteStateless(context.Background(), &plain, vm.Config{}, block, nil); err != nil {
			if strings.Contains(err.Error(), "binary tree") {
				t.Errorf("the binary tree guard fired on a merkle-patricia configuration: %v", err)
			}
		}
	}()
}

// TestPBTStatelessSelfValidationRefusedOnImport drives the same guard through
// block import, which is the way a node reaches it.
//
// StatelessSelfValidation is an ordinary flag, and witness building is gated
// only on IsByzantium, so nothing stopped a binary tree node from turning this
// on. Without the guard the import still fails - the rebuilt merkle database
// cannot resolve binary tree records, so it reports a missing trie node - but
// it fails opaquely, as though the state were corrupt, on a block that is
// perfectly valid. The refusal has to name the reason.
func TestPBTStatelessSelfValidationRefusedOnImport(t *testing.T) {
	genesis, key, sender, recipient := pbtChainGenesis(t)
	engine := beacon.New(ethash.NewFaker())
	signer := types.LatestSigner(genesis.Config)

	db, blocks, _ := GenerateChainWithGenesis(genesis, engine, 1, func(i int, gen *BlockGen) {
		tx, err := types.SignTx(types.NewTransaction(
			gen.TxNonce(sender), recipient, big.NewInt(1000), pbtTestTxGas,
			new(big.Int).Add(gen.BaseFee(), common.Big1), nil,
		), signer, key)
		if err != nil {
			t.Fatal(err)
		}
		gen.AddTx(tx)
	})

	options := DefaultConfig().WithStateScheme(rawdb.PathScheme)
	options.StatelessSelfValidation = true

	chain, err := NewBlockChain(db, genesis, engine, options)
	if err != nil {
		t.Fatal(err)
	}
	defer chain.Stop()

	_, err = chain.InsertChain(blocks)
	if err == nil {
		t.Fatal("a binary tree block was imported with stateless self-validation on; the cross-check cannot have run")
	}
	if !strings.Contains(err.Error(), "binary tree") {
		t.Fatalf("the import failed for some other reason than the refusal: %v", err)
	}
}
