package core

import (
	"errors"
	"fmt"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/params"
	"github.com/holiman/uint256"
	"math/big"
	"strings"
)

const MAGIC_VALUE_SENDER = uint64(0xbf45c166)
const MAGIC_VALUE_PAYMASTER = uint64(0xe0e6183a)
const MAGIC_VALUE_SIGFAIL = uint64(0x31665494)
const PAYMASTER_MAX_CONTEXT_SIZE = 65536

func PackValidationData(authorizerMagic uint64, validUntil, validAfter uint64) []byte {

	t := new(big.Int).SetUint64(uint64(validAfter))
	t = t.Lsh(t, 48).Add(t, new(big.Int).SetUint64(validUntil&0xffffff))
	t = t.Lsh(t, 160).Add(t, new(big.Int).SetUint64(uint64(authorizerMagic)))
	return common.LeftPadBytes(t.Bytes(), 32)
}

func UnpackValidationData(validationData []byte) (authorizerMagic uint64, validUntil, validAfter uint64) {

	t := new(big.Int).SetBytes(validationData)
	authorizerMagic = t.Uint64()
	validUntil = t.Rsh(t, 160).Uint64() & 0xffffff
	validAfter = t.Rsh(t, 48).Uint64()
	return
}

func UnpackPaymasterValidationReturn(paymasterValidationReturn []byte) (validationData, context []byte) {
	if len(paymasterValidationReturn) < 96 {
		return nil, nil
	}
	validationData = paymasterValidationReturn[0:32]
	//2nd bytes32 is ignored (its an offset value)
	contextLen := new(big.Int).SetBytes(paymasterValidationReturn[64:96])
	if uint64(len(paymasterValidationReturn)) < 96+contextLen.Uint64() {
		return nil, nil
	}

	context = paymasterValidationReturn[96 : 96+contextLen.Uint64()]
	return
}

type ValidationPhaseResult struct {
	TxIndex             int
	Tx                  *types.Transaction
	TxHash              common.Hash
	PaymasterContext    []byte
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

		aatx := tx.Rip7560TransactionData()
		statedb.SetTxContext(tx.Hash(), index+i)
		err := CheckNonceRip7560(aatx, statedb)
		if err != nil {
			return nil, nil, nil, err
		}
		err = BuyGasRip7560Transaction(aatx, statedb)
		if err != nil {
			return nil, nil, nil, err
		}

		var vpr *ValidationPhaseResult
		vpr, err = ApplyRip7560ValidationPhases(chainConfig, bc, coinbase, gp, statedb, header, tx, cfg)
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
		receipts = append(receipts, receipt)
		allLogs = append(allLogs, receipt.Logs...)
	}
	return validatedTransactions, receipts, allLogs, nil
}

