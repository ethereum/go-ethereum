package bzz

import (
	"bytes"
	"fmt"
	"github.com/ethereum/go-ethereum/bzz/test"
	"os"
	"testing"
	"time"
)

func TestDPA(t *testing.T) {
	test.LogInit()
	os.RemoveAll("/tmp/bzz")
	dbStore, err := newDbStore("/tmp/bzz")
	if err != nil {
		t.Errorf("DB error: %v", err)
	}
	//	memStore := newMemStore(dbStore)
	//	localStore := &localStore{
	//		memStore,
	//		dbStore,
	//	}
	chunker := &TreeChunker{}
	chunker.Init()
	dpa := &DPA{
		Chunker:    chunker,
		ChunkStore: dbStore,
	}
	dpa.Start()
	reader, slice := testDataReader(0x100)
	fmt.Printf("Chunk size: %d.", len(slice))
	key, err := dpa.Store(reader)
	// _, err = dpa.Store(reader)
	if err != nil {
		t.Errorf("Store error: %v", err)
	}
	resultReader := dpa.Retrieve(key)
	resultSlice := make([]byte, len(slice))
	n, err := resultReader.Read(resultSlice)
	if err != nil {
		t.Errorf("Retrieve error: %v", err)
	}
	if n != len(slice) {
		t.Errorf("Slice size error got %d, expected %d.", n, len(slice))
	}
	if !bytes.Equal(slice, resultSlice) {
		t.Errorf("Comparison error.")
	}
	time.Sleep(time.Second)
}
