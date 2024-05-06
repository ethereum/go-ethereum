---
title: Live tracing
description: Continuous tracing of the blockchain
---

Geth v1.14.0 introduces a new optional feature, allowing users to stream (a subset of) all observable blockchain data in real-time. By writing some Go code you can develop a data indexing solution which will receive events from Geth as it is syncing and processing blocks. You may find the full list of events in the source code [here](https://github.com/ethereum/go-ethereum/blob/master/core/tracing/hooks.go), but below is a summary:

- Initialization: receives chain configuration including hard forks and chain ID. Also receives the genesis block.
- Block processing: receives the block which geth will process next and any error encountered during processing.
- Transaction processing: receives the transaction which geth will process next and the receipt post-execution.
- EVM:
  - Call frame level events
  - Opcode level events
  - Logs
  - Gas changes
    - For more transparency into gas changes we have assigned a reason to each gas change.
- State modifications: receives any changes to the accounts.
  - Balance changes come with a reason for more transparency into the change.

As this is a real-time stream, the data indexing solution must be able to handle chain reorgs. Upon receiving `OnBlock` events, it should check the chain of hashes it has already processed and unroll internal state that will be invalidated by the reorg.

<Note>A live tracer can impact the performance of your node as it is run synchronously within the sync process. It is better to keep the tracer code minimal and only with the purpose of getting raw data out and doing heavy post-processing of data in a later stage.</Note>

## Implementing a live tracer

The process is very similar to implementing a [custom native tracer](/docs/developers/evm-tracing/custom-tracer). These are the main differences:

- Custom native tracers are invoked through the API and will be instantiated for each request. Live tracers are instantiated once on startup and used throughout the lifetime of Geth.
- Live tracers will receive chain-related events as opposed to custom native tracers.
- The constructor for each of these types has a different signature. Live tracer constructors must return a `*tracing.Hooks` object, while custom native tracers must return a `*tracers.Tracer` object.

Below is a tracer that tracks changes of Ether supply across blocks.

### Set-up

First follow the instructions to [clone and build](/docs/getting-started/installing-geth) Geth from source code.

### Tracer code

Save the following snippet to a file under `eth/tracers/live/`.

```go
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

type SupplyInfo struct {
	Delta       *big.Int `json:"delta"`
	Reward      *big.Int `json:"reward"`
	Withdrawals *big.Int `json:"withdrawals"`
	Burn        *big.Int `json:"burn"`

	// Block info
	Number     uint64      `json:"blockNumber"`
	Hash       common.Hash `json:"hash"`
	ParentHash common.Hash `json:"parentHash"`
}

func newSupplyInfo() SupplyInfo {
  return SupplyInfo{
    Delta:       big.NewInt(0),
    Reward:      big.NewInt(0),
    Withdrawals: big.NewInt(0),
    Burn:        big.NewInt(0),

    Number:     0,
    Hash:       common.Hash{},
    ParentHash: common.Hash{},
  }
}

func (s *SupplyInfo) burn(amount *big.Int) {
	s.Burn.Add(s.Burn, amount)
	s.Delta.Sub(s.Delta, amount)
}

type supplyTxCallstack struct {
	calls []supplyTxCallstack
	burn  *big.Int
}

type Supply struct {
	delta       SupplyInfo
	txCallstack []supplyTxCallstack // Callstack for current transaction
	logger      *log.Logger
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
	loggerOutput := &lumberjack.Logger{
		Filename: filepath.Join(config.Path, "supply.jsonl"),
	}

	if config.MaxSize > 0 {
		loggerOutput.MaxSize = config.MaxSize
	}

	logger := log.New(loggerOutput, "", 0)

	supplyInfo := newSupplyInfo()

	t := &Supply{
		delta:  supplyInfo,
		logger: logger,
	}
	return &tracing.Hooks{
		OnBlockStart:    t.OnBlockStart,
		OnBlockEnd:      t.OnBlockEnd,
		OnGenesisBlock:  t.OnGenesisBlock,
		OnTxStart:       t.OnTxStart,
		OnBalanceChange: t.OnBalanceChange,
		OnEnter:         t.OnEnter,
		OnExit:          t.OnExit,
	}, nil
}

func (s *Supply) resetDelta() {
	s.delta = newSupplyInfo()
}

func (s *Supply) OnBlockStart(ev tracing.BlockEvent) {
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

func (s *Supply) OnBlockEnd(err error) {
	out, _ := json.Marshal(s.delta)
	s.logger.Println(string(out))
}

func (s *Supply) OnGenesisBlock(b *types.Block, alloc types.GenesisAlloc) {
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

func (s *Supply) OnBalanceChange(a common.Address, prevBalance, newBalance *big.Int, reason tracing.BalanceChangeReason) {
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

func (s *Supply) OnTxStart(vm *tracing.VMContext, tx *types.Transaction, from common.Address) {
	s.txCallstack = make([]supplyTxCallstack, 0, 1)
}

// internalTxsHandler handles internal transactions burned amount
func (s *Supply) internalTxsHandler(call *supplyTxCallstack) {
	// Handle Burned amount
	if call.burn != nil {
		s.delta.burn(call.burn)
	}

	if len(call.calls) > 0 {
		// Recursivelly handle internal calls
		for _, call := range call.calls {
			callCopy := call
			s.interalTxsHandler(&callCopy)
		}
	}
}

func (s *Supply) OnEnter(depth int, typ byte, from common.Address, to common.Address, input []byte, gas uint64, value *big.Int) {
	call := supplyTxCallstack{
		calls: make([]supplyTxCallstack, 0),
	}

	// This is a special case of burned amount which has to be handled here
	// which happens when type == selfdestruct and from == to.
	if vm.OpCode(typ) == vm.SELFDESTRUCT && from == to && value.Cmp(common.Big0) == 1 {
		call.burn = value
	}

	// Append call to the callstack, so we can fill the details in OnExit
	s.txCallstack = append(s.txCallstack, call)
}

func (s *Supply) OnExit(depth int, output []byte, gasUsed uint64, err error, reverted bool) {
	if depth == 0 {
		// No need to handle Burned amount if transaction is reverted
		if !reverted {
			s.interalTxsHandler(&s.txCallstack[0])
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
```

Note-worthy explanations:

- Tracer is registered in the Live Tracer directory as part of the `init()` function.
- It is possible to configure the tracer, e.g. pass in the path where the logs will be stored.
- Tracers don't have access to Geth's database. They will have to implement their own persistence layer or a way to extract data. In this example, the tracer logs the data to a file.
- Note that we are resetting the delta on every new block, because the same tracer instance will be used all the time.

### Running the tracer

First compile the source by running `make geth`. Then run the following command:

```bash
./build/bin/geth --vmtrace supply --vmtrace.jsonconfig '{"config": "supply-logs"}' [OTHER_GETH_FLAGS]
```

Soon you will see `supply-logs/supply.jsonl` file being populated with lines such as:

```json lines
{"delta":97373601373111356,"reward":0,"withdrawals":466087699000000000,"burn":368714097626888644,"blockNumber":19503066,"hash":"0x6ad7b65b1ba0de044c490df739ea1e6605cbcae3685dcb69cca9afeb4edeb86b","parentHash":"0x8e68cc87ea7cef3643955f376aacf02ebfe3ff6ac6a28f30683fbd1da0fa0482"}
{"delta":-78769000248388266,"reward":0,"withdrawals":336059502000000000,"burn":414828502248388266,"blockNumber":19503067,"hash":"0xa17379379ecac8c37358ba26d6ef7de6e059aba18c752177b0b4aeb4d3377888","parentHash":"0x6ad7b65b1ba0de044c490df739ea1e6605cbcae3685dcb69cca9afeb4edeb86b"}
{"delta":201614678898811488,"reward":0,"withdrawals":335502106000000000,"burn":133887427101188512,"blockNumber":19503068,"hash":"0xbfb71586616e4d73ae0e12d9123b39168af71f2b52ab6cf94298cc8619a79b09","parentHash":"0xa17379379ecac8c37358ba26d6ef7de6e059aba18c752177b0b4aeb4d3377888"}
```
