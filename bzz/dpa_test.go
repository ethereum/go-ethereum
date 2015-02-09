package bzz

import (
	"bytes"
	"github.com/ethereum/go-ethereum/bzz/test"
	"os"
	"testing"
	// "time"
)

const testDataSize = 0x1000000

func TestDPA(t *testing.T) {
	test.LogInit()
	os.RemoveAll("/tmp/bzz")
	dbStore, err := newDbStore("/tmp/bzz")
	if err != nil {
		t.Errorf("DB error: %v", err)
	}
	memStore := newMemStore(dbStore)
	localStore := &localStore{
		memStore,
		dbStore,
	}
	chunker := &TreeChunker{}
	chunker.Init()
	dpa := &DPA{
		Chunker:    chunker,
		ChunkStore: localStore,
	}
	dpa.Start()
	reader, slice := testDataReader(testDataSize)
	key, err := dpa.Store(reader)
	if err != nil {
		t.Errorf("Store error: %v", err)
	}
	resultReader := dpa.Retrieve(key)
	resultSlice := make([]byte, len(slice))
	n, err := resultReader.ReadAt(resultSlice, 0)
	if err != nil {
		t.Errorf("Retrieve error: %v", err)
	}
	if n != len(slice) {
		t.Errorf("Slice size error got %d, expected %d.", n, len(slice))
	}
	if !bytes.Equal(slice, resultSlice) {
		t.Errorf("Comparison error.")
	}
	localStore.memStore = newMemStore(dbStore)
	resultReader = dpa.Retrieve(key)
	for i, _ := range resultSlice {
		resultSlice[i] = 0
	}
	n, err = resultReader.ReadAt(resultSlice, 0)
	if err != nil {
		t.Errorf("Retrieve error after removing memStore: %v", err)
	}
	if n != len(slice) {
		t.Errorf("Slice size error after removing memStore got %d, expected %d.", n, len(slice))
	}
	if !bytes.Equal(slice, resultSlice) {
		t.Errorf("Comparison error after removing memStore.")
	}
}

func TestDPA_capacity(t *testing.T) {
	test.LogInit()
	os.RemoveAll("/tmp/bzz")
	dbStore, err := newDbStore("/tmp/bzz")
	if err != nil {
		t.Errorf("DB error: %v", err)
	}
	memStore := newMemStore(dbStore)
	localStore := &localStore{
		memStore,
		dbStore,
	}
	localStore.memStore.setCapacity(0)
	chunker := &TreeChunker{}
	chunker.Init()
	dpa := &DPA{
		Chunker:    chunker,
		ChunkStore: localStore,
	}
	dpa.Start()
	reader, slice := testDataReader(testDataSize)
	key, err := dpa.Store(reader)
	if err != nil {
		t.Errorf("Store error: %v", err)
	}
	resultReader := dpa.Retrieve(key)
	resultSlice := make([]byte, len(slice))
	n, err := resultReader.ReadAt(resultSlice, 0)
	if err != nil {
		t.Errorf("Retrieve error: %v", err)
	}
	if n != len(slice) {
		t.Errorf("Slice size error got %d, expected %d.", n, len(slice))
	}
	if !bytes.Equal(slice, resultSlice) {
		t.Errorf("Comparison error.")
	}
	localStore.memStore.setCapacity(0)
	//	localStore.dbStore.setCapacity(0)
	resultReader = dpa.Retrieve(key)
	for i, _ := range resultSlice {
		resultSlice[i] = 0
	}
	n, err = resultReader.ReadAt(resultSlice, 0)
	if err != nil {
		t.Errorf("Retrieve error after removing memStore: %v", err)
	}
	if n != len(slice) {
		t.Errorf("Slice size error after removing memStore got %d, expected %d.", n, len(slice))
	}
	if !bytes.Equal(slice, resultSlice) {
		t.Errorf("Comparison error after removing memStore.")
	}
}
