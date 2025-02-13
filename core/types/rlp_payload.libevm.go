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
	"github.com/ava-labs/libevm/libevm/register"
	"github.com/ava-labs/libevm/libevm/testonly"
	"github.com/ava-labs/libevm/rlp"
)

// RegisterExtras registers the type `HPtr` to be carried as an extra payload in
// [Header] structs, the type `BPtr` in [Block] and [Body] structs, and the type
// `SA` in [StateAccount] and [SlimAccount] structs. It is expected to be called
// in an `init()` function and MUST NOT be called more than once.
//
// The `SA` payload will be treated as an extra struct field for the purposes of
// RLP encoding and decoding. RLP handling is plumbed through to the `SA` via
// the [StateAccountExtra] that holds it such that it acts as if there were a
// field of type `SA` in all StateAccount and SlimAccount structs.
//
// The payloads can be accessed via the [pseudo.Accessor] methods of the
// [ExtraPayloads] returned by RegisterExtras. The default `SA` value accessed
// in this manner will be a zero-value `SA` while the default value from a
// [Header] or [Block] / [Body] is a non-nil `HPtr` or `BPtr` respectively. The
// latter guarantee ensures that hooks won't be called on nil-pointer receivers.
func RegisterExtras[
	H any, HPtr interface {
		HeaderHooks
		*H
	},
	B any, BPtr interface {
		BlockBodyPayload[BPtr]
		*B
	},
	SA any,
]() ExtraPayloads[HPtr, BPtr, SA] {
	extra := ExtraPayloads[HPtr, BPtr, SA]{
		Header: pseudo.NewAccessor[*Header, HPtr](
			(*Header).extraPayload,
			func(h *Header, t *pseudo.Type) { h.extra = t },
		),
		Body: pseudo.NewAccessor[*Body, BPtr](
			(*Body).extraPayload,
			func(b *Body, t *pseudo.Type) { b.extra = t },
		),
		Block: pseudo.NewAccessor[*Block, BPtr](
			(*Block).extraPayload,
			func(b *Block, t *pseudo.Type) { b.extra = t },
		),
		StateAccount: pseudo.NewAccessor[StateOrSlimAccount, SA](
			func(a StateOrSlimAccount) *pseudo.Type { return a.extra().payload() },
			func(a StateOrSlimAccount, t *pseudo.Type) { a.extra().t = t },
		),
	}
	registeredExtras.MustRegister(&extraConstructors{
		stateAccountType: func() string {
			var x SA
			return fmt.Sprintf("%T", x)
		}(),
		// The [ExtraPayloads] that we returns is based on [HPtr,BPtr,SA], not
		// [H,B,SA] so our constructors MUST match that. This guarantees that
		// calls to the [HeaderHooks] and [BlockBodyHooks] methods will never be
		// performed on a nil pointer.
		newHeader:       pseudo.NewConstructor[H]().NewPointer, // i.e. non-nil HPtr
		newBlockOrBody:  pseudo.NewConstructor[B]().NewPointer, // i.e. non-nil BPtr
		newStateAccount: pseudo.NewConstructor[SA]().Zero,
		hooks:           extra,
	})
	return extra
}

// A BlockBodyPayload is an implementation of [BlockBodyHooks] that is also able
// to clone itself. Both [Block.Body] and [Block.WithBody] require this
// functionality to copy the payload between the types.
type BlockBodyPayload[BPtr any] interface {
	BlockBodyHooks
	Copy() BPtr
}

// TestOnlyClearRegisteredExtras clears the [Extras] previously passed to
// [RegisterExtras]. It panics if called from a non-testing call stack.
//
// In tests it SHOULD be called before every call to [RegisterExtras] and then
// defer-called afterwards, either directly or via testing.TB.Cleanup(). This is
// a workaround for the single-call limitation on [RegisterExtras].
func TestOnlyClearRegisteredExtras() {
	registeredExtras.TestOnlyClear()
}

var registeredExtras register.AtMostOnce[*extraConstructors]

type extraConstructors struct {
	stateAccountType string
	newHeader        func() *pseudo.Type
	newBlockOrBody   func() *pseudo.Type
	newStateAccount  func() *pseudo.Type
	hooks            interface {
		hooksFromHeader(*Header) HeaderHooks
		hooksFromBody(*Body) BlockBodyHooks
		hooksFromBlock(*Block) BlockBodyHooks
		cloneBlockPayload(*Block) *pseudo.Type
		cloneBodyPayload(*Body) *pseudo.Type
		cloneStateAccount(*StateAccountExtra) *StateAccountExtra
	}
}

