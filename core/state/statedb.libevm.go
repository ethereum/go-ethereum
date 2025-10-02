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
	"github.com/ava-labs/libevm/libevm/register"
	"github.com/ava-labs/libevm/libevm/stateconf"
)

// TxHash returns the current transaction hash set by [StateDB.SetTxContext].
func (s *StateDB) TxHash() common.Hash {
	return s.thash
}

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

// StateDBHooks modify the behaviour of [StateDB] instances.
type StateDBHooks interface {
	// TransformStateKey receives the arguments passed to [StateDB.GetState],
	// [StateDB.GetCommittedState] or [StateDB.SetState], and returns the key
	// that each of those methods will use for accessing state. This method will
	// not, however, be called if any of the aforementioned [StateDB] methods
	// receives a [stateconf.SkipStateKeyTransformation] option.
	//
	// This method SHOULD NOT be used for anything other than achieving
	// backwards compatibility with an existing chain. In the event that other
	// methods are added to the [StateDBHooks] interface and no key
	// transformation is required, it is acceptable for this method to echo the
	// [common.Hash], unchanged.
	TransformStateKey(_ common.Address, key common.Hash) (newKey common.Hash)
}

// RegisterExtras registers the [StateDBHooks] such that they modify the
// behaviour of all [StateDB] instances. It is expected to be called in an
// `init()` function and MUST NOT be called more than once.
func RegisterExtras(s StateDBHooks) {
	registeredExtras.MustRegister(s)
}

// WithTempRegisteredExtras temporarily registers `s` as if calling
// [RegisterExtras] the same type parameter. After `fn` returns, the
// registration is returned to its former state, be that none or the types
// originally passed to [RegisterExtras].
//
// This MUST NOT be used on a live chain. It is solely intended for off-chain
// consumers that require access to extras. Said consumers SHOULD NOT, however
// call this function directly. Use the libevm/temporary.WithRegisteredExtras()
// function instead as it atomically overrides all possible packages.
func WithTempRegisteredExtras(s StateDBHooks, fn func()) {
	registeredExtras.TempOverride(s, fn)
}

// TestOnlyClearRegisteredExtras clears the arguments previously passed to
// [RegisterExtras]. It panics if called from a non-testing call stack.
//
// In tests it SHOULD be called before every call to [RegisterExtras] and then
// defer-called afterwards, either directly or via testing.TB.Cleanup(). This is
// a workaround for the single-call limitation on [RegisterExtras].
func TestOnlyClearRegisteredExtras() {
	registeredExtras.TestOnlyClear()
}

var registeredExtras register.AtMostOnce[StateDBHooks]

func transformStateKey(addr common.Address, key common.Hash, opts ...stateconf.StateDBStateOption) common.Hash {
	r := &registeredExtras
	if !r.Registered() || !stateconf.ShouldTransformStateKey(opts...) {
		return key
	}
	return r.Get().TransformStateKey(addr, key)
}
