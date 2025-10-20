// Copyright 2024 The go-ethereum Authors
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
	"fmt"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/internal/testrand"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/ethereum/go-ethereum/trie"
	"github.com/ethereum/go-ethereum/trie/trienode"
	"github.com/holiman/uint256"
)

func hashData(input []byte) common.Hash {
	return crypto.Keccak256Hash(input)
}

type genTester struct {
	diskdb   ethdb.Database
	db       *Database
	acctTrie *trie.Trie
	nodes    *trienode.MergedNodeSet
	states   *StateSetWithOrigin
}

func newGenTester() *genTester {
	disk := rawdb.NewMemoryDatabase()
	config := *Defaults
	config.SnapshotNoBuild = true // no background generation
	config.NoAsyncFlush = true    // no async flush
	db := New(disk, &config, false)
	tr, _ := trie.New(trie.StateTrieID(types.EmptyRootHash), db)
	return &genTester{
		diskdb:   disk,
		db:       db,
		acctTrie: tr,
		nodes:    trienode.NewMergedNodeSet(),
		states:   NewStateSetWithOrigin(nil, nil, nil, nil, false),
	}
}

func (t *genTester) addTrieAccount(acckey string, acc *types.StateAccount) {
	var (
		addr   = common.BytesToAddress([]byte(acckey))
		key    = hashData([]byte(acckey))
		val, _ = rlp.EncodeToBytes(acc)
	)
	t.acctTrie.MustUpdate(key.Bytes(), val)

	t.states.accountData[key] = val
	t.states.accountOrigin[addr] = nil
}

func (t *genTester) addSnapAccount(acckey string, acc *types.StateAccount) {
	key := hashData([]byte(acckey))
	rawdb.WriteAccountSnapshot(t.diskdb, key, types.SlimAccountRLP(*acc))
}

func (t *genTester) addAccount(acckey string, acc *types.StateAccount) {
	t.addTrieAccount(acckey, acc)
	t.addSnapAccount(acckey, acc)
}

func (t *genTester) addSnapStorage(accKey string, keys []string, vals []string) {
	accHash := hashData([]byte(accKey))
	for i, key := range keys {
		rawdb.WriteStorageSnapshot(t.diskdb, accHash, hashData([]byte(key)), []byte(vals[i]))
	}
}

func (t *genTester) makeStorageTrie(accKey string, keys []string, vals []string, commit bool) common.Hash {
	var (
		owner = hashData([]byte(accKey))
		addr  = common.BytesToAddress([]byte(accKey))
		id    = trie.StorageTrieID(types.EmptyRootHash, owner, types.EmptyRootHash)
		tr, _ = trie.New(id, t.db)

		storages       = make(map[common.Hash][]byte)
		storageOrigins = make(map[common.Hash][]byte)
	)
	for i, k := range keys {
		key := hashData([]byte(k))
		tr.MustUpdate(key.Bytes(), []byte(vals[i]))
		storages[key] = []byte(vals[i])
		storageOrigins[key] = nil
	}
	if !commit {
		return tr.Hash()
	}
	root, nodes := tr.Commit(false)
	if nodes != nil {
		t.nodes.Merge(nodes)
	}
	t.states.storageData[owner] = storages
	t.states.storageOrigin[addr] = storageOrigins
	return root
}

func (t *genTester) Commit() common.Hash {
	root, nodes := t.acctTrie.Commit(true)
	if nodes != nil {
		t.nodes.Merge(nodes)
	}
	t.db.Update(root, types.EmptyRootHash, 0, t.nodes, t.states)
	t.db.Commit(root, false)
	return root
}

func (t *genTester) CommitAndGenerate() (common.Hash, *diskLayer) {
	root := t.Commit()
	dl := generateSnapshot(t.db, root, false)
	return root, dl
}

// Tests that snapshot generation from an empty database.
func TestGeneration(t *testing.T) {
	helper := newGenTester()
	stRoot := helper.makeStorageTrie("", []string{"key-1", "key-2", "key-3"}, []string{"val-1", "val-2", "val-3"}, false)

	helper.addTrieAccount("acc-1", &types.StateAccount{Balance: uint256.NewInt(1), Root: stRoot, CodeHash: types.EmptyCodeHash.Bytes()})
	helper.addTrieAccount("acc-2", &types.StateAccount{Balance: uint256.NewInt(2), Root: types.EmptyRootHash, CodeHash: types.EmptyCodeHash.Bytes()})
	helper.addTrieAccount("acc-3", &types.StateAccount{Balance: uint256.NewInt(3), Root: stRoot, CodeHash: types.EmptyCodeHash.Bytes()})

	helper.makeStorageTrie("acc-1", []string{"key-1", "key-2", "key-3"}, []string{"val-1", "val-2", "val-3"}, true)
	helper.makeStorageTrie("acc-3", []string{"key-1", "key-2", "key-3"}, []string{"val-1", "val-2", "val-3"}, true)

	root, dl := helper.CommitAndGenerate()
	if have, want := root, common.HexToHash("0xe3712f1a226f3782caca78ca770ccc19ee000552813a9f59d479f8611db9b1fd"); have != want {
		t.Fatalf("have %#x want %#x", have, want)
	}
	select {
	case <-dl.generator.done:
		// Snapshot generation succeeded
	case <-time.After(3 * time.Second):
		t.Errorf("Snapshot generation failed")
	}
	// TODO(rjl493456442) enable the snapshot tests
	// checkSnapRoot(t, snap, root)

	// Signal abortion to the generator and wait for it to tear down
	dl.generator.stop()
}

