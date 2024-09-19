package vm

import (
	"math/big"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/params"
)

type chainIDOverrider struct {
	chainID int64
}

func (o chainIDOverrider) OverrideNewEVMArgs(args *NewEVMArgs) *NewEVMArgs {
	args.ChainConfig = &params.ChainConfig{ChainID: big.NewInt(o.chainID)}
	return args
}

func TestOverrideNewEVMArgs(t *testing.T) {
	// The overrideNewEVMArgs function accepts and returns all arguments to
	// NewEVM(), in order. Here we lock in our assumption of that order. If this
	// breaks then all functionality overriding the args MUST be updated.
	var _ func(BlockContext, TxContext, StateDB, *params.ChainConfig, Config) *EVM = NewEVM

	const chainID = 13579
	libevmHooks = nil
	RegisterHooks(chainIDOverrider{chainID: chainID})
	defer func() { libevmHooks = nil }()

	got := NewEVM(BlockContext{}, TxContext{}, nil, nil, Config{}).ChainConfig().ChainID
	require.Equal(t, big.NewInt(chainID), got)
}
