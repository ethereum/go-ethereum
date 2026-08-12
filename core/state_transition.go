// Copyright 2014 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

package core

import (
	"bytes"
	"fmt"
	"math"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/tracing"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/crypto/kzg4844"
	"github.com/ethereum/go-ethereum/params"
	"github.com/holiman/uint256"
)

// ExecutionResult includes all output after executing given evm
// message no matter the execution itself is successful or not.
type ExecutionResult struct {
	UsedGas    uint64 // Total used gas, refunded gas is deducted
	MaxUsedGas uint64 // Maximum gas consumed during execution, excluding gas refunds.
	Err        error  // Any error encountered during the execution(listed in core/vm/errors.go)
	ReturnData []byte // Returned data from evm(function result or data supplied with revert opcode)

	// EIP-8141 frame transaction fields:
	Payer         common.Address
	FrameReceipts []*types.FrameReceipt
}

// Unwrap returns the internal evm error which allows us for further
// analysis outside.
func (result *ExecutionResult) Unwrap() error {
	return result.Err
}

// Failed returns the indicator whether the execution is successful or not
func (result *ExecutionResult) Failed() bool { return result.Err != nil }

// Return is a helper function to help caller distinguish between revert reason
// and function return. Return returns the data after execution if no error occurs.
func (result *ExecutionResult) Return() []byte {
	if result.Err != nil {
		return nil
	}
	return common.CopyBytes(result.ReturnData)
}

// Revert returns the concrete revert reason if the execution is aborted by `REVERT`
// opcode. Note the reason can be nil if no data supplied with revert opcode.
func (result *ExecutionResult) Revert() []byte {
	if result.Err != vm.ErrExecutionReverted {
		return nil
	}
	return common.CopyBytes(result.ReturnData)
}

// IntrinsicGas computes the 'intrinsic gas' for a message with the given data.
func IntrinsicGas(data []byte, accessList types.AccessList, authList []types.SetCodeAuthorization, from common.Address, to *common.Address, value *uint256.Int, rules params.Rules) (uint64, error) {
	isContractCreation := to == nil

	// Set the starting gas for the raw transaction
	var gas uint64
	if rules.IsAmsterdam {
		gas = intrinsicBaseGasEIP2780(from, to, value)
	} else if isContractCreation && rules.IsHomestead {
		gas = params.TxGasContractCreation
	} else {
		gas = params.TxGas
	}
	// Add gas for authorizations
	if authList != nil {
		if rules.IsAmsterdam {
			gas += uint64(len(authList)) * params.ExecutionPerAuthBaseCost
		} else {
			gas += uint64(len(authList)) * params.CallNewAccountGas
		}
	}
	// Bump the required gas by the amount of transactional data
	dataLen := uint64(len(data))
	if dataLen > 0 {
		// Zero and non-zero bytes are priced differently
		z := uint64(bytes.Count(data, []byte{0}))
		nz := dataLen - z

		// Make sure we don't exceed uint64 for all data combinations
		nonZeroGas := params.TxDataNonZeroGasFrontier
		if rules.IsIstanbul {
			nonZeroGas = params.TxDataNonZeroGasEIP2028
		}
		if (math.MaxUint64-gas)/nonZeroGas < nz {
			return 0, ErrGasUintOverflow
		}
		gas += nz * nonZeroGas

		if (math.MaxUint64-gas)/params.TxDataZeroGas < z {
			return 0, ErrGasUintOverflow
		}
		gas += z * params.TxDataZeroGas

		if isContractCreation && rules.IsShanghai {
			lenWords := toWordSize(dataLen)
			if (math.MaxUint64-gas)/params.InitCodeWordGas < lenWords {
				return 0, ErrGasUintOverflow
			}
			gas += lenWords * params.InitCodeWordGas
		}
	}
	// Add the gas for accessList
	if accessList != nil {
		addresses := uint64(len(accessList))
		storageKeys := uint64(accessList.StorageKeys())

		// Amsterdam re-prices the per-entry access-list cost
		addressCost := params.TxAccessListAddressGas
		storageKeyCost := params.TxAccessListStorageKeyGas
		if rules.IsAmsterdam {
			addressCost = params.TxAccessListAddressGasAmsterdam
			storageKeyCost = params.TxAccessListStorageKeyGasAmsterdam
		}
		if (math.MaxUint64-gas)/addressCost < addresses {
			return 0, ErrGasUintOverflow
		}
		gas += addresses * addressCost
		if (math.MaxUint64-gas)/storageKeyCost < storageKeys {
			return 0, ErrGasUintOverflow
		}
		gas += storageKeys * storageKeyCost

		// EIP-7981: access list data is charged in addition to the base charge.
		if rules.IsAmsterdam {
			const (
				addressCost    = common.AddressLength * params.TxCostFloorPerToken7976 * params.TxTokenPerNonZeroByte
				storageKeyCost = common.HashLength * params.TxCostFloorPerToken7976 * params.TxTokenPerNonZeroByte
			)
			if (math.MaxUint64-gas)/addressCost < addresses {
				return 0, ErrGasUintOverflow
			}
			gas += addresses * addressCost
			if (math.MaxUint64-gas)/storageKeyCost < storageKeys {
				return 0, ErrGasUintOverflow
			}
			gas += storageKeys * storageKeyCost
		}
	}
	return gas, nil
}

// intrinsicBaseGasEIP2780 computes the intrinsic base cost of the transaction.
func intrinsicBaseGasEIP2780(from common.Address, to *common.Address, value *uint256.Int) uint64 {
	var (
		isContractCreation = to == nil
		isSelfTransfer     = to != nil && *to == from
		hasValue           = value != nil && !value.IsZero()
	)
	// tx.sender: signature recovery, the sender account's access and write,
	// and the inclusion of the transaction in the block (which is transient
	// and expires with history).
	gas := params.TxBaseCost2780

	// tx.to charge. Per EIP-2780 the recipient touch is charged at the cold
	// rate unconditionally at the intrinsic phase, independent of the account's
	// warm/cold state.
	switch {
	case isSelfTransfer:
		// The recipient account is already accessed and written as the sender.
	case isContractCreation:
		gas += params.CreateAccessAmsterdam
	default:
		gas += params.ColdAccountAccessAmsterdam
	}

	// tx.value charge.
	switch {
	case !hasValue || isSelfTransfer || isContractCreation:
		// No transfer log and no recipient balance write.
	default:
		gas += params.TxValueCost2780
	}
	return gas
}

// FloorDataGas computes the minimum gas required for a transaction based on its data tokens (EIP-7623).
func FloorDataGas(rules params.Rules, from common.Address, to *common.Address, value *uint256.Int, data []byte, accessList types.AccessList) (uint64, error) {
	var (
		tokens    uint64
		tokenCost uint64
	)
	if rules.IsAmsterdam {
		// EIP-7976 changes how calldata is priced.
		// From 10/40 to 64/64 for zero/non-zero bytes.
		tokenCost = params.TxCostFloorPerToken7976
		dataLen := uint64(len(data))
		if math.MaxUint64/params.TxTokenPerNonZeroByte < dataLen {
			return 0, ErrGasUintOverflow
		}
		tokens = dataLen * params.TxTokenPerNonZeroByte

		// EIP-7981 adds additional tokens for every entry in the accesslist
		const addressTokenCost = uint64(common.AddressLength) * params.TxTokenPerNonZeroByte
		addresses := uint64(len(accessList))
		if (math.MaxUint64-tokens)/addressTokenCost < addresses {
			return 0, ErrGasUintOverflow
		}
		tokens += addresses * addressTokenCost

		const storageKeyTokenCost = uint64(common.HashLength) * params.TxTokenPerNonZeroByte
		storageKeys := uint64(accessList.StorageKeys())
		if (math.MaxUint64-tokens)/storageKeyTokenCost < storageKeys {
			return 0, ErrGasUintOverflow
		}
		tokens += storageKeys * storageKeyTokenCost
	} else {
		var (
			z  = uint64(bytes.Count(data, []byte{0}))
			nz = uint64(len(data)) - z
		)
		// Pre-Amsterdam
		if math.MaxUint64/params.TxTokenPerNonZeroByte < nz {
			return 0, ErrGasUintOverflow
		}
		tokens = nz * params.TxTokenPerNonZeroByte
		if math.MaxUint64-tokens < z {
			return 0, ErrGasUintOverflow
		}
		tokens += z
		tokenCost = params.TxCostFloorPerToken
	}

	// The floor is anchored to the transaction base cost. Under EIP-2780 that
	// base is the per-resource decomposition (the same one used by the intrinsic
	// gas), so the floor never undercuts the transaction's own base.
	floorBase := params.TxGas
	if rules.IsAmsterdam {
		floorBase = intrinsicBaseGasEIP2780(from, to, value)
	}
	// Check for overflow
	if (math.MaxUint64-floorBase)/tokenCost < tokens {
		return 0, ErrGasUintOverflow
	}
	// Minimum gas required for a transaction based on its data tokens (EIP-7623).
	return floorBase + tokens*tokenCost, nil
}

// toWordSize returns the ceiled word size required for init code payment calculation.
func toWordSize(size uint64) uint64 {
	if size > math.MaxUint64-31 {
		return math.MaxUint64/32 + 1
	}
	return (size + 31) / 32
}

