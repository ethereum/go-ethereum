// Copyright 2023 The go-ethereum Authors
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

package txpool

import (
	"bytes"
	"errors"
	"fmt"
	"math/big"
	"slices"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/tracing"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/crypto/kzg4844"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"
	"github.com/holiman/uint256"
)

var (
	// blobTxMinBlobGasPrice is the big.Int version of the configured protocol
	// parameter to avoid constructing a new big integer for every transaction.
	blobTxMinBlobGasPrice = big.NewInt(params.BlobTxMinBlobGasprice)
)

const (
	frameTxMaxVerifyGas      = 100_000
	frameTxMaxVerifyStateGas = 500_000
)

// ValidationOptions define certain differences between transaction validation
// across the different pools without having to duplicate those checks.
type ValidationOptions struct {
	Config *params.ChainConfig // Chain configuration to selectively validate based on current fork rules

	Accept       uint8    // Bitmap of transaction types that should be accepted for the calling pool
	MaxSize      uint64   // Maximum size of a transaction that the caller can meaningfully handle
	MaxBlobCount int      // Maximum number of blobs allowed per transaction
	MinTip       *big.Int // Minimum gas tip needed to allow a transaction into the caller pool
}

// ValidationFunction is an method type which the pools use to perform the tx-validations which do not
// require state access. Production code typically uses ValidateTransaction, whereas testing-code
// might choose to instead use something else, e.g. to always fail or avoid heavy cpu usage.
type ValidationFunction func(tx *types.Transaction, head *types.Header, signer types.Signer, opts *ValidationOptions) error

