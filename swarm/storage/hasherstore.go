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
	"fmt"
	"sync"

	"github.com/ethereum/go-ethereum/crypto/sha3"
	"github.com/ethereum/go-ethereum/swarm/storage/encryption"
)

type chunkEncryption struct {
	spanEncryption encryption.Encryption
	dataEncryption encryption.Encryption
}

type hasherStore struct {
	store           ChunkStore
	hashFunc        SwarmHasher
	chunkEncryption *chunkEncryption
	hashSize        int   // content hash size
	refSize         int64 // reference size (content hash + possibly encryption key)
	wg              *sync.WaitGroup
	closed          chan struct{}
}

func newChunkEncryption(chunkSize, refSize int64) *chunkEncryption {
	return &chunkEncryption{
		spanEncryption: encryption.New(0, uint32(chunkSize/refSize), sha3.NewKeccak256),
		dataEncryption: encryption.New(int(chunkSize), 0, sha3.NewKeccak256),
	}
}

// NewHasherStore creates a hasherStore object, which implements Putter and Getter interfaces.
// With the HasherStore you can put and get chunk data (which is just []byte) into a ChunkStore
// and the hasherStore will take core of encryption/decryption of data if necessary
func NewHasherStore(chunkStore ChunkStore, hashFunc SwarmHasher, toEncrypt bool) *hasherStore {
	var chunkEncryption *chunkEncryption

	hashSize := hashFunc().Size()
	refSize := int64(hashSize)
	if toEncrypt {
		refSize += encryption.KeyLength
		chunkEncryption = newChunkEncryption(DefaultChunkSize, refSize)
	}

	return &hasherStore{
		store:           chunkStore,
		hashFunc:        hashFunc,
		chunkEncryption: chunkEncryption,
		hashSize:        hashSize,
		refSize:         refSize,
		wg:              &sync.WaitGroup{},
		closed:          make(chan struct{}),
	}
}

// Put stores the chunkData into the ChunkStore of the hasherStore and returns the reference.
// If hasherStore has a chunkEncryption object, the data will be encrypted.
// Asynchronous function, the data will not necessarily be stored when it returns.
func (h *hasherStore) Put(ctx context.Context, chunkData ChunkData) (Reference, error) {
	c := chunkData
	size := chunkData.Size()
	var encryptionKey encryption.Key
	if h.chunkEncryption != nil {
		var err error
		c, encryptionKey, err = h.encryptChunkData(chunkData)
		if err != nil {
			return nil, err
		}
	}
	chunk := h.createChunk(c, size)

	h.storeChunk(ctx, chunk)

	return Reference(append(chunk.Addr, encryptionKey...)), nil
}

// Get returns data of the chunk with the given reference (retrieved from the ChunkStore of hasherStore).
// If the data is encrypted and the reference contains an encryption key, it will be decrypted before
// return.
func (h *hasherStore) Get(ctx context.Context, ref Reference) (ChunkData, error) {
	key, encryptionKey, err := parseReference(ref, h.hashSize)
	if err != nil {
		return nil, err
	}
	toDecrypt := (encryptionKey != nil)

	chunk, err := h.store.Get(ctx, key)
	if err != nil {
		return nil, err
	}

	chunkData := chunk.SData
	if toDecrypt {
		var err error
		chunkData, err = h.decryptChunkData(chunkData, encryptionKey)
		if err != nil {
			return nil, err
		}
	}
	return chunkData, nil
}

// Close indicates that no more chunks will be put with the hasherStore, so the Wait
// function can return when all the previously put chunks has been stored.
func (h *hasherStore) Close() {
	close(h.closed)
}

// Wait returns when
//    1) the Close() function has been called and
//    2) all the chunks which has been Put has been stored
func (h *hasherStore) Wait(ctx context.Context) error {
	<-h.closed
	h.wg.Wait()
	return nil
}

func (h *hasherStore) createHash(chunkData ChunkData) Address {
	hasher := h.hashFunc()
	hasher.ResetWithLength(chunkData[:8]) // 8 bytes of length
	hasher.Write(chunkData[8:])           // minus 8 []byte length
	return hasher.Sum(nil)
}

func (h *hasherStore) createChunk(chunkData ChunkData, chunkSize int64) *Chunk {
	hash := h.createHash(chunkData)
	chunk := NewChunk(hash, nil)
	chunk.SData = chunkData
	chunk.Size = chunkSize

	return chunk
}

func (h *hasherStore) encryptChunkData(chunkData ChunkData) (ChunkData, encryption.Key, error) {
	if len(chunkData) < 8 {
		return nil, nil, fmt.Errorf("Invalid ChunkData, min length 8 got %v", len(chunkData))
	}

	encryptionKey, err := encryption.GenerateRandomKey()
	if err != nil {
		return nil, nil, err
	}

	encryptedSpan, err := h.chunkEncryption.spanEncryption.Encrypt(chunkData[:8], encryptionKey)
	if err != nil {
		return nil, nil, err
	}
	encryptedData, err := h.chunkEncryption.dataEncryption.Encrypt(chunkData[8:], encryptionKey)
	if err != nil {
		return nil, nil, err
	}
	c := make(ChunkData, len(encryptedSpan)+len(encryptedData))
	copy(c[:8], encryptedSpan)
	copy(c[8:], encryptedData)
	return c, encryptionKey, nil
}

func (h *hasherStore) decryptChunkData(chunkData ChunkData, encryptionKey encryption.Key) (ChunkData, error) {
	if len(chunkData) < 8 {
		return nil, fmt.Errorf("Invalid ChunkData, min length 8 got %v", len(chunkData))
	}

	decryptedSpan, err := h.chunkEncryption.spanEncryption.Decrypt(chunkData[:8], encryptionKey)
	if err != nil {
		return nil, err
	}

	decryptedData, err := h.chunkEncryption.dataEncryption.Decrypt(chunkData[8:], encryptionKey)
	if err != nil {
		return nil, err
	}

	// removing extra bytes which were just added for padding
	length := ChunkData(decryptedSpan).Size()
	for length > DefaultChunkSize {
		length = length + (DefaultChunkSize - 1)
		length = length / DefaultChunkSize
		length *= h.refSize
	}

	c := make(ChunkData, length+8)
	copy(c[:8], decryptedSpan)
	copy(c[8:], decryptedData[:length])

	return c[:length+8], nil
}

func (h *hasherStore) RefSize() int64 {
	return h.refSize
}

func (h *hasherStore) storeChunk(ctx context.Context, chunk *Chunk) {
	h.wg.Add(1)
	go func() {
		<-chunk.dbStoredC
		h.wg.Done()
	}()
	h.store.Put(ctx, chunk)
}

func parseReference(ref Reference, hashSize int) (Address, encryption.Key, error) {
	encryptedKeyLength := hashSize + encryption.KeyLength
	switch len(ref) {
	case KeyLength:
		return Address(ref), nil, nil
	case encryptedKeyLength:
		encKeyIdx := len(ref) - encryption.KeyLength
		return Address(ref[:encKeyIdx]), encryption.Key(ref[encKeyIdx:]), nil
	default:
		return nil, nil, fmt.Errorf("Invalid reference length, expected %v or %v got %v", hashSize, encryptedKeyLength, len(ref))
	}

}
