package core

import (
	"bytes"
	"errors"
	"fmt"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/tracing"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/params"
	"github.com/holiman/uint256"
	"math/big"
)

var AA_ENTRY_POINT = common.HexToAddress("0x0000000000000000000000000000000000007560")
var AA_SENDER_CREATOR = common.HexToAddress("0x00000000000000000000000000000000ffff7560")

type EntryPointCallEntry struct {
	From  common.Address
	Input []byte
}
type EntryPointCall struct {
	entries []EntryPointCallEntry
	err     error
}

func NewEntryPointCall() *EntryPointCall {
	return &EntryPointCall{
		entries: make([]EntryPointCallEntry, 0),
	}
}

type ValidationPhaseResult struct {
	TxIndex             int
	Tx                  *types.Transaction
	TxHash              common.Hash
	PaymasterContext    []byte
	PreCharge           *uint256.Int
	EffectiveGasPrice   *uint256.Int
	DeploymentUsedGas   uint64
	ValidationUsedGas   uint64
	PmValidationUsedGas uint64
	SenderValidAfter    uint64
	SenderValidUntil    uint64
	PmValidAfter        uint64
	PmValidUntil        uint64
}

// HandleRip7560Transactions apply state changes of all sequential RIP-7560 transactions and return
// the number of handled transactions
// the transactions array must start with the RIP-7560 transaction
func HandleRip7560Transactions(transactions []*types.Transaction, index int, statedb *state.StateDB, coinbase *common.Address, header *types.Header, gp *GasPool, chainConfig *params.ChainConfig, bc ChainContext, cfg vm.Config) ([]*types.Transaction, types.Receipts, []*types.Log, error) {
	validatedTransactions := make([]*types.Transaction, 0)
	receipts := make([]*types.Receipt, 0)
	allLogs := make([]*types.Log, 0)

	iTransactions, iReceipts, iLogs, err := handleRip7560Transactions(transactions, index, statedb, coinbase, header, gp, chainConfig, bc, cfg)
	if err != nil {
		return nil, nil, nil, err
	}
	validatedTransactions = append(validatedTransactions, iTransactions...)
	receipts = append(receipts, iReceipts...)
	allLogs = append(allLogs, iLogs...)
	return validatedTransactions, receipts, allLogs, nil
}

func handleRip7560Transactions(transactions []*types.Transaction, index int, statedb *state.StateDB, coinbase *common.Address, header *types.Header, gp *GasPool, chainConfig *params.ChainConfig, bc ChainContext, cfg vm.Config) ([]*types.Transaction, types.Receipts, []*types.Log, error) {
	validationPhaseResults := make([]*ValidationPhaseResult, 0)
	validatedTransactions := make([]*types.Transaction, 0)
	receipts := make([]*types.Receipt, 0)
	allLogs := make([]*types.Log, 0)
	for i, tx := range transactions[index:] {
		if tx.Type() != types.Rip7560Type {
			break
		}

		statedb.SetTxContext(tx.Hash(), index+i)

		vpr, err := ApplyRip7560ValidationPhases(chainConfig, bc, coinbase, gp, statedb, header, tx, cfg)
		if err != nil {
			return nil, nil, nil, err
		}
		validationPhaseResults = append(validationPhaseResults, vpr)
		validatedTransactions = append(validatedTransactions, tx)

		// This is the line separating the Validation and Execution phases
		// It should be separated to implement the mempool-friendly AA RIP-7711
		// for i, vpr := range validationPhaseResults

		// TODO: this will miss all validation phase events - pass in 'vpr'
		// statedb.SetTxContext(vpr.Tx.Hash(), i)

		receipt, err := ApplyRip7560ExecutionPhase(chainConfig, vpr, bc, coinbase, gp, statedb, header, cfg)

		if err != nil {
			return nil, nil, nil, err
		}
		statedb.Finalise(true)

		receipts = append(receipts, receipt)
		allLogs = append(allLogs, receipt.Logs...)
	}
	return validatedTransactions, receipts, allLogs, nil
}

