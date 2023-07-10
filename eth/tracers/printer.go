package tracers

import (
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
)

type Printer struct{}

func NewPrinter() *Printer {
	return &Printer{}
}

func NewPrinterWithFeed(bc core.BlockChain) *Printer {
    Ptr := &Printer{}
	Ptr.EventLoop(bc)
	return Ptr
}

// CaptureStart implements the EVMLogger interface to initialize the tracing operation.
func (p *Printer) CaptureStart(from common.Address, to common.Address, create bool, input []byte, gas uint64, value *big.Int) {
	fmt.Printf("CaptureStart: from=%v, to=%v, create=%v, input=%v, gas=%v, value=%v\n", from, to, create, input, gas, value)
}

// CaptureEnd is called after the call finishes to finalize the tracing.
func (p *Printer) CaptureEnd(output []byte, gasUsed uint64, err error) {
	fmt.Printf("CaptureEnd: output=%v, gasUsed=%v, err=%v\n", output, gasUsed, err)
}

// CaptureState implements the EVMLogger interface to trace a single step of VM execution.
func (p *Printer) CaptureState(pc uint64, op vm.OpCode, gas, cost uint64, scope *vm.ScopeContext, rData []byte, depth int, err error) {
	//fmt.Printf("CaptureState: pc=%v, op=%v, gas=%v, cost=%v, scope=%v, rData=%v, depth=%v, err=%v\n", pc, op, gas, cost, scope, rData, depth, err)
}

// CaptureFault implements the EVMLogger interface to trace an execution fault.
func (p *Printer) CaptureFault(pc uint64, op vm.OpCode, gas, cost uint64, _ *vm.ScopeContext, depth int, err error) {
	fmt.Printf("CaptureFault: pc=%v, op=%v, gas=%v, cost=%v, depth=%v, err=%v\n", pc, op, gas, cost, depth, err)
}

// CaptureKeccakPreimage is called during the KECCAK256 opcode.
func (p *Printer) CaptureKeccakPreimage(hash common.Hash, data []byte) {}

// CaptureEnter is called when EVM enters a new scope (via call, create or selfdestruct).
func (p *Printer) CaptureEnter(typ vm.OpCode, from common.Address, to common.Address, input []byte, gas uint64, value *big.Int) {
	fmt.Printf("CaptureEnter: typ=%v, from=%v, to=%v, input=%v, gas=%v, value=%v\n", typ, from, to, input, gas, value)
}

// CaptureExit is called when EVM exits a scope, even if the scope didn't
// execute any code.
func (p *Printer) CaptureExit(output []byte, gasUsed uint64, err error) {
	fmt.Printf("CaptureExit: output=%v, gasUsed=%v, err=%v\n", output, gasUsed, err)
}

func (p *Printer) CaptureTxStart(env *vm.EVM, tx *types.Transaction) {
	fmt.Printf("CaptureTxStart: tx=%v\n", tx)

}

func (p *Printer) CaptureTxEnd(receipt *types.Receipt) {
	fmt.Printf("CaptureTxEnd: receipt=%v\n", receipt)
}

func (p *Printer) OnBlockStart(b *types.Block) {
	fmt.Printf("OnBlockStart: b=%v\n", b.NumberU64())
}

func (p *Printer) OnBlockEnd(td *big.Int, err error) {
	fmt.Printf("OnBlockEnd: td=%v, err=%v\n", td, err)
}

func (p *Printer) OnBlockValidationError(block *types.Block, err error) {
	fmt.Printf("OnBlockValidationError: b=%v, err=%v\n", block.NumberU64(), err)
}

func (p *Printer) OnGenesisBlock(b *types.Block) {
	fmt.Printf("OnGenesisBlock: b=%v\n", b.NumberU64())
}

func (p *Printer) OnBalanceChange(a common.Address, prev, new *big.Int) {
	fmt.Printf("OnBalanceChange: a=%v, prev=%v, new=%v\n", a, prev, new)
}

func (p *Printer) OnNonceChange(a common.Address, prev, new uint64) {
	fmt.Printf("OnNonceChange: a=%v, prev=%v, new=%v\n", a, prev, new)
}

func (p *Printer) OnCodeChange(a common.Address, prevCodeHash common.Hash, prev []byte, codeHash common.Hash, code []byte) {
	fmt.Printf("OnCodeChange: a=%v, prevCodeHash=%v, prev=%v, codeHash=%v, code=%v\n", a, prevCodeHash, prev, codeHash, code)
}

func (p *Printer) OnStorageChange(a common.Address, k, prev, new common.Hash) {
	fmt.Printf("OnStorageChange: a=%v, k=%v, prev=%v, new=%v\n", a, k, prev, new)
}

func (p *Printer) OnLog(l *types.Log) {
	fmt.Printf("OnLog: l=%v\n", l)
}

func (p *Printer) OnNewAccount(a common.Address) {
	fmt.Printf("OnNewAccount: a=%v\n", a)
}

func (p *Printer) OnGasConsumed(gas, amount uint64) {
	fmt.Printf("OnGasConsumed: gas=%v, amount=%v\n", gas, amount)
}

// EventLoop receives data from channels, adds them to Trace,
// and sends Trace when the OnBlockEnd event occurs. This function operates
// in a loop and should typically be run in a separate goroutine.
func (p *Printer) EventLoop(bc core.BlockChain) {
	for {
		select {
		//TODO
		case <-bc.quit:
			return

		}
	}
}
