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
	"io/ioutil"
	"math/rand"
	"os"
	"reflect"
	"testing"
	"testing/quick"

	"github.com/davecgh/go-spew/spew"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethdb"
)

func init() {
	spew.Config.Indent = "    "
	spew.Config.DisableMethods = true
}

// Used for testing
func newEmpty() *Trie {
	db, _ := ethdb.NewMemDatabase()
	trie, _ := New(common.Hash{}, db)
	return trie
}

func TestEmptyTrie(t *testing.T) {
	var trie Trie
	res := trie.Hash()
	exp := emptyRoot
	if res != common.Hash(exp) {
		t.Errorf("expected %x got %x", exp, res)
	}
}

func TestNull(t *testing.T) {
	var trie Trie
	key := make([]byte, 32)
	value := common.FromHex("0x823140710bf13990e4500136726d8b55")
	trie.Update(key, value)
	value = trie.Get(key)
}

func TestMissingRoot(t *testing.T) {
	db, _ := ethdb.NewMemDatabase()
	trie, err := New(common.HexToHash("0beec7b5ea3f0fdbc95d0dd47f3c5bc275da8a33"), db)
	if trie != nil {
		t.Error("New returned non-nil trie for invalid root")
	}
	if _, ok := err.(*MissingNodeError); !ok {
		t.Errorf("New returned wrong error: %v", err)
	}
}

func TestMissingNode(t *testing.T) {
	db, _ := ethdb.NewMemDatabase()
	trie, _ := New(common.Hash{}, db)
	updateString(trie, "120000", "qwerqwerqwerqwerqwerqwerqwerqwer")
	updateString(trie, "123456", "asdfasdfasdfasdfasdfasdfasdfasdf")
	root, _ := trie.Commit()

	trie, _ = New(root, db)
	_, err := trie.TryGet([]byte("120000"))
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	trie, _ = New(root, db)
	_, err = trie.TryGet([]byte("120099"))
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	trie, _ = New(root, db)
	_, err = trie.TryGet([]byte("123456"))
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	trie, _ = New(root, db)
	err = trie.TryUpdate([]byte("120099"), []byte("zxcvzxcvzxcvzxcvzxcvzxcvzxcvzxcv"))
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	trie, _ = New(root, db)
	err = trie.TryDelete([]byte("123456"))
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	db.Delete(common.FromHex("e1d943cc8f061a0c0b98162830b970395ac9315654824bf21b73b891365262f9"))

	trie, _ = New(root, db)
	_, err = trie.TryGet([]byte("120000"))
	if _, ok := err.(*MissingNodeError); !ok {
		t.Errorf("Wrong error: %v", err)
	}

	trie, _ = New(root, db)
	_, err = trie.TryGet([]byte("120099"))
	if _, ok := err.(*MissingNodeError); !ok {
		t.Errorf("Wrong error: %v", err)
	}

	trie, _ = New(root, db)
	_, err = trie.TryGet([]byte("123456"))
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	trie, _ = New(root, db)
	err = trie.TryUpdate([]byte("120099"), []byte("zxcv"))
	if _, ok := err.(*MissingNodeError); !ok {
		t.Errorf("Wrong error: %v", err)
	}

	trie, _ = New(root, db)
	err = trie.TryDelete([]byte("123456"))
	if _, ok := err.(*MissingNodeError); !ok {
		t.Errorf("Wrong error: %v", err)
	}
}

func TestInsert(t *testing.T) {
	trie := newEmpty()

	updateString(trie, "doe", "reindeer")
	updateString(trie, "dog", "puppy")
	updateString(trie, "dogglesworth", "cat")

	exp := common.HexToHash("8aad789dff2f538bca5d8ea56e8abe10f4c7ba3a5dea95fea4cd6e7c3a1168d3")
	root := trie.Hash()
	if root != exp {
		t.Errorf("exp %x got %x", exp, root)
	}

	trie = newEmpty()
	updateString(trie, "A", "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa")

	exp = common.HexToHash("d23786fb4a010da3ce639d66d5e904a11dbc02746d1ce25029e53290cabf28ab")
	root, err := trie.Commit()
	if err != nil {
		t.Fatalf("commit error: %v", err)
	}
	if root != exp {
		t.Errorf("exp %x got %x", exp, root)
	}
}

func TestGet(t *testing.T) {
	trie := newEmpty()
	updateString(trie, "doe", "reindeer")
	updateString(trie, "dog", "puppy")
	updateString(trie, "dogglesworth", "cat")

	for i := 0; i < 2; i++ {
		res := getString(trie, "dog")
		if !bytes.Equal(res, []byte("puppy")) {
			t.Errorf("expected puppy got %x", res)
		}

		unknown := getString(trie, "unknown")
		if unknown != nil {
			t.Errorf("expected nil got %x", unknown)
		}

		if i == 1 {
			return
		}
		trie.Commit()
	}
}

