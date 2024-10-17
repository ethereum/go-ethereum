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

package types_test

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ava-labs/libevm/common"
	"github.com/ava-labs/libevm/core/rawdb"
	"github.com/ava-labs/libevm/core/types"
	"github.com/ava-labs/libevm/libevm/ethtest"
	"github.com/ava-labs/libevm/trie"
	"github.com/ava-labs/libevm/triedb"
)

func TestStateAccountExtraViaTrieStorage(t *testing.T) {
	rng := ethtest.NewPseudoRand(1984)
	addr := rng.Address()

	type arbitraryPayload struct {
		Data string
	}
	const arbitraryData = "Hello, RLP world!"

	var (
		// The specific trie hashes after inserting the account are irrelevant;
		// what's important is that: (a) they are all different; and (b) tests
		// of implicit and explicit zero-value payloads have the same hash.
		vanillaGeth = common.HexToHash("0x2108846aaec8a88cfa02887527ad8c1beffc11b5ec428b68f15d9ce4e71e4ce1")
		trueBool    = common.HexToHash("0x665576885e52711e4cf90b72750fc1c17c80c5528bc54244e327414d486a10a4")
		falseBool   = common.HexToHash("0xa53fcb27d01347e202fb092d0af2a809cb84390c6001cbc151052ee29edc2294")
		arbitrary   = common.HexToHash("0x94eecff1444ab69437636630918c15596e001b30b973f03e06006ae20aa6e307")
	)

	tests := []struct {
		name                string
		registerAndSetExtra func(*types.StateAccount) *types.StateAccount
		assertExtra         func(*testing.T, *types.StateAccount)
		wantTrieHash        common.Hash
	}{
		{
			name: "vanilla geth",
			registerAndSetExtra: func(a *types.StateAccount) *types.StateAccount {
				return a
			},
			assertExtra: func(t *testing.T, a *types.StateAccount) {
				t.Helper()
				assert.Truef(t, a.Extra.Equal(nil), "%T.%T.IsEmpty()", a, a.Extra)
			},
			wantTrieHash: vanillaGeth,
		},
		{
			name: "true-boolean payload",
			registerAndSetExtra: func(a *types.StateAccount) *types.StateAccount {
				types.RegisterExtras[bool]().SetOnStateAccount(a, true)
				return a
			},
			assertExtra: func(t *testing.T, sa *types.StateAccount) {
				t.Helper()
				assert.Truef(t, types.ExtraPayloads[bool]{}.FromStateAccount(sa), "")
			},
			wantTrieHash: trueBool,
		},
		{
			name: "explicit false-boolean payload",
			registerAndSetExtra: func(a *types.StateAccount) *types.StateAccount {
				p := types.RegisterExtras[bool]()
				p.SetOnStateAccount(a, false) // the explicit part
				return a
			},
			assertExtra: func(t *testing.T, sa *types.StateAccount) {
				t.Helper()
				assert.Falsef(t, types.ExtraPayloads[bool]{}.FromStateAccount(sa), "")
			},
			wantTrieHash: falseBool,
		},
		{
			name: "implicit false-boolean payload",
			registerAndSetExtra: func(a *types.StateAccount) *types.StateAccount {
				types.RegisterExtras[bool]()
				// Note that `a` is reflected, unchanged (the implicit part).
				return a
			},
			assertExtra: func(t *testing.T, sa *types.StateAccount) {
				t.Helper()
				assert.Falsef(t, types.ExtraPayloads[bool]{}.FromStateAccount(sa), "")
			},
			wantTrieHash: falseBool,
		},
		{
			name: "arbitrary payload",
			registerAndSetExtra: func(a *types.StateAccount) *types.StateAccount {
				p := arbitraryPayload{arbitraryData}
				types.RegisterExtras[arbitraryPayload]().SetOnStateAccount(a, p)
				return a
			},
			assertExtra: func(t *testing.T, sa *types.StateAccount) {
				t.Helper()
				got := types.ExtraPayloads[arbitraryPayload]{}.FromStateAccount(sa)
				assert.Equalf(t, arbitraryPayload{arbitraryData}, got, "")
			},
			wantTrieHash: arbitrary,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			types.TestOnlyClearRegisteredExtras()
			t.Cleanup(types.TestOnlyClearRegisteredExtras)

			acct := tt.registerAndSetExtra(&types.StateAccount{
				Nonce:    42,
				Balance:  uint256.NewInt(314159),
				Root:     types.EmptyRootHash,
				CodeHash: types.EmptyCodeHash[:],
			})

			db := triedb.NewDatabase(rawdb.NewMemoryDatabase(), nil)
			id := trie.TrieID(types.EmptyRootHash)
			state, err := trie.NewStateTrie(id, db)
			require.NoError(t, err, "trie.NewStateTrie(types.EmptyRootHash, ...)")

			require.NoErrorf(t, state.UpdateAccount(addr, acct), "%T.UpdateAccount(...)", state)
			assert.Equalf(t, tt.wantTrieHash, state.Hash(), "%T.Hash() after UpdateAccount()", state)

			got, err := state.GetAccount(addr)
			require.NoError(t, err, "state.GetAccount({account updated earlier})")
			if diff := cmp.Diff(acct, got); diff != "" {
				t.Errorf("%T.GetAccount() not equal to value passed to %[1]T.UpdateAccount(); diff (-want +got):\n%s", state, diff)
			}
			tt.assertExtra(t, got)
		})
	}
}