func extraPayloadOrSetDefault(field **pseudo.Type, construct func(*extraConstructors) *pseudo.Type) *pseudo.Type {
	r := registeredExtras
	if !r.Registered() {
		// See params.ChainConfig.extraPayload() for panic rationale.
		panic("<T>.extraPayload() called before RegisterExtras()")
	}
	if *field == nil {
		*field = construct(r.Get())
	}
	return *field
}

func (h *Header) extraPayload() *pseudo.Type {
	return extraPayloadOrSetDefault(&h.extra, func(c *extraConstructors) *pseudo.Type {
		return c.newHeader()
	})
}

func (b *Body) extraPayload() *pseudo.Type {
	return extraPayloadOrSetDefault(&b.extra, func(c *extraConstructors) *pseudo.Type {
		return c.newBlockOrBody()
	})
}

func (b *Block) extraPayload() *pseudo.Type {
	return extraPayloadOrSetDefault(&b.extra, func(c *extraConstructors) *pseudo.Type {
		return c.newBlockOrBody()
	})
}

func (h *Header) hooks() HeaderHooks {
	if r := registeredExtras; r.Registered() {
		return r.Get().hooks.hooksFromHeader(h)
	}
	return new(NOOPHeaderHooks)
}

func (b *Body) hooks() BlockBodyHooks {
	if r := registeredExtras; r.Registered() {
		return r.Get().hooks.hooksFromBody(b)
	}
	return NOOPBlockBodyHooks{}
}

func (b *Block) hooks() BlockBodyHooks {
	if r := registeredExtras; r.Registered() {
		return r.Get().hooks.hooksFromBlock(b)
	}
	return NOOPBlockBodyHooks{}
}

func (e *StateAccountExtra) clone() *StateAccountExtra {
	switch r := registeredExtras; {
	case !r.Registered(), e == nil:
		return nil
	default:
		return r.Get().hooks.cloneStateAccount(e)
	}
}

// ExtraPayloads provides strongly typed access to the extra payload carried by
// [Header], [Body], [StateAccount], and [SlimAccount] structs. The only valid way to
// construct an instance is by a call to [RegisterExtras].
type ExtraPayloads[HPtr HeaderHooks, BPtr BlockBodyPayload[BPtr], SA any] struct {
	Header       pseudo.Accessor[*Header, HPtr]
	Block        pseudo.Accessor[*Block, BPtr]
	Body         pseudo.Accessor[*Body, BPtr]
	StateAccount pseudo.Accessor[StateOrSlimAccount, SA] // Also provides [SlimAccount] access.
}

func (e ExtraPayloads[HPtr, BPtr, SA]) hooksFromHeader(h *Header) HeaderHooks  { return e.Header.Get(h) }
func (e ExtraPayloads[HPtr, BPtr, SA]) hooksFromBody(b *Body) BlockBodyHooks   { return e.Body.Get(b) }
func (e ExtraPayloads[HPtr, BPtr, SA]) hooksFromBlock(b *Block) BlockBodyHooks { return e.Block.Get(b) }

func (ExtraPayloads[HPtr, BPtr, SA]) cloneStateAccount(s *StateAccountExtra) *StateAccountExtra {
	v := pseudo.MustNewValue[SA](s.t)
	return &StateAccountExtra{
		t: pseudo.From(v.Get()).Type,
	}
}

// blockOrBody is an interface for use as a method argument as they can't
// introduce new generic type parameters.
type blockOrBody interface {
	isBlockOrBody() // noop to restrict type as [Header.extraPayload] otherwise matches
	extraPayload() *pseudo.Type
}

func (*Block) isBlockOrBody() {}
func (*Body) isBlockOrBody()  {}

func (e ExtraPayloads[HPtr, BPtr, SA]) cloneBodyPayload(b *Body) *pseudo.Type {
	return e.cloneBlockOrBodyPayload(b)
}

func (e ExtraPayloads[HPtr, BPtr, SA]) cloneBlockPayload(b *Block) *pseudo.Type {
	return e.cloneBlockOrBodyPayload(b)
}

