package tracers

import (
	"encoding/json"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
)

type Printer struct {
	eventsChan     chan json.RawMessage
	triggerEvent   chan json.RawMessage
	feed           bool
	eventsBuffer   []json.RawMessage  // added field to buffer events
}

func NewPrinter() *Printer {
	return &Printer{}
}

func NewPrinterWithFeed(bc *core.BlockChain) *Printer {
	p := &Printer{
		eventsChan:   make(chan json.RawMessage, 100),
		triggerEvent: make(chan json.RawMessage, 100),
        feed:         true,
	}
	//TODO: Collecting streaming logs through the event loop and distributing them 
	//triggered by specific events, such as onBlockEnd.
	go p.EventLoop(bc)
	return p
}

// CaptureStart implements the EVMLogger interface to initialize the tracing operation.
func (p *Printer) CaptureStart(from common.Address, to common.Address, create bool, input []byte, gas uint64, value *big.Int) {
	fmt.Printf("CaptureStart: from=%v, to=%v, create=%v, input=%v, gas=%v, value=%v\n", from, to, create, hexutil.Bytes(input), gas, value)
	if p.feed {
		message := map[string]interface{}{
			"event": "CaptureStart",
			"from":  from.Hex(),
			"to":    to.Hex(),
			"create": create,
			"input":  input,
			"gas":    gas,
			"value":  value.String(),
		}

		data, err := json.Marshal(message)
		if err != nil {
			fmt.Printf("Failed to marshal json: %v\n", err)
			return
		}

		p.eventsChan <- json.RawMessage(data)
	}
}

// CaptureEnd is called after the call finishes to finalize the tracing.
func (p *Printer) CaptureEnd(output []byte, gasUsed uint64, err error) {
	fmt.Printf("CaptureEnd: output=%v, gasUsed=%v, err=%v\n", hexutil.Bytes(output), gasUsed, err)

	if p.feed {
		message := map[string]interface{}{
			"event": "CaptureEnd",
			"output": output,
			"gasUsed": gasUsed,
			"error": err.Error(),
		}

		data, err := json.Marshal(message)
		if err != nil {
			fmt.Printf("Failed to marshal json: %v\n", err)
			return
		}

		p.eventsChan <- json.RawMessage(data)
	}
}

// CaptureState implements the EVMLogger interface to trace a single step of VM execution.
func (p *Printer) CaptureState(pc uint64, op vm.OpCode, gas, cost uint64, scope *vm.ScopeContext, rData []byte, depth int, err error) {
	//fmt.Printf("CaptureState: pc=%v, op=%v, gas=%v, cost=%v, scope=%v, rData=%v, depth=%v, err=%v\n", pc, op, gas, cost, scope, rData, depth, err)
/* 	if p.feed {
		message := map[string]interface{}{
			"event": "CaptureState",
			"pc":    pc,
			"op":    op,
			"gas":   gas,
			"cost":  cost,
			"scope": scope,
			"rData": rData,
			"depth": depth,
			"error": err,
		}

		data, err := json.Marshal(message)
		if err != nil {
			fmt.Printf("CaptureState: failed to marshal JSON: %v\n", err)
			return
		}

		p.eventsChan <- data
	} */
}

// CaptureFault implements the EVMLogger interface to trace an execution fault.
func (p *Printer) CaptureFault(pc uint64, op vm.OpCode, gas, cost uint64, _ *vm.ScopeContext, depth int, err error) {
	fmt.Printf("CaptureFault: pc=%v, op=%v, gas=%v, cost=%v, depth=%v, err=%v\n", pc, op, gas, cost, depth, err)

	if p.feed {
		message := map[string]interface{}{
			"event": "CaptureFault",
			"pc":    pc,
			"op":    op.String(),
			"gas":   gas,
			"cost":  cost,
			"depth": depth,
			"error": err.Error(),
		}

		data, err := json.Marshal(message)
		if err != nil {
			fmt.Printf("Failed to marshal json: %v\n", err)
			return
		}

		p.eventsChan <- json.RawMessage(data)
	}
}

// CaptureKeccakPreimage is called during the KECCAK256 opcode.
func (p *Printer) CaptureKeccakPreimage(hash common.Hash, data []byte) {}

