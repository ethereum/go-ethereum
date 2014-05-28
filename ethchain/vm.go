package ethchain

import (
	_ "bytes"
	"fmt"
	"github.com/ethereum/eth-go/ethutil"
	_ "github.com/obscuren/secp256k1-go"
	_ "math"
	"math/big"
)

var (
	GasStep    = big.NewInt(1)
	GasSha     = big.NewInt(20)
	GasSLoad   = big.NewInt(20)
	GasSStore  = big.NewInt(100)
	GasBalance = big.NewInt(20)
	GasCreate  = big.NewInt(100)
	GasCall    = big.NewInt(20)
	GasMemory  = big.NewInt(1)
	GasTx      = big.NewInt(500)
)

func CalculateTxGas(initSize *big.Int) *big.Int {
	totalGas := new(big.Int)

	txTotalBytes := new(big.Int).Set(initSize)
	txTotalBytes.Div(txTotalBytes, ethutil.Big32)
	totalGas.Add(totalGas, new(big.Int).Mul(txTotalBytes, GasSStore))

	return totalGas
}

type Vm struct {
	txPool *TxPool
	// Stack for processing contracts
	stack *Stack
	// non-persistent key/value memory storage
	mem map[string]*big.Int

	vars RuntimeVars

	state *State

	stateManager *StateManager
}

type RuntimeVars struct {
	Origin      []byte
	BlockNumber uint64
	PrevHash    []byte
	Coinbase    []byte
	Time        int64
	Diff        *big.Int
	TxData      []string
	Value       *big.Int
}

func NewVm(state *State, stateManager *StateManager, vars RuntimeVars) *Vm {
	return &Vm{vars: vars, state: state, stateManager: stateManager}
}

var Pow256 = ethutil.BigPow(2, 256)

var isRequireError = false

