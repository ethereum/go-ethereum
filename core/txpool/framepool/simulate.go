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
	"math"
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

// creatorNonceMargin bounds how far below 2^64-1 a creator's nonce must be for
// no single block to reach the creation nonce-overflow precheck.
const creatorNonceMargin = 1 << 32

// frameParamGasUsed is the FRAMEPARAM selector of a frame's execution gas usage.
const frameParamGasUsed = 0x0a

type accountFields uint8

const (
	dependencyNonce accountFields = 1 << iota
	dependencyBalance
	dependencyCode
)

// dependencies records account fields and storage locations independently.
// System storage values also retain the expected recent-root entry hashes.
type dependencies struct {
	accounts map[common.Address]accountFields
	slots    map[storageLocation]common.Hash
}

type storageLocation struct {
	address common.Address
	slot    common.Hash
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

// simulate is the common pool execution path for admission and revalidation.
func (p *FramePool) simulate(head *types.Header, statedb *state.StateDB, tx *types.Transaction, prefix Prefix) (*simResult, error) {
	p.simulations.Add(1)
	return simulate(p.chain.Config(), head, statedb, tx, prefix)
}

// simulate consumes a fresh disposable StateDB at head.Root. Signatures must
// already have been verified and prefix classified. The full transaction is
// retained for introspection and maximum-cost approval accounting.
// APPROVE's nonce increment and full maximum-cost debit remain in statedb;
// callers must discard it on both success and failure. Dependencies describe
// head-state inputs, including native default VERIFY and delegated frame targets.
// Coded paymasters use the generic trace policy, with no canonical exception.
func simulate(config *params.ChainConfig, head *types.Header, statedb *state.StateDB, tx *types.Transaction, prefix Prefix) (*simResult, error) {
	if err := checkNonces(statedb, tx); err != nil {
		return nil, err
	}
	if !prefix.validRootAge(head) {
		return nil, fmt.Errorf("%w: recent root outside usable window", ErrTraceViolation)
	}
	msg, err := core.TransactionToMessage(tx, types.MakeSigner(config, head.Number, head.Time), head.BaseFee)
	if err != nil {
		return nil, err
	}
	sender := msg.From
	expectedPayer := prefix.Payer
	result := &simResult{dependencies: dependencies{accounts: make(map[common.Address]accountFields), slots: make(map[storageLocation]common.Hash)}}
	result.payerBalance = new(uint256.Int).Set(statedb.GetBalance(expectedPayer))
	deps := &result.dependencies
	deps.add(sender, dependencyBalance|dependencyCode)
	keys := tx.FrameNonceKeys()
	for i := range keys {
		if keys[i].IsZero() {
			deps.add(sender, dependencyNonce)
		} else {
			deps.slots[storageLocation{params.NonceManagerAddress, types.FrameTxNonceSlot(sender, &keys[i])}] = common.Hash{}
		}
	}
	senderHadCode := len(statedb.GetCode(sender)) != 0
	if prefix.DeployFrame >= 0 {
		if senderHadCode {
			return nil, fmt.Errorf("%w: deploy sender already has code", ErrTraceViolation)
		}
		// Stricter than the pinned EIP-8141 deploy rules: a single
		// authorization per block could re-delegate a shared factory, making
		// every pending deploy through it re-simulate. Nested delegated
		// targets are rejected during execution.
		factory := msg.Frames[prefix.DeployFrame].ResolvedTarget(sender)
		if _, delegated := types.ParseDelegation(statedb.GetCode(factory)); delegated {
			return nil, fmt.Errorf("%w: delegated deploy factory", ErrTraceViolation)
		}
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
	if prefix.RecentRootFrame >= 0 {
		if !bytes.Equal(statedb.GetCode(params.RecentRootAddress), params.RecentRootCode) {
			return nil, fmt.Errorf("%w: non-canonical recent root verifier", ErrTraceViolation)
		}
		data := msg.Frames[prefix.RecentRootFrame].Data
		for offset := 0; offset < len(data); offset += recentRootTupleBytes {
			key, entry := recentRootDependency(data[offset : offset+recentRootTupleBytes])
			deps.slots[storageLocation{params.RecentRootAddress, key}] = entry
		}
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
	if head.SlotNumber != nil {
		ctx.SlotNum = *head.SlotNumber + 1
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
			switch op := vm.OpCode(typ); op {
			case vm.CREATE, vm.CREATE2:
				if evm.TxContext.FrameContext.CurrentFrame != prefix.DeployFrame || to != sender {
					reject("creation outside sender deploy")
				}
				if op == vm.CREATE {
					deps.add(from, dependencyNonce)
				}
			case vm.CALL, vm.CALLCODE, vm.DELEGATECALL, vm.STATICCALL:
				if depth >= 1 {
					checkTarget(to)
				}
			}
		},
		OnOpcode: func(pc uint64, raw byte, gas, cost uint64, scope tracing.OpContext, data []byte, depth int, opcodeErr error) {
			frame := evm.TxContext.FrameContext.CurrentFrame
			op := vm.OpCode(raw)
			switch op {
			case vm.GASPRICE, vm.BLOCKHASH, vm.COINBASE, vm.NUMBER, vm.PREVRANDAO, vm.GASLIMIT, vm.BASEFEE, vm.BLOBBASEFEE, vm.INVALID, vm.SELFDESTRUCT, vm.BALANCE, vm.SELFBALANCE:
				reject("banned opcode " + op.String())
			case vm.SLOTNUM:
				if frame != prefix.RecentRootFrame || depth != 1 || scope.Address() != params.RecentRootAddress || !bytes.Equal(scope.ContractCode(), params.RecentRootCode) {
					reject("slot number outside canonical recent root verifier")
				}
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
			case vm.CALL, vm.CALLCODE:
				// Values are checked before execution, see CREATE.
				if stack := scope.StackData(); len(stack) >= 3 && !stack[len(stack)-3].IsZero() {
					reject("value-bearing call or creation")
				}
			case vm.CREATE, vm.CREATE2:
				if frame != prefix.DeployFrame {
					reject("creation outside deploy frame")
				}
				// A creator unable to afford the endowment fails the creation
				// without entering it, which would leave its balance an
				// unrecorded input.
				if stack := scope.StackData(); len(stack) > 0 && !stack[len(stack)-1].IsZero() {
					reject("value-bearing call or creation")
				}
				// A creation fails its precheck, before entering, at the
				// creator's maximum nonce. Nonces grow by one per creation,
				// transaction or authorization, so no block advances one by
				// creatorNonceMargin; only creators this close to the limit
				// depend on their nonce. Shared CREATE2 factories, whose nonce
				// every deployment bumps, otherwise stay untracked.
				if creator := scope.Address(); math.MaxUint64-statedb.GetNonce(creator) <= creatorNonceMargin {
					deps.add(creator, dependencyNonce)
				}
			case vm.SSTORE, vm.SLOAD:
				if op == vm.SLOAD && frame == prefix.RecentRootFrame && depth == 1 && scope.Address() == params.RecentRootAddress && bytes.Equal(scope.ContractCode(), params.RecentRootCode) {
					if stack := scope.StackData(); len(stack) > 0 {
						if _, ok := deps.slots[storageLocation{params.RecentRootAddress, common.Hash(stack[len(stack)-1].Bytes32())}]; !ok {
							reject("undeclared recent root storage read")
						}
					}
					break
				}
				if op == vm.SSTORE && (frame != prefix.DeployFrame || scope.Address() != sender) {
					reject("storage write outside sender deploy")
				} else if scope.Address() != sender {
					reject("foreign storage read")
				} else if stack := scope.StackData(); len(stack) > 0 {
					// SSTORE gas depends on the slot's head value too.
					deps.slots[storageLocation{sender, common.Hash(stack[len(stack)-1].Bytes32())}] = common.Hash{}
				}
			case vm.EXTCODESIZE, vm.EXTCODECOPY, vm.EXTCODEHASH:
				if stack := scope.StackData(); len(stack) > 0 {
					target := common.Address(stack[len(stack)-1].Bytes20())
					checkTarget(target)
					// EXTCODEHASH of a codeless account, such as a precompile,
					// is zero exactly when the account is EIP-161 empty.
					if op == vm.EXTCODEHASH && len(statedb.GetCode(target)) == 0 {
						deps.add(target, dependencyNonce|dependencyBalance)
					}
				}
			case vm.TXPARAM:
				if stack := scope.StackData(); len(stack) > 0 && stack[len(stack)-1].IsUint64() && stack[len(stack)-1].Uint64() == 0x0d {
					deps.add(sender, dependencyNonce)
				}
			case vm.FRAMEPARAM:
				// An earlier frame's gas usage reveals gas, see OnExit.
				if stack := scope.StackData(); len(stack) >= 2 && stack[len(stack)-2].IsUint64() && stack[len(stack)-2].Uint64() == frameParamGasUsed {
					reject("frame gas usage read")
				}
			}
		},
		// Stricter than the pinned EIP-8141 trace rules, like ERC-7562 OP-020:
		// execution gas depends on the block author, whose account is warm
		// during inclusion (EIP-3651) but not during simulation, so a prefix
		// observing gas may pass here and fail for some builders. GAS is
		// banned except before *CALL; a nested frame running out of gas and
		// FRAMEPARAM's gas usage of an earlier frame are the other channels.
		// A failed top-level frame already rejects the prefix.
		OnExit: func(depth int, output []byte, gasUsed uint64, err error, reverted bool) {
			if depth >= 1 && (errors.Is(err, vm.ErrOutOfGas) || errors.Is(err, vm.ErrCodeStoreOutOfGas)) {
				reject("out of gas in nested call")
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
