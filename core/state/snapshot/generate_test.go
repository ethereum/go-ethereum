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

package snapshot

import (
	"fmt"
	"math/big"
	"os"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/ethdb/memorydb"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/ethereum/go-ethereum/trie"
	"golang.org/x/crypto/sha3"
)

// Tests that snapshot generation from an empty database.
func TestGeneration(t *testing.T) {
	// We can't use statedb to make a test trie (circular dependency), so make
	// a fake one manually. We're going with a small account trie of 3 accounts,
	// two of which also has the same 3-slot storage trie attached.
	var (
		diskdb = memorydb.New()
		triedb = trie.NewDatabase(diskdb)
	)
	stTrie, _ := trie.NewSecure(common.Hash{}, triedb)
	stTrie.Update([]byte("key-1"), []byte("val-1")) // 0x1314700b81afc49f94db3623ef1df38f3ed18b73a1b7ea2f6c095118cf6118a0
	stTrie.Update([]byte("key-2"), []byte("val-2")) // 0x18a0f4d79cff4459642dd7604f303886ad9d77c30cf3d7d7cedb3a693ab6d371
	stTrie.Update([]byte("key-3"), []byte("val-3")) // 0x51c71a47af0695957647fb68766d0becee77e953df17c29b3c2f25436f055c78
	stTrie.Commit(nil)                              // Root: 0xddefcd9376dd029653ef384bd2f0a126bb755fe84fdcc9e7cf421ba454f2bc67

	accTrie, _ := trie.NewSecure(common.Hash{}, triedb)
	acc := &Account{Balance: big.NewInt(1), Root: stTrie.Hash().Bytes(), CodeHash: emptyCode.Bytes()}
	val, _ := rlp.EncodeToBytes(acc)
	accTrie.Update([]byte("acc-1"), val) // 0x9250573b9c18c664139f3b6a7a8081b7d8f8916a8fcc5d94feec6c29f5fd4e9e

	acc = &Account{Balance: big.NewInt(2), Root: emptyRoot.Bytes(), CodeHash: emptyCode.Bytes()}
	val, _ = rlp.EncodeToBytes(acc)
	accTrie.Update([]byte("acc-2"), val) // 0x65145f923027566669a1ae5ccac66f945b55ff6eaeb17d2ea8e048b7d381f2d7

	acc = &Account{Balance: big.NewInt(3), Root: stTrie.Hash().Bytes(), CodeHash: emptyCode.Bytes()}
	val, _ = rlp.EncodeToBytes(acc)
	accTrie.Update([]byte("acc-3"), val) // 0x50815097425d000edfc8b3a4a13e175fc2bdcfee8bdfbf2d1ff61041d3c235b2
	root, _, _ := accTrie.Commit(nil)    // Root: 0xe3712f1a226f3782caca78ca770ccc19ee000552813a9f59d479f8611db9b1fd
	triedb.Commit(root, false, nil)

	if have, want := root, common.HexToHash("0xe3712f1a226f3782caca78ca770ccc19ee000552813a9f59d479f8611db9b1fd"); have != want {
		t.Fatalf("have %#x want %#x", have, want)
	}
	snap := generateSnapshot(diskdb, triedb, 16, root)
	select {
	case <-snap.genPending:
		// Snapshot generation succeeded

	case <-time.After(3 * time.Second):
		t.Errorf("Snapshot generation failed")
	}
	checkSnapRoot(t, snap, root)
	// Signal abortion to the generator and wait for it to tear down
	stop := make(chan *generatorStats)
	snap.genAbort <- stop
	<-stop
}

func hashData(input []byte) common.Hash {
	var hasher = sha3.NewLegacyKeccak256()
	var hash common.Hash
	hasher.Reset()
	hasher.Write(input)
	hasher.Sum(hash[:0])
	return hash
}

