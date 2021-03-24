// Copyright 2020 The go-ethereum Authors
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
	"sort"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/light"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/ethereum/go-ethereum/trie"
	"golang.org/x/crypto/sha3"
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
		hasher := sha3.NewLegacyKeccak256()
		for i := 0; i < len(bytecodes); i++ {
			hasher.Reset()
			hasher.Write(bytecodes[i])
			hash := hasher.Sum(nil)
			got = fmt.Sprintf("%v\n%v", got, hash)
		}
	}
	var new = func() {
		hasher := sha3.NewLegacyKeccak256().(crypto.KeccakState)
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
		hasher := sha3.NewLegacyKeccak256()
		for i := 0; i < len(bytecodes); i++ {
			hasher.Reset()
			hasher.Write(bytecodes[i])
			hasher.Sum(nil)
		}
	}
	var new = func() {
		hasher := sha3.NewLegacyKeccak256().(crypto.KeccakState)
		var hash = make([]byte, 32)
		for i := 0; i < len(bytecodes); i++ {
			hasher.Reset()
			hasher.Write(bytecodes[i])
			hasher.Read(hash)
		}
	}
	b.Run("old", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			old()
		}
	})
	b.Run("new", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			new()
		}
	})
}

type storageHandlerFunc func(t *testPeer, requestId uint64, root common.Hash, accounts []common.Hash, origin, limit []byte, max uint64) error
type accountHandlerFunc func(t *testPeer, requestId uint64, root common.Hash, origin common.Hash, cap uint64) error
type trieHandlerFunc func(t *testPeer, requestId uint64, root common.Hash, paths []TrieNodePathSet, cap uint64) error
type codeHandlerFunc func(t *testPeer, id uint64, hashes []common.Hash, max uint64) error

type testPeer struct {
	id            string
	test          *testing.T
	remote        *Syncer
	logger        log.Logger
	accountTrie   *trie.Trie
	accountValues entrySlice
	storageTries  map[common.Hash]*trie.Trie
	storageValues map[common.Hash]entrySlice

	accountRequestHandler accountHandlerFunc
	storageRequestHandler storageHandlerFunc
	trieRequestHandler    trieHandlerFunc
	codeRequestHandler    codeHandlerFunc
	cancelCh              chan struct{}
}

func newTestPeer(id string, t *testing.T, cancelCh chan struct{}) *testPeer {
	peer := &testPeer{
		id:                    id,
		test:                  t,
		logger:                log.New("id", id),
		accountRequestHandler: defaultAccountRequestHandler,
		trieRequestHandler:    defaultTrieRequestHandler,
		storageRequestHandler: defaultStorageRequestHandler,
		codeRequestHandler:    defaultCodeRequestHandler,
		cancelCh:              cancelCh,
	}
	//stderrHandler := log.StreamHandler(os.Stderr, log.TerminalFormat(true))
	//peer.logger.SetHandler(stderrHandler)
	return peer

}

func (t *testPeer) ID() string      { return t.id }
func (t *testPeer) Log() log.Logger { return t.logger }

func (t *testPeer) RequestAccountRange(id uint64, root, origin, limit common.Hash, bytes uint64) error {
	t.logger.Trace("Fetching range of accounts", "reqid", id, "root", root, "origin", origin, "limit", limit, "bytes", common.StorageSize(bytes))
	go t.accountRequestHandler(t, id, root, origin, bytes)
	return nil
}

func (t *testPeer) RequestTrieNodes(id uint64, root common.Hash, paths []TrieNodePathSet, bytes uint64) error {
	t.logger.Trace("Fetching set of trie nodes", "reqid", id, "root", root, "pathsets", len(paths), "bytes", common.StorageSize(bytes))
	go t.trieRequestHandler(t, id, root, paths, bytes)
	return nil
}

func (t *testPeer) RequestStorageRanges(id uint64, root common.Hash, accounts []common.Hash, origin, limit []byte, bytes uint64) error {
	if len(accounts) == 1 && origin != nil {
		t.logger.Trace("Fetching range of large storage slots", "reqid", id, "root", root, "account", accounts[0], "origin", common.BytesToHash(origin), "limit", common.BytesToHash(limit), "bytes", common.StorageSize(bytes))
	} else {
		t.logger.Trace("Fetching ranges of small storage slots", "reqid", id, "root", root, "accounts", len(accounts), "first", accounts[0], "bytes", common.StorageSize(bytes))
	}
	go t.storageRequestHandler(t, id, root, accounts, origin, limit, bytes)
	return nil
}

