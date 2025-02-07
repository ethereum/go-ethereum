// Copyright 2024-2025 the libevm authors.
//
// The libevm additions to go-ethereum are free software: you can redistribute
// them and/or modify them under the terms of the GNU Lesser General Public License
// as published by the Free Software Foundation, either version 3 of the License,
// or (at your option) any later version.
//
// The libevm additions are distributed in the hope that they will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the GNU Lesser
// General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see
// <http://www.gnu.org/licenses/>.

package types

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/ava-labs/libevm/libevm/pseudo"
	"github.com/ava-labs/libevm/libevm/testonly"
	"github.com/ava-labs/libevm/rlp"
)

// HeaderHooks are required for all types registered with [RegisterExtras] for
// [Header] payloads.
type HeaderHooks interface {
	MarshalJSON(*Header) ([]byte, error) //nolint:govet // Type-specific override hook
	UnmarshalJSON(*Header, []byte) error //nolint:govet
	EncodeRLP(*Header, io.Writer) error
	DecodeRLP(*Header, *rlp.Stream) error
	PostCopy(dst *Header)
}

// hooks returns the Header's registered HeaderHooks, if any, otherwise a
// [NOOPHeaderHooks] suitable for running default behaviour.
func (h *Header) hooks() HeaderHooks {
	if r := registeredExtras; r.Registered() {
		return r.Get().hooks.hooksFromHeader(h)
	}
	return new(NOOPHeaderHooks)
}

func (e ExtraPayloads[HPtr, SA]) hooksFromHeader(h *Header) HeaderHooks {
	return e.Header.Get(h)
}

var _ interface {
	rlp.Encoder
	rlp.Decoder
	json.Marshaler
	json.Unmarshaler
} = (*Header)(nil)

// MarshalJSON implements the [json.Marshaler] interface.
func (h *Header) MarshalJSON() ([]byte, error) {
	return h.hooks().MarshalJSON(h)
}

// UnmarshalJSON implements the [json.Unmarshaler] interface.
func (h *Header) UnmarshalJSON(b []byte) error {
	return h.hooks().UnmarshalJSON(h, b)
}

// EncodeRLP implements the [rlp.Encoder] interface.
func (h *Header) EncodeRLP(w io.Writer) error {
	return h.hooks().EncodeRLP(h, w)
}

// DecodeRLP implements the [rlp.Decoder] interface.
func (h *Header) DecodeRLP(s *rlp.Stream) error {
	return h.hooks().DecodeRLP(h, s)
}

func (h *Header) extraPayload() *pseudo.Type {
	r := registeredExtras
	if !r.Registered() {
		// See params.ChainConfig.extraPayload() for panic rationale.
		panic(fmt.Sprintf("%T.extraPayload() called before RegisterExtras()", r))
	}
	if h.extra == nil {
		h.extra = r.Get().newHeader()
	}
	return h.extra
}

// NOOPHeaderHooks implements [HeaderHooks] such that they are equivalent to
// no type having been registered.
type NOOPHeaderHooks struct{}

var _ HeaderHooks = (*NOOPHeaderHooks)(nil)

func (*NOOPHeaderHooks) MarshalJSON(h *Header) ([]byte, error) { //nolint:govet
	return h.marshalJSON()
}

func (*NOOPHeaderHooks) UnmarshalJSON(h *Header, b []byte) error { //nolint:govet
	return h.unmarshalJSON(b)
}

func (*NOOPHeaderHooks) EncodeRLP(h *Header, w io.Writer) error {
	return h.encodeRLP(w)
}

func (*NOOPHeaderHooks) DecodeRLP(h *Header, s *rlp.Stream) error {
	type withoutMethods Header
	return s.Decode((*withoutMethods)(h))
}
func (*NOOPHeaderHooks) PostCopy(dst *Header) {}

var _ interface {
	rlp.Encoder
	rlp.Decoder
} = (*Body)(nil)

// EncodeRLP implements the [rlp.Encoder] interface.
func (b *Body) EncodeRLP(w io.Writer) error {
	return b.hooks().RLPFieldsForEncoding(b).EncodeRLP(w)
}

// DecodeRLP implements the [rlp.Decoder] interface.
func (b *Body) DecodeRLP(s *rlp.Stream) error {
	return b.hooks().RLPFieldPointersForDecoding(b).DecodeRLP(s)
}

// BodyHooks are required for all types registered with [RegisterExtras] for
// [Body] payloads.
type BodyHooks interface {
	RLPFieldsForEncoding(*Body) *rlp.Fields
	RLPFieldPointersForDecoding(*Body) *rlp.Fields
}

// TestOnlyRegisterBodyHooks is a temporary means of "registering" BodyHooks for
// the purpose of testing. It will panic if called outside of a test.
func TestOnlyRegisterBodyHooks(h BodyHooks) {
	testonly.OrPanic(func() {
		todoRegisteredBodyHooks = h
	})
}

// todoRegisteredBodyHooks is a temporary placeholder for "registering"
// BodyHooks, before they are included in [RegisterExtras].
var todoRegisteredBodyHooks BodyHooks = NOOPBodyHooks{}

func (b *Body) hooks() BodyHooks {
	// TODO(arr4n): when incorporating BodyHooks into [RegisterExtras], the
	// [todoRegisteredBodyHooks] variable MUST be removed.
	return todoRegisteredBodyHooks
}

// NOOPBodyHooks implements [BodyHooks] such that they are equivalent to no type
// having been registered.
type NOOPBodyHooks struct{}

// The RLP-related methods of [NOOPBodyHooks] make assumptions about the struct
// fields and their order, which we lock in here as a change detector. If this
// breaks then it MUST be updated and the RLP methods reviewed + new
// backwards-compatibility tests added.
var _ = &Body{[]*Transaction{}, []*Header{}, []*Withdrawal{}}

func (NOOPBodyHooks) RLPFieldsForEncoding(b *Body) *rlp.Fields {
	return &rlp.Fields{
		Required: []any{b.Transactions, b.Uncles},
		Optional: []any{b.Withdrawals},
	}
}

func (NOOPBodyHooks) RLPFieldPointersForDecoding(b *Body) *rlp.Fields {
	return &rlp.Fields{
		Required: []any{&b.Transactions, &b.Uncles},
		Optional: []any{&b.Withdrawals},
	}
}
