package rip7560

import (
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/tests"
	"github.com/stretchr/testify/assert"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

func TestValidation_OOG(t *testing.T) {
	magic := big.NewInt(0xbf45c166)
	magic.Lsh(magic, 256-32)

	validatePhase(newTestContextBuilder(t).withCode(DEFAULT_SENDER, returnData(magic.Bytes()), 0), types.Rip7560AccountAbstractionTx{
		ValidationGas: uint64(1),
		GasFeeCap:     big.NewInt(1000000000),
	}, "out of gas")
}

func TestValidation_ok(t *testing.T) {
	magic := big.NewInt(0xbf45c166)
	magic.Lsh(magic, 256-32)

	validatePhase(newTestContextBuilder(t).withCode(DEFAULT_SENDER, returnData(magic.Bytes()), 0), types.Rip7560AccountAbstractionTx{
		ValidationGas: uint64(1000000000),
		GasFeeCap:     big.NewInt(1000000000),
	}, "")
}

func TestValidation_account_revert(t *testing.T) {
	validatePhase(newTestContextBuilder(t).withCode(DEFAULT_SENDER,
		createCode(vm.PUSH1, 0, vm.DUP1, vm.REVERT), 0), types.Rip7560AccountAbstractionTx{
		ValidationGas: uint64(1000000000),
		GasFeeCap:     big.NewInt(1000000000),
	}, "execution reverted")
}

func TestValidation_account_no_return_value(t *testing.T) {
	validatePhase(newTestContextBuilder(t).withCode(DEFAULT_SENDER, []byte{
		byte(vm.PUSH1), 0, byte(vm.DUP1), byte(vm.RETURN),
	}, 0), types.Rip7560AccountAbstractionTx{
		ValidationGas: uint64(1000000000),
		GasFeeCap:     big.NewInt(1000000000),
	}, "invalid account return data length")
}

func TestValidation_account_wrong_return_value(t *testing.T) {
	validatePhase(newTestContextBuilder(t).withCode(DEFAULT_SENDER,
		returnData(createCode(1)),
		0), types.Rip7560AccountAbstractionTx{
		ValidationGas: uint64(1000000000),
		GasFeeCap:     big.NewInt(1000000000),
	}, "account did not return correct MAGIC_VALUE")
}

func validatePhase(tb *testContextBuilder, aatx types.Rip7560AccountAbstractionTx, expectedErr string) {
	t := tb.build()
	if aatx.Sender == nil {
		//pre-deployed sender account
		Sender := common.HexToAddress(DEFAULT_SENDER)
		aatx.Sender = &Sender
	}
	tx := types.NewTx(&aatx)

	var state = tests.MakePreState(rawdb.NewMemoryDatabase(), t.genesisAlloc, false, rawdb.HashScheme)
	defer state.Close()

	_, err := core.ApplyRip7560ValidationPhases(t.chainConfig, t.chainContext, &common.Address{}, t.gaspool, state.StateDB, t.genesisBlock.Header(), tx, vm.Config{})
	// err string or empty if nil
	errStr := ""
	if err != nil {
		errStr = err.Error()
	}
	assert.Equal(t.t, expectedErr, errStr)
}

//test failure on non-rip7560

//IntrinsicGas: for validation frame, should return the max possible gas.
// - execution should be "free" (and refund the excess)
// geth increment nonce before "call" our validation frame. (in ApplyMessage)
