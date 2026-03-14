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

import (
	"encoding/hex"
	"encoding/json"
	"errors"
	"slices"

	"github.com/ethereum/go-ethereum/common/hexutil"
)

// estimateBlobsBundleSize returns a rough estimate of the JSON size for a BlobsBundle.
func estimateBlobsBundleSize(b *BlobsBundle) int {
	size := 80 // JSON structure overhead
	for _, blob := range b.Blobs {
		size += len(blob)*2 + 6
	}
	for _, c := range b.Commitments {
		size += len(c)*2 + 6
	}
	for _, p := range b.Proofs {
		size += len(p)*2 + 6
	}
	return size
}

// marshalBlobsBundle writes BlobsBundle as JSON and appends it to buf.
func marshalBlobsBundle(buf []byte, b *BlobsBundle) []byte {
	buf = append(buf, `{"commitments":`...)
	buf = marshalHexBytesArray(buf, b.Commitments)

	buf = append(buf, `,"proofs":`...)
	buf = marshalHexBytesArray(buf, b.Proofs)

	buf = append(buf, `,"blobs":`...)
	buf = marshalHexBytesArray(buf, b.Blobs)

	buf = append(buf, '}')
	return buf
}

// marshalHexBytesArray writes an array of hex-encoded byte slices to buf.
// A nil slice is written as "null" to match encoding/json semantics.
func marshalHexBytesArray(buf []byte, items []hexutil.Bytes) []byte {
	if items == nil {
		return append(buf, "null"...)
	}
	buf = append(buf, '[')
	for i, item := range items {
		if i > 0 {
			buf = append(buf, ',')
		}
		buf = writeHexBytes(buf, item)
	}
	buf = append(buf, ']')
	return buf
}

// writeHexBytes writes a hex-encoded byte slice as a JSON string ("0x...") to buf.
func writeHexBytes(buf []byte, data []byte) []byte {
	buf = append(buf, '"', '0', 'x')
	buf = slices.Grow(buf, len(data)*2+1)
	cur := len(buf)
	buf = buf[:cur+len(data)*2]
	hex.Encode(buf[cur:], data)
	buf = append(buf, '"')
	return buf
}

// MarshalJSON implements json.Marshaler.
func (e ExecutionPayloadEnvelope) MarshalJSON() ([]byte, error) {
	if e.ExecutionPayload == nil {
		return nil, errors.New("missing required field 'executionPayload' for ExecutionPayloadEnvelope")
	}

	// Marshal the execution payload using its gencodec MarshalJSON.
	payload, err := e.ExecutionPayload.MarshalJSON()
	if err != nil {
		return nil, err
	}

	// Marshal the block value.
	blockValue, err := json.Marshal((*hexutil.Big)(e.BlockValue))
	if err != nil {
		return nil, err
	}

	// Marshal the execution requests.
	var requests []byte
	if e.Requests != nil {
		hexRequests := make([]hexutil.Bytes, len(e.Requests))
		for i, req := range e.Requests {
			hexRequests[i] = req
		}
		requests, err = json.Marshal(hexRequests)
		if err != nil {
			return nil, err
		}
	}

	// Marshal the override.
	override, err := json.Marshal(e.Override)
	if err != nil {
		return nil, err
	}

	// Marshal the witness.
	var witness []byte
	if e.Witness != nil {
		witness, err = json.Marshal(e.Witness)
		if err != nil {
			return nil, err
		}
	}

	// Estimate buffer size.
	size := len(payload) + len(blockValue) + len(requests) + len(override) + len(witness)
	if e.BlobsBundle != nil {
		size += estimateBlobsBundleSize(e.BlobsBundle)
	}
	size += 256 // JSON bloat (keys, braces, commas, etc. and room for growth)
	buf := make([]byte, 0, size)

	// Write the execution payload to the buffer
	buf = append(buf, `{"executionPayload":`...)
	buf = append(buf, payload...)

	// Write the block value to the buffer
	buf = append(buf, `,"blockValue":`...)
	buf = append(buf, blockValue...)

	// Write the blobs bundle to the buffer
	buf = append(buf, `,"blobsBundle":`...)
	if e.BlobsBundle != nil {
		buf = marshalBlobsBundle(buf, e.BlobsBundle)
	} else {
		buf = append(buf, "null"...)
	}

	// Write the execution requests to the buffer
	buf = append(buf, `,"executionRequests":`...)
	if requests != nil {
		buf = append(buf, requests...)
	} else {
		buf = append(buf, "null"...)
	}

	// Write the override to the buffer
	buf = append(buf, `,"shouldOverrideBuilder":`...)
	buf = append(buf, override...)

	// Write the witness to the buffer if present
	if witness != nil {
		buf = append(buf, `,"witness":`...)
		buf = append(buf, witness...)
	}

	// Close the envelope
	buf = append(buf, '}')
	return buf, nil
}
