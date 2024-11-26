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

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ava-labs/libevm/common"
	"github.com/ava-labs/libevm/core/rawdb"
	"github.com/ava-labs/libevm/core/state/snapshot"
	"github.com/ava-labs/libevm/core/types"
	"github.com/ava-labs/libevm/libevm/stateconf"
)

func TestStateDBCommitPropagatesOptions(t *testing.T) {
	var rec snapTreeRecorder
	sdb, err := New(types.EmptyRootHash, NewDatabase(rawdb.NewMemoryDatabase()), &rec)
	require.NoError(t, err, "New()")

	// Ensures that rec.Update() will be called.
	sdb.SetNonce(common.Address{}, 42)

	const payload = "hello world"
	opt := stateconf.WithUpdatePayload(payload)
	_, err = sdb.Commit(0, false, opt)
	require.NoErrorf(t, err, "%T.Commit(..., %T)", sdb, opt)

	assert.Equalf(t, payload, rec.gotPayload, "%T payload propagated via %T.Commit() to %T.Update()", opt, sdb, rec)
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
	r.gotPayload = stateconf.ExtractUpdatePayload(opts...)
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
