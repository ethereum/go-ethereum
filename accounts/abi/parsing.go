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

package abi

import (
	"encoding/binary"
	"fmt"
)

// separates byte slice into a slice of 32 byte slices.
func chunkBytes(output []byte) (chunked [][32]byte) {
	for i, j := 0, 0; i < len(output); i, j = i+32, j+1 {
		copy(chunked[j][:], output[i:i+32])
	}
	return
}

// check to see that the bytes are correctly formatted to 32 byte slices
func bytesAreProper(output []byte) error {
	if len(output)%32 != 0 {
		return fmt.Errorf("abi: improper length of byte slice detected in output.")
	}
	return nil
}

// interprets a 32 byte slice as an offset and then determines which indice to look to decode the type.
func offsetPointsTo(offset [32]byte) uint {
	return uint(binary.BigEndian.Uint64(offset[24:32])) / 32
}

// gives the starting and ending indices in the chunked byte slice for the values of an array
func parseSliceSize(size [32]byte, currentLength uint) (start uint, end uint) {
	return currentLength + 1, currentLength + uint(binary.BigEndian.Uint64(size[24:32])/32)
}
