package tracers

import (
	"bytes"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"math/big"
	"os"
	"runtime/debug"
	"sort"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/math"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/crypto"
	pbeth "github.com/streamingfast/firehose-ethereum/types/pb/sf/ethereum/type/v2"
	"go.uber.org/atomic"
	"golang.org/x/exp/maps"
	"golang.org/x/exp/slices"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
)

var isFirehoseDebugEnabled = os.Getenv("GETH_FIREHOSE_TRACER_DEBUG") != ""
var isFirehoseTracerEnabled = os.Getenv("GETH_FIREHOSE_TRACER_TRACE") != ""

var _ core.BlockchainLogger = (*Firehose)(nil)

var emptyAddress = common.Address{}.Bytes()
var emptyHash = common.Hash{}.Bytes()

type Firehose struct {
	// Global state
	outputBuffer *bytes.Buffer

	// Block state
	inBlock       *atomic.Bool
	block         *pbeth.Block
	blockBaseFee  *big.Int
	blockOrdinal  *Ordinal
	blockLogIndex uint32

	// Transaction state
	inTransaction       *atomic.Bool
	transaction         *pbeth.TransactionTrace
	transactionLogIndex uint32
	isPrecompiledAddr   func(addr common.Address) bool

	// Call state
	callStack               *CallStack
	deferredCallState       *DeferredCallState
	latestCallStartSuicided bool
}

func NewFirehoseLogger() *Firehose {
	// FIXME: Where should we put our actual INIT line?
	// FIXME: Pickup version from go-ethereum (PR comment)
	printToFirehose("INIT", "2.3", "geth", "1.12.0")

	return &Firehose{
		outputBuffer: bytes.NewBuffer(make([]byte, 0, 100*1024*1024)),

		inBlock:       atomic.NewBool(false),
		blockOrdinal:  &Ordinal{},
		blockLogIndex: 0,

		inTransaction:       atomic.NewBool(false),
		transactionLogIndex: 0,

		callStack:               NewCallStack(),
		deferredCallState:       NewDeferredCallState(),
		latestCallStartSuicided: false,
	}
}

// resetBlock resets the block state only, do not reset transaction or call state
func (f *Firehose) resetBlock() {
	f.inBlock.Store(false)
	f.block = nil
	f.blockBaseFee = nil
	f.blockOrdinal.Reset()
	f.blockLogIndex = 0
}

// resetTransaction resets the transaction state and the call state in one shot
func (f *Firehose) resetTransaction() {
	f.inTransaction.Store(false)
	f.transactionLogIndex = 0
	f.isPrecompiledAddr = nil

	f.callStack.Reset()
	f.latestCallStartSuicided = false
	f.deferredCallState.Reset()
}

func (f *Firehose) OnBlockStart(b *types.Block, td *big.Int, finalized *types.Header, safe *types.Header) {
	firehoseDebug("block start number=%d hash=%s", b.NumberU64(), b.Hash())

	f.ensureNotInBlock()

	f.inBlock.Store(true)
	f.block = &pbeth.Block{
		Hash:   b.Hash().Bytes(),
		Number: b.Number().Uint64(),
		Header: newBlockHeaderFromChainBlock(b, firehoseBigIntFromNative(new(big.Int).Add(td, b.Difficulty()))),
		Size:   b.Size(),
		Ver:    3,
	}

	if f.block.Header.BaseFeePerGas != nil {
		f.blockBaseFee = f.block.Header.BaseFeePerGas.Native()
	}

	// FIXME: How are we going to pass `finalized` data around? We will probably need to pass it in the text
	// version of the format. This poses interesting question for a standard convential format.
}

func (f *Firehose) OnBlockEnd(err error) {
	firehoseDebug("block ending err=%s", errorView(err))

	if err == nil {
		f.ensureInBlockAndNotInTrx()
		f.printBlockToFirehose(f.block)
	} else {
		// An error occurred, could have happen in transaction/call context, we must not check if in trx/call, only check in block
		f.ensureInBlock()
	}

	f.resetBlock()
	f.resetTransaction()

	firehoseDebug("block end")
}

func (f *Firehose) CaptureTxStart(evm *vm.EVM, tx *types.Transaction) {
	firehoseDebug("trx start hash=%s type=%d gas=%d input=%s", tx.Hash(), tx.Type(), tx.Gas(), inputView(tx.Data()))

	f.ensureInBlockAndNotInTrxAndNotInCall()

	f.inTransaction.Store(true)
	f.isPrecompiledAddr = evm.IsPrecompileAddr

	signer := types.MakeSigner(evm.ChainConfig(), evm.Context.BlockNumber, evm.Context.Time)

	from, err := types.Sender(signer, tx)
	if err != nil {
		panic(fmt.Errorf("could not recover sender address: %w", err))
	}

	var to common.Address
	if tx.To() == nil {
		to = crypto.CreateAddress(from, evm.StateDB.GetNonce(from))
	} else {
		to = *tx.To()
	}

	v, r, s := tx.RawSignatureValues()

	f.transaction = &pbeth.TransactionTrace{
		BeginOrdinal:         f.blockOrdinal.Next(),
		Hash:                 tx.Hash().Bytes(),
		From:                 from.Bytes(),
		To:                   to.Bytes(),
		Nonce:                tx.Nonce(),
		GasLimit:             tx.Gas(),
		GasPrice:             gasPrice(tx, f.blockBaseFee),
		Value:                firehoseBigIntFromNative(tx.Value()),
		Input:                tx.Data(),
		V:                    emptyBytesToNil(v.Bytes()),
		R:                    emptyBytesToNil(r.Bytes()),
		S:                    emptyBytesToNil(s.Bytes()),
		Type:                 transactionTypeFromChainTxType(tx.Type()),
		AccessList:           newAccessListFromChain(tx.AccessList()),
		MaxFeePerGas:         maxFeePerGas(tx),
		MaxPriorityFeePerGas: maxPriorityFeePerGas(tx),
	}
}