func (vm *Vm) RunClosure(closure *Closure, hook DebugHook) (ret []byte, err error) {
	// Recover from any require exception
	defer func() {
		if r := recover(); r != nil /*&& isRequireError*/ {
			ret = closure.Return(nil)
			err = fmt.Errorf("%v", r)
			fmt.Println("vm err", err)
		}
	}()

	ethutil.Config.Log.Debugf("[VM] Running closure %x\n", closure.object.Address())

	// Memory for the current closure
	mem := &Memory{}
	// New stack (should this be shared?)
	stack := NewStack()
	require := func(m int) {
		if stack.Len() < m {
			isRequireError = true
			panic(fmt.Sprintf("stack = %d, req = %d", stack.Len(), m))
		}
	}

	// Instruction pointer
	pc := big.NewInt(0)
	// Current step count
	step := 0
	prevStep := 0

	if ethutil.Config.Debug {
		ethutil.Config.Log.Debugf("#   op\n")
	}

	for {
		prevStep = step
		// The base for all big integer arithmetic
		base := new(big.Int)

		step++
		// Get the memory location of pc
		val := closure.Get(pc)
		// Get the opcode (it must be an opcode!)
		op := OpCode(val.Uint())
		if ethutil.Config.Debug {
			ethutil.Config.Log.Debugf("%-3d %-4s", pc, op.String())
		}

		gas := new(big.Int)
		setStepGasUsage := func(amount *big.Int) {
			gas.Add(gas, amount)
		}

		switch op {
		case SHA3:
			setStepGasUsage(GasSha)
		case SLOAD:
			setStepGasUsage(GasSLoad)
		case SSTORE:
			var mult *big.Int
			y, x := stack.Peekn()
			val := closure.GetMem(x)
			if val.IsEmpty() && len(y.Bytes()) > 0 {
				mult = ethutil.Big2
			} else if !val.IsEmpty() && len(y.Bytes()) == 0 {
				mult = ethutil.Big0
			} else {
				mult = ethutil.Big1
			}
			setStepGasUsage(new(big.Int).Mul(mult, GasSStore))
		case BALANCE:
			setStepGasUsage(GasBalance)
		case CREATE:
			require(3)

			args := stack.Get(big.NewInt(3))
			initSize := new(big.Int).Add(args[1], args[0])

			setStepGasUsage(CalculateTxGas(initSize))
		case CALL:
			setStepGasUsage(GasCall)
		case MLOAD, MSIZE, MSTORE8, MSTORE:
			setStepGasUsage(GasMemory)
		default:
			setStepGasUsage(GasStep)
		}

		if !closure.UseGas(gas) {
			ethutil.Config.Log.Debugln("Insufficient gas", closure.Gas, gas)

			return closure.Return(nil), fmt.Errorf("insufficient gas %v %v", closure.Gas, gas)
		}

		switch op {
		case LOG:
			stack.Print()
			mem.Print()
			// 0x20 range
		case ADD:
			require(2)
			x, y := stack.Popn()
			// (x + y) % 2 ** 256
			base.Add(x, y)
			// Pop result back on the stack
			stack.Push(base)
		case SUB:
			require(2)
			x, y := stack.Popn()
			// (x - y) % 2 ** 256
			base.Sub(x, y)
			// Pop result back on the stack
			stack.Push(base)
		case MUL:
			require(2)
			x, y := stack.Popn()
			// (x * y) % 2 ** 256
			base.Mul(x, y)
			// Pop result back on the stack
			stack.Push(base)
		case DIV:
			require(2)
			x, y := stack.Popn()
			// floor(x / y)
			base.Div(x, y)
			// Pop result back on the stack
			stack.Push(base)
		case SDIV:
			require(2)
			x, y := stack.Popn()
			// n > 2**255
			if x.Cmp(Pow256) > 0 {
				x.Sub(Pow256, x)
			}
			if y.Cmp(Pow256) > 0 {
				y.Sub(Pow256, y)
			}
			z := new(big.Int)
			z.Div(x, y)
			if z.Cmp(Pow256) > 0 {
				z.Sub(Pow256, z)
			}
			// Push result on to the stack
			stack.Push(z)
		case MOD:
			require(2)
			x, y := stack.Popn()
			base.Mod(x, y)
			stack.Push(base)
		case SMOD:
			require(2)
			x, y := stack.Popn()
			// n > 2**255
			if x.Cmp(Pow256) > 0 {
				x.Sub(Pow256, x)
			}
			if y.Cmp(Pow256) > 0 {
				y.Sub(Pow256, y)
			}
			z := new(big.Int)
			z.Mod(x, y)
			if z.Cmp(Pow256) > 0 {
				z.Sub(Pow256, z)
			}
			// Push result on to the stack
			stack.Push(z)
		case EXP:
			require(2)
			x, y := stack.Popn()
			base.Exp(x, y, Pow256)

			stack.Push(base)
		case NEG:
			require(1)
			base.Sub(Pow256, stack.Pop())
			stack.Push(base)
		case LT:
			require(2)
			x, y := stack.Popn()
			// x < y
			if x.Cmp(y) < 0 {
				stack.Push(ethutil.BigTrue)
			} else {
				stack.Push(ethutil.BigFalse)
			}
		case GT:
			require(2)
			x, y := stack.Popn()
			// x > y
			if x.Cmp(y) > 0 {
				stack.Push(ethutil.BigTrue)
			} else {
				stack.Push(ethutil.BigFalse)
			}
		case EQ:
			require(2)
			x, y := stack.Popn()
			// x == y
			if x.Cmp(y) == 0 {
				stack.Push(ethutil.BigTrue)
			} else {
				stack.Push(ethutil.BigFalse)
			}
		case NOT:
			require(1)
			x := stack.Pop()
			if x.Cmp(ethutil.BigFalse) == 0 {
				stack.Push(ethutil.BigTrue)
			} else {
				stack.Push(ethutil.BigFalse)
			}

			// 0x10 range
		case AND:
			require(2)
			x, y := stack.Popn()
			if (x.Cmp(ethutil.BigTrue) >= 0) && (y.Cmp(ethutil.BigTrue) >= 0) {
				stack.Push(ethutil.BigTrue)
			} else {
				stack.Push(ethutil.BigFalse)
			}

		case OR:
			require(2)
			x, y := stack.Popn()
			if (x.Cmp(ethutil.BigInt0) >= 0) || (y.Cmp(ethutil.BigInt0) >= 0) {
				stack.Push(ethutil.BigTrue)
			} else {
				stack.Push(ethutil.BigFalse)
			}
		case XOR:
			require(2)
			x, y := stack.Popn()
			stack.Push(base.Xor(x, y))
		case BYTE:
			require(2)
			val, th := stack.Popn()
			if th.Cmp(big.NewInt(32)) < 0 {
				stack.Push(big.NewInt(int64(len(val.Bytes())-1) - th.Int64()))
			} else {
				stack.Push(ethutil.BigFalse)
			}

			// 0x20 range
		case SHA3:
			require(2)
			size, offset := stack.Popn()
			data := mem.Get(offset.Int64(), size.Int64())

			stack.Push(ethutil.BigD(data))
			// 0x30 range
		case ADDRESS:
			stack.Push(ethutil.BigD(closure.Object().Address()))
		case BALANCE:
			stack.Push(closure.object.Amount)
		case ORIGIN:
			stack.Push(ethutil.BigD(vm.vars.Origin))
		case CALLER:
			stack.Push(ethutil.BigD(closure.Callee().Address()))
		case CALLVALUE:
			stack.Push(vm.vars.Value)
		case CALLDATALOAD:
			require(1)
			offset := stack.Pop().Int64()
			val := closure.Args[offset : offset+32]

			stack.Push(ethutil.BigD(val))
		case CALLDATASIZE:
			stack.Push(big.NewInt(int64(len(closure.Args))))
		case GASPRICE:
			stack.Push(closure.Price)

			// 0x40 range
		case PREVHASH:
			stack.Push(ethutil.BigD(vm.vars.PrevHash))
		case COINBASE:
			stack.Push(ethutil.BigD(vm.vars.Coinbase))
		case TIMESTAMP:
			stack.Push(big.NewInt(vm.vars.Time))
		case NUMBER:
			stack.Push(big.NewInt(int64(vm.vars.BlockNumber)))
		case DIFFICULTY:
			stack.Push(vm.vars.Diff)
		case GASLIMIT:
			// TODO
			stack.Push(big.NewInt(0))

			// 0x50 range
		case PUSH1, PUSH2, PUSH3, PUSH4, PUSH5, PUSH6, PUSH7, PUSH8, PUSH9, PUSH10, PUSH11, PUSH12, PUSH13, PUSH14, PUSH15, PUSH16, PUSH17, PUSH18, PUSH19, PUSH20, PUSH21, PUSH22, PUSH23, PUSH24, PUSH25, PUSH26, PUSH27, PUSH28, PUSH29, PUSH30, PUSH31, PUSH32:
			a := big.NewInt(int64(op) - int64(PUSH1) + 1)
			pc.Add(pc, ethutil.Big1)
			data := closure.Gets(pc, a)
			val := ethutil.BigD(data.Bytes())
			// Push value to stack
			stack.Push(val)
			pc.Add(pc, a.Sub(a, big.NewInt(1)))

			step += int(op) - int(PUSH1) + 1
		case POP:
			require(1)
			stack.Pop()
		case DUP:
			require(1)
			stack.Push(stack.Peek())
		case SWAP:
			require(2)
			x, y := stack.Popn()
			stack.Push(y)
			stack.Push(x)
		case MLOAD:
			require(1)
			offset := stack.Pop()
			stack.Push(ethutil.BigD(mem.Get(offset.Int64(), 32)))
		case MSTORE: // Store the value at stack top-1 in to memory at location stack top
			require(2)
			// Pop value of the stack
			val, mStart := stack.Popn()
			mem.Set(mStart.Int64(), 32, ethutil.BigToBytes(val, 256))
		case MSTORE8:
			require(2)
			val, mStart := stack.Popn()
			base.And(val, new(big.Int).SetInt64(0xff))
			mem.Set(mStart.Int64(), 32, ethutil.BigToBytes(base, 256))
		case SLOAD:
			require(1)
			loc := stack.Pop()
			val := closure.GetMem(loc)
			//fmt.Println("get", val.BigInt(), "@", loc)
			stack.Push(val.BigInt())
		case SSTORE:
			require(2)
			val, loc := stack.Popn()
			//fmt.Println("storing", val, "@", loc)
			closure.SetStorage(loc, ethutil.NewValue(val))

			// Add the change to manifest
			vm.state.manifest.AddStorageChange(closure.Object(), loc.Bytes(), val)
		case JUMP:
			require(1)
			pc = stack.Pop()
			// Reduce pc by one because of the increment that's at the end of this for loop
			//pc.Sub(pc, ethutil.Big1)
			continue
		case JUMPI:
			require(2)
			cond, pos := stack.Popn()
			if cond.Cmp(ethutil.BigTrue) == 0 {
				pc = pos
				//pc.Sub(pc, ethutil.Big1)
				continue
			}
		case PC:
			stack.Push(pc)
		case MSIZE:
			stack.Push(big.NewInt(int64(mem.Len())))
			// 0x60 range
		case CREATE:
			require(3)

			value := stack.Pop()
			size, offset := stack.Popn()

			// Snapshot the current stack so we are able to
			// revert back to it later.
			snapshot := vm.state.Snapshot()

			// Generate a new address
			addr := ethutil.CreateAddress(closure.callee.Address(), closure.callee.N())
			// Create a new contract
			contract := NewContract(addr, value, []byte(""))
			// Set the init script
			contract.initScript = mem.Get(offset.Int64(), size.Int64())
			// Transfer all remaining gas to the new
			// contract so it may run the init script
			gas := new(big.Int).Set(closure.Gas)
			closure.UseGas(gas)

			// Create the closure
			c := NewClosure(closure.callee,
				closure.Object(),
				contract.initScript,
				vm.state,
				gas,
				closure.Price)
			// Call the closure and set the return value as
			// main script.
			c.Script, gas, err = c.Call(vm, nil, hook)

			if err != nil {
				stack.Push(ethutil.BigFalse)

				// Revert the state as it was before.
				vm.state.Revert(snapshot)
			} else {
				stack.Push(ethutil.BigD(addr))

				vm.state.UpdateStateObject(contract)
			}
		case CALL:
			require(7)
			// Closure addr
			addr := stack.Pop()
			// Pop gas and value of the stack.
			gas, value := stack.Popn()
			// Pop input size and offset
			inSize, inOffset := stack.Popn()
			// Pop return size and offset
			retSize, retOffset := stack.Popn()
			// Make sure there's enough gas
			if closure.Gas.Cmp(gas) < 0 {
				stack.Push(ethutil.BigFalse)

				break
			}

			// Get the arguments from the memory
			args := mem.Get(inOffset.Int64(), inSize.Int64())

			snapshot := vm.state.Snapshot()

			// Fetch the contract which will serve as the closure body
			contract := vm.state.GetStateObject(addr.Bytes())

			if contract != nil {
				// Prepay for the gas
				// If gas is set to 0 use all remaining gas for the next call
				if gas.Cmp(big.NewInt(0)) == 0 {
					// Copy
					gas = new(big.Int).Set(closure.Gas)
				}
				closure.UseGas(gas)

				// Add the value to the state object
				contract.AddAmount(value)

				// Create a new callable closure
				closure := NewClosure(closure.Object(), contract, contract.script, vm.state, gas, closure.Price)
				// Executer the closure and get the return value (if any)
				ret, _, err := closure.Call(vm, args, hook)
				if err != nil {
					stack.Push(ethutil.BigFalse)
					// Reset the changes applied this object
					vm.state.Revert(snapshot)
				} else {
					stack.Push(ethutil.BigTrue)

					vm.state.UpdateStateObject(contract)

					mem.Set(retOffset.Int64(), retSize.Int64(), ret)
				}
			} else {
				ethutil.Config.Log.Debugf("Contract %x not found\n", addr.Bytes())
				stack.Push(ethutil.BigFalse)
			}
		case RETURN:
			require(2)
			size, offset := stack.Popn()
			ret := mem.Get(offset.Int64(), size.Int64())

			return closure.Return(ret), nil
		case SUICIDE:
			require(1)

			receiver := vm.state.GetAccount(stack.Pop().Bytes())
			receiver.AddAmount(closure.object.Amount)
			vm.state.UpdateStateObject(receiver)

			closure.object.state.Purge()

			fallthrough
		case STOP: // Stop the closure
			return closure.Return(nil), nil
		default:
			ethutil.Config.Log.Debugf("Invalid opcode %x\n", op)

			return closure.Return(nil), fmt.Errorf("Invalid opcode %x", op)
		}

		pc.Add(pc, ethutil.Big1)

		if hook != nil {
			if !hook(prevStep, op, mem, stack, closure.Object()) {
				return nil, nil
			}
		}
	}
}
