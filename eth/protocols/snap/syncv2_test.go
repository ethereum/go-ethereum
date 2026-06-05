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
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/ethereum/go-ethereum/trie"
	"github.com/ethereum/go-ethereum/trie/trienode"
)

// SyncerV2 (skeleton) only downloads the flat state (accounts, storage slots,
// bytecodes) and does not perform trie generation or state healing. These tests
// verify that, in a single uninterrupted sync cycle, the syncer fully downloads
// all the expected flat state from the source peer(s).

type (
	accountHandlerFuncV2 func(t *testPeerV2, requestId uint64, root common.Hash, origin common.Hash, limit common.Hash, cap int) error
	storageHandlerFuncV2 func(t *testPeerV2, requestId uint64, root common.Hash, accounts []common.Hash, origin, limit []byte, max int) error
	codeHandlerFuncV2    func(t *testPeerV2, id uint64, hashes []common.Hash, max int) error
)

type testPeerV2 struct {
	id            string
	test          *testing.T
	remote        *SyncerV2
	logger        log.Logger
	accountTrie   *trie.Trie
	accountValues []*kv
	storageTries  map[common.Hash]*trie.Trie
	storageValues map[common.Hash][]*kv

	accountRequestHandler accountHandlerFuncV2
	storageRequestHandler storageHandlerFuncV2
	codeRequestHandler    codeHandlerFuncV2
	term                  func()

	// counters
	nAccountRequests  atomic.Int64
	nStorageRequests  atomic.Int64
	nBytecodeRequests atomic.Int64
}

func newTestPeerV2(id string, t *testing.T, term func()) *testPeerV2 {
	return &testPeerV2{
		id:                    id,
		test:                  t,
		logger:                log.New("id", id),
		accountRequestHandler: defaultAccountRequestHandlerV2,
		storageRequestHandler: defaultStorageRequestHandlerV2,
		codeRequestHandler:    defaultCodeRequestHandlerV2,
		term:                  term,
	}
}

func (t *testPeerV2) setStorageTries(tries map[common.Hash]*trie.Trie) {
	t.storageTries = make(map[common.Hash]*trie.Trie)
	for root, trie := range tries {
		t.storageTries[root] = trie.Copy()
	}
}

func (t *testPeerV2) ID() string      { return t.id }
func (t *testPeerV2) Log() log.Logger { return t.logger }

func (t *testPeerV2) Stats() string {
	return fmt.Sprintf(`Account requests: %d
Storage requests: %d
Bytecode requests: %d
`, t.nAccountRequests.Load(), t.nStorageRequests.Load(), t.nBytecodeRequests.Load())
}

func (t *testPeerV2) RequestAccountRange(id uint64, root, origin, limit common.Hash, bytes int) error {
	t.logger.Trace("Fetching range of accounts", "reqid", id, "root", root, "origin", origin, "limit", limit, "bytes", common.StorageSize(bytes))
	t.nAccountRequests.Add(1)
	go t.accountRequestHandler(t, id, root, origin, limit, bytes)
	return nil
}

func (t *testPeerV2) RequestStorageRanges(id uint64, root common.Hash, accounts []common.Hash, origin, limit []byte, bytes int) error {
	t.nStorageRequests.Add(1)
	if len(accounts) == 1 && origin != nil {
		t.logger.Trace("Fetching range of large storage slots", "reqid", id, "root", root, "account", accounts[0], "origin", common.BytesToHash(origin), "limit", common.BytesToHash(limit), "bytes", common.StorageSize(bytes))
	} else {
		t.logger.Trace("Fetching ranges of small storage slots", "reqid", id, "root", root, "accounts", len(accounts), "first", accounts[0], "bytes", common.StorageSize(bytes))
	}
	go t.storageRequestHandler(t, id, root, accounts, origin, limit, bytes)
	return nil
}

func (t *testPeerV2) RequestByteCodes(id uint64, hashes []common.Hash, bytes int) error {
	t.nBytecodeRequests.Add(1)
	t.logger.Trace("Fetching set of byte codes", "reqid", id, "hashes", len(hashes), "bytes", common.StorageSize(bytes))
	go t.codeRequestHandler(t, id, hashes, bytes)
	return nil
}

func createAccountRequestResponseV2(t *testPeerV2, root common.Hash, origin common.Hash, limit common.Hash, cap int) (keys []common.Hash, vals [][]byte, proofs [][]byte) {
	var size int
	if limit == (common.Hash{}) {
		limit = common.MaxHash
	}
	for _, entry := range t.accountValues {
		if size > cap {
			break
		}
		if bytes.Compare(origin[:], entry.k) <= 0 {
			keys = append(keys, common.BytesToHash(entry.k))
			vals = append(vals, entry.v)
			size += 32 + len(entry.v)
		}
		if bytes.Compare(entry.k, limit[:]) >= 0 {
			break
		}
	}
	proof := trienode.NewProofSet()
	if err := t.accountTrie.Prove(origin[:], proof); err != nil {
		t.logger.Error("Could not prove inexistence of origin", "origin", origin, "error", err)
	}
	if len(keys) > 0 {
		lastK := (keys[len(keys)-1])[:]
		if err := t.accountTrie.Prove(lastK, proof); err != nil {
			t.logger.Error("Could not prove last item", "error", err)
		}
	}
	return keys, vals, proof.List()
}