// Tests that snapshot generation with existent flat state, where the flat state
// contains some errors:
// - the contract with empty storage root but has storage entries in the disk
// - the contract with non empty storage root but empty storage slots
// - the contract(non-empty storage) misses some storage slots
//   - miss in the beginning
//   - miss in the middle
//   - miss in the end
//
// - the contract(non-empty storage) has wrong storage slots
//   - wrong slots in the beginning
//   - wrong slots in the middle
//   - wrong slots in the end
//
// - the contract(non-empty storage) has extra storage slots
//   - extra slots in the beginning
//   - extra slots in the middle
//   - extra slots in the end
func TestGenerateExistentStateWithWrongStorage(t *testing.T) {
	helper := newGenTester()

	// Account one, empty storage trie root but non-empty flat states
	helper.addAccount("acc-1", &types.StateAccount{Balance: uint256.NewInt(1), Root: types.EmptyRootHash, CodeHash: types.EmptyCodeHash.Bytes()})
	helper.addSnapStorage("acc-1", []string{"key-1", "key-2", "key-3"}, []string{"val-1", "val-2", "val-3"})

	// Account two, non-empty storage trie root but empty flat states
	stRoot := helper.makeStorageTrie("acc-2", []string{"key-1", "key-2", "key-3"}, []string{"val-1", "val-2", "val-3"}, true)
	helper.addAccount("acc-2", &types.StateAccount{Balance: uint256.NewInt(1), Root: stRoot, CodeHash: types.EmptyCodeHash.Bytes()})

	// Miss slots
	{
		// Account three, non-empty root but misses slots in the beginning
		helper.makeStorageTrie("acc-3", []string{"key-1", "key-2", "key-3"}, []string{"val-1", "val-2", "val-3"}, true)
		helper.addAccount("acc-3", &types.StateAccount{Balance: uint256.NewInt(1), Root: stRoot, CodeHash: types.EmptyCodeHash.Bytes()})
		helper.addSnapStorage("acc-3", []string{"key-2", "key-3"}, []string{"val-2", "val-3"})

		// Account four, non-empty root but misses slots in the middle
		helper.makeStorageTrie("acc-4", []string{"key-1", "key-2", "key-3"}, []string{"val-1", "val-2", "val-3"}, true)
		helper.addAccount("acc-4", &types.StateAccount{Balance: uint256.NewInt(1), Root: stRoot, CodeHash: types.EmptyCodeHash.Bytes()})
		helper.addSnapStorage("acc-4", []string{"key-1", "key-3"}, []string{"val-1", "val-3"})

		// Account five, non-empty root but misses slots in the end
		helper.makeStorageTrie("acc-5", []string{"key-1", "key-2", "key-3"}, []string{"val-1", "val-2", "val-3"}, true)
		helper.addAccount("acc-5", &types.StateAccount{Balance: uint256.NewInt(1), Root: stRoot, CodeHash: types.EmptyCodeHash.Bytes()})
		helper.addSnapStorage("acc-5", []string{"key-1", "key-2"}, []string{"val-1", "val-2"})
	}

	// Wrong storage slots
	{
		// Account six, non-empty root but wrong slots in the beginning
		helper.makeStorageTrie("acc-6", []string{"key-1", "key-2", "key-3"}, []string{"val-1", "val-2", "val-3"}, true)
		helper.addAccount("acc-6", &types.StateAccount{Balance: uint256.NewInt(1), Root: stRoot, CodeHash: types.EmptyCodeHash.Bytes()})
		helper.addSnapStorage("acc-6", []string{"key-1", "key-2", "key-3"}, []string{"badval-1", "val-2", "val-3"})

		// Account seven, non-empty root but wrong slots in the middle
		helper.makeStorageTrie("acc-7", []string{"key-1", "key-2", "key-3"}, []string{"val-1", "val-2", "val-3"}, true)
		helper.addAccount("acc-7", &types.StateAccount{Balance: uint256.NewInt(1), Root: stRoot, CodeHash: types.EmptyCodeHash.Bytes()})
		helper.addSnapStorage("acc-7", []string{"key-1", "key-2", "key-3"}, []string{"val-1", "badval-2", "val-3"})

		// Account eight, non-empty root but wrong slots in the end
		helper.makeStorageTrie("acc-8", []string{"key-1", "key-2", "key-3"}, []string{"val-1", "val-2", "val-3"}, true)
		helper.addAccount("acc-8", &types.StateAccount{Balance: uint256.NewInt(1), Root: stRoot, CodeHash: types.EmptyCodeHash.Bytes()})
		helper.addSnapStorage("acc-8", []string{"key-1", "key-2", "key-3"}, []string{"val-1", "val-2", "badval-3"})

		// Account 9, non-empty root but rotated slots
		helper.makeStorageTrie("acc-9", []string{"key-1", "key-2", "key-3"}, []string{"val-1", "val-2", "val-3"}, true)
		helper.addAccount("acc-9", &types.StateAccount{Balance: uint256.NewInt(1), Root: stRoot, CodeHash: types.EmptyCodeHash.Bytes()})
		helper.addSnapStorage("acc-9", []string{"key-1", "key-2", "key-3"}, []string{"val-1", "val-3", "val-2"})
	}

	// Extra storage slots
	{
		// Account 10, non-empty root but extra slots in the beginning
		helper.makeStorageTrie("acc-10", []string{"key-1", "key-2", "key-3"}, []string{"val-1", "val-2", "val-3"}, true)
		helper.addAccount("acc-10", &types.StateAccount{Balance: uint256.NewInt(1), Root: stRoot, CodeHash: types.EmptyCodeHash.Bytes()})
		helper.addSnapStorage("acc-10", []string{"key-0", "key-1", "key-2", "key-3"}, []string{"val-0", "val-1", "val-2", "val-3"})

		// Account 11, non-empty root but extra slots in the middle
		helper.makeStorageTrie("acc-11", []string{"key-1", "key-2", "key-3"}, []string{"val-1", "val-2", "val-3"}, true)
		helper.addAccount("acc-11", &types.StateAccount{Balance: uint256.NewInt(1), Root: stRoot, CodeHash: types.EmptyCodeHash.Bytes()})
		helper.addSnapStorage("acc-11", []string{"key-1", "key-2", "key-2-1", "key-3"}, []string{"val-1", "val-2", "val-2-1", "val-3"})

		// Account 12, non-empty root but extra slots in the end
		helper.makeStorageTrie("acc-12", []string{"key-1", "key-2", "key-3"}, []string{"val-1", "val-2", "val-3"}, true)
		helper.addAccount("acc-12", &types.StateAccount{Balance: uint256.NewInt(1), Root: stRoot, CodeHash: types.EmptyCodeHash.Bytes()})
		helper.addSnapStorage("acc-12", []string{"key-1", "key-2", "key-3", "key-4"}, []string{"val-1", "val-2", "val-3", "val-4"})
	}

	root, dl := helper.CommitAndGenerate()
	t.Logf("Root: %#x\n", root) // Root = 0x8746cce9fd9c658b2cfd639878ed6584b7a2b3e73bb40f607fcfa156002429a0

	select {
	case <-dl.generator.done:
		// Snapshot generation succeeded

	case <-time.After(3 * time.Second):
		t.Errorf("Snapshot generation failed")
	}
	// TODO(rjl493456442) enable the snapshot tests
	// checkSnapRoot(t, snap, root)

	// Signal abortion to the generator and wait for it to tear down
	dl.generator.stop()
}

