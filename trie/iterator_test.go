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
	"fmt"
	"maps"
	"math/rand"
	"slices"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/trie/trienode"
)

func TestEmptyIterator(t *testing.T) {
	trie := NewEmpty(newTestDatabase(rawdb.NewMemoryDatabase(), rawdb.HashScheme))
	iter := trie.MustNodeIterator(nil)

	seen := make(map[string]struct{})
	for iter.Next(true) {
		seen[string(iter.Path())] = struct{}{}
	}
	if len(seen) != 0 {
		t.Fatal("Unexpected trie node iterated")
	}
}

func TestIterator(t *testing.T) {
	db := newTestDatabase(rawdb.NewMemoryDatabase(), rawdb.HashScheme)
	trie := NewEmpty(db)
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
		trie.MustUpdate([]byte(val.k), []byte(val.v))
	}
	root, nodes := trie.Commit(false)
	db.Update(root, types.EmptyRootHash, trienode.NewWithNodeSet(nodes))

	trie, _ = New(TrieID(root), db)
	found := make(map[string]string)
	it := NewIterator(trie.MustNodeIterator(nil))
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

func (k *kv) cmp(other *kv) int {
	return bytes.Compare(k.k, other.k)
}

func TestIteratorLargeData(t *testing.T) {
	trie := NewEmpty(newTestDatabase(rawdb.NewMemoryDatabase(), rawdb.HashScheme))
	vals := make(map[string]*kv)

	for i := byte(0); i < 255; i++ {
		value := &kv{common.LeftPadBytes([]byte{i}, 32), []byte{i}, false}
		value2 := &kv{common.LeftPadBytes([]byte{10, i}, 32), []byte{i}, false}
		trie.MustUpdate(value.k, value.v)
		trie.MustUpdate(value2.k, value2.v)
		vals[string(value.k)] = value
		vals[string(value2.k)] = value2
	}

	it := NewIterator(trie.MustNodeIterator(nil))
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
	path []byte
	blob []byte
}

// Tests that the node iterator indeed walks over the entire database contents.
func TestNodeIteratorCoverage(t *testing.T) {
	testNodeIteratorCoverage(t, rawdb.HashScheme)
	testNodeIteratorCoverage(t, rawdb.PathScheme)
}

func testNodeIteratorCoverage(t *testing.T, scheme string) {
	// Create some arbitrary test trie to iterate
	db, nodeDb, trie, _ := makeTestTrie(scheme)

	// Gather all the node hashes found by the iterator
	var elements = make(map[common.Hash]iterationElement)
	for it := trie.MustNodeIterator(nil); it.Next(true); {
		if it.Hash() != (common.Hash{}) {
			elements[it.Hash()] = iterationElement{
				hash: it.Hash(),
				path: common.CopyBytes(it.Path()),
				blob: common.CopyBytes(it.NodeBlob()),
			}
		}
	}
	// Cross check the hashes and the database itself
	reader, err := nodeDb.NodeReader(trie.Hash())
	if err != nil {
		t.Fatalf("state is not available %x", trie.Hash())
	}
	for _, element := range elements {
		if blob, err := reader.Node(common.Hash{}, element.path, element.hash); err != nil {
			t.Errorf("failed to retrieve reported node %x: %v", element.hash, err)
		} else if !bytes.Equal(blob, element.blob) {
			t.Errorf("node blob is different, want %v got %v", element.blob, blob)
		}
	}
	var (
		count int
		it    = db.NewIterator(nil, nil)
	)
	for it.Next() {
		res, _, _ := isTrieNode(nodeDb.Scheme(), it.Key(), it.Value())
		if !res {
			continue
		}
		count += 1
		if elem, ok := elements[crypto.Keccak256Hash(it.Value())]; !ok {
			t.Error("state entry not reported")
		} else if !bytes.Equal(it.Value(), elem.blob) {
			t.Errorf("node blob is different, want %v got %v", elem.blob, it.Value())
		}
	}
	it.Release()
	if count != len(elements) {
		t.Errorf("state entry is mismatched %d %d", count, len(elements))
	}
}

type kvs struct{ k, v string }

