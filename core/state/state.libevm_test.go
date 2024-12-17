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

package state_test

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ava-labs/libevm/common"
	"github.com/ava-labs/libevm/core/rawdb"
	"github.com/ava-labs/libevm/core/state"
	"github.com/ava-labs/libevm/core/state/snapshot"
	"github.com/ava-labs/libevm/core/types"
	"github.com/ava-labs/libevm/ethdb/memorydb"
	"github.com/ava-labs/libevm/libevm/ethtest"
	"github.com/ava-labs/libevm/triedb"
)

func TestGetSetExtra(t *testing.T) {
	type accountExtra struct {
		// Data is a pointer to test deep copying.
		Data *[]byte // MUST be exported; I spent 20 minutes investigating failing tests because I'm an idiot
	}

	types.TestOnlyClearRegisteredExtras()
	t.Cleanup(types.TestOnlyClearRegisteredExtras)
	// Just as its Data field is a pointer, the registered type is a pointer to
	// test deep copying.
	payloads := types.RegisterExtras[types.NOOPHeaderHooks, *types.NOOPHeaderHooks, *accountExtra]().StateAccount

	rng := ethtest.NewPseudoRand(42)
	addr := rng.Address()
	nonce := rng.Uint64()
	balance := rng.Uint256()
	buf := rng.Bytes(8)
	extra := &accountExtra{Data: &buf}

	views := newWithSnaps(t)
	stateDB := views.newStateDB(t, types.EmptyRootHash)

	assert.Nilf(t, state.GetExtra(stateDB, payloads, addr), "state.GetExtra() returns zero-value %T if before account creation", extra)
	stateDB.CreateAccount(addr)
	stateDB.SetNonce(addr, nonce)
	stateDB.SetBalance(addr, balance)
	assert.Nilf(t, state.GetExtra(stateDB, payloads, addr), "state.GetExtra() returns zero-value %T if after account creation but before SetExtra()", extra)
	state.SetExtra(stateDB, payloads, addr, extra)
	require.Equal(t, extra, state.GetExtra(stateDB, payloads, addr), "state.GetExtra() immediately after SetExtra()")

	root, err := stateDB.Commit(1, false) // arbitrary block number
	require.NoErrorf(t, err, "%T.Commit(1, false)", stateDB)
	require.NotEqualf(t, types.EmptyRootHash, root, "root hash returned by %T.Commit() is not the empty root", stateDB)

	t.Run(fmt.Sprintf("retrieve from %T", views.snaps), func(t *testing.T) {
		iter, err := views.snaps.AccountIterator(root, common.Hash{})
		require.NoErrorf(t, err, "%T.AccountIterator(...)", views.snaps)
		defer iter.Release()

		require.Truef(t, iter.Next(), "%T.Next() (i.e. at least one account)", iter)
		require.NoErrorf(t, iter.Error(), "%T.Error()", iter)

		t.Run("types.FullAccount()", func(t *testing.T) {
			got, err := types.FullAccount(iter.Account())
			require.NoErrorf(t, err, "types.FullAccount(%T.Account())", iter)

			want := &types.StateAccount{
				Nonce:    nonce,
				Balance:  balance,
				Root:     types.EmptyRootHash,
				CodeHash: types.EmptyCodeHash[:],
			}
			payloads.Set(want, extra)

			if diff := cmp.Diff(want, got); diff != "" {
				t.Errorf("types.FullAccount(%T.Account()) diff (-want +got):\n%s", iter, diff)
			}
		})

		require.Falsef(t, iter.Next(), "%T.Next() after first account (i.e. only one)", iter)
	})

	t.Run(fmt.Sprintf("retrieve from new %T", stateDB), func(t *testing.T) {
		s := views.newStateDB(t, root)
		assert.Equalf(t, nonce, s.GetNonce(addr), "%T.GetNonce()", s)
		assert.Equalf(t, balance, s.GetBalance(addr), "%T.GetBalance()", s)
		assert.Equal(t, extra, state.GetExtra(s, payloads, addr), "state.GetExtra()")
	})

	t.Run("reverting to snapshot", func(t *testing.T) {
		s := views.newStateDB(t, root)
		snap := s.Snapshot()

		oldExtra := extra
		buf := append(*oldExtra.Data, rng.Bytes(8)...)
		newExtra := &accountExtra{Data: &buf}

		state.SetExtra(s, payloads, addr, newExtra)
		assert.Equalf(t, newExtra, state.GetExtra(s, payloads, addr), "state.GetExtra() after overwriting with new value")
		s.RevertToSnapshot(snap)
		assert.Equalf(t, oldExtra, state.GetExtra(s, payloads, addr), "state.GetExtra() after reverting to snapshot")
	})

	t.Run(fmt.Sprintf("%T.Copy()", stateDB), func(t *testing.T) {
		require.Equalf(t, reflect.Pointer, reflect.TypeOf(extra).Kind(), "extra-payload type")
		require.Equalf(t, reflect.Pointer, reflect.TypeOf(extra.Data).Kind(), "extra-payload field")

		orig := views.newStateDB(t, root)
		cp := orig.Copy()

		oldExtra := extra
		buf := append(*oldExtra.Data, rng.Bytes(8)...)
		newExtra := &accountExtra{Data: &buf}

		assert.Equalf(t, oldExtra, state.GetExtra(orig, payloads, addr), "GetExtra([original %T]) before setting", orig)
		assert.Equalf(t, oldExtra, state.GetExtra(cp, payloads, addr), "GetExtra([copy of %T]) returns the same payload", orig)
		state.SetExtra(orig, payloads, addr, newExtra)
		assert.Equalf(t, newExtra, state.GetExtra(orig, payloads, addr), "GetExtra([original %T]) returns overwritten payload", orig)
		assert.Equalf(t, oldExtra, state.GetExtra(cp, payloads, addr), "GetExtra([copy of %T]) returns original payload despite overwriting on original", orig)
	})
}

// stateViews are different ways to access the same data.
type stateViews struct {
	snaps    *snapshot.Tree
	database state.Database
}

func (v stateViews) newStateDB(t *testing.T, root common.Hash) *state.StateDB {
	t.Helper()
	s, err := state.New(root, v.database, v.snaps)
	require.NoError(t, err, "state.New()")
	return s
}

func newWithSnaps(t *testing.T) stateViews {
	t.Helper()
	empty := types.EmptyRootHash
	kvStore := memorydb.New()
	ethDB := rawdb.NewDatabase(kvStore)
	snaps, err := snapshot.New(
		snapshot.Config{
			CacheSize: 16, // Mb (arbitrary but non-zero)
		},
		kvStore,
		triedb.NewDatabase(ethDB, nil),
		empty,
	)
	require.NoError(t, err, "snapshot.New()")

	return stateViews{
		snaps:    snaps,
		database: state.NewDatabase(ethDB),
	}
}
