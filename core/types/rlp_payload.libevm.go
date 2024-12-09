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

// RegisterExtras registers the type `SA` to be carried as an extra payload in
// [StateAccount] and [SlimAccount] structs. It is expected to be called in an
// `init()` function and MUST NOT be called more than once.
//
// The payload will be treated as an extra struct field for the purposes of RLP
// encoding and decoding. RLP handling is plumbed through to the `SA` via the
// [StateAccountExtra] that holds it such that it acts as if there were a field
// of type `SA` in all StateAccount and SlimAccount structs.
//
// The payload can be accessed via the [ExtraPayloads.FromPayloadCarrier] method
// of the accessor returned by RegisterExtras.
func RegisterExtras[SA any]() ExtraPayloads[SA] {
	var extra ExtraPayloads[SA]
	registeredExtras.MustRegister(&extraConstructors{
		stateAccountType: func() string {
			var x SA
			return fmt.Sprintf("%T", x)
		}(),
		newStateAccount:   pseudo.NewConstructor[SA]().Zero,
		cloneStateAccount: extra.cloneStateAccount,
	})
	return extra
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
	stateAccountType  string
	newStateAccount   func() *pseudo.Type
	cloneStateAccount func(*StateAccountExtra) *StateAccountExtra
}

func (e *StateAccountExtra) clone() *StateAccountExtra {
	switch r := registeredExtras; {
	case !r.Registered(), e == nil:
		return nil
	default:
		return r.Get().cloneStateAccount(e)
	}
}

// ExtraPayloads provides strongly typed access to the extra payload carried by
// [StateAccount] and [SlimAccount] structs. The only valid way to construct an
// instance is by a call to [RegisterExtras].
type ExtraPayloads[SA any] struct {
	_ struct{} // make godoc show unexported fields so nobody tries to make their own instance ;)
}

func (ExtraPayloads[SA]) cloneStateAccount(s *StateAccountExtra) *StateAccountExtra {
	v := pseudo.MustNewValue[SA](s.t)
	return &StateAccountExtra{
		t: pseudo.From(v.Get()).Type,
	}
}

// ExtraPayloadCarrier is implemented by both [StateAccount] and [SlimAccount],
// allowing for their [StateAccountExtra] payloads to be accessed in a type-safe
// manner by [ExtraPayloads] instances.
type ExtraPayloadCarrier interface {
	extra() *StateAccountExtra
}

var _ = []ExtraPayloadCarrier{
	(*StateAccount)(nil),
	(*SlimAccount)(nil),
}

// FromPayloadCarrier returns the carriers's payload.
func (ExtraPayloads[SA]) FromPayloadCarrier(a ExtraPayloadCarrier) SA {
	return pseudo.MustNewValue[SA](a.extra().payload()).Get()
}

// PointerFromPayloadCarrier returns a pointer to the carriers's extra payload.
// This is guaranteed to be non-nil.
//
// Note that copying a [StateAccount] or [SlimAccount] by dereferencing a
// pointer will result in a shallow copy and that the *SA returned here will
// therefore be shared by all copies. If this is not the desired behaviour, use
// [StateAccount.Copy] or [ExtraPayloads.SetOnPayloadCarrier].
func (ExtraPayloads[SA]) PointerFromPayloadCarrier(a ExtraPayloadCarrier) *SA {
	return pseudo.MustPointerTo[SA](a.extra().payload()).Value.Get()
}

// SetOnPayloadCarrier sets the carriers's payload.
func (ExtraPayloads[SA]) SetOnPayloadCarrier(a ExtraPayloadCarrier, val SA) {
	a.extra().t = pseudo.From(val).Type
}

// A StateAccountExtra carries the extra payload, if any, registered with
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