func TestDelete(t *testing.T) {
	trie := newEmpty()
	vals := []struct{ k, v string }{
		{"do", "verb"},
		{"ether", "wookiedoo"},
		{"horse", "stallion"},
		{"shaman", "horse"},
		{"doge", "coin"},
		{"ether", ""},
		{"dog", "puppy"},
		{"shaman", ""},
	}
	for _, val := range vals {
		if val.v != "" {
			updateString(trie, val.k, val.v)
		} else {
			deleteString(trie, val.k)
		}
	}

	hash := trie.Hash()
	exp := common.HexToHash("5991bb8c6514148a29db676a14ac506cd2cd5775ace63c30a4fe457715e9ac84")
	if hash != exp {
		t.Errorf("expected %x got %x", exp, hash)
	}
}

func TestEmptyValues(t *testing.T) {
	trie := newEmpty()

	vals := []struct{ k, v string }{
		{"do", "verb"},
		{"ether", "wookiedoo"},
		{"horse", "stallion"},
		{"shaman", "horse"},
		{"doge", "coin"},
		{"ether", ""},
		{"dog", "puppy"},
		{"shaman", ""},
	}
	for _, val := range vals {
		updateString(trie, val.k, val.v)
	}

	hash := trie.Hash()
	exp := common.HexToHash("5991bb8c6514148a29db676a14ac506cd2cd5775ace63c30a4fe457715e9ac84")
	if hash != exp {
		t.Errorf("expected %x got %x", exp, hash)
	}
}

func TestReplication(t *testing.T) {
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
	for _, val := range vals {
		updateString(trie, val.k, val.v)
	}
	exp, err := trie.Commit()
	if err != nil {
		t.Fatalf("commit error: %v", err)
	}

	// create a new trie on top of the database and check that lookups work.
	trie2, err := New(exp, trie.db)
	if err != nil {
		t.Fatalf("can't recreate trie at %x: %v", exp, err)
	}
	for _, kv := range vals {
		if string(getString(trie2, kv.k)) != kv.v {
			t.Errorf("trie2 doesn't have %q => %q", kv.k, kv.v)
		}
	}
	hash, err := trie2.Commit()
	if err != nil {
		t.Fatalf("commit error: %v", err)
	}
	if hash != exp {
		t.Errorf("root failure. expected %x got %x", exp, hash)
	}

	// perform some insertions on the new trie.
	vals2 := []struct{ k, v string }{
		{"do", "verb"},
		{"ether", "wookiedoo"},
		{"horse", "stallion"},
		// {"shaman", "horse"},
		// {"doge", "coin"},
		// {"ether", ""},
		// {"dog", "puppy"},
		// {"somethingveryoddindeedthis is", "myothernodedata"},
		// {"shaman", ""},
	}
	for _, val := range vals2 {
		updateString(trie2, val.k, val.v)
	}
	if hash := trie2.Hash(); hash != exp {
		t.Errorf("root failure. expected %x got %x", exp, hash)
	}
}

// Not an actual test
func TestOutput(t *testing.T) {
	t.Skip()

	base := "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	trie := newEmpty()
	for i := 0; i < 50; i++ {
		updateString(trie, fmt.Sprintf("%s%d", base, i), "valueeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee")
	}
	fmt.Println("############################## FULL ################################")
	fmt.Println(trie.root)

	trie.Commit()
	fmt.Println("############################## SMALL ################################")
	trie2, _ := New(trie.Hash(), trie.db)
	getString(trie2, base+"20")
	fmt.Println(trie2.root)
}

func TestLargeValue(t *testing.T) {
	trie := newEmpty()
	trie.Update([]byte("key1"), []byte{99, 99, 99, 99})
	trie.Update([]byte("key2"), bytes.Repeat([]byte{1}, 32))
	trie.Hash()
}

type randTestStep struct {
	op    int
	key   []byte // for opUpdate, opDelete, opGet
	value []byte // for opUpdate
}

type randTest []randTestStep

const (
	opUpdate = iota
	opDelete
	opGet
	opCommit
	opHash
	opReset
	opItercheckhash
	opMax // boundary value, not an actual op
)

func (randTest) Generate(r *rand.Rand, size int) reflect.Value {
	var allKeys [][]byte
	genKey := func() []byte {
		if len(allKeys) < 2 || r.Intn(100) < 10 {
			// new key
			key := make([]byte, r.Intn(50))
			randRead(r, key)
			allKeys = append(allKeys, key)
			return key
		}
		// use existing key
		return allKeys[r.Intn(len(allKeys))]
	}

	var steps randTest
	for i := 0; i < size; i++ {
		step := randTestStep{op: r.Intn(opMax)}
		switch step.op {
		case opUpdate:
			step.key = genKey()
			step.value = make([]byte, 8)
			binary.BigEndian.PutUint64(step.value, uint64(i))
		case opGet, opDelete:
			step.key = genKey()
		}
		steps = append(steps, step)
	}
	return reflect.ValueOf(steps)
}