// BuyGasRip7560Transaction
// todo: move to a suitable interface, whatever that is
// todo 2: maybe handle the "shared gas pool" situation instead of just overriding it completely?
func BuyGasRip7560Transaction(st *types.Rip7560AccountAbstractionTx, state vm.StateDB) error {
	gasLimit := st.Gas + st.ValidationGas + st.PaymasterGas + st.PostOpGas
	mgval := new(uint256.Int).SetUint64(gasLimit)
	gasFeeCap, _ := uint256.FromBig(st.GasFeeCap)
	mgval = mgval.Mul(mgval, gasFeeCap)
	balanceCheck := new(uint256.Int).Set(mgval)

	chargeFrom := st.Sender

	if st.Paymaster != nil {
		chargeFrom = st.Paymaster
	}

	if have, want := state.GetBalance(*chargeFrom), balanceCheck; have.Cmp(want) < 0 {
		return fmt.Errorf("%w: address %v have %v want %v", ErrInsufficientFunds, chargeFrom.Hex(), have, want)
	}

	state.SubBalance(*chargeFrom, mgval, 0)
	return nil
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

func ApplyRip7560ValidationPhases(chainConfig *params.ChainConfig, bc ChainContext, author *common.Address, gp *GasPool, statedb *state.StateDB, header *types.Header, tx *types.Transaction, cfg vm.Config) (*ValidationPhaseResult, error) {
	blockContext := NewEVMBlockContext(header, bc, author)
	txContext := vm.TxContext{
		Origin:   *tx.Rip7560TransactionData().Sender,
		GasPrice: tx.GasFeeCap(),
	}
	evm := vm.NewEVM(blockContext, txContext, statedb, chainConfig, cfg)
	/*** Deployer Frame ***/
	deployerMsg := prepareDeployerMessage(tx, chainConfig)
	var deploymentUsedGas uint64
	if deployerMsg != nil {
		resultDeployer, err := ApplyMessage(evm, deployerMsg, gp)
		if err != nil {
			return nil, err
		}
		statedb.IntermediateRoot(true)
		if resultDeployer.Failed() {
			// TODO: bubble up the inner error message to the user, if possible
			return nil, errors.New("account deployment  failed - invalid transaction")
		}
		deploymentUsedGas = resultDeployer.UsedGas
	}

	/*** Account Validation Frame ***/
	signer := types.MakeSigner(chainConfig, header.Number, header.Time)
	signingHash := signer.Hash(tx)
	accountValidationMsg, err := prepareAccountValidationMessage(tx, chainConfig, signingHash, deploymentUsedGas)
	resultAccountValidation, err := ApplyMessage(evm, accountValidationMsg, gp)
	if err != nil {
		return nil, err
	}
	statedb.IntermediateRoot(true)
	if resultAccountValidation.Err != nil {
		return nil, resultAccountValidation.Err
	}
	validAfter, validUntil, err := validateAccountReturnData(resultAccountValidation.ReturnData)
	if err != nil {
		return nil, err
	}
	err = validateValidityTimeRange(header.Time, validAfter, validUntil)
	if err != nil {
		return nil, err
	}

	paymasterContext, pmValidationUsedGas, pmValidAfter, pmValidUntil, err := applyPaymasterValidationFrame(tx, chainConfig, signingHash, evm, gp, statedb, header)
	if err != nil {
		return nil, err
	}
	vpr := &ValidationPhaseResult{
		Tx:                  tx,
		TxHash:              tx.Hash(),
		PaymasterContext:    paymasterContext,
		DeploymentUsedGas:   deploymentUsedGas,
		ValidationUsedGas:   resultAccountValidation.UsedGas,
		PmValidationUsedGas: pmValidationUsedGas,
		SenderValidAfter:    validAfter,
		SenderValidUntil:    validUntil,
		PmValidAfter:        pmValidAfter,
		PmValidUntil:        pmValidUntil,
	}

	return vpr, nil
}

func applyPaymasterValidationFrame(tx *types.Transaction, chainConfig *params.ChainConfig, signingHash common.Hash, evm *vm.EVM, gp *GasPool, statedb *state.StateDB, header *types.Header) ([]byte, uint64, uint64, uint64, error) {
	/*** Paymaster Validation Frame ***/
	var pmValidationUsedGas uint64
	var paymasterContext []byte
	var pmValidAfter uint64
	var pmValidUntil uint64
	paymasterMsg, err := preparePaymasterValidationMessage(tx, chainConfig, signingHash)
	if err != nil {
		return nil, 0, 0, 0, err
	}
	if paymasterMsg != nil {
		resultPm, err := ApplyMessage(evm, paymasterMsg, gp)
		if err != nil {
			return nil, 0, 0, 0, err
		}
		if resultPm.Failed() {
			return nil, 0, 0, 0, resultPm.Err
		}
		statedb.IntermediateRoot(true)
		if resultPm.Failed() {
			return nil, 0, 0, 0, errors.New("paymaster validation failed - invalid transaction")
		}
		pmValidationUsedGas = resultPm.UsedGas
		paymasterContext, pmValidAfter, pmValidUntil, err = validatePaymasterReturnData(resultPm.ReturnData)
		if err != nil {
			return nil, 0, 0, 0, err
		}
		err = validateValidityTimeRange(header.Time, pmValidAfter, pmValidUntil)
		if err != nil {
			return nil, 0, 0, 0, err
		}
	}
	return paymasterContext, pmValidationUsedGas, pmValidAfter, pmValidUntil, nil
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
	root := statedb.IntermediateRoot(true).Bytes()
	var paymasterPostOpResult *ExecutionResult
	if len(vpr.PaymasterContext) != 0 {
		paymasterPostOpResult, err = applyPaymasterPostOpFrame(vpr, executionResult, evm, gp, statedb, header)
		root = statedb.IntermediateRoot(true).Bytes()
	}
	if err != nil {
		return nil, err
	}

	cumulativeGasUsed :=
		vpr.ValidationUsedGas +
			vpr.DeploymentUsedGas +
			vpr.PmValidationUsedGas +
			executionResult.UsedGas
	if paymasterPostOpResult != nil {
		cumulativeGasUsed +=
			paymasterPostOpResult.UsedGas
	}

	receipt := &types.Receipt{Type: vpr.Tx.Type(), PostState: root, CumulativeGasUsed: cumulativeGasUsed}

	// Set the receipt logs and create the bloom filter.
	receipt.Logs = statedb.GetLogs(vpr.Tx.Hash(), header.Number.Uint64(), header.Hash())

	if executionResult.Failed() || (paymasterPostOpResult != nil && paymasterPostOpResult.Failed()) {
		receipt.Status = types.ReceiptStatusFailed
	} else {
		receipt.Status = types.ReceiptStatusSuccessful
	}
	return receipt, err
}

func prepareDeployerMessage(baseTx *types.Transaction, config *params.ChainConfig) *Message {
	tx := baseTx.Rip7560TransactionData()
	if len(tx.DeployerData) < 20 {
		return nil
	}
	var deployerAddress common.Address = [20]byte(tx.DeployerData[0:20])
	return &Message{
		From:              config.DeployerCallerAddress,
		To:                &deployerAddress,
		Value:             big.NewInt(0),
		GasLimit:          tx.ValidationGas,
		GasPrice:          tx.GasFeeCap,
		GasFeeCap:         tx.GasFeeCap,
		GasTipCap:         tx.GasTipCap,
		Data:              tx.DeployerData[20:],
		AccessList:        make(types.AccessList, 0),
		SkipAccountChecks: true,
		IsRip7560Frame:    true,
	}
}

func prepareAccountValidationMessage(baseTx *types.Transaction, chainConfig *params.ChainConfig, signingHash common.Hash, deploymentUsedGas uint64) (*Message, error) {
	tx := baseTx.Rip7560TransactionData()
	jsondata := `[
	{"type":"function","name":"validateTransaction","inputs": [{"name": "version","type": "uint256"},{"name": "txHash","type": "bytes32"},{"name": "transaction","type": "bytes"}]}
	]`

	validateTransactionAbi, err := abi.JSON(strings.NewReader(jsondata))
	if err != nil {
		return nil, err
	}
	txAbiEncoding, err := tx.AbiEncode()
	validateTransactionData, err := validateTransactionAbi.Pack("validateTransaction", big.NewInt(0), signingHash, txAbiEncoding)
	return &Message{
		From:              chainConfig.EntryPointAddress,
		To:                tx.Sender,
		Value:             big.NewInt(0),
		GasLimit:          tx.ValidationGas - deploymentUsedGas,
		GasPrice:          tx.GasFeeCap,
		GasFeeCap:         tx.GasFeeCap,
		GasTipCap:         tx.GasTipCap,
		Data:              validateTransactionData,
		AccessList:        make(types.AccessList, 0),
		SkipAccountChecks: true,
		IsRip7560Frame:    true,
	}, nil
}

func preparePaymasterValidationMessage(baseTx *types.Transaction, config *params.ChainConfig, signingHash common.Hash) (*Message, error) {
	tx := baseTx.Rip7560TransactionData()
	paymasterAddress := tx.Paymaster
	if paymasterAddress == nil {
		return nil, nil
	}
	jsondata := `[
	{"type":"function","name":"validatePaymasterTransaction","inputs": [{"name": "version","type": "uint256"},{"name": "txHash","type": "bytes32"},{"name": "transaction","type": "bytes"}]}
	]`

	validateTransactionAbi, err := abi.JSON(strings.NewReader(jsondata))
	txAbiEncoding, err := tx.AbiEncode()
	data, err := validateTransactionAbi.Pack("validatePaymasterTransaction", big.NewInt(0), signingHash, txAbiEncoding)

	if err != nil {
		return nil, err
	}
	return &Message{
		From:              config.EntryPointAddress,
		To:                paymasterAddress,
		Value:             big.NewInt(0),
		GasLimit:          tx.PaymasterGas,
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
		From:              config.EntryPointAddress,
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
	jsondata := `[
			{"type":"function","name":"postPaymasterTransaction","inputs": [{"name": "success","type": "bool"},{"name": "actualGasCost","type": "uint256"},{"name": "context","type": "bytes"}]}
		]`
	postPaymasterTransactionAbi, err := abi.JSON(strings.NewReader(jsondata))
	if err != nil {
		return nil, err
	}
	postOpData, err := postPaymasterTransactionAbi.Pack("postPaymasterTransaction", true, big.NewInt(0), vpr.PaymasterContext)
	if err != nil {
		return nil, err
	}
	var paymasterAddress = tx.Paymaster
	return &Message{
		From:              chainConfig.EntryPointAddress,
		To:                paymasterAddress,
		Value:             big.NewInt(0),
		GasLimit:          tx.PaymasterGas - executionResult.UsedGas,
		GasPrice:          tx.GasFeeCap,
		GasFeeCap:         tx.GasFeeCap,
		GasTipCap:         tx.GasTipCap,
		Data:              postOpData,
		AccessList:        tx.AccessList,
		SkipAccountChecks: true,
		IsRip7560Frame:    true,
	}, nil
}

func validateAccountReturnData(data []byte) (uint64, uint64, error) {
	if len(data) != 32 {
		return 0, 0, errors.New("invalid account return data length")
	}
	magicExpected, validUntil, validAfter := UnpackValidationData(data)
	//todo: we check first 8 bytes of the 20-byte address (the rest is expected to be zeros)
	if magicExpected != MAGIC_VALUE_SENDER {
		if magicExpected == MAGIC_VALUE_SIGFAIL {
			return 0, 0, errors.New("account signature error")
		}
		return 0, 0, errors.New("account did not return correct MAGIC_VALUE")
	}
	return validAfter, validUntil, nil
}

func validatePaymasterReturnData(data []byte) (context []byte, validAfter, validUntil uint64, error error) {
	if len(data) < 32 {
		return nil, 0, 0, errors.New("invalid paymaster return data length")
	}
	validationData, context := UnpackPaymasterValidationReturn(data)
	if validationData == nil {
		return nil, 0, 0, errors.New("invalid paymaster return data")
	}
	magicExpected, validAfter, validUntil := UnpackValidationData(validationData)
	if magicExpected != MAGIC_VALUE_PAYMASTER {
		return nil, 0, 0, errors.New("paymaster did not return correct MAGIC_VALUE")
	}
	if len(context) > PAYMASTER_MAX_CONTEXT_SIZE {
		return nil, 0, 0, errors.New("paymaster context too large")
	}
	return context, validAfter, validUntil, nil
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
