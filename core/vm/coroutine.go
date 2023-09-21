package vm

import (
	"github.com/ethereum/go-ethereum/common/math"
)

//TODO: Use references where possible
type Coroutine struct {
  PC uint64
  Stack Stack
}

func NewCoroutine(pc uint64, stack Stack) Coroutine {
  newStack := deepCopyStack(&stack)
  return Coroutine{pc, *newStack}
}

func (co *Coroutine) ExecuteCoroutine(interpreter *EVMInterpreter, scope *ScopeContext) (ret []byte, err error) {
  // Don't bother with the execution if there's no code.
  if len(scope.Contract.Code) == 0 {
    return nil, nil
  }

  var (
    op          OpCode        // current opcode
    mem         = scope.Memory // memory of the contract
    stack       = &co.Stack  // stack of the contract
    contract    = scope.Contract
    callContext = scope
    pc   = co.PC
    cost uint64 // TODO: Bake in with other run gas
    // copies used by tracer
    res     []byte // result of the opcode execution function
  )
  // Don't move this deferred function, it's placed before the capturestate-deferred method,
  // so that it get's executed _after_: the capturestate needs the stacks before
  // they are returned to the pools
  defer func() {
    // TODO: Think more about this
    // callContext.AwaitTermination(interpreter)
    returnStack(stack)
  }()
  callContext.Stack = stack

  //TODO : Debug and other removed stuff?
  // parent context.
  for {
    // Get the operation from the jump table and validate the stack to ensure there are
    // enough stack items available to perform the operation.
    op = contract.GetOp(pc)
    operation := interpreter.table[op]
    cost = operation.constantGas // For tracing
    // Validate stack
    if sLen := stack.len(); sLen < operation.minStack {
      return nil, &ErrStackUnderflow{stackLen: sLen, required: operation.minStack}
    } else if sLen > operation.maxStack {
      return nil, &ErrStackOverflow{stackLen: sLen, limit: operation.maxStack}
    }
    if !contract.UseGas(cost) {
      return nil, ErrOutOfGas
    }
    if operation.dynamicGas != nil {
      // All ops with a dynamic memory usage also has a dynamic gas cost.
      var memorySize uint64
      // calculate the new memory size and expand the memory to fit
      // the operation
      // Memory check needs to be done prior to evaluating the dynamic gas portion,
      // to detect calculation overflows
      if operation.memorySize != nil {
        memSize, overflow := operation.memorySize(stack)
        if overflow {
          return nil, ErrGasUintOverflow
        }
        // memory is expanded in words of 32 bytes. Gas
        // is also calculated in words.
        if memorySize, overflow = math.SafeMul(toWordSize(memSize), 32); overflow {
          return nil, ErrGasUintOverflow
        }
      }
      // Consume the gas and return an error if not enough gas is available.
      // cost is explicitly set so that the capture state defer method can get the proper cost
      var dynamicCost uint64
      dynamicCost, err = operation.dynamicGas(interpreter.evm, contract, stack, mem, memorySize)
      cost += dynamicCost // for tracing
      if err != nil || !contract.UseGas(dynamicCost) {
        return nil, ErrOutOfGas
      }
      if memorySize > 0 {
        mem.Resize(memorySize)
      }
    }
    // execute the operation
    res, err = operation.execute(&pc, interpreter, callContext)
    if err != nil {
      break
    }
    pc++
  }

  if err == errStopToken {
    err = nil // clear stop token error
  }

  return res, err
}