func (ExtraPayloads[HPtr, BPtr, SA]) cloneBlockOrBodyPayload(b blockOrBody) *pseudo.Type {
	v := pseudo.MustNewValue[BPtr](b.extraPayload())
	return pseudo.From(v.Get().Copy()).Type
}

func (b *Body) cloneExtra() *pseudo.Type {
	if r := registeredExtras; r.Registered() {
		return r.Get().hooks.cloneBodyPayload(b)
	}
	return nil
}

func (b *Block) cloneExtra() *pseudo.Type {
	if r := registeredExtras; r.Registered() {
		return r.Get().hooks.cloneBlockPayload(b)
	}
	return nil
}

// StateOrSlimAccount is implemented by both [StateAccount] and [SlimAccount],
// allowing for their [StateAccountExtra] payloads to be accessed in a type-safe
// manner by [ExtraPayloads] instances.
type StateOrSlimAccount interface {
	extra() *StateAccountExtra
}

var _ = []StateOrSlimAccount{
	(*StateAccount)(nil),
	(*SlimAccount)(nil),
}

// A StateAccountExtra carries the `SA` extra payload, if any, registered with
// [RegisterExtras]. It SHOULD NOT be used directly; instead use the
// [ExtraPayloads] accessor returned by RegisterExtras.
type StateAccountExtra struct {
	t *pseudo.Type
}

func (a *StateAccount) extra() *StateAccountExtra {
	return getOrSetNewStateAccountExtra(&a.Extra)
}

func (a *SlimAccount) extra() *StateAccountExtra {
	return getOrSetNewStateAccountExtra(&a.Extra)
}

func getOrSetNewStateAccountExtra(curr **StateAccountExtra) *StateAccountExtra {
	if *curr == nil {
		*curr = &StateAccountExtra{
			t: registeredExtras.Get().newStateAccount(),
		}
	}
	return *curr
}

func (e *StateAccountExtra) payload() *pseudo.Type {
	if e.t == nil {
		e.t = registeredExtras.Get().newStateAccount()
	}
	return e.t
}

// Equal reports whether `e` is semantically equivalent to `f` for the purpose
// of tests.
//
// Equal MUST NOT be used in production. Instead, compare values returned by
// [ExtraPayloads.FromPayloadCarrier].
func (e *StateAccountExtra) Equal(f *StateAccountExtra) bool {
	if false {
		// TODO(arr4n): calling this results in an error from cmp.Diff():
		// "non-deterministic or non-symmetric function detected". Explore the
		// issue and then enable the enforcement.
		testonly.OrPanic(func() {})
	}

	eNil := e == nil || e.t == nil
	fNil := f == nil || f.t == nil
	if eNil && fNil || eNil && f.t.IsZero() || fNil && e.t.IsZero() {
		return true
	}
	return e.t.Equal(f.t)
}

// IsZero reports whether e carries the the zero value for its type, as
// registered via [RegisterExtras]. It returns true if no type was registered or
// if `e == nil`.
func (e *StateAccountExtra) IsZero() bool {
	return e == nil || e.t == nil || e.t.IsZero()
}

var _ interface {
	rlp.Encoder
	rlp.Decoder
	fmt.Formatter
} = (*StateAccountExtra)(nil)

// EncodeRLP implements the [rlp.Encoder] interface.
func (e *StateAccountExtra) EncodeRLP(w io.Writer) error {
	switch r := registeredExtras; {
	case !r.Registered():
		return nil
	case e == nil:
		e = &StateAccountExtra{}
		fallthrough
	case e.t == nil:
		e.t = r.Get().newStateAccount()
	}
	return e.t.EncodeRLP(w)
}

// DecodeRLP implements the [rlp.Decoder] interface.
func (e *StateAccountExtra) DecodeRLP(s *rlp.Stream) error {
	switch r := registeredExtras; {
	case !r.Registered():
		return nil
	case e.t == nil:
		e.t = r.Get().newStateAccount()
		fallthrough
	default:
		return s.Decode(e.t)
	}
}

// Format implements the [fmt.Formatter] interface.
func (e *StateAccountExtra) Format(s fmt.State, verb rune) {
	var out string
	switch r := registeredExtras; {
	case !r.Registered():
		out = "<nil>"
	case e == nil, e.t == nil:
		out = fmt.Sprintf("<nil>[*StateAccountExtra[%s]]", r.Get().stateAccountType)
	default:
		e.t.Format(s, verb)
		return
	}
	_, _ = s.Write([]byte(out))
}
