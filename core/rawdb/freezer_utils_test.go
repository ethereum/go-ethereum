package rawdb

import (
	"bytes"
	mrand "math/rand"
	"os"
	"path/filepath"
	"testing"
)

func TestCopyFrom(t *testing.T) {
	for i := 0; i < 20; i++ {
		testCopyFrom(t)
	}
}

func testCopyFrom(t *testing.T) {
	data := make([]byte, 1024*33)
	mrand.Read(data)
	src := filepath.Join(os.TempDir(), "tmp-source")
	dst := filepath.Join(os.TempDir(), "tmp-dest")
	os.WriteFile(src, data, 0600)
	offset := uint64(mrand.Intn(len(data)))
	if err := copyFrom(src, dst, offset); err != nil {
		t.Fatal(err)
	}
	// Now validate that the contents match
	haveData, err := os.ReadFile(dst)
	if err != nil {
		t.Fatal(err)
	}
	if have, want := len(haveData), len(data[offset:]); have != want {
		t.Fatalf("wrong data, have length %d, want length %d", have, want)
	}
	if !bytes.Equal(haveData, data[offset:]) {
		t.Fatalf("data mismatch\nhave:\n%x\nwant\n%x", haveData, data[offset:])
	}
}
