package ethutil

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"reflect"
	"testing"
)

const LONG_WORD = "1234567890abcdefghijklmnopqrstuvwxxzABCEFGHIJKLMNOPQRSTUVWXYZ"

type MemDatabase struct {
	db map[string][]byte
}

func NewMemDatabase() (*MemDatabase, error) {
	db := &MemDatabase{db: make(map[string][]byte)}
	return db, nil
}
func (db *MemDatabase) Put(key []byte, value []byte) {
	db.db[string(key)] = value
}
func (db *MemDatabase) Get(key []byte) ([]byte, error) {
	return db.db[string(key)], nil
}
func (db *MemDatabase) Delete(key []byte) error {
	delete(db.db, string(key))
	return nil
}
func (db *MemDatabase) Print()              {}
func (db *MemDatabase) Close()              {}
func (db *MemDatabase) LastKnownTD() []byte { return nil }

func New() (*MemDatabase, *Trie) {
	db, _ := NewMemDatabase()
	return db, NewTrie(db, "")
}

func TestTrieSync(t *testing.T) {
	db, trie := New()

	trie.Update("dog", LONG_WORD)
	if len(db.db) != 0 {
		t.Error("Expected no data in database")
	}

	trie.Sync()
	if len(db.db) == 0 {
		t.Error("Expected data to be persisted")
	}
}

func TestTrieDirtyTracking(t *testing.T) {
	_, trie := New()
	trie.Update("dog", LONG_WORD)
	if !trie.cache.IsDirty {
		t.Error("Expected trie to be dirty")
	}

	trie.Sync()
	if trie.cache.IsDirty {
		t.Error("Expected trie not to be dirty")
	}

	trie.Update("test", LONG_WORD)
	trie.cache.Undo()
	if trie.cache.IsDirty {
		t.Error("Expected trie not to be dirty")
	}

}

func TestTrieReset(t *testing.T) {
	_, trie := New()

	trie.Update("cat", LONG_WORD)
	if len(trie.cache.nodes) == 0 {
		t.Error("Expected cached nodes")
	}

	trie.cache.Undo()

	if len(trie.cache.nodes) != 0 {
		t.Error("Expected no nodes after undo")
	}
}

func TestTrieGet(t *testing.T) {
	_, trie := New()

	trie.Update("cat", LONG_WORD)
	x := trie.Get("cat")
	if x != LONG_WORD {
		t.Error("expected %s, got %s", LONG_WORD, x)
	}
}

func TestTrieUpdating(t *testing.T) {
	_, trie := New()
	trie.Update("cat", LONG_WORD)
	trie.Update("cat", LONG_WORD+"1")
	x := trie.Get("cat")
	if x != LONG_WORD+"1" {
		t.Error("expected %S, got %s", LONG_WORD+"1", x)
	}
}

func TestTrieCmp(t *testing.T) {
	_, trie1 := New()
	_, trie2 := New()

	trie1.Update("doge", LONG_WORD)
	trie2.Update("doge", LONG_WORD)
	if !trie1.Cmp(trie2) {
		t.Error("Expected tries to be equal")
	}

	trie1.Update("dog", LONG_WORD)
	trie2.Update("cat", LONG_WORD)
	if trie1.Cmp(trie2) {
		t.Errorf("Expected tries not to be equal %x %x", trie1.Root, trie2.Root)
	}
}

func TestTrieDelete(t *testing.T) {
	_, trie := New()
	trie.Update("cat", LONG_WORD)
	exp := trie.Root
	trie.Update("dog", LONG_WORD)
	trie.Delete("dog")
	if !reflect.DeepEqual(exp, trie.Root) {
		t.Errorf("Expected tries to be equal %x : %x", exp, trie.Root)
	}

	trie.Update("dog", LONG_WORD)
	exp = trie.Root
	trie.Update("dude", LONG_WORD)
	trie.Delete("dude")
	if !reflect.DeepEqual(exp, trie.Root) {
		t.Errorf("Expected tries to be equal %x : %x", exp, trie.Root)
	}
}

func TestTrieDeleteWithValue(t *testing.T) {
	_, trie := New()
	trie.Update("c", LONG_WORD)
	exp := trie.Root
	trie.Update("ca", LONG_WORD)
	trie.Update("cat", LONG_WORD)
	trie.Delete("ca")
	trie.Delete("cat")
	if !reflect.DeepEqual(exp, trie.Root) {
		t.Errorf("Expected tries to be equal %x : %x", exp, trie.Root)
	}

}

