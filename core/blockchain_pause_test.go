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
	"bytes"
	"context"
	"errors"
	"io/fs"
	"math/big"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/consensus/beacon"
	"github.com/ethereum/go-ethereum/consensus/ethash"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/ethdb/memorydb"
	"github.com/ethereum/go-ethereum/params"
)

// syncedStore is an in-memory key-value store whose copies hold only what it
// held at the last SyncKeyValue: the least that a copy of the data directory
// holds of a Pebble store.
type syncedStore struct {
	ethdb.KeyValueStore
	lock   sync.Mutex
	synced map[string][]byte
}

func newSyncedStore() *syncedStore {
	return &syncedStore{KeyValueStore: memorydb.New(), synced: make(map[string][]byte)}
}

func (s *syncedStore) SyncKeyValue() error {
	s.lock.Lock()
	defer s.lock.Unlock()

	s.synced = make(map[string][]byte)
	it := s.NewIterator(nil, nil)
	defer it.Release()
	for it.Next() {
		s.synced[string(it.Key())] = common.CopyBytes(it.Value())
	}
	return it.Error()
}

// copy returns a key-value store holding the content as of the last sync.
func (s *syncedStore) copy() ethdb.KeyValueStore {
	s.lock.Lock()
	defer s.lock.Unlock()

	db := memorydb.New()
	for k, v := range s.synced {
		db.Put([]byte(k), v)
	}
	return db
}

var errSyncFailure = errors.New("sync failure")

// failingSyncs is a database whose syncs of the ancient or the key-value store
// fail while told to.
type failingSyncs struct {
	ethdb.Database
	ancient, keyValue atomic.Bool
}

func (db *failingSyncs) SyncAncient() error {
	if db.ancient.Load() {
		return errSyncFailure
	}
	return db.Database.SyncAncient()
}

func (db *failingSyncs) SyncKeyValue() error {
	if db.keyValue.Load() {
		return errSyncFailure
	}
	return db.Database.SyncKeyValue()
}

// newPauseTestChain generates a chain of the given length, sending ether to a
// new account in every block.
func newPauseTestChain(n int) (*Genesis, []*types.Block) {
	var (
		key, _  = crypto.HexToECDSA("b71c71a67e1177ad4e901695e1b4b9ee17ae16c6668d313eac2f96dbcda3f291")
		address = crypto.PubkeyToAddress(key.PublicKey)
		gspec   = &Genesis{
			Config:  params.TestChainConfig,
			Alloc:   types.GenesisAlloc{address: {Balance: big.NewInt(params.Ether)}},
			BaseFee: big.NewInt(params.InitialBaseFee),
		}
		signer = types.LatestSigner(gspec.Config)
	)
	_, blocks, _ := GenerateChainWithGenesis(gspec, beacon.New(ethash.NewFaker()), n, func(i int, block *BlockGen) {
		tx, err := types.SignTx(types.NewTransaction(block.TxNonce(address), common.Address{byte(i >> 8), byte(i), 0xff}, big.NewInt(1000), params.TxGas, block.header.BaseFee, nil), signer, key)
		if err != nil {
			panic(err)
		}
		block.AddTx(tx)
	})
	return gspec, blocks
}

// checkHeld checks that the chain mutex is held: LockContext takes a free one
// even with a context that is done.
func checkHeld(t *testing.T, chain *BlockChain) {
	t.Helper()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	err := chain.chainmu.LockContext(ctx)
	if err == nil {
		chain.chainmu.Unlock()
	}
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("The chain mutex isn't held, err: %v", err)
	}
}

// dirCopy is a copy of a chain's data directory, as a snapshot taken at the
// time holds it: the key-value store as of its last sync, and the files.
type dirCopy struct {
	kv      ethdb.KeyValueStore
	ancient string
	journal string
}

// withoutLockFiles leaves the freezers' FLOCK files out of a copy. Windows
// doesn't let another handle read a held one, and the copy's freezers create
// their own.
type withoutLockFiles struct{ fs.FS }

