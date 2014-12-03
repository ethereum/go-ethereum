package vm

import (
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/crypto"
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

type Options struct {
	Address, Caller   []byte
	Data              []byte
	Code              []byte
	Value, Gas, Price *big.Int
}

func NewDebugVm(env Environment) *DebugVm {
	lt := LogTyPretty
	if ethutil.Config.Diff {
		lt = LogTyDiff
	}

	return &DebugVm{env: env, logTy: lt, Recoverable: false}
}

//func (self *DebugVm) RunClosure(closure *Closure) (ret []byte, err error) {
func (self *DebugVm) Run(call Options) (ret []byte, gas *big.Int, err error) {
	// Don't bother with the execution if there's no code.
	if len(call.Code) == 0 {
		return nil, new(big.Int), nil
	}

	self.depth++

	if self.Recoverable {
		// Recover from any require exception
		defer func() {
			if r := recover(); r != nil {
				self.Endl()

				gas = new(big.Int)
				err = fmt.Errorf("%v", r)

			}
		}()
	}

	gas = new(big.Int).Set(opt.Gas)
	var (
		op OpCode

		destinations = analyseJumpDests(call.Code)
		mem          = NewMemory()
		stack        = NewStack()
		pc           = big.NewInt(0)
		step         = 0
		prevStep     = 0
		//statedb      = self.env.State()
		require = func(m int) {
			if stack.Len() < m {
				panic(fmt.Sprintf("%04v (%v) stack err size = %d, required = %d", pc, op, stack.Len(), m))
			}
		}

		useGas = func(amount *big.Int) bool {
			if amount.Cmp(gas) > 0 {
				return false
			}

			gas.Sub(gas, amount)

			return true
		}

		jump = func(from, to *big.Int) {
			p := to.Uint64()

			self.Printf(" ~> %v", to)
			// Return to start
			if p == 0 {
				pc = big.NewInt(0)
			} else {
				nop := OpCode(call.GetOp(p))
				if !(nop == JUMPDEST || destinations[from.Int64()] != nil) {
					panic(fmt.Sprintf("JUMP missed JUMPDEST (%v) %v", nop, p))
				} else if nop == JUMP || nop == JUMPI {
					panic(fmt.Sprintf("not allowed to JUMP(I) in to JUMP"))
				}

				pc = to

			}

			self.Endl()
		}
	)

	vmlogger.Debugf("(%d) %x gas: %v (d) %x\n", self.depth, call.Address, gas, call.Data)

	for {
		prevStep = step
		// The base for all big integer arithmetic
		base := new(big.Int)

		step++
		// Get the memory location of pc
		op = call.GetOp(pc.Uint64())

		// XXX Leave this Println intact. Don't change this to the log system.
		// Used for creating diffs between implementations
		/*
			if self.logTy == LogTyDiff {
				switch op {
				case STOP, RETURN, SUICIDE:
					statedb.GetStateObject(closure.Address()).EachStorage(func(key string, value *ethutil.Value) {
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
		*/

		reqGas := new(big.Int)
		addStepGasUsage := func(amount *big.Int) {
			if amount.Cmp(ethutil.Big0) >= 0 {
				reqGas.Add(reqGas, amount)
			}
		}

		addStepGasUsage(GasStep)

		var newMemSize *big.Int = ethutil.Big0
		// Stack Check, memory resize & gas phase
		switch op {
		// Stack checks only
		case ISZERO, CALLDATALOAD, POP, JUMP, NOT: // 1
			require(1)
		case ADD, SUB, DIV, SDIV, MOD, SMOD, LT, GT, SLT, SGT, EQ, AND, OR, XOR, BYTE: // 2
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

			gas.Set(GasLog)
			addStepGasUsage(new(big.Int).Mul(big.NewInt(int64(n)), GasLog))

			mSize, mStart := stack.Peekn()
			addStepGasUsage(mSize)

			newMemSize = calcMemSize(mStart, mSize)
		case EXP:
			require(2)

			exp := new(big.Int).Set(stack.data[stack.Len()-2])
			nbytes := 0
			for exp.Cmp(ethutil.Big0) > 0 {
				nbytes += 1
				exp.Rsh(exp, 8)
			}
			gas.Set(big.NewInt(int64(nbytes + 1)))
		// Gas only
		case STOP:
			reqGas.Set(ethutil.Big0)
		case SUICIDE:
			require(1)

			reqGas.Set(ethutil.Big0)
		case SLOAD:
			require(1)

			reqGas.Set(GasSLoad)
		// Memory resize & Gas
		case SSTORE:
			require(2)

			var mult *big.Int
			y, x := stack.Peekn()
			val := ethutil.BigD(self.env.GetState(x.Bytes())) //closure.GetStorage(x)
			if val.BigInt().Cmp(ethutil.Big0) == 0 && len(y.Bytes()) > 0 {
				// 0 => non 0
				mult = ethutil.Big3
			} else if val.BigInt().Cmp(ethutil.Big0) != 0 && len(y.Bytes()) == 0 {
				//statedb.Refund(closure.caller.Address(), GasSStoreRefund, closure.Price)
				self.env.Refund(call.Caller, GasSStoreRefund, call.Price)

				mult = ethutil.Big0
			} else {
				// non 0 => non 0
				mult = ethutil.Big1
			}
			reqGas.Set(new(big.Int).Mul(mult, GasSStore))
		case BALANCE:
			require(1)
			reqGas.Set(GasBalance)
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

			reqGas.Set(GasSha)

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
			reqGas.Set(GasCall)
			addStepGasUsage(stack.data[stack.Len()-1])

			x := calcMemSize(stack.data[stack.Len()-6], stack.data[stack.Len()-7])
			y := calcMemSize(stack.data[stack.Len()-4], stack.data[stack.Len()-5])

			newMemSize = ethutil.BigMax(x, y)
		case CREATE:
			require(3)
			reqGas.Set(GasCreate)

			newMemSize = calcMemSize(stack.data[stack.Len()-2], stack.data[stack.Len()-3])
		}

		if newMemSize.Cmp(ethutil.Big0) > 0 {
			newMemSize.Add(newMemSize, u256(31))
			newMemSize.Div(newMemSize, u256(32))
			newMemSize.Mul(newMemSize, u256(32))

			switch op {
			// Additional gas usage on *CODPY
			case CALLDATACOPY, CODECOPY, EXTCODECOPY:
				addStepGasUsage(new(big.Int).Div(newMemSize, u256(32)))
			}

			if newMemSize.Cmp(u256(int64(mem.Len()))) > 0 {
				memGasUsage := new(big.Int).Sub(newMemSize, u256(int64(mem.Len())))
				memGasUsage.Mul(GasMemory, memGasUsage)
				memGasUsage.Div(memGasUsage, u256(32))

				addStepGasUsage(memGasUsage)

			}

		}

		self.Printf("(pc) %-3d -o- %-14s", pc, op.String())
		self.Printf(" (m) %-4d (s) %-4d (g) %-3v (%v)", mem.Len(), stack.Len(), reqGas, gas)

		if !useGas(regGas) {
			self.Endl()

			return nil, new(big.Int), OOG(reqGas, gas)
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
		case SIGNEXTEND:
			back := stack.Pop().Uint64()
			if back < 31 {
				bit := uint(back*8 + 7)
				num := stack.Pop()
				mask := new(big.Int).Lsh(ethutil.Big1, bit)
				mask.Sub(mask, ethutil.Big1)
				if ethutil.BitTest(num, int(bit)) {
					num.Or(num, mask.Not(mask))
				} else {
					num.And(num, mask)
				}

				num = U256(num)

				self.Printf(" = %v", num)

				stack.Push(num)
			}
		case NOT:
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
		case ISZERO:
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
			data := crypto.Sha3(mem.Get(offset.Int64(), size.Int64()))

			stack.Push(ethutil.BigD(data))

			self.Printf(" => %x", data)
			// 0x30 range
		case ADDRESS:
			//stack.Push(ethutil.BigD(closure.Address()))
			stack.Push(ethutil.BigD(call.Address))

			self.Printf(" => %x", call.Address)
		case BALANCE:

			addr := stack.Pop().Bytes()
			//balance := statedb.GetBalance(addr)
			balance := self.env.GetBalance(addr)

			stack.Push(balance)

			self.Printf(" => %v (%x)", balance, addr)
		case ORIGIN:
			origin := self.env.Origin()

			stack.Push(ethutil.BigD(origin))

			self.Printf(" => %x", origin)
		case CALLER:
			//caller := closure.caller.Address()
			//stack.Push(ethutil.BigD(caller))
			stack.Push(call.Caller)

			self.Printf(" => %x", call.Caller)
		case CALLVALUE:
			//value := closure.exe.value

			stack.Push(call.Value)

			self.Printf(" => %v", call.Value)
		case CALLDATALOAD:
			var (
				offset  = stack.Pop()
				data    = make([]byte, 32)
				lenData = big.NewInt(int64(len(call.Data)))
			)

			if lenData.Cmp(offset) >= 0 {
				length := new(big.Int).Add(offset, ethutil.Big32)
				length = ethutil.BigMin(length, lenData)

				copy(data, call.Data[offset.Int64():length.Int64()])
			}

			self.Printf(" => 0x%x", data)

			stack.Push(ethutil.BigD(data))
		case CALLDATASIZE:
			l := int64(len(call.Data))
			stack.Push(big.NewInt(l))

			self.Printf(" => %d", l)
		case CALLDATACOPY:
			var (
				size = int64(len(call.Data))
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

			code := call.Data[cOff : cOff+l]

			mem.Set(mOff, l, code)

			self.Printf(" => [%v, %v, %v] %x", mOff, cOff, l, code[cOff:cOff+l])
		case CODESIZE, EXTCODESIZE:
			var code []byte
			if op == EXTCODESIZE {
				addr := stack.Pop().Bytes()

				self.env.GetCode(addr)
			} else {
				code = call.Code
			}

			l := big.NewInt(int64(len(code)))
			stack.Push(l)

			self.Printf(" => %d", l)
		case CODECOPY, EXTCODECOPY:
			var code []byte
			if op == EXTCODECOPY {
				addr := stack.Pop().Bytes()

				code = self.env.GetCode(addr)
			} else {
				code = call.Code
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

			self.Printf(" => [%v, %v, %v] %x", mOff, cOff, l, code[cOff:cOff+l])
		case GASPRICE:
			stack.Push(call.Price)

			self.Printf(" => %v", call.Price)

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
			a := uint64(op) - uint64(PUSH1) + 1
			pc.Add(pc, ethutil.Big1)
			data := call.Get(pc.Uint64(), a) //closure.Gets(pc, a)
			val := ethutil.BigD(data)
			// Push value to stack
			stack.Push(val)
			pc.Add(pc, big.NewInt(int64(a)-1))

			step += uint64(op) - uint64(PUSH1) + 1

			self.Printf(" => 0x%x", data)
		case POP:
			stack.Pop()
		case DUP1, DUP2, DUP3, DUP4, DUP5, DUP6, DUP7, DUP8, DUP9, DUP10, DUP11, DUP12, DUP13, DUP14, DUP15, DUP16:
			n := int(op - DUP1 + 1)
			v := stack.Dupn(n)

			self.Printf(" => [%d] 0x%x", n, stack.Peek().Bytes())
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
				topics[i] = ethutil.LeftPadBytes(stack.Pop().Bytes(), 32)
			}

			//log := &state.Log{closure.Address(), topics, data}
			self.env.AddLog(call.Address, topics, data)

			self.Printf(" => %v", log)
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
			val := ethutil.BigD(self.env.GetState(call.Address, loc.Bytes()))
			stack.Push(val)

			self.Printf(" {0x%x : 0x%x}", loc.Bytes(), val.Bytes())
		case SSTORE:
			val, loc := stack.Popn()
			self.env.SetState(call.Address, loc.Bytes(), val.Bytes())
			//statedb.SetState(closure.Address(), loc.Bytes(), val)

			//closure.message.AddStorageChange(loc.Bytes())

			self.Printf(" {0x%x : 0x%x}", loc.Bytes(), val.Bytes())
		case JUMP:

			jump(pc, stack.Pop())

			continue
		case JUMPI:
			cond, pos := stack.Popn()

			if cond.Cmp(ethutil.BigTrue) >= 0 {
				jump(pc, pos)

				continue
			}

		case JUMPDEST:
		case PC:
			stack.Push(pc)
		case MSIZE:
			stack.Push(big.NewInt(int64(mem.Len())))
		case GAS:
			stack.Push(call.Gas)
			// 0x60 range
		case CREATE:
			var (
				err          error
				value        = stack.Pop()
				size, offset = stack.Popn()
				input        = mem.Get(offset.Int64(), size.Int64())
				gas          = new(big.Int).Set(call.Gas)

				// Snapshot the current stack so we are able to
				// revert back to it later.
				//snapshot = self.env.State().Copy()
			)

			// Generate a new address
			//n := statedb.GetNonce(closure.Address())
			//addr := crypto.CreateAddress(closure.Address(), n)
			//statedb.SetNonce(closure.Address(), n+1)
			n := self.env.GetNonce(call.Address)
			addr := crypto.CreateAddress(call.Address, n)
			self.env.SetNonce(call.Address, n+1)

			self.Printf(" (*) %x", addr).Endl()

			//closure.UseGas(closure.Gas)

			msg := NewExecution(self, addr, input, gas, call.Price, value)
			ret, lgas, err := msg.Create(call.Address)
			if err != nil {
				stack.Push(ethutil.BigFalse)

				// Revert the state as it was before.
				//self.env.State().Set(snapshot)

				self.Printf("CREATE err %v", err)
			} else {
				msg.object.Code = ret

				stack.Push(ethutil.BigD(addr))
			}

			gas = lgas

			self.Endl()
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
				executeAddr = call.Address //closure.Address()
			} else {
				executeAddr = addr.Bytes()
			}

			msg := NewExecution(self, executeAddr, args, gas, call.Price, value)
			ret, err := msg.Exec(addr.Bytes(), closure)
			if err != nil {
				stack.Push(ethutil.BigFalse)

				vmlogger.Debugln(err)
			} else {
				stack.Push(ethutil.BigTrue)

				mem.Set(retOffset.Int64(), retSize.Int64(), ret)
			}
			self.Printf("resume %x", closure.Address())

		case RETURN:
			size, offset := stack.Popn()
			ret := mem.Get(offset.Int64(), size.Int64())

			self.Printf(" => (%d) 0x%x", len(ret), ret).Endl()

			return ret, gas, nil

			//return closure.Return(ret), gas, nil
		case SUICIDE:
			//receiver := statedb.GetOrNewStateObject(stack.Pop().Bytes())
			//receiver.AddAmount(statedb.GetBalance(closure.Address()))
			//statedb.Delete(closure.Address())

			self.env.AddBalance(stack.Pop().Bytes(), self.env.Balance(call.Address))
			self.env.DeleteAccount(call.Address)

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