func (f *Firehose) CaptureTxEnd(receipt *types.Receipt, err error) {
	firehoseDebug("trx ending")
	f.ensureInBlockAndInTrx()

	f.block.TransactionTraces = append(f.block.TransactionTraces, f.completeTransaction(receipt))

	// The reset must be done as the very last thing as the CallStack needs to be
	// properly populated for the `completeTransaction` call above to complete correctly.
	f.resetTransaction()

	firehoseDebug("trx end")
}

func (f *Firehose) completeTransaction(receipt *types.Receipt) *pbeth.TransactionTrace {
	firehoseDebug("completing transaction call_count=%d receipt=%s", len(f.transaction.Calls), (*receiptView)(receipt))

	// Sorting needs to happen first, before we populate the state reverted
	slices.SortFunc(f.transaction.Calls, func(i, j *pbeth.Call) bool {
		return i.Index < j.Index
	})

	rootCall := f.transaction.Calls[0]

	if !f.deferredCallState.IsEmpty() {
		f.deferredCallState.MaybePopulateCallAndReset("root", rootCall)
	}

	// Receipt can be nil if an error occurred during the transaction execution, right now we don't have it
	if receipt != nil {
		f.transaction.Index = uint32(receipt.TransactionIndex)
		f.transaction.GasUsed = receipt.GasUsed
		f.transaction.Receipt = newTxReceiptFromChain(receipt)
		f.transaction.Status = transactionStatusFromChainTxReceipt(receipt.Status)
	}

	// It's possible that the transaction was reverted, but we still have a receipt, in that case, we must
	// check the root call
	if rootCall.StatusReverted {
		f.transaction.Status = pbeth.TransactionTraceStatus_REVERTED
	}

	// Order is important, we must populate the state reverted before we remove the log block index
	f.populateStateReverted()
	f.removeLogBlockIndexOnStateRevertedCalls()

	// I think this was never used in Firehose instrumentation actually
	// f.transaction.ReturnData = rootCall.ReturnData
	f.transaction.EndOrdinal = f.blockOrdinal.Next()

	return f.transaction
}

func (f *Firehose) populateStateReverted() {
	// Calls are ordered by execution index. So the algo is quite simple.
	// We loop through the flat calls, at each call, if the parent is present
	// and reverted, the current call is reverted. Otherwise, if the current call
	// is failed, the state is reverted. In all other cases, we simply continue
	// our iteration loop.
	//
	// This works because we see the parent before its children, and since we
	// trickle down the state reverted value down the children, checking the parent
	// of a call will always tell us if the whole chain of parent/child should
	// be reverted
	//
	calls := f.transaction.Calls
	for _, call := range f.transaction.Calls {
		var parent *pbeth.Call
		if call.ParentIndex > 0 {
			parent = calls[call.ParentIndex-1]
		}

		call.StateReverted = (parent != nil && parent.StateReverted) || call.StatusFailed
	}
}

func (f *Firehose) removeLogBlockIndexOnStateRevertedCalls() {
	for _, call := range f.transaction.Calls {
		if call.StateReverted {
			for _, log := range call.Logs {
				log.BlockIndex = 0
			}
		}
	}
}

// CaptureStart implements the EVMLogger interface to initialize the tracing operation.
func (f *Firehose) CaptureStart(from common.Address, to common.Address, create bool, input []byte, gas uint64, value *big.Int) {
	f.callStart("root", rootCallType(create), from, to, input, gas, value)
}

// CaptureEnd is called after the call finishes to finalize the tracing.
func (f *Firehose) CaptureEnd(output []byte, gasUsed uint64, err error) {
	f.callEnd("root", output, gasUsed, err)
}

// CaptureState implements the EVMLogger interface to trace a single step of VM execution.
func (f *Firehose) CaptureState(pc uint64, op vm.OpCode, gas, cost uint64, scope *vm.ScopeContext, rData []byte, depth int, err error) {
	f.captureInterpreterStep(pc, op, gas, cost, scope, rData, depth, err)
}

// CaptureFault implements the EVMLogger interface to trace an execution fault.
func (f *Firehose) CaptureFault(pc uint64, op vm.OpCode, gas, cost uint64, scope *vm.ScopeContext, depth int, err error) {
	f.captureInterpreterStep(pc, op, gas, cost, scope, nil, depth, err)
}

func (f *Firehose) captureInterpreterStep(pc uint64, op vm.OpCode, gas, cost uint64, _ *vm.ScopeContext, rData []byte, depth int, err error) {
	if activeCall := f.callStack.Peek(); activeCall != nil && !activeCall.ExecutedCode {
		activeCall.ExecutedCode = true
		firehoseDebug("setting active call executed code to true")
	}
}

func (f *Firehose) CaptureEnter(typ vm.OpCode, from common.Address, to common.Address, input []byte, gas uint64, value *big.Int) {
	f.ensureInBlockAndInTrx()

	// The invokation for vm.SELFDESTRUCT is called while already in another call, so we must not check that we are not in a call here
	if typ == vm.SELFDESTRUCT {
		f.ensureInCall()
		f.callStack.Peek().Suicide = true

		// The next CaptureExit must be ignored, this variable will make the next CaptureExit to be ignored
		f.latestCallStartSuicided = true
		return
	}

	callType := callTypeFromOpCode(typ)
	if callType == pbeth.CallType_UNSPECIFIED {
		panic(fmt.Errorf("unexpected call type, received OpCode %s but only call related opcode (CALL, CREATE, CREATE2, STATIC, DELEGATECALL and CALLCODE) or SELFDESTRUCT is accepted", typ))
	}

	f.callStart("child", callType, from, to, input, gas, value)
}

