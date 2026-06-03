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
	"errors"
	"fmt"
	"math/big"
	"slices"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/types/bal"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/ethereum/go-ethereum/trie"
	"github.com/ethereum/go-ethereum/trie/trienode"
	"github.com/ethereum/go-ethereum/triedb"
	"github.com/holiman/uint256"
)

type (
	accountHandlerFuncV2  func(t *testPeerV2, requestId uint64, root common.Hash, origin common.Hash, limit common.Hash, cap int) error
	storageHandlerFuncV2  func(t *testPeerV2, requestId uint64, root common.Hash, accounts []common.Hash, origin, limit []byte, max int) error
	codeHandlerFuncV2     func(t *testPeerV2, id uint64, hashes []common.Hash, max int) error
	accessListHandlerFunc func(t *testPeerV2, id uint64, hashes []common.Hash, max int) error
)

type testPeerV2 struct {
	id            string
	test          *testing.T
	remote        *syncerV2
	logger        log.Logger
	accountTrie   *trie.Trie
	accountValues []*kv
	storageTries  map[common.Hash]*trie.Trie
	storageValues map[common.Hash][]*kv
	accessLists   map[common.Hash]rlp.RawValue // block hash -> RLP-encoded BAL

	accountRequestV2Handler  accountHandlerFuncV2
	storageRequestV2Handler  storageHandlerFuncV2
	codeRequestHandler       codeHandlerFuncV2
	accessListRequestHandler accessListHandlerFunc
	term                     func()

	// counters
	nAccountRequests    atomic.Int64
	nStorageRequests    atomic.Int64
	nBytecodeRequests   atomic.Int64
	nAccessListRequests atomic.Int64
}

func newTestPeerV2(id string, t *testing.T, term func()) *testPeerV2 {
	peer := &testPeerV2{
		id:                       id,
		test:                     t,
		logger:                   log.New("id", id),
		accountRequestV2Handler:  defaultAccountRequestHandlerV2,
		storageRequestV2Handler:  defaultStorageRequestHandlerV2,
		codeRequestHandler:       defaultCodeRequestHandlerV2,
		accessListRequestHandler: defaultAccessListRequestHandler,
		term:                     term,
	}
	return peer
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
	return fmt.Sprintf(`Account requests: %d Storage requests: %d Bytecode requests: %d`, t.nAccountRequests.Load(), t.nStorageRequests.Load(), t.nBytecodeRequests.Load())
}

func (t *testPeerV2) RequestAccountRange(id uint64, root, origin, limit common.Hash, bytes int) error {
	t.logger.Trace("Fetching range of accounts", "reqid", id, "root", root, "origin", origin, "limit", limit, "bytes", common.StorageSize(bytes))
	t.nAccountRequests.Add(1)
	go t.accountRequestV2Handler(t, id, root, origin, limit, bytes)
	return nil
}

func (t *testPeerV2) RequestStorageRanges(id uint64, root common.Hash, accounts []common.Hash, origin, limit []byte, bytes int) error {
	t.nStorageRequests.Add(1)
	if len(accounts) == 1 && origin != nil {
		t.logger.Trace("Fetching range of large storage slots", "reqid", id, "root", root, "account", accounts[0], "origin", common.BytesToHash(origin), "limit", common.BytesToHash(limit), "bytes", common.StorageSize(bytes))
	} else {
		t.logger.Trace("Fetching ranges of small storage slots", "reqid", id, "root", root, "accounts", len(accounts), "first", accounts[0], "bytes", common.StorageSize(bytes))
	}
	go t.storageRequestV2Handler(t, id, root, accounts, origin, limit, bytes)
	return nil
}

func (t *testPeerV2) RequestByteCodes(id uint64, hashes []common.Hash, bytes int) error {
	t.nBytecodeRequests.Add(1)
	t.logger.Trace("Fetching set of byte codes", "reqid", id, "hashes", len(hashes), "bytes", common.StorageSize(bytes))
	go t.codeRequestHandler(t, id, hashes, bytes)
	return nil
}

func (t *testPeerV2) RequestTrieNodes(id uint64, root common.Hash, count int, paths []TrieNodePathSet, bytes int) error {
	// snap/2 never requests trie nodes.
	return nil
}

func (t *testPeerV2) RequestAccessLists(id uint64, hashes []common.Hash, bytes int) error {
	t.nAccessListRequests.Add(1)
	t.logger.Trace("Fetching set of BALs", "reqid", id, "hashes", len(hashes), "bytes", common.StorageSize(bytes))
	go t.accessListRequestHandler(t, id, hashes, bytes)
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

// defaultAccessListRequestHandler serves BALs from the peer's accessLists map.
// If the peer has no BAL data, it returns empty (peer rejection).
func defaultAccessListRequestHandler(t *testPeerV2, id uint64, hashes []common.Hash, max int) error {
	var results []rlp.RawValue
	if t.accessLists != nil {
		for _, h := range hashes {
			if raw, ok := t.accessLists[h]; ok {
				results = append(results, raw)
			}
		}
	}
	rawList, _ := rlp.EncodeToRawList(results)
	if err := t.remote.OnAccessLists(t, id, rawList); err != nil {
		t.test.Errorf("Remote side rejected our delivery: %v", err)
		t.term()
	}
	return nil
}

// emptyRequestAccountRangeFnV2 is a rejects AccountRangeRequests
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

// TestSyncBloatedProofV2 tests a scenario where we provide only _one_ value, but
// also ship the entire trie inside the proof. If the attack is successful,
// the remote side does not do any follow-up requests
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

	source.accountRequestV2Handler = func(t *testPeerV2, requestId uint64, root common.Hash, origin common.Hash, limit common.Hash, cap int) error {
		var (
			keys []common.Hash
			vals [][]byte
		)
		// The values
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
		// The proofs
		proof := trienode.NewProofSet()
		if err := t.accountTrie.Prove(origin[:], proof); err != nil {
			t.logger.Error("Could not prove origin", "origin", origin, "error", err)
		}
		// The bloat: add proof of every single element
		for _, entry := range t.accountValues {
			if err := t.accountTrie.Prove(entry.k, proof); err != nil {
				t.logger.Error("Could not prove item", "error", err)
			}
		}
		// And remove one item from the elements
		if len(keys) > 2 {
			keys = append(keys[:1], keys[2:]...)
			vals = append(vals[:1], vals[2:]...)
		}
		if err := t.remote.OnAccounts(t, requestId, keys, vals, proof.List()); err != nil {
			t.logger.Info("remote error on delivery (as expected)", "error", err)
			t.term()
			// This is actually correct, signal to exit the test successfully
		}
		return nil
	}
	syncer := setupSyncerV2(nodeScheme, source)
	if err := syncer.Sync(mkPivot(0, sourceAccountTrie.Hash()), cancel); err == nil {
		t.Fatal("No error returned from incomplete/cancelled sync")
	}
}

func setupSyncerV2(scheme string, peers ...*testPeerV2) *syncerV2 {
	stateDb := rawdb.NewMemoryDatabase()
	syncer := newSyncerV2(stateDb, scheme)
	for _, peer := range peers {
		syncer.Register(peer)
		peer.remote = syncer
	}
	return syncer
}

// mkPivot builds a minimal pivot header with the given block number and state
// root, suitable for test calls into syncerV2.Sync.
func mkPivot(num uint64, root common.Hash) *types.Header {
	return &types.Header{
		Number:     new(big.Int).SetUint64(num),
		Root:       root,
		Difficulty: common.Big0,
	}
}

// makeAccessListHeaders builds a header map keyed by block hash where each
// header's BlockAccessListHash matches the BAL it points to. fetchAccessLists
// uses these headers to verify peer responses, so tests need to provide them
// alongside any BALs they expect to be accepted.
func makeAccessListHeaders(bals map[common.Hash]rlp.RawValue) map[common.Hash]*types.Header {
	headers := make(map[common.Hash]*types.Header, len(bals))
	for h, raw := range bals {
		var b bal.BlockAccessList
		if err := rlp.DecodeBytes(raw, &b); err != nil {
			continue
		}
		bh := b.Hash()
		headers[h] = &types.Header{BlockAccessListHash: &bh}
	}
	return headers
}

// TestSyncV2 tests a basic sync with one peer
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
	if err := syncer.Sync(mkPivot(0, sourceAccountTrie.Hash()), cancel); err != nil {
		t.Fatalf("sync failed: %v", err)
	}
	verifyTrie(scheme, syncer.db, sourceAccountTrie.Hash(), t)
	verifyAdoptedSyncedState(scheme, syncer.db, sourceAccountTrie.Hash(), elems, t)
}

