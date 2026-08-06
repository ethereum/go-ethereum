// Copyright 2025 The go-ethereum Authors
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
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/trie/bintrie"
	"github.com/holiman/uint256"
)

// EIP-7702 delegation under the binary tree.
//
// Every other test of the delegation leaf drives it synthetically through
// core/state. These run a real signed authorization down the whole stack -
// transaction, state transition, tree write, commit - because that is the only
// path production takes, and it is the one path none of them covered.

// headerLeafAt reads one sub-index of an account's header stem at a root,
// returning nil when the leaf is absent.
func headerLeafAt(t *testing.T, chain *BlockChain, root common.Hash, addr common.Address, sub byte) []byte {
	t.Helper()

	tr, err := bintrie.NewBinaryTrie(root, chain.TrieDB())
	if err != nil {
		t.Fatalf("cannot open the tree at %x: %v", root, err)
	}
	value, err := tr.GetStemValue(bintrie.HeaderKey(addr, sub))
	if err != nil {
		t.Fatalf("failed to read sub-index %d of %x at %x: %v", sub, addr, root, err)
	}
	return value
}

// pbtDelegationChain builds a one-block chain whose only transaction carries an
// authorization delegating authority to delegate, and returns the chain, the
// block and the designator the authority should end up holding.
func pbtDelegationChain(t *testing.T, delegate common.Address) (*BlockChain, *types.Block, common.Address, []byte) {
	t.Helper()

	genesis, key, sender := pbtCodeReorgFixture(t)
	engine := beacon.New(ethash.NewFaker())
	signer := types.LatestSigner(genesis.Config)

	// A separate account authorizes, so the delegated account is not also the
	// one paying: the two roles hitting one account would hide a write that
	// only happens because the account was touched for its balance.
	authKey, err := crypto.HexToECDSA("0202020202020202020202020202020202020202020202020202002020202020")
	if err != nil {
		t.Fatal(err)
	}
	authority := crypto.PubkeyToAddress(authKey.PublicKey)

	auth, err := types.SignSetCode(authKey, types.SetCodeAuthorization{
		ChainID: *uint256.MustFromBig(genesis.Config.ChainID),
		Address: delegate,
		Nonce:   0,
	})
	if err != nil {
		t.Fatalf("signing the authorization: %v", err)
	}

	db, blocks, _ := GenerateChainWithGenesis(genesis, engine, 1, func(i int, gen *BlockGen) {
		gen.AddTx(types.MustSignNewTx(key, signer, &types.SetCodeTx{
			ChainID:   uint256.MustFromBig(genesis.Config.ChainID),
			Nonce:     gen.TxNonce(sender),
			To:        sender,
			Value:     new(uint256.Int),
			Gas:       1_000_000,
			GasFeeCap: uint256.MustFromBig(new(big.Int).Add(gen.BaseFee(), common.Big1)),
			GasTipCap: new(uint256.Int),
			AuthList:  []types.SetCodeAuthorization{auth},
		}))
	})
	chain, err := NewBlockChain(db, genesis, engine, DefaultConfig().WithStateScheme(rawdb.PathScheme))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(chain.Stop)
	if !chain.Config().IsPBT() {
		t.Fatal("the fixture is not a binary tree configuration; this proves nothing")
	}
	return chain, blocks[0], authority, types.AddressToDelegation(delegate)
}

