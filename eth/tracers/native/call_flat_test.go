package native_test

import (
	"encoding/json"
	"errors"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/tracing"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/eth/tracers"
	"github.com/ethereum/go-ethereum/params"
	"github.com/stretchr/testify/require"
)

func TestCallFlatStop(t *testing.T) {

	ctx := &tracers.Context{}

	tracer, err := tracers.DefaultDirectory.New("flatCallTracer", ctx, json.RawMessage(`{}`))
	require.NoError(t, err)

	// this error should be returned by GetResult
	stopError := errors.New("stop error")

	// simulate a transaction
	tx := types.NewTransaction(0, common.Address{}, big.NewInt(0), 0, big.NewInt(0), nil)

	tracer.OnTxStart(&tracing.VMContext{
		ChainConfig: params.MainnetChainConfig,
	}, tx, common.Address{})

	tracer.OnEnter(0, byte(vm.CALL), common.Address{}, common.Address{}, nil, 0, big.NewInt(0))

	// stop before the transaction is finished
	tracer.Stop(stopError)

	tracer.OnTxEnd(&types.Receipt{GasUsed: 0}, nil)

	// check that the error is returned by GetResult
	_, tracerError := tracer.GetResult()
	require.Equal(t, stopError, tracerError)
}