func (t *testPeer) RequestByteCodes(id uint64, hashes []common.Hash, bytes uint64) error {
	t.logger.Trace("Fetching set of byte codes", "reqid", id, "hashes", len(hashes), "bytes", common.StorageSize(bytes))
	go t.codeRequestHandler(t, id, hashes, bytes)
	return nil
}

// defaultTrieRequestHandler is a well-behaving handler for trie healing requests
func defaultTrieRequestHandler(t *testPeer, requestId uint64, root common.Hash, paths []TrieNodePathSet, cap uint64) error {
	// Pass the response
	var nodes [][]byte
	for _, pathset := range paths {
		switch len(pathset) {
		case 1:
			blob, _, err := t.accountTrie.TryGetNode(pathset[0])
			if err != nil {
				t.logger.Info("Error handling req", "error", err)
				break
			}
			nodes = append(nodes, blob)
		default:
			account := t.storageTries[(common.BytesToHash(pathset[0]))]
			for _, path := range pathset[1:] {
				blob, _, err := account.TryGetNode(path)
				if err != nil {
					t.logger.Info("Error handling req", "error", err)
					break
				}
				nodes = append(nodes, blob)
			}
		}
	}
	t.remote.OnTrieNodes(t, requestId, nodes)
	return nil
}

// defaultAccountRequestHandler is a well-behaving handler for AccountRangeRequests
func defaultAccountRequestHandler(t *testPeer, id uint64, root common.Hash, origin common.Hash, cap uint64) error {
	keys, vals, proofs := createAccountRequestResponse(t, root, origin, cap)
	if err := t.remote.OnAccounts(t, id, keys, vals, proofs); err != nil {
		t.logger.Error("remote error on delivery", "error", err)
		t.test.Errorf("Remote side rejected our delivery: %v", err)
		t.remote.Unregister(t.id)
		close(t.cancelCh)
		return err
	}
	return nil
}

func createAccountRequestResponse(t *testPeer, root common.Hash, origin common.Hash, cap uint64) (keys []common.Hash, vals [][]byte, proofs [][]byte) {
	var size uint64
	for _, entry := range t.accountValues {
		if size > cap {
			break
		}
		if bytes.Compare(origin[:], entry.k) <= 0 {
			keys = append(keys, common.BytesToHash(entry.k))
			vals = append(vals, entry.v)
			size += uint64(32 + len(entry.v))
		}
	}
	// Unless we send the entire trie, we need to supply proofs
	// Actually, we need to supply proofs either way! This seems tob be an implementation
	// quirk in go-ethereum
	proof := light.NewNodeSet()
	if err := t.accountTrie.Prove(origin[:], 0, proof); err != nil {
		t.logger.Error("Could not prove inexistence of origin", "origin", origin,
			"error", err)
	}
	if len(keys) > 0 {
		lastK := (keys[len(keys)-1])[:]
		if err := t.accountTrie.Prove(lastK, 0, proof); err != nil {
			t.logger.Error("Could not prove last item",
				"error", err)
		}
	}
	for _, blob := range proof.NodeList() {
		proofs = append(proofs, blob)
	}
	return keys, vals, proofs
}

// defaultStorageRequestHandler is a well-behaving storage request handler
func defaultStorageRequestHandler(t *testPeer, requestId uint64, root common.Hash, accounts []common.Hash, bOrigin, bLimit []byte, max uint64) error {
	hashes, slots, proofs := createStorageRequestResponse(t, root, accounts, bOrigin, bLimit, max)
	if err := t.remote.OnStorage(t, requestId, hashes, slots, proofs); err != nil {
		t.logger.Error("remote error on delivery", "error", err)
		t.test.Errorf("Remote side rejected our delivery: %v", err)
		close(t.cancelCh)
	}
	return nil
}

func defaultCodeRequestHandler(t *testPeer, id uint64, hashes []common.Hash, max uint64) error {
	var bytecodes [][]byte
	for _, h := range hashes {
		bytecodes = append(bytecodes, getCode(h))
	}
	if err := t.remote.OnByteCodes(t, id, bytecodes); err != nil {
		t.logger.Error("remote error on delivery", "error", err)
		t.test.Errorf("Remote side rejected our delivery: %v", err)
		close(t.cancelCh)
	}
	return nil
}

