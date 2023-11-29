package live

import (
	"encoding/json"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/eth/tracers/directory"
)

func init() {
	directory.LiveDirectory.Register("printer", newPrinter)
}

type Printer struct{}

func newPrinter() (core.BlockchainLogger, error) {
	return &Printer{}, nil
}

// CaptureStart implements the EVMLogger interface to initialize the tracing operation.
func (p *Printer) CaptureStart(from common.Address, to common.Address, create bool, input []byte, gas uint64, value *big.Int) {
	fmt.Printf("CaptureStart: from=%v, to=%v, create=%v, input=%s, gas=%v, value=%v\n", from, to, create, hexutil.Bytes(input), gas, value)
}

// CaptureEnd is called after the call finishes to finalize the tracing.
func (p *Printer) CaptureEnd(output []byte, gasUsed uint64, err error) {
	fmt.Printf("CaptureEnd: output=%s, gasUsed=%v, err=%v\n", hexutil.Bytes(output), gasUsed, err)
}

// CaptureState implements the EVMLogger interface to trace a single step of VM execution.
func (p *Printer) CaptureState(pc uint64, op vm.OpCode, gas, cost uint64, scope *vm.ScopeContext, rData []byte, depth int, err error) {
	//fmt.Printf("CaptureState: pc=%v, op=%v, gas=%v, cost=%v, scope=%v, rData=%s, depth=%v, err=%v\n", pc, op, gas, cost, scope, hexutil.Bytes(rData), depth, err)
}

// CaptureFault implements the EVMLogger interface to trace an execution fault.
func (p *Printer) CaptureFault(pc uint64, op vm.OpCode, gas, cost uint64, _ *vm.ScopeContext, depth int, err error) {
	fmt.Printf("CaptureFault: pc=%v, op=%v, gas=%v, cost=%v, depth=%v, err=%v\n", pc, op, gas, cost, depth, err)
}

// CaptureKeccakPreimage is called during the KECCAK256 opcode.
func (p *Printer) CaptureKeccakPreimage(hash common.Hash, data []byte) {}

// CaptureEnter is called when EVM enters a new scope (via call, create or selfdestruct).
func (p *Printer) CaptureEnter(typ vm.OpCode, from common.Address, to common.Address, input []byte, gas uint64, value *big.Int) {
	fmt.Printf("CaptureEnter: typ=%v, from=%v, to=%v, input=%s, gas=%v, value=%v\n", typ, from, to, hexutil.Bytes(input), gas, value)
}

// CaptureExit is called when EVM exits a scope, even if the scope didn't
// execute any code.
func (p *Printer) CaptureExit(output []byte, gasUsed uint64, err error) {
	fmt.Printf("CaptureExit: output=%s, gasUsed=%v, err=%v\n", hexutil.Bytes(output), gasUsed, err)
}

func (p *Printer) OnBeaconBlockRootStart(root common.Hash) {}
func (p *Printer) OnBeaconBlockRootEnd()                   {}

func (p *Printer) CaptureTxStart(env *vm.EVM, tx *types.Transaction, from common.Address) {
	buf, err := json.Marshal(tx)
	if err != nil {
		fmt.Printf("err: %v\n", err)
		return
	}
	fmt.Printf("CaptureTxStart: tx=%s\n", buf)

}

func (p *Printer) CaptureTxEnd(receipt *types.Receipt, err error) {
	if err != nil {
		fmt.Printf("CaptureTxEnd err: %v\n", err)
		return
	}
	buf, err := json.Marshal(receipt)
	if err != nil {
		fmt.Printf("err: %v\n", err)
		return
	}
	fmt.Printf("CaptureTxEnd: receipt=%s\n", buf)
}

func (p *Printer) OnBlockStart(b *types.Block, td *big.Int, finalized, safe *types.Header) {
	if finalized != nil && safe != nil {
		fmt.Printf("OnBlockStart: b=%v, td=%v, finalized=%v, safe=%v\n", b.NumberU64(), td, finalized.Number.Uint64(), safe.Number.Uint64())
	} else {
		fmt.Printf("OnBlockStart: b=%v, td=%v\n", b.NumberU64(), td)
	}
}

func (p *Printer) OnBlockEnd(err error) {
	fmt.Printf("OnBlockEnd: err=%v\n", err)
}

func (p *Printer) OnGenesisBlock(b *types.Block, alloc core.GenesisAlloc) {
	fmt.Printf("OnGenesisBlock: b=%v, allocLength=%d\n", b.NumberU64(), len(alloc))
}

func (p *Printer) OnBalanceChange(a common.Address, prev, new *big.Int, reason state.BalanceChangeReason) {
	fmt.Printf("OnBalanceChange: a=%v, prev=%v, new=%v\n", a, prev, new)
}

func (p *Printer) OnNonceChange(a common.Address, prev, new uint64) {
	fmt.Printf("OnNonceChange: a=%v, prev=%v, new=%v\n", a, prev, new)
}

func (p *Printer) OnCodeChange(a common.Address, prevCodeHash common.Hash, prev []byte, codeHash common.Hash, code []byte) {
	fmt.Printf("OnCodeChange: a=%v, prevCodeHash=%v, prev=%s, codeHash=%v, code=%s\n", a, prevCodeHash, hexutil.Bytes(prev), codeHash, hexutil.Bytes(code))
}

func (p *Printer) OnStorageChange(a common.Address, k, prev, new common.Hash) {
	fmt.Printf("OnStorageChange: a=%v, k=%v, prev=%v, new=%v\n", a, k, prev, new)
}

func (p *Printer) OnLog(l *types.Log) {
	buf, err := json.Marshal(l)
	if err != nil {
		fmt.Printf("err: %v\n", err)
		return
	}
	fmt.Printf("OnLog: l=%s\n", buf)
}

func (p *Printer) OnNewAccount(a common.Address) {
	fmt.Printf("OnNewAccount: a=%v\n", a)
}

func (p *Printer) OnGasChange(old, new uint64, reason vm.GasChangeReason) {
	fmt.Printf("OnGasChange: old=%v, new=%v, diff=%v\n", old, new, new-old)
}
