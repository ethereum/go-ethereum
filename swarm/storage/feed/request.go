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

package feed

import (
	"bytes"
	"encoding/json"
	"hash"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/swarm/storage"
	"github.com/ethereum/go-ethereum/swarm/storage/feed/lookup"
)

// Request represents a request to sign or signed feed update message
type Request struct {
	Update     // actual content that will be put on the chunk, less signature
	Signature  *Signature
	idAddr     storage.Address // cached chunk address for the update (not serialized, for internal use)
	binaryData []byte          // cached serialized data (does not get serialized again!, for efficiency/internal use)
}

// updateRequestJSON represents a JSON-serialized UpdateRequest
type updateRequestJSON struct {
	ID
	ProtocolVersion uint8  `json:"protocolVersion"`
	Data            string `json:"data,omitempty"`
	Signature       string `json:"signature,omitempty"`
}

// Request layout
// Update bytes
// SignatureLength bytes
const minimumSignedUpdateLength = minimumUpdateDataLength + signatureLength

// NewFirstRequest returns a ready to sign request to publish a first feed update
func NewFirstRequest(topic Topic) *Request {

	request := new(Request)

	// get the current time
	now := TimestampProvider.Now().Time
	request.Epoch = lookup.GetFirstEpoch(now)
	request.Feed.Topic = topic
	request.Header.Version = ProtocolVersion

	return request
}

// SetData stores the payload data the feed update will be updated with
func (r *Request) SetData(data []byte) {
	r.data = data
	r.Signature = nil
}

// IsUpdate returns true if this request models a signed update or otherwise it is a signature request
func (r *Request) IsUpdate() bool {
	return r.Signature != nil
}

// Verify checks that signatures are valid
func (r *Request) Verify() (err error) {
	if len(r.data) == 0 {
		return NewError(ErrInvalidValue, "Update does not contain data")
	}
	if r.Signature == nil {
		return NewError(ErrInvalidSignature, "Missing signature field")
	}

	digest, err := r.GetDigest()
	if err != nil {
		return err
	}

	// get the address of the signer (which also checks that it's a valid signature)
	r.Feed.User, err = getUserAddr(digest, *r.Signature)
	if err != nil {
		return err
	}

	// check that the lookup information contained in the chunk matches the updateAddr (chunk search key)
	// that was used to retrieve this chunk
	// if this validation fails, someone forged a chunk.
	if !bytes.Equal(r.idAddr, r.Addr()) {
		return NewError(ErrInvalidSignature, "Signature address does not match with update user address")
	}

	return nil
}

// Sign executes the signature to validate the update message
func (r *Request) Sign(signer Signer) error {
	r.Feed.User = signer.Address()
	r.binaryData = nil           //invalidate serialized data
	digest, err := r.GetDigest() // computes digest and serializes into .binaryData
	if err != nil {
		return err
	}

	signature, err := signer.Sign(digest)
	if err != nil {
		return err
	}

	// Although the Signer interface returns the public address of the signer,
	// recover it from the signature to see if they match
	userAddr, err := getUserAddr(digest, signature)
	if err != nil {
		return NewError(ErrInvalidSignature, "Error verifying signature")
	}

	if userAddr != signer.Address() { // sanity check to make sure the Signer is declaring the same address used to sign!
		return NewError(ErrInvalidSignature, "Signer address does not match update user address")
	}

	r.Signature = &signature
	r.idAddr = r.Addr()
	return nil
}

// GetDigest creates the feed update digest used in signatures
// the serialized payload is cached in .binaryData
func (r *Request) GetDigest() (result common.Hash, err error) {
	hasher := hashPool.Get().(hash.Hash)
	defer hashPool.Put(hasher)
	hasher.Reset()
	dataLength := r.Update.binaryLength()
	if r.binaryData == nil {
		r.binaryData = make([]byte, dataLength+signatureLength)
		if err := r.Update.binaryPut(r.binaryData[:dataLength]); err != nil {
			return result, err
		}
	}
	hasher.Write(r.binaryData[:dataLength]) //everything except the signature.

	return common.BytesToHash(hasher.Sum(nil)), nil
}

