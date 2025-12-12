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

package pseudo

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestType(t *testing.T) {
	testType(t, "Zero[int]", Zero[int], 0, 42, "I'm not an int")
	testType(t, "Zero[string]", Zero[string], "", "hello, world", 99)

	testType(
		t, "From[uint](314159)",
		func() *Pseudo[uint] {
			return From[uint](314159)
		},
		314159, 0, struct{}{},
	)

	testType(t, "nil pointer", Zero[*float64], (*float64)(nil), new(float64), 0)
}

//nolint:thelper // This is the test itself so we want local line numbers reported.
func testType[T any](t *testing.T, name string, ctor func() *Pseudo[T], init T, setTo T, invalid any) {
	t.Run(name, func(t *testing.T) {
		typ, val := ctor().TypeAndValue()
		assert.Equal(t, init, val.Get())
		val.Set(setTo)
		assert.Equal(t, setTo, val.Get())

		t.Run("set to invalid type", func(t *testing.T) {
			wantErr := &invalidTypeError[T]{SetTo: invalid}

			assertError := func(t *testing.T, err any) {
				t.Helper()
				switch err := err.(type) {
				case *invalidTypeError[T]:
					assert.Equal(t, wantErr, err)
				default:
					t.Errorf("got error %v; want %v", err, wantErr)
				}
			}

			t.Run(fmt.Sprintf("Set(%T{%v})", invalid, invalid), func(t *testing.T) {
				assertError(t, typ.val.set(invalid))
			})

			t.Run(fmt.Sprintf("MustSet(%T{%v})", invalid, invalid), func(t *testing.T) {
				defer func() {
					assertError(t, recover())
				}()
				typ.val.mustSet(invalid)
			})
		})

		t.Run("JSON round trip", func(t *testing.T) {
			buf, err := json.Marshal(typ)
			require.NoError(t, err)

			got, gotVal := Zero[T]().TypeAndValue()
			require.NoError(t, json.Unmarshal(buf, &got))
			assert.Equal(t, val.Get(), gotVal.Get())
		})
	})
}

//nolint:ineffassign,testableexamples // Although `typ` is overwritten it's to demonstrate different approaches
func ExamplePseudo_TypeAndValue() {
	typ, val := From("hello").TypeAndValue()

	// But, if only one is needed:
	typ = From("world").Type
	val = From("this isn't coupled to the Type").Value

	_ = typ
	_ = val
}

func TestPointer(t *testing.T) {
	type carrier struct {
		payload int
	}

	typ, val := From(carrier{42}).TypeAndValue()

	t.Run("invalid type", func(t *testing.T) {
		_, err := PointerTo[int](typ)
		require.Errorf(t, err, "PointerTo[int](%T)", carrier{})
	})

	t.Run("valid type", func(t *testing.T) {
		ptrVal := MustPointerTo[carrier](typ).Value

		assert.Equal(t, 42, val.Get().payload, "before setting via pointer")
		var ptr *carrier = ptrVal.Get()
		ptr.payload = 314159
		assert.Equal(t, 314159, val.Get().payload, "after setting via pointer")
	})
}

func TestIsZero(t *testing.T) {
	tests := []struct {
		typ  *Type
		want bool
	}{
		{From(0).Type, true},
		{From(1).Type, false},
		{From("").Type, true},
		{From("x").Type, false},
		{From((*testing.T)(nil)).Type, true},
		{From(t).Type, false},
		{From(false).Type, true},
		{From(true).Type, false},
	}

	for _, tt := range tests {
		assert.Equalf(t, tt.want, tt.typ.IsZero(), "%T(%[1]v) IsZero()", tt.typ.Interface())
	}
}

type isEqualStub struct {
	isEqual bool
}

var _ EqualityChecker[isEqualStub] = (*isEqualStub)(nil)

func (s isEqualStub) Equal(isEqualStub) bool {
	return s.isEqual
}

func TestEqual(t *testing.T) {
	isEqual := isEqualStub{true}
	notEqual := isEqualStub{false}

	tests := []struct {
		a, b *Type
		want bool
	}{
		{From(42).Type, From(42).Type, true},
		{From(99).Type, From("").Type, false},
		{From(false).Type, From("").Type, false}, // sorry JavaScript, you're wrong
		{From(isEqual).Type, From(isEqual).Type, true},
		{From(notEqual).Type, From(notEqual).Type, false},
	}

	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			t.Logf("a = %+v", tt.a)
			t.Logf("b = %+v", tt.b)
			assert.Equal(t, tt.want, tt.a.Equal(tt.b), "a.Equals(b)")
			assert.Equal(t, tt.want, tt.b.Equal(tt.a), "b.Equals(a)")
		})
	}
}