func createStorageRequestResponseV2(t *testPeerV2, root common.Hash, accounts []common.Hash, origin, limit []byte, max int) (hashes [][]common.Hash, slots [][][]byte, proofs [][]byte) {
	var size int
	for _, account := range accounts {
		var originHash common.Hash
		if len(origin) > 0 {
			originHash = common.BytesToHash(origin)
		}
		var limitHash = common.MaxHash
		if len(limit) > 0 {
			limitHash = common.BytesToHash(limit)
		}
		var (
			keys  []common.Hash
			vals  [][]byte
			abort bool
		)
		for _, entry := range t.storageValues[account] {
			if size >= max {
				abort = true
				break
			}
			if bytes.Compare(entry.k, originHash[:]) < 0 {
				continue
			}
			keys = append(keys, common.BytesToHash(entry.k))
			vals = append(vals, entry.v)
			size += 32 + len(entry.v)
			if bytes.Compare(entry.k, limitHash[:]) >= 0 {
				break
			}
		}
		if len(keys) > 0 {
			hashes = append(hashes, keys)
			slots = append(slots, vals)
		}
		if originHash != (common.Hash{}) || (abort && len(keys) > 0) {
			proof := trienode.NewProofSet()
			stTrie := t.storageTries[account]

			if err := stTrie.Prove(originHash[:], proof); err != nil {
				t.logger.Error("Could not prove inexistence of origin", "origin", originHash, "error", err)
			}
			if len(keys) > 0 {
				lastK := (keys[len(keys)-1])[:]
				if err := stTrie.Prove(lastK, proof); err != nil {
					t.logger.Error("Could not prove last item", "error", err)
				}
			}
			proofs = append(proofs, proof.List()...)
			break
		}
	}
	return hashes, slots, proofs
}

func createStorageRequestResponseAlwaysProveV2(t *testPeerV2, root common.Hash, accounts []common.Hash, bOrigin, bLimit []byte, max int) (hashes [][]common.Hash, slots [][][]byte, proofs [][]byte) {
	var size int
	max = max * 3 / 4

	var origin common.Hash
	if len(bOrigin) > 0 {
		origin = common.BytesToHash(bOrigin)
	}
	var exit bool
	for i, account := range accounts {
		var keys []common.Hash
		var vals [][]byte
		for _, entry := range t.storageValues[account] {
			if bytes.Compare(entry.k, origin[:]) < 0 {
				exit = true
			}
			keys = append(keys, common.BytesToHash(entry.k))
			vals = append(vals, entry.v)
			size += 32 + len(entry.v)
			if size > max {
				exit = true
			}
		}
		if i == len(accounts)-1 {
			exit = true
		}
		hashes = append(hashes, keys)
		slots = append(slots, vals)

		if exit {
			proof := trienode.NewProofSet()
			stTrie := t.storageTries[account]

			if err := stTrie.Prove(origin[:], proof); err != nil {
				t.logger.Error("Could not prove inexistence of origin", "origin", origin, "error", err)
			}
			if len(keys) > 0 {
				lastK := (keys[len(keys)-1])[:]
				if err := stTrie.Prove(lastK, proof); err != nil {
					t.logger.Error("Could not prove last item", "error", err)
				}
			}
			proofs = append(proofs, proof.List()...)
			break
		}
	}
	return hashes, slots, proofs
}

// defaultAccountRequestHandlerV2 is a well-behaving handler for AccountRangeRequests.
func defaultAccountRequestHandlerV2(t *testPeerV2, id uint64, root common.Hash, origin common.Hash, limit common.Hash, cap int) error {
	keys, vals, proofs := createAccountRequestResponseV2(t, root, origin, limit, cap)
	if err := t.remote.OnAccounts(t, id, keys, vals, proofs); err != nil {
		t.test.Errorf("Remote side rejected our delivery: %v", err)
		t.term()
		return err
	}
	return nil
}

func defaultStorageRequestHandlerV2(t *testPeerV2, requestId uint64, root common.Hash, accounts []common.Hash, bOrigin, bLimit []byte, max int) error {
	hashes, slots, proofs := createStorageRequestResponseV2(t, root, accounts, bOrigin, bLimit, max)
	if err := t.remote.OnStorage(t, requestId, hashes, slots, proofs); err != nil {
		t.test.Errorf("Remote side rejected our delivery: %v", err)
		t.term()
	}
	return nil
}

func defaultCodeRequestHandlerV2(t *testPeerV2, id uint64, hashes []common.Hash, max int) error {
	var bytecodes [][]byte
	for _, h := range hashes {
		bytecodes = append(bytecodes, getCodeByHash(h))
	}
	if err := t.remote.OnByteCodes(t, id, bytecodes); err != nil {
		t.test.Errorf("Remote side rejected our delivery: %v", err)
		t.term()
	}
	return nil
}

// Misbehaving handlers.

func emptyRequestAccountRangeFnV2(t *testPeerV2, requestId uint64, root common.Hash, origin common.Hash, limit common.Hash, cap int) error {
	t.remote.OnAccounts(t, requestId, nil, nil, nil)
	return nil
}

func nonResponsiveRequestAccountRangeFnV2(t *testPeerV2, requestId uint64, root common.Hash, origin common.Hash, limit common.Hash, cap int) error {
	return nil
}

func emptyStorageRequestHandlerV2(t *testPeerV2, requestId uint64, root common.Hash, accounts []common.Hash, origin, limit []byte, max int) error {
	t.remote.OnStorage(t, requestId, nil, nil, nil)
	return nil
}

func nonResponsiveStorageRequestHandlerV2(t *testPeerV2, requestId uint64, root common.Hash, accounts []common.Hash, origin, limit []byte, max int) error {
	return nil
}

func proofHappyStorageRequestHandlerV2(t *testPeerV2, requestId uint64, root common.Hash, accounts []common.Hash, origin, limit []byte, max int) error {
	hashes, slots, proofs := createStorageRequestResponseAlwaysProveV2(t, root, accounts, origin, limit, max)
	if err := t.remote.OnStorage(t, requestId, hashes, slots, proofs); err != nil {
		t.test.Errorf("Remote side rejected our delivery: %v", err)
		t.term()
	}
	return nil
}

