// Copyright 2025 the libevm authors.
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

package register

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAtMostOnce(t *testing.T) {
	var sut AtMostOnce[int]
	assertRegistered := func(t *testing.T, want int) {
		t.Helper()
		require.True(t, sut.Registered(), "Registered()")
		assert.Equal(t, want, sut.Get(), "Get()")
	}

	const val int = 42
	require.NoError(t, sut.Register(val), "Register()")
	assertRegistered(t, val)

	assert.PanicsWithValue(
		t, ErrReRegistration,
		func() { sut.MustRegister(0) },
		"MustRegister() after Register()",
	)

	t.Run("TestOnlyClear", func(t *testing.T) {
		sut.TestOnlyClear()
		require.False(t, sut.Registered(), "Registered()")

		t.Run("re-registration", func(t *testing.T) {
			sut.MustRegister(val)
			assertRegistered(t, val)
		})
	})
	if t.Failed() {
		return
	}

	t.Run("TempOverride", func(t *testing.T) {
		t.Run("during", func(t *testing.T) {
			require.NoError(t, sut.TempOverride(val+1, func() error {
				assertRegistered(t, val+1)
				return nil
			}))
		})
		t.Run("after", func(t *testing.T) {
			assertRegistered(t, val)
		})
	})

	t.Run("TempClear", func(t *testing.T) {
		t.Run("during", func(t *testing.T) {
			require.NoError(t, sut.TempClear(func() error {
				assert.False(t, sut.Registered(), "Registered()")
				return nil
			}))
		})
		t.Run("after", func(t *testing.T) {
			assertRegistered(t, val)
		})
	})

	t.Run("error_propagation", func(t *testing.T) {
		errFoo := errors.New("foo")
		fn := func() error { return errFoo }
		assert.ErrorIs(t, sut.TempOverride(0, fn), errFoo, "TempOverride()") //nolint:testifylint // Blindly using require is an anti-pattern!!!
		assert.ErrorIs(t, sut.TempClear(fn), errFoo, "TempClear()")
	})
}
