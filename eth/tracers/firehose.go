package tracers

import (
	"bytes"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"io"
	"io/ioutil"
	"math/big"
	"os"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/common/math"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/crypto"
	pbeth "github.com/streamingfast/firehose-ethereum/types/pb/sf/ethereum/type/v2"
	"go.uber.org/atomic"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
)

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
	out = o.value
	o.value++

	return out
}

var _ core.BlockchainLogger = (*Firehose)(nil)

type Firehose struct {
	// Global state
	outputBuffer *bytes.Buffer

	// Block state
	inBlock      *atomic.Bool
	block        *pbeth.Block
	blockBaseFee *big.Int
	blockOrdinal *Ordinal

	// Transaction state
	inTransaction *atomic.Bool
	transaction   *pbeth.TransactionTrace

	// Call state
	inCall *atomic.Bool
	call   *pbeth.Call
}

func NewFirehoseLogger() *Firehose {
	// FIXME: Where should we put our actual INIT line?
	// FIXME: Pickup version from go-ethereum (PR comment)
	printToFirehose("INIT", "2.3", "geth", "1.12.0")

	return &Firehose{
		outputBuffer: bytes.NewBuffer(make([]byte, 0, 100*1024*1024)),

		inBlock:      atomic.NewBool(false),
		blockOrdinal: &Ordinal{},

		inTransaction: atomic.NewBool(false),

		inCall: atomic.NewBool(false),
	}
}

func (f *Firehose) resetBlock() {
	f.inBlock.Store(false)
	f.block = nil
	f.blockBaseFee = nil
	f.blockOrdinal.Reset()
	// f.blockLogIndex = 0
}

func (f *Firehose) resetTransaction() {
	f.inTransaction.Store(false)
	// f.nextCallIndex = 0
	// f.activeCallIndex = "0"
	// f.callIndexStack = &ExtendedStack{}
	// f.callIndexStack.Push(ctx.activeCallIndex)
}

func (f *Firehose) resetCall() {
	f.inCall.Store(false)
}

func (f *Firehose) ensureInBlock() {
	if !f.inBlock.Load() {
		panic("caller expected to be in block state but we were not, this is a bug")
	}
}

func (f *Firehose) ensureNotInBlock() {
	if f.inBlock.Load() {
		panic("caller expected to not be in block state but we were, this is a bug")
	}
}

func (f *Firehose) ensureInBlockAndInTrx() {
	f.ensureInBlock()

	if !f.inTransaction.Load() {
		panic("caller expected to be in transaction state but we were not, this is a bug")
	}
}

func (f *Firehose) ensureInBlockAndNotInTrx() {
	f.ensureInBlock()

	if f.inTransaction.Load() {
		panic("caller expected to not be in transaction state but we were, this is a bug")
	}
}

func (f *Firehose) ensureInBlockOrTrx() {
	if !f.inTransaction.Load() && !f.inBlock.Load() {
		panic("caller expected to be in either block or  transaction state but we were not, this is a bug")
	}
}

// CaptureStart implements the EVMLogger interface to initialize the tracing operation.
func (f *Firehose) CaptureStart(from common.Address, to common.Address, create bool, input []byte, gas uint64, value *big.Int) {
	fmt.Fprintf(os.Stderr, "CaptureStart: from=%v, to=%v, create=%v, input=%s, gas=%v, value=%v\n", from, to, create, hexutil.Bytes(input), gas, value)
}

// CaptureEnd is called after the call finishes to finalize the tracing.
func (f *Firehose) CaptureEnd(output []byte, gasUsed uint64, err error) {
	fmt.Fprintf(os.Stderr, "CaptureEnd: output=%s, gasUsed=%v, err=%v\n", hexutil.Bytes(output), gasUsed, err)
}

// CaptureState implements the EVMLogger interface to trace a single step of VM execution.
func (f *Firehose) CaptureState(pc uint64, op vm.OpCode, gas, cost uint64, scope *vm.ScopeContext, rData []byte, depth int, err error) {
	//fmt.Fprintf(os.Stderr, "CaptureState: pc=%v, op=%v, gas=%v, cost=%v, scope=%v, rData=%v, depth=%v, err=%v\n", pc, op, gas, cost, scope, rData, depth, err)
}