func (f withoutLockFiles) ReadDir(name string) ([]fs.DirEntry, error) {
	entries, err := fs.ReadDir(f.FS, name)
	return slices.DeleteFunc(entries, func(e fs.DirEntry) bool { return e.Name() == "FLOCK" }), err
}

// takeCopy copies the data directory of a chain on the given key-value store,
// ancient directory and journal directory, which may be empty.
func takeCopy(t *testing.T, store *syncedStore, ancient, journal string) *dirCopy {
	t.Helper()

	c := &dirCopy{kv: store.copy(), ancient: t.TempDir()}
	if err := os.CopyFS(c.ancient, withoutLockFiles{os.DirFS(ancient)}); err != nil {
		t.Fatalf("Failed to copy the ancient store: %v", err)
	}
	if journal != "" {
		c.journal = t.TempDir()
		if err := os.CopyFS(c.journal, os.DirFS(journal)); err != nil {
			t.Fatalf("Failed to copy the journal: %v", err)
		}
	}
	return c
}

// open opens a chain on the copy, configured as the chain it was taken from.
func (c *dirCopy) open(t *testing.T, gspec *Genesis, config *BlockChainConfig) (*BlockChain, ethdb.Database) {
	t.Helper()

	db, err := rawdb.Open(c.kv, rawdb.OpenOptions{Ancient: c.ancient})
	if err != nil {
		t.Fatalf("Failed to open the copy: %v", err)
	}
	cconfig := *config
	cconfig.TrieJournalDirectory = c.journal
	chain, err := NewBlockChain(db, gspec, beacon.New(ethash.NewFaker()), &cconfig)
	if err != nil {
		db.Close()
		t.Fatalf("Failed to open the chain of the copy: %v", err)
	}
	t.Cleanup(func() {
		chain.Stop()
		db.Close()
	})
	return chain, db
}

// TestPauseForClone checks that a copy of the data directory taken during a
// pause opens at the head of the pause, with its state and the preimages of its
// keys, while the chain writes wait for the pause to end.
func TestPauseForClone(t *testing.T) {
	var (
		gspec, blocks = newPauseTestChain(300)
		store         = newSyncedStore()
		ancient       = t.TempDir()
		journal       = t.TempDir()
	)
	db, err := rawdb.Open(store, rawdb.OpenOptions{Ancient: ancient})
	if err != nil {
		t.Fatalf("Failed to open the database: %v", err)
	}
	defer db.Close()

	// Keep a short state history, which makes the write buffer flush often, and
	// record the preimages of the trie keys.
	config := DefaultConfig().WithStateScheme(rawdb.PathScheme)
	config.TrieJournalDirectory = journal
	config.StateHistory = 32
	config.Preimages = true

	chain, err := NewBlockChain(db, gspec, beacon.New(ethash.NewFaker()), config)
	if err != nil {
		t.Fatalf("Failed to create the chain: %v", err)
	}
	defer chain.Stop()

	if _, err := chain.InsertChain(blocks[:250]); err != nil {
		t.Fatalf("Failed to insert the chain: %v", err)
	}
	p, err := chain.PauseForClone(context.Background())
	if err != nil {
		t.Fatalf("Failed to pause: %v", err)
	}
	if p.Head.Hash() != blocks[249].Hash() {
		t.Fatalf("Unexpected head of the pause: want %d, got %d", blocks[249].NumberU64(), p.Head.Number)
	}
	if rawdb.ReadPersistentStateID(db) == 0 {
		t.Fatal("No state was flushed before the pause")
	}
	// Copy the data directory as a snapshot taken now would.
	copied := takeCopy(t, store, ancient, journal)

	// The pause holds the chain mutex, which imports wait for.
	checkHeld(t, chain)
	p.Resume()
	if _, err := chain.InsertChain(blocks[250:]); err != nil {
		t.Fatalf("Failed to import after the pause: %v", err)
	}
	// The copy opens at the head of the pause, and goes on from there.
	clone, cdb := copied.open(t, gspec, config)
	if head := clone.CurrentBlock(); head.Hash() != p.Head.Hash() {
		t.Fatalf("The copy opened at block %d, not at the head of the pause %d", head.Number, p.Head.Number)
	}
	statedb, err := clone.StateAt(p.Head.Root, p.Head.Number, p.Head.Time)
	if err != nil {
		t.Fatalf("No state at the head of the copy: %v", err)
	}
	recipient := common.Address{0, 249, 0xff}
	if balance := statedb.GetBalance(recipient); balance.Uint64() != 1000 {
		t.Fatalf("Unexpected balance at the head of the copy: %v", balance)
	}
	if preimage := rawdb.ReadPreimage(cdb, crypto.Keccak256Hash(recipient.Bytes())); !bytes.Equal(preimage, recipient.Bytes()) {
		t.Fatalf("The copy lacks the preimage of %x", recipient)
	}
	if _, err := clone.InsertChain(blocks[250:]); err != nil {
		t.Fatalf("Failed to import on the copy: %v", err)
	}
}

