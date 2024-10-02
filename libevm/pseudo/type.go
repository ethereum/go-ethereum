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

// Package pseudo provides a bridge between generic and non-generic code via
// pseudo-types and pseudo-values. With careful usage, there is minimal
// reduction in type safety.
//
// Adding generic type parameters to anything (e.g. struct, function, etc)
// "pollutes" all code that uses the generic type. Refactoring all uses isn't
// always feasible, and a [Type] acts as an intermediate fix. Although their
// constructors are generic, they are not, and they are instead coupled with a
// generic [Value] that SHOULD be used for access.
//
// Packages typically SHOULD NOT expose a [Type] and SHOULD instead provide
// users with a type-safe [Value].
package pseudo

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/ethereum/go-ethereum/rlp"
)

// A Type wraps a strongly-typed value without exposing information about its
// type. It can be used in lieu of a generic field / parameter.
type Type struct {
	val value
}

// A Value provides strongly-typed access to the payload carried by a [Type].
type Value[T any] struct {
	t *Type
}

// A Pseudo type couples a [Type] and a [Value]. If returned by a constructor
// from this package, both wrap the same payload.
type Pseudo[T any] struct {
	Type  *Type
	Value *Value[T]
}

// TypeAndValue is a convenience function for splitting the contents of `p`,
// typically at construction.
func (p *Pseudo[T]) TypeAndValue() (*Type, *Value[T]) {
	return p.Type, p.Value
}

// From returns a Pseudo[T] constructed from `v`.
func From[T any](v T) *Pseudo[T] {
	t := &Type{
		val: &concrete[T]{
			val: v,
		},
	}
	return &Pseudo[T]{t, MustNewValue[T](t)}
}

// Zero is equivalent to [From] called with the [zero value] of type `T`. Note
// that pointers, slices, maps, etc. will therefore be nil.
//
// [zero value]: https://go.dev/tour/basics/12
func Zero[T any]() *Pseudo[T] {
	var x T
	return From[T](x)
}

// PointerTo is equivalent to [From] called with a pointer to the payload
// carried by `t`. It first confirms that the payload is of type `T`.
func PointerTo[T any](t *Type) (*Pseudo[*T], error) {
	c, ok := t.val.(*concrete[T])
	if !ok {
		var want *T
		return nil, fmt.Errorf("cannot create *Pseudo[%T] from *Type carrying %T", want, t.val.get())
	}
	return From(&c.val), nil
}

// MustPointerTo is equivalent to [PointerTo] except that it panics instead of
// returning an error.
func MustPointerTo[T any](t *Type) *Pseudo[*T] {
	p, err := PointerTo[T](t)
	if err != nil {
		panic(err)
	}
	return p
}

// Interface returns the wrapped value as an `any`, equivalent to
// [reflect.Value.Interface]. Prefer [Value.Get].
func (t *Type) Interface() any { return t.val.get() }

// NewValue constructs a [Value] from a [Type], first confirming that `t` wraps
// a payload of type `T`.
func NewValue[T any](t *Type) (*Value[T], error) {
	var x T
	if !t.val.canSetTo(x) {
		return nil, fmt.Errorf("cannot create *Value[%T] with *Type carrying %T", x, t.val.get())
	}
	return &Value[T]{t}, nil
}

// MustNewValue is equivalent to [NewValue] except that it panics instead of
// returning an error.
func MustNewValue[T any](t *Type) *Value[T] {
	v, err := NewValue[T](t)
	if err != nil {
		panic(err)
	}
	return v
}

// IsZero reports whether t carries the the zero value for its type.
func (t *Type) IsZero() bool { return t.val.isZero() }

// An EqualityChecker reports if it is equal to another value of the same type.
type EqualityChecker[T any] interface {
	Equal(T) bool
}

// Equal reports whether t carries a value equal to that carried by u. If t and
// u carry different types then Equal returns false. If t and u carry the same
// type and said type implements [EqualityChecker] then Equal propagates the
// value returned by the checker. In all other cases, Equal returns
// [reflect.DeepEqual] performed on the payloads carried by t and u.
func (t *Type) Equal(u *Type) bool { return t.val.equal(u) }

// Get returns the value.
func (v *Value[T]) Get() T { return v.t.val.get().(T) } //nolint:forcetypeassert // invariant

// Set sets the value.
func (v *Value[T]) Set(val T) { v.t.val.mustSet(val) }

// MarshalJSON implements the [json.Marshaler] interface.
func (t *Type) MarshalJSON() ([]byte, error) { return t.val.MarshalJSON() }

// UnmarshalJSON implements the [json.Unmarshaler] interface.
func (t *Type) UnmarshalJSON(b []byte) error { return t.val.UnmarshalJSON(b) }

// MarshalJSON implements the [json.Marshaler] interface.
func (v *Value[T]) MarshalJSON() ([]byte, error) { return v.t.MarshalJSON() }

// UnmarshalJSON implements the [json.Unmarshaler] interface.
func (v *Value[T]) UnmarshalJSON(b []byte) error { return v.t.UnmarshalJSON(b) }

// EncodeRLP implements the [rlp.Encoder] interface.
func (t *Type) EncodeRLP(w io.Writer) error { return t.val.EncodeRLP(w) }

// DecodeRLP implements the [rlp.Decoder] interface.
func (t *Type) DecodeRLP(s *rlp.Stream) error { return t.val.DecodeRLP(s) }

var _ = []interface {
	json.Marshaler
	json.Unmarshaler
}{
	(*Type)(nil),
	(*Value[struct{}])(nil),
	(*concrete[struct{}])(nil),
}

var _ = []interface {
	rlp.Encoder
	rlp.Decoder
}{
	(*Type)(nil),
	(*concrete[struct{}])(nil),
}

// A value is a non-generic wrapper around a [concrete] struct.
type value interface {
	get() any
	isZero() bool
	equal(*Type) bool
	canSetTo(any) bool
	set(any) error
	mustSet(any)

	json.Marshaler
	json.Unmarshaler
	rlp.Encoder
	rlp.Decoder
	fmt.Formatter
}

type concrete[T any] struct {
	val T
}

func (c *concrete[T]) get() any { return c.val }

func (c *concrete[T]) canSetTo(v any) bool {
	_, ok := v.(T)
	return ok
}

// An invalidTypeError is returned by [conrete.set] if the value is incompatible
// with its type. This should never leave this package and exists only to
// provide precise testing of unhappy paths.
type invalidTypeError[T any] struct {
	SetTo any
}

func (e *invalidTypeError[T]) Error() string {
	var t T
	return fmt.Sprintf("cannot set %T to %T", t, e.SetTo)
}

func (c *concrete[T]) set(v any) error {
	vv, ok := v.(T)
	if !ok {
		// Other invariants in this implementation (aim to) guarantee that this
		// will never happen.
		return &invalidTypeError[T]{SetTo: v}
	}
	c.val = vv
	return nil
}

func (c *concrete[T]) mustSet(v any) {
	if err := c.set(v); err != nil {
		panic(err)
	}
	_ = 0 // for happy-path coverage inspection
}

func (c *concrete[T]) MarshalJSON() ([]byte, error) { return json.Marshal(c.val) }

func (c *concrete[T]) UnmarshalJSON(b []byte) error {
	var v T
	if err := json.Unmarshal(b, &v); err != nil {
		return err
	}
	c.val = v
	return nil
}

func (c *concrete[T]) EncodeRLP(w io.Writer) error { return rlp.Encode(w, c.val) }