// A Message contains the data derived from a single transaction that is relevant to state
// processing.
type Message struct {
	To                    *common.Address
	From                  common.Address
	Nonce                 uint64
	Value                 *uint256.Int
	GasLimit              uint64
	GasPrice              *uint256.Int
	GasFeeCap             *uint256.Int
	GasTipCap             *uint256.Int
	Data                  []byte
	AccessList            types.AccessList
	BlobGasFeeCap         *uint256.Int
	BlobHashes            []common.Hash
	SetCodeAuthorizations []types.SetCodeAuthorization

	// FrameTx carries the underlying frame transaction for EIP-8141. When
	// non-nil, the message is executed via the frame transaction path.
	FrameTx *types.FrameTx

	// When SkipNonceChecks is true, the message nonce is not checked against the
	// account nonce in state.
	//
	// This field will be set to true for operations like RPC eth_call
	// or the state prefetching.
	SkipNonceChecks bool

	// When set, the message is not treated as a transaction, and certain
	// transaction-specific checks are skipped:
	//
	// - From is not verified to be an EOA
	// - GasLimit is not checked against the protocol defined tx gaslimit
	SkipTransactionChecks bool
}

// TransactionToMessage converts a transaction into a Message.
func TransactionToMessage(tx *types.Transaction, s types.Signer, baseFee *big.Int) (*Message, error) {
	from, err := types.Sender(s, tx)
	if err != nil {
		return nil, err
	}
	gasPrice, overflow := uint256.FromBig(tx.GasPrice())
	if overflow {
		return nil, fmt.Errorf("%w: address %v, maxFeePerGas bit length: %d", ErrFeeCapVeryHigh,
			from.Hex(), tx.GasPrice().BitLen())
	}
	txGasFeeCap := tx.GasFeeCap()
	gasFeeCap, overflow := uint256.FromBig(txGasFeeCap)
	if overflow {
		return nil, fmt.Errorf("%w: address %v, maxFeePerGas bit length: %d", ErrFeeCapVeryHigh,
			from.Hex(), tx.GasFeeCap().BitLen())
	}
	txGasTipCap := tx.GasTipCap()
	gasTipCap, overflow := uint256.FromBig(txGasTipCap)
	if overflow {
		return nil, fmt.Errorf("%w: address %v, maxPriorityFeePerGas bit length: %d", ErrTipVeryHigh,
			from.Hex(), tx.GasTipCap().BitLen())
	}
	value, overflow := uint256.FromBig(tx.Value())
	if overflow {
		return nil, fmt.Errorf("value exceeds 256 bits: address %v", from.Hex())
	}
	blobGasFeeCap, overflow := uint256.FromBig(tx.BlobGasFeeCap())
	if overflow {
		return nil, fmt.Errorf("blobGasFeeCap exceeds 256 bits: address %v", from.Hex())
	}

	msg := &Message{
		From:                  from,
		Nonce:                 tx.Nonce(),
		GasLimit:              tx.Gas(),
		GasPrice:              gasPrice,
		GasFeeCap:             gasFeeCap,
		GasTipCap:             gasTipCap,
		To:                    tx.To(),
		Value:                 value,
		Data:                  tx.Data(),
		AccessList:            tx.AccessList(),
		SetCodeAuthorizations: tx.SetCodeAuthorizations(),
		SkipNonceChecks:       false,
		SkipTransactionChecks: false,
		BlobHashes:            tx.BlobHashes(),
		BlobGasFeeCap:         blobGasFeeCap,
	}
	if ftx, ok := tx.Inner().(*types.FrameTx); ok {
		msg.FrameTx = ftx
	}
	// If baseFee provided, set gasPrice to effectiveGasPrice.
	if baseFee != nil {
		effectiveGasPrice := new(big.Int).Add(baseFee, txGasTipCap)
		if effectiveGasPrice.Cmp(txGasFeeCap) > 0 {
			effectiveGasPrice = txGasFeeCap
		}
		// EffectiveGasPrice is already capped by txGasFeeCap, therefore
		// the overflow check is not required.
		msg.GasPrice = uint256.MustFromBig(effectiveGasPrice)
	}
	return msg, nil
}

// isExpiryVerifierFrame reports whether a frame is an EIP-8141 expiry verifier
// frame: a VERIFY frame whose target is EXPIRY_VERIFIER. Frames in other modes
// that happen to target that address are ordinary calls into the predeploy.
func isExpiryVerifierFrame(frame *types.Frame) bool {
	return frame.Mode == types.FrameModeVerify && frame.Target != nil && *frame.Target == params.FrameExpiryVerifierAddress
}

// validateFrameTx performs the static validity checks on a frame transaction
// envelope defined by EIP-8141.
func validateFrameTx(fTx *types.FrameTx) error {
	if fTx.ChainID == nil || fTx.ChainID.BitLen() >= 256 {
		return fmt.Errorf("%w: chain id out of range", ErrFrameInvalid)
	}
	if len(fTx.Frames) == 0 || len(fTx.Frames) > params.FrameTxMaxFrames {
		return fmt.Errorf("%w: invalid number of frames %d", ErrFrameInvalid, len(fTx.Frames))
	}
	// EIP-4844 blob constraints. The frame path does not run preCheck, so the
	// versioned hashes have to be validated here.
	if len(fTx.BlobHashes) > params.BlobTxMaxBlobs {
		return fmt.Errorf("%w: %d blobs exceeds the per-transaction limit", ErrFrameInvalid, len(fTx.BlobHashes))
	}
	for i, hash := range fTx.BlobHashes {
		if !kzg4844.IsValidVersionedHash(hash[:]) {
			return fmt.Errorf("%w: blob %d has invalid hash version", ErrFrameInvalid, i)
		}
	}
	if len(fTx.BlobHashes) == 0 && fTx.BlobFeeCap != nil && !fTx.BlobFeeCap.IsZero() {
		return fmt.Errorf("%w: non-zero blob fee with no blob hashes", ErrFrameInvalid)
	}
	for i := range fTx.Signatures {
		sig := &fTx.Signatures[i]
		switch sig.Scheme {
		case types.SignatureSchemeSecp256k1, types.SignatureSchemeP256:
			if len(sig.Signer) != 0 && len(sig.Signer) != 20 {
				return fmt.Errorf("%w: invalid signer length", ErrFrameInvalid)
			}
		case types.SignatureSchemeArbitrary:
			if len(sig.Signer) != 0 {
				return fmt.Errorf("%w: ARBITRARY signature must have empty signer", ErrFrameInvalid)
			}
		default:
			return fmt.Errorf("%w: invalid signature scheme %d", ErrFrameInvalid, sig.Scheme)
		}
		if len(sig.Msg) != 0 && len(sig.Msg) != 32 {
			return fmt.Errorf("%w: invalid msg length", ErrFrameInvalid)
		}
	}
	var (
		totalFrameGas uint64
		expiryFrames  int
	)
	for i := range fTx.Frames {
		frame := &fTx.Frames[i]
		if frame.Mode >= 3 || frame.Flags >= 8 {
			return fmt.Errorf("%w: invalid mode or flags", ErrFrameInvalid)
		}
		if frame.Mode != types.FrameModeSender && (frame.Value != nil && !frame.Value.IsZero()) {
			return fmt.Errorf("%w: non-zero value on non-SENDER frame", ErrFrameInvalid)
		}
		if math.MaxUint64-totalFrameGas < frame.GasLimit {
			return fmt.Errorf("%w: frame gas limit overflow", ErrFrameInvalid)
		}
		totalFrameGas += frame.GasLimit
		// Approval of execution is only allowed when target is nil or the sender.
		if frame.Flags&types.FrameFlagApproveExecution != 0 {
			if frame.Target != nil && *frame.Target != fTx.Sender {
				return fmt.Errorf("%w: APPROVE_EXECUTION target must be the sender", ErrFrameInvalid)
			}
		}
		// Atomic batch flag requires a subsequent non-VERIFY frame.
		if frame.Flags&types.FrameFlagAtomicBatch != 0 {
			if frame.Mode == types.FrameModeVerify {
				return fmt.Errorf("%w: atomic batch on VERIFY frame", ErrFrameInvalid)
			}
			if i+1 >= len(fTx.Frames) || fTx.Frames[i+1].Mode == types.FrameModeVerify {
				return fmt.Errorf("%w: atomic batch must be followed by a non-VERIFY frame", ErrFrameInvalid)
			}
		}
		// An expiry verifier frame must carry exactly an 8-byte deadline and no
		// flags or value, and at most one may appear per transaction.
		if isExpiryVerifierFrame(frame) {
			expiryFrames++
			if expiryFrames > 1 {
				return fmt.Errorf("%w: multiple expiry verifier frames", ErrFrameInvalid)
			}
			if frame.Flags != 0 || len(frame.Data) != params.FrameTxExpiryDataLength {
				return fmt.Errorf("%w: malformed expiry verifier frame %d", ErrFrameInvalid, i)
			}
		}
	}
	return nil
}