// todo: move to a suitable interface, whatever that is
// todo 2: maybe handle the "shared gas pool" situation instead of just overriding it completely?
func BuyGasRip7560Transaction(st *types.Rip7560AccountAbstractionTx, state vm.StateDB, gasPrice *uint256.Int) (*uint256.Int, error) {
	gasLimit := st.Gas + st.ValidationGasLimit + st.PaymasterValidationGasLimit + st.PostOpGas
	preCharge := new(uint256.Int).SetUint64(gasLimit)
	preCharge = preCharge.Mul(preCharge, gasPrice)
	balanceCheck := new(uint256.Int).Set(preCharge)

	chargeFrom := st.Sender

	if st.Paymaster != nil && st.Paymaster.Cmp(common.Address{}) != 0 {
		chargeFrom = st.Paymaster
	}

	if have, want := state.GetBalance(*chargeFrom), balanceCheck; have.Cmp(want) < 0 {
		return nil, fmt.Errorf("%w: address %v have %v want %v", ErrInsufficientFunds, chargeFrom.Hex(), have, want)
	}

	state.SubBalance(*chargeFrom, preCharge, 0)
	return preCharge, nil
}

// refund the transaction payer (either account or paymaster) with the excess gas cost
func refundPayer(vpr *ValidationPhaseResult, state vm.StateDB, gasUsed uint64) {
	var chargeFrom *common.Address
	if vpr.PmValidationUsedGas == 0 {
		chargeFrom = vpr.Tx.Rip7560TransactionData().Sender
	} else {
		chargeFrom = vpr.Tx.Rip7560TransactionData().Paymaster
	}

	actualGasCost := new(uint256.Int).Mul(vpr.EffectiveGasPrice, new(uint256.Int).SetUint64(gasUsed))

	refund := new(uint256.Int).Sub(vpr.PreCharge, actualGasCost)

	state.AddBalance(*chargeFrom, refund, tracing.BalanceIncreaseGasReturn)
}

// precheck nonce of transaction.
// (standard preCheck function check both nonce and no-code of account)
func CheckNonceRip7560(tx *types.Rip7560AccountAbstractionTx, st *state.StateDB) error {
	// Make sure this transaction's nonce is correct.
	stNonce := st.GetNonce(*tx.Sender)
	if msgNonce := tx.Nonce; stNonce < msgNonce {
		return fmt.Errorf("%w: address %v, tx: %d state: %d", ErrNonceTooHigh,
			tx.Sender.Hex(), msgNonce, stNonce)
	} else if stNonce > msgNonce {
		return fmt.Errorf("%w: address %v, tx: %d state: %d", ErrNonceTooLow,
			tx.Sender.Hex(), msgNonce, stNonce)
	} else if stNonce+1 < stNonce {
		return fmt.Errorf("%w: address %v, nonce: %d", ErrNonceMax,
			tx.Sender.Hex(), stNonce)
	}
	return nil
}

// finalize validation return data from account and paymaster
func ValidateAccountAndPaymaster(time uint64, sender, paymaster *common.Address, epc *EntryPointCall) (*AcceptAccountData, *AcceptPaymasterData, error) {
	if epc.err != nil {
		return nil, nil, epc.err
	}
	if len(epc.entries) == 0 || epc.entries[0].From.Cmp(*sender) != 0 {
		return nil, nil, errors.New("account validation did not call the EntryPoint 'acceptAccount' callback")
	}
	if paymaster == nil {
		if len(epc.entries) > 1 {
			return nil, nil, errors.New("EntryPoint callback called more than once")
		}
	} else {
		if len(epc.entries) < 2 || epc.entries[1].From.Cmp(*paymaster) != 0 {
			return nil, nil, errors.New("paymaster validation did not call the EntryPoint 'acceptPaymaster' callback")
		}
		if len(epc.entries) > 2 {
			return nil, nil, errors.New("EntryPoint callback called too many times")
		}
	}
	aad, err := ValidateAccountEntryPointCall(sender, epc.entries[0].Input)
	if err != nil {
		return nil, nil, err
	}

	err = validateValidityTimeRange(time, aad.ValidAfter.Uint64(), aad.ValidUntil.Uint64())
	if err != nil {
		return nil, nil, err
	}
	var apd *AcceptPaymasterData
	if paymaster != nil {
		apd, err = validatePaymasterEntryPointCall(paymaster, epc.entries[1].Input)
		if err != nil {
			return nil, nil, err
		}
		err = validateValidityTimeRange(time, apd.ValidAfter.Uint64(), apd.ValidUntil.Uint64())
		if err != nil {
			return nil, nil, err
		}
	}
	return aad, apd, nil
}

