package vm

import "github.com/ethereum/go-ethereum/params"

// RegisterHooks registers the Hooks. It is expected to be called in an `init()`
// function and MUST NOT be called more than once.
func RegisterHooks(h Hooks) {
	if libevmHooks != nil {
		panic("already registered")
	}
	libevmHooks = h
}

var libevmHooks Hooks

// Hooks are arbitrary configuration functions to modify default VM behaviour.
type Hooks interface {
	OverrideNewEVMArgs(*NewEVMArgs) *NewEVMArgs
}

// NewEVMArgs are the arguments received by [NewEVM], available for override.
type NewEVMArgs struct {
	BlockContext BlockContext
	TxContext    TxContext
	StateDB      StateDB
	ChainConfig  *params.ChainConfig
	Config       Config
}

func overrideNewEVMArgs(
	blockCtx BlockContext,
	txCtx TxContext,
	statedb StateDB,
	chainConfig *params.ChainConfig,
	config Config,
) (BlockContext, TxContext, StateDB, *params.ChainConfig, Config) {
	if libevmHooks == nil {
		return blockCtx, txCtx, statedb, chainConfig, config
	}
	args := libevmHooks.OverrideNewEVMArgs(&NewEVMArgs{blockCtx, txCtx, statedb, chainConfig, config})
	return args.BlockContext, args.TxContext, args.StateDB, args.ChainConfig, args.Config
}
