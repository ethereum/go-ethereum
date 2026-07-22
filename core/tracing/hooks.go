// Copyright 2024 The go-ethereum Authors
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

// Package tracing defines hooks for 'live tracing' of block processing and transaction
// execution. Here we define the low-level [Hooks] object that carries hooks which are
// invoked by the go-ethereum core at various points in the state transition.
//
// To create a tracer that can be invoked with Geth, you need to register it using
// [github.com/ethereum/go-ethereum/eth/tracers.LiveDirectory.Register].
//
// See https://geth.ethereum.org/docs/developers/evm-tracing/live-tracing for a tutorial.
package tracing

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/trie/trienode"
	"github.com/holiman/uint256"
)

// OpContext provides the context at which the opcode is being
// executed in, including the memory, stack and various contract-level information.
type OpContext interface {
	MemoryData() []byte
	StackData() []uint256.Int
	Caller() common.Address
	Address() common.Address
	CallValue() *uint256.Int
	CallInput() []byte
	ContractCode() []byte
}

// StateDB gives tracers access to the whole state.
type StateDB interface {
	GetBalance(common.Address) *uint256.Int
	GetNonce(common.Address) uint64
	GetCode(common.Address) []byte
	GetCodeHash(common.Address) common.Hash
	GetState(common.Address, common.Hash) common.Hash
	GetTransientState(common.Address, common.Hash) common.Hash
	Exist(common.Address) bool
	GetRefund() uint64
}

// VMContext provides the context for the EVM execution.
type VMContext struct {
	Coinbase    common.Address
	BlockNumber *big.Int
	Time        uint64
	Random      *common.Hash
	BaseFee     *big.Int
	StateDB     StateDB
}

// BlockEvent is emitted upon tracing an incoming block.
// It contains the block as well as consensus related information.
type BlockEvent struct {
	Block     *types.Block
	Finalized *types.Header
	Safe      *types.Header
}

// StateUpdate represents the state mutations resulting from block execution.
// It provides access to account changes, storage changes, and contract code
// deployments with both previous and new values.
type StateUpdate struct {
	OriginRoot  common.Hash // State root before the update
	Root        common.Hash // State root after the update
	BlockNumber uint64

	// AccountChanges contains all account state changes keyed by address.
	AccountChanges map[common.Address]*AccountChange

	// StorageChanges contains all storage slot changes keyed by address and storage slot key.
	StorageChanges map[common.Address]map[common.Hash]*StorageChange

	// CodeChanges contains all contract code changes keyed by address.
	CodeChanges map[common.Address]*CodeChange

	// TrieChanges contains trie node mutations keyed by address hash and trie node path.
	TrieChanges map[common.Hash]map[string]*TrieNodeChange
}

// AccountChange represents a change to an account's state.
type AccountChange struct {
	Prev *types.StateAccount // nil if account was created
	New  *types.StateAccount // nil if account was deleted
}

// StorageChange represents a change to a storage slot.
type StorageChange struct {
	Prev common.Hash // previous value (zero if slot was created)
	New  common.Hash // new value (zero if slot was deleted)
}

type ContractCode struct {
	Hash   common.Hash
	Code   []byte
	Exists bool // true if the code was existent
}

// CodeChange represents a change in contract code of an account.
type CodeChange struct {
	Prev *ContractCode // nil if no code existed before
	New  *ContractCode
}

type TrieNodeChange struct {
	Prev *trienode.Node
	New  *trienode.Node
}

