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

package snap

import (
	"bytes"
	"math/big"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/types/bal"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/ethereum/go-ethereum/trie"
	"github.com/ethereum/go-ethereum/triedb"
	"github.com/holiman/uint256"
)

// completeFixture is a chain of three states A, B, C for the complete-phase
// pivot move tests. Pivot A is fully synced by the seed sync, the moves to B
// and C are described by BALs and cover every kind of transition the catch-up
// has to carry into the tries: storage overwrites, deletions and additions, a
// storage trie emptied out entirely, balance, nonce and code changes, a new
// account, and an existing account drained to empty.
type completeFixture struct {
	db     ethdb.Database
	scheme string

	contractAddr common.Address
	contractHash common.Hash
	newAddr      common.Address
	newHash      common.Hash
	drainAddr    common.Address
	drainHash    common.Hash
	code         []byte

	trieA   *trie.Trie
	elemsA  []*kv
	stTrieA *trie.Trie
	stElemA []*kv
	stElemB []*kv       // Expected storage leaves of the contract at B
	goneB   common.Hash // Hash of the slot deleted in B
	elemsC  []*kv

	hdrA, hdrB, hdrC *types.Header
	bals             map[common.Hash]rlp.RawValue
}

func newCompleteFixture(t *testing.T, scheme string) *completeFixture {
	t.Helper()
	f := &completeFixture{
		db:           rawdb.NewMemoryDatabase(),
		scheme:       scheme,
		contractAddr: common.HexToAddress("0x00000000000000000000000000000000c0ffee01"),
		newAddr:      common.HexToAddress("0x00000000000000000000000000000000c0ffee02"),
		drainAddr:    common.HexToAddress("0x00000000000000000000000000000000c0ffee03"),
		code:         []byte{0x60, 0x01, 0x60, 0x02, 0x01},
		bals:         make(map[common.Hash]rlp.RawValue),
	}
	f.contractHash = crypto.Keccak256Hash(f.contractAddr[:])
	f.newHash = crypto.Keccak256Hash(f.newAddr[:])
	f.drainHash = crypto.Keccak256Hash(f.drainAddr[:])

	var (
		slotKeep = common.HexToHash("0x01")
		slotOver = common.HexToHash("0x02")
		slotZero = common.HexToHash("0x03")
		slotNew  = common.HexToHash("0x04")

		vKeep  = common.HexToHash("0x1111")
		vOver0 = common.HexToHash("0x2222")
		vOver1 = common.HexToHash("0x22220000aaaa")
		vZero0 = common.HexToHash("0x3333")
		vNew   = common.HexToHash("0x4444")
	)
	// Storage-less filler accounts plus a pre-funded EOA (nonce 0, no code,
	// no storage) that block B drains to empty.
	_, _, plain, _ := makeAccountTrieWithAddresses(20, scheme)
	drainVal, _ := rlp.EncodeToBytes(&types.StateAccount{
		Nonce:    0,
		Balance:  uint256.NewInt(777),
		Root:     types.EmptyRootHash,
		CodeHash: types.EmptyCodeHash[:],
	})
	plainA := append(append([]*kv{}, plain...), &kv{f.drainHash[:], drainVal})

	// State A: the contract has no code yet and three slots.
	contractA := types.StateAccount{
		Nonce:    7,
		Balance:  uint256.NewInt(123456),
		CodeHash: types.EmptyCodeHash[:],
	}
	slotsA := map[common.Hash]common.Hash{
		slotKeep: vKeep,
		slotOver: vOver0,
		slotZero: vZero0,
	}
	var rootA common.Hash
	f.trieA, f.elemsA, f.stTrieA, f.stElemA, rootA = makeStateWithStorageContract(scheme, plainA, f.contractAddr, contractA, slotsA)

	// State B: storage mutated, contract gets code and a new balance, a new
	// account appears, the pre-funded EOA is drained.
	contractB := types.StateAccount{
		Nonce:    8,
		Balance:  uint256.NewInt(654321),
		CodeHash: crypto.Keccak256(f.code),
	}
	slotsB := map[common.Hash]common.Hash{
		slotKeep: vKeep,
		slotOver: vOver1,
		slotNew:  vNew,
	}
	newVal, _ := rlp.EncodeToBytes(&types.StateAccount{
		Nonce:    1,
		Balance:  uint256.NewInt(5000),
		Root:     types.EmptyRootHash,
		CodeHash: types.EmptyCodeHash[:],
	})
	plainB := append(append([]*kv{}, plain...), &kv{f.newHash[:], newVal})
	var rootB common.Hash
	_, _, _, f.stElemB, rootB = makeStateWithStorageContract(scheme, plainB, f.contractAddr, contractB, slotsB)
	f.goneB = crypto.Keccak256Hash(slotZero[:])

	balB := bal.NewConstructionBlockAccessList()
	balB.StorageWrite(0, f.contractAddr, slotOver, vOver1)
	balB.StorageWrite(0, f.contractAddr, slotZero, common.Hash{})
	balB.StorageWrite(1, f.contractAddr, slotNew, vNew)
	balB.BalanceChange(1, f.contractAddr, uint256.NewInt(654321))
	balB.NonceChange(f.contractAddr, 1, 8)
	balB.CodeChange(f.contractAddr, 1, f.code)
	balB.BalanceChange(2, f.newAddr, uint256.NewInt(5000))
	balB.NonceChange(f.newAddr, 2, 1)
	balB.BalanceChange(3, f.drainAddr, uint256.NewInt(0))

	// State C: the contract's storage is wiped entirely, the new account
	// moves again.
	contractC := contractB
	newValC, _ := rlp.EncodeToBytes(&types.StateAccount{
		Nonce:    1,
		Balance:  uint256.NewInt(6000),
		Root:     types.EmptyRootHash,
		CodeHash: types.EmptyCodeHash[:],
	})
	plainC := append(append([]*kv{}, plain...), &kv{f.newHash[:], newValC})
	var rootC common.Hash
	_, f.elemsC, _, _, rootC = makeStateWithStorageContract(scheme, plainC, f.contractAddr, contractC, nil)

	balC := bal.NewConstructionBlockAccessList()
	balC.StorageWrite(0, f.contractAddr, slotKeep, common.Hash{})
	balC.StorageWrite(0, f.contractAddr, slotOver, common.Hash{})
	balC.StorageWrite(0, f.contractAddr, slotNew, common.Hash{})
	balC.BalanceChange(1, f.newAddr, uint256.NewInt(6000))

	// Headers, linked by parent hash as the catch-up walks them backward.
	var (
		emptyH = common.Hash{}
		zero   = uint64(0)
		numA   = uint64(128)
	)
	mkHeader := func(num uint64, parent common.Hash, root common.Hash, cb *bal.ConstructionBlockAccessList) *types.Header {
		header := &types.Header{
			ParentHash:       parent,
			Number:           new(big.Int).SetUint64(num),
			Root:             root,
			Difficulty:       common.Big0,
			BaseFee:          common.Big0,
			WithdrawalsHash:  &emptyH,
			BlobGasUsed:      &zero,
			ExcessBlobGas:    &zero,
			ParentBeaconRoot: &emptyH,
			RequestsHash:     &emptyH,
		}
		if cb != nil {
			var buf bytes.Buffer
			if err := cb.EncodeRLP(&buf); err != nil {
				t.Fatal(err)
			}
			var b bal.BlockAccessList
			if err := rlp.DecodeBytes(buf.Bytes(), &b); err != nil {
				t.Fatal(err)
			}
			balHash := b.Hash()
			header.BlockAccessListHash = &balHash
			f.bals[header.Hash()] = buf.Bytes()
		}
		rawdb.WriteHeader(f.db, header)
		rawdb.WriteCanonicalHash(f.db, header.Hash(), num)
		return header
	}
	f.hdrA = mkHeader(numA, common.Hash{}, rootA, nil)
	f.hdrB = mkHeader(numA+1, f.hdrA.Hash(), rootB, balB)
	f.hdrC = mkHeader(numA+2, f.hdrB.Hash(), rootC, balC)
	return f
}

