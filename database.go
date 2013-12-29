package main

import (
  "path"
  "os/user"
  "github.com/syndtr/goleveldb/leveldb"
  "fmt"
)

type Database struct {
  db        *leveldb.DB
  trie      *Trie
}

func NewDatabase() (*Database, error) {
  // This will eventually have to be something like a resource folder.
  // it works on my system for now. Probably won't work on Windows
  usr, _ := user.Current()
  dbPath := path.Join(usr.HomeDir, ".ethereum", "database")

  // Open the db
  db, err := leveldb.OpenFile(dbPath, nil)
  if err != nil {
    return nil, err
  }

  database := &Database{db: db}

  // Bootstrap database. Sets a few defaults; such as the last block
  database.Bootstrap()

  return database, nil
}

func (db *Database) Bootstrap() error {
  db.trie = NewTrie(db)

  return nil
}

func (db *Database) Put(key []byte, value []byte) {
  err := db.db.Put(key, value, nil)
  if err != nil {
    fmt.Println("Error put", err)
  }
}

func (db *Database) Close() {
  // Close the leveldb database
  db.db.Close()
}

type Trie struct {
  root       string
  db         *Database
}

func NewTrie(db *Database) *Trie {
  return &Trie{db: db, root: ""}
}

func (t *Trie) Update(key string, value string) {
  k := CompactHexDecode(key)

  t.root = t.UpdateState(t.root, k, value)
}

func (t *Trie) Get(key []byte) ([]byte, error) {
  return nil, nil
}

// Inserts a new sate or delete a state based on the value
func (t *Trie) UpdateState(node, key, value string) string {
  if value != "" {
    return t.InsertState(node, key, value)
  } else {
    // delete it
  }

  return ""
}

func (t *Trie) InsertState(node, key, value string) string {
  return ""
}

func (t *Trie) Put(node []byte) []byte {
  enc := Encode(node)
  sha := Sha256Bin(enc)

  t.db.Put([]byte(sha), enc)

  return sha
}
