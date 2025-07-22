// Copyright 2024-2025 the libevm authors.
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

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ava-labs/libevm/common"
	"github.com/ava-labs/libevm/core/rawdb"
	"github.com/ava-labs/libevm/core/state/snapshot"
	"github.com/ava-labs/libevm/core/types"
	"github.com/ava-labs/libevm/ethdb"
	"github.com/ava-labs/libevm/libevm/stateconf"
	"github.com/ava-labs/libevm/trie"
	"github.com/ava-labs/libevm/trie/trienode"
	"github.com/ava-labs/libevm/trie/triestate"
	"github.com/ava-labs/libevm/triedb"
	"github.com/ava-labs/libevm/triedb/database"
	"github.com/ava-labs/libevm/triedb/hashdb"
)

func TestTxHash(t *testing.T) {
	db := NewDatabase(rawdb.NewMemoryDatabase())
	state, err := New(types.EmptyRootHash, db, nil)
	require.NoError(t, err)

	assert.Zero(t, state.TxHash(), "Tx hash should initially be uninitialized")

	hash := common.Hash{1}
	state.SetTxContext(hash, 3)
	assert.Equal(t, hash, state.TxHash(), "Tx hash should have been updated")
}

func TestStateDBCommitPropagatesOptions(t *testing.T) {
	memdb := rawdb.NewMemoryDatabase()
	trieRec := &triedbRecorder{Database: hashdb.New(memdb, nil, &trie.MerkleResolver{})}
	triedb := triedb.NewDatabase(
		memdb,
		&triedb.Config{
			DBOverride: func(_ ethdb.Database) triedb.DBOverride {
				return trieRec
			},
		},
	)
	var snapRec snapTreeRecorder
	sdb, err := New(types.EmptyRootHash, NewDatabaseWithNodeDB(memdb, triedb), &snapRec)
	require.NoError(t, err, "New()")

	// Ensures that rec.Update() will be called.
	sdb.SetNonce(common.Address{}, 42)

	const snapshotPayload = "hello world"
	var (
		parentHash  = common.HexToHash("0x0102030405060708090a0b0c0d0e0f1011121314151617181920212223242526")
		currentHash = common.HexToHash("0x1234567890123456789012345678901234567890123456789012345678901234")
	)
	snapshotOpt := stateconf.WithSnapshotUpdatePayload(snapshotPayload)
	triedbOpt := stateconf.WithTrieDBUpdatePayload(parentHash, currentHash)
	_, err = sdb.Commit(0, false, stateconf.WithSnapshotUpdateOpts(snapshotOpt), stateconf.WithTrieDBUpdateOpts(triedbOpt))

	require.NoErrorf(t, err, "%T.Commit(..., %T, %T)", sdb, snapshotOpt, triedbOpt)
	assert.Equalf(t, snapshotPayload, snapRec.gotPayload, "%T payload propagated via %T.Commit() to %T.Update()", snapshotOpt, sdb, snapRec)
	assert.Truef(t, trieRec.exists, "%T exists propagated via %T.Commit() to %T.Update()", triedbOpt, sdb, trieRec)
	assert.Equalf(t, parentHash, trieRec.parentBlockHash, "%T parentHash propagated via %T.Commit() to %T.Update()", triedbOpt, sdb, trieRec)
	assert.Equalf(t, currentHash, trieRec.currentBlockHash, "%T currentHash propagated via %T.Commit() to %T.Update()", triedbOpt, sdb, trieRec)
}

type snapTreeRecorder struct {
	SnapshotTree
	gotPayload any
}

func (*snapTreeRecorder) Cap(common.Hash, int) error {
	return nil
}

func (r *snapTreeRecorder) Update(
	_, _ common.Hash,
	_ map[common.Hash]struct{}, _ map[common.Hash][]byte, _ map[common.Hash]map[common.Hash][]byte,
	opts ...stateconf.SnapshotUpdateOption,
) error {
	r.gotPayload = stateconf.ExtractSnapshotUpdatePayload(opts...)
	return nil
}

func (*snapTreeRecorder) Snapshot(common.Hash) snapshot.Snapshot {
	return snapshotStub{}
}

type snapshotStub struct {
	snapshot.Snapshot
}

