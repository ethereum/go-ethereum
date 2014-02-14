package ethutil

import (
	_ "encoding/hex"
	_ "fmt"
	"testing"
)

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
func (db *MemDatabase) Print()              {}
func (db *MemDatabase) Close()              {}
func (db *MemDatabase) LastKnownTD() []byte { return nil }

func TestTrieSync(t *testing.T) {
	db, _ := NewMemDatabase()
	trie := NewTrie(db, "")

	trie.Update("dog", "kindofalongsentencewhichshouldbeencodedinitsentirety")
	if len(db.db) != 0 {
		t.Error("Expected no data in database")
	}

	trie.Sync()
	if len(db.db) == 0 {
		t.Error("Expected data to be persisted")
	}
}
