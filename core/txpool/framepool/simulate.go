// Copyright 2026 The go-ethereum Authors
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

package framepool

import (
	"bytes"
	"errors"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/tracing"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/params"
	"github.com/holiman/uint256"
)

// ErrTraceViolation marks validation prefixes forbidden by public mempool policy.
var ErrTraceViolation = errors.New("frame validation trace violation")

type accountFields uint8

const (
	dependencyNonce accountFields = 1 << iota
	dependencyBalance
	dependencyCode
)

// dependencies stores keys only. Each field is independently invalidated by
// block access list changes; storage reads are always in the sender's account.
type dependencies struct {
	accounts map[common.Address]accountFields
	slots    map[common.Hash]struct{}
}

func (d *dependencies) add(address common.Address, fields accountFields) {
	d.accounts[address] |= fields
}

type simResult struct {
	payer          common.Address
	payerBalance   *uint256.Int // Head balance, before APPROVE's maximum-cost debit.
	codedPaymaster bool
	dependencies   dependencies
	expiryDeadline *uint64
}

// simulate consumes a fresh disposable StateDB at head.Root. Signatures must
// already have been verified and prefix classified. The full transaction is
// retained for introspection and maximum-cost approval accounting.
// APPROVE's nonce increment and full maximum-cost debit remain in statedb;
// callers must discard it on both success and failure. Dependencies describe
// head-state inputs, including native default VERIFY and delegated frame targets.
// Coded paymasters use the generic trace policy, with no canonical exception.
func simulate(config *params.ChainConfig, head *types.Header, statedb *state.StateDB, tx *types.Transaction, prefix Prefix) (*simResult, error) {
	msg, err := core.TransactionToMessage(tx, types.MakeSigner(config, head.Number, head.Time), head.BaseFee)
	if err != nil {
		return nil, err
	}
	sender := msg.From
	expectedPayer := msg.Frames[prefix.End].ResolvedTarget(sender)
	result := &simResult{dependencies: dependencies{accounts: make(map[common.Address]accountFields), slots: make(map[common.Hash]struct{})}}
	result.payerBalance = new(uint256.Int).Set(statedb.GetBalance(expectedPayer))
	deps := &result.dependencies
	deps.add(sender, dependencyNonce|dependencyBalance|dependencyCode)
	senderHadCode := len(statedb.GetCode(sender)) != 0
	if prefix.DeployFrame >= 0 && senderHadCode {
		return nil, fmt.Errorf("%w: deploy sender already has code", ErrTraceViolation)
	}
	if prefix.PayFrame >= 0 {
		target := msg.Frames[prefix.PayFrame].ResolvedTarget(sender)
		result.codedPaymaster = len(statedb.GetCode(target)) != 0
	}
	for i := 0; i <= prefix.End; i++ {
		target := msg.Frames[i].ResolvedTarget(sender)
		deps.add(target, dependencyCode)
		if delegate, ok := types.ParseDelegation(statedb.GetCode(target)); ok {
			deps.add(delegate, dependencyCode)
		}
	}
	if prefix.ExpiryFrame >= 0 {
		if !bytes.Equal(statedb.GetCode(params.FrameTxExpiryVerifier), params.FrameTxExpiryVerifierCode) {
			return nil, fmt.Errorf("%w: non-canonical expiry verifier", ErrTraceViolation)
		}
		deadline := prefix.ExpiryDeadline
		result.expiryDeadline = &deadline
	}
	// Only head number/time and merge status affect fork selection. Other block
	// reads are banned, but safe values let execution finish after a violation.
	ctx := vm.BlockContext{
		CanTransfer: core.CanTransfer, Transfer: core.Transfer,
		GetHash:     func(uint64) common.Hash { return common.Hash{} },
		BlockNumber: new(big.Int).Set(head.Number), Time: head.Time,
		Difficulty: new(big.Int), BaseFee: new(big.Int), BlobBaseFee: new(big.Int),
		GasLimit: head.GasLimit, CostPerStateByte: params.CostPerStateByte,
	}
	if head.BaseFee != nil {
		ctx.BaseFee.Set(head.BaseFee)
	}
	if head.Difficulty != nil {
		ctx.Difficulty.Set(head.Difficulty)
	}
	if ctx.Difficulty.Sign() == 0 {
		ctx.Random = &head.MixDigest
	}
	precompiles := make(map[common.Address]struct{})
	for _, address := range vm.ActivePrecompiles(config.Rules(head.Number, ctx.Random != nil, head.Time)) {
		precompiles[address] = struct{}{}
	}
	var evm *vm.EVM
	var violation error
	reject := func(reason string) {
		if violation == nil {
			violation = fmt.Errorf("%w: %s", ErrTraceViolation, reason)
		}
	}
	checkTarget := func(target common.Address) {
		deps.add(target, dependencyCode)
		if target == sender {
			return
		}
		code := statedb.GetCode(target)
		if _, delegated := types.ParseDelegation(code); delegated {
			reject("delegated nested target")
		} else if _, precompile := precompiles[target]; !precompile && len(code) == 0 {
			reject("empty nested target")
		}
	}
	hooks := &tracing.Hooks{
		OnEnter: func(depth int, typ byte, from, to common.Address, input []byte, gas uint64, value *big.Int) {
			op := vm.OpCode(typ)
			switch op {
			case vm.CALL, vm.CALLCODE, vm.DELEGATECALL, vm.STATICCALL, vm.CREATE, vm.CREATE2:
				if value != nil && value.Sign() != 0 {
					reject("value-bearing call or creation")
				}
			default:
				return
			}
			if op == vm.CREATE || op == vm.CREATE2 {
				if evm.TxContext.FrameContext.CurrentFrame != prefix.DeployFrame || to != sender {
					reject("creation outside sender deploy")
				}
				if op == vm.CREATE {
					deps.add(from, dependencyNonce)
				}
			} else if depth >= 1 {
				checkTarget(to)
			}
		},
		OnOpcode: func(pc uint64, raw byte, gas, cost uint64, scope tracing.OpContext, data []byte, depth int, opcodeErr error) {
			frame := evm.TxContext.FrameContext.CurrentFrame
			op := vm.OpCode(raw)
			switch op {
			case vm.GASPRICE, vm.BLOCKHASH, vm.COINBASE, vm.NUMBER, vm.PREVRANDAO, vm.GASLIMIT, vm.BASEFEE, vm.BLOBBASEFEE, vm.SLOTNUM, vm.INVALID, vm.SELFDESTRUCT, vm.BALANCE, vm.SELFBALANCE:
				reject("banned opcode " + op.String())
			case vm.TIMESTAMP:
				if frame != prefix.ExpiryFrame || scope.Address() != params.FrameTxExpiryVerifier || !bytes.Equal(scope.ContractCode(), params.FrameTxExpiryVerifierCode) {
					reject("timestamp outside canonical expiry verifier")
				}
			case vm.GAS:
				code := scope.ContractCode()
				if pc+1 >= uint64(len(code)) {
					reject("gas not followed by call")
					break
				}
				switch vm.OpCode(code[pc+1]) {
				case vm.CALL, vm.CALLCODE, vm.DELEGATECALL, vm.STATICCALL:
				default:
					reject("gas not followed by call")
				}
			case vm.CREATE, vm.CREATE2:
				if frame != prefix.DeployFrame {
					reject("creation outside deploy frame")
				}
			case vm.SSTORE:
				if frame != prefix.DeployFrame || scope.Address() != sender {
					reject("storage write outside sender deploy")
				}
			case vm.SLOAD:
				if scope.Address() != sender {
					reject("foreign storage read")
				} else if stack := scope.StackData(); len(stack) > 0 {
					deps.slots[common.Hash(stack[len(stack)-1].Bytes32())] = struct{}{}
				}
			case vm.EXTCODESIZE, vm.EXTCODECOPY, vm.EXTCODEHASH:
				if stack := scope.StackData(); len(stack) > 0 {
					checkTarget(common.Address(stack[len(stack)-1].Bytes20()))
				}
			}
		},
	}
	evm = vm.NewEVM(ctx, statedb, config, vm.Config{Tracer: hooks})
	approval, err := core.ApplyFramePrefix(evm, msg, nil, prefix.End)
	if violation != nil {
		return nil, violation
	}
	if err != nil {
		return nil, err
	}
	if approval.Payer != expectedPayer {
		return nil, fmt.Errorf("%w: unexpected prefix payer", core.ErrFrameTxInvalidExecution)
	}
	if prefix.DeployFrame >= 0 && len(statedb.GetCode(sender)) == 0 {
		return nil, fmt.Errorf("%w: deploy installed no sender code", ErrTraceViolation)
	}
	result.payer = approval.Payer
	deps.add(result.payer, dependencyBalance|dependencyCode)
	return result, nil
}
