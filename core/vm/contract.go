// Copyright 2015 The go-ethereum Authors
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

package vm

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/tracing"
	"github.com/holiman/uint256"
)

// Contract represents an ethereum contract in the state database. It contains
// the contract code, calling arguments. Contract implements ContractRef
type Contract struct {
	// caller is the result of the caller which initialised this
	// contract. However, when the "call method" is delegated this
	// value needs to be initialised to that of the caller's caller.
	caller  common.Address
	address common.Address

	jumpDests JumpDestCache // Aggregated result of JUMPDEST analysis.
	analysis  BitVec        // Locally cached result of JUMPDEST analysis

	Code     []byte
	CodeHash common.Hash
	Input    []byte

	// is the execution frame represented by this object a contract deployment
	IsDeployment bool
	IsSystemCall bool

	Gas     GasBudget
	GasUsed GasUsed // EIP-8037: canonical per-frame gas usage accumulator
	value   *uint256.Int
}

// NewContract returns a new contract environment for the execution of EVM.
func NewContract(caller common.Address, address common.Address, value *uint256.Int, gas GasBudget, jumpDests JumpDestCache) *Contract {
	// Initialize the jump analysis cache if it's nil, mostly for tests
	if jumpDests == nil {
		jumpDests = newMapJumpDests()
	}
	return &Contract{
		caller:    caller,
		address:   address,
		jumpDests: jumpDests,
		Gas:       gas,
		value:     value,
	}
}

func (c *Contract) validJumpdest(dest *uint256.Int) bool {
	udest, overflow := dest.Uint64WithOverflow()
	// PC cannot go beyond len(code) and certainly can't be bigger than 63bits.
	// Don't bother checking for JUMPDEST in that case.
	if overflow || udest >= uint64(len(c.Code)) {
		return false
	}
	// Only JUMPDESTs allowed for destinations
	if OpCode(c.Code[udest]) != JUMPDEST {
		return false
	}
	return c.isCode(udest)
}

// isCode returns true if the provided PC location is an actual opcode, as
// opposed to a data-segment following a PUSHN operation.
func (c *Contract) isCode(udest uint64) bool {
	// Do we already have an analysis laying around?
	if c.analysis != nil {
		return c.analysis.codeSegment(udest)
	}
	// Do we have a contract hash already?
	// If we do have a hash, that means it's a 'regular' contract. For regular
	// contracts ( not temporary initcode), we store the analysis in a map
	if c.CodeHash != (common.Hash{}) {
		// Does parent context have the analysis?
		analysis, exist := c.jumpDests.Load(c.CodeHash)
		if !exist {
			// Do the analysis and save in parent context
			// We do not need to store it in c.analysis
			analysis = codeBitmap(c.Code)
			c.jumpDests.Store(c.CodeHash, analysis)
		}
		// Also stash it in current contract for faster access
		c.analysis = analysis
		return analysis.codeSegment(udest)
	}
	// We don't have the code hash, most likely a piece of initcode not already
	// in state trie. In that case, we do an analysis, and save it locally, so
	// we don't have to recalculate it for every JUMP instruction in the execution
	// However, we don't save it within the parent context
	if c.analysis == nil {
		c.analysis = codeBitmap(c.Code)
	}
	return c.analysis.codeSegment(udest)
}

// GetOp returns the n'th element in the contract's byte array
func (c *Contract) GetOp(n uint64) OpCode {
	if n < uint64(len(c.Code)) {
		return OpCode(c.Code[n])
	}

	return STOP
}

// Caller returns the caller of the contract.
//
// Caller will recursively call caller when the contract is a delegate
// call, including that of caller's caller.
func (c *Contract) Caller() common.Address {
	return c.caller
}

// UseGas attempts the use gas and subtracts it and returns true on success
func (c *Contract) UseGas(cost GasCosts, logger *tracing.Hooks, reason tracing.GasChangeReason) (ok bool) {
	prior, ok := c.Gas.Charge(cost)
	if !ok {
		return false
	}
	if logger != nil && logger.OnGasChange != nil && reason != tracing.GasChangeIgnored {
		logger.OnGasChange(prior, c.Gas.RegularGas, reason)
	}
	c.GasUsed.Add(cost)
	return true
}

// RefundGas refunds gas to the contract. gasUsed carries the child frame's
// accumulated gas usage metrics (EIP-8037), incorporated on both success and error.
func (c *Contract) RefundGas(err error, initialRegularGasUsed uint64, gas GasBudget, gasUsed GasUsed, logger *tracing.Hooks, reason tracing.GasChangeReason) {
	// If the preceding call errored, return the state gas
	// to the parent call
	if err != nil {
		gas.StateGas += gasUsed.StateGas
		gasUsed.StateGas = 0
	}
	if gas.RegularGas == 0 && gas.StateGas == 0 && gasUsed.StateGas == 0 && gasUsed.RegularGas == 0 {
		return
	}
	if logger != nil && logger.OnGasChange != nil && reason != tracing.GasChangeIgnored {
		logger.OnGasChange(c.Gas.RegularGas, c.Gas.RegularGas+gas.RegularGas, reason)
	}
	c.Gas.RegularGas += gas.RegularGas
	c.Gas.StateGas = gas.StateGas
	c.GasUsed.StateGas += gasUsed.StateGas
	c.GasUsed.RegularGas = initialRegularGasUsed + gasUsed.RegularGas
}

// Refunds the account creation state costs if a CREATE/CREATE2 call fails.
func (c *Contract) RefundCreateStateGas(refund uint64) {
	if refund > 0 {
		c.Gas.StateGas += refund
		if c.GasUsed.StateGas >= refund {
			c.GasUsed.StateGas -= refund
		} else {
			c.GasUsed.StateGas = 0
		}
	}
}

// Address returns the contracts address
func (c *Contract) Address() common.Address {
	return c.address
}

// Value returns the contract's value (sent to it from it's caller)
func (c *Contract) Value() *uint256.Int {
	return c.value
}

// SetCallCode sets the code of the contract,
func (c *Contract) SetCallCode(hash common.Hash, code []byte) {
	c.Code = code
	c.CodeHash = hash
}