// rand.Rand provides a Read method in Go 1.7 and later, but
// we can't use it yet.
func randRead(r *rand.Rand, b []byte) {
	pos := 0
	val := 0
	for n := 0; n < len(b); n++ {
		if pos == 0 {
			val = r.Int()
			pos = 7
		}
		b[n] = byte(val)
		val >>= 8
		pos--
	}
}

func runRandTest(rt randTest) bool {
	db, _ := ethdb.NewMemDatabase()
	tr, _ := New(common.Hash{}, db)
	values := make(map[string]string) // tracks content of the trie

	for _, step := range rt {
		switch step.op {
		case opUpdate:
			tr.Update(step.key, step.value)
			values[string(step.key)] = string(step.value)
		case opDelete:
			tr.Delete(step.key)
			delete(values, string(step.key))
		case opGet:
			v := tr.Get(step.key)
			want := values[string(step.key)]
			if string(v) != want {
				fmt.Printf("mismatch for key 0x%x, got 0x%x want 0x%x", step.key, v, want)
				return false
			}
		case opCommit:
			if _, err := tr.Commit(); err != nil {
				panic(err)
			}
		case opHash:
			tr.Hash()
		case opReset:
			hash, err := tr.Commit()
			if err != nil {
				panic(err)
			}
			newtr, err := New(hash, db)
			if err != nil {
				panic(err)
			}
			tr = newtr
		case opItercheckhash:
			checktr, _ := New(common.Hash{}, nil)
			it := tr.Iterator()
			for it.Next() {
				checktr.Update(it.Key, it.Value)
			}
			if tr.Hash() != checktr.Hash() {
				fmt.Println("hashes not equal")
				return false
			}
		}
	}
	return true
}

func TestRandom(t *testing.T) {
	if err := quick.Check(runRandTest, nil); err != nil {
		t.Fatal(err)
	}
}

func BenchmarkGet(b *testing.B)      { benchGet(b, false) }
func BenchmarkGetDB(b *testing.B)    { benchGet(b, true) }
func BenchmarkUpdateBE(b *testing.B) { benchUpdate(b, binary.BigEndian) }
func BenchmarkUpdateLE(b *testing.B) { benchUpdate(b, binary.LittleEndian) }
func BenchmarkHashBE(b *testing.B)   { benchHash(b, binary.BigEndian) }
func BenchmarkHashLE(b *testing.B)   { benchHash(b, binary.LittleEndian) }

const benchElemCount = 20000

func benchGet(b *testing.B, commit bool) {
	trie := new(Trie)
	if commit {
		_, tmpdb := tempDB()
		trie, _ = New(common.Hash{}, tmpdb)
	}
	k := make([]byte, 32)
	for i := 0; i < benchElemCount; i++ {
		binary.LittleEndian.PutUint64(k, uint64(i))
		trie.Update(k, k)
	}
	binary.LittleEndian.PutUint64(k, benchElemCount/2)
	if commit {
		trie.Commit()
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		trie.Get(k)
	}
	b.StopTimer()

	if commit {
		ldb := trie.db.(*ethdb.LDBDatabase)
		ldb.Close()
		os.RemoveAll(ldb.Path())
	}
}

func benchUpdate(b *testing.B, e binary.ByteOrder) *Trie {
	trie := newEmpty()
	k := make([]byte, 32)
	for i := 0; i < b.N; i++ {
		e.PutUint64(k, uint64(i))
		trie.Update(k, k)
	}
	return trie
}

func benchHash(b *testing.B, e binary.ByteOrder) {
	trie := newEmpty()
	k := make([]byte, 32)
	for i := 0; i < benchElemCount; i++ {
		e.PutUint64(k, uint64(i))
		trie.Update(k, k)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		trie.Hash()
	}
}

func tempDB() (string, Database) {
	dir, err := ioutil.TempDir("", "trie-bench")
	if err != nil {
		panic(fmt.Sprintf("can't create temporary directory: %v", err))
	}
	db, err := ethdb.NewLDBDatabase(dir, 256, 0)
	if err != nil {
		panic(fmt.Sprintf("can't create temporary database: %v", err))
	}
	return dir, db
}

func getString(trie *Trie, k string) []byte {
	return trie.Get([]byte(k))
}

func updateString(trie *Trie, k, v string) {
	trie.Update([]byte(k), []byte(v))
}

func deleteString(trie *Trie, k string) {
	trie.Delete([]byte(k))
}
