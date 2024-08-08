package core

import (
	"errors"
	"fmt"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/tracing"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/params"
	"github.com/holiman/uint256"
	"math/big"
	"strings"
)

const MAGIC_VALUE_SENDER = uint64(0x1256ebd1)    // acceptAccount(uint256,uint256)
const MAGIC_VALUE_PAYMASTER = uint64(0x03be8439) // acceptPaymaster(uint256,uint256,bytes)
const MAGIC_VALUE_SIGFAIL = uint64(0x7715fac2)   // sigFailAccount(uint256,uint256)
const PAYMASTER_MAX_CONTEXT_SIZE = 65536

var AA_ENTRY_POINT = common.HexToAddress("0x0000000000000000000000000000000000007560")
var AA_SENDER_CREATOR = common.HexToAddress("0x00000000000000000000000000000000ffff7560")

func PackValidationData(authorizerMagic uint64, validUntil, validAfter uint64) []byte {

	t := new(big.Int).SetUint64(uint64(validAfter))
	t = t.Lsh(t, 48).Add(t, new(big.Int).SetUint64(validUntil&0xffffff))
	t = t.Lsh(t, 160).Add(t, new(big.Int).SetUint64(uint64(authorizerMagic)))
	return common.LeftPadBytes(t.Bytes(), 32)
}

func UnpackValidationData(validationData []byte) (authorizerMagic uint64, validUntil uint64, validAfter uint64) {
	authorizerMagic = new(big.Int).SetBytes(validationData[:4]).Uint64()
	validAfter = new(big.Int).SetBytes(validationData[4:36]).Uint64()
	validUntil = new(big.Int).SetBytes(validationData[36:68]).Uint64()
	return
}

func UnpackPaymasterValidationReturn(paymasterValidationReturn []byte) (authorizerMagic uint64, validUntil uint64, validAfter uint64, context []byte, err error) {
	if len(paymasterValidationReturn) < 100 {
		return 0, 0, 0, nil, errors.New("paymaster return data: too short")
	}
	authorizerMagic = new(big.Int).SetBytes(paymasterValidationReturn[:4]).Uint64()
	validAfter = new(big.Int).SetBytes(paymasterValidationReturn[4:36]).Uint64()
	validUntil = new(big.Int).SetBytes(paymasterValidationReturn[36:68]).Uint64()
	contextDataLength := paymasterValidationReturn[100:132]
	contextLen := new(big.Int).SetBytes(contextDataLength)
	if uint64(len(paymasterValidationReturn)) < 96+contextLen.Uint64() {
		return 0, 0, 0, nil, errors.New("paymaster return data: unable to decode context")
	}
	if contextLen.Cmp(big.NewInt(PAYMASTER_MAX_CONTEXT_SIZE)) > 0 {
		return 0, 0, 0, nil, errors.New("paymaster return data: context too large")
	}

	context = paymasterValidationReturn[132 : 132+contextLen.Uint64()]
	return
}

