// Copyright 2014 The go-ethereum Authors
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

package trie

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"math/rand"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/ethdb/memorydb"
)

func TestIterator(t *testing.T) {
	trie := newEmpty()
	vals := []struct{ k, v string }{
		{"do", "verb"},
		{"ether", "wookiedoo"},
		{"horse", "stallion"},
		{"shaman", "horse"},
		{"doge", "coin"},
		{"dog", "puppy"},
		{"somethingveryoddindeedthis is", "myothernodedata"},
	}
	all := make(map[string]string)
	for _, val := range vals {
		all[val.k] = val.v
		trie.Update([]byte(val.k), []byte(val.v))
	}
	trie.Commit(nil)
	found := make(map[string]string)
	it := NewIterator(trie.NodeIterator(nil))
	for it.Next() {
		found[string(it.Key)] = string(it.Value)
	}

	for k, v := range all {
		if found[k] != v {
			t.Errorf("iterator value mismatch for %s: got %q want %q", k, found[k], v)
		}
	}
}

type kv struct {
	k, v []byte
	t    bool
}

func TestIteratorLargeData(t *testing.T) {
	trie := newEmpty()
	vals := make(map[string]*kv)

	for i := byte(0); i < 255; i++ {
		value := &kv{common.LeftPadBytes([]byte{i}, 32), []byte{i}, false}
		value2 := &kv{common.LeftPadBytes([]byte{10, i}, 32), []byte{i}, false}
		trie.Update(value.k, value.v)
		trie.Update(value2.k, value2.v)
		vals[string(value.k)] = value
		vals[string(value2.k)] = value2
	}

	it := NewIterator(trie.NodeIterator(nil))
	for it.Next() {
		vals[string(it.Key)].t = true
	}

	var untouched []*kv
	for _, value := range vals {
		if !value.t {
			untouched = append(untouched, value)
		}
	}

	if len(untouched) > 0 {
		t.Errorf("Missed %d nodes", len(untouched))
		for _, value := range untouched {
			t.Error(value)
		}
	}
}

type iterationElement struct {
	hash common.Hash
	key  []byte
}

// Tests that the node iterator indeed walks over the entire database contents.
func TestNodeIteratorCoverage(t *testing.T) {
	// Create some arbitrary test trie to iterate
	db, trie, _ := makeTestTrie()

	// Gather all the node hashes found by the iterator
	var elements = make(map[string]iterationElement)
	for it := trie.NodeIterator(nil); it.Next(true); {
		if it.Hash() != (common.Hash{}) {
			key := string(EncodeInternalKey(it.StorageKey(), it.Hash()))
			elements[key] = iterationElement{
				hash: it.Hash(),
				key:  it.StorageKey(),
			}
		}
	}
	// Cross check the hashes and the database itself
	for key, element := range elements {
		if _, err := db.Snapshot(trie.Hash()).NodeBlob([]byte(key)); err != nil {
			t.Errorf("failed to retrieve reported node %x: %v", element.hash, err)
		}
	}
	it := db.DiskDB().NewIterator(nil, nil)
	for it.Next() {
		key := it.Key()
		ok, nodeKey := rawdb.IsTrieNodeKey(key)
		if !ok {
			t.Errorf("state entry not reported %v", key)
		}
		hash := crypto.Keccak256Hash(it.Value())
		ikey := string(EncodeInternalKey(nodeKey, hash))
		if _, ok := elements[ikey]; !ok {
			t.Errorf("state entry not reported %v", suffixCompactToHex(nodeKey))
		}
	}
	it.Release()
}

type kvs struct{ k, v string }

var testdata1 = []kvs{
	{"barb", "ba"},
	{"bard", "bc"},
	{"bars", "bb"},
	{"bar", "b"},
	{"fab", "z"},
	{"food", "ab"},
	{"foos", "aa"},
	{"foo", "a"},
}

var testdata2 = []kvs{
	{"aardvark", "c"},
	{"bar", "b"},
	{"barb", "bd"},
	{"bars", "be"},
	{"fab", "z"},
	{"foo", "a"},
	{"foos", "aa"},
	{"food", "ab"},
	{"jars", "d"},
}