// Tests that snapshot generation with existent flat state, where the flat state
// contains some errors:
// - miss accounts
// - wrong accounts
// - extra accounts
func TestGenerateExistentStateWithWrongAccounts(t *testing.T) {
	helper := newGenTester()

	// Trie accounts [acc-1, acc-2, acc-3, acc-4, acc-6]
	helper.makeStorageTrie("acc-1", []string{"key-1", "key-2", "key-3"}, []string{"val-1", "val-2", "val-3"}, true)
	helper.makeStorageTrie("acc-2", []string{"key-1", "key-2", "key-3"}, []string{"val-1", "val-2", "val-3"}, true)
	helper.makeStorageTrie("acc-3", []string{"key-1", "key-2", "key-3"}, []string{"val-1", "val-2", "val-3"}, true)
	helper.makeStorageTrie("acc-4", []string{"key-1", "key-2", "key-3"}, []string{"val-1", "val-2", "val-3"}, true)
	stRoot := helper.makeStorageTrie("acc-6", []string{"key-1", "key-2", "key-3"}, []string{"val-1", "val-2", "val-3"}, true)

	// Missing accounts, only in the trie
	{
		helper.addTrieAccount("acc-1", &types.StateAccount{Balance: uint256.NewInt(1), Root: stRoot, CodeHash: types.EmptyCodeHash.Bytes()}) // Beginning
		helper.addTrieAccount("acc-4", &types.StateAccount{Balance: uint256.NewInt(1), Root: stRoot, CodeHash: types.EmptyCodeHash.Bytes()}) // Middle
		helper.addTrieAccount("acc-6", &types.StateAccount{Balance: uint256.NewInt(1), Root: stRoot, CodeHash: types.EmptyCodeHash.Bytes()}) // End
	}

	// Wrong accounts
	{
		helper.addTrieAccount("acc-2", &types.StateAccount{Balance: uint256.NewInt(1), Root: stRoot, CodeHash: types.EmptyCodeHash.Bytes()})
		helper.addSnapAccount("acc-2", &types.StateAccount{Balance: uint256.NewInt(1), Root: stRoot, CodeHash: common.Hex2Bytes("0x1234")})

		helper.addTrieAccount("acc-3", &types.StateAccount{Balance: uint256.NewInt(1), Root: stRoot, CodeHash: types.EmptyCodeHash.Bytes()})
		helper.addSnapAccount("acc-3", &types.StateAccount{Balance: uint256.NewInt(1), Root: types.EmptyRootHash, CodeHash: types.EmptyCodeHash.Bytes()})
	}

	// Extra accounts, only in the snap
	{
		helper.addSnapAccount("acc-0", &types.StateAccount{Balance: uint256.NewInt(1), Root: stRoot, CodeHash: types.EmptyCodeHash.Bytes()})              // before the beginning
		helper.addSnapAccount("acc-5", &types.StateAccount{Balance: uint256.NewInt(1), Root: types.EmptyRootHash, CodeHash: common.Hex2Bytes("0x1234")})  // Middle
		helper.addSnapAccount("acc-7", &types.StateAccount{Balance: uint256.NewInt(1), Root: types.EmptyRootHash, CodeHash: types.EmptyCodeHash.Bytes()}) // after the end
	}

	root, dl := helper.CommitAndGenerate()
	t.Logf("Root: %#x\n", root) // Root = 0x825891472281463511e7ebcc7f109e4f9200c20fa384754e11fd605cd98464e8

	select {
	case <-dl.generator.done:
		// Snapshot generation succeeded

	case <-time.After(3 * time.Second):
		t.Errorf("Snapshot generation failed")
	}
	// TODO(rjl493456442) enable the snapshot tests
	// checkSnapRoot(t, snap, root)

	// Signal abortion to the generator and wait for it to tear down
	dl.generator.stop()
}

