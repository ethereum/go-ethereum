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

package core

// canExecuteTransaction is a convenience wrapper for calling the
// [params.RulesHooks.CanExecuteTransaction] hook.
func (st *StateTransition) canExecuteTransaction() error {
	bCtx := st.evm.Context
	rules := st.evm.ChainConfig().Rules(bCtx.BlockNumber, bCtx.Random != nil, bCtx.Time)
	return rules.Hooks().CanExecuteTransaction(st.msg.From, st.msg.To, st.state)
}