// seed runs a full sync (download and trie generation) against pivot A.
func (f *completeFixture) seed(t *testing.T) {
	t.Helper()

	var (
		once   sync.Once
		cancel = make(chan struct{})
		term   = func() { once.Do(func() { close(cancel) }) }
	)
	syncer := newSyncerV2(f.db, f.scheme)
	src := newTestPeerV2("seed", t, term)
	src.accountTrie = f.trieA.Copy()
	src.accountValues = f.elemsA
	src.setStorageTries(map[common.Hash]*trie.Trie{f.contractHash: f.stTrieA})
	src.storageValues = map[common.Hash][]*kv{f.contractHash: f.stElemA}

	syncer.Register(src)
	src.remote = syncer

	done := checkStall(t, term)
	if err := syncer.Sync(f.hdrA, cancel); err != nil {
		t.Fatalf("pivot A sync failed: %v", err)
	}
	close(done)
	if syncer.getPhase() != phaseComplete {
		t.Fatal("seed sync did not complete")
	}
	verifyTrie(f.scheme, f.db, f.hdrA.Root, t)
}

// move runs a Sync against the given target on the seeded database with a
// fresh syncer, as after a restart. The peer serves state too, but a
// completed sync must roll forward purely through the BAL catch-up, so the
// number of state requests it saw is returned for the caller to assert.
func (f *completeFixture) move(t *testing.T, target *types.Header) (error, int32) {
	t.Helper()

	var (
		once   sync.Once
		cancel = make(chan struct{})
		term   = func() { once.Do(func() { close(cancel) }) }
		reqs   atomic.Int32
	)
	syncer := newSyncerV2(f.db, f.scheme)
	src := newTestPeerV2("mover", t, term)
	src.accountTrie = f.trieA.Copy()
	src.accountValues = f.elemsA
	src.setStorageTries(map[common.Hash]*trie.Trie{f.contractHash: f.stTrieA})
	src.storageValues = map[common.Hash][]*kv{f.contractHash: f.stElemA}
	src.accessLists = f.bals

	src.accountRequestV2Handler = func(tp *testPeerV2, id uint64, root common.Hash, origin common.Hash, limit common.Hash, cap int) error {
		reqs.Add(1)
		return defaultAccountRequestHandlerV2(tp, id, root, origin, limit, cap)
	}
	src.storageRequestV2Handler = func(tp *testPeerV2, id uint64, root common.Hash, accounts []common.Hash, origin, limit []byte, max int) error {
		reqs.Add(1)
		return defaultStorageRequestHandlerV2(tp, id, root, accounts, origin, limit, max)
	}
	syncer.Register(src)
	src.remote = syncer
	done := checkStall(t, term)
	err := syncer.Sync(target, cancel)
	close(done)
	if frozen := syncer.FrozenPivot(); frozen != nil {
		t.Fatalf("pivot frozen in the complete phase, got %v", frozen.Number)
	}
	return err, reqs.Load()
}

