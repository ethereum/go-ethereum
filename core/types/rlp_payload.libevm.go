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
	"github.com/ava-labs/libevm/libevm/testonly"
	"github.com/ava-labs/libevm/rlp"
)

// RegisterExtras registers the type `SA` to be carried as an extra payload in
// [StateAccount] structs. It is expected to be called in an `init()` function
// and MUST NOT be called more than once.
//
// The payload will be treated as an extra struct field for the purposes of RLP
// encoding and decoding. RLP handling is plumbed through to the `SA` via the
// [StateAccountExtra] that holds it such that it acts as if there were a field
// of type `SA` in all StateAccount structs.
//
// The payload can be acced via the [ExtraPayloads.FromStateAccount] method of
// the accessor returned by RegisterExtras.
func RegisterExtras[SA any]() ExtraPayloads[SA] {
	if registeredExtras != nil {
		panic("re-registration of Extras")
	}
	var extra ExtraPayloads[SA]
	registeredExtras = &extraConstructors{
		stateAccountType: func() string {
			var x SA
			return fmt.Sprintf("%T", x)
		}(),
		newStateAccount:   pseudo.NewConstructor[SA]().Zero,
		cloneStateAccount: extra.cloneStateAccount,
	}
	return extra
}

// TestOnlyClearRegisteredExtras clears the [Extras] previously passed to
// [RegisterExtras]. It panics if called from a non-testing call stack.
//
// In tests it SHOULD be called before every call to [RegisterExtras] and then
// defer-called afterwards, either directly or via testing.TB.Cleanup(). This is
// a workaround for the single-call limitation on [RegisterExtras].
func TestOnlyClearRegisteredExtras() {
	testonly.OrPanic(func() {
		registeredExtras = nil
	})
}

var registeredExtras *extraConstructors

type extraConstructors struct {
	stateAccountType  string
	newStateAccount   func() *pseudo.Type
	cloneStateAccount func(*StateAccountExtra) *StateAccountExtra
}

func (e *StateAccountExtra) clone() *StateAccountExtra {
	switch r := registeredExtras; {
	case r == nil, e == nil:
		return nil
	default:
		return r.cloneStateAccount(e)
	}
}

// ExtraPayloads provides strongly typed access to the extra payload carried by
// [StateAccount] structs. The only valid way to construct an instance is by a
// call to [RegisterExtras].
type ExtraPayloads[SA any] struct {
	_ struct{} // make godoc show unexported fields so nobody tries to make their own instance ;)
}

func (ExtraPayloads[SA]) cloneStateAccount(s *StateAccountExtra) *StateAccountExtra {
	v := pseudo.MustNewValue[SA](s.t)
	return &StateAccountExtra{
		t: pseudo.From(v.Get()).Type,
	}
}

// FromStateAccount returns the StateAccount's payload.
func (ExtraPayloads[SA]) FromStateAccount(a *StateAccount) SA {
	return pseudo.MustNewValue[SA](a.extra().payload()).Get()
}

// PointerFromStateAccount returns a pointer to the StateAccounts's extra
// payload. This is guaranteed to be non-nil.
//
// Note that copying a StateAccount by dereferencing a pointer will result in a
// shallow copy and that the *SA returned here will therefore be shared by all
// copies. If this is not the desired behaviour, use
// [StateAccount.Copy] or [ExtraPayloads.SetOnStateAccount].
func (ExtraPayloads[SA]) PointerFromStateAccount(a *StateAccount) *SA {
	return pseudo.MustPointerTo[SA](a.extra().payload()).Value.Get()
}

// SetOnStateAccount sets the StateAccount's payload.
func (ExtraPayloads[SA]) SetOnStateAccount(a *StateAccount, val SA) {
	a.extra().t = pseudo.From(val).Type
}

// A StateAccountExtra carries the extra payload, if any, registered with
// [RegisterExtras]. It SHOULD NOT be used directly; instead use the
// [ExtraPayloads] accessor returned by RegisterExtras.
type StateAccountExtra struct {
	t *pseudo.Type
}

func (a *StateAccount) extra() *StateAccountExtra {
	if a.Extra == nil {
		a.Extra = &StateAccountExtra{
			t: registeredExtras.newStateAccount(),
		}
	}
	return a.Extra
}

func (e *StateAccountExtra) payload() *pseudo.Type {
	if e.t == nil {
		e.t = registeredExtras.newStateAccount()
	}
	return e.t
}

// Equal reports whether `e` is semantically equivalent to `f` for the purpose
// of tests.
//
// Equal MUST NOT be used in production. Instead, compare values returned by
// [ExtraPayloads.FromStateAccount].
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

var _ interface {
	rlp.Encoder
	rlp.Decoder
	fmt.Formatter
} = (*StateAccountExtra)(nil)

// EncodeRLP implements the [rlp.Encoder] interface.
func (e *StateAccountExtra) EncodeRLP(w io.Writer) error {
	switch r := registeredExtras; {
	case r == nil:
		return nil
	case e == nil:
		e = &StateAccountExtra{}
		fallthrough
	case e.t == nil:
		e.t = r.newStateAccount()
	}
	return e.t.EncodeRLP(w)
}

// DecodeRLP implements the [rlp.Decoder] interface.
func (e *StateAccountExtra) DecodeRLP(s *rlp.Stream) error {
	switch r := registeredExtras; {
	case r == nil:
		return nil
	case e.t == nil:
		e.t = r.newStateAccount()
		fallthrough
	default:
		return s.Decode(e.t)
	}
}

// Format implements the [fmt.Formatter] interface.
func (e *StateAccountExtra) Format(s fmt.State, verb rune) {
	var out string
	switch r := registeredExtras; {
	case r == nil:
		out = "<nil>"
	case e == nil, e.t == nil:
		out = fmt.Sprintf("<nil>[*StateAccountExtra[%s]]", r.stateAccountType)
	default:
		e.t.Format(s, verb)
		return
	}
	_, _ = s.Write([]byte(out))
}