type (
	/*
		- VM events -
	*/

	// TxStartHook is called before the execution of a transaction starts.
	// Call simulations don't come with a valid signature. `from` field
	// to be used for address of the caller.
	TxStartHook = func(vm *VMContext, tx *types.Transaction, from common.Address)

	// TxEndHook is called after the execution of a transaction ends.
	TxEndHook = func(receipt *types.Receipt, err error)

	// EnterHook is invoked when the processing of a message starts.
	//
	// Take note that EnterHook, when in the context of a live tracer, can be invoked
	// outside of the `OnTxStart` and `OnTxEnd` hooks when dealing with system calls,
	// see [OnSystemCallStartHook] and [OnSystemCallEndHook] for more information.
	EnterHook = func(depth int, typ byte, from common.Address, to common.Address, input []byte, gas uint64, value *big.Int)

	// ExitHook is invoked when the processing of a message ends.
	// `revert` is true when there was an error during the execution.
	// Exceptionally, before the homestead hardfork a contract creation that
	// ran out of gas when attempting to persist the code to database did not
	// count as a call failure and did not cause a revert of the call. This will
	// be indicated by `reverted == false` and `err == ErrCodeStoreOutOfGas`.
	//
	// Take note that ExitHook, when in the context of a live tracer, can be invoked
	// outside of the `OnTxStart` and `OnTxEnd` hooks when dealing with system calls,
	// see [OnSystemCallStartHook] and [OnSystemCallEndHook] for more information.
	ExitHook = func(depth int, output []byte, gasUsed uint64, err error, reverted bool)

	// OpcodeHook is invoked just prior to the execution of an opcode.
	OpcodeHook = func(pc uint64, op byte, gas, cost uint64, scope OpContext, rData []byte, depth int, err error)

	// FaultHook is invoked when an error occurs during the execution of an opcode.
	FaultHook = func(pc uint64, op byte, gas, cost uint64, scope OpContext, depth int, err error)

	// GasChangeHook reports changes to the regular execution gas. Tracers
	// that don't need the EIP-8037 (Amsterdam) state-access dimension can
	// implement only this hook; it fires unchanged across the fork. If both
	// this and GasChangeHookV2 are set, only V2 is invoked; implement exactly
	// one to avoid double-counting.
	GasChangeHook = func(old, new uint64, reason GasChangeReason)

	// GasChangeHookV2 is the multi-dimensional successor to GasChangeHook,
	// invoked when any gas dimension changes and exposing the EIP-8037
	// (Amsterdam) state-access dimension alongside the regular one. The
	// non-changing dimension is passed through unchanged in both `old` and
	// `new`, so consumers always see the complete gas vector. Pre-Amsterdam
	// the State field is always zero, making a V2-only tracer behave exactly
	// like a V1 one. If both hooks are set, only V2 is invoked; register at
	// most one to avoid double-counting.
	GasChangeHookV2 = func(old, new Gas, reason GasChangeReason)

	/*
		- Chain events -
	*/

	// BlockchainInitHook is called when the blockchain is initialized.
	BlockchainInitHook = func(chainConfig *params.ChainConfig)

	// CloseHook is called when the blockchain closes.
	CloseHook = func()

	// BlockStartHook is called before executing `block`.
	// `td` is the total difficulty prior to `block`.
	BlockStartHook = func(event BlockEvent)

	// BlockEndHook is called after executing a block.
	BlockEndHook = func(err error)

	// SkippedBlockHook indicates a block was skipped during processing
	// due to it being known previously. This can happen e.g. when recovering
	// from a crash.
	SkippedBlockHook = func(event BlockEvent)

	// GenesisBlockHook is called when the genesis block is being processed.
	GenesisBlockHook = func(genesis *types.Block, alloc types.GenesisAlloc)

	// OnSystemCallStartHook is called when a system call is about to be executed.
	//
	// After this hook, the EVM call tracing will happened as usual so you will
	// receive a `OnEnter/OnExit` as well as state hooks between this hook and the
	// `OnSystemCallEndHook`.
	//
	// Note that system call happens outside normal transaction execution, so the
	// `OnTxStart/OnTxEnd` hooks will not be invoked.
	OnSystemCallStartHook = func()

	// OnSystemCallStartHookV2 is called when a system call is about to be executed. Refer
	// to `OnSystemCallStartHook` for more information.
	OnSystemCallStartHookV2 = func(vm *VMContext)

	// OnSystemCallEndHook is called when a system call has finished executing. Today,
	// this hook is invoked when the EIP-4788 system call is about to be executed to set the
	// beacon block root.
	OnSystemCallEndHook = func()

	// StateUpdateHook is called after state is committed for a block.
	// It provides access to the complete state mutations including account changes,
	// storage changes, trie node mutations, and contract code deployments.
	StateUpdateHook = func(update *StateUpdate)

	/*
		- State events -
	*/

	// BalanceChangeHook is called when the balance of an account changes.
	BalanceChangeHook = func(addr common.Address, prev, new *big.Int, reason BalanceChangeReason)

	// NonceChangeHook is called when the nonce of an account changes.
	NonceChangeHook = func(addr common.Address, prev, new uint64)

	// NonceChangeHookV2 is called when the nonce of an account changes.
	NonceChangeHookV2 = func(addr common.Address, prev, new uint64, reason NonceChangeReason)

	// CodeChangeHook is called when the code of an account changes.
	CodeChangeHook = func(addr common.Address, prevCodeHash common.Hash, prevCode []byte, codeHash common.Hash, code []byte)

	// CodeChangeHookV2 is called when the code of an account changes.
	CodeChangeHookV2 = func(addr common.Address, prevCodeHash common.Hash, prevCode []byte, codeHash common.Hash, code []byte, reason CodeChangeReason)

	// StorageChangeHook is called when the storage of an account changes.
	StorageChangeHook = func(addr common.Address, slot common.Hash, prev, new common.Hash)

	// LogHook is called when a log is emitted.
	LogHook = func(log *types.Log)

	// BlockHashReadHook is called when EVM reads the blockhash of a block.
	BlockHashReadHook = func(blockNumber uint64, hash common.Hash)
)

