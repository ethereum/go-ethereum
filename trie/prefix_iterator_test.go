package trie

import (
	"bytes"
	"testing"

	"github.com/ethereum/go-ethereum/core/rawdb"
)

func TestPrefixIterator(t *testing.T) {
	// Create a new trie
	trie := NewEmpty(newTestDatabase(rawdb.NewMemoryDatabase(), rawdb.HashScheme))

	// Insert test data
	testData := map[string]string{
		"key1":      "value1",
		"key2":      "value2",
		"key10":     "value10",
		"key11":     "value11",
		"different": "value_different",
	}

	for key, value := range testData {
		trie.Update([]byte(key), []byte(value))
	}

	// Test prefix iteration for "key1" prefix
	prefix := []byte("key1")
	iter, err := trie.NodeIteratorWithPrefix(prefix)
	if err != nil {
		t.Fatalf("Failed to create prefix iterator: %v", err)
	}

	var foundKeys [][]byte
	for iter.Next(true) {
		if iter.Leaf() {
			foundKeys = append(foundKeys, iter.LeafKey())
		}
	}

	if err := iter.Error(); err != nil {
		t.Fatalf("Iterator error: %v", err)
	}

	// Verify only keys starting with "key1" were found
	expectedCount := 3 // "key1", "key10", "key11"
	if len(foundKeys) != expectedCount {
		t.Errorf("Expected %d keys, found %d", expectedCount, len(foundKeys))
	}

	for _, key := range foundKeys {
		keyStr := string(key)
		if !bytes.HasPrefix(key, prefix) {
			t.Errorf("Found key %s doesn't have prefix %s", keyStr, string(prefix))
		}
	}
}

func TestPrefixIteratorVsFullIterator(t *testing.T) {
	// Create a new trie with more structured data
	trie := NewEmpty(newTestDatabase(rawdb.NewMemoryDatabase(), rawdb.HashScheme))

	// Insert structured test data
	testData := map[string]string{
		"aaa": "value_aaa",
		"aab": "value_aab",
		"aba": "value_aba",
		"bbb": "value_bbb",
	}

	for key, value := range testData {
		trie.Update([]byte(key), []byte(value))
	}

	// Test that prefix iterator stops at boundary
	prefix := []byte("aa")
	prefixIter, err := trie.NodeIteratorWithPrefix(prefix)
	if err != nil {
		t.Fatalf("Failed to create prefix iterator: %v", err)
	}

	var prefixKeys [][]byte
	for prefixIter.Next(true) {
		if prefixIter.Leaf() {
			prefixKeys = append(prefixKeys, prefixIter.LeafKey())
		}
	}

	// Should only find "aaa" and "aab", not "aba" or "bbb"
	if len(prefixKeys) != 2 {
		t.Errorf("Expected 2 keys with prefix 'aa', found %d", len(prefixKeys))
	}

	// Verify no keys outside prefix were found
	for _, key := range prefixKeys {
		if !bytes.HasPrefix(key, prefix) {
			t.Errorf("Prefix iterator returned key %s outside prefix %s", string(key), string(prefix))
		}
	}
}

func TestEmptyPrefixIterator(t *testing.T) {
	// Test with empty trie
	trie := NewEmpty(newTestDatabase(rawdb.NewMemoryDatabase(), rawdb.HashScheme))

	iter, err := trie.NodeIteratorWithPrefix([]byte("nonexistent"))
	if err != nil {
		t.Fatalf("Failed to create iterator: %v", err)
	}

	if iter.Next(true) {
		t.Error("Expected no results from empty trie")
	}
}
