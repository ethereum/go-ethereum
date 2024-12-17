// Copyright 2024 the libevm authors.
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
	"fmt"
	"io"

	"github.com/ava-labs/libevm/libevm/pseudo"
	"github.com/ava-labs/libevm/rlp"
)

// HeaderHooks are required for all types registered with [RegisterExtras] for
// [Header] payloads.
type HeaderHooks interface {
	EncodeRLP(*Header, io.Writer) error
	DecodeRLP(*Header, *rlp.Stream) error
}

var _ interface {
	rlp.Encoder
	rlp.Decoder
} = (*Header)(nil)

// EncodeRLP implements the [rlp.Encoder] interface.
func (h *Header) EncodeRLP(w io.Writer) error {
	if r := registeredExtras; r.Registered() {
		return r.Get().hooks.hooksFromHeader(h).EncodeRLP(h, w)
	}
	return h.encodeRLP(w)
}

// decodeHeaderRLPDirectly bypasses the [Header.DecodeRLP] method to avoid
// infinite recursion.
func decodeHeaderRLPDirectly(h *Header, s *rlp.Stream) error {
	type withoutMethods Header
	return s.Decode((*withoutMethods)(h))
}

// DecodeRLP implements the [rlp.Decoder] interface.
func (h *Header) DecodeRLP(s *rlp.Stream) error {
	if r := registeredExtras; r.Registered() {
		return r.Get().hooks.hooksFromHeader(h).DecodeRLP(h, s)
	}
	return decodeHeaderRLPDirectly(h, s)
}

func (e ExtraPayloads[HPtr, SA]) hooksFromHeader(h *Header) HeaderHooks {
	return e.Header.Get(h)
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

func (*NOOPHeaderHooks) EncodeRLP(h *Header, w io.Writer) error {
	return h.encodeRLP(w)
}

func (*NOOPHeaderHooks) DecodeRLP(h *Header, s *rlp.Stream) error {
	return decodeHeaderRLPDirectly(h, s)
}