// Tests that snapshot generation with existent flat state.
func TestGenerateExistentState(t *testing.T) {
	// We can't use statedb to make a test trie (circular dependency), so make
	// a fake one manually. We're going with a small account trie of 3 accounts,
	// two of which also has the same 3-slot storage trie attached.
	var (
		diskdb = memorydb.New()
		triedb = trie.NewDatabase(diskdb)
	)
	stTrie, _ := trie.NewSecure(common.Hash{}, triedb)
	stTrie.Update([]byte("key-1"), []byte("val-1")) // 0x1314700b81afc49f94db3623ef1df38f3ed18b73a1b7ea2f6c095118cf6118a0
	stTrie.Update([]byte("key-2"), []byte("val-2")) // 0x18a0f4d79cff4459642dd7604f303886ad9d77c30cf3d7d7cedb3a693ab6d371
	stTrie.Update([]byte("key-3"), []byte("val-3")) // 0x51c71a47af0695957647fb68766d0becee77e953df17c29b3c2f25436f055c78
	stTrie.Commit(nil)                              // Root: 0xddefcd9376dd029653ef384bd2f0a126bb755fe84fdcc9e7cf421ba454f2bc67

	accTrie, _ := trie.NewSecure(common.Hash{}, triedb)
	acc := &Account{Balance: big.NewInt(1), Root: stTrie.Hash().Bytes(), CodeHash: emptyCode.Bytes()}
	val, _ := rlp.EncodeToBytes(acc)
	accTrie.Update([]byte("acc-1"), val) // 0x9250573b9c18c664139f3b6a7a8081b7d8f8916a8fcc5d94feec6c29f5fd4e9e
	rawdb.WriteAccountSnapshot(diskdb, hashData([]byte("acc-1")), val)
	rawdb.WriteStorageSnapshot(diskdb, hashData([]byte("acc-1")), hashData([]byte("key-1")), []byte("val-1"))
	rawdb.WriteStorageSnapshot(diskdb, hashData([]byte("acc-1")), hashData([]byte("key-2")), []byte("val-2"))
	rawdb.WriteStorageSnapshot(diskdb, hashData([]byte("acc-1")), hashData([]byte("key-3")), []byte("val-3"))

	acc = &Account{Balance: big.NewInt(2), Root: emptyRoot.Bytes(), CodeHash: emptyCode.Bytes()}
	val, _ = rlp.EncodeToBytes(acc)
	accTrie.Update([]byte("acc-2"), val) // 0x65145f923027566669a1ae5ccac66f945b55ff6eaeb17d2ea8e048b7d381f2d7
	diskdb.Put(hashData([]byte("acc-2")).Bytes(), val)
	rawdb.WriteAccountSnapshot(diskdb, hashData([]byte("acc-2")), val)

	acc = &Account{Balance: big.NewInt(3), Root: stTrie.Hash().Bytes(), CodeHash: emptyCode.Bytes()}
	val, _ = rlp.EncodeToBytes(acc)
	accTrie.Update([]byte("acc-3"), val) // 0x50815097425d000edfc8b3a4a13e175fc2bdcfee8bdfbf2d1ff61041d3c235b2
	rawdb.WriteAccountSnapshot(diskdb, hashData([]byte("acc-3")), val)
	rawdb.WriteStorageSnapshot(diskdb, hashData([]byte("acc-3")), hashData([]byte("key-1")), []byte("val-1"))
	rawdb.WriteStorageSnapshot(diskdb, hashData([]byte("acc-3")), hashData([]byte("key-2")), []byte("val-2"))
	rawdb.WriteStorageSnapshot(diskdb, hashData([]byte("acc-3")), hashData([]byte("key-3")), []byte("val-3"))

	root, _, _ := accTrie.Commit(nil) // Root: 0xe3712f1a226f3782caca78ca770ccc19ee000552813a9f59d479f8611db9b1fd
	triedb.Commit(root, false, nil)

	snap := generateSnapshot(diskdb, triedb, 16, root)
	select {
	case <-snap.genPending:
		// Snapshot generation succeeded

	case <-time.After(3 * time.Second):
		t.Errorf("Snapshot generation failed")
	}
	checkSnapRoot(t, snap, root)
	// Signal abortion to the generator and wait for it to tear down
	stop := make(chan *generatorStats)
	snap.genAbort <- stop
	<-stop
}

func checkSnapRoot(t *testing.T, snap *diskLayer, trieRoot common.Hash) {
	t.Helper()
	accIt := snap.AccountIterator(common.Hash{})
	defer accIt.Release()
	snapRoot, err := generateTrieRoot(nil, accIt, common.Hash{}, stackTrieGenerate,
		func(db ethdb.KeyValueWriter, accountHash, codeHash common.Hash, stat *generateStats) (common.Hash, error) {
			storageIt, _ := snap.StorageIterator(accountHash, common.Hash{})
			defer storageIt.Release()

			hash, err := generateTrieRoot(nil, storageIt, accountHash, stackTrieGenerate, nil, stat, false)
			if err != nil {
				return common.Hash{}, err
			}
			return hash, nil
		}, newGenerateStats(), true)

	if err != nil {
		t.Fatal(err)
	}
	if snapRoot != trieRoot {
		t.Fatalf("snaproot: %#x != trieroot #%x", snapRoot, trieRoot)
	}
}

type testHelper struct {
	diskdb  *memorydb.Database
	triedb  *trie.Database
	accTrie *trie.SecureTrie
}

func newHelper() *testHelper {
	diskdb := memorydb.New()
	triedb := trie.NewDatabase(diskdb)
	accTrie, _ := trie.NewSecure(common.Hash{}, triedb)
	return &testHelper{
		diskdb:  diskdb,
		triedb:  triedb,
		accTrie: accTrie,
	}
}

func (t *testHelper) addTrieAccount(acckey string, acc *Account) {
	val, _ := rlp.EncodeToBytes(acc)
	t.accTrie.Update([]byte(acckey), val)
}

func (t *testHelper) addSnapAccount(acckey string, acc *Account) {
	val, _ := rlp.EncodeToBytes(acc)
	key := hashData([]byte(acckey))
	rawdb.WriteAccountSnapshot(t.diskdb, key, val)
}

func (t *testHelper) addAccount(acckey string, acc *Account) {
	t.addTrieAccount(acckey, acc)
	t.addSnapAccount(acckey, acc)
}

func (t *testHelper) addSnapStorage(accKey string, keys []string, vals []string) {
	accHash := hashData([]byte(accKey))
	for i, key := range keys {
		rawdb.WriteStorageSnapshot(t.diskdb, accHash, hashData([]byte(key)), []byte(vals[i]))
	}
}

func (t *testHelper) makeStorageTrie(keys []string, vals []string) []byte {
	stTrie, _ := trie.NewSecure(common.Hash{}, t.triedb)
	for i, k := range keys {
		stTrie.Update([]byte(k), []byte(vals[i]))
	}
	root, _, _ := stTrie.Commit(nil)
	return root.Bytes()
}

func (t *testHelper) Generate() (common.Hash, *diskLayer) {
	root, _, _ := t.accTrie.Commit(nil)
	t.triedb.Commit(root, false, nil)
	snap := generateSnapshot(t.diskdb, t.triedb, 16, root)
	return root, snap
}

