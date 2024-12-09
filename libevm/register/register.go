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

// Package register provides functionality for optional registration of types.
package register

import (
	"errors"

	"github.com/ava-labs/libevm/libevm/testonly"
)

// An AtMostOnce allows zero or one registration of a T.
type AtMostOnce[T any] struct {
	v *T
}

// ErrReRegistration is returned on all but the first of calls to
// [AtMostOnce.Register].
var ErrReRegistration = errors.New("re-registration")

// Register registers `v` or returns [ErrReRegistration] if already called.
func (o *AtMostOnce[T]) Register(v T) error {
	if o.Registered() {
		return ErrReRegistration
	}
	o.v = &v
	return nil
}

// MustRegister is equivalent to [AtMostOnce.Register], panicking on error.
func (o *AtMostOnce[T]) MustRegister(v T) {
	if err := o.Register(v); err != nil {
		panic(err)
	}
}

// Registered reports whether [AtMostOnce.Register] has been called.
func (o *AtMostOnce[T]) Registered() bool {
	return o.v != nil
}

// Get returns the registered value. It MUST NOT be called before
// [AtMostOnce.Register].
func (o *AtMostOnce[T]) Get() T {
	return *o.v
}

// TestOnlyClear clears any previously registered value, returning `o` to its
// default state. It panics if called from a non-testing call stack.
func (o *AtMostOnce[T]) TestOnlyClear() {
	testonly.OrPanic(func() {
		o.v = nil
	})
}
