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
	"bytes"
	"encoding/json"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/swarm/storage"
)

// updateRequestJSON represents a JSON-serialized UpdateRequest
type updateRequestJSON struct {
	Name      string `json:"name,omitempty"`
	Frequency uint64 `json:"frequency,omitempty"`
	StartTime uint64 `json:"startTime,omitempty"`
	Owner     string `json:"ownerAddr,omitempty"`
	RootAddr  string `json:"rootAddr,omitempty"`
	MetaHash  string `json:"metaHash,omitempty"`
	Version   uint32 `json:"version,omitempty"`
	Period    uint32 `json:"period,omitempty"`
	Data      string `json:"data,omitempty"`
	Multihash bool   `json:"multiHash"`
	Signature string `json:"signature,omitempty"`
}

// Request represents an update and/or resource create message
type Request struct {
	SignedResourceUpdate
	metadata ResourceMetadata
	isNew    bool
}

var zeroAddr = common.Address{}

// NewCreateUpdateRequest returns a ready to sign request to create and initialize a resource with data
func NewCreateUpdateRequest(metadata *ResourceMetadata) (*Request, error) {

	request, err := NewCreateRequest(metadata)
	if err != nil {
		return nil, err
	}

	// get the current time
	now := TimestampProvider.Now().Time

	request.version = 1
	request.period, err = getNextPeriod(metadata.StartTime.Time, now, metadata.Frequency)
	if err != nil {
		return nil, err
	}
	return request, nil
}

// NewCreateRequest returns a request to create a new resource
func NewCreateRequest(metadata *ResourceMetadata) (request *Request, err error) {
	if metadata.StartTime.Time == 0 { // get the current time
		metadata.StartTime = TimestampProvider.Now()
	}

	if metadata.Owner == zeroAddr {
		return nil, NewError(ErrInvalidValue, "OwnerAddr is not set")
	}

	request = &Request{
		metadata: *metadata,
	}
	request.rootAddr, request.metaHash, _, err = request.metadata.serializeAndHash()
	request.isNew = true
	return request, nil
}

// Frequency returns the resource's expected update frequency
func (r *Request) Frequency() uint64 {
	return r.metadata.Frequency
}

// Name returns the resource human-readable name
func (r *Request) Name() string {
	return r.metadata.Name
}

// Multihash returns true if the resource data should be interpreted as a multihash
func (r *Request) Multihash() bool {
	return r.multihash
}

// Period returns in which period the resource will be published
func (r *Request) Period() uint32 {
	return r.period
}

// Version returns the resource version to publish
func (r *Request) Version() uint32 {
	return r.version
}

// RootAddr returns the metadata chunk address
func (r *Request) RootAddr() storage.Address {
	return r.rootAddr
}

// StartTime returns the time that the resource was/will be created at
func (r *Request) StartTime() Timestamp {
	return r.metadata.StartTime
}

// Owner returns the resource owner's address
func (r *Request) Owner() common.Address {
	return r.metadata.Owner
}

// Sign executes the signature to validate the resource and sets the owner address field
func (r *Request) Sign(signer Signer) error {
	if r.metadata.Owner != zeroAddr && r.metadata.Owner != signer.Address() {
		return NewError(ErrInvalidSignature, "Signer does not match current owner of the resource")
	}

	if err := r.SignedResourceUpdate.Sign(signer); err != nil {
		return err
	}
	r.metadata.Owner = signer.Address()
	return nil
}

// SetData stores the payload data the resource will be updated with
func (r *Request) SetData(data []byte, multihash bool) {
	r.data = data
	r.multihash = multihash
	r.signature = nil
	if !r.isNew {
		r.metadata.Frequency = 0 // mark as update
	}
}

func (r *Request) IsNew() bool {
	return r.metadata.Frequency > 0 && (r.period <= 1 || r.version <= 1)
}

func (r *Request) IsUpdate() bool {
	return r.signature != nil
}