// CaptureExit is called when EVM exits a scope, even if the scope didn't
// execute any code.
func (f *Firehose) CaptureExit(output []byte, gasUsed uint64, err error) {
	f.callEnd("child", output, gasUsed, err)
}

func (f *Firehose) callStart(source string, callType pbeth.CallType, from common.Address, to common.Address, input []byte, gas uint64, value *big.Int) {
	firehoseDebug("call start source=%s index=%d type=%s input=%s", source, f.callStack.NextIndex(), callType, inputView(input))
	f.ensureInBlockAndInTrx()

	// First to avoid paying the `bytes.Clone` below
	if callType == pbeth.CallType_CREATE {
		// Replicates a previous bug in Firehose instrumentation where create calls input is always
		// set to `nil` (this was done on purpose but it missed the fact that we are missing the
		// actual input data of the constuctor call, if present and invoked).
		input = nil
	}

	call := &pbeth.Call{
		BeginOrdinal: f.blockOrdinal.Next(),
		CallType:     callType,
		Depth:        0,
		Caller:       from.Bytes(),
		Address:      to.Bytes(),
		Input:        bytes.Clone(input),
		Value:        firehoseBigIntFromNative(value),
		GasLimit:     gas,
	}

	if err := f.deferredCallState.MaybePopulateCallAndReset(source, call); err != nil {
		panic(err)
	}

	if source == "root" {
		// Re-do a current existing bug where the BeginOrdinal of the root call is always 0
		call.BeginOrdinal = 0
	}

	f.callStack.Push(call)
}

func (f *Firehose) callEnd(source string, output []byte, gasUsed uint64, err error) {
	firehoseDebug("call end source=%s index=%d output=%s gasUsed=%d err=%s", source, f.callStack.ActiveIndex(), outputView(output), gasUsed, errorView(err))

	if f.latestCallStartSuicided {
		if source != "child" {
			panic(fmt.Errorf("unexpected source for suicided call end, expected child but got %s, suicide are always produced on a 'child' source", source))
		}

		// Geth native tracer does a `CaptureEnter(SELFDESTRUCT, ...)/CaptureExit(...)`, we must skip the `CaptureExit` call
		// in that case because we did not push it on the stack.
		f.latestCallStartSuicided = false
		return
	}

	f.ensureInBlockAndInTrxAndInCall()

	call := f.callStack.Pop()
	call.GasConsumed = gasUsed

	// For create call, we do not save the returned value which is the actual contract's code
	if call.CallType != pbeth.CallType_CREATE {
		call.ReturnData = bytes.Clone(output)
	}

	// At this point, `call.ExecutedCode`` is tied to `EVMInterpreter#Run` execution (in `core/vm/interpreter.go`)
	// and is `true` if the run/loop of the interpreter executed.
	//
	// This means that if `false` the interpreter did not run at all and we would had emitted a
	// `account_without_code` event in the old Firehose patch which you have set `call.ExecutecCode`
	// to false
	//
	// For precompiled address however, interpreter does not run so determine  there was a bug in Firehose instrumentation where we would
	if call.ExecutedCode || f.isPrecompiledAddr(common.BytesToAddress(call.Address)) {
		// In this case, we are sure that some code executed. This translates in the old Firehose instrumentation
		// that it would have **never** emitted an `account_without_code`.
		//
		// When no `account_without_code` was executed in the previous Firehose instrumentation,
		// the `call.ExecutedCode` defaulted to the condition below
		call.ExecutedCode = call.CallType != pbeth.CallType_CREATE && len(call.Input) > 0
	} else {
		// In all other cases, we are sure that no code executed. This translates in the old Firehose instrumentation
		// that it would have emitted an `account_without_code` and it would have then forced set the `call.ExecutedCode`
		// to `false`.
		call.ExecutedCode = false
	}

	if err != nil {
		call.FailureReason = err.Error()
		call.StatusFailed = true

		// We also treat ErrInsufficientBalance and ErrDepth as reverted in Firehose model
		// because they do not cost any gas.
		call.StatusReverted = errors.Is(err, vm.ErrExecutionReverted) || errors.Is(err, vm.ErrInsufficientBalance) || errors.Is(err, vm.ErrDepth)
	}

	call.EndOrdinal = f.blockOrdinal.Next()

	f.transaction.Calls = append(f.transaction.Calls, call)
}

// CaptureKeccakPreimage is called during the KECCAK256 opcode.
func (f *Firehose) CaptureKeccakPreimage(hash common.Hash, data []byte) {
	f.ensureInBlockAndInTrxAndInCall()

	activeCall := f.callStack.Peek()
	if activeCall.KeccakPreimages == nil {
		activeCall.KeccakPreimages = make(map[string]string)
	}

	activeCall.KeccakPreimages[hex.EncodeToString(hash.Bytes())] = hex.EncodeToString(data)
}