// ApplyMessage computes the new state by applying the given message
// against the old state within the environment.
//
// ApplyMessage returns the bytes returned by any EVM execution (if it took place),
// the gas used (which includes gas refunds) and an error if it failed. An error always
// indicates a core error meaning that the message would always fail for that particular
// state and would never be accepted within a block.
func ApplyMessage(evm *vm.EVM, msg *Message, gp *GasPool) (*ExecutionResult, error) {
	// Do not panic if the gas pool is nil. This is allowed when executing
	// a single message via RPC invocation.
	if gp == nil {
		gp = NewGasPool(msg.GasLimit)
	}
	evm.SetTxContext(NewEVMTxContext(msg))
	return newStateTransition(evm, msg, gp).execute()
}

// stateTransition represents a state transition.
//
// == The State Transitioning Model
//
// A state transition is a change made when a transaction is applied to the current world
// state. The state transitioning model does all the necessary work to work out a valid new
// state root.
//
//  1. Nonce handling
//  2. Pre pay gas
//  3. Create a new state object if the recipient is nil
//  4. Value transfer
//
// == If contract creation ==
//
//	4a. Attempt to run transaction data
//	4b. If valid, use result as code for the new state object
//
// == end ==
//
//  5. Run Script section
//  6. Derive new state root
type stateTransition struct {
	gp           *GasPool
	msg          *Message
	gasRemaining vm.GasBudget
	state        vm.StateDB
	evm          *vm.EVM
}

// newStateTransition initialises and returns a new state transition object.
func newStateTransition(evm *vm.EVM, msg *Message, gp *GasPool) *stateTransition {
	return &stateTransition{
		gp:    gp,
		evm:   evm,
		msg:   msg,
		state: evm.StateDB,
	}
}

// to returns the recipient of the message.
func (st *stateTransition) to() common.Address {
	if st.msg == nil || st.msg.To == nil /* contract creation */ {
		return common.Address{}
	}
	return *st.msg.To
}

// buyGas pre-pays gas from the sender's balance.
//
// The balance requirement is the worst-case ETH the tx may need to lock
// up: `msg.GasLimit × max(msg.GasPrice, msg.GasFeeCap) + msg.Value`,
// plus `blobGas × msg.BlobGasFeeCap` under Cancun. Insufficient balance
// returns ErrInsufficientFunds.
//
// After the check, the sender is actually debited `msg.GasLimit × msg.GasPrice`
// (plus `blobGas × blobBaseFee` under Cancun), the cap-vs-tip differential
// is settled at tx end.
func (st *stateTransition) buyGas() error {
	mgval := new(uint256.Int).SetUint64(st.msg.GasLimit)
	_, overflow := mgval.MulOverflow(mgval, st.msg.GasPrice)
	if overflow {
		return fmt.Errorf("%w: address %v required balance exceeds 256 bits", ErrInsufficientFunds, st.msg.From.Hex())
	}
	balanceCheck := new(uint256.Int).Set(mgval)
	if st.msg.GasFeeCap != nil {
		balanceCheck.SetUint64(st.msg.GasLimit)
		if _, overflow := balanceCheck.MulOverflow(balanceCheck, st.msg.GasFeeCap); overflow {
			return fmt.Errorf("%w: address %v required balance exceeds 256 bits", ErrInsufficientFunds, st.msg.From.Hex())
		}
	}
	if st.msg.Value != nil {
		if _, overflow := balanceCheck.AddOverflow(balanceCheck, st.msg.Value); overflow {
			return fmt.Errorf("%w: address %v required balance exceeds 256 bits", ErrInsufficientFunds, st.msg.From.Hex())
		}
	}

	if st.evm.ChainConfig().IsCancun(st.evm.Context.BlockNumber, st.evm.Context.Time) {
		if blobGas := st.blobGasUsed(); blobGas > 0 {
			// Check that the user has enough funds to cover blobGasUsed * tx.BlobGasFeeCap
			blobBalanceCheck := new(uint256.Int).SetUint64(blobGas)
			if _, overflow := blobBalanceCheck.MulOverflow(blobBalanceCheck, st.msg.BlobGasFeeCap); overflow {
				return fmt.Errorf("%w: address %v required balance exceeds 256 bits", ErrInsufficientFunds, st.msg.From.Hex())
			}
			if _, overflow := balanceCheck.AddOverflow(balanceCheck, blobBalanceCheck); overflow {
				return fmt.Errorf("%w: address %v required balance exceeds 256 bits", ErrInsufficientFunds, st.msg.From.Hex())
			}
			// Pay for blobGasUsed * actual blob fee
			blobBaseFee, overflow := uint256.FromBig(st.evm.Context.BlobBaseFee)
			if overflow {
				return fmt.Errorf("invalid blobBaseFee: %v", st.evm.Context.BlobBaseFee)
			}
			blobFee := new(uint256.Int).SetUint64(blobGas)

			// In practice, overflow checking is unnecessary, as blobBaseFee cannot exceed
			// BlobGasFeeCap. However, in eth_call it is still possible for users to specify
			// an excessively large blob base fee and bypass the blob base fee validation.
			_, overflow = blobFee.MulOverflow(blobFee, blobBaseFee)
			if overflow {
				return fmt.Errorf("%w: address %v required balance exceeds 256 bits", ErrInsufficientFunds, st.msg.From.Hex())
			}
			_, overflow = mgval.AddOverflow(mgval, blobFee)
			if overflow {
				return fmt.Errorf("%w: address %v required balance exceeds 256 bits", ErrInsufficientFunds, st.msg.From.Hex())
			}
		}
	}
	if have, want := st.state.GetBalance(st.msg.From), balanceCheck; have.Cmp(want) < 0 {
		return fmt.Errorf("%w: address %v have %v want %v", ErrInsufficientFunds, st.msg.From.Hex(), have, want)
	}
	// Deduct the gas cost from the sender's balance
	st.state.SubBalance(st.msg.From, mgval, tracing.BalanceDecreaseGasBuy)
	return nil
}

// initRuntimeGasBudget initializes the transaction's running gas budget with the
// gas remaining after the intrinsic cost has been deducted.
//
// After Amsterdam (EIP-8037) the intrinsic cost counts towards the EIP-7825
// execution-gas cap:
//
//	evm_gas              = tx.gas - intrinsic_gas
//	execution_gas_budget = TX_MAX_GAS_LIMIT - intrinsic_gas
//	gas_left             = min(execution_gas_budget, evm_gas)
//	state_gas_reservoir  = evm_gas - gas_left
func (st *stateTransition) initRuntimeGasBudget(rules params.Rules, intrinsicGas uint64) {
	evmGas := st.msg.GasLimit - intrinsicGas
	gasLeft := evmGas
	if rules.IsAmsterdam {
		gasLeft = min(params.MaxTxGas-intrinsicGas, evmGas)
	}
	st.gasRemaining = vm.NewGasBudget(gasLeft, evmGas-gasLeft)

	if st.evm.Config.Tracer.HasGasHook() {
		st.evm.Config.Tracer.EmitGasChange(tracing.Gas{}, tracing.Gas{Execution: st.msg.GasLimit}, tracing.GasChangeTxInitialBalance)
		st.evm.Config.Tracer.EmitGasChange(tracing.Gas{Execution: st.msg.GasLimit}, st.gasRemaining.AsTracing(), tracing.GasChangeTxIntrinsicGas)
	}
}

