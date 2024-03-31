package live

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"math/big"
	"path/filepath"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/consensus/misc/eip4844"
	"github.com/ethereum/go-ethereum/core/tracing"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/eth/tracers"
	"gopkg.in/natefinch/lumberjack.v2"
)

func init() {
	tracers.LiveDirectory.Register("supply", newSupply)
}

type supplyInfo struct {
	Delta       *big.Int `json:"delta"`
	Reward      *big.Int `json:"reward"`
	Withdrawals *big.Int `json:"withdrawals"`
	Burn        *big.Int `json:"burn"`

	// Block info
	Number     uint64      `json:"blockNumber"`
	Hash       common.Hash `json:"hash"`
	ParentHash common.Hash `json:"parentHash"`
}

func (s *supplyInfo) burn(amount *big.Int) {
	s.Burn.Add(s.Burn, amount)
	s.Delta.Sub(s.Delta, amount)
}

type supplyTxCallstack struct {
	calls []supplyTxCallstack
	burn  *big.Int
}

type supply struct {
	delta            supplyInfo
	txCallstack      []supplyTxCallstack // Callstack for current transaction
	loggerOutputFile *lumberjack.Logger
	logger           *log.Logger
}

type supplyTracerConfig struct {
	Path    string `json:"path"`    // Path to the directory where the tracer logs will be stored
	MaxSize int    `json:"maxSize"` // MaxSize is the maximum size in megabytes of the tracer log file before it gets rotated. It defaults to 100 megabytes.
}

func newSupply(cfg json.RawMessage) (*tracing.Hooks, error) {
	var config supplyTracerConfig
	if cfg != nil {
		if err := json.Unmarshal(cfg, &config); err != nil {
			return nil, fmt.Errorf("failed to parse config: %v", err)
		}
	}

	if config.Path == "" {
		return nil, errors.New("supply tracer output path is required")
	}

	// Store traces in a rotating file
	loggerOutputFile := &lumberjack.Logger{
		Filename: filepath.Join(config.Path, "supply.jsonl"),
	}

	if config.MaxSize > 0 {
		loggerOutputFile.MaxSize = config.MaxSize
	}

	logger := log.New(loggerOutputFile, "", 0)

	supplyInfo := newSupplyInfo()

	t := &supply{
		delta:            supplyInfo,
		loggerOutputFile: loggerOutputFile,
		logger:           logger,
	}
	return &tracing.Hooks{
		OnBlockStart:    t.OnBlockStart,
		OnBlockEnd:      t.OnBlockEnd,
		OnGenesisBlock:  t.OnGenesisBlock,
		OnTxStart:       t.OnTxStart,
		OnBalanceChange: t.OnBalanceChange,
		OnEnter:         t.OnEnter,
		OnExit:          t.OnExit,
		OnTerminate:     t.OnTerminate,
	}, nil
}

func newSupplyInfo() supplyInfo {
	return supplyInfo{
		Delta:       big.NewInt(0),
		Reward:      big.NewInt(0),
		Withdrawals: big.NewInt(0),
		Burn:        big.NewInt(0),

		Number:     0,
		Hash:       common.Hash{},
		ParentHash: common.Hash{},
	}
}

func (s *supply) resetDelta() {
	s.delta = newSupplyInfo()
}

func (s *supply) OnBlockStart(ev tracing.BlockEvent) {
	s.resetDelta()

	s.delta.Number = ev.Block.NumberU64()
	s.delta.Hash = ev.Block.Hash()
	s.delta.ParentHash = ev.Block.ParentHash()

	// Calculate Burn for this block
	if ev.Block.BaseFee() != nil {
		burn := new(big.Int).Mul(new(big.Int).SetUint64(ev.Block.GasUsed()), ev.Block.BaseFee())
		s.delta.burn(burn)
	}
	// Blob burnt gas
	if blobGas := ev.Block.BlobGasUsed(); blobGas != nil && *blobGas > 0 && ev.Block.ExcessBlobGas() != nil {
		var (
			excess  = *ev.Block.ExcessBlobGas()
			baseFee = eip4844.CalcBlobFee(excess)
			burn    = new(big.Int).Mul(new(big.Int).SetUint64(*blobGas), baseFee)
		)
		s.delta.burn(burn)
	}
}