func TestIteratorSeek(t *testing.T) {
	trie := newEmpty()
	for _, val := range testdata1 {
		trie.Update([]byte(val.k), []byte(val.v))
	}

	// Seek to the middle.
	it := NewIterator(trie.NodeIterator([]byte("fab")))
	if err := checkIteratorOrder(testdata1[4:], it); err != nil {
		t.Fatal(err)
	}

	// Seek to a non-existent key.
	it = NewIterator(trie.NodeIterator([]byte("barc")))
	if err := checkIteratorOrder(testdata1[1:], it); err != nil {
		t.Fatal(err)
	}

	// Seek beyond the end.
	it = NewIterator(trie.NodeIterator([]byte("z")))
	if err := checkIteratorOrder(nil, it); err != nil {
		t.Fatal(err)
	}
}

func checkIteratorOrder(want []kvs, it *Iterator) error {
	for it.Next() {
		if len(want) == 0 {
			return fmt.Errorf("didn't expect any more values, got key %q", it.Key)
		}
		if !bytes.Equal(it.Key, []byte(want[0].k)) {
			return fmt.Errorf("wrong key: got %q, want %q", it.Key, want[0].k)
		}
		want = want[1:]
	}
	if len(want) > 0 {
		return fmt.Errorf("iterator ended early, want key %q", want[0])
	}
	return nil
}

func TestDifferenceIterator(t *testing.T) {
	triea := newEmpty()
	for _, val := range testdata1 {
		triea.Update([]byte(val.k), []byte(val.v))
	}
	triea.Commit(nil)

	trieb := newEmpty()
	for _, val := range testdata2 {
		trieb.Update([]byte(val.k), []byte(val.v))
	}
	trieb.Commit(nil)

	found := make(map[string]string)
	di, _ := NewDifferenceIterator(triea.NodeIterator(nil), trieb.NodeIterator(nil))
	it := NewIterator(di)
	for it.Next() {
		found[string(it.Key)] = string(it.Value)
	}

	all := []struct{ k, v string }{
		{"aardvark", "c"},
		{"barb", "bd"},
		{"bars", "be"},
		{"jars", "d"},
	}
	for _, item := range all {
		if found[item.k] != item.v {
			t.Errorf("iterator value mismatch for %s: got %v want %v", item.k, found[item.k], item.v)
		}
	}
	if len(found) != len(all) {
		t.Errorf("iterator count mismatch: got %d values, want %d", len(found), len(all))
	}
}

func TestUnionIterator(t *testing.T) {
	triea := newEmpty()
	for _, val := range testdata1 {
		triea.Update([]byte(val.k), []byte(val.v))
	}
	triea.Commit(nil)

	trieb := newEmpty()
	for _, val := range testdata2 {
		trieb.Update([]byte(val.k), []byte(val.v))
	}
	trieb.Commit(nil)

	di, _ := NewUnionIterator([]NodeIterator{triea.NodeIterator(nil), trieb.NodeIterator(nil)})
	it := NewIterator(di)

	all := []struct{ k, v string }{
		{"aardvark", "c"},
		{"barb", "ba"},
		{"barb", "bd"},
		{"bard", "bc"},
		{"bars", "bb"},
		{"bars", "be"},
		{"bar", "b"},
		{"fab", "z"},
		{"food", "ab"},
		{"foos", "aa"},
		{"foo", "a"},
		{"jars", "d"},
	}

	for i, kv := range all {
		if !it.Next() {
			t.Errorf("Iterator ends prematurely at element %d", i)
		}
		if kv.k != string(it.Key) {
			t.Errorf("iterator value mismatch for element %d: got key %s want %s", i, it.Key, kv.k)
		}
		if kv.v != string(it.Value) {
			t.Errorf("iterator value mismatch for element %d: got value %s want %s", i, it.Value, kv.v)
		}
	}
	if it.Next() {
		t.Errorf("Iterator returned extra values.")
	}
}

func TestIteratorNoDups(t *testing.T) {
	var tr Trie
	for _, val := range testdata1 {
		tr.Update([]byte(val.k), []byte(val.v))
	}
	checkIteratorNoDups(t, tr.NodeIterator(nil), nil)
}

// This test checks that nodeIterator.Next can be retried after inserting missing trie nodes.
func TestIteratorContinueAfterErrorDisk(t *testing.T)    { testIteratorContinueAfterError(t, false) }
func TestIteratorContinueAfterErrorMemonly(t *testing.T) { testIteratorContinueAfterError(t, true) }

