package trie

import (
	"sort"
	"testing"

	"github.com/ethereum/go-ethereum/ethdb/memorydb"
)

type keyValue struct {
	k []byte
	v []byte
}

func TestStRangeProofLeftside(t *testing.T) {
	trie, vals := randomTrie(4096)
	var entries entrySlice
	for _, kv := range vals {
		entries = append(entries, kv)
	}
	sort.Sort(entries)
	{
		start := 10
		end := len(vals)
		proof := memorydb.New()
		if err := trie.Prove(entries[start].k, 0, proof); err != nil {
			t.Fatalf("Failed to prove the first node %v", err)
		}
		if err := trie.Prove(entries[end-1].k, 0, proof); err != nil {
			t.Fatalf("Failed to prove the last node %v", err)
		}
		stRoot, _, err := initLeftside(trie.Hash(), nil, entries[start].k, proof, false)
		if err != nil {
			t.Fatal(err)
		}
		for i := start + 1; i < end; i++ {
			stRoot.Update(entries[i].k, entries[i].v)
		}
		if got, want := stRoot.Hash(), trie.Hash(); got != want {
			t.Fatalf("got %x want %x\n", stRoot.Hash(), trie.Hash())
		}
	}
}

func TestStRangeProofRightSide(t *testing.T) {
	trie, vals := randomTrie(4096)
	var entries entrySlice
	for _, kv := range vals {
		entries = append(entries, kv)
	}
	sort.Sort(entries)
	{
		start := 0
		end := 50
		proof := memorydb.New()
		if err := trie.Prove(entries[start].k, 0, proof); err != nil {
			t.Fatalf("Failed to prove the first node %v", err)
		}
		if err := trie.Prove(entries[end-1].k, 0, proof); err != nil {
			t.Fatalf("Failed to prove the last node %v", err)
		}
		stRoot := NewStackTrie(nil)
		for i := 0; i < end; i++ {
			stRoot.Update(entries[i].k, entries[i].v)
		}
		var err error
		stRoot, _, err = finalizeRightSide(trie.Hash(), stRoot, entries[end-1].k, proof, false)
		if err != nil {
			t.Fatal(err)
		}
		if got, want := stRoot.Hash(), trie.Hash(); got != want {
			t.Fatalf("got %x want %x\n", stRoot.Hash(), trie.Hash())
		}
	}
}
