package main

import (
  "fmt"
)

// Database interface
type Database interface {
  Put(key []byte, value []byte)
  Get(key []byte) ([]byte, error)
}

type Trie struct {
  root       string
  db         Database
}

func NewTrie(db Database, root string) *Trie {
  return &Trie{db: db, root: ""}
}

func (t *Trie) Put(node interface{}) []byte {
  //if s, ok := node.([]string); ok {
  //  PrintSlice(s)
  //}
  enc := Encode(node)
  sha := Sha256Bin(enc)

  t.db.Put([]byte(sha), enc)

  return sha
}

func (t *Trie) Update(key string, value string) {
  k := CompactHexDecode(key)

  t.root = t.UpdateState(t.root, k, value)
}

func (t *Trie) Get(key string) string {
  k := CompactHexDecode(key)

  return t.GetState(t.root, k)
}

// Returns the state of an object
func (t *Trie) GetState(node string, key []int) string {
  //if Debug { fmt.Println("get =", key) }

  // Return the node if key is empty (= found)
  if len(key) == 0 || node == "" {
    return node
  }

  // Fetch the encoded node from the db
  n, err := t.db.Get([]byte(node))
  if err != nil { fmt.Println("Error in GetState for node", node, "with key", key); return "" }

  // Decode it
  currentNode := DecodeNode(n)

  if len(currentNode) == 0 {
    return ""
  } else if len(currentNode) == 2 {
    // Decode the key
    k := CompactDecode(currentNode[0])
    v := currentNode[1]

    //fmt.Println(k, key)
    //fmt.Printf("k1:%v\nk2:%v\n", k, key[:len(k)-1])

    //fmt.Println(len(key), ">=", len(k)-1, "&&", k, key[:len(k)])
    if len(key) >= len(k) && CompareIntSlice(k, key[:len(k)]) {
      return t.GetState(v, key[len(k):])
    } else {
      return ""
    }
  } else if len(currentNode) == 17 {
    return t.GetState(currentNode[key[0]], key[1:])
  }

  // It shouldn't come this far
  fmt.Println("GetState unexpected return")
  return ""
}

// Inserts a new sate or delete a state based on the value
func (t *Trie) UpdateState(node string, key []int, value string) string {
  if value != "" {
    return t.InsertState(node, key, value)
  } else {
    // delete it
  }

  return ""
}

func DecodeNode(data []byte) []string {
  dec, _ := Decode(data, 0)
  if slice, ok := dec.([]interface{}); ok {
    strSlice := make([]string, len(slice))

    for i, s := range slice {
      if str, ok := s.([]byte); ok {
        strSlice[i] = string(str)
      }
    }

    return strSlice
  }

  return nil
}

func (t *Trie) PrintNode(n string) {
  data, _ := t.db.Get([]byte(n))
  d := DecodeNode(data)
  PrintSlice(d)
}

func PrintSlice(slice []string) {
  fmt.Printf("[")
  for i, val := range slice {
    fmt.Printf("%q", val)
    if i != len(slice)-1 { fmt.Printf(",") }
  }
  fmt.Printf("]\n")
}

func (t *Trie) InsertState(node string, key []int, value string) string {
  //if Debug { fmt.Println("insrt", key, value, "node:", node) }

  if len(key) == 0 {
    return value
  }

  //fmt.Println(node)
  // Root node!
  if node == "" {
    newNode := []string{ CompactEncode(key), value }

    return string(t.Put(newNode))
  }

  // Fetch the encoded node from the db
  n, err := t.db.Get([]byte(node))
  if err != nil { fmt.Println("Error InsertState", err); return "" }

  // Decode it
  currentNode := DecodeNode(n)
  // Check for "special" 2 slice type node
  if len(currentNode) == 2 {
    // Decode the key
    k := CompactDecode(currentNode[0])
    v := currentNode[1]

    // Matching key pair (ie. there's already an object with this key)
    if CompareIntSlice(k, key) {
      return string(t.Put([]string{ CompactEncode(key), value }))
    }

    var newHash string
    matchingLength := MatchingNibbleLength(key, k)
    if matchingLength == len(k) {
      // Insert the hash, creating a new node
      newHash = t.InsertState(v, key[matchingLength:], value)
    } else {
      // Expand the 2 length slice to a 17 length slice
      oldNode := t.InsertState("", k[matchingLength+1:], v)
      newNode := t.InsertState("", key[matchingLength+1:], value)
      // Create an expanded slice
      scaledSlice := make([]string, 17)
      // Set the copied and new node
      scaledSlice[k[matchingLength]] = oldNode
      scaledSlice[key[matchingLength]] = newNode

      newHash = string(t.Put(scaledSlice))
    }

    if matchingLength == 0 {
      // End of the chain, return
      return newHash
    } else {
      newNode := []string{ CompactEncode(key[:matchingLength]), newHash }
      return string(t.Put(newNode))
    }
  } else {
    // Copy the current node over to the new node and replace the first nibble in the key
    newNode := make([]string, 17); copy(newNode, currentNode)
    newNode[key[0]] = t.InsertState(currentNode[key[0]], key[1:], value)

    return string(t.Put(newNode))
  }

  return ""
}