// CaptureFault implements the EVMLogger interface to trace an execution fault.
func (f *Firehose) CaptureFault(pc uint64, op vm.OpCode, gas, cost uint64, _ *vm.ScopeContext, depth int, err error) {
	fmt.Fprintf(os.Stderr, "CaptureFault: pc=%v, op=%v, gas=%v, cost=%v, depth=%v, err=%v\n", pc, op, gas, cost, depth, err)
}

// CaptureKeccakPreimage is called during the KECCAK256 opcode.
func (f *Firehose) CaptureKeccakPreimage(hash common.Hash, data []byte) {}

// CaptureEnter is called when EVM enters a new scope (via call, create or selfdestruct).
func (f *Firehose) CaptureEnter(typ vm.OpCode, from common.Address, to common.Address, input []byte, gas uint64, value *big.Int) {
	fmt.Fprintf(os.Stderr, "CaptureEnter: typ=%v, from=%v, to=%v, input=%s, gas=%v, value=%v\n", typ, from, to, hexutil.Bytes(input), gas, value)
}

// CaptureExit is called when EVM exits a scope, even if the scope didn't
// execute any code.
func (f *Firehose) CaptureExit(output []byte, gasUsed uint64, err error) {
	fmt.Fprintf(os.Stderr, "CaptureExit: output=%s, gasUsed=%v, err=%v\n", hexutil.Bytes(output), gasUsed, err)
}

func (f *Firehose) CaptureTxStart(env *vm.EVM, tx *types.Transaction) {
	f.ensureInBlockAndNotInTrx()
	// TODO: not in call

	f.inTransaction.Store(true)

	signer := types.MakeSigner(env.ChainConfig(), env.Context.BlockNumber, env.Context.Time)

	from, err := types.Sender(signer, tx)
	if err != nil {
		panic(fmt.Errorf("could not recover sender address: %w", err))
	}

	var to common.Address
	if tx.To() == nil {
		to = crypto.CreateAddress(from, env.StateDB.GetNonce(from))
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
		Value:                pbeth.BigIntFromNative(tx.Value()),
		Input:                tx.Data(),
		V:                    pbeth.BigIntFromNative(v).Bytes,
		R:                    pbeth.BigIntFromNative(r).Bytes,
		S:                    pbeth.BigIntFromNative(s).Bytes,
		Type:                 transactionTypeFromChainTxType(tx.Type()),
		AccessList:           newAccessListFromChain(tx.AccessList()),
		MaxFeePerGas:         maxFeePerGas(tx),
		MaxPriorityFeePerGas: maxPriorityFeePerGas(tx),
	}
}

func (f *Firehose) CaptureTxEnd(receipt *types.Receipt) {
	f.ensureInBlockAndInTrx()

	f.transaction.Index = uint32(receipt.TransactionIndex)
	f.transaction.GasUsed = receipt.GasUsed
	f.transaction.Receipt = newTxReceiptFromChain(receipt)

	// FIXME: How are we going to decide about a reverted transaction?
	f.transaction.Status = transactionStatusFromChainTxReceipt(receipt.Status)

	// FIXME: Where do I get return data? Top call return data probably?
	// f.transaction.ReturnData = ???

	f.transaction.EndOrdinal = f.blockOrdinal.Next()

	// Add to block
	f.block.TransactionTraces = append(f.block.TransactionTraces, f.transaction)

	// Reset transaction state
	f.resetTransaction()
}

func (f *Firehose) OnBlockStart(b *types.Block) {
	f.ensureNotInBlock()

	f.inBlock.Store(true)
	f.block = &pbeth.Block{
		Hash:   b.Hash().Bytes(),
		Number: b.Number().Uint64(),
		// Deferred total difficulty (td) to be received `OnBlockEnd`
		Header: newBlockHeaderFromChainBlock(b, nil),
		Size:   b.Size(),
		Ver:    3,
	}

	if f.block.Header.BaseFeePerGas != nil {
		f.blockBaseFee = f.block.Header.BaseFeePerGas.Native()
	}
}

