package ethchain

import (
	_ "bytes"
	_ "fmt"
	"github.com/ethereum/eth-go/ethutil"
	_ "github.com/obscuren/secp256k1-go"
	"log"
	_ "math"
	"math/big"
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
	origin      []byte
	blockNumber uint64
	prevHash    []byte
	coinbase    []byte
	time        int64
	diff        *big.Int
	txData      []string
}

func NewVm(state *State, vars RuntimeVars) *Vm {
	return &Vm{vars: vars, state: state}
}

var Pow256 = ethutil.BigPow(2, 256)

func (vm *Vm) RunClosure(closure *Closure) []byte {
	// If the amount of gas supplied is less equal to 0
	if closure.GetGas().Cmp(big.NewInt(0)) <= 0 {
		// TODO Do something
	}

	// Memory for the current closure
	mem := &Memory{}
	// New stack (should this be shared?)
	stack := NewStack()
	// Instruction pointer
	pc := int64(0)
	// Current step count
	step := 0
	// The base for all big integer arithmetic
	base := new(big.Int)

	if ethutil.Config.Debug {
		ethutil.Config.Log.Debugf("#   op\n")
	}

	for {
		step++
		// Get the memory location of pc
		val := closure.GetMem(pc)
		// Get the opcode (it must be an opcode!)
		op := OpCode(val.Uint())
		if ethutil.Config.Debug {
			ethutil.Config.Log.Debugf("%-3d %-4s", pc, op.String())
		}

		// TODO Get each instruction cost properly
		fee := new(big.Int)
		fee.Add(fee, big.NewInt(1000))

		if closure.GetGas().Cmp(fee) < 0 {
			return closure.Return(nil)
		}

		switch op {
		case oLOG:
			stack.Print()
			mem.Print()
		case oSTOP: // Stop the closure
			return closure.Return(nil)

		// 0x20 range
		case oADD:
			x, y := stack.Popn()
			// (x + y) % 2 ** 256
			base.Add(x, y)
			base.Mod(base, Pow256)
			// Pop result back on the stack
			stack.Push(base)
		case oSUB:
			x, y := stack.Popn()
			// (x - y) % 2 ** 256
			base.Sub(x, y)
			base.Mod(base, Pow256)
			// Pop result back on the stack
			stack.Push(base)
		case oMUL:
			x, y := stack.Popn()
			// (x * y) % 2 ** 256
			base.Mul(x, y)
			base.Mod(base, Pow256)
			// Pop result back on the stack
			stack.Push(base)
		case oDIV:
			x, y := stack.Popn()
			// floor(x / y)
			base.Div(x, y)
			// Pop result back on the stack
			stack.Push(base)
		case oSDIV:
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
			x, y := stack.Popn()
			base.Mod(x, y)
			stack.Push(base)
		case oSMOD:
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
			x, y := stack.Popn()
			base.Exp(x, y, Pow256)

			stack.Push(base)
		case oNEG:
			base.Sub(Pow256, stack.Pop())
			stack.Push(base)
		case oLT:
			x, y := stack.Popn()
			// x < y
			if x.Cmp(y) < 0 {
				stack.Push(ethutil.BigTrue)
			} else {
				stack.Push(ethutil.BigFalse)
			}
		case oGT:
			x, y := stack.Popn()
			// x > y
			if x.Cmp(y) > 0 {
				stack.Push(ethutil.BigTrue)
			} else {
				stack.Push(ethutil.BigFalse)
			}
		case oNOT:
			x, y := stack.Popn()
			// x != y
			if x.Cmp(y) != 0 {
				stack.Push(ethutil.BigTrue)
			} else {
				stack.Push(ethutil.BigFalse)
			}

		// 0x10 range
		case oAND:
		case oOR:
		case oXOR:
		case oBYTE:

		// 0x20 range
		case oSHA3:

		// 0x30 range
		case oADDRESS:
		case oBALANCE:
		case oORIGIN:
		case oCALLER:
		case oCALLVALUE:
		case oCALLDATA:
			offset := stack.Pop()
			mem.Set(offset.Int64(), int64(len(closure.Args)), closure.Args)
		case oCALLDATASIZE:
		case oRETURNDATASIZE:
		case oTXGASPRICE:

		// 0x40 range
		case oPREVHASH:
		case oPREVNONCE:
		case oCOINBASE:
		case oTIMESTAMP:
		case oNUMBER:
		case oDIFFICULTY:
		case oGASLIMIT:

		// 0x50 range
		case oPUSH: // Push PC+1 on to the stack
			pc++
			val := closure.GetMem(pc).BigInt()
			stack.Push(val)
		case oPOP:
		case oDUP:
		case oSWAP:
		case oMLOAD:
			offset := stack.Pop()
			stack.Push(ethutil.BigD(mem.Get(offset.Int64(), 32)))
		case oMSTORE: // Store the value at stack top-1 in to memory at location stack top
			// Pop value of the stack
			val, mStart := stack.Popn()
			mem.Set(mStart.Int64(), 32, ethutil.BigToBytes(val, 256))
		case oMSTORE8:
		case oSLOAD:
		case oSSTORE:
		case oJUMP:
		case oJUMPI:
		case oPC:
		case oMSIZE:

		// 0x60 range
		case oCALL:
			// Pop return size and offset
			retSize, retOffset := stack.Popn()
			// Pop input size and offset
			inSize, inOffset := stack.Popn()
			// Get the arguments from the memory
			args := mem.Get(inOffset.Int64(), inSize.Int64())
			// Pop gas and value of the stack.
			gas, value := stack.Popn()
			// Closure addr
			addr := stack.Pop()
			// Fetch the contract which will serve as the closure body
			contract := vm.state.GetContract(addr.Bytes())
			// Create a new callable closure
			closure := NewClosure(closure, contract, vm.state, gas, value)
			// Executer the closure and get the return value (if any)
			ret := closure.Call(vm, args)

			mem.Set(retOffset.Int64(), retSize.Int64(), ret)
		case oRETURN:
			size, offset := stack.Popn()
			ret := mem.Get(offset.Int64(), size.Int64())

			return closure.Return(ret)
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
			ethutil.Config.Log.Debugln("Invalid opcode", op)
		}

		pc++
	}
}

