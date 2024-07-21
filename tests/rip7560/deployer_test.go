package rip7560

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"math/big"
	"testing"
)

var DEPLOYER = common.HexToAddress("0xddddddddddeeeeeeeeeeddddddddddeeeeeeeeee")

func TestValidationFailure_deployerRevert(t *testing.T) {
	handleTransaction(newTestContextBuilder(t).
		withCode(DEFAULT_SENDER, []byte{}, DEFAULT_BALANCE).
		withCode(DEPLOYER.Hex(), revertWithData([]byte{}), 0),
		types.Rip7560AccountAbstractionTx{
			Deployer:           &DEPLOYER,
			ValidationGasLimit: 1000000000,
			GasFeeCap:          big.NewInt(1000000000),
		}, "account deployment failed: execution reverted")
}

func TestValidationFailure_deployerOOG(t *testing.T) {
	handleTransaction(newTestContextBuilder(t).
		withCode(DEFAULT_SENDER, []byte{}, DEFAULT_BALANCE).
		withCode(DEPLOYER.Hex(), revertWithData([]byte{}), 0),
		types.Rip7560AccountAbstractionTx{
			Deployer:           &DEPLOYER,
			ValidationGasLimit: 1,
			GasFeeCap:          big.NewInt(1000000000),
		}, "account deployment failed: out of gas")
}

func TestValidationFailure_senderNotDeployed(t *testing.T) {
	handleTransaction(newTestContextBuilder(t).
		withCode(DEFAULT_SENDER, []byte{}, DEFAULT_BALANCE).
		withCode(DEPLOYER.Hex(), returnWithData([]byte{}), 0),
		types.Rip7560AccountAbstractionTx{
			Deployer:           &DEPLOYER,
			ValidationGasLimit: 1000000000,
			GasFeeCap:          big.NewInt(1000000000),
		}, "account deployment failed: sender not deployed")
}

func TestValidationFailure_senderAlreadyDeployed(t *testing.T) {
	accountCode := revertWithData([]byte{})
	deployerCode := create2(accountCode)
	sender := create2_addr(DEPLOYER, accountCode)
	handleTransaction(newTestContextBuilder(t).
		withCode(sender.Hex(), accountCode, DEFAULT_BALANCE).
		withCode(DEPLOYER.Hex(), deployerCode, 0),
		types.Rip7560AccountAbstractionTx{
			Sender:             &sender,
			Deployer:           &DEPLOYER,
			ValidationGasLimit: 1000000000,
			GasFeeCap:          big.NewInt(1000000000),
		}, "account deployment failed: sender already deployed")
}

func TestValidationFailure_senderReverts(t *testing.T) {
	accountCode := revertWithData([]byte{})
	deployerCode := createCode(create2(accountCode), returnWithData([]byte{}))
	sender := create2_addr(DEPLOYER, accountCode)
	handleTransaction(newTestContextBuilder(t).
		withCode(sender.Hex(), []byte{}, DEFAULT_BALANCE).
		withCode(DEPLOYER.Hex(), deployerCode, 0),
		types.Rip7560AccountAbstractionTx{
			Sender:             &sender,
			Deployer:           &DEPLOYER,
			ValidationGasLimit: 1000000000,
			GasFeeCap:          big.NewInt(1000000000),
		}, "execution reverted")
}

func TestValidation_deployer_ok(t *testing.T) {
	accountCode := createAccountCode()
	deployerCode := createCode(create2(accountCode), returnWithData([]byte{}))
	sender := create2_addr(DEPLOYER, accountCode)
	handleTransaction(newTestContextBuilder(t).
		withCode(sender.Hex(), []byte{}, DEFAULT_BALANCE).
		withCode(DEPLOYER.Hex(), deployerCode, 0),
		types.Rip7560AccountAbstractionTx{
			Sender:             &sender,
			Deployer:           &DEPLOYER,
			ValidationGasLimit: 1000000000,
			GasFeeCap:          big.NewInt(1000000000),
		}, "ok")
}
