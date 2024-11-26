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
	"reflect"

	"github.com/ava-labs/libevm/common"
	"github.com/ava-labs/libevm/core/state/snapshot"
	"github.com/ava-labs/libevm/libevm/stateconf"
)

// SnapshotTree mirrors the functionality of a [snapshot.Tree], allowing for
// drop-in replacements. This is intended as a temporary feature as a workaround
// until a standard Tree can be used.
type SnapshotTree interface {
	Cap(common.Hash, int) error
	Snapshot(common.Hash) snapshot.Snapshot
	StorageIterator(root, account, seek common.Hash) (snapshot.StorageIterator, error)
	Update(
		blockRoot common.Hash,
		parentRoot common.Hash,
		destructs map[common.Hash]struct{},
		accounts map[common.Hash][]byte,
		storage map[common.Hash]map[common.Hash][]byte,
		opts ...stateconf.SnapshotUpdateOption,
	) error
}

var _ SnapshotTree = (*snapshot.Tree)(nil)

// clearTypedNilPointer returns nil if `snaps == nil` or if it holds a nil
// pointer. The default geth behaviour expected a [snapshot.Tree] pointer
// instead of a SnapshotTree interface, which could result in typed-nil bugs.
func clearTypedNilPointer(snaps SnapshotTree) SnapshotTree {
	if v := reflect.ValueOf(snaps); v.Kind() == reflect.Pointer && v.IsNil() {
		return nil
	}
	return snaps
}