func testIteratorContinueAfterError(t *testing.T, memonly bool) {
	if memonly {
		t.Skip("FIX IT")
	}
	diskdb := memorydb.New()
	tdb := NewDatabase(diskdb, nil)

	tr, _ := New(common.Hash{}, tdb)
	for _, val := range testdata1 {
		tr.Update([]byte(val.k), []byte(val.v))
	}
	result, _ := tr.Commit(nil)
	tdb.Update(result.Root, common.Hash{}, result.CommitTo(nil))
	if !memonly {
		tdb.Cap(result.Root, 0)
	}
	wantNodeCount := checkIteratorNoDups(t, tr.NodeIterator(nil), nil)

	var (
		diskKeys [][]byte
		//memKeys  []string
	)
	if memonly {
		//for k, v := range result.CommitTo(nil) {
		//	if len(v) == 0 {
		//		continue
		//	}
		//	memKeys = append(memKeys, k)
		//}
	} else {
		it := diskdb.NewIterator(nil, nil)
		for it.Next() {
			ok, nodeKey := rawdb.IsTrieNodeKey(it.Key())
			if ok {
				diskKeys = append(diskKeys, common.CopyBytes(nodeKey))
			}
		}
		it.Release()
	}
	for i := 0; i < 20; i++ {
		// Create trie that will load all nodes from DB.
		tr, _ := New(tr.Hash(), tdb)

		// Remove a random node from the database. It can't be the root node
		// because that one is already loaded.
		var (
			rkey  []byte
			rval  []byte
			rhash common.Hash
		)
		for {
			if memonly {
				//rkey = []byte(memKeys[rand.Intn(len(memKeys))])
				//val := result.CommitTo(nil)[string(rkey)]
				//if val == nil {
				//	continue
				//}
				//rhash = crypto.Keccak256Hash(val)
			} else {
				rkey = common.CopyBytes(diskKeys[rand.Intn(len(diskKeys))])
				_, rhash = rawdb.ReadTrieNode(diskdb, rkey)
			}
			if rhash != tr.Hash() {
				break
			}
		}
		if memonly {
			//rval = tr.dirty.updated[string(rkey)]
			//delete(tr.dirty.updated, string(rkey))
		} else {
			rval, _ = rawdb.ReadTrieNode(diskdb, rkey)
			rawdb.DeleteTrieNode(diskdb, rkey)
		}
		// Iterate until the error is hit.
		seen := make(map[string]bool)
		it := tr.NodeIterator(nil)
		checkIteratorNoDups(t, it, seen)
		missing, ok := it.Error().(*MissingNodeError)
		if !ok || missing.NodeHash != rhash {
			t.Fatal("didn't hit missing node, got", it.Error())
		}

		// Add the node back and continue iteration.
		if memonly {
			//tr.dirty.updated[string(rkey)] = rval
		} else {
			rawdb.WriteTrieNode(diskdb, rkey, rval)
		}
		checkIteratorNoDups(t, it, seen)
		if it.Error() != nil {
			t.Fatal("unexpected error", it.Error())
		}
		if len(seen) != wantNodeCount {
			t.Fatal("wrong node iteration count, got", len(seen), "want", wantNodeCount)
		}
	}
}

// Similar to the test above, this one checks that failure to create nodeIterator at a
// certain key prefix behaves correctly when Next is called. The expectation is that Next
// should retry seeking before returning true for the first time.
func TestIteratorContinueAfterSeekErrorDisk(t *testing.T) {
	testIteratorContinueAfterSeekError(t, false)
}
func TestIteratorContinueAfterSeekErrorMemonly(t *testing.T) {
	testIteratorContinueAfterSeekError(t, true)
}

