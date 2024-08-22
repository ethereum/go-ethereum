// Package pseudo ...
package pseudo

import (
	"encoding/json"
	"fmt"
)

// Zero ...
func Zero[T any]() (*Type, *Value[T]) {
	var x T
	return From[T](x)
}

// From ...
func From[T any](x T) (*Type, *Value[T]) {
	t := &Type{
		val: &concrete[T]{
			val: x,
		},
	}
	return t, NewValueUnsafe[T](t)
}

// OnlyType ...
func OnlyType[T any](t *Type, _ *Value[T]) *Type {
	return t
}

// A Type ...
type Type struct {
	val value
}

func (t *Type) Interface() any { return t.val.get() }

// func (t *Type) Set(v any) error              { return t.val.Set(v) }
// func (t *Type) MustSet(v any)                { t.val.MustSet(v) }
func (t *Type) MarshalJSON() ([]byte, error) { return t.val.MarshalJSON() }
func (t *Type) UnmarshalJSON(b []byte) error { return t.val.UnmarshalJSON(b) }

var (
	_ json.Marshaler   = (*Type)(nil)
	_ json.Unmarshaler = (*Type)(nil)
)

func NewValueUnsafe[T any](t *Type) *Value[T] {
	return &Value[T]{t: t}
}

func NewValue[T any](t *Type) (*Value[T], error) {
	var x T
	if !t.val.canSetTo(x) {
		return nil, fmt.Errorf("cannot create *Accessor[%T] with *Type carrying %T", x, t.val.get())
	}
	return NewValueUnsafe[T](t), nil
}

type Value[T any] struct {
	t *Type
}

func (a *Value[T]) Get() T  { return a.t.val.get().(T) }
func (a *Value[T]) Set(v T) { a.t.val.mustSet(v) }

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

type InvalidTypeError[T any] struct {
	SetTo any
}

func (e *InvalidTypeError[T]) Error() string {
	var t T
	return fmt.Sprintf("cannot set %T to %T", t, e.SetTo)
}

func (c *concrete[T]) set(v any) error {
	vv, ok := v.(T)
	if !ok {
		return &InvalidTypeError[T]{SetTo: v}
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