// Tests that snapshot generation with existent flat state, where the flat state
// contains some errors:
// - the contract with empty storage root but has storage entries in the disk
// - the contract with non empty storage root but empty storage slots
// - the contract(non-empty storage) misses some storage slots
//   - miss in the beginning
//   - miss in the middle
//   - miss in the end
// - the contract(non-empty storage) has wrong storage slots
//   - wrong slots in the beginning
//   - wrong slots in the middle
//   - wrong slots in the end
// - the contract(non-empty storage) has extra storage slots
//   - extra slots in the beginning
//   - extra slots in the middle
//   - extra slots in the end
func TestGenerateExistentStateWithWrongStorage(t *testing.T) {
	helper := newHelper()
	stRoot := helper.makeStorageTrie([]string{"key-1", "key-2", "key-3"}, []string{"val-1", "val-2", "val-3"})

	// Account one, empty root but non-empty database
	helper.addAccount("acc-1", &Account{Balance: big.NewInt(1), Root: emptyRoot.Bytes(), CodeHash: emptyCode.Bytes()})
	helper.addSnapStorage("acc-1", []string{"key-1", "key-2", "key-3"}, []string{"val-1", "val-2", "val-3"})

	// Account two, non empty root but empty database
	helper.addAccount("acc-2", &Account{Balance: big.NewInt(1), Root: stRoot, CodeHash: emptyCode.Bytes()})

	// Miss slots
	{
		// Account three, non empty root but misses slots in the beginning
		helper.addAccount("acc-3", &Account{Balance: big.NewInt(1), Root: stRoot, CodeHash: emptyCode.Bytes()})
		helper.addSnapStorage("acc-3", []string{"key-2", "key-3"}, []string{"val-2", "val-3"})

		// Account four, non empty root but misses slots in the middle
		helper.addAccount("acc-4", &Account{Balance: big.NewInt(1), Root: stRoot, CodeHash: emptyCode.Bytes()})
		helper.addSnapStorage("acc-4", []string{"key-1", "key-3"}, []string{"val-1", "val-3"})

		// Account five, non empty root but misses slots in the end
		helper.addAccount("acc-5", &Account{Balance: big.NewInt(1), Root: stRoot, CodeHash: emptyCode.Bytes()})
		helper.addSnapStorage("acc-5", []string{"key-1", "key-2"}, []string{"val-1", "val-2"})
	}

	// Wrong storage slots
	{
		// Account six, non empty root but wrong slots in the beginning
		helper.addAccount("acc-6", &Account{Balance: big.NewInt(1), Root: stRoot, CodeHash: emptyCode.Bytes()})
		helper.addSnapStorage("acc-6", []string{"key-1", "key-2", "key-3"}, []string{"badval-1", "val-2", "val-3"})

		// Account seven, non empty root but wrong slots in the middle
		helper.addAccount("acc-7", &Account{Balance: big.NewInt(1), Root: stRoot, CodeHash: emptyCode.Bytes()})
		helper.addSnapStorage("acc-7", []string{"key-1", "key-2", "key-3"}, []string{"val-1", "badval-2", "val-3"})

		// Account eight, non empty root but wrong slots in the end
		helper.addAccount("acc-8", &Account{Balance: big.NewInt(1), Root: stRoot, CodeHash: emptyCode.Bytes()})
		helper.addSnapStorage("acc-8", []string{"key-1", "key-2", "key-3"}, []string{"val-1", "val-2", "badval-3"})

		// Account 9, non empty root but rotated slots
		helper.addAccount("acc-9", &Account{Balance: big.NewInt(1), Root: stRoot, CodeHash: emptyCode.Bytes()})
		helper.addSnapStorage("acc-9", []string{"key-1", "key-2", "key-3"}, []string{"val-1", "val-3", "val-2"})
	}

	// Extra storage slots
	{
		// Account 10, non empty root but extra slots in the beginning
		helper.addAccount("acc-10", &Account{Balance: big.NewInt(1), Root: stRoot, CodeHash: emptyCode.Bytes()})
		helper.addSnapStorage("acc-10", []string{"key-0", "key-1", "key-2", "key-3"}, []string{"val-0", "val-1", "val-2", "val-3"})

		// Account 11, non empty root but extra slots in the middle
		helper.addAccount("acc-11", &Account{Balance: big.NewInt(1), Root: stRoot, CodeHash: emptyCode.Bytes()})
		helper.addSnapStorage("acc-11", []string{"key-1", "key-2", "key-2-1", "key-3"}, []string{"val-1", "val-2", "val-2-1", "val-3"})

		// Account 12, non empty root but extra slots in the end
		helper.addAccount("acc-12", &Account{Balance: big.NewInt(1), Root: stRoot, CodeHash: emptyCode.Bytes()})
		helper.addSnapStorage("acc-12", []string{"key-1", "key-2", "key-3", "key-4"}, []string{"val-1", "val-2", "val-3", "val-4"})
	}

	root, snap := helper.Generate()
	t.Logf("Root: %#x\n", root) // Root = 0x8746cce9fd9c658b2cfd639878ed6584b7a2b3e73bb40f607fcfa156002429a0

	select {
	case <-snap.genPending:
		// Snapshot generation succeeded

	case <-time.After(3 * time.Second):
		t.Errorf("Snapshot generation failed")
	}
	checkSnapRoot(t, snap, root)
	// Signal abortion to the generator and wait for it to tear down
	stop := make(chan *generatorStats)
	snap.genAbort <- stop
	<-stop
}