func (s *supply) OnBlockEnd(err error) {
	out, _ := json.Marshal(s.delta)
	s.logger.Println(string(out))
}

func (s *supply) OnGenesisBlock(b *types.Block, alloc types.GenesisAlloc) {
	s.resetDelta()

	s.delta.Number = b.NumberU64()
	s.delta.Hash = b.Hash()
	s.delta.ParentHash = b.ParentHash()

	// Initialize supply with total allocation in genesis block
	for _, account := range alloc {
		s.delta.Delta.Add(s.delta.Delta, account.Balance)
	}

	out, _ := json.Marshal(s.delta)
	s.logger.Println(string(out))
}

func (s *supply) OnBalanceChange(a common.Address, prevBalance, newBalance *big.Int, reason tracing.BalanceChangeReason) {
	diff := new(big.Int).Sub(newBalance, prevBalance)

	// NOTE: don't handle "BalanceIncreaseGenesisBalance" because it is handled in OnGenesisBlock
	switch reason {
	case tracing.BalanceIncreaseRewardMineUncle:
	case tracing.BalanceIncreaseRewardMineBlock:
		s.delta.Reward.Add(s.delta.Reward, diff)
	case tracing.BalanceIncreaseWithdrawal:
		s.delta.Withdrawals.Add(s.delta.Withdrawals, diff)
	case tracing.BalanceDecreaseSelfdestructBurn:
		// BalanceDecreaseSelfdestructBurn is non-reversible as it happens
		// at the end of the transaction.
		s.delta.Burn.Sub(s.delta.Burn, diff)
	default:
		return
	}

	s.delta.Delta.Add(s.delta.Delta, diff)
}

func (s *supply) OnTxStart(vm *tracing.VMContext, tx *types.Transaction, from common.Address) {
	s.txCallstack = make([]supplyTxCallstack, 0, 1)
}

// internalTxsHandler handles internal transactions burned amount
func (s *supply) internalTxsHandler(call *supplyTxCallstack) {
	// Handle Burned amount
	if call.burn != nil {
		s.delta.burn(call.burn)
	}

	if len(call.calls) > 0 {
		// Recursivelly handle internal calls
		for _, call := range call.calls {
			callCopy := call
			s.internalTxsHandler(&callCopy)
		}
	}
}

func (s *supply) OnEnter(depth int, typ byte, from common.Address, to common.Address, input []byte, gas uint64, value *big.Int) {
	call := supplyTxCallstack{
		calls: make([]supplyTxCallstack, 0),
	}

	// This is a special case of burned amount which has to be handled here
	// which happens when type == selfdestruct and from == to.
	if vm.OpCode(typ) == vm.SELFDESTRUCT && from == to && value.Cmp(common.Big0) == 1 {
		call.burn = value
	}

	// Append call to the callstack, so we can fill the details in CaptureExit
	s.txCallstack = append(s.txCallstack, call)
}

func (s *supply) OnExit(depth int, output []byte, gasUsed uint64, err error, reverted bool) {
	if depth == 0 {
		// No need to handle Burned amount if transaction is reverted
		if !reverted {
			s.internalTxsHandler(&s.txCallstack[0])
		}
		return
	}

	size := len(s.txCallstack)
	if size <= 1 {
		return
	}
	// Pop call
	call := s.txCallstack[size-1]
	s.txCallstack = s.txCallstack[:size-1]
	size -= 1

	// In case of a revert, we can drop the call and all its subcalls.
	// Caution, that this has to happen after popping the call from the stack.
	if reverted {
		return
	}
	s.txCallstack[size-1].calls = append(s.txCallstack[size-1].calls, call)
}

func (s *supply) OnTerminate() {
	if err := s.loggerOutputFile.Close(); err != nil {
		log.Printf("failed to close supply tracer log file: %v", err)
	}
}