func ApplyRip7560ValidationPhases(chainConfig *params.ChainConfig, bc ChainContext, author *common.Address, gp *GasPool, statedb *state.StateDB, header *types.Header, tx *types.Transaction, cfg vm.Config) (*ValidationPhaseResult, error) {
	aatx := tx.Rip7560TransactionData()
	err := CheckNonceRip7560(aatx, statedb)
	if err != nil {
		return nil, err
	}

	gasPrice := new(big.Int).Add(header.BaseFee, tx.GasTipCap())
	if gasPrice.Cmp(tx.GasFeeCap()) > 0 {
		gasPrice = tx.GasFeeCap()
	}
	gasPriceUint256, _ := uint256.FromBig(gasPrice)

	preCharge, err := BuyGasRip7560Transaction(aatx, statedb, gasPriceUint256)
	if err != nil {
		return nil, err
	}

	blockContext := NewEVMBlockContext(header, bc, author)
	sender := tx.Rip7560TransactionData().Sender
	paymaster := tx.Rip7560TransactionData().Paymaster
	txContext := vm.TxContext{
		Origin:   *sender,
		GasPrice: gasPrice,
	}
	evm := vm.NewEVM(blockContext, txContext, statedb, chainConfig, cfg)

	var epc *EntryPointCall
	if evm.Config.Tracer == nil {
		epc = NewEntryPointCall()
		evm.Config.Tracer = &tracing.Hooks{
			OnEnter: epc.OnEnter,
		}
	}

	if evm.Config.Tracer.OnTxStart != nil {
		evm.Config.Tracer.OnTxStart(evm.GetVMContext(), tx, common.Address{})
	}

	/*** Deployer Frame ***/
	deployerMsg := prepareDeployerMessage(tx, chainConfig)
	var deploymentUsedGas uint64
	if deployerMsg != nil {
		var err error
		var resultDeployer *ExecutionResult
		if statedb.GetCodeSize(*sender) != 0 {
			err = errors.New("sender already deployed")
		} else {
			resultDeployer, err = ApplyMessage(evm, deployerMsg, gp)
		}
		if err == nil && resultDeployer != nil {
			err = resultDeployer.Err
			deploymentUsedGas = resultDeployer.UsedGas
		}
		if err == nil && statedb.GetCodeSize(*sender) == 0 {
			err = errors.New("sender not deployed")
		}
		if err != nil {
			return nil, fmt.Errorf("account deployment failed: %v", err)
		}
	} else {
		statedb.SetNonce(*sender, statedb.GetNonce(*sender)+1)
	}

	/*** Account Validation Frame ***/
	signer := types.MakeSigner(chainConfig, header.Number, header.Time)
	signingHash := signer.Hash(tx)
	accountValidationMsg, err := prepareAccountValidationMessage(tx, chainConfig, signingHash, deploymentUsedGas)
	resultAccountValidation, err := ApplyMessage(evm, accountValidationMsg, gp)
	if err != nil {
		return nil, err
	}
	if resultAccountValidation.Err != nil {
		return nil, resultAccountValidation.Err
	}

	vpr := &ValidationPhaseResult{}
	pmValidationUsedGas, err := applyPaymasterValidationFrame(tx, chainConfig, signingHash, evm, gp, statedb, header)
	if err != nil {
		return nil, err
	}

	aad, apd, err := ValidateAccountAndPaymaster(header.Time, sender, paymaster, epc)
	if err != nil {
		return nil, err
	}
	vpr.Tx = tx
	vpr.TxHash = tx.Hash()
	vpr.PreCharge = preCharge
	vpr.EffectiveGasPrice = gasPriceUint256
	vpr.PaymasterContext = apd.Context
	vpr.DeploymentUsedGas = deploymentUsedGas
	vpr.ValidationUsedGas = resultAccountValidation.UsedGas
	vpr.PmValidationUsedGas = pmValidationUsedGas
	vpr.SenderValidAfter = aad.ValidAfter.Uint64()
	vpr.SenderValidUntil = aad.ValidUntil.Uint64()
	vpr.PmValidAfter = apd.ValidAfter.Uint64()
	vpr.PmValidUntil = apd.ValidUntil.Uint64()

	statedb.Finalise(true)

	return vpr, nil
}

func applyPaymasterValidationFrame(tx *types.Transaction, chainConfig *params.ChainConfig, signingHash common.Hash, evm *vm.EVM, gp *GasPool, statedb *state.StateDB, header *types.Header) (uint64, error) {
	/*** Paymaster Validation Frame ***/
	var pmValidationUsedGas uint64
	paymasterMsg, err := preparePaymasterValidationMessage(tx, chainConfig, signingHash)
	if paymasterMsg == nil || err != nil {
		return 0, err
	}
	resultPm, err := ApplyMessage(evm, paymasterMsg, gp)
	if err != nil {
		return 0, err
	}
	if resultPm.Failed() {
		return 0, resultPm.Err
	}
	if resultPm.Failed() {
		return 0, errors.New("paymaster validation failed - invalid transaction")
	}
	pmValidationUsedGas = resultPm.UsedGas
	return pmValidationUsedGas, nil
}

