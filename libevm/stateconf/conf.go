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

// Package stateconf configures state management.
package stateconf

import "github.com/ava-labs/libevm/libevm/options"

// A SnapshotUpdateOption configures the behaviour of
// state.SnapshotTree.Update() implementations. This will be removed along with
// state.SnapshotTree.
type SnapshotUpdateOption = options.Option[snapshotUpdateConfig]

type snapshotUpdateConfig struct {
	payload any
}

// WithUpdatePayload returns a SnapshotUpdateOption carrying an arbitrary
// payload. It acts only as a carrier to exploit existing function plumbing and
// the effect on behaviour is left to the implementation receiving it.
func WithUpdatePayload(p any) SnapshotUpdateOption {
	return options.Func[snapshotUpdateConfig](func(c *snapshotUpdateConfig) {
		c.payload = p
	})
}

// ExtractUpdatePayload returns the payload carried by a [WithUpdatePayload]
// option. Only one such option can be used at once; behaviour is otherwise
// undefined.
func ExtractUpdatePayload(opts ...SnapshotUpdateOption) any {
	return options.As(opts...).payload
}