// Tests that snapshot generation with existent flat state, where the flat state
// contains some errors:
// - miss accounts
// - wrong accounts
// - extra accounts
func TestGenerateExistentStateWithWrongAccounts(t *testing.T) {
	helper := newHelper()
	stRoot := helper.makeStorageTrie([]string{"key-1", "key-2", "key-3"}, []string{"val-1", "val-2", "val-3"})

	// Trie accounts [acc-1, acc-2, acc-3, acc-4, acc-6]
	// Extra accounts [acc-0, acc-5, acc-7]

	// Missing accounts, only in the trie
	{
		helper.addTrieAccount("acc-1", &Account{Balance: big.NewInt(1), Root: stRoot, CodeHash: emptyCode.Bytes()}) // Beginning
		helper.addTrieAccount("acc-4", &Account{Balance: big.NewInt(1), Root: stRoot, CodeHash: emptyCode.Bytes()}) // Middle
		helper.addTrieAccount("acc-6", &Account{Balance: big.NewInt(1), Root: stRoot, CodeHash: emptyCode.Bytes()}) // End
	}

	// Wrong accounts
	{
		helper.addTrieAccount("acc-2", &Account{Balance: big.NewInt(1), Root: stRoot, CodeHash: emptyCode.Bytes()})
		helper.addSnapAccount("acc-2", &Account{Balance: big.NewInt(1), Root: stRoot, CodeHash: common.Hex2Bytes("0x1234")})

		helper.addTrieAccount("acc-3", &Account{Balance: big.NewInt(1), Root: stRoot, CodeHash: emptyCode.Bytes()})
		helper.addSnapAccount("acc-3", &Account{Balance: big.NewInt(1), Root: emptyRoot.Bytes(), CodeHash: emptyCode.Bytes()})
	}

	// Extra accounts, only in the snap
	{
		helper.addSnapAccount("acc-0", &Account{Balance: big.NewInt(1), Root: stRoot, CodeHash: emptyRoot.Bytes()})                     // before the beginning
		helper.addSnapAccount("acc-5", &Account{Balance: big.NewInt(1), Root: emptyRoot.Bytes(), CodeHash: common.Hex2Bytes("0x1234")}) // Middle
		helper.addSnapAccount("acc-7", &Account{Balance: big.NewInt(1), Root: emptyRoot.Bytes(), CodeHash: emptyRoot.Bytes()})          // after the end
	}

	root, snap := helper.Generate()
	t.Logf("Root: %#x\n", root) // Root = 0x825891472281463511e7ebcc7f109e4f9200c20fa384754e11fd605cd98464e8

	select {
	case <-snap.genPending:
		// Snapshot generation succeeded

	case <-time.After(3 * time.Second):
		t.Errorf("Snapshot generation failed")
	}
	checkSnapRoot(t, snap, root)

	// Signal abortion to the generator and wait for it to tear down
	stop := make(chan *generatorStats)
	snap.genAbort <- stop
	<-stop
}

// Tests that snapshot generation errors out correctly in case of a missing trie
// node in the account trie.
func TestGenerateCorruptAccountTrie(t *testing.T) {
	// We can't use statedb to make a test trie (circular dependency), so make
	// a fake one manually. We're going with a small account trie of 3 accounts,
	// without any storage slots to keep the test smaller.
	var (
		diskdb = memorydb.New()
		triedb = trie.NewDatabase(diskdb)
	)
	tr, _ := trie.NewSecure(common.Hash{}, triedb)
	acc := &Account{Balance: big.NewInt(1), Root: emptyRoot.Bytes(), CodeHash: emptyCode.Bytes()}
	val, _ := rlp.EncodeToBytes(acc)
	tr.Update([]byte("acc-1"), val) // 0xc7a30f39aff471c95d8a837497ad0e49b65be475cc0953540f80cfcdbdcd9074

	acc = &Account{Balance: big.NewInt(2), Root: emptyRoot.Bytes(), CodeHash: emptyCode.Bytes()}
	val, _ = rlp.EncodeToBytes(acc)
	tr.Update([]byte("acc-2"), val) // 0x65145f923027566669a1ae5ccac66f945b55ff6eaeb17d2ea8e048b7d381f2d7

	acc = &Account{Balance: big.NewInt(3), Root: emptyRoot.Bytes(), CodeHash: emptyCode.Bytes()}
	val, _ = rlp.EncodeToBytes(acc)
	tr.Update([]byte("acc-3"), val) // 0x19ead688e907b0fab07176120dceec244a72aff2f0aa51e8b827584e378772f4
	tr.Commit(nil)                  // Root: 0xa04693ea110a31037fb5ee814308a6f1d76bdab0b11676bdf4541d2de55ba978

	// Delete an account trie leaf and ensure the generator chokes
	triedb.Commit(common.HexToHash("0xa04693ea110a31037fb5ee814308a6f1d76bdab0b11676bdf4541d2de55ba978"), false, nil)
	diskdb.Delete(common.HexToHash("0x65145f923027566669a1ae5ccac66f945b55ff6eaeb17d2ea8e048b7d381f2d7").Bytes())

	snap := generateSnapshot(diskdb, triedb, 16, common.HexToHash("0xa04693ea110a31037fb5ee814308a6f1d76bdab0b11676bdf4541d2de55ba978"))
	select {
	case <-snap.genPending:
		// Snapshot generation succeeded
		t.Errorf("Snapshot generated against corrupt account trie")

	case <-time.After(time.Second):
		// Not generated fast enough, hopefully blocked inside on missing trie node fail
	}
	// Signal abortion to the generator and wait for it to tear down
	stop := make(chan *generatorStats)
	snap.genAbort <- stop
	<-stop
}