// verifyAdoptedSyncedState exercises the snap/2 completion contract end-to-end:
// after a real sync, opening a fresh triedb and calling AdoptSyncedState must
// (a) succeed and (b) leave flat-state reads serving immediately, with no
// background regeneration gating them.
func verifyAdoptedSyncedState(scheme string, db ethdb.KeyValueStore, root common.Hash, elems []*kv, t *testing.T) {
	t.Helper()
	if scheme != rawdb.PathScheme {
		return
	}
	tdb := triedb.NewDatabase(rawdb.NewDatabase(db), newDbConfig(scheme))
	defer tdb.Close()

	if err := tdb.AdoptSyncedState(root); err != nil {
		t.Fatalf("AdoptSyncedState failed: %v", err)
	}
	// Read one of the synced accounts via the public flat-state API. If this
	// returned errNotCoveredYet we'd know AdoptSyncedState left a generator
	// gating reads, exactly the bug we're trying to prevent.
	sr, err := tdb.StateReader(root)
	if err != nil {
		t.Fatalf("StateReader: %v", err)
	}
	if len(elems) == 0 {
		return
	}
	acc, err := sr.Account(common.BytesToHash(elems[0].k))
	if err != nil {
		t.Fatalf("flat-state read failed after AdoptSyncedState: %v", err)
	}
	if acc == nil {
		t.Fatal("flat-state read returned nil account; sync did not populate the snapshot namespace")
	}
}

// TestSyncTinyTriePanicV2 tests a basic sync with one peer, and a tiny trie. This caused a
// panic within the prover
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
	if err := syncer.Sync(mkPivot(0, sourceAccountTrie.Hash()), cancel); err != nil {
		t.Fatalf("sync failed: %v", err)
	}
	close(done)
	verifyTrie(scheme, syncer.db, sourceAccountTrie.Hash(), t)
}

// TestMultiSyncV2 tests a basic sync with multiple peers
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
	if err := syncer.Sync(mkPivot(0, sourceAccountTrie.Hash()), cancel); err != nil {
		t.Fatalf("sync failed: %v", err)
	}
	close(done)
	verifyTrie(scheme, syncer.db, sourceAccountTrie.Hash(), t)
}

// TestSyncWithStorageV2 tests  basic sync using accounts + storage + code
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
	if err := syncer.Sync(mkPivot(0, sourceAccountTrie.Hash()), cancel); err != nil {
		t.Fatalf("sync failed: %v", err)
	}
	close(done)
	verifyTrie(scheme, syncer.db, sourceAccountTrie.Hash(), t)
}

// TestMultiSyncManyUselessV2 contains one good peer, and many which doesn't return anything valuable at all
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
		if !noAccount {
			source.accountRequestV2Handler = emptyRequestAccountRangeFnV2
		}
		if !noStorage {
			source.storageRequestV2Handler = emptyStorageRequestHandlerV2
		}
		return source
	}
	syncer := setupSyncerV2(
		scheme,
		mkSource("full", true, true),
		mkSource("noAccounts", false, true),
		mkSource("noStorage", true, false),
	)
	done := checkStall(t, term)
	if err := syncer.Sync(mkPivot(0, sourceAccountTrie.Hash()), cancel); err != nil {
		t.Fatalf("sync failed: %v", err)
	}
	close(done)
	verifyTrie(scheme, syncer.db, sourceAccountTrie.Hash(), t)
}

// TestMultiSyncManyUselessWithLowTimeoutV2 contains one good peer, and many which doesn't return anything valuable at all
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
			source.accountRequestV2Handler = emptyRequestAccountRangeFnV2
		}
		if !noStorage {
			source.storageRequestV2Handler = emptyStorageRequestHandlerV2
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
	if err := syncer.Sync(mkPivot(0, sourceAccountTrie.Hash()), cancel); err != nil {
		t.Fatalf("sync failed: %v", err)
	}
	close(done)
	verifyTrie(scheme, syncer.db, sourceAccountTrie.Hash(), t)
}

// TestMultiSyncManyUnresponsiveV2 contains one good peer, and many which doesn't respond at all
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
		if !noAccount {
			source.accountRequestV2Handler = nonResponsiveRequestAccountRangeFnV2
		}
		if !noStorage {
			source.storageRequestV2Handler = nonResponsiveStorageRequestHandlerV2
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
	if err := syncer.Sync(mkPivot(0, sourceAccountTrie.Hash()), cancel); err != nil {
		t.Fatalf("sync failed: %v", err)
	}
	close(done)
	verifyTrie(scheme, syncer.db, sourceAccountTrie.Hash(), t)
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
	if err := syncer.Sync(mkPivot(0, sourceAccountTrie.Hash()), cancel); err != nil {
		t.Fatalf("sync failed: %v", err)
	}
	close(done)
	verifyTrie(scheme, syncer.db, sourceAccountTrie.Hash(), t)
}

// TestSyncNoStorageAndOneCappedPeerV2 tests sync using accounts and no storage, where one peer is
// consistently returning very small results
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
			source.accountRequestV2Handler = starvingAccountRequestHandlerV2
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
	if err := syncer.Sync(mkPivot(0, sourceAccountTrie.Hash()), cancel); err != nil {
		t.Fatalf("sync failed: %v", err)
	}
	close(done)
	verifyTrie(scheme, syncer.db, sourceAccountTrie.Hash(), t)
}

// TestSyncNoStorageAndOneCodeCorruptPeerV2 has one peer which doesn't deliver
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
	if err := syncer.Sync(mkPivot(0, sourceAccountTrie.Hash()), cancel); err != nil {
		t.Fatalf("sync failed: %v", err)
	}
	close(done)
	verifyTrie(scheme, syncer.db, sourceAccountTrie.Hash(), t)
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
		source.accountRequestV2Handler = accFn
		return source
	}
	syncer := setupSyncerV2(
		nodeScheme,
		mkSource("capped", defaultAccountRequestHandlerV2),
		mkSource("corrupt", corruptAccountRequestHandlerV2),
	)
	done := checkStall(t, term)
	if err := syncer.Sync(mkPivot(0, sourceAccountTrie.Hash()), cancel); err != nil {
		t.Fatalf("sync failed: %v", err)
	}
	close(done)
	verifyTrie(scheme, syncer.db, sourceAccountTrie.Hash(), t)
}

// TestSyncNoStorageAndOneCodeCappedPeerV2 has one peer which delivers code hashes
// one by one
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
	if err := syncer.Sync(mkPivot(0, sourceAccountTrie.Hash()), cancel); err != nil {
		t.Fatalf("sync failed: %v", err)
	}
	close(done)

	if threshold := 100; counter > threshold {
		t.Logf("Error, expected < %d invocations, got %d", threshold, counter)
	}
	verifyTrie(scheme, syncer.db, sourceAccountTrie.Hash(), t)
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
	if err := syncer.Sync(mkPivot(0, sourceAccountTrie.Hash()), cancel); err != nil {
		t.Fatalf("sync failed: %v", err)
	}
	close(done)
	verifyTrie(scheme, syncer.db, sourceAccountTrie.Hash(), t)
}

// TestSyncWithStorageAndOneCappedPeerV2 tests sync using accounts + storage, where one peer is
// consistently returning very small results
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
			source.storageRequestV2Handler = starvingStorageRequestHandlerV2
		}
		return source
	}
	syncer := setupSyncerV2(
		scheme,
		mkSource("nice-a", false),
		mkSource("slow", true),
	)
	done := checkStall(t, term)
	if err := syncer.Sync(mkPivot(0, sourceAccountTrie.Hash()), cancel); err != nil {
		t.Fatalf("sync failed: %v", err)
	}
	close(done)
	verifyTrie(scheme, syncer.db, sourceAccountTrie.Hash(), t)
}

// TestSyncWithStorageAndCorruptPeerV2 tests sync using accounts + storage, where one peer is
// sometimes sending bad proofs
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
		source.storageRequestV2Handler = handler
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
	if err := syncer.Sync(mkPivot(0, sourceAccountTrie.Hash()), cancel); err != nil {
		t.Fatalf("sync failed: %v", err)
	}
	close(done)
	verifyTrie(scheme, syncer.db, sourceAccountTrie.Hash(), t)
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
		source.storageRequestV2Handler = handler
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
	if err := syncer.Sync(mkPivot(0, sourceAccountTrie.Hash()), cancel); err != nil {
		t.Fatalf("sync failed: %v", err)
	}
	close(done)
	verifyTrie(scheme, syncer.db, sourceAccountTrie.Hash(), t)
}