type Hooks struct {
	// VM events
	OnTxStart     TxStartHook
	OnTxEnd       TxEndHook
	OnEnter       EnterHook
	OnExit        ExitHook
	OnOpcode      OpcodeHook
	OnFault       FaultHook
	OnGasChange   GasChangeHook
	OnGasChangeV2 GasChangeHookV2
	// Chain events
	OnBlockchainInit    BlockchainInitHook
	OnClose             CloseHook
	OnBlockStart        BlockStartHook
	OnBlockEnd          BlockEndHook
	OnSkippedBlock      SkippedBlockHook
	OnGenesisBlock      GenesisBlockHook
	OnSystemCallStart   OnSystemCallStartHook
	OnSystemCallStartV2 OnSystemCallStartHookV2
	OnSystemCallEnd     OnSystemCallEndHook
	OnStateUpdate       StateUpdateHook
	// State events
	OnBalanceChange BalanceChangeHook
	OnNonceChange   NonceChangeHook
	OnNonceChangeV2 NonceChangeHookV2
	OnCodeChange    CodeChangeHook
	OnCodeChangeV2  CodeChangeHookV2
	OnStorageChange StorageChangeHook
	OnLog           LogHook
	// Block hash read
	OnBlockHashRead BlockHashReadHook
}

// HasGasHook reports whether any gas-change hook is registered. Call sites
// should use this to short-circuit before constructing the Gas / GasBudget
// arguments to EmitGasChange when tracing is off — the dispatch is otherwise
// always paid the cost of evaluating those args.
func (h *Hooks) HasGasHook() bool {
	return h != nil && (h.OnGasChangeV2 != nil || h.OnGasChange != nil)
}

