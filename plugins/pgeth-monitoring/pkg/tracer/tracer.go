package tracer

import (
	"encoding/hex"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/vm"
)

type MonitoringTracer struct {
	Action Action
	Cursor Action
}

func (m *MonitoringTracer) Clear() {
	m.Action = nil
	m.Cursor = nil
}

func (m *MonitoringTracer) CaptureTxStart(gasLimit uint64) {

}

func (m *MonitoringTracer) CaptureTxEnd(restGas uint64) {

}

func (m *MonitoringTracer) CaptureStart(env *vm.EVM, from common.Address, to common.Address, create bool, input []byte, gas uint64, value *big.Int) {
	copyInput := make([]byte, len(input))
	usedValue := big.NewInt(0)
	if value != nil {
		usedValue.Set(value)
	}
	copy(copyInput, input)
	m.Action = &Call{
		CallType:      "initial_call",
		TypeValue:     "call",
		ChildrenValue: []Action{},
		ParentValue:   nil,
		DepthValue:    0,

		ContextValue: common.Address{},
		CodeValue:    common.Address{},

		ForwardedContext: to,
		ForwardedCode:    to,

		From:  from,
		To:    to,
		In:    copyInput,
		InHex: "0x" + hex.EncodeToString(copyInput),
		Value: "0x" + usedValue.Text(16),
	}
	m.Cursor = m.Action
}

func (m *MonitoringTracer) CaptureEnd(output []byte, gasUsed uint64, err error) {
	copyOutput := make([]byte, len(output))
	copy(copyOutput, output)
	m.Cursor.(*Call).Out = copyOutput
	m.Cursor.(*Call).OutHex = "0x" + hex.EncodeToString(copyOutput)
}

func (m *MonitoringTracer) CaptureEnter(typ vm.OpCode, from common.Address, to common.Address, input []byte, gas uint64, value *big.Int) {
	callType := callOpcodeToString(typ)
	ctx, code := parentContextAndCode(m.Cursor)
	forwardedCode := to
	forwardedContext := to
	if callType == "delegatecall" {
		forwardedContext = from
	}
	usedValue := big.NewInt(0)
	if value != nil {
		usedValue.Set(value)
	}
	copyInput := make([]byte, len(input))
	copy(copyInput, input)
	call := &Call{
		CallType:      callType,
		TypeValue:     "call",
		ChildrenValue: []Action{},
		ParentValue:   m.Cursor,
		DepthValue:    m.Cursor.Depth() + 1,

		ForwardedContext: forwardedContext,
		ForwardedCode:    forwardedCode,
		ContextValue:     ctx,
		CodeValue:        code,
		From:             from,
		To:               to,
		In:               copyInput,
		InHex:            "0x" + hex.EncodeToString(copyInput),
		Value:            "0x" + usedValue.Text(16),
	}
	m.Cursor.AddChildren(call)
	m.Cursor = call
}

func (m *MonitoringTracer) CaptureExit(output []byte, gasUsed uint64, err error) {
	copyOutput := make([]byte, len(output))
	copy(copyOutput, output)
	m.Cursor.(*Call).Out = copyOutput
	m.Cursor.(*Call).OutHex = "0x" + hex.EncodeToString(copyOutput)
	m.Cursor = m.Cursor.Parent()
}

func (m *MonitoringTracer) CaptureState(pc uint64, op vm.OpCode, gas, cost uint64, scope *vm.ScopeContext, rData []byte, depth int, err error) {
	if op >= 160 && op <= 164 {
		stack := scope.Stack.Data()
		stackLen := len(stack)
		var offset int64 = 0
		var size int64 = 0
		if stackLen >= 2 {
			offset = stack[stackLen-1].ToBig().Int64()
			size = stack[stackLen-2].ToBig().Int64()
		}
		fetchSize := size
		var data []byte = []byte{}
		if int64(scope.Memory.Len()) < offset {
			fetchSize = 0
			// generate zero array
		} else if int64(scope.Memory.Len()) < offset+size {
			fetchSize -= (offset + size) - int64(scope.Memory.Len())
		}

		if fetchSize > 0 {
			data = scope.Memory.GetCopy(offset, fetchSize)
		}

		if fetchSize < size {
			data = addZeros(data, size-fetchSize)
		}

		topics := []common.Hash{}
		for idx := 0; idx < int(op-160); idx++ {
			if stackLen-3-idx >= 0 {
				topics = append(topics, stack[stackLen-3-idx].Bytes32())
			}
		}

		ctx, code := parentContextAndCode(m.Cursor)

		m.Cursor.AddChildren(&Event{
			LogType:   fmt.Sprintf("log%d", op-160),
			TypeValue: "event",
			Data:      data,
			DataHex:   "0x" + hex.EncodeToString(data),
			Topics:    topics,
			From:      scope.Contract.Address(),

			ContextValue: ctx,
			CodeValue:    code,
			ParentValue:  m.Cursor,
			DepthValue:   m.Cursor.Depth() + 1,
		})
	}
	if op == 253 {
		errorType := "revert"
		data := []byte{}
		stack := scope.Stack.Data()
		stackLen := len(stack)
		var offset int64 = 0
		var size int64 = 0
		if stackLen >= 2 {
			offset = stack[stackLen-1].ToBig().Int64()
			size = stack[stackLen-2].ToBig().Int64()
		}
		fetchSize := size
		if int64(scope.Memory.Len()) < offset {
			fetchSize = 0
			// generate zero array
		} else if int64(scope.Memory.Len()) < offset+size {
			fetchSize -= (offset + size) - int64(scope.Memory.Len())
		}

		if fetchSize > 0 {
			data = scope.Memory.GetCopy(offset, fetchSize)
		}

		if fetchSize < size {
			data = addZeros(data, size-fetchSize)
		}

		ctx, code := parentContextAndCode(m.Cursor)

		m.Cursor.AddChildren(&Revert{
			ErrorType:    errorType,
			TypeValue:    "revert",
			Data:         data,
			DataHex:      "0x" + hex.EncodeToString(data),
			From:         scope.Contract.Address(),
			ContextValue: ctx,
			CodeValue:    code,
			ParentValue:  m.Cursor,
			DepthValue:   m.Cursor.Depth() + 1,
		})
	}
}

func (m *MonitoringTracer) CaptureFault(pc uint64, op vm.OpCode, gas, cost uint64, scope *vm.ScopeContext, depth int, err error) {
	if op != 253 {
		ctx, code := parentContextAndCode(m.Cursor)
		m.Cursor.AddChildren(&Revert{
			ErrorType:    "panic",
			TypeValue:    "revert",
			Data:         []byte{},
			DataHex:      "0x" + hex.EncodeToString([]byte{}),
			From:         scope.Contract.Address(),
			ContextValue: ctx,
			CodeValue:    code,
			ParentValue:  m.Cursor,
			DepthValue:   m.Cursor.Depth() + 1,
		})
	}
}

func callOpcodeToString(c vm.OpCode) string {
	switch c {
	case 241:
		return "call"
	case 244:
		return "delegatecall"
	case 250:
		return "staticcall"
	default:
		return fmt.Sprintf("unknown %d", c)
	}
}

func parentContextAndCode(p Action) (common.Address, common.Address) {
	if p != nil {
		return p.(*Call).ForwardedContext, p.(*Call).ForwardedCode
	}
	return common.Address{}, common.Address{}
}

func addZeros(arr []byte, zeros int64) []byte {
	return append(arr, make([]byte, zeros)...)
}
