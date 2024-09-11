package core_test

import (
	"fmt"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/libevm"
	"github.com/ethereum/go-ethereum/libevm/ethtest"
	"github.com/ethereum/go-ethereum/libevm/hookstest"
	"github.com/stretchr/testify/require"
)

func TestCanExecuteTransaction(t *testing.T) {
	rng := ethtest.NewPseudoRand(42)
	account := rng.Address()
	slot := rng.Hash()

	makeErr := func(from common.Address, to *common.Address, val common.Hash) error {
		return fmt.Errorf("From: %v To: %v State: %v", from, to, val)
	}
	hooks := &hookstest.Stub{
		CanExecuteTransactionFn: func(from common.Address, to *common.Address, s libevm.StateReader) error {
			return makeErr(from, to, s.GetState(account, slot))
		},
	}
	hooks.Register(t)

	value := rng.Hash()

	state, evm := ethtest.NewZeroEVM(t)
	state.SetState(account, slot, value)
	msg := &core.Message{
		From: rng.Address(),
		To:   rng.AddressPtr(),
	}
	_, err := core.ApplyMessage(evm, msg, new(core.GasPool).AddGas(30e6))
	require.EqualError(t, err, makeErr(msg.From, msg.To, value).Error())
}
