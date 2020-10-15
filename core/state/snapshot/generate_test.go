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
	"math/big"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethdb/memorydb"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/ethereum/go-ethereum/trie"
)

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

	snap := generateSnapshot(diskdb, triedb, 16, common.HexToHash("0xa04693ea110a31037fb5ee814308a6f1d76bdab0b11676bdf4541d2de55ba978"), nil)
	select {
	case <-snap.genPending:
		// Snapshot generation succeeded
		t.Errorf("Snapshot generated against corrupt account trie")

	case <-time.After(250 * time.Millisecond):
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

	snap := generateSnapshot(diskdb, triedb, 16, common.HexToHash("0xe3712f1a226f3782caca78ca770ccc19ee000552813a9f59d479f8611db9b1fd"), nil)
	select {
	case <-snap.genPending:
		// Snapshot generation succeeded
		t.Errorf("Snapshot generated against corrupt storage trie")

	case <-time.After(250 * time.Millisecond):
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

	snap := generateSnapshot(diskdb, triedb, 16, common.HexToHash("0xe3712f1a226f3782caca78ca770ccc19ee000552813a9f59d479f8611db9b1fd"), nil)
	select {
	case <-snap.genPending:
		// Snapshot generation succeeded
		t.Errorf("Snapshot generated against corrupt storage trie")

	case <-time.After(250 * time.Millisecond):
		// Not generated fast enough, hopefully blocked inside on missing trie node fail
	}
	// Signal abortion to the generator and wait for it to tear down
	stop := make(chan *generatorStats)
	snap.genAbort <- stop
	<-stop
}
