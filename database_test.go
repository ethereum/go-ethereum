package main

import (
  "testing"
  _"fmt"
)

func TestTriePut(t *testing.T) {
  db, err := NewDatabase()
  defer db.Close()

  if err != nil {
    t.Error("Error starting db")
  }

  key := db.trie.Put([]byte("testing node"))

  data, err := db.db.Get(key, nil)
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
  db, err := NewDatabase()
  defer db.Close()

  if err != nil {
    t.Error("Error starting db")
  }

  db.trie.Update("test", "test")
}