func TestGenerateCorruptAccountTrie(t *testing.T) {
	helper := newGenTester()
	helper.addTrieAccount("acc-1", &types.StateAccount{Balance: uint256.NewInt(1), Root: types.EmptyRootHash, CodeHash: types.EmptyCodeHash.Bytes()}) // 0xc7a30f39aff471c95d8a837497ad0e49b65be475cc0953540f80cfcdbdcd9074
	helper.addTrieAccount("acc-2", &types.StateAccount{Balance: uint256.NewInt(2), Root: types.EmptyRootHash, CodeHash: types.EmptyCodeHash.Bytes()}) // 0x65145f923027566669a1ae5ccac66f945b55ff6eaeb17d2ea8e048b7d381f2d7
	helper.addTrieAccount("acc-3", &types.StateAccount{Balance: uint256.NewInt(3), Root: types.EmptyRootHash, CodeHash: types.EmptyCodeHash.Bytes()}) // 0x19ead688e907b0fab07176120dceec244a72aff2f0aa51e8b827584e378772f4

	root := helper.Commit() // Root: 0xa04693ea110a31037fb5ee814308a6f1d76bdab0b11676bdf4541d2de55ba978

	// Delete an account trie node and ensure the generator chokes
	path := []byte{0xc}
	if !rawdb.HasAccountTrieNode(helper.diskdb, path) {
		t.Logf("Invalid node path to delete, %v", path)
	}
	rawdb.DeleteAccountTrieNode(helper.diskdb, path)
	helper.db.tree.bottom().resetCache()

	dl := generateSnapshot(helper.db, root, false)
	select {
	case <-dl.generator.done:
		// Snapshot generation succeeded
		t.Errorf("Snapshot generated against corrupt account trie")

	case <-time.After(time.Second):
		// Not generated fast enough, hopefully blocked inside on missing trie node fail
	}
	// Signal abortion to the generator and wait for it to tear down
	dl.generator.stop()
}

func TestGenerateMissingStorageTrie(t *testing.T) {
	var (
		acc1   = hashData([]byte("acc-1"))
		acc3   = hashData([]byte("acc-3"))
		helper = newGenTester()
	)
	stRoot := helper.makeStorageTrie("acc-1", []string{"key-1", "key-2", "key-3"}, []string{"val-1", "val-2", "val-3"}, true)                         // 0xddefcd9376dd029653ef384bd2f0a126bb755fe84fdcc9e7cf421ba454f2bc67
	helper.addTrieAccount("acc-1", &types.StateAccount{Balance: uint256.NewInt(1), Root: stRoot, CodeHash: types.EmptyCodeHash.Bytes()})              // 0x9250573b9c18c664139f3b6a7a8081b7d8f8916a8fcc5d94feec6c29f5fd4e9e
	helper.addTrieAccount("acc-2", &types.StateAccount{Balance: uint256.NewInt(2), Root: types.EmptyRootHash, CodeHash: types.EmptyCodeHash.Bytes()}) // 0x65145f923027566669a1ae5ccac66f945b55ff6eaeb17d2ea8e048b7d381f2d7
	stRoot = helper.makeStorageTrie("acc-3", []string{"key-1", "key-2", "key-3"}, []string{"val-1", "val-2", "val-3"}, true)
	helper.addTrieAccount("acc-3", &types.StateAccount{Balance: uint256.NewInt(3), Root: stRoot, CodeHash: types.EmptyCodeHash.Bytes()}) // 0x50815097425d000edfc8b3a4a13e175fc2bdcfee8bdfbf2d1ff61041d3c235b2

	root := helper.Commit()

	// Delete storage trie root of account one and three.
	rawdb.DeleteStorageTrieNode(helper.diskdb, acc1, nil)
	rawdb.DeleteStorageTrieNode(helper.diskdb, acc3, nil)
	helper.db.tree.bottom().resetCache()

	dl := generateSnapshot(helper.db, root, false)
	select {
	case <-dl.generator.done:
		// Snapshot generation succeeded
		t.Errorf("Snapshot generated against corrupt storage trie")

	case <-time.After(time.Second):
		// Not generated fast enough, hopefully blocked inside on missing trie node fail
	}
	// Signal abortion to the generator and wait for it to tear down
	dl.generator.stop()
}

