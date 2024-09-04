// Copyright 2024 The go-ethereum Authors
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

package types

import (
	"bytes"
	"errors"
	"fmt"
	"io"

	"github.com/ethereum/go-ethereum/rlp"
)

var (
	ErrRequestTypeNotSupported = errors.New("request type not supported")
	errShortTypedRequest       = errors.New("typed request too short")
)

// Request types.
const (
	DepositRequestType = 0x00
)

// Request is an EIP-7685 request object. It represents execution layer
// triggered messages bound for the consensus layer.
type Request struct {
	inner RequestData
}

// Type returns the EIP-7685 type of the request.
func (r *Request) Type() byte {
	return r.inner.requestType()
}

// Inner returns the inner request data.
func (r *Request) Inner() RequestData {
	return r.inner
}

// NewRequest creates a new request.
func NewRequest(inner RequestData) *Request {
	req := new(Request)
	req.inner = inner.copy()
	return req
}

// Requests implements DerivableList for requests.
type Requests []*Request

// Len returns the length of s.
func (s Requests) Len() int { return len(s) }

// EncodeIndex encodes the i'th request to s.
func (s Requests) EncodeIndex(i int, w *bytes.Buffer) {
	s[i].encode(w)
}

// RequestData is the underlying data of a request.
type RequestData interface {
	requestType() byte
	encode(*bytes.Buffer) error
	decode([]byte) error
	copy() RequestData // creates a deep copy and initializes all fields
}

// EncodeRLP implements rlp.Encoder
func (r *Request) EncodeRLP(w io.Writer) error {
	buf := encodeBufferPool.Get().(*bytes.Buffer)
	defer encodeBufferPool.Put(buf)
	buf.Reset()
	if err := r.encode(buf); err != nil {
		return err
	}
	return rlp.Encode(w, buf.Bytes())
}

// encode writes the canonical encoding of a request to w.
func (r *Request) encode(w *bytes.Buffer) error {
	w.WriteByte(r.Type())
	return r.inner.encode(w)
}

// MarshalBinary returns the canonical encoding of the request.
func (r *Request) MarshalBinary() ([]byte, error) {
	var buf bytes.Buffer
	err := r.encode(&buf)
	return buf.Bytes(), err
}

// DecodeRLP implements rlp.Decoder
func (r *Request) DecodeRLP(s *rlp.Stream) error {
	kind, size, err := s.Kind()
	switch {
	case err != nil:
		return err
	case kind == rlp.List:
		return fmt.Errorf("untyped request")
	case kind == rlp.Byte:
		return errShortTypedRequest
	default:
		// First read the request payload bytes into a temporary buffer.
		b, buf, err := getPooledBuffer(size)
		if err != nil {
			return err
		}
		defer encodeBufferPool.Put(buf)
		if err := s.ReadBytes(b); err != nil {
			return err
		}
		// Now decode the inner request.
		inner, err := r.decode(b)
		if err == nil {
			r.inner = inner
		}
		return err
	}
}

// UnmarshalBinary decodes the canonical encoding of requests.
func (r *Request) UnmarshalBinary(b []byte) error {
	inner, err := r.decode(b)
	if err != nil {
		return err
	}
	r.inner = inner
	return nil
}

// decode decodes a request from the canonical format.
func (r *Request) decode(b []byte) (RequestData, error) {
	if len(b) <= 1 {
		return nil, errShortTypedRequest
	}
	var inner RequestData
	switch b[0] {
	case DepositRequestType:
		inner = new(Deposit)
	default:
		return nil, ErrRequestTypeNotSupported
	}
	err := inner.decode(b[1:])
	return inner, err
}
