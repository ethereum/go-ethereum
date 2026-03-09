// Package nomtdb implements the triedb backend for NOMT (Near-Optimal Merkle
// Trie), a page-based binary merkle trie engine.
//
// NOMT handles only the trie structure (merkle pages). Flat key-value storage
// (accounts, storage slots) is stored in geth's existing ethdb (PebbleDB)
// under NOMT-specific key prefixes.
package nomtdb

// Config holds configuration for the NOMT triedb backend.
type Config struct {
	// NumWorkers is the number of parallel goroutines for trie updates.
	// Defaults to runtime.NumCPU() if zero.
	NumWorkers int
}
