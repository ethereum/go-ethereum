package native

import (
	"encoding/json"
	"errors"
	"math/big"
	"strconv"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/vm"
)

func init() {
	Register("callTracerNative", NewCallTracer)
}

type CallFrame struct {
	Type    string      `json:"type"`
	From    string      `json:"from"`
	To      string      `json:"to,omitempty"`
	Value   string      `json:"value,omitempty"`
	Gas     string      `json:"gas"`
	GasUsed string      `json:"gasUsed"`
	Input   string      `json:"input"`
	Output  string      `json:"output,omitempty"`
	Error   string      `json:"error,omitempty"`
	Calls   []CallFrame `json:"calls,omitempty"`
}

type CallTracer struct {
	callstack []CallFrame
}

func NewCallTracer() Tracer {
	t := &CallTracer{callstack: make([]CallFrame, 1)}
	return t
}

func (t *CallTracer) CaptureStart(env *vm.EVM, from common.Address, to common.Address, create bool, input []byte, gas uint64, value *big.Int) {
	t.callstack[0] = CallFrame{
		Type:  "CALL",
		From:  addrToHex(from),
		To:    addrToHex(to),
		Input: bytesToHex(input),
		Gas:   uintToHex(gas),
		Value: bigToHex(value),
	}
	if create {
		t.callstack[0].Type = "CREATE"
	}
}

func (t *CallTracer) CaptureEnd(output []byte, gasUsed uint64, _ time.Duration, err error) {
	t.callstack[0].Output = bytesToHex(output)
	t.callstack[0].GasUsed = uintToHex(gasUsed)
	if err != nil {
		t.callstack[0].Error = err.Error()
	}
}

func (t *CallTracer) CaptureState(env *vm.EVM, pc uint64, op vm.OpCode, gas, cost uint64, scope *vm.ScopeContext, rData []byte, depth int, err error) {
}

func (t *CallTracer) CaptureFault(env *vm.EVM, pc uint64, op vm.OpCode, gas, cost uint64, _ *vm.ScopeContext, depth int, err error) {
}

func (t *CallTracer) CaptureEnter(typ vm.OpCode, from common.Address, to common.Address, input []byte, gas uint64, value *big.Int) {
	call := CallFrame{
		Type:  typ.String(),
		From:  addrToHex(from),
		To:    addrToHex(to),
		Input: bytesToHex(input),
		Gas:   uintToHex(gas),
		Value: bigToHex(value),
		Calls: make([]CallFrame, 0),
	}
	t.callstack = append(t.callstack, call)
}

func (t *CallTracer) CaptureExit(output []byte, gasUsed uint64, err error) {
	size := len(t.callstack)
	if size <= 1 {
		return
	}
	// pop call
	call := t.callstack[size-1]
	t.callstack = t.callstack[:size-1]
	size -= 1

	call.GasUsed = uintToHex(gasUsed)
	if err == nil {
		call.Output = bytesToHex(output)
	} else {
		call.Error = err.Error()
		if call.Type == "CREATE" || call.Type == "CREATE2" {
			call.To = ""
		}
	}
	t.callstack[size-1].Calls = append(t.callstack[size-1].Calls, call)
}

func (t *CallTracer) GetResult() (json.RawMessage, error) {
	if len(t.callstack) != 1 {
		return nil, errors.New("incorrect number of top-level calls")
	}
	res, err := json.Marshal(t.callstack[0])
	if err != nil {
		return nil, err
	}
	return json.RawMessage(res), nil
}

func bytesToHex(s []byte) string {
	return "0x" + common.Bytes2Hex(s)
}

func bigToHex(n *big.Int) string {
	if n == nil {
		return ""
	}
	return "0x" + n.Text(16)
}

func uintToHex(n uint64) string {
	return "0x" + strconv.FormatUint(n, 16)
}

func addrToHex(a common.Address) string {
	return strings.ToLower(a.Hex())
}