func createStorageRequestResponse(t *testPeer, root common.Hash, accounts []common.Hash, bOrigin, bLimit []byte, max uint64) (hashes [][]common.Hash, slots [][][]byte, proofs [][]byte) {
	var (
		size  uint64
		limit = common.HexToHash("0xffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff")
	)
	if len(bLimit) > 0 {
		limit = common.BytesToHash(bLimit)
	}
	var origin common.Hash
	if len(bOrigin) > 0 {
		origin = common.BytesToHash(bOrigin)
	}

	var limitExceeded bool
	var incomplete bool
	for _, account := range accounts {

		var keys []common.Hash
		var vals [][]byte
		for _, entry := range t.storageValues[account] {
			if limitExceeded {
				incomplete = true
				break
			}
			if bytes.Compare(entry.k, origin[:]) < 0 {
				incomplete = true
				continue
			}
			keys = append(keys, common.BytesToHash(entry.k))
			vals = append(vals, entry.v)
			size += uint64(32 + len(entry.v))
			if bytes.Compare(entry.k, limit[:]) >= 0 {
				limitExceeded = true
			}
			if size > max {
				limitExceeded = true
			}
		}
		hashes = append(hashes, keys)
		slots = append(slots, vals)

		if incomplete {
			// If we're aborting, we need to prove the first and last item
			// This terminates the response (and thus the loop)
			proof := light.NewNodeSet()
			stTrie := t.storageTries[account]

			// Here's a potential gotcha: when constructing the proof, we cannot
			// use the 'origin' slice directly, but must use the full 32-byte
			// hash form.
			if err := stTrie.Prove(origin[:], 0, proof); err != nil {
				t.logger.Error("Could not prove inexistence of origin", "origin", origin,
					"error", err)
			}
			if len(keys) > 0 {
				lastK := (keys[len(keys)-1])[:]
				if err := stTrie.Prove(lastK, 0, proof); err != nil {
					t.logger.Error("Could not prove last item", "error", err)
				}
			}
			for _, blob := range proof.NodeList() {
				proofs = append(proofs, blob)
			}
			break
		}
	}
	return hashes, slots, proofs
}

// emptyRequestAccountRangeFn is a rejects AccountRangeRequests
func emptyRequestAccountRangeFn(t *testPeer, requestId uint64, root common.Hash, origin common.Hash, cap uint64) error {
	var proofs [][]byte
	var keys []common.Hash
	var vals [][]byte
	t.remote.OnAccounts(t, requestId, keys, vals, proofs)
	return nil
}

func nonResponsiveRequestAccountRangeFn(t *testPeer, requestId uint64, root common.Hash, origin common.Hash, cap uint64) error {
	return nil
}

func emptyTrieRequestHandler(t *testPeer, requestId uint64, root common.Hash, paths []TrieNodePathSet, cap uint64) error {
	var nodes [][]byte
	t.remote.OnTrieNodes(t, requestId, nodes)
	return nil
}

func nonResponsiveTrieRequestHandler(t *testPeer, requestId uint64, root common.Hash, paths []TrieNodePathSet, cap uint64) error {
	return nil
}

func emptyStorageRequestHandler(t *testPeer, requestId uint64, root common.Hash, accounts []common.Hash, origin, limit []byte, max uint64) error {
	var hashes [][]common.Hash
	var slots [][][]byte
	var proofs [][]byte
	t.remote.OnStorage(t, requestId, hashes, slots, proofs)
	return nil
}

func nonResponsiveStorageRequestHandler(t *testPeer, requestId uint64, root common.Hash, accounts []common.Hash, origin, limit []byte, max uint64) error {
	return nil
}

//func emptyCodeRequestHandler(t *testPeer, id uint64, hashes []common.Hash, max uint64) error {
//	var bytecodes [][]byte
//	t.remote.OnByteCodes(t, id, bytecodes)
//	return nil
//}

func corruptCodeRequestHandler(t *testPeer, id uint64, hashes []common.Hash, max uint64) error {
	var bytecodes [][]byte
	for _, h := range hashes {
		// Send back the hashes
		bytecodes = append(bytecodes, h[:])
	}
	if err := t.remote.OnByteCodes(t, id, bytecodes); err != nil {
		t.logger.Error("remote error on delivery", "error", err)
		// Mimic the real-life handler, which drops a peer on errors
		t.remote.Unregister(t.id)
	}
	return nil
}

func cappedCodeRequestHandler(t *testPeer, id uint64, hashes []common.Hash, max uint64) error {
	var bytecodes [][]byte
	for _, h := range hashes[:1] {
		bytecodes = append(bytecodes, getCode(h))
	}
	if err := t.remote.OnByteCodes(t, id, bytecodes); err != nil {
		t.logger.Error("remote error on delivery", "error", err)
		// Mimic the real-life handler, which drops a peer on errors
		t.remote.Unregister(t.id)
	}
	return nil
}

