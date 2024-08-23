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

// Get returns the value.
func (a *Value[T]) Get() T { return a.t.val.get().(T) }

// Set sets the value.
func (a *Value[T]) Set(v T) { a.t.val.mustSet(v) }

// MarshalJSON implements the [json.Marshaler] interface.
func (t *Type) MarshalJSON() ([]byte, error) { return t.val.MarshalJSON() }

// UnmarshalJSON implements the [json.Unmarshaler] interface.
func (t *Type) UnmarshalJSON(b []byte) error { return t.val.UnmarshalJSON(b) }

// MarshalJSON implements the [json.Marshaler] interface.
func (v *Value[T]) MarshalJSON() ([]byte, error) { return v.t.MarshalJSON() }

// UnmarshalJSON implements the [json.Unmarshaler] interface.
func (v *Value[T]) UnmarshalJSON(b []byte) error { return v.t.UnmarshalJSON(b) }

var _ = []interface {
	json.Marshaler
	json.Unmarshaler
}{
	(*Type)(nil),
	(*Value[struct{}])(nil),
	(*concrete[struct{}])(nil),
}

// A value is a non-generic wrapper around a [concrete] struct.
type value interface {
	get() any
	canSetTo(any) bool
	set(any) error
	mustSet(any)

	json.Marshaler
	json.Unmarshaler
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