// create an update chunk.
func (r *Request) toChunk() (storage.Chunk, error) {

	// Check that the update is signed and serialized
	// For efficiency, data is serialized during signature and cached in
	// the binaryData field when computing the signature digest in .getDigest()
	if r.Signature == nil || r.binaryData == nil {
		return nil, NewError(ErrInvalidSignature, "toChunk called without a valid signature or payload data. Call .Sign() first.")
	}

	updateLength := r.Update.binaryLength()

	// signature is the last item in the chunk data
	copy(r.binaryData[updateLength:], r.Signature[:])

	chunk := storage.NewChunk(r.idAddr, r.binaryData)
	return chunk, nil
}

// fromChunk populates this structure from chunk data. It does not verify the signature is valid.
func (r *Request) fromChunk(updateAddr storage.Address, chunkdata []byte) error {
	// for update chunk layout see Request definition

	//deserialize the feed update portion
	if err := r.Update.binaryGet(chunkdata[:len(chunkdata)-signatureLength]); err != nil {
		return err
	}

	// Extract the signature
	var signature *Signature
	cursor := r.Update.binaryLength()
	sigdata := chunkdata[cursor : cursor+signatureLength]
	if len(sigdata) > 0 {
		signature = &Signature{}
		copy(signature[:], sigdata)
	}

	r.Signature = signature
	r.idAddr = updateAddr
	r.binaryData = chunkdata

	return nil

}

// FromValues deserializes this instance from a string key-value store
// useful to parse query strings
func (r *Request) FromValues(values Values, data []byte) error {
	signatureBytes, err := hexutil.Decode(values.Get("signature"))
	if err != nil {
		r.Signature = nil
	} else {
		if len(signatureBytes) != signatureLength {
			return NewError(ErrInvalidSignature, "Incorrect signature length")
		}
		r.Signature = new(Signature)
		copy(r.Signature[:], signatureBytes)
	}
	err = r.Update.FromValues(values, data)
	if err != nil {
		return err
	}
	r.idAddr = r.Addr()
	return err
}

// AppendValues serializes this structure into the provided string key-value store
// useful to build query strings
func (r *Request) AppendValues(values Values) []byte {
	if r.Signature != nil {
		values.Set("signature", hexutil.Encode(r.Signature[:]))
	}
	return r.Update.AppendValues(values)
}

// fromJSON takes an update request JSON and populates an UpdateRequest
func (r *Request) fromJSON(j *updateRequestJSON) error {

	r.ID = j.ID
	r.Header.Version = j.ProtocolVersion

	var err error
	if j.Data != "" {
		r.data, err = hexutil.Decode(j.Data)
		if err != nil {
			return NewError(ErrInvalidValue, "Cannot decode data")
		}
	}

	if j.Signature != "" {
		sigBytes, err := hexutil.Decode(j.Signature)
		if err != nil || len(sigBytes) != signatureLength {
			return NewError(ErrInvalidSignature, "Cannot decode signature")
		}
		r.Signature = new(Signature)
		r.idAddr = r.Addr()
		copy(r.Signature[:], sigBytes)
	}
	return nil
}

// UnmarshalJSON takes a JSON structure stored in a byte array and populates the Request object
// Implements json.Unmarshaler interface
func (r *Request) UnmarshalJSON(rawData []byte) error {
	var requestJSON updateRequestJSON
	if err := json.Unmarshal(rawData, &requestJSON); err != nil {
		return err
	}
	return r.fromJSON(&requestJSON)
}

// MarshalJSON takes an update request and encodes it as a JSON structure into a byte array
// Implements json.Marshaler interface
func (r *Request) MarshalJSON() (rawData []byte, err error) {
	var signatureString, dataString string
	if r.Signature != nil {
		signatureString = hexutil.Encode(r.Signature[:])
	}
	if r.data != nil {
		dataString = hexutil.Encode(r.data)
	}

	requestJSON := &updateRequestJSON{
		ID:              r.ID,
		ProtocolVersion: r.Header.Version,
		Data:            dataString,
		Signature:       signatureString,
	}

	return json.Marshal(requestJSON)
}
