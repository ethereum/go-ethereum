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
	"encoding/binary"
	"errors"
	"math"

	"github.com/ethereum/go-ethereum/crypto"
)

var (
	errBloomFilterLength = errors.New("The length of the bloom filter must be a power of two between 2 bytes and 4MB")
	errLengthMismatch    = errors.New("Length of topic and bloom filter don't match")
	errInvalidTopic      = errors.New("Topic is not a 4 byte array")
)

// CalculateBloomFilter ceates a bloom filter of `nbytes` bytes with 3 bits set
// depending on the Keccak256 hash of the topic.
func CalculateBloomFilter(topic []byte, nbytes uint) (output []byte, err error) {
	var nbits uint = 0    // Number of bits set in the bloom filter
	var bitsread uint = 0 // Number of bits read so far in the 256-bits hash

	if nbytes == 0 || nbytes > BloomFilterLengthMax {
		err = errBloomFilterLength
		return nil, err
	}

	if len(topic) != 4 {
		err = errInvalidTopic
		return nil, err
	}

	// Check that nbytes is a power of two, as it is assumed by the rest of that
	// function. `bitpitch` is the amount by which the read will advance.
	bitpitch := uint(math.Floor(math.Log(float64(nbytes*8))/math.Log(2.0) + .5))
	if 8*nbytes != 1<<bitpitch {
		err = errBloomFilterLength
		return nil, err
	}

	output = make([]byte, nbytes)
	hash := crypto.Keccak256(topic[:])

	for {
		// The max size of the bloom filter is 512KB, which is represented by
		// enough bits to fit in an unsigned 32-bits integer. Extract the correct
		// number of bits from that integer.
		buffer := binary.LittleEndian.Uint32(hash[bitsread/8:])
		buffer >>= bitsread % 8
		bitnum := buffer & (1<<bitpitch - 1) // bitnum is the bit number in the bloom filter

		// If that specific bit hasn't been set yet, allocate it. Otherwise, look
		// for the next one.
		var targetMask byte = 1 << (bitnum % 8)
		if output[bitnum/8]&targetMask == 0 {
			output[bitnum/8] |= targetMask
			nbits += 1
		}

		// End as soon as 3 bits have been found.
		if nbits == 3 {
			return output, nil
		} else {
			// advance the read in order to find another bit
			bitsread += bitpitch

			// If 256 bits weren't enough, hash again to get another sequence.
			if int(bitsread+bitpitch) >= len(hash)*8 {
				hash = crypto.Keccak256(hash)
				bitsread = 0
			}
		}
	}
}

// bloomContainsTopics checks if a bloom filter "contains" all the topics that are passed.
func bloomContainsTopics(bloomFilter []byte, topics []byte) (found bool) {
	if len(bloomFilter) != len(topics) {
		return false
	}

	// Go over every byte of topics and check its set bits are also present
	// in the filter
	for i, v := range topics {
		if bloomFilter[i]&v != v {
			return false
		}
	}

	return true
}

// bloomAddTopics adds a list of topics to a given Bloom filter
func bloomAddTopics(bloomFilter []byte, topics []byte) error {
	if len(bloomFilter) != len(topics) {
		return errLengthMismatch
	}

	// Just set each bits that are set in v
	for i, v := range topics {
		bloomFilter[i] |= v
	}

	return nil
}
