package ethvm

import (
	"container/list"
	"fmt"
	"math/big"

	"github.com/ethereum/eth-go/ethcrypto"
	"github.com/ethereum/eth-go/ethstate"
	"github.com/ethereum/eth-go/ethutil"
)

type Debugger interface {
	BreakHook(step int, op OpCode, mem *Memory, stack *Stack, object *ethstate.StateObject) bool
	StepHook(step int, op OpCode, mem *Memory, stack *Stack, object *ethstate.StateObject) bool
	BreakPoints() []int64
	SetCode(byteCode []byte)
}

type Vm struct {
	env Environment

	Verbose bool

	logTy  byte
	logStr string

	err error

	// Debugging
	Dbg Debugger

	BreakPoints []int64
	Stepping    bool
	Fn          string

	Recoverable bool

	queue *list.List
}

type Environment interface {
	State() *ethstate.State

	Origin() []byte
	BlockNumber() *big.Int
	PrevHash() []byte
	Coinbase() []byte
	Time() int64
	Difficulty() *big.Int
	Value() *big.Int
	BlockHash() []byte
}

type Object interface {
	GetStorage(key *big.Int) *ethutil.Value
	SetStorage(key *big.Int, value *ethutil.Value)
}

func New(env Environment) *Vm {
	lt := LogTyPretty
	if ethutil.Config.Diff {
		lt = LogTyDiff
	}

	return &Vm{env: env, logTy: lt, Recoverable: true, queue: list.New()}
}

func calcMemSize(off, l *big.Int) *big.Int {
	if l.Cmp(ethutil.Big0) == 0 {
		return ethutil.Big0
	}

	return new(big.Int).Add(off, l)
}

// Simple helper
func u256(n int64) *big.Int {
	return big.NewInt(n)
}

