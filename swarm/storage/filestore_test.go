// Copyright 2016 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

package storage

import (
	"bytes"
	"context"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/ethereum/go-ethereum/swarm/storage/localstore"
	"github.com/ethereum/go-ethereum/swarm/testutil"
)

const testDataSize = 0x0001000

func TestFileStorerandom(t *testing.T) {
	testFileStoreRandom(false, t)
	testFileStoreRandom(true, t)
}

func testFileStoreRandom(toEncrypt bool, t *testing.T) {
	dir, err := ioutil.TempDir("", "swarm-storage-")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)
	localStore, err := localstore.New(dir, make([]byte, 32), nil)
	if err != nil {
		t.Fatal(err)
	}
	defer localStore.Close()

	fileStore := NewFileStore(localStore, NewFileStoreParams())

	slice := testutil.RandomBytes(1, testDataSize)
	ctx := context.TODO()
	key, wait, err := fileStore.Store(ctx, bytes.NewReader(slice), testDataSize, toEncrypt)
	if err != nil {
		t.Fatalf("Store error: %v", err)
	}
	err = wait(ctx)
	if err != nil {
		t.Fatalf("Store waitt error: %v", err.Error())
	}
	resultReader, isEncrypted := fileStore.Retrieve(context.TODO(), key)
	if isEncrypted != toEncrypt {
		t.Fatalf("isEncrypted expected %v got %v", toEncrypt, isEncrypted)
	}
	resultSlice := make([]byte, testDataSize)
	n, err := resultReader.ReadAt(resultSlice, 0)
	if err != io.EOF {
		t.Fatalf("Retrieve error: %v", err)
	}
	if n != testDataSize {
		t.Fatalf("Slice size error got %d, expected %d.", n, testDataSize)
	}
	if !bytes.Equal(slice, resultSlice) {
		t.Fatalf("Comparison error.")
	}
	ioutil.WriteFile(filepath.Join(dir, "slice.bzz.16M"), slice, 0666)
	ioutil.WriteFile(filepath.Join(dir, "result.bzz.16M"), resultSlice, 0666)
	resultReader, isEncrypted = fileStore.Retrieve(context.TODO(), key)
	if isEncrypted != toEncrypt {
		t.Fatalf("isEncrypted expected %v got %v", toEncrypt, isEncrypted)
	}
	for i := range resultSlice {
		resultSlice[i] = 0
	}
	n, err = resultReader.ReadAt(resultSlice, 0)
	if err != io.EOF {
		t.Fatalf("Retrieve error after removing memStore: %v", err)
	}
	if n != len(slice) {
		t.Fatalf("Slice size error after removing memStore got %d, expected %d.", n, len(slice))
	}
	if !bytes.Equal(slice, resultSlice) {
		t.Fatalf("Comparison error after removing memStore.")
	}
}

func TestFileStoreCapacity(t *testing.T) {
	testFileStoreCapacity(false, t)
	testFileStoreCapacity(true, t)
}

func testFileStoreCapacity(toEncrypt bool, t *testing.T) {
	dir, err := ioutil.TempDir("", "swarm-storage-")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)
	localStore, err := localstore.New(dir, make([]byte, 32), nil)
	if err != nil {
		t.Fatal(err)
	}
	defer localStore.Close()

	fileStore := NewFileStore(localStore, NewFileStoreParams())
	slice := testutil.RandomBytes(1, testDataSize)
	ctx := context.TODO()
	key, wait, err := fileStore.Store(ctx, bytes.NewReader(slice), testDataSize, toEncrypt)
	if err != nil {
		t.Errorf("Store error: %v", err)
	}
	err = wait(ctx)
	if err != nil {
		t.Fatalf("Store error: %v", err)
	}
	resultReader, isEncrypted := fileStore.Retrieve(context.TODO(), key)
	if isEncrypted != toEncrypt {
		t.Fatalf("isEncrypted expected %v got %v", toEncrypt, isEncrypted)
	}
	resultSlice := make([]byte, len(slice))
	n, err := resultReader.ReadAt(resultSlice, 0)
	if err != io.EOF {
		t.Fatalf("Retrieve error: %v", err)
	}
	if n != len(slice) {
		t.Fatalf("Slice size error got %d, expected %d.", n, len(slice))
	}
	if !bytes.Equal(slice, resultSlice) {
		t.Fatalf("Comparison error.")
	}
	resultReader, isEncrypted = fileStore.Retrieve(context.TODO(), key)
	if isEncrypted != toEncrypt {
		t.Fatalf("isEncrypted expected %v got %v", toEncrypt, isEncrypted)
	}
	if _, err = resultReader.ReadAt(resultSlice, 0); err == nil {
		t.Fatalf("Was able to read %d bytes from an empty memStore.", len(slice))
	}
	// check how it works with localStore
	fileStore.ChunkStore = localStore
	//	localStore.dbStore.setCapacity(0)
	resultReader, isEncrypted = fileStore.Retrieve(context.TODO(), key)
	if isEncrypted != toEncrypt {
		t.Fatalf("isEncrypted expected %v got %v", toEncrypt, isEncrypted)
	}
	for i := range resultSlice {
		resultSlice[i] = 0
	}
	n, err = resultReader.ReadAt(resultSlice, 0)
	if err != io.EOF {
		t.Fatalf("Retrieve error after clearing memStore: %v", err)
	}
	if n != len(slice) {
		t.Fatalf("Slice size error after clearing memStore got %d, expected %d.", n, len(slice))
	}
	if !bytes.Equal(slice, resultSlice) {
		t.Fatalf("Comparison error after clearing memStore.")
	}
}

// TestGetAllReferences only tests that GetAllReferences returns an expected
// number of references for a given file
func TestGetAllReferences(t *testing.T) {
	dir, err := ioutil.TempDir("", "swarm-storage-")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)
	localStore, err := localstore.New(dir, make([]byte, 32), nil)
	if err != nil {
		t.Fatal(err)
	}
	defer localStore.Close()

	fileStore := NewFileStore(localStore, NewFileStoreParams())

	// testRuns[i] and expectedLen[i] are dataSize and expected length respectively
	testRuns := []int{1024, 8192, 16000, 30000, 1000000}
	expectedLens := []int{1, 3, 5, 9, 248}
	for i, r := range testRuns {
		slice := testutil.RandomBytes(1, r)

		addrs, err := fileStore.GetAllReferences(context.Background(), bytes.NewReader(slice), false)
		if err != nil {
			t.Fatal(err)
		}
		if len(addrs) != expectedLens[i] {
			t.Fatalf("Expected reference array length to be %d, but is %d", expectedLens[i], len(addrs))
		}
	}
}
