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

package state

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ava-labs/libevm/common"
	"github.com/ava-labs/libevm/core/types"
)

func TestStateObjectEmpty(t *testing.T) {
	tests := []struct {
		name           string
		registerAndSet func(*types.StateAccount)
		wantEmpty      bool
	}{
		{
			name:           "no registered types.StateAccount extra payload",
			registerAndSet: func(*types.StateAccount) {},
			wantEmpty:      true,
		},
		{
			name: "erroneously non-nil types.StateAccountExtra when no registered payload",
			registerAndSet: func(acc *types.StateAccount) {
				acc.Extra = &types.StateAccountExtra{}
			},
			wantEmpty: true,
		},
		{
			name: "explicit false bool",
			registerAndSet: func(acc *types.StateAccount) {
				types.RegisterExtras[types.NOOPHeaderHooks, *types.NOOPHeaderHooks, bool]().StateAccount.Set(acc, false)
			},
			wantEmpty: true,
		},
		{
			name: "implicit false bool",
			registerAndSet: func(*types.StateAccount) {
				types.RegisterExtras[types.NOOPHeaderHooks, *types.NOOPHeaderHooks, bool]()
			},
			wantEmpty: true,
		},
		{
			name: "true bool",
			registerAndSet: func(acc *types.StateAccount) {
				types.RegisterExtras[types.NOOPHeaderHooks, *types.NOOPHeaderHooks, bool]().StateAccount.Set(acc, true)
			},
			wantEmpty: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			types.TestOnlyClearRegisteredExtras()
			t.Cleanup(types.TestOnlyClearRegisteredExtras)

			obj := newObject(nil, common.Address{}, nil)
			tt.registerAndSet(&obj.data)
			require.Equalf(t, tt.wantEmpty, obj.empty(), "%T.empty()", obj)
		})
	}
}
