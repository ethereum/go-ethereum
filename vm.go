package main

import (
  _"math"
  "math/big"
  "fmt"
  _"strconv"
  _ "encoding/hex"
  "strconv"
)

// Op codes
const (
  oSTOP       int = 0x00
  oADD        int = 0x01
  oMUL        int = 0x02
  oSUB        int = 0x03
  oDIV        int = 0x04
  oSDIV       int = 0x05
  oMOD        int = 0x06
  oSMOD       int = 0x07
  oEXP        int = 0x08
  oNEG        int = 0x09
  oLT         int = 0x0a
  oLE         int = 0x0b
  oGT         int = 0x0c
  oGE         int = 0x0d
  oEQ         int = 0x0e
  oNOT        int = 0x0f
  oMYADDRESS  int = 0x10
  oTXSENDER   int = 0x11


  oPUSH       int = 0x30
  oPOP        int = 0x31
  oLOAD       int = 0x36
)

type OpType int
const (
  tNorm = iota
  tData
  tExtro
  tCrypto
)
type TxCallback func(opType OpType) bool

// Simple push/pop stack mechanism
type Stack struct {
  data []string
}
func NewStack() *Stack {
  return &Stack{}
}
func (st *Stack) Pop() string {
  s := len(st.data)

  str := st.data[s-1]
  st.data = st.data[:s-1]

  return str
}

func (st *Stack) Popn() (*big.Int, *big.Int) {
  s := len(st.data)

  strs := st.data[s-2:]
  st.data = st.data[:s-2]

  return Big(strs[0]), Big(strs[1])
}

func (st *Stack) Push(d string) {
  st.data = append(st.data, d)
}
func (st *Stack) Print() {
  fmt.Println(st.data)
}

type Vm struct {
  // Stack
  stack *Stack
}

func NewVm() *Vm {
  return &Vm{
    stack: NewStack(),
  }
}

func (vm *Vm) ProcContract(tx *Transaction, block *Block, cb TxCallback) {
  // Instruction pointer
  pc := 0

  contract := block.GetContract(tx.Hash())
  if contract == nil {
    fmt.Println("Contract not found")
    return
  }

  Pow256 := BigPow(2, 256)

  //fmt.Printf("#   op   arg\n")
out:
  for {
    // The base big int for all calculations. Use this for any results.
    base := new(big.Int)
    // XXX Should Instr return big int slice instead of string slice?
    // Get the next instruction from the contract
    //op, _, _ := Instr(contract.state.Get(string(Encode(uint32(pc)))))
    op, _, _ := Instr(contract.state.Get(string(NumberToBytes(uint64(pc), 32))))

    if !cb(0) { break }

    if Debug {
      //fmt.Printf("%-3d %-4d\n", pc, op)
    }

    switch op {
    case oADD:
      x, y := vm.stack.Popn()
      // (x + y) % 2 ** 256
      base.Add(x, y)
      base.Mod(base, Pow256)
      // Pop result back on the stack
      vm.stack.Push(base.String())
    case oSUB:
      x, y := vm.stack.Popn()
      // (x - y) % 2 ** 256
      base.Sub(x, y)
      base.Mod(base, Pow256)
      // Pop result back on the stack
      vm.stack.Push(base.String())
    case oMUL:
      x, y := vm.stack.Popn()
      // (x * y) % 2 ** 256
      base.Mul(x, y)
      base.Mod(base, Pow256)
      // Pop result back on the stack
      vm.stack.Push(base.String())
    case oDIV:
      x, y := vm.stack.Popn()
      // floor(x / y)
      base.Div(x, y)
      // Pop result back on the stack
      vm.stack.Push(base.String())
    case oSDIV:
      x, y := vm.stack.Popn()
      // n > 2**255
      if x.Cmp(Pow256) > 0 { x.Sub(Pow256, x) }
      if y.Cmp(Pow256) > 0 { y.Sub(Pow256, y) }
      z := new(big.Int)
      z.Div(x, y)
      if z.Cmp(Pow256) > 0 { z.Sub(Pow256, z) }
      // Push result on to the stack
      vm.stack.Push(z.String())
    case oMOD:
      x, y := vm.stack.Popn()
      base.Mod(x, y)
      vm.stack.Push(base.String())
    case oSMOD:
      x, y := vm.stack.Popn()
      // n > 2**255
      if x.Cmp(Pow256) > 0 { x.Sub(Pow256, x) }
      if y.Cmp(Pow256) > 0 { y.Sub(Pow256, y) }
      z := new(big.Int)
      z.Mod(x, y)
      if z.Cmp(Pow256) > 0 { z.Sub(Pow256, z) }
      // Push result on to the stack
      vm.stack.Push(z.String())
    case oEXP:
      x, y := vm.stack.Popn()
      base.Exp(x, y, Pow256)

      vm.stack.Push(base.String())
    case oNEG:
      base.Sub(Pow256, Big(vm.stack.Pop()))
      vm.stack.Push(base.String())
    case oLT:
      x, y := vm.stack.Popn()
      // x < y
      if x.Cmp(y) < 0 {
        vm.stack.Push("1")
      } else {
        vm.stack.Push("0")
      }
    case oLE:
      x, y := vm.stack.Popn()
      // x <= y
      if x.Cmp(y) < 1 {
        vm.stack.Push("1")
      } else {
        vm.stack.Push("0")
      }
    case oGT:
      x, y := vm.stack.Popn()
      // x > y
      if x.Cmp(y) > 0 {
        vm.stack.Push("1")
      } else {
        vm.stack.Push("0")
      }
    case oGE:
      x, y := vm.stack.Popn()
      // x >= y
      if x.Cmp(y) > -1 {
        vm.stack.Push("1")
      } else {
        vm.stack.Push("0")
      }
    case oNOT:
      x, y := vm.stack.Popn()
      // x != y
      if x.Cmp(y) != 0 {
        vm.stack.Push("1")
      } else {
        vm.stack.Push("0")
      }
    case oMYADDRESS:
      vm.stack.Push(string(tx.Hash()))
    case oTXSENDER:
      vm.stack.Push(tx.sender)
    case oPUSH:
      // Get the next entry and pushes the value on the stack
      pc++
      vm.stack.Push(contract.state.Get(string(NumberToBytes(uint64(pc), 32))))
    case oPOP:
      // Pop current value of the stack
      vm.stack.Pop()
    case oLOAD:
      // Load instruction X on the stack
      i, _ := strconv.Atoi(vm.stack.Pop())
      vm.stack.Push(contract.state.Get(string(NumberToBytes(uint64(i), 32))))
    case oSTOP:
      break out
    }
    pc++
  }

  vm.stack.Print()
}
