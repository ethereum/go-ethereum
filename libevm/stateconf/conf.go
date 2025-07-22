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

// Package stateconf configures state management.
package stateconf

import (
	"github.com/ava-labs/libevm/common"
	"github.com/ava-labs/libevm/libevm/options"
)

// A StateDBCommitOption configures the behaviour of state.StateDB.Commit()
type StateDBCommitOption = options.Option[stateDBCommitConfig]

type stateDBCommitConfig struct {
	snapshotOpts []SnapshotUpdateOption
	triedbOpts   []TrieDBUpdateOption
}

// WithSnapshotUpdateOpts returns a StateDBCommitOption carrying a list of
// SnapshotUpdateOptions.
// If multiple such options are used, only the last will be applied as they overwrite each other.
func WithSnapshotUpdateOpts(opts ...SnapshotUpdateOption) StateDBCommitOption {
	return options.Func[stateDBCommitConfig](func(c *stateDBCommitConfig) {
		c.snapshotOpts = opts
	})
}

// ExtractSnapshotUpdateOpts returns the list of SnapshotUpdateOptions carried
// by the provided slice of StateDBCommitOption.
func ExtractSnapshotUpdateOpts(opts ...StateDBCommitOption) []SnapshotUpdateOption {
	return options.As(opts...).snapshotOpts
}

// WithTrieDBUpdateOpts returns a StateDBCommitOption carrying a list of
// TrieDBUpdateOptions. If multiple such options are used, only the last will be
// applied as they overwrite each other.
func WithTrieDBUpdateOpts(opts ...TrieDBUpdateOption) StateDBCommitOption {
	return options.Func[stateDBCommitConfig](func(c *stateDBCommitConfig) {
		c.triedbOpts = opts
	})
}

// ExtractTrieDBUpdateOpts returns the list of TrieDBUpdateOptions carried by
// the provided slice of StateDBCommitOption.
func ExtractTrieDBUpdateOpts(opts ...StateDBCommitOption) []TrieDBUpdateOption {
	return options.As(opts...).triedbOpts
}

// A SnapshotUpdateOption configures the behaviour of
// state.SnapshotTree.Update() implementations. This will be removed along with
// state.SnapshotTree.
type SnapshotUpdateOption = options.Option[snapshotUpdateConfig]

type snapshotUpdateConfig struct {
	payload any
}

// WithSnapshotUpdatePayload returns a SnapshotUpdateOption carrying an arbitrary
// payload. It acts only as a carrier to exploit existing function plumbing and
// the effect on behaviour is left to the implementation receiving it.
func WithSnapshotUpdatePayload(p any) SnapshotUpdateOption {
	return options.Func[snapshotUpdateConfig](func(c *snapshotUpdateConfig) {
		c.payload = p
	})
}

// ExtractSnapshotUpdatePayload returns the payload carried by a [WithSnapshotUpdatePayload]
// option. Only one such option can be used at once; behaviour is otherwise
// undefined.
func ExtractSnapshotUpdatePayload(opts ...SnapshotUpdateOption) any {
	return options.As(opts...).payload
}

// A TrieDBUpdateOption configures the behaviour of triedb.Database.Update() implementations.
type TrieDBUpdateOption = options.Option[triedbUpdateConfig]

type triedbUpdateConfig struct {
	parentBlockHash  *common.Hash
	currentBlockHash *common.Hash
}

// WithTrieDBUpdatePayload returns a TrieDBUpdateOption carrying two block hashes.
// It acts only as a carrier to exploit existing function plumbing and
// the effect on behaviour is left to the implementation receiving it.
func WithTrieDBUpdatePayload(parent common.Hash, current common.Hash) TrieDBUpdateOption {
	return options.Func[triedbUpdateConfig](func(c *triedbUpdateConfig) {
		c.parentBlockHash = &parent
		c.currentBlockHash = &current
	})
}

// ExtractTrieDBUpdatePayload returns the payload carried by a [WithTrieDBUpdatePayload]
// option. Only one such option can be used at once; behaviour is otherwise
// undefined.
func ExtractTrieDBUpdatePayload(opts ...TrieDBUpdateOption) (common.Hash, common.Hash, bool) {
	conf := options.As(opts...)
	if conf.parentBlockHash == nil && conf.currentBlockHash == nil {
		return common.Hash{}, common.Hash{}, false
	}
	return *conf.parentBlockHash, *conf.currentBlockHash, true
}

// A StateDBStateOption configures the behaviour of state.StateDB methods for
// getting and setting state: GetState(), GetCommittedState(), and SetState().
type StateDBStateOption = options.Option[stateDBStateConfig]

type stateDBStateConfig struct {
	skipKeyTransformation bool
}

// SkipStateKeyTransformation causes any registered state-key transformation
// hook to be ignored. See state.RegisterExtras() for details.
func SkipStateKeyTransformation() StateDBStateOption {
	return options.Func[stateDBStateConfig](func(c *stateDBStateConfig) {
		c.skipKeyTransformation = true
	})
}

// ShouldTransformStateKey parses the options, returning whether or not any
// registered state-key transformation hook should be used; i.e. it returns
// `true` i.f.f. there are no [SkipStateKeyTransformation] options in the
// arguments.
func ShouldTransformStateKey(opts ...StateDBStateOption) bool {
	return !options.As(opts...).skipKeyTransformation
}