// CaptureEnter is called when EVM enters a new scope (via call, create or selfdestruct).
func (p *Printer) CaptureEnter(typ vm.OpCode, from common.Address, to common.Address, input []byte, gas uint64, value *big.Int) {
	fmt.Printf("CaptureEnter: typ=%v, from=%v, to=%v, input=%v, gas=%v, value=%v\n", typ, from, to, hexutil.Bytes(input), gas, value)

	if p.feed {
		message := map[string]interface{}{
			"event": "CaptureEnter",
			"typ":   typ.String(),
			"from":  from.Hex(),
			"to":    to.Hex(),
			"input": input,
			"gas":   gas,
			"value": value.String(),
		}

		data, err := json.Marshal(message)
		if err != nil {
			fmt.Printf("Failed to marshal json: %v\n", err)
			return
		}

		p.eventsChan <- json.RawMessage(data)
	}
}

// CaptureExit is called when EVM exits a scope, even if the scope didn't
// execute any code.
func (p *Printer) CaptureExit(output []byte, gasUsed uint64, err error) {
	fmt.Printf("CaptureExit: output=%v, gasUsed=%v, err=%v\n", hexutil.Bytes(output), gasUsed, err)

	if p.feed {
		message := map[string]interface{}{
			"event": "CaptureExit",
			"output": output,
			"gasUsed": gasUsed,
			"error": err.Error(),
		}

		data, err := json.Marshal(message)
		if err != nil {
			fmt.Printf("Failed to marshal json: %v\n", err)
			return
		}

		p.eventsChan <- json.RawMessage(data)
	}
}

func (p *Printer) CaptureTxStart(env *vm.EVM, tx *types.Transaction) {
	buf, err := json.Marshal(tx)
	if err != nil {
		fmt.Printf("err: %v\n", err)
		return
	}
	fmt.Printf("CaptureTxStart: tx=%s\n", buf)

	if p.feed {
		message := map[string]interface{}{
			"event": "CaptureTxStart",
			"tx":    tx.Hash().Hex(),
		}

		data, err := json.Marshal(message)
		if err != nil {
			fmt.Printf("Failed to marshal json: %v\n", err)
			return
		}

		p.triggerEvent <- json.RawMessage(data)
	}
}

func (p *Printer) CaptureTxEnd(receipt *types.Receipt) {
	buf, err := json.Marshal(receipt)
	if err != nil {
		fmt.Printf("err: %v\n", err)
		return
	}
	fmt.Printf("CaptureTxEnd: receipt=%s\n", buf)

	if p.feed {
		message := map[string]interface{}{
			"event": "CaptureTxEnd",
			"receipt": receipt.TxHash.Hex(),
		}

		data, err := json.Marshal(message)
		if err != nil {
			fmt.Printf("Failed to marshal json: %v\n", err)
			return
		}

		p.triggerEvent <- json.RawMessage(data)
	}
}

func (p *Printer) OnBlockStart(b *types.Block) {
	fmt.Printf("OnBlockStart: b=%v\n", b.NumberU64())

	if p.feed {
		message := map[string]interface{}{
			"event": "OnBlockStart",
			"blockNumber": b.NumberU64(),
		}

		data, err := json.Marshal(message)
		if err != nil {
			fmt.Printf("Failed to marshal json: %v\n", err)
			return
		}

		p.triggerEvent <- json.RawMessage(data)
	}
}

func (p *Printer) OnBlockEnd(td *big.Int, err error) {
	fmt.Printf("OnBlockEnd: td=%v, err=%v\n", td, err)

	if p.feed {
		message := map[string]interface{}{
			"event": "OnBlockEnd",
			"totalDifficulty": td.String(),
			"error": err,
		}

		data, err := json.Marshal(message)
		if err != nil {
			fmt.Printf("Failed to marshal json: %v\n", err)
			return
		}

		p.triggerEvent <- json.RawMessage(data)
	}
}

func (p *Printer) OnGenesisBlock(b *types.Block) {
	fmt.Printf("OnGenesisBlock: b=%v\n", b.NumberU64())

	if p.feed {
		message := map[string]interface{}{
			"event": "OnGenesisBlock",
			"blockNumber": b.NumberU64(),
		}

		data, err := json.Marshal(message)
		if err != nil {
			fmt.Printf("Failed to marshal json: %v\n", err)
			return
		}

		p.eventsChan <- json.RawMessage(data)
	}
}

func (p *Printer) OnBalanceChange(a common.Address, prev, new *big.Int) {
	fmt.Printf("OnBalanceChange: a=%v, prev=%v, new=%v\n", a, prev, new)

	if p.feed {
		message := map[string]interface{}{
			"event": "OnBalanceChange",
			"address": a.Hex(),
			"prevBalance": prev.String(),
			"newBalance": new.String(),
		}

		data, err := json.Marshal(message)
		if err != nil {
			fmt.Printf("Failed to marshal json: %v\n", err)
			return
		}

		p.eventsChan <- json.RawMessage(data)
	}
}

