package bzz

import (
	"bytes"
	"io"
	"io/ioutil"
	"os"
	"sync"
	"testing"
)

const testDataSize = 0x1000000

func TestDPArandom(t *testing.T) {
	os.RemoveAll("/tmp/bzz")
	dbStore, err := newDbStore("/tmp/bzz")
	dbStore.setCapacity(50000)
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
	wg := &sync.WaitGroup{}
	key, err := dpa.Store(reader, wg)
	if err != nil {
		t.Errorf("Store error: %v", err)
	}
	wg.Wait()
	resultReader := dpa.Retrieve(key)
	resultSlice := make([]byte, len(slice))
	n, err := resultReader.ReadAt(resultSlice, 0)
	if err != io.EOF {
		t.Errorf("Retrieve error: %v", err)
	}
	if n != len(slice) {
		t.Errorf("Slice size error got %d, expected %d.", n, len(slice))
	}
	if !bytes.Equal(slice, resultSlice) {
		t.Errorf("Comparison error.")
	}
	ioutil.WriteFile("/tmp/slice.bzz.16M", slice, 0666)
	ioutil.WriteFile("/tmp/result.bzz.16M", resultSlice, 0666)
	localStore.memStore = newMemStore(dbStore)
	resultReader = dpa.Retrieve(key)
	for i, _ := range resultSlice {
		resultSlice[i] = 0
	}
	n, err = resultReader.ReadAt(resultSlice, 0)
	if err != io.EOF {
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
	wg := &sync.WaitGroup{}
	key, err := dpa.Store(reader, wg)
	if err != nil {
		t.Errorf("Store error: %v", err)
	}
	wg.Wait()
	resultReader := dpa.Retrieve(key)
	resultSlice := make([]byte, len(slice))
	n, err := resultReader.ReadAt(resultSlice, 0)
	if err != io.EOF {
		t.Errorf("Retrieve error: %v", err)
	}
	if n != len(slice) {
		t.Errorf("Slice size error got %d, expected %d.", n, len(slice))
	}
	if !bytes.Equal(slice, resultSlice) {
		t.Errorf("Comparison error.")
	}
	// Clear memStore
	localStore.memStore.setCapacity(0)
	// check whether it is, indeed, empty
	dpa.ChunkStore = memStore
	resultReader = dpa.Retrieve(key)
	n, err = resultReader.ReadAt(resultSlice, 0)
	if err == nil {
		t.Errorf("Was able to read %d bytes from an empty memStore.")
	}
	// check how it works with localStore
	dpa.ChunkStore = localStore
	//	localStore.dbStore.setCapacity(0)
	resultReader = dpa.Retrieve(key)
	for i, _ := range resultSlice {
		resultSlice[i] = 0
	}
	n, err = resultReader.ReadAt(resultSlice, 0)
	if err != io.EOF {
		t.Errorf("Retrieve error after clearing memStore: %v", err)
	}
	if n != len(slice) {
		t.Errorf("Slice size error after clearing memStore got %d, expected %d.", n, len(slice))
	}
	if !bytes.Equal(slice, resultSlice) {
		t.Errorf("Comparison error after clearing memStore.")
	}
}
