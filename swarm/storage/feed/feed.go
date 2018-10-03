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

package feed

import (
	"hash"
	"unsafe"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/swarm/storage"
)

// Feed represents a particular user's stream of updates on a topic
type Feed struct {
	Topic Topic          `json:"topic"`
	User  common.Address `json:"user"`
}

// Feed layout:
// TopicLength bytes
// userAddr common.AddressLength bytes
const feedLength = TopicLength + common.AddressLength

// mapKey calculates a unique id for this feed. Used by the cache map in `Handler`
func (f *Feed) mapKey() uint64 {
	serializedData := make([]byte, feedLength)
	f.binaryPut(serializedData)
	hasher := hashPool.Get().(hash.Hash)
	defer hashPool.Put(hasher)
	hasher.Reset()
	hasher.Write(serializedData)
	hash := hasher.Sum(nil)
	return *(*uint64)(unsafe.Pointer(&hash[0]))
}

// binaryPut serializes this feed instance into the provided slice
func (f *Feed) binaryPut(serializedData []byte) error {
	if len(serializedData) != feedLength {
		return NewErrorf(ErrInvalidValue, "Incorrect slice size to serialize feed. Expected %d, got %d", feedLength, len(serializedData))
	}
	var cursor int
	copy(serializedData[cursor:cursor+TopicLength], f.Topic[:TopicLength])
	cursor += TopicLength

	copy(serializedData[cursor:cursor+common.AddressLength], f.User[:])
	cursor += common.AddressLength

	return nil
}

// binaryLength returns the expected size of this structure when serialized
func (f *Feed) binaryLength() int {
	return feedLength
}

// binaryGet restores the current instance from the information contained in the passed slice
func (f *Feed) binaryGet(serializedData []byte) error {
	if len(serializedData) != feedLength {
		return NewErrorf(ErrInvalidValue, "Incorrect slice size to read feed. Expected %d, got %d", feedLength, len(serializedData))
	}

	var cursor int
	copy(f.Topic[:], serializedData[cursor:cursor+TopicLength])
	cursor += TopicLength

	copy(f.User[:], serializedData[cursor:cursor+common.AddressLength])
	cursor += common.AddressLength

	return nil
}

// Hex serializes the feed to a hex string
func (f *Feed) Hex() string {
	serializedData := make([]byte, feedLength)
	f.binaryPut(serializedData)
	return hexutil.Encode(serializedData)
}

// FromValues deserializes this instance from a string key-value store
// useful to parse query strings
func (f *Feed) FromValues(values Values) (err error) {
	topic := values.Get("topic")
	if topic != "" {
		if err := f.Topic.FromHex(values.Get("topic")); err != nil {
			return err
		}
	} else { // see if the user set name and relatedcontent
		name := values.Get("name")
		relatedContent, _ := hexutil.Decode(values.Get("relatedcontent"))
		if len(relatedContent) > 0 {
			if len(relatedContent) < storage.AddressLength {
				return NewErrorf(ErrInvalidValue, "relatedcontent field must be a hex-encoded byte array exactly %d bytes long", storage.AddressLength)
			}
			relatedContent = relatedContent[:storage.AddressLength]
		}
		f.Topic, err = NewTopic(name, relatedContent)
		if err != nil {
			return err
		}
	}
	f.User = common.HexToAddress(values.Get("user"))
	return nil
}

// AppendValues serializes this structure into the provided string key-value store
// useful to build query strings
func (f *Feed) AppendValues(values Values) {
	values.Set("topic", f.Topic.Hex())
	values.Set("user", f.User.Hex())
}