func TestGenerateCorruptStorageTrie(t *testing.T) {
	helper := newGenTester()

	stRoot := helper.makeStorageTrie("acc-1", []string{"key-1", "key-2", "key-3"}, []string{"val-1", "val-2", "val-3"}, true)                         // 0xddefcd9376dd029653ef384bd2f0a126bb755fe84fdcc9e7cf421ba454f2bc67
	helper.addTrieAccount("acc-1", &types.StateAccount{Balance: uint256.NewInt(1), Root: stRoot, CodeHash: types.EmptyCodeHash.Bytes()})              // 0x9250573b9c18c664139f3b6a7a8081b7d8f8916a8fcc5d94feec6c29f5fd4e9e
	helper.addTrieAccount("acc-2", &types.StateAccount{Balance: uint256.NewInt(2), Root: types.EmptyRootHash, CodeHash: types.EmptyCodeHash.Bytes()}) // 0x65145f923027566669a1ae5ccac66f945b55ff6eaeb17d2ea8e048b7d381f2d7
	stRoot = helper.makeStorageTrie("acc-3", []string{"key-1", "key-2", "key-3"}, []string{"val-1", "val-2", "val-3"}, true)
	helper.addTrieAccount("acc-3", &types.StateAccount{Balance: uint256.NewInt(3), Root: stRoot, CodeHash: types.EmptyCodeHash.Bytes()}) // 0x50815097425d000edfc8b3a4a13e175fc2bdcfee8bdfbf2d1ff61041d3c235b2

	root := helper.Commit()

	// Delete a node in the storage trie.
	path := []byte{0x4}
	if !rawdb.HasStorageTrieNode(helper.diskdb, hashData([]byte("acc-1")), path) {
		t.Logf("Invalid node path to delete, %v", path)
	}
	rawdb.DeleteStorageTrieNode(helper.diskdb, hashData([]byte("acc-1")), []byte{0x4})

	if !rawdb.HasStorageTrieNode(helper.diskdb, hashData([]byte("acc-3")), path) {
		t.Logf("Invalid node path to delete, %v", path)
	}
	rawdb.DeleteStorageTrieNode(helper.diskdb, hashData([]byte("acc-3")), []byte{0x4})

	helper.db.tree.bottom().resetCache()

	dl := generateSnapshot(helper.db, root, false)
	select {
	case <-dl.generator.done:
		// Snapshot generation succeeded
		t.Errorf("Snapshot generated against corrupt storage trie")

	case <-time.After(time.Second):
		// Not generated fast enough, hopefully blocked inside on missing trie node fail
	}
	// Signal abortion to the generator and wait for it to tear down
	dl.generator.stop()
}

func TestGenerateWithExtraAccounts(t *testing.T) {
	helper := newGenTester()

	// Account one in the trie
	stRoot := helper.makeStorageTrie("acc-1",
		[]string{"key-1", "key-2", "key-3", "key-4", "key-5"},
		[]string{"val-1", "val-2", "val-3", "val-4", "val-5"},
		true,
	)
	acc := &types.StateAccount{Balance: uint256.NewInt(1), Root: stRoot, CodeHash: types.EmptyCodeHash.Bytes()}
	val, _ := rlp.EncodeToBytes(acc)
	helper.acctTrie.MustUpdate(hashData([]byte("acc-1")).Bytes(), val) // 0x9250573b9c18c664139f3b6a7a8081b7d8f8916a8fcc5d94feec6c29f5fd4e9e

	// Identical in the snap
	key := hashData([]byte("acc-1"))
	rawdb.WriteAccountSnapshot(helper.diskdb, key, val)
	rawdb.WriteStorageSnapshot(helper.diskdb, key, hashData([]byte("key-1")), []byte("val-1"))
	rawdb.WriteStorageSnapshot(helper.diskdb, key, hashData([]byte("key-2")), []byte("val-2"))
	rawdb.WriteStorageSnapshot(helper.diskdb, key, hashData([]byte("key-3")), []byte("val-3"))
	rawdb.WriteStorageSnapshot(helper.diskdb, key, hashData([]byte("key-4")), []byte("val-4"))
	rawdb.WriteStorageSnapshot(helper.diskdb, key, hashData([]byte("key-5")), []byte("val-5"))

	// Account two exists only in the snapshot
	stRoot = helper.makeStorageTrie("acc-2",
		[]string{"key-1", "key-2", "key-3", "key-4", "key-5"},
		[]string{"val-1", "val-2", "val-3", "val-4", "val-5"},
		true,
	)
	acc = &types.StateAccount{Balance: uint256.NewInt(1), Root: stRoot, CodeHash: types.EmptyCodeHash.Bytes()}
	val, _ = rlp.EncodeToBytes(acc)
	key = hashData([]byte("acc-2"))
	rawdb.WriteAccountSnapshot(helper.diskdb, key, val)
	rawdb.WriteStorageSnapshot(helper.diskdb, key, hashData([]byte("b-key-1")), []byte("b-val-1"))
	rawdb.WriteStorageSnapshot(helper.diskdb, key, hashData([]byte("b-key-2")), []byte("b-val-2"))
	rawdb.WriteStorageSnapshot(helper.diskdb, key, hashData([]byte("b-key-3")), []byte("b-val-3"))

	root := helper.Commit()

	// To verify the test: If we now inspect the snap db, there should exist extraneous storage items
	if data := rawdb.ReadStorageSnapshot(helper.diskdb, hashData([]byte("acc-2")), hashData([]byte("b-key-1"))); data == nil {
		t.Fatalf("expected snap storage to exist")
	}
	dl := generateSnapshot(helper.db, root, false)
	select {
	case <-dl.generator.done:
		// Snapshot generation succeeded

	case <-time.After(3 * time.Second):
		t.Errorf("Snapshot generation failed")
	}
	// TODO(rjl493456442) enable the snapshot tests
	// checkSnapRoot(t, snap, root)

	// Signal abortion to the generator and wait for it to tear down
	dl.generator.stop()

	// If we now inspect the snap db, there should exist no extraneous storage items
	if data := rawdb.ReadStorageSnapshot(helper.diskdb, hashData([]byte("acc-2")), hashData([]byte("b-key-1"))); data != nil {
		t.Fatalf("expected slot to be removed, got %v", string(data))
	}
}