// EmitGasChange dispatches a gas change event to the registered hooks. If the
// multi-dimensional OnGasChangeV2 hook is set it is invoked with the full Gas
// vectors; otherwise the single-dimensional OnGasChange hook is invoked with
// the regular-gas dimension only. The call is a no-op when the receiver is
// nil, when neither hook is registered, or when the reason is GasChangeIgnored.
//
// Call sites SHOULD use this helper instead of invoking the hooks directly so
// that both variants stay consistent across the Amsterdam fork boundary.
func (h *Hooks) EmitGasChange(old, new Gas, reason GasChangeReason) {
	if h == nil || reason == GasChangeIgnored {
		return
	}
	if h.OnGasChangeV2 != nil {
		h.OnGasChangeV2(old, new, reason)
		return
	}
	if h.OnGasChange != nil {
		h.OnGasChange(old.Regular, new.Regular, reason)
	}
}

// BalanceChangeReason is used to indicate the reason for a balance change, useful
// for tracing and reporting.
type BalanceChangeReason byte

//go:generate go run golang.org/x/tools/cmd/stringer -type=BalanceChangeReason -trimprefix=BalanceChange -output gen_balance_change_reason_stringer.go

const (
	BalanceChangeUnspecified BalanceChangeReason = 0

	// Issuance
	// BalanceIncreaseRewardMineUncle is a reward for mining an uncle block.
	BalanceIncreaseRewardMineUncle BalanceChangeReason = 1

	// BalanceIncreaseRewardMineBlock is a reward for mining a block.
	BalanceIncreaseRewardMineBlock BalanceChangeReason = 2

	// BalanceIncreaseWithdrawal is ether withdrawn from the beacon chain.
	BalanceIncreaseWithdrawal BalanceChangeReason = 3

	// BalanceIncreaseGenesisBalance is ether allocated at the genesis block.
	BalanceIncreaseGenesisBalance BalanceChangeReason = 4

	// Transaction fees
	// BalanceIncreaseRewardTransactionFee is the transaction tip increasing
	// block builder's balance.
	BalanceIncreaseRewardTransactionFee BalanceChangeReason = 5

	// BalanceDecreaseGasBuy is ether spent to purchase gas for a transaction,
	// part of which is burnt under EIP-1559.
	BalanceDecreaseGasBuy BalanceChangeReason = 6

	// BalanceIncreaseGasReturn is ether returned for unused gas at the end
	// of execution.
	BalanceIncreaseGasReturn BalanceChangeReason = 7

	// DAO fork
	// BalanceIncreaseDaoContract is ether sent to the DAO refund contract.
	BalanceIncreaseDaoContract BalanceChangeReason = 8

	// BalanceDecreaseDaoAccount is ether taken from a DAO account to be moved
	// to the refund contract.
	BalanceDecreaseDaoAccount BalanceChangeReason = 9

	// BalanceChangeTransfer is ether transferred via a call: a decrease for the
	// sender and an increase for the recipient.
	BalanceChangeTransfer BalanceChangeReason = 10

	// BalanceChangeTouchAccount is a zero-value transfer that only touch-creates
	// an account.
	BalanceChangeTouchAccount BalanceChangeReason = 11

	// BalanceIncreaseSelfdestruct is added to the recipient as indicated by a
	// selfdestructing account.
	BalanceIncreaseSelfdestruct BalanceChangeReason = 12

	// BalanceDecreaseSelfdestruct is deducted from a contract due to self-destruct.
	BalanceDecreaseSelfdestruct BalanceChangeReason = 13

	// BalanceDecreaseSelfdestructBurn is ether sent to an account already
	// self-destructed within the same tx (captured at end of tx). It excludes a
	// self-destruct that appoints itself as recipient.
	BalanceDecreaseSelfdestructBurn BalanceChangeReason = 14

	// BalanceChangeRevert reverts the balance to a previous value on call
	// failure. Emitted only when the tracer opts into WrapWithJournal.
	BalanceChangeRevert BalanceChangeReason = 15
)

