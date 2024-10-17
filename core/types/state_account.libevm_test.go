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

package types

import (
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"

	"github.com/ava-labs/libevm/common"
	"github.com/ava-labs/libevm/libevm/pseudo"
	"github.com/ava-labs/libevm/rlp"
)

func TestStateAccountRLP(t *testing.T) {
	// RLP encodings that don't involve extra payloads were generated on raw
	// geth StateAccounts *before* any libevm modifications, thus locking in
	// default behaviour. Encodings that involve a boolean payload were
	// generated on ava-labs/coreth StateAccounts to guarantee equivalence.

	type test struct {
		name     string
		register func()
		acc      *StateAccount
		wantHex  string
	}

	explicitFalseBoolean := test{
		name: "explicit false-boolean extra",
		register: func() {
			RegisterExtras[bool]()
		},
		acc: &StateAccount{
			Nonce:    0x444444,
			Balance:  uint256.NewInt(0x666666),
			Root:     common.Hash{},
			CodeHash: []byte{0xbb, 0xbb, 0xbb},
			Extra: &StateAccountExtra{
				t: pseudo.From(false).Type,
			},
		},
		wantHex: `0xee8344444483666666a0000000000000000000000000000000000000000000000000000000000000000083bbbbbb80`,
	}

	// The vanilla geth code won't set payloads so we need to ensure that the
	// zero-value encoding is used instead of the null-value default as when
	// no type is registered.
	implicitFalseBoolean := explicitFalseBoolean
	implicitFalseBoolean.name = "implicit false-boolean extra as zero-value of registered type"
	// Clearing the Extra makes the `false` value implicit and due only to the
	// fact that we register `bool`. Most importantly, note that `wantHex`
	// remains identical.
	implicitFalseBoolean.acc.Extra = nil

	tests := []test{
		explicitFalseBoolean,
		implicitFalseBoolean,
		{
			name: "true-boolean extra",
			register: func() {
				RegisterExtras[bool]()
			},
			acc: &StateAccount{
				Nonce:    0x444444,
				Balance:  uint256.NewInt(0x666666),
				Root:     common.Hash{},
				CodeHash: []byte{0xbb, 0xbb, 0xbb},
				Extra: &StateAccountExtra{
					t: pseudo.From(true).Type,
				},
			},
			wantHex: `0xee8344444483666666a0000000000000000000000000000000000000000000000000000000000000000083bbbbbb01`,
		},
		{
			name: "vanilla geth account",
			acc: &StateAccount{
				Nonce:    0xcccccc,
				Balance:  uint256.NewInt(0x555555),
				Root:     common.MaxHash,
				CodeHash: []byte{0x77, 0x77, 0x77},
			},
			wantHex: `0xed83cccccc83555555a0ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff83777777`,
		},
		{
			name: "vanilla geth account",
			acc: &StateAccount{
				Nonce:    0x444444,
				Balance:  uint256.NewInt(0x666666),
				Root:     common.Hash{},
				CodeHash: []byte{0xbb, 0xbb, 0xbb},
			},
			wantHex: `0xed8344444483666666a0000000000000000000000000000000000000000000000000000000000000000083bbbbbb`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.register != nil {
				TestOnlyClearRegisteredExtras()
				tt.register()
				t.Cleanup(TestOnlyClearRegisteredExtras)
			}
			assertRLPEncodingAndReturn(t, tt.acc, tt.wantHex)

			t.Run("RLP round trip via SlimAccount", func(t *testing.T) {
				got, err := FullAccount(SlimAccountRLP(*tt.acc))
				require.NoError(t, err)

				if diff := cmp.Diff(tt.acc, got); diff != "" {
					t.Errorf("FullAccount(SlimAccountRLP(x)) != x; diff (-want +got):\n%s", diff)
				}
			})
		})
	}
}

func assertRLPEncodingAndReturn(t *testing.T, val any, wantHex string) []byte {
	t.Helper()
	got, err := rlp.EncodeToBytes(val)
	require.NoError(t, err, "rlp.EncodeToBytes()")

	t.Logf("got RLP: %#x", got)
	wantHex = strings.TrimPrefix(wantHex, "0x")
	require.Equalf(t, common.Hex2Bytes(wantHex), got, "RLP encoding of %T", val)

	return got
}

func TestSlimAccountRLP(t *testing.T) {
	// All RLP encodings were generated on geth SlimAccounts *before* libevm
	// modifications, to lock in default behaviour.
	tests := []struct {
		name    string
		acc     *SlimAccount
		wantHex string
	}{
		{
			acc: &SlimAccount{
				Nonce:   0x444444,
				Balance: uint256.NewInt(0x777777),
			},
			wantHex: `0xca83444444837777778080`,
		},
		{
			acc: &SlimAccount{
				Nonce:   0x444444,
				Balance: uint256.NewInt(0x777777),
				Root:    common.MaxHash[:],
			},
			wantHex: `0xea8344444483777777a0ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff80`,
		},
		{
			acc: &SlimAccount{
				Nonce:    0x444444,
				Balance:  uint256.NewInt(0x777777),
				CodeHash: common.MaxHash[:],
			},
			wantHex: `0xea834444448377777780a0ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff`,
		},
		{
			acc: &SlimAccount{
				Nonce:    0x444444,
				Balance:  uint256.NewInt(0x777777),
				Root:     common.MaxHash[:],
				CodeHash: repeatAsHash(0xee).Bytes(),
			},
			wantHex: `0xf84a8344444483777777a0ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffa0eeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf := assertRLPEncodingAndReturn(t, tt.acc, tt.wantHex)

			got := new(SlimAccount)
			require.NoError(t, rlp.DecodeBytes(buf, got), "rlp.DecodeBytes()")

			opts := []cmp.Option{
				// The require package differentiates between empty and nil
				// slices and doesn't have a configuration mechanism.
				cmpopts.EquateEmpty(),
			}

			if diff := cmp.Diff(tt.acc, got, opts...); diff != "" {
				t.Errorf("rlp.DecodeBytes(rlp.EncodeToBytes(%T), ...) round trip; diff (-want +got):\n%s", tt.acc, diff)
			}
		})
	}
}

func repeatAsHash(x byte) (h common.Hash) {
	for i := range h {
		h[i] = x
	}
	return h
}
