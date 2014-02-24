package ethutil

import (
	"fmt"
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

func TestTrieIterator(t *testing.T) {
	_, trie := New()
	trie.Update("c", LONG_WORD)
	trie.Update("ca", LONG_WORD)
	trie.Update("cat", LONG_WORD)

	it := trie.NewIterator()
	fmt.Println("purging")
	fmt.Println("len =", it.Purge())
	/*
		for it.Next() {
			k := it.Key()
			v := it.Value()

			fmt.Println(k, v)
		}
	*/
}