func corruptCodeRequestHandlerV2(t *testPeerV2, id uint64, hashes []common.Hash, max int) error {
	var bytecodes [][]byte
	for _, h := range hashes {
		bytecodes = append(bytecodes, h[:])
	}
	if err := t.remote.OnByteCodes(t, id, bytecodes); err != nil {
		t.logger.Info("remote error on delivery (as expected)", "error", err)
		t.remote.Unregister(t.id)
	}
	return nil
}

func cappedCodeRequestHandlerV2(t *testPeerV2, id uint64, hashes []common.Hash, max int) error {
	var bytecodes [][]byte
	for _, h := range hashes[:1] {
		bytecodes = append(bytecodes, getCodeByHash(h))
	}
	if err := t.remote.OnByteCodes(t, id, bytecodes); err != nil {
		t.test.Errorf("Remote side rejected our delivery: %v", err)
		t.term()
	}
	return nil
}

func starvingStorageRequestHandlerV2(t *testPeerV2, requestId uint64, root common.Hash, accounts []common.Hash, origin, limit []byte, max int) error {
	return defaultStorageRequestHandlerV2(t, requestId, root, accounts, origin, limit, 500)
}

func starvingAccountRequestHandlerV2(t *testPeerV2, requestId uint64, root common.Hash, origin common.Hash, limit common.Hash, cap int) error {
	return defaultAccountRequestHandlerV2(t, requestId, root, origin, limit, 500)
}

func corruptAccountRequestHandlerV2(t *testPeerV2, requestId uint64, root common.Hash, origin common.Hash, limit common.Hash, cap int) error {
	hashes, accounts, proofs := createAccountRequestResponseV2(t, root, origin, limit, cap)
	if len(proofs) > 0 {
		proofs = proofs[1:]
	}
	if err := t.remote.OnAccounts(t, requestId, hashes, accounts, proofs); err != nil {
		t.logger.Info("remote error on delivery (as expected)", "error", err)
		t.remote.Unregister(t.id)
	}
	return nil
}

func corruptStorageRequestHandlerV2(t *testPeerV2, requestId uint64, root common.Hash, accounts []common.Hash, origin, limit []byte, max int) error {
	hashes, slots, proofs := createStorageRequestResponseV2(t, root, accounts, origin, limit, max)
	if len(proofs) > 0 {
		proofs = proofs[1:]
	}
	if err := t.remote.OnStorage(t, requestId, hashes, slots, proofs); err != nil {
		t.logger.Info("remote error on delivery (as expected)", "error", err)
		t.remote.Unregister(t.id)
	}
	return nil
}

func noProofStorageRequestHandlerV2(t *testPeerV2, requestId uint64, root common.Hash, accounts []common.Hash, origin, limit []byte, max int) error {
	hashes, slots, _ := createStorageRequestResponseV2(t, root, accounts, origin, limit, max)
	if err := t.remote.OnStorage(t, requestId, hashes, slots, nil); err != nil {
		t.logger.Info("remote error on delivery (as expected)", "error", err)
		t.remote.Unregister(t.id)
	}
	return nil
}

func setupSyncerV2(scheme string, peers ...*testPeerV2) *SyncerV2 {
	stateDb := rawdb.NewMemoryDatabase()
	syncer := NewSyncerV2(stateDb, scheme)
	for _, peer := range peers {
		syncer.Register(peer)
		peer.remote = syncer
	}
	return syncer
}

// verifyFlatState checks that the database contains the snapshot entries for
// every expected account and storage slot, plus the bytecode for every account
// that has one. Trie node presence is intentionally not checked: SyncerV2 only
// downloads flat state.
func verifyFlatState(t *testing.T, db ethdb.KeyValueStore, accountValues []*kv, storageValues map[common.Hash][]*kv) {
	t.Helper()

	for _, entry := range accountValues {
		hash := common.BytesToHash(entry.k)
		got := rawdb.ReadAccountSnapshot(db, hash)
		if got == nil {
			t.Fatalf("missing account snapshot for %x", hash)
		}
		var acc types.StateAccount
		if err := rlp.DecodeBytes(entry.v, &acc); err != nil {
			t.Fatalf("failed to decode source account %x: %v", hash, err)
		}
		want := types.SlimAccountRLP(acc)
		if !bytes.Equal(got, want) {
			t.Fatalf("account snapshot mismatch for %x:\n got  %x\n want %x", hash, got, want)
		}
		if !bytes.Equal(acc.CodeHash, types.EmptyCodeHash.Bytes()) {
			if !rawdb.HasCode(db, common.BytesToHash(acc.CodeHash)) {
				t.Fatalf("missing code for hash %x (account %x)", acc.CodeHash, hash)
			}
		}
	}
	var accounts, slots int
	for _, entry := range accountValues {
		accounts++
		account := common.BytesToHash(entry.k)
		for _, slot := range storageValues[account] {
			slotHash := common.BytesToHash(slot.k)
			got := rawdb.ReadStorageSnapshot(db, account, slotHash)
			if got == nil {
				t.Fatalf("missing storage snapshot for account %x slot %x", account, slotHash)
			}
			if !bytes.Equal(got, slot.v) {
				t.Fatalf("storage snapshot mismatch for account %x slot %x:\n got  %x\n want %x", account, slotHash, got, slot.v)
			}
			slots++
		}
	}
	t.Logf("flat state verified: accounts=%d slots=%d", accounts, slots)
}

// TestSyncV2 tests a basic sync with one peer.
func TestSyncV2(t *testing.T) {
	t.Parallel()
	testSyncV2(t, rawdb.HashScheme)
	testSyncV2(t, rawdb.PathScheme)
}