// Gas represents a multi-dimensional gas budget introduced by EIP-8037.
// It carries the regular execution gas and the state-access gas, which are
// metered independently from the Amsterdam fork onwards.
//
// Before Amsterdam, gas metering is single-dimensional and only the Regular
// field is meaningful; State is always zero. The struct is shaped so that
// pre-Amsterdam call sites can populate it as Gas{Regular: g} without loss
// of fidelity relative to the legacy single-uint64 hook.
type Gas struct {
	Regular uint64 // Regular is the budget for ordinary execution gas.
	State   uint64 // State is the budget dedicated to state-access gas (zero pre-Amsterdam).
}

// GasChangeReason is used to indicate the reason for a gas change, useful
// for tracing and reporting.
//
// There is essentially two types of gas changes, those that can be emitted once
// per transaction and those that can be emitted on a call basis, so possibly
// multiple times per transaction.
//
// They can be recognized easily by their name, those that start with `GasChangeTx`
// are emitted once per transaction, while those that start with `GasChangeCall`
// are emitted on a call basis.
type GasChangeReason byte

//go:generate go run golang.org/x/tools/cmd/stringer -type=GasChangeReason -trimprefix=GasChange -output gen_gas_change_reason_stringer.go

const (
	GasChangeUnspecified GasChangeReason = 0

	// GasChangeTxInitialBalance is the tx's initial balance, equal to its gas
	// limit. At most one per transaction.
	GasChangeTxInitialBalance GasChangeReason = 1

	// GasChangeTxIntrinsicGas is the intrinsic cost of the transaction. Exactly
	// one per transaction.
	GasChangeTxIntrinsicGas GasChangeReason = 2

	// GasChangeTxRefunds is the sum of all refunds accrued during execution
	// (e.g. a storage slot being cleared), an increase. At most one per tx.
	GasChangeTxRefunds GasChangeReason = 3

	// GasChangeTxLeftOverReturned is the gas left at the end of the transaction,
	// returned to the account (its Wei value refunded to the caller). Always a
	// decrease towards 0; not emitted when no gas is left. At most one per tx.
	GasChangeTxLeftOverReturned GasChangeReason = 4

	// GasChangeCallInitialBalance is the initial balance of a call, equal to its
	// gas limit. At most one per call.
	GasChangeCallInitialBalance GasChangeReason = 5

	// GasChangeCallLeftOverReturned is the gas left over that is returned to the
	// caller. Always a decrease towards 0; not emitted when no gas is left.
	GasChangeCallLeftOverReturned GasChangeReason = 6

	// GasChangeCallLeftOverRefunded is the child call's left-over gas given back
	// to the caller. Always an increase; not emitted when nothing is refunded.
	GasChangeCallLeftOverRefunded GasChangeReason = 7

	// GasChangeCallContractCreation is the gas burned for a CREATE.
	GasChangeCallContractCreation GasChangeReason = 8

	// GasChangeCallContractCreation2 is the gas burned for a CREATE2.
	GasChangeCallContractCreation2 GasChangeReason = 9

	// GasChangeCallCodeStorage is the gas charged for code storage.
	GasChangeCallCodeStorage GasChangeReason = 10

	// GasChangeCallOpCode is the gas charged for an executed opcode; the exact
	// opcode is available via OnOpcode.
	GasChangeCallOpCode GasChangeReason = 11

	// GasChangeCallPrecompiledContract is the gas charged for a precompile.
	GasChangeCallPrecompiledContract GasChangeReason = 12

	// GasChangeCallStorageColdAccess is the gas charged for a cold storage
	// access under EIP-2929.
	GasChangeCallStorageColdAccess GasChangeReason = 13

	// GasChangeCallFailedExecution is the remaining gas burned when execution
	// fails without a revert.
	GasChangeCallFailedExecution GasChangeReason = 14

	// GasChangeWitnessContractInit flags a witness addition during the contract
	// creation initialization step.
	GasChangeWitnessContractInit GasChangeReason = 15

	// GasChangeWitnessContractCreation flags a witness addition during the
	// contract creation finalization step.
	GasChangeWitnessContractCreation GasChangeReason = 16

	// GasChangeWitnessCodeChunk flags adding one or more code chunks to the
	// witness.
	GasChangeWitnessCodeChunk GasChangeReason = 17

	// GasChangeWitnessContractCollisionCheck flags a witness addition during the
	// contract address collision check.
	GasChangeWitnessContractCollisionCheck GasChangeReason = 18

	// GasChangeTxDataFloor is the extra gas charged to meet the EIP-7623
	// calldata floor. Always a decrease.
	GasChangeTxDataFloor GasChangeReason = 19

	// GasChangeRefundAccountCreation cancels a pre-charged account-creation cost
	// when no account is created.
	GasChangeRefundAccountCreation GasChangeReason = 20

	// GasChangeTxRuntimeGas is the gas charged for the state-dependent costs of
	// the transaction per EIP-2780.
	GasChangeTxRuntimeGas GasChangeReason = 21

	// GasChangeAccountCreation is the conditional account-creation state cost
	// charged in the creating frame when a CREATE/CREATE2 is about to create a
	// new account (EIP-8037).
	GasChangeAccountCreation GasChangeReason = 22

	// GasChangeIgnored indicates the gas change should be ignored, as it is
	// tracked manually by a direct emit of the gas change event.
	GasChangeIgnored GasChangeReason = 0xFF
)

