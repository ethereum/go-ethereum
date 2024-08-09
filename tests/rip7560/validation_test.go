package rip7560

import (
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/tests"
	"github.com/status-im/keycard-go/hexutils"
	"github.com/stretchr/testify/assert"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

func TestPackValidationData(t *testing.T) {
	// --------------- after 6bytes     before 6 bytes   magic 20 bytes
	validationData := "000000000002" + "000000000001" + "0000000000000000000000000000000000001234"
	packed, _ := new(big.Int).SetString(validationData, 16)
	assert.Equal(t, packed.Text(16), new(big.Int).SetBytes(core.PackValidationData(0x1234, 1, 2)).Text(16))
}

// func TestUnpackValidationData(t *testing.T) {
// 	packed := core.PackValidationData(0xdead, 0xcafe, 0xface)
// 	magic, until, after := core.UnpackValidationData(packed)
// 	assert.Equal(t, []uint64{0xdead, 0xcafe, 0xface}, []uint64{magic, until, after})
// }

func TestValidationFailure_OOG(t *testing.T) {

	handleTransaction(newTestContextBuilder(t).withCode(DEFAULT_SENDER, createAccountCode(), DEFAULT_BALANCE), types.Rip7560AccountAbstractionTx{
		ValidationGasLimit: uint64(1),
		GasFeeCap:          big.NewInt(1000000000),
	}, "out of gas")
}

func TestValidationFailure_no_balance(t *testing.T) {

	handleTransaction(newTestContextBuilder(t).withCode(DEFAULT_SENDER, createAccountCode(), 1), types.Rip7560AccountAbstractionTx{
		ValidationGasLimit: uint64(1),
		GasFeeCap:          big.NewInt(1000000000),
	}, "insufficient funds for gas * price + value: address 0x1111111111222222222233333333334444444444 have 1 want 1000000000")
}

func TestValidationFailure_sigerror(t *testing.T) {
	handleTransaction(newTestContextBuilder(t).withCode(DEFAULT_SENDER, returnWithData(core.PackValidationData(core.SigFailAccountMethodSig, 0, 0)), DEFAULT_BALANCE), types.Rip7560AccountAbstractionTx{
		ValidationGasLimit: uint64(1000000000),
		GasFeeCap:          big.NewInt(1000000000),
	}, "account signature error")
}

func TestValidationFailure_validAfter(t *testing.T) {

	handleTransaction(newTestContextBuilder(t).withCode(DEFAULT_SENDER,
		returnWithData(core.PackValidationData(core.AcceptAccountMethodSig, 300, 200)), DEFAULT_BALANCE), types.Rip7560AccountAbstractionTx{
		ValidationGasLimit: uint64(1000000000),
		GasFeeCap:          big.NewInt(1000000000),
	}, "RIP-7560 transaction validity not reached yet")
}

func TestValidationFailure_validUntil(t *testing.T) {

	handleTransaction(newTestContextBuilder(t).withCode(DEFAULT_SENDER,
		returnWithData(core.PackValidationData(core.AcceptAccountMethodSig, 1, 0)), DEFAULT_BALANCE), types.Rip7560AccountAbstractionTx{
		ValidationGasLimit: uint64(1000000000),
		GasFeeCap:          big.NewInt(1000000000),
	}, "RIP-7560 transaction validity expired")
}

func TestValidation_ok(t *testing.T) {

	handleTransaction(newTestContextBuilder(t).withCode(DEFAULT_SENDER, createAccountCode(), DEFAULT_BALANCE), types.Rip7560AccountAbstractionTx{
		ValidationGasLimit: uint64(1000000000),
		GasFeeCap:          big.NewInt(1000000000),
	}, "ok")
}

func TestValidation_ok_paid(t *testing.T) {

	aatx := types.Rip7560AccountAbstractionTx{
		ValidationGasLimit: uint64(1000000000),
		GasFeeCap:          big.NewInt(1000000000),
	}
	tb := newTestContextBuilder(t).withCode(DEFAULT_SENDER, createAccountCode(), DEFAULT_BALANCE)
	handleTransaction(tb, aatx, "ok")

	maxCost := new(big.Int).SetUint64(aatx.ValidationGasLimit + aatx.PaymasterValidationGasLimit + aatx.Gas)
	maxCost.Mul(maxCost, aatx.GasFeeCap)
}

func TestValidationFailure_account_nonce(t *testing.T) {
	handleTransaction(newTestContextBuilder(t).withCode(DEFAULT_SENDER, createAccountCode(), DEFAULT_BALANCE), types.Rip7560AccountAbstractionTx{
		Nonce:              1234,
		ValidationGasLimit: uint64(1000000000),
		GasFeeCap:          big.NewInt(1000000000),
	}, "nonce too high: address 0x1111111111222222222233333333334444444444, tx: 1234 state: 0")
}

func TestValidationFailure_account_revert(t *testing.T) {
	handleTransaction(newTestContextBuilder(t).withCode(DEFAULT_SENDER,
		createCode(vm.PUSH0, vm.DUP1, vm.REVERT), DEFAULT_BALANCE), types.Rip7560AccountAbstractionTx{
		ValidationGasLimit: uint64(1000000000),
		GasFeeCap:          big.NewInt(1000000000),
	}, "execution reverted")
}

func TestValidationFailure_account_revert_with_reason(t *testing.T) {
	// cast abi-encode 'Error(string)' hello
	reason := hexutils.HexToBytes("0000000000000000000000000000000000000000000000000000000000000020000000000000000000000000000000000000000000000000000000000000000568656c6c6f000000000000000000000000000000000000000000000000000000")
	handleTransaction(newTestContextBuilder(t).withCode(DEFAULT_SENDER,
		revertWithData(reason), DEFAULT_BALANCE), types.Rip7560AccountAbstractionTx{
		ValidationGasLimit: uint64(1000000000),
		GasFeeCap:          big.NewInt(1000000000),
	}, "execution reverted")
}

func TestValidationFailure_account_wrong_return_length(t *testing.T) {
	handleTransaction(newTestContextBuilder(t).withCode(DEFAULT_SENDER,
		returnWithData([]byte{1, 2, 3}), DEFAULT_BALANCE), types.Rip7560AccountAbstractionTx{
		ValidationGasLimit: uint64(1000000000),
		GasFeeCap:          big.NewInt(1000000000),
	}, "invalid account return data length")
}

func TestValidationFailure_account_no_return_value(t *testing.T) {
	handleTransaction(newTestContextBuilder(t).withCode(DEFAULT_SENDER,
		returnWithData([]byte{}), DEFAULT_BALANCE), types.Rip7560AccountAbstractionTx{
		ValidationGasLimit: uint64(1000000000),
		GasFeeCap:          big.NewInt(1000000000),
	}, "invalid account return data length")
}

func TestValidationFailure_account_wrong_return_value(t *testing.T) {
	// create buffer of 32 byte array
	handleTransaction(newTestContextBuilder(t).withCode(DEFAULT_SENDER,
		returnWithData(make([]byte, 32)),
		DEFAULT_BALANCE), types.Rip7560AccountAbstractionTx{
		ValidationGasLimit: uint64(1000000000),
		GasFeeCap:          big.NewInt(1000000000),
	}, "account did not return correct MAGIC_VALUE")
}

func handleTransaction(tb *testContextBuilder, aatx types.Rip7560AccountAbstractionTx, expectedErr string) {
	t := tb.build()
	if aatx.Sender == nil {
		//pre-deployed sender account
		Sender := common.HexToAddress(DEFAULT_SENDER)
		aatx.Sender = &Sender
	}
	tx := types.NewTx(&aatx)

	var state = tests.MakePreState(rawdb.NewMemoryDatabase(), t.genesisAlloc, false, rawdb.HashScheme)
	defer state.Close()

	state.StateDB.SetTxContext(tx.Hash(), 0)
	_, _, _, err := core.HandleRip7560Transactions([]*types.Transaction{tx}, 0, state.StateDB, &common.Address{}, t.genesisBlock.Header(), t.gaspool, t.genesis.Config, t.chainContext, vm.Config{})

	errStr := "ok"
	if err != nil {
		errStr = err.Error()
	}
	assert.Equal(t.t, expectedErr, errStr)
}

//test failure on non-rip7560

//IntrinsicGas: for validation frame, should return the max possible gas.
// - execution should be "free" (and refund the excess)
// geth increment nonce before "call" our validation frame. (in ApplyMessage)
