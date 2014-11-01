package vm

import (
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethutil"
)

// BIG FAT WARNING. THIS VM IS NOT YET IS USE!
// I want to get all VM tests pass first before updating this VM

type Vm struct {
	env   Environment
	err   error
	depth int
}

func New(env Environment, typ Type) VirtualMachine {
	switch typ {
	case DebugVmTy:
		return NewDebugVm(env)
	default:
		return &Vm{env: env}
	}
}

func (self *Vm) RunClosure(closure *Closure) (ret []byte, err error) {
	self.depth++

	// Recover from any require exception
	defer func() {
		if r := recover(); r != nil {
			ret = closure.Return(nil)
			err = fmt.Errorf("%v", r)
		}
	}()

	// Don't bother with the execution if there's no code.
	if len(closure.Code) == 0 {
		return closure.Return(nil), nil
	}

	var (
		op OpCode

		mem     = &Memory{}
		stack   = NewStack()
		pc      = 0
		step    = 0
		require = func(m int) {
			if stack.Len() < m {
				panic(fmt.Sprintf("%04v (%v) stack err size = %d, required = %d", pc, op, stack.Len(), m))
			}
		}
	)

	for {
		// The base for all big integer arithmetic
		base := new(big.Int)

		step++
		// Get the memory location of pc
		op := closure.GetOp(pc)

		gas := new(big.Int)
		addStepGasUsage := func(amount *big.Int) {
			gas.Add(gas, amount)
		}

		addStepGasUsage(GasStep)

		var newMemSize *big.Int = ethutil.Big0
		switch op {
		case STOP:
			gas.Set(ethutil.Big0)
		case SUICIDE:
			gas.Set(ethutil.Big0)
		case SLOAD:
			gas.Set(GasSLoad)
		case SSTORE:
			var mult *big.Int
			y, x := stack.Peekn()
			val := closure.GetStorage(x)
			if val.BigInt().Cmp(ethutil.Big0) == 0 && len(y.Bytes()) > 0 {
				mult = ethutil.Big2
			} else if val.BigInt().Cmp(ethutil.Big0) != 0 && len(y.Bytes()) == 0 {
				mult = ethutil.Big0
			} else {
				mult = ethutil.Big1
			}
			gas = new(big.Int).Mul(mult, GasSStore)
		case BALANCE:
			gas.Set(GasBalance)
		case MSTORE:
			require(2)
			newMemSize = calcMemSize(stack.Peek(), u256(32))
		case MLOAD:
			require(1)

			newMemSize = calcMemSize(stack.Peek(), u256(32))
		case MSTORE8:
			require(2)
			newMemSize = calcMemSize(stack.Peek(), u256(1))
		case RETURN:
			require(2)

			newMemSize = calcMemSize(stack.Peek(), stack.data[stack.Len()-2])
		case SHA3:
			require(2)

			gas.Set(GasSha)

			newMemSize = calcMemSize(stack.Peek(), stack.data[stack.Len()-2])
		case CALLDATACOPY:
			require(2)

			newMemSize = calcMemSize(stack.Peek(), stack.data[stack.Len()-3])
		case CODECOPY:
			require(3)

			newMemSize = calcMemSize(stack.Peek(), stack.data[stack.Len()-3])
		case EXTCODECOPY:
			require(4)

			newMemSize = calcMemSize(stack.data[stack.Len()-2], stack.data[stack.Len()-4])
		case CALL, CALLCODE:
			require(7)
			gas.Set(GasCall)
			addStepGasUsage(stack.data[stack.Len()-1])

			x := calcMemSize(stack.data[stack.Len()-6], stack.data[stack.Len()-7])
			y := calcMemSize(stack.data[stack.Len()-4], stack.data[stack.Len()-5])

			newMemSize = ethutil.BigMax(x, y)
		case CREATE:
			require(3)
			gas.Set(GasCreate)

			newMemSize = calcMemSize(stack.data[stack.Len()-2], stack.data[stack.Len()-3])
		}

		if newMemSize.Cmp(ethutil.Big0) > 0 {
			newMemSize.Add(newMemSize, u256(31))
			newMemSize.Div(newMemSize, u256(32))
			newMemSize.Mul(newMemSize, u256(32))

			if newMemSize.Cmp(u256(int64(mem.Len()))) > 0 {
				memGasUsage := new(big.Int).Sub(newMemSize, u256(int64(mem.Len())))
				memGasUsage.Mul(GasMemory, memGasUsage)
				memGasUsage.Div(memGasUsage, u256(32))

				addStepGasUsage(memGasUsage)
			}
		}

		if !closure.UseGas(gas) {
			err := fmt.Errorf("Insufficient gas for %v. req %v has %v", op, gas, closure.Gas)

			closure.UseGas(closure.Gas)

			return closure.Return(nil), err
		}

		mem.Resize(newMemSize.Uint64())

		switch op {
		// 0x20 range
		case ADD:
			require(2)
			x, y := stack.Popn()

			base.Add(y, x)

			U256(base)

			// Pop result back on the stack
			stack.Push(base)
		case SUB:
			require(2)
			x, y := stack.Popn()

			base.Sub(y, x)

			U256(base)

			// Pop result back on the stack
			stack.Push(base)
		case MUL:
			require(2)
			x, y := stack.Popn()

			base.Mul(y, x)

			U256(base)

			// Pop result back on the stack
			stack.Push(base)
		case DIV:
			require(2)
			x, y := stack.Popn()

			if x.Cmp(ethutil.Big0) != 0 {
				base.Div(y, x)
			}

			U256(base)

			// Pop result back on the stack
			stack.Push(base)
		case SDIV:
			require(2)
			y, x := S256(stack.Pop()), S256(stack.Pop())

			if x.Cmp(ethutil.Big0) == 0 {
				base.Set(ethutil.Big0)
			} else {
				n := new(big.Int)
				if new(big.Int).Mul(y, x).Cmp(ethutil.Big0) < 0 {
					n.SetInt64(-1)
				} else {
					n.SetInt64(1)
				}

				base.Div(y.Abs(y), x.Mul(x.Abs(x), n))

				U256(base)
			}

			stack.Push(base)
		case MOD:
			require(2)
			x, y := stack.Popn()

			base.Mod(y, x)

			U256(base)

			stack.Push(base)
		case SMOD:
			require(2)
			y, x := S256(stack.Pop()), S256(stack.Pop())

			if x.Cmp(ethutil.Big0) == 0 {
				base.Set(ethutil.Big0)
			} else {
				n := new(big.Int)
				if y.Cmp(ethutil.Big0) < 0 {
					n.SetInt64(-1)
				} else {
					n.SetInt64(1)
				}

				base.Mod(y.Abs(y), x.Mul(x.Abs(x), n))

				U256(base)
			}

			stack.Push(base)

		case EXP:
			require(2)
			x, y := stack.Popn()

			base.Exp(y, x, Pow256)

			U256(base)

			stack.Push(base)
		case NOT:
			require(1)
			base.Sub(Pow256, stack.Pop())

			base = U256(base)

			stack.Push(base)
		case LT:
			require(2)
			x, y := stack.Popn()
			// x < y
			if y.Cmp(x) < 0 {
				stack.Push(ethutil.BigTrue)
			} else {
				stack.Push(ethutil.BigFalse)
			}
		case GT:
			require(2)
			x, y := stack.Popn()

			// x > y
			if y.Cmp(x) > 0 {
				stack.Push(ethutil.BigTrue)
			} else {
				stack.Push(ethutil.BigFalse)
			}

		case SLT:
			require(2)
			y, x := S256(stack.Pop()), S256(stack.Pop())
			// x < y
			if y.Cmp(S256(x)) < 0 {
				stack.Push(ethutil.BigTrue)
			} else {
				stack.Push(ethutil.BigFalse)
			}
		case SGT:
			require(2)
			y, x := S256(stack.Pop()), S256(stack.Pop())

			// x > y
			if y.Cmp(x) > 0 {
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
		case ISZERO:
			require(1)
			x := stack.Pop()
			if x.Cmp(ethutil.BigFalse) > 0 {
				stack.Push(ethutil.BigFalse)
			} else {
				stack.Push(ethutil.BigTrue)
			}

			// 0x10 range
		case AND:
			require(2)
			x, y := stack.Popn()

			stack.Push(base.And(y, x))
		case OR:
			require(2)
			x, y := stack.Popn()

			stack.Push(base.Or(y, x))
		case XOR:
			require(2)
			x, y := stack.Popn()

			stack.Push(base.Xor(y, x))
		case BYTE:
			require(2)
			val, th := stack.Popn()
			if th.Cmp(big.NewInt(32)) < 0 && th.Cmp(big.NewInt(int64(len(val.Bytes())))) < 0 {
				byt := big.NewInt(int64(ethutil.LeftPadBytes(val.Bytes(), 32)[th.Int64()]))
				stack.Push(byt)

			} else {
				stack.Push(ethutil.BigFalse)
			}
		case ADDMOD:
			require(3)

			x := stack.Pop()
			y := stack.Pop()
			z := stack.Pop()

			base.Add(x, y)
			base.Mod(base, z)

			U256(base)

			stack.Push(base)
		case MULMOD:
			require(3)

			x := stack.Pop()
			y := stack.Pop()
			z := stack.Pop()

			base.Mul(x, y)
			base.Mod(base, z)

			U256(base)

			stack.Push(base)

			// 0x20 range
		case SHA3:
			require(2)
			size, offset := stack.Popn()
			data := crypto.Sha3(mem.Get(offset.Int64(), size.Int64()))

			stack.Push(ethutil.BigD(data))

			// 0x30 range
		case ADDRESS:
			stack.Push(ethutil.BigD(closure.Address()))

		case BALANCE:
			require(1)

			addr := stack.Pop().Bytes()
			balance := self.env.State().GetBalance(addr)

			stack.Push(balance)

		case ORIGIN:
			origin := self.env.Origin()

			stack.Push(ethutil.BigD(origin))

		case CALLER:
			caller := closure.caller.Address()
			stack.Push(ethutil.BigD(caller))

		case CALLVALUE:
			value := closure.exe.value

			stack.Push(value)

		case CALLDATALOAD:
			require(1)
			var (
				offset  = stack.Pop()
				data    = make([]byte, 32)
				lenData = big.NewInt(int64(len(closure.Args)))
			)

			if lenData.Cmp(offset) >= 0 {
				length := new(big.Int).Add(offset, ethutil.Big32)
				length = ethutil.BigMin(length, lenData)

				copy(data, closure.Args[offset.Int64():length.Int64()])
			}

			stack.Push(ethutil.BigD(data))
		case CALLDATASIZE:
			l := int64(len(closure.Args))
			stack.Push(big.NewInt(l))

		case CALLDATACOPY:
			var (
				size = int64(len(closure.Args))
				mOff = stack.Pop().Int64()
				cOff = stack.Pop().Int64()
				l    = stack.Pop().Int64()
			)

			if cOff > size {
				cOff = 0
				l = 0
			} else if cOff+l > size {
				l = 0
			}

			code := closure.Args[cOff : cOff+l]

			mem.Set(mOff, l, code)
		case CODESIZE, EXTCODESIZE:
			var code []byte
			if op == EXTCODECOPY {
				addr := stack.Pop().Bytes()

				code = self.env.State().GetCode(addr)
			} else {
				code = closure.Code
			}

			l := big.NewInt(int64(len(code)))
			stack.Push(l)

		case CODECOPY, EXTCODECOPY:
			var code []byte
			if op == EXTCODECOPY {
				addr := stack.Pop().Bytes()

				code = self.env.State().GetCode(addr)
			} else {
				code = closure.Code
			}

			var (
				size = int64(len(code))
				mOff = stack.Pop().Int64()
				cOff = stack.Pop().Int64()
				l    = stack.Pop().Int64()
			)

			if cOff > size {
				cOff = 0
				l = 0
			} else if cOff+l > size {
				l = 0
			}

			codeCopy := code[cOff : cOff+l]

			mem.Set(mOff, l, codeCopy)
		case GASPRICE:
			stack.Push(closure.Price)

			// 0x40 range
		case PREVHASH:
			prevHash := self.env.PrevHash()

			stack.Push(ethutil.BigD(prevHash))

		case COINBASE:
			coinbase := self.env.Coinbase()

			stack.Push(ethutil.BigD(coinbase))

		case TIMESTAMP:
			time := self.env.Time()

			stack.Push(big.NewInt(time))

		case NUMBER:
			number := self.env.BlockNumber()

			stack.Push(number)

		case DIFFICULTY:
			difficulty := self.env.Difficulty()

			stack.Push(difficulty)

		case GASLIMIT:
			// TODO
			stack.Push(big.NewInt(0))

			// 0x50 range
		case PUSH1, PUSH2, PUSH3, PUSH4, PUSH5, PUSH6, PUSH7, PUSH8, PUSH9, PUSH10, PUSH11, PUSH12, PUSH13, PUSH14, PUSH15, PUSH16, PUSH17, PUSH18, PUSH19, PUSH20, PUSH21, PUSH22, PUSH23, PUSH24, PUSH25, PUSH26, PUSH27, PUSH28, PUSH29, PUSH30, PUSH31, PUSH32:
			a := int(op - PUSH1 + 1)
			val := ethutil.BigD(closure.GetBytes(int(pc+1), a))
			// Push value to stack
			stack.Push(val)

			pc += a

			step += int(op) - int(PUSH1) + 1
		case POP:
			require(1)
			stack.Pop()
		case DUP1, DUP2, DUP3, DUP4, DUP5, DUP6, DUP7, DUP8, DUP9, DUP10, DUP11, DUP12, DUP13, DUP14, DUP15, DUP16:
			n := int(op - DUP1 + 1)
			stack.Dupn(n)
		case SWAP1, SWAP2, SWAP3, SWAP4, SWAP5, SWAP6, SWAP7, SWAP8, SWAP9, SWAP10, SWAP11, SWAP12, SWAP13, SWAP14, SWAP15, SWAP16:
			n := int(op - SWAP1 + 2)
			stack.Swapn(n)

		case MLOAD:
			require(1)
			offset := stack.Pop()
			val := ethutil.BigD(mem.Get(offset.Int64(), 32))
			stack.Push(val)

		case MSTORE: // Store the value at stack top-1 in to memory at location stack top
			require(2)
			// Pop value of the stack
			val, mStart := stack.Popn()
			mem.Set(mStart.Int64(), 32, ethutil.BigToBytes(val, 256))

		case MSTORE8:
			require(2)
			off := stack.Pop()
			val := stack.Pop()

			mem.store[off.Int64()] = byte(val.Int64() & 0xff)

		case SLOAD:
			require(1)
			loc := stack.Pop()
			val := closure.GetStorage(loc)

			stack.Push(val.BigInt())

		case SSTORE:
			require(2)
			val, loc := stack.Popn()
			closure.SetStorage(loc, ethutil.NewValue(val))

			closure.message.AddStorageChange(loc.Bytes())

		case JUMP:
			require(1)
			pc = int(stack.Pop().Int64())
			// Reduce pc by one because of the increment that's at the end of this for loop

			continue
		case JUMPI:
			require(2)
			cond, pos := stack.Popn()
			if cond.Cmp(ethutil.BigTrue) >= 0 {
				pc = int(pos.Int64())

				if closure.GetOp(int(pc)) != JUMPDEST {
					return closure.Return(nil), fmt.Errorf("JUMP missed JUMPDEST %v", pc)
				}

				continue
			}
		case JUMPDEST:
		case PC:
			stack.Push(u256(int64(pc)))
		case MSIZE:
			stack.Push(big.NewInt(int64(mem.Len())))
		case GAS:
			stack.Push(closure.Gas)
			// 0x60 range
		case CREATE:
			require(3)

			var (
				err          error
				value        = stack.Pop()
				size, offset = stack.Popn()
				input        = mem.Get(offset.Int64(), size.Int64())
				gas          = new(big.Int).Set(closure.Gas)

				// Snapshot the current stack so we are able to
				// revert back to it later.
				//snapshot = self.env.State().Copy()
			)

			// Generate a new address
			addr := crypto.CreateAddress(closure.Address(), closure.object.Nonce)
			closure.object.Nonce++

			closure.UseGas(closure.Gas)

			msg := NewExecution(self, addr, input, gas, closure.Price, value)
			ret, err := msg.Exec(addr, closure)
			if err != nil {
				stack.Push(ethutil.BigFalse)

				// Revert the state as it was before.
				//self.env.State().Set(snapshot)

			} else {
				msg.object.Code = ret

				stack.Push(ethutil.BigD(addr))
			}

		case CALL, CALLCODE:
			require(7)

			gas := stack.Pop()
			// Pop gas and value of the stack.
			value, addr := stack.Popn()
			// Pop input size and offset
			inSize, inOffset := stack.Popn()
			// Pop return size and offset
			retSize, retOffset := stack.Popn()

			// Get the arguments from the memory
			args := mem.Get(inOffset.Int64(), inSize.Int64())

			var executeAddr []byte
			if op == CALLCODE {
				executeAddr = closure.Address()
			} else {
				executeAddr = addr.Bytes()
			}

			msg := NewExecution(self, executeAddr, args, gas, closure.Price, value)
			ret, err := msg.Exec(addr.Bytes(), closure)
			if err != nil {
				stack.Push(ethutil.BigFalse)
			} else {
				stack.Push(ethutil.BigTrue)

				mem.Set(retOffset.Int64(), retSize.Int64(), ret)
			}

		case RETURN:
			require(2)
			size, offset := stack.Popn()
			ret := mem.Get(offset.Int64(), size.Int64())

			return closure.Return(ret), nil
		case SUICIDE:
			require(1)

			receiver := self.env.State().GetOrNewStateObject(stack.Pop().Bytes())

			receiver.AddAmount(closure.object.Balance())

			closure.object.MarkForDeletion()

			fallthrough
		case STOP: // Stop the closure

			return closure.Return(nil), nil
		default:
			vmlogger.Debugf("(pc) %-3v Invalid opcode %x\n", pc, op)

			//panic(fmt.Sprintf("Invalid opcode %x", op))

			return closure.Return(nil), fmt.Errorf("Invalid opcode %x", op)
		}

		pc++
	}
}

func (self *Vm) Env() Environment {
	return self.env
}

func (self *Vm) Depth() int {
	return self.depth
}

func (self *Vm) Printf(format string, v ...interface{}) VirtualMachine { return self }
func (self *Vm) Endl() VirtualMachine                                  { return self }
