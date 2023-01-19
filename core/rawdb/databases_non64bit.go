//go:build !(arm64 || amd64 || x84_64)

package rawdb

import (
	"errors"

	"github.com/ethereum/go-ethereum/ethdb"
)

// Pebble is unsuported on 32bit architecture
const PebbleEnabled = false

// NewPebbleDBDatabase creates a persistent key-value database without a freezer
// moving immutable chain segments into cold storage.
func NewPebbleDBDatabase(file string, cache int, handles int, namespace string, readonly bool) (ethdb.Database, error) {
	return nil, errors.New("Pebble is not supported on this platform")
}