func (f *Firehose) OnBlockEnd(td *big.Int, err error) {
	// OnBlockEnd can be called while a transaction/call is still in progress, so must only assert that we are in a block
	f.ensureInBlock()

	if err == nil {
		// No reset, next step is either OnBlockValidationError (skip and reset) or OnBlockWritten (flush and reset)
		f.block.Header.TotalDifficulty = pbeth.BigIntFromNative(td)
	} else {
		// OnBlockEnd with error means that we could have been in any state
		f.resetBlock()
		f.resetTransaction()
		f.resetCall()
	}
}

func (f *Firehose) OnBlockValidationError(block *types.Block, err error) {
	f.ensureInBlockAndNotInTrx()

	fmt.Fprintf(os.Stderr, "OnBlockValidationError: b=%v, err=%v\n", block.NumberU64(), err)
	f.resetBlock()
}

func (f *Firehose) OnBlockWritten() {
	f.ensureInBlockAndNotInTrx()

	f.printBlockToFirehose(f.block)

	f.resetBlock()
}

func (f *Firehose) OnGenesisBlock(b *types.Block, alloc core.GenesisAlloc) {
	block := &pbeth.Block{
		Hash:   b.Hash().Bytes(),
		Number: b.Number().Uint64(),
		Header: newBlockHeaderFromChainBlock(b, pbeth.BigIntFromNative(b.Difficulty())),
		TransactionTraces: []*pbeth.TransactionTrace{
			{
				BeginOrdinal: f.blockOrdinal.Next(),
				Receipt: &pbeth.TransactionReceipt{
					StateRoot: b.Root().Bytes(),
				},
				Calls: []*pbeth.Call{
					{
						BeginOrdinal: f.blockOrdinal.Next(),
					},
				},
			},
		},
		Size: uint64(b.Size()),
		Ver:  3,
	}

	rootTrx := block.TransactionTraces[0]
	rootCall := rootTrx.Calls[0]

	for addr, account := range alloc {
		rootCall.BalanceChanges = append(rootCall.BalanceChanges, &pbeth.BalanceChange{
			Address:  addr.Bytes(),
			NewValue: pbeth.BigIntFromNative(account.Balance),
			Reason:   pbeth.BalanceChange_REASON_GENESIS_BALANCE,
			Ordinal:  f.blockOrdinal.Next(),
		})

		rootCall.CodeChanges = append(rootCall.CodeChanges, &pbeth.CodeChange{
			Address: addr.Bytes(),
			NewCode: account.Code,
			NewHash: crypto.Keccak256(account.Code),
			Ordinal: f.blockOrdinal.Next(),
		})

		rootCall.NonceChanges = append(rootCall.NonceChanges, &pbeth.NonceChange{
			Address:  addr.Bytes(),
			NewValue: account.Nonce,
			Ordinal:  f.blockOrdinal.Next(),
		})

		for key, value := range account.Storage {
			rootCall.StorageChanges = append(rootCall.StorageChanges, &pbeth.StorageChange{
				Address:  addr.Bytes(),
				Key:      key.Bytes(),
				NewValue: value.Bytes(),
				Ordinal:  f.blockOrdinal.Next(),
			})
		}
	}

	rootCall.EndOrdinal = f.blockOrdinal.Next()
	rootTrx.EndOrdinal = f.blockOrdinal.Next()

	f.printBlockToFirehose(block)

	f.resetBlock()
}

func (f *Firehose) OnBalanceChange(a common.Address, prev, new *big.Int) {
	f.ensureInBlockOrTrx()
	fmt.Fprintf(os.Stderr, "OnBalanceChange: a=%v, prev=%v, new=%v\n", a, prev, new)
}

func (f *Firehose) OnNonceChange(a common.Address, prev, new uint64) {
	fmt.Fprintf(os.Stderr, "OnNonceChange: a=%v, prev=%v, new=%v\n", a, prev, new)
}