func testSyncV2(t *testing.T, scheme string) {
	var (
		once   sync.Once
		cancel = make(chan struct{})
		term   = func() { once.Do(func() { close(cancel) }) }
	)
	nodeScheme, sourceAccountTrie, elems := makeAccountTrieNoStorage(100, scheme)

	mkSource := func(name string) *testPeerV2 {
		source := newTestPeerV2(name, t, term)
		source.accountTrie = sourceAccountTrie.Copy()
		source.accountValues = elems
		return source
	}
	syncer := setupSyncerV2(nodeScheme, mkSource("source"))
	if err := syncer.Sync(sourceAccountTrie.Hash(), cancel); err != nil {
		t.Fatalf("sync failed: %v", err)
	}
	verifyFlatState(t, syncer.db, elems, nil)
}

// TestSyncTinyTriePanicV2 tests a basic sync with one peer and a tiny trie.
func TestSyncTinyTriePanicV2(t *testing.T) {
	t.Parallel()
	testSyncTinyTriePanicV2(t, rawdb.HashScheme)
	testSyncTinyTriePanicV2(t, rawdb.PathScheme)
}

func testSyncTinyTriePanicV2(t *testing.T, scheme string) {
	var (
		once   sync.Once
		cancel = make(chan struct{})
		term   = func() { once.Do(func() { close(cancel) }) }
	)
	nodeScheme, sourceAccountTrie, elems := makeAccountTrieNoStorage(1, scheme)

	mkSource := func(name string) *testPeerV2 {
		source := newTestPeerV2(name, t, term)
		source.accountTrie = sourceAccountTrie.Copy()
		source.accountValues = elems
		return source
	}
	syncer := setupSyncerV2(nodeScheme, mkSource("source"))
	done := checkStall(t, term)
	if err := syncer.Sync(sourceAccountTrie.Hash(), cancel); err != nil {
		t.Fatalf("sync failed: %v", err)
	}
	close(done)
	verifyFlatState(t, syncer.db, elems, nil)
}

// TestMultiSyncV2 tests a basic sync with multiple peers.
func TestMultiSyncV2(t *testing.T) {
	t.Parallel()
	testMultiSyncV2(t, rawdb.HashScheme)
	testMultiSyncV2(t, rawdb.PathScheme)
}

func testMultiSyncV2(t *testing.T, scheme string) {
	var (
		once   sync.Once
		cancel = make(chan struct{})
		term   = func() { once.Do(func() { close(cancel) }) }
	)
	nodeScheme, sourceAccountTrie, elems := makeAccountTrieNoStorage(100, scheme)

	mkSource := func(name string) *testPeerV2 {
		source := newTestPeerV2(name, t, term)
		source.accountTrie = sourceAccountTrie.Copy()
		source.accountValues = elems
		return source
	}
	syncer := setupSyncerV2(nodeScheme, mkSource("sourceA"), mkSource("sourceB"))
	done := checkStall(t, term)
	if err := syncer.Sync(sourceAccountTrie.Hash(), cancel); err != nil {
		t.Fatalf("sync failed: %v", err)
	}
	close(done)
	verifyFlatState(t, syncer.db, elems, nil)
}

// TestSyncWithStorageV2 tests basic sync using accounts + storage + code.
func TestSyncWithStorageV2(t *testing.T) {
	t.Parallel()
	testSyncWithStorageV2(t, rawdb.HashScheme)
	testSyncWithStorageV2(t, rawdb.PathScheme)
}

func testSyncWithStorageV2(t *testing.T, scheme string) {
	var (
		once   sync.Once
		cancel = make(chan struct{})
		term   = func() { once.Do(func() { close(cancel) }) }
	)
	sourceAccountTrie, elems, storageTries, storageElems := makeAccountTrieWithStorage(scheme, 3, 3000, true, false, false)

	mkSource := func(name string) *testPeerV2 {
		source := newTestPeerV2(name, t, term)
		source.accountTrie = sourceAccountTrie.Copy()
		source.accountValues = elems
		source.setStorageTries(storageTries)
		source.storageValues = storageElems
		return source
	}
	syncer := setupSyncerV2(scheme, mkSource("sourceA"))
	done := checkStall(t, term)
	if err := syncer.Sync(sourceAccountTrie.Hash(), cancel); err != nil {
		t.Fatalf("sync failed: %v", err)
	}
	close(done)
	verifyFlatState(t, syncer.db, elems, storageElems)
}

// TestMultiSyncManyUselessV2 keeps one good peer and several that return empty.
func TestMultiSyncManyUselessV2(t *testing.T) {
	t.Parallel()
	testMultiSyncManyUselessV2(t, rawdb.HashScheme)
	testMultiSyncManyUselessV2(t, rawdb.PathScheme)
}

func testMultiSyncManyUselessV2(t *testing.T, scheme string) {
	var (
		once   sync.Once
		cancel = make(chan struct{})
		term   = func() { once.Do(func() { close(cancel) }) }
	)
	sourceAccountTrie, elems, storageTries, storageElems := makeAccountTrieWithStorage(scheme, 100, 3000, true, false, false)

	mkSource := func(name string, noAccount, noStorage bool) *testPeerV2 {
		source := newTestPeerV2(name, t, term)
		source.accountTrie = sourceAccountTrie.Copy()
		source.accountValues = elems
		source.setStorageTries(storageTries)
		source.storageValues = storageElems
		if noAccount {
			source.accountRequestHandler = emptyRequestAccountRangeFnV2
		}
		if noStorage {
			source.storageRequestHandler = emptyStorageRequestHandlerV2
		}
		return source
	}
	syncer := setupSyncerV2(
		scheme,
		mkSource("full", false, false),
		mkSource("noAccounts", true, false),
		mkSource("noStorage", false, true),
	)
	done := checkStall(t, term)
	if err := syncer.Sync(sourceAccountTrie.Hash(), cancel); err != nil {
		t.Fatalf("sync failed: %v", err)
	}
	close(done)
	verifyFlatState(t, syncer.db, elems, storageElems)
}

