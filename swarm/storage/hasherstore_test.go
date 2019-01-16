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
	"bytes"
	"context"
	"testing"

	"github.com/ethereum/go-ethereum/swarm/storage/encryption"

	"github.com/ethereum/go-ethereum/common"
)

func TestHasherStore(t *testing.T) {
	var tests = []struct {
		chunkLength int
		toEncrypt   bool
	}{
		{10, false},
		{100, false},
		{1000, false},
		{4096, false},
		{10, true},
		{100, true},
		{1000, true},
		{4096, true},
	}

	for _, tt := range tests {
		chunkStore := NewMapChunkStore()
		hasherStore := NewHasherStore(chunkStore, MakeHashFunc(DefaultHash), tt.toEncrypt)

		// Put two random chunks into the hasherStore
		chunkData1 := GenerateRandomChunk(int64(tt.chunkLength)).Data()
		ctx, cancel := context.WithTimeout(context.Background(), getTimeout)
		defer cancel()
		key1, err := hasherStore.Put(ctx, chunkData1)
		if err != nil {
			t.Fatalf("Expected no error got \"%v\"", err)
		}

		chunkData2 := GenerateRandomChunk(int64(tt.chunkLength)).Data()
		key2, err := hasherStore.Put(ctx, chunkData2)
		if err != nil {
			t.Fatalf("Expected no error got \"%v\"", err)
		}

		hasherStore.Close()

		// Wait until chunks are really stored
		err = hasherStore.Wait(ctx)
		if err != nil {
			t.Fatalf("Expected no error got \"%v\"", err)
		}

		// Get the first chunk
		retrievedChunkData1, err := hasherStore.Get(ctx, key1)
		if err != nil {
			t.Fatalf("Expected no error, got \"%v\"", err)
		}

		// Retrieved data should be same as the original
		if !bytes.Equal(chunkData1, retrievedChunkData1) {
			t.Fatalf("Expected retrieved chunk data %v, got %v", common.Bytes2Hex(chunkData1), common.Bytes2Hex(retrievedChunkData1))
		}

		// Get the second chunk
		retrievedChunkData2, err := hasherStore.Get(ctx, key2)
		if err != nil {
			t.Fatalf("Expected no error, got \"%v\"", err)
		}

		// Retrieved data should be same as the original
		if !bytes.Equal(chunkData2, retrievedChunkData2) {
			t.Fatalf("Expected retrieved chunk data %v, got %v", common.Bytes2Hex(chunkData2), common.Bytes2Hex(retrievedChunkData2))
		}

		hash1, encryptionKey1, err := parseReference(key1, hasherStore.hashSize)
		if err != nil {
			t.Fatalf("Expected no error, got \"%v\"", err)
		}

		if tt.toEncrypt {
			if encryptionKey1 == nil {
				t.Fatal("Expected non-nil encryption key, got nil")
			} else if len(encryptionKey1) != encryption.KeyLength {
				t.Fatalf("Expected encryption key length %v, got %v", encryption.KeyLength, len(encryptionKey1))
			}
		}
		if !tt.toEncrypt && encryptionKey1 != nil {
			t.Fatalf("Expected nil encryption key, got key with length %v", len(encryptionKey1))
		}

		// Check if chunk data in store is encrypted or not
		chunkInStore, err := chunkStore.Get(ctx, hash1)
		if err != nil {
			t.Fatalf("Expected no error got \"%v\"", err)
		}

		chunkDataInStore := chunkInStore.Data()

		if tt.toEncrypt && bytes.Equal(chunkData1, chunkDataInStore) {
			t.Fatalf("Chunk expected to be encrypted but it is stored without encryption")
		}
		if !tt.toEncrypt && !bytes.Equal(chunkData1, chunkDataInStore) {
			t.Fatalf("Chunk expected to be not encrypted but stored content is different. Expected %v got %v", common.Bytes2Hex(chunkData1), common.Bytes2Hex(chunkDataInStore))
		}
	}
}