// preCheck performs all pre-execution validation that does not require
// the EVM to run, then ends by calling buyGas to lock ether for prepay.
// It returns a consensus error if any of the following fail:
//
//   - Sender nonce matches state and is not at 2^64-1 (EIP-2681).
//
//   - EIP-7825 per-tx gas-limit cap on Osaka chains pre-Amsterdam.
//
//   - EIP-3607 sender-is-EOA, allowing accounts whose only code is an
//     EIP-7702 delegation designator.
//
//   - EIP-1559 fee-cap, tip-cap and base-fee constraints (London+).
//
//   - Blob-tx structural checks: non-nil `To`, non-empty hash list,
//     valid KZG versioned hashes, count below `BlobTxMaxBlobs` (Osaka+).
//
//   - Blob fee-cap not below the current blob base fee (Cancun+).
//
//   - EIP-7702 set-code-tx shape: non-nil `To` and non-empty
//     authorization list.
//
//   - EIP-3860 init code size cap on create transactions (Shanghai+,
//     with the raised Amsterdam cap).
//
//   - Insufficient block gas budget for including the transaction.
//
// The SkipNonceChecks / SkipTransactionChecks / NoBaseFee flags bypass
// subsets of these checks for simulation paths (eth_call, eth_estimateGas).
func (st *stateTransition) preCheck(rules params.Rules) error {
	// Only check transactions that are not fake
	msg := st.msg
	if !msg.SkipNonceChecks {
		// Make sure this transaction's nonce is correct.
		stNonce := st.state.GetNonce(msg.From)
		if msgNonce := msg.Nonce; stNonce < msgNonce {
			return fmt.Errorf("%w: address %v, tx: %d state: %d", ErrNonceTooHigh,
				msg.From.Hex(), msgNonce, stNonce)
		} else if stNonce > msgNonce {
			return fmt.Errorf("%w: address %v, tx: %d state: %d", ErrNonceTooLow,
				msg.From.Hex(), msgNonce, stNonce)
		} else if stNonce+1 < stNonce {
			return fmt.Errorf("%w: address %v, nonce: %d", ErrNonceMax,
				msg.From.Hex(), stNonce)
		}
	}
	if !msg.SkipTransactionChecks {
		// Verify tx gas limit does not exceed EIP-7825 cap.
		if !rules.IsAmsterdam && rules.IsOsaka && msg.GasLimit > params.MaxTxGas {
			return fmt.Errorf("%w (cap: %d, tx: %d)", ErrGasLimitTooHigh, params.MaxTxGas, msg.GasLimit)
		}
		// Make sure the sender is an EOA
		code := st.state.GetCode(msg.From)
		_, delegated := types.ParseDelegation(code)
		if len(code) > 0 && !delegated {
			return fmt.Errorf("%w: address %v, len(code): %d", ErrSenderNoEOA, msg.From.Hex(), len(code))
		}
	}
	// Make sure that transaction gasFeeCap is greater than the baseFee (post london)
	if rules.IsLondon {
		// Skip the checks if gas fields are zero and baseFee was explicitly disabled (eth_call)
		skipCheck := st.evm.Config.NoBaseFee && msg.GasFeeCap.BitLen() == 0 && msg.GasTipCap.BitLen() == 0
		if !skipCheck {
			if msg.GasFeeCap.Cmp(msg.GasTipCap) < 0 {
				return fmt.Errorf("%w: address %v, maxPriorityFeePerGas: %s, maxFeePerGas: %s", ErrTipAboveFeeCap,
					msg.From.Hex(), msg.GasTipCap, msg.GasFeeCap)
			}
			// This will panic if baseFee is nil, but basefee presence is verified
			// as part of header validation.
			if msg.GasFeeCap.CmpBig(st.evm.Context.BaseFee) < 0 {
				return fmt.Errorf("%w: address %v, maxFeePerGas: %s, baseFee: %s", ErrFeeCapTooLow,
					msg.From.Hex(), msg.GasFeeCap, st.evm.Context.BaseFee)
			}
		}
	}
	// Check the blob version validity
	if msg.BlobHashes != nil {
		// The to field of a blob tx type is mandatory, and a `BlobTx` transaction internally
		// has it as a non-nillable value, so any msg derived from blob transaction has it non-nil.
		// However, messages created through RPC (eth_call) don't have this restriction.
		if msg.To == nil {
			return ErrBlobTxCreate
		}
		if len(msg.BlobHashes) == 0 {
			return ErrMissingBlobHashes
		}
		if rules.IsOsaka && len(msg.BlobHashes) > params.BlobTxMaxBlobs {
			return ErrTooManyBlobs
		}
		for i, hash := range msg.BlobHashes {
			if !kzg4844.IsValidVersionedHash(hash[:]) {
				return fmt.Errorf("blob %d has invalid hash version", i)
			}
		}
	}
	// Check that the user is paying at least the current blob fee
	if rules.IsCancun {
		if st.blobGasUsed() > 0 {
			// Skip the checks if gas fields are zero and blobBaseFee was explicitly disabled (eth_call)
			skipCheck := st.evm.Config.NoBaseFee && msg.BlobGasFeeCap.BitLen() == 0
			if !skipCheck {
				// This will panic if blobBaseFee is nil, but blobBaseFee presence
				// is verified as part of header validation.
				if msg.BlobGasFeeCap.CmpBig(st.evm.Context.BlobBaseFee) < 0 {
					return fmt.Errorf("%w: address %v blobGasFeeCap: %v, blobBaseFee: %v", ErrBlobFeeCapTooLow,
						msg.From.Hex(), msg.BlobGasFeeCap, st.evm.Context.BlobBaseFee)
				}
			}
		}
	}
	// Check that EIP-7702 authorization list signatures are well formed.
	if msg.SetCodeAuthorizations != nil {
		if msg.To == nil {
			return fmt.Errorf("%w (sender %v)", ErrSetCodeTxCreate, msg.From)
		}
		if len(msg.SetCodeAuthorizations) == 0 {
			return fmt.Errorf("%w (sender %v)", ErrEmptyAuthList, msg.From)
		}
	}
	// Check whether the init code size has been exceeded (EIP-3860).
	if msg.To == nil {
		if err := vm.CheckMaxInitCodeSize(&rules, uint64(len(msg.Data))); err != nil {
			return err
		}
	}
	// Reserve the gas budget in the block gas pool
	var err error
	if rules.IsAmsterdam {
		err = st.gp.CheckGasAmsterdam(min(st.msg.GasLimit, params.MaxTxGas), st.msg.GasLimit)
	} else {
		err = st.gp.CheckGasLegacy(st.msg.GasLimit)
	}
	if err != nil {
		return err
	}
	return st.buyGas()
}

// execute transitions the state by applying the current message and
// returns the EVM execution result with the following fields:
//
//   - used gas: total gas used, including gas refunded
//   - peak used gas: maximum gas used before applying refunds
//   - returndata: data returned by the EVM
//   - execution error: EVM-level errors that abort execution, such as
//     ErrOutOfGas or ErrExecutionReverted
//
// If a consensus error is encountered, it is returned directly with a
// nil EVM execution result.
func (st *stateTransition) execute() (*ExecutionResult, error) {
	var (
		msg              = st.msg
		rules            = st.evm.ChainConfig().Rules(st.evm.Context.BlockNumber, st.evm.Context.Random != nil, st.evm.Context.Time)
		contractCreation = msg.To == nil
		floorDataGas     uint64
	)
	// EIP-8141 frame transactions follow a dedicated execution path.
	if msg.FrameTx != nil {
		return st.executeFrame(rules)
	}
	// Validate the message and pre-pay gas.
	if err := st.preCheck(rules); err != nil {
		return nil, err
	}
	// Calculate the intrinsic gas of this transaction and make sure the gas limit
	// is sufficient to cover that.
	intrinsicGas, err := IntrinsicGas(msg.Data, msg.AccessList, msg.SetCodeAuthorizations, msg.From, msg.To, msg.Value, rules)
	if err != nil {
		return nil, err
	}
	if msg.GasLimit < intrinsicGas {
		return nil, fmt.Errorf("%w: have %d, want %d", ErrIntrinsicGas, msg.GasLimit, intrinsicGas)
	}
	// Validate the EIP-7623 calldata floor against the gas limit. The floor inflates
	// the total gas usage at tx end, so the gas limit must be sufficient to cover that.
	if rules.IsPrague {
		floorDataGas, err = FloorDataGas(rules, msg.From, msg.To, msg.Value, msg.Data, msg.AccessList)
		if err != nil {
			return nil, err
		}
		// Make sure the transaction has sufficient gas allowance to
		// pay the floor cost.
		if msg.GasLimit < floorDataGas {
			return nil, fmt.Errorf("%w: have %d, want %d", ErrFloorDataGas, msg.GasLimit, floorDataGas)
		}
	}
	// In Amsterdam, the transaction gas limit is allowed to exceed
	// params.MaxTxGas, but the intrinsic cost and calldata floor
	// cost is still capped by it.
	if rules.IsAmsterdam && max(intrinsicGas, floorDataGas) > params.MaxTxGas {
		return nil, fmt.Errorf("%w: intrinsic cost %v, floor: %v", ErrFloorDataGas, intrinsicGas, floorDataGas)
	}

	// EIP-4762 setup
	if rules.IsEIP4762 {
		st.evm.AccessEvents.AddTxOrigin(msg.From)

		if targetAddr := msg.To; targetAddr != nil {
			st.evm.AccessEvents.AddTxDestination(*targetAddr, msg.Value.Sign() != 0, !st.state.Exist(*targetAddr))
		}
	}

	// Top-call affordability, the sender must still be able to cover the value
	// transfer of the top frame after gas pre-pay.
	value := msg.Value
	if value == nil {
		value = new(uint256.Int)
	}
	if !value.IsZero() && !st.evm.Context.CanTransfer(st.state, msg.From, value) {
		return nil, fmt.Errorf("%w: address %v", ErrInsufficientFundsForTransfer, msg.From.Hex())
	}

	// Execute the preparatory steps for state transition which includes:
	// - prepare accessList(post-berlin)
	// - reset transient storage(EIP-1153)
	// - enable block-level accessList construction (EIP-7928)
	st.state.Prepare(rules, msg.From, st.evm.Context.Coinbase, msg.To, vm.ActivePrecompiles(rules), msg.AccessList)

	// Initialize the running gas budget with the post-intrinsic remainder.
	st.initRuntimeGasBudget(rules, intrinsicGas)

	// Execute the top-most frame
	var (
		ret   []byte
		vmerr error // vm errors do not effect consensus
	)
	if contractCreation {
		ret, vmerr = st.executeCreate(rules, value)
	} else {
		ret, vmerr = st.executeCall(rules, value)
	}

	// Settle down the gas usage and refund the ETH back if any remaining
	gasUsed, peakUsed, err := st.settleGas(rules, floorDataGas)
	if err != nil {
		return nil, err
	}

	// Pay the effective transaction fee to the specific coinbase
	effectiveTip := msg.GasPrice
	if rules.IsLondon {
		baseFee, overflow := uint256.FromBig(st.evm.Context.BaseFee)
		if overflow {
			return nil, fmt.Errorf("invalid baseFee: %v", st.evm.Context.BaseFee)
		}
		effectiveTip = new(uint256.Int).Sub(msg.GasPrice, baseFee)
	}
	if st.evm.Config.NoBaseFee && msg.GasFeeCap.Sign() == 0 && msg.GasTipCap.Sign() == 0 {
		// Skip fee payment when NoBaseFee is set and the fee fields
		// are 0. This avoids a negative effectiveTip being applied to
		// the coinbase when simulating calls.
	} else {
		fee := new(uint256.Int).SetUint64(gasUsed)
		fee.Mul(fee, effectiveTip)
		st.state.AddBalance(st.evm.Context.Coinbase, fee, tracing.BalanceIncreaseRewardTransactionFee)

		// add the coinbase to the witness iff the fee is greater than 0
		if rules.IsEIP4762 && fee.Sign() != 0 {
			st.evm.AccessEvents.AddAccount(st.evm.Context.Coinbase, true, math.MaxUint64)
		}
	}

	return &ExecutionResult{
		UsedGas:    gasUsed,
		MaxUsedGas: peakUsed,
		Err:        vmerr,
		ReturnData: ret,
	}, nil
}