// TestSyncWithStorageMisbehavingProveV2 tests  basic sync using accounts + storage + code, against
// a peer who insists on delivering full storage sets _and_ proofs. This triggered
// an error, where the recipient erroneously clipped the boundary nodes, but
// did not mark the account for healing.
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
		source.storageRequestV2Handler = proofHappyStorageRequestHandlerV2
		return source
	}
	syncer := setupSyncerV2(nodeScheme, mkSource("sourceA"))
	if err := syncer.Sync(mkPivot(0, sourceAccountTrie.Hash()), cancel); err != nil {
		t.Fatalf("sync failed: %v", err)
	}
	verifyTrie(scheme, syncer.db, sourceAccountTrie.Hash(), t)
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
		source.storageRequestV2Handler = func(t *testPeerV2, reqId uint64, root common.Hash, accounts []common.Hash, origin, limit []byte, max int) error {
			return defaultStorageRequestHandlerV2(t, reqId, root, accounts, origin, limit, 128) // retrieve storage in large mode
		}
		return source
	}
	syncer := setupSyncerV2(scheme, mkSource("source"))
	if err := syncer.Sync(mkPivot(0, accountTrie.Hash()), cancel); err != nil {
		t.Fatalf("sync failed: %v", err)
	}
	verifyTrie(scheme, syncer.db, accountTrie.Hash(), t)
}

// makeAccountTrieWithAddresses creates an account trie keyed by keccak(address),
// matching production behavior. Returns the trie, sorted entries, and the
// addresses used. This allows BAL-based tests to target specific addresses and
// have applyAccessList write to the same snapshot keys as the download.
func makeAccountTrieWithAddresses(n int, scheme string) (string, *trie.Trie, []*kv, []common.Address) {
	var (
		db      = triedb.NewDatabase(rawdb.NewMemoryDatabase(), newDbConfig(scheme))
		accTrie = trie.NewEmpty(db)
		entries []*kv
		addrs   []common.Address
	)
	for i := uint64(1); i <= uint64(n); i++ {
		// Deterministic address from index
		addr := common.BigToAddress(new(big.Int).SetUint64(i))
		addrs = append(addrs, addr)

		value, _ := rlp.EncodeToBytes(&types.StateAccount{
			Nonce:    i,
			Balance:  uint256.NewInt(i),
			Root:     types.EmptyRootHash,
			CodeHash: types.EmptyCodeHash[:],
		})
		key := crypto.Keccak256(addr[:])
		elem := &kv{key, value}
		accTrie.MustUpdate(elem.k, elem.v)
		entries = append(entries, elem)
	}
	slices.SortFunc(entries, (*kv).cmp)

	root, nodes := accTrie.Commit(false)
	db.Update(root, types.EmptyRootHash, 0, trienode.NewWithNodeSet(nodes), triedb.NewStateSet())

	accTrie, _ = trie.New(trie.StateTrieID(root), db)
	return db.Scheme(), accTrie, entries, addrs
}

// TestIsPivotReorged verifies the four conditions isPivotReorged covers:
// reorged out, non-advancing pivot, missing canonical, and the happy path
// where the previous pivot is still canonical and the new pivot advances.
func TestIsPivotReorged(t *testing.T) {
	t.Parallel()

	// Reorged: canonical hash at prev's height differs from prev. The
	// previous pivot was reorged out by an alternate chain at the same
	// (or higher) height.
	t.Run("Reorged_DifferentHash", func(t *testing.T) {
		db := rawdb.NewMemoryDatabase()
		prev := mkPivot(100, common.HexToHash("0xaaaa"))
		curr := mkPivot(105, common.HexToHash("0xcccc"))
		canonical := mkPivot(100, common.HexToHash("0xbbbb"))
		rawdb.WriteHeader(db, canonical)
		rawdb.WriteCanonicalHash(db, canonical.Hash(), canonical.Number.Uint64())

		if !isPivotReorged(db, prev, curr) {
			t.Fatal("expected reorg detection when canonical hash differs")
		}
	})

	// NonAdvancingPivot: new pivot is at or below the old one. There's
	// nothing for catchUp to roll forward, regardless of canonical state.
	t.Run("NonAdvancingPivot", func(t *testing.T) {
		db := rawdb.NewMemoryDatabase()
		prev := mkPivot(100, common.HexToHash("0xaaaa"))
		curr := mkPivot(95, common.HexToHash("0xcccc"))
		rawdb.WriteHeader(db, prev)
		rawdb.WriteCanonicalHash(db, prev.Hash(), prev.Number.Uint64())

		if !isPivotReorged(db, prev, curr) {
			t.Fatal("expected reorg detection when new pivot is at or below the old one")
		}
	})

	// MissingCanonical: canonical hash at prev's height is absent while
	// curr advances past it. By the time Sync is called, headers up to
	// curr should be indexed, so this implies broken chain state.
	t.Run("MissingCanonical", func(t *testing.T) {
		db := rawdb.NewMemoryDatabase()
		prev := mkPivot(100, common.HexToHash("0xaaaa"))
		curr := mkPivot(105, common.HexToHash("0xcccc"))

		if !isPivotReorged(db, prev, curr) {
			t.Fatal("expected reorg detection when canonical hash is missing at prev's height")
		}
	})

	// NotReorged_SameHash: prev is still canonical and curr advances past
	// it. Catch-up is feasible.
	t.Run("NotReorged_SameHash", func(t *testing.T) {
		db := rawdb.NewMemoryDatabase()
		prev := mkPivot(100, common.HexToHash("0xaaaa"))
		curr := mkPivot(105, common.HexToHash("0xcccc"))
		rawdb.WriteHeader(db, prev)
		rawdb.WriteCanonicalHash(db, prev.Hash(), prev.Number.Uint64())

		if isPivotReorged(db, prev, curr) {
			t.Fatal("should not detect reorg when prev is canonical and curr advances")
		}
	})
}

// TestSyncDetectsPivotReorged exercises the reorg-handling branch in Sync
// end-to-end.
//
// Setup: persisted progress points at an orphan pivot at block 100; the new
// canonical header at block 100 has a different hash. Sync is then called with
// a new pivot at the same height.
//
// If isPivotReorged works, loadSyncStatus restores previousPivot, the check
// flags it as reorged, resetSyncState clears previousPivot, catchUp is
// skipped, and the fresh download proceeds to completion.
//
// If detection doesn't fire, the pivot-move check would call catchUp with
// from = 101 and to = 100 — the inverted-range guard surfaces that as an
// error, failing the test. So Sync returning nil is the positive signal that
// reorg detection and the reset worked.
func TestSyncDetectsPivotReorged(t *testing.T) {
	t.Parallel()

	nodeScheme, sourceAccountTrie, elems := makeAccountTrieNoStorage(100, rawdb.HashScheme)
	root := sourceAccountTrie.Hash()

	db := rawdb.NewMemoryDatabase()

	// Persist progress against an orphan pivot — same height as the new
	// canonical pivot we'll sync to, different hash. Populate a partial task
	// and non-zero counter so the reset path has something to clean up.
	orphanPivot := mkPivot(100, common.HexToHash("0xdead"))
	seed := newSyncerV2(db, nodeScheme)
	// previousPivot reflects where flat state matches and it is what
	// saveSyncStatus persists. Set it to simulate a prior sync reaching
	// orphanPivot.
	seed.previousPivot = orphanPivot
	seed.pivot = orphanPivot
	seed.accountSynced = 42
	seed.tasks = []*accountTaskV2{{
		Next:           common.HexToHash("0x80"),
		Last:           common.MaxHash,
		SubTasks:       make(map[common.Hash][]*storageTaskV2),
		stateCompleted: make(map[common.Hash]struct{}),
	}}
	seed.saveSyncStatus()

	// Pre-write orphan flat-state entries at hashes the test peer won't
	// re-serve. After resetSyncState wipes the snapshot ranges, these
	// should be gone.
	orphanAccountHash := common.HexToHash("0xdeadbeef")
	rawdb.WriteAccountSnapshot(db, orphanAccountHash, []byte{0xde, 0xad})
	orphanStorageAccount := common.HexToHash("0xfeedfacefeedfacefeedfacefeedfacefeedfacefeedfacefeedfacefeedface")
	orphanStorageSlot := common.HexToHash("0xabcd")
	rawdb.WriteStorageSnapshot(db, orphanStorageAccount, orphanStorageSlot, []byte{0xff, 0xff})

	// Canonical header at block 100 is newPivot — different hash from the
	// orphan pivot, which is what isPivotReorged will detect.
	newPivot := mkPivot(100, root)
	rawdb.WriteHeader(db, newPivot)
	rawdb.WriteCanonicalHash(db, newPivot.Hash(), newPivot.Number.Uint64())

	var (
		once   sync.Once
		cancel = make(chan struct{})
		term   = func() { once.Do(func() { close(cancel) }) }
	)
	syncer := newSyncerV2(db, nodeScheme)
	src := newTestPeerV2("source", t, term)
	src.accountTrie = sourceAccountTrie.Copy()
	src.accountValues = elems
	syncer.Register(src)
	src.remote = syncer

	if err := syncer.Sync(newPivot, cancel); err != nil {
		t.Fatalf("sync failed (reorg detection likely broken): %v", err)
	}
	// After successful completion, status should be marked Complete=true
	// against the new (canonical) pivot.
	loader := newSyncerV2(db, nodeScheme)
	loader.loadSyncStatus()
	if !loader.complete {
		t.Fatal("sync status should be marked Complete=true after successful completion")
	}
	if loader.previousPivot == nil || loader.previousPivot.Hash() != newPivot.Hash() {
		t.Fatalf("expected persisted pivot to match new pivot")
	}
	if data := rawdb.ReadAccountSnapshot(db, orphanAccountHash); len(data) != 0 {
		t.Errorf("orphan account snapshot should be wiped, got %x", data)
	}
	if val := rawdb.ReadStorageSnapshot(db, orphanStorageAccount, orphanStorageSlot); len(val) != 0 {
		t.Errorf("orphan storage snapshot should be wiped, got %x", val)
	}
}

