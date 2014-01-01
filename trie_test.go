package main

import (
  "testing"
  "encoding/hex"
)

type MemDatabase struct {
  db      map[string][]byte
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

func TestTriePut(t *testing.T) {
  db, err := NewMemDatabase()
  trie := NewTrie(db, "")

  if err != nil {
    t.Error("Error starting db")
  }

  key := trie.Put([]byte("testing node"))

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
  trie := NewTrie(db, "")

  if err != nil {
    t.Error("Error starting db")
  }


  trie.Update("doe", "reindeer")
  trie.Update("dog", "puppy")
  trie.Update("dogglesworth", "cat")
  root := hex.EncodeToString([]byte(trie.root))
  req := "e378927bfc1bd4f01a2e8d9f59bd18db8a208bb493ac0b00f93ce51d4d2af76c" 
  if root != req {
    t.Error("trie.root do not match, expected", req, "got", root)
  }
}

