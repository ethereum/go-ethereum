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
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/

package pathdb

import (
	"bytes"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/internal/testrand"
	"github.com/ethereum/go-ethereum/trie/trienode"
	"github.com/ethereum/go-ethereum/triedb/database"
	"github.com/holiman/uint256"
)

func waitIndexing(db *Database) {
	for {
		// The metadata is persisted before the indexer reports inited, so check both
		metadata := loadIndexMetadata(db.diskdb, typeStateHistory)
		if metadata != nil && metadata.Last >= db.tree.bottom().stateID() && db.stateIndexer.inited() {
			return
		}
		time.Sleep(100 * time.Millisecond)
	}
}

func stateAvail(id uint64, env *tester) bool {
	if env.db.config.StateHistory == 0 {
		return true
	}
	dl := env.db.tree.bottom()
	if dl.stateID() <= env.db.config.StateHistory {
		return true
	}
	firstID := dl.stateID() - env.db.config.StateHistory + 1

	return id+1 >= firstID
}

func checkHistoricalState(env *tester, root common.Hash, id uint64, hr *stateHistoryReader) error {
	if !stateAvail(id, env) {
		return nil
	}

	// Short circuit if the historical state is no longer available
	if rawdb.ReadStateID(env.db.diskdb, root) == nil {
		return fmt.Errorf("state not found %d %x", id, root)
	}

	var (
		dl       = env.db.tree.bottom()
		stateID  = rawdb.ReadStateID(env.db.diskdb, root)
		accounts = env.snapAccounts[root]
		storages = env.snapStorages[root]
	)
	for addrHash, accountData := range accounts {
		latest, _ := dl.account(addrHash, 0)
		blob, err := hr.read(newAccountIdentQuery(env.accountPreimage(addrHash), addrHash), *stateID, dl.stateID(), latest)
		if err != nil {
			return err
		}
		if !bytes.Equal(accountData, blob) {
			return fmt.Errorf("wrong account data, expected %x, got %x", accountData, blob)
		}
	}
	for i := 0; i < len(env.roots); i++ {
		if env.roots[i] == root {
			break
		}
		// Find all accounts deleted in the past, ensure the associated data is null
		for addrHash := range env.snapAccounts[env.roots[i]] {
			if _, ok := accounts[addrHash]; !ok {
				latest, _ := dl.account(addrHash, 0)
				blob, err := hr.read(newAccountIdentQuery(env.accountPreimage(addrHash), addrHash), *stateID, dl.stateID(), latest)
				if err != nil {
					return err
				}
				if len(blob) != 0 {
					return fmt.Errorf("wrong account data, expected null, got %x", blob)
				}
			}
		}
	}
	for addrHash, slots := range storages {
		for slotHash, slotData := range slots {
			latest, _ := dl.storage(addrHash, slotHash, 0)
			blob, err := hr.read(newStorageIdentQuery(env.accountPreimage(addrHash), addrHash, env.hashPreimage(slotHash), slotHash), *stateID, dl.stateID(), latest)
			if err != nil {
				return err
			}
			if !bytes.Equal(slotData, blob) {
				return fmt.Errorf("wrong storage data, expected %x, got %x", slotData, blob)
			}
		}
	}
	for i := 0; i < len(env.roots); i++ {
		if env.roots[i] == root {
			break
		}
		// Find all storage slots deleted in the past, ensure the associated data is null
		for addrHash, slots := range env.snapStorages[env.roots[i]] {
			for slotHash := range slots {
				_, ok := storages[addrHash]
				if ok {
					_, ok = storages[addrHash][slotHash]
				}
				if !ok {
					latest, _ := dl.storage(addrHash, slotHash, 0)
					blob, err := hr.read(newStorageIdentQuery(env.accountPreimage(addrHash), addrHash, env.hashPreimage(slotHash), slotHash), *stateID, dl.stateID(), latest)
					if err != nil {
						return err
					}
					if len(blob) != 0 {
						return fmt.Errorf("wrong storage data, expected null, got %x", blob)
					}
				}
			}
		}
	}
	return nil
}

func TestHistoryReader(t *testing.T) {
	testHistoryReader(t, 0)  // with all histories reserved
	testHistoryReader(t, 10) // with latest 10 histories reserved
}