func (f *Firehose) OnGenesisBlock(b *types.Block, alloc core.GenesisAlloc) {
	// FIXME: Re-implement by actualling callin `OnBlockStart/OnTrxStart/CaptureStart` etc.
	block := &pbeth.Block{
		Hash:   b.Hash().Bytes(),
		Number: b.Number().Uint64(),
		Header: newBlockHeaderFromChainBlock(b, firehoseBigIntFromNative(b.Difficulty())),
		TransactionTraces: []*pbeth.TransactionTrace{
			{
				BeginOrdinal: f.blockOrdinal.Next(),
				Hash:         emptyHash,
				From:         emptyAddress,
				To:           emptyAddress,
				Receipt: &pbeth.TransactionReceipt{
					StateRoot: b.Root().Bytes(),
					LogsBloom: types.Bloom{}.Bytes(),
				},
				Status: pbeth.TransactionTraceStatus_SUCCEEDED,
				Calls: []*pbeth.Call{
					{
						// It seems we never properly set the BeginOrdinal/EndOrdinal of the root call of the genesis block
						BeginOrdinal: 0,
						EndOrdinal:   0,

						CallType: pbeth.CallType_CALL,
						Caller:   emptyAddress,
						Address:  emptyAddress,
						Index:    1,
					},
				},
			},
		},
		Size: uint64(b.Size()),
		Ver:  3,
	}

	rootTrx := block.TransactionTraces[0]
	rootCall := rootTrx.Calls[0]

	sortedAddrs := make([]common.Address, len(alloc))
	i := 0
	for addr := range alloc {
		sortedAddrs[i] = addr
		i++
	}

	sort.Slice(sortedAddrs, func(i, j int) bool {
		return bytes.Compare(sortedAddrs[i][:], sortedAddrs[j][:]) <= -1
	})

	for _, addr := range sortedKeys(alloc) {
		account := alloc[addr]

		rootCall.AccountCreations = append(rootCall.AccountCreations, &pbeth.AccountCreation{
			Account: addr.Bytes(),
			Ordinal: f.blockOrdinal.Next(),
		})

		rootCall.BalanceChanges = append(rootCall.BalanceChanges, &pbeth.BalanceChange{
			Address:  addr.Bytes(),
			NewValue: firehoseBigIntFromNative(account.Balance),
			Reason:   pbeth.BalanceChange_REASON_GENESIS_BALANCE,
			Ordinal:  f.blockOrdinal.Next(),
		})

		if len(account.Code) > 0 {
			rootCall.CodeChanges = append(rootCall.CodeChanges, &pbeth.CodeChange{
				Address: addr.Bytes(),
				NewCode: account.Code,
				NewHash: crypto.Keccak256(account.Code),
				Ordinal: f.blockOrdinal.Next(),
			})
		}

		if account.Nonce > 0 {
			rootCall.NonceChanges = append(rootCall.NonceChanges, &pbeth.NonceChange{
				Address:  addr.Bytes(),
				OldValue: 0,
				NewValue: account.Nonce,
				Ordinal:  f.blockOrdinal.Next(),
			})
		}

		// This is bad, we were not sorting the storage changes before! Not a big deal,
		// just make it harder for verifiability of previous blocks, they are now sorted.
		for _, key := range sortedKeys(account.Storage) {
			rootCall.StorageChanges = append(rootCall.StorageChanges, &pbeth.StorageChange{
				Address:  addr.Bytes(),
				Key:      key.Bytes(),
				NewValue: account.Storage[key].Bytes(),
				Ordinal:  f.blockOrdinal.Next(),
			})
		}
	}

	// Ordering matters here, we must set the end ordinal of the call before the transaction
	rootTrx.EndOrdinal = f.blockOrdinal.Next()

	f.printBlockToFirehose(block)
	f.resetBlock()
}

type bytesGetter interface {
	comparable
	Bytes() []byte
}

func sortedKeys[K bytesGetter, V any](m map[K]V) []K {
	keys := maps.Keys(m)
	slices.SortFunc(keys, func(i, j K) bool {
		return bytes.Compare(i.Bytes(), j.Bytes()) == -1
	})

	return keys
}

func (f *Firehose) OnBalanceChange(a common.Address, prev, new *big.Int, reason state.BalanceChangeReason) {
	if reason == state.BalanceChangeUnspecified {
		// We ignore those, if they are mislabelled, too bad so particular attention needs to be ported to this
		return
	}

	f.ensureInBlockOrTrx()

	change := &pbeth.BalanceChange{
		Ordinal:  f.blockOrdinal.Next(),
		Address:  a.Bytes(),
		OldValue: firehoseBigIntFromNative(prev),
		NewValue: firehoseBigIntFromNative(new),
		Reason:   balanceChangeReasonFromChain(reason),
	}

	if change.Reason == pbeth.BalanceChange_REASON_UNKNOWN {
		panic(fmt.Errorf("unknown balance change reason %s from code %d not accepted here", change.Reason, reason))
	}

	if f.inTransaction.Load() {
		activeCall := f.callStack.Peek()

		// There is an initial transfer happening will the call is not yet started, we track it manually
		if activeCall == nil {
			f.deferredCallState.balanceChanges = append(f.deferredCallState.balanceChanges, change)
			return
		}

		activeCall.BalanceChanges = append(activeCall.BalanceChanges, change)
	} else {
		f.block.BalanceChanges = append(f.block.BalanceChanges, change)
	}
}

func (f *Firehose) OnNonceChange(a common.Address, prev, new uint64) {
	f.ensureInBlockAndInTrx()

	activeCall := f.callStack.Peek()
	change := &pbeth.NonceChange{
		Address:  a.Bytes(),
		OldValue: prev,
		NewValue: new,
		Ordinal:  f.blockOrdinal.Next(),
	}

	// There is an initial nonce change happening when the call is not yet started, we track it manually
	if activeCall == nil {
		f.deferredCallState.nonceChanges = append(f.deferredCallState.nonceChanges, change)
		return
	}

	activeCall.NonceChanges = append(activeCall.NonceChanges, change)
}