// TestPauseForCloneResumeCrash checks that a crash after a pause ended opens at
// the head of the pause, with the journal written for it, until blocks after the
// pause flush state; then the journal no longer matches, and the crash opens
// from the persisted state. A crash right after a pause that failed to sync the
// key-value store opens with the file journal that pause wrote.
func TestPauseForCloneResumeCrash(t *testing.T) {
	t.Run("file", func(t *testing.T) { testPauseForCloneResumeCrash(t, t.TempDir()) })
	t.Run("key-value", func(t *testing.T) { testPauseForCloneResumeCrash(t, "") })
}

func testPauseForCloneResumeCrash(t *testing.T, journal string) {
	var (
		gspec, blocks = newPauseTestChain(300)
		store         = newSyncedStore()
		ancient       = t.TempDir()
	)
	diskdb, err := rawdb.Open(store, rawdb.OpenOptions{Ancient: ancient})
	if err != nil {
		t.Fatalf("Failed to open the database: %v", err)
	}
	defer diskdb.Close()
	db := &failingSyncs{Database: diskdb}

	config := DefaultConfig().WithStateScheme(rawdb.PathScheme)
	config.TrieJournalDirectory = journal
	config.StateHistory = 32

	chain, err := NewBlockChain(db, gspec, beacon.New(ethash.NewFaker()), config)
	if err != nil {
		t.Fatalf("Failed to create the chain: %v", err)
	}
	defer chain.Stop()

	if _, err := chain.InsertChain(blocks[:250]); err != nil {
		t.Fatalf("Failed to insert the chain: %v", err)
	}
	p, err := chain.PauseForClone(context.Background())
	if err != nil {
		t.Fatalf("Failed to pause: %v", err)
	}
	p.Resume()

	// The journal of the pause stays, also once the key-value store is synced
	// and after blocks that flush no state.
	openAtPause := func() {
		t.Helper()
		if err := db.SyncKeyValue(); err != nil {
			t.Fatalf("Failed to sync: %v", err)
		}
		clone, _ := takeCopy(t, store, ancient, journal).open(t, gspec, config)
		if head := clone.CurrentBlock(); head.Hash() != p.Head.Hash() {
			t.Fatalf("The copy opened at block %d, not at the head of the pause %d", head.Number, p.Head.Number)
		}
		if _, err := clone.InsertChain(blocks[250:]); err != nil {
			t.Fatalf("Failed to import on the copy: %v", err)
		}
	}
	openAtPause()
	persisted := rawdb.ReadPersistentStateID(db)
	if _, err := chain.InsertChain(blocks[250:252]); err != nil {
		t.Fatalf("Failed to import after the pause: %v", err)
	}
	if rawdb.ReadPersistentStateID(db) != persisted {
		t.Fatal("The blocks after the pause flushed state")
	}
	openAtPause()

	// A pause that fails to sync the key-value store leaves its file journal,
	// with a layer above the synced head, and a crash opens at that head.
	if journal != "" {
		if _, err := chain.InsertChain(blocks[252:253]); err != nil {
			t.Fatalf("Failed to import after the pause: %v", err)
		}
		db.keyValue.Store(true)
		if _, err := chain.PauseForClone(context.Background()); !errors.Is(err, errSyncFailure) {
			t.Fatalf("Unexpected error of a pause failing to sync: %v", err)
		}
		db.keyValue.Store(false)
		if rawdb.ReadPersistentStateID(db) != persisted {
			t.Fatal("The blocks after the pause flushed state")
		}
		clone, _ := takeCopy(t, store, ancient, journal).open(t, gspec, config)
		if head := clone.CurrentBlock(); head.Hash() != blocks[251].Hash() {
			t.Fatalf("The copy opened at block %d, not at the synced head %d", head.Number, blocks[251].NumberU64())
		}
		if _, err := clone.InsertChain(blocks[252:]); err != nil {
			t.Fatalf("Failed to import on the copy: %v", err)
		}
	}
	// Blocks that flush state, with its histories: the journal no longer
	// matches it.
	if _, err := chain.InsertChain(blocks[252:280]); err != nil {
		t.Fatalf("Failed to import after the pause: %v", err)
	}
	if rawdb.ReadPersistentStateID(db) == persisted {
		t.Fatal("No state was flushed after the pause")
	}
	if err := db.SyncKeyValue(); err != nil {
		t.Fatalf("Failed to sync: %v", err)
	}
	clone, _ := takeCopy(t, store, ancient, journal).open(t, gspec, config)
	checkRecovered(t, clone, blocks[279].Header(), blocks)
}