// executeCreate runs the top-level frame of a contract-creation transaction
// and returns the EVM return data and the frame-level execution error.
func (st *stateTransition) executeCreate(rules params.Rules, value *uint256.Int) ([]byte, error) {
	msg := st.msg

	var chargedCreation bool
	if rules.IsAmsterdam {
		addr := crypto.CreateAddress(msg.From, st.state.GetNonce(msg.From))
		if st.state.Empty(addr) {
			if !st.chargeRuntimeGas(vm.GasCosts{StateGas: params.AccountCreationSize * st.evm.Context.CostPerStateByte}) {
				// The nonce increment normally performed inside evm.Create
				// must still happen for the included transaction.
				st.state.SetNonce(msg.From, st.state.GetNonce(msg.From)+1, tracing.NonceChangeContractCreator)

				entryGas := st.gasRemaining
				st.gasRemaining = st.gasRemaining.ExitHalt()
				st.traceHaltedTopFrame(vm.CREATE, addr, msg.Data, entryGas, st.gasRemaining, value)
				return nil, vm.ErrOutOfGas
			}
			chargedCreation = true
		}
	}
	// The first frame is entered with the gas remaining after the runtime
	// charges.
	ret, _, result, vmerr := st.evm.Create(msg.From, msg.Data, st.gasRemaining.ForwardAll(), value)
	st.gasRemaining.Absorb(result)

	// If the contract creation failed (e.g. the initcode reverted or halted),
	// refill the account-creation state gas charged at runtime.
	if rules.IsAmsterdam && chargedCreation && vmerr != nil {
		st.gasRemaining.RefundState(params.AccountCreationSize * st.evm.Context.CostPerStateByte)
	}
	// If the top-most frame halted, drain the leftover execution gas rather
	// than returning it to the sender. The frame exit itself already burned
	// its gas left, but the refill above repays the execution gas the charge
	// originally borrowed, and on a halt that repayment must be burned as
	// well. The state dimension is left untouched.
	if rules.IsAmsterdam && vmerr != nil && vmerr != vm.ErrExecutionReverted {
		st.gasRemaining.DrainExecution()
	}
	return ret, vmerr
}

// executeCall runs the top-level frame of a message-call transaction and
// returns the EVM return data and the frame-level execution error.
func (st *stateTransition) executeCall(rules params.Rules, value *uint256.Int) ([]byte, error) {
	msg := st.msg

	// Increment the nonce for the next transaction.
	st.state.SetNonce(msg.From, st.state.GetNonce(msg.From)+1, tracing.NonceChangeEoACall)

	if rules.IsAmsterdam {
		snapshot := st.state.Snapshot()
		entryGas := st.gasRemaining
		if !st.applyAuthorizations(rules, st.msg.SetCodeAuthorizations) {
			st.state.RevertToSnapshot(snapshot)
			st.gasRemaining = st.gasRemaining.ExitHalt()
			st.traceHaltedTopFrame(vm.CALL, st.to(), msg.Data, entryGas, st.gasRemaining, value)
			return nil, vm.ErrOutOfGas
		}
		if !st.chargeCallRecipientEIP2780(value) {
			st.state.RevertToSnapshot(snapshot)
			st.gasRemaining = st.gasRemaining.ExitHalt()
			st.traceHaltedTopFrame(vm.CALL, st.to(), msg.Data, entryGas, st.gasRemaining, value)
			return nil, vm.ErrOutOfGas
		}
	} else {
		// Apply EIP-7702 authorizations.
		st.applyAuthorizations(rules, msg.SetCodeAuthorizations)

		// Perform convenience warming of sender's delegation target. Although the
		// sender is already warmed in Prepare(..), it's possible a delegation to
		// the account was deployed during this transaction. To handle correctly,
		// simply wait until the final state of delegations is determined before
		// performing the resolution and warming.
		if addr, ok := types.ParseDelegation(st.state.GetCode(*msg.To)); ok {
			st.state.AddAddressToAccessList(addr)
		}
	}
	ret, result, vmerr := st.evm.Call(msg.From, st.to(), msg.Data, st.gasRemaining.ForwardAll(), value)
	st.gasRemaining.Absorb(result)

	// If the call frame reverts or halts exceptionally, the charged state-gas
	// is refilled back to the state reservoir in Amsterdam.
	if rules.IsAmsterdam && vmerr != nil && !value.IsZero() && st.evm.StateDB.Empty(st.to()) {
		st.gasRemaining.RefundState(params.AccountCreationSize * st.evm.Context.CostPerStateByte)
	}
	// If the top-most frame halted, drain the leftover execution gas rather
	// than returning it to the sender. The frame exit itself already burned
	// its gas left, but the refill above repays the execution gas the charge
	// originally borrowed, and on a halt that repayment must be burned as
	// well.
	if rules.IsAmsterdam && vmerr != nil && vmerr != vm.ErrExecutionReverted {
		st.gasRemaining.DrainExecution()
	}
	return ret, vmerr
}