// NonceChangeReason is used to indicate the reason for a nonce change.
type NonceChangeReason byte

//go:generate go run golang.org/x/tools/cmd/stringer -type=NonceChangeReason -trimprefix NonceChange -output gen_nonce_change_reason_stringer.go

const (
	NonceChangeUnspecified NonceChangeReason = 0

	// NonceChangeGenesis is the nonce allocated to accounts at genesis.
	NonceChangeGenesis NonceChangeReason = 1

	// NonceChangeEoACall is the nonce change due to an EoA call.
	NonceChangeEoACall NonceChangeReason = 2

	// NonceChangeContractCreator is the nonce change of an account creating a contract.
	NonceChangeContractCreator NonceChangeReason = 3

	// NonceChangeNewContract is the nonce change of a newly created contract.
	NonceChangeNewContract NonceChangeReason = 4

	// NonceChangeAuthorization is the nonce change due to a EIP-7702 authorization.
	NonceChangeAuthorization NonceChangeReason = 5

	// NonceChangeRevert reverts the nonce to a previous value on call failure.
	// Emitted only when the tracer opts into WrapWithJournal.
	NonceChangeRevert NonceChangeReason = 6

	// NonceChangeSelfdestruct resets the nonce to zero on a self-destruct.
	NonceChangeSelfdestruct NonceChangeReason = 7
)

// CodeChangeReason is used to indicate the reason for a code change.
type CodeChangeReason byte

//go:generate go run golang.org/x/tools/cmd/stringer -type=CodeChangeReason -trimprefix=CodeChange -output gen_code_change_reason_stringer.go

const (
	CodeChangeUnspecified CodeChangeReason = 0

	// CodeChangeContractCreation is when a new contract is deployed via
	// CREATE/CREATE2 operations.
	CodeChangeContractCreation CodeChangeReason = 1

	// CodeChangeGenesis is when contract code is set during blockchain genesis
	// or initial setup.
	CodeChangeGenesis CodeChangeReason = 2

	// CodeChangeAuthorization is when code is set via EIP-7702 Set Code Authorization.
	CodeChangeAuthorization CodeChangeReason = 3

	// CodeChangeAuthorizationClear is when EIP-7702 delegation is cleared by
	// setting to zero address.
	CodeChangeAuthorizationClear CodeChangeReason = 4

	// CodeChangeSelfDestruct is when contract code is cleared due to self-destruct.
	CodeChangeSelfDestruct CodeChangeReason = 5

	// CodeChangeRevert reverts the code to a previous value on call failure.
	// Emitted only when the tracer opts into WrapWithJournal.
	CodeChangeRevert CodeChangeReason = 6
)