// assertPersistedRoot checks that the persisted trie is complete for the given
// root. In the path scheme the account root node at the empty path must hash
// to it as well, which is what the trie database checks when it adopts the
// synced state.
func (f *completeFixture) assertPersistedRoot(t *testing.T, root common.Hash) {
	t.Helper()
	if f.scheme == rawdb.PathScheme {
		if have := crypto.Keccak256Hash(rawdb.ReadAccountTrieNode(f.db, nil)); have != root {
			t.Fatalf("persisted account root mismatch: have %x, want %x", have, root)
		}
	} else if !rawdb.HasLegacyTrieNode(f.db, root) {
		t.Fatalf("root node %x not persisted", root)
	}
	verifyTrie(f.scheme, f.db, root, t)
}

// trieAccount reads an account leaf out of the persisted account trie at the
// given root, nil if absent.
func (f *completeFixture) trieAccount(t *testing.T, root common.Hash, hash common.Hash) *types.StateAccount {
	t.Helper()
	tdb := triedb.NewDatabase(rawdb.NewDatabase(f.db), newDbConfig(f.scheme))
	defer tdb.Close()

	tr, err := trie.New(trie.StateTrieID(root), tdb)
	if err != nil {
		t.Fatal(err)
	}
	leaf, err := tr.Get(hash[:])
	if err != nil {
		t.Fatal(err)
	}
	if len(leaf) == 0 {
		return nil
	}
	var account types.StateAccount
	if err := rlp.DecodeBytes(leaf, &account); err != nil {
		t.Fatal(err)
	}
	return &account
}

// flatAccount reads an account out of the flat state, nil if absent.
func (f *completeFixture) flatAccount(t *testing.T, hash common.Hash) *types.StateAccount {
	t.Helper()
	data := rawdb.ReadAccountSnapshot(f.db, hash)
	if len(data) == 0 {
		return nil
	}
	account, err := types.FullAccount(data)
	if err != nil {
		t.Fatalf("failed to decode account %x: %v", hash, err)
	}
	return account
}

