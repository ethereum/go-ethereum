package utils

import (
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/rlp"
)

// TestExport does basic sanity checks on the export/import functionality
func TestExport(t *testing.T) {
	f := fmt.Sprintf("%v/tempdump", os.TempDir())
	defer func() {
		os.Remove(f)
	}()
	testExport(t, f)
}

func TestExportGzip(t *testing.T) {
	f := fmt.Sprintf("%v/tempdump.gz", os.TempDir())
	defer func() {
		os.Remove(f)
	}()
	testExport(t, f)
}

func testExport(t *testing.T, f string) {
	db := rawdb.NewMemoryDatabase()
	// Populate some keys
	for i := 0; i < 1000; i++ {
		db.Put([]byte(fmt.Sprintf("key-%04d", i)), []byte(fmt.Sprintf("value %d", i)))
	}
	checker := func(key []byte) bool {
		return string(key) != "key-0042"
	}
	err := ExportChaindata(db, f, "testdata", checker, [][]byte{[]byte("key")}, make(chan struct{}))
	if err != nil {
		t.Fatal(err)
	}
	db2 := rawdb.NewMemoryDatabase()
	err = ImportLDBData(db2, f, 5, make(chan struct{}))
	if err != nil {
		t.Fatal(err)
	}
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

// TestImportFutureFormat tests that we reject unsupported future versions.
func TestImportFutureFormat(t *testing.T) {
	f := fmt.Sprintf("%v/tempdump-future", os.TempDir())
	defer func() {
		os.Remove(f)
	}()
	fh, err := os.OpenFile(f, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, os.ModePerm)
	if err != nil {
		t.Fatal(err)
	}
	defer fh.Close()
	if err := rlp.Encode(fh, &exportHeader{
		Magic:    exportMagic,
		Version:  500,
		Kind:     "testdata",
		UnixTime: uint64(time.Now().Unix()),
	}); err != nil {
		t.Fatal(err)
	}
	db2 := rawdb.NewMemoryDatabase()
	err = ImportLDBData(db2, f, 0, make(chan struct{}))
	if err == nil {
		t.Fatal("Expected error, got none")
	}
	if !strings.HasPrefix(err.Error(), "incompatible version") {
		t.Fatalf("wrong error: %v", err)
	}
}
