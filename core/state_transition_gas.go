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

	"github.com/ethereum/go-ethereum/core/tracing"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/params"
	"github.com/holiman/uint256"
)

// IntrinsicGas computes the 'intrinsic gas' for a message with the given data.
func IntrinsicGas(data []byte, accessList types.AccessList, authList []types.SetCodeAuthorization, isContractCreation, isHomestead, isEIP2028, isEIP3860 bool) (uint64, error) {
	// Set the starting gas for the raw transaction
	var gas uint64
	if isContractCreation && isHomestead {
		gas = params.TxGasContractCreation
	} else {
		gas = params.TxGas
	}
	dataLen := uint64(len(data))
	// Bump the required gas by the amount of transactional data
	if dataLen > 0 {
		// Zero and non-zero bytes are priced differently
		z := uint64(bytes.Count(data, []byte{0}))
		nz := dataLen - z

		// Make sure we don't exceed uint64 for all data combinations
		nonZeroGas := params.TxDataNonZeroGasFrontier
		if isEIP2028 {
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

		if isContractCreation && isEIP3860 {
			lenWords := toWordSize(dataLen)
			if (math.MaxUint64-gas)/params.InitCodeWordGas < lenWords {
				return 0, ErrGasUintOverflow
			}
			gas += lenWords * params.InitCodeWordGas
		}
	}
	if accessList != nil {
		gas += uint64(len(accessList)) * params.TxAccessListAddressGas
		gas += uint64(accessList.StorageKeys()) * params.TxAccessListStorageKeyGas
	}
	if authList != nil {
		gas += uint64(len(authList)) * params.CallNewAccountGas
	}
	return gas, nil
}

// FloorDataGas computes the minimum gas required for a transaction based on its data tokens (EIP-7623).
func FloorDataGas(data []byte) (uint64, error) {
	var (
		z      = uint64(bytes.Count(data, []byte{0}))
		nz     = uint64(len(data)) - z
		tokens = nz*params.TxTokenPerNonZeroByte + z
	)
	// Check for overflow
	if (math.MaxUint64-params.TxGas)/params.TxCostFloorPerToken < tokens {
		return 0, ErrGasUintOverflow
	}
	// Minimum gas required for a transaction based on its data tokens (EIP-7623).
	return params.TxGas + tokens*params.TxCostFloorPerToken, nil
}

// preCheckGasCancun validates gas fields per London rules
func (st *stateTransition) preCheckGasLondon(msg *Message) error {
	if notLondon := !st.evm.ChainConfig().IsLondon(st.evm.Context.BlockNumber); notLondon {
		return nil
	}

	// Skip the checks if gas fields are zero and baseFee was explicitly disabled (eth_call)
	if skipCheck := st.evm.Config.NoBaseFee && msg.GasFeeCap.BitLen() == 0 && msg.GasTipCap.BitLen() == 0; skipCheck {
		return nil
	}

	if l := msg.GasFeeCap.BitLen(); l > 256 {
		return fmt.Errorf("%w: address %v, maxFeePerGas bit length: %d", ErrFeeCapVeryHigh,
			msg.From.Hex(), l)
	}
	if l := msg.GasTipCap.BitLen(); l > 256 {
		return fmt.Errorf("%w: address %v, maxPriorityFeePerGas bit length: %d", ErrTipVeryHigh,
			msg.From.Hex(), l)
	}
	if msg.GasFeeCap.Cmp(msg.GasTipCap) < 0 {
		return fmt.Errorf("%w: address %v, maxPriorityFeePerGas: %s, maxFeePerGas: %s", ErrTipAboveFeeCap,
			msg.From.Hex(), msg.GasTipCap, msg.GasFeeCap)
	}

	// This will panic if baseFee is nil, but basefee presence is verified
	// as part of header validation.
	if msg.GasFeeCap.Cmp(st.evm.Context.BaseFee) < 0 {
		return fmt.Errorf("%w: address %v, maxFeePerGas: %s, baseFee: %s", ErrFeeCapTooLow,
			msg.From.Hex(), msg.GasFeeCap, st.evm.Context.BaseFee)
	}

	return nil
}

// preCheckGasCancun validates gas fields per Cancun rules
func (st *stateTransition) preCheckGasCancun(msg *Message) error {
	if notCancun := !st.evm.ChainConfig().IsCancun(st.evm.Context.BlockNumber, st.evm.Context.Time); notCancun {
		return nil
	}
	if noBlobGasUsed := st.blobGasUsed() == 0; noBlobGasUsed {
		return nil
	}

	// Skip the checks if gas fields are zero and blobBaseFee was explicitly disabled (eth_call)
	skipCheck := st.evm.Config.NoBaseFee && msg.BlobGasFeeCap.BitLen() == 0
	if skipCheck {
		return nil
	}

	// This will panic if blobBaseFee is nil, but blobBaseFee presence
	// is verified as part of header validation.
	if msg.BlobGasFeeCap.Cmp(st.evm.Context.BlobBaseFee) < 0 {
		return fmt.Errorf("%w: address %v blobGasFeeCap: %v, blobBaseFee: %v", ErrBlobFeeCapTooLow,
			msg.From.Hex(), msg.BlobGasFeeCap, st.evm.Context.BlobBaseFee)
	}

	return nil
}

// buyGas handles paying for max possible gas to be used.
func (st *stateTransition) buyGas() error {
	mgval := new(big.Int).SetUint64(st.msg.GasLimit)
	mgval.Mul(mgval, st.msg.GasPrice)
	balanceCheck := new(big.Int).Set(mgval)
	if st.msg.GasFeeCap != nil {
		balanceCheck.SetUint64(st.msg.GasLimit)
		balanceCheck = balanceCheck.Mul(balanceCheck, st.msg.GasFeeCap)
	}
	balanceCheck.Add(balanceCheck, st.msg.Value)

	st.buyGasCancun(balanceCheck, mgval)

	balanceCheckU256, overflow := uint256.FromBig(balanceCheck)
	if overflow {
		return fmt.Errorf("%w: address %v required balance exceeds 256 bits", ErrInsufficientFunds, st.msg.From.Hex())
	}
	if have, want := st.state.GetBalance(st.msg.From), balanceCheckU256; have.Cmp(want) < 0 {
		return fmt.Errorf("%w: address %v have %v want %v", ErrInsufficientFunds, st.msg.From.Hex(), have, want)
	}
	if err := st.gp.SubGas(st.msg.GasLimit); err != nil {
		return err
	}

	if st.evm.Config.Tracer != nil && st.evm.Config.Tracer.OnGasChange != nil {
		st.evm.Config.Tracer.OnGasChange(0, st.msg.GasLimit, tracing.GasChangeTxInitialBalance)
	}
	st.gasRemaining = st.msg.GasLimit

	st.initialGas = st.msg.GasLimit
	mgvalU256, _ := uint256.FromBig(mgval)
	st.state.SubBalance(st.msg.From, mgvalU256, tracing.BalanceDecreaseGasBuy)
	return nil
}

// buyGasCancun handles paying for (blob) gas per Cancun hardfork rules.
// Note: This function call might mutate passed in balanceCheck and mgval
// This way we skip any unnecessary allocations
func (st *stateTransition) buyGasCancun(balanceCheck, mgval *big.Int) {
	if notCancun := !st.evm.ChainConfig().IsCancun(st.evm.Context.BlockNumber, st.evm.Context.Time); notCancun {
		return
	}

	blobGas := st.blobGasUsed()
	if noBlobGasUsed := blobGas == 0; noBlobGasUsed {
		return
	}

	// Check that the user has enough funds to cover blobGasUsed * tx.BlobGasFeeCap
	blobBalanceCheck := new(big.Int).SetUint64(blobGas)
	blobBalanceCheck.Mul(blobBalanceCheck, st.msg.BlobGasFeeCap)
	balanceCheck.Add(balanceCheck, blobBalanceCheck)
	// Pay for blobGasUsed * actual blob fee
	blobFee := new(big.Int).SetUint64(blobGas)
	blobFee.Mul(blobFee, st.evm.Context.BlobBaseFee)
	mgval.Add(mgval, blobFee)
}

// calcGasRefund computes refund counter, capped to a refund quotient.
func (st *stateTransition) calcGasRefund() uint64 {
	var refund uint64
	if !st.evm.ChainConfig().IsLondon(st.evm.Context.BlockNumber) {
		// Before EIP-3529: refunds were capped to gasUsed / 2
		refund = st.gasUsed() / params.RefundQuotient
	} else {
		// After EIP-3529: refunds are capped to gasUsed / 5
		refund = st.gasUsed() / params.RefundQuotientEIP3529
	}
	if refund > st.state.GetRefund() {
		refund = st.state.GetRefund()
	}
	if st.evm.Config.Tracer != nil && st.evm.Config.Tracer.OnGasChange != nil && refund > 0 {
		st.evm.Config.Tracer.OnGasChange(st.gasRemaining, st.gasRemaining+refund, tracing.GasChangeTxRefunds)
	}
	return refund
}

// returnGas returns ETH for remaining gas,
// exchanged at the original rate.
func (st *stateTransition) returnGas(gasRefund, floorDataGas uint64) {
	st.gasRemaining += gasRefund

	st.returnGasPrague(floorDataGas)

	remaining := uint256.NewInt(st.gasRemaining)
	remaining.Mul(remaining, uint256.MustFromBig(st.msg.GasPrice))
	st.state.AddBalance(st.msg.From, remaining, tracing.BalanceIncreaseGasReturn)

	if st.evm.Config.Tracer != nil && st.evm.Config.Tracer.OnGasChange != nil && st.gasRemaining > 0 {
		st.evm.Config.Tracer.OnGasChange(st.gasRemaining, 0, tracing.GasChangeTxLeftOverReturned)
	}

	// Also return remaining gas to the block gas counter so it is
	// available for the next transaction.
	st.gp.AddGas(st.gasRemaining)
}

// returnGasPrague handles return gas calculation per Prague hardfork rules.
func (st *stateTransition) returnGasPrague(floorDataGas uint64) {
	if notPrague := !st.evm.ChainConfig().IsPrague(st.evm.Context.BlockNumber, st.evm.Context.Time); notPrague {
		return
	}

	// After EIP-7623: Data-heavy transactions pay the floor gas.
	if noNeedToPayFloorGas := st.gasUsed() >= floorDataGas; noNeedToPayFloorGas {
		return
	}

	prev := st.gasRemaining
	st.gasRemaining = st.initialGas - floorDataGas
	if t := st.evm.Config.Tracer; t != nil && t.OnGasChange != nil {
		t.OnGasChange(prev, st.gasRemaining, tracing.GasChangeTxDataFloor)
	}
}

// payTip pays the fee (aka tip) to the validator.
func (st *stateTransition) payTip(rules params.Rules, msg *Message) {
	effectiveTip := st.payTipLondon(msg, msg.GasPrice)
	effectiveTipU256, _ := uint256.FromBig(effectiveTip)

	if st.evm.Config.NoBaseFee && msg.GasFeeCap.Sign() == 0 && msg.GasTipCap.Sign() == 0 {
		// Skip fee payment when NoBaseFee is set and the fee fields
		// are 0. This avoids a negative effectiveTip being applied to
		// the coinbase when simulating calls.
		return
	}

	fee := new(uint256.Int).SetUint64(st.gasUsed())
	fee.Mul(fee, effectiveTipU256)
	st.state.AddBalance(st.evm.Context.Coinbase, fee, tracing.BalanceIncreaseRewardTransactionFee)

	// add the coinbase to the witness iff the fee is greater than 0
	if rules.IsEIP4762 && fee.Sign() != 0 {
		st.evm.AccessEvents.AddAccount(st.evm.Context.Coinbase, true)
	}
}

// payTipLondon handles fee (tip) calculation per London hardfork rules.
// Note: This function call might mutate passed in effectiveTip
func (st *stateTransition) payTipLondon(msg *Message, effectiveTip *big.Int) *big.Int {
	if notLondon := !st.evm.ChainConfig().IsLondon(st.evm.Context.BlockNumber); notLondon {
		return effectiveTip
	}

	effectiveTip = new(big.Int).Sub(msg.GasFeeCap, st.evm.Context.BaseFee)
	if effectiveTip.Cmp(msg.GasTipCap) > 0 {
		effectiveTip = msg.GasTipCap
	}

	return effectiveTip
}

// gasUsed returns the amount of gas used up by the state transition.
func (st *stateTransition) gasUsed() uint64 {
	return st.initialGas - st.gasRemaining
}

// blobGasUsed returns the amount of blob gas used by the message.
func (st *stateTransition) blobGasUsed() uint64 {
	return uint64(len(st.msg.BlobHashes) * params.BlobTxBlobGasPerBlob)
}

// toWordSize returns the ceiled word size required for init code payment calculation.
func toWordSize(size uint64) uint64 {
	if size > math.MaxUint64-31 {
		return math.MaxUint64/32 + 1
	}

	return (size + 31) / 32
}
