package vm

import (
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/ethcrypto"
	"github.com/ethereum/go-ethereum/ethstate"
	"github.com/ethereum/go-ethereum/ethutil"
)

type DebugVm struct {
	env Environment

	logTy  byte
	logStr string

	err error

	// Debugging
	Dbg Debugger

	BreakPoints []int64
	Stepping    bool
	Fn          string

	Recoverable bool

	depth int
}

func NewDebugVm(env Environment) *DebugVm {
	lt := LogTyPretty
	if ethutil.Config.Diff {
		lt = LogTyDiff
	}

	return &DebugVm{env: env, logTy: lt, Recoverable: true}
}

func (self *DebugVm) RunClosure(closure *Closure) (ret []byte, err error) {
	self.depth++

	var (
		op OpCode

		mem      = &Memory{}
		stack    = NewStack()
		pc       = big.NewInt(0)
		step     = 0
		prevStep = 0
		state    = self.env.State()
		require  = func(m int) {
			if stack.Len() < m {
				panic(fmt.Sprintf("%04v (%v) stack err size = %d, required = %d", pc, op, stack.Len(), m))
			}
		}

		jump = func(pos *big.Int) {
			p := int(pos.Int64())

			self.Printf(" ~> %v", pos)
			// Return to start
			if p == 0 {
				pc = big.NewInt(0)
			} else {
				nop := OpCode(closure.GetOp(p - 1))
				if nop != JUMPDEST {
					panic(fmt.Sprintf("JUMP missed JUMPDEST (%v) %v", nop, p))
				}

				pc = pos
			}

			self.Endl()
		}
	)

	if self.Recoverable {
		// Recover from any require exception
		defer func() {
			if r := recover(); r != nil {
				self.Endl()

				ret = closure.Return(nil)
				// No error should be set. Recover is used with require
				// Is this too error prone?
			}
		}()
	}

	// Debug hook
	if self.Dbg != nil {
		self.Dbg.SetCode(closure.Code)
	}

	// Don't bother with the execution if there's no code.
	if len(closure.Code) == 0 {
		return closure.Return(nil), nil
	}

	vmlogger.Debugf("(%d) %x gas: %v (d) %x\n", self.depth, closure.Address(), closure.Gas, closure.Args)

	for {
		prevStep = step
		// The base for all big integer arithmetic
		base := new(big.Int)

		step++
		// Get the memory location of pc
		op = closure.GetOp(int(pc.Uint64()))

		// XXX Leave this Println intact. Don't change this to the log system.
		// Used for creating diffs between implementations
		if self.logTy == LogTyDiff {
			switch op {
			case STOP, RETURN, SUICIDE:
				state.GetStateObject(closure.Address()).EachStorage(func(key string, value *ethutil.Value) {
					value.Decode()
					fmt.Printf("%x %x\n", new(big.Int).SetBytes([]byte(key)).Bytes(), value.Bytes())
				})
			}

			b := pc.Bytes()
			if len(b) == 0 {
				b = []byte{0}
			}

			fmt.Printf("%x %x %x %x\n", closure.Address(), b, []byte{byte(op)}, closure.Gas.Bytes())
		}

		gas := new(big.Int)
		addStepGasUsage := func(amount *big.Int) {
			if amount.Cmp(ethutil.Big0) >= 0 {
				gas.Add(gas, amount)
			}
		}

		addStepGasUsage(GasStep)

		var newMemSize *big.Int = ethutil.Big0
		// Stack Check, memory resize & gas phase
		switch op {
		// Stack checks only
		case NOT, CALLDATALOAD, POP, JUMP, BNOT: // 1
			require(1)
		case ADD, SUB, DIV, SDIV, MOD, SMOD, EXP, LT, GT, SLT, SGT, EQ, AND, OR, XOR, BYTE: // 2
			require(2)
		case ADDMOD, MULMOD: // 3
			require(3)
		case SWAP1, SWAP2, SWAP3, SWAP4, SWAP5, SWAP6, SWAP7, SWAP8, SWAP9, SWAP10, SWAP11, SWAP12, SWAP13, SWAP14, SWAP15, SWAP16:
			n := int(op - SWAP1 + 2)
			require(n)
		case DUP1, DUP2, DUP3, DUP4, DUP5, DUP6, DUP7, DUP8, DUP9, DUP10, DUP11, DUP12, DUP13, DUP14, DUP15, DUP16:
			n := int(op - DUP1 + 1)
			require(n)
		case LOG0, LOG1, LOG2, LOG3, LOG4:
			n := int(op - LOG0)
			require(n + 2)

			mSize, mStart := stack.Peekn()
			gas.Set(GasLog)
			addStepGasUsage(new(big.Int).Mul(big.NewInt(int64(n)), GasLog))
			addStepGasUsage(new(big.Int).Add(mSize, mStart))
		// Gas only
		case STOP:
			gas.Set(ethutil.Big0)
		case SUICIDE:
			require(1)

			gas.Set(ethutil.Big0)
		case SLOAD:
			require(1)

			gas.Set(GasSLoad)
		// Memory resize & Gas
		case SSTORE:
			require(2)

			var mult *big.Int
			y, x := stack.Peekn()
			val := closure.GetStorage(x)
			if val.BigInt().Cmp(ethutil.Big0) == 0 && len(y.Bytes()) > 0 {
				// 0 => non 0
				mult = ethutil.Big3
			} else if val.BigInt().Cmp(ethutil.Big0) != 0 && len(y.Bytes()) == 0 {
				state.Refund(closure.caller.Address(), GasSStoreRefund, closure.Price)

				mult = ethutil.Big0
			} else {
				// non 0 => non 0
				mult = ethutil.Big1
			}
			gas.Set(new(big.Int).Mul(mult, GasSStore))
		case BALANCE:
			require(1)
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

		self.Printf("(pc) %-3d -o- %-14s", pc, op.String())
		self.Printf(" (g) %-3v (%v)", gas, closure.Gas)

		if !closure.UseGas(gas) {
			self.Endl()

			tmp := new(big.Int).Set(closure.Gas)

			closure.UseGas(closure.Gas)

			return closure.Return(nil), OOG(gas, tmp)
		}

		mem.Resize(newMemSize.Uint64())

		switch op {
		// 0x20 range
		case ADD:
			x, y := stack.Popn()
			self.Printf(" %v + %v", y, x)

			base.Add(y, x)

			U256(base)

			self.Printf(" = %v", base)
			// Pop result back on the stack
			stack.Push(base)
		case SUB:
			x, y := stack.Popn()
			self.Printf(" %v - %v", y, x)

			base.Sub(y, x)

			U256(base)

			self.Printf(" = %v", base)
			// Pop result back on the stack
			stack.Push(base)
		case MUL:
			x, y := stack.Popn()
			self.Printf(" %v * %v", y, x)

			base.Mul(y, x)

			U256(base)

			self.Printf(" = %v", base)
			// Pop result back on the stack
			stack.Push(base)
		case DIV:
			x, y := stack.Pop(), stack.Pop()
			self.Printf(" %v / %v", x, y)

			if y.Cmp(ethutil.Big0) != 0 {
				base.Div(x, y)
			}

			U256(base)

			self.Printf(" = %v", base)
			// Pop result back on the stack
			stack.Push(base)
		case SDIV:
			x, y := S256(stack.Pop()), S256(stack.Pop())

			self.Printf(" %v / %v", x, y)

			if y.Cmp(ethutil.Big0) == 0 {
				base.Set(ethutil.Big0)
			} else {
				n := new(big.Int)
				if new(big.Int).Mul(x, y).Cmp(ethutil.Big0) < 0 {
					n.SetInt64(-1)
				} else {
					n.SetInt64(1)
				}

				base.Div(x.Abs(x), y.Abs(y)).Mul(base, n)

				U256(base)
			}

			self.Printf(" = %v", base)
			stack.Push(base)
		case MOD:
			x, y := stack.Pop(), stack.Pop()

			self.Printf(" %v %% %v", x, y)

			if y.Cmp(ethutil.Big0) == 0 {
				base.Set(ethutil.Big0)
			} else {
				base.Mod(x, y)
			}

			U256(base)

			self.Printf(" = %v", base)
			stack.Push(base)
		case SMOD:
			x, y := S256(stack.Pop()), S256(stack.Pop())

			self.Printf(" %v %% %v", x, y)

			if y.Cmp(ethutil.Big0) == 0 {
				base.Set(ethutil.Big0)
			} else {
				n := new(big.Int)
				if x.Cmp(ethutil.Big0) < 0 {
					n.SetInt64(-1)
				} else {
					n.SetInt64(1)
				}

				base.Mod(x.Abs(x), y.Abs(y)).Mul(base, n)

				U256(base)
			}

			self.Printf(" = %v", base)
			stack.Push(base)

		case EXP:
			x, y := stack.Popn()

			self.Printf(" %v ** %v", y, x)

			base.Exp(y, x, Pow256)

			U256(base)

			self.Printf(" = %v", base)

			stack.Push(base)
		case BNOT:
			base.Sub(Pow256, stack.Pop()).Sub(base, ethutil.Big1)

			// Not needed
			//base = U256(base)

			stack.Push(base)
		case LT:
			x, y := stack.Popn()
			self.Printf(" %v < %v", y, x)
			// x < y
			if y.Cmp(x) < 0 {
				stack.Push(ethutil.BigTrue)
			} else {
				stack.Push(ethutil.BigFalse)
			}
		case GT:
			x, y := stack.Popn()
			self.Printf(" %v > %v", y, x)

			// x > y
			if y.Cmp(x) > 0 {
				stack.Push(ethutil.BigTrue)
			} else {
				stack.Push(ethutil.BigFalse)
			}

		case SLT:
			y, x := S256(stack.Pop()), S256(stack.Pop())
			self.Printf(" %v < %v", y, x)
			// x < y
			if y.Cmp(S256(x)) < 0 {
				stack.Push(ethutil.BigTrue)
			} else {
				stack.Push(ethutil.BigFalse)
			}
		case SGT:
			y, x := S256(stack.Pop()), S256(stack.Pop())
			self.Printf(" %v > %v", y, x)

			// x > y
			if y.Cmp(x) > 0 {
				stack.Push(ethutil.BigTrue)
			} else {
				stack.Push(ethutil.BigFalse)
			}

		case EQ:
			x, y := stack.Popn()
			self.Printf(" %v == %v", y, x)

			// x == y
			if x.Cmp(y) == 0 {
				stack.Push(ethutil.BigTrue)
			} else {
				stack.Push(ethutil.BigFalse)
			}
		case NOT:
			x := stack.Pop()
			if x.Cmp(ethutil.BigFalse) > 0 {
				stack.Push(ethutil.BigFalse)
			} else {
				stack.Push(ethutil.BigTrue)
			}

			// 0x10 range
		case AND:
			x, y := stack.Popn()
			self.Printf(" %v & %v", y, x)

			stack.Push(base.And(y, x))
		case OR:
			x, y := stack.Popn()
			self.Printf(" %v | %v", y, x)

			stack.Push(base.Or(y, x))
		case XOR:
			x, y := stack.Popn()
			self.Printf(" %v ^ %v", y, x)

			stack.Push(base.Xor(y, x))
		case BYTE:
			val, th := stack.Popn()

			if th.Cmp(big.NewInt(32)) < 0 {
				byt := big.NewInt(int64(ethutil.LeftPadBytes(val.Bytes(), 32)[th.Int64()]))

				base.Set(byt)
			} else {
				base.Set(ethutil.BigFalse)
			}

			self.Printf(" => 0x%x", base.Bytes())

			stack.Push(base)
		case ADDMOD:

			x := stack.Pop()
			y := stack.Pop()
			z := stack.Pop()

			base.Add(x, y)
			base.Mod(base, z)

			U256(base)

			self.Printf(" = %v", base)

			stack.Push(base)
		case MULMOD:

			x := stack.Pop()
			y := stack.Pop()
			z := stack.Pop()

			base.Mul(x, y)
			base.Mod(base, z)

			U256(base)

			self.Printf(" = %v", base)

			stack.Push(base)

			// 0x20 range
		case SHA3:
			size, offset := stack.Popn()
			data := ethcrypto.Sha3(mem.Get(offset.Int64(), size.Int64()))

			stack.Push(ethutil.BigD(data))

			self.Printf(" => %x", data)
			// 0x30 range
		case ADDRESS:
			stack.Push(ethutil.BigD(closure.Address()))

			self.Printf(" => %x", closure.Address())
		case BALANCE:

			addr := stack.Pop().Bytes()
			balance := state.GetBalance(addr)

			stack.Push(balance)

			self.Printf(" => %v (%x)", balance, addr)
		case ORIGIN:
			origin := self.env.Origin()

			stack.Push(ethutil.BigD(origin))

			self.Printf(" => %x", origin)
		case CALLER:
			caller := closure.caller.Address()
			stack.Push(ethutil.BigD(caller))

			self.Printf(" => %x", caller)
		case CALLVALUE:
			value := closure.exe.value

			stack.Push(value)

			self.Printf(" => %v", value)
		case CALLDATALOAD:
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

			self.Printf(" => 0x%x", data)

			stack.Push(ethutil.BigD(data))
		case CALLDATASIZE:
			l := int64(len(closure.Args))
			stack.Push(big.NewInt(l))

			self.Printf(" => %d", l)
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
			if op == EXTCODESIZE {
				addr := stack.Pop().Bytes()

				code = state.GetCode(addr)
			} else {
				code = closure.Code
			}

			l := big.NewInt(int64(len(code)))
			stack.Push(l)

			self.Printf(" => %d", l)
		case CODECOPY, EXTCODECOPY:
			var code []byte
			if op == EXTCODECOPY {
				addr := stack.Pop().Bytes()

				code = state.GetCode(addr)
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

			self.Printf(" => %v", closure.Price)

			// 0x40 range
		case PREVHASH:
			prevHash := self.env.PrevHash()

			stack.Push(ethutil.BigD(prevHash))

			self.Printf(" => 0x%x", prevHash)
		case COINBASE:
			coinbase := self.env.Coinbase()

			stack.Push(ethutil.BigD(coinbase))

			self.Printf(" => 0x%x", coinbase)
		case TIMESTAMP:
			time := self.env.Time()

			stack.Push(big.NewInt(time))

			self.Printf(" => 0x%x", time)
		case NUMBER:
			number := self.env.BlockNumber()

			stack.Push(number)

			self.Printf(" => 0x%x", number.Bytes())
		case DIFFICULTY:
			difficulty := self.env.Difficulty()

			stack.Push(difficulty)

			self.Printf(" => 0x%x", difficulty.Bytes())
		case GASLIMIT:
			stack.Push(self.env.GasLimit())

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

			self.Printf(" => 0x%x", data.Bytes())
		case POP:
			stack.Pop()
		case DUP1, DUP2, DUP3, DUP4, DUP5, DUP6, DUP7, DUP8, DUP9, DUP10, DUP11, DUP12, DUP13, DUP14, DUP15, DUP16:
			n := int(op - DUP1 + 1)
			v := stack.Dupn(n)

			self.Printf(" => [%d] 0x%x", n, stack.Peek().Bytes())

			if OpCode(closure.Get(new(big.Int).Add(pc, ethutil.Big1)).Uint()) == POP && OpCode(closure.Get(new(big.Int).Add(pc, big.NewInt(2))).Uint()) == POP {
				fmt.Println(toValue(v))
			}
		case SWAP1, SWAP2, SWAP3, SWAP4, SWAP5, SWAP6, SWAP7, SWAP8, SWAP9, SWAP10, SWAP11, SWAP12, SWAP13, SWAP14, SWAP15, SWAP16:
			n := int(op - SWAP1 + 2)
			x, y := stack.Swapn(n)

			self.Printf(" => [%d] %x [0] %x", n, x.Bytes(), y.Bytes())
		case LOG0, LOG1, LOG2, LOG3, LOG4:
			n := int(op - LOG0)
			topics := make([][]byte, n)
			mSize, mStart := stack.Pop().Int64(), stack.Pop().Int64()
			data := mem.Geti(mStart, mSize)
			for i := 0; i < n; i++ {
				topics[i] = stack.Pop().Bytes()
			}
			self.env.AddLog(ethstate.Log{closure.Address(), topics, data})
		case MLOAD:
			offset := stack.Pop()
			val := ethutil.BigD(mem.Get(offset.Int64(), 32))
			stack.Push(val)

			self.Printf(" => 0x%x", val.Bytes())
		case MSTORE: // Store the value at stack top-1 in to memory at location stack top
			// Pop value of the stack
			val, mStart := stack.Popn()
			mem.Set(mStart.Int64(), 32, ethutil.BigToBytes(val, 256))

			self.Printf(" => 0x%x", val)
		case MSTORE8:
			off := stack.Pop()
			val := stack.Pop()

			mem.store[off.Int64()] = byte(val.Int64() & 0xff)

			self.Printf(" => [%v] 0x%x", off, val)
		case SLOAD:
			loc := stack.Pop()
			val := ethutil.BigD(state.GetState(closure.Address(), loc.Bytes()))
			stack.Push(val)

			self.Printf(" {0x%x : 0x%x}", loc.Bytes(), val.Bytes())
		case SSTORE:
			val, loc := stack.Popn()
			state.SetState(closure.Address(), loc.Bytes(), val)

			// Debug sessions are allowed to run without message
			if closure.message != nil {
				closure.message.AddStorageChange(loc.Bytes())
			}

			self.Printf(" {0x%x : 0x%x}", loc.Bytes(), val.Bytes())
		case JUMP:

			jump(stack.Pop())

			continue
		case JUMPI:
			cond, pos := stack.Popn()

			if cond.Cmp(ethutil.BigTrue) >= 0 {
				jump(pos)

				continue
			}

		case JUMPDEST:
		case PC:
			stack.Push(pc)
		case MSIZE:
			stack.Push(big.NewInt(int64(mem.Len())))
		case GAS:
			stack.Push(closure.Gas)
			// 0x60 range
		case CREATE:

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
			n := state.GetNonce(closure.Address())
			addr := ethcrypto.CreateAddress(closure.Address(), n)
			state.SetNonce(closure.Address(), n+1)

			self.Printf(" (*) %x", addr).Endl()

			closure.UseGas(closure.Gas)

			msg := NewExecution(self, addr, input, gas, closure.Price, value)
			ret, err := msg.Create(closure)
			if err != nil {
				stack.Push(ethutil.BigFalse)

				// Revert the state as it was before.
				//self.env.State().Set(snapshot)

				self.Printf("CREATE err %v", err)
			} else {
				msg.object.Code = ret

				stack.Push(ethutil.BigD(addr))
			}

			self.Endl()

			// Debug hook
			if self.Dbg != nil {
				self.Dbg.SetCode(closure.Code)
			}
		case CALL, CALLCODE:
			self.Endl()

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

				vmlogger.Debugln(err)
			} else {
				stack.Push(ethutil.BigTrue)

				mem.Set(retOffset.Int64(), retSize.Int64(), ret)
			}
			self.Printf("resume %x", closure.Address())

			// Debug hook
			if self.Dbg != nil {
				self.Dbg.SetCode(closure.Code)
			}

		case RETURN:
			size, offset := stack.Popn()
			ret := mem.Get(offset.Int64(), size.Int64())

			self.Printf(" => (%d) 0x%x", len(ret), ret).Endl()

			return closure.Return(ret), nil
		case SUICIDE:

			receiver := state.GetOrNewStateObject(stack.Pop().Bytes())

			receiver.AddAmount(state.GetBalance(closure.Address()))
			state.Delete(closure.Address())

			fallthrough
		case STOP: // Stop the closure
			self.Endl()

			return closure.Return(nil), nil
		default:
			vmlogger.Debugf("(pc) %-3v Invalid opcode %x\n", pc, op)

			//panic(fmt.Sprintf("Invalid opcode %x", op))
			closure.ReturnGas(big.NewInt(1), nil)

			return closure.Return(nil), fmt.Errorf("Invalid opcode %x", op)
		}

		pc.Add(pc, ethutil.Big1)

		self.Endl()

		if self.Dbg != nil {
			for _, instrNo := range self.Dbg.BreakPoints() {
				if pc.Cmp(big.NewInt(instrNo)) == 0 {
					self.Stepping = true

					if !self.Dbg.BreakHook(prevStep, op, mem, stack, state.GetStateObject(closure.Address())) {
						return nil, nil
					}
				} else if self.Stepping {
					if !self.Dbg.StepHook(prevStep, op, mem, stack, state.GetStateObject(closure.Address())) {
						return nil, nil
					}
				}
			}
		}

	}
}

func (self *DebugVm) Printf(format string, v ...interface{}) VirtualMachine {
	if self.logTy == LogTyPretty {
		self.logStr += fmt.Sprintf(format, v...)
	}

	return self
}

func (self *DebugVm) Endl() VirtualMachine {
	if self.logTy == LogTyPretty {
		vmlogger.Debugln(self.logStr)
		self.logStr = ""
	}

	return self
}

func (self *DebugVm) Env() Environment {
	return self.env
}

func (self *DebugVm) Depth() int {
	return self.depth
}
