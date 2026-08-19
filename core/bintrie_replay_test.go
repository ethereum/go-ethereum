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
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/consensus/beacon"
	"github.com/ethereum/go-ethereum/consensus/ethash"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/tracing"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/triedb"
	"github.com/holiman/uint256"
)

// replayUniverse enumerates every account and slot a scenario can touch, so
// the expected tree can be rebuilt from canonical state without preimages.
type replayUniverse struct {
	addrs map[common.Address]struct{}
	slots map[common.Address][]common.Hash
}

func newReplayUniverse() *replayUniverse {
	return &replayUniverse{
		addrs: make(map[common.Address]struct{}),
		slots: make(map[common.Address][]common.Hash),
	}
}

func (u *replayUniverse) account(addrs ...common.Address) {
	for _, addr := range addrs {
		u.addrs[addr] = struct{}{}
	}
}

func (u *replayUniverse) storage(addr common.Address, slots ...common.Hash) {
	u.account(addr)
	u.slots[addr] = append(u.slots[addr], slots...)
}

// convertCanonical rebuilds a binary tree from the canonical merkle state at
// the given header, over the known universe - the converter's answer,
// reconstructed in-test. Setter usage mirrors flushAlloc.
func convertCanonical(t *testing.T, chain *BlockChain, header *types.Header, u *replayUniverse) common.Hash {
	t.Helper()
	src, err := chain.StateAt(header)
	if err != nil {
		t.Fatal(err)
	}
	tdb := triedb.NewDatabase(rawdb.NewMemoryDatabase(), triedb.PBTDefaults)
	defer tdb.Close()
	dst, err := state.New(types.EmptyBinaryHash, state.NewDatabase(tdb, nil))
	if err != nil {
		t.Fatal(err)
	}
	for addr := range u.addrs {
		if !src.Exist(addr) {
			continue
		}
		dst.AddBalance(addr, src.GetBalance(addr), tracing.BalanceIncreaseGenesisBalance)
		dst.SetCode(addr, src.GetCode(addr), tracing.CodeChangeGenesis)
		dst.SetNonce(addr, src.GetNonce(addr), tracing.NonceChangeGenesis)
		for _, slot := range u.slots[addr] {
			if value := src.GetState(addr, slot); value != (common.Hash{}) {
				dst.SetState(addr, slot, value)
			}
		}
	}
	root, err := dst.Commit(0, false, false)
	if err != nil {
		t.Fatal(err)
	}
	return root
}

// seedShadow flushes the genesis allocation into a shadow tree over the given
// database and returns the shadow's genesis root.
func seedShadow(t *testing.T, tdb *triedb.Database, genesis *Genesis) common.Hash {
	t.Helper()
	root, err := flushAlloc(&genesis.Alloc, tdb, nil)
	if err != nil {
		t.Fatal(err)
	}
	return root
}

// runShadowReplay builds a migrating chain, replays every block's access list
// onto a shadow tree seeded from genesis, and checks each shadow root against
// the converter rebuild of the canonical state - the EIP-8347 invariant. A
// second shadow replays the same range through the batching fold and must
// land on the same final root.
func runShadowReplay(t *testing.T, genesis *Genesis, n int, gen func(int, *BlockGen), u *replayUniverse) {
	t.Helper()
	for addr := range genesis.Alloc {
		u.account(addr)
	}
	u.account(common.Address{}) // fee recipient of the generated blocks

	engine := beacon.New(ethash.NewFaker())
	db, blocks, _ := GenerateChainWithGenesis(genesis, engine, n, gen)
	chain, err := NewBlockChain(db, genesis, engine, DefaultConfig().WithStateScheme(rawdb.PathScheme))
	if err != nil {
		t.Fatal(err)
	}
	defer chain.Stop()
	if _, err := chain.InsertChain(blocks); err != nil {
		t.Fatal(err)
	}
	if _, err := chain.SetCanonical(blocks[len(blocks)-1]); err != nil {
		t.Fatal(err)
	}

	// Per-block replay over a shadow living beside the chain's own state.
	shadow := triedb.NewDatabase(db, triedb.PBTDefaults)
	defer shadow.Close()
	var (
		sdb        = state.NewPBTDatabase(shadow, nil)
		shadowRoot = seedShadow(t, shadow, genesis)
	)
	for i, b := range blocks {
		list := rawdb.ReadAccessList(db, b.Hash(), b.NumberU64())
		root, err := replayAccessList(sdb, genesis.Config, shadowRoot, b.Number(), b.Time(), list)
		if err != nil {
			t.Fatalf("block %d: replay: %v", i+1, err)
		}
		shadowRoot = root
		if want := convertCanonical(t, chain, b.Header(), u); root != want {
			t.Fatalf("block %d: shadow root %x, converting the canonical state says %x", i+1, root, want)
		}
	}

	// Batched replay on a fresh shadow: only the end state, same answer.
	batched := triedb.NewDatabase(rawdb.NewMemoryDatabase(), triedb.PBTDefaults)
	defer batched.Close()
	var (
		bdb  = state.NewPBTDatabase(batched, nil)
		root = seedShadow(t, batched, genesis)
		rest = make([]replayBlock, 0, len(blocks))
	)
	for _, b := range blocks {
		rest = append(rest, replayBlock{
			number: b.Number(),
			time:   b.Time(),
			list:   rawdb.ReadAccessList(db, b.Hash(), b.NumberU64()),
		})
	}
	for len(rest) > 0 {
		next, taken, err := replayRange(bdb, genesis.Config, root, rest)
		if err != nil {
			t.Fatalf("batched replay: %v", err)
		}
		if taken == 0 {
			t.Fatal("batched replay made no progress")
		}
		root, rest = next, rest[taken:]
	}
	if root != shadowRoot {
		t.Fatalf("batched shadow root %x diverges from per-block root %x", root, shadowRoot)
	}
}