// Tests that snapshot generation errors out correctly in case of a missing root
// trie node for a storage trie. It's similar to internal corruption but it is
// handled differently inside the generator.
func TestGenerateMissingStorageTrie(t *testing.T) {
	// We can't use statedb to make a test trie (circular dependency), so make
	// a fake one manually. We're going with a small account trie of 3 accounts,
	// two of which also has the same 3-slot storage trie attached.
	var (
		diskdb = memorydb.New()
		triedb = trie.NewDatabase(diskdb)
	)
	stTrie, _ := trie.NewSecure(common.Hash{}, triedb)
	stTrie.Update([]byte("key-1"), []byte("val-1")) // 0x1314700b81afc49f94db3623ef1df38f3ed18b73a1b7ea2f6c095118cf6118a0
	stTrie.Update([]byte("key-2"), []byte("val-2")) // 0x18a0f4d79cff4459642dd7604f303886ad9d77c30cf3d7d7cedb3a693ab6d371
	stTrie.Update([]byte("key-3"), []byte("val-3")) // 0x51c71a47af0695957647fb68766d0becee77e953df17c29b3c2f25436f055c78
	stTrie.Commit(nil)                              // Root: 0xddefcd9376dd029653ef384bd2f0a126bb755fe84fdcc9e7cf421ba454f2bc67

	accTrie, _ := trie.NewSecure(common.Hash{}, triedb)
	acc := &Account{Balance: big.NewInt(1), Root: stTrie.Hash().Bytes(), CodeHash: emptyCode.Bytes()}
	val, _ := rlp.EncodeToBytes(acc)
	accTrie.Update([]byte("acc-1"), val) // 0x9250573b9c18c664139f3b6a7a8081b7d8f8916a8fcc5d94feec6c29f5fd4e9e

	acc = &Account{Balance: big.NewInt(2), Root: emptyRoot.Bytes(), CodeHash: emptyCode.Bytes()}
	val, _ = rlp.EncodeToBytes(acc)
	accTrie.Update([]byte("acc-2"), val) // 0x65145f923027566669a1ae5ccac66f945b55ff6eaeb17d2ea8e048b7d381f2d7

	acc = &Account{Balance: big.NewInt(3), Root: stTrie.Hash().Bytes(), CodeHash: emptyCode.Bytes()}
	val, _ = rlp.EncodeToBytes(acc)
	accTrie.Update([]byte("acc-3"), val) // 0x50815097425d000edfc8b3a4a13e175fc2bdcfee8bdfbf2d1ff61041d3c235b2
	accTrie.Commit(nil)                  // Root: 0xe3712f1a226f3782caca78ca770ccc19ee000552813a9f59d479f8611db9b1fd

	// We can only corrupt the disk database, so flush the tries out
	triedb.Reference(
		common.HexToHash("0xddefcd9376dd029653ef384bd2f0a126bb755fe84fdcc9e7cf421ba454f2bc67"),
		common.HexToHash("0x9250573b9c18c664139f3b6a7a8081b7d8f8916a8fcc5d94feec6c29f5fd4e9e"),
	)
	triedb.Reference(
		common.HexToHash("0xddefcd9376dd029653ef384bd2f0a126bb755fe84fdcc9e7cf421ba454f2bc67"),
		common.HexToHash("0x50815097425d000edfc8b3a4a13e175fc2bdcfee8bdfbf2d1ff61041d3c235b2"),
	)
	triedb.Commit(common.HexToHash("0xe3712f1a226f3782caca78ca770ccc19ee000552813a9f59d479f8611db9b1fd"), false, nil)

	// Delete a storage trie root and ensure the generator chokes
	diskdb.Delete(common.HexToHash("0xddefcd9376dd029653ef384bd2f0a126bb755fe84fdcc9e7cf421ba454f2bc67").Bytes())

	snap := generateSnapshot(diskdb, triedb, 16, common.HexToHash("0xe3712f1a226f3782caca78ca770ccc19ee000552813a9f59d479f8611db9b1fd"))
	select {
	case <-snap.genPending:
		// Snapshot generation succeeded
		t.Errorf("Snapshot generated against corrupt storage trie")

	case <-time.After(time.Second):
		// Not generated fast enough, hopefully blocked inside on missing trie node fail
	}
	// Signal abortion to the generator and wait for it to tear down
	stop := make(chan *generatorStats)
	snap.genAbort <- stop
	<-stop
}

