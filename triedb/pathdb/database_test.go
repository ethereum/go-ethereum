// Copyright 2022 The go-ethereum Authors
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

package pathdb

import (
	"bytes"
	"errors"
	"fmt"
	"math/rand"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/internal/testrand"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/ethereum/go-ethereum/trie"
	"github.com/ethereum/go-ethereum/trie/trienode"
	"github.com/ethereum/go-ethereum/trie/utils"
	"github.com/ethereum/go-ethereum/triedb/database"
	"github.com/holiman/uint256"
)

func updateMerkleTrie(db *Database, stateRoot common.Hash, addrHash common.Hash, root common.Hash, dirties map[common.Hash][]byte) (common.Hash, *trienode.NodeSet) {
	var id *trie.ID
	if addrHash == (common.Hash{}) {
		id = trie.StateTrieID(stateRoot)
	} else {
		id = trie.StorageTrieID(stateRoot, addrHash, root)
	}
	tr, err := trie.New(id, db)
	if err != nil {
		panic(fmt.Errorf("failed to load trie, err: %w", err))
	}
	for key, val := range dirties {
		if len(val) == 0 {
			tr.Delete(key.Bytes())
		} else {
			tr.Update(key.Bytes(), val)
		}
	}
	return tr.Commit(false)
}

func generateAccount(storageRoot common.Hash, codeHash []byte) types.StateAccount {
	return types.StateAccount{
		Nonce:    uint64(rand.Intn(100)),
		Balance:  uint256.NewInt(rand.Uint64()),
		CodeHash: codeHash,
		Root:     storageRoot,
	}
}

const (
	createAccountOp int = iota
	modifyAccountOp
	deleteAccountOp
	opLen
)

type genctx struct {
	stateRoot     common.Hash                               // Parent state root
	accounts      map[common.Hash][]byte                    // Keyed by the hash of account address
	storages      map[common.Hash]map[common.Hash][]byte    // Keyed by the hash of account address and the hash of storage key
	codes         map[common.Hash][]byte                    // Keyed by the contract code hash
	accountOrigin map[common.Address][]byte                 // Keyed by the account address
	storageOrigin map[common.Address]map[common.Hash][]byte // Keyed by the account address and the hash of storage key
	nodes         *trienode.MergedNodeSet                   // Aggregated dirty trie nodes

	// Verkle hasher fields
	isVerkle bool
	tr       *trie.VerkleTrie
}

func newCtx(stateRoot common.Hash, db database.NodeDatabase, isVerkle bool) *genctx {
	ctx := &genctx{
		isVerkle:      isVerkle,
		stateRoot:     stateRoot,
		accounts:      make(map[common.Hash][]byte),
		storages:      make(map[common.Hash]map[common.Hash][]byte),
		codes:         make(map[common.Hash][]byte),
		accountOrigin: make(map[common.Address][]byte),
		storageOrigin: make(map[common.Address]map[common.Hash][]byte),
		nodes:         trienode.NewMergedNodeSet(),
	}
	if isVerkle {
		tr, err := trie.NewVerkleTrie(stateRoot, db, utils.NewPointCache(1024))
		if err != nil {
			panic(fmt.Errorf("failed to load trie, err: %w", err))
		}
		ctx.tr = tr
	}
	return ctx
}

func (ctx *genctx) storageOriginSet(rawStorageKey bool, t *tester) map[common.Address]map[common.Hash][]byte {
	if !rawStorageKey {
		return ctx.storageOrigin
	}
	set := make(map[common.Address]map[common.Hash][]byte)
	for addr, storage := range ctx.storageOrigin {
		subset := make(map[common.Hash][]byte)
		for hash, val := range storage {
			key := t.hashPreimage(hash)
			subset[key] = val
		}
		set[addr] = subset
	}
	return set
}

type tester struct {
	disk      ethdb.Database
	db        *Database
	roots     []common.Hash
	preimages map[common.Hash][]byte

	// current state set
	accounts map[common.Hash][]byte                 // Keyed by the hash of account address
	storages map[common.Hash]map[common.Hash][]byte // Keyed by the hash of account address and the hash of storage key

	// state snapshots
	snapAccounts map[common.Hash]map[common.Hash][]byte                 // Keyed by the hash of account address
	snapStorages map[common.Hash]map[common.Hash]map[common.Hash][]byte // Keyed by the hash of account address and the hash of storage key
}

