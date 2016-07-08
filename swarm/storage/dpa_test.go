package storage

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
	dbStore, err := NewDbStore("/tmp/bzz", MakeHashFunc(defaultHash), defaultDbCapacity, defaultRadius)
	dbStore.setCapacity(50000)
	if err != nil {
		t.Errorf("DB error: %v", err)
	}
	memStore := NewMemStore(dbStore, defaultCacheCapacity)
	localStore := &LocalStore{
		memStore,
		dbStore,
	}
	chunker := NewTreeChunker(NewChunkerParams())
	dpa := &DPA{
		Chunker:    chunker,
		ChunkStore: localStore,
	}
	dpa.Start()
	reader, slice := testDataReaderAndSlice(testDataSize)
	wg := &sync.WaitGroup{}
	key, err := dpa.Store(reader, testDataSize, wg)
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
	localStore.memStore = NewMemStore(dbStore, defaultCacheCapacity)
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
	dbStore, err := NewDbStore("/tmp/bzz", MakeHashFunc(defaultHash), defaultDbCapacity, defaultRadius)
	if err != nil {
		t.Errorf("DB error: %v", err)
	}
	memStore := NewMemStore(dbStore, defaultCacheCapacity)
	localStore := &LocalStore{
		memStore,
		dbStore,
	}
	memStore.setCapacity(0)
	chunker := NewTreeChunker(NewChunkerParams())
	dpa := &DPA{
		Chunker:    chunker,
		ChunkStore: localStore,
	}
	dpa.Start()
	reader, slice := testDataReaderAndSlice(testDataSize)
	wg := &sync.WaitGroup{}
	key, err := dpa.Store(reader, testDataSize, wg)
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
	memStore.setCapacity(0)
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
