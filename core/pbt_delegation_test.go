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

// EIP-7702 delegation under the binary tree. Every other test of the
// delegation leaf drives it synthetically through core/state; these run a real
// signed authorization down the whole stack, which is the only path production
// takes and the one none of them covered.

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
	// one paying: both roles on one account would hide a balance-only write.
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
// authorization lands: the authority's own header, and nowhere else.
//
// The negatives carry as much as the positive. No code-hash leaf, since a
// stale one would make the account read back as ordinary code. No code chunk,
// since chunking the indicator would put a per-account value in the shared
// zone. And the account must still report the designator's hash, which is what
// keeps EIP-161 emptiness and the txpools' probe correct.
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

// TestPBTSetCodeTxSameTargetReauth pins the one shape that reaches the tree
// with a live delegation and no code write: re-authorizing the same target.
//
// applyAuthorization bumps the nonce unconditionally but skips SetCode when
// the target is unchanged, so the account arrives dirty with no stated size
// and no designator. Only the preserve branch stops that write putting back a
// code-hash leaf and silently un-delegating a just-renewed account.
//
// The nonce is checked too: reading only the leaves would also pass on a block
// where nothing happened at all.
func TestPBTSetCodeTxSameTargetReauth(t *testing.T) {
	var (
		delegate             = common.Address{0xde, 0x1e, 0x9a, 0x7e}
		designator           = types.AddressToDelegation(delegate)
		genesis, key, sender = pbtCodeReorgFixture(t)
		engine               = beacon.New(ethash.NewFaker())
		signer               = types.LatestSigner(genesis.Config)
		chainID              = uint256.MustFromBig(genesis.Config.ChainID)
	)
	authKey, err := crypto.HexToECDSA("0202020202020202020202020202020202020202020202020202002020202020")
	if err != nil {
		t.Fatal(err)
	}
	authority := crypto.PubkeyToAddress(authKey.PublicKey)

	// Both authorizations name the same target. The authority's nonce is
	// bumped by the first, so the second has to carry the incremented value or
	// it is rejected before reaching the state.
	auths := make([]types.SetCodeAuthorization, 2)
	for i := range auths {
		auth, err := types.SignSetCode(authKey, types.SetCodeAuthorization{
			ChainID: *chainID,
			Address: delegate,
			Nonce:   uint64(i),
		})
		if err != nil {
			t.Fatalf("signing authorization %d: %v", i, err)
		}
		auths[i] = auth
	}

	db, blocks, _ := GenerateChainWithGenesis(genesis, engine, 2, func(i int, gen *BlockGen) {
		gen.AddTx(types.MustSignNewTx(key, signer, &types.SetCodeTx{
			ChainID:   chainID,
			Nonce:     gen.TxNonce(sender),
			To:        sender,
			Value:     new(uint256.Int),
			Gas:       1_000_000,
			GasFeeCap: uint256.MustFromBig(new(big.Int).Add(gen.BaseFee(), common.Big1)),
			GasTipCap: new(uint256.Int),
			AuthList:  []types.SetCodeAuthorization{auths[i]},
		}))
	})
	chain, cerr := NewBlockChain(db, genesis, engine, DefaultConfig().WithStateScheme(rawdb.PathScheme))
	if cerr != nil {
		t.Fatal(cerr)
	}
	defer chain.Stop()
	if _, err := chain.InsertChain(blocks); err != nil {
		t.Fatalf("importing the re-authorizing chain: %v", err)
	}

	// After the first block the delegation is installed the ordinary way.
	want := bintrie.EncodeDelegation(designator)
	if got := headerLeafAt(t, chain, blocks[0].Root(), authority, bintrie.DelegationLeafKey); !bytes.Equal(got, want) {
		t.Fatalf("delegation leaf is %x after the first authorization, want %x", got, want)
	}
	// After the second it must be exactly where it was, with no code-hash leaf
	// beside it, even though nothing wrote code in that block.
	if got := headerLeafAt(t, chain, blocks[1].Root(), authority, bintrie.DelegationLeafKey); !bytes.Equal(got, want) {
		t.Fatalf("re-authorizing the same target changed the delegation leaf to %x, want %x", got, want)
	}
	if got := headerLeafAt(t, chain, blocks[1].Root(), authority, bintrie.CodeHashLeafKey); got != nil {
		t.Fatalf("re-authorizing the same target installed a code-hash leaf: %x", got)
	}
	acct := accountAt(t, chain, blocks[1].Root(), authority)
	if acct == nil {
		t.Fatal("the re-authorized account is missing")
	}
	if acct.Nonce != 2 {
		t.Fatalf("authority nonce is %d after two authorizations, want 2; the second never landed", acct.Nonce)
	}
	if wantHash := crypto.Keccak256(designator); !bytes.Equal(acct.CodeHash, wantHash) {
		t.Fatalf("authority code hash is %x, want the designator's %x", acct.CodeHash, wantHash)
	}
}

// TestPBTStatelessSetCodeTx re-executes the delegating block from its witness
// alone.
//
// Moving the indicator out of the code zone raised a question for replay: it
// is witnessed as a code blob by keccak, and had it stopped being one the
// replay would have needed to rebuild it from the header stem. It has not -
// only the tree representation changed - and the hash replay looks it up by is
// the one GetAccount synthesises. Checked here rather than assumed.
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