func newTester(t *testing.T, historyLimit uint64, isVerkle bool, layers int) *tester {
	var (
		disk, _ = rawdb.NewDatabaseWithFreezer(rawdb.NewMemoryDatabase(), t.TempDir(), "", false)
		db      = New(disk, &Config{
			StateHistory:    historyLimit,
			CleanCacheSize:  256 * 1024,
			WriteBufferSize: 256 * 1024,
		}, isVerkle)

		obj = &tester{
			disk:         disk,
			db:           db,
			preimages:    make(map[common.Hash][]byte),
			accounts:     make(map[common.Hash][]byte),
			storages:     make(map[common.Hash]map[common.Hash][]byte),
			snapAccounts: make(map[common.Hash]map[common.Hash][]byte),
			snapStorages: make(map[common.Hash]map[common.Hash]map[common.Hash][]byte),
		}
	)
	for i := 0; i < layers; i++ {
		var parent = types.EmptyRootHash
		if isVerkle {
			parent = types.EmptyVerkleHash
		}
		if len(obj.roots) != 0 {
			parent = obj.roots[len(obj.roots)-1]
		}
		// raw storage key is required in verkle for rollback
		rawStorageKey := i > 6
		if isVerkle {
			rawStorageKey = true
		}
		root, nodes, states := obj.generate(parent, rawStorageKey, isVerkle)

		if err := db.Update(root, parent, uint64(i), nodes, states); err != nil {
			panic(fmt.Errorf("failed to update state changes, err: %w", err))
		}
		obj.roots = append(obj.roots, root)
	}
	return obj
}

func (t *tester) accountPreimage(hash common.Hash) common.Address {
	return common.BytesToAddress(t.preimages[hash])
}

func (t *tester) hashPreimage(hash common.Hash) common.Hash {
	return common.BytesToHash(t.preimages[hash])
}

func (t *tester) release() {
	t.db.Close()
	t.db.diskdb.Close()
}

func (t *tester) randAccount() (common.Address, []byte, []byte) {
	for addrHash, account := range t.accounts {
		acct, err := types.FullAccount(account)
		if err != nil {
			panic(fmt.Errorf("failed to decode account, %v", err))
		}
		return t.accountPreimage(addrHash), account, acct.CodeHash
	}
	return common.Address{}, nil, nil
}

func (t *tester) generateCode(ctx *genctx, addr common.Address) common.Hash {
	size := rand.Intn(128 * 1024)
	code := testrand.Bytes(size)
	hash := crypto.Keccak256Hash(code)
	ctx.codes[hash] = code

	if ctx.isVerkle {
		if err := ctx.tr.UpdateContractCode(addr, hash, code); err != nil {
			panic(fmt.Errorf("failed to update contract code, %v", err))
		}
	}
	return hash
}

// generateStorage inserts a batch of new storage slots for the specified account.
// The storage is assumed to be empty before insertion.
func (t *tester) generateStorage(ctx *genctx, addr common.Address) common.Hash {
	var (
		addrHash = crypto.Keccak256Hash(addr.Bytes())
		storage  = make(map[common.Hash][]byte)
		origin   = make(map[common.Hash][]byte)
	)
	for i := 0; i < 10; i++ {
		v, _ := rlp.EncodeToBytes(common.TrimLeftZeroes(testrand.Bytes(32)))
		key := testrand.Bytes(32)
		hash := crypto.Keccak256Hash(key)
		t.preimages[hash] = key

		storage[hash] = v
		origin[hash] = nil
	}

	ctx.storages[addrHash] = storage
	ctx.storageOrigin[addr] = origin

	// Generate a merkle storage trie for the newly constructed storage.
	if !ctx.isVerkle {
		root, set := updateMerkleTrie(t.db, ctx.stateRoot, addrHash, types.EmptyRootHash, storage)
		ctx.nodes.Merge(set)
		return root
	}
	// Insert the storage slots in the global verkle trie
	for key, val := range storage {
		if err := ctx.tr.UpdateStorage(addr, t.preimages[key], val); err != nil {
			panic(fmt.Errorf("failed to update storage, %v", err))
		}
	}
	return common.Hash{}
}