// checkRecovered checks that a copy opened below the given head, at a block
// with state, and imports the given blocks from there.
func checkRecovered(t *testing.T, chain *BlockChain, head *types.Header, blocks []*types.Block) {
	t.Helper()

	current := chain.CurrentBlock()
	if current.Number.Cmp(head.Number) >= 0 {
		t.Fatalf("The copy opened at block %d, not below %d", current.Number, head.Number)
	}
	if !chain.HasState(current.Root) {
		t.Fatalf("No state at the head %d of the copy", current.Number)
	}
	if _, err := chain.InsertChain(blocks[current.Number.Uint64():]); err != nil {
		t.Fatalf("Failed to import on the copy: %v", err)
	}
}

// TestPauseForCloneStop checks that stopping the chain ends a pause and writes
// the journal of the shutdown, and that a resume after it has no effect.
func TestPauseForCloneStop(t *testing.T) {
	var (
		gspec, blocks = newPauseTestChain(10)
		journal       = t.TempDir()
	)
	db, err := rawdb.Open(rawdb.NewMemoryDatabase(), rawdb.OpenOptions{Ancient: t.TempDir()})
	if err != nil {
		t.Fatalf("Failed to open the database: %v", err)
	}
	defer db.Close()

	config := DefaultConfig().WithStateScheme(rawdb.PathScheme)
	config.TrieJournalDirectory = journal
	chain, err := NewBlockChain(db, gspec, beacon.New(ethash.NewFaker()), config)
	if err != nil {
		t.Fatalf("Failed to create the chain: %v", err)
	}
	if _, err := chain.InsertChain(blocks); err != nil {
		t.Fatalf("Failed to insert the chain: %v", err)
	}
	p, err := chain.PauseForClone(context.Background())
	if err != nil {
		t.Fatalf("Failed to pause: %v", err)
	}
	stopped := make(chan struct{})
	go func() {
		chain.Stop()
		close(stopped)
	}()
	select {
	case <-stopped:
	case <-time.After(10 * time.Second):
		t.Fatal("The pause blocks the chain from stopping")
	}
	if !common.FileExist(filepath.Join(journal, "merkle.journal")) {
		t.Fatal("The journal of the shutdown is missing")
	}
	p.Resume()
	if _, err := chain.PauseForClone(context.Background()); !errors.Is(err, errChainStopped) {
		t.Fatalf("Unexpected error pausing a stopped chain: %v", err)
	}
}