func (f *Firehose) OnCodeChange(a common.Address, prevCodeHash common.Hash, prev []byte, codeHash common.Hash, code []byte) {
	f.ensureInBlockOrTrx()

	change := &pbeth.CodeChange{
		Address: a.Bytes(),
		OldHash: prevCodeHash.Bytes(),
		OldCode: prev,
		NewHash: codeHash.Bytes(),
		NewCode: code,
		Ordinal: f.blockOrdinal.Next(),
	}

	if f.inTransaction.Load() {
		activeCall := f.callStack.Peek()
		if activeCall == nil {
			f.panicNotInState("caller expected to be in call state but we were not, this is a bug")
		}

		activeCall.CodeChanges = append(activeCall.CodeChanges, change)
	} else {
		f.block.CodeChanges = append(f.block.CodeChanges, change)
	}
}

func (f *Firehose) OnStorageChange(a common.Address, k, prev, new common.Hash) {
	f.ensureInBlockAndInTrxAndInCall()

	activeCall := f.callStack.Peek()
	activeCall.StorageChanges = append(activeCall.StorageChanges, &pbeth.StorageChange{
		Address:  a.Bytes(),
		Key:      k.Bytes(),
		OldValue: prev.Bytes(),
		NewValue: new.Bytes(),
		Ordinal:  f.blockOrdinal.Next(),
	})
}

func (f *Firehose) OnLog(l *types.Log) {
	f.ensureInBlockAndInTrxAndInCall()

	topics := make([][]byte, len(l.Topics))
	for i, topic := range l.Topics {
		topics[i] = topic.Bytes()
	}

	activeCall := f.callStack.Peek()
	activeCall.Logs = append(activeCall.Logs, &pbeth.Log{
		Address:    l.Address.Bytes(),
		Topics:     topics,
		Data:       l.Data,
		Index:      f.transactionLogIndex,
		BlockIndex: uint32(l.Index),
		Ordinal:    f.blockOrdinal.Next(),
	})

	f.transactionLogIndex++
	f.blockLogIndex++
}

func (f *Firehose) OnNewAccount(a common.Address) {
	f.ensureInBlockAndInTrxAndInCall()

	if f.isPrecompiledAddr(a) {
		return
	}

	activeCall := f.callStack.Peek()
	activeCall.AccountCreations = append(activeCall.AccountCreations, &pbeth.AccountCreation{
		Account: a.Bytes(),
		Ordinal: f.blockOrdinal.Next(),
	})
}

func (f *Firehose) OnGasConsumed(gas, amount uint64) {
	f.ensureInBlockAndInTrx()

	if amount == 0 {
		return
	}

	firehoseTrace("gas consumed before=%d after=%d", gas, gas-amount)

	activeCall := f.callStack.Peek()
	change := &pbeth.GasChange{
		OldValue: gas,
		NewValue: gas - amount,
		Ordinal:  f.blockOrdinal.Next(),
	}

	// There is an initial gas consumption happening will the call is not yet started, we track it manually
	if activeCall == nil {
		f.deferredCallState.gasChanges = append(f.deferredCallState.gasChanges, change)
		return
	}

	activeCall.GasChanges = append(activeCall.GasChanges, change)
}

func (f *Firehose) ensureInBlock() {
	if !f.inBlock.Load() {
		f.panicNotInState("caller expected to be in block state but we were not, this is a bug")
	}
}

func (f *Firehose) ensureNotInBlock() {
	if f.inBlock.Load() {
		f.panicNotInState("caller expected to not be in block state but we were, this is a bug")
	}
}

func (f *Firehose) ensureInBlockAndInTrx() {
	f.ensureInBlock()

	if !f.inTransaction.Load() {
		f.panicNotInState("caller expected to be in transaction state but we were not, this is a bug")
	}
}

func (f *Firehose) ensureInBlockAndNotInTrx() {
	f.ensureInBlock()

	if f.inTransaction.Load() {
		f.panicNotInState("caller expected to not be in transaction state but we were, this is a bug")
	}
}

func (f *Firehose) ensureInBlockAndNotInTrxAndNotInCall() {
	f.ensureInBlock()

	if f.inTransaction.Load() {
		f.panicNotInState("caller expected to not be in transaction state but we were, this is a bug")
	}

	if f.callStack.HasActiveCall() {
		f.panicNotInState("caller expected to not be in call state but we were, this is a bug")
	}
}

func (f *Firehose) ensureInBlockOrTrx() {
	if !f.inTransaction.Load() && !f.inBlock.Load() {
		f.panicNotInState("caller expected to be in either block or  transaction state but we were not, this is a bug")
	}
}

func (f *Firehose) ensureInBlockAndInTrxAndInCall() {
	if !f.inTransaction.Load() || !f.inBlock.Load() {
		f.panicNotInState("caller expected to be in block and in transaction but we were not, this is a bug")
	}

	if !f.callStack.HasActiveCall() {
		f.panicNotInState("caller expected to be in call state but we were not, this is a bug")
	}
}

func (f *Firehose) ensureInCall() {
	if !f.inBlock.Load() {
		f.panicNotInState("caller expected to be in call state but we were not, this is a bug")
	}
}

func (f *Firehose) panicNotInState(msg string) string {
	panic(fmt.Errorf("%s (inBlock=%t, inTransaction=%t, inCall=%t)", msg, f.inBlock.Load(), f.inTransaction.Load(), f.callStack.HasActiveCall()))
}

