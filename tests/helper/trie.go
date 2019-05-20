package helper

import "github.com/ethereum/go-ethereum/trie"

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

func NewTrie() *trie.Trie {
	db, _ := NewMemDatabase()

	return trie.New(nil, db)
}