// TestPauseForCloneWait checks that a pause during another one ends with its
// context, that one is taken after it, and that neither a chain syncing its
// state nor the hash scheme can pause.
func TestPauseForCloneWait(t *testing.T) {
	gspec, blocks := newPauseTestChain(10)

	chain, err := NewBlockChain(rawdb.NewMemoryDatabase(), gspec, beacon.New(ethash.NewFaker()), DefaultConfig().WithStateScheme(rawdb.PathScheme))
	if err != nil {
		t.Fatalf("Failed to create the chain: %v", err)
	}
	defer chain.Stop()

	if _, err := chain.InsertChain(blocks); err != nil {
		t.Fatalf("Failed to insert the chain: %v", err)
	}
	p, err := chain.PauseForClone(context.Background())
	if err != nil {
		t.Fatalf("Failed to pause: %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := chain.PauseForClone(ctx); !errors.Is(err, context.Canceled) {
		t.Fatalf("Unexpected error pausing during a pause: %v", err)
	}
	p.Resume()
	if p, err = chain.PauseForClone(context.Background()); err != nil {
		t.Fatalf("Failed to pause after a pause: %v", err)
	}
	p.Resume()

	// A chain syncing its state refuses, and leaves the chain mutex free.
	if err := chain.triedb.Disable(); err != nil {
		t.Fatalf("Failed to start syncing: %v", err)
	}
	if _, err := chain.PauseForClone(context.Background()); err == nil {
		t.Fatal("A chain syncing its state paused")
	}
	ctx, cancel = context.WithCancel(context.Background())
	cancel()
	if err := chain.chainmu.LockContext(ctx); err != nil {
		t.Fatalf("The refused pause holds the chain mutex: %v", err)
	}
	chain.chainmu.Unlock()

	hashchain, err := NewBlockChain(rawdb.NewMemoryDatabase(), gspec, beacon.New(ethash.NewFaker()), DefaultConfig())
	if err != nil {
		t.Fatalf("Failed to create the chain: %v", err)
	}
	defer hashchain.Stop()
	if _, err := hashchain.PauseForClone(context.Background()); err == nil {
		t.Fatal("The hash scheme paused")
	}
}

// blockingSync is a database whose key-value syncs wait for release while it is
// armed. waiting is closed when the first of them starts waiting.
type blockingSync struct {
	ethdb.Database
	armed   atomic.Bool
	once    sync.Once
	waiting chan struct{}
	release chan struct{}
}

func newBlockingSync(db ethdb.Database) *blockingSync {
	return &blockingSync{Database: db, waiting: make(chan struct{}), release: make(chan struct{})}
}

func (db *blockingSync) SyncKeyValue() error {
	if db.armed.Load() {
		db.once.Do(func() { close(db.waiting) })
		<-db.release
	}
	return db.Database.SyncKeyValue()
}

// importOrHold imports the given blocks in background, and reports whether that
// ended within 10 s, which it doesn't while the chain writes are held.
func importOrHold(chain *BlockChain, blocks []*types.Block) (bool, error) {
	imported := make(chan error, 1)
	go func() {
		_, err := chain.InsertChain(blocks)
		imported <- err
	}()
	select {
	case err := <-imported:
		return true, err
	case <-time.After(10 * time.Second):
		return false, nil
	}
}

// newBlockingPauseChain creates a chain on the path scheme, on a database whose
// key-value syncs can be held, and imports the given blocks.
func newBlockingPauseChain(t *testing.T, gspec *Genesis, blocks []*types.Block) (*BlockChain, *blockingSync, string) {
	t.Helper()

	diskdb, err := rawdb.Open(rawdb.NewMemoryDatabase(), rawdb.OpenOptions{Ancient: t.TempDir()})
	if err != nil {
		t.Fatalf("Failed to open the database: %v", err)
	}
	t.Cleanup(func() { diskdb.Close() })
	db := newBlockingSync(diskdb)

	config := DefaultConfig().WithStateScheme(rawdb.PathScheme)
	config.TrieJournalDirectory = t.TempDir()
	chain, err := NewBlockChain(db, gspec, beacon.New(ethash.NewFaker()), config)
	if err != nil {
		t.Fatalf("Failed to create the chain: %v", err)
	}
	if _, err := chain.InsertChain(blocks); err != nil {
		t.Fatalf("Failed to insert the chain: %v", err)
	}
	return chain, db, config.TrieJournalDirectory
}

// waitPreparing waits until a pause makes the key-value store consistent.
func waitPreparing(t *testing.T, db *blockingSync, paused chan error) {
	t.Helper()

	select {
	case <-db.waiting:
	case err := <-paused:
		t.Fatalf("The pause ended before the key-value sync, err: %v", err)
	case <-time.After(10 * time.Second):
		t.Fatal("The pause didn't reach the key-value sync")
	}
}

// TestPauseForCloneCancel checks that a pause whose context is done before it
// holds the chain mutex isn't prepared, and that one whose context is done while
// the database is made consistent is given up once it is, and the chain writes
// go on.
func TestPauseForCloneCancel(t *testing.T) {
	gspec, blocks := newPauseTestChain(20)
	chain, db, _ := newBlockingPauseChain(t, gspec, blocks[:10])
	defer func() {
		if chain != nil {
			chain.Stop()
		}
	}()
	db.armed.Store(true)

	// With its context done already, the pause doesn't reach the sync.
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	paused := make(chan error, 1)
	go func() {
		_, err := chain.PauseForClone(ctx)
		paused <- err
	}()
	select {
	case err := <-paused:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("Unexpected error of a pause cancelled before it began: %v", err)
		}
	case <-db.waiting:
		chain = nil // stopping would wait for the writes too
		t.Fatal("A pause cancelled before it began made the database consistent")
	}
	ctx, cancel = context.WithCancel(context.Background())
	go func() {
		_, err := chain.PauseForClone(ctx)
		paused <- err
	}()
	waitPreparing(t, db, paused)
	cancel()
	db.armed.Store(false)
	close(db.release)

	if err := <-paused; !errors.Is(err, context.Canceled) {
		t.Fatalf("Unexpected error of a cancelled pause: %v", err)
	}
	if ok, err := importOrHold(chain, blocks[10:]); !ok {
		chain = nil // stopping would wait for the writes too
		t.Fatal("The cancelled pause holds the chain writes")
	} else if err != nil {
		t.Fatalf("Failed to import after the cancelled pause: %v", err)
	}
}