// ValidateTransaction is a helper method to check whether a transaction is valid
// according to the consensus rules, but does not check state-dependent validation
// (balance, nonce, etc).
//
// This check is public to allow different transaction pools to check the basic
// rules without duplicating code and running the risk of missed updates.
func ValidateTransaction(tx *types.Transaction, head *types.Header, signer types.Signer, opts *ValidationOptions) error {
	// Ensure transactions not implemented by the calling pool are rejected
	if opts.Accept&(1<<tx.Type()) == 0 {
		return fmt.Errorf("%w: tx type %v not supported by this pool", core.ErrTxTypeNotSupported, tx.Type())
	}
	// Before performing any expensive validations, sanity check that the tx is
	// smaller than the maximum limit the pool can meaningfully handle
	if tx.Size() > opts.MaxSize {
		return fmt.Errorf("%w: transaction size %v, limit %v", ErrOversizedData, tx.Size(), opts.MaxSize)
	}
	// Ensure only transactions that have been enabled are accepted
	rules := opts.Config.Rules(head.Number, head.Difficulty.Sign() == 0, head.Time)
	if !rules.IsBerlin && tx.Type() != types.LegacyTxType {
		return fmt.Errorf("%w: type %d rejected, pool not yet in Berlin", core.ErrTxTypeNotSupported, tx.Type())
	}
	if !rules.IsLondon && tx.Type() == types.DynamicFeeTxType {
		return fmt.Errorf("%w: type %d rejected, pool not yet in London", core.ErrTxTypeNotSupported, tx.Type())
	}
	if !rules.IsCancun && tx.Type() == types.BlobTxType {
		return fmt.Errorf("%w: type %d rejected, pool not yet in Cancun", core.ErrTxTypeNotSupported, tx.Type())
	}
	if !rules.IsPrague && tx.Type() == types.SetCodeTxType {
		return fmt.Errorf("%w: type %d rejected, pool not yet in Prague", core.ErrTxTypeNotSupported, tx.Type())
	}
	if !rules.IsBogota && tx.Type() == types.FrameTxType {
		return fmt.Errorf("%w: type %d rejected, pool not yet in Bogota", core.ErrTxTypeNotSupported, tx.Type())
	}
	// Check whether the init code size has been exceeded
	if tx.To() == nil {
		if err := vm.CheckMaxInitCodeSize(&rules, uint64(len(tx.Data()))); err != nil {
			return err
		}
	}
	if rules.IsOsaka && !rules.IsAmsterdam && tx.Gas() > params.MaxTxGas {
		return fmt.Errorf("%w (cap: %d, tx: %d)", core.ErrGasLimitTooHigh, params.MaxTxGas, tx.Gas())
	}
	// Transactions can't be negative. This may never happen using RLP decoded
	// transactions but may occur for transactions created using the RPC.
	if tx.Value().Sign() < 0 {
		return ErrNegativeValue
	}
	// Ensure the transaction doesn't exceed the current block limit gas
	if head.GasLimit < tx.Gas() {
		return ErrGasLimit
	}
	// Sanity check for extremely large numbers (supported by RLP or RPC)
	if tx.GasFeeCap().BitLen() > 256 {
		return core.ErrFeeCapVeryHigh
	}
	if tx.GasTipCap().BitLen() > 256 {
		return core.ErrTipVeryHigh
	}
	// Ensure gasFeeCap is greater than or equal to gasTipCap
	if tx.GasFeeCapIntCmp(tx.GasTipCap()) < 0 {
		return core.ErrTipAboveFeeCap
	}
	// Make sure the transaction is signed properly
	from, err := types.Sender(signer, tx)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrInvalidSender, err)
	}
	// Limit nonce to 2^64-1 per EIP-2681
	if tx.Nonce()+1 < tx.Nonce() {
		return core.ErrNonceMax
	}
	// Sanity check for extremely large numbers (supported by RLP or RPC)
	value, overflow := uint256.FromBig(tx.Value())
	if overflow {
		return core.ErrInsufficientFunds
	}
	// Ensure the transaction has more gas than the bare minimum needed to cover
	// the transaction metadata
	var intrGas uint64
	if tx.Type() == types.FrameTxType {
		intrGas, err = core.FrameTxIntrinsicGas(tx.Frames(), tx.FrameSignatures(), from)
	} else {
		intrGas, err = core.IntrinsicGas(tx.Data(), tx.AccessList(), tx.SetCodeAuthorizations(), from, tx.To(), value, rules)
	}
	if err != nil {
		return err
	}
	if tx.Gas() < intrGas {
		return fmt.Errorf("%w: gas %v, minimum needed %v", core.ErrIntrinsicGas, tx.Gas(), intrGas)
	}
	// Ensure the transaction can cover floor data gas.
	if rules.IsPrague {
		floorDataGas, err := core.FloorDataGas(rules, from, tx.To(), value, tx.Data(), tx.AccessList())
		if err != nil {
			return err
		}
		// Make sure the transaction has sufficient gas allowance to
		// pay the floor cost.
		if tx.Gas() < floorDataGas {
			return fmt.Errorf("%w: gas %v, minimum needed %v", core.ErrFloorDataGas, tx.Gas(), floorDataGas)
		}
		// In Amsterdam, the transaction gas limit is allowed to exceed
		// params.MaxTxGas, but the calldata floor cost is capped by it.
		if rules.IsAmsterdam && max(intrGas, floorDataGas) > params.MaxTxGas {
			return fmt.Errorf("%w: intrinsic cost %v, floor: %v", core.ErrFloorDataGas, intrGas, floorDataGas)
		}
	}
	// Ensure the gasprice is high enough to cover the requirement of the calling pool
	if tx.GasTipCapIntCmp(opts.MinTip) < 0 {
		return fmt.Errorf("%w: gas tip cap %v, minimum needed %v", ErrTxGasPriceTooLow, tx.GasTipCap(), opts.MinTip)
	}
	if tx.Type() == types.BlobTxType {
		return validateBlobSidecar(tx, head, opts)
	}
	if tx.Type() == types.SetCodeTxType {
		if len(tx.SetCodeAuthorizations()) == 0 {
			return errors.New("set code tx must have at least one authorization tuple")
		}
	}
	return nil
}

