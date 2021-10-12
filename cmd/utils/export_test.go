package utils

import (
	"fmt"
	"testing"

	"github.com/ethereum/go-ethereum/core/rawdb"
)

// TestExport does basic sanity checks on the export/import functionality
func TestExport(t *testing.T) {
	db := rawdb.NewMemoryDatabase()
	// Populate some keys
	for i := 0; i < 1000; i++ {
		db.Put([]byte(fmt.Sprintf("key-%04d", i)), []byte(fmt.Sprintf("value %d", i)))
	}
	checker := func(key []byte) bool {
		return string(key) != "key-0042"
	}
	err := ExportChaindata(db, "temp-dump", "testdata", checker, [][]byte{[]byte("key")}, make(chan struct{}))
	if err != nil {
		t.Fatal(err)
	}
	db2 := rawdb.NewMemoryDatabase()
	ImportLDBData(db2, "temp-dump", 5, make(chan struct{}))
	// verify
	for i := 0; i < 1000; i++ {
		v, err := db2.Get([]byte(fmt.Sprintf("key-%04d", i)))
		if (i < 5 || i == 42) && err == nil {
			t.Fatalf("expected no element at idx %d, got '%v'", i, string(v))
		}
		if !(i < 5 || i == 42) {
			if err != nil {
				t.Fatalf("expected element idx %d: %v", i, err)
			}
			if have, want := string(v), fmt.Sprintf("value %d", i); have != want {
				t.Fatalf("have %v, want %v", have, want)
			}
		}
	}
}
