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

	"github.com/ethereum/go-ethereum/common/hexutil"
	jsonw "github.com/fjl/jsonw"
)

// marshalBlobsBundle writes BlobsBundle as JSON and appends it to buf.
func marshalBlobsBundle(b *jsonw.Buffer, bundle *BlobsBundle) {
	if bundle == nil {
		b.Null()
		return
	}
	b.Object(func() {
		b.Key("commitments")
		appendHexBytesArray(b, bundle.Commitments)
		b.Key("proofs")
		appendHexBytesArray(b, bundle.Proofs)
		b.Key("blobs")
		appendHexBytesArray(b, bundle.Blobs)
	})
}

// MarshalJSON implements json.Marshaler.
func (e ExecutionPayloadEnvelope) MarshalJSON() ([]byte, error) {
	if e.ExecutionPayload == nil {
		return nil, errors.New("missing required field 'executionPayload' for ExecutionPayloadEnvelope")
	}

	// Pre-marshal the execution payload using its gencodec MarshalJSON.
	payload, err := e.ExecutionPayload.MarshalJSON()
	if err != nil {
		return nil, err
	}
	// Pre-marshal the witness.
	var witness []byte
	if e.Witness != nil {
		witness, err = json.Marshal(e.Witness)
		if err != nil {
			return nil, err
		}
	}

	// Write the execution payload to the buffer
	var b jsonw.Buffer
	b.Object(func() {
		b.Key("executionPayload")
		b.RawValue(payload)
		b.Key("blockValue")
		b.MustValue((*hexutil.Big)(e.BlockValue))
		b.Key("blobsBundle")
		marshalBlobsBundle(&b, e.BlobsBundle)
		b.Key("executionRequests")
		if e.Requests == nil {
			b.Null()
		} else {
			appendHexBytesArray(&b, e.Requests)
		}
		b.Key("shouldOverrideBuilder")
		b.Bool(e.Override)
		if e.Witness != nil {
			b.Key("witness")
			b.RawValue(witness)
		}
	})
	return b.Output(), nil
}

func appendHexBytesArray[T ~[]byte](b *jsonw.Buffer, slice []T) {
	b.Array(func() {
		for _, elem := range slice {
			b.HexBytes(elem)
		}
	})
}