func (snapshotStub) Account(common.Hash) (*types.SlimAccount, error) {
	return &types.SlimAccount{}, nil
}

func (snapshotStub) Root() common.Hash {
	return common.Hash{}
}

type triedbRecorder struct {
	*hashdb.Database
	parentBlockHash  common.Hash
	currentBlockHash common.Hash
	exists           bool
}

func (r *triedbRecorder) Update(
	root common.Hash,
	parent common.Hash,
	block uint64,
	nodes *trienode.MergedNodeSet,
	states *triestate.Set,
	opts ...stateconf.TrieDBUpdateOption,
) error {
	r.parentBlockHash, r.currentBlockHash, r.exists = stateconf.ExtractTrieDBUpdatePayload(opts...)
	return r.Database.Update(root, parent, block, nodes, states)
}

func (r *triedbRecorder) Reader(_ common.Hash) (database.Reader, error) {
	return r.Database.Reader(common.Hash{})
}

type highByteFlipper struct{}

func flipHighByte(h common.Hash) common.Hash {
	h[0] = ^h[0]
	return h
}

func (highByteFlipper) TransformStateKey(_ common.Address, key common.Hash) common.Hash {
	return flipHighByte(key)
}

func TestTransformStateKey(t *testing.T) {
	rawdb := rawdb.NewMemoryDatabase()
	trie := triedb.NewDatabase(rawdb, nil)
	db := NewDatabaseWithNodeDB(rawdb, trie)
	sdb, err := New(types.EmptyRootHash, db, nil)
	require.NoErrorf(t, err, "New()")

	addr := common.Address{1}
	regularKey := common.Hash{0, 'k', 'e', 'y'}
	flippedKey := flipHighByte(regularKey)
	regularVal := common.Hash{'r', 'e', 'g', 'u', 'l', 'a', 'r'}
	flippedVal := common.Hash{'f', 'l', 'i', 'p', 'p', 'e', 'd'}

	sdb.SetState(addr, regularKey, regularVal)
	sdb.SetState(addr, flippedKey, flippedVal)

	assertEq := func(t *testing.T, key, want common.Hash, opts ...stateconf.StateDBStateOption) {
		t.Helper()
		assert.Equal(t, want, sdb.GetState(addr, key, opts...))
	}

	assertEq(t, regularKey, regularVal)
	assertEq(t, flippedKey, flippedVal)

	root, err := sdb.Commit(0, false)
	require.NoErrorf(t, err, "state.Commit()")

	err = trie.Commit(root, false)
	require.NoErrorf(t, err, "trie.Commit()")

	sdb, err = New(root, db, nil)
	require.NoErrorf(t, err, "New()")

	assertCommittedEq := func(t *testing.T, key, want common.Hash, opts ...stateconf.StateDBStateOption) {
		t.Helper()
		assert.Equal(t, want, sdb.GetCommittedState(addr, key, opts...))
	}

	assertEq(t, regularKey, regularVal)
	assertEq(t, flippedKey, flippedVal)
	assertCommittedEq(t, regularKey, regularVal)
	assertCommittedEq(t, flippedKey, flippedVal)

	// Typically the hook would be registered before any state access or
	// setting, but doing it here aids testing by showing the before-and-after
	// effects.
	RegisterExtras(highByteFlipper{})
	t.Cleanup(TestOnlyClearRegisteredExtras)

	noTransform := stateconf.SkipStateKeyTransformation()
	assertEq(t, regularKey, flippedVal)
	assertEq(t, regularKey, regularVal, noTransform)
	assertEq(t, flippedKey, regularVal)
	assertEq(t, flippedKey, flippedVal, noTransform)
	assertCommittedEq(t, regularKey, flippedVal)
	assertCommittedEq(t, regularKey, regularVal, noTransform)
	assertCommittedEq(t, flippedKey, regularVal)
	assertCommittedEq(t, flippedKey, flippedVal, noTransform)

	updatedVal := common.Hash{'u', 'p', 'd', 'a', 't', 'e', 'd'}
	sdb.SetState(addr, regularKey, updatedVal)
	assertEq(t, regularKey, updatedVal)
	assertEq(t, flippedKey, updatedVal, noTransform)
	assertCommittedEq(t, regularKey, flippedVal)
	assertCommittedEq(t, flippedKey, flippedVal, noTransform)
}