type EntryPointCall struct {
	caller common.Address
	input  []byte
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
	// tracking the calls to the EntryPoint precompile
	PmUsed       bool
	EpCalls      []*EntryPointCall
	OnEnterSuper tracing.EnterHook
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
	txContext := vm.TxContext{
		Origin:   *sender,
		GasPrice: gasPrice,
	}
	evm := vm.NewEVM(blockContext, txContext, statedb, chainConfig, cfg)
	vpr := &ValidationPhaseResult{
		PmUsed:  false,
		EpCalls: make([]*EntryPointCall, 0),
	}

	if evm.Config.Tracer == nil {
		evm.Config.Tracer = &tracing.Hooks{
			OnEnter: vpr.OnEnter,
		}
	} else {
		// keep the original tracer's OnEnter hook
		vpr.OnEnterSuper = evm.Config.Tracer.OnEnter
		evm.Config.Tracer.OnEnter = vpr.OnEnter
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
	validAfter, validUntil, err := vpr.validateAccountEntryPointCall()
	if err != nil {
		return nil, err
	}
	err = validateValidityTimeRange(header.Time, validAfter, validUntil)
	if err != nil {
		return nil, err
	}

	paymasterContext, pmValidationUsedGas, pmValidAfter, pmValidUntil, err := applyPaymasterValidationFrame(vpr, tx, chainConfig, signingHash, evm, gp, statedb, header)
	if err != nil {
		return nil, err
	}

	vpr.Tx = tx
	vpr.TxHash = tx.Hash()
	vpr.PreCharge = preCharge
	vpr.EffectiveGasPrice = gasPriceUint256
	vpr.PaymasterContext = paymasterContext
	vpr.DeploymentUsedGas = deploymentUsedGas
	vpr.ValidationUsedGas = resultAccountValidation.UsedGas
	vpr.PmValidationUsedGas = pmValidationUsedGas
	vpr.SenderValidAfter = validAfter
	vpr.SenderValidUntil = validUntil
	vpr.PmValidAfter = pmValidAfter
	vpr.PmValidUntil = pmValidUntil
	statedb.Finalise(true)

	return vpr, nil
}

func applyPaymasterValidationFrame(vpr *ValidationPhaseResult, tx *types.Transaction, chainConfig *params.ChainConfig, signingHash common.Hash, evm *vm.EVM, gp *GasPool, statedb *state.StateDB, header *types.Header) ([]byte, uint64, uint64, uint64, error) {
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
		vpr.PmUsed = true
		resultPm, err := ApplyMessage(evm, paymasterMsg, gp)
		if err != nil {
			return nil, 0, 0, 0, err
		}
		if resultPm.Failed() {
			return nil, 0, 0, 0, resultPm.Err
		}
		if resultPm.Failed() {
			return nil, 0, 0, 0, errors.New("paymaster validation failed - invalid transaction")
		}
		pmValidationUsedGas = resultPm.UsedGas
		paymasterContext, pmValidAfter, pmValidUntil, err = vpr.validatePaymasterEntryPointCall()
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
		From:              AA_ENTRY_POINT,
		To:                tx.Sender,
		Value:             big.NewInt(0),
		GasLimit:          tx.ValidationGasLimit - deploymentUsedGas,
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
	if tx.Paymaster == nil || tx.Paymaster.Cmp(common.Address{}) == 0 {
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

func (vpr *ValidationPhaseResult) validateAccountEntryPointCall() (uint64, uint64, error) {
	if len(vpr.EpCalls) == 0 {
		return 0, 0, errors.New("validation did not call the EntryPoint callback")
	}
	if (!vpr.PmUsed && len(vpr.EpCalls) > 1) || (vpr.PmUsed && len(vpr.EpCalls) > 2) {
		return 0, 0, errors.New("validation illegally called the EntryPoint callback multiple times")
	}
	epCall := vpr.EpCalls[0]

	if len(epCall.input) != 68 {
		return 0, 0, errors.New("invalid account return data length")
	}
	magicExpected, validUntil, validAfter := UnpackValidationData(epCall.input)
	//todo: we check first 8 bytes of the 20-byte address (the rest is expected to be zeros)
	if magicExpected != MAGIC_VALUE_SENDER {
		if magicExpected == MAGIC_VALUE_SIGFAIL {
			return 0, 0, errors.New("account signature error")
		}
		return 0, 0, errors.New("account did not return correct MAGIC_VALUE")
	}
	return validAfter, validUntil, nil
}

func (vpr *ValidationPhaseResult) validatePaymasterEntryPointCall() (context []byte, validAfter, validUntil uint64, error error) {
	if len(vpr.EpCalls) < 2 {
		return nil, 0, 0, errors.New("validation did not call the EntryPoint callback")
	}
	if vpr.PmUsed && len(vpr.EpCalls) > 2 {
		return nil, 0, 0, errors.New("validation illegally called the EntryPoint callback multiple times")
	}
	epCall := vpr.EpCalls[1]

	if len(epCall.input) < 100 {
		return nil, 0, 0, errors.New("invalid paymaster callback data length")
	}
	magicExpected, validUntil, validAfter, context, err := UnpackPaymasterValidationReturn(epCall.input)
	if err != nil {
		return nil, 0, 0, err
	}
	//,  := UnpackValidationData(validationData)
	if magicExpected != MAGIC_VALUE_PAYMASTER {
		return nil, 0, 0, errors.New("paymaster did not return correct MAGIC_VALUE")
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

func (vpr *ValidationPhaseResult) OnEnter(depth int, typ byte, from common.Address, to common.Address, input []byte, gas uint64, value *big.Int) {
	if vpr.OnEnterSuper != nil {
		vpr.OnEnterSuper(depth, typ, from, to, input, gas, value)
	}
	isRip7560EntryPoint := to.Cmp(AA_ENTRY_POINT) == 0
	if isRip7560EntryPoint {
		inputBytes := make([]byte, len(input))
		copy(inputBytes, input)
		vpr.EpCalls = append(vpr.EpCalls, &EntryPointCall{
			caller: from,
			input:  inputBytes,
		})
	}
}