/*
// Old VM code
func (vm *Vm) Process(contract *Contract, state *State, vars RuntimeVars) {
	vm.mem = make(map[string]*big.Int)
	vm.stack = NewStack()

	addr := vars.address // tx.Hash()[12:]
	// Instruction pointer
	pc := int64(0)

	if contract == nil {
		fmt.Println("Contract not found")
		return
	}

	Pow256 := ethutil.BigPow(2, 256)

	if ethutil.Config.Debug {
		ethutil.Config.Log.Debugf("#   op\n")
	}

	stepcount := 0
	totalFee := new(big.Int)

out:
	for {
		stepcount++
		// The base big int for all calculations. Use this for any results.
		base := new(big.Int)
		val := contract.GetMem(pc)
		//fmt.Printf("%x = %d, %v %x\n", r, len(r), v, nb)
		op := OpCode(val.Uint())

		var fee *big.Int = new(big.Int)
		var fee2 *big.Int = new(big.Int)
		if stepcount > 16 {
			fee.Add(fee, StepFee)
		}

		// Calculate the fees
		switch op {
		case oSSTORE:
			y, x := vm.stack.Peekn()
			val := contract.Addr(ethutil.BigToBytes(x, 256))
			if val.IsEmpty() && len(y.Bytes()) > 0 {
				fee2.Add(DataFee, StoreFee)
			} else {
				fee2.Sub(DataFee, StoreFee)
			}
		case oSLOAD:
			fee.Add(fee, StoreFee)
		case oEXTRO, oBALANCE:
			fee.Add(fee, ExtroFee)
		case oSHA256, oRIPEMD160, oECMUL, oECADD, oECSIGN, oECRECOVER, oECVALID:
			fee.Add(fee, CryptoFee)
		case oMKTX:
			fee.Add(fee, ContractFee)
		}

		tf := new(big.Int).Add(fee, fee2)
		if contract.Amount.Cmp(tf) < 0 {
			fmt.Println("Insufficient fees to continue running the contract", tf, contract.Amount)
			break
		}
		// Add the fee to the total fee. It's subtracted when we're done looping
		totalFee.Add(totalFee, tf)

		if ethutil.Config.Debug {
			ethutil.Config.Log.Debugf("%-3d %-4s", pc, op.String())
		}

		switch op {
		case oSTOP:
			fmt.Println("")
			break out
		case oADD:
			x, y := vm.stack.Popn()
			// (x + y) % 2 ** 256
			base.Add(x, y)
			base.Mod(base, Pow256)
			// Pop result back on the stack
			vm.stack.Push(base)
		case oSUB:
			x, y := vm.stack.Popn()
			// (x - y) % 2 ** 256
			base.Sub(x, y)
			base.Mod(base, Pow256)
			// Pop result back on the stack
			vm.stack.Push(base)
		case oMUL:
			x, y := vm.stack.Popn()
			// (x * y) % 2 ** 256
			base.Mul(x, y)
			base.Mod(base, Pow256)
			// Pop result back on the stack
			vm.stack.Push(base)
		case oDIV:
			x, y := vm.stack.Popn()
			// floor(x / y)
			base.Div(x, y)
			// Pop result back on the stack
			vm.stack.Push(base)
		case oSDIV:
			x, y := vm.stack.Popn()
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
			vm.stack.Push(z)
		case oMOD:
			x, y := vm.stack.Popn()
			base.Mod(x, y)
			vm.stack.Push(base)
		case oSMOD:
			x, y := vm.stack.Popn()
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
			vm.stack.Push(z)
		case oEXP:
			x, y := vm.stack.Popn()
			base.Exp(x, y, Pow256)

			vm.stack.Push(base)
		case oNEG:
			base.Sub(Pow256, vm.stack.Pop())
			vm.stack.Push(base)
		case oLT:
			x, y := vm.stack.Popn()
			// x < y
			if x.Cmp(y) < 0 {
				vm.stack.Push(ethutil.BigTrue)
			} else {
				vm.stack.Push(ethutil.BigFalse)
			}
		case oLE:
			x, y := vm.stack.Popn()
			// x <= y
			if x.Cmp(y) < 1 {
				vm.stack.Push(ethutil.BigTrue)
			} else {
				vm.stack.Push(ethutil.BigFalse)
			}
		case oGT:
			x, y := vm.stack.Popn()
			// x > y
			if x.Cmp(y) > 0 {
				vm.stack.Push(ethutil.BigTrue)
			} else {
				vm.stack.Push(ethutil.BigFalse)
			}
		case oGE:
			x, y := vm.stack.Popn()
			// x >= y
			if x.Cmp(y) > -1 {
				vm.stack.Push(ethutil.BigTrue)
			} else {
				vm.stack.Push(ethutil.BigFalse)
			}
		case oNOT:
			x, y := vm.stack.Popn()
			// x != y
			if x.Cmp(y) != 0 {
				vm.stack.Push(ethutil.BigTrue)
			} else {
				vm.stack.Push(ethutil.BigFalse)
			}
		case oMYADDRESS:
			vm.stack.Push(ethutil.BigD(addr))
		case oTXSENDER:
			vm.stack.Push(ethutil.BigD(vars.sender))
		case oTXVALUE:
			vm.stack.Push(vars.txValue)
		case oTXDATAN:
			vm.stack.Push(big.NewInt(int64(len(vars.txData))))
		case oTXDATA:
			v := vm.stack.Pop()
			// v >= len(data)
			if v.Cmp(big.NewInt(int64(len(vars.txData)))) >= 0 {
				vm.stack.Push(ethutil.Big("0"))
			} else {
				vm.stack.Push(ethutil.Big(vars.txData[v.Uint64()]))
			}
		case oBLK_PREVHASH:
			vm.stack.Push(ethutil.BigD(vars.prevHash))
		case oBLK_COINBASE:
			vm.stack.Push(ethutil.BigD(vars.coinbase))
		case oBLK_TIMESTAMP:
			vm.stack.Push(big.NewInt(vars.time))
		case oBLK_NUMBER:
			vm.stack.Push(big.NewInt(int64(vars.blockNumber)))
		case oBLK_DIFFICULTY:
			vm.stack.Push(vars.diff)
		case oBASEFEE:
			// e = 10^21
			e := big.NewInt(0).Exp(big.NewInt(10), big.NewInt(21), big.NewInt(0))
			d := new(big.Rat)
			d.SetInt(vars.diff)
			c := new(big.Rat)
			c.SetFloat64(0.5)
			// d = diff / 0.5
			d.Quo(d, c)
			// base = floor(d)
			base.Div(d.Num(), d.Denom())

			x := new(big.Int)
			x.Div(e, base)

			// x = floor(10^21 / floor(diff^0.5))
			vm.stack.Push(x)
		case oSHA256, oSHA3, oRIPEMD160:
			// This is probably save
			// ceil(pop / 32)
			length := int(math.Ceil(float64(vm.stack.Pop().Uint64()) / 32.0))
			// New buffer which will contain the concatenated popped items
			data := new(bytes.Buffer)
			for i := 0; i < length; i++ {
				// Encode the number to bytes and have it 32bytes long
				num := ethutil.NumberToBytes(vm.stack.Pop().Bytes(), 256)
				data.WriteString(string(num))
			}

			if op == oSHA256 {
				vm.stack.Push(base.SetBytes(ethutil.Sha256Bin(data.Bytes())))
			} else if op == oSHA3 {
				vm.stack.Push(base.SetBytes(ethutil.Sha3Bin(data.Bytes())))
			} else {
				vm.stack.Push(base.SetBytes(ethutil.Ripemd160(data.Bytes())))
			}
		case oECMUL:
			y := vm.stack.Pop()
			x := vm.stack.Pop()
			//n := vm.stack.Pop()

			//if ethutil.Big(x).Cmp(ethutil.Big(y)) {
			data := new(bytes.Buffer)
			data.WriteString(x.String())
			data.WriteString(y.String())
			if secp256k1.VerifyPubkeyValidity(data.Bytes()) == 1 {
				// TODO
			} else {
				// Invalid, push infinity
				vm.stack.Push(ethutil.Big("0"))
				vm.stack.Push(ethutil.Big("0"))
			}
			//} else {
			//	// Invalid, push infinity
			//	vm.stack.Push("0")
			//	vm.stack.Push("0")
			//}

		case oECADD:
		case oECSIGN:
		case oECRECOVER:
		case oECVALID:
		case oPUSH:
			pc++
			vm.stack.Push(contract.GetMem(pc).BigInt())
		case oPOP:
			// Pop current value of the stack
			vm.stack.Pop()
		case oDUP:
			// Dup top stack
			x := vm.stack.Pop()
			vm.stack.Push(x)
			vm.stack.Push(x)
		case oSWAP:
			// Swap two top most values
			x, y := vm.stack.Popn()
			vm.stack.Push(y)
			vm.stack.Push(x)
		case oMLOAD:
			x := vm.stack.Pop()
			vm.stack.Push(vm.mem[x.String()])
		case oMSTORE:
			x, y := vm.stack.Popn()
			vm.mem[x.String()] = y
		case oSLOAD:
			// Load the value in storage and push it on the stack
			x := vm.stack.Pop()
			// decode the object as a big integer
			decoder := contract.Addr(x.Bytes())
			if !decoder.IsNil() {
				vm.stack.Push(decoder.BigInt())
			} else {
				vm.stack.Push(ethutil.BigFalse)
			}
		case oSSTORE:
			// Store Y at index X
			y, x := vm.stack.Popn()
			addr := ethutil.BigToBytes(x, 256)
			fmt.Printf(" => %x (%v) @ %v", y.Bytes(), y, ethutil.BigD(addr))
			contract.SetAddr(addr, y)
			//contract.State().Update(string(idx), string(y))
		case oJMP:
			x := vm.stack.Pop().Int64()
			// Set pc to x - 1 (minus one so the incrementing at the end won't effect it)
			pc = x
			pc--
		case oJMPI:
			x := vm.stack.Pop()
			// Set pc to x if it's non zero
			if x.Cmp(ethutil.BigFalse) != 0 {
				pc = x.Int64()
				pc--
			}
		case oIND:
			vm.stack.Push(big.NewInt(int64(pc)))
		case oEXTRO:
			memAddr := vm.stack.Pop()
			contractAddr := vm.stack.Pop().Bytes()

			// Push the contract's memory on to the stack
			vm.stack.Push(contractMemory(state, contractAddr, memAddr))
		case oBALANCE:
			// Pushes the balance of the popped value on to the stack
			account := state.GetAccount(vm.stack.Pop().Bytes())
			vm.stack.Push(account.Amount)
		case oMKTX:
			addr, value := vm.stack.Popn()
			from, length := vm.stack.Popn()

			makeInlineTx(addr.Bytes(), value, from, length, contract, state)
		case oSUICIDE:
			recAddr := vm.stack.Pop().Bytes()
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
		default:
			fmt.Printf("Invalid OPCODE: %x\n", op)
		}
		ethutil.Config.Log.Debugln("")
		//vm.stack.Print()
		pc++
	}

	state.UpdateContract(addr, contract)
}
*/