// mutateStorage modifies existing storage slots by deleting/updating some and
// creating new ones, simulating a typical storage mutation.
func (t *tester) mutateStorage(ctx *genctx, addr common.Address, root common.Hash) common.Hash {
	var (
		deletes int
		updates int

		addrHash = crypto.Keccak256Hash(addr.Bytes())
		storage  = make(map[common.Hash][]byte)
		origin   = make(map[common.Hash][]byte)
	)
	for hash, val := range t.storages[addrHash] {
		if rand.Intn(2) == 0 {
			origin[hash] = val
			storage[hash] = nil
			deletes++
		} else {
			origin[hash] = val
			v, _ := rlp.EncodeToBytes(common.TrimLeftZeroes(testrand.Bytes(32)))
			storage[hash] = v
			updates++
		}
		if deletes >= 3 && updates >= 3 {
			break
		}
	}
	for i := 0; i < deletes; i++ {
		v, _ := rlp.EncodeToBytes(common.TrimLeftZeroes(testrand.Bytes(32)))
		key := testrand.Bytes(32)
		hash := crypto.Keccak256Hash(key)
		t.preimages[hash] = key

		storage[hash] = v
		origin[hash] = nil
	}

	ctx.storages[addrHash] = storage
	ctx.storageOrigin[addr] = origin

	// Update the merkle storage trie with the generated mutation set
	if !ctx.isVerkle {
		root, set := updateMerkleTrie(t.db, ctx.stateRoot, crypto.Keccak256Hash(addr.Bytes()), root, storage)
		ctx.nodes.Merge(set)
		return root
	}
	// Apply the storage mutation into the global verkle trie
	for key, val := range storage {
		if len(val) != 0 {
			if err := ctx.tr.UpdateStorage(addr, t.preimages[key], val); err != nil {
				panic(fmt.Errorf("failed to update storage, %v", err))
			}
		} else {
			// TODO(rjl493456442) here the zero markers are written instead of
			// wiping the nodes from the trie.
			if err := ctx.tr.DeleteStorage(addr, t.preimages[key]); err != nil {
				panic(fmt.Errorf("failed to delete storage, %v", err))
			}
		}
	}
	return common.Hash{}
}

// clearStorage removes the existing storage slots belonging to the specified account.
func (t *tester) clearStorage(ctx *genctx, addr common.Address, root common.Hash) common.Hash {
	var (
		addrHash = crypto.Keccak256Hash(addr.Bytes())
		storage  = make(map[common.Hash][]byte)
		origin   = make(map[common.Hash][]byte)
	)
	for hash, val := range t.storages[addrHash] {
		origin[hash] = val
		storage[hash] = nil
	}
	ctx.storages[addrHash] = storage
	ctx.storageOrigin[addr] = origin

	// Clear the merkle storage trie with the generated deletion set
	if !ctx.isVerkle {
		root, set := updateMerkleTrie(t.db, ctx.stateRoot, addrHash, root, storage)
		if root != types.EmptyRootHash {
			panic("failed to clear storage trie")
		}
		ctx.nodes.Merge(set)
		return root
	}
	// Apply the storage deletion into the global verkle trie
	for key := range storage {
		// TODO(rjl493456442) here the zero markers are written instead of
		// wiping the nodes from the trie.
		if err := ctx.tr.DeleteStorage(addr, t.preimages[key]); err != nil {
			panic(fmt.Errorf("failed to delete storage, %v", err))
		}
	}
	return common.Hash{}
}