// validateBlobSidecar implements the blob sidecar validation.
// Note that this doesn't verify the consistency between blobs(cells) and
// proofs. For proof verification, use validateCells.
func validateBlobSidecar(tx *types.Transaction, head *types.Header, opts *ValidationOptions) error {
	if tx.BlobGasFeeCapIntCmp(blobTxMinBlobGasPrice) < 0 {
		return fmt.Errorf("%w: blob fee cap %v, minimum needed %v", ErrTxGasPriceTooLow, tx.BlobGasFeeCap(), blobTxMinBlobGasPrice)
	}
	sidecar := tx.BlobTxSidecar()
	if sidecar == nil {
		return errors.New("missing sidecar in blob transaction")
	}
	hashes := tx.BlobHashes()
	if err := sidecar.ValidateBlobCommitmentHashes(hashes); err != nil {
		return err
	}
	if len(hashes) > opts.MaxBlobCount {
		return fmt.Errorf("%w: blob count %v, limit %v", ErrTxBlobLimitExceeded, len(hashes), opts.MaxBlobCount)
	}
	if sidecar.Version != types.BlobSidecarVersion1 {
		return fmt.Errorf("%w: unexpected sidecar version, want: %d, got: %d", ErrSidecarFormatError, types.BlobSidecarVersion1, sidecar.Version)
	}
	if len(sidecar.Proofs) != len(sidecar.Commitments)*kzg4844.CellProofsPerBlob {
		return fmt.Errorf("%w: invalid number of %d blob proofs expected %d", ErrSidecarFormatError, len(sidecar.Proofs), len(sidecar.Commitments)*kzg4844.CellProofsPerBlob)
	}
	return nil
}

func ValidateCells(sidecar *types.BlobTxCellSidecar) error {
	// Two checks here (custody count check and blobCount check) is duplicated in buffer.go
	// However it is required to 1) serve eth71 peer and direct submission 2) catch any bug in
	// merging cell delivery.
	if sidecar.Custody.OneCount() == 0 {
		return errors.New("blobless blob transaction")
	}
	// Verify whether the blob count is consistent with other parts of the sidecar and the transaction
	blobCount := len(sidecar.Cells) / sidecar.Custody.OneCount()
	if blobCount == 0 {
		return errors.New("blobless blob transaction")
	}
	if blobCount != len(sidecar.Commitments) {
		return fmt.Errorf("invalid number of %d blobs compared to %d commitments", blobCount, len(sidecar.Commitments))
	}
	if len(sidecar.Proofs) != len(sidecar.Commitments)*kzg4844.CellProofsPerBlob {
		return fmt.Errorf("invalid number of %d proofs compared to %d commitments", len(sidecar.Proofs), len(sidecar.Commitments))
	}
	if sidecar.Version != types.BlobSidecarVersion1 {
		return fmt.Errorf("unexpected sidecar version, want: %d, got: %d", types.BlobSidecarVersion1, sidecar.Version)
	}
	return validateCellsOsaka(sidecar)
}

func validateCellsOsaka(sidecar *types.BlobTxCellSidecar) error {
	indices := sidecar.Custody.Indices()
	cellProofs := make([]kzg4844.Proof, 0)
	for blobIdx := range len(sidecar.Commitments) {
		for _, proofIdx := range indices {
			idx := blobIdx*kzg4844.CellProofsPerBlob + int(proofIdx)
			cellProofs = append(cellProofs, sidecar.Proofs[idx])
		}
	}
	if err := kzg4844.VerifyCells(sidecar.Cells, sidecar.Commitments, cellProofs, sidecar.Custody.Indices()); err != nil {
		return fmt.Errorf("%w: %v", ErrKZGVerificationError, err)
	}
	return nil
}

// ValidationOptionsWithState define certain differences between stateful transaction
// validation across the different pools without having to duplicate those checks.
type ValidationOptionsWithState struct {
	State *state.StateDB // State database to check nonces and balances against

	Config *params.ChainConfig
	Head   *types.Header

	// FirstNonceGap is an optional callback to retrieve the first nonce gap in
	// the list of pooled transactions of a specific account. If this method is
	// set, nonce gaps will be checked and forbidden. If this method is not set,
	// nonce gaps will be ignored and permitted.
	FirstNonceGap func(addr common.Address) uint64

	// UsedAndLeftSlots is an optional callback to retrieve the number of tx slots
	// used and the number still permitted for an account. New transactions will
	// be rejected once the number of remaining slots reaches zero.
	UsedAndLeftSlots func(addr common.Address) (int, int)

	// ExistingExpenditure is a mandatory callback to retrieve the cumulative
	// cost of the already pooled transactions to check for overdrafts.
	ExistingExpenditure func(addr common.Address) *big.Int

	// ExistingCost is a mandatory callback to retrieve an already pooled
	// transaction's cost with the given nonce to check for overdrafts.
	ExistingCost func(addr common.Address, nonce uint64) *big.Int

	FrameTxPayment func(tx *types.Transaction, payer common.Address, cost *uint256.Int, paymaster bool, reads FrameTxReads) error
}

