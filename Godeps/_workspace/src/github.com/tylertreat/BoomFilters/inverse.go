/*
Original work Copyright (c) 2012 Jeff Hodges. All rights reserved.
Modified work Copyright (c) 2015 Tyler Treat. All rights reserved.

Redistribution and use in source and binary forms, with or without
modification, are permitted provided that the following conditions are
met:

   * Redistributions of source code must retain the above copyright
notice, this list of conditions and the following disclaimer.
   * Redistributions in binary form must reproduce the above
copyright notice, this list of conditions and the following disclaimer
in the documentation and/or other materials provided with the
distribution.
   * Neither the name of Jeff Hodges nor the names of this project's
contributors may be used to endorse or promote products derived from
this software without specific prior written permission.

THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS
"AS IS" AND ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT
LIMITED TO, THE IMPLIED WARRANTIES OF MERCHANTABILITY AND FITNESS FOR
A PARTICULAR PURPOSE ARE DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT
OWNER OR CONTRIBUTORS BE LIABLE FOR ANY DIRECT, INDIRECT, INCIDENTAL,
SPECIAL, EXEMPLARY, OR CONSEQUENTIAL DAMAGES (INCLUDING, BUT NOT
LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR SERVICES; LOSS OF USE,
DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER CAUSED AND ON ANY
THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY, OR TORT
(INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE
OF THIS SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.
*/

package boom

import (
	"bytes"
	"hash"
	"hash/fnv"
	"sync"
	"sync/atomic"
	"unsafe"
)

// InverseBloomFilter is a concurrent "inverse" Bloom filter, which is
// effectively the opposite of a classic Bloom filter. This was originally
// described and written by Jeff Hodges:
//
// http://www.somethingsimilar.com/2012/05/21/the-opposite-of-a-bloom-filter/
//
// The InverseBloomFilter may report a false negative but can never report a
// false positive. That is, it may report that an item has not been seen when
// it actually has, but it will never report an item as seen which it hasn't
// come across. This behaves in a similar manner to a fixed-size hashmap which
// does not handle conflicts.
//
// An example use case is deduplicating events while processing a stream of
// data. Ideally, duplicate events are relatively close together.
type InverseBloomFilter struct {
	array    []*[]byte
	hashPool *sync.Pool
	capacity uint
}

// NewInverseBloomFilter creates and returns a new InverseBloomFilter with the
// specified capacity.
func NewInverseBloomFilter(capacity uint) *InverseBloomFilter {
	return &InverseBloomFilter{
		array:    make([]*[]byte, capacity),
		hashPool: &sync.Pool{New: func() interface{} { return fnv.New32() }},
		capacity: capacity,
	}
}

// Test will test for membership of the data and returns true if it is a
// member, false if not. This is a probabilistic test, meaning there is a
// non-zero probability of false negatives but a zero probability of false
// positives. That is, it may return false even though the data was added, but
// it will never return true for data that hasn't been added.
func (i *InverseBloomFilter) Test(data []byte) bool {
	index := i.index(data)
	indexPtr := (*unsafe.Pointer)(unsafe.Pointer(&i.array[index]))
	val := (*[]byte)(atomic.LoadPointer(indexPtr))
	if val == nil {
		return false
	}
	return bytes.Equal(*val, data)
}

// Add will add the data to the filter. It returns the filter to allow for
// chaining.
func (i *InverseBloomFilter) Add(data []byte) Filter {
	index := i.index(data)
	i.getAndSet(index, data)
	return i
}

// TestAndAdd is equivalent to calling Test followed by Add atomically. It
// returns true if the data is a member, false if not.
func (i *InverseBloomFilter) TestAndAdd(data []byte) bool {
	oldID := i.getAndSet(i.index(data), data)
	return bytes.Equal(oldID, data)
}

// Capacity returns the filter capacity.
func (i *InverseBloomFilter) Capacity() uint {
	return i.capacity
}

// getAndSet returns the data that was in the slice at the given index after
// putting the new data in the slice at that index, atomically.
func (i *InverseBloomFilter) getAndSet(index uint32, data []byte) []byte {
	indexPtr := (*unsafe.Pointer)(unsafe.Pointer(&i.array[index]))
	keyUnsafe := unsafe.Pointer(&data)
	var oldKey []byte
	for {
		oldKeyUnsafe := atomic.LoadPointer(indexPtr)
		if atomic.CompareAndSwapPointer(indexPtr, oldKeyUnsafe, keyUnsafe) {
			oldKeyPtr := (*[]byte)(oldKeyUnsafe)
			if oldKeyPtr != nil {
				oldKey = *oldKeyPtr
			}
			break
		}
	}
	return oldKey
}

// index returns the array index for the given data.
func (i *InverseBloomFilter) index(data []byte) uint32 {
	hash := i.hashPool.Get().(hash.Hash32)
	hash.Write(data)
	index := hash.Sum32() % uint32(i.capacity)
	hash.Reset()
	i.hashPool.Put(hash)
	return index
}

// SetHashFactory sets the hashing function factory used in the filter.
func (i *InverseBloomFilter) SetHashFactory(h func() hash.Hash32) {
	i.hashPool = &sync.Pool{New: func() interface{} { return h() }}
}