func (t *tester) generate(parent common.Hash, rawStorageKey bool, isVerkle bool) (common.Hash, *trienode.MergedNodeSet, *StateSetWithOrigin) {
	var (
		ctx     = newCtx(parent, t.db, isVerkle)
		dirties = make(map[common.Hash]struct{})
	)
	for i := 0; i < 20; i++ {
		// Start with account creation always
		op := createAccountOp
		if i > 0 {
			op = rand.Intn(opLen)
		}
		switch op {
		case createAccountOp:
			// account creation
			addr := testrand.Address()
			addrHash := crypto.Keccak256Hash(addr.Bytes())

			// Short circuit if the account was already existent
			if _, ok := t.accounts[addrHash]; ok {
				continue
			}
			// Short circuit if the account has been modified within the same transition
			if _, ok := dirties[addrHash]; ok {
				continue
			}
			dirties[addrHash] = struct{}{}

			root := t.generateStorage(ctx, addr)
			codeHash := t.generateCode(ctx, addr)
			ctx.accounts[addrHash] = types.SlimAccountRLP(generateAccount(root, codeHash.Bytes()))
			ctx.accountOrigin[addr] = nil
			t.preimages[addrHash] = addr.Bytes()

		case modifyAccountOp:
			// account mutation
			addr, account, codeHash := t.randAccount()
			if addr == (common.Address{}) {
				continue
			}
			addrHash := crypto.Keccak256Hash(addr.Bytes())

			// short circuit if the account has been modified within the same transition
			if _, ok := dirties[addrHash]; ok {
				continue
			}
			dirties[addrHash] = struct{}{}

			acct, _ := types.FullAccount(account)
			stRoot := t.mutateStorage(ctx, addr, acct.Root)
			newAccount := types.SlimAccountRLP(generateAccount(stRoot, codeHash)) // TODO support contract code rewrite

			ctx.accounts[addrHash] = newAccount
			ctx.accountOrigin[addr] = account

		case deleteAccountOp:
			// account deletion
			addr, account, _ := t.randAccount()
			if addr == (common.Address{}) {
				continue
			}
			addrHash := crypto.Keccak256Hash(addr.Bytes())

			// short circuit if the account has been modified within the same transition
			if _, ok := dirties[addrHash]; ok {
				continue
			}
			dirties[addrHash] = struct{}{}

			acct, _ := types.FullAccount(account)
			if acct.Root != types.EmptyRootHash {
				t.clearStorage(ctx, addr, acct.Root)
			}
			ctx.accounts[addrHash] = nil
			ctx.accountOrigin[addr] = account
		}
	}
	// Save state snapshot before commit
	t.snapAccounts[parent] = copyAccounts(t.accounts)
	t.snapStorages[parent] = copyStorages(t.storages)

	// Commit all changes to live state set
	for addrHash, account := range ctx.accounts {
		if len(account) == 0 {
			delete(t.accounts, addrHash)
		} else {
			t.accounts[addrHash] = account
		}
	}
	for addrHash, slots := range ctx.storages {
		if _, ok := t.storages[addrHash]; !ok {
			t.storages[addrHash] = make(map[common.Hash][]byte)
		}
		for sHash, slot := range slots {
			if len(slot) == 0 {
				delete(t.storages[addrHash], sHash)
			} else {
				t.storages[addrHash][sHash] = slot
			}
		}
		if len(t.storages[addrHash]) == 0 {
			delete(t.storages, addrHash)
		}
	}
	// Flush the contract code changes into the key value store
	for codeHash, code := range ctx.codes {
		rawdb.WriteCode(t.disk, codeHash, code)
	}
	// Flush the pending account changes into the global merkle trie
	storageOrigin := ctx.storageOriginSet(rawStorageKey, t)
	if !ctx.isVerkle {
		root, set := updateMerkleTrie(t.db, parent, common.Hash{}, parent, ctx.accounts)
		ctx.nodes.Merge(set)
		return root, ctx.nodes, NewStateSetWithOrigin(ctx.accounts, ctx.storages, ctx.accountOrigin, storageOrigin, rawStorageKey)
	}
	// Flush the pending account changes into the global verkle trie
	for addrHash, account := range ctx.accounts {
		if len(account) == 0 {
			// TODO(rjl493456442) here the zero markers are written instead of
			// wiping the nodes from the trie.
			ctx.tr.DeleteAccount(common.BytesToAddress(t.preimages[addrHash]))
		} else {
			ctx.tr.UpdateAccount(common.BytesToAddress(t.preimages[addrHash]), nil, 0)
		}
	}
	root, nodes := ctx.tr.Commit(false)
	merged := trienode.NewMergedNodeSet()
	merged.Merge(nodes)
	return root, merged, NewStateSetWithOrigin(ctx.accounts, ctx.storages, ctx.accountOrigin, storageOrigin, rawStorageKey)
}