var testdata1 = []kvs{
	{"bar", "b"},
	{"barb", "ba"},
	{"bard", "bc"},
	{"bars", "bb"},
	{"fab", "z"},
	{"foo", "a"},
	{"food", "ab"},
	{"foos", "aa"},
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
	trie := NewEmpty(newTestDatabase(rawdb.NewMemoryDatabase(), rawdb.HashScheme))
	for _, val := range testdata1 {
		trie.MustUpdate([]byte(val.k), []byte(val.v))
	}

	// Seek to the middle.
	it := NewIterator(trie.MustNodeIterator([]byte("fab")))
	if err := checkIteratorOrder(testdata1[4:], it); err != nil {
		t.Fatal(err)
	}

	// Seek to a non-existent key.
	it = NewIterator(trie.MustNodeIterator([]byte("barc")))
	if err := checkIteratorOrder(testdata1[2:], it); err != nil {
		t.Fatal(err)
	}

	// Seek beyond the end.
	it = NewIterator(trie.MustNodeIterator([]byte("z")))
	if err := checkIteratorOrder(nil, it); err != nil {
		t.Fatal(err)
	}

	// Seek to a key for which a prefixing key exists.
	it = NewIterator(trie.MustNodeIterator([]byte("food")))
	if err := checkIteratorOrder(testdata1[6:], it); err != nil {
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
	dba := newTestDatabase(rawdb.NewMemoryDatabase(), rawdb.HashScheme)
	triea := NewEmpty(dba)
	for _, val := range testdata1 {
		triea.MustUpdate([]byte(val.k), []byte(val.v))
	}
	rootA, nodesA := triea.Commit(false)
	dba.Update(rootA, types.EmptyRootHash, trienode.NewWithNodeSet(nodesA))
	triea, _ = New(TrieID(rootA), dba)

	dbb := newTestDatabase(rawdb.NewMemoryDatabase(), rawdb.HashScheme)
	trieb := NewEmpty(dbb)
	for _, val := range testdata2 {
		trieb.MustUpdate([]byte(val.k), []byte(val.v))
	}
	rootB, nodesB := trieb.Commit(false)
	dbb.Update(rootB, types.EmptyRootHash, trienode.NewWithNodeSet(nodesB))
	trieb, _ = New(TrieID(rootB), dbb)

	found := make(map[string]string)
	di, _ := NewDifferenceIterator(triea.MustNodeIterator(nil), trieb.MustNodeIterator(nil))
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
	dba := newTestDatabase(rawdb.NewMemoryDatabase(), rawdb.HashScheme)
	triea := NewEmpty(dba)
	for _, val := range testdata1 {
		triea.MustUpdate([]byte(val.k), []byte(val.v))
	}
	rootA, nodesA := triea.Commit(false)
	dba.Update(rootA, types.EmptyRootHash, trienode.NewWithNodeSet(nodesA))
	triea, _ = New(TrieID(rootA), dba)

	dbb := newTestDatabase(rawdb.NewMemoryDatabase(), rawdb.HashScheme)
	trieb := NewEmpty(dbb)
	for _, val := range testdata2 {
		trieb.MustUpdate([]byte(val.k), []byte(val.v))
	}
	rootB, nodesB := trieb.Commit(false)
	dbb.Update(rootB, types.EmptyRootHash, trienode.NewWithNodeSet(nodesB))
	trieb, _ = New(TrieID(rootB), dbb)

	di, _ := NewUnionIterator([]NodeIterator{triea.MustNodeIterator(nil), trieb.MustNodeIterator(nil)})
	it := NewIterator(di)

	all := []struct{ k, v string }{
		{"aardvark", "c"},
		{"bar", "b"},
		{"barb", "ba"},
		{"barb", "bd"},
		{"bard", "bc"},
		{"bars", "bb"},
		{"bars", "be"},
		{"fab", "z"},
		{"foo", "a"},
		{"food", "ab"},
		{"foos", "aa"},
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
	db := newTestDatabase(rawdb.NewMemoryDatabase(), rawdb.HashScheme)
	tr := NewEmpty(db)
	for _, val := range testdata1 {
		tr.MustUpdate([]byte(val.k), []byte(val.v))
	}
	checkIteratorNoDups(t, tr.MustNodeIterator(nil), nil)
}

// This test checks that nodeIterator.Next can be retried after inserting missing trie nodes.
func TestIteratorContinueAfterError(t *testing.T) {
	testIteratorContinueAfterError(t, false, rawdb.HashScheme)
	testIteratorContinueAfterError(t, true, rawdb.HashScheme)
	testIteratorContinueAfterError(t, false, rawdb.PathScheme)
	testIteratorContinueAfterError(t, true, rawdb.PathScheme)
}

func testIteratorContinueAfterError(t *testing.T, memonly bool, scheme string) {
	diskdb := rawdb.NewMemoryDatabase()
	tdb := newTestDatabase(diskdb, scheme)

	tr := NewEmpty(tdb)
	for _, val := range testdata1 {
		tr.MustUpdate([]byte(val.k), []byte(val.v))
	}
	root, nodes := tr.Commit(false)
	tdb.Update(root, types.EmptyRootHash, trienode.NewWithNodeSet(nodes))
	if !memonly {
		tdb.Commit(root)
	}
	tr, _ = New(TrieID(root), tdb)
	wantNodeCount := checkIteratorNoDups(t, tr.MustNodeIterator(nil), nil)

	var (
		paths  [][]byte
		hashes []common.Hash
	)
	if memonly {
		for path, n := range nodes.Nodes {
			paths = append(paths, []byte(path))
			hashes = append(hashes, n.Hash)
		}
	} else {
		it := diskdb.NewIterator(nil, nil)
		for it.Next() {
			ok, path, hash := isTrieNode(tdb.Scheme(), it.Key(), it.Value())
			if !ok {
				continue
			}
			paths = append(paths, path)
			hashes = append(hashes, hash)
		}
		it.Release()
	}
	for i := 0; i < 20; i++ {
		// Create trie that will load all nodes from DB.
		tr, _ := New(TrieID(tr.Hash()), tdb)

		// Remove a random node from the database. It can't be the root node
		// because that one is already loaded.
		var (
			rval  []byte
			rpath []byte
			rhash common.Hash
		)
		for {
			if memonly {
				rpath = paths[rand.Intn(len(paths))]
				n := nodes.Nodes[string(rpath)]
				if n == nil {
					continue
				}
				rhash = n.Hash
			} else {
				index := rand.Intn(len(paths))
				rpath = paths[index]
				rhash = hashes[index]
			}
			if rhash != tr.Hash() {
				break
			}
		}
		if memonly {
			tr.reader.banned = map[string]struct{}{string(rpath): {}}
		} else {
			rval = rawdb.ReadTrieNode(diskdb, common.Hash{}, rpath, rhash, tdb.Scheme())
			rawdb.DeleteTrieNode(diskdb, common.Hash{}, rpath, rhash, tdb.Scheme())
		}
		// Iterate until the error is hit.
		seen := make(map[string]bool)
		it := tr.MustNodeIterator(nil)
		checkIteratorNoDups(t, it, seen)
		missing, ok := it.Error().(*MissingNodeError)
		if !ok || missing.NodeHash != rhash {
			t.Fatal("didn't hit missing node, got", it.Error())
		}

		// Add the node back and continue iteration.
		if memonly {
			delete(tr.reader.banned, string(rpath))
		} else {
			rawdb.WriteTrieNode(diskdb, common.Hash{}, rpath, rhash, rval, tdb.Scheme())
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
func TestIteratorContinueAfterSeekError(t *testing.T) {
	testIteratorContinueAfterSeekError(t, false, rawdb.HashScheme)
	testIteratorContinueAfterSeekError(t, true, rawdb.HashScheme)
	testIteratorContinueAfterSeekError(t, false, rawdb.PathScheme)
	testIteratorContinueAfterSeekError(t, true, rawdb.PathScheme)
}

func testIteratorContinueAfterSeekError(t *testing.T, memonly bool, scheme string) {
	// Commit test trie to db, then remove the node containing "bars".
	var (
		barNodePath []byte
		barNodeHash = common.HexToHash("05041990364eb72fcb1127652ce40d8bab765f2bfe53225b1170d276cc101c2e")
	)
	diskdb := rawdb.NewMemoryDatabase()
	triedb := newTestDatabase(diskdb, scheme)
	ctr := NewEmpty(triedb)
	for _, val := range testdata1 {
		ctr.MustUpdate([]byte(val.k), []byte(val.v))
	}
	root, nodes := ctr.Commit(false)
	for path, n := range nodes.Nodes {
		if n.Hash == barNodeHash {
			barNodePath = []byte(path)
			break
		}
	}
	triedb.Update(root, types.EmptyRootHash, trienode.NewWithNodeSet(nodes))
	if !memonly {
		triedb.Commit(root)
	}
	var (
		barNodeBlob []byte
	)
	tr, _ := New(TrieID(root), triedb)
	if memonly {
		tr.reader.banned = map[string]struct{}{string(barNodePath): {}}
	} else {
		barNodeBlob = rawdb.ReadTrieNode(diskdb, common.Hash{}, barNodePath, barNodeHash, triedb.Scheme())
		rawdb.DeleteTrieNode(diskdb, common.Hash{}, barNodePath, barNodeHash, triedb.Scheme())
	}
	// Create a new iterator that seeks to "bars". Seeking can't proceed because
	// the node is missing.
	it := tr.MustNodeIterator([]byte("bars"))
	missing, ok := it.Error().(*MissingNodeError)
	if !ok {
		t.Fatal("want MissingNodeError, got", it.Error())
	} else if missing.NodeHash != barNodeHash {
		t.Fatal("wrong node missing")
	}
	// Reinsert the missing node.
	if memonly {
		delete(tr.reader.banned, string(barNodePath))
	} else {
		rawdb.WriteTrieNode(diskdb, common.Hash{}, barNodePath, barNodeHash, barNodeBlob, triedb.Scheme())
	}
	// Check that iteration produces the right set of values.
	if err := checkIteratorOrder(testdata1[3:], NewIterator(it)); err != nil {
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

func TestIteratorNodeBlob(t *testing.T) {
	testIteratorNodeBlob(t, rawdb.HashScheme)
	testIteratorNodeBlob(t, rawdb.PathScheme)
}

func testIteratorNodeBlob(t *testing.T, scheme string) {
	var (
		db     = rawdb.NewMemoryDatabase()
		triedb = newTestDatabase(db, scheme)
		trie   = NewEmpty(triedb)
	)
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
		trie.MustUpdate([]byte(val.k), []byte(val.v))
	}
	root, nodes := trie.Commit(false)
	triedb.Update(root, types.EmptyRootHash, trienode.NewWithNodeSet(nodes))
	triedb.Commit(root)

	var found = make(map[common.Hash][]byte)
	trie, _ = New(TrieID(root), triedb)
	it := trie.MustNodeIterator(nil)
	for it.Next(true) {
		if it.Hash() == (common.Hash{}) {
			continue
		}
		found[it.Hash()] = it.NodeBlob()
	}

	dbIter := db.NewIterator(nil, nil)
	defer dbIter.Release()

	var count int
	for dbIter.Next() {
		ok, _, _ := isTrieNode(triedb.Scheme(), dbIter.Key(), dbIter.Value())
		if !ok {
			continue
		}
		got, present := found[crypto.Keccak256Hash(dbIter.Value())]
		if !present {
			t.Fatal("Miss trie node")
		}
		if !bytes.Equal(got, dbIter.Value()) {
			t.Fatalf("Unexpected trie node want %v got %v", dbIter.Value(), got)
		}
		count += 1
	}
	if count != len(found) {
		t.Fatal("Find extra trie node via iterator")
	}
}

// isTrieNode is a helper function which reports if the provided
// database entry belongs to a trie node or not. Note in tests
// only single layer trie is used, namely storage trie is not
// considered at all.
func isTrieNode(scheme string, key, val []byte) (bool, []byte, common.Hash) {
	var (
		path []byte
		hash common.Hash
	)
	if scheme == rawdb.HashScheme {
		ok := rawdb.IsLegacyTrieNode(key, val)
		if !ok {
			return false, nil, common.Hash{}
		}
		hash = common.BytesToHash(key)
	} else {
		ok, remain := rawdb.ResolveAccountTrieNodeKey(key)
		if !ok {
			return false, nil, common.Hash{}
		}
		path = common.CopyBytes(remain)
		hash = crypto.Keccak256Hash(val)
	}
	return true, path, hash
}

func TestSubtreeIterator(t *testing.T) {
	var (
		db = newTestDatabase(rawdb.NewMemoryDatabase(), rawdb.HashScheme)
		tr = NewEmpty(db)
	)
	vals := []struct{ k, v string }{
		{"do", "verb"},
		{"dog", "puppy"},
		{"doge", "coin"},
		{"dog\xff", "value6"},
		{"dog\xff\xff", "value7"},
		{"horse", "stallion"},
		{"house", "building"},
		{"houses", "multiple"},
		{"xyz", "value"},
		{"xyz\xff", "value"},
		{"xyz\xff\xff", "value"},
	}
	all := make(map[string]string)
	for _, val := range vals {
		all[val.k] = val.v
		tr.MustUpdate([]byte(val.k), []byte(val.v))
	}
	root, nodes := tr.Commit(false)
	db.Update(root, types.EmptyRootHash, trienode.NewWithNodeSet(nodes))

	allNodes := make(map[string][]byte)
	tr, _ = New(TrieID(root), db)
	it, err := tr.NodeIterator(nil)
	if err != nil {
		t.Fatal(err)
	}
	for it.Next(true) {
		allNodes[string(it.Path())] = it.NodeBlob()
	}
	allKeys := slices.Collect(maps.Keys(all))

	suites := []struct {
		start    []byte
		end      []byte
		expected []string
	}{
		// entire key range
		{
			start:    nil,
			end:      nil,
			expected: allKeys,
		},
		{
			start:    nil,
			end:      bytes.Repeat([]byte{0xff}, 32),
			expected: allKeys,
		},
		{
			start:    bytes.Repeat([]byte{0x0}, 32),
			end:      bytes.Repeat([]byte{0xff}, 32),
			expected: allKeys,
		},
		// key range with start
		{
			start:    []byte("do"),
			end:      nil,
			expected: allKeys,
		},
		{
			start:    []byte("doe"),
			end:      nil,
			expected: allKeys[1:],
		},
		{
			start:    []byte("dog"),
			end:      nil,
			expected: allKeys[1:],
		},
		{
			start:    []byte("doge"),
			end:      nil,
			expected: allKeys[2:],
		},
		{
			start:    []byte("dog\xff"),
			end:      nil,
			expected: allKeys[3:],
		},
		{
			start:    []byte("dog\xff\xff"),
			end:      nil,
			expected: allKeys[4:],
		},
		{
			start:    []byte("dog\xff\xff\xff"),
			end:      nil,
			expected: allKeys[5:],
		},
		// key range with limit
		{
			start:    nil,
			end:      []byte("xyz"),
			expected: allKeys[:len(allKeys)-3],
		},
		{
			start:    nil,
			end:      []byte("xyz\xff"),
			expected: allKeys[:len(allKeys)-2],
		},
		{
			start:    nil,
			end:      []byte("xyz\xff\xff"),
			expected: allKeys[:len(allKeys)-1],
		},
		{
			start:    nil,
			end:      []byte("xyz\xff\xff\xff"),
			expected: allKeys,
		},
	}
	for _, suite := range suites {
		// We need to re-open the trie from the committed state
		tr, _ = New(TrieID(root), db)
		it, err := newSubtreeIterator(tr, suite.start, suite.end)
		if err != nil {
			t.Fatal(err)
		}

		found := make(map[string]string)
		for it.Next(true) {
			if it.Leaf() {
				found[string(it.LeafKey())] = string(it.LeafBlob())
			}
		}
		if len(found) != len(suite.expected) {
			t.Errorf("wrong number of values: got %d, want %d", len(found), len(suite.expected))
		}
		for k, v := range found {
			if all[k] != v {
				t.Errorf("wrong value for %s: got %s, want %s", k, found[k], all[k])
			}
		}

		expectedNodes := make(map[string][]byte)
		for path, blob := range allNodes {
			if suite.start != nil {
				hexStart := keybytesToHex(suite.start)
				hexStart = hexStart[:len(hexStart)-1]
				if !reachedPath([]byte(path), hexStart) {
					continue
				}
			}
			if suite.end != nil {
				hexEnd := keybytesToHex(suite.end)
				hexEnd = hexEnd[:len(hexEnd)-1]
				if reachedPath([]byte(path), hexEnd) {
					continue
				}
			}
			expectedNodes[path] = bytes.Clone(blob)
		}

		// Compare the result yield from the subtree iterator
		var (
			subCount int
			subIt, _ = newSubtreeIterator(tr, suite.start, suite.end)
		)
		for subIt.Next(true) {
			blob, ok := expectedNodes[string(subIt.Path())]
			if !ok {
				t.Errorf("Unexpected node iterated, path: %v", subIt.Path())
			}
			subCount++

			if !bytes.Equal(blob, subIt.NodeBlob()) {
				t.Errorf("Unexpected node blob, path: %v, want: %v, got: %v", subIt.Path(), blob, subIt.NodeBlob())
			}
		}
		if subCount != len(expectedNodes) {
			t.Errorf("Unexpected node being iterated, want: %d, got: %d", len(expectedNodes), subCount)
		}
	}
}

func TestPrefixIterator(t *testing.T) {
	// Create a new trie
	trie := NewEmpty(newTestDatabase(rawdb.NewMemoryDatabase(), rawdb.HashScheme))

	// Insert test data
	testData := map[string]string{
		"key1":      "value1",
		"key2":      "value2",
		"key10":     "value10",
		"key11":     "value11",
		"different": "value_different",
	}

	for key, value := range testData {
		trie.Update([]byte(key), []byte(value))
	}

	// Test prefix iteration for "key1" prefix
	prefix := []byte("key1")
	iter, err := trie.NodeIteratorWithPrefix(prefix)
	if err != nil {
		t.Fatalf("Failed to create prefix iterator: %v", err)
	}

	var foundKeys [][]byte
	for iter.Next(true) {
		if iter.Leaf() {
			foundKeys = append(foundKeys, iter.LeafKey())
		}
	}

	if err := iter.Error(); err != nil {
		t.Fatalf("Iterator error: %v", err)
	}

	// Verify only keys starting with "key1" were found
	expectedCount := 3 // "key1", "key10", "key11"
	if len(foundKeys) != expectedCount {
		t.Errorf("Expected %d keys, found %d", expectedCount, len(foundKeys))
	}

	for _, key := range foundKeys {
		keyStr := string(key)
		if !bytes.HasPrefix(key, prefix) {
			t.Errorf("Found key %s doesn't have prefix %s", keyStr, string(prefix))
		}
	}
}

func TestPrefixIteratorVsFullIterator(t *testing.T) {
	// Create a new trie with more structured data
	trie := NewEmpty(newTestDatabase(rawdb.NewMemoryDatabase(), rawdb.HashScheme))

	// Insert structured test data
	testData := map[string]string{
		"aaa": "value_aaa",
		"aab": "value_aab",
		"aba": "value_aba",
		"bbb": "value_bbb",
	}

	for key, value := range testData {
		trie.Update([]byte(key), []byte(value))
	}

	// Test that prefix iterator stops at boundary
	prefix := []byte("aa")
	prefixIter, err := trie.NodeIteratorWithPrefix(prefix)
	if err != nil {
		t.Fatalf("Failed to create prefix iterator: %v", err)
	}

	var prefixKeys [][]byte
	for prefixIter.Next(true) {
		if prefixIter.Leaf() {
			prefixKeys = append(prefixKeys, prefixIter.LeafKey())
		}
	}

	// Should only find "aaa" and "aab", not "aba" or "bbb"
	if len(prefixKeys) != 2 {
		t.Errorf("Expected 2 keys with prefix 'aa', found %d", len(prefixKeys))
	}

	// Verify no keys outside prefix were found
	for _, key := range prefixKeys {
		if !bytes.HasPrefix(key, prefix) {
			t.Errorf("Prefix iterator returned key %s outside prefix %s", string(key), string(prefix))
		}
	}
}

func TestEmptyPrefixIterator(t *testing.T) {
	// Test with empty trie
	trie := NewEmpty(newTestDatabase(rawdb.NewMemoryDatabase(), rawdb.HashScheme))

	iter, err := trie.NodeIteratorWithPrefix([]byte("nonexistent"))
	if err != nil {
		t.Fatalf("Failed to create iterator: %v", err)
	}

	if iter.Next(true) {
		t.Error("Expected no results from empty trie")
	}
}

// TestPrefixIteratorEdgeCases tests various edge cases for prefix iteration
func TestPrefixIteratorEdgeCases(t *testing.T) {
	// Create a trie with test data
	trie := NewEmpty(newTestDatabase(rawdb.NewMemoryDatabase(), rawdb.HashScheme))
	testData := map[string]string{
		"abc":         "value1",
		"abcd":        "value2",
		"abce":        "value3",
		"abd":         "value4",
		"dog":         "value5",
		"dog\xff":     "value6", // Test with 0xff byte
		"dog\xff\xff": "value7", // Multiple 0xff bytes
	}
	for key, value := range testData {
		trie.Update([]byte(key), []byte(value))
	}

	// Test 1: Prefix not present in trie
	t.Run("NonexistentPrefix", func(t *testing.T) {
		iter, err := trie.NodeIteratorWithPrefix([]byte("xyz"))
		if err != nil {
			t.Fatalf("Failed to create iterator: %v", err)
		}
		count := 0
		for iter.Next(true) {
			if iter.Leaf() {
				count++
			}
		}
		if count != 0 {
			t.Errorf("Expected 0 results for nonexistent prefix, got %d", count)
		}
	})

	// Test 2: Prefix exactly equals an existing key
	t.Run("ExactKeyPrefix", func(t *testing.T) {
		iter, err := trie.NodeIteratorWithPrefix([]byte("abc"))
		if err != nil {
			t.Fatalf("Failed to create iterator: %v", err)
		}
		found := make(map[string]bool)
		for iter.Next(true) {
			if iter.Leaf() {
				found[string(iter.LeafKey())] = true
			}
		}
		// Should find "abc", "abcd", "abce" but not "abd"
		if !found["abc"] || !found["abcd"] || !found["abce"] {
			t.Errorf("Missing expected keys: got %v", found)
		}
		if found["abd"] {
			t.Errorf("Found unexpected key 'abd' with prefix 'abc'")
		}
	})

	// Test 3: Prefix with trailing 0xff
	t.Run("TrailingFFPrefix", func(t *testing.T) {
		iter, err := trie.NodeIteratorWithPrefix([]byte("dog\xff"))
		if err != nil {
			t.Fatalf("Failed to create iterator: %v", err)
		}
		found := make(map[string]bool)
		for iter.Next(true) {
			if iter.Leaf() {
				found[string(iter.LeafKey())] = true
			}
		}
		// Should find "dog\xff" and "dog\xff\xff"
		if !found["dog\xff"] || !found["dog\xff\xff"] {
			t.Errorf("Missing expected keys with 0xff: got %v", found)
		}
		if found["dog"] {
			t.Errorf("Found unexpected key 'dog' with prefix 'dog\\xff'")
		}
	})

	// Test 4: All 0xff case (edge case for nextKey)
	t.Run("AllFFPrefix", func(t *testing.T) {
		// Add a key with all 0xff bytes
		allFF := []byte{0xff, 0xff}
		trie.Update(allFF, []byte("all_ff_value"))
		trie.Update(append(allFF, 0x00), []byte("all_ff_plus"))

		iter, err := trie.NodeIteratorWithPrefix(allFF)
		if err != nil {
			t.Fatalf("Failed to create iterator: %v", err)
		}
		count := 0
		for iter.Next(true) {
			if iter.Leaf() {
				count++
			}
		}
		// Should find exactly the two keys with the all-0xff prefix
		if count != 2 {
			t.Errorf("Expected 2 results for all-0xff prefix, got %d", count)
		}
	})

	// Test 5: Empty prefix (should iterate entire trie)
	t.Run("EmptyPrefix", func(t *testing.T) {
		iter, err := trie.NodeIteratorWithPrefix([]byte{})
		if err != nil {
			t.Fatalf("Failed to create iterator: %v", err)
		}
		count := 0
		for iter.Next(true) {
			if iter.Leaf() {
				count++
			}
		}
		// Should find all keys in the trie
		expectedCount := len(testData) + 2 // +2 for the extra keys added in test 4
		if count != expectedCount {
			t.Errorf("Expected %d results for empty prefix, got %d", expectedCount, count)
		}
	})
}

// TestGeneralRangeIteration tests NewSubtreeIterator with arbitrary start/stop ranges
func TestGeneralRangeIteration(t *testing.T) {
	// Create a trie with test data
	trie := NewEmpty(newTestDatabase(rawdb.NewMemoryDatabase(), rawdb.HashScheme))
	testData := map[string]string{
		"apple":   "fruit1",
		"apricot": "fruit2",
		"banana":  "fruit3",
		"cherry":  "fruit4",
		"date":    "fruit5",
		"fig":     "fruit6",
		"grape":   "fruit7",
	}
	for key, value := range testData {
		trie.Update([]byte(key), []byte(value))
	}

	// Test range iteration from "banana" to "fig" (exclusive)
	t.Run("RangeIteration", func(t *testing.T) {
		iter, _ := newSubtreeIterator(trie, []byte("banana"), []byte("fig"))
		found := make(map[string]bool)
		for iter.Next(true) {
			if iter.Leaf() {
				found[string(iter.LeafKey())] = true
			}
		}
		// Should find "banana", "cherry", "date" but not "fig"
		if !found["banana"] || !found["cherry"] || !found["date"] {
			t.Errorf("Missing expected keys in range: got %v", found)
		}
		if found["apple"] || found["apricot"] || found["fig"] || found["grape"] {
			t.Errorf("Found unexpected keys outside range: got %v", found)
		}
	})

	// Test with nil stopKey (iterate to end)
	t.Run("NilStopKey", func(t *testing.T) {
		iter, _ := newSubtreeIterator(trie, []byte("date"), nil)
		found := make(map[string]bool)
		for iter.Next(true) {
			if iter.Leaf() {
				found[string(iter.LeafKey())] = true
			}
		}
		// Should find "date", "fig", "grape"
		if !found["date"] || !found["fig"] || !found["grape"] {
			t.Errorf("Missing expected keys from 'date' to end: got %v", found)
		}
		if found["apple"] || found["banana"] || found["cherry"] {
			t.Errorf("Found unexpected keys before 'date': got %v", found)
		}
	})

	// Test with nil startKey (iterate from beginning)
	t.Run("NilStartKey", func(t *testing.T) {
		iter, _ := newSubtreeIterator(trie, nil, []byte("cherry"))
		found := make(map[string]bool)
		for iter.Next(true) {
			if iter.Leaf() {
				found[string(iter.LeafKey())] = true
			}
		}
		// Should find "apple", "apricot", "banana" but not "cherry" or later
		if !found["apple"] || !found["apricot"] || !found["banana"] {
			t.Errorf("Missing expected keys before 'cherry': got %v", found)
		}
		if found["cherry"] || found["date"] || found["fig"] || found["grape"] {
			t.Errorf("Found unexpected keys at or after 'cherry': got %v", found)
		}
	})
}

// TestPrefixIteratorWithDescend tests prefix iteration with descend=false
func TestPrefixIteratorWithDescend(t *testing.T) {
	// Create a trie with nested structure
	trie := NewEmpty(newTestDatabase(rawdb.NewMemoryDatabase(), rawdb.HashScheme))
	testData := map[string]string{
		"a":     "value_a",
		"a/b":   "value_ab",
		"a/b/c": "value_abc",
		"a/b/d": "value_abd",
		"a/e":   "value_ae",
		"b":     "value_b",
	}
	for key, value := range testData {
		trie.Update([]byte(key), []byte(value))
	}

	// Test skipping subtrees with descend=false
	t.Run("SkipSubtrees", func(t *testing.T) {
		iter, err := trie.NodeIteratorWithPrefix([]byte("a"))
		if err != nil {
			t.Fatalf("Failed to create iterator: %v", err)
		}

		// Count nodes at each level
		nodesVisited := 0
		leafsFound := make(map[string]bool)

		// First call with descend=true to enter the "a" subtree
		if !iter.Next(true) {
			t.Fatal("Expected to find at least one node")
		}
		nodesVisited++

		// Continue iteration, sometimes with descend=false
		descendPattern := []bool{false, true, false, true, true}
		for i := 0; iter.Next(descendPattern[i%len(descendPattern)]); i++ {
			nodesVisited++
			if iter.Leaf() {
				leafsFound[string(iter.LeafKey())] = true
			}
		}

		// We should still respect the prefix boundary even when skipping
		prefix := []byte("a")
		for key := range leafsFound {
			if !bytes.HasPrefix([]byte(key), prefix) {
				t.Errorf("Found key outside prefix when using descend=false: %s", key)
			}
		}

		// Should not have found "b" even if we skip some subtrees
		if leafsFound["b"] {
			t.Error("Iterator leaked outside prefix boundary with descend=false")
		}
	})
}

func BenchmarkIterator(b *testing.B) {
	diskDb, srcDb, tr, _ := makeTestTrie(rawdb.HashScheme)
	root := tr.Hash()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if err := checkTrieConsistency(diskDb, srcDb.Scheme(), root, false); err != nil {
			b.Fatal(err)
		}
	}
}
