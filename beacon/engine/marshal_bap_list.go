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

package engine

// estimateBlobAndProofV1Size returns a rough estimate of the JSON size for a BlobAndProofV1.
func estimateBlobAndProofV1Size(item *BlobAndProofV1) int {
	if item == nil {
		return 4
	}
	return len(item.Blob)*2 + len(item.Proof)*2 + 30
}

// marshalBlobAndProofV1 writes a BlobAndProofV1 as JSON and appends it to buf.
func marshalBlobAndProofV1(buf []byte, item *BlobAndProofV1) []byte {
	if item == nil {
		return append(buf, "null"...)
	}
	buf = append(buf, `{"blob":`...)
	buf = writeHexBytes(buf, item.Blob)

	buf = append(buf, `,"proof":`...)
	buf = writeHexBytes(buf, item.Proof)

	buf = append(buf, '}')
	return buf
}

// estimateBlobAndProofV2Size returns a rough estimate of the JSON size for a BlobAndProofV2.
func estimateBlobAndProofV2Size(item *BlobAndProofV2) int {
	if item == nil {
		return 4
	}
	size := len(item.Blob)*2 + 30
	for _, proof := range item.CellProofs {
		size += len(proof)*2 + 6
	}
	return size
}

// marshalBlobAndProofV2 writes a BlobAndProofV2 as JSON and appends it to buf.
func marshalBlobAndProofV2(buf []byte, item *BlobAndProofV2) []byte {
	if item == nil {
		return append(buf, "null"...)
	}
	buf = append(buf, `{"blob":`...)
	buf = writeHexBytes(buf, item.Blob)

	buf = append(buf, `,"proofs":`...)
	buf = marshalHexBytesArray(buf, item.CellProofs)

	buf = append(buf, '}')
	return buf
}

// MarshalJSON implements json.Marshaler.
func (list BlobAndProofListV1) MarshalJSON() ([]byte, error) {
	// Estimate buffer size.
	size := 2
	for _, item := range list {
		size += estimateBlobAndProofV1Size(item) + 1
	}
	buf := make([]byte, 0, size)

	// Write the array elements to the buffer.
	buf = append(buf, '[')
	for i, item := range list {
		if i > 0 {
			buf = append(buf, ',')
		}
		buf = marshalBlobAndProofV1(buf, item)
	}
	buf = append(buf, ']')
	return buf, nil
}

// MarshalJSON implements json.Marshaler.
func (list BlobAndProofListV2) MarshalJSON() ([]byte, error) {
	// Estimate buffer size.
	size := 2
	for _, item := range list {
		size += estimateBlobAndProofV2Size(item) + 1
	}
	buf := make([]byte, 0, size)

	// Write the array elements to the buffer.
	buf = append(buf, '[')
	for i, item := range list {
		if i > 0 {
			buf = append(buf, ',')
		}
		buf = marshalBlobAndProofV2(buf, item)
	}
	buf = append(buf, ']')
	return buf, nil
}