// TestInterruptedDownloadRecovery verifies that partially completed download
// state is persisted and resumed on restart.
func TestInterruptedDownloadRecovery(t *testing.T) {
	t.Parallel()
	testInterruptedDownloadRecovery(t, rawdb.HashScheme)
	testInterruptedDownloadRecovery(t, rawdb.PathScheme)
}

func testInterruptedDownloadRecovery(t *testing.T, scheme string) {
	nodeScheme, sourceAccountTrie, elems := makeAccountTrieNoStorage(100, scheme)
	root := sourceAccountTrie.Hash()

	// Cancel after exactly 2 account range responses, guaranteeing partial
	// completion without any timing dependency.
	var (
		once1     sync.Once
		cancel1   = make(chan struct{})
		term1     = func() { once1.Do(func() { close(cancel1) }) }
		responses atomic.Int32
	)
	cancelAfterHandler := func(tp *testPeerV2, id uint64, root common.Hash, origin common.Hash, limit common.Hash, cap int) error {
		if responses.Add(1) > 2 {
			term1()
			return nil
		}
		return defaultAccountRequestHandlerV2(tp, id, root, origin, limit, cap)
	}
	db := rawdb.NewMemoryDatabase()
	syncer1 := newSyncerV2(db, nodeScheme)
	src1 := newTestPeerV2("source1", t, term1)
	src1.accountTrie = sourceAccountTrie.Copy()
	src1.accountValues = elems
	src1.accountRequestV2Handler = cancelAfterHandler
	syncer1.Register(src1)
	src1.remote = syncer1
	pivot := mkPivot(0, root)
	syncer1.pivot = pivot
	syncer1.previousPivot = pivot // Sync sets this before downloadState
	syncer1.loadSyncStatus()
	syncer1.downloadState(cancel1)

	// Save progress
	for _, task := range syncer1.tasks {
		syncer1.forwardAccountTask(task)
	}
	syncer1.cleanAccountTasks()
	syncer1.saveSyncStatus()

	// Count how many accounts were downloaded in the first run.
	// Due to the async nature of response processing, the cancel may race
	// with delivery so 0 accounts may be written.
	firstRunCount := 0
	for _, entry := range elems {
		if data := rawdb.ReadAccountSnapshot(db, common.BytesToHash(entry.k)); len(data) > 0 {
			firstRunCount++
		}
	}
	if firstRunCount == len(elems) {
		t.Fatal("first run should not have downloaded everything")
	}

	// Second run: resume with same root, should complete the download
	var (
		once2   sync.Once
		cancel2 = make(chan struct{})
		term2   = func() { once2.Do(func() { close(cancel2) }) }
	)
	syncer2 := newSyncerV2(db, nodeScheme)
	src2 := newTestPeerV2("source2", t, term2)
	src2.accountTrie = sourceAccountTrie.Copy()
	src2.accountValues = elems
	syncer2.Register(src2)
	src2.remote = syncer2
	pivot2 := mkPivot(0, root)
	syncer2.pivot = pivot2
	syncer2.previousPivot = pivot2 // Sync sets this before downloadState
	syncer2.loadSyncStatus()
	if err := syncer2.downloadState(cancel2); err != nil {
		t.Fatalf("resumed download failed: %v", err)
	}

	// Verify all accounts are now present
	for _, entry := range elems {
		if data := rawdb.ReadAccountSnapshot(db, common.BytesToHash(entry.k)); len(data) == 0 {
			t.Errorf("missing account after resumed download: %x", entry.k)
		}
	}
}

// TestSyncPersistsPivotDuringDownload verifies that after a fresh Sync is
// interrupted mid-download, the persisted previousPivot equals the current
// pivot (not nil). Without this, a follow-up Sync at a different pivot
// would not see that the partial flat state belongs to the old pivot, and
// would mix old-pivot accounts with new-pivot data.
func TestSyncPersistsPivotDuringDownload(t *testing.T) {
	t.Parallel()
	nodeScheme, sourceAccountTrie, elems := makeAccountTrieNoStorage(100, rawdb.HashScheme)

	var (
		once      sync.Once
		cancel    = make(chan struct{})
		term      = func() { once.Do(func() { close(cancel) }) }
		responses atomic.Int32
	)
	db := rawdb.NewMemoryDatabase()
	syncer := newSyncerV2(db, nodeScheme)
	src := newTestPeerV2("source", t, term)
	src.accountTrie = sourceAccountTrie.Copy()
	src.accountValues = elems
	src.accountRequestV2Handler = func(tp *testPeerV2, id uint64, root common.Hash, origin common.Hash, limit common.Hash, cap int) error {
		if responses.Add(1) > 2 {
			term()
			return nil
		}
		return defaultAccountRequestHandlerV2(tp, id, root, origin, limit, cap)
	}
	syncer.Register(src)
	src.remote = syncer

	pivot := mkPivot(0, sourceAccountTrie.Hash())
	// Sync should be interrupted by the cancel after a couple of responses.
	_ = syncer.Sync(pivot, cancel)

	// Persisted previousPivot must equal the pivot, so a follow-up Sync at a
	// different pivot can recognize the partial flat state belongs to this one.
	loader := newSyncerV2(db, nodeScheme)
	loader.loadSyncStatus()
	if loader.previousPivot == nil {
		t.Fatal("expected persisted previousPivot to be set after interrupted download, got nil")
	}
	if loader.previousPivot.Hash() != pivot.Hash() {
		t.Errorf("persisted previousPivot mismatch: got %v, want %v", loader.previousPivot.Hash(), pivot.Hash())
	}
}

// TestPivotMovement verifies the full pivot move flow: download with rootA,
// cancel+restart with rootB, catch-up applies BAL diffs, download resumes
// and completes against the new state.
func TestPivotMovement(t *testing.T) {
	t.Parallel()
	testPivotMovement(t, rawdb.HashScheme, 1)
}

// TestPivotMovementRepeated verifies that multiple pivot moves work correctly.
func TestPivotMovementRepeated(t *testing.T) {
	t.Parallel()
	testPivotMovement(t, rawdb.HashScheme, 2)
}

