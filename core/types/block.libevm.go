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
	"io"

	"github.com/ava-labs/libevm/libevm/pseudo"
	"github.com/ava-labs/libevm/rlp"
)

// HeaderHooks are required for all types registered with [RegisterExtras] for
// [Header] payloads.
type HeaderHooks interface {
	EncodeJSON(*Header) ([]byte, error)
	DecodeJSON(*Header, []byte) error
	EncodeRLP(*Header, io.Writer) error
	DecodeRLP(*Header, *rlp.Stream) error
	PostCopy(dst *Header)
}

var _ interface {
	rlp.Encoder
	rlp.Decoder
	json.Marshaler
	json.Unmarshaler
} = (*Header)(nil)

// MarshalJSON implements the [json.Marshaler] interface.
func (h *Header) MarshalJSON() ([]byte, error) {
	return h.hooks().EncodeJSON(h)
}

// UnmarshalJSON implements the [json.Unmarshaler] interface.
func (h *Header) UnmarshalJSON(b []byte) error {
	return h.hooks().DecodeJSON(h, b)
}

// EncodeRLP implements the [rlp.Encoder] interface.
func (h *Header) EncodeRLP(w io.Writer) error {
	return h.hooks().EncodeRLP(h, w)
}

// DecodeRLP implements the [rlp.Decoder] interface.
func (h *Header) DecodeRLP(s *rlp.Stream) error {
	return h.hooks().DecodeRLP(h, s)
}

// NOOPHeaderHooks implements [HeaderHooks] such that they are equivalent to
// no type having been registered.
type NOOPHeaderHooks struct{}

var _ HeaderHooks = (*NOOPHeaderHooks)(nil)

func (*NOOPHeaderHooks) EncodeJSON(h *Header) ([]byte, error) {
	return h.marshalJSON()
}

func (*NOOPHeaderHooks) DecodeJSON(h *Header, b []byte) error {
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

var _ = []interface {
	rlp.Encoder
	rlp.Decoder
}{
	(*Body)(nil),
	(*extblock)(nil),
}

// EncodeRLP implements the [rlp.Encoder] interface.
func (b *Body) EncodeRLP(w io.Writer) error {
	return b.hooks().BodyRLPFieldsForEncoding(b).EncodeRLP(w)
}

// DecodeRLP implements the [rlp.Decoder] interface.
func (b *Body) DecodeRLP(s *rlp.Stream) error {
	return b.hooks().BodyRLPFieldPointersForDecoding(b).DecodeRLP(s)
}

// BlockRLPProxy exports the geth-internal type used for RLP {en,de}coding of a
// [Block].
type BlockRLPProxy extblock

func (b *extblock) EncodeRLP(w io.Writer) error {
	bb := (*BlockRLPProxy)(b)
	return b.hooks.BlockRLPFieldsForEncoding(bb).EncodeRLP(w)
}

func (b *extblock) DecodeRLP(s *rlp.Stream) error {
	bb := (*BlockRLPProxy)(b)
	return b.hooks.BlockRLPFieldPointersForDecoding(bb).DecodeRLP(s)
}

// BlockBodyHooks are required for all types registered with [RegisterExtras]
// for [Block] and [Body] payloads.
type BlockBodyHooks interface {
	BlockRLPFieldsForEncoding(*BlockRLPProxy) *rlp.Fields
	BlockRLPFieldPointersForDecoding(*BlockRLPProxy) *rlp.Fields
	BodyRLPFieldsForEncoding(*Body) *rlp.Fields
	BodyRLPFieldPointersForDecoding(*Body) *rlp.Fields
}

// NOOPBlockBodyHooks implements [BlockBodyHooks] such that they are equivalent
// to no type having been registered.
type NOOPBlockBodyHooks struct{}

var _ BlockBodyPayload[*NOOPBlockBodyHooks] = NOOPBlockBodyHooks{}

func (NOOPBlockBodyHooks) Copy() *NOOPBlockBodyHooks { return &NOOPBlockBodyHooks{} }

// The RLP-related methods of [NOOPBlockBodyHooks] make assumptions about the
// struct fields and their order, which we lock in here as a change detector. If
// these break then they MUST be updated and the RLP methods reviewed + new
// backwards-compatibility tests added.
var (
	_ = &Body{
		[]*Transaction{}, []*Header{}, []*Withdrawal{}, // geth
		&pseudo.Type{}, // libevm
	}
	_ = extblock{
		&Header{}, []*Transaction{}, []*Header{}, []*Withdrawal{}, // geth
		BlockBodyHooks(nil), // libevm
	}
	// Demonstrate identity of these two types, by definition but useful for
	// inspection here.
	_ = extblock(BlockRLPProxy{})
)

func (NOOPBlockBodyHooks) BlockRLPFieldsForEncoding(b *BlockRLPProxy) *rlp.Fields {
	return &rlp.Fields{
		Required: []any{b.Header, b.Txs, b.Uncles},
		Optional: []any{b.Withdrawals},
	}
}

func (NOOPBlockBodyHooks) BlockRLPFieldPointersForDecoding(b *BlockRLPProxy) *rlp.Fields {
	return &rlp.Fields{
		Required: []any{&b.Header, &b.Txs, &b.Uncles},
		Optional: []any{&b.Withdrawals},
	}
}

func (NOOPBlockBodyHooks) BodyRLPFieldsForEncoding(b *Body) *rlp.Fields {
	return &rlp.Fields{
		Required: []any{b.Transactions, b.Uncles},
		Optional: []any{b.Withdrawals},
	}
}

func (NOOPBlockBodyHooks) BodyRLPFieldPointersForDecoding(b *Body) *rlp.Fields {
	return &rlp.Fields{
		Required: []any{&b.Transactions, &b.Uncles},
		Optional: []any{&b.Withdrawals},
	}
}