func TestGenerateWithManyExtraAccounts(t *testing.T) {
	helper := newGenTester()

	// Account one in the trie
	stRoot := helper.makeStorageTrie("acc-1",
		[]string{"key-1", "key-2", "key-3"},
		[]string{"val-1", "val-2", "val-3"},
		true,
	)
	acc := &types.StateAccount{Balance: uint256.NewInt(1), Root: stRoot, CodeHash: types.EmptyCodeHash.Bytes()}
	val, _ := rlp.EncodeToBytes(acc)
	helper.acctTrie.MustUpdate(hashData([]byte("acc-1")).Bytes(), val) // 0x9250573b9c18c664139f3b6a7a8081b7d8f8916a8fcc5d94feec6c29f5fd4e9e

	// Identical in the snap
	key := hashData([]byte("acc-1"))
	rawdb.WriteAccountSnapshot(helper.diskdb, key, val)
	rawdb.WriteStorageSnapshot(helper.diskdb, key, hashData([]byte("key-1")), []byte("val-1"))
	rawdb.WriteStorageSnapshot(helper.diskdb, key, hashData([]byte("key-2")), []byte("val-2"))
	rawdb.WriteStorageSnapshot(helper.diskdb, key, hashData([]byte("key-3")), []byte("val-3"))

	// 100 accounts exist only in snapshot
	for i := 0; i < 1000; i++ {
		acc := &types.StateAccount{Balance: uint256.NewInt(uint64(i)), Root: types.EmptyRootHash, CodeHash: types.EmptyCodeHash.Bytes()}
		val, _ := rlp.EncodeToBytes(acc)
		key := hashData([]byte(fmt.Sprintf("acc-%d", i)))
		rawdb.WriteAccountSnapshot(helper.diskdb, key, val)
	}

	_, dl := helper.CommitAndGenerate()
	select {
	case <-dl.generator.done:
		// Snapshot generation succeeded

	case <-time.After(3 * time.Second):
		t.Errorf("Snapshot generation failed")
	}
	// TODO(rjl493456442) enable the snapshot tests
	// checkSnapRoot(t, snap, root)

	// Signal abortion to the generator and wait for it to tear down
	dl.generator.stop()
}

func TestGenerateWithExtraBeforeAndAfter(t *testing.T) {
	helper := newGenTester()

	acc := &types.StateAccount{Balance: uint256.NewInt(1), Root: types.EmptyRootHash, CodeHash: types.EmptyCodeHash.Bytes()}
	val, _ := rlp.EncodeToBytes(acc)

	acctHashA := hashData([]byte("acc-1"))
	acctHashB := hashData([]byte("acc-2"))

	helper.acctTrie.MustUpdate(acctHashA.Bytes(), val)
	helper.acctTrie.MustUpdate(acctHashB.Bytes(), val)

	rawdb.WriteAccountSnapshot(helper.diskdb, acctHashA, val)
	rawdb.WriteAccountSnapshot(helper.diskdb, acctHashB, val)

	for i := 0; i < 16; i++ {
		rawdb.WriteAccountSnapshot(helper.diskdb, common.Hash{byte(i)}, val)
	}
	_, dl := helper.CommitAndGenerate()
	select {
	case <-dl.generator.done:
		// Snapshot generation succeeded

	case <-time.After(3 * time.Second):
		t.Errorf("Snapshot generation failed")
	}
	// TODO(rjl493456442) enable the snapshot tests
	// checkSnapRoot(t, snap, root)

	// Signal abortion to the generator and wait for it to tear down
	dl.generator.stop()
}

