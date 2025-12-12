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