// storeCalldata is runtime code storing CALLDATALOAD(32) at slot
// CALLDATALOAD(0): a settable storage cell for exercising writes and zeroing.
var storeCalldata = []byte{0x60, 0x20, 0x35, 0x60, 0x00, 0x35, 0x55, 0x00}

// storeTx calls the storage contract, writing value into slot.
func storeTx(t *testing.T, gen *BlockGen, key *ecdsa.PrivateKey, sender, contract common.Address, slot, value common.Hash) {
	t.Helper()
	data := append(slot.Bytes(), value.Bytes()...)
	tx, err := types.SignTx(types.NewTransaction(
		gen.TxNonce(sender), contract, new(big.Int), 1_000_000,
		new(big.Int).Add(gen.BaseFee(), common.Big1), data,
	), gen.Signer(), key)
	if err != nil {
		t.Fatal(err)
	}
	gen.AddTx(tx)
}

// TestShadowReplayTransfers pins plain balance movement and fresh-account
// creation: the shadow must grow the recipient's leaves - code hash included
// - from balance changes alone.
func TestShadowReplayTransfers(t *testing.T) {
	genesis, key, sender, recipient := migrationChainGenesis(t)
	fresh := common.Address{0xf4, 0xe5, 0x11}
	signer := types.LatestSigner(genesis.Config)

	u := newReplayUniverse()
	u.account(recipient, fresh)

	runShadowReplay(t, genesis, 3, func(i int, gen *BlockGen) {
		to := recipient
		if i == 1 {
			to = fresh
		}
		payTo(t, key, sender, to, signer, 1000)(i, gen)
	}, u)
}

// TestShadowReplayContractStorage pins storage writes, overwrites and
// zero-writes in both the header stem (slots 0-63) and the storage zone.
func TestShadowReplayContractStorage(t *testing.T) {
	genesis, key, sender, _ := migrationChainGenesis(t)
	contract := common.Address{0xc0, 0xff, 0xee}
	genesis.Alloc[contract] = types.Account{Balance: big.NewInt(1), Code: storeCalldata}

	var (
		low  = common.BigToHash(big.NewInt(1))   // header stem
		mid  = common.BigToHash(big.NewInt(100)) // storage zone
		high = common.HexToHash("0xffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff0000")
	)
	u := newReplayUniverse()
	u.storage(contract, low, mid, high)

	runShadowReplay(t, genesis, 3, func(i int, gen *BlockGen) {
		switch i {
		case 0:
			storeTx(t, gen, key, sender, contract, low, common.Hash{0xaa})
			storeTx(t, gen, key, sender, contract, mid, common.Hash{0xbb})
			storeTx(t, gen, key, sender, contract, high, common.Hash{0xcc})
		case 1:
			storeTx(t, gen, key, sender, contract, low, common.Hash{0xdd})
		case 2:
			storeTx(t, gen, key, sender, contract, low, common.Hash{})
			storeTx(t, gen, key, sender, contract, mid, common.Hash{})
		}
	}, u)
}

// TestShadowReplayAccountRemoval pins the EIP-161 removal and a later
// recreation: the batch fold must split at the removal, and the shadow must
// drop the victim's header stem and storage-zone leaves before regrowing it.
func TestShadowReplayAccountRemoval(t *testing.T) {
	base, _, _, _ := migrationChainGenesis(t)
	genesis, key, sender, victim := eip161Genesis(t, *base.Config, 48)

	u := newReplayUniverse()
	for _, slot := range pbtStorageSlots(48) {
		u.storage(victim, common.BigToHash(new(big.Int).SetUint64(slot)))
	}
	signer := types.LatestSigner(genesis.Config)

	runShadowReplay(t, genesis, 2, func(i int, gen *BlockGen) {
		switch i {
		case 0:
			pbtCallTx(t, key, sender, victim)(i, gen)
		case 1:
			payTo(t, key, sender, victim, signer, 1000)(i, gen)
		}
	}, u)
}

