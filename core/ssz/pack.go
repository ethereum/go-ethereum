// Copyright 2026 The go-ethereum Authors
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

package ssz

// BytesPerChunk is the size of a Merkle tree leaf chunk.
const BytesPerChunk = 32

// Pack splits data into 32-byte chunks, the leaves of a Merkle tree, filling
// the unused tail of the last chunk with zeroes. It is the pack() helper of
// the SSZ spec: data is the serialization of one or more basic values, which
// is just their bytes concatenated, so 40 bytes become two chunks with the
// second one zero-padded.
func Pack(data []byte) [][32]byte {
	chunks := make([][32]byte, (len(data)+BytesPerChunk-1)/BytesPerChunk)
	for i := range chunks {
		copy(chunks[i][:], data[i*BytesPerChunk:])
	}
	return chunks
}