// TestPBTSetCodeTxWritesDelegationLeaf pins where a real EIP-7702
// authorization lands in the tree: the authority's own header, and nowhere
// else.
//
// The three negatives carry as much as the positive. No code-hash leaf,
// because the two are exclusive and a stale one would make the account read
// back as ordinary code. No code chunk, because chunking the indicator would
// put a per-account value in the shared zone, where nothing removes it. And
// the account still has to report the designator's hash to everything above
// the tree, which is what keeps EIP-161 emptiness and the txpools' delegation
// probe correct without either of them knowing the leaf exists.
func TestPBTSetCodeTxWritesDelegationLeaf(t *testing.T) {
	delegate := common.Address{0xde, 0x1e, 0x9a, 0x7e}
	chain, block, authority, designator := pbtDelegationChain(t, delegate)

	if _, err := chain.InsertChain([]*types.Block{block}); err != nil {
		t.Fatalf("importing the delegating block: %v", err)
	}
	root := block.Root()

	want := bintrie.EncodeDelegation(designator)
	if got := headerLeafAt(t, chain, root, authority, bintrie.DelegationLeafKey); !bytes.Equal(got, want) {
		t.Fatalf("delegation leaf is %x, want %x", got, want)
	}
	if got := headerLeafAt(t, chain, root, authority, bintrie.CodeHashLeafKey); got != nil {
		t.Fatalf("the delegated account also holds a code-hash leaf: %x", got)
	}
	if got := codeChunkAt(t, chain, root, crypto.Keccak256Hash(designator), 0); got != nil {
		t.Fatalf("the designator was chunked into the shared code zone: %x", got)
	}
	acct := accountAt(t, chain, root, authority)
	if acct == nil {
		t.Fatal("the delegated authority has no account at all")
	}
	if wantHash := crypto.Keccak256(designator); !bytes.Equal(acct.CodeHash, wantHash) {
		t.Fatalf("authority code hash is %x, want the designator's %x", acct.CodeHash, wantHash)
	}

	// The state API has to agree with the leaves, or the delegation is
	// invisible to every consumer that goes through StateDB.
	state, err := chain.State()
	if err != nil {
		t.Fatal(err)
	}
	if got := state.GetCode(authority); !bytes.Equal(got, designator) {
		t.Fatalf("GetCode returns %x for the delegated account, want %x", got, designator)
	}
	if got := state.GetCodeSize(authority); got != len(designator) {
		t.Fatalf("GetCodeSize is %d, want %d", got, len(designator))
	}
}

// TestPBTStatelessSetCodeTx re-executes the delegating block from its witness
// alone.
//
// Moving the indicator out of the code zone raised a real question for
// stateless replay: it is witnessed as a code blob, by keccak, and if it had
// stopped being one the replay would have had to rebuild it from the header
// stem instead. It has not - the designator is still ordinary code everywhere
// above the trie, and only its tree representation changed - and the hash the
// replay looks it up by is the one GetAccount synthesises from the leaf. This
// is that reasoning checked rather than asserted.
func TestPBTStatelessSetCodeTx(t *testing.T) {
	chain, block, authority, designator := pbtDelegationChain(t, common.Address{0xde, 0x1e, 0x9a, 0x7e})

	parent := chain.GetHeaderByNumber(0)
	res, err := chain.ProcessBlock(context.Background(), parent.Root, block, ExecuteConfig{MakeWitness: true})
	if err != nil {
		t.Fatalf("processing the delegating block: %v", err)
	}
	witness := res.Witness()
	if len(witness.Nodes) == 0 {
		t.Fatal("the witness holds no nodes")
	}

	header := types.CopyHeader(block.Header())
	header.Root, header.ReceiptHash = common.Hash{}, common.Hash{}
	task := types.NewBlockWithHeader(header).WithBody(*block.Body())

	stateRoot, _, err := ExecuteStateless(context.Background(), chain.Config(), vm.Config{}, task, witness)
	if err != nil {
		t.Fatalf("stateless execution of a delegating block failed: %v", err)
	}
	if stateRoot != block.Root() {
		t.Fatalf("stateless state root mismatch: got %x, want %x", stateRoot, block.Root())
	}
	// The delegation is what the block was for, so a matching root that did
	// not install it would mean the fixture never exercised any of this.
	if _, err := chain.InsertChain([]*types.Block{block}); err != nil {
		t.Fatal(err)
	}
	if got := headerLeafAt(t, chain, block.Root(), authority, bintrie.DelegationLeafKey); !bytes.Equal(got, bintrie.EncodeDelegation(designator)) {
		t.Fatalf("delegation leaf is %x after the replayed block, want %x", got, bintrie.EncodeDelegation(designator))
	}
}