// lastHash returns the latest root hash, or empty if nothing is cached.
func (t *tester) lastHash() common.Hash {
	if len(t.roots) == 0 {
		return common.Hash{}
	}
	return t.roots[len(t.roots)-1]
}

func (t *tester) verifyMerkleState(root common.Hash) error {
	tr, err := trie.New(trie.StateTrieID(root), t.db)
	if err != nil {
		return err
	}
	for addrHash, account := range t.snapAccounts[root] {
		blob, err := tr.Get(addrHash.Bytes())
		if err != nil || !bytes.Equal(blob, account) {
			return fmt.Errorf("account is mismatched: %w", err)
		}
	}
	for addrHash, slots := range t.snapStorages[root] {
		blob := t.snapAccounts[root][addrHash]
		if len(blob) == 0 {
			return fmt.Errorf("account %x is missing", addrHash)
		}
		account := new(types.StateAccount)
		if err := rlp.DecodeBytes(blob, account); err != nil {
			return err
		}
		storageIt, err := trie.New(trie.StorageTrieID(root, addrHash, account.Root), t.db)
		if err != nil {
			return err
		}
		for hash, slot := range slots {
			blob, err := storageIt.Get(hash.Bytes())
			if err != nil || !bytes.Equal(blob, slot) {
				return fmt.Errorf("slot is mismatched: %w", err)
			}
		}
	}
	return nil
}

func (t *tester) verifyHistory() error {
	bottom := t.bottomIndex()
	for i, root := range t.roots {
		// The state history related to the state above disk layer should not exist.
		if i > bottom {
			_, err := readHistory(t.db.freezer, uint64(i+1))
			if err == nil {
				return errors.New("unexpected state history")
			}
			continue
		}
		// The state history related to the state below or equal to the disk layer
		// should exist.
		obj, err := readHistory(t.db.freezer, uint64(i+1))
		if err != nil {
			return err
		}
		parent := types.EmptyRootHash
		if i != 0 {
			parent = t.roots[i-1]
		}
		if obj.meta.parent != parent {
			return fmt.Errorf("unexpected parent, want: %x, got: %x", parent, obj.meta.parent)
		}
		if obj.meta.root != root {
			return fmt.Errorf("unexpected root, want: %x, got: %x", root, obj.meta.root)
		}
	}
	return nil
}

// bottomIndex returns the index of current disk layer.
func (t *tester) bottomIndex() int {
	bottom := t.db.tree.bottom()
	for i := 0; i < len(t.roots); i++ {
		if t.roots[i] == bottom.rootHash() {
			return i
		}
	}
	return -1
}

func TestDatabaseRollback(t *testing.T) {
	// Redefine the diff layer depth allowance for faster testing.
	maxDiffLayers = 4
	defer func() {
		maxDiffLayers = 128
	}()

	// Verify state histories
	tester := newTester(t, 0, false, 32)
	defer tester.release()

	if err := tester.verifyHistory(); err != nil {
		t.Fatalf("Invalid state history, err: %v", err)
	}
	// Revert database from top to bottom
	for i := tester.bottomIndex(); i >= 0; i-- {
		parent := types.EmptyRootHash
		if i > 0 {
			parent = tester.roots[i-1]
		}
		if err := tester.db.Recover(parent); err != nil {
			t.Fatalf("Failed to revert db, err: %v", err)
		}
		if i > 0 {
			if err := tester.verifyMerkleState(parent); err != nil {
				t.Fatalf("Failed to verify state, err: %v", err)
			}
		}
	}
	if tester.db.tree.len() != 1 {
		t.Fatal("Only disk layer is expected")
	}
}

