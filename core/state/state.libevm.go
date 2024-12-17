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
	"github.com/ava-labs/libevm/common"
	"github.com/ava-labs/libevm/core/types"
	"github.com/ava-labs/libevm/libevm/pseudo"
)

// GetExtra returns the extra payload from the [types.StateAccount] associated
// with the address, or a zero-value `SA` if not found. The [pseudo.Accessor]
// MUST be sourced from [types.RegisterExtras].
func GetExtra[SA any](s *StateDB, a pseudo.Accessor[types.StateOrSlimAccount, SA], addr common.Address) SA {
	stateObject := s.getStateObject(addr)
	if stateObject != nil {
		return a.Get(&stateObject.data)
	}
	var zero SA
	return zero
}

// SetExtra sets the extra payload for the address. See [GetExtra] for details.
func SetExtra[SA any](s *StateDB, a pseudo.Accessor[types.StateOrSlimAccount, SA], addr common.Address, extra SA) {
	stateObject := s.getOrNewStateObject(addr)
	if stateObject != nil {
		setExtraOnObject(stateObject, a, addr, extra)
	}
}

func setExtraOnObject[SA any](s *stateObject, a pseudo.Accessor[types.StateOrSlimAccount, SA], addr common.Address, extra SA) {
	s.db.journal.append(extraChange[SA]{
		accessor: a,
		account:  &addr,
		prev:     a.Get(&s.data),
	})
	a.Set(&s.data, extra)
}

// extraChange is a [journalEntry] for [SetExtra] / [setExtraOnObject].
type extraChange[SA any] struct {
	accessor pseudo.Accessor[types.StateOrSlimAccount, SA]
	account  *common.Address
	prev     SA
}

func (e extraChange[SA]) dirtied() *common.Address { return e.account }

func (e extraChange[SA]) revert(s *StateDB) {
	e.accessor.Set(&s.getStateObject(*e.account).data, e.prev)
}
