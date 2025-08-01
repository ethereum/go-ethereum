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

package core

import (
	"fmt"

	"github.com/ava-labs/libevm/core/vm"
	"github.com/ava-labs/libevm/log"
	"github.com/ava-labs/libevm/params"
)

func (st *StateTransition) rulesHooks() params.RulesHooks {
	bCtx := st.evm.Context
	rules := st.evm.ChainConfig().Rules(bCtx.BlockNumber, bCtx.Random != nil, bCtx.Time)
	return rules.Hooks()
}

// NOTE: other than the final paragraph, the comment on
// [StateTransition.TransitionDb] is copied, verbatim, from the upstream
// version, which has been changed to [StateTransition.transitionDb] to allow
// its behaviour to be augmented.

// Keeps the vm package imported by this specific file so VS Code can support
// comments like [vm.EVM].
var _ = (*vm.EVM)(nil)

// TransitionDb will transition the state by applying the current message and
// returning the evm execution result with following fields.
//
//   - used gas: total gas used (including gas being refunded)
//   - returndata: the returned data from evm
//   - concrete execution error: various EVM errors which abort the execution, e.g.
//     ErrOutOfGas, ErrExecutionReverted
//
// However if any consensus issue encountered, return the error directly with
// nil evm execution result.
//
// libevm-specific behaviour: if, during execution, [vm.EVM.InvalidateExecution]
// is called with a non-nil error then said error will be returned, wrapped. All
// state transitions (e.g. nonce incrementing) will be reverted to a snapshot
// taken before execution.
func (st *StateTransition) TransitionDb() (*ExecutionResult, error) {
	if err := st.canExecuteTransaction(); err != nil {
		return nil, err
	}

	snap := st.state.Snapshot()   // computationally cheap operation
	res, err := st.transitionDb() // original geth implementation

	// [NOTE]: At the time of implementation of this libevm override, non-nil
	// values of `err` and `invalid` (below) are mutually exclusive. However, as
	// a defensive measure, we don't return early on non-nil `err` in case an
	// upstream update breaks this invariant.

	if invalid := st.evm.ExecutionInvalidated(); invalid != nil {
		st.state.RevertToSnapshot(snap)
		err = fmt.Errorf("execution invalidated: %w", invalid)
	}
	return res, err
}

// canExecuteTransaction is a convenience wrapper for calling the
// [params.RulesHooks.CanExecuteTransaction] hook.
func (st *StateTransition) canExecuteTransaction() error {
	hooks := st.rulesHooks()
	if err := hooks.CanExecuteTransaction(st.msg.From, st.msg.To, st.state); err != nil {
		log.Debug(
			"Transaction execution blocked by libevm hook",
			"from", st.msg.From,
			"to", st.msg.To,
			"hooks", log.TypeOf(hooks),
			"reason", err,
		)
		return err
	}
	return nil
}

// consumeMinimumGas updates the gas remaining to reflect the value returned by
// [params.RulesHooks.MinimumGasConsumption]. It MUST be called after all code
// that modifies gas consumption but before the balance is returned for
// remaining gas.
func (st *StateTransition) consumeMinimumGas() {
	limit := st.msg.GasLimit
	minConsume := min(
		limit, // as documented in [params.RulesHooks]
		st.rulesHooks().MinimumGasConsumption(limit),
	)
	st.gasRemaining = min(
		st.gasRemaining,
		limit-minConsume,
	)
}
