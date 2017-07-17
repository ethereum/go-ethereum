// Copyright 2017 The go-ethereum Authors
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

package whisperv5

import (
	"testing"
)

var fakeTopic = []byte{0xde, 0xad, 0xbe, 0xef}

// Size of a KB
const KB = 0x400

// Ensure the alignment of the bloom filter has been checked
func TestBloomFilterInvalidByteAligment(t *testing.T) {
	if _, ok := CalculateBloomFilter(fakeTopic, 3); ok != errBloomFilterLength {
		t.Fatal("Did not catch an invalid bloom filter size")
	}
}

// Ensure that the passed topic is checked for validity
func TestBloomFilterCheckTopicValid(t *testing.T) {
	if _, ok := CalculateBloomFilter([]byte{0x01}, 2); ok != errInvalidTopic {
		t.Fatal("Did not check for an invalid topic")
	}

	if _, ok := CalculateBloomFilter(nil, 2); ok != errInvalidTopic {
		t.Fatal("Did not check for an invalid topic")
	}
}

// Ensure that the function checks for the lower boundary of the bloom filter
func TestBloomFilterRequestedSizeTooSmall(t *testing.T) {
	if _, ok := CalculateBloomFilter(fakeTopic, 0); ok != errBloomFilterLength {
		t.Fatal("Did not catch an invalid bloom filter size")
	}
}

// Ensure that the function checks for the upper boundary of the bloom filter
func TestBloomFilterRequestedSizeTooBig(t *testing.T) {
	if _, ok := CalculateBloomFilter(fakeTopic, 1024*KB); ok != errBloomFilterLength {
		t.Fatal("Did not catch an invalid bloom filter size")
	}
}

// Test different size to ensure the proper size is always returned
func TestBloomFilterReturnsCorrectSize(t *testing.T) {
	for size := 1; size <= 512*KB; size *= 2 {
		if filter, ok := CalculateBloomFilter(fakeTopic, uint(size)); ok != nil || len(filter) != size {
			t.Fatalf("Returned a size of %v instead of %v", len(filter), size)
		}
	}
}

// Test that, for different topics, the function always returns exactly 3 bits
func TestBloomFilterReturns3BitsSetDifferentTopic(t *testing.T) {
	for i := 0; i < 256; i++ {
		for j := 0; j < 256; j++ {
			topic := []byte{0x00, 0x00, byte(j), byte(i)}

			filter, ok := CalculateBloomFilter(topic, 512)

			if ok != nil {
				t.Fatal("Failed to return the proper bloom filter")
			}

			nbits := 0
			for _, v := range filter {
				for ; v != 0; v >>= 1 {
					if v&1 == 1 {
						nbits++
					}
				}
			}

			if nbits != 3 {
				t.Fatalf("Returned an invalid number of bits (%v, %v)", nbits, filter)
			}
		}
	}
}

// Test that, for different sizes, the function always returns exactly 3 bits
func TestBloomFilterReturns3BitsSetDifferentSize(t *testing.T) {
	for size := 2; size <= 512*KB; size *= 2 {
		filter, ok := CalculateBloomFilter(fakeTopic, uint(size))

		if ok != nil {
			t.Fatal("Failed to return the proper bloom filter")
		}

		nbits := 0
		for _, v := range filter {
			for ; v != 0; v >>= 1 {
				if v&1 == 1 {
					nbits++
				}
			}
		}

		if nbits != 3 {
			t.Fatalf("Returned an invalid number of bits (%v, %v)", nbits, filter)
		}
	}
}

// Test that two bloom filters with different lengths never match
func TestBloomFilterContainsTopicsLengthMismatch(t *testing.T) {
	if ok := bfContainsTopics([]byte{0x01}, []byte{0x02, 0x03}); ok {
		t.Fatal("Non-matching lengths reported as matching")
	}
}

// Test that two bloom filters with different content don't match _a priori_
func TestBloomFilterContainsTopicsContentMismatch(t *testing.T) {
	if ok := bfContainsTopics([]byte{0x01}, []byte{0x02}); ok {
		t.Fatal("Non-matching lengths reported as matching")
	}
}

// Test that identical topics match
func TestBloomFilterContainsTopicsEqual(t *testing.T) {
	if ok := bfContainsTopics([]byte{0x01}, []byte{0x01}); !ok {
		t.Fatal("Identical topics didn't match")
	}
}

// Test inclusion match`
func TestBloomFilterContainsTopicsInclusion(t *testing.T) {
	if ok := bfContainsTopics([]byte{0x03}, []byte{0x01}); ok {
		t.Fatal("Included topics not found")
	}
}

// Test that reverse-inclusion fails
func TestBloomFilterContainsTopicsSwapInclusion(t *testing.T) {
	if ok := bfContainsTopics([]byte{0x01}, []byte{0x03}); !ok {
		t.Fatal("Swapping filter and topics shouldn't match")
	}
}

// Test that two bloom filters with different lengths can never be added
func TestBloomFilterAddTopicsLengthMismatch(t *testing.T) {
	if ok := bfAddTopics([]byte{0x01}, []byte{0x02, 0x03}); ok == nil {
		t.Fatal("Non-matching lengths where added")
	}
}

// Test that two bloom filters with different lengths can never be added
func TestBloomFilterAddTopicWorks(t *testing.T) {
	filter := []byte{0x02}
	if ok := bfAddTopics([]byte{0x01}, filter); ok == nil {
		if filter[0] != 0x03 {
			t.Fatal("Adding a topic creates an invalid bloom filter")
		}
	}
}