func applyPaymasterPostOpFrame(vpr *ValidationPhaseResult, executionResult *ExecutionResult, evm *vm.EVM, gp *GasPool, statedb *state.StateDB, header *types.Header) (*ExecutionResult, error) {
	var paymasterPostOpResult *ExecutionResult
	paymasterPostOpMsg, err := preparePostOpMessage(vpr, evm.ChainConfig(), executionResult)
	if err != nil {
		return nil, err
	}
	paymasterPostOpResult, err = ApplyMessage(evm, paymasterPostOpMsg, gp)
	if err != nil {
		return nil, err
	}
	// TODO: revert the execution phase changes
	return paymasterPostOpResult, nil
}

func ApplyRip7560ExecutionPhase(config *params.ChainConfig, vpr *ValidationPhaseResult, bc ChainContext, author *common.Address, gp *GasPool, statedb *state.StateDB, header *types.Header, cfg vm.Config) (*types.Receipt, error) {

	// TODO: snapshot EVM - we will revert back here if postOp fails

	blockContext := NewEVMBlockContext(header, bc, author)
	message, err := TransactionToMessage(vpr.Tx, types.MakeSigner(config, header.Number, header.Time), header.BaseFee)
	txContext := NewEVMTxContext(message)
	txContext.Origin = *vpr.Tx.Rip7560TransactionData().Sender
	evm := vm.NewEVM(blockContext, txContext, statedb, config, cfg)

	accountExecutionMsg := prepareAccountExecutionMessage(vpr.Tx, evm.ChainConfig())
	executionResult, err := ApplyMessage(evm, accountExecutionMsg, gp)
	if err != nil {
		return nil, err
	}
	var paymasterPostOpResult *ExecutionResult
	if len(vpr.PaymasterContext) != 0 {
		paymasterPostOpResult, err = applyPaymasterPostOpFrame(vpr, executionResult, evm, gp, statedb, header)
	}
	if err != nil {
		return nil, err
	}

	gasUsed :=
		vpr.ValidationUsedGas +
			vpr.DeploymentUsedGas +
			vpr.PmValidationUsedGas +
			executionResult.UsedGas
	if paymasterPostOpResult != nil {
		gasUsed +=
			paymasterPostOpResult.UsedGas
	}

	receipt := &types.Receipt{Type: vpr.Tx.Type(), TxHash: vpr.Tx.Hash(), GasUsed: gasUsed, CumulativeGasUsed: gasUsed}

	if executionResult.Failed() || (paymasterPostOpResult != nil && paymasterPostOpResult.Failed()) {
		receipt.Status = types.ReceiptStatusFailed
	} else {
		receipt.Status = types.ReceiptStatusSuccessful
	}

	refundPayer(vpr, statedb, gasUsed)

	// Set the receipt logs and create the bloom filter.
	blockNumber := header.Number
	receipt.Logs = statedb.GetLogs(vpr.TxHash, blockNumber.Uint64(), common.Hash{})
	receipt.Bloom = types.CreateBloom(types.Receipts{receipt})
	receipt.TransactionIndex = uint(vpr.TxIndex)
	// other fields are filled in DeriveFields (all tx, block fields, and updating CumulativeGasUsed
	return receipt, err
}

func prepareDeployerMessage(baseTx *types.Transaction, config *params.ChainConfig) *Message {
	tx := baseTx.Rip7560TransactionData()
	if tx.Deployer == nil || tx.Deployer.Cmp(common.Address{}) == 0 {
		return nil
	}
	return &Message{
		From:              AA_SENDER_CREATOR,
		To:                tx.Deployer,
		Value:             big.NewInt(0),
		GasLimit:          tx.ValidationGasLimit,
		GasPrice:          tx.GasFeeCap,
		GasFeeCap:         tx.GasFeeCap,
		GasTipCap:         tx.GasTipCap,
		Data:              tx.DeployerData,
		AccessList:        make(types.AccessList, 0),
		SkipAccountChecks: true,
		IsRip7560Frame:    true,
	}
}

func prepareAccountValidationMessage(baseTx *types.Transaction, chainConfig *params.ChainConfig, signingHash common.Hash, deploymentUsedGas uint64) (*Message, error) {
	tx := baseTx.Rip7560TransactionData()
	data, err := abiEncodeValidateTransaction(tx, signingHash)
	if err != nil {
		return nil, err
	}
	return &Message{
		From:              AA_ENTRY_POINT,
		To:                tx.Sender,
		Value:             big.NewInt(0),
		GasLimit:          tx.ValidationGasLimit - deploymentUsedGas,
		GasPrice:          tx.GasFeeCap,
		GasFeeCap:         tx.GasFeeCap,
		GasTipCap:         tx.GasTipCap,
		Data:              data,
		AccessList:        make(types.AccessList, 0),
		SkipAccountChecks: true,
		IsRip7560Frame:    true,
	}, nil
}