func testPivotMovement(t *testing.T, scheme string, pivotMoves int) {
	// Use makeAccountTrieWithAddresses so trie keys are keccak(addr),
	// matching what applyAccessList writes to the snapshot DB.
	nodeScheme, sourceAccountTrie, elems, addrs := makeAccountTrieWithAddresses(100, scheme)
	numA := uint64(100)

	// Target account 50 for BAL changes
	targetAddr := addrs[49]
	targetHash := crypto.Keccak256Hash(targetAddr[:])

	type pivotMove struct {
		blockNum uint64
		trie     *trie.Trie
		elems    []*kv
		root     common.Hash
		bals     map[common.Hash]rlp.RawValue // header hash -> encoded BAL
		balance  *uint256.Int
	}

	// Build each pivot move: update account 50's balance in both the trie
	// and a BAL, write the header, and record everything.
	db := rawdb.NewMemoryDatabase()
	currentElems := elems
	moves := make([]pivotMove, pivotMoves)
	emptyHash := common.Hash{}
	zero := uint64(0)
	for m := 0; m < pivotMoves; m++ {
		blockNum := numA + uint64(m) + 1
		balance := uint256.NewInt(uint64(1000 * (m + 1)))

		// Build updated trie with new balance for account 50
		trieDB := triedb.NewDatabase(rawdb.NewMemoryDatabase(), newDbConfig(scheme))
		newTrie := trie.NewEmpty(trieDB)
		newElems := make([]*kv, len(currentElems))
		for i, entry := range currentElems {
			if bytes.Equal(entry.k, targetHash[:]) {
				val, _ := rlp.EncodeToBytes(&types.StateAccount{
					Nonce: 50, Balance: balance,
					Root: types.EmptyRootHash, CodeHash: types.EmptyCodeHash[:],
				})
				newElems[i] = &kv{entry.k, val}
			} else {
				newElems[i] = entry
			}
			newTrie.MustUpdate(newElems[i].k, newElems[i].v)
		}
		newRoot, nodes := newTrie.Commit(false)
		trieDB.Update(newRoot, types.EmptyRootHash, 0, trienode.NewWithNodeSet(nodes), triedb.NewStateSet())
		resultTrie, _ := trie.New(trie.StateTrieID(newRoot), trieDB)

		// Build BAL matching the trie diff
		cb := bal.NewConstructionBlockAccessList()
		cb.BalanceChange(0, targetAddr, balance)
		var buf bytes.Buffer
		if err := cb.EncodeRLP(&buf); err != nil {
			t.Fatal(err)
		}

		// Compute BAL hash, write header, store BAL keyed by header hash
		var b bal.BlockAccessList
		if err := rlp.DecodeBytes(buf.Bytes(), &b); err != nil {
			t.Fatal(err)
		}
		balHash := b.Hash()
		header := &types.Header{
			Number: new(big.Int).SetUint64(blockNum), Difficulty: common.Big0,
			BaseFee: common.Big0, WithdrawalsHash: &emptyHash,
			BlobGasUsed: &zero, ExcessBlobGas: &zero,
			ParentBeaconRoot: &emptyHash, RequestsHash: &emptyHash,
			BlockAccessListHash: &balHash,
		}
		rawdb.WriteHeader(db, header)
		headerHash := header.Hash()
		rawdb.WriteCanonicalHash(db, headerHash, blockNum)
		moves[m] = pivotMove{
			blockNum: blockNum,
			trie:     resultTrie,
			elems:    newElems,
			root:     newRoot,
			bals:     map[common.Hash]rlp.RawValue{headerHash: buf.Bytes()},
			balance:  balance,
		}
		currentElems = newElems
	}

	// First run: download against rootA, cancel after 2 responses
	rootA := sourceAccountTrie.Hash()
	var (
		once1     sync.Once
		cancel1   = make(chan struct{})
		term1     = func() { once1.Do(func() { close(cancel1) }) }
		responses atomic.Int32
	)
	syncer1 := newSyncerV2(db, nodeScheme)
	src1 := newTestPeerV2("source1", t, term1)
	src1.accountTrie = sourceAccountTrie.Copy()
	src1.accountValues = elems
	src1.accountRequestV2Handler = func(tp *testPeerV2, id uint64, root common.Hash, origin common.Hash, limit common.Hash, cap int) error {
		if responses.Add(1) > 2 {
			term1()
			return nil
		}
		return defaultAccountRequestHandlerV2(tp, id, root, origin, limit, cap)
	}
	syncer1.Register(src1)
	src1.remote = syncer1
	syncer1.Sync(mkPivot(numA, rootA), cancel1)

	// Subsequent runs: each move triggers catch-up then resumes download
	for i, move := range moves {
		var (
			once   sync.Once
			cancel = make(chan struct{})
			term   = func() { once.Do(func() { close(cancel) }) }
		)
		syncer := newSyncerV2(db, nodeScheme)
		src := newTestPeerV2(fmt.Sprintf("source-%d", i+2), t, term)
		src.accountTrie = move.trie.Copy()
		src.accountValues = move.elems
		src.accessLists = move.bals
		syncer.Register(src)
		src.remote = syncer
		if err := syncer.Sync(mkPivot(move.blockNum, move.root), cancel); err != nil {
			t.Fatalf("pivot move %d: sync failed: %v", i+1, err)
		}

		// Verify account 50's balance was updated by catch-up
		data := rawdb.ReadAccountSnapshot(db, targetHash)
		if len(data) == 0 {
			t.Fatalf("pivot move %d: account 50 not found after sync", i+1)
		}
		account, aErr := types.FullAccount(data)
		if aErr != nil {
			t.Fatalf("pivot move %d: failed to decode account: %v", i+1, aErr)
		}
		if account.Balance.Cmp(move.balance) != 0 {
			t.Errorf("pivot move %d: balance wrong: got %v, want %v", i+1, account.Balance, move.balance)
		}
	}
}

// TestCatchUpPersistsIncrementally verifies that catchUp updates and persists
// previousPivot after each successfully applied BAL. If a later block in the
// gap fails to apply, the persisted state reflects the last successful block,
// so a follow-up Sync can resume from there rather than reapplying everything.
func TestCatchUpPersistsIncrementally(t *testing.T) {
	t.Parallel()

	nodeScheme, sourceAccountTrie, elems, addrs := makeAccountTrieWithAddresses(100, rawdb.HashScheme)
	rootA := sourceAccountTrie.Hash()
	numA := uint64(100)

	goodAddr := addrs[0]
	corruptAddr := addrs[1]

	type balBlock struct {
		header *types.Header
		bal    rlp.RawValue
	}

	db := rawdb.NewMemoryDatabase()
	emptyHash := common.Hash{}
	zero := uint64(0)

	// Write the header and canonical hash for block A so the reorg-detection
	// canonical-lookup in Sync passes (otherwise it'd treat A as reorged out
	// and reset instead of running catchUp).
	pivotAHeader := &types.Header{
		Number: new(big.Int).SetUint64(numA), Root: rootA, Difficulty: common.Big0,
		BaseFee: common.Big0, WithdrawalsHash: &emptyHash,
		BlobGasUsed: &zero, ExcessBlobGas: &zero,
		ParentBeaconRoot: &emptyHash, RequestsHash: &emptyHash,
	}
	rawdb.WriteHeader(db, pivotAHeader)
	rawdb.WriteCanonicalHash(db, pivotAHeader.Hash(), numA)
	pivotA := pivotAHeader

	// Build three sequential BAL blocks (A+1, A+2, A+3). The first two touch
	// goodAddr, the third touches corruptAddr so that block's apply fails
	// once we've corrupted that account's snapshot.
	blocks := make([]balBlock, 3)
	for i := 0; i < 3; i++ {
		blockNum := numA + uint64(i) + 1
		target := goodAddr
		if i == 2 {
			target = corruptAddr
		}
		balance := uint256.NewInt(uint64(1000 * (i + 1)))

		cb := bal.NewConstructionBlockAccessList()
		cb.BalanceChange(0, target, balance)
		var buf bytes.Buffer
		if err := cb.EncodeRLP(&buf); err != nil {
			t.Fatal(err)
		}
		var b bal.BlockAccessList
		if err := rlp.DecodeBytes(buf.Bytes(), &b); err != nil {
			t.Fatal(err)
		}
		balHash := b.Hash()
		header := &types.Header{
			Number: new(big.Int).SetUint64(blockNum), Difficulty: common.Big0,
			BaseFee: common.Big0, WithdrawalsHash: &emptyHash,
			BlobGasUsed: &zero, ExcessBlobGas: &zero,
			ParentBeaconRoot: &emptyHash, RequestsHash: &emptyHash,
			BlockAccessListHash: &balHash,
		}
		rawdb.WriteHeader(db, header)
		rawdb.WriteCanonicalHash(db, header.Hash(), blockNum)
		blocks[i] = balBlock{header: header, bal: buf.Bytes()}
	}

	// First sync: complete sync to A so persisted state has previousPivot=A,
	// flat state covers all accounts.
	{
		var (
			once   sync.Once
			cancel = make(chan struct{})
			term   = func() { once.Do(func() { close(cancel) }) }
		)
		syncer := newSyncerV2(db, nodeScheme)
		src := newTestPeerV2("seed", t, term)
		src.accountTrie = sourceAccountTrie.Copy()
		src.accountValues = elems
		syncer.Register(src)
		src.remote = syncer
		if err := syncer.Sync(pivotA, cancel); err != nil {
			t.Fatalf("seed sync failed: %v", err)
		}
	}

	// Corrupt the flat-state snapshot for corruptAddr so applyAccessList will
	// fail when block A+3's BAL touches it. types.FullAccount rejects this
	// payload as undecodable.
	rawdb.WriteAccountSnapshot(db, crypto.Keccak256Hash(corruptAddr[:]), []byte{0xff, 0xff, 0xff, 0xff})

	// Second sync: target is A+3. catchUp should apply A+1 and A+2 (good
	// account), persist after each, then fail on A+3 (corrupt account).
	pivotB := blocks[2].header
	balsByHash := map[common.Hash]rlp.RawValue{
		blocks[0].header.Hash(): blocks[0].bal,
		blocks[1].header.Hash(): blocks[1].bal,
		blocks[2].header.Hash(): blocks[2].bal,
	}

	var (
		once   sync.Once
		cancel = make(chan struct{})
		term   = func() { once.Do(func() { close(cancel) }) }
	)
	syncer := newSyncerV2(db, nodeScheme)
	src := newTestPeerV2("catchup", t, term)
	src.accountTrie = sourceAccountTrie.Copy()
	src.accountValues = elems
	src.accessLists = balsByHash
	syncer.Register(src)
	src.remote = syncer

	if err := syncer.Sync(pivotB, cancel); err == nil {
		t.Fatal("expected Sync to fail when applyAccessList hits corrupt flat state")
	}

	// Persisted previousPivot should now reflect the last successfully applied
	// block (A+2). Without per-iteration saves, it would still be at A.
	loader := newSyncerV2(db, nodeScheme)
	loader.loadSyncStatus()
	if loader.previousPivot == nil {
		t.Fatal("expected persisted previousPivot to be set after partial catchUp")
	}
	wantHash := blocks[1].header.Hash()
	if loader.previousPivot.Hash() != wantHash {
		t.Errorf("persisted previousPivot mismatch after partial catchUp: got %v, want %v (block A+2)",
			loader.previousPivot.Hash(), wantHash)
	}
}