func makeInlineTx(addr []byte, value, from, length *big.Int, contract *Contract, state *State) {
	ethutil.Config.Log.Debugf(" => creating inline tx %x %v %v %v", addr, value, from, length)
	j := int64(0)
	dataItems := make([]string, int(length.Uint64()))
	for i := from.Int64(); i < length.Int64(); i++ {
		dataItems[j] = contract.GetMem(j).Str()
		j++
	}

	tx := NewTransaction(addr, value, dataItems)
	if tx.IsContract() {
		contract := MakeContract(tx, state)
		state.UpdateContract(contract)
	} else {
		account := state.GetAccount(tx.Recipient)
		account.Amount.Add(account.Amount, tx.Value)
		state.UpdateAccount(tx.Recipient, account)
	}
}

// Returns an address from the specified contract's address
func contractMemory(state *State, contractAddr []byte, memAddr *big.Int) *big.Int {
	contract := state.GetContract(contractAddr)
	if contract == nil {
		log.Panicf("invalid contract addr %x", contractAddr)
	}
	val := state.trie.Get(memAddr.String())

	// decode the object as a big integer
	decoder := ethutil.NewValueFromBytes([]byte(val))
	if decoder.IsNil() {
		return ethutil.BigFalse
	}

	return decoder.BigInt()
}
