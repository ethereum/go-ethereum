// Copyright 2018 The go-ethereum Authors
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
	"context"
	"io/ioutil"
	"os"
	"testing"

	ch "github.com/ethereum/go-ethereum/swarm/chunk"
)

var (
	hashfunc = MakeHashFunc(DefaultHash)
)

// tests that the content address validator correctly checks the data
// tests that feed update chunks are passed through content address validator
// the test checking the resouce update validator internal correctness is found in storage/feeds/handler_test.go
func TestValidator(t *testing.T) {
	// set up localstore
	datadir, err := ioutil.TempDir("", "storage-testvalidator")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(datadir)

	params := NewDefaultLocalStoreParams()
	params.Init(datadir)
	store, err := NewLocalStore(params, nil)
	if err != nil {
		t.Fatal(err)
	}

	// check puts with no validators, both succeed
	chunks := GenerateRandomChunks(259, 2)
	goodChunk := chunks[0]
	badChunk := chunks[1]
	copy(badChunk.Data(), goodChunk.Data())

	errs := putChunks(store, goodChunk, badChunk)
	if errs[0] != nil {
		t.Fatalf("expected no error on good content address chunk in spite of no validation, but got: %s", err)
	}
	if errs[1] != nil {
		t.Fatalf("expected no error on bad content address chunk in spite of no validation, but got: %s", err)
	}

	// add content address validator and check puts
	// bad should fail, good should pass
	store.Validators = append(store.Validators, NewContentAddressValidator(hashfunc))
	chunks = GenerateRandomChunks(ch.DefaultSize, 2)
	goodChunk = chunks[0]
	badChunk = chunks[1]
	copy(badChunk.Data(), goodChunk.Data())

	errs = putChunks(store, goodChunk, badChunk)
	if errs[0] != nil {
		t.Fatalf("expected no error on good content address chunk with content address validator only, but got: %s", err)
	}
	if errs[1] == nil {
		t.Fatal("expected error on bad content address chunk with content address validator only, but got nil")
	}

	// append a validator that always denies
	// bad should fail, good should pass,
	var negV boolTestValidator
	store.Validators = append(store.Validators, negV)

	chunks = GenerateRandomChunks(ch.DefaultSize, 2)
	goodChunk = chunks[0]
	badChunk = chunks[1]
	copy(badChunk.Data(), goodChunk.Data())

	errs = putChunks(store, goodChunk, badChunk)
	if errs[0] != nil {
		t.Fatalf("expected no error on good content address chunk with content address validator only, but got: %s", err)
	}
	if errs[1] == nil {
		t.Fatal("expected error on bad content address chunk with content address validator only, but got nil")
	}

	// append a validator that always approves
	// all shall pass
	var posV boolTestValidator = true
	store.Validators = append(store.Validators, posV)

	chunks = GenerateRandomChunks(ch.DefaultSize, 2)
	goodChunk = chunks[0]
	badChunk = chunks[1]
	copy(badChunk.Data(), goodChunk.Data())

	errs = putChunks(store, goodChunk, badChunk)
	if errs[0] != nil {
		t.Fatalf("expected no error on good content address chunk with content address validator only, but got: %s", err)
	}
	if errs[1] != nil {
		t.Fatalf("expected no error on bad content address chunk in spite of no validation, but got: %s", err)
	}

}

type boolTestValidator bool

func (self boolTestValidator) Validate(addr Address, data []byte) bool {
	return bool(self)
}

// putChunks adds chunks  to localstore
// It waits for receive on the stored channel
// It logs but does not fail on delivery error
func putChunks(store *LocalStore, chunks ...Chunk) []error {
	i := 0
	f := func(n int64) Chunk {
		chunk := chunks[i]
		i++
		return chunk
	}
	_, errs := put(store, len(chunks), f)
	return errs
}

func put(store *LocalStore, n int, f func(i int64) Chunk) (hs []Address, errs []error) {
	for i := int64(0); i < int64(n); i++ {
		chunk := f(ch.DefaultSize)
		err := store.Put(context.TODO(), chunk)
		errs = append(errs, err)
		hs = append(hs, chunk.Address())
	}
	return hs, errs
}