type FrameTxRead struct {
	Nonce    uint64
	Balance  *uint256.Int
	CodeHash common.Hash
	Storage  map[common.Hash]common.Hash
}

type FrameTxReads map[common.Address]*FrameTxRead

func (reads FrameTxReads) Changed(statedb *state.StateDB) bool {
	for addr, read := range reads {
		if statedb.GetNonce(addr) != read.Nonce || statedb.GetBalance(addr).Cmp(read.Balance) != 0 || statedb.GetCodeHash(addr) != read.CodeHash {
			return true
		}
		for slot, value := range read.Storage {
			if statedb.GetState(addr, slot) != value {
				return true
			}
		}
	}
	return false
}

// ValidateTransactionWithState is a helper method to check whether a transaction
// is valid according to the pool's internal state checks (balance, nonce, gaps).
//
// This check is public to allow different transaction pools to check the stateful
// rules without duplicating code and running the risk of missed updates.
func ValidateTransactionWithState(tx *types.Transaction, signer types.Signer, opts *ValidationOptionsWithState) error {
	// Ensure the transaction adheres to nonce ordering
	from, err := types.Sender(signer, tx) // already validated (and cached), but cleaner to check
	if err != nil {
		log.Error("Transaction sender recovery failed", "err", err)
		return err
	}
	next := opts.State.GetNonce(from)
	if next > tx.Nonce() {
		return fmt.Errorf("%w: next nonce %v, tx nonce %v", core.ErrNonceTooLow, next, tx.Nonce())
	}
	// Ensure the transaction doesn't produce a nonce gap in pools that do not
	// support arbitrary orderings
	if opts.FirstNonceGap != nil {
		if gap := opts.FirstNonceGap(from); gap < tx.Nonce() {
			return fmt.Errorf("%w: tx nonce %v, gapped nonce %v", core.ErrNonceTooHigh, tx.Nonce(), gap)
		}
	}
	if tx.Type() == types.FrameTxType {
		return validateFrameTxPrefix(tx, signer, opts)
	}
	// Ensure the transactor has enough funds to cover the transaction costs
	var (
		balance = opts.State.GetBalance(from).ToBig()
		cost    = tx.Cost()
	)
	if balance.Cmp(cost) < 0 {
		return fmt.Errorf("%w: balance %v, tx cost %v, overshot %v", core.ErrInsufficientFunds, balance, cost, new(big.Int).Sub(cost, balance))
	}
	// Ensure the transactor has enough funds to cover for replacements or nonce
	// expansions without overdrafts
	spent := opts.ExistingExpenditure(from)
	if prev := opts.ExistingCost(from, tx.Nonce()); prev != nil {
		bump := new(big.Int).Sub(cost, prev)
		need := new(big.Int).Add(spent, bump)
		if balance.Cmp(need) < 0 {
			return fmt.Errorf("%w: balance %v, queued cost %v, tx bumped %v, overshot %v", core.ErrInsufficientFunds, balance, spent, bump, new(big.Int).Sub(need, balance))
		}
	} else {
		need := new(big.Int).Add(spent, cost)
		if balance.Cmp(need) < 0 {
			return fmt.Errorf("%w: balance %v, queued cost %v, tx cost %v, overshot %v", core.ErrInsufficientFunds, balance, spent, cost, new(big.Int).Sub(need, balance))
		}
		// Transaction takes a new nonce value out of the pool. Ensure it doesn't
		// overflow the number of permitted transactions from a single account
		// (i.e. max cancellable via out-of-bound transaction).
		if opts.UsedAndLeftSlots != nil {
			if used, left := opts.UsedAndLeftSlots(from); left <= 0 {
				return fmt.Errorf("%w: pooled %d txs", ErrAccountLimitExceeded, used)
			}
		}
	}
	return nil
}

