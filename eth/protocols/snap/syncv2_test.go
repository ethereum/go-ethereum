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
	"encoding/json"
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

// TestSyncV2FrozenPivot checks the pivot freeze signal around the sync
// lifecycle. The pivot is unfrozen while flat state is downloading, frozen
// once the download completes, stays frozen after the sync returns so the
// downloader resumes against it until the pivot block is committed, and
// unfreezes again after a state reset.
func TestSyncV2FrozenPivot(t *testing.T) {
	t.Parallel()
	testSyncV2FrozenPivot(t, rawdb.HashScheme)
	testSyncV2FrozenPivot(t, rawdb.PathScheme)
}

func testSyncV2FrozenPivot(t *testing.T, scheme string) {
	var (
		once   sync.Once
		cancel = make(chan struct{})
		term   = func() { once.Do(func() { close(cancel) }) }
	)
	nodeScheme, sourceAccountTrie, elems := makeAccountTrieNoStorage(100, scheme)

	source := newTestPeerV2("source", t, term)
	source.accountTrie = sourceAccountTrie.Copy()
	source.accountValues = elems

	syncer := setupSyncerV2(nodeScheme, source)
	pivot := mkPivot(0, sourceAccountTrie.Hash())

	// The handler runs while account ranges are still being served, so it
	// can observe the signal mid download.
	source.accountRequestV2Handler = func(p *testPeerV2, requestId uint64, root common.Hash, origin common.Hash, limit common.Hash, cap int) error {
		if syncer.FrozenPivot() != nil {
			t.Error("pivot frozen during flat state download")
		}
		return defaultAccountRequestHandlerV2(p, requestId, root, origin, limit, cap)
	}
	if syncer.FrozenPivot() != nil {
		t.Fatal("pivot frozen before sync started")
	}
	if err := syncer.Sync(pivot, cancel); err != nil {
		t.Fatalf("sync failed: %v", err)
	}
	if frozen := syncer.FrozenPivot(); frozen == nil || frozen.Hash() != pivot.Hash() {
		t.Fatal("pivot not frozen at the synced header after download completed")
	}
	// A restart must not lose the freeze: a fresh syncer instance on the same
	// database derives it from the persisted journal, before any Sync call.
	restarted := newSyncerV2(syncer.db, nodeScheme)
	if frozen := restarted.FrozenPivot(); frozen == nil || frozen.Hash() != pivot.Hash() {
		t.Fatal("pivot freeze lost after restart")
	}
	syncer.resetSyncState()
	if syncer.FrozenPivot() != nil {
		t.Fatal("pivot still frozen after state reset")
	}
	// The reset deletes the journal, so a restarted instance is unfrozen too.
	if restarted := newSyncerV2(syncer.db, nodeScheme); restarted.FrozenPivot() != nil {
		t.Fatal("pivot still frozen after restart following a state reset")
	}
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
	sourceAccountTrie, elems, storageTries, storageElems := makeAccountTrieWithStorage(scheme, 100, 300, true, false, false)

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
	sourceAccountTrie, elems, storageTries, storageElems := makeAccountTrieWithStorage(scheme, 100, 300, true, false, false)

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
	sourceAccountTrie, elems, storageTries, storageElems := makeAccountTrieWithStorage(scheme, 100, 300, true, false, false)

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
	sourceAccountTrie, elems, storageTries, storageElems := makeAccountTrieWithStorage(scheme, 300, 100, false, false, false)

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
	sourceAccountTrie, elems, storageTries, storageElems := makeAccountTrieWithStorage(scheme, 100, 300, true, false, false)

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
	sourceAccountTrie, elems, storageTries, storageElems := makeAccountTrieWithStorage(scheme, 100, 300, true, false, false)

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

// TestIsPivotCommitted checks the commit detection against the head block
// positions that matter.
func TestIsPivotCommitted(t *testing.T) {
	t.Parallel()

	tests := []struct {
		head      int64 // head block number, -1 for no head block at all
		pivot     uint64
		canonical bool // pivot still canonical at its height
		committed bool
	}{
		{-1, 100, true, false},   // no head block, nothing committed
		{50, 100, true, false},   // head behind the pivot, crash before the commit
		{100, 100, true, true},   // head at the pivot, the commit just happened
		{103, 100, true, true},   // head past the pivot, full sync ran after
		{0, 0, true, false},      // genesis head is just an empty chain
		{103, 100, false, false}, // pivot reorged out, head past it on a fork
	}
	for _, tt := range tests {
		db := rawdb.NewMemoryDatabase()
		pivot := mkPivot(tt.pivot, common.HexToHash("0xaaaa"))
		if tt.canonical {
			rawdb.WriteCanonicalHash(db, pivot.Hash(), tt.pivot)
		} else {
			rawdb.WriteCanonicalHash(db, common.HexToHash("0xdead"), tt.pivot)
		}
		if tt.head >= 0 {
			head := mkPivot(uint64(tt.head), common.HexToHash("0xbbbb"))
			rawdb.WriteHeader(db, head)
			rawdb.WriteHeadBlockHash(db, head.Hash())
		}
		if have := isPivotCommitted(db, pivot); have != tt.committed {
			t.Errorf("head %d pivot %d canonical %v: isPivotCommitted = %v, want %v", tt.head, tt.pivot, tt.canonical, have, tt.committed)
		}
	}
}

// TestSyncDetectsPivotReorged exercises the reorg-handling branch in Sync
// end-to-end.
//
// Setup: persisted progress points at an orphan pivot at block 100; the new
// canonical header at block 100 has a different hash. Sync is then called with
// a new pivot at the same height.
//
// If isPivotReorged works, loadSyncStatus restores the persisted pivot, the
// check flags it as reorged, resetSyncState clears it, catchUp is skipped,
// and the fresh download proceeds to completion.
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
	// pivot reflects where flat state matches and it is what saveSyncStatus
	// persists. Set it to simulate a prior sync reaching orphanPivot.
	seed.pivot = orphanPivot
	seed.accountSynced = 42
	seed.tasks = []*accountTaskV2{{
		Next:     common.HexToHash("0x80"),
		Last:     common.MaxHash,
		SubTasks: make(map[common.Hash][]*storageTaskV2),
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
	// After successful completion, status should reach the complete phase
	// against the new (canonical) pivot.
	loader := newSyncerV2(db, nodeScheme)
	loader.loadSyncStatus()
	if loader.getPhase() != phaseComplete {
		t.Fatal("sync status should reach the complete phase after successful completion")
	}
	if loader.pivot == nil || loader.pivot.Hash() != newPivot.Hash() {
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
	syncer1.loadSyncStatus()
	syncer1.pivot = pivot // Sync pins this before downloadState
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
	syncer2.loadSyncStatus()
	syncer2.pivot = pivot2 // Sync pins this before downloadState
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
// interrupted mid-download, the persisted pivot equals the current pivot
// (not nil). Without this, a follow-up Sync at a different pivot would not
// see that the partial flat state belongs to the old pivot, and would mix
// old-pivot accounts with new-pivot data.
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

	// Persisted pivot must equal the pivot, so a follow-up Sync at a different
	// pivot can recognize the partial flat state belongs to this one.
	loader := newSyncerV2(db, nodeScheme)
	loader.loadSyncStatus()
	if loader.pivot == nil {
		t.Fatal("expected persisted pivot to be set after interrupted download, got nil")
	}
	if loader.pivot.Hash() != pivot.Hash() {
		t.Errorf("persisted pivot mismatch: got %v, want %v", loader.pivot.Hash(), pivot.Hash())
	}
}

// TestPivotMovement verifies the full pivot move flow: download with rootA,
// cancel+restart with rootB, catch-up applies BAL diffs, download resumes
// and completes against the new state.
func TestPivotMovement(t *testing.T) {
	t.Parallel()
	testPivotMovement(t, rawdb.HashScheme, 1)
	testPivotMovement(t, rawdb.PathScheme, 1)
}

// TestPivotMovementRepeated verifies that multiple pivot moves work correctly.
func TestPivotMovementRepeated(t *testing.T) {
	t.Parallel()
	testPivotMovement(t, rawdb.HashScheme, 2)
	testPivotMovement(t, rawdb.PathScheme, 2)
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
// the pivot after each successfully applied BAL. If a later block in the
// gap fails to apply, the persisted state reflects the last successful block,
// so a follow-up Sync can resume from there rather than reapplying everything.
func TestCatchUpPersistsIncrementally(t *testing.T) {
	t.Parallel()
	testCatchUpPersistsIncrementally(t, rawdb.HashScheme)
	testCatchUpPersistsIncrementally(t, rawdb.PathScheme)
}

func testCatchUpPersistsIncrementally(t *testing.T, scheme string) {
	nodeScheme, sourceAccountTrie, elems, addrs := makeAccountTrieWithAddresses(100, scheme)
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
	// once we've corrupted that account's snapshot. The headers link up via
	// their parent hashes, as the catch-up walks the chain backward from the
	// target down to the previous pivot.
	blocks := make([]balBlock, 3)
	parent := pivotAHeader.Hash()
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
			ParentHash: parent,
			Number:     new(big.Int).SetUint64(blockNum), Difficulty: common.Big0,
			BaseFee: common.Big0, WithdrawalsHash: &emptyHash,
			BlobGasUsed: &zero, ExcessBlobGas: &zero,
			ParentBeaconRoot: &emptyHash, RequestsHash: &emptyHash,
			BlockAccessListHash: &balHash,
		}
		rawdb.WriteHeader(db, header)
		rawdb.WriteCanonicalHash(db, header.Hash(), blockNum)
		blocks[i] = balBlock{header: header, bal: buf.Bytes()}
		parent = header.Hash()
	}

	// First sync: complete sync to A so persisted state has pivot=A,
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

	// Persisted pivot should now reflect the last successfully applied
	// block (A+2). Without per-iteration saves, it would still be at A.
	loader := newSyncerV2(db, nodeScheme)
	loader.loadSyncStatus()
	if loader.pivot == nil {
		t.Fatal("expected persisted pivot to be set after partial catchUp")
	}
	wantHash := blocks[1].header.Hash()
	if loader.pivot.Hash() != wantHash {
		t.Errorf("persisted pivot mismatch after partial catchUp: got %v, want %v (block A+2)",
			loader.pivot.Hash(), wantHash)
	}
}

// TestCatchUpWindowed verifies that catch-up correctly rolls the flat state
// forward when the gap spans several windows. With catchUpWindow shrunk to 2,
// a 5-block gap is processed as three windows ([A+1,A+2], [A+3,A+4], [A+5]),
// exercising the window boundaries. Every block's BAL must be fetched and
// applied, and the final pivot must reach the target.
func TestCatchUpWindowed(t *testing.T) {
	t.Parallel()
	testCatchUpWindowed(t, rawdb.HashScheme)
	testCatchUpWindowed(t, rawdb.PathScheme)
}

func testCatchUpWindowed(t *testing.T, scheme string) {
	nodeScheme, sourceAccountTrie, elems, addrs := makeAccountTrieWithAddresses(100, scheme)
	rootA := sourceAccountTrie.Hash()
	numA := uint64(100)

	targetAddr := addrs[0]
	targetHash := crypto.Keccak256Hash(targetAddr[:])

	db := rawdb.NewMemoryDatabase()
	emptyHash := common.Hash{}
	zero := uint64(0)

	// Persist header + canonical hash for the base pivot A so reorg detection
	// in Sync passes and catchUp (not reset) runs.
	pivotA := &types.Header{
		Number: new(big.Int).SetUint64(numA), Root: rootA, Difficulty: common.Big0,
		BaseFee: common.Big0, WithdrawalsHash: &emptyHash,
		BlobGasUsed: &zero, ExcessBlobGas: &zero,
		ParentBeaconRoot: &emptyHash, RequestsHash: &emptyHash,
	}
	rawdb.WriteHeader(db, pivotA)
	rawdb.WriteCanonicalHash(db, pivotA.Hash(), numA)

	// Build a 5-block gap, each block bumping targetAddr's balance. The last
	// block's balance is the expected final state. The headers link up via
	// their parent hashes, as the catch-up walks the chain backward from the
	// target down to the previous pivot.
	const gap = 5
	var (
		lastHeader  *types.Header
		lastBalance *uint256.Int
		balsByHash  = make(map[common.Hash]rlp.RawValue, gap)
		parent      = pivotA.Hash()
	)
	for i := 0; i < gap; i++ {
		blockNum := numA + uint64(i) + 1
		balance := uint256.NewInt(uint64(1000 * (i + 1)))

		cb := bal.NewConstructionBlockAccessList()
		cb.BalanceChange(0, targetAddr, balance)
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
			ParentHash: parent,
			Number:     new(big.Int).SetUint64(blockNum), Difficulty: common.Big0,
			BaseFee: common.Big0, WithdrawalsHash: &emptyHash,
			BlobGasUsed: &zero, ExcessBlobGas: &zero,
			ParentBeaconRoot: &emptyHash, RequestsHash: &emptyHash,
			BlockAccessListHash: &balHash,
		}
		rawdb.WriteHeader(db, header)
		rawdb.WriteCanonicalHash(db, header.Hash(), blockNum)
		balsByHash[header.Hash()] = buf.Bytes()
		lastHeader, lastBalance = header, balance
		parent = header.Hash()
	}

	// Seed sync to A: persisted state ends with pivot=A and full flat state.
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

	// Run catch-up to A+5 directly with a window of 2, forcing three windows
	// ([A+1,A+2], [A+3,A+4], [A+5]). Calling catchUp in isolation keeps the
	// focus on the windowing logic without the surrounding download/trie-gen
	// phases (which would need a real target state root).
	var (
		once   sync.Once
		cancel = make(chan struct{})
		term   = func() { once.Do(func() { close(cancel) }) }
	)
	syncer := newSyncerV2(db, nodeScheme)
	syncer.loadSyncStatus() // restores pivot=A from the seed sync
	if syncer.pivot == nil || syncer.pivot.Number.Uint64() != numA {
		t.Fatalf("expected restored pivot at block %d, got %v", numA, syncer.pivot)
	}
	syncer.catchUpWindow = 2
	src := newTestPeerV2("catchup", t, term)
	src.accountTrie = sourceAccountTrie.Copy()
	src.accountValues = elems
	src.accessLists = balsByHash
	syncer.Register(src)
	src.remote = syncer
	if err := syncer.catchUp(lastHeader, cancel); err != nil {
		t.Fatalf("windowed catch-up failed: %v", err)
	}

	// The account must reflect the final block's balance, proving every window
	// (including the trailing partial one) was fetched and applied in order.
	data := rawdb.ReadAccountSnapshot(db, targetHash)
	if len(data) == 0 {
		t.Fatal("target account missing after windowed catch-up")
	}
	account, err := types.FullAccount(data)
	if err != nil {
		t.Fatalf("failed to decode account: %v", err)
	}
	if account.Balance.Cmp(lastBalance) != 0 {
		t.Errorf("balance after windowed catch-up: got %v, want %v", account.Balance, lastBalance)
	}

	// The persisted pivot must have advanced all the way to the target.
	loader := newSyncerV2(db, nodeScheme)
	loader.loadSyncStatus()
	if loader.pivot == nil || loader.pivot.Hash() != lastHeader.Hash() {
		t.Errorf("persisted pivot did not reach target after windowed catch-up")
	}
}

// TestSyncStatusMarkedCompleteAfterCompletion verifies that after a full sync
// completes, the persisted sync status reaches the complete phase. This lets a
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
	// the complete phase and the pivot we synced to.
	loader := newSyncerV2(syncer.db, nodeScheme)
	loader.loadSyncStatus()
	if loader.getPhase() != phaseComplete {
		t.Fatal("expected persisted status to reach the complete phase after successful sync")
	}
	if loader.pivot == nil || loader.pivot.Hash() != pivot.Hash() {
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

	// Wipe the flat state. The persisted status (in the complete phase) stays.
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

// reenableFixture is a completed snap sync at pivot 100 with its journal
// still on disk, plus the pieces for a follow-up sync at block 102.
type reenableFixture struct {
	db         ethdb.Database
	nodeScheme string
	pivotA     *types.Header
	trieA      *trie.Trie
	elemsA     []*kv
	header102  *types.Header
	trie102    *trie.Trie
	elems102   []*kv
	bals       map[common.Hash]rlp.RawValue
	hashX      common.Hash // account untouched by the BALs of blocks 101 and 102
	hashY      common.Hash // account moved by the BALs of blocks 101 and 102
	mkHeader   func(num uint64, root common.Hash, balHash *common.Hash) *types.Header
}

func seedCompletedSync(t *testing.T, scheme string) *reenableFixture {
	t.Helper()
	nodeScheme, sourceAccountTrie, elems, addrs := makeAccountTrieWithAddresses(100, scheme)
	rootA := sourceAccountTrie.Hash()

	db := rawdb.NewMemoryDatabase()
	emptyHash := common.Hash{}
	zero := uint64(0)
	mkHeader := func(num uint64, root common.Hash, balHash *common.Hash) *types.Header {
		header := &types.Header{
			Number: new(big.Int).SetUint64(num), Root: root, Difficulty: common.Big0,
			BaseFee: common.Big0, WithdrawalsHash: &emptyHash,
			BlobGasUsed: &zero, ExcessBlobGas: &zero,
			ParentBeaconRoot: &emptyHash, RequestsHash: &emptyHash,
			BlockAccessListHash: balHash,
		}
		rawdb.WriteHeader(db, header)
		rawdb.WriteCanonicalHash(db, header.Hash(), num)
		return header
	}
	pivotA := mkHeader(100, rootA, nil)

	// Run a sync to completion at pivot A, leaving its journal on disk.
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

	// Blocks 101 and 102 move account Y to balance 1000 and then 2000,
	// account X is left alone.
	addrY := addrs[49]
	hashY := crypto.Keccak256Hash(addrY[:])
	addrX := addrs[9]
	hashX := crypto.Keccak256Hash(addrX[:])

	bals := make(map[common.Hash]rlp.RawValue)
	// Link the parent hashes down from the pivot, the catch-up walks the
	// chain backward from the target header.
	parent := pivotA.Hash()
	mkBALHeader := func(num uint64, root common.Hash, balance uint64) *types.Header {
		cb := bal.NewConstructionBlockAccessList()
		cb.BalanceChange(0, addrY, uint256.NewInt(balance))
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
			Number: new(big.Int).SetUint64(num), ParentHash: parent, Root: root,
			Difficulty: common.Big0, BaseFee: common.Big0, WithdrawalsHash: &emptyHash,
			BlobGasUsed: &zero, ExcessBlobGas: &zero,
			ParentBeaconRoot: &emptyHash, RequestsHash: &emptyHash,
			BlockAccessListHash: &balHash,
		}
		rawdb.WriteHeader(db, header)
		rawdb.WriteCanonicalHash(db, header.Hash(), num)
		parent = header.Hash()
		bals[header.Hash()] = buf.Bytes()
		return header
	}

	// The account trie as of block 102, only Y differs from pivot A.
	trieDB := triedb.NewDatabase(rawdb.NewMemoryDatabase(), newDbConfig(scheme))
	newTrie := trie.NewEmpty(trieDB)
	elems102 := make([]*kv, len(elems))
	for i, entry := range elems {
		if bytes.Equal(entry.k, hashY[:]) {
			val, _ := rlp.EncodeToBytes(&types.StateAccount{
				Nonce: 50, Balance: uint256.NewInt(2000),
				Root: types.EmptyRootHash, CodeHash: types.EmptyCodeHash[:],
			})
			elems102[i] = &kv{entry.k, val}
		} else {
			elems102[i] = entry
		}
		newTrie.MustUpdate(elems102[i].k, elems102[i].v)
	}
	root102, nodes := newTrie.Commit(false)
	trieDB.Update(root102, types.EmptyRootHash, 0, trienode.NewWithNodeSet(nodes), triedb.NewStateSet())
	trie102, err := trie.New(trie.StateTrieID(root102), trieDB)
	if err != nil {
		t.Fatal(err)
	}
	mkBALHeader(101, common.HexToHash("0x65"), 1000)
	header102 := mkBALHeader(102, root102, 2000)

	return &reenableFixture{
		db:         db,
		nodeScheme: nodeScheme,
		pivotA:     pivotA,
		trieA:      sourceAccountTrie,
		elemsA:     elems,
		header102:  header102,
		trie102:    trie102,
		elems102:   elems102,
		bals:       bals,
		hashX:      hashX,
		hashY:      hashY,
		mkHeader:   mkHeader,
	}
}

// decodeJournal reads the journal straight from disk. Loading it through a
// fresh syncer no longer works, the constructor may drop it first.
func decodeJournal(t *testing.T, db ethdb.Database) *syncProgressV2 {
	t.Helper()
	raw := rawdb.ReadSnapshotSyncStatus(db)
	if len(raw) == 0 {
		t.Fatal("expected sync journal on disk")
	}
	if raw[0] != syncProgressVersion {
		t.Fatalf("unexpected journal version %d", raw[0])
	}
	progress := new(syncProgressV2)
	if err := json.Unmarshal(raw[1:], progress); err != nil {
		t.Fatalf("failed to decode journal: %v", err)
	}
	return progress
}

func assertAccountBalance(t *testing.T, db ethdb.Database, hash common.Hash, want uint64) {
	t.Helper()
	data := rawdb.ReadAccountSnapshot(db, hash)
	if len(data) == 0 {
		t.Fatalf("account %x missing from flat state", hash)
	}
	account, err := types.FullAccount(data)
	if err != nil {
		t.Fatalf("failed to decode account %x: %v", hash, err)
	}
	if account.Balance.Uint64() != want {
		t.Fatalf("account %x balance mismatch: got %v, want %d", hash, account.Balance, want)
	}
}

// TestReenableAfterCompletedSyncPurges reproduces the poisoned re-enable.
// A sync completed, its pivot block was committed and full sync moved the
// flat state past it, but the journal is still on disk. A new sync must
// wipe everything and download fresh. Without the purge it would roll
// forward with BALs instead, miss the rows full sync changed outside the
// BAL range and fail trie generation. Runs with the head both between the
// old pivot and the target and past it.
func TestReenableAfterCompletedSyncPurges(t *testing.T) {
	t.Parallel()
	testReenableAfterCompletedSyncPurges(t, rawdb.HashScheme, 101)
	testReenableAfterCompletedSyncPurges(t, rawdb.HashScheme, 103)
	testReenableAfterCompletedSyncPurges(t, rawdb.PathScheme, 101)
	testReenableAfterCompletedSyncPurges(t, rawdb.PathScheme, 103)
}

func testReenableAfterCompletedSyncPurges(t *testing.T, scheme string, headNum uint64) {
	fix := seedCompletedSync(t, scheme)

	// Construct the syncer before the head moves so the re-enable is
	// handled by the purge in Sync, not the constructor drop.
	syncer := newSyncerV2(fix.db, fix.nodeScheme)

	// Commit the pivot and let full sync move the head and the flat state.
	headHash := rawdb.ReadCanonicalHash(fix.db, headNum)
	if headHash == (common.Hash{}) {
		headHash = fix.mkHeader(headNum, common.HexToHash("0x67"), nil).Hash()
	}
	rawdb.WriteHeadBlockHash(fix.db, headHash)
	rawdb.WriteAccountSnapshot(fix.db, fix.hashY, types.SlimAccountRLP(types.StateAccount{
		Nonce: 50, Balance: uint256.NewInt(2000),
		Root: types.EmptyRootHash, CodeHash: types.EmptyCodeHash[:],
	}))
	rawdb.WriteAccountSnapshot(fix.db, fix.hashX, types.SlimAccountRLP(types.StateAccount{
		Nonce: 10, Balance: uint256.NewInt(9999),
		Root: types.EmptyRootHash, CodeHash: types.EmptyCodeHash[:],
	}))
	orphan := common.HexToHash("0xdeadbeef")
	rawdb.WriteAccountSnapshot(fix.db, orphan, types.SlimAccountRLP(types.StateAccount{
		Nonce: 1, Balance: uint256.NewInt(1),
		Root: types.EmptyRootHash, CodeHash: types.EmptyCodeHash[:],
	}))

	// Re-enable snap sync with a target near the old pivot.
	var (
		once   sync.Once
		cancel = make(chan struct{})
		term   = func() { once.Do(func() { close(cancel) }) }
	)
	src := newTestPeerV2("source2", t, term)
	src.accountTrie = fix.trie102.Copy()
	src.accountValues = fix.elems102
	src.accessLists = fix.bals
	syncer.Register(src)
	src.remote = syncer
	if err := syncer.Sync(fix.header102, cancel); err != nil {
		t.Fatalf("re-enabled sync failed: %v", err)
	}

	// The stale journal is gone, a fresh sync completed at the new target.
	progress := decodeJournal(t, fix.db)
	if progress.Phase != phaseComplete {
		t.Fatal("expected re-enabled sync to reach the complete phase")
	}
	if progress.Pivot == nil || progress.Pivot.Hash() != fix.header102.Hash() {
		t.Fatal("expected persisted pivot to match the new target")
	}
	// Fresh download, not catch-up. X is back to its real value and the
	// orphan row is gone.
	assertAccountBalance(t, fix.db, fix.hashX, 10)
	assertAccountBalance(t, fix.db, fix.hashY, 2000)
	if data := rawdb.ReadAccountSnapshot(fix.db, orphan); len(data) != 0 {
		t.Errorf("orphan account row should be wiped, got %x", data)
	}
}

// TestCommittedPivotRetrySkips guards the retry after a commit. If the
// cycle dies before snap mode flips to full sync, the downloader retries
// against the frozen pivot. The skip must fire before the purge so the
// pivot is recommitted instead of a good sync being wiped.
func TestCommittedPivotRetrySkips(t *testing.T) {
	t.Parallel()
	testCommittedPivotRetrySkips(t, rawdb.HashScheme)
	testCommittedPivotRetrySkips(t, rawdb.PathScheme)
}

func testCommittedPivotRetrySkips(t *testing.T, scheme string) {
	fix := seedCompletedSync(t, scheme)

	// Same process, the constructor never saw a committed journal.
	syncer := newSyncerV2(fix.db, fix.nodeScheme)

	header103 := fix.mkHeader(103, common.HexToHash("0x67"), nil)
	rawdb.WriteHeadBlockHash(fix.db, header103.Hash())

	var (
		once   sync.Once
		cancel = make(chan struct{})
		term   = func() { once.Do(func() { close(cancel) }) }
	)
	src := newTestPeerV2("source2", t, term)
	src.accountTrie = fix.trieA.Copy()
	src.accountValues = fix.elemsA
	var accountReqs atomic.Int32
	src.accountRequestV2Handler = func(tp *testPeerV2, id uint64, root common.Hash, origin common.Hash, limit common.Hash, cap int) error {
		accountReqs.Add(1)
		return defaultAccountRequestHandlerV2(tp, id, root, origin, limit, cap)
	}
	syncer.Register(src)
	src.remote = syncer
	if err := syncer.Sync(fix.pivotA, cancel); err != nil {
		t.Fatalf("retried sync failed: %v", err)
	}

	// The skip must fire, no purge and no re-download.
	if n := accountReqs.Load(); n != 0 {
		t.Fatalf("expected already-complete skip, got %d account requests", n)
	}
	assertAccountBalance(t, fix.db, fix.hashY, 50)
	progress := decodeJournal(t, fix.db)
	if progress.Phase != phaseComplete {
		t.Fatal("expected journal to stay in the complete phase")
	}
	if progress.Pivot == nil || progress.Pivot.Hash() != fix.pivotA.Hash() {
		t.Fatal("expected journal to keep the committed pivot")
	}
}

// TestCommittedPivotNotFrozen covers FrozenPivot. A pivot from a crash
// before the commit must stay frozen so the downloader resumes against it.
// Once the pivot block is committed and the head has moved on, FrozenPivot
// must report it as unfrozen so a re-enabled sync is not pinned to it. The
// journal stays on disk either way, the next Sync cleans it.
func TestCommittedPivotNotFrozen(t *testing.T) {
	t.Parallel()
	testCommittedPivotNotFrozen(t, rawdb.HashScheme)
	testCommittedPivotNotFrozen(t, rawdb.PathScheme)
}

func testCommittedPivotNotFrozen(t *testing.T, scheme string) {
	fix := seedCompletedSync(t, scheme)

	// Crash before the commit, the head never reached the pivot, so it
	// stays frozen for the resume.
	head50 := fix.mkHeader(50, common.HexToHash("0x50"), nil)
	rawdb.WriteHeadBlockHash(fix.db, head50.Hash())
	syncer := newSyncerV2(fix.db, fix.nodeScheme)
	if frozen := syncer.FrozenPivot(); frozen == nil || frozen.Hash() != fix.pivotA.Hash() {
		t.Fatal("expected the pivot to stay frozen before the commit")
	}

	// Commit the pivot and advance the head, the pivot is now a dead
	// leftover and must not be reported as frozen.
	header103 := fix.mkHeader(103, common.HexToHash("0x67"), nil)
	rawdb.WriteHeadBlockHash(fix.db, header103.Hash())
	syncer = newSyncerV2(fix.db, fix.nodeScheme)
	if frozen := syncer.FrozenPivot(); frozen != nil {
		t.Fatalf("expected the committed pivot to be unfrozen, got %v", frozen.Number)
	}
	// The journal stays on disk until a re-enabled Sync wipes it, and the
	// flat state is untouched, on a full-sync node it is the live snapshot.
	if raw := rawdb.ReadSnapshotSyncStatus(fix.db); len(raw) == 0 {
		t.Fatal("expected the journal to stay on disk")
	}
	assertAccountBalance(t, fix.db, fix.hashY, 50)
}

// TestCompletedSyncResumesBeforeCommit guards the healthy resume. The sync
// completed but died before the pivot commit, so the head never caught up
// and the flat state still matches the journal. A restart with a moved
// target must catch up cheaply, not purge and re-download.
func TestCompletedSyncResumesBeforeCommit(t *testing.T) {
	t.Parallel()
	testCompletedSyncResumesBeforeCommit(t, rawdb.HashScheme)
	testCompletedSyncResumesBeforeCommit(t, rawdb.PathScheme)
}

func testCompletedSyncResumesBeforeCommit(t *testing.T, scheme string) {
	fix := seedCompletedSync(t, scheme)

	// The head block is an old block well below the never committed pivot.
	// The head header is already past the target, as usual during a sync,
	// and must not count as a commit.
	head50 := fix.mkHeader(50, common.HexToHash("0x50"), nil)
	rawdb.WriteHeadBlockHash(fix.db, head50.Hash())
	rawdb.WriteHeadHeaderHash(fix.db, fix.header102.Hash())

	var (
		once   sync.Once
		cancel = make(chan struct{})
		term   = func() { once.Do(func() { close(cancel) }) }
	)
	syncer := newSyncerV2(fix.db, fix.nodeScheme)
	src := newTestPeerV2("source2", t, term)
	src.accountTrie = fix.trie102.Copy()
	src.accountValues = fix.elems102
	src.accessLists = fix.bals
	var accountReqs atomic.Int32
	src.accountRequestV2Handler = func(tp *testPeerV2, id uint64, root common.Hash, origin common.Hash, limit common.Hash, cap int) error {
		accountReqs.Add(1)
		return defaultAccountRequestHandlerV2(tp, id, root, origin, limit, cap)
	}
	syncer.Register(src)
	src.remote = syncer
	if err := syncer.Sync(fix.header102, cancel); err != nil {
		t.Fatalf("resumed sync failed: %v", err)
	}

	// Pure catch-up. A purge would have re-downloaded the account ranges.
	if n := accountReqs.Load(); n != 0 {
		t.Fatalf("expected pure catch-up without re-download, got %d account requests", n)
	}
	assertAccountBalance(t, fix.db, fix.hashY, 2000)
	progress := decodeJournal(t, fix.db)
	if progress.Phase != phaseComplete {
		t.Fatal("expected resumed sync to reach the complete phase")
	}
	if progress.Pivot == nil || progress.Pivot.Hash() != fix.header102.Hash() {
		t.Fatal("expected persisted pivot to match the new target")
	}
}

// TestInterruptedGenerationRecovery verifies that if sync is interrupted after
// download completes but before trie generation finishes, the next Sync() call
// re-runs the download (which completes immediately) and generation.
func TestInterruptedGenerationRecovery(t *testing.T) {
	t.Parallel()

	nodeScheme, sourceAccountTrie, elems := makeAccountTrieNoStorage(100, rawdb.HashScheme)
	root := sourceAccountTrie.Hash()

	// First run: complete download, save status, simulate interruption
	// before generation by calling downloadState() directly
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
	syncer1.loadSyncStatus()
	syncer1.pivot = pivot // Sync pins this before downloadState

	if err := syncer1.downloadState(cancel1); err != nil {
		t.Fatalf("download failed: %v", err)
	}
	// Save status (simulating what Sync's defer does)
	for _, task := range syncer1.tasks {
		syncer1.forwardAccountTask(task)
	}
	syncer1.cleanAccountTasks()
	syncer1.saveSyncStatus()

	// Status should exist (generation hasn't run yet)
	if rawdb.ReadSnapshotSyncStatus(db) == nil {
		t.Fatal("sync status should exist after download")
	}
	// Second run: full Sync should detect tasks are done, run generation
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
	// The resumed run re-arms the pivot freeze once its no-op download
	// completes, the downloader relies on it until the pivot block commits.
	if syncer2.FrozenPivot() == nil {
		t.Fatal("pivot not frozen after resumed sync")
	}
	// After generation completes, status should reach the complete phase.
	loader := newSyncerV2(db, nodeScheme)
	loader.loadSyncStatus()
	if loader.getPhase() != phaseComplete {
		t.Fatal("sync status should reach the complete phase after generation completes")
	}
}

// TestPruneStaleState verifies that resuming a sync wipes exactly the flat
// state the journal does not cover: account and storage entries beyond the
// task cursors are removed, while everything below the cursors — and the
// journaled storage of completed-storage accounts and chunked large
// contracts inside the window — survives.
func TestPruneStaleState(t *testing.T) {
	t.Parallel()

	var (
		db     = rawdb.NewMemoryDatabase()
		syncer = newSyncerV2(db, rawdb.HashScheme)

		fetched   = common.Hash{0x20} // below the cursor, journal-covered
		next      = common.Hash{0x40} // task cursor
		stale     = common.Hash{0x50} // beyond the cursor, not covered
		completed = common.Hash{0x60} // beyond the cursor, storage journaled complete
		chunked   = common.Hash{0x80} // beyond the cursor, large contract, partially journaled
		staleToo  = common.Hash{0xa0} // beyond the cursor, not covered

		subNext = common.Hash{0x50} // slot cursor of the chunked contract
		slotLo  = common.Hash{0x11} // below the slot cursor, journal-covered
		slotHi  = common.Hash{0x77} // beyond the slot cursor, not covered

		val = []byte{0xde, 0xad}
	)
	syncer.tasks = []*accountTaskV2{{
		Next: next,
		Last: common.MaxHash,
		SubTasks: map[common.Hash][]*storageTaskV2{
			chunked: {{Next: subNext, Last: common.MaxHash}},
		},
		stateCompleted: map[common.Hash]struct{}{completed: {}},
	}}

	// Journal-covered state: account below the cursor with a storage slot,
	// the completed account's storage, the chunked contract's low slots.
	rawdb.WriteAccountSnapshot(db, fetched, val)
	rawdb.WriteStorageSnapshot(db, fetched, slotLo, val)
	rawdb.WriteStorageSnapshot(db, completed, slotLo, val)
	rawdb.WriteStorageSnapshot(db, chunked, slotLo, val)

	// Uncovered state, flushed after the last journal save: accounts beyond
	// the cursor (including the completed-storage one, whose account entry
	// is only legitimate below the cursor), their storage, and the chunked
	// contract's slots beyond its slot cursor.
	rawdb.WriteAccountSnapshot(db, stale, val)
	rawdb.WriteAccountSnapshot(db, completed, val)
	rawdb.WriteAccountSnapshot(db, staleToo, val)
	rawdb.WriteStorageSnapshot(db, stale, slotLo, val)
	rawdb.WriteStorageSnapshot(db, staleToo, slotHi, val)
	rawdb.WriteStorageSnapshot(db, chunked, slotHi, val)

	// Plant bare keys right at the start of the neighboring keyspaces. The
	// task ranges end at the maximum hash, so a limit computed by bumping
	// the full key would roll the carry over into these keyspaces and
	// swallow such keys.
	neighborOfAccounts := bytes.Clone(rawdb.SnapshotAccountPrefix)
	neighborOfAccounts[len(neighborOfAccounts)-1]++
	db.Put(neighborOfAccounts, val)

	neighborOfStorage := bytes.Clone(rawdb.SnapshotStoragePrefix)
	neighborOfStorage[len(neighborOfStorage)-1]++
	db.Put(neighborOfStorage, val)

	if err := syncer.pruneStaleState(); err != nil {
		t.Fatalf("prune failed: %v", err)
	}
	for _, key := range [][]byte{neighborOfAccounts, neighborOfStorage} {
		if has, _ := db.Has(key); !has {
			t.Errorf("neighboring keyspace entry %x swallowed by the wipe", key)
		}
	}

	for _, tc := range []struct {
		account common.Hash
		slot    *common.Hash
		want    bool
		desc    string
	}{
		{fetched, nil, true, "account below cursor"},
		{fetched, &slotLo, true, "storage below cursor"},
		{completed, &slotLo, true, "storage of completed account"},
		{chunked, &slotLo, true, "chunked contract slot below slot cursor"},
		{stale, nil, false, "account beyond cursor"},
		{completed, nil, false, "account entry of completed account"},
		{staleToo, nil, false, "account beyond cursor (second)"},
		{stale, &slotLo, false, "storage of unprotected account"},
		{staleToo, &slotHi, false, "storage of unprotected account (second)"},
		{chunked, &slotHi, false, "chunked contract slot beyond slot cursor"},
	} {
		var have bool
		if tc.slot == nil {
			have = len(rawdb.ReadAccountSnapshot(db, tc.account)) > 0
		} else {
			have = len(rawdb.ReadStorageSnapshot(db, tc.account, *tc.slot)) > 0
		}
		if have != tc.want {
			t.Errorf("%s: present = %v, want %v", tc.desc, have, tc.want)
		}
	}
}

// TestResumeWipesUncoveredState simulates an unclean shutdown mid-download:
// flat state flushed beyond the journaled cursor (which the journal therefore
// considers unfetched) must be wiped when the sync resumes — otherwise it
// would survive the re-download untouched and corrupt the generated trie.
func TestResumeWipesUncoveredState(t *testing.T) {
	t.Parallel()
	testResumeWipesUncoveredState(t, rawdb.HashScheme)
	testResumeWipesUncoveredState(t, rawdb.PathScheme)
}

func testResumeWipesUncoveredState(t *testing.T, scheme string) {
	nodeScheme, sourceAccountTrie, elems := makeAccountTrieNoStorage(100, scheme)
	root := sourceAccountTrie.Hash()
	db := rawdb.NewMemoryDatabase()
	pivot := mkPivot(0, root)

	// First run: interrupt the download after a couple of responses, leaving
	// a journal behind (saved on the graceful teardown).
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
	syncer1.Sync(pivot, cancel1)

	// Simulate the unclean shutdown: plant a bogus account that the journal
	// does not cover (right beyond an unfinished task's cursor) — as if it
	// was flushed after the journal was last saved. The account is not part
	// of the source trie, so the re-download would never overwrite it and
	// trie generation would fail the root check if it survived.
	loader := newSyncerV2(db, nodeScheme)
	loader.loadSyncStatus()
	if len(loader.tasks) == 0 {
		t.Fatal("expected unfinished tasks in the journal")
	}
	task := loader.tasks[len(loader.tasks)-1]
	bogus := incHash(task.Next)
	rawdb.WriteAccountSnapshot(db, bogus, types.SlimAccountRLP(types.StateAccount{
		Nonce: 1, Balance: uint256.NewInt(1),
		Root: types.EmptyRootHash, CodeHash: types.EmptyCodeHash[:],
	}))

	// Second run: the resume must prune the bogus entry and complete cleanly.
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

	if err := syncer2.Sync(pivot, cancel2); err != nil {
		t.Fatalf("resumed sync failed: %v", err)
	}
	if len(rawdb.ReadAccountSnapshot(db, bogus)) != 0 {
		t.Fatal("uncovered account entry should have been pruned on resume")
	}
	verifyTrie(scheme, db, root, t)
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

// makeStorageTrieFromSlots builds a storage trie for owner from raw slot
// key->value pairs, using the exact on-disk encoding the flat snapshot and the
// trie generation expect: each leaf is keyed by keccak256(slotKey) and its value is
// rlp(TrimLeftZeroes(value)). Zero-valued slots are skipped (an unset slot has
// no leaf). It returns the storage root, the dirty node set, and the sorted
// snapshot leaves (which a test peer serves verbatim).
func makeStorageTrieFromSlots(db *triedb.Database, owner common.Hash, slots map[common.Hash]common.Hash) (common.Hash, *trienode.NodeSet, []*kv) {
	st, _ := trie.New(trie.StorageTrieID(types.EmptyRootHash, owner, types.EmptyRootHash), db)
	var entries []*kv
	for rawKey, value := range slots {
		if value == (common.Hash{}) {
			continue // unset slot: no leaf
		}
		slotHash := crypto.Keccak256Hash(rawKey[:])
		enc, _ := rlp.EncodeToBytes(common.TrimLeftZeroes(value[:]))
		st.MustUpdate(slotHash[:], enc)
		entries = append(entries, &kv{slotHash[:], enc})
	}
	slices.SortFunc(entries, (*kv).cmp)
	root, nodes := st.Commit(false)
	return root, nodes, entries
}

// makeStateWithStorageContract builds an account trie holding the given
// storage-less accounts plus a single contract account whose storage trie is
// built from slots. Everything is committed into a fresh triedb so the tries
// can be served by a test peer. It returns the recreated account trie, the
// sorted account leaves, the recreated contract storage trie, the sorted
// storage leaves, and the resulting state root.
func makeStateWithStorageContract(scheme string, plain []*kv, contractAddr common.Address, contract types.StateAccount, slots map[common.Hash]common.Hash) (*trie.Trie, []*kv, *trie.Trie, []*kv, common.Hash) {
	db := triedb.NewDatabase(rawdb.NewMemoryDatabase(), newDbConfig(scheme))
	accTrie := trie.NewEmpty(db)
	merged := trienode.NewMergedNodeSet()

	// Contract storage trie.
	contractHash := crypto.Keccak256Hash(contractAddr[:])
	stRoot, stNodes, stEntries := makeStorageTrieFromSlots(db, contractHash, slots)
	if stNodes != nil {
		merged.Merge(stNodes)
	}

	// Contract account leaf carries the (live) storage root.
	contract.Root = stRoot
	cval, _ := rlp.EncodeToBytes(&contract)
	accTrie.MustUpdate(contractHash[:], cval)
	accEntries := []*kv{{contractHash[:], cval}}

	// Storage-less filler accounts.
	for _, e := range plain {
		accTrie.MustUpdate(e.k, e.v)
		accEntries = append(accEntries, &kv{e.k, e.v})
	}
	slices.SortFunc(accEntries, (*kv).cmp)

	// Commit account + storage nodes together, then re-open for serving.
	root, set := accTrie.Commit(true)
	merged.Merge(set)
	db.Update(root, types.EmptyRootHash, 0, merged, triedb.NewStateSet())

	accTrie, _ = trie.New(trie.StateTrieID(root), db)
	stTrie, _ := trie.New(trie.StorageTrieID(root, contractHash, stRoot), db)
	return accTrie, accEntries, stTrie, stEntries, root
}

// TestCatchUpAppliesStorageBALs exercises the snap/2 catch-up path with a BAL
// that mutates storage slots (not just balances): a non-zero write to a fresh
// slot, an overwrite of an existing slot, a write of zero (deletion), and a
// multi-tx write where the post-block value wins.
//
// It fully syncs pivot A (flat-state download + trie generation), then moves the
// pivot to A+1. The move triggers catchUp, which fetches the A+1 BAL, applies
// the storage diffs to the flat state, and generates the trie. The generation
// verifies the recomputed root against the pivot's expected post-catch-up root,
// so a successful Sync proves the storage mutations were applied in the exact
// encoding the trie generation consumes. verifyTrie re-walks the result as an
// independent confirmation.
func TestCatchUpAppliesStorageBALs(t *testing.T) {
	t.Parallel()
	testCatchUpAppliesStorageBALs(t, rawdb.HashScheme)
	testCatchUpAppliesStorageBALs(t, rawdb.PathScheme)
}

func testCatchUpAppliesStorageBALs(t *testing.T, scheme string) {
	// The contract whose storage the A+1 BAL mutates.
	contractAddr := common.HexToAddress("0x00000000000000000000000000000000c0ffee01")
	contractHash := crypto.Keccak256Hash(contractAddr[:])

	// Raw storage slot keys.
	var (
		slotKeep    = common.HexToHash("0x01") // untouched by the BAL
		slotOver    = common.HexToHash("0x02") // overwritten with a new non-zero value
		slotZero    = common.HexToHash("0x03") // written to zero (deletion)
		slotNew     = common.HexToHash("0x04") // unset in A, written non-zero in A+1
		slotMultiTx = common.HexToHash("0x05") // written several times within the block
	)
	// Slot values. Multi-byte values force RLP length prefixes, so the encoding
	// differs sharply from the raw 32-byte form and a format mismatch surfaces.
	var (
		vKeep       = common.HexToHash("0x1111")
		vOver0      = common.HexToHash("0x2222")
		vOver1      = common.HexToHash("0x22220000aaaa")
		vZero0      = common.HexToHash("0x3333")
		vNew        = common.HexToHash("0x4444")
		vMulti0     = common.HexToHash("0x5555")
		vMultiMid   = common.HexToHash("0x5556")
		vMultiFinal = common.HexToHash("0x55570000bbbb")
	)
	// Storage at pivot A.
	slotsA := map[common.Hash]common.Hash{
		slotKeep:    vKeep,
		slotOver:    vOver0,
		slotZero:    vZero0,
		slotMultiTx: vMulti0,
	}
	// Expected storage at pivot A+1 after applying the BAL writes below.
	slotsB := map[common.Hash]common.Hash{
		slotKeep:    vKeep,       // unchanged
		slotOver:    vOver1,      // overwritten
		slotNew:     vNew,        // newly written
		slotMultiTx: vMultiFinal, // post-block (highest-tx) value wins
		// slotZero deleted
	}
	contractTmpl := types.StateAccount{
		Nonce:    7,
		Balance:  uint256.NewInt(123456),
		CodeHash: types.EmptyCodeHash[:],
	}

	// Storage-less filler accounts, identical in A and A+1.
	_, _, plain, _ := makeAccountTrieWithAddresses(20, scheme)

	// Build the state at pivot A (served by the seed peer) and the expected
	// state at pivot A+1 (only its root is needed).
	accTrieA, accElemsA, stTrieA, stElemsA, rootA := makeStateWithStorageContract(scheme, plain, contractAddr, contractTmpl, slotsA)
	_, _, _, _, rootB := makeStateWithStorageContract(scheme, plain, contractAddr, contractTmpl, slotsB)
	if rootA == rootB {
		t.Fatal("test bug: pivot A and A+1 must have different state roots")
	}

	// Build the A+1 BAL describing the storage mutations.
	cb := bal.NewConstructionBlockAccessList()
	cb.StorageWrite(0, contractAddr, slotOver, vOver1)         // overwrite
	cb.StorageWrite(0, contractAddr, slotZero, common.Hash{})  // write zero -> delete
	cb.StorageWrite(0, contractAddr, slotNew, vNew)            // new non-zero
	cb.StorageWrite(0, contractAddr, slotMultiTx, vMultiMid)   // tx 0
	cb.StorageWrite(2, contractAddr, slotMultiTx, vMultiFinal) // tx 2 (post-block)
	var balBuf bytes.Buffer
	if err := cb.EncodeRLP(&balBuf); err != nil {
		t.Fatal(err)
	}
	var decodedBAL bal.BlockAccessList
	if err := rlp.DecodeBytes(balBuf.Bytes(), &decodedBAL); err != nil {
		t.Fatal(err)
	}
	balHash := decodedBAL.Hash()

	// Chain headers. The pivot-A header is the same object passed to the first
	// Sync, so the follow-up Sync's reorg check sees A as still-canonical and
	// runs catchUp instead of resetting. The A+1 header carries the BAL hash
	// (verified during catch-up) and the expected post-catch-up state root
	// (verified by the trie generation).
	db := rawdb.NewMemoryDatabase()
	numA := uint64(128)
	emptyH := common.Hash{}
	zero := uint64(0)
	hdrA := &types.Header{
		Number: new(big.Int).SetUint64(numA), Root: rootA, Difficulty: common.Big0,
		BaseFee: common.Big0, WithdrawalsHash: &emptyH,
		BlobGasUsed: &zero, ExcessBlobGas: &zero,
		ParentBeaconRoot: &emptyH, RequestsHash: &emptyH,
	}
	rawdb.WriteHeader(db, hdrA)
	rawdb.WriteCanonicalHash(db, hdrA.Hash(), numA)

	hdrB := &types.Header{
		ParentHash: hdrA.Hash(),
		Number:     new(big.Int).SetUint64(numA + 1), Root: rootB, Difficulty: common.Big0,
		BaseFee: common.Big0, WithdrawalsHash: &emptyH,
		BlobGasUsed: &zero, ExcessBlobGas: &zero,
		ParentBeaconRoot: &emptyH, RequestsHash: &emptyH,
		BlockAccessListHash: &balHash,
	}
	rawdb.WriteHeader(db, hdrB)
	rawdb.WriteCanonicalHash(db, hdrB.Hash(), numA+1)

	// Sync 1: full flat-state download + trie generation against pivot A.
	{
		var (
			once   sync.Once
			cancel = make(chan struct{})
			term   = func() { once.Do(func() { close(cancel) }) }
		)
		syncer := newSyncerV2(db, scheme)
		src := newTestPeerV2("seed", t, term)
		src.accountTrie = accTrieA.Copy()
		src.accountValues = accElemsA
		src.setStorageTries(map[common.Hash]*trie.Trie{contractHash: stTrieA})
		src.storageValues = map[common.Hash][]*kv{contractHash: stElemsA}
		syncer.Register(src)
		src.remote = syncer
		done := checkStall(t, term)
		if err := syncer.Sync(hdrA, cancel); err != nil {
			t.Fatalf("pivot A sync failed: %v", err)
		}
		close(done)
	}
	// Sanity: the generated trie for pivot A is complete and matches rootA. This
	// also confirms the test fixture itself is internally consistent.
	verifyTrie(scheme, db, rootA, t)

	// Sync 2: the pivot moves to A+1, exercising the BAL catch-up path.
	{
		var (
			once   sync.Once
			cancel = make(chan struct{})
			term   = func() { once.Do(func() { close(cancel) }) }
		)
		syncer := newSyncerV2(db, scheme)
		src := newTestPeerV2("catchup", t, term)
		// Pivot A is fully synced, so no download tasks remain; the peer only
		// needs to serve the A+1 BAL. The trie data is provided defensively in
		// case a stray account request is issued.
		src.accountTrie = accTrieA.Copy()
		src.accountValues = accElemsA
		src.accessLists = map[common.Hash]rlp.RawValue{hdrB.Hash(): balBuf.Bytes()}
		syncer.Register(src)
		src.remote = syncer
		done := checkStall(t, term)
		if err := syncer.Sync(hdrB, cancel); err != nil {
			t.Fatalf("pivot A+1 catch-up sync failed: %v", err)
		}
		// The freeze must re-arm on a pivot-moved cycle too, the downloader
		// relies on it from download completion until commit, and it must
		// point at the new pivot the catch-up rolled forward to.
		if frozen := syncer.FrozenPivot(); frozen == nil || frozen.Hash() != hdrB.Hash() {
			t.Fatal("pivot not frozen at the new header after catch-up sync")
		}
		close(done)
	}

	// A successful Sync already means GenerateTrie reproduced rootB from the
	// BAL-updated flat state (it errors on root mismatch). Re-walk the trie as
	// an independent confirmation that rootB is fully materialized.
	verifyTrie(scheme, db, rootB, t)

	// Spot-check each storage mutation landed in the flat snapshot in the
	// canonical encoding.
	checkSlot := func(raw common.Hash, want common.Hash, present bool) {
		t.Helper()
		got := rawdb.ReadStorageSnapshot(db, contractHash, crypto.Keccak256Hash(raw[:]))
		if !present {
			if len(got) != 0 {
				t.Errorf("slot %x: expected deletion, got %x", raw, got)
			}
			return
		}
		wantEnc, _ := rlp.EncodeToBytes(common.TrimLeftZeroes(want[:]))
		if !bytes.Equal(got, wantEnc) {
			t.Errorf("slot %x: got %x, want %x", raw, got, wantEnc)
		}
	}
	checkSlot(slotKeep, vKeep, true)
	checkSlot(slotOver, vOver1, true)
	checkSlot(slotZero, common.Hash{}, false)
	checkSlot(slotNew, vNew, true)
	checkSlot(slotMultiTx, vMultiFinal, true)
}
