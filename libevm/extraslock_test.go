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

package libevm_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	// Testing from outside the package to guarantee usage of the public API only.
	. "github.com/ava-labs/libevm/libevm"
)

func TestExtrasLock(t *testing.T) {
	var zero ExtrasLock
	assert.Panics(t, func() { _ = zero.Verify() }, "Verify() method of zero-value ExtrasLock{}")

	testIntegration := func(t *testing.T) {
		t.Helper()
		require.NoError(t,
			WithTemporaryExtrasLock((ExtrasLock).Verify),
			"WithTemporaryExtrasLock((ExtrasLock).Verify)",
		)
	}
	t.Run("initial_usage", testIntegration)

	t.Run("lock_expiration", func(t *testing.T) {
		var persisted ExtrasLock
		require.NoError(t, WithTemporaryExtrasLock(func(l ExtrasLock) error {
			persisted = l
			return l.Verify()
		}))
		assert.ErrorIs(t, persisted.Verify(), ErrExpiredExtrasLock, "Verify() of persisted ExtrasLock")
	})

	t.Run("repeat_usage", testIntegration)
}