func testHistoryReader(t *testing.T, historyLimit uint64) {
	maxDiffLayers = 4
	defer func() {
		maxDiffLayers = 128
	}()

	//log.SetDefault(log.NewLogger(log.NewTerminalHandlerWithLevel(os.Stderr, log.LevelDebug, true)))
	config := &testerConfig{
		stateHistory: historyLimit,
		layers:       64,
		enableIndex:  true,
	}
	env := newTester(t, config)
	defer env.release()
	waitIndexing(env.db)

	var (
		roots = env.roots
		dl    = env.db.tree.bottom()
		hr    = newStateHistoryReader(env.db.diskdb, env.db.stateFreezer, nil)
	)
	for i, root := range roots {
		if root == dl.rootHash() {
			break
		}
		if err := checkHistoricalState(env, root, uint64(i+1), hr); err != nil {
			t.Fatal(err)
		}
	}

	// Pile up more histories on top, ensuring the historic reader is not affected
	env.extend(4)
	waitIndexing(env.db)

	for i, root := range roots {
		if root == dl.rootHash() {
			break
		}
		if err := checkHistoricalState(env, root, uint64(i+1), hr); err != nil {
			t.Fatal(err)
		}
	}
}

func TestHistoricalStateReader(t *testing.T) {
	maxDiffLayers = 4
	defer func() {
		maxDiffLayers = 128
	}()

	//log.SetDefault(log.NewLogger(log.NewTerminalHandlerWithLevel(os.Stderr, log.LevelDebug, true)))
	config := &testerConfig{
		stateHistory: 0,
		layers:       64,
		enableIndex:  true,
	}
	env := newTester(t, config)
	defer env.release()
	waitIndexing(env.db)

	// non-canonical state
	fakeRoot := testrand.Hash()
	rawdb.WriteStateID(env.db.diskdb, fakeRoot, 10)

	_, err := env.db.HistoricReader(fakeRoot)
	if err == nil {
		t.Fatal("expected error")
	}
	t.Log(err)

	// canonical state
	realRoot := env.roots[9]
	_, err = env.db.HistoricReader(realRoot)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
}

// TestHistoricalStateReaderPendingWrites ensures the historical reads wait for
// the histories written in background, rather than failing while the disk layer
// is ahead of the indexed histories, and are served correctly while the disk
// layer is behind.
func TestHistoricalStateReaderPendingWrites(t *testing.T) {
	maxDiffLayers = 4
	defer func() {
		maxDiffLayers = 128
	}()

	// Disable the buffer flushes, which wait for the stalled writer below
	writeBuffer := 64 * 1024 * 1024
	env := newTester(t, &testerConfig{layers: 32, enableIndex: true, writeBuffer: &writeBuffer})
	defer env.release()
	waitIndexing(env.db)

	root := env.roots[9]
	reader, err := env.db.HistoricReader(root)
	if err != nil {
		t.Fatalf("Failed to construct the reader: %v", err)
	}
	// Stall the background writer and commit a new layer, leaving the disk layer
	// ahead of the indexed histories.
	var (
		stall  = make(chan struct{})
		resume = sync.OnceFunc(func() { close(stall) })
	)
	defer resume() // runs before the release, which waits for the writer

	if err := env.db.writer.submit(func() error { <-stall; return nil }); err != nil {
		t.Fatalf("Failed to stall the writer: %v", err)
	}
	env.extend(1)
	if meta := loadIndexMetadata(env.db.diskdb, typeStateHistory); meta.Last >= env.db.tree.bottom().stateID() {
		t.Fatalf("Unexpected indexed histories, indexed: %d, disk: %d", meta.Last, env.db.tree.bottom().stateID())
	}
	time.AfterFunc(100*time.Millisecond, resume)

	for addrHash, want := range env.snapAccounts[root] {
		blob, err := reader.AccountRLP(env.accountPreimage(addrHash))
		if err != nil {
			t.Fatalf("Failed to read the historical account: %v", err)
		}
		if !bytes.Equal(blob, want) {
			t.Fatalf("Unexpected account data, want: %x, got: %x", want, blob)
		}
	}
	// Commit more layers after resolving the latest values of the disk layer,
	// leaving it behind the indexed histories.
	var (
		dl     = env.db.tree.bottom()
		latest = make(map[common.Hash][]byte)
	)
	for addrHash := range env.snapAccounts[root] {
		latest[addrHash], _ = dl.account(addrHash, 0)
	}
	env.extend(2)
	if err := env.db.waitWrites(); err != nil {
		t.Fatalf("Failed to write histories: %v", err)
	}
	hr := newStateHistoryReader(env.db.diskdb, env.db.stateFreezer, nil)
	for addrHash, want := range env.snapAccounts[root] {
		blob, err := hr.read(newAccountIdentQuery(env.accountPreimage(addrHash), addrHash), reader.id, dl.stateID(), latest[addrHash])
		if err != nil {
			t.Fatalf("Failed to read the historical account: %v", err)
		}
		if !bytes.Equal(blob, want) {
			t.Fatalf("Unexpected account data, want: %x, got: %x", want, blob)
		}
	}
}