// Tests that snapshot generation errors out correctly in case of a missing trie
// node in a storage trie.
func TestGenerateCorruptStorageTrie(t *testing.T) {
	// We can't use statedb to make a test trie (circular dependency), so make
	// a fake one manually. We're going with a small account trie of 3 accounts,
	// two of which also has the same 3-slot storage trie attached.
	var (
		diskdb = memorydb.New()
		triedb = trie.NewDatabase(diskdb)
	)
	stTrie, _ := trie.NewSecure(common.Hash{}, triedb)
	stTrie.Update([]byte("key-1"), []byte("val-1")) // 0x1314700b81afc49f94db3623ef1df38f3ed18b73a1b7ea2f6c095118cf6118a0
	stTrie.Update([]byte("key-2"), []byte("val-2")) // 0x18a0f4d79cff4459642dd7604f303886ad9d77c30cf3d7d7cedb3a693ab6d371
	stTrie.Update([]byte("key-3"), []byte("val-3")) // 0x51c71a47af0695957647fb68766d0becee77e953df17c29b3c2f25436f055c78
	stTrie.Commit(nil)                              // Root: 0xddefcd9376dd029653ef384bd2f0a126bb755fe84fdcc9e7cf421ba454f2bc67

	accTrie, _ := trie.NewSecure(common.Hash{}, triedb)
	acc := &Account{Balance: big.NewInt(1), Root: stTrie.Hash().Bytes(), CodeHash: emptyCode.Bytes()}
	val, _ := rlp.EncodeToBytes(acc)
	accTrie.Update([]byte("acc-1"), val) // 0x9250573b9c18c664139f3b6a7a8081b7d8f8916a8fcc5d94feec6c29f5fd4e9e

	acc = &Account{Balance: big.NewInt(2), Root: emptyRoot.Bytes(), CodeHash: emptyCode.Bytes()}
	val, _ = rlp.EncodeToBytes(acc)
	accTrie.Update([]byte("acc-2"), val) // 0x65145f923027566669a1ae5ccac66f945b55ff6eaeb17d2ea8e048b7d381f2d7

	acc = &Account{Balance: big.NewInt(3), Root: stTrie.Hash().Bytes(), CodeHash: emptyCode.Bytes()}
	val, _ = rlp.EncodeToBytes(acc)
	accTrie.Update([]byte("acc-3"), val) // 0x50815097425d000edfc8b3a4a13e175fc2bdcfee8bdfbf2d1ff61041d3c235b2
	accTrie.Commit(nil)                  // Root: 0xe3712f1a226f3782caca78ca770ccc19ee000552813a9f59d479f8611db9b1fd

	// We can only corrupt the disk database, so flush the tries out
	triedb.Reference(
		common.HexToHash("0xddefcd9376dd029653ef384bd2f0a126bb755fe84fdcc9e7cf421ba454f2bc67"),
		common.HexToHash("0x9250573b9c18c664139f3b6a7a8081b7d8f8916a8fcc5d94feec6c29f5fd4e9e"),
	)
	triedb.Reference(
		common.HexToHash("0xddefcd9376dd029653ef384bd2f0a126bb755fe84fdcc9e7cf421ba454f2bc67"),
		common.HexToHash("0x50815097425d000edfc8b3a4a13e175fc2bdcfee8bdfbf2d1ff61041d3c235b2"),
	)
	triedb.Commit(common.HexToHash("0xe3712f1a226f3782caca78ca770ccc19ee000552813a9f59d479f8611db9b1fd"), false, nil)

	// Delete a storage trie leaf and ensure the generator chokes
	diskdb.Delete(common.HexToHash("0x18a0f4d79cff4459642dd7604f303886ad9d77c30cf3d7d7cedb3a693ab6d371").Bytes())

	snap := generateSnapshot(diskdb, triedb, 16, common.HexToHash("0xe3712f1a226f3782caca78ca770ccc19ee000552813a9f59d479f8611db9b1fd"))
	select {
	case <-snap.genPending:
		// Snapshot generation succeeded
		t.Errorf("Snapshot generated against corrupt storage trie")

	case <-time.After(time.Second):
		// Not generated fast enough, hopefully blocked inside on missing trie node fail
	}
	// Signal abortion to the generator and wait for it to tear down
	stop := make(chan *generatorStats)
	snap.genAbort <- stop
	<-stop
}

func getStorageTrie(n int, triedb *trie.Database) *trie.SecureTrie {
	stTrie, _ := trie.NewSecure(common.Hash{}, triedb)
	for i := 0; i < n; i++ {
		k := fmt.Sprintf("key-%d", i)
		v := fmt.Sprintf("val-%d", i)
		stTrie.Update([]byte(k), []byte(v))
	}
	stTrie.Commit(nil)
	return stTrie
}

// Tests that snapshot generation when an extra account with storage exists in the snap state.
func TestGenerateWithExtraAccounts(t *testing.T) {
	var (
		diskdb = memorydb.New()
		triedb = trie.NewDatabase(diskdb)
		stTrie = getStorageTrie(5, triedb)
	)
	accTrie, _ := trie.NewSecure(common.Hash{}, triedb)
	{ // Account one in the trie
		acc := &Account{Balance: big.NewInt(1), Root: stTrie.Hash().Bytes(), CodeHash: emptyCode.Bytes()}
		val, _ := rlp.EncodeToBytes(acc)
		accTrie.Update([]byte("acc-1"), val) // 0x9250573b9c18c664139f3b6a7a8081b7d8f8916a8fcc5d94feec6c29f5fd4e9e
		// Identical in the snap
		key := hashData([]byte("acc-1"))
		rawdb.WriteAccountSnapshot(diskdb, key, val)
		rawdb.WriteStorageSnapshot(diskdb, key, hashData([]byte("key-1")), []byte("val-1"))
		rawdb.WriteStorageSnapshot(diskdb, key, hashData([]byte("key-2")), []byte("val-2"))
		rawdb.WriteStorageSnapshot(diskdb, key, hashData([]byte("key-3")), []byte("val-3"))
		rawdb.WriteStorageSnapshot(diskdb, key, hashData([]byte("key-4")), []byte("val-4"))
		rawdb.WriteStorageSnapshot(diskdb, key, hashData([]byte("key-5")), []byte("val-5"))
	}
	{ // Account two exists only in the snapshot
		acc := &Account{Balance: big.NewInt(1), Root: stTrie.Hash().Bytes(), CodeHash: emptyCode.Bytes()}
		val, _ := rlp.EncodeToBytes(acc)
		key := hashData([]byte("acc-2"))
		rawdb.WriteAccountSnapshot(diskdb, key, val)
		rawdb.WriteStorageSnapshot(diskdb, key, hashData([]byte("b-key-1")), []byte("b-val-1"))
		rawdb.WriteStorageSnapshot(diskdb, key, hashData([]byte("b-key-2")), []byte("b-val-2"))
		rawdb.WriteStorageSnapshot(diskdb, key, hashData([]byte("b-key-3")), []byte("b-val-3"))
	}
	root, _, _ := accTrie.Commit(nil)
	t.Logf("root: %x", root)
	triedb.Commit(root, false, nil)
	// To verify the test: If we now inspect the snap db, there should exist extraneous storage items
	if data := rawdb.ReadStorageSnapshot(diskdb, hashData([]byte("acc-2")), hashData([]byte("b-key-1"))); data == nil {
		t.Fatalf("expected snap storage to exist")
	}

	snap := generateSnapshot(diskdb, triedb, 16, root)
	select {
	case <-snap.genPending:
		// Snapshot generation succeeded

	case <-time.After(3 * time.Second):
		t.Errorf("Snapshot generation failed")
	}
	checkSnapRoot(t, snap, root)
	// Signal abortion to the generator and wait for it to tear down
	stop := make(chan *generatorStats)
	snap.genAbort <- stop
	<-stop
	// If we now inspect the snap db, there should exist no extraneous storage items
	if data := rawdb.ReadStorageSnapshot(diskdb, hashData([]byte("acc-2")), hashData([]byte("b-key-1"))); data != nil {
		t.Fatalf("expected slot to be removed, got %v", string(data))
	}
}

