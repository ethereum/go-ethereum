package pseudo

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConstructor(t *testing.T) {
	testConstructor[uint](t)
	testConstructor[string](t)
	testConstructor[struct{ x string }](t)
}

//nolint:thelper // This is the test itself so we want local line numbers reported.
func testConstructor[T any](t *testing.T) {
	var zero T
	t.Run(fmt.Sprintf("%T", zero), func(t *testing.T) {
		ctor := NewConstructor[T]()

		t.Run("NilPointer()", func(t *testing.T) {
			got := get[*T](t, ctor.NilPointer())
			assert.Nil(t, got)
		})

		t.Run("NewPointer()", func(t *testing.T) {
			got := get[*T](t, ctor.NewPointer())
			require.NotNil(t, got)
			assert.Equal(t, zero, *got)
		})

		t.Run("Zero()", func(t *testing.T) {
			got := get[T](t, ctor.Zero())
			assert.Equal(t, zero, got)
		})
	})
}

func get[T any](t *testing.T, typ *Type) (x T) {
	t.Helper()
	val, err := NewValue[T](typ)
	require.NoError(t, err, "NewValue[%T]()", x)
	return val.Get()
}