// TestSyncStatusMarkedCompleteAfterCompletion verifies that after a full sync
// completes, the persisted sync status has Complete=true. This lets a
// subsequent Sync call distinguish "already done" from "fresh node" and skip.
func TestSyncStatusMarkedCompleteAfterCompletion(t *testing.T) {
	t.Parallel()
	testSyncStatusMarkedCompleteAfterCompletion(t, rawdb.HashScheme)
	testSyncStatusMarkedCompleteAfterCompletion(t, rawdb.PathScheme)
}

func testSyncStatusMarkedCompleteAfterCompletion(t *testing.T, scheme string) {
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
	pivot := mkPivot(0, sourceAccountTrie.Hash())
	if err := syncer.Sync(pivot, cancel); err != nil {
		t.Fatalf("sync failed: %v", err)
	}

	// After successful sync, persisted status should be present with
	// Complete=true and the pivot we synced to.
	loader := newSyncerV2(syncer.db, nodeScheme)
	loader.loadSyncStatus()
	if !loader.complete {
		t.Fatal("expected persisted status to have Complete=true after successful sync")
	}
	if loader.previousPivot == nil || loader.previousPivot.Hash() != pivot.Hash() {
		t.Fatalf("expected persisted pivot to match synced pivot")
	}
}

// TestSyncSkipsIfAlreadyComplete verifies that a follow-up Sync call for the
// same pivot returns immediately without doing any work, since the persisted
// status indicates the sync is already complete. To prove the skip path actually
// fires, we deliberately wipe the flat state between the two calls. If it skips,
// Sync returns nil without touching flat state. If it doesn't kip, GenerateTrie
// would run against an empty snapshot and fail with a root mismatch.
func TestSyncSkipsIfAlreadyComplete(t *testing.T) {
	t.Parallel()

	nodeScheme, sourceAccountTrie, elems := makeAccountTrieNoStorage(100, rawdb.HashScheme)
	pivot := mkPivot(0, sourceAccountTrie.Hash())

	var (
		once1   sync.Once
		cancel1 = make(chan struct{})
		term1   = func() { once1.Do(func() { close(cancel1) }) }
	)
	src1 := newTestPeerV2("source1", t, term1)
	src1.accountTrie = sourceAccountTrie.Copy()
	src1.accountValues = elems
	syncer := setupSyncerV2(nodeScheme, src1)
	if err := syncer.Sync(pivot, cancel1); err != nil {
		t.Fatalf("first sync failed: %v", err)
	}

	// Wipe the flat state. The persisted status (with Complete=true) stays.
	if err := syncer.db.DeleteRange(rawdb.SnapshotAccountPrefix, []byte{rawdb.SnapshotAccountPrefix[0] + 1}); err != nil {
		t.Fatalf("failed to wipe account snapshot: %v", err)
	}
	if err := syncer.db.DeleteRange(rawdb.SnapshotStoragePrefix, []byte{rawdb.SnapshotStoragePrefix[0] + 1}); err != nil {
		t.Fatalf("failed to wipe storage snapshot: %v", err)
	}

	// Second sync must take the skip path. If it didn't, the empty flat
	// state would cause GenerateTrie to fail with a root mismatch.
	cancel2 := make(chan struct{})
	if err := syncer.Sync(pivot, cancel2); err != nil {
		t.Fatalf("second sync should have skipped, got error: %v", err)
	}
}

// TestInterruptedRebuildRecovery verifies that if sync is interrupted after
// download completes but before trie rebuild finishes, the next Sync() call
// re-runs the download (which completes immediately) and rebuild.
func TestInterruptedRebuildRecovery(t *testing.T) {
	t.Parallel()

	nodeScheme, sourceAccountTrie, elems := makeAccountTrieNoStorage(100, rawdb.HashScheme)
	root := sourceAccountTrie.Hash()

	// First run: complete download, save status, simulate interruption
	// before rebuild by calling downloadState() directly
	var (
		once1   sync.Once
		cancel1 = make(chan struct{})
		term1   = func() { once1.Do(func() { close(cancel1) }) }
	)
	db := rawdb.NewMemoryDatabase()
	syncer1 := newSyncerV2(db, nodeScheme)
	src1 := newTestPeerV2("source1", t, term1)
	src1.accountTrie = sourceAccountTrie.Copy()
	src1.accountValues = elems
	syncer1.Register(src1)
	src1.remote = syncer1
	pivot := mkPivot(0, root)
	syncer1.pivot = pivot
	syncer1.previousPivot = pivot // Sync sets this before downloadState
	syncer1.loadSyncStatus()

	if err := syncer1.downloadState(cancel1); err != nil {
		t.Fatalf("download failed: %v", err)
	}
	// Save status (simulating what Sync's defer does)
	for _, task := range syncer1.tasks {
		syncer1.forwardAccountTask(task)
	}
	syncer1.cleanAccountTasks()
	syncer1.saveSyncStatus()

	// Status should exist (rebuild hasn't run yet)
	if rawdb.ReadSnapshotSyncStatus(db) == nil {
		t.Fatal("sync status should exist after download")
	}
	// Second run: full Sync should detect tasks are done, run rebuild
	var (
		once2   sync.Once
		cancel2 = make(chan struct{})
		term2   = func() { once2.Do(func() { close(cancel2) }) }
	)
	syncer2 := newSyncerV2(db, nodeScheme)
	src2 := newTestPeerV2("source2", t, term2)
	src2.accountTrie = sourceAccountTrie.Copy()
	src2.accountValues = elems
	syncer2.Register(src2)
	src2.remote = syncer2

	if err := syncer2.Sync(mkPivot(0, root), cancel2); err != nil {
		t.Fatalf("resumed sync failed: %v", err)
	}
	// After rebuild completes, status should be marked Complete=true.
	loader := newSyncerV2(db, nodeScheme)
	loader.loadSyncStatus()
	if !loader.complete {
		t.Fatal("sync status should be marked Complete=true after rebuild completes")
	}
}

