// Copyright 2021 The go-ethereum Authors
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
	"crypto/rand"
	"encoding/binary"
	"fmt"
	"math/big"
	mrand "math/rand"
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
	"github.com/ethereum/go-ethereum/crypto/keccak"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/internal/testrand"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/ethereum/go-ethereum/trie"
	"github.com/ethereum/go-ethereum/trie/trienode"
	"github.com/ethereum/go-ethereum/triedb"
	"github.com/ethereum/go-ethereum/triedb/pathdb"
	"github.com/holiman/uint256"
)

func TestHashing(t *testing.T) {
	t.Parallel()

	var bytecodes = make([][]byte, 10)
	for i := 0; i < len(bytecodes); i++ {
		buf := make([]byte, 100)
		rand.Read(buf)
		bytecodes[i] = buf
	}
	var want, got string
	var old = func() {
		hasher := keccak.NewLegacyKeccak256()
		for i := 0; i < len(bytecodes); i++ {
			hasher.Reset()
			hasher.Write(bytecodes[i])
			hash := hasher.Sum(nil)
			got = fmt.Sprintf("%v\n%v", got, hash)
		}
	}
	var new = func() {
		hasher := crypto.NewKeccakState()
		var hash = make([]byte, 32)
		for i := 0; i < len(bytecodes); i++ {
			hasher.Reset()
			hasher.Write(bytecodes[i])
			hasher.Read(hash)
			want = fmt.Sprintf("%v\n%v", want, hash)
		}
	}
	old()
	new()
	if want != got {
		t.Errorf("want\n%v\ngot\n%v\n", want, got)
	}
}

func BenchmarkHashing(b *testing.B) {
	var bytecodes = make([][]byte, 10000)
	for i := 0; i < len(bytecodes); i++ {
		buf := make([]byte, 100)
		rand.Read(buf)
		bytecodes[i] = buf
	}
	var old = func() {
		hasher := keccak.NewLegacyKeccak256()
		for i := 0; i < len(bytecodes); i++ {
			hasher.Reset()
			hasher.Write(bytecodes[i])
			hasher.Sum(nil)
		}
	}
	var new = func() {
		hasher := crypto.NewKeccakState()
		var hash = make([]byte, 32)
		for i := 0; i < len(bytecodes); i++ {
			hasher.Reset()
			hasher.Write(bytecodes[i])
			hasher.Read(hash)
		}
	}
	b.Run("old", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			old()
		}
	})
	b.Run("new", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			new()
		}
	})
}

type (
	accountHandlerFunc    func(t *testPeer, requestId uint64, root common.Hash, origin common.Hash, limit common.Hash, cap int) error
	storageHandlerFunc    func(t *testPeer, requestId uint64, root common.Hash, accounts []common.Hash, origin, limit []byte, max int) error
	codeHandlerFunc       func(t *testPeer, id uint64, hashes []common.Hash, max int) error
	accessListHandlerFunc func(t *testPeer, id uint64, hashes []common.Hash, max int) error
)

type testPeer struct {
	id            string
	test          *testing.T
	remote        *Syncer
	logger        log.Logger
	accountTrie   *trie.Trie
	accountValues []*kv
	storageTries  map[common.Hash]*trie.Trie
	storageValues map[common.Hash][]*kv
	accessLists   map[common.Hash]rlp.RawValue // block hash -> RLP-encoded BAL

	accountRequestHandler    accountHandlerFunc
	storageRequestHandler    storageHandlerFunc
	codeRequestHandler       codeHandlerFunc
	accessListRequestHandler accessListHandlerFunc
	term                     func()

	// counters
	nAccountRequests    atomic.Int64
	nStorageRequests    atomic.Int64
	nBytecodeRequests   atomic.Int64
	nAccessListRequests atomic.Int64
}

func newTestPeer(id string, t *testing.T, term func()) *testPeer {
	peer := &testPeer{
		id:                       id,
		test:                     t,
		logger:                   log.New("id", id),
		accountRequestHandler:    defaultAccountRequestHandler,
		storageRequestHandler:    defaultStorageRequestHandler,
		codeRequestHandler:       defaultCodeRequestHandler,
		accessListRequestHandler: defaultAccessListRequestHandler,
		term:                     term,
	}
	//stderrHandler := log.StreamHandler(os.Stderr, log.TerminalFormat(true))
	//peer.logger.SetHandler(stderrHandler)
	return peer
}

func (t *testPeer) setStorageTries(tries map[common.Hash]*trie.Trie) {
	t.storageTries = make(map[common.Hash]*trie.Trie)
	for root, trie := range tries {
		t.storageTries[root] = trie.Copy()
	}
}

func (t *testPeer) ID() string      { return t.id }
func (t *testPeer) Log() log.Logger { return t.logger }

func (t *testPeer) Stats() string {
	return fmt.Sprintf(`Account requests: %d Storage requests: %d Bytecode requests: %d`, t.nAccountRequests, t.nStorageRequests, t.nBytecodeRequests)
}

func (t *testPeer) RequestAccountRange(id uint64, root, origin, limit common.Hash, bytes int) error {
	t.logger.Trace("Fetching range of accounts", "reqid", id, "root", root, "origin", origin, "limit", limit, "bytes", common.StorageSize(bytes))
	t.nAccountRequests.Add(1)
	go t.accountRequestHandler(t, id, root, origin, limit, bytes)
	return nil
}

func (t *testPeer) RequestStorageRanges(id uint64, root common.Hash, accounts []common.Hash, origin, limit []byte, bytes int) error {
	t.nStorageRequests.Add(1)
	if len(accounts) == 1 && origin != nil {
		t.logger.Trace("Fetching range of large storage slots", "reqid", id, "root", root, "account", accounts[0], "origin", common.BytesToHash(origin), "limit", common.BytesToHash(limit), "bytes", common.StorageSize(bytes))
	} else {
		t.logger.Trace("Fetching ranges of small storage slots", "reqid", id, "root", root, "accounts", len(accounts), "first", accounts[0], "bytes", common.StorageSize(bytes))
	}
	go t.storageRequestHandler(t, id, root, accounts, origin, limit, bytes)
	return nil
}

func (t *testPeer) RequestByteCodes(id uint64, hashes []common.Hash, bytes int) error {
	t.nBytecodeRequests.Add(1)
	t.logger.Trace("Fetching set of byte codes", "reqid", id, "hashes", len(hashes), "bytes", common.StorageSize(bytes))
	go t.codeRequestHandler(t, id, hashes, bytes)
	return nil
}

func (t *testPeer) RequestAccessLists(id uint64, hashes []common.Hash, bytes int) error {
	t.nAccessListRequests++
	t.logger.Trace("Fetching set of BALs", "reqid", id, "hashes", len(hashes), "bytes", common.StorageSize(bytes))
	go t.accessListRequestHandler(t, id, hashes, bytes)
	return nil
}

// defaultTrieRequestHandler is a well-behaving handler for trie healing requests
// defaultAccountRequestHandler is a well-behaving handler for AccountRangeRequests
func defaultAccountRequestHandler(t *testPeer, id uint64, root common.Hash, origin common.Hash, limit common.Hash, cap int) error {
	keys, vals, proofs := createAccountRequestResponse(t, root, origin, limit, cap)
	if err := t.remote.OnAccounts(t, id, keys, vals, proofs); err != nil {
		t.test.Errorf("Remote side rejected our delivery: %v", err)
		t.term()
		return err
	}
	return nil
}

func createAccountRequestResponse(t *testPeer, root common.Hash, origin common.Hash, limit common.Hash, cap int) (keys []common.Hash, vals [][]byte, proofs [][]byte) {
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
		// If we've exceeded the request threshold, abort
		if bytes.Compare(entry.k, limit[:]) >= 0 {
			break
		}
	}
	// Unless we send the entire trie, we need to supply proofs
	// Actually, we need to supply proofs either way! This seems to be an implementation
	// quirk in go-ethereum
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

// defaultStorageRequestHandler is a well-behaving storage request handler
func defaultStorageRequestHandler(t *testPeer, requestId uint64, root common.Hash, accounts []common.Hash, bOrigin, bLimit []byte, max int) error {
	hashes, slots, proofs := createStorageRequestResponse(t, root, accounts, bOrigin, bLimit, max)
	if err := t.remote.OnStorage(t, requestId, hashes, slots, proofs); err != nil {
		t.test.Errorf("Remote side rejected our delivery: %v", err)
		t.term()
	}
	return nil
}