func enableLogging() {
	log.Root().SetHandler(log.LvlFilterHandler(log.LvlTrace, log.StreamHandler(os.Stderr, log.TerminalFormat(true))))
}

// Tests that snapshot generation when an extra account with storage exists in the snap state.
func TestGenerateWithManyExtraAccounts(t *testing.T) {
	if false {
		enableLogging()
	}
	var (
		diskdb = memorydb.New()
		triedb = trie.NewDatabase(diskdb)
		stTrie = getStorageTrie(3, triedb)
	)
	accTrie, _ := trie.NewSecure(common.Hash{}, triedb)
	{ // Account one in the trie
		acc := &Account{Balance: big.NewInt(1), Root: stTrie.Hash().Bytes(), CodeHash: emptyCode.Bytes()}
		val, _ := rlp.EncodeToBytes(acc)
		accTrie.Update([]byte("acc-1"), val) // 0x9250573b9c18c664139f3b6a7a8081b7d8f8916a8fcc5d94feec6c29f5fd4e9e
		// Identical in the snap
		key := hashData([]byte("acc-1"))
		rawdb.WriteAccountSnapshot(diskdb, key, val)
		rawdb.WriteStorageSnapshot(diskdb, key, hashData([]byte("key-1")), []byte("val-1"))
		rawdb.WriteStorageSnapshot(diskdb, key, hashData([]byte("key-2")), []byte("val-2"))
		rawdb.WriteStorageSnapshot(diskdb, key, hashData([]byte("key-3")), []byte("val-3"))
	}
	{ // 100 accounts exist only in snapshot
		for i := 0; i < 1000; i++ {
			//acc := &Account{Balance: big.NewInt(int64(i)), Root: stTrie.Hash().Bytes(), CodeHash: emptyCode.Bytes()}
			acc := &Account{Balance: big.NewInt(int64(i)), Root: emptyRoot.Bytes(), CodeHash: emptyCode.Bytes()}
			val, _ := rlp.EncodeToBytes(acc)
			key := hashData([]byte(fmt.Sprintf("acc-%d", i)))
			rawdb.WriteAccountSnapshot(diskdb, key, val)
		}
	}
	root, _, _ := accTrie.Commit(nil)
	t.Logf("root: %x", root)
	triedb.Commit(root, false, nil)

	snap := generateSnapshot(diskdb, triedb, 16, root)
	select {
	case <-snap.genPending:
		// Snapshot generation succeeded

	case <-time.After(3 * time.Second):
		t.Errorf("Snapshot generation failed")
	}
	checkSnapRoot(t, snap, root)
	// Signal abortion to the generator and wait for it to tear down
	stop := make(chan *generatorStats)
	snap.genAbort <- stop
	<-stop
}

// Tests this case
// maxAccountRange 3
// snapshot-accounts: 01, 02, 03, 04, 05, 06, 07
// trie-accounts:             03,             07
//
// We iterate three snapshot storage slots (max = 3) from the database. They are 0x01, 0x02, 0x03.
// The trie has a lot of deletions.
// So in trie, we iterate 2 entries 0x03, 0x07. We create the 0x07 in the database and abort the procedure, because the trie is exhausted.
// But in the database, we still have the stale storage slots 0x04, 0x05. They are not iterated yet, but the procedure is finished.
func TestGenerateWithExtraBeforeAndAfter(t *testing.T) {
	accountCheckRange = 3
	if false {
		enableLogging()
	}
	var (
		diskdb = memorydb.New()
		triedb = trie.NewDatabase(diskdb)
	)
	accTrie, _ := trie.New(common.Hash{}, triedb)
	{
		acc := &Account{Balance: big.NewInt(1), Root: emptyRoot.Bytes(), CodeHash: emptyCode.Bytes()}
		val, _ := rlp.EncodeToBytes(acc)
		accTrie.Update(common.HexToHash("0x03").Bytes(), val)
		accTrie.Update(common.HexToHash("0x07").Bytes(), val)

		rawdb.WriteAccountSnapshot(diskdb, common.HexToHash("0x01"), val)
		rawdb.WriteAccountSnapshot(diskdb, common.HexToHash("0x02"), val)
		rawdb.WriteAccountSnapshot(diskdb, common.HexToHash("0x03"), val)
		rawdb.WriteAccountSnapshot(diskdb, common.HexToHash("0x04"), val)
		rawdb.WriteAccountSnapshot(diskdb, common.HexToHash("0x05"), val)
		rawdb.WriteAccountSnapshot(diskdb, common.HexToHash("0x06"), val)
		rawdb.WriteAccountSnapshot(diskdb, common.HexToHash("0x07"), val)
	}

	root, _, _ := accTrie.Commit(nil)
	t.Logf("root: %x", root)
	triedb.Commit(root, false, nil)

	snap := generateSnapshot(diskdb, triedb, 16, root)
	select {
	case <-snap.genPending:
		// Snapshot generation succeeded

	case <-time.After(3 * time.Second):
		t.Errorf("Snapshot generation failed")
	}
	checkSnapRoot(t, snap, root)
	// Signal abortion to the generator and wait for it to tear down
	stop := make(chan *generatorStats)
	snap.genAbort <- stop
	<-stop
}