// TestFetchAccessListsMultiplePeers verifies that fetch distributes work
// across multiple idle peers.
func TestFetchAccessListsMultiplePeers(t *testing.T) {
	t.Parallel()
	var (
		once   sync.Once
		cancel = make(chan struct{})
		term   = func() { once.Do(func() { close(cancel) }) }
	)

	// Create enough BALs to potentially split across peers
	var hashes []common.Hash
	bals := make(map[common.Hash]rlp.RawValue)
	for i := 0; i < 10; i++ {
		h := common.HexToHash(fmt.Sprintf("0x%02x", i+1))
		hashes = append(hashes, h)
		cb := bal.NewConstructionBlockAccessList()
		cb.BalanceChange(0, common.HexToAddress("0xaa"), uint256.NewInt(uint64(i)))
		var buf bytes.Buffer
		if err := cb.EncodeRLP(&buf); err != nil {
			t.Fatal(err)
		}
		bals[h] = buf.Bytes()
	}
	mkSource := func(name string) *testPeerV2 {
		source := newTestPeerV2(name, t, term)
		source.accessLists = bals
		return source
	}
	syncer := setupSyncerV2(rawdb.HashScheme, mkSource("peer-a"), mkSource("peer-b"), mkSource("peer-c"))
	results, err := syncer.fetchAccessLists(hashes, makeAccessListHeaders(bals), cancel)
	if err != nil {
		t.Fatalf("fetchAccessLists failed: %v", err)
	}
	if len(results) != len(hashes) {
		t.Fatalf("result count mismatch: got %d, want %d", len(results), len(hashes))
	}
	// Verify results match expected content in request order
	for i, h := range hashes {
		if !bytes.Equal(results[i], bals[h]) {
			t.Errorf("result %d content mismatch for hash %v", i, h)
		}
	}
}