// TestMultiSyncManyUselessWithLowTimeoutV2 is the same as above but with a very
// low timeout, exercising the timeout/reschedule paths.
func TestMultiSyncManyUselessWithLowTimeoutV2(t *testing.T) {
	t.Parallel()
	testMultiSyncManyUselessWithLowTimeoutV2(t, rawdb.HashScheme)
	testMultiSyncManyUselessWithLowTimeoutV2(t, rawdb.PathScheme)
}

func testMultiSyncManyUselessWithLowTimeoutV2(t *testing.T, scheme string) {
	var (
		once   sync.Once
		cancel = make(chan struct{})
		term   = func() { once.Do(func() { close(cancel) }) }
	)
	sourceAccountTrie, elems, storageTries, storageElems := makeAccountTrieWithStorage(scheme, 100, 3000, true, false, false)

	mkSource := func(name string, noAccount, noStorage bool) *testPeerV2 {
		source := newTestPeerV2(name, t, term)
		source.accountTrie = sourceAccountTrie.Copy()
		source.accountValues = elems
		source.setStorageTries(storageTries)
		source.storageValues = storageElems
		if !noAccount {
			source.accountRequestHandler = emptyRequestAccountRangeFnV2
		}
		if !noStorage {
			source.storageRequestHandler = emptyStorageRequestHandlerV2
		}
		return source
	}
	syncer := setupSyncerV2(
		scheme,
		mkSource("full", true, true),
		mkSource("noAccounts", false, true),
		mkSource("noStorage", true, false),
	)
	syncer.rates.OverrideTTLLimit = time.Millisecond

	done := checkStall(t, term)
	if err := syncer.Sync(sourceAccountTrie.Hash(), cancel); err != nil {
		t.Fatalf("sync failed: %v", err)
	}
	close(done)
	verifyFlatState(t, syncer.db, elems, storageElems)
}

// TestMultiSyncManyUnresponsiveV2 keeps one good peer and several that don't
// respond at all.
func TestMultiSyncManyUnresponsiveV2(t *testing.T) {
	t.Parallel()
	testMultiSyncManyUnresponsiveV2(t, rawdb.HashScheme)
	testMultiSyncManyUnresponsiveV2(t, rawdb.PathScheme)
}

func testMultiSyncManyUnresponsiveV2(t *testing.T, scheme string) {
	var (
		once   sync.Once
		cancel = make(chan struct{})
		term   = func() { once.Do(func() { close(cancel) }) }
	)
	sourceAccountTrie, elems, storageTries, storageElems := makeAccountTrieWithStorage(scheme, 100, 3000, true, false, false)

	mkSource := func(name string, noAccount, noStorage bool) *testPeerV2 {
		source := newTestPeerV2(name, t, term)
		source.accountTrie = sourceAccountTrie.Copy()
		source.accountValues = elems
		source.setStorageTries(storageTries)
		source.storageValues = storageElems
		if noAccount {
			source.accountRequestHandler = nonResponsiveRequestAccountRangeFnV2
		}
		if noStorage {
			source.storageRequestHandler = nonResponsiveStorageRequestHandlerV2
		}
		return source
	}
	syncer := setupSyncerV2(
		scheme,
		mkSource("full", false, false),
		mkSource("noAccounts", true, false),
		mkSource("noStorage", false, true),
	)
	syncer.rates.OverrideTTLLimit = time.Millisecond

	done := checkStall(t, term)
	if err := syncer.Sync(sourceAccountTrie.Hash(), cancel); err != nil {
		t.Fatalf("sync failed: %v", err)
	}
	close(done)
	verifyFlatState(t, syncer.db, elems, storageElems)
}

// TestSyncBoundaryAccountTrieV2 tests sync against a few normal peers, but the
// account trie has a few boundary elements.
func TestSyncBoundaryAccountTrieV2(t *testing.T) {
	t.Parallel()
	testSyncBoundaryAccountTrieV2(t, rawdb.HashScheme)
	testSyncBoundaryAccountTrieV2(t, rawdb.PathScheme)
}

func testSyncBoundaryAccountTrieV2(t *testing.T, scheme string) {
	var (
		once   sync.Once
		cancel = make(chan struct{})
		term   = func() { once.Do(func() { close(cancel) }) }
	)
	nodeScheme, sourceAccountTrie, elems := makeBoundaryAccountTrie(scheme, 3000)

	mkSource := func(name string) *testPeerV2 {
		source := newTestPeerV2(name, t, term)
		source.accountTrie = sourceAccountTrie.Copy()
		source.accountValues = elems
		return source
	}
	syncer := setupSyncerV2(
		nodeScheme,
		mkSource("peer-a"),
		mkSource("peer-b"),
	)
	done := checkStall(t, term)
	if err := syncer.Sync(sourceAccountTrie.Hash(), cancel); err != nil {
		t.Fatalf("sync failed: %v", err)
	}
	close(done)
	verifyFlatState(t, syncer.db, elems, nil)
}

// TestSyncNoStorageAndOneCappedPeerV2 tests sync using accounts and no storage,
// where one peer is consistently returning very small results.
func TestSyncNoStorageAndOneCappedPeerV2(t *testing.T) {
	t.Parallel()
	testSyncNoStorageAndOneCappedPeerV2(t, rawdb.HashScheme)
	testSyncNoStorageAndOneCappedPeerV2(t, rawdb.PathScheme)
}