// assertStorage checks the contract's storage against the expected leaves, in
// the flat state slot by slot and in the storage trie as a whole, and that the
// given slots are absent from both.
func (f *completeFixture) assertStorage(t *testing.T, stateRoot, storageRoot common.Hash, want []*kv, absent ...common.Hash) {
	t.Helper()
	for _, e := range want {
		if have := rawdb.ReadStorageSnapshot(f.db, f.contractHash, common.BytesToHash(e.k)); !bytes.Equal(have, e.v) {
			t.Fatalf("flat slot %x: have %x, want %x", e.k, have, e.v)
		}
	}
	for _, hash := range absent {
		if have := rawdb.ReadStorageSnapshot(f.db, f.contractHash, hash); len(have) != 0 {
			t.Fatalf("flat slot %x still present: %x", hash, have)
		}
	}
	tdb := triedb.NewDatabase(rawdb.NewDatabase(f.db), newDbConfig(f.scheme))
	defer tdb.Close()

	tr, err := trie.New(trie.StorageTrieID(stateRoot, f.contractHash, storageRoot), tdb)
	if err != nil {
		t.Fatal(err)
	}
	var leaves []*kv
	it := trie.NewIterator(tr.MustNodeIterator(nil))
	for it.Next() {
		leaves = append(leaves, &kv{bytes.Clone(it.Key), bytes.Clone(it.Value)})
	}
	if it.Err != nil {
		t.Fatal(it.Err)
	}
	if len(leaves) != len(want) {
		t.Fatalf("storage trie has %d leaves, want %d", len(leaves), len(want))
	}
	for i, e := range want { // both sorted by key
		if !bytes.Equal(leaves[i].k, e.k) || !bytes.Equal(leaves[i].v, e.v) {
			t.Fatalf("storage trie leaf %d: have %x=%x, want %x=%x", i, leaves[i].k, leaves[i].v, e.k, e.v)
		}
	}
	for _, hash := range absent {
		if have, _ := tr.Get(hash[:]); len(have) != 0 {
			t.Fatalf("storage trie slot %x still present: %x", hash, have)
		}
	}
}

// sameAccount reports whether two accounts hold identical fields.
func sameAccount(a, b *types.StateAccount) bool {
	return a.Nonce == b.Nonce && a.Balance.Cmp(b.Balance) == 0 && a.Root == b.Root && bytes.Equal(a.CodeHash, b.CodeHash)
}

// assertJournal checks the persisted journal is complete at the given pivot.
func (f *completeFixture) assertJournal(t *testing.T, pivot *types.Header) {
	t.Helper()
	progress := decodeJournal(t, f.db)
	if progress.Phase != phaseComplete {
		t.Fatalf("journal phase %d, want complete", progress.Phase)
	}
	if progress.Pivot == nil || progress.Pivot.Hash() != pivot.Hash() {
		t.Fatalf("journal pivot %v, want %v", progress.Pivot, pivot.Number)
	}
}

// TestCompletePhasePivotMove moves the pivot of a completed sync twice. The
// moves must be pure catch-up, no state re-download, and must carry every
// kind of state transition into both the flat state and the tries, leaving
// the sync complete at the new pivot with the persisted trie matching its
// root.
func TestCompletePhasePivotMove(t *testing.T) {
	t.Parallel()
	testCompletePhasePivotMove(t, rawdb.HashScheme)
	testCompletePhasePivotMove(t, rawdb.PathScheme)
}