func (f *Firehose) OnCodeChange(a common.Address, prevCodeHash common.Hash, prev []byte, codeHash common.Hash, code []byte) {
	fmt.Fprintf(os.Stderr, "OnCodeChange: a=%v, prevCodeHash=%v, prev=%s, codeHash=%v, code=%s\n", a, prevCodeHash, hexutil.Bytes(prev), codeHash, hexutil.Bytes(code))
}

func (f *Firehose) OnStorageChange(a common.Address, k, prev, new common.Hash) {
	fmt.Fprintf(os.Stderr, "OnStorageChange: a=%v, k=%v, prev=%v, new=%v\n", a, k, prev, new)
}

func (f *Firehose) OnLog(l *types.Log) {
	fmt.Fprintf(os.Stderr, "OnLog: l=%v\n", l)
}

func (f *Firehose) OnNewAccount(a common.Address) {
	fmt.Fprintf(os.Stderr, "OnNewAccount: a=%v\n", a)
}

func (f *Firehose) OnGasConsumed(gas, amount uint64) {
	fmt.Fprintf(os.Stderr, "OnGasConsumed: gas=%v, amount=%v\n", gas, amount)
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
	ioutil.WriteFile("/tmp/firehose_writer_failed_print.log", []byte(errstr), 0644)
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
		Difficulty:       pbeth.BigIntFromNative(b.Difficulty()),
		TotalDifficulty:  td,
		GasLimit:         b.GasLimit(),
		GasUsed:          b.GasUsed(),
		Timestamp:        timestamppb.New(time.Unix(int64(b.Time()), 0)),
		ExtraData:        b.Extra(),
		MixHash:          b.MixDigest().Bytes(),
		Nonce:            b.Nonce(),
		BaseFeePerGas:    pbeth.BigIntFromNative(b.BaseFee()),
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

func newTxReceiptFromChain(receipt *types.Receipt) (out *pbeth.TransactionReceipt) {
	out = &pbeth.TransactionReceipt{
		StateRoot:         receipt.PostState,
		CumulativeGasUsed: receipt.CumulativeGasUsed,
		LogsBloom:         receipt.Bloom[:],
	}

	if len(receipt.Logs) > 0 {
		out.Logs = make([]*pbeth.Log, len(receipt.Logs))
		for i, log := range receipt.Logs {
			out.Logs = append(out.Logs, &pbeth.Log{
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
			})
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

func maxFeePerGas(tx *types.Transaction) *pbeth.BigInt {
	switch tx.Type() {
	case types.LegacyTxType, types.AccessListTxType:
		return nil

	case types.DynamicFeeTxType, types.BlobTxType:
		return pbeth.BigIntFromNative(tx.GasFeeCap())
	}

	panic(errUnhandledTransactionType("maxFeePerGas", tx.Type()))
}

func maxPriorityFeePerGas(tx *types.Transaction) *pbeth.BigInt {
	switch tx.Type() {
	case types.LegacyTxType, types.AccessListTxType:
		return nil

	case types.DynamicFeeTxType, types.BlobTxType:
		return pbeth.BigIntFromNative(tx.GasTipCap())
	}

	panic(errUnhandledTransactionType("maxPriorityFeePerGas", tx.Type()))
}

func gasPrice(tx *types.Transaction, baseFee *big.Int) *pbeth.BigInt {
	switch tx.Type() {
	case types.LegacyTxType, types.AccessListTxType:
		return pbeth.BigIntFromNative(tx.GasPrice())

	case types.DynamicFeeTxType, types.BlobTxType:
		if baseFee == nil {
			return pbeth.BigIntFromNative(tx.GasPrice())
		}

		return pbeth.BigIntFromNative(math.BigMin(new(big.Int).Add(tx.GasTipCap(), baseFee), tx.GasFeeCap()))
	}

	panic(errUnhandledTransactionType("gasPrice", tx.Type()))
}

func errUnhandledTransactionType(tag string, value uint8) error {
	return fmt.Errorf("unhandled transaction type's %d for firehose.%s(), carefully review the patch, if this new transaction type add new fields, think about adding them to Firehose Block format, when you see this message, it means something changed in the chain model and great care and thinking most be put here to properly understand the changes and the consequences they bring for the instrumentation", value, tag)
}
