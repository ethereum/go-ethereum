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

package bitutil

/*
The compression algorithm implemented by CompressBytes and DecompressBytes is
optimized for "sparse" input data which contains a lot of zero bytes. Decompression
requires knowledge of the decompressed data length. Compression works as follows:

if data only contains zeroes,
  CompressBytes(data) == nil
otherwise if len(data) <= 1,
 CompressBytes(data) == data
otherwise:
 CompressBytes(data) == append(CompressBytes(nonZeroBits(data)), nonZeroBytes(data)...)
where
 nonZeroBits(data) is a bit vector with len(data) bits (MSB first):
  nonZeroBits(data)[i/8] && (1 << (7-i%8)) != 0  if data[i] != 0
  len(nonZeroBits(data)) == (len(data)+7)/8
 nonZeroBytes(data) contains the non-zero bytes of data in the same order
*/

// CompressBytes compresses the input byte slice
func CompressBytes(data []byte) []byte {
	if len(data) == 0 {
		return nil
	}
	if len(data) == 1 {
		if data[0] == 0 {
			return nil
		} else {
			return data
		}
	}

	bitsLen := (len(data) + 7) / 8
	nonZeroBits := make([]byte, bitsLen)
	nonZeroBytes := make([]byte, 0, len(data))
	for i, b := range data {
		if b != 0 {
			nonZeroBytes = append(nonZeroBytes, b)
			nonZeroBits[i/8] |= 1 << byte(7-i%8)
		}
	}
	if len(nonZeroBytes) == 0 {
		return nil
	}
	return append(CompressBytes(nonZeroBits), nonZeroBytes...)
}

// DecompressBytes decompresses data with a known target size.
// In addition to the decompressed output, the function returns the length of
// compressed input data corresponding to the output. The input slice may be longer.
// If the input slice is too short, (nil, -1) is returned.
func DecompressBytes(data []byte, targetLen int) ([]byte, int) {
	decomp := make([]byte, targetLen)
	if len(data) == 0 {
		return decomp, 0
	}
	if targetLen == 1 {
		return data[0:1], 1
	}

	bitsLen := (targetLen + 7) / 8
	nonZeroBits, ptr := DecompressBytes(data, bitsLen)
	if ptr < 0 {
		return nil, -1
	}
	for i, _ := range decomp {
		if nonZeroBits[i/8]&(1<<byte(7-i%8)) != 0 {
			if ptr == len(data) {
				return nil, -1
			}
			decomp[i] = data[ptr]
			ptr++
		}
	}
	return decomp, ptr
}