func (p *Printer) OnNonceChange(a common.Address, prev, new uint64) {
	fmt.Printf("OnNonceChange: a=%v, prev=%v, new=%v\n", a, prev, new)

	if p.feed {
		message := map[string]interface{}{
			"event": "OnNonceChange",
			"address": a.Hex(),
			"prevNonce": prev,
			"newNonce": new,
		}

		data, err := json.Marshal(message)
		if err != nil {
			fmt.Printf("Failed to marshal json: %v\n", err)
			return
		}

		p.eventsChan <- json.RawMessage(data)
	}
}

func (p *Printer) OnCodeChange(a common.Address, prevCodeHash common.Hash, prev []byte, codeHash common.Hash, code []byte) {
	fmt.Printf("OnCodeChange: a=%v, prevCodeHash=%v, prev=%v, codeHash=%v, code=%v\n", a, prevCodeHash, hexutil.Bytes(prev), codeHash, code)

	if p.feed {
		message := map[string]interface{}{
			"event": "OnCodeChange",
			"address": a.Hex(),
			"prevCodeHash": prevCodeHash.Hex(),
			"prevCode": string(prev),
			"codeHash": codeHash.Hex(),
			"code": string(code),
		}

		data, err := json.Marshal(message)
		if err != nil {
			fmt.Printf("Failed to marshal json: %v\n", err)
			return
		}

		p.eventsChan <- json.RawMessage(data)
	}
}

func (p *Printer) OnStorageChange(a common.Address, k, prev, new common.Hash) {
	fmt.Printf("OnStorageChange: a=%v, k=%v, prev=%v, new=%v\n", a, k, prev, new)

	if p.feed {
		message := map[string]interface{}{
			"event": "OnStorageChange",
			"address": a.Hex(),
			"key": k.Hex(),
			"prevHash": prev.Hex(),
			"newHash": new.Hex(),
		}

		data, err := json.Marshal(message)
		if err != nil {
			fmt.Printf("Failed to marshal json: %v\n", err)
			return
		}

		p.eventsChan <- json.RawMessage(data)
	}
}

func (p *Printer) OnLog(l *types.Log) {
	buf, err := json.Marshal(l)
	if err != nil {
		fmt.Printf("err: %v\n", err)
		return
	}
	fmt.Printf("OnLog: l=%s\n", buf)

	if p.feed {
		message := map[string]interface{}{
			"event": "OnLog",
			"log": l,
		}

		data, err := json.Marshal(message)
		if err != nil {
			fmt.Printf("Failed to marshal json: %v\n", err)
			return
		}

		p.eventsChan <- json.RawMessage(data)
	}
}

func (p *Printer) OnNewAccount(a common.Address) {
	fmt.Printf("OnNewAccount: a=%v\n", a)

	if p.feed {
		message := map[string]interface{}{
			"event": "OnNewAccount",
			"address": a.Hex(),
		}

		data, err := json.Marshal(message)
		if err != nil {
			fmt.Printf("Failed to marshal json: %v\n", err)
			return
		}

		p.eventsChan <- json.RawMessage(data)
	}
}

func (p *Printer) OnGasConsumed(gas, amount uint64) {
	fmt.Printf("OnGasConsumed: gas=%v, amount=%v\n", gas, amount)

	if p.feed {
		message := map[string]interface{}{
			"event": "OnGasConsumed",
			"gas": gas,
			"amount": amount,
		}

		data, err := json.Marshal(message)
		if err != nil {
			fmt.Printf("Failed to marshal json: %v\n", err)
			return
		}

		p.eventsChan <- json.RawMessage(data)
	}
}

// EventLoop receives data from channels, adds them to Trace,
// and sends Trace when the OnBlockEnd event occurs. This function operates
// in a loop and should typically be run in a separate goroutine.

//TODO: Collecting streaming logs through the event loop and distributing them 
//triggered by specific events, such as onBlockEnd.
func (p *Printer) EventLoop(bc *core.BlockChain) {
	for {
		select {
		case data := <-p.triggerEvent:
			// Append the triggerEvent to the buffer
			p.eventsBuffer = append(p.eventsBuffer, data)
			// Send all buffered events
			bc.TracersEventsSent(p.eventsBuffer)
			// Clear the events buffer
			p.eventsBuffer = []json.RawMessage{}
		case data := <-p.eventsChan:
			// Buffer the events from eventsChan
			p.eventsBuffer = append(p.eventsBuffer, data)
		}
	}
}