// TestFetchAccessListsPeerTimeout verifies that timed-out requests are retried
// with a different peer.
func TestFetchAccessListsPeerTimeout(t *testing.T) {
	t.Parallel()
	var (
		once   sync.Once
		cancel = make(chan struct{})
		term   = func() { once.Do(func() { close(cancel) }) }
	)
	hashes := []common.Hash{common.HexToHash("0x01")}
	bals := make(map[common.Hash]rlp.RawValue)
	cb := bal.NewConstructionBlockAccessList()
	cb.BalanceChange(0, common.HexToAddress("0xaa"), uint256.NewInt(42))
	var buf bytes.Buffer
	if err := cb.EncodeRLP(&buf); err != nil {
		t.Fatal(err)
	}
	bals[hashes[0]] = buf.Bytes()

	// First peer never responds
	nonResponsive := newTestPeerV2("non-responsive", t, term)
	nonResponsive.accessListRequestHandler = func(t *testPeerV2, id uint64, hashes []common.Hash, max int) error {
		// Don't respond — let it time out
		return nil
	}

	// Second peer serves correctly
	good := newTestPeerV2("good", t, term)
	good.accessLists = bals
	syncer := setupSyncerV2(rawdb.HashScheme, nonResponsive, good)
	syncer.rates.OverrideTTLLimit = time.Millisecond // Fast timeout
	results, err := syncer.fetchAccessLists(hashes, makeAccessListHeaders(bals), cancel)
	if err != nil {
		t.Fatalf("fetchAccessLists failed: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("result count mismatch: got %d, want 1", len(results))
	}
}

// TestFetchAccessListsPeerRejection verifies that peers returning empty
// responses are marked stateless and work is retried with another peer.
func TestFetchAccessListsPeerRejection(t *testing.T) {
	t.Parallel()
	var (
		once   sync.Once
		cancel = make(chan struct{})
		term   = func() { once.Do(func() { close(cancel) }) }
	)
	hashes := []common.Hash{common.HexToHash("0x01")}
	bals := make(map[common.Hash]rlp.RawValue)
	cb := bal.NewConstructionBlockAccessList()
	cb.BalanceChange(0, common.HexToAddress("0xaa"), uint256.NewInt(42))
	var buf bytes.Buffer
	if err := cb.EncodeRLP(&buf); err != nil {
		t.Fatal(err)
	}
	bals[hashes[0]] = buf.Bytes()

	// First peer rejects (has no BAL data, returns empty)
	// accessLists is nil, so defaultAccessListRequestHandler returns empty
	rejector := newTestPeerV2("rejector", t, term)

	// Second peer serves correctly
	good := newTestPeerV2("good", t, term)
	good.accessLists = bals
	syncer := setupSyncerV2(rawdb.HashScheme, rejector, good)
	results, err := syncer.fetchAccessLists(hashes, makeAccessListHeaders(bals), cancel)
	if err != nil {
		t.Fatalf("fetchAccessLists failed: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("result count mismatch: got %d, want 1", len(results))
	}
}

// TestFetchAccessListsCancel verifies that fetchAccessLists returns promptly
// when cancelled.
func TestFetchAccessListsCancel(t *testing.T) {
	t.Parallel()
	cancel := make(chan struct{})

	// Peer that never responds
	nonResponsive := newTestPeerV2("non-responsive", t, func() {})
	nonResponsive.accessListRequestHandler = func(t *testPeerV2, id uint64, hashes []common.Hash, max int) error {
		return nil // never deliver
	}
	syncer := setupSyncerV2(rawdb.HashScheme, nonResponsive)
	hashes := []common.Hash{common.HexToHash("0x01")}

	// Cancel after a short delay
	go func() {
		time.Sleep(50 * time.Millisecond)
		close(cancel)
	}()
	_, err := syncer.fetchAccessLists(hashes, nil, cancel)
	if err != ErrCancelled {
		t.Fatalf("expected ErrCancelled, got %v", err)
	}
}

// TestFetchAccessListsPeerDrop verifies that dropping a peer mid-request
// causes the request to be retried with a different peer.
func TestFetchAccessListsPeerDrop(t *testing.T) {
	t.Parallel()
	var (
		once   sync.Once
		cancel = make(chan struct{})
		term   = func() { once.Do(func() { close(cancel) }) }
	)
	hashes := []common.Hash{common.HexToHash("0x01")}
	bals := make(map[common.Hash]rlp.RawValue)
	cb := bal.NewConstructionBlockAccessList()
	cb.BalanceChange(0, common.HexToAddress("0xaa"), uint256.NewInt(42))
	var buf bytes.Buffer
	if err := cb.EncodeRLP(&buf); err != nil {
		t.Fatal(err)
	}
	bals[hashes[0]] = buf.Bytes()

	// First peer will be dropped mid-request
	dropped := newTestPeerV2("dropped", t, term)
	dropped.accessListRequestHandler = func(tp *testPeerV2, id uint64, hashes []common.Hash, max int) error {
		// Simulate peer dropping by unregistering
		tp.remote.Unregister(tp.id)
		return nil
	}

	// Second peer serves correctly
	good := newTestPeerV2("good", t, term)
	good.accessLists = bals
	syncer := setupSyncerV2(rawdb.HashScheme, dropped, good)
	results, err := syncer.fetchAccessLists(hashes, makeAccessListHeaders(bals), cancel)
	if err != nil {
		t.Fatalf("fetchAccessLists failed: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("result count mismatch: got %d, want 1", len(results))
	}
}

// TestFetchAccessListsShortResponse verifies that when a peer returns fewer
// BALs than requested (a short/partial response), the un-served hashes are
// retried and eventually all results are collected.
func TestFetchAccessListsShortResponse(t *testing.T) {
	t.Parallel()
	var (
		once   sync.Once
		cancel = make(chan struct{})
		term   = func() { once.Do(func() { close(cancel) }) }
	)

	// Request 4 hashes but the peer only returns the first 2.
	hashes := []common.Hash{
		common.HexToHash("0x01"),
		common.HexToHash("0x02"),
		common.HexToHash("0x03"),
		common.HexToHash("0x04"),
	}
	allBALs := make(map[common.Hash]rlp.RawValue)
	for _, h := range hashes {
		cb := bal.NewConstructionBlockAccessList()
		cb.BalanceChange(0, common.HexToAddress("0xaa"), uint256.NewInt(uint64(h[31])))
		var buf bytes.Buffer
		if err := cb.EncodeRLP(&buf); err != nil {
			t.Fatal(err)
		}
		allBALs[h] = buf.Bytes()
	}

	// shortPeer returns only the first 2 BALs regardless of how many are
	// requested. This simulates a peer that truncates its response (e.g.,
	// hitting the 2 MiB response soft limit).
	shortPeer := newTestPeerV2("short", t, term)
	shortPeer.accessListRequestHandler = func(tp *testPeerV2, id uint64, reqHashes []common.Hash, max int) error {
		// Return only the first 2 of however many were requested.
		limit := 2
		if len(reqHashes) < limit {
			limit = len(reqHashes)
		}
		var results []rlp.RawValue
		for i := 0; i < limit; i++ {
			results = append(results, allBALs[reqHashes[i]])
		}
		rawList, _ := rlp.EncodeToRawList(results)
		if err := tp.remote.OnAccessLists(tp, id, rawList); err != nil {
			tp.test.Errorf("delivery rejected: %v", err)
			tp.term()
		}
		return nil
	}
	syncer := setupSyncerV2(rawdb.HashScheme, shortPeer)

	// Pre-seed the rate tracker so the peer's capacity for AccessListsMsg is
	// high enough to get all 4 hashes assigned in a single request. Without
	// this, the default capacity is 1, so the peer would only get 1 hash per
	// round and the short-response scenario never triggers.
	syncer.rates.Update(shortPeer.id, AccessListsMsg, time.Millisecond, 100)

	// If the bug exists, this will hang.
	done := make(chan struct{})
	var (
		results  []rlp.RawValue
		fetchErr error
	)
	go func() {
		results, fetchErr = syncer.fetchAccessLists(hashes, makeAccessListHeaders(allBALs), cancel)
		close(done)
	}()

	select {
	case <-done:
		// fetchAccessLists returned
	case <-time.After(5 * time.Second):
		t.Fatal("fetchAccessLists has hung. This means unserved hashes were never re-added to pending.")
	}
	if fetchErr != nil {
		t.Fatalf("fetchAccessLists failed: %v", fetchErr)
	}
	if len(results) != len(hashes) {
		t.Fatalf("result count mismatch: got %d, want %d", len(results), len(hashes))
	}

	// Verify all results are non-nil and in correct order
	for i, h := range hashes {
		if results[i] == nil {
			t.Errorf("result %d (hash %v) is nil", i, h)
		}
	}
}

// TestFetchAccessListsEmptyPlaceholder verifies that when a peer returns
// rlp.EmptyString placeholders for BALs it doesn't have, those placeholders
// are not silently accepted as valid results.
func TestFetchAccessListsEmptyPlaceholder(t *testing.T) {
	t.Parallel()
	var (
		once   sync.Once
		cancel = make(chan struct{})
		term   = func() { once.Do(func() { close(cancel) }) }
	)
	hashes := []common.Hash{
		common.HexToHash("0x01"),
		common.HexToHash("0x02"),
		common.HexToHash("0x03"),
	}

	// Build BALs for all 3 hashes
	allBALs := make(map[common.Hash]rlp.RawValue)
	for _, h := range hashes {
		cb := bal.NewConstructionBlockAccessList()
		cb.BalanceChange(0, common.HexToAddress("0xaa"), uint256.NewInt(uint64(h[31])))
		var buf bytes.Buffer
		if err := cb.EncodeRLP(&buf); err != nil {
			t.Fatal(err)
		}
		allBALs[h] = buf.Bytes()
	}

	// partialPeer has BALs for hashes 0 and 2. The server
	// handler returns rlp.EmptyString for the missing BAL.
	partialPeer := newTestPeerV2("partial", t, term)
	partialPeer.accessListRequestHandler = func(tp *testPeerV2, id uint64, reqHashes []common.Hash, max int) error {
		var results []rlp.RawValue
		for _, h := range reqHashes {
			if raw, ok := allBALs[h]; ok && h != hashes[1] {
				results = append(results, raw)
			} else {
				results = append(results, rlp.EmptyString)
			}
		}
		rawList, _ := rlp.EncodeToRawList(results)
		if err := tp.remote.OnAccessLists(tp, id, rawList); err != nil {
			tp.test.Errorf("delivery rejected: %v", err)
			tp.term()
		}
		return nil
	}

	// fullPeer has all BALs
	fullPeer := newTestPeerV2("full", t, term)
	fullPeer.accessLists = allBALs
	syncer := setupSyncerV2(rawdb.HashScheme, partialPeer, fullPeer)

	// Pre-seed capacity so partialPeer gets all 3 hashes
	syncer.rates.Update(partialPeer.id, AccessListsMsg, time.Millisecond, 100)
	done := make(chan struct{})
	var (
		results  []rlp.RawValue
		fetchErr error
	)
	go func() {
		results, fetchErr = syncer.fetchAccessLists(hashes, makeAccessListHeaders(allBALs), cancel)
		close(done)
	}()

	// Wait for fetch to complete
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("fetchAccessLists hung")
	}
	if fetchErr != nil {
		t.Fatalf("fetchAccessLists failed: %v", fetchErr)
	}

	// Verify the results are valid.
	for i, raw := range results {
		var accessList bal.BlockAccessList
		if err := rlp.DecodeBytes(raw, &accessList); err != nil {
			t.Errorf("result %d (hash %v) is not a valid BAL: %v (got raw bytes %x)",
				i, hashes[i], err, raw)
		}
	}
}

// TestFetchAccessListsRejectsBadBAL verifies that when a peer delivers a BAL
// whose hash doesn't match the canonical block header, fetchAccessLists marks
// the peer stateless, drops the response, and surfaces the exhaustion error
// once no other peers can serve the work.
func TestFetchAccessListsRejectsBadBAL(t *testing.T) {
	t.Parallel()
	var (
		once   sync.Once
		cancel = make(chan struct{})
		term   = func() { once.Do(func() { close(cancel) }) }
	)
	hash := common.HexToHash("0x01")
	hashes := []common.Hash{hash}

	// Build a BAL we'll actually serve.
	cb := bal.NewConstructionBlockAccessList()
	cb.BalanceChange(0, common.HexToAddress("0xaa"), uint256.NewInt(42))
	var buf bytes.Buffer
	if err := cb.EncodeRLP(&buf); err != nil {
		t.Fatal(err)
	}
	served := buf.Bytes()

	// Build a header whose BlockAccessListHash points at something else, so
	// the served BAL fails verification.
	mismatch := common.HexToHash("0xdeadbeef")
	headers := map[common.Hash]*types.Header{
		hash: {BlockAccessListHash: &mismatch},
	}

	peer := newTestPeerV2("liar", t, term)
	peer.accessLists = map[common.Hash]rlp.RawValue{hash: served}
	syncer := setupSyncerV2(rawdb.HashScheme, peer)

	results, err := syncer.fetchAccessLists(hashes, headers, cancel)
	if !errors.Is(err, errAccessListPeersExhausted) {
		t.Fatalf("expected errAccessListPeersExhausted, got %v", err)
	}
	if results != nil {
		t.Errorf("expected nil results on error, got %v", results)
	}
	syncer.lock.RLock()
	_, stateless := syncer.statelessPeers[peer.id]
	syncer.lock.RUnlock()
	if !stateless {
		t.Error("expected liar peer to be marked stateless after bad BAL")
	}
}

// TestCatchUpRetriesOnBadBAL verifies that when one peer serves a BAL that
// fails verification but another serves a valid one, fetchAccessLists routes
// the work around the bad peer and returns the verified BAL.
func TestCatchUpRetriesOnBadBAL(t *testing.T) {
	t.Parallel()
	var (
		once   sync.Once
		cancel = make(chan struct{})
		term   = func() { once.Do(func() { close(cancel) }) }
	)
	hash := common.HexToHash("0x01")
	hashes := []common.Hash{hash}

	cb := bal.NewConstructionBlockAccessList()
	cb.BalanceChange(0, common.HexToAddress("0xaa"), uint256.NewInt(42))
	var buf bytes.Buffer
	if err := cb.EncodeRLP(&buf); err != nil {
		t.Fatal(err)
	}
	good := buf.Bytes()

	// A second BAL with different content used as the "bad" payload. It
	// decodes cleanly but its hash will not match the header.
	other := bal.NewConstructionBlockAccessList()
	other.BalanceChange(0, common.HexToAddress("0xbb"), uint256.NewInt(99))
	var otherBuf bytes.Buffer
	if err := other.EncodeRLP(&otherBuf); err != nil {
		t.Fatal(err)
	}
	bad := otherBuf.Bytes()

	headers := makeAccessListHeaders(map[common.Hash]rlp.RawValue{hash: good})

	liar := newTestPeerV2("liar", t, term)
	liar.accessLists = map[common.Hash]rlp.RawValue{hash: bad}
	honest := newTestPeerV2("honest", t, term)
	honest.accessLists = map[common.Hash]rlp.RawValue{hash: good}

	syncer := setupSyncerV2(rawdb.HashScheme, liar, honest)
	// Bias the capacity sort so the liar is asked first, exercising the
	// reject-and-retry path rather than getting lucky on assignment order.
	syncer.rates.Update(liar.id, AccessListsMsg, time.Millisecond, 1000)

	results, err := syncer.fetchAccessLists(hashes, headers, cancel)
	if err != nil {
		t.Fatalf("fetchAccessLists failed: %v", err)
	}
	if !bytes.Equal(results[0], good) {
		t.Errorf("expected the honest BAL, got %x", results[0])
	}
	syncer.lock.RLock()
	_, liarStateless := syncer.statelessPeers[liar.id]
	_, honestStateless := syncer.statelessPeers[honest.id]
	syncer.lock.RUnlock()
	if !liarStateless {
		t.Error("expected liar to be marked stateless")
	}
	if honestStateless {
		t.Error("expected honest peer to remain in good standing")
	}
}