func testCompletePhasePivotMove(t *testing.T, scheme string) {
	f := newCompleteFixture(t, scheme)
	f.seed(t)

	// A -> B: storage mutations, code, new and drained accounts.
	if err, reqs := f.move(t, f.hdrB); err != nil || reqs != 0 {
		t.Fatalf("move to B: err %v, state requests %d", err, reqs)
	}
	f.assertJournal(t, f.hdrB)
	f.assertPersistedRoot(t, f.hdrB.Root)

	// The contract picked up its balance, nonce and code, and the flat
	// account carries the live storage root, the same one the trie holds.
	contract := f.flatAccount(t, f.contractHash)
	if contract == nil || contract.Nonce != 8 || contract.Balance.Uint64() != 654321 || !bytes.Equal(contract.CodeHash, crypto.Keccak256(f.code)) {
		t.Fatalf("contract account after B: %+v", contract)
	}
	if contract.Root == types.EmptyRootHash {
		t.Fatal("contract storage root not carried into the flat account")
	}
	if inTrie := f.trieAccount(t, f.hdrB.Root, f.contractHash); inTrie == nil || !sameAccount(inTrie, contract) {
		t.Fatalf("contract in trie %+v differs from flat state %+v", inTrie, contract)
	}
	if len(rawdb.ReadCode(f.db, common.BytesToHash(contract.CodeHash))) == 0 {
		t.Fatal("contract code not persisted")
	}
	// Slot by slot: the kept one unchanged, the overwritten and the new one
	// at their B values, the zeroed one gone, in flat state and trie alike.
	f.assertStorage(t, f.hdrB.Root, contract.Root, f.stElemB, f.goneB)

	// The new account
	if acc := f.flatAccount(t, f.newHash); acc == nil || acc.Balance.Uint64() != 5000 || acc.Nonce != 1 {
		t.Fatalf("new account after B: %+v", acc)
	}
	if acc := f.trieAccount(t, f.hdrB.Root, f.newHash); acc == nil || acc.Balance.Uint64() != 5000 || acc.Nonce != 1 {
		t.Fatalf("new account in trie after B: %+v", acc)
	}

	// The drained account
	if acc := f.flatAccount(t, f.drainHash); acc != nil {
		t.Fatalf("drained account still in flat state after B: %+v", acc)
	}
	if acc := f.trieAccount(t, f.hdrB.Root, f.drainHash); acc != nil {
		t.Fatalf("drained account still in trie after B: %+v", acc)
	}

	// B -> C: the storage trie is emptied out, the new account moves again.
	if err, reqs := f.move(t, f.hdrC); err != nil || reqs != 0 {
		t.Fatalf("move to C: err %v, state requests %d", err, reqs)
	}
	f.assertJournal(t, f.hdrC)
	f.assertPersistedRoot(t, f.hdrC.Root)

	contract = f.flatAccount(t, f.contractHash)
	if contract == nil || contract.Root != types.EmptyRootHash {
		t.Fatalf("contract account after C: %+v", contract)
	}
	if inTrie := f.trieAccount(t, f.hdrC.Root, f.contractHash); inTrie == nil || !sameAccount(inTrie, contract) {
		t.Fatalf("contract in trie %+v differs from flat state %+v", inTrie, contract)
	}
	it := f.db.NewIterator(append(bytes.Clone(rawdb.SnapshotStoragePrefix), f.contractHash[:]...), nil)
	if it.Next() {
		t.Fatalf("stale flat storage after C: %x", it.Key())
	}
	it.Release()

	// Emptying the storage trie deletes its nodes. Only the path scheme
	// removes nodes, hash scheme nodes are content addressed and kept.
	if scheme == rawdb.PathScheme {
		it := f.db.NewIterator(append(bytes.Clone(rawdb.TrieNodeStoragePrefix), f.contractHash[:]...), nil)
		if it.Next() {
			t.Fatalf("stale storage trie node after C: %x", it.Key())
		}
		it.Release()
	}

	if acc := f.flatAccount(t, f.newHash); acc == nil || acc.Balance.Uint64() != 6000 {
		t.Fatalf("new account after C: %+v", acc)
	}

	// A same-pivot retry is a no-op skip, as for any completed sync.
	if err, reqs := f.move(t, f.hdrC); err != nil || reqs != 0 {
		t.Fatalf("retry at C: err %v, state requests %d", err, reqs)
	}
	// The result is adoptable by the trie database like a freshly generated one.
	verifyAdoptedSyncedState(scheme, f.db, f.hdrC.Root, f.elemsC, t)
}

// TestCompletePhaseRootMismatch moves the pivot of a completed sync to a
// target whose header root does not match the BAL-derived state. The block
// must be rejected as a whole: the journal, the flat state and the tries all
// stay at the last block that verified, so the sync remains complete and
// consistent for it.
func TestCompletePhaseRootMismatch(t *testing.T) {
	t.Parallel()
	testCompletePhaseRootMismatch(t, rawdb.HashScheme)
	testCompletePhaseRootMismatch(t, rawdb.PathScheme)
}

func testCompletePhaseRootMismatch(t *testing.T, scheme string) {
	f := newCompleteFixture(t, scheme)
	f.seed(t)

	// Corrupt the root of C. The BAL is still valid against the header, so
	// the catch-up fetches and applies it, and only the root check fails.
	bad := *f.hdrC
	bad.Root = common.HexToHash("0xbad")
	f.bals[bad.Hash()] = f.bals[f.hdrC.Hash()]
	rawdb.WriteHeader(f.db, &bad)
	rawdb.WriteCanonicalHash(f.db, bad.Hash(), bad.Number.Uint64())

	err, reqs := f.move(t, &bad)
	if err == nil {
		t.Fatal("expected the move to fail on the root mismatch")
	}
	if reqs != 0 {
		t.Fatalf("expected no state requests, got %d", reqs)
	}
	// B applied and persisted, the bad C left no trace.
	f.assertJournal(t, f.hdrB)
	f.assertPersistedRoot(t, f.hdrB.Root)
	if acc := f.flatAccount(t, f.newHash); acc == nil || acc.Balance.Uint64() != 5000 {
		t.Fatalf("new account after failed move: %+v", acc)
	}
	if contract := f.flatAccount(t, f.contractHash); contract == nil || contract.Root == types.EmptyRootHash {
		t.Fatalf("contract storage wrongly wiped by the rejected block: %+v", contract)
	}
}
