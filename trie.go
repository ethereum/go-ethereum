package main

// Database interface
type Database interface {
  Put(key []byte, value []byte)
  Get(key []byte) ([]byte, error)
}

type Trie struct {
  root       string
  db         Database
}

func NewTrie(db Database) *Trie {
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
func (t *Trie) UpdateState(node string, key []int, value string) string {
  if value != "" {
    return t.InsertState(node, ""/*key*/, value)
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