// executeFrame runs an EIP-8141 frame transaction. It validates the transaction
// envelope and all protocol-validated signatures, then executes each frame as a
// top-level call, tracking the transaction-scoped approval context (payer and
// sender approval). Fees are settled against the payer after all frames run.
func (st *stateTransition) executeFrame(rules params.Rules) (*ExecutionResult, error) {
	fTx := st.msg.FrameTx
	msg := st.msg

	if err := validateFrameTx(fTx); err != nil {
		return nil, err
	}
	// Compute the canonical signature hash and validate every signature.
	sigHash := fTx.ComputeSigHash()
	for i := range fTx.Signatures {
		if !fTx.ValidateSignature(&fTx.Signatures[i], sigHash) {
			return nil, fmt.Errorf("%w: signature %d invalid", types.ErrInvalidSig, i)
		}
	}
	// EIP-1559 fee constraints (base fee and tip cap).
	if rules.IsLondon {
		if msg.GasFeeCap.Cmp(msg.GasTipCap) < 0 {
			return nil, fmt.Errorf("%w: address %v, maxPriorityFeePerGas: %s, maxFeePerGas: %s", ErrTipAboveFeeCap, msg.From.Hex(), msg.GasTipCap, msg.GasFeeCap)
		}
		if msg.GasFeeCap.CmpBig(st.evm.Context.BaseFee) < 0 {
			return nil, fmt.Errorf("%w: address %v, maxFeePerGas: %s, baseFee: %s", ErrFeeCapTooLow, msg.From.Hex(), msg.GasFeeCap, st.evm.Context.BaseFee)
		}
	}
	// Blob fee constraint.
	if st.blobGasUsed() > 0 {
		skipCheck := st.evm.Config.NoBaseFee && msg.BlobGasFeeCap.BitLen() == 0
		if !skipCheck && msg.BlobGasFeeCap.CmpBig(st.evm.Context.BlobBaseFee) < 0 {
			return nil, fmt.Errorf("%w: address %v blobGasFeeCap: %v, blobBaseFee: %v", ErrBlobFeeCapTooLow, msg.From.Hex(), msg.BlobGasFeeCap, st.evm.Context.BlobBaseFee)
		}
	}
	// Nonce must match the sender's state nonce.
	if !msg.SkipNonceChecks {
		if stNonce := st.state.GetNonce(msg.From); stNonce != msg.Nonce {
			if stNonce < msg.Nonce {
				return nil, fmt.Errorf("%w: address %v, tx: %d state: %d", ErrNonceTooHigh, msg.From.Hex(), msg.Nonce, stNonce)
			}
			return nil, fmt.Errorf("%w: address %v, tx: %d state: %d", ErrNonceTooLow, msg.From.Hex(), msg.Nonce, stNonce)
		}
	}

	standardGas, floorGas, maxGas, err := fTx.GasLimits()
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrFrameInvalid, err)
	}
	sumFrameGas, err := fTx.SumFrameGas()
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrFrameInvalid, err)
	}
	// The overhead is everything charged outside the frames' own gas limits:
	// the intrinsic cost, the per-frame cost, the calldata cost and the
	// signature verification cost.
	overhead := standardGas - sumFrameGas
	// EIP-7825/EIP-8037: the fixed costs alone must fit the execution-gas cap,
	// otherwise no amount of state-gas reservoir can cover them.
	if rules.IsAmsterdam && max(overhead, floorGas) > params.MaxTxGas {
		return nil, fmt.Errorf("%w (cap: %d, overhead: %d, floor: %d)", ErrGasLimitTooHigh, params.MaxTxGas, overhead, floorGas)
	}
	// Reserve the gas budget in the block gas pool. Frame transactions have to
	// use the same dimension as every other transaction in the block, or the
	// two accountings drift apart and the block can be overfilled.
	if rules.IsAmsterdam {
		err = st.gp.CheckGasAmsterdam(min(maxGas, params.MaxTxGas), maxGas)
	} else {
		err = st.gp.CheckGasLegacy(maxGas)
	}
	if err != nil {
		return nil, err
	}

	// Compute max_cost. EIP-8141 requires max_cost < 2**256; without the check
	// the wrapped value underflows the payer refund below and mints ether.
	maxCost, overflow := uint256.FromBig(msg.GasFeeCap.ToBig())
	if overflow {
		return nil, fmt.Errorf("%w: address %v, maxFeePerGas bit length: %d", ErrFeeCapVeryHigh, msg.From.Hex(), msg.GasFeeCap.BitLen())
	}
	if _, overflow = maxCost.MulOverflow(maxCost, uint256.NewInt(maxGas)); overflow {
		return nil, fmt.Errorf("%w: max cost exceeds 256 bits", ErrFrameInvalid)
	}
	blobFee := new(uint256.Int)
	if blobGas := st.blobGasUsed(); blobGas > 0 {
		blobBaseFee, blobOverflow := uint256.FromBig(st.evm.Context.BlobBaseFee)
		if blobOverflow {
			return nil, fmt.Errorf("%w: blob base fee exceeds 256 bits", ErrFrameInvalid)
		}
		if _, overflow = blobFee.MulOverflow(uint256.NewInt(blobGas), blobBaseFee); overflow {
			return nil, fmt.Errorf("%w: blob cost exceeds 256 bits", ErrFrameInvalid)
		}
		if _, overflow = maxCost.AddOverflow(maxCost, blobFee); overflow {
			return nil, fmt.Errorf("%w: max cost exceeds 256 bits", ErrFrameInvalid)
		}
	}

	st.state.Prepare(rules, msg.From, st.evm.Context.Coinbase, msg.To, vm.ActivePrecompiles(rules), nil)

	// The transaction's running gas budget covers the sum of the frame gas
	// limits, split into the execution balance and the EIP-8037 state-gas
	// reservoir exactly as initRuntimeGasBudget does for other transactions.
	frameGasLeft := sumFrameGas
	if rules.IsAmsterdam {
		frameGasLeft = min(params.MaxTxGas-overhead, sumFrameGas)
	}
	st.gasRemaining = vm.NewGasBudget(frameGasLeft, sumFrameGas-frameGasLeft)

	// Set up the frame context shared across all frames.
	fc := &vm.FrameContext{
		Tx:            fTx,
		SigHash:       sigHash,
		MaxGas:        maxGas,
		State:         &vm.FrameTxState{MaxCost: maxCost.ToBig()},
		FrameStatuses: make([]byte, len(fTx.Frames)),
	}
	st.evm.SetFrameContext(fc)
	// Clear the context on every exit, including the error paths: an invalid
	// frame transaction must not leave the EVM believing it is still inside one.
	defer func() {
		st.evm.SetFrameContext(nil)
		st.evm.Origin = msg.From
	}()

	frameReceipts := make([]*types.FrameReceipt, len(fTx.Frames))

	// Execute the frames, one atomic batch at a time. An atomic batch is the
	// maximal run [i, batchEnd] where frames i..batchEnd-1 carry the flag and
	// batchEnd does not; a lone frame is the degenerate batch [i, i].
	for i := 0; i < len(fTx.Frames); {
		batchEnd := i
		for batchEnd < len(fTx.Frames)-1 && fTx.Frames[batchEnd].Flags&types.FrameFlagAtomicBatch != 0 {
			batchEnd++
		}
		// Snapshot the state before the batch so a failure anywhere inside it
		// rolls the whole batch back.
		batchSnapshot := -1
		if batchEnd > i {
			batchSnapshot = st.state.Snapshot()
		}
		failed := -1
		for j := i; j <= batchEnd; j++ {
			status, err := st.executeFrameAt(fc, frameReceipts, j)
			if err != nil {
				return nil, err
			}
			if status == types.ReceiptStatusFailed {
				failed = j
				break
			}
		}
		if failed >= 0 && batchEnd > i {
			// Roll the batch back. Frames that ran keep their status and gas
			// used but lose their logs; frames after the failure never ran and
			// are reported as skipped, their gas allotment left unspent.
			st.state.RevertToSnapshot(batchSnapshot)
			for j := i; j <= failed; j++ {
				frameReceipts[j].Logs = nil
			}
			for j := failed + 1; j <= batchEnd; j++ {
				fc.FrameStatuses[j] = byte(types.ReceiptStatusSkipped)
				frameReceipts[j] = &types.FrameReceipt{Status: types.ReceiptStatusSkipped}
			}
		}
		i = batchEnd + 1
	}

	// The payer must have been set via an APPROVE call.
	if !fc.State.PayerSet {
		return nil, fmt.Errorf("%w: no payer approved", ErrFrameInvalid)
	}

	// Final gas accounting (EIP-8141).
	if st.gasRemaining.UsedStateGas < 0 {
		return nil, fmt.Errorf("%w: negative state gas usage %d", ErrFrameInvalid, st.gasRemaining.UsedStateGas)
	}
	txStateGas := uint64(st.gasRemaining.UsedStateGas)
	gasLeft := st.gasRemaining.ExecutionGas + st.gasRemaining.StateGas
	gasUsedBeforeRefund := standardGas - gasLeft
	if gasUsedBeforeRefund < txStateGas {
		return nil, fmt.Errorf("%w: state gas %d exceeds total %d", ErrFrameInvalid, txStateGas, gasUsedBeforeRefund)
	}
	txExecutionGas := max(gasUsedBeforeRefund-txStateGas, floorGas)
	gasUsed := max(gasUsedBeforeRefund-st.calcRefund(gasUsedBeforeRefund), floorGas)

	// Charge the block gas pool in the same dimension it was reserved.
	if rules.IsAmsterdam {
		err = st.gp.ChargeGasAmsterdam(txExecutionGas, txStateGas, gasUsed)
	} else {
		err = st.gp.ChargeGasLegacy(maxGas-gasUsed, gasUsed)
	}
	if err != nil {
		return nil, err
	}

	// Settle the fee: the payer was debited max_cost when it approved payment,
	// so refund the difference against the final charged fee. The base-fee share
	// stays destroyed and the tip goes to the coinbase.
	chargedFee := new(uint256.Int).Mul(uint256.NewInt(gasUsed), msg.GasPrice)
	chargedFee.Add(chargedFee, blobFee)
	payerRefund, underflow := new(uint256.Int).SubOverflow(maxCost, chargedFee)
	if underflow {
		// Unreachable: gasUsed <= maxGas and GasPrice <= GasFeeCap, so the
		// charged fee cannot exceed the collected max cost. Fail loudly rather
		// than credit the payer a wrapped balance.
		return nil, fmt.Errorf("%w: charged fee %v exceeds max cost %v", ErrFrameInvalid, chargedFee, maxCost)
	}
	if !payerRefund.IsZero() {
		st.state.AddBalance(fc.State.Payer, payerRefund, tracing.BalanceIncreaseGasReturn)
	}
	// Pay the tip to the coinbase, skipping the eth_call case where fees are off.
	if !(st.evm.Config.NoBaseFee && msg.GasFeeCap.Sign() == 0 && msg.GasTipCap.Sign() == 0) {
		tipPerGas := msg.GasPrice
		if rules.IsLondon {
			baseFee, _ := uint256.FromBig(st.evm.Context.BaseFee)
			tipPerGas = new(uint256.Int).Sub(msg.GasPrice, baseFee)
		}
		tipFee := new(uint256.Int).Mul(uint256.NewInt(gasUsed), tipPerGas)
		st.state.AddBalance(st.evm.Context.Coinbase, tipFee, tracing.BalanceIncreaseRewardTransactionFee)
	}

	return &ExecutionResult{
		UsedGas:       gasUsed,
		MaxUsedGas:    gasUsedBeforeRefund,
		Err:           nil,
		ReturnData:    nil,
		Payer:         fc.State.Payer,
		FrameReceipts: frameReceipts,
	}, nil
}