// printToFirehose is an easy way to print to Firehose format, it essentially
// adds the "FIRE" prefix to the input and joins the input with spaces as well
// as adding a newline at the end.
//
// It flushes this through [flushToFirehose] to the `os.Stdout` writer.
func (f *Firehose) printBlockToFirehose(block *pbeth.Block) {
	marshalled, err := proto.Marshal(block)
	if err != nil {
		panic(fmt.Errorf("failed to marshal block: %w", err))
	}

	f.outputBuffer.Reset()

	// Final space is important!
	f.outputBuffer.WriteString(fmt.Sprintf("FIRE BLOCK %d %s ", block.Number, hex.EncodeToString(block.Hash)))

	encoder := base64.NewEncoder(base64.StdEncoding, f.outputBuffer)
	if _, err = encoder.Write(marshalled); err != nil {
		panic(fmt.Errorf("write to encoder should have been infaillible: %w", err))
	}

	if err := encoder.Close(); err != nil {
		panic(fmt.Errorf("closing encoder should have been infaillible: %w", err))
	}

	f.outputBuffer.WriteString("\n")

	flushToFirehose(f.outputBuffer.Bytes(), os.Stdout)
}

// printToFirehose is an easy way to print to Firehose format, it essentially
// adds the "FIRE" prefix to the input and joins the input with spaces as well
// as adding a newline at the end.
//
// It flushes this through [flushToFirehose] to the `os.Stdout` writer.
func printToFirehose(input ...string) {
	flushToFirehose([]byte("FIRE "+strings.Join(input, " ")+"\n"), os.Stdout)
}

// flushToFirehose sends data to Firehose via `io.Writter` checking for errors
// and retrying if necessary.
//
// If error is still present after 10 retries, prints an error message to `writer`
// as well as writing file `/tmp/firehose_writer_failed_print.log` with the same
// error message.
func flushToFirehose(in []byte, writer io.Writer) {
	var written int
	var err error
	loops := 10
	for i := 0; i < loops; i++ {
		written, err = writer.Write(in)

		if len(in) == written {
			return
		}

		in = in[written:]
		if i == loops-1 {
			break
		}
	}

	errstr := fmt.Sprintf("\nFIREHOSE FAILED WRITING %dx: %s\n", loops, err)
	os.WriteFile("/tmp/firehose_writer_failed_print.log", []byte(errstr), 0644)
	fmt.Fprint(writer, errstr)
}

// FIXME: Bring back Firehose block header test ensuring we are not missing any fields!
func newBlockHeaderFromChainBlock(b *types.Block, td *pbeth.BigInt) *pbeth.BlockHeader {
	var withdrawalsHashBytes []byte
	if hash := b.Header().WithdrawalsHash; hash != nil {
		withdrawalsHashBytes = hash.Bytes()
	}

	return &pbeth.BlockHeader{
		Hash:             b.Hash().Bytes(),
		Number:           b.NumberU64(),
		ParentHash:       b.ParentHash().Bytes(),
		UncleHash:        b.UncleHash().Bytes(),
		Coinbase:         b.Coinbase().Bytes(),
		StateRoot:        b.Root().Bytes(),
		TransactionsRoot: b.TxHash().Bytes(),
		ReceiptRoot:      b.ReceiptHash().Bytes(),
		LogsBloom:        b.Bloom().Bytes(),
		Difficulty:       firehoseBigIntFromNative(b.Difficulty()),
		TotalDifficulty:  td,
		GasLimit:         b.GasLimit(),
		GasUsed:          b.GasUsed(),
		Timestamp:        timestamppb.New(time.Unix(int64(b.Time()), 0)),
		ExtraData:        b.Extra(),
		MixHash:          b.MixDigest().Bytes(),
		Nonce:            b.Nonce(),
		BaseFeePerGas:    firehoseBigIntFromNative(b.BaseFee()),
		WithdrawalsRoot:  withdrawalsHashBytes,
	}
}

// FIXME: Bring back Firehose test that ensures no new tx type are missed
func transactionTypeFromChainTxType(txType uint8) pbeth.TransactionTrace_Type {
	switch txType {
	case types.AccessListTxType:
		return pbeth.TransactionTrace_TRX_TYPE_ACCESS_LIST
	case types.DynamicFeeTxType:
		return pbeth.TransactionTrace_TRX_TYPE_DYNAMIC_FEE
	case types.LegacyTxType:
		return pbeth.TransactionTrace_TRX_TYPE_LEGACY
	// Add when enabled in a fork
	// case types.BlobTxType:
	// 	return pbeth.TransactionTrace_TRX_TYPE_BLOB
	default:
		panic(fmt.Errorf("unknown transaction type %d", txType))
	}
}

func transactionStatusFromChainTxReceipt(txStatus uint64) pbeth.TransactionTraceStatus {
	switch txStatus {
	case types.ReceiptStatusSuccessful:
		return pbeth.TransactionTraceStatus_SUCCEEDED
	case types.ReceiptStatusFailed:
		return pbeth.TransactionTraceStatus_FAILED
	default:
		panic(fmt.Errorf("unknown transaction status %d", txStatus))
	}
}

func rootCallType(create bool) pbeth.CallType {
	if create {
		return pbeth.CallType_CREATE
	}

	return pbeth.CallType_CALL
}

func callTypeFromOpCode(typ vm.OpCode) pbeth.CallType {
	switch typ {
	case vm.CALL:
		return pbeth.CallType_CALL
	case vm.STATICCALL:
		return pbeth.CallType_STATIC
	case vm.DELEGATECALL:
		return pbeth.CallType_DELEGATE
	case vm.CREATE, vm.CREATE2:
		return pbeth.CallType_CREATE
	case vm.CALLCODE:
		return pbeth.CallType_CALLCODE
	}

	return pbeth.CallType_UNSPECIFIED
}