func TestTriePurge(t *testing.T) {
	_, trie := New()
	trie.Update("c", LONG_WORD)
	trie.Update("ca", LONG_WORD)
	trie.Update("cat", LONG_WORD)

	lenBefore := len(trie.cache.nodes)
	it := trie.NewIterator()
	if num := it.Purge(); num != 3 {
		t.Errorf("Expected purge to return 3, got %d", num)
	}

	if lenBefore == len(trie.cache.nodes) {
		t.Errorf("Expected cached nodes to be deleted")
	}
}

func h(str string) string {
	d, err := hex.DecodeString(str)
	if err != nil {
		panic(err)
	}

	return string(d)
}

func get(in string) (out string) {
	if len(in) > 2 && in[:2] == "0x" {
		out = h(in[2:])
	} else {
		out = in
	}

	return
}

type Test struct {
	Name string
	In   map[string]string
	Root string
}

func CreateTest(name string, data []byte) (Test, error) {
	t := Test{Name: name}
	err := json.Unmarshal(data, &t)
	if err != nil {
		return Test{}, fmt.Errorf("%v", err)
	}

	return t, nil
}

func CreateTests(uri string, cb func(Test)) {
	resp, err := http.Get(uri)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()

	data, err := ioutil.ReadAll(resp.Body)

	var objmap map[string]*json.RawMessage
	err = json.Unmarshal(data, &objmap)
	if err != nil {
		panic(err)
	}

	for name, testData := range objmap {
		test, err := CreateTest(name, *testData)
		if err != nil {
			panic(err)
		}

		cb(test)
	}
}

func TestRemote(t *testing.T) {
	CreateTests("https://raw.githubusercontent.com/ethereum/tests/develop/trietest.json", func(test Test) {
		_, trie := New()
		for key, value := range test.In {
			trie.Update(get(key), get(value))
		}

		a := NewValue(h(test.Root)).Bytes()
		b := NewValue(trie.Root).Bytes()
		if bytes.Compare(a, b) != 0 {
			t.Errorf("%-10s: %x %x", test.Name, a, b)
		}
	})
}

func TestTrieReplay(t *testing.T) {
	CreateTests("https://raw.githubusercontent.com/ethereum/tests/develop/trietest.json", func(test Test) {
		_, trie := New()
		for key, value := range test.In {
			trie.Update(get(key), get(value))
		}

		_, trie2 := New()
		trie.NewIterator().Each(func(key string, v *Value) {
			trie2.Update(key, v.Str())
		})

		a := NewValue(trie.Root).Bytes()
		b := NewValue(trie2.Root).Bytes()
		if bytes.Compare(a, b) != 0 {
			t.Errorf("%s %x %x\n", test.Name, trie.Root, trie2.Root)
		}
	})
}

func TestIt(t *testing.T) {
	_, trie := New()

	test := map[string]string{
		"0x000000000000000000000000ec4f34c97e43fbb2816cfd95e388353c7181dab1": "0x4e616d6552656700000000000000000000000000000000000000000000000000",
		"0x0000000000000000000000000000000000000000000000000000000000000045": "0x22b224a1420a802ab51d326e29fa98e34c4f24ea",
		"0x0000000000000000000000000000000000000000000000000000000000000046": "0x67706c2076330000000000000000000000000000000000000000000000000000",
		"0x000000000000000000000000697c7b8c961b56f675d570498424ac8de1a918f6": "0x6f6f6f6820736f2067726561742c207265616c6c6c793f000000000000000000",
		"0x0000000000000000000000007ef9e639e2733cb34e4dfc576d4b23f72db776b2": "0x4655474156000000000000000000000000000000000000000000000000000000",
		"0x6f6f6f6820736f2067726561742c207265616c6c6c793f000000000000000000": "0x697c7b8c961b56f675d570498424ac8de1a918f6",
		"0x4655474156000000000000000000000000000000000000000000000000000000": "0x7ef9e639e2733cb34e4dfc576d4b23f72db776b2",
		"0x4e616d6552656700000000000000000000000000000000000000000000000000": "0xec4f34c97e43fbb2816cfd95e388353c7181dab1",
	}

	for k, v := range test {
		trie.Update(k, v)
	}
	fmt.Printf("root : %x\n", trie.Root)
}