// TestHistoricalStateReaderStaleDiskLayerRetry ensures that when the bottom disk
// layer is replaced (and marked stale) by concurrent block commits,
// HistoricalStateReader transparently re-reads from the updated bottom layer
// instead of failing with errSnapshotStale.
func TestHistoricalStateReaderStaleDiskLayerRetry(t *testing.T) {
	maxDiffLayers = 2
	defer func() {
		maxDiffLayers = 128
	}()

	env := newTester(t, &testerConfig{layers: 16, enableIndex: true})
	defer env.release()
	waitIndexing(env.db)

	root := env.roots[5]

	// Capture test accounts and storage slots in local slices so readers don't touch
	// tester maps concurrently while the writer extends the chain.
	type acctEntry struct {
		addr common.Address
		want []byte
	}
	type storageEntry struct {
		addr common.Address
		key  common.Hash
		want []byte
	}
	var (
		accounts []acctEntry
		storages []storageEntry
	)
	for addrHash, want := range env.snapAccounts[root] {
		addr := env.accountPreimage(addrHash)
		accounts = append(accounts, acctEntry{addr: addr, want: want})
	}
	for addrHash, storageMap := range env.snapStorages[root] {
		addr := env.accountPreimage(addrHash)
		for storageHash, want := range storageMap {
			slotKey := env.hashPreimage(storageHash)
			storages = append(storages, storageEntry{addr: addr, key: slotKey, want: want})
		}
	}

	var (
		stop    = make(chan struct{})
		errc    = make(chan error, 8)
		wg      sync.WaitGroup
		commits = 50
	)

	// Writer goroutine: repeatedly extends the chain past maxDiffLayers, forcing
	// disk layer merges and marking the replaced bottom disk layer stale.
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < commits; i++ {
			env.extend(1)
		}
		close(stop)
	}()

	// Reader goroutines: continuously query historical accounts and storage slots
	// while the writer is committing and invalidating disk layers.
	for i := 0; i < 4; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			reader, err := env.db.HistoricReader(root)
			if err != nil {
				select {
				case errc <- fmt.Errorf("HistoricReader: %w", err):
				default:
				}
				return
			}
			for {
				select {
				case <-stop:
					return
				default:
				}
				for _, entry := range accounts {
					blob, err := reader.AccountRLP(entry.addr)
					if err != nil {
						select {
						case errc <- fmt.Errorf("AccountRLP(%x): %w", entry.addr, err):
						default:
						}
						return
					}
					if !bytes.Equal(blob, entry.want) {
						select {
						case errc <- fmt.Errorf("unexpected account data for %x", entry.addr):
						default:
						}
						return
					}
				}
				for _, entry := range storages {
					blob, err := reader.Storage(entry.addr, entry.key)
					if err != nil {
						select {
						case errc <- fmt.Errorf("Storage(%x, %x): %w", entry.addr, entry.key, err):
						default:
						}
						return
					}
					if !bytes.Equal(blob, entry.want) {
						select {
						case errc <- fmt.Errorf("unexpected storage data for %x, %x", entry.addr, entry.key):
						default:
						}
						return
					}
				}
			}
		}()
	}

	wg.Wait()
	close(errc)
	for err := range errc {
		t.Fatal(err)
	}

	// Ensure that if the database is disabled (bottom layer marked stale without
	// replacement), the reader returns errSnapshotStale rather than spinning indefinitely.
	if err := env.db.Disable(); err != nil {
		t.Fatalf("Failed to disable database: %v", err)
	}
	disabledReader, err := env.db.HistoricReader(root)
	if err != nil {
		t.Fatalf("Failed to construct reader: %v", err)
	}
	if _, err := disabledReader.AccountRLP(accounts[0].addr); !errors.Is(err, errSnapshotStale) {
		t.Fatalf("Expected errSnapshotStale on disabled database, got: %v", err)
	}
	if _, err := disabledReader.Storage(storages[0].addr, storages[0].key); !errors.Is(err, errSnapshotStale) {
		t.Fatalf("Expected errSnapshotStale on disabled database, got: %v", err)
	}
	disabledNodeReader := &HistoricalNodeReader{db: env.db, id: 1}
	if _, err := disabledNodeReader.Node(common.Hash{}, nil, common.Hash{}); !errors.Is(err, errSnapshotStale) {
		t.Fatalf("Expected errSnapshotStale on disabled database for Node, got: %v", err)
	}
}