// TestPauseForCloneStopPreparing checks that stopping the chain while a pause
// makes the database consistent waits for that, then ends the pause and writes
// the journal of the shutdown.
func TestPauseForCloneStopPreparing(t *testing.T) {
	gspec, blocks := newPauseTestChain(10)
	chain, db, journal := newBlockingPauseChain(t, gspec, blocks)

	db.armed.Store(true)
	paused := make(chan error, 1)
	go func() {
		_, err := chain.PauseForClone(context.Background())
		paused <- err
	}()
	waitPreparing(t, db, paused)
	if chain.clonePause.Load() != nil {
		t.Fatal("A pause getting ready is registered for Stop to end")
	}
	// Stop ends the chain's subscriptions once it is under way, then waits for
	// the pause, which holds the chain mutex.
	sub := chain.SubscribeChainHeadEvent(make(chan ChainHeadEvent, 1))
	stopped := make(chan struct{})
	go func() {
		chain.Stop()
		close(stopped)
	}()
	select {
	case <-sub.Err():
	case <-time.After(10 * time.Second):
		t.Fatal("The chain didn't start stopping")
	}
	checkHeld(t, chain)
	db.armed.Store(false)
	close(db.release)

	if err := <-paused; !errors.Is(err, errChainStopped) {
		t.Fatalf("Unexpected error of a pause ended by a stop: %v", err)
	}
	select {
	case <-stopped:
	case <-time.After(10 * time.Second):
		t.Fatal("The chain didn't stop after the pause ended")
	}
	if !common.FileExist(filepath.Join(journal, "merkle.journal")) {
		t.Fatal("The journal of the shutdown is missing")
	}
}

