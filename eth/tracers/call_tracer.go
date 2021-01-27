package tracers

import (
  "fmt"
  "math/big"
  "github.com/ethereum/go-ethereum/common"
  "github.com/ethereum/go-ethereum/common/hexutil"
  "github.com/ethereum/go-ethereum/core/state"
  "github.com/ethereum/go-ethereum/core/vm"
  // "github.com/ethereum/go-ethereum/log"
  "github.com/holiman/uint256"
  "time"
)

type call struct {
  Type vm.OpCode `json:"type"`
  From common.Address `json:"from"`
  To common.Address `json:"to"`
  Value *hexutil.Big `json:"value,omitempty"`
  Gas hexutil.Uint64 `json:"gas"`
  GasUsed hexutil.Uint64 `json:"gasUsed"`
  Input hexutil.Bytes `json:"input"`
  Output hexutil.Bytes `json:"output"`
  Time string `json:"time,omitempty"`
  Calls []*call `json:"calls,omitempty"`
  Error string `json:"error,omitempty"`
  startTime time.Time
  outOff uint64
  outLen uint64
  gasIn uint64
  gasCost uint64
}

type TracerResult interface {
  vm.Tracer
  GetResult() (interface{}, error)
}


type CallTracer struct {
  callStack []*call
  descended bool
  statedb *state.StateDB
}

func NewCallTracer(statedb *state.StateDB) TracerResult {
  return &CallTracer{
    callStack: []*call{},
    descended: false,
    statedb: statedb,
  }
}

func (tracer *CallTracer) i() int {
  return len(tracer.callStack) - 1
}

func (tracer *CallTracer) GetResult() (interface{}, error) {
  return tracer.callStack[0], nil
}

func (tracer *CallTracer) CaptureStart(from common.Address, to common.Address, create bool, input []byte, gas uint64, value *big.Int) error {
  hvalue := hexutil.Big(*value)
  tracer.callStack = []*call{&call{
    From: from,
    To: to,
    Value: &hvalue,
    Gas: hexutil.Uint64(gas),
    Input: hexutil.Bytes(input),
    Calls: []*call{},
  }}
  return nil
}
func (tracer *CallTracer) CaptureEnd(output []byte, gasUsed uint64, t time.Duration, err error) error {
  tracer.callStack[tracer.i()].GasUsed = hexutil.Uint64(gasUsed)
  tracer.callStack[tracer.i()].Time = fmt.Sprintf("%v", t)
  tracer.callStack[tracer.i()].Output = hexutil.Bytes(output)
  return nil
}


func (tracer *CallTracer) descend(newCall *call) {
  tracer.callStack[tracer.i()].Calls = append(tracer.callStack[tracer.i()].Calls, newCall)
  tracer.callStack = append(tracer.callStack, newCall)
  tracer.descended = true
}

func toAddress(value *uint256.Int) common.Address {
  return common.BytesToAddress(value.Bytes())
}

func (tracer *CallTracer) CaptureState(env *vm.EVM, pc uint64, op vm.OpCode, gas, cost uint64, memory *vm.Memory, stack *vm.Stack, rStack *vm.ReturnStack, rData []byte, contract *vm.Contract, depth int, err error) error {
  // for depth < len(tracer.callStack) {
  //   c := tracer.callStack[tracer.i()]
  //   c.GasUsed = c.Gas - gas
  //   tracer.callStack = tracer.callStack[:tracer.i()]
  // }
  if op == vm.CREATE || op == vm.CREATE2 {
    inOff :=  stack.Back(1).Uint64()
    inLen := stack.Back(2).Uint64()
    hvalue := hexutil.Big(*contract.Value())
    tracer.descend(&call{
      Type: op,
      From: contract.Caller(),
      Input: memory.GetCopy(int64(inOff), int64(inLen)),
      gasIn: gas,
      gasCost: cost,
      Value: &hvalue,
      startTime: time.Now(),
    })
    return nil
  }
  if op == vm.SELFDESTRUCT {
    hvalue := hexutil.Big(*tracer.statedb.GetBalance(contract.Caller()))
    tracer.descend(&call{
      Type: op,
      From: contract.Caller(),
      To: toAddress(stack.Back(0)),
      // TODO: Is this input correct?
      Input: contract.Input,
      Value: &hvalue,
      gasIn: gas,
      gasCost: cost,
      startTime: time.Now(),
    })
    return nil
  }
  if op == vm.CALL || op == vm.CALLCODE || op == vm.DELEGATECALL || op == vm.STATICCALL {
    toAddress := toAddress(stack.Back(1))
    if _, isPrecompile := vm.PrecompiledContractsIstanbul[toAddress]; isPrecompile { return nil }
    off := 1
    if op == vm.DELEGATECALL || op == vm.STATICCALL { off = 0 }
    inOff := stack.Back(2 + off).Uint64()
    inLength := stack.Back(3 + off).Uint64()
    newCall := &call{
      Type: op,
      From: contract.Address(),
      To: toAddress,
      Input: memory.GetCopy(int64(inOff), int64(inLength)),
      gasIn: gas,
      gasCost: cost,
      outOff: stack.Back(4 + off).Uint64(),
      outLen: stack.Back(5 + off).Uint64(),
      startTime: time.Now(),
    }
    if off == 1 {
      value := hexutil.Big(*new(big.Int).SetBytes(stack.Back(2).Bytes()))
      newCall.Value = &value
    }
    tracer.descend(newCall)
    return nil
  }
  if tracer.descended {
    if depth >= len(tracer.callStack) {
      tracer.callStack[tracer.i()].Gas = hexutil.Uint64(gas)
    }
    tracer.descended = false
  }
  if op == vm.REVERT {
    tracer.callStack[tracer.i()].Error = "execution reverted";
    return nil
  }
  if depth == len(tracer.callStack) - 1 {
    c := tracer.callStack[tracer.i()]
    // c.Time = fmt.Sprintf("%v", time.Since(c.startTime))
    tracer.callStack = tracer.callStack[:len(tracer.callStack) - 1]
    if c.Type == vm.CREATE || c.Type == vm.CREATE2 {
      c.GasUsed = hexutil.Uint64(c.gasIn - c.gasCost - gas)
      ret := stack.Back(0)
      if ret.Uint64() != 0 {
        c.To = common.BytesToAddress(ret.Bytes())
        c.Output = tracer.statedb.GetCode(c.To)
      } else if c.Error == "" {
        c.Error = "internal failure"
      }
    } else {
      c.GasUsed = hexutil.Uint64(c.gasIn - c.gasCost + uint64(c.Gas) - gas)
      ret := stack.Back(0)
      if ret.Uint64() != 0 {
        c.Output = hexutil.Bytes(memory.GetCopy(int64(c.outOff), int64(c.outLen)))
      } else if c.Error == "" {
        c.Error = "internal failure"
      }
    }
  }
  return nil
}
func (tracer *CallTracer) CaptureFault(env *vm.EVM, pc uint64, op vm.OpCode, gas, cost uint64, memory *vm.Memory, stack *vm.Stack, rStack *vm.ReturnStack, contract *vm.Contract, depth int, err error) error {
  return fmt.Errorf("Not implemented")
}


// 1/25: 3h
// 1/26: 3h