func newTxReceiptFromChain(receipt *types.Receipt) (out *pbeth.TransactionReceipt) {
	out = &pbeth.TransactionReceipt{
		StateRoot:         receipt.PostState,
		CumulativeGasUsed: receipt.CumulativeGasUsed,
		LogsBloom:         receipt.Bloom[:],
	}

	if len(receipt.Logs) > 0 {
		out.Logs = make([]*pbeth.Log, len(receipt.Logs))
		for i, log := range receipt.Logs {
			out.Logs[i] = &pbeth.Log{
				Address: log.Address.Bytes(),
				Topics: func() [][]byte {
					if len(log.Topics) == 0 {
						return nil
					}

					out := make([][]byte, len(log.Topics))
					for i, topic := range log.Topics {
						out[i] = topic.Bytes()
					}
					return out
				}(),
				Data:       log.Data,
				Index:      uint32(i),
				BlockIndex: uint32(log.Index),

				// FIXME: Fix ordinal for logs in receipt!
				// Ordinal: uint64,
			}
		}
	}

	return out
}

func newAccessListFromChain(accessList types.AccessList) (out []*pbeth.AccessTuple) {
	if len(accessList) == 0 {
		return nil
	}

	out = make([]*pbeth.AccessTuple, len(accessList))
	for i, tuple := range accessList {
		out[i] = &pbeth.AccessTuple{
			Address: tuple.Address.Bytes(),
			StorageKeys: func() [][]byte {
				out := make([][]byte, len(tuple.StorageKeys))
				for i, key := range tuple.StorageKeys {
					out[i] = key.Bytes()
				}
				return out
			}(),
		}
	}

	return
}

func balanceChangeReasonFromChain(reason state.BalanceChangeReason) pbeth.BalanceChange_Reason {
	switch reason {
	case state.BalanceChangeRewardMineUncle:
		return pbeth.BalanceChange_REASON_REWARD_MINE_UNCLE
	case state.BalanceChangeRewardMineBlock:
		return pbeth.BalanceChange_REASON_REWARD_MINE_BLOCK
	case state.BalanceChangeDaoRefundContract:
		return pbeth.BalanceChange_REASON_DAO_REFUND_CONTRACT
	case state.BalanceChangeDaoAdjustBalance:
		return pbeth.BalanceChange_REASON_DAO_ADJUST_BALANCE
	case state.BalanceChangeTransfer:
		return pbeth.BalanceChange_REASON_TRANSFER
	case state.BalanceChangeGenesisBalance:
		return pbeth.BalanceChange_REASON_GENESIS_BALANCE
	case state.BalanceChangeGasBuy:
		return pbeth.BalanceChange_REASON_GAS_BUY
	case state.BalanceChangeRewardTransactionFee:
		return pbeth.BalanceChange_REASON_REWARD_TRANSACTION_FEE
	case state.BalanceChangeGasRefund:
		return pbeth.BalanceChange_REASON_GAS_REFUND
	case state.BalanceChangeTouchAccount:
		return pbeth.BalanceChange_REASON_TOUCH_ACCOUNT
	case state.BalanceChangeSuicideRefund:
		return pbeth.BalanceChange_REASON_SUICIDE_REFUND
	case state.BalanceChangeSuicideWithdraw:
		return pbeth.BalanceChange_REASON_SUICIDE_WITHDRAW
	case state.BalanceChangeBurn:
		return pbeth.BalanceChange_REASON_BURN
	case state.BalanceChangeWithdrawal:
		return pbeth.BalanceChange_REASON_WITHDRAWAL

	case state.BalanceChangeUnspecified:
		return pbeth.BalanceChange_REASON_UNKNOWN
	}

	panic(fmt.Errorf("unknown tracer balance change reason value '%d', check state.BalanceChangeReason so see to which constant it refers to", reason))
}

func maxFeePerGas(tx *types.Transaction) *pbeth.BigInt {
	switch tx.Type() {
	case types.LegacyTxType, types.AccessListTxType:
		return nil

	case types.DynamicFeeTxType, types.BlobTxType:
		return firehoseBigIntFromNative(tx.GasFeeCap())
	}

	panic(errUnhandledTransactionType("maxFeePerGas", tx.Type()))
}

func maxPriorityFeePerGas(tx *types.Transaction) *pbeth.BigInt {
	switch tx.Type() {
	case types.LegacyTxType, types.AccessListTxType:
		return nil

	case types.DynamicFeeTxType, types.BlobTxType:
		return firehoseBigIntFromNative(tx.GasTipCap())
	}

	panic(errUnhandledTransactionType("maxPriorityFeePerGas", tx.Type()))
}

func gasPrice(tx *types.Transaction, baseFee *big.Int) *pbeth.BigInt {
	switch tx.Type() {
	case types.LegacyTxType, types.AccessListTxType:
		return firehoseBigIntFromNative(tx.GasPrice())

	case types.DynamicFeeTxType, types.BlobTxType:
		if baseFee == nil {
			return firehoseBigIntFromNative(tx.GasPrice())
		}

		return firehoseBigIntFromNative(math.BigMin(new(big.Int).Add(tx.GasTipCap(), baseFee), tx.GasFeeCap()))
	}

	panic(errUnhandledTransactionType("gasPrice", tx.Type()))
}

func firehoseDebug(msg string, args ...interface{}) {
	if isFirehoseDebugEnabled {
		fmt.Fprintf(os.Stderr, "[Firehose] "+msg+"\n", args...)
	}
}

func firehoseTrace(msg string, args ...interface{}) {
	if isFirehoseTracerEnabled {
		fmt.Fprintf(os.Stderr, "[Firehose] "+msg+"\n", args...)
	}
}