// executeFrameAt runs the frame at index i as a top-level call and records its
// frame receipt. It returns the frame's receipt status, or an error when the
// frame makes the whole transaction invalid (a SENDER frame before approval, or
// a reverting VERIFY frame).
func (st *stateTransition) executeFrameAt(fc *vm.FrameContext, frameReceipts []*types.FrameReceipt, i int) (uint64, error) {
	fTx := fc.Tx
	frame := &fTx.Frames[i]

	// Update the per-frame context.
	fc.FrameIndex = i
	fc.Frame = frame
	resolvedTarget := fTx.Sender
	if frame.Target != nil {
		resolvedTarget = *frame.Target
	}
	fc.ResolvedTarget = resolvedTarget

	// Determine the frame's caller.
	caller := params.FrameEntryPointAddress
	if frame.Mode == types.FrameModeSender {
		if !fc.State.SenderApproved {
			return 0, fmt.Errorf("%w: SENDER frame %d before approval", ErrFrameInvalid, i)
		}
		caller = fTx.Sender
	}
	// ORIGIN returns the frame's caller throughout all call depths.
	st.evm.Origin = caller

	// Transient storage does not carry across frames.
	st.state.ClearTransientStorage()

	logStart := st.frameLogsLen()
	snapshot := st.state.Snapshot()

	// Forward the frame's gas budget.
	child := st.gasRemaining.Forward(min(frame.GasLimit, st.gasRemaining.ExecutionGas))
	childStart := child

	// Charge the target's account access from the frame's own budget. A
	// top-level evm.Call does not do this for us, and leaving it free would let
	// a transaction touch MAX_FRAMES cold accounts for nothing. The Frame fork
	// implies Amsterdam, so the Amsterdam prices apply.
	accessCost := vm.GasCosts{ExecutionGas: params.ColdAccountAccessAmsterdam}
	if st.state.AddressInAccessList(resolvedTarget) {
		accessCost.ExecutionGas = params.WarmAccountAccessAmsterdam
	}
	var vmerr error
	if _, ok := child.Charge(accessCost); !ok {
		child, vmerr = child.ExitHalt(), vm.ErrOutOfGas
	} else {
		st.state.AddAddressToAccessList(resolvedTarget)

		prevReadOnly := st.evm.ReadOnly()
		if frame.Mode == types.FrameModeVerify {
			st.evm.SetReadOnly(true)
		}
		_, child, vmerr = st.evm.Call(caller, resolvedTarget, frame.Data, child, frame.Value256())
		st.evm.SetReadOnly(prevReadOnly)

		// EIP-8141 default code: an account with no code of its own still
		// validates frame transactions through the protocol-defined behaviour.
		// In DEFAULT and SENDER mode that is a plain empty-code call, which is
		// what evm.Call just did; only VERIFY mode has extra semantics.
		if vmerr == nil && frame.Mode == types.FrameModeVerify && st.frameTargetHasNoCode(resolvedTarget) {
			vmerr = st.runDefaultVerifyCode(fc, resolvedTarget)
		}
	}
	st.gasRemaining.Absorb(child)

	gasUsed := child.Used(childStart)
	status := uint64(types.ReceiptStatusSuccessful)
	if vmerr != nil {
		status = types.ReceiptStatusFailed
		st.state.RevertToSnapshot(snapshot)
		// A reverting VERIFY frame makes the whole transaction invalid.
		if frame.Mode == types.FrameModeVerify {
			return 0, fmt.Errorf("%w: VERIFY frame %d failed: %w", ErrFrameInvalid, i, vmerr)
		}
	}
	fc.FrameStatuses[i] = byte(status)
	frameReceipts[i] = &types.FrameReceipt{Status: status, GasUsed: gasUsed, Logs: st.frameLogsSlice(logStart)}
	return status, nil
}

// frameTargetHasNoCode reports whether the resolved target of a frame has no
// code of its own, which selects the EIP-8141 default code behaviour. A
// non-existent account and an account with the empty code hash both qualify; an
// EIP-7702 delegation indicator does not, since it resolves to real code.
func (st *stateTransition) frameTargetHasNoCode(target common.Address) bool {
	codeHash := st.state.GetCodeHash(target)
	return codeHash == types.EmptyCodeHash || codeHash == (common.Hash{})
}

// runDefaultVerifyCode implements the EIP-8141 default code behaviour for a
// VERIFY frame whose target carries no code: it approves the scope named by the
// frame's flags, provided the transaction carries a matching secp256k1
// signature over the canonical signature hash at the expected index.
func (st *stateTransition) runDefaultVerifyCode(fc *vm.FrameContext, resolvedTarget common.Address) error {
	allowedScope := fc.Frame.Flags & types.FrameFlagApproveExecutionPayment
	if allowedScope == 0 {
		return vm.ErrExecutionReverted
	}
	// The sender's own approval is signature 0; a sponsor's is signature 1.
	sigIndex := 1
	if allowedScope&types.FrameFlagApproveExecution != 0 {
		sigIndex = 0
	}
	if sigIndex >= len(fc.Tx.Signatures) {
		return vm.ErrExecutionReverted
	}
	sig := &fc.Tx.Signatures[sigIndex]
	if sig.Scheme != types.SignatureSchemeSecp256k1 || len(sig.Msg) != 0 {
		return vm.ErrExecutionReverted
	}
	if signer, ok := sig.ResolvedSigner(fc.Tx.Sender); !ok || signer != resolvedTarget {
		return vm.ErrExecutionReverted
	}
	return fc.Approve(st.evm, uint64(allowedScope))
}

// frameLogsLen returns the number of logs the current transaction has emitted
// so far, or -1 if the underlying state does not expose them.
func (st *stateTransition) frameLogsLen() int {
	if ls, ok := st.state.(logProvider); ok {
		return len(ls.TxLogs())
	}
	return -1
}

// frameLogsSlice returns the logs emitted since the given index, or nil if the
// underlying state does not expose the log list.
func (st *stateTransition) frameLogsSlice(from int) []*types.Log {
	if from < 0 {
		return nil
	}
	if ls, ok := st.state.(logProvider); ok {
		if logs := ls.TxLogs(); from < len(logs) {
			return logs[from:]
		}
	}
	return nil
}

// logProvider is satisfied by *state.StateDB to allow frame execution to split
// logs by frame.
type logProvider interface {
	TxLogs() []*types.Log
}

// traceHaltedTopFrame calls the Enter and Exit functions on the tracer,
// in order to produce correct tracing results if the EVM exits early (after Amsterdam).
// Tracers assume every transaction producing a receipt also produces a depth-zero frame.
func (st *stateTransition) traceHaltedTopFrame(typ vm.OpCode, to common.Address, input []byte, entryGas vm.GasBudget, endGas vm.GasBudget, value *uint256.Int) {
	tracer := st.evm.Config.Tracer
	if tracer == nil {
		return
	}
	if tracer.OnEnter != nil {
		tracer.OnEnter(0, byte(typ), st.msg.From, to, input, entryGas.ExecutionGas, value.ToBig())
	}
	if tracer.HasGasHook() {
		tracer.EmitGasChange(tracing.Gas{}, entryGas.AsTracing(), tracing.GasChangeCallInitialBalance)
		tracer.EmitGasChange(entryGas.AsTracing(), endGas.AsTracing(), tracing.GasChangeCallFailedExecution)
	}
	if tracer.OnExit != nil {
		tracer.OnExit(0, nil, entryGas.ExecutionGas, vm.VMErrorFromErr(vm.ErrOutOfGas), true)
	}
}

// chargeRuntimeGas deducts an EIP-2780 runtime charge from the transaction's
// gas budget and reports whether the budget covered it.
func (st *stateTransition) chargeRuntimeGas(cost vm.GasCosts) bool {
	prior, ok := st.gasRemaining.Charge(cost)
	if !ok {
		return false
	}
	if st.evm.Config.Tracer.HasGasHook() {
		st.evm.Config.Tracer.EmitGasChange(prior.AsTracing(), st.gasRemaining.AsTracing(), tracing.GasChangeTxRuntimeGas)
	}
	return true
}

// chargeCallRecipientEIP2780 applies the EIP-2780 runtime charges for the
// top-level recipient of a message-call transaction, before the first frame is
// entered:
//
//   - if the recipient is EIP-161 empty and the transaction carries value,
//     the durable state growth of the new account;
//
//   - if the recipient is an EIP-7702 delegated account, resolving the
//     delegation loads the target's code: a cold account access, or a warm
//     access if the target is already warm.
//
// Each charge is deducted before the state access it prices is performed:
// under EIP-7928 every account load is recorded in the block access list, so
// an access the budget cannot cover must not happen at all.
func (st *stateTransition) chargeCallRecipientEIP2780(value *uint256.Int) bool {
	to := *st.msg.To

	// This runs in the topmost frame before any bytecode executes, non-existence
	// is equivalent with EIP-161-empty, as no preceding operation can leave a
	// transient EIP-161-empty account (such as zero-value transfer).
	if !value.IsZero() && st.state.Empty(to) {
		if !st.chargeRuntimeGas(vm.GasCosts{StateGas: params.AccountCreationSize * st.evm.Context.CostPerStateByte}) {
			return false
		}
	}
	if target, delegated := types.ParseDelegation(st.state.GetCode(to)); delegated {
		// Pay the delegation-target access before the target is warmed and
		// its code resolved (loaded).
		cost := vm.GasCosts{ExecutionGas: params.ColdAccountAccessAmsterdam}
		if st.state.AddressInAccessList(target) {
			cost.ExecutionGas = params.WarmAccountAccessAmsterdam
		}
		if !st.chargeRuntimeGas(cost) {
			return false
		}
		st.state.AddAddressToAccessList(target)

		// Record the delegation in the block level accessList explicitly
		st.state.GetCode(target)
	}
	return true
}

