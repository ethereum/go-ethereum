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
)

type Vm struct {
	txPool *TxPool
	// Stack for processing contracts
	stack *Stack
	// non-persistent key/value memory storage
	mem map[string]*big.Int

	vars RuntimeVars

	state *State
}

type RuntimeVars struct {
	Origin      []byte
	BlockNumber uint64
	PrevHash    []byte
	Coinbase    []byte
	Time        int64
	Diff        *big.Int
	TxData      []string
}

func NewVm(state *State, vars RuntimeVars) *Vm {
	return &Vm{vars: vars, state: state}
}

var Pow256 = ethutil.BigPow(2, 256)

var isRequireError = false

func (vm *Vm) RunClosure(closure *Closure, hook DebugHook) (ret []byte, err error) {
	// Recover from any require exception
	defer func() {
		if r := recover(); r != nil && isRequireError {
			fmt.Println(r)

			ret = closure.Return(nil)
			err = fmt.Errorf("%v", r)
		}
	}()

	// If the amount of gas supplied is less equal to 0
	if closure.Gas.Cmp(big.NewInt(0)) <= 0 {
		// TODO Do something
	}

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
	// The base for all big integer arithmetic
	base := new(big.Int)

	/*
		if ethutil.Config.Debug {
			ethutil.Config.Log.Debugf("#   op\n")
		}
	*/

	for {
		step++
		// Get the memory location of pc
		val := closure.Get(pc)
		// Get the opcode (it must be an opcode!)
		op := OpCode(val.Uint())
		/*
			if ethutil.Config.Debug {
				ethutil.Config.Log.Debugf("%-3d %-4s", pc, op.String())
			}
		*/

		// TODO Get each instruction cost properly
		gas := new(big.Int)
		useGas := func(amount *big.Int) {
			gas.Add(gas, new(big.Int).Mul(amount, closure.Price))
		}

		switch op {
		case oSHA3:
			useGas(GasSha)
		case oSLOAD:
			useGas(GasSLoad)
		case oSSTORE:
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
			useGas(base.Mul(mult, GasSStore))
		case oBALANCE:
			useGas(GasBalance)
		case oCREATE:
			useGas(GasCreate)
		case oCALL:
			useGas(GasCall)
		case oMLOAD, oMSIZE, oMSTORE8, oMSTORE:
			useGas(GasMemory)
		default:
			useGas(GasStep)
		}

		if closure.Gas.Cmp(gas) < 0 {
			ethutil.Config.Log.Debugln("Insufficient gas", closure.Gas, gas)

			return closure.Return(nil), fmt.Errorf("insufficient gas %v %v", closure.Gas, gas)
		}
		closure.Gas.Sub(closure.Gas, gas)

		switch op {
		case oLOG:
			stack.Print()
			mem.Print()
		case oSTOP: // Stop the closure
			return closure.Return(nil), nil

			// 0x20 range
		case oADD:
			require(2)
			x, y := stack.Popn()
			// (x + y) % 2 ** 256
			base.Add(x, y)
			base.Mod(base, Pow256)
			// Pop result back on the stack
			stack.Push(base)
		case oSUB:
			require(2)
			x, y := stack.Popn()
			// (x - y) % 2 ** 256
			base.Sub(x, y)
			base.Mod(base, Pow256)
			// Pop result back on the stack
			stack.Push(base)
		case oMUL:
			require(2)
			x, y := stack.Popn()
			// (x * y) % 2 ** 256
			base.Mul(x, y)
			base.Mod(base, Pow256)
			// Pop result back on the stack
			stack.Push(base)
		case oDIV:
			require(2)
			x, y := stack.Popn()
			// floor(x / y)
			base.Div(x, y)
			// Pop result back on the stack
			stack.Push(base)
		case oSDIV:
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
		case oMOD:
			require(2)
			x, y := stack.Popn()
			base.Mod(x, y)
			stack.Push(base)
		case oSMOD:
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
		case oEXP:
			require(2)
			x, y := stack.Popn()
			base.Exp(x, y, Pow256)

			stack.Push(base)
		case oNEG:
			require(1)
			base.Sub(Pow256, stack.Pop())
			stack.Push(base)
		case oLT:
			require(2)
			x, y := stack.Popn()
			// x < y
			if x.Cmp(y) < 0 {
				stack.Push(ethutil.BigTrue)
			} else {
				stack.Push(ethutil.BigFalse)
			}
		case oGT:
			require(2)
			x, y := stack.Popn()
			// x > y
			if x.Cmp(y) > 0 {
				stack.Push(ethutil.BigTrue)
			} else {
				stack.Push(ethutil.BigFalse)
			}
		case oEQ:
			require(2)
			x, y := stack.Popn()
			// x == y
			if x.Cmp(y) == 0 {
				stack.Push(ethutil.BigTrue)
			} else {
				stack.Push(ethutil.BigFalse)
			}
		case oNOT:
			require(1)
			x := stack.Pop()
			if x.Cmp(ethutil.BigFalse) == 0 {
				stack.Push(ethutil.BigTrue)
			} else {
				stack.Push(ethutil.BigFalse)
			}

			// 0x10 range
		case oAND:
			require(2)
			x, y := stack.Popn()
			if (x.Cmp(ethutil.BigTrue) >= 0) && (y.Cmp(ethutil.BigTrue) >= 0) {
				stack.Push(ethutil.BigTrue)
			} else {
				stack.Push(ethutil.BigFalse)
			}

		case oOR:
			require(2)
			x, y := stack.Popn()
			if (x.Cmp(ethutil.BigInt0) >= 0) || (y.Cmp(ethutil.BigInt0) >= 0) {
				stack.Push(ethutil.BigTrue)
			} else {
				stack.Push(ethutil.BigFalse)
			}
		case oXOR:
			require(2)
			x, y := stack.Popn()
			stack.Push(base.Xor(x, y))
		case oBYTE:
			require(2)
			val, th := stack.Popn()
			if th.Cmp(big.NewInt(32)) < 0 {
				stack.Push(big.NewInt(int64(len(val.Bytes())-1) - th.Int64()))
			} else {
				stack.Push(ethutil.BigFalse)
			}

			// 0x20 range
		case oSHA3:
			require(2)
			size, offset := stack.Popn()
			data := mem.Get(offset.Int64(), size.Int64())

			stack.Push(ethutil.BigD(data))
			// 0x30 range
		case oADDRESS:
			stack.Push(ethutil.BigD(closure.Object().Address()))
		case oBALANCE:
			stack.Push(closure.Value)
		case oORIGIN:
			stack.Push(ethutil.BigD(vm.vars.Origin))
		case oCALLER:
			stack.Push(ethutil.BigD(closure.Callee().Address()))
		case oCALLVALUE:
			// FIXME: Original value of the call, not the current value
			stack.Push(closure.Value)
		case oCALLDATA:
			require(1)
			offset := stack.Pop()
			mem.Set(offset.Int64(), int64(len(closure.Args)), closure.Args)
		case oCALLDATASIZE:
			stack.Push(big.NewInt(int64(len(closure.Args))))
		case oGASPRICE:
			// TODO

			// 0x40 range
		case oPREVHASH:
			stack.Push(ethutil.BigD(vm.vars.PrevHash))
		case oCOINBASE:
			stack.Push(ethutil.BigD(vm.vars.Coinbase))
		case oTIMESTAMP:
			stack.Push(big.NewInt(vm.vars.Time))
		case oNUMBER:
			stack.Push(big.NewInt(int64(vm.vars.BlockNumber)))
		case oDIFFICULTY:
			stack.Push(vm.vars.Diff)
		case oGASLIMIT:
		// TODO

		// 0x50 range
		case oPUSH: // Push PC+1 on to the stack
			pc.Add(pc, ethutil.Big1)
			data := closure.Gets(pc, big.NewInt(32))
			val := ethutil.BigD(data.Bytes())

			// Push value to stack
			stack.Push(val)

			pc.Add(pc, big.NewInt(31))
			step++
		case oPUSH20:
			pc.Add(pc, ethutil.Big1)
			data := closure.Gets(pc, big.NewInt(20))
			val := ethutil.BigD(data.Bytes())

			// Push value to stack
			stack.Push(val)

			pc.Add(pc, big.NewInt(19))
			step++
		case oPOP:
			require(1)
			stack.Pop()
		case oDUP:
			require(1)
			stack.Push(stack.Peek())
		case oSWAP:
			require(2)
			x, y := stack.Popn()
			stack.Push(y)
			stack.Push(x)
		case oMLOAD:
			require(1)
			offset := stack.Pop()
			stack.Push(ethutil.BigD(mem.Get(offset.Int64(), 32)))
		case oMSTORE: // Store the value at stack top-1 in to memory at location stack top
			require(2)
			// Pop value of the stack
			val, mStart := stack.Popn()
			mem.Set(mStart.Int64(), 32, ethutil.BigToBytes(val, 256))
		case oMSTORE8:
			require(2)
			val, mStart := stack.Popn()
			base.And(val, new(big.Int).SetInt64(0xff))
			mem.Set(mStart.Int64(), 32, ethutil.BigToBytes(base, 256))
		case oSLOAD:
			require(1)
			loc := stack.Pop()
			val := closure.GetMem(loc)
			stack.Push(val.BigInt())
		case oSSTORE:
			require(2)
			val, loc := stack.Popn()
			closure.SetMem(loc, ethutil.NewValue(val))
		case oJUMP:
			require(1)
			pc = stack.Pop()
		case oJUMPI:
			require(2)
			cond, pos := stack.Popn()
			if cond.Cmp(ethutil.BigTrue) == 0 {
				pc = pos
			}
		case oPC:
			stack.Push(pc)
		case oMSIZE:
			stack.Push(big.NewInt(int64(mem.Len())))
			// 0x60 range
		case oCREATE:
		case oCALL:
			require(7)
			// Closure addr
			addr := stack.Pop()
			// Pop gas and value of the stack.
			gas, value := stack.Popn()
			// Pop input size and offset
			inSize, inOffset := stack.Popn()
			// Pop return size and offset
			retSize, retOffset := stack.Popn()
			// Get the arguments from the memory
			args := mem.Get(inOffset.Int64(), inSize.Int64())
			// Fetch the contract which will serve as the closure body
			contract := vm.state.GetContract(addr.Bytes())
			// Create a new callable closure
			closure := NewClosure(closure, contract, contract.script, vm.state, gas, closure.Price, value)
			// Executer the closure and get the return value (if any)
			ret, err := closure.Call(vm, args, hook)
			if err != nil {
				stack.Push(ethutil.BigFalse)
			} else {
				stack.Push(ethutil.BigTrue)
			}

			mem.Set(retOffset.Int64(), retSize.Int64(), ret)
		case oRETURN:
			require(2)
			size, offset := stack.Popn()
			ret := mem.Get(offset.Int64(), size.Int64())

			return closure.Return(ret), nil
		case oSUICIDE:
			/*
				recAddr := stack.Pop().Bytes()
				// Purge all memory
				deletedMemory := contract.state.Purge()
				// Add refunds to the pop'ed address
				refund := new(big.Int).Mul(StoreFee, big.NewInt(int64(deletedMemory)))
				account := state.GetAccount(recAddr)
				account.Amount.Add(account.Amount, refund)
				// Update the refunding address
				state.UpdateAccount(recAddr, account)
				// Delete the contract
				state.trie.Update(string(addr), "")

				ethutil.Config.Log.Debugf("(%d) => %x\n", deletedMemory, recAddr)
				break out
			*/
		default:
			ethutil.Config.Log.Debugf("Invalid opcode %x\n", op)

			return closure.Return(nil), fmt.Errorf("Invalid opcode %x", op)
		}

		pc.Add(pc, ethutil.Big1)

		if hook != nil {
			hook(step-1, op, mem, stack)
		}
	}
}