func (self *Vm) RunClosure(closure *Closure) (ret []byte, err error) {
	if self.Recoverable {
		// Recover from any require exception
		defer func() {
			if r := recover(); r != nil {
				ret = closure.Return(nil)
				err = fmt.Errorf("%v", r)
				vmlogger.Errorln("vm err", err)
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

	vmlogger.Debugf("(%s) %x gas: %v (d) %x\n", self.Fn, closure.Address(), closure.Gas, closure.Args)

	var (
		op OpCode

		mem      = &Memory{}
		stack    = NewStack()
		pc       = big.NewInt(0)
		step     = 0
		prevStep = 0
		require  = func(m int) {
			if stack.Len() < m {
				panic(fmt.Sprintf("%04v (%v) stack err size = %d, required = %d", pc, op, stack.Len(), m))
			}
		}
	)

	for {
		prevStep = step
		// The base for all big integer arithmetic
		base := new(big.Int)

		step++
		// Get the memory location of pc
		val := closure.Get(pc)
		// Get the opcode (it must be an opcode!)
		op = OpCode(val.Uint())

		// XXX Leave this Println intact. Don't change this to the log system.
		// Used for creating diffs between implementations
		if self.logTy == LogTyDiff {
			switch op {
			case STOP, RETURN, SUICIDE:
				closure.object.EachStorage(func(key string, value *ethutil.Value) {
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
		case CALL, CALLSTATELESS:
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

		self.Printf("(pc) %-3d -o- %-14s", pc, op.String())
		self.Printf(" (g) %-3v (%v)", gas, closure.Gas)

		mem.Resize(newMemSize.Uint64())

		switch op {
		case LOG:
			stack.Print()
			mem.Print()
			// 0x20 range
		case ADD:
			require(2)
			x, y := stack.Popn()
			self.Printf(" %v + %v", y, x)

			base.Add(y, x)

			ensure256(base)

			self.Printf(" = %v", base)
			// Pop result back on the stack
			stack.Push(base)
		case SUB:
			require(2)
			x, y := stack.Popn()
			self.Printf(" %v - %v", y, x)

			base.Sub(y, x)

			ensure256(base)

			self.Printf(" = %v", base)
			// Pop result back on the stack
			stack.Push(base)
		case MUL:
			require(2)
			x, y := stack.Popn()
			self.Printf(" %v * %v", y, x)

			base.Mul(y, x)

			ensure256(base)

			self.Printf(" = %v", base)
			// Pop result back on the stack
			stack.Push(base)
		case DIV:
			require(2)
			x, y := stack.Popn()
			self.Printf(" %v / %v", y, x)

			if x.Cmp(ethutil.Big0) != 0 {
				base.Div(y, x)
			}

			ensure256(base)

			self.Printf(" = %v", base)
			// Pop result back on the stack
			stack.Push(base)
		case SDIV:
			require(2)
			x, y := stack.Popn()
			self.Printf(" %v / %v", y, x)

			if x.Cmp(ethutil.Big0) != 0 {
				base.Div(y, x)
			}

			ensure256(base)

			self.Printf(" = %v", base)
			// Pop result back on the stack
			stack.Push(base)
		case MOD:
			require(2)
			x, y := stack.Popn()

			self.Printf(" %v %% %v", y, x)

			base.Mod(y, x)

			ensure256(base)

			self.Printf(" = %v", base)
			stack.Push(base)
		case SMOD:
			require(2)
			x, y := stack.Popn()

			self.Printf(" %v %% %v", y, x)

			base.Mod(y, x)

			ensure256(base)

			self.Printf(" = %v", base)
			stack.Push(base)

		case EXP:
			require(2)
			x, y := stack.Popn()

			self.Printf(" %v ** %v", y, x)

			base.Exp(y, x, Pow256)

			ensure256(base)

			self.Printf(" = %v", base)

			stack.Push(base)
		case NEG:
			require(1)
			base.Sub(Pow256, stack.Pop())
			stack.Push(base)
		case LT:
			require(2)
			x, y := stack.Popn()
			self.Printf(" %v < %v", y, x)
			// x < y
			if y.Cmp(x) < 0 {
				stack.Push(ethutil.BigTrue)
			} else {
				stack.Push(ethutil.BigFalse)
			}
		case GT:
			require(2)
			x, y := stack.Popn()
			self.Printf(" %v > %v", y, x)

			// x > y
			if y.Cmp(x) > 0 {
				stack.Push(ethutil.BigTrue)
			} else {
				stack.Push(ethutil.BigFalse)
			}

		case SLT:
			require(2)
			x, y := stack.Popn()
			self.Printf(" %v < %v", y, x)
			// x < y
			if y.Cmp(x) < 0 {
				stack.Push(ethutil.BigTrue)
			} else {
				stack.Push(ethutil.BigFalse)
			}
		case SGT:
			require(2)
			x, y := stack.Popn()
			self.Printf(" %v > %v", y, x)

			// x > y
			if y.Cmp(x) > 0 {
				stack.Push(ethutil.BigTrue)
			} else {
				stack.Push(ethutil.BigFalse)
			}

		case EQ:
			require(2)
			x, y := stack.Popn()
			self.Printf(" %v == %v", y, x)

			// x == y
			if x.Cmp(y) == 0 {
				stack.Push(ethutil.BigTrue)
			} else {
				stack.Push(ethutil.BigFalse)
			}
		case NOT:
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
			self.Printf(" %v & %v", y, x)

			stack.Push(base.And(y, x))
		case OR:
			require(2)
			x, y := stack.Popn()
			self.Printf(" %v | %v", y, x)

			stack.Push(base.Or(y, x))
		case XOR:
			require(2)
			x, y := stack.Popn()
			self.Printf(" %v ^ %v", y, x)

			stack.Push(base.Xor(y, x))
		case BYTE:
			require(2)
			val, th := stack.Popn()
			if th.Cmp(big.NewInt(32)) < 0 && th.Cmp(big.NewInt(int64(len(val.Bytes())))) < 0 {
				byt := big.NewInt(int64(ethutil.LeftPadBytes(val.Bytes(), 32)[th.Int64()]))
				stack.Push(byt)

				self.Printf(" => 0x%x", byt.Bytes())
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

			ensure256(base)

			self.Printf(" = %v", base)

			stack.Push(base)
		case MULMOD:
			require(3)

			x := stack.Pop()
			y := stack.Pop()
			z := stack.Pop()

			base.Mul(x, y)
			base.Mod(base, z)

			ensure256(base)

			self.Printf(" = %v", base)

			stack.Push(base)

			// 0x20 range
		case SHA3:
			require(2)
			size, offset := stack.Popn()
			data := ethcrypto.Sha3Bin(mem.Get(offset.Int64(), size.Int64()))

			stack.Push(ethutil.BigD(data))

			self.Printf(" => %x", data)
			// 0x30 range
		case ADDRESS:
			stack.Push(ethutil.BigD(closure.Address()))

			self.Printf(" => %x", closure.Address())
		case BALANCE:
			require(1)

			addr := stack.Pop().Bytes()
			balance := self.env.State().GetBalance(addr)

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
			value := self.env.Value()

			stack.Push(value)

			self.Printf(" => %v", value)
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
			if op == EXTCODECOPY {
				addr := stack.Pop().Bytes()

				code = self.env.State().GetCode(addr)
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

			self.Printf(" => 0x%x", data.Bytes())
		case POP:
			require(1)
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
		case MLOAD:
			require(1)
			offset := stack.Pop()
			val := ethutil.BigD(mem.Get(offset.Int64(), 32))
			stack.Push(val)

			self.Printf(" => 0x%x", val.Bytes())
		case MSTORE: // Store the value at stack top-1 in to memory at location stack top
			require(2)
			// Pop value of the stack
			val, mStart := stack.Popn()
			mem.Set(mStart.Int64(), 32, ethutil.BigToBytes(val, 256))

			self.Printf(" => 0x%x", val)
		case MSTORE8:
			require(2)
			off := stack.Pop()
			val := stack.Pop()

			mem.store[off.Int64()] = byte(val.Int64() & 0xff)

			self.Printf(" => [%v] 0x%x", off, val)
		case SLOAD:
			require(1)
			loc := stack.Pop()
			val := closure.GetStorage(loc)

			stack.Push(val.BigInt())

			self.Printf(" {0x%x : 0x%x}", loc.Bytes(), val.Bytes())
		case SSTORE:
			require(2)
			val, loc := stack.Popn()
			closure.SetStorage(loc, ethutil.NewValue(val))

			closure.message.AddStorageChange(loc.Bytes())

			self.Printf(" {0x%x : 0x%x}", loc.Bytes(), val.Bytes())
		case JUMP:
			require(1)
			pc = stack.Pop()
			// Reduce pc by one because of the increment that's at the end of this for loop
			self.Printf(" ~> %v", pc).Endl()

			continue
		case JUMPI:
			require(2)
			cond, pos := stack.Popn()
			if cond.Cmp(ethutil.BigTrue) >= 0 {
				pc = pos

				self.Printf(" ~> %v (t)", pc).Endl()

				continue
			} else {
				self.Printf(" (f)")
			}
		case PC:
			stack.Push(pc)
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
				snapshot = self.env.State().Copy()
			)

			// Generate a new address
			addr := ethcrypto.CreateAddress(closure.Address(), closure.object.Nonce)
			//for i := uint64(0); self.env.State().GetStateObject(addr) != nil; i++ {
			//	ethcrypto.CreateAddress(closure.Address(), closure.object.Nonce+i)
			//}
			closure.object.Nonce++

			self.Printf(" (*) %x", addr).Endl()

			closure.UseGas(closure.Gas)

			msg := NewMessage(self, addr, input, gas, closure.Price, value)
			ret, err := msg.Exec(addr, closure)
			if err != nil {
				stack.Push(ethutil.BigFalse)

				// Revert the state as it was before.
				self.env.State().Set(snapshot)

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
		case CALL, CALLSTATELESS:
			require(7)

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

			snapshot := self.env.State().Copy()

			var executeAddr []byte
			if op == CALLSTATELESS {
				executeAddr = closure.Address()
			} else {
				executeAddr = addr.Bytes()
			}

			msg := NewMessage(self, executeAddr, args, gas, closure.Price, value)
			ret, err := msg.Exec(addr.Bytes(), closure)
			if err != nil {
				stack.Push(ethutil.BigFalse)

				self.env.State().Set(snapshot)
			} else {
				stack.Push(ethutil.BigTrue)

				mem.Set(retOffset.Int64(), retSize.Int64(), ret)
			}

			// Debug hook
			if self.Dbg != nil {
				self.Dbg.SetCode(closure.Code)
			}

		case POST:
			require(5)

			self.Endl()

			gas := stack.Pop()
			// Pop gas and value of the stack.
			value, addr := stack.Popn()
			// Pop input size and offset
			inSize, inOffset := stack.Popn()
			// Get the arguments from the memory
			args := mem.Get(inOffset.Int64(), inSize.Int64())

			msg := NewMessage(self, addr.Bytes(), args, gas, closure.Price, value)

			msg.Postpone()
		case RETURN:
			require(2)
			size, offset := stack.Popn()
			ret := mem.Get(offset.Int64(), size.Int64())

			self.Printf(" => (%d) 0x%x", len(ret), ret).Endl()

			return closure.Return(ret), nil
		case SUICIDE:
			require(1)

			receiver := self.env.State().GetOrNewStateObject(stack.Pop().Bytes())

			receiver.AddAmount(closure.object.Balance)

			closure.object.MarkForDeletion()

			fallthrough
		case STOP: // Stop the closure
			self.Endl()

			return closure.Return(nil), nil
		default:
			vmlogger.Debugf("(pc) %-3v Invalid opcode %x\n", pc, op)

			//panic(fmt.Sprintf("Invalid opcode %x", op))

			return closure.Return(nil), fmt.Errorf("Invalid opcode %x", op)
		}

		pc.Add(pc, ethutil.Big1)

		self.Endl()

		if self.Dbg != nil {
			for _, instrNo := range self.Dbg.BreakPoints() {
				if pc.Cmp(big.NewInt(instrNo)) == 0 {
					self.Stepping = true

					if !self.Dbg.BreakHook(prevStep, op, mem, stack, closure.Object()) {
						return nil, nil
					}
				} else if self.Stepping {
					if !self.Dbg.StepHook(prevStep, op, mem, stack, closure.Object()) {
						return nil, nil
					}
				}
			}
		}

	}
}

func (self *Vm) Queue() *list.List {
	return self.queue
}

func (self *Vm) Printf(format string, v ...interface{}) *Vm {
	if self.Verbose && self.logTy == LogTyPretty {
		self.logStr += fmt.Sprintf(format, v...)
	}

	return self
}

func (self *Vm) Endl() *Vm {
	if self.Verbose && self.logTy == LogTyPretty {
		vmlogger.Debugln(self.logStr)
		self.logStr = ""
	}

	return self
}

func ensure256(x *big.Int) {
	//max, _ := big.NewInt(0).SetString("115792089237316195423570985008687907853269984665640564039457584007913129639936", 0)
	//if x.Cmp(max) >= 0 {
	d := big.NewInt(1)
	d.Lsh(d, 256).Sub(d, big.NewInt(1))
	x.And(x, d)
	//}

	// Could have done this with an OR, but big ints are costly.

	if x.Cmp(new(big.Int)) < 0 {
		x.SetInt64(0)
	}
}

type Message struct {
	vm                *Vm
	closure           *Closure
	address, input    []byte
	gas, price, value *big.Int
	object            *ethstate.StateObject
}

func NewMessage(vm *Vm, address, input []byte, gas, gasPrice, value *big.Int) *Message {
	return &Message{vm: vm, address: address, input: input, gas: gas, price: gasPrice, value: value}
}

func (self *Message) Postpone() {
	self.vm.queue.PushBack(self)
}

func (self *Message) Addr() []byte {
	return self.address
}

func (self *Message) Exec(codeAddr []byte, caller ClosureRef) (ret []byte, err error) {
	queue := self.vm.queue
	self.vm.queue = list.New()

	defer func() {
		if err == nil {
			queue.PushBackList(self.vm.queue)
		}

		self.vm.queue = queue
	}()

	msg := self.vm.env.State().Manifest().AddMessage(&ethstate.Message{
		To: self.address, From: caller.Address(),
		Input:  self.input,
		Origin: self.vm.env.Origin(),
		Block:  self.vm.env.BlockHash(), Timestamp: self.vm.env.Time(), Coinbase: self.vm.env.Coinbase(), Number: self.vm.env.BlockNumber(),
		Value: self.value,
	})

	object := caller.Object()
	if object.Balance.Cmp(self.value) < 0 {
		caller.ReturnGas(self.gas, self.price)

		err = fmt.Errorf("Insufficient funds to transfer value. Req %v, has %v", self.value, object.Balance)
	} else {
		stateObject := self.vm.env.State().GetOrNewStateObject(self.address)
		self.object = stateObject

		caller.Object().SubAmount(self.value)
		stateObject.AddAmount(self.value)

		// Retrieve the executing code
		code := self.vm.env.State().GetCode(codeAddr)

		// Create a new callable closure
		c := NewClosure(msg, caller, stateObject, code, self.gas, self.price)
		// Executer the closure and get the return value (if any)
		ret, _, err = c.Call(self.vm, self.input)

		msg.Output = ret

		return ret, err
	}

	return
}

// Mainly used for print variables and passing to Print*
func toValue(val *big.Int) interface{} {
	// Let's assume a string on right padded zero's
	b := val.Bytes()
	if b[0] != 0 && b[len(b)-1] == 0x0 && b[len(b)-2] == 0x0 {
		return string(b)
	}

	return val
}