// starvingStorageRequestHandler is somewhat well-behaving storage handler, but it caps the returned results to be very small
func starvingStorageRequestHandler(t *testPeer, requestId uint64, root common.Hash, accounts []common.Hash, origin, limit []byte, max uint64) error {
	return defaultStorageRequestHandler(t, requestId, root, accounts, origin, limit, 500)
}

func starvingAccountRequestHandler(t *testPeer, requestId uint64, root common.Hash, origin common.Hash, cap uint64) error {
	return defaultAccountRequestHandler(t, requestId, root, origin, 500)
}

//func misdeliveringAccountRequestHandler(t *testPeer, requestId uint64, root common.Hash, origin common.Hash, cap uint64) error {
//	return defaultAccountRequestHandler(t, requestId-1, root, origin, 500)
//}

func corruptAccountRequestHandler(t *testPeer, requestId uint64, root common.Hash, origin common.Hash, cap uint64) error {
	hashes, accounts, proofs := createAccountRequestResponse(t, root, origin, cap)
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
func corruptStorageRequestHandler(t *testPeer, requestId uint64, root common.Hash, accounts []common.Hash, origin, limit []byte, max uint64) error {
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

func noProofStorageRequestHandler(t *testPeer, requestId uint64, root common.Hash, accounts []common.Hash, origin, limit []byte, max uint64) error {
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

	sourceAccountTrie, elems := makeAccountTrieNoStorage(100)
	cancel := make(chan struct{})
	source := newTestPeer("source", t, cancel)
	source.accountTrie = sourceAccountTrie
	source.accountValues = elems

	source.accountRequestHandler = func(t *testPeer, requestId uint64, root common.Hash, origin common.Hash, cap uint64) error {
		var proofs [][]byte
		var keys []common.Hash
		var vals [][]byte

		// The values
		for _, entry := range t.accountValues {
			if bytes.Compare(origin[:], entry.k) <= 0 {
				keys = append(keys, common.BytesToHash(entry.k))
				vals = append(vals, entry.v)
			}
		}
		// The proofs
		proof := light.NewNodeSet()
		if err := t.accountTrie.Prove(origin[:], 0, proof); err != nil {
			t.logger.Error("Could not prove origin", "origin", origin, "error", err)
		}
		// The bloat: add proof of every single element
		for _, entry := range t.accountValues {
			if err := t.accountTrie.Prove(entry.k, 0, proof); err != nil {
				t.logger.Error("Could not prove item", "error", err)
			}
		}
		// And remove one item from the elements
		if len(keys) > 2 {
			keys = append(keys[:1], keys[2:]...)
			vals = append(vals[:1], vals[2:]...)
		}
		for _, blob := range proof.NodeList() {
			proofs = append(proofs, blob)
		}
		if err := t.remote.OnAccounts(t, requestId, keys, vals, proofs); err != nil {
			t.logger.Info("remote error on delivery", "error", err)
			// This is actually correct, signal to exit the test successfully
			close(t.cancelCh)
		}
		return nil
	}
	syncer := setupSyncer(source)
	if err := syncer.Sync(sourceAccountTrie.Hash(), cancel); err == nil {
		t.Fatal("No error returned from incomplete/cancelled sync")
	}
}

func setupSyncer(peers ...*testPeer) *Syncer {
	stateDb := rawdb.NewMemoryDatabase()
	syncer := NewSyncer(stateDb)
	for _, peer := range peers {
		syncer.Register(peer)
		peer.remote = syncer
	}
	return syncer
}

// TestSync tests a basic sync with one peer
func TestSync(t *testing.T) {
	t.Parallel()

	cancel := make(chan struct{})
	sourceAccountTrie, elems := makeAccountTrieNoStorage(100)

	mkSource := func(name string) *testPeer {
		source := newTestPeer(name, t, cancel)
		source.accountTrie = sourceAccountTrie
		source.accountValues = elems
		return source
	}

	syncer := setupSyncer(mkSource("sourceA"))
	if err := syncer.Sync(sourceAccountTrie.Hash(), cancel); err != nil {
		t.Fatalf("sync failed: %v", err)
	}
}

// TestSyncTinyTriePanic tests a basic sync with one peer, and a tiny trie. This caused a
// panic within the prover
func TestSyncTinyTriePanic(t *testing.T) {
	t.Parallel()

	cancel := make(chan struct{})

	sourceAccountTrie, elems := makeAccountTrieNoStorage(1)

	mkSource := func(name string) *testPeer {
		source := newTestPeer(name, t, cancel)
		source.accountTrie = sourceAccountTrie
		source.accountValues = elems
		return source
	}

	syncer := setupSyncer(
		mkSource("nice-a"),
	)
	done := checkStall(t, cancel)
	if err := syncer.Sync(sourceAccountTrie.Hash(), cancel); err != nil {
		t.Fatalf("sync failed: %v", err)
	}
	close(done)
}

// TestMultiSync tests a basic sync with multiple peers
func TestMultiSync(t *testing.T) {
	t.Parallel()

	cancel := make(chan struct{})
	sourceAccountTrie, elems := makeAccountTrieNoStorage(100)

	mkSource := func(name string) *testPeer {
		source := newTestPeer(name, t, cancel)
		source.accountTrie = sourceAccountTrie
		source.accountValues = elems
		return source
	}

	syncer := setupSyncer(mkSource("sourceA"), mkSource("sourceB"))
	if err := syncer.Sync(sourceAccountTrie.Hash(), cancel); err != nil {
		t.Fatalf("sync failed: %v", err)
	}
}

// TestSyncWithStorage tests  basic sync using accounts + storage + code
func TestSyncWithStorage(t *testing.T) {
	t.Parallel()

	cancel := make(chan struct{})
	sourceAccountTrie, elems, storageTries, storageElems := makeAccountTrieWithStorage(3, 3000, true)

	mkSource := func(name string) *testPeer {
		source := newTestPeer(name, t, cancel)
		source.accountTrie = sourceAccountTrie
		source.accountValues = elems
		source.storageTries = storageTries
		source.storageValues = storageElems
		return source
	}
	syncer := setupSyncer(mkSource("sourceA"))
	if err := syncer.Sync(sourceAccountTrie.Hash(), cancel); err != nil {
		t.Fatalf("sync failed: %v", err)
	}
}

// TestMultiSyncManyUseless contains one good peer, and many which doesn't return anything valuable at all
func TestMultiSyncManyUseless(t *testing.T) {
	t.Parallel()

	cancel := make(chan struct{})

	sourceAccountTrie, elems, storageTries, storageElems := makeAccountTrieWithStorage(100, 3000, true)

	mkSource := func(name string, a, b, c bool) *testPeer {
		source := newTestPeer(name, t, cancel)
		source.accountTrie = sourceAccountTrie
		source.accountValues = elems
		source.storageTries = storageTries
		source.storageValues = storageElems

		if !a {
			source.accountRequestHandler = emptyRequestAccountRangeFn
		}
		if !b {
			source.storageRequestHandler = emptyStorageRequestHandler
		}
		if !c {
			source.trieRequestHandler = emptyTrieRequestHandler
		}
		return source
	}

	syncer := setupSyncer(
		mkSource("full", true, true, true),
		mkSource("noAccounts", false, true, true),
		mkSource("noStorage", true, false, true),
		mkSource("noTrie", true, true, false),
	)
	if err := syncer.Sync(sourceAccountTrie.Hash(), cancel); err != nil {
		t.Fatalf("sync failed: %v", err)
	}
}

// TestMultiSyncManyUseless contains one good peer, and many which doesn't return anything valuable at all
func TestMultiSyncManyUselessWithLowTimeout(t *testing.T) {
	// We're setting the timeout to very low, to increase the chance of the timeout
	// being triggered. This was previously a cause of panic, when a response
	// arrived simultaneously as a timeout was triggered.
	defer func(old time.Duration) { requestTimeout = old }(requestTimeout)
	requestTimeout = time.Millisecond

	cancel := make(chan struct{})

	sourceAccountTrie, elems, storageTries, storageElems := makeAccountTrieWithStorage(100, 3000, true)

	mkSource := func(name string, a, b, c bool) *testPeer {
		source := newTestPeer(name, t, cancel)
		source.accountTrie = sourceAccountTrie
		source.accountValues = elems
		source.storageTries = storageTries
		source.storageValues = storageElems

		if !a {
			source.accountRequestHandler = emptyRequestAccountRangeFn
		}
		if !b {
			source.storageRequestHandler = emptyStorageRequestHandler
		}
		if !c {
			source.trieRequestHandler = emptyTrieRequestHandler
		}
		return source
	}

	syncer := setupSyncer(
		mkSource("full", true, true, true),
		mkSource("noAccounts", false, true, true),
		mkSource("noStorage", true, false, true),
		mkSource("noTrie", true, true, false),
	)
	if err := syncer.Sync(sourceAccountTrie.Hash(), cancel); err != nil {
		t.Fatalf("sync failed: %v", err)
	}
}

// TestMultiSyncManyUnresponsive contains one good peer, and many which doesn't respond at all
func TestMultiSyncManyUnresponsive(t *testing.T) {
	// We're setting the timeout to very low, to make the test run a bit faster
	defer func(old time.Duration) { requestTimeout = old }(requestTimeout)
	requestTimeout = time.Millisecond

	cancel := make(chan struct{})

	sourceAccountTrie, elems, storageTries, storageElems := makeAccountTrieWithStorage(100, 3000, true)

	mkSource := func(name string, a, b, c bool) *testPeer {
		source := newTestPeer(name, t, cancel)
		source.accountTrie = sourceAccountTrie
		source.accountValues = elems
		source.storageTries = storageTries
		source.storageValues = storageElems

		if !a {
			source.accountRequestHandler = nonResponsiveRequestAccountRangeFn
		}
		if !b {
			source.storageRequestHandler = nonResponsiveStorageRequestHandler
		}
		if !c {
			source.trieRequestHandler = nonResponsiveTrieRequestHandler
		}
		return source
	}

	syncer := setupSyncer(
		mkSource("full", true, true, true),
		mkSource("noAccounts", false, true, true),
		mkSource("noStorage", true, false, true),
		mkSource("noTrie", true, true, false),
	)
	if err := syncer.Sync(sourceAccountTrie.Hash(), cancel); err != nil {
		t.Fatalf("sync failed: %v", err)
	}
}

func checkStall(t *testing.T, cancel chan struct{}) chan struct{} {
	testDone := make(chan struct{})
	go func() {
		select {
		case <-time.After(time.Minute): // TODO(karalabe): Make tests smaller, this is too much
			t.Log("Sync stalled")
			close(cancel)
		case <-testDone:
			return
		}
	}()
	return testDone
}

// TestSyncNoStorageAndOneCappedPeer tests sync using accounts and no storage, where one peer is
// consistently returning very small results
func TestSyncNoStorageAndOneCappedPeer(t *testing.T) {
	t.Parallel()

	cancel := make(chan struct{})

	sourceAccountTrie, elems := makeAccountTrieNoStorage(3000)

	mkSource := func(name string, slow bool) *testPeer {
		source := newTestPeer(name, t, cancel)
		source.accountTrie = sourceAccountTrie
		source.accountValues = elems

		if slow {
			source.accountRequestHandler = starvingAccountRequestHandler
		}
		return source
	}

	syncer := setupSyncer(
		mkSource("nice-a", false),
		mkSource("nice-b", false),
		mkSource("nice-c", false),
		mkSource("capped", true),
	)
	done := checkStall(t, cancel)
	if err := syncer.Sync(sourceAccountTrie.Hash(), cancel); err != nil {
		t.Fatalf("sync failed: %v", err)
	}
	close(done)
}

// TestSyncNoStorageAndOneCodeCorruptPeer has one peer which doesn't deliver
// code requests properly.
func TestSyncNoStorageAndOneCodeCorruptPeer(t *testing.T) {
	t.Parallel()

	cancel := make(chan struct{})

	sourceAccountTrie, elems := makeAccountTrieNoStorage(3000)

	mkSource := func(name string, codeFn codeHandlerFunc) *testPeer {
		source := newTestPeer(name, t, cancel)
		source.accountTrie = sourceAccountTrie
		source.accountValues = elems
		source.codeRequestHandler = codeFn
		return source
	}
	// One is capped, one is corrupt. If we don't use a capped one, there's a 50%
	// chance that the full set of codes requested are sent only to the
	// non-corrupt peer, which delivers everything in one go, and makes the
	// test moot
	syncer := setupSyncer(
		mkSource("capped", cappedCodeRequestHandler),
		mkSource("corrupt", corruptCodeRequestHandler),
	)
	done := checkStall(t, cancel)
	if err := syncer.Sync(sourceAccountTrie.Hash(), cancel); err != nil {
		t.Fatalf("sync failed: %v", err)
	}
	close(done)
}

func TestSyncNoStorageAndOneAccountCorruptPeer(t *testing.T) {
	t.Parallel()

	cancel := make(chan struct{})

	sourceAccountTrie, elems := makeAccountTrieNoStorage(3000)

	mkSource := func(name string, accFn accountHandlerFunc) *testPeer {
		source := newTestPeer(name, t, cancel)
		source.accountTrie = sourceAccountTrie
		source.accountValues = elems
		source.accountRequestHandler = accFn
		return source
	}
	// One is capped, one is corrupt. If we don't use a capped one, there's a 50%
	// chance that the full set of codes requested are sent only to the
	// non-corrupt peer, which delivers everything in one go, and makes the
	// test moot
	syncer := setupSyncer(
		mkSource("capped", defaultAccountRequestHandler),
		mkSource("corrupt", corruptAccountRequestHandler),
	)
	done := checkStall(t, cancel)
	if err := syncer.Sync(sourceAccountTrie.Hash(), cancel); err != nil {
		t.Fatalf("sync failed: %v", err)
	}
	close(done)
}

// TestSyncNoStorageAndOneCodeCappedPeer has one peer which delivers code hashes
// one by one
func TestSyncNoStorageAndOneCodeCappedPeer(t *testing.T) {
	t.Parallel()

	cancel := make(chan struct{})

	sourceAccountTrie, elems := makeAccountTrieNoStorage(3000)

	mkSource := func(name string, codeFn codeHandlerFunc) *testPeer {
		source := newTestPeer(name, t, cancel)
		source.accountTrie = sourceAccountTrie
		source.accountValues = elems
		source.codeRequestHandler = codeFn
		return source
	}
	// Count how many times it's invoked. Remember, there are only 8 unique hashes,
	// so it shouldn't be more than that
	var counter int
	syncer := setupSyncer(
		mkSource("capped", func(t *testPeer, id uint64, hashes []common.Hash, max uint64) error {
			counter++
			return cappedCodeRequestHandler(t, id, hashes, max)
		}),
	)
	done := checkStall(t, cancel)
	if err := syncer.Sync(sourceAccountTrie.Hash(), cancel); err != nil {
		t.Fatalf("sync failed: %v", err)
	}
	close(done)
	// There are only 8 unique hashes, and 3K accounts. However, the code
	// deduplication is per request batch. If it were a perfect global dedup,
	// we would expect only 8 requests. If there were no dedup, there would be
	// 3k requests.
	// We expect somewhere below 100 requests for these 8 unique hashes.
	if threshold := 100; counter > threshold {
		t.Fatalf("Error, expected < %d invocations, got %d", threshold, counter)
	}
}

// TestSyncWithStorageAndOneCappedPeer tests sync using accounts + storage, where one peer is
// consistently returning very small results
func TestSyncWithStorageAndOneCappedPeer(t *testing.T) {
	t.Parallel()

	cancel := make(chan struct{})

	sourceAccountTrie, elems, storageTries, storageElems := makeAccountTrieWithStorage(300, 1000, false)

	mkSource := func(name string, slow bool) *testPeer {
		source := newTestPeer(name, t, cancel)
		source.accountTrie = sourceAccountTrie
		source.accountValues = elems
		source.storageTries = storageTries
		source.storageValues = storageElems

		if slow {
			source.storageRequestHandler = starvingStorageRequestHandler
		}
		return source
	}

	syncer := setupSyncer(
		mkSource("nice-a", false),
		mkSource("slow", true),
	)
	done := checkStall(t, cancel)
	if err := syncer.Sync(sourceAccountTrie.Hash(), cancel); err != nil {
		t.Fatalf("sync failed: %v", err)
	}
	close(done)
}

// TestSyncWithStorageAndCorruptPeer tests sync using accounts + storage, where one peer is
// sometimes sending bad proofs
func TestSyncWithStorageAndCorruptPeer(t *testing.T) {
	t.Parallel()

	cancel := make(chan struct{})

	sourceAccountTrie, elems, storageTries, storageElems := makeAccountTrieWithStorage(100, 3000, true)

	mkSource := func(name string, handler storageHandlerFunc) *testPeer {
		source := newTestPeer(name, t, cancel)
		source.accountTrie = sourceAccountTrie
		source.accountValues = elems
		source.storageTries = storageTries
		source.storageValues = storageElems
		source.storageRequestHandler = handler
		return source
	}

	syncer := setupSyncer(
		mkSource("nice-a", defaultStorageRequestHandler),
		mkSource("nice-b", defaultStorageRequestHandler),
		mkSource("nice-c", defaultStorageRequestHandler),
		mkSource("corrupt", corruptStorageRequestHandler),
	)
	done := checkStall(t, cancel)
	if err := syncer.Sync(sourceAccountTrie.Hash(), cancel); err != nil {
		t.Fatalf("sync failed: %v", err)
	}
	close(done)
}

func TestSyncWithStorageAndNonProvingPeer(t *testing.T) {
	t.Parallel()

	cancel := make(chan struct{})

	sourceAccountTrie, elems, storageTries, storageElems := makeAccountTrieWithStorage(100, 3000, true)

	mkSource := func(name string, handler storageHandlerFunc) *testPeer {
		source := newTestPeer(name, t, cancel)
		source.accountTrie = sourceAccountTrie
		source.accountValues = elems
		source.storageTries = storageTries
		source.storageValues = storageElems
		source.storageRequestHandler = handler
		return source
	}

	syncer := setupSyncer(
		mkSource("nice-a", defaultStorageRequestHandler),
		mkSource("nice-b", defaultStorageRequestHandler),
		mkSource("nice-c", defaultStorageRequestHandler),
		mkSource("corrupt", noProofStorageRequestHandler),
	)
	done := checkStall(t, cancel)
	if err := syncer.Sync(sourceAccountTrie.Hash(), cancel); err != nil {
		t.Fatalf("sync failed: %v", err)
	}
	close(done)
}

type kv struct {
	k, v []byte
	t    bool
}

// Some helpers for sorting
type entrySlice []*kv

func (p entrySlice) Len() int           { return len(p) }
func (p entrySlice) Less(i, j int) bool { return bytes.Compare(p[i].k, p[j].k) < 0 }
func (p entrySlice) Swap(i, j int)      { p[i], p[j] = p[j], p[i] }

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

// getACodeHash returns a pseudo-random code hash
func getACodeHash(i uint64) []byte {
	h := codehashes[int(i)%len(codehashes)]
	return common.CopyBytes(h[:])
}

// convenience function to lookup the code from the code hash
func getCode(hash common.Hash) []byte {
	if hash == emptyCode {
		return nil
	}
	for i, h := range codehashes {
		if h == hash {
			return []byte{byte(i)}
		}
	}
	return nil
}

// makeAccountTrieNoStorage spits out a trie, along with the leafs
func makeAccountTrieNoStorage(n int) (*trie.Trie, entrySlice) {
	db := trie.NewDatabase(rawdb.NewMemoryDatabase())
	accTrie, _ := trie.New(common.Hash{}, db)
	var entries entrySlice
	for i := uint64(1); i <= uint64(n); i++ {
		value, _ := rlp.EncodeToBytes(state.Account{
			Nonce:    i,
			Balance:  big.NewInt(int64(i)),
			Root:     emptyRoot,
			CodeHash: getACodeHash(i),
		})
		key := key32(i)
		elem := &kv{key, value, false}
		accTrie.Update(elem.k, elem.v)
		entries = append(entries, elem)
	}
	sort.Sort(entries)
	// Push to disk layer
	accTrie.Commit(nil)
	return accTrie, entries
}

// makeAccountTrieWithStorage spits out a trie, along with the leafs
func makeAccountTrieWithStorage(accounts, slots int, code bool) (*trie.Trie, entrySlice,
	map[common.Hash]*trie.Trie, map[common.Hash]entrySlice) {

	var (
		db             = trie.NewDatabase(rawdb.NewMemoryDatabase())
		accTrie, _     = trie.New(common.Hash{}, db)
		entries        entrySlice
		storageTries   = make(map[common.Hash]*trie.Trie)
		storageEntries = make(map[common.Hash]entrySlice)
	)

	// Make a storage trie which we reuse for the whole lot
	stTrie, stEntries := makeStorageTrie(slots, db)
	stRoot := stTrie.Hash()
	// Create n accounts in the trie
	for i := uint64(1); i <= uint64(accounts); i++ {
		key := key32(i)
		codehash := emptyCode[:]
		if code {
			codehash = getACodeHash(i)
		}
		value, _ := rlp.EncodeToBytes(state.Account{
			Nonce:    i,
			Balance:  big.NewInt(int64(i)),
			Root:     stRoot,
			CodeHash: codehash,
		})
		elem := &kv{key, value, false}
		accTrie.Update(elem.k, elem.v)
		entries = append(entries, elem)
		// we reuse the same one for all accounts
		storageTries[common.BytesToHash(key)] = stTrie
		storageEntries[common.BytesToHash(key)] = stEntries
	}
	sort.Sort(entries)
	stTrie.Commit(nil)
	accTrie.Commit(nil)
	return accTrie, entries, storageTries, storageEntries
}

// makeStorageTrie fills a storage trie with n items, returning the
// not-yet-committed trie and the sorted entries
func makeStorageTrie(n int, db *trie.Database) (*trie.Trie, entrySlice) {
	trie, _ := trie.New(common.Hash{}, db)
	var entries entrySlice
	for i := uint64(1); i <= uint64(n); i++ {
		// store 'i' at slot 'i'
		slotValue := key32(i)
		rlpSlotValue, _ := rlp.EncodeToBytes(common.TrimLeftZeroes(slotValue[:]))

		slotKey := key32(i)
		key := crypto.Keccak256Hash(slotKey[:])

		elem := &kv{key[:], rlpSlotValue, false}
		trie.Update(elem.k, elem.v)
		entries = append(entries, elem)
	}
	sort.Sort(entries)
	return trie, entries
}
