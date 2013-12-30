package main

import (
  "path"
  "os/user"
  "github.com/syndtr/goleveldb/leveldb"
  "fmt"
)

type LDBDatabase struct {
  db        *leveldb.DB
  trie      *Trie
}

func NewLDBDatabase() (*LDBDatabase, error) {
  // This will eventually have to be something like a resource folder.
  // it works on my system for now. Probably won't work on Windows
  usr, _ := user.Current()
  dbPath := path.Join(usr.HomeDir, ".ethereum", "database")

  // Open the db
  db, err := leveldb.OpenFile(dbPath, nil)
  if err != nil {
    return nil, err
  }

  database := &LDBDatabase{db: db}

  // Bootstrap database. Sets a few defaults; such as the last block
  database.Bootstrap()

  return database, nil
}

func (db *LDBDatabase) Bootstrap() error {
  db.trie = NewTrie(db)

  return nil
}

func (db *LDBDatabase) Put(key []byte, value []byte) {
  err := db.db.Put(key, value, nil)
  if err != nil {
    fmt.Println("Error put", err)
  }
}

func (db *LDBDatabase) Get(key []byte) ([]byte, error) {
  return nil, nil
}

func (db *LDBDatabase) Close() {
  // Close the leveldb database
  db.db.Close()
}