// TestPauseForCloneFailure checks that a pause failing on its way to a consistent
// database releases the chain writes, and that the chain goes on: it imports and
// pauses again.
func TestPauseForCloneFailure(t *testing.T) {
	for _, step := range []string{"journal", "journal sync", "ancient sync", "key-value sync"} {
		t.Run(step, func(t *testing.T) { testPauseForCloneFailure(t, step) })
	}
}

func testPauseForCloneFailure(t *testing.T, step string) {
	// The journal file can't be created in its directory, or the directory
	// can't be opened for its sync once the journal is renamed into it.
	var mode os.FileMode
	switch step {
	case "journal":
		mode = 0500
	case "journal sync":
		mode = 0300
	}
	if mode != 0 && (runtime.GOOS == "windows" || os.Geteuid() == 0) {
		t.Skip("Directory permissions aren't enforced")
	}
	var (
		gspec, blocks = newPauseTestChain(20)
		journal       = t.TempDir()
	)
	diskdb, err := rawdb.Open(rawdb.NewMemoryDatabase(), rawdb.OpenOptions{Ancient: t.TempDir()})
	if err != nil {
		t.Fatalf("Failed to open the database: %v", err)
	}
	defer diskdb.Close()
	db := &failingSyncs{Database: diskdb}

	config := DefaultConfig().WithStateScheme(rawdb.PathScheme)
	config.TrieJournalDirectory = journal
	chain, err := NewBlockChain(db, gspec, beacon.New(ethash.NewFaker()), config)
	if err != nil {
		t.Fatalf("Failed to create the chain: %v", err)
	}
	defer func() {
		if chain != nil {
			chain.Stop()
		}
	}()
	if _, err := chain.InsertChain(blocks[:10]); err != nil {
		t.Fatalf("Failed to insert the chain: %v", err)
	}
	if mode != 0 {
		if err := os.Chmod(journal, mode); err != nil {
			t.Fatalf("Failed to change the journal directory: %v", err)
		}
		t.Cleanup(func() { os.Chmod(journal, 0700) })
	}
	db.ancient.Store(step == "ancient sync")
	db.keyValue.Store(step == "key-value sync")

	_, err = chain.PauseForClone(context.Background())
	if err == nil {
		t.Fatal("Paused with the step failing")
	}
	if mode == 0 && !errors.Is(err, errSyncFailure) {
		t.Fatalf("Unexpected error of the pause: %v", err)
	}
	if err := os.Chmod(journal, 0700); err != nil {
		t.Fatalf("Failed to change the journal directory: %v", err)
	}
	db.ancient.Store(false)
	db.keyValue.Store(false)

	// The chain writes go on, and the chain pauses again.
	if ok, err := importOrHold(chain, blocks[10:15]); !ok {
		chain = nil // stopping would wait for the writes too
		t.Fatal("The failed pause holds the chain writes")
	} else if err != nil {
		t.Fatalf("Failed to import after the failed pause: %v", err)
	}
	p, err := chain.PauseForClone(context.Background())
	if err != nil {
		t.Fatalf("Failed to pause after the failed pause: %v", err)
	}
	if p.Head.Hash() != blocks[14].Hash() {
		t.Fatalf("Unexpected head of the pause: want %d, got %d", blocks[14].NumberU64(), p.Head.Number)
	}
	p.Resume()
	if _, err := chain.InsertChain(blocks[15:]); err != nil {
		t.Fatalf("Failed to import after the pause: %v", err)
	}
}
