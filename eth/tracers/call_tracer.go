package tracers

import (
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/vm"

	// "github.com/ethereum/go-ethereum/log"
	"time"

	"github.com/holiman/uint256"
)

type call struct {
	Type      string         `json:"type"`
	From      common.Address `json:"from"`
	To        common.Address `json:"to"`
	Value     *hexutil.Big   `json:"value,omitempty"`
	Gas       hexutil.Uint64 `json:"gas"`
	GasUsed   hexutil.Uint64 `json:"gasUsed"`
	Input     hexutil.Bytes  `json:"input"`
	Output    hexutil.Bytes  `json:"output"`
	Time      string         `json:"time,omitempty"`
	Calls     []*call        `json:"calls,omitempty"`
	Error     string         `json:"error,omitempty"`
	startTime time.Time
	outOff    uint64
	outLen    uint64
	gasIn     uint64
	gasCost   uint64
}

type TracerResult interface {
	vm.Tracer
	GetResult() (interface{}, error)
}

type CallTracer struct {
	callStack []*call
	descended bool
	statedb   *state.StateDB
}

func NewCallTracer(statedb *state.StateDB) TracerResult {
	return &CallTracer{
		callStack: []*call{},
		descended: false,
		statedb:   statedb,
	}
}

func (tracer *CallTracer) i() int {
	return len(tracer.callStack) - 1
}

func (tracer *CallTracer) GetResult() (interface{}, error) {
	return tracer.callStack[0], nil
}

func (tracer *CallTracer) CaptureStart(evm *vm.EVM, from common.Address, to common.Address, create bool, input []byte, gas uint64, value *big.Int) {
	hvalue := hexutil.Big(*value)
	tracer.callStack = []*call{&call{
		From:  from,
		To:    to,
		Value: &hvalue,
		Gas:   hexutil.Uint64(gas),
		Input: hexutil.Bytes(input),
		Calls: []*call{},
	}}
}
func (tracer *CallTracer) CaptureEnd(output []byte, gasUsed uint64, t time.Duration, err error) {
	tracer.callStack[tracer.i()].GasUsed = hexutil.Uint64(gasUsed)
	tracer.callStack[tracer.i()].Time = fmt.Sprintf("%v", t)
	tracer.callStack[tracer.i()].Output = hexutil.Bytes(output)
}

func (tracer *CallTracer) descend(newCall *call) {
	tracer.callStack[tracer.i()].Calls = append(tracer.callStack[tracer.i()].Calls, newCall)
	tracer.callStack = append(tracer.callStack, newCall)
	tracer.descended = true
}

func toAddress(value *uint256.Int) common.Address {
	return common.BytesToAddress(value.Bytes())
}

func (tracer *CallTracer) CaptureState(env *vm.EVM, pc uint64, op vm.OpCode, gas, cost uint64, scope *vm.ScopeContext, rData []byte, depth int, err error) {
	// for depth < len(tracer.callStack) {
	//   c := tracer.callStack[tracer.i()]
	//   c.GasUsed = c.Gas - gas
	//   tracer.callStack = tracer.callStack[:tracer.i()]
	// }
	if op == vm.CREATE || op == vm.CREATE2 {
		inOff := scope.Stack.Back(1).Uint64()
		inLen := scope.Stack.Back(2).Uint64()
		hvalue := hexutil.Big(*scope.Contract.Value())
		tracer.descend(&call{
			Type:      op.String(),
			From:      scope.Contract.Caller(),
			Input:     scope.Memory.GetCopy(int64(inOff), int64(inLen)),
			gasIn:     gas,
			gasCost:   cost,
			Value:     &hvalue,
			startTime: time.Now(),
		})
		return
	}
	if op == vm.SELFDESTRUCT {
		hvalue := hexutil.Big(*tracer.statedb.GetBalance(scope.Contract.Caller()))
		tracer.descend(&call{
			Type: op.String(),
			From: scope.Contract.Caller(),
			To:   toAddress(scope.Stack.Back(0)),
			// TODO: Is this input correct?
			Input:     scope.Contract.Input,
			Value:     &hvalue,
			gasIn:     gas,
			gasCost:   cost,
			startTime: time.Now(),
		})
		return
	}
	if op == vm.CALL || op == vm.CALLCODE || op == vm.DELEGATECALL || op == vm.STATICCALL {
		toAddress := toAddress(scope.Stack.Back(1))
		if _, isPrecompile := vm.PrecompiledContractsIstanbul[toAddress]; isPrecompile {
			return
		}
		off := 1
		if op == vm.DELEGATECALL || op == vm.STATICCALL {
			off = 0
		}
		inOff := scope.Stack.Back(2 + off).Uint64()
		inLength := scope.Stack.Back(3 + off).Uint64()
		newCall := &call{
			Type:      op.String(),
			From:      scope.Contract.Address(),
			To:        toAddress,
			Input:     scope.Memory.GetCopy(int64(inOff), int64(inLength)),
			gasIn:     gas,
			gasCost:   cost,
			outOff:    scope.Stack.Back(4 + off).Uint64(),
			outLen:    scope.Stack.Back(5 + off).Uint64(),
			startTime: time.Now(),
		}
		if off == 1 {
			value := hexutil.Big(*new(big.Int).SetBytes(scope.Stack.Back(2).Bytes()))
			newCall.Value = &value
		}
		tracer.descend(newCall)
		return
	}
	if tracer.descended {
		if depth >= len(tracer.callStack) {
			tracer.callStack[tracer.i()].Gas = hexutil.Uint64(gas)
		}
		tracer.descended = false
	}
	if op == vm.REVERT {
		tracer.callStack[tracer.i()].Error = "execution reverted"
		return
	}
	if depth == len(tracer.callStack)-1 {
		c := tracer.callStack[tracer.i()]
		// c.Time = fmt.Sprintf("%v", time.Since(c.startTime))
		tracer.callStack = tracer.callStack[:len(tracer.callStack)-1]
		if vm.StringToOp(c.Type) == vm.CREATE || vm.StringToOp(c.Type) == vm.CREATE2 {
			c.GasUsed = hexutil.Uint64(c.gasIn - c.gasCost - gas)
			ret := scope.Stack.Back(0)
			if ret.Uint64() != 0 {
				c.To = common.BytesToAddress(ret.Bytes())
				c.Output = tracer.statedb.GetCode(c.To)
			} else if c.Error == "" {
				c.Error = "internal failure"
			}
		} else {
			c.GasUsed = hexutil.Uint64(c.gasIn - c.gasCost + uint64(c.Gas) - gas)
			ret := scope.Stack.Back(0)
			if ret.Uint64() != 0 {
				c.Output = hexutil.Bytes(scope.Memory.GetCopy(int64(c.outOff), int64(c.outLen)))
			} else if c.Error == "" {
				c.Error = "internal failure"
			}
		}
	}
	return
}
func (tracer *CallTracer) CaptureFault(env *vm.EVM, pc uint64, op vm.OpCode, gas, cost uint64, memory *vm.ScopeContext, depth int, err error) { }

// 1/25: 3h
// 1/26: 3h