func testSyncNoStorageAndOneCappedPeerV2(t *testing.T, scheme string) {
	var (
		once   sync.Once
		cancel = make(chan struct{})
		term   = func() { once.Do(func() { close(cancel) }) }
	)
	nodeScheme, sourceAccountTrie, elems := makeAccountTrieNoStorage(3000, scheme)

	mkSource := func(name string, slow bool) *testPeerV2 {
		source := newTestPeerV2(name, t, term)
		source.accountTrie = sourceAccountTrie.Copy()
		source.accountValues = elems
		if slow {
			source.accountRequestHandler = starvingAccountRequestHandlerV2
		}
		return source
	}

	syncer := setupSyncerV2(
		nodeScheme,
		mkSource("nice-a", false),
		mkSource("nice-b", false),
		mkSource("nice-c", false),
		mkSource("capped", true),
	)
	done := checkStall(t, term)
	if err := syncer.Sync(sourceAccountTrie.Hash(), cancel); err != nil {
		t.Fatalf("sync failed: %v", err)
	}
	close(done)
	verifyFlatState(t, syncer.db, elems, nil)
}

// TestSyncNoStorageAndOneCodeCorruptPeerV2 has one peer that doesn't deliver
// code requests properly.
func TestSyncNoStorageAndOneCodeCorruptPeerV2(t *testing.T) {
	t.Parallel()
	testSyncNoStorageAndOneCodeCorruptPeerV2(t, rawdb.HashScheme)
	testSyncNoStorageAndOneCodeCorruptPeerV2(t, rawdb.PathScheme)
}

func testSyncNoStorageAndOneCodeCorruptPeerV2(t *testing.T, scheme string) {
	var (
		once   sync.Once
		cancel = make(chan struct{})
		term   = func() { once.Do(func() { close(cancel) }) }
	)
	nodeScheme, sourceAccountTrie, elems := makeAccountTrieNoStorage(3000, scheme)

	mkSource := func(name string, codeFn codeHandlerFuncV2) *testPeerV2 {
		source := newTestPeerV2(name, t, term)
		source.accountTrie = sourceAccountTrie.Copy()
		source.accountValues = elems
		source.codeRequestHandler = codeFn
		return source
	}
	syncer := setupSyncerV2(
		nodeScheme,
		mkSource("capped", cappedCodeRequestHandlerV2),
		mkSource("corrupt", corruptCodeRequestHandlerV2),
	)
	done := checkStall(t, term)
	if err := syncer.Sync(sourceAccountTrie.Hash(), cancel); err != nil {
		t.Fatalf("sync failed: %v", err)
	}
	close(done)
	verifyFlatState(t, syncer.db, elems, nil)
}

func TestSyncNoStorageAndOneAccountCorruptPeerV2(t *testing.T) {
	t.Parallel()
	testSyncNoStorageAndOneAccountCorruptPeerV2(t, rawdb.HashScheme)
	testSyncNoStorageAndOneAccountCorruptPeerV2(t, rawdb.PathScheme)
}

func testSyncNoStorageAndOneAccountCorruptPeerV2(t *testing.T, scheme string) {
	var (
		once   sync.Once
		cancel = make(chan struct{})
		term   = func() { once.Do(func() { close(cancel) }) }
	)
	nodeScheme, sourceAccountTrie, elems := makeAccountTrieNoStorage(3000, scheme)

	mkSource := func(name string, accFn accountHandlerFuncV2) *testPeerV2 {
		source := newTestPeerV2(name, t, term)
		source.accountTrie = sourceAccountTrie.Copy()
		source.accountValues = elems
		source.accountRequestHandler = accFn
		return source
	}
	syncer := setupSyncerV2(
		nodeScheme,
		mkSource("capped", starvingAccountRequestHandlerV2),
		mkSource("corrupt", corruptAccountRequestHandlerV2),
	)
	done := checkStall(t, term)
	if err := syncer.Sync(sourceAccountTrie.Hash(), cancel); err != nil {
		t.Fatalf("sync failed: %v", err)
	}
	close(done)
	verifyFlatState(t, syncer.db, elems, nil)
}

// TestSyncNoStorageAndOneCodeCappedPeerV2 has one peer that delivers code
// hashes one by one.
func TestSyncNoStorageAndOneCodeCappedPeerV2(t *testing.T) {
	t.Parallel()
	testSyncNoStorageAndOneCodeCappedPeerV2(t, rawdb.HashScheme)
	testSyncNoStorageAndOneCodeCappedPeerV2(t, rawdb.PathScheme)
}

func testSyncNoStorageAndOneCodeCappedPeerV2(t *testing.T, scheme string) {
	var (
		once   sync.Once
		cancel = make(chan struct{})
		term   = func() { once.Do(func() { close(cancel) }) }
	)
	nodeScheme, sourceAccountTrie, elems := makeAccountTrieNoStorage(3000, scheme)

	mkSource := func(name string, codeFn codeHandlerFuncV2) *testPeerV2 {
		source := newTestPeerV2(name, t, term)
		source.accountTrie = sourceAccountTrie.Copy()
		source.accountValues = elems
		source.codeRequestHandler = codeFn
		return source
	}
	var counter int
	syncer := setupSyncerV2(
		nodeScheme,
		mkSource("capped", func(t *testPeerV2, id uint64, hashes []common.Hash, max int) error {
			counter++
			return cappedCodeRequestHandlerV2(t, id, hashes, max)
		}),
	)
	done := checkStall(t, term)
	if err := syncer.Sync(sourceAccountTrie.Hash(), cancel); err != nil {
		t.Fatalf("sync failed: %v", err)
	}
	close(done)

	if threshold := 100; counter > threshold {
		t.Logf("Error, expected < %d invocations, got %d", threshold, counter)
	}
	verifyFlatState(t, syncer.db, elems, nil)
}

// TestSyncBoundaryStorageTrieV2 tests sync against a few normal peers, but the
// storage trie has a few boundary elements.
func TestSyncBoundaryStorageTrieV2(t *testing.T) {
	t.Parallel()
	testSyncBoundaryStorageTrieV2(t, rawdb.HashScheme)
	testSyncBoundaryStorageTrieV2(t, rawdb.PathScheme)
}

