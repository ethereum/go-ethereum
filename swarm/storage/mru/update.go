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
	"errors"

	"github.com/ethereum/go-ethereum/swarm/chunk"
	"github.com/ethereum/go-ethereum/swarm/log"
	"github.com/ethereum/go-ethereum/swarm/multihash"
)

// resourceUpdate encapsulates the information sent as part of a resource update
type resourceUpdate struct {
	updateHeader        // metainformationa about this resource update
	data         []byte // actual data payload
}

// Update chunk layout
// Prefix:
// 2 bytes updateHeaderLength
// 2 bytes data length
const chunkPrefixLength = 2 + 2

// Header: (see updateHeader)
// Data:
// data (datalength bytes)
//
// Minimum size is Header + 1 (minimum data length, enforced)
const minimumUpdateDataLength = updateHeaderLength + 1
const maxUpdateDataLength = chunk.DefaultSize - signatureLength - updateHeaderLength - chunkPrefixLength

// binaryPut serializes the resource update information into the given slice
func (r *resourceUpdate) binaryPut(serializedData []byte) error {
	datalength := len(r.data)
	if datalength == 0 {
		return NewError(ErrInvalidValue, "cannot update a resource with no data")
	}

	if datalength > maxUpdateDataLength {
		return NewErrorf(ErrInvalidValue, "data is too big (length=%d). Max length=%d", datalength, maxUpdateDataLength)
	}

	if len(serializedData) != r.binaryLength() {
		return NewErrorf(ErrInvalidValue, "slice passed to putBinary must be of exact size. Expected %d bytes", r.binaryLength())
	}

	if r.multihash {
		if _, _, err := multihash.GetMultihashLength(r.data); err != nil {
			return NewError(ErrInvalidValue, "Invalid multihash")
		}
	}

	// Add prefix: updateHeaderLength and actual data length
	cursor := 0
	binary.LittleEndian.PutUint16(serializedData[cursor:], uint16(updateHeaderLength))
	cursor += 2

	// data length
	binary.LittleEndian.PutUint16(serializedData[cursor:], uint16(datalength))
	cursor += 2

	// serialize header (see updateHeader)
	if err := r.updateHeader.binaryPut(serializedData[cursor : cursor+updateHeaderLength]); err != nil {
		return err
	}
	cursor += updateHeaderLength

	// add the data
	copy(serializedData[cursor:], r.data)
	cursor += datalength

	return nil
}

// binaryLength returns the expected number of bytes this structure will take to encode
func (r *resourceUpdate) binaryLength() int {
	return chunkPrefixLength + updateHeaderLength + len(r.data)
}

// binaryGet populates this instance from the information contained in the passed byte slice
func (r *resourceUpdate) binaryGet(serializedData []byte) error {
	if len(serializedData) < minimumUpdateDataLength {
		return NewErrorf(ErrNothingToReturn, "chunk less than %d bytes cannot be a resource update chunk", minimumUpdateDataLength)
	}
	cursor := 0
	declaredHeaderlength := binary.LittleEndian.Uint16(serializedData[cursor : cursor+2])
	if declaredHeaderlength != updateHeaderLength {
		return NewErrorf(ErrCorruptData, "Invalid header length. Expected %d, got %d", updateHeaderLength, declaredHeaderlength)
	}

	cursor += 2
	datalength := int(binary.LittleEndian.Uint16(serializedData[cursor : cursor+2]))
	cursor += 2

	if chunkPrefixLength+updateHeaderLength+datalength+signatureLength != len(serializedData) {
		return NewError(ErrNothingToReturn, "length specified in header is different than actual chunk size")
	}

	// at this point we can be satisfied that we have the correct data length to read
	if err := r.updateHeader.binaryGet(serializedData[cursor : cursor+updateHeaderLength]); err != nil {
		return err
	}
	cursor += updateHeaderLength

	data := serializedData[cursor : cursor+datalength]
	cursor += datalength

	// if multihash content is indicated we check the validity of the multihash
	if r.updateHeader.multihash {
		mhLength, mhHeaderLength, err := multihash.GetMultihashLength(data)
		if err != nil {
			log.Error("multihash parse error", "err", err)
			return err
		}
		if datalength != mhLength+mhHeaderLength {
			log.Debug("multihash error", "datalength", datalength, "mhLength", mhLength, "mhHeaderLength", mhHeaderLength)
			return errors.New("Corrupt multihash data")
		}
	}

	// now that all checks have passed, copy data into structure
	r.data = make([]byte, datalength)
	copy(r.data, data)

	return nil

}

// Multihash specifies whether the resource data should be interpreted as multihash
func (r *resourceUpdate) Multihash() bool {
	return r.multihash
}