// settleGas finalizes the per-tx gas accounting after EVM execution:
//
//   - Snapshots the EIP-8037 block-level 2D figures (tx_execution_gas,
//     tx_state_gas) before any refund.
//   - Computes the receipt scalar tx_gas_used by applying the EIP-3529
//     refund and the EIP-7623 calldata floor.
//   - Charges the block gas pool (2D under Amsterdam, scalar pre-Amsterdam).
//   - Refunds the leftover gas to the sender as ETH.
func (st *stateTransition) settleGas(rules params.Rules, floorDataGas uint64) (gasUsed, peakUsed uint64, err error) {
	if st.gasRemaining.UsedStateGas < 0 {
		return 0, 0, fmt.Errorf("negative topmost frame state gas usage, %d", st.gasRemaining.UsedStateGas)
	}
	txStateGas := uint64(st.gasRemaining.UsedStateGas)

	// EIP-8037:
	// tx_gas_used_before_refund = tx.gas - tx_output.gas_left - tx_output.state_gas_reservoir
	// tx_state_gas = tx_output.execution_state_gas_used
	// tx_execution_gas = max(tx_gas_used_before_refund - tx_state_gas, calldata_floor_gas_cost)
	gasLeft := st.gasRemaining.ExecutionGas + st.gasRemaining.StateGas
	gasUsedBeforeRefund := st.msg.GasLimit - gasLeft

	if gasUsedBeforeRefund < txStateGas {
		return 0, 0, fmt.Errorf("negative topmost frame execution gas usage, total: %d, state: %d", gasUsedBeforeRefund, txStateGas)
	}
	txExecutionGas := max(gasUsedBeforeRefund-txStateGas, floorDataGas)

	// EIP-3529: tx_gas_refund = min(tx_gas_used_before_refund/5, refund_counter).
	refund := st.calcRefund(gasUsedBeforeRefund)
	if st.evm.Config.Tracer.HasGasHook() {
		st.evm.Config.Tracer.EmitGasChange(tracing.Gas{Execution: gasLeft}, tracing.Gas{Execution: gasLeft + refund}, tracing.GasChangeTxRefunds)
	}
	gasLeft += refund
	gasUsed = gasUsedBeforeRefund - refund

	// EIP-7623: tx_gas_used = max(tx_gas_used_after_refund, calldata_floor).
	peakUsed = gasUsedBeforeRefund
	if rules.IsPrague && gasUsed < floorDataGas {
		diff := floorDataGas - gasUsed
		if st.evm.Config.Tracer.HasGasHook() {
			st.evm.Config.Tracer.EmitGasChange(tracing.Gas{Execution: gasLeft}, tracing.Gas{Execution: gasLeft - diff}, tracing.GasChangeTxDataFloor)
		}
		gasLeft -= diff
		gasUsed = floorDataGas
		peakUsed = max(peakUsed, floorDataGas)
	}

	// Settle down the final gas consumption in the block-level pool
	if rules.IsAmsterdam {
		if err = st.gp.ChargeGasAmsterdam(txExecutionGas, txStateGas, gasUsed); err != nil {
			return 0, 0, err
		}
	} else {
		if err = st.gp.ChargeGasLegacy(gasLeft, gasUsed); err != nil {
			return 0, 0, err
		}
	}

	// Refund leftover gas to the sender
	if gasLeft > 0 {
		refund := new(uint256.Int).Mul(uint256.NewInt(gasLeft), st.msg.GasPrice)
		st.state.AddBalance(st.msg.From, refund, tracing.BalanceIncreaseGasReturn)

		if st.evm.Config.Tracer.HasGasHook() {
			st.evm.Config.Tracer.EmitGasChange(tracing.Gas{Execution: gasLeft}, tracing.Gas{}, tracing.GasChangeTxLeftOverReturned)
		}
	}
	return gasUsed, peakUsed, nil
}

// validateAuthorization validates an EIP-7702 authorization against the state.
func (st *stateTransition) validateAuthorization(auth *types.SetCodeAuthorization) (authority common.Address, err error) {
	// Verify chain ID is null or equal to current chain ID.
	if !auth.ChainID.IsZero() && auth.ChainID.CmpBig(st.evm.ChainConfig().ChainID) != 0 {
		return authority, ErrAuthorizationWrongChainID
	}
	// Limit nonce to 2^64-1 per EIP-2681.
	if auth.Nonce+1 < auth.Nonce {
		return authority, ErrAuthorizationNonceOverflow
	}
	// Validate signature values and recover authority.
	authority, err = auth.Authority()
	if err != nil {
		return authority, fmt.Errorf("%w: %v", ErrAuthorizationInvalidSignature, err)
	}
	// Check the authority account
	//  1) doesn't have code or has existing delegation
	//  2) matches the auth's nonce
	//
	// Note it is added to the access list even if the authorization is invalid.
	st.state.AddAddressToAccessList(authority)
	code := st.state.GetCode(authority)
	if _, ok := types.ParseDelegation(code); len(code) != 0 && !ok {
		return authority, ErrAuthorizationDestinationHasCode
	}
	if have := st.state.GetNonce(authority); have != auth.Nonce {
		return authority, ErrAuthorizationNonceMismatch
	}
	return authority, nil
}

// authTracking tracks the charges already paid for an authority by earlier
// authorizations in the same transaction.
type authTracking struct {
	written         bool // first-write ACCOUNT_WRITE surcharge paid
	authBaseCovered bool // indicator exists at tx start, or paid earlier
}

// applyAuthorization applies an EIP-7702 code delegation to the state.
func (st *stateTransition) applyAuthorization(rules params.Rules, auth *types.SetCodeAuthorization, authorities map[common.Address]*authTracking) error {
	authority, err := st.validateAuthorization(auth)
	if err != nil {
		return err
	}
	oldDelegation, curDelegated := types.ParseDelegation(st.state.GetCode(authority))

	if !rules.IsAmsterdam {
		if st.state.Exist(authority) {
			st.state.AddRefund(params.CallNewAccountGas - params.TxAuthTupleGas)
		}
	} else {
		// EIP-2780: charge the state-dependent authorization costs at runtime.
		// The authority's cold access was already charged unconditionally at the
		// intrinsic phase, so only state-dependent costs remain here.
		var cost vm.GasCosts

		track := authorities[authority]
		if track == nil {
			track = &authTracking{authBaseCovered: curDelegated}
			authorities[authority] = track
		}
		// Every valid authorization writes the authority account: the
		// nonce bump, and possibly the delegation indicator. The first
		// write to an account within the transaction carries the
		// first-write surcharge. At this point the accounts whose write
		// has already been paid for are:
		//
		//   - the sender: TX_BASE_COST prices its account write, and the
		//     gas prepayment and nonce bump have already happened;
		//
		//   - authorities written by preceding valid authorizations in
		//     this list, which carried the surcharge themselves;
		//
		//   - tx.to, but only when the transaction carries value:
		//     TX_VALUE_COST prepaid the recipient write at the intrinsic
		//     phase. A zero-value transaction pays no TX_VALUE_COST, so a
		//     write to tx.to here is still the first paid write.
		hasValue := st.msg.Value != nil && !st.msg.Value.IsZero()
		if !track.written && authority != st.msg.From && (authority != st.to() || !hasValue) {
			cost.ExecutionGas += params.AccountWriteAmsterdam
			track.written = true
		}
		// Durable state growth of the new account
		if st.state.Empty(authority) {
			cost.StateGas += params.AccountCreationSize * st.evm.Context.CostPerStateByte
		}
		// Charge the net-new indicator bytes at most once per authority;
		// clearing within the same transaction refunds nothing.
		if auth.Address != (common.Address{}) && !track.authBaseCovered {
			cost.StateGas += params.AuthorizationCreationSize * st.evm.Context.CostPerStateByte
			track.authBaseCovered = true
		}
		if !st.chargeRuntimeGas(cost) {
			return ErrOutOfGasRuntime
		}
	}
	// Update nonce and account code.
	st.state.SetNonce(authority, auth.Nonce+1, tracing.NonceChangeAuthorization)

	// Delegation to zero address means clear.
	if auth.Address == (common.Address{}) {
		if curDelegated {
			st.state.SetCode(authority, nil, tracing.CodeChangeAuthorizationClear)
		}
		return nil
	}
	// Install delegation to auth.Address if the delegation changed
	if !curDelegated || auth.Address != oldDelegation {
		st.state.SetCode(authority, types.AddressToDelegation(auth.Address), tracing.CodeChangeAuthorization)
	}
	return nil
}

// applyAuthorizations applies the EIP-7702 code delegations to the state.
// It reports whether the transaction budget covered all runtime authorization
// charges.
func (st *stateTransition) applyAuthorizations(rules params.Rules, auths []types.SetCodeAuthorization) bool {
	authorities := make(map[common.Address]*authTracking)
	for _, auth := range auths {
		if err := st.applyAuthorization(rules, &auth, authorities); err == ErrOutOfGasRuntime {
			return false
		}
	}
	return true
}

// calcRefund computes the EIP-3529 refund cap against tx_gas_used_before_refund.
func (st *stateTransition) calcRefund(gasUsedBeforeRefund uint64) uint64 {
	quotient := params.RefundQuotient
	if st.evm.ChainConfig().IsLondon(st.evm.Context.BlockNumber) {
		quotient = params.RefundQuotientEIP3529
	}
	refund := gasUsedBeforeRefund / quotient
	if refund > st.state.GetRefund() {
		refund = st.state.GetRefund()
	}
	return refund
}

// blobGasUsed returns the amount of blob gas used by the message.
func (st *stateTransition) blobGasUsed() uint64 {
	return uint64(len(st.msg.BlobHashes) * params.BlobTxBlobGasPerBlob)
}