func defaultCodeRequestHandler(t *testPeer, id uint64, hashes []common.Hash, max int) error {
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
func defaultAccessListRequestHandler(t *testPeer, id uint64, hashes []common.Hash, max int) error {
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

func createStorageRequestResponse(t *testPeer, root common.Hash, accounts []common.Hash, origin, limit []byte, max int) (hashes [][]common.Hash, slots [][][]byte, proofs [][]byte) {
	var size int
	for _, account := range accounts {
		// The first account might start from a different origin and end sooner
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
		// Generate the Merkle proofs for the first and last storage slot, but
		// only if the response was capped. If the entire storage trie included
		// in the response, no need for any proofs.
		if originHash != (common.Hash{}) || (abort && len(keys) > 0) {
			// If we're aborting, we need to prove the first and last item
			// This terminates the response (and thus the loop)
			proof := trienode.NewProofSet()
			stTrie := t.storageTries[account]

			// Here's a potential gotcha: when constructing the proof, we cannot
			// use the 'origin' slice directly, but must use the full 32-byte
			// hash form.
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

// createStorageRequestResponseAlwaysProve tests a cornercase, where the peer always
// supplies the proof for the last account, even if it is 'complete'.
func createStorageRequestResponseAlwaysProve(t *testPeer, root common.Hash, accounts []common.Hash, bOrigin, bLimit []byte, max int) (hashes [][]common.Hash, slots [][][]byte, proofs [][]byte) {
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
			// If we're aborting, we need to prove the first and last item
			// This terminates the response (and thus the loop)
			proof := trienode.NewProofSet()
			stTrie := t.storageTries[account]

			// Here's a potential gotcha: when constructing the proof, we cannot
			// use the 'origin' slice directly, but must use the full 32-byte
			// hash form.
			if err := stTrie.Prove(origin[:], proof); err != nil {
				t.logger.Error("Could not prove inexistence of origin", "origin", origin,
					"error", err)
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

// emptyRequestAccountRangeFn is a rejects AccountRangeRequests
func emptyRequestAccountRangeFn(t *testPeer, requestId uint64, root common.Hash, origin common.Hash, limit common.Hash, cap int) error {
	t.remote.OnAccounts(t, requestId, nil, nil, nil)
	return nil
}

func nonResponsiveRequestAccountRangeFn(t *testPeer, requestId uint64, root common.Hash, origin common.Hash, limit common.Hash, cap int) error {
	return nil
}

func emptyStorageRequestHandler(t *testPeer, requestId uint64, root common.Hash, accounts []common.Hash, origin, limit []byte, max int) error {
	t.remote.OnStorage(t, requestId, nil, nil, nil)
	return nil
}

func nonResponsiveStorageRequestHandler(t *testPeer, requestId uint64, root common.Hash, accounts []common.Hash, origin, limit []byte, max int) error {
	return nil
}

func proofHappyStorageRequestHandler(t *testPeer, requestId uint64, root common.Hash, accounts []common.Hash, origin, limit []byte, max int) error {
	hashes, slots, proofs := createStorageRequestResponseAlwaysProve(t, root, accounts, origin, limit, max)
	if err := t.remote.OnStorage(t, requestId, hashes, slots, proofs); err != nil {
		t.test.Errorf("Remote side rejected our delivery: %v", err)
		t.term()
	}
	return nil
}

func corruptCodeRequestHandler(t *testPeer, id uint64, hashes []common.Hash, max int) error {
	var bytecodes [][]byte
	for _, h := range hashes {
		// Send back the hashes
		bytecodes = append(bytecodes, h[:])
	}
	if err := t.remote.OnByteCodes(t, id, bytecodes); err != nil {
		t.logger.Info("remote error on delivery (as expected)", "error", err)
		// Mimic the real-life handler, which drops a peer on errors
		t.remote.Unregister(t.id)
	}
	return nil
}

func cappedCodeRequestHandler(t *testPeer, id uint64, hashes []common.Hash, max int) error {
	var bytecodes [][]byte
	for _, h := range hashes[:1] {
		bytecodes = append(bytecodes, getCodeByHash(h))
	}
	// Missing bytecode can be retrieved again, no error expected
	if err := t.remote.OnByteCodes(t, id, bytecodes); err != nil {
		t.test.Errorf("Remote side rejected our delivery: %v", err)
		t.term()
	}
	return nil
}

// starvingStorageRequestHandler is somewhat well-behaving storage handler, but it caps the returned results to be very small
func starvingStorageRequestHandler(t *testPeer, requestId uint64, root common.Hash, accounts []common.Hash, origin, limit []byte, max int) error {
	return defaultStorageRequestHandler(t, requestId, root, accounts, origin, limit, 500)
}

func starvingAccountRequestHandler(t *testPeer, requestId uint64, root common.Hash, origin common.Hash, limit common.Hash, cap int) error {
	return defaultAccountRequestHandler(t, requestId, root, origin, limit, 500)
}

//func misdeliveringAccountRequestHandler(t *testPeer, requestId uint64, root common.Hash, origin common.Hash, cap uint64) error {
//	return defaultAccountRequestHandler(t, requestId-1, root, origin, 500)
//}

func corruptAccountRequestHandler(t *testPeer, requestId uint64, root common.Hash, origin common.Hash, limit common.Hash, cap int) error {
	hashes, accounts, proofs := createAccountRequestResponse(t, root, origin, limit, cap)
	if len(proofs) > 0 {
		proofs = proofs[1:]
	}
	if err := t.remote.OnAccounts(t, requestId, hashes, accounts, proofs); err != nil {
		t.logger.Info("remote error on delivery (as expected)", "error", err)
		// Mimic the real-life handler, which drops a peer on errors
		t.remote.Unregister(t.id)
	}
	return nil
}

// corruptStorageRequestHandler doesn't provide good proofs
func corruptStorageRequestHandler(t *testPeer, requestId uint64, root common.Hash, accounts []common.Hash, origin, limit []byte, max int) error {
	hashes, slots, proofs := createStorageRequestResponse(t, root, accounts, origin, limit, max)
	if len(proofs) > 0 {
		proofs = proofs[1:]
	}
	if err := t.remote.OnStorage(t, requestId, hashes, slots, proofs); err != nil {
		t.logger.Info("remote error on delivery (as expected)", "error", err)
		// Mimic the real-life handler, which drops a peer on errors
		t.remote.Unregister(t.id)
	}
	return nil
}

func noProofStorageRequestHandler(t *testPeer, requestId uint64, root common.Hash, accounts []common.Hash, origin, limit []byte, max int) error {
	hashes, slots, _ := createStorageRequestResponse(t, root, accounts, origin, limit, max)
	if err := t.remote.OnStorage(t, requestId, hashes, slots, nil); err != nil {
		t.logger.Info("remote error on delivery (as expected)", "error", err)
		// Mimic the real-life handler, which drops a peer on errors
		t.remote.Unregister(t.id)
	}
	return nil
}

// TestSyncBloatedProof tests a scenario where we provide only _one_ value, but
// also ship the entire trie inside the proof. If the attack is successful,
// the remote side does not do any follow-up requests
func TestSyncBloatedProof(t *testing.T) {
	t.Parallel()

	testSyncBloatedProof(t, rawdb.HashScheme)
	testSyncBloatedProof(t, rawdb.PathScheme)
}

func testSyncBloatedProof(t *testing.T, scheme string) {
	var (
		once   sync.Once
		cancel = make(chan struct{})
		term   = func() {
			once.Do(func() {
				close(cancel)
			})
		}
	)
	nodeScheme, sourceAccountTrie, elems := makeAccountTrieNoStorage(100, scheme)
	source := newTestPeer("source", t, term)
	source.accountTrie = sourceAccountTrie.Copy()
	source.accountValues = elems

	source.accountRequestHandler = func(t *testPeer, requestId uint64, root common.Hash, origin common.Hash, limit common.Hash, cap int) error {
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
	syncer := setupSyncer(nodeScheme, source)
	if err := syncer.Sync(sourceAccountTrie.Hash(), 0, cancel); err == nil {
		t.Fatal("No error returned from incomplete/cancelled sync")
	}
}

func setupSyncer(scheme string, peers ...*testPeer) *Syncer {
	stateDb := rawdb.NewMemoryDatabase()
	syncer := NewSyncer(stateDb, scheme)
	for _, peer := range peers {
		syncer.Register(peer)
		peer.remote = syncer
	}
	return syncer
}

// TestSync tests a basic sync with one peer
func TestSync(t *testing.T) {
	t.Parallel()

	testSync(t, rawdb.HashScheme)
	testSync(t, rawdb.PathScheme)
}

func testSync(t *testing.T, scheme string) {
	var (
		once   sync.Once
		cancel = make(chan struct{})
		term   = func() {
			once.Do(func() {
				close(cancel)
			})
		}
	)
	nodeScheme, sourceAccountTrie, elems := makeAccountTrieNoStorage(100, scheme)

	mkSource := func(name string) *testPeer {
		source := newTestPeer(name, t, term)
		source.accountTrie = sourceAccountTrie.Copy()
		source.accountValues = elems
		return source
	}
	syncer := setupSyncer(nodeScheme, mkSource("source"))
	if err := syncer.Sync(sourceAccountTrie.Hash(), 0, cancel); err != nil {
		t.Fatalf("sync failed: %v", err)
	}
	verifyTrie(scheme, syncer.db, sourceAccountTrie.Hash(), t)
}

// TestSyncTinyTriePanic tests a basic sync with one peer, and a tiny trie. This caused a
// panic within the prover
func TestSyncTinyTriePanic(t *testing.T) {
	t.Parallel()

	testSyncTinyTriePanic(t, rawdb.HashScheme)
	testSyncTinyTriePanic(t, rawdb.PathScheme)
}

func testSyncTinyTriePanic(t *testing.T, scheme string) {
	var (
		once   sync.Once
		cancel = make(chan struct{})
		term   = func() {
			once.Do(func() {
				close(cancel)
			})
		}
	)
	nodeScheme, sourceAccountTrie, elems := makeAccountTrieNoStorage(1, scheme)

	mkSource := func(name string) *testPeer {
		source := newTestPeer(name, t, term)
		source.accountTrie = sourceAccountTrie.Copy()
		source.accountValues = elems
		return source
	}
	syncer := setupSyncer(nodeScheme, mkSource("source"))
	done := checkStall(t, term)
	if err := syncer.Sync(sourceAccountTrie.Hash(), 0, cancel); err != nil {
		t.Fatalf("sync failed: %v", err)
	}
	close(done)
	verifyTrie(scheme, syncer.db, sourceAccountTrie.Hash(), t)
}

// TestMultiSync tests a basic sync with multiple peers
func TestMultiSync(t *testing.T) {
	t.Parallel()

	testMultiSync(t, rawdb.HashScheme)
	testMultiSync(t, rawdb.PathScheme)
}

func testMultiSync(t *testing.T, scheme string) {
	var (
		once   sync.Once
		cancel = make(chan struct{})
		term   = func() {
			once.Do(func() {
				close(cancel)
			})
		}
	)
	nodeScheme, sourceAccountTrie, elems := makeAccountTrieNoStorage(100, scheme)

	mkSource := func(name string) *testPeer {
		source := newTestPeer(name, t, term)
		source.accountTrie = sourceAccountTrie.Copy()
		source.accountValues = elems
		return source
	}
	syncer := setupSyncer(nodeScheme, mkSource("sourceA"), mkSource("sourceB"))
	done := checkStall(t, term)
	if err := syncer.Sync(sourceAccountTrie.Hash(), 0, cancel); err != nil {
		t.Fatalf("sync failed: %v", err)
	}
	close(done)
	verifyTrie(scheme, syncer.db, sourceAccountTrie.Hash(), t)
}

// TestSyncWithStorage tests  basic sync using accounts + storage + code
func TestSyncWithStorage(t *testing.T) {
	t.Parallel()

	testSyncWithStorage(t, rawdb.HashScheme)
	testSyncWithStorage(t, rawdb.PathScheme)
}

func testSyncWithStorage(t *testing.T, scheme string) {
	var (
		once   sync.Once
		cancel = make(chan struct{})
		term   = func() {
			once.Do(func() {
				close(cancel)
			})
		}
	)
	sourceAccountTrie, elems, storageTries, storageElems := makeAccountTrieWithStorage(scheme, 3, 3000, true, false, false)

	mkSource := func(name string) *testPeer {
		source := newTestPeer(name, t, term)
		source.accountTrie = sourceAccountTrie.Copy()
		source.accountValues = elems
		source.setStorageTries(storageTries)
		source.storageValues = storageElems
		return source
	}
	syncer := setupSyncer(scheme, mkSource("sourceA"))
	done := checkStall(t, term)
	if err := syncer.Sync(sourceAccountTrie.Hash(), 0, cancel); err != nil {
		t.Fatalf("sync failed: %v", err)
	}
	close(done)
	verifyTrie(scheme, syncer.db, sourceAccountTrie.Hash(), t)
}

// TestMultiSyncManyUseless contains one good peer, and many which doesn't return anything valuable at all
func TestMultiSyncManyUseless(t *testing.T) {
	t.Parallel()

	testMultiSyncManyUseless(t, rawdb.HashScheme)
	testMultiSyncManyUseless(t, rawdb.PathScheme)
}

func testMultiSyncManyUseless(t *testing.T, scheme string) {
	var (
		once   sync.Once
		cancel = make(chan struct{})
		term   = func() {
			once.Do(func() {
				close(cancel)
			})
		}
	)
	sourceAccountTrie, elems, storageTries, storageElems := makeAccountTrieWithStorage(scheme, 100, 3000, true, false, false)

	mkSource := func(name string, noAccount, noStorage bool) *testPeer {
		source := newTestPeer(name, t, term)
		source.accountTrie = sourceAccountTrie.Copy()
		source.accountValues = elems
		source.setStorageTries(storageTries)
		source.storageValues = storageElems

		if !noAccount {
			source.accountRequestHandler = emptyRequestAccountRangeFn
		}
		if !noStorage {
			source.storageRequestHandler = emptyStorageRequestHandler
		}
		return source
	}

	syncer := setupSyncer(
		scheme,
		mkSource("full", true, true),
		mkSource("noAccounts", false, true),
		mkSource("noStorage", true, false),
	)
	done := checkStall(t, term)
	if err := syncer.Sync(sourceAccountTrie.Hash(), 0, cancel); err != nil {
		t.Fatalf("sync failed: %v", err)
	}
	close(done)
	verifyTrie(scheme, syncer.db, sourceAccountTrie.Hash(), t)
}

// TestMultiSyncManyUselessWithLowTimeout contains one good peer, and many which doesn't return anything valuable at all
func TestMultiSyncManyUselessWithLowTimeout(t *testing.T) {
	t.Parallel()

	testMultiSyncManyUselessWithLowTimeout(t, rawdb.HashScheme)
	testMultiSyncManyUselessWithLowTimeout(t, rawdb.PathScheme)
}

func testMultiSyncManyUselessWithLowTimeout(t *testing.T, scheme string) {
	var (
		once   sync.Once
		cancel = make(chan struct{})
		term   = func() {
			once.Do(func() {
				close(cancel)
			})
		}
	)
	sourceAccountTrie, elems, storageTries, storageElems := makeAccountTrieWithStorage(scheme, 100, 3000, true, false, false)

	mkSource := func(name string, noAccount, noStorage bool) *testPeer {
		source := newTestPeer(name, t, term)
		source.accountTrie = sourceAccountTrie.Copy()
		source.accountValues = elems
		source.setStorageTries(storageTries)
		source.storageValues = storageElems

		if !noAccount {
			source.accountRequestHandler = emptyRequestAccountRangeFn
		}
		if !noStorage {
			source.storageRequestHandler = emptyStorageRequestHandler
		}
		return source
	}

	syncer := setupSyncer(
		scheme,
		mkSource("full", true, true),
		mkSource("noAccounts", false, true),
		mkSource("noStorage", true, false),
	)
	// We're setting the timeout to very low, to increase the chance of the timeout
	// being triggered. This was previously a cause of panic, when a response
	// arrived simultaneously as a timeout was triggered.
	syncer.rates.OverrideTTLLimit = time.Millisecond

	done := checkStall(t, term)
	if err := syncer.Sync(sourceAccountTrie.Hash(), 0, cancel); err != nil {
		t.Fatalf("sync failed: %v", err)
	}
	close(done)
	verifyTrie(scheme, syncer.db, sourceAccountTrie.Hash(), t)
}

// TestMultiSyncManyUnresponsive contains one good peer, and many which doesn't respond at all
func TestMultiSyncManyUnresponsive(t *testing.T) {
	t.Parallel()

	testMultiSyncManyUnresponsive(t, rawdb.HashScheme)
	testMultiSyncManyUnresponsive(t, rawdb.PathScheme)
}

func testMultiSyncManyUnresponsive(t *testing.T, scheme string) {
	var (
		once   sync.Once
		cancel = make(chan struct{})
		term   = func() {
			once.Do(func() {
				close(cancel)
			})
		}
	)
	sourceAccountTrie, elems, storageTries, storageElems := makeAccountTrieWithStorage(scheme, 100, 3000, true, false, false)

	mkSource := func(name string, noAccount, noStorage bool) *testPeer {
		source := newTestPeer(name, t, term)
		source.accountTrie = sourceAccountTrie.Copy()
		source.accountValues = elems
		source.setStorageTries(storageTries)
		source.storageValues = storageElems

		if !noAccount {
			source.accountRequestHandler = nonResponsiveRequestAccountRangeFn
		}
		if !noStorage {
			source.storageRequestHandler = nonResponsiveStorageRequestHandler
		}
		return source
	}

	syncer := setupSyncer(
		scheme,
		mkSource("full", true, true),
		mkSource("noAccounts", false, true),
		mkSource("noStorage", true, false),
	)
	// We're setting the timeout to very low, to make the test run a bit faster
	syncer.rates.OverrideTTLLimit = time.Millisecond

	done := checkStall(t, term)
	if err := syncer.Sync(sourceAccountTrie.Hash(), 0, cancel); err != nil {
		t.Fatalf("sync failed: %v", err)
	}
	close(done)
	verifyTrie(scheme, syncer.db, sourceAccountTrie.Hash(), t)
}

func checkStall(t *testing.T, term func()) chan struct{} {
	testDone := make(chan struct{})
	go func() {
		select {
		case <-time.After(time.Minute): // TODO(karalabe): Make tests smaller, this is too much
			t.Log("Sync stalled")
			term()
		case <-testDone:
			return
		}
	}()
	return testDone
}

// TestSyncBoundaryAccountTrie tests sync against a few normal peers, but the
// account trie has a few boundary elements.
func TestSyncBoundaryAccountTrie(t *testing.T) {
	t.Parallel()

	testSyncBoundaryAccountTrie(t, rawdb.HashScheme)
	testSyncBoundaryAccountTrie(t, rawdb.PathScheme)
}

func testSyncBoundaryAccountTrie(t *testing.T, scheme string) {
	var (
		once   sync.Once
		cancel = make(chan struct{})
		term   = func() {
			once.Do(func() {
				close(cancel)
			})
		}
	)
	nodeScheme, sourceAccountTrie, elems := makeBoundaryAccountTrie(scheme, 3000)

	mkSource := func(name string) *testPeer {
		source := newTestPeer(name, t, term)
		source.accountTrie = sourceAccountTrie.Copy()
		source.accountValues = elems
		return source
	}
	syncer := setupSyncer(
		nodeScheme,
		mkSource("peer-a"),
		mkSource("peer-b"),
	)
	done := checkStall(t, term)
	if err := syncer.Sync(sourceAccountTrie.Hash(), 0, cancel); err != nil {
		t.Fatalf("sync failed: %v", err)
	}
	close(done)
	verifyTrie(scheme, syncer.db, sourceAccountTrie.Hash(), t)
}

// TestSyncNoStorageAndOneCappedPeer tests sync using accounts and no storage, where one peer is
// consistently returning very small results
func TestSyncNoStorageAndOneCappedPeer(t *testing.T) {
	t.Parallel()

	testSyncNoStorageAndOneCappedPeer(t, rawdb.HashScheme)
	testSyncNoStorageAndOneCappedPeer(t, rawdb.PathScheme)
}

func testSyncNoStorageAndOneCappedPeer(t *testing.T, scheme string) {
	var (
		once   sync.Once
		cancel = make(chan struct{})
		term   = func() {
			once.Do(func() {
				close(cancel)
			})
		}
	)
	nodeScheme, sourceAccountTrie, elems := makeAccountTrieNoStorage(3000, scheme)

	mkSource := func(name string, slow bool) *testPeer {
		source := newTestPeer(name, t, term)
		source.accountTrie = sourceAccountTrie.Copy()
		source.accountValues = elems

		if slow {
			source.accountRequestHandler = starvingAccountRequestHandler
		}
		return source
	}

	syncer := setupSyncer(
		nodeScheme,
		mkSource("nice-a", false),
		mkSource("nice-b", false),
		mkSource("nice-c", false),
		mkSource("capped", true),
	)
	done := checkStall(t, term)
	if err := syncer.Sync(sourceAccountTrie.Hash(), 0, cancel); err != nil {
		t.Fatalf("sync failed: %v", err)
	}
	close(done)
	verifyTrie(scheme, syncer.db, sourceAccountTrie.Hash(), t)
}

// TestSyncNoStorageAndOneCodeCorruptPeer has one peer which doesn't deliver
// code requests properly.
func TestSyncNoStorageAndOneCodeCorruptPeer(t *testing.T) {
	t.Parallel()

	testSyncNoStorageAndOneCodeCorruptPeer(t, rawdb.HashScheme)
	testSyncNoStorageAndOneCodeCorruptPeer(t, rawdb.PathScheme)
}

func testSyncNoStorageAndOneCodeCorruptPeer(t *testing.T, scheme string) {
	var (
		once   sync.Once
		cancel = make(chan struct{})
		term   = func() {
			once.Do(func() {
				close(cancel)
			})
		}
	)
	nodeScheme, sourceAccountTrie, elems := makeAccountTrieNoStorage(3000, scheme)

	mkSource := func(name string, codeFn codeHandlerFunc) *testPeer {
		source := newTestPeer(name, t, term)
		source.accountTrie = sourceAccountTrie.Copy()
		source.accountValues = elems
		source.codeRequestHandler = codeFn
		return source
	}
	// One is capped, one is corrupt. If we don't use a capped one, there's a 50%
	// chance that the full set of codes requested are sent only to the
	// non-corrupt peer, which delivers everything in one go, and makes the
	// test moot
	syncer := setupSyncer(
		nodeScheme,
		mkSource("capped", cappedCodeRequestHandler),
		mkSource("corrupt", corruptCodeRequestHandler),
	)
	done := checkStall(t, term)
	if err := syncer.Sync(sourceAccountTrie.Hash(), 0, cancel); err != nil {
		t.Fatalf("sync failed: %v", err)
	}
	close(done)
	verifyTrie(scheme, syncer.db, sourceAccountTrie.Hash(), t)
}

func TestSyncNoStorageAndOneAccountCorruptPeer(t *testing.T) {
	t.Parallel()

	testSyncNoStorageAndOneAccountCorruptPeer(t, rawdb.HashScheme)
	testSyncNoStorageAndOneAccountCorruptPeer(t, rawdb.PathScheme)
}

func testSyncNoStorageAndOneAccountCorruptPeer(t *testing.T, scheme string) {
	var (
		once   sync.Once
		cancel = make(chan struct{})
		term   = func() {
			once.Do(func() {
				close(cancel)
			})
		}
	)
	nodeScheme, sourceAccountTrie, elems := makeAccountTrieNoStorage(3000, scheme)

	mkSource := func(name string, accFn accountHandlerFunc) *testPeer {
		source := newTestPeer(name, t, term)
		source.accountTrie = sourceAccountTrie.Copy()
		source.accountValues = elems
		source.accountRequestHandler = accFn
		return source
	}
	// One is capped, one is corrupt. If we don't use a capped one, there's a 50%
	// chance that the full set of codes requested are sent only to the
	// non-corrupt peer, which delivers everything in one go, and makes the
	// test moot
	syncer := setupSyncer(
		nodeScheme,
		mkSource("capped", defaultAccountRequestHandler),
		mkSource("corrupt", corruptAccountRequestHandler),
	)
	done := checkStall(t, term)
	if err := syncer.Sync(sourceAccountTrie.Hash(), 0, cancel); err != nil {
		t.Fatalf("sync failed: %v", err)
	}
	close(done)
	verifyTrie(scheme, syncer.db, sourceAccountTrie.Hash(), t)
}

// TestSyncNoStorageAndOneCodeCappedPeer has one peer which delivers code hashes
// one by one
func TestSyncNoStorageAndOneCodeCappedPeer(t *testing.T) {
	t.Parallel()

	testSyncNoStorageAndOneCodeCappedPeer(t, rawdb.HashScheme)
	testSyncNoStorageAndOneCodeCappedPeer(t, rawdb.PathScheme)
}

func testSyncNoStorageAndOneCodeCappedPeer(t *testing.T, scheme string) {
	var (
		once   sync.Once
		cancel = make(chan struct{})
		term   = func() {
			once.Do(func() {
				close(cancel)
			})
		}
	)
	nodeScheme, sourceAccountTrie, elems := makeAccountTrieNoStorage(3000, scheme)

	mkSource := func(name string, codeFn codeHandlerFunc) *testPeer {
		source := newTestPeer(name, t, term)
		source.accountTrie = sourceAccountTrie.Copy()
		source.accountValues = elems
		source.codeRequestHandler = codeFn
		return source
	}
	// Count how many times it's invoked. Remember, there are only 8 unique hashes,
	// so it shouldn't be more than that
	var counter int
	syncer := setupSyncer(
		nodeScheme,
		mkSource("capped", func(t *testPeer, id uint64, hashes []common.Hash, max int) error {
			counter++
			return cappedCodeRequestHandler(t, id, hashes, max)
		}),
	)
	done := checkStall(t, term)
	if err := syncer.Sync(sourceAccountTrie.Hash(), 0, cancel); err != nil {
		t.Fatalf("sync failed: %v", err)
	}
	close(done)

	// There are only 8 unique hashes, and 3K accounts. However, the code
	// deduplication is per request batch. If it were a perfect global dedup,
	// we would expect only 8 requests. If there were no dedup, there would be
	// 3k requests.
	// We expect somewhere below 100 requests for these 8 unique hashes. But
	// the number can be flaky, so don't limit it so strictly.
	if threshold := 100; counter > threshold {
		t.Logf("Error, expected < %d invocations, got %d", threshold, counter)
	}
	verifyTrie(scheme, syncer.db, sourceAccountTrie.Hash(), t)
}

// TestSyncBoundaryStorageTrie tests sync against a few normal peers, but the
// storage trie has a few boundary elements.
func TestSyncBoundaryStorageTrie(t *testing.T) {
	t.Parallel()

	testSyncBoundaryStorageTrie(t, rawdb.HashScheme)
	testSyncBoundaryStorageTrie(t, rawdb.PathScheme)
}

func testSyncBoundaryStorageTrie(t *testing.T, scheme string) {
	var (
		once   sync.Once
		cancel = make(chan struct{})
		term   = func() {
			once.Do(func() {
				close(cancel)
			})
		}
	)
	sourceAccountTrie, elems, storageTries, storageElems := makeAccountTrieWithStorage(scheme, 10, 1000, false, true, false)

	mkSource := func(name string) *testPeer {
		source := newTestPeer(name, t, term)
		source.accountTrie = sourceAccountTrie.Copy()
		source.accountValues = elems
		source.setStorageTries(storageTries)
		source.storageValues = storageElems
		return source
	}
	syncer := setupSyncer(
		scheme,
		mkSource("peer-a"),
		mkSource("peer-b"),
	)
	done := checkStall(t, term)
	if err := syncer.Sync(sourceAccountTrie.Hash(), 0, cancel); err != nil {
		t.Fatalf("sync failed: %v", err)
	}
	close(done)
	verifyTrie(scheme, syncer.db, sourceAccountTrie.Hash(), t)
}

// TestSyncWithStorageAndOneCappedPeer tests sync using accounts + storage, where one peer is
// consistently returning very small results
func TestSyncWithStorageAndOneCappedPeer(t *testing.T) {
	t.Parallel()

	testSyncWithStorageAndOneCappedPeer(t, rawdb.HashScheme)
	testSyncWithStorageAndOneCappedPeer(t, rawdb.PathScheme)
}

func testSyncWithStorageAndOneCappedPeer(t *testing.T, scheme string) {
	var (
		once   sync.Once
		cancel = make(chan struct{})
		term   = func() {
			once.Do(func() {
				close(cancel)
			})
		}
	)
	sourceAccountTrie, elems, storageTries, storageElems := makeAccountTrieWithStorage(scheme, 300, 1000, false, false, false)

	mkSource := func(name string, slow bool) *testPeer {
		source := newTestPeer(name, t, term)
		source.accountTrie = sourceAccountTrie.Copy()
		source.accountValues = elems
		source.setStorageTries(storageTries)
		source.storageValues = storageElems

		if slow {
			source.storageRequestHandler = starvingStorageRequestHandler
		}
		return source
	}

	syncer := setupSyncer(
		scheme,
		mkSource("nice-a", false),
		mkSource("slow", true),
	)
	done := checkStall(t, term)
	if err := syncer.Sync(sourceAccountTrie.Hash(), 0, cancel); err != nil {
		t.Fatalf("sync failed: %v", err)
	}
	close(done)
	verifyTrie(scheme, syncer.db, sourceAccountTrie.Hash(), t)
}

// TestSyncWithStorageAndCorruptPeer tests sync using accounts + storage, where one peer is
// sometimes sending bad proofs
func TestSyncWithStorageAndCorruptPeer(t *testing.T) {
	t.Parallel()

	testSyncWithStorageAndCorruptPeer(t, rawdb.HashScheme)
	testSyncWithStorageAndCorruptPeer(t, rawdb.PathScheme)
}

func testSyncWithStorageAndCorruptPeer(t *testing.T, scheme string) {
	var (
		once   sync.Once
		cancel = make(chan struct{})
		term   = func() {
			once.Do(func() {
				close(cancel)
			})
		}
	)
	sourceAccountTrie, elems, storageTries, storageElems := makeAccountTrieWithStorage(scheme, 100, 3000, true, false, false)

	mkSource := func(name string, handler storageHandlerFunc) *testPeer {
		source := newTestPeer(name, t, term)
		source.accountTrie = sourceAccountTrie.Copy()
		source.accountValues = elems
		source.setStorageTries(storageTries)
		source.storageValues = storageElems
		source.storageRequestHandler = handler
		return source
	}

	syncer := setupSyncer(
		scheme,
		mkSource("nice-a", defaultStorageRequestHandler),
		mkSource("nice-b", defaultStorageRequestHandler),
		mkSource("nice-c", defaultStorageRequestHandler),
		mkSource("corrupt", corruptStorageRequestHandler),
	)
	done := checkStall(t, term)
	if err := syncer.Sync(sourceAccountTrie.Hash(), 0, cancel); err != nil {
		t.Fatalf("sync failed: %v", err)
	}
	close(done)
	verifyTrie(scheme, syncer.db, sourceAccountTrie.Hash(), t)
}

func TestSyncWithStorageAndNonProvingPeer(t *testing.T) {
	t.Parallel()

	testSyncWithStorageAndNonProvingPeer(t, rawdb.HashScheme)
	testSyncWithStorageAndNonProvingPeer(t, rawdb.PathScheme)
}

func testSyncWithStorageAndNonProvingPeer(t *testing.T, scheme string) {
	var (
		once   sync.Once
		cancel = make(chan struct{})
		term   = func() {
			once.Do(func() {
				close(cancel)
			})
		}
	)
	sourceAccountTrie, elems, storageTries, storageElems := makeAccountTrieWithStorage(scheme, 100, 3000, true, false, false)

	mkSource := func(name string, handler storageHandlerFunc) *testPeer {
		source := newTestPeer(name, t, term)
		source.accountTrie = sourceAccountTrie.Copy()
		source.accountValues = elems
		source.setStorageTries(storageTries)
		source.storageValues = storageElems
		source.storageRequestHandler = handler
		return source
	}
	syncer := setupSyncer(
		scheme,
		mkSource("nice-a", defaultStorageRequestHandler),
		mkSource("nice-b", defaultStorageRequestHandler),
		mkSource("nice-c", defaultStorageRequestHandler),
		mkSource("corrupt", noProofStorageRequestHandler),
	)
	done := checkStall(t, term)
	if err := syncer.Sync(sourceAccountTrie.Hash(), 0, cancel); err != nil {
		t.Fatalf("sync failed: %v", err)
	}
	close(done)
	verifyTrie(scheme, syncer.db, sourceAccountTrie.Hash(), t)
}

// TestSyncWithStorageMisbehavingProve tests  basic sync using accounts + storage + code, against
// a peer who insists on delivering full storage sets _and_ proofs. This triggered
// an error, where the recipient erroneously clipped the boundary nodes, but
// did not mark the account for healing.
func TestSyncWithStorageMisbehavingProve(t *testing.T) {
	t.Parallel()

	testSyncWithStorageMisbehavingProve(t, rawdb.HashScheme)
	testSyncWithStorageMisbehavingProve(t, rawdb.PathScheme)
}

func testSyncWithStorageMisbehavingProve(t *testing.T, scheme string) {
	var (
		once   sync.Once
		cancel = make(chan struct{})
		term   = func() {
			once.Do(func() {
				close(cancel)
			})
		}
	)
	nodeScheme, sourceAccountTrie, elems, storageTries, storageElems := makeAccountTrieWithStorageWithUniqueStorage(scheme, 10, 30, false)

	mkSource := func(name string) *testPeer {
		source := newTestPeer(name, t, term)
		source.accountTrie = sourceAccountTrie.Copy()
		source.accountValues = elems
		source.setStorageTries(storageTries)
		source.storageValues = storageElems
		source.storageRequestHandler = proofHappyStorageRequestHandler
		return source
	}
	syncer := setupSyncer(nodeScheme, mkSource("sourceA"))
	if err := syncer.Sync(sourceAccountTrie.Hash(), 0, cancel); err != nil {
		t.Fatalf("sync failed: %v", err)
	}
	verifyTrie(scheme, syncer.db, sourceAccountTrie.Hash(), t)
}

// TestSyncWithUnevenStorage tests sync where the storage trie is not even
// and with a few empty ranges.
func TestSyncWithUnevenStorage(t *testing.T) {
	t.Parallel()

	testSyncWithUnevenStorage(t, rawdb.HashScheme)
	testSyncWithUnevenStorage(t, rawdb.PathScheme)
}

func testSyncWithUnevenStorage(t *testing.T, scheme string) {
	var (
		once   sync.Once
		cancel = make(chan struct{})
		term   = func() {
			once.Do(func() {
				close(cancel)
			})
		}
	)
	accountTrie, accounts, storageTries, storageElems := makeAccountTrieWithStorage(scheme, 3, 256, false, false, true)

	mkSource := func(name string) *testPeer {
		source := newTestPeer(name, t, term)
		source.accountTrie = accountTrie.Copy()
		source.accountValues = accounts
		source.setStorageTries(storageTries)
		source.storageValues = storageElems
		source.storageRequestHandler = func(t *testPeer, reqId uint64, root common.Hash, accounts []common.Hash, origin, limit []byte, max int) error {
			return defaultStorageRequestHandler(t, reqId, root, accounts, origin, limit, 128) // retrieve storage in large mode
		}
		return source
	}
	syncer := setupSyncer(scheme, mkSource("source"))
	if err := syncer.Sync(accountTrie.Hash(), 0, cancel); err != nil {
		t.Fatalf("sync failed: %v", err)
	}
	verifyTrie(scheme, syncer.db, accountTrie.Hash(), t)
}

type kv struct {
	k, v []byte
}

func (k *kv) cmp(other *kv) int {
	return bytes.Compare(k.k, other.k)
}

func key32(i uint64) []byte {
	key := make([]byte, 32)
	binary.LittleEndian.PutUint64(key, i)
	return key
}

var (
	codehashes = []common.Hash{
		crypto.Keccak256Hash([]byte{0}),
		crypto.Keccak256Hash([]byte{1}),
		crypto.Keccak256Hash([]byte{2}),
		crypto.Keccak256Hash([]byte{3}),
		crypto.Keccak256Hash([]byte{4}),
		crypto.Keccak256Hash([]byte{5}),
		crypto.Keccak256Hash([]byte{6}),
		crypto.Keccak256Hash([]byte{7}),
	}
)

// getCodeHash returns a pseudo-random code hash
func getCodeHash(i uint64) []byte {
	h := codehashes[int(i)%len(codehashes)]
	return common.CopyBytes(h[:])
}

// getCodeByHash convenience function to lookup the code from the code hash
func getCodeByHash(hash common.Hash) []byte {
	if hash == types.EmptyCodeHash {
		return nil
	}
	for i, h := range codehashes {
		if h == hash {
			return []byte{byte(i)}
		}
	}
	return nil
}

// makeAccountTrieNoStorage spits out a trie, along with the leaves
func makeAccountTrieNoStorage(n int, scheme string) (string, *trie.Trie, []*kv) {
	var (
		db      = triedb.NewDatabase(rawdb.NewMemoryDatabase(), newDbConfig(scheme))
		accTrie = trie.NewEmpty(db)
		entries []*kv
	)
	for i := uint64(1); i <= uint64(n); i++ {
		value, _ := rlp.EncodeToBytes(&types.StateAccount{
			Nonce:    i,
			Balance:  uint256.NewInt(i),
			Root:     types.EmptyRootHash,
			CodeHash: getCodeHash(i),
		})
		key := key32(i)
		elem := &kv{key, value}
		accTrie.MustUpdate(elem.k, elem.v)
		entries = append(entries, elem)
	}
	slices.SortFunc(entries, (*kv).cmp)

	// Commit the state changes into db and re-create the trie
	// for accessing later.
	root, nodes := accTrie.Commit(false)
	db.Update(root, types.EmptyRootHash, 0, trienode.NewWithNodeSet(nodes), triedb.NewStateSet())

	accTrie, _ = trie.New(trie.StateTrieID(root), db)
	return db.Scheme(), accTrie, entries
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

// makeBoundaryAccountTrie constructs an account trie. Instead of filling
// accounts normally, this function will fill a few accounts which have
// boundary hash.
func makeBoundaryAccountTrie(scheme string, n int) (string, *trie.Trie, []*kv) {
	var (
		entries    []*kv
		boundaries []common.Hash

		db      = triedb.NewDatabase(rawdb.NewMemoryDatabase(), newDbConfig(scheme))
		accTrie = trie.NewEmpty(db)
	)
	// Initialize boundaries
	var next common.Hash
	step := new(big.Int).Sub(
		new(big.Int).Div(
			new(big.Int).Exp(common.Big2, common.Big256, nil),
			big.NewInt(int64(accountConcurrency)),
		), common.Big1,
	)
	for i := 0; i < accountConcurrency; i++ {
		last := common.BigToHash(new(big.Int).Add(next.Big(), step))
		if i == accountConcurrency-1 {
			last = common.MaxHash
		}
		boundaries = append(boundaries, last)
		next = common.BigToHash(new(big.Int).Add(last.Big(), common.Big1))
	}
	// Fill boundary accounts
	for i := 0; i < len(boundaries); i++ {
		value, _ := rlp.EncodeToBytes(&types.StateAccount{
			Nonce:    uint64(0),
			Balance:  uint256.NewInt(uint64(i)),
			Root:     types.EmptyRootHash,
			CodeHash: getCodeHash(uint64(i)),
		})
		elem := &kv{boundaries[i].Bytes(), value}
		accTrie.MustUpdate(elem.k, elem.v)
		entries = append(entries, elem)
	}
	// Fill other accounts if required
	for i := uint64(1); i <= uint64(n); i++ {
		value, _ := rlp.EncodeToBytes(&types.StateAccount{
			Nonce:    i,
			Balance:  uint256.NewInt(i),
			Root:     types.EmptyRootHash,
			CodeHash: getCodeHash(i),
		})
		elem := &kv{key32(i), value}
		accTrie.MustUpdate(elem.k, elem.v)
		entries = append(entries, elem)
	}
	slices.SortFunc(entries, (*kv).cmp)

	// Commit the state changes into db and re-create the trie
	// for accessing later.
	root, nodes := accTrie.Commit(false)
	db.Update(root, types.EmptyRootHash, 0, trienode.NewWithNodeSet(nodes), triedb.NewStateSet())

	accTrie, _ = trie.New(trie.StateTrieID(root), db)
	return db.Scheme(), accTrie, entries
}

// makeAccountTrieWithStorageWithUniqueStorage creates an account trie where each accounts
// has a unique storage set.
func makeAccountTrieWithStorageWithUniqueStorage(scheme string, accounts, slots int, code bool) (string, *trie.Trie, []*kv, map[common.Hash]*trie.Trie, map[common.Hash][]*kv) {
	var (
		db             = triedb.NewDatabase(rawdb.NewMemoryDatabase(), newDbConfig(scheme))
		accTrie        = trie.NewEmpty(db)
		entries        []*kv
		storageRoots   = make(map[common.Hash]common.Hash)
		storageTries   = make(map[common.Hash]*trie.Trie)
		storageEntries = make(map[common.Hash][]*kv)
		nodes          = trienode.NewMergedNodeSet()
	)
	// Create n accounts in the trie
	for i := uint64(1); i <= uint64(accounts); i++ {
		key := key32(i)
		codehash := types.EmptyCodeHash.Bytes()
		if code {
			codehash = getCodeHash(i)
		}
		// Create a storage trie
		stRoot, stNodes, stEntries := makeStorageTrieWithSeed(common.BytesToHash(key), uint64(slots), i, db)
		nodes.Merge(stNodes)

		value, _ := rlp.EncodeToBytes(&types.StateAccount{
			Nonce:    i,
			Balance:  uint256.NewInt(i),
			Root:     stRoot,
			CodeHash: codehash,
		})
		elem := &kv{key, value}
		accTrie.MustUpdate(elem.k, elem.v)
		entries = append(entries, elem)

		storageRoots[common.BytesToHash(key)] = stRoot
		storageEntries[common.BytesToHash(key)] = stEntries
	}
	slices.SortFunc(entries, (*kv).cmp)

	// Commit account trie
	root, set := accTrie.Commit(true)
	nodes.Merge(set)

	// Commit gathered dirty nodes into database
	db.Update(root, types.EmptyRootHash, 0, nodes, triedb.NewStateSet())

	// Re-create tries with new root
	accTrie, _ = trie.New(trie.StateTrieID(root), db)
	for i := uint64(1); i <= uint64(accounts); i++ {
		key := key32(i)
		id := trie.StorageTrieID(root, common.BytesToHash(key), storageRoots[common.BytesToHash(key)])
		trie, _ := trie.New(id, db)
		storageTries[common.BytesToHash(key)] = trie
	}
	return db.Scheme(), accTrie, entries, storageTries, storageEntries
}

// makeAccountTrieWithStorage spits out a trie, along with the leaves
func makeAccountTrieWithStorage(scheme string, accounts, slots int, code, boundary bool, uneven bool) (*trie.Trie, []*kv, map[common.Hash]*trie.Trie, map[common.Hash][]*kv) {
	var (
		db             = triedb.NewDatabase(rawdb.NewMemoryDatabase(), newDbConfig(scheme))
		accTrie        = trie.NewEmpty(db)
		entries        []*kv
		storageRoots   = make(map[common.Hash]common.Hash)
		storageTries   = make(map[common.Hash]*trie.Trie)
		storageEntries = make(map[common.Hash][]*kv)
		nodes          = trienode.NewMergedNodeSet()
	)
	// Create n accounts in the trie
	for i := uint64(1); i <= uint64(accounts); i++ {
		key := key32(i)
		codehash := types.EmptyCodeHash.Bytes()
		if code {
			codehash = getCodeHash(i)
		}
		// Make a storage trie
		var (
			stRoot    common.Hash
			stNodes   *trienode.NodeSet
			stEntries []*kv
		)
		if boundary {
			stRoot, stNodes, stEntries = makeBoundaryStorageTrie(common.BytesToHash(key), slots, db)
		} else if uneven {
			stRoot, stNodes, stEntries = makeUnevenStorageTrie(common.BytesToHash(key), slots, db)
		} else {
			stRoot, stNodes, stEntries = makeStorageTrieWithSeed(common.BytesToHash(key), uint64(slots), 0, db)
		}
		nodes.Merge(stNodes)

		value, _ := rlp.EncodeToBytes(&types.StateAccount{
			Nonce:    i,
			Balance:  uint256.NewInt(i),
			Root:     stRoot,
			CodeHash: codehash,
		})
		elem := &kv{key, value}
		accTrie.MustUpdate(elem.k, elem.v)
		entries = append(entries, elem)

		// we reuse the same one for all accounts
		storageRoots[common.BytesToHash(key)] = stRoot
		storageEntries[common.BytesToHash(key)] = stEntries
	}
	slices.SortFunc(entries, (*kv).cmp)

	// Commit account trie
	root, set := accTrie.Commit(true)
	nodes.Merge(set)

	// Commit gathered dirty nodes into database
	db.Update(root, types.EmptyRootHash, 0, nodes, triedb.NewStateSet())

	// Re-create tries with new root
	accTrie, err := trie.New(trie.StateTrieID(root), db)
	if err != nil {
		panic(err)
	}
	for i := uint64(1); i <= uint64(accounts); i++ {
		key := key32(i)
		id := trie.StorageTrieID(root, common.BytesToHash(key), storageRoots[common.BytesToHash(key)])
		trie, err := trie.New(id, db)
		if err != nil {
			panic(err)
		}
		storageTries[common.BytesToHash(key)] = trie
	}
	return accTrie, entries, storageTries, storageEntries
}

// makeStorageTrieWithSeed fills a storage trie with n items, returning the
// not-yet-committed trie and the sorted entries. The seeds can be used to ensure
// that tries are unique.
func makeStorageTrieWithSeed(owner common.Hash, n, seed uint64, db *triedb.Database) (common.Hash, *trienode.NodeSet, []*kv) {
	trie, _ := trie.New(trie.StorageTrieID(types.EmptyRootHash, owner, types.EmptyRootHash), db)
	var entries []*kv
	for i := uint64(1); i <= n; i++ {
		// store 'x' at slot 'x'
		slotValue := key32(i + seed)
		rlpSlotValue, _ := rlp.EncodeToBytes(common.TrimLeftZeroes(slotValue[:]))

		slotKey := key32(i)
		key := crypto.Keccak256Hash(slotKey[:])

		elem := &kv{key[:], rlpSlotValue}
		trie.MustUpdate(elem.k, elem.v)
		entries = append(entries, elem)
	}
	slices.SortFunc(entries, (*kv).cmp)
	root, nodes := trie.Commit(false)
	return root, nodes, entries
}

// makeBoundaryStorageTrie constructs a storage trie. Instead of filling
// storage slots normally, this function will fill a few slots which have
// boundary hash.
func makeBoundaryStorageTrie(owner common.Hash, n int, db *triedb.Database) (common.Hash, *trienode.NodeSet, []*kv) {
	var (
		entries    []*kv
		boundaries []common.Hash
		trie, _    = trie.New(trie.StorageTrieID(types.EmptyRootHash, owner, types.EmptyRootHash), db)
	)
	// Initialize boundaries
	var next common.Hash
	step := new(big.Int).Sub(
		new(big.Int).Div(
			new(big.Int).Exp(common.Big2, common.Big256, nil),
			big.NewInt(int64(accountConcurrency)),
		), common.Big1,
	)
	for i := 0; i < accountConcurrency; i++ {
		last := common.BigToHash(new(big.Int).Add(next.Big(), step))
		if i == accountConcurrency-1 {
			last = common.MaxHash
		}
		boundaries = append(boundaries, last)
		next = common.BigToHash(new(big.Int).Add(last.Big(), common.Big1))
	}
	// Fill boundary slots
	for i := 0; i < len(boundaries); i++ {
		key := boundaries[i]
		val := []byte{0xde, 0xad, 0xbe, 0xef}

		elem := &kv{key[:], val}
		trie.MustUpdate(elem.k, elem.v)
		entries = append(entries, elem)
	}
	// Fill other slots if required
	for i := uint64(1); i <= uint64(n); i++ {
		slotKey := key32(i)
		key := crypto.Keccak256Hash(slotKey[:])

		slotValue := key32(i)
		rlpSlotValue, _ := rlp.EncodeToBytes(common.TrimLeftZeroes(slotValue[:]))

		elem := &kv{key[:], rlpSlotValue}
		trie.MustUpdate(elem.k, elem.v)
		entries = append(entries, elem)
	}
	slices.SortFunc(entries, (*kv).cmp)
	root, nodes := trie.Commit(false)
	return root, nodes, entries
}

// makeUnevenStorageTrie constructs a storage tries will states distributed in
// different range unevenly.
func makeUnevenStorageTrie(owner common.Hash, slots int, db *triedb.Database) (common.Hash, *trienode.NodeSet, []*kv) {
	var (
		entries []*kv
		tr, _   = trie.New(trie.StorageTrieID(types.EmptyRootHash, owner, types.EmptyRootHash), db)
		chosen  = make(map[byte]struct{})
	)
	for i := 0; i < 3; i++ {
		var n int
		for {
			n = mrand.Intn(15) // the last range is set empty deliberately
			if _, ok := chosen[byte(n)]; ok {
				continue
			}
			chosen[byte(n)] = struct{}{}
			break
		}
		for j := 0; j < slots/3; j++ {
			key := append([]byte{byte(n)}, testrand.Bytes(31)...)
			val, _ := rlp.EncodeToBytes(testrand.Bytes(32))

			elem := &kv{key, val}
			tr.MustUpdate(elem.k, elem.v)
			entries = append(entries, elem)
		}
	}
	slices.SortFunc(entries, (*kv).cmp)
	root, nodes := tr.Commit(false)
	return root, nodes, entries
}

func verifyTrie(scheme string, db ethdb.KeyValueStore, root common.Hash, t *testing.T) {
	t.Helper()
	triedb := triedb.NewDatabase(rawdb.NewDatabase(db), newDbConfig(scheme))
	accTrie, err := trie.New(trie.StateTrieID(root), triedb)
	if err != nil {
		t.Fatal(err)
	}
	accounts, slots := 0, 0
	accIt := trie.NewIterator(accTrie.MustNodeIterator(nil))
	for accIt.Next() {
		var acc struct {
			Nonce    uint64
			Balance  *big.Int
			Root     common.Hash
			CodeHash []byte
		}
		if err := rlp.DecodeBytes(accIt.Value, &acc); err != nil {
			log.Crit("Invalid account encountered during snapshot creation", "err", err)
		}
		accounts++
		if acc.Root != types.EmptyRootHash {
			id := trie.StorageTrieID(root, common.BytesToHash(accIt.Key), acc.Root)
			storeTrie, err := trie.NewStateTrie(id, triedb)
			if err != nil {
				t.Fatal(err)
			}
			storeIt := trie.NewIterator(storeTrie.MustNodeIterator(nil))
			for storeIt.Next() {
				slots++
			}
			if err := storeIt.Err; err != nil {
				t.Fatal(err)
			}
		}
	}
	if err := accIt.Err; err != nil {
		t.Fatal(err)
	}
	t.Logf("accounts: %d, slots: %d", accounts, slots)
}

func TestSlotEstimation(t *testing.T) {
	for i, tc := range []struct {
		last  common.Hash
		count int
		want  uint64
	}{
		{
			// Half the space
			common.HexToHash("0x7fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff"),
			100,
			100,
		},
		{
			// 1 / 16th
			common.HexToHash("0x0fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff"),
			100,
			1500,
		},
		{
			// Bit more than 1 / 16th
			common.HexToHash("0x1000000000000000000000000000000000000000000000000000000000000000"),
			100,
			1499,
		},
		{
			// Almost everything
			common.HexToHash("0xF000000000000000000000000000000000000000000000000000000000000000"),
			100,
			6,
		},
		{
			// Almost nothing -- should lead to error
			common.HexToHash("0x0000000000000000000000000000000000000000000000000000000000000001"),
			1,
			0,
		},
		{
			// Nothing -- should lead to error
			common.Hash{},
			100,
			0,
		},
	} {
		have, _ := estimateRemainingSlots(tc.count, tc.last)
		if want := tc.want; have != want {
			t.Errorf("test %d: have %d want %d", i, have, want)
		}
	}
}

// TestPivotMoveDetection verifies that when the syncer is restarted with a
// different root (simulating the downloader's cancel+restart on pivot move),
// downloadState() returns errPivotStale immediately.
func TestPivotMoveDetection(t *testing.T) {
	t.Parallel()

	rootA := common.HexToHash("0xaaaa")
	rootB := common.HexToHash("0xbbbb")

	db := rawdb.NewMemoryDatabase()
	syncer := NewSyncer(db, rawdb.HashScheme)

	// Simulate a previous sync run against rootA with some pending tasks
	syncer.root = rootA
	syncer.tasks = []*accountTask{
		{Next: common.Hash{}, Last: common.MaxHash, SubTasks: make(map[common.Hash][]*storageTask), stateCompleted: make(map[common.Hash]struct{})},
	}
	syncer.saveSyncStatus()

	// Simulate downloader restarting us with rootB (as Sync() would do)
	syncer.root = rootB
	syncer.previousRoot = rootB // Sync() sets this as default
	syncer.loadSyncStatus()     // Overwrites previousRoot with persisted rootA

	if syncer.previousRoot != rootA {
		t.Fatalf("previousRoot mismatch: got %v, want %v", syncer.previousRoot, rootA)
	}
	if syncer.root != rootB {
		t.Fatalf("root mismatch: got %v, want %v", syncer.root, rootB)
	}
	// downloadState() should detect the mismatch and return errPivotStale
	cancel := make(chan struct{})
	err := syncer.downloadState(cancel)
	if err != errPivotStale {
		t.Fatalf("expected errPivotStale, got %v", err)
	}
}

// TestCatchUpInvertedRange verifies that catchUp returns an error and wipes
// sync progress when the new pivot is at the same (or lower) block number as
// the old pivot..
func TestCatchUpInvertedRange(t *testing.T) {
	t.Parallel()
	db := rawdb.NewMemoryDatabase()
	syncer := NewSyncer(db, rawdb.HashScheme)

	// Simulate: old pivot at block 100, new pivot at block 100 (same number,
	// different root). This happens when a reorg replaces the pivot block.
	syncer.previousNumber = 100
	syncer.number = 100

	// Write some sync progress so we can verify it gets wiped
	rawdb.WriteSnapshotSyncStatus(db, []byte("some progress"))
	cancel := make(chan struct{})
	err := syncer.catchUp(cancel)
	if err == nil {
		t.Fatal("expected error from catchUp with inverted range")
	}

	// Verify sync progress was wiped
	if status := rawdb.ReadSnapshotSyncStatus(db); status != nil {
		t.Fatal("sync progress should be wiped after inverted catch-up range")
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
	cancelAfterHandler := func(tp *testPeer, id uint64, root common.Hash, origin common.Hash, limit common.Hash, cap int) error {
		if responses.Add(1) > 2 {
			term1()
			return nil
		}
		return defaultAccountRequestHandler(tp, id, root, origin, limit, cap)
	}
	db := rawdb.NewMemoryDatabase()
	syncer1 := NewSyncer(db, nodeScheme)
	src1 := newTestPeer("source1", t, term1)
	src1.accountTrie = sourceAccountTrie.Copy()
	src1.accountValues = elems
	src1.accountRequestHandler = cancelAfterHandler
	syncer1.Register(src1)
	src1.remote = syncer1
	syncer1.root = root
	syncer1.previousRoot = root
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
	syncer2 := NewSyncer(db, nodeScheme)
	src2 := newTestPeer("source2", t, term2)
	src2.accountTrie = sourceAccountTrie.Copy()
	src2.accountValues = elems
	syncer2.Register(src2)
	src2.remote = syncer2
	syncer2.root = root
	syncer2.previousRoot = root
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
	syncer1 := NewSyncer(db, nodeScheme)
	src1 := newTestPeer("source1", t, term1)
	src1.accountTrie = sourceAccountTrie.Copy()
	src1.accountValues = elems
	src1.accountRequestHandler = func(tp *testPeer, id uint64, root common.Hash, origin common.Hash, limit common.Hash, cap int) error {
		if responses.Add(1) > 2 {
			term1()
			return nil
		}
		return defaultAccountRequestHandler(tp, id, root, origin, limit, cap)
	}
	syncer1.Register(src1)
	src1.remote = syncer1
	syncer1.Sync(rootA, numA, cancel1)

	// Subsequent runs: each move triggers catch-up then resumes download
	for i, move := range moves {
		var (
			once   sync.Once
			cancel = make(chan struct{})
			term   = func() { once.Do(func() { close(cancel) }) }
		)
		syncer := NewSyncer(db, nodeScheme)
		src := newTestPeer(fmt.Sprintf("source-%d", i+2), t, term)
		src.accountTrie = move.trie.Copy()
		src.accountValues = move.elems
		src.accessLists = move.bals
		syncer.Register(src)
		src.remote = syncer
		if err := syncer.Sync(move.root, move.blockNum, cancel); err != nil {
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

// TestSyncStatusClearedAfterCompletion verifies that the persisted sync status
// is cleared after a full sync completes (download + trie rebuild), so the
// next Sync() call starts fresh.
func TestSyncStatusClearedAfterCompletion(t *testing.T) {
	t.Parallel()
	testSyncStatusClearedAfterCompletion(t, rawdb.HashScheme)
	testSyncStatusClearedAfterCompletion(t, rawdb.PathScheme)
}

func testSyncStatusClearedAfterCompletion(t *testing.T, scheme string) {
	var (
		once   sync.Once
		cancel = make(chan struct{})
		term   = func() { once.Do(func() { close(cancel) }) }
	)
	nodeScheme, sourceAccountTrie, elems := makeAccountTrieNoStorage(100, scheme)

	mkSource := func(name string) *testPeer {
		source := newTestPeer(name, t, term)
		source.accountTrie = sourceAccountTrie.Copy()
		source.accountValues = elems
		return source
	}
	syncer := setupSyncer(nodeScheme, mkSource("source"))
	if err := syncer.Sync(sourceAccountTrie.Hash(), 0, cancel); err != nil {
		t.Fatalf("sync failed: %v", err)
	}
	// After successful sync, status should be cleared
	if status := rawdb.ReadSnapshotSyncStatus(syncer.db); status != nil {
		t.Fatal("sync status should be nil after successful completion")
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
	syncer1 := NewSyncer(db, nodeScheme)
	src1 := newTestPeer("source1", t, term1)
	src1.accountTrie = sourceAccountTrie.Copy()
	src1.accountValues = elems
	syncer1.Register(src1)
	src1.remote = syncer1
	syncer1.root = root
	syncer1.previousRoot = root
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
	syncer2 := NewSyncer(db, nodeScheme)
	src2 := newTestPeer("source2", t, term2)
	src2.accountTrie = sourceAccountTrie.Copy()
	src2.accountValues = elems
	syncer2.Register(src2)
	src2.remote = syncer2

	if err := syncer2.Sync(root, 0, cancel2); err != nil {
		t.Fatalf("resumed sync failed: %v", err)
	}
	// After rebuild completes, status should be cleared
	if status := rawdb.ReadSnapshotSyncStatus(db); status != nil {
		t.Fatal("sync status should be nil after rebuild completes")
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
	mkSource := func(name string) *testPeer {
		source := newTestPeer(name, t, term)
		source.accessLists = bals
		return source
	}
	syncer := setupSyncer(rawdb.HashScheme, mkSource("peer-a"), mkSource("peer-b"), mkSource("peer-c"))
	results, err := syncer.fetchAccessLists(hashes, cancel)
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
	nonResponsive := newTestPeer("non-responsive", t, term)
	nonResponsive.accessListRequestHandler = func(t *testPeer, id uint64, hashes []common.Hash, max int) error {
		// Don't respond — let it time out
		return nil
	}

	// Second peer serves correctly
	good := newTestPeer("good", t, term)
	good.accessLists = bals
	syncer := setupSyncer(rawdb.HashScheme, nonResponsive, good)
	syncer.rates.OverrideTTLLimit = time.Millisecond // Fast timeout
	results, err := syncer.fetchAccessLists(hashes, cancel)
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
	rejector := newTestPeer("rejector", t, term)

	// Second peer serves correctly
	good := newTestPeer("good", t, term)
	good.accessLists = bals
	syncer := setupSyncer(rawdb.HashScheme, rejector, good)
	results, err := syncer.fetchAccessLists(hashes, cancel)
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
	nonResponsive := newTestPeer("non-responsive", t, func() {})
	nonResponsive.accessListRequestHandler = func(t *testPeer, id uint64, hashes []common.Hash, max int) error {
		return nil // never deliver
	}
	syncer := setupSyncer(rawdb.HashScheme, nonResponsive)
	hashes := []common.Hash{common.HexToHash("0x01")}

	// Cancel after a short delay
	go func() {
		time.Sleep(50 * time.Millisecond)
		close(cancel)
	}()
	_, err := syncer.fetchAccessLists(hashes, cancel)
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
	dropped := newTestPeer("dropped", t, term)
	dropped.accessListRequestHandler = func(tp *testPeer, id uint64, hashes []common.Hash, max int) error {
		// Simulate peer dropping by unregistering
		tp.remote.Unregister(tp.id)
		return nil
	}

	// Second peer serves correctly
	good := newTestPeer("good", t, term)
	good.accessLists = bals
	syncer := setupSyncer(rawdb.HashScheme, dropped, good)
	results, err := syncer.fetchAccessLists(hashes, cancel)
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
	shortPeer := newTestPeer("short", t, term)
	shortPeer.accessListRequestHandler = func(tp *testPeer, id uint64, reqHashes []common.Hash, max int) error {
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
	syncer := setupSyncer(rawdb.HashScheme, shortPeer)

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
		results, fetchErr = syncer.fetchAccessLists(hashes, cancel)
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
	partialPeer := newTestPeer("partial", t, term)
	partialPeer.accessListRequestHandler = func(tp *testPeer, id uint64, reqHashes []common.Hash, max int) error {
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
	fullPeer := newTestPeer("full", t, term)
	fullPeer.accessLists = allBALs
	syncer := setupSyncer(rawdb.HashScheme, partialPeer, fullPeer)

	// Pre-seed capacity so partialPeer gets all 3 hashes
	syncer.rates.Update(partialPeer.id, AccessListsMsg, time.Millisecond, 100)
	done := make(chan struct{})
	var (
		results  []rlp.RawValue
		fetchErr error
	)
	go func() {
		results, fetchErr = syncer.fetchAccessLists(hashes, cancel)
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

func newDbConfig(scheme string) *triedb.Config {
	if scheme == rawdb.HashScheme {
		return &triedb.Config{}
	}
	return &triedb.Config{PathDB: &pathdb.Config{SnapshotNoBuild: true}}
}