func testSyncBoundaryStorageTrieV2(t *testing.T, scheme string) {
	var (
		once   sync.Once
		cancel = make(chan struct{})
		term   = func() { once.Do(func() { close(cancel) }) }
	)
	sourceAccountTrie, elems, storageTries, storageElems := makeAccountTrieWithStorage(scheme, 10, 1000, false, true, false)

	mkSource := func(name string) *testPeerV2 {
		source := newTestPeerV2(name, t, term)
		source.accountTrie = sourceAccountTrie.Copy()
		source.accountValues = elems
		source.setStorageTries(storageTries)
		source.storageValues = storageElems
		return source
	}
	syncer := setupSyncerV2(
		scheme,
		mkSource("peer-a"),
		mkSource("peer-b"),
	)
	done := checkStall(t, term)
	if err := syncer.Sync(sourceAccountTrie.Hash(), cancel); err != nil {
		t.Fatalf("sync failed: %v", err)
	}
	close(done)
	verifyFlatState(t, syncer.db, elems, storageElems)
}

// TestSyncWithStorageAndOneCappedPeerV2 tests sync using accounts + storage,
// where one peer is consistently returning very small results.
func TestSyncWithStorageAndOneCappedPeerV2(t *testing.T) {
	t.Parallel()
	testSyncWithStorageAndOneCappedPeerV2(t, rawdb.HashScheme)
	testSyncWithStorageAndOneCappedPeerV2(t, rawdb.PathScheme)
}

func testSyncWithStorageAndOneCappedPeerV2(t *testing.T, scheme string) {
	var (
		once   sync.Once
		cancel = make(chan struct{})
		term   = func() { once.Do(func() { close(cancel) }) }
	)
	sourceAccountTrie, elems, storageTries, storageElems := makeAccountTrieWithStorage(scheme, 300, 1000, false, false, false)

	mkSource := func(name string, slow bool) *testPeerV2 {
		source := newTestPeerV2(name, t, term)
		source.accountTrie = sourceAccountTrie.Copy()
		source.accountValues = elems
		source.setStorageTries(storageTries)
		source.storageValues = storageElems
		if slow {
			source.storageRequestHandler = starvingStorageRequestHandlerV2
		}
		return source
	}
	syncer := setupSyncerV2(
		scheme,
		mkSource("nice-a", false),
		mkSource("slow", true),
	)
	done := checkStall(t, term)
	if err := syncer.Sync(sourceAccountTrie.Hash(), cancel); err != nil {
		t.Fatalf("sync failed: %v", err)
	}
	close(done)
	verifyFlatState(t, syncer.db, elems, storageElems)
}

// TestSyncWithStorageAndCorruptPeerV2 tests sync using accounts + storage,
// where one peer is sometimes sending bad proofs.
func TestSyncWithStorageAndCorruptPeerV2(t *testing.T) {
	t.Parallel()
	testSyncWithStorageAndCorruptPeerV2(t, rawdb.HashScheme)
	testSyncWithStorageAndCorruptPeerV2(t, rawdb.PathScheme)
}

func testSyncWithStorageAndCorruptPeerV2(t *testing.T, scheme string) {
	var (
		once   sync.Once
		cancel = make(chan struct{})
		term   = func() { once.Do(func() { close(cancel) }) }
	)
	sourceAccountTrie, elems, storageTries, storageElems := makeAccountTrieWithStorage(scheme, 100, 3000, true, false, false)

	mkSource := func(name string, handler storageHandlerFuncV2) *testPeerV2 {
		source := newTestPeerV2(name, t, term)
		source.accountTrie = sourceAccountTrie.Copy()
		source.accountValues = elems
		source.setStorageTries(storageTries)
		source.storageValues = storageElems
		source.storageRequestHandler = handler
		return source
	}
	syncer := setupSyncerV2(
		scheme,
		mkSource("nice-a", defaultStorageRequestHandlerV2),
		mkSource("nice-b", defaultStorageRequestHandlerV2),
		mkSource("nice-c", defaultStorageRequestHandlerV2),
		mkSource("corrupt", corruptStorageRequestHandlerV2),
	)
	done := checkStall(t, term)
	if err := syncer.Sync(sourceAccountTrie.Hash(), cancel); err != nil {
		t.Fatalf("sync failed: %v", err)
	}
	close(done)
	verifyFlatState(t, syncer.db, elems, storageElems)
}

func TestSyncWithStorageAndNonProvingPeerV2(t *testing.T) {
	t.Parallel()
	testSyncWithStorageAndNonProvingPeerV2(t, rawdb.HashScheme)
	testSyncWithStorageAndNonProvingPeerV2(t, rawdb.PathScheme)
}

func testSyncWithStorageAndNonProvingPeerV2(t *testing.T, scheme string) {
	var (
		once   sync.Once
		cancel = make(chan struct{})
		term   = func() { once.Do(func() { close(cancel) }) }
	)
	sourceAccountTrie, elems, storageTries, storageElems := makeAccountTrieWithStorage(scheme, 100, 3000, true, false, false)

	mkSource := func(name string, handler storageHandlerFuncV2) *testPeerV2 {
		source := newTestPeerV2(name, t, term)
		source.accountTrie = sourceAccountTrie.Copy()
		source.accountValues = elems
		source.setStorageTries(storageTries)
		source.storageValues = storageElems
		source.storageRequestHandler = handler
		return source
	}
	syncer := setupSyncerV2(
		scheme,
		mkSource("nice-a", defaultStorageRequestHandlerV2),
		mkSource("nice-b", defaultStorageRequestHandlerV2),
		mkSource("nice-c", defaultStorageRequestHandlerV2),
		mkSource("corrupt", noProofStorageRequestHandlerV2),
	)
	done := checkStall(t, term)
	if err := syncer.Sync(sourceAccountTrie.Hash(), cancel); err != nil {
		t.Fatalf("sync failed: %v", err)
	}
	close(done)
	verifyFlatState(t, syncer.db, elems, storageElems)
}