// fromJSON takes an update request JSON and populates an UpdateRequest
func (r *Request) fromJSON(j *updateRequestJSON) error {

	r.version = j.Version
	r.period = j.Period
	r.multihash = j.Multihash
	r.metadata.Name = j.Name
	r.metadata.Frequency = j.Frequency
	r.metadata.StartTime.Time = j.StartTime

	if err := decodeHexArray(r.metadata.Owner[:], j.Owner, "ownerAddr"); err != nil {
		return err
	}

	var err error
	if j.Data != "" {
		r.data, err = hexutil.Decode(j.Data)
		if err != nil {
			return NewError(ErrInvalidValue, "Cannot decode data")
		}
	}

	var declaredRootAddr storage.Address
	var declaredMetaHash []byte

	declaredRootAddr, err = decodeHexSlice(j.RootAddr, storage.AddressLength, "rootAddr")
	if err != nil {
		return err
	}
	declaredMetaHash, err = decodeHexSlice(j.MetaHash, 32, "metaHash")
	if err != nil {
		return err
	}

	if r.IsNew() {
		// for new resource creation, rootAddr and metaHash are optional because
		// we can derive them from the content itself.
		// however, if the user sent them, we check them for consistency.

		r.rootAddr, r.metaHash, _, err = r.metadata.serializeAndHash()
		if err != nil {
			return err
		}
		if j.RootAddr != "" && !bytes.Equal(declaredRootAddr, r.rootAddr) {
			return NewError(ErrInvalidValue, "rootAddr does not match resource metadata")
		}
		if j.MetaHash != "" && !bytes.Equal(declaredMetaHash, r.metaHash) {
			return NewError(ErrInvalidValue, "metaHash does not match resource metadata")
		}

	} else {
		//Update message
		r.rootAddr = declaredRootAddr
		r.metaHash = declaredMetaHash
	}

	if j.Signature != "" {
		sigBytes, err := hexutil.Decode(j.Signature)
		if err != nil || len(sigBytes) != signatureLength {
			return NewError(ErrInvalidSignature, "Cannot decode signature")
		}
		r.signature = new(Signature)
		r.updateAddr = r.UpdateAddr()
		copy(r.signature[:], sigBytes)
	}
	return nil
}

func decodeHexArray(dst []byte, src, name string) error {
	bytes, err := decodeHexSlice(src, len(dst), name)
	if err != nil {
		return err
	}
	if bytes != nil {
		copy(dst, bytes)
	}
	return nil
}

func decodeHexSlice(src string, expectedLength int, name string) (bytes []byte, err error) {
	if src != "" {
		bytes, err = hexutil.Decode(src)
		if err != nil || len(bytes) != expectedLength {
			return nil, NewErrorf(ErrInvalidValue, "Cannot decode %s", name)
		}
	}
	return bytes, nil
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
	var signatureString, dataHashString, rootAddrString, metaHashString string
	if r.signature != nil {
		signatureString = hexutil.Encode(r.signature[:])
	}
	if r.data != nil {
		dataHashString = hexutil.Encode(r.data)
	}
	if r.rootAddr != nil {
		rootAddrString = hexutil.Encode(r.rootAddr)
	}
	if r.metaHash != nil {
		metaHashString = hexutil.Encode(r.metaHash)
	}
	var ownerAddrString string
	if r.metadata.Frequency == 0 {
		ownerAddrString = ""
	} else {
		ownerAddrString = hexutil.Encode(r.metadata.Owner[:])
	}

	requestJSON := &updateRequestJSON{
		Name:      r.metadata.Name,
		Frequency: r.metadata.Frequency,
		StartTime: r.metadata.StartTime.Time,
		Version:   r.version,
		Period:    r.period,
		Owner:     ownerAddrString,
		Data:      dataHashString,
		Multihash: r.multihash,
		Signature: signatureString,
		RootAddr:  rootAddrString,
		MetaHash:  metaHashString,
	}

	return json.Marshal(requestJSON)
}