func TestGenerateWithMalformedStateData(t *testing.T) {
	helper := newGenTester()

	acctHash := hashData([]byte("acc"))
	acc := &types.StateAccount{Balance: uint256.NewInt(1), Root: types.EmptyRootHash, CodeHash: types.EmptyCodeHash.Bytes()}
	val, _ := rlp.EncodeToBytes(acc)
	helper.acctTrie.MustUpdate(acctHash.Bytes(), val)

	junk := make([]byte, 100)
	copy(junk, []byte{0xde, 0xad})
	rawdb.WriteAccountSnapshot(helper.diskdb, acctHash, junk)
	for i := 0; i < 16; i++ {
		rawdb.WriteAccountSnapshot(helper.diskdb, common.Hash{byte(i)}, junk)
	}

	_, dl := helper.CommitAndGenerate()
	select {
	case <-dl.generator.done:
		// Snapshot generation succeeded

	case <-time.After(3 * time.Second):
		t.Errorf("Snapshot generation failed")
	}
	// TODO(rjl493456442) enable the snapshot tests
	// checkSnapRoot(t, snap, root)

	// Signal abortion to the generator and wait for it to tear down
	dl.generator.stop()
}

func TestGenerateFromEmptySnap(t *testing.T) {
	helper := newGenTester()

	for i := 0; i < 400; i++ {
		stRoot := helper.makeStorageTrie(fmt.Sprintf("acc-%d", i), []string{"key-1", "key-2", "key-3"}, []string{"val-1", "val-2", "val-3"}, true)
		helper.addTrieAccount(fmt.Sprintf("acc-%d", i), &types.StateAccount{Balance: uint256.NewInt(1), Root: stRoot, CodeHash: types.EmptyCodeHash.Bytes()})
	}
	root, snap := helper.CommitAndGenerate()
	t.Logf("Root: %#x\n", root) // Root: 0x6f7af6d2e1a1bf2b84a3beb3f8b64388465fbc1e274ca5d5d3fc787ca78f59e4

	select {
	case <-snap.generator.done:
		// Snapshot generation succeeded

	case <-time.After(3 * time.Second):
		t.Errorf("Snapshot generation failed")
	}
	// TODO(rjl493456442) enable the snapshot tests
	// checkSnapRoot(t, snap, root)

	// Signal abortion to the generator and wait for it to tear down
	snap.generator.stop()
}

func TestGenerateWithIncompleteStorage(t *testing.T) {
	helper := newGenTester()
	stKeys := []string{"1", "2", "3", "4", "5", "6", "7", "8"}
	stVals := []string{"v1", "v2", "v3", "v4", "v5", "v6", "v7", "v8"}

	// We add 8 accounts, each one is missing exactly one of the storage slots. This means
	// we don't have to order the keys and figure out exactly which hash-key winds up
	// on the sensitive spots at the boundaries
	for i := 0; i < 8; i++ {
		accKey := fmt.Sprintf("acc-%d", i)
		stRoot := helper.makeStorageTrie(accKey, stKeys, stVals, true)
		helper.addAccount(accKey, &types.StateAccount{Balance: uint256.NewInt(uint64(i)), Root: stRoot, CodeHash: types.EmptyCodeHash.Bytes()})
		var moddedKeys []string
		var moddedVals []string
		for ii := 0; ii < 8; ii++ {
			if ii != i {
				moddedKeys = append(moddedKeys, stKeys[ii])
				moddedVals = append(moddedVals, stVals[ii])
			}
		}
		helper.addSnapStorage(accKey, moddedKeys, moddedVals)
	}
	root, dl := helper.CommitAndGenerate()
	t.Logf("Root: %#x\n", root) // Root: 0xca73f6f05ba4ca3024ef340ef3dfca8fdabc1b677ff13f5a9571fd49c16e67ff

	select {
	case <-dl.generator.done:
		// Snapshot generation succeeded

	case <-time.After(3 * time.Second):
		t.Errorf("Snapshot generation failed")
	}
	// TODO(rjl493456442) enable the snapshot tests
	// checkSnapRoot(t, snap, root)

	// Signal abortion to the generator and wait for it to tear down
	dl.generator.stop()
}

func incKey(key []byte) []byte {
	for i := len(key) - 1; i >= 0; i-- {
		key[i]++
		if key[i] != 0x0 {
			break
		}
	}
	return key
}

func decKey(key []byte) []byte {
	for i := len(key) - 1; i >= 0; i-- {
		key[i]--
		if key[i] != 0xff {
			break
		}
	}
	return key
}

