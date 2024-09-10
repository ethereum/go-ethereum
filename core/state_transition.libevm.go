package core

// canExecuteTransaction is a convenience wrapper for calling the
// [params.RulesHooks.CanExecuteTransaction] hook.
func (st *StateTransition) canExecuteTransaction() error {
	bCtx := st.evm.Context
	rules := st.evm.ChainConfig().Rules(bCtx.BlockNumber, bCtx.Random != nil, bCtx.Time)
	return rules.Hooks().CanExecuteTransaction(st.msg.From, st.msg.To, st.state)
}