// TestShadowReplaySelfDestruct pins EIP-8264: a zero-balance account
// destructed in its creation transaction vanishes, a funded one survives as
// balance only, its storage gone either way.
func TestShadowReplaySelfDestruct(t *testing.T) {
	genesis, key, sender, _ := migrationChainGenesis(t)
	beneficiary := common.Address{0xbe, 0xef}

	// SSTORE(5, 0x42), then SELFDESTRUCT to the beneficiary.
	init := []byte{0x60, 0x42, 0x60, 0x05, 0x55, 0x73}
	init = append(init, beneficiary.Bytes()...)
	init = append(init, 0xff)

	u := newReplayUniverse()
	u.account(beneficiary)
	slot := common.BigToHash(big.NewInt(5))

	runShadowReplay(t, genesis, 2, func(i int, gen *BlockGen) {
		value := big.NewInt(0)
		if i == 1 {
			value = big.NewInt(100)
		}
		nonce := gen.TxNonce(sender)
		u.storage(crypto.CreateAddress(sender, nonce), slot)
		tx, err := types.SignTx(types.NewContractCreation(
			nonce, value, 1_000_000, new(big.Int).Add(gen.BaseFee(), common.Big1), init,
		), gen.Signer(), key)
		if err != nil {
			t.Fatal(err)
		}
		gen.AddTx(tx)
	}, u)
}

// TestShadowReplayDelegation pins EIP-7702: setting a delegation swaps the
// code-hash leaf for the delegation leaf, clearing it swaps back.
func TestShadowReplayDelegation(t *testing.T) {
	genesis, key, sender, _ := migrationChainGenesis(t)
	contract := common.Address{0xc0, 0xff, 0xee}
	genesis.Alloc[contract] = types.Account{Balance: big.NewInt(1), Code: storeCalldata}

	authKey, err := crypto.GenerateKey()
	if err != nil {
		t.Fatal(err)
	}
	authority := crypto.PubkeyToAddress(authKey.PublicKey)
	genesis.Alloc[authority] = types.Account{Balance: big.NewInt(1_000_000)}

	u := newReplayUniverse()
	u.account(authority, contract)

	chainID := uint256.MustFromBig(genesis.Config.ChainID)
	setCodeTx := func(gen *BlockGen, to common.Address, authNonce uint64) {
		auth, err := types.SignSetCode(authKey, types.SetCodeAuthorization{
			ChainID: *chainID,
			Address: to,
			Nonce:   authNonce,
		})
		if err != nil {
			t.Fatal(err)
		}
		tx, err := types.SignTx(types.NewTx(&types.SetCodeTx{
			ChainID:   chainID,
			Nonce:     gen.TxNonce(sender),
			To:        contract,
			Value:     new(uint256.Int),
			Gas:       1_000_000,
			GasFeeCap: uint256.MustFromBig(new(big.Int).Add(gen.BaseFee(), common.Big1)),
			GasTipCap: uint256.NewInt(1),
			AuthList:  []types.SetCodeAuthorization{auth},
		}), gen.Signer(), key)
		if err != nil {
			t.Fatal(err)
		}
		gen.AddTx(tx)
	}
	runShadowReplay(t, genesis, 2, func(i int, gen *BlockGen) {
		switch i {
		case 0:
			setCodeTx(gen, contract, 0) // delegate to the contract
		case 1:
			setCodeTx(gen, common.Address{}, 1) // clear the delegation
		}
	}, u)
}

// TestShadowReplayWithdrawals pins that withdrawal credits - post-execution
// system changes carried by the access list - reach the shadow.
func TestShadowReplayWithdrawals(t *testing.T) {
	genesis, _, _, _ := migrationChainGenesis(t)
	payee := common.Address{0x77}

	u := newReplayUniverse()
	u.account(payee)

	runShadowReplay(t, genesis, 2, func(i int, gen *BlockGen) {
		if i == 0 {
			gen.AddWithdrawal(&types.Withdrawal{
				Validator: 7,
				Address:   payee,
				Amount:    1_000_000, // gwei
			})
		}
	}, u)
}

// TestShadowReplayEmptyBlocks pins that state-identical blocks advance the
// replay without moving the root.
func TestShadowReplayEmptyBlocks(t *testing.T) {
	genesis, _, _, _ := migrationChainGenesis(t)
	runShadowReplay(t, genesis, 2, func(int, *BlockGen) {}, newReplayUniverse())
}
