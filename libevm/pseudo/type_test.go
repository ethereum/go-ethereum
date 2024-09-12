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