// TestHistoricalStateReaderRollback verifies that a HistoricalStateReader
// returns an error after a rollback rather than answering from another state:
//   - After a rollback below the reader's state, reads fail instead of
//     returning the value at the new disk layer.
//   - After replacement histories reuse the state IDs and re-index as far, a
//     reader with a cached index reader fails instead of answering from the
//     discarded index, while a newly opened reader returns the canonical value.
func TestHistoricalStateReaderRollback(t *testing.T) {
	disk, err := rawdb.Open(rawdb.NewMemoryDatabase(), rawdb.OpenOptions{Ancient: t.TempDir()})
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer disk.Close()

	db := New(disk, &Config{
		EnableStateIndexing: true,
		TrieCleanSize:       256 * 1024,
		StateCleanSize:      256 * 1024,
		WriteBufferSize:     256 * 1024,
		NoAsyncFlush:        true,
		NoHistoryIndexDelay: true,
	})
	defer db.Close()
	waitIndexing(db)

	var (
		addr  = common.Address{0x01}
		other = common.Address{0x02}
		slot  = common.Hash{0x01}

		accounts = make(map[common.Address]types.StateAccount)
		storages = make(map[common.Address]map[common.Hash][]byte)
		root     = types.EmptyRootHash
	)
	applyBlock := func(block uint64, balances map[common.Address]uint64, slots map[common.Address]map[common.Hash][]byte) common.Hash {
		var (
			nodes          = trienode.NewMergedNodeSet()
			trieAccounts   = make(map[common.Hash][]byte)
			accountUpdates = make(map[common.Hash][]byte)
			storageUpdates = make(map[common.Hash]map[common.Hash][]byte)
			accountOrigin  = make(map[common.Address][]byte)
			storageOrigin  = make(map[common.Address]map[common.Hash][]byte)
		)
		for a, bal := range balances {
			addrHash := crypto.Keccak256Hash(a.Bytes())
			prev, exists := accounts[a]
			if !exists {
				prev = types.StateAccount{
					Root:     types.EmptyRootHash,
					CodeHash: types.EmptyCodeHash.Bytes(),
				}
				accountOrigin[a] = nil
			} else {
				accountOrigin[a] = types.SlimAccountRLP(prev)
			}
			stRoot := prev.Root
			if slotChanges, ok := slots[a]; ok {
				if storages[a] == nil {
					storages[a] = make(map[common.Hash][]byte)
				}
				storageUpdates[addrHash] = make(map[common.Hash][]byte)
				storageOrigin[a] = make(map[common.Hash][]byte)
				for k, v := range slotChanges {
					kHash := crypto.Keccak256Hash(k.Bytes())
					storageOrigin[a][k] = storages[a][k]
					storages[a][k] = v
					storageUpdates[addrHash][kHash] = v
				}
				var stNodes *trienode.NodeSet
				stRoot, stNodes = updateTrie(db, root, addrHash, prev.Root, storageUpdates[addrHash])
				nodes.Merge(stNodes)
			}
			next := prev
			next.Balance = uint256.NewInt(bal)
			next.Root = stRoot
			accounts[a] = next
			accountUpdates[addrHash] = types.SlimAccountRLP(next)
			full, err := types.FullAccountRLP(accountUpdates[addrHash])
			if err != nil {
				t.Fatalf("Failed to encode full account: %v", err)
			}
			trieAccounts[addrHash] = full
		}
		newRoot, acctNodes := updateTrie(db, root, common.Hash{}, root, trieAccounts)
		nodes.Merge(acctNodes)

		stateSet := NewStateSetWithOrigin(accountUpdates, storageUpdates, accountOrigin, storageOrigin, true)
		if err := db.Update(newRoot, root, block, nodes, stateSet); err != nil {
			t.Fatalf("Failed to update block %d: %v", block, err)
		}
		if err := db.Commit(newRoot, false); err != nil {
			t.Fatalf("Failed to commit block %d: %v", block, err)
		}
		if err := db.waitWrites(); err != nil {
			t.Fatalf("Failed to wait writes for block %d: %v", block, err)
		}
		root = newRoot
		return newRoot
	}

	// Block 1 sets addr balance to 101 (and slot to 0x01); blocks 2 and 3 keep it
	// unchanged; block 4 sets addr balance to 150 (and slot to 0x02).
	root1 := applyBlock(1, map[common.Address]uint64{addr: 101, other: 1}, map[common.Address]map[common.Hash][]byte{addr: {slot: {0x01}}})
	root2 := applyBlock(2, map[common.Address]uint64{other: 2}, nil)
	root3 := applyBlock(3, map[common.Address]uint64{other: 3}, nil)
	_ = applyBlock(4, map[common.Address]uint64{addr: 150}, map[common.Address]map[common.Hash][]byte{addr: {slot: {0x02}}})
	waitIndexing(db)

	// Open a reader at state 1 and read addr and slot, caching their index readers.
	reader1, err := db.HistoricReader(root1)
	if err != nil {
		t.Fatalf("Failed to open reader at state 1: %v", err)
	}
	acct, err := reader1.Account(addr)
	if err != nil {
		t.Fatalf("Failed to read account at state 1: %v", err)
	}
	if got := acct.Balance.Uint64(); got != 101 {
		t.Fatalf("Unexpected balance at state 1: got %d, want 101", got)
	}
	val, err := reader1.Storage(addr, slot)
	if err != nil {
		t.Fatalf("Failed to read storage at state 1: %v", err)
	}
	if !bytes.Equal(val, []byte{0x01}) {
		t.Fatalf("Unexpected storage at state 1: got %x, want 01", val)
	}

	// Also open a reader at state 3 before the rollback without reading yet.
	reader3, err := db.HistoricReader(root3)
	if err != nil {
		t.Fatalf("Failed to open reader at state 3: %v", err)
	}

	// Roll back to state 2 (below reader3's state 3, above reader1's state 1).
	if err := db.Recover(root2); err != nil {
		t.Fatalf("Failed to recover to state 2: %v", err)
	}
	accounts[addr] = types.StateAccount{
		Balance:  uint256.NewInt(101),
		Root:     accounts[addr].Root,
		CodeHash: types.EmptyCodeHash.Bytes(),
	}
	storages[addr][slot] = []byte{0x01}
	// Restore accounts[addr].Root to state 1/2's storage root.
	state2Acct, err := types.FullAccount(rawdb.ReadAccountSnapshot(disk, crypto.Keccak256Hash(addr.Bytes())))
	if err != nil {
		t.Fatalf("Failed to decode state 2 account: %v", err)
	}
	accounts[addr] = *state2Acct
	state2Other, err := types.FullAccount(rawdb.ReadAccountSnapshot(disk, crypto.Keccak256Hash(other.Bytes())))
	if err != nil {
		t.Fatalf("Failed to decode state 2 other account: %v", err)
	}
	accounts[other] = *state2Other
	root = root2

	// Case 1: reader3's target state (3) is now above the disk layer (2).
	// Reads must fail rather than returning the disk layer's state 2 value.
	if _, err := reader3.Account(other); !errors.Is(err, database.ErrStateUnavailable) {
		t.Fatalf("Expected ErrStateUnavailable from reader3.Account after rollback, got: %v", err)
	}
	if _, err := reader3.Storage(addr, slot); !errors.Is(err, database.ErrStateUnavailable) {
		t.Fatalf("Expected ErrStateUnavailable from reader3.Storage after rollback, got: %v", err)
	}
	// Direct stateHistoryReader check: checkStateAvail rejects stateID > lastID.
	hr := newStateHistoryReader(db.diskdb, db.stateFreezer, nil)
	otherHash := crypto.Keccak256Hash(other.Bytes())
	latestOther, _ := db.tree.bottom().account(otherHash, 0)
	if _, err := hr.read(newAccountIdentQuery(other, otherHash), 3, db.tree.bottom().stateID(), latestOther); !errors.Is(err, database.ErrStateUnavailable) {
		t.Fatalf("Expected ErrStateUnavailable from checkStateAvail when stateID > lastID, got: %v", err)
	}

	// Pre-rollback reader1 must also fail immediately after the rollback.
	if _, err := reader1.Account(addr); !errors.Is(err, database.ErrStateUnavailable) {
		t.Fatalf("Expected ErrStateUnavailable from reader1.Account after rollback, got: %v", err)
	}
	if _, err := reader1.Storage(addr, slot); !errors.Is(err, database.ErrStateUnavailable) {
		t.Fatalf("Expected ErrStateUnavailable from reader1.Storage after rollback, got: %v", err)
	}

	// Apply replacement blocks 3 and 4 setting addr balance to 205, then 300
	// (and slot to 0x03, then 0x04).
	root3b := applyBlock(3, map[common.Address]uint64{addr: 205}, map[common.Address]map[common.Hash][]byte{addr: {slot: {0x03}}})
	_ = applyBlock(4, map[common.Address]uint64{addr: 300}, map[common.Address]map[common.Hash][]byte{addr: {slot: {0x04}}})
	waitIndexing(db)

	// Case 2: Replacement histories are now indexed up to state 4 again.
	// Pre-rollback reader1 must still fail rather than answering 205 / 0x03
	// from its cached pre-rollback index.
	if acct, err := reader1.Account(addr); !errors.Is(err, database.ErrStateUnavailable) {
		var got uint64
		if acct != nil {
			got = acct.Balance.Uint64()
		}
		t.Fatalf("Expected ErrStateUnavailable from pre-rollback reader1.Account, got balance=%d, err=%v", got, err)
	}
	if val, err := reader1.Storage(addr, slot); !errors.Is(err, database.ErrStateUnavailable) {
		t.Fatalf("Expected ErrStateUnavailable from pre-rollback reader1.Storage, got val=%x, err=%v", val, err)
	}

	// A newly opened reader at state 1 must succeed and answer 101 and 0x01.
	newReader1, err := db.HistoricReader(root1)
	if err != nil {
		t.Fatalf("Failed to open new reader at state 1: %v", err)
	}
	acct, err = newReader1.Account(addr)
	if err != nil {
		t.Fatalf("Failed to read account from new reader at state 1: %v", err)
	}
	if got := acct.Balance.Uint64(); got != 101 {
		t.Fatalf("Unexpected balance from new reader at state 1: got %d, want 101", got)
	}
	val, err = newReader1.Storage(addr, slot)
	if err != nil {
		t.Fatalf("Failed to read storage from new reader at state 1: %v", err)
	}
	if !bytes.Equal(val, []byte{0x01}) {
		t.Fatalf("Unexpected storage from new reader at state 1: got %x, want 01", val)
	}

	// A newly opened reader at replacement state 3 must answer 205 and 0x03.
	newReader3, err := db.HistoricReader(root3b)
	if err != nil {
		t.Fatalf("Failed to open new reader at replacement state 3: %v", err)
	}
	acct, err = newReader3.Account(addr)
	if err != nil {
		t.Fatalf("Failed to read account from new reader at state 3: %v", err)
	}
	if got := acct.Balance.Uint64(); got != 205 {
		t.Fatalf("Unexpected balance from new reader at state 3: got %d, want 205", got)
	}
	val, err = newReader3.Storage(addr, slot)
	if err != nil {
		t.Fatalf("Failed to read storage from new reader at state 3: %v", err)
	}
	if !bytes.Equal(val, []byte{0x03}) {
		t.Fatalf("Unexpected storage from new reader at state 3: got %x, want 03", val)
	}
}