func preparePaymasterValidationMessage(baseTx *types.Transaction, config *params.ChainConfig, signingHash common.Hash) (*Message, error) {
	tx := baseTx.Rip7560TransactionData()
	if tx.Paymaster == nil || tx.Paymaster.Cmp(common.Address{}) == 0 {
		return nil, nil
	}
	data, err := abiEncodeValidatePaymasterTransaction(tx, signingHash)
	if err != nil {
		return nil, err
	}
	return &Message{
		From:              AA_ENTRY_POINT,
		To:                tx.Paymaster,
		Value:             big.NewInt(0),
		GasLimit:          tx.PaymasterValidationGasLimit,
		GasPrice:          tx.GasFeeCap,
		GasFeeCap:         tx.GasFeeCap,
		GasTipCap:         tx.GasTipCap,
		Data:              data,
		AccessList:        make(types.AccessList, 0),
		SkipAccountChecks: true,
		IsRip7560Frame:    true,
	}, nil
}

func prepareAccountExecutionMessage(baseTx *types.Transaction, config *params.ChainConfig) *Message {
	tx := baseTx.Rip7560TransactionData()
	return &Message{
		From:              AA_ENTRY_POINT,
		To:                tx.Sender,
		Value:             big.NewInt(0),
		GasLimit:          tx.Gas,
		GasPrice:          tx.GasFeeCap,
		GasFeeCap:         tx.GasFeeCap,
		GasTipCap:         tx.GasTipCap,
		Data:              tx.Data,
		AccessList:        make(types.AccessList, 0),
		SkipAccountChecks: true,
		IsRip7560Frame:    true,
	}
}

func preparePostOpMessage(vpr *ValidationPhaseResult, chainConfig *params.ChainConfig, executionResult *ExecutionResult) (*Message, error) {
	if len(vpr.PaymasterContext) == 0 {
		return nil, nil
	}
	tx := vpr.Tx.Rip7560TransactionData()
	postOpData, err := abiEncodePostPaymasterTransaction(vpr.PaymasterContext)
	if err != nil {
		return nil, err
	}
	return &Message{
		From:              AA_ENTRY_POINT,
		To:                tx.Paymaster,
		Value:             big.NewInt(0),
		GasLimit:          tx.PaymasterValidationGasLimit - executionResult.UsedGas,
		GasPrice:          tx.GasFeeCap,
		GasFeeCap:         tx.GasFeeCap,
		GasTipCap:         tx.GasTipCap,
		Data:              postOpData,
		AccessList:        tx.AccessList,
		SkipAccountChecks: true,
		IsRip7560Frame:    true,
	}, nil
}

func ValidateAccountEntryPointCall(sender *common.Address, input []byte) (*AcceptAccountData, error) {
	if len(input) != 68 {
		return nil, errors.New("invalid account return data length")
	}
	return abiDecodeAcceptAccount(input)
}

func validatePaymasterEntryPointCall(paymaster *common.Address, input []byte) (*AcceptPaymasterData, error) {
	if len(input) < 100 {
		return nil, errors.New("invalid paymaster callback data length")
	}
	apd, err := abiDecodeAcceptPaymaster(input)
	if err != nil {
		return nil, err
	}
	return apd, nil
}

func validateValidityTimeRange(time uint64, validAfter uint64, validUntil uint64) error {
	if validUntil == 0 && validAfter == 0 {
		return nil
	}
	if validUntil < validAfter {
		return errors.New("RIP-7560 transaction validity range invalid")
	}
	if time > validUntil {
		return errors.New("RIP-7560 transaction validity expired")
	}
	if time < validAfter {
		return errors.New("RIP-7560 transaction validity not reached yet")
	}
	return nil
}

func (epc *EntryPointCall) OnEnter(depth int, typ byte, from common.Address, to common.Address, input []byte, gas uint64, value *big.Int) {
	isRip7560EntryPoint := to.Cmp(AA_ENTRY_POINT) == 0
	if !isRip7560EntryPoint {
		return
	}

	if depth != 1 {
		epc.err = errors.New("called EntryPoint not from top level account")
		println(epc.err)
		return
	}
	epc.entries = append(epc.entries, EntryPointCallEntry{
		From:  from,
		Input: bytes.Clone(input),
	})
}
