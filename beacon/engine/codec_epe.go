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
	"encoding/json"
	"errors"
	"math/big"

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

func unmarshalBlobsBundle(input []byte) (*BlobsBundle, error) {
	if isJSONNull(input) {
		return nil, nil
	}
	var bundle BlobsBundle
	if err := decodeJSONObject(input, func(key string, value json.RawMessage) error {
		var err error
		switch key {
		case "commitments":
			bundle.Commitments, err = unmarshalHexBytesArray(value)
		case "proofs":
			bundle.Proofs, err = unmarshalHexBytesArray(value)
		case "blobs":
			bundle.Blobs, err = unmarshalHexBytesArray(value)
		}
		return err
	}); err != nil {
		return nil, err
	}
	return &bundle, nil
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

// UnmarshalJSON implements json.Unmarshaler.
func (e *ExecutionPayloadEnvelope) UnmarshalJSON(input []byte) error {
	var (
		payloadSeen    bool
		blockValueSeen bool
	)
	*e = ExecutionPayloadEnvelope{}
	if err := decodeJSONObject(input, func(key string, value json.RawMessage) error {
		switch key {
		case "executionPayload":
			payloadSeen = true
			if isJSONNull(value) {
				e.ExecutionPayload = nil
				return nil
			}
			var payload ExecutableData
			if err := payload.UnmarshalJSON(value); err != nil {
				return err
			}
			e.ExecutionPayload = &payload
		case "blockValue":
			blockValueSeen = true
			if isJSONNull(value) {
				e.BlockValue = nil
				return nil
			}
			var blockValue hexutil.Big
			if err := blockValue.UnmarshalJSON(value); err != nil {
				return err
			}
			e.BlockValue = (*big.Int)(&blockValue)
		case "blobsBundle":
			bundle, err := unmarshalBlobsBundle(value)
			if err != nil {
				return err
			}
			e.BlobsBundle = bundle
		case "executionRequests":
			requests, err := unmarshalHexBytesArray(value)
			if err != nil {
				return err
			}
			if requests == nil {
				e.Requests = nil
				return nil
			}
			e.Requests = make([][]byte, len(requests))
			for i, req := range requests {
				e.Requests[i] = req
			}
		case "shouldOverrideBuilder":
			if isJSONNull(value) {
				e.Override = false
				return nil
			}
			if err := json.Unmarshal(value, &e.Override); err != nil {
				return err
			}
		case "witness":
			if isJSONNull(value) {
				e.Witness = nil
				return nil
			}
			var witness hexutil.Bytes
			if err := witness.UnmarshalJSON(value); err != nil {
				return err
			}
			e.Witness = &witness
		}
		return nil
	}); err != nil {
		return err
	}
	if !payloadSeen || e.ExecutionPayload == nil {
		return errors.New("missing required field 'executionPayload' for ExecutionPayloadEnvelope")
	}
	if !blockValueSeen || e.BlockValue == nil {
		return errors.New("missing required field 'blockValue' for ExecutionPayloadEnvelope")
	}
	return nil
}
