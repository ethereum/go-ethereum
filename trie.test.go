package main

import (
  "testing"
)

type MemDatabase struct {
  db      map[string][]byte
  trie    *Trie
}

func NewMemDatabase() (*MemDatabase, error) {
  db := &MemDatabase{db: make(map[string][]byte)}

  db.trie = NewTrie(db)

  return db, nil
}

func (db *MemDatabase) Put(key []byte, value []byte) {
  db.db[string(key)] = value
}

func (db *MemDatabase) Get(key []byte) ([]byte, error) {
  return db.db[string(key)], nil
}

func TestTriePut(t *testing.T) {
  db, err := NewMemDatabase()

  if err != nil {
    t.Error("Error starting db")
  }

  key := db.trie.Put([]byte("testing node"))

  data, err := db.Get(key)
  if err != nil {
    t.Error("Nothing at node")
  }

  s, _ := Decode(data, 0)
  if str, ok := s.([]byte); ok {
    if string(str) != "testing node" {
      t.Error("Wrong value node", str)
    }
  } else {
    t.Error("Invalid return type")
  }
}

func TestTrieUpdate(t *testing.T) {
  db, err := NewMemDatabase()

  if err != nil {
    t.Error("Error starting db")
  }

  db.trie.Update("test", "test")
}