// TestGenerateWithMalformedSnapdata tests what happes if we have some junk
// in the snapshot database, which cannot be parsed back to an account
func TestGenerateWithMalformedSnapdata(t *testing.T) {
	accountCheckRange = 3
	if false {
		enableLogging()
	}
	var (
		diskdb = memorydb.New()
		triedb = trie.NewDatabase(diskdb)
	)
	accTrie, _ := trie.New(common.Hash{}, triedb)
	{
		acc := &Account{Balance: big.NewInt(1), Root: emptyRoot.Bytes(), CodeHash: emptyCode.Bytes()}
		val, _ := rlp.EncodeToBytes(acc)
		accTrie.Update(common.HexToHash("0x03").Bytes(), val)

		junk := make([]byte, 100)
		copy(junk, []byte{0xde, 0xad})
		rawdb.WriteAccountSnapshot(diskdb, common.HexToHash("0x02"), junk)
		rawdb.WriteAccountSnapshot(diskdb, common.HexToHash("0x03"), junk)
		rawdb.WriteAccountSnapshot(diskdb, common.HexToHash("0x04"), junk)
		rawdb.WriteAccountSnapshot(diskdb, common.HexToHash("0x05"), junk)
	}

	root, _, _ := accTrie.Commit(nil)
	t.Logf("root: %x", root)
	triedb.Commit(root, false, nil)

	snap := generateSnapshot(diskdb, triedb, 16, root)
	select {
	case <-snap.genPending:
		// Snapshot generation succeeded

	case <-time.After(3 * time.Second):
		t.Errorf("Snapshot generation failed")
	}
	checkSnapRoot(t, snap, root)
	// Signal abortion to the generator and wait for it to tear down
	stop := make(chan *generatorStats)
	snap.genAbort <- stop
	<-stop
	// If we now inspect the snap db, there should exist no extraneous storage items
	if data := rawdb.ReadStorageSnapshot(diskdb, hashData([]byte("acc-2")), hashData([]byte("b-key-1"))); data != nil {
		t.Fatalf("expected slot to be removed, got %v", string(data))
	}
}

func TestGenerateFromEmptySnap(t *testing.T) {
	//enableLogging()
	accountCheckRange = 10
	storageCheckRange = 20
	helper := newHelper()
	stRoot := helper.makeStorageTrie([]string{"key-1", "key-2", "key-3"}, []string{"val-1", "val-2", "val-3"})
	// Add 1K accounts to the trie
	for i := 0; i < 400; i++ {
		helper.addTrieAccount(fmt.Sprintf("acc-%d", i),
			&Account{Balance: big.NewInt(1), Root: stRoot, CodeHash: emptyCode.Bytes()})
	}
	root, snap := helper.Generate()
	t.Logf("Root: %#x\n", root) // Root: 0x6f7af6d2e1a1bf2b84a3beb3f8b64388465fbc1e274ca5d5d3fc787ca78f59e4

	select {
	case <-snap.genPending:
		// Snapshot generation succeeded

	case <-time.After(3 * time.Second):
		t.Errorf("Snapshot generation failed")
	}
	checkSnapRoot(t, snap, root)
	// Signal abortion to the generator and wait for it to tear down
	stop := make(chan *generatorStats)
	snap.genAbort <- stop
	<-stop
}

// Tests that snapshot generation with existent flat state, where the flat state
// storage is correct, but incomplete.
// The incomplete part is on the second range
// snap: [ 0x01, 0x02, 0x03, 0x04] , [ 0x05, 0x06, 0x07, {missing}] (with storageCheck = 4)
// trie:  0x01, 0x02, 0x03, 0x04,  0x05, 0x06, 0x07, 0x08
// This hits a case where the snap verification passes, but there are more elements in the trie
// which we must also add.
func TestGenerateWithIncompleteStorage(t *testing.T) {
	storageCheckRange = 4
	helper := newHelper()
	stKeys := []string{"1", "2", "3", "4", "5", "6", "7", "8"}
	stVals := []string{"v1", "v2", "v3", "v4", "v5", "v6", "v7", "v8"}
	stRoot := helper.makeStorageTrie(stKeys, stVals)
	// We add 8 accounts, each one is missing exactly one of the storage slots. This means
	// we don't have to order the keys and figure out exactly which hash-key winds up
	// on the sensitive spots at the boundaries
	for i := 0; i < 8; i++ {
		accKey := fmt.Sprintf("acc-%d", i)
		helper.addAccount(accKey, &Account{Balance: big.NewInt(int64(i)), Root: stRoot, CodeHash: emptyCode.Bytes()})
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

	root, snap := helper.Generate()
	t.Logf("Root: %#x\n", root) // Root: 0xca73f6f05ba4ca3024ef340ef3dfca8fdabc1b677ff13f5a9571fd49c16e67ff

	select {
	case <-snap.genPending:
		// Snapshot generation succeeded

	case <-time.After(3 * time.Second):
		t.Errorf("Snapshot generation failed")
	}
	checkSnapRoot(t, snap, root)
	// Signal abortion to the generator and wait for it to tear down
	stop := make(chan *generatorStats)
	snap.genAbort <- stop
	<-stop
}