func testIteratorContinueAfterSeekError(t *testing.T, memonly bool) {
	if memonly {
		t.Skip("FIX ME")
	}
	// Commit test trie to db, then remove the node containing "bars".
	var (
		barNodeHash = common.HexToHash("05041990364eb72fcb1127652ce40d8bab765f2bfe53225b1170d276cc101c2e")
		barNodeKey  []byte
	)
	diskdb := memorydb.New()
	triedb := NewDatabase(diskdb, nil)
	ctr, _ := New(common.Hash{}, triedb)
	for _, val := range testdata1 {
		ctr.Update([]byte(val.k), []byte(val.v))
	}
	result, _ := ctr.Commit(nil)
	root := result.Root

	for key := range result.Nodes() {
		_, hash := DecodeInternalKey([]byte(key))
		if hash == barNodeHash {
			barNodeKey = []byte(key)
		}
	}
	if !memonly {
		triedb.Update(root, common.Hash{}, result.CommitTo(nil))
		triedb.Cap(root, 0)
	}
	var (
		barNodeBlob []byte
	)
	if memonly {
	} else {
		storage, _ := DecodeInternalKey(barNodeKey)
		blob, _ := rawdb.ReadTrieNode(diskdb, storage)
		rawdb.DeleteTrieNode(diskdb, storage)
		barNodeBlob = blob
	}
	// Create a new iterator that seeks to "bars". Seeking can't proceed because
	// the node is missing.
	tr, _ := New(root, triedb)
	it := tr.NodeIterator([]byte("bars"))
	missing, ok := it.Error().(*MissingNodeError)
	if !ok {
		t.Fatal("want MissingNodeError, got", it.Error())
	} else if missing.NodeHash != barNodeHash {
		t.Fatal("wrong node missing")
	}
	// Reinsert the missing node.
	if memonly {
	} else {
		storage, _ := DecodeInternalKey(barNodeKey)
		rawdb.WriteTrieNode(diskdb, storage, barNodeBlob)
	}
	// Check that iteration produces the right set of values.
	if err := checkIteratorOrder(testdata1[2:], NewIterator(it)); err != nil {
		t.Fatal(err)
	}
}

func checkIteratorNoDups(t *testing.T, it NodeIterator, seen map[string]bool) int {
	if seen == nil {
		seen = make(map[string]bool)
	}
	for it.Next(true) {
		if seen[string(it.Path())] {
			t.Fatalf("iterator visited node path %x twice", it.Path())
		}
		seen[string(it.Path())] = true
	}
	return len(seen)
}

type loggingDb struct {
	getCount uint64
	backend  ethdb.KeyValueStore
}

func (l *loggingDb) Has(key []byte) (bool, error) {
	return l.backend.Has(key)
}

func (l *loggingDb) Get(key []byte) ([]byte, error) {
	l.getCount++
	return l.backend.Get(key)
}

func (l *loggingDb) Put(key []byte, value []byte) error {
	return l.backend.Put(key, value)
}

func (l *loggingDb) Delete(key []byte) error {
	return l.backend.Delete(key)
}

func (l *loggingDb) NewBatch() ethdb.Batch {
	return l.backend.NewBatch()
}

func (l *loggingDb) NewIterator(prefix []byte, start []byte) ethdb.Iterator {
	return l.backend.NewIterator(prefix, start)
}
func (l *loggingDb) Stat(property string) (string, error) {
	return l.backend.Stat(property)
}

func (l *loggingDb) Compact(start []byte, limit []byte) error {
	return l.backend.Compact(start, limit)
}

func (l *loggingDb) Close() error {
	return l.backend.Close()
}

// makeLargeTestTrie create a sample test trie
func makeLargeTestTrie() (*Database, *SecureTrie, *loggingDb) {
	// Create an empty trie
	logDb := &loggingDb{0, memorydb.New()}
	triedb := NewDatabase(logDb, nil)
	trie, _ := NewSecure(common.Hash{}, triedb)

	// Fill it with some arbitrary data
	for i := 0; i < 10000; i++ {
		key := make([]byte, 32)
		val := make([]byte, 32)
		binary.BigEndian.PutUint64(key, uint64(i))
		binary.BigEndian.PutUint64(val, uint64(i))
		key = crypto.Keccak256(key)
		val = crypto.Keccak256(val)
		trie.Update(key, val)
	}
	result, _ := trie.Commit(nil)
	triedb.Update(result.Root, common.Hash{}, result.CommitTo(nil))
	triedb.Cap(result.Root, 0)
	// Return the generated trie
	return triedb, trie, logDb
}

// Tests that the node iterator indeed walks over the entire database contents.
func TestNodeIteratorLargeTrie(t *testing.T) {
	// Create some arbitrary test trie to iterate
	_, trie, logDb := makeLargeTestTrie()
	// Do a seek operation
	trie.NodeIterator(common.FromHex("0x77667766776677766778855885885885"))
	// master: 24 get operations
	// this pr: 2 get operations
	if have, want := logDb.getCount, uint64(2); have != want {
		t.Fatalf("Too many lookups during seek, have %d want %d", have, want)
	}
}