// syncHookedKV wraps a KeyValueStore and runs an optional hook during SyncKeyValue.
type syncHookedKV struct {
	ethdb.KeyValueStore
	onSync func()
}

func (s *syncHookedKV) SyncKeyValue() error {
	if s.onSync != nil {
		s.onSync()
	}
	return s.KeyValueStore.SyncKeyValue()
}

// TestHistoricalStateReaderDuringRollback verifies that:
//  1. Opening a HistoricReader or reading from an existing reader while a
//     rollback is actively in progress fails with ErrStateUnavailable.
//  2. A read that started before a rollback and is in flight when a rollback
//     runs is rejected by the post-read rollback counter check.
func TestHistoricalStateReaderDuringRollback(t *testing.T) {
	maxDiffLayers = 4
	defer func() {
		maxDiffLayers = 128
	}()

	var (
		inSync      = make(chan struct{})
		releaseSync = make(chan struct{})
		hookedKV    = &syncHookedKV{KeyValueStore: rawdb.NewMemoryDatabase()}
	)
	env := newTester(t, &testerConfig{
		layers:      16,
		enableIndex: true,
		kvdb:        hookedKV,
	})
	defer env.release()
	waitIndexing(env.db)

	var (
		root1      = env.roots[0]
		targetRoot = env.roots[env.bottomIndex()-1]
		testAddr   common.Address
		testSlot   common.Hash
	)
	for addrHash, slots := range env.snapStorages[root1] {
		testAddr = env.accountPreimage(addrHash)
		for slotHash := range slots {
			testSlot = env.hashPreimage(slotHash)
			break
		}
		break
	}

	reader, err := env.db.HistoricReader(root1)
	if err != nil {
		t.Fatalf("Failed to open reader before rollback: %v", err)
	}

	// Pause Recover inside SyncKeyValue while the rollback counter is odd.
	var once sync.Once
	hookedKV.onSync = func() {
		once.Do(func() {
			close(inSync)
			<-releaseSync
		})
	}

	recoverErr := make(chan error, 1)
	go func() {
		recoverErr <- env.db.Recover(targetRoot)
	}()

	<-inSync
	// While Recover is in progress, creating a new reader or reading from an
	// existing one must fail with ErrStateUnavailable.
	if _, err := env.db.HistoricReader(root1); !errors.Is(err, database.ErrStateUnavailable) {
		t.Fatalf("Expected ErrStateUnavailable from HistoricReader during rollback, got: %v", err)
	}
	if _, err := reader.AccountRLP(testAddr); !errors.Is(err, database.ErrStateUnavailable) {
		t.Fatalf("Expected ErrStateUnavailable from AccountRLP during rollback, got: %v", err)
	}
	if _, err := reader.Storage(testAddr, testSlot); !errors.Is(err, database.ErrStateUnavailable) {
		t.Fatalf("Expected ErrStateUnavailable from Storage during rollback, got: %v", err)
	}
	close(releaseSync)
	if err := <-recoverErr; err != nil {
		t.Fatalf("Recover failed: %v", err)
	}
	hookedKV.onSync = nil
	waitIndexing(env.db)

	// Next, verify the post-read check when a rollback occurs mid-read, after the
	// pre-read check and disk layer lookup have already passed.
	reader2, err := env.db.HistoricReader(root1)
	if err != nil {
		t.Fatalf("Failed to open reader after first rollback: %v", err)
	}
	dlID := env.db.tree.bottom().stateID()
	nextTarget := env.roots[env.bottomIndex()-1]
	reader2.reader.wait = func() error {
		storeIndexMetadata(env.db.diskdb, typeStateHistory, dlID)
		if err := env.db.Recover(nextTarget); err != nil {
			return err
		}
		// Bump metadata back to dlID so checkStateAvail and r.reader.read succeed,
		// ensuring the post-read rollback check is what rejects the read.
		storeIndexMetadata(env.db.diskdb, typeStateHistory, dlID)
		return nil
	}
	storeIndexMetadata(env.db.diskdb, typeStateHistory, dlID-1)
	if _, err := reader2.AccountRLP(testAddr); !errors.Is(err, database.ErrStateUnavailable) {
		t.Fatalf("Expected ErrStateUnavailable when rollback occurs during AccountRLP, got: %v", err)
	}
	storeIndexMetadata(env.db.diskdb, typeStateHistory, env.db.tree.bottom().stateID())

	// Repeat for Storage mid-read rollback.
	reader3, err := env.db.HistoricReader(root1)
	if err != nil {
		t.Fatalf("Failed to open reader for storage mid-read test: %v", err)
	}
	dlID = env.db.tree.bottom().stateID()
	nextTarget2 := env.roots[env.bottomIndex()-1]
	reader3.reader.wait = func() error {
		storeIndexMetadata(env.db.diskdb, typeStateHistory, dlID)
		if err := env.db.Recover(nextTarget2); err != nil {
			return err
		}
		storeIndexMetadata(env.db.diskdb, typeStateHistory, dlID)
		return nil
	}
	storeIndexMetadata(env.db.diskdb, typeStateHistory, dlID-1)
	if _, err := reader3.Storage(testAddr, testSlot); !errors.Is(err, database.ErrStateUnavailable) {
		t.Fatalf("Expected ErrStateUnavailable when rollback occurs during Storage, got: %v", err)
	}
	storeIndexMetadata(env.db.diskdb, typeStateHistory, env.db.tree.bottom().stateID())
}