// Ignore unused, we keep it around for debugging purposes
var _ = firehoseDebugPrintStack

func firehoseDebugPrintStack() {
	if isFirehoseDebugEnabled {
		fmt.Fprintf(os.Stderr, "[Firehose] Stacktrace\n")

		// PrintStack prints to Stderr
		debug.PrintStack()
	}
}

func errUnhandledTransactionType(tag string, value uint8) error {
	return fmt.Errorf("unhandled transaction type's %d for firehose.%s(), carefully review the patch, if this new transaction type add new fields, think about adding them to Firehose Block format, when you see this message, it means something changed in the chain model and great care and thinking most be put here to properly understand the changes and the consequences they bring for the instrumentation", value, tag)
}

type Ordinal struct {
	value uint64
}

// Reset resets the ordinal to zero.
func (o *Ordinal) Reset() {
	o.value = 0
}

// Next gives you the next sequential ordinal value that you should
// use to assign to your exeuction trace (block, transaction, call, etc).
func (o *Ordinal) Next() (out uint64) {
	o.value++

	return o.value
}

type CallStack struct {
	index uint32
	stack []*pbeth.Call
	depth int
}

func NewCallStack() *CallStack {
	return &CallStack{}
}

func (s *CallStack) Reset() {
	s.index = 0
	s.stack = s.stack[:0]
	s.depth = 0
}

func (s *CallStack) HasActiveCall() bool {
	return len(s.stack) > 0
}

// Push a call onto the stack. The `Index` and `ParentIndex` of this call are
// assigned by this method which knowns how to find the parent call and deal with
// it.
func (s *CallStack) Push(call *pbeth.Call) {
	s.index++
	call.Index = s.index

	call.Depth = uint32(s.depth)
	s.depth++

	// If a current call is active, it's the parent of this call
	if parent := s.Peek(); parent != nil {
		call.ParentIndex = parent.Index
	}

	s.stack = append(s.stack, call)
}

func (s *CallStack) ActiveIndex() uint32 {
	if len(s.stack) == 0 {
		return 0
	}

	return s.stack[len(s.stack)-1].Index
}

func (s *CallStack) NextIndex() uint32 {
	return s.index + 1
}

func (s *CallStack) Pop() (out *pbeth.Call) {
	if len(s.stack) == 0 {
		panic(fmt.Errorf("pop from empty call stack"))
	}

	out = s.stack[len(s.stack)-1]
	s.stack = s.stack[:len(s.stack)-1]
	s.depth--

	return
}

// Peek returns the top of the stack without removing it, it's the
// activate call.
func (s *CallStack) Peek() *pbeth.Call {
	if len(s.stack) == 0 {
		return nil
	}

	return s.stack[len(s.stack)-1]
}

// DeferredCallState is a helper struct that can be used to accumulate call's state
// that is recorded before the Call has been started. This happens on the "starting"
// portion of the call/created.
type DeferredCallState struct {
	balanceChanges []*pbeth.BalanceChange
	gasChanges     []*pbeth.GasChange
	nonceChanges   []*pbeth.NonceChange
}

func NewDeferredCallState() *DeferredCallState {
	return &DeferredCallState{}
}

func (d *DeferredCallState) MaybePopulateCallAndReset(source string, call *pbeth.Call) error {
	if d.IsEmpty() {
		return nil
	}

	if source != "root" {
		return fmt.Errorf("unexpected source for deferred call state, expected root but got %s, deferred call's state are always produced on the 'root' call", source)
	}

	// We must happen because it's populated at beginning of the call as well as at the very end
	call.BalanceChanges = append(call.BalanceChanges, d.balanceChanges...)
	call.GasChanges = append(call.GasChanges, d.gasChanges...)
	call.NonceChanges = append(call.NonceChanges, d.nonceChanges...)

	d.Reset()

	return nil
}

func (d *DeferredCallState) IsEmpty() bool {
	return len(d.balanceChanges) == 0 && len(d.gasChanges) == 0 && len(d.nonceChanges) == 0
}

func (d *DeferredCallState) Reset() {
	d.balanceChanges = nil
	d.gasChanges = nil
	d.nonceChanges = nil
}

func errorView(err error) _errorView {
	return _errorView{err}
}

type _errorView struct {
	err error
}

func (e _errorView) String() string {
	if e.err == nil {
		return "<no error>"
	}

	return e.err.Error()
}

type inputView []byte

func (b inputView) String() string {
	if len(b) == 0 {
		return "<empty>"
	}

	if len(b) < 4 {
		return common.Bytes2Hex(b)
	}

	method := b[:4]
	rest := b[4:]

	if len(rest)%32 == 0 {
		return fmt.Sprintf("%s (%d params)", common.Bytes2Hex(method), len(rest)/32)
	}

	// Contract input starts with pre-defined chracters AFAIK, we could show them more nicely

	return fmt.Sprintf("%d bytes", len(rest))
}

type outputView []byte

func (b outputView) String() string {
	if len(b) == 0 {
		return "<empty>"
	}

	return fmt.Sprintf("%d bytes", len(b))
}

type receiptView types.Receipt

func (r *receiptView) String() string {
	if r == nil {
		return "<failed>"
	}

	return fmt.Sprintf("[status=%d, gasUsed=%d, logs=%d]", r.Status, r.GasUsed, len(r.Logs))
}

func emptyBytesToNil(in []byte) []byte {
	if len(in) == 0 {
		return nil
	}

	return in
}

func firehoseBigIntFromNative(in *big.Int) *pbeth.BigInt {
	if in == nil || in.Sign() == 0 {
		return nil
	}

	return &pbeth.BigInt{Bytes: in.Bytes()}
}