func frameTxValidationPrefix(frames []types.Frame, sender common.Address) (int, int, error) {
	verifies := func(frame *types.Frame, flags uint64) bool {
		return frame.Mode == types.ModeVerify && frame.Flags == flags && (flags == types.ApprovePayment || frame.ResolvedTarget(sender) == sender)
	}
	start, deploy := 0, -1
	if len(frames) > 0 && frames[0].IsExpiryVerifier() {
		start = 1
	}
	if start < len(frames) && frames[start].Mode == types.ModeDefault && frames[start].Flags == 0 {
		deploy, start = start, start+1
	}
	var prefix int
	switch {
	case start < len(frames) && verifies(&frames[start], types.ApproveExecutionAndPayment):
		prefix = start + 1
	case start+1 < len(frames) && verifies(&frames[start], types.ApproveExecution) && verifies(&frames[start+1], types.ApprovePayment):
		prefix = start + 2
	default:
		return 0, 0, errors.New("frame transaction validation prefix is not a recognized shape")
	}
	for i := prefix; i < len(frames); i++ {
		if frames[i].Mode == types.ModeVerify {
			return 0, 0, errors.New("frame transaction has a VERIFY frame after the validation prefix")
		}
	}
	return prefix, deploy, nil
}

func validateFrameTxPrefix(tx *types.Transaction, signer types.Signer, opts *ValidationOptionsWithState) error {
	msg, err := core.TransactionToMessage(tx, signer, nil)
	if err != nil {
		return err
	}
	prefix, deploy, err := frameTxValidationPrefix(msg.Frames, msg.From)
	if err != nil {
		return err
	}
	verifyGas, verifyStateGas := types.FrameTxBudgetTotals(msg.Frames[:prefix])
	for i := range msg.FrameSignatures {
		verifyGas += types.FrameTxSignatureGas(&msg.FrameSignatures[i])
	}
	if verifyGas > frameTxMaxVerifyGas {
		return fmt.Errorf("frame transaction validation prefix gas %d exceeds limit %d", verifyGas, frameTxMaxVerifyGas)
	}
	if verifyStateGas > frameTxMaxVerifyStateGas {
		return fmt.Errorf("frame transaction validation prefix state gas %d exceeds limit %d", verifyStateGas, frameTxMaxVerifyStateGas)
	}
	msg.SkipNonceChecks = true
	msg.ValidationPrefixOnly = true

	head := opts.Head
	blockContext := vm.BlockContext{
		CanTransfer:      core.CanTransfer,
		Transfer:         core.Transfer,
		GetHash:          func(uint64) common.Hash { return common.Hash{} },
		BlockNumber:      new(big.Int).Set(head.Number),
		Time:             head.Time,
		Difficulty:       new(big.Int).Set(head.Difficulty),
		BaseFee:          new(big.Int),
		BlobBaseFee:      msg.BlobGasFeeCap.ToBig(),
		GasLimit:         head.GasLimit,
		CostPerStateByte: params.CostPerStateByte,
	}
	if head.Difficulty.Sign() == 0 {
		blockContext.Random = &head.MixDigest
	}
	var (
		statedb     = opts.State.Copy()
		precompiles = vm.ActivePrecompiles(opts.Config.Rules(head.Number, head.Difficulty.Sign() == 0, head.Time))
		calls       = []vm.OpCode{vm.CALL, vm.CALLCODE, vm.DELEGATECALL, vm.STATICCALL}
		evm         *vm.EVM
		violation   error
		reads       = make(FrameTxReads)
	)
	read := func(addr common.Address) *FrameTxRead {
		if reads[addr] == nil {
			reads[addr] = &FrameTxRead{
				Nonce:    opts.State.GetNonce(addr),
				Balance:  opts.State.GetBalance(addr).Clone(),
				CodeHash: opts.State.GetCodeHash(addr),
				Storage:  make(map[common.Hash]common.Hash),
			}
		}
		return reads[addr]
	}
	read(msg.From)
	violate := func(format string, args ...any) {
		if violation == nil {
			violation = fmt.Errorf("frame transaction validation prefix "+format, args...)
			evm.Cancel()
		}
	}
	callable := func(addr common.Address) bool {
		if addr == msg.From || slices.Contains(precompiles, addr) {
			return true
		}
		code := statedb.GetCode(addr)
		_, delegated := types.ParseDelegation(code)
		return len(code) > 0 && !delegated
	}
	hooks := &tracing.Hooks{
		OnEnter: func(depth int, typ byte, from, to common.Address, input []byte, gas uint64, value *big.Int) {
			read(to)
			if value != nil && value.Sign() != 0 {
				violate("transfers value to %v", to)
			}
			if (vm.OpCode(typ) == vm.CREATE || vm.OpCode(typ) == vm.CREATE2) && to != msg.From {
				violate("creates contract %v outside the sender", to)
			}
		},
		OnOpcode: func(pc uint64, op byte, gas, cost uint64, scope tracing.OpContext, rData []byte, depth int, err error) {
			if err != nil {
				return
			}
			var (
				frame  = evm.TxContext.FrameContext.CurrentFrame
				stack  = scope.StackData()
				opcode = vm.OpCode(op)
			)
			switch opcode {
			case vm.GASPRICE, vm.BLOCKHASH, vm.COINBASE, vm.NUMBER, vm.PREVRANDAO, vm.GASLIMIT, vm.BASEFEE, vm.BLOBBASEFEE, vm.SLOTNUM, vm.INVALID, vm.SELFDESTRUCT, vm.BALANCE, vm.SELFBALANCE:
				violate("uses banned opcode %v", opcode)
			case vm.TIMESTAMP:
				if !msg.Frames[frame].IsExpiryVerifier() || scope.Address() != params.FrameTxExpiryVerifier || !bytes.Equal(scope.ContractCode(), params.FrameTxExpiryVerifierCode) {
					violate("uses banned opcode %v", opcode)
				}
			case vm.GAS:
				if code := scope.ContractCode(); pc+1 >= uint64(len(code)) || !slices.Contains(calls, vm.OpCode(code[pc+1])) {
					violate("uses banned opcode %v", opcode)
				}
			case vm.CREATE, vm.CREATE2:
				if frame != deploy {
					violate("uses banned opcode %v", opcode)
				}
			case vm.SSTORE:
				if frame != deploy || scope.Address() != msg.From {
					violate("writes storage of %v", scope.Address())
				}
			case vm.SLOAD:
				slot := common.Hash(stack[len(stack)-1].Bytes32())
				read(scope.Address()).Storage[slot] = opts.State.GetState(scope.Address(), slot)
				if scope.Address() != msg.From {
					violate("reads storage of %v", scope.Address())
				}
			case vm.CALL, vm.CALLCODE, vm.DELEGATECALL, vm.STATICCALL:
				target := common.Address(stack[len(stack)-2].Bytes20())
				read(target)
				if !callable(target) {
					violate("calls %v, which is not an undelegated contract or precompile", target)
				}
			case vm.EXTCODESIZE, vm.EXTCODECOPY, vm.EXTCODEHASH:
				target := common.Address(stack[len(stack)-1].Bytes20())
				read(target)
				if !callable(target) {
					violate("reads code of %v, which is not an undelegated contract or precompile", target)
				}
			}
		},
	}
	evm = vm.NewEVM(blockContext, statedb, opts.Config, vm.Config{Tracer: hooks})
	result, err := core.ApplyMessage(evm, msg, core.NewGasPool(head.GasLimit))
	if violation != nil {
		return violation
	}
	if err != nil {
		return fmt.Errorf("frame transaction validation prefix failed: %w", err)
	}
	for i, receipt := range result.FrameReceipts {
		if receipt.Status != types.ReceiptStatusSuccessful {
			return fmt.Errorf("frame transaction validation prefix frame %d failed", i)
		}
	}
	if deploy >= 0 && len(statedb.GetCode(msg.From)) == 0 {
		return errors.New("frame transaction deploy frame installed no code at the sender")
	}
	if opts.FrameTxPayment == nil {
		return nil
	}
	cost := new(uint256.Int).Mul(uint256.NewInt(msg.GasLimit), msg.GasFeeCap)
	cost.Add(cost, new(uint256.Int).Mul(uint256.NewInt(tx.BlobGas()), msg.BlobGasFeeCap))
	payer := *result.FramePayer
	read(payer)
	paymaster := msg.Frames[prefix-1].Flags == types.ApprovePayment && len(statedb.GetCode(payer)) > 0
	return opts.FrameTxPayment(tx, payer, cost, paymaster, reads)
}