func populateDangling(disk ethdb.KeyValueStore) {
	populate := func(accountHash common.Hash, keys []string, vals []string) {
		for i, key := range keys {
			rawdb.WriteStorageSnapshot(disk, accountHash, hashData([]byte(key)), []byte(vals[i]))
		}
	}
	// Dangling storages of the "first" account
	populate(common.Hash{}, []string{"key-1", "key-2", "key-3"}, []string{"val-1", "val-2", "val-3"})

	// Dangling storages of the "last" account
	populate(common.HexToHash("ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff"), []string{"key-1", "key-2", "key-3"}, []string{"val-1", "val-2", "val-3"})

	// Dangling storages around the account 1
	hash := decKey(hashData([]byte("acc-1")).Bytes())
	populate(common.BytesToHash(hash), []string{"key-1", "key-2", "key-3"}, []string{"val-1", "val-2", "val-3"})
	hash = incKey(hashData([]byte("acc-1")).Bytes())
	populate(common.BytesToHash(hash), []string{"key-1", "key-2", "key-3"}, []string{"val-1", "val-2", "val-3"})

	// Dangling storages around the account 2
	hash = decKey(hashData([]byte("acc-2")).Bytes())
	populate(common.BytesToHash(hash), []string{"key-1", "key-2", "key-3"}, []string{"val-1", "val-2", "val-3"})
	hash = incKey(hashData([]byte("acc-2")).Bytes())
	populate(common.BytesToHash(hash), []string{"key-1", "key-2", "key-3"}, []string{"val-1", "val-2", "val-3"})

	// Dangling storages around the account 3
	hash = decKey(hashData([]byte("acc-3")).Bytes())
	populate(common.BytesToHash(hash), []string{"key-1", "key-2", "key-3"}, []string{"val-1", "val-2", "val-3"})
	hash = incKey(hashData([]byte("acc-3")).Bytes())
	populate(common.BytesToHash(hash), []string{"key-1", "key-2", "key-3"}, []string{"val-1", "val-2", "val-3"})

	// Dangling storages of the random account
	populate(testrand.Hash(), []string{"key-1", "key-2", "key-3"}, []string{"val-1", "val-2", "val-3"})
	populate(testrand.Hash(), []string{"key-1", "key-2", "key-3"}, []string{"val-1", "val-2", "val-3"})
	populate(testrand.Hash(), []string{"key-1", "key-2", "key-3"}, []string{"val-1", "val-2", "val-3"})
}

func TestGenerateCompleteSnapshotWithDanglingStorage(t *testing.T) {
	var helper = newGenTester()

	stRoot := helper.makeStorageTrie("acc-1", []string{"key-1", "key-2", "key-3"}, []string{"val-1", "val-2", "val-3"}, true)
	helper.addAccount("acc-1", &types.StateAccount{Balance: uint256.NewInt(1), Root: stRoot, CodeHash: types.EmptyCodeHash.Bytes()})
	helper.addAccount("acc-2", &types.StateAccount{Balance: uint256.NewInt(1), Root: types.EmptyRootHash, CodeHash: types.EmptyCodeHash.Bytes()})

	helper.makeStorageTrie("acc-3", []string{"key-1", "key-2", "key-3"}, []string{"val-1", "val-2", "val-3"}, true)
	helper.addAccount("acc-3", &types.StateAccount{Balance: uint256.NewInt(1), Root: stRoot, CodeHash: types.EmptyCodeHash.Bytes()})

	helper.addSnapStorage("acc-1", []string{"key-1", "key-2", "key-3"}, []string{"val-1", "val-2", "val-3"})
	helper.addSnapStorage("acc-3", []string{"key-1", "key-2", "key-3"}, []string{"val-1", "val-2", "val-3"})

	populateDangling(helper.diskdb)

	_, dl := helper.CommitAndGenerate()
	select {
	case <-dl.generator.done:
		// Snapshot generation succeeded

	case <-time.After(3 * time.Second):
		t.Errorf("Snapshot generation failed")
	}
	// TODO(rjl493456442) enable the snapshot tests
	// checkSnapRoot(t, snap, root)

	// Signal abortion to the generator and wait for it to tear down
	dl.generator.stop()
}

func TestGenerateBrokenSnapshotWithDanglingStorage(t *testing.T) {
	var helper = newGenTester()

	stRoot := helper.makeStorageTrie("acc-1", []string{"key-1", "key-2", "key-3"}, []string{"val-1", "val-2", "val-3"}, true)
	helper.addTrieAccount("acc-1", &types.StateAccount{Balance: uint256.NewInt(1), Root: stRoot, CodeHash: types.EmptyCodeHash.Bytes()})
	helper.addTrieAccount("acc-2", &types.StateAccount{Balance: uint256.NewInt(2), Root: types.EmptyRootHash, CodeHash: types.EmptyCodeHash.Bytes()})

	helper.makeStorageTrie("acc-3", []string{"key-1", "key-2", "key-3"}, []string{"val-1", "val-2", "val-3"}, true)
	helper.addTrieAccount("acc-3", &types.StateAccount{Balance: uint256.NewInt(3), Root: stRoot, CodeHash: types.EmptyCodeHash.Bytes()})

	populateDangling(helper.diskdb)

	_, dl := helper.CommitAndGenerate()
	select {
	case <-dl.generator.done:
		// Snapshot generation succeeded

	case <-time.After(3 * time.Second):
		t.Errorf("Snapshot generation failed")
	}
	// TODO(rjl493456442) enable the snapshot tests
	// checkSnapRoot(t, snap, root)

	// Signal abortion to the generator and wait for it to tear down
	dl.generator.stop()
}
