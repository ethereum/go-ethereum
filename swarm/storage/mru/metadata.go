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

package mru

import (
	"encoding/binary"
	"hash"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/swarm/storage"
)

// ResourceMetadata encapsulates the immutable information about a mutable resource :)
// once serialized into a chunk, the resource can be retrieved by knowing its content-addressed rootAddr
type ResourceMetadata struct {
	StartTime Timestamp      // time at which the resource starts to be valid
	Frequency uint64         // expected update frequency for the resource
	Name      string         // name of the resource, for the reference of the user or to disambiguate resources with same starttime, frequency, owneraddr
	Owner     common.Address // public address of the resource owner
}

const frequencyLength = 8 // sizeof(uint64)
const nameLengthLength = 1

// Resource metadata chunk layout:
// 4 prefix bytes (chunkPrefixLength). The first two set to zero. The second two indicate the length
// Timestamp: timestampLength bytes
// frequency: frequencyLength bytes
// name length: nameLengthLength bytes
// name (variable length, can be empty, up to 255 bytes)
// ownerAddr: common.AddressLength
const minimumMetadataLength = chunkPrefixLength + timestampLength + frequencyLength + nameLengthLength + common.AddressLength

// binaryGet populates the resource metadata from a byte array
func (r *ResourceMetadata) binaryGet(serializedData []byte) error {
	if len(serializedData) < minimumMetadataLength {
		return NewErrorf(ErrInvalidValue, "Metadata chunk to deserialize is too short. Expected at least %d. Got %d.", minimumMetadataLength, len(serializedData))
	}

	// first two bytes must be set to zero to indicate metadata chunks, so enforce this.
	if serializedData[0] != 0 || serializedData[1] != 0 {
		return NewError(ErrCorruptData, "Invalid metadata chunk")
	}

	cursor := 2
	metadataLength := int(binary.LittleEndian.Uint16(serializedData[cursor : cursor+2])) // metadataLength does not include the 4 prefix bytes
	if metadataLength+chunkPrefixLength != len(serializedData) {
		return NewErrorf(ErrCorruptData, "Incorrect declared metadata length. Expected %d, got %d.", metadataLength+chunkPrefixLength, len(serializedData))
	}

	cursor += 2

	if err := r.StartTime.binaryGet(serializedData[cursor : cursor+timestampLength]); err != nil {
		return err
	}
	cursor += timestampLength

	r.Frequency = binary.LittleEndian.Uint64(serializedData[cursor : cursor+frequencyLength])
	cursor += frequencyLength

	nameLength := int(serializedData[cursor])
	if nameLength+minimumMetadataLength > len(serializedData) {
		return NewErrorf(ErrInvalidValue, "Metadata chunk to deserialize is too short when decoding resource name. Expected at least %d. Got %d.", nameLength+minimumMetadataLength, len(serializedData))
	}
	cursor++
	r.Name = string(serializedData[cursor : cursor+nameLength])
	cursor += nameLength

	copy(r.Owner[:], serializedData[cursor:])
	cursor += common.AddressLength
	if cursor != len(serializedData) {
		return NewErrorf(ErrInvalidValue, "Metadata chunk has leftover data after deserialization. %d left to read", len(serializedData)-cursor)
	}
	return nil
}

// binaryPut encodes the metadata into a byte array
func (r *ResourceMetadata) binaryPut(serializedData []byte) error {
	metadataChunkLength := r.binaryLength()
	if len(serializedData) != metadataChunkLength {
		return NewErrorf(ErrInvalidValue, "Need a slice of exactly %d bytes to serialize this metadata, but got a slice of size %d.", metadataChunkLength, len(serializedData))
	}

	// root chunk has first two bytes both set to 0, which distinguishes from update bytes
	// therefore, skip the first two bytes of a zero-initialized array.
	cursor := 2
	binary.LittleEndian.PutUint16(serializedData[cursor:cursor+2], uint16(metadataChunkLength-chunkPrefixLength)) // metadataLength does not include the 4 prefix bytes
	cursor += 2

	r.StartTime.binaryPut(serializedData[cursor : cursor+timestampLength])
	cursor += timestampLength

	binary.LittleEndian.PutUint64(serializedData[cursor:cursor+frequencyLength], r.Frequency)
	cursor += frequencyLength

	// Encode the name string as a 1 byte length followed by the encoded string.
	// Longer strings will be truncated.
	nameLength := len(r.Name)
	if nameLength > 255 {
		nameLength = 255
	}
	serializedData[cursor] = uint8(nameLength)
	cursor++
	copy(serializedData[cursor:cursor+nameLength], []byte(r.Name[:nameLength]))
	cursor += nameLength

	copy(serializedData[cursor:cursor+common.AddressLength], r.Owner[:])
	cursor += common.AddressLength

	return nil
}

func (r *ResourceMetadata) binaryLength() int {
	return minimumMetadataLength + len(r.Name)
}

// serializeAndHash returns the root chunk addr and metadata hash that help identify and ascertain ownership of this resource
// returns the serialized metadata as a byproduct of having to hash it.
func (r *ResourceMetadata) serializeAndHash() (rootAddr, metaHash []byte, chunkData []byte, err error) {

	chunkData = make([]byte, r.binaryLength())
	if err := r.binaryPut(chunkData); err != nil {
		return nil, nil, nil, err
	}
	rootAddr, metaHash = metadataHash(chunkData)
	return rootAddr, metaHash, chunkData, nil

}

// creates a metadata chunk out of a resourceMetadata structure
func (metadata *ResourceMetadata) newChunk() (chunk storage.Chunk, metaHash []byte, err error) {
	// the metadata chunk contains a timestamp of when the resource starts to be valid
	// and also how frequently it is expected to be updated
	// from this we know at what time we should look for updates, and how often
	// it also contains the name of the resource, so we know what resource we are working with

	// the key (rootAddr) of the metadata chunk is content-addressed
	// if it wasn't we couldn't replace it later
	// resolving this relationship is left up to external agents (for example ENS)
	rootAddr, metaHash, chunkData, err := metadata.serializeAndHash()
	if err != nil {
		return nil, nil, err
	}

	// make the chunk and send it to swarm
	chunk = storage.NewChunk(rootAddr, chunkData)

	return chunk, metaHash, nil
}

// metadataHash returns the metadata chunk root address and metadata hash
// that help identify and ascertain ownership of this resource
// We compute it as rootAddr = H(ownerAddr, H(metadata))
// Where H() is SHA3
// metadata are all the metadata fields, except ownerAddr
// ownerAddr is the public address of the resource owner
// Update chunks must carry a rootAddr reference and metaHash in order to be verified
// This way, a node that receives an update can check the signature, recover the public address
// and check the ownership by computing H(ownerAddr, metaHash) and comparing it to the rootAddr
// the resource is claiming to update without having to lookup the metadata chunk.
// see verifyResourceOwnerhsip in signedupdate.go
func metadataHash(chunkData []byte) (rootAddr, metaHash []byte) {
	hasher := hashPool.Get().(hash.Hash)
	defer hashPool.Put(hasher)
	hasher.Reset()
	hasher.Write(chunkData[:len(chunkData)-common.AddressLength])
	metaHash = hasher.Sum(nil)
	hasher.Reset()
	hasher.Write(metaHash)
	hasher.Write(chunkData[len(chunkData)-common.AddressLength:])
	rootAddr = hasher.Sum(nil)
	return
}