// TestSyncWithStorageMisbehavingProveV2 tests basic sync using accounts +
// storage + code against a peer that insists on delivering full storage sets
// _and_ proofs.
func TestSyncWithStorageMisbehavingProveV2(t *testing.T) {
	t.Parallel()
	testSyncWithStorageMisbehavingProveV2(t, rawdb.HashScheme)
	testSyncWithStorageMisbehavingProveV2(t, rawdb.PathScheme)
}

func testSyncWithStorageMisbehavingProveV2(t *testing.T, scheme string) {
	var (
		once   sync.Once
		cancel = make(chan struct{})
		term   = func() { once.Do(func() { close(cancel) }) }
	)
	nodeScheme, sourceAccountTrie, elems, storageTries, storageElems := makeAccountTrieWithStorageWithUniqueStorage(scheme, 10, 30, false)

	mkSource := func(name string) *testPeerV2 {
		source := newTestPeerV2(name, t, term)
		source.accountTrie = sourceAccountTrie.Copy()
		source.accountValues = elems
		source.setStorageTries(storageTries)
		source.storageValues = storageElems
		source.storageRequestHandler = proofHappyStorageRequestHandlerV2
		return source
	}
	syncer := setupSyncerV2(nodeScheme, mkSource("sourceA"))
	if err := syncer.Sync(sourceAccountTrie.Hash(), cancel); err != nil {
		t.Fatalf("sync failed: %v", err)
	}
	verifyFlatState(t, syncer.db, elems, storageElems)
}

// TestSyncWithUnevenStorageV2 tests sync where the storage trie is not even
// and with a few empty ranges.
func TestSyncWithUnevenStorageV2(t *testing.T) {
	t.Parallel()
	testSyncWithUnevenStorageV2(t, rawdb.HashScheme)
	testSyncWithUnevenStorageV2(t, rawdb.PathScheme)
}

func testSyncWithUnevenStorageV2(t *testing.T, scheme string) {
	var (
		once   sync.Once
		cancel = make(chan struct{})
		term   = func() { once.Do(func() { close(cancel) }) }
	)
	accountTrie, accounts, storageTries, storageElems := makeAccountTrieWithStorage(scheme, 3, 256, false, false, true)

	mkSource := func(name string) *testPeerV2 {
		source := newTestPeerV2(name, t, term)
		source.accountTrie = accountTrie.Copy()
		source.accountValues = accounts
		source.setStorageTries(storageTries)
		source.storageValues = storageElems
		source.storageRequestHandler = func(t *testPeerV2, reqId uint64, root common.Hash, accounts []common.Hash, origin, limit []byte, max int) error {
			return defaultStorageRequestHandlerV2(t, reqId, root, accounts, origin, limit, 128)
		}
		return source
	}
	syncer := setupSyncerV2(scheme, mkSource("source"))
	if err := syncer.Sync(accountTrie.Hash(), cancel); err != nil {
		t.Fatalf("sync failed: %v", err)
	}
	verifyFlatState(t, syncer.db, accounts, storageElems)
}

// TestSyncBloatedProofV2 tests a scenario where the peer ships only one value
// but inflates the proof with the entire trie. If the attack is successful the
// remote side does not do any follow-up requests.
func TestSyncBloatedProofV2(t *testing.T) {
	t.Parallel()
	testSyncBloatedProofV2(t, rawdb.HashScheme)
	testSyncBloatedProofV2(t, rawdb.PathScheme)
}

func testSyncBloatedProofV2(t *testing.T, scheme string) {
	var (
		once   sync.Once
		cancel = make(chan struct{})
		term   = func() { once.Do(func() { close(cancel) }) }
	)
	nodeScheme, sourceAccountTrie, elems := makeAccountTrieNoStorage(100, scheme)
	source := newTestPeerV2("source", t, term)
	source.accountTrie = sourceAccountTrie.Copy()
	source.accountValues = elems

	source.accountRequestHandler = func(t *testPeerV2, requestId uint64, root common.Hash, origin common.Hash, limit common.Hash, cap int) error {
		var (
			keys []common.Hash
			vals [][]byte
		)
		for _, entry := range t.accountValues {
			if bytes.Compare(entry.k, origin[:]) < 0 {
				continue
			}
			if bytes.Compare(entry.k, limit[:]) > 0 {
				continue
			}
			keys = append(keys, common.BytesToHash(entry.k))
			vals = append(vals, entry.v)
		}
		proof := trienode.NewProofSet()
		if err := t.accountTrie.Prove(origin[:], proof); err != nil {
			t.logger.Error("Could not prove origin", "origin", origin, "error", err)
		}
		for _, entry := range t.accountValues {
			if err := t.accountTrie.Prove(entry.k, proof); err != nil {
				t.logger.Error("Could not prove item", "error", err)
			}
		}
		if len(keys) > 2 {
			keys = append(keys[:1], keys[2:]...)
			vals = append(vals[:1], vals[2:]...)
		}
		if err := t.remote.OnAccounts(t, requestId, keys, vals, proof.List()); err != nil {
			t.logger.Info("remote error on delivery (as expected)", "error", err)
			t.term()
		}
		return nil
	}
	syncer := setupSyncerV2(nodeScheme, source)
	if err := syncer.Sync(sourceAccountTrie.Hash(), cancel); err == nil {
		t.Fatal("No error returned from incomplete/cancelled sync")
	}
}
