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

package types

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ava-labs/libevm/libevm"
	"github.com/ava-labs/libevm/rlp"
)

type tempBlockBodyHooks struct {
	X string
	NOOPBlockBodyHooks
}

func (b *tempBlockBodyHooks) Copy() *tempBlockBodyHooks {
	return &tempBlockBodyHooks{X: b.X}
}

func (b *tempBlockBodyHooks) BlockRLPFieldsForEncoding(*BlockRLPProxy) *rlp.Fields {
	return &rlp.Fields{
		Required: []any{b.X},
	}
}

func TestTempRegisteredExtras(t *testing.T) {
	TestOnlyClearRegisteredExtras()
	t.Cleanup(TestOnlyClearRegisteredExtras)

	rlpWithoutHooks, err := rlp.EncodeToBytes(&Block{})
	require.NoErrorf(t, err, "rlp.EncodeToBytes(%T) without hooks", &Block{})

	extras := RegisterExtras[NOOPHeaderHooks, *NOOPHeaderHooks, NOOPBlockBodyHooks, *NOOPBlockBodyHooks, bool]()
	testPrimaryExtras := func(t *testing.T) {
		t.Helper()
		b := new(Block)
		got, err := rlp.EncodeToBytes(b)
		require.NoErrorf(t, err, "rlp.EncodeToBytes(%T) with %T hooks", b, extras.Block.Get(b))
		assert.Equalf(t, rlpWithoutHooks, got, "rlp.EncodeToBytes(%T) with noop hooks; expect same as without hooks", b)
	}

	t.Run("before_temp", testPrimaryExtras)
	t.Run("WithTempRegisteredExtras", func(t *testing.T) {
		err := libevm.WithTemporaryExtrasLock(func(lock libevm.ExtrasLock) error {
			return WithTempRegisteredExtras(lock, func(extras ExtraPayloads[*NOOPHeaderHooks, *tempBlockBodyHooks, bool]) error {
				const val = "Hello, world"
				b := new(Block)
				payload := &tempBlockBodyHooks{X: val}
				extras.Block.Set(b, payload)

				got, err := rlp.EncodeToBytes(b)
				require.NoErrorf(t, err, "rlp.EncodeToBytes(%T) with %T hooks", b, extras.Block.Get(b))
				want, err := rlp.EncodeToBytes([]string{val})
				require.NoErrorf(t, err, "rlp.EncodeToBytes(%T{%[1]v})", []string{val})

				assert.Equalf(t, want, got, "rlp.EncodeToBytes(%T) with %T hooks", b, payload)
				return nil
			})
		})
		require.NoError(t, err)
	})
	t.Run("after_temp", testPrimaryExtras)
}
