package rip7560

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"math/big"
	"slices"
	"testing"
)

var DEFAULT_PAYMASTER = common.HexToAddress("0xaaaaaaaaaabbbbbbbbbbccccccccccdddddddddd")

func TestPaymasterValidationFailure_nobalance(t *testing.T) {

	handleTransaction(newTestContextBuilder(t).withCode(DEFAULT_SENDER, createAccountCode(), 0).
		withCode(DEFAULT_PAYMASTER.String(), createCode(vm.PUSH0, vm.DUP1, vm.REVERT), 1), types.Rip7560AccountAbstractionTx{
		ValidationGas: 1000000000,
		GasFeeCap:     big.NewInt(1000000000),
		Paymaster:     &DEFAULT_PAYMASTER,
	}, "insufficient funds for gas * price + value: address 0xaaAaaAAAAAbBbbbbBbBBCCCCcCCCcCdddDDDdddd have 1 want 1000000000000000000")
}

func TestPaymasterValidationFailure_oog(t *testing.T) {

	handleTransaction(newTestContextBuilder(t).withCode(DEFAULT_SENDER, createAccountCode(), 0).
		withCode(DEFAULT_PAYMASTER.String(), createCode(vm.PUSH0, vm.DUP1, vm.REVERT), DEFAULT_BALANCE), types.Rip7560AccountAbstractionTx{
		ValidationGas: 1000000000,
		GasFeeCap:     big.NewInt(1000000000),
		Paymaster:     &DEFAULT_PAYMASTER,
	}, "out of gas")
}
func TestPaymasterValidationFailure_revert(t *testing.T) {

	handleTransaction(newTestContextBuilder(t).withCode(DEFAULT_SENDER, createAccountCode(), 0).
		withCode(DEFAULT_PAYMASTER.String(), createCode(vm.PUSH0, vm.DUP1, vm.REVERT), DEFAULT_BALANCE), types.Rip7560AccountAbstractionTx{
		ValidationGas: uint64(1000000000),
		GasFeeCap:     big.NewInt(1000000000),
		Paymaster:     &DEFAULT_PAYMASTER,
		PaymasterGas:  1000000000,
	}, "execution reverted")
}

func asBytes32(a int) []byte {
	return common.LeftPadBytes(big.NewInt(int64(a)).Bytes(), 32)
}
func paymasterReturnValue(magic, validAfter, validUntil uint64, context []byte) []byte {
	validationData := core.PackValidationData(magic, validUntil, validAfter)
	//manual encode (bytes32 validationData, bytes context)
	return slices.Concat(
		common.LeftPadBytes(validationData, 32),
		asBytes32(64),
		asBytes32(len(context)),
		context)
}

func TestPaymasterValidationFailure_unparseable_return_value(t *testing.T) {

	handleTransaction(newTestContextBuilder(t).withCode(DEFAULT_SENDER, createAccountCode(), 0).
		withCode(DEFAULT_PAYMASTER.String(), createAccountCode(), DEFAULT_BALANCE), types.Rip7560AccountAbstractionTx{
		ValidationGas: 1000000000,
		PaymasterGas:  1000000000,
		GasFeeCap:     big.NewInt(1000000000),
		Paymaster:     &DEFAULT_PAYMASTER,
	}, "invalid paymaster return data")
}