func TestDatabaseRecoverable(t *testing.T) {
	// Redefine the diff layer depth allowance for faster testing.
	maxDiffLayers = 4
	defer func() {
		maxDiffLayers = 128
	}()

	var (
		tester = newTester(t, 0, false, 12)
		index  = tester.bottomIndex()
	)
	defer tester.release()

	var cases = []struct {
		root   common.Hash
		expect bool
	}{
		// Unknown state should be unrecoverable
		{common.Hash{0x1}, false},

		// Initial state should be recoverable
		{types.EmptyRootHash, true},

		// common.Hash{} is not a valid state root for revert
		{common.Hash{}, false},

		// Layers below current disk layer are recoverable
		{tester.roots[index-1], true},

		// Disklayer itself is not recoverable, since it's
		// available for accessing.
		{tester.roots[index], false},

		// Layers above current disk layer are not recoverable
		// since they are available for accessing.
		{tester.roots[index+1], false},
	}
	for i, c := range cases {
		result := tester.db.Recoverable(c.root)
		if result != c.expect {
			t.Fatalf("case: %d, unexpected result, want %t, got %t", i, c.expect, result)
		}
	}
}

func TestDisable(t *testing.T) {
	// Redefine the diff layer depth allowance for faster testing.
	maxDiffLayers = 4
	defer func() {
		maxDiffLayers = 128
	}()

	tester := newTester(t, 0, false, 32)
	defer tester.release()

	stored := crypto.Keccak256Hash(rawdb.ReadAccountTrieNode(tester.db.diskdb, nil))
	if err := tester.db.Disable(); err != nil {
		t.Fatalf("Failed to deactivate database: %v", err)
	}
	if err := tester.db.Enable(types.EmptyRootHash); err == nil {
		t.Fatal("Invalid activation should be rejected")
	}
	if err := tester.db.Enable(stored); err != nil {
		t.Fatalf("Failed to activate database: %v", err)
	}

	// Ensure journal is deleted from disk
	if blob := rawdb.ReadTrieJournal(tester.db.diskdb); len(blob) != 0 {
		t.Fatal("Failed to clean journal")
	}
	// Ensure all trie histories are removed
	n, err := tester.db.freezer.Ancients()
	if err != nil {
		t.Fatal("Failed to clean state history")
	}
	if n != 0 {
		t.Fatal("Failed to clean state history")
	}
	// Verify layer tree structure, single disk layer is expected
	if tester.db.tree.len() != 1 {
		t.Fatalf("Extra layer kept %d", tester.db.tree.len())
	}
	if tester.db.tree.bottom().rootHash() != stored {
		t.Fatalf("Root hash is not matched exp %x got %x", stored, tester.db.tree.bottom().rootHash())
	}
}

func TestCommit(t *testing.T) {
	// Redefine the diff layer depth allowance for faster testing.
	maxDiffLayers = 4
	defer func() {
		maxDiffLayers = 128
	}()

	tester := newTester(t, 0, false, 12)
	defer tester.release()

	if err := tester.db.Commit(tester.lastHash(), false); err != nil {
		t.Fatalf("Failed to cap database, err: %v", err)
	}
	// Verify layer tree structure, single disk layer is expected
	if tester.db.tree.len() != 1 {
		t.Fatal("Layer tree structure is invalid")
	}
	if tester.db.tree.bottom().rootHash() != tester.lastHash() {
		t.Fatal("Layer tree structure is invalid")
	}
	// Verify states
	if err := tester.verifyMerkleState(tester.lastHash()); err != nil {
		t.Fatalf("State is invalid, err: %v", err)
	}
	// Verify state histories
	if err := tester.verifyHistory(); err != nil {
		t.Fatalf("State history is invalid, err: %v", err)
	}
}

func TestJournal(t *testing.T) {
	// Redefine the diff layer depth allowance for faster testing.
	maxDiffLayers = 4
	defer func() {
		maxDiffLayers = 128
	}()

	tester := newTester(t, 0, false, 12)
	defer tester.release()

	if err := tester.db.Journal(tester.lastHash()); err != nil {
		t.Errorf("Failed to journal, err: %v", err)
	}
	tester.db.Close()
	tester.db = New(tester.db.diskdb, nil, false)

	// Verify states including disk layer and all diff on top.
	for i := 0; i < len(tester.roots); i++ {
		if i >= tester.bottomIndex() {
			if err := tester.verifyMerkleState(tester.roots[i]); err != nil {
				t.Fatalf("Invalid state, err: %v", err)
			}
			continue
		}
		if err := tester.verifyMerkleState(tester.roots[i]); err == nil {
			t.Fatal("Unexpected state")
		}
	}
}

func TestCorruptedJournal(t *testing.T) {
	// Redefine the diff layer depth allowance for faster testing.
	maxDiffLayers = 4
	defer func() {
		maxDiffLayers = 128
	}()

	tester := newTester(t, 0, false, 12)
	defer tester.release()

	if err := tester.db.Journal(tester.lastHash()); err != nil {
		t.Errorf("Failed to journal, err: %v", err)
	}
	tester.db.Close()
	root := crypto.Keccak256Hash(rawdb.ReadAccountTrieNode(tester.db.diskdb, nil))

	// Mutate the journal in disk, it should be regarded as invalid
	blob := rawdb.ReadTrieJournal(tester.db.diskdb)
	blob[0] = 0xa
	rawdb.WriteTrieJournal(tester.db.diskdb, blob)

	// Verify states, all not-yet-written states should be discarded
	tester.db = New(tester.db.diskdb, nil, false)
	for i := 0; i < len(tester.roots); i++ {
		if tester.roots[i] == root {
			if err := tester.verifyMerkleState(root); err != nil {
				t.Fatalf("Disk state is corrupted, err: %v", err)
			}
			continue
		}
		if err := tester.verifyMerkleState(tester.roots[i]); err == nil {
			t.Fatal("Unexpected state")
		}
	}
}

// TestTailTruncateHistory function is designed to test a specific edge case where,
// when history objects are removed from the end, it should trigger a state flush
// if the ID of the new tail object is even higher than the persisted state ID.
//
// For example, let's say the ID of the persistent state is 10, and the current
// history objects range from ID(5) to ID(15). As we accumulate six more objects,
// the history will expand to cover ID(11) to ID(21). ID(11) then becomes the
// oldest history object, and its ID is even higher than the stored state.
//
// In this scenario, it is mandatory to update the persistent state before
// truncating the tail histories. This ensures that the ID of the persistent state
// always falls within the range of [oldest-history-id, latest-history-id].
func TestTailTruncateHistory(t *testing.T) {
	// Redefine the diff layer depth allowance for faster testing.
	maxDiffLayers = 4
	defer func() {
		maxDiffLayers = 128
	}()

	tester := newTester(t, 10, false, 12)
	defer tester.release()

	tester.db.Close()
	tester.db = New(tester.db.diskdb, &Config{StateHistory: 10}, false)

	head, err := tester.db.freezer.Ancients()
	if err != nil {
		t.Fatalf("Failed to obtain freezer head")
	}
	stored := rawdb.ReadPersistentStateID(tester.db.diskdb)
	if head != stored {
		t.Fatalf("Failed to truncate excess history object above, stored: %d, head: %d", stored, head)
	}
}

// copyAccounts returns a deep-copied account set of the provided one.
func copyAccounts(set map[common.Hash][]byte) map[common.Hash][]byte {
	copied := make(map[common.Hash][]byte, len(set))
	for key, val := range set {
		copied[key] = common.CopyBytes(val)
	}
	return copied
}

// copyStorages returns a deep-copied storage set of the provided one.
func copyStorages(set map[common.Hash]map[common.Hash][]byte) map[common.Hash]map[common.Hash][]byte {
	copied := make(map[common.Hash]map[common.Hash][]byte, len(set))
	for addrHash, subset := range set {
		copied[addrHash] = make(map[common.Hash][]byte, len(subset))
		for key, val := range subset {
			copied[addrHash][key] = common.CopyBytes(val)
		}
	}
	return copied
}
