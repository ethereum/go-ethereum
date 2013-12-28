package main

import (
  "math"
  "math/big"
  "fmt"
  "strconv"
  _ "encoding/hex"
)

// Op codes
const (
  oSTOP       int = 0x00
  oADD        int = 0x10
  oSUB        int = 0x11
  oMUL        int = 0x12
  oDIV        int = 0x13
  oSDIV       int = 0x14
  oMOD        int = 0x15
  oSMOD       int = 0x16
  oEXP        int = 0x17
  oNEG        int = 0x18
  oLT         int = 0x20
  oLE         int = 0x21
  oGT         int = 0x22
  oGE         int = 0x23
  oEQ         int = 0x24
  oNOT        int = 0x25
  oSHA256     int = 0x30
  oRIPEMD160  int = 0x31
  oECMUL      int = 0x32
  oECADD      int = 0x33
  oSIGN       int = 0x34
  oRECOVER    int = 0x35
  oCOPY       int = 0x40
  oST         int = 0x41
  oLD         int = 0x42
  oSET        int = 0x43
  oJMP        int = 0x50
  oJMPI       int = 0x51
  oIND        int = 0x52
  oEXTRO      int = 0x60
  oBALANCE    int = 0x61
  oMKTX       int = 0x70
  oDATA       int = 0x80
  oDATAN      int = 0x81
  oMYADDRESS  int = 0x90
  oSUICIDE    int = 0xff
)

type OpType int
const (
  tNorm = iota
  tData
  tExtro
  tCrypto
)
type TxCallback func(opType OpType) bool

type Vm struct {
  // Memory stack
  stack map[string]string
  memory map[string]map[string]string
}

func NewVm() *Vm {
  fmt.Println("init Ethereum VM")

  stackSize := uint(256)
  fmt.Println("stack size =", stackSize)

  return &Vm{
    stack: make(map[string]string),
    memory: make(map[string]map[string]string),
  }
}

func (vm *Vm) RunTransaction(tx *Transaction, cb TxCallback) {
  fmt.Printf(`
# processing Tx (%v)
# fee = %f, ops = %d, sender = %s, value = %d
`, tx.addr, float32(tx.fee) / 1e8, len(tx.data), tx.sender, tx.value)

  vm.stack = make(map[string]string)
  vm.stack["0"] = tx.sender
  vm.stack["1"] = "100"  //int(tx.value)
  vm.stack["1"] = "1000" //int(tx.fee)
  // Stack pointer
  stPtr := 0

  //vm.memory[tx.addr] = make([]int, 256)
  vm.memory[tx.addr] = make(map[string]string)

  // Define instruction 'accessors' for the instruction, which makes it more readable
  // also called register values, shorthanded as Rx/y/z. Memory address are shorthanded as Mx/y/z.
  // Instructions are shorthanded as Ix/y/z
  x := 0; y := 1; z := 2; //a := 3; b := 4; c := 5
out:
  for stPtr < len(tx.data) {
    // The base big int for all calculations. Use this for any results.
    base := new(big.Int)
    // XXX Should Instr return big int slice instead of string slice?
    op, args, _ := Instr(tx.data[stPtr])

    fmt.Printf("%-3d %d %v\n", stPtr, op, args)

    opType     := OpType(tNorm)
    // Determine the op type (used for calculating fees by the block manager)
    switch op {
    case oEXTRO, oBALANCE:
      opType = tExtro
    case oSHA256, oRIPEMD160, oECMUL, oECADD: // TODO add rest
      opType = tCrypto
    }

    // If the callback yielded a negative result abort execution
    if !cb(opType) { break out }

    nptr := stPtr
    switch op {
    case oSTOP:
      fmt.Println("exiting (oSTOP), idx =", nptr)

      break out
    case oADD:
      // (Rx + Ry) % 2 ** 256
      base.Add(Big(vm.stack[args[ x ]]), Big(vm.stack[args[ y ]]))
      base.Mod(base, big.NewInt(int64(math.Pow(2, 256))))
      // Set the result to Rz
      vm.stack[args[ z ]] = base.String()
    case oSUB:
      // (Rx - Ry) % 2 ** 256
      base.Sub(Big(vm.stack[args[ x ]]), Big(vm.stack[args[ y ]]))
      base.Mod(base, big.NewInt(int64(math.Pow(2, 256))))
      // Set the result to Rz
      vm.stack[args[ z ]] = base.String()
    case oMUL:
      // (Rx * Ry) % 2 ** 256
      base.Mul(Big(vm.stack[args[ x ]]), Big(vm.stack[args[ y ]]))
      base.Mod(base, big.NewInt(int64(math.Pow(2, 256))))
      // Set the result to Rz
      vm.stack[args[ z ]] = base.String()
    case oDIV:
      // floor(Rx / Ry)
      base.Div(Big(vm.stack[args[ x ]]), Big(vm.stack[args[ y ]]))
      // Set the result to Rz
      vm.stack[args[ z ]] = base.String()
    case oSET:
      // Set the (numeric) value at Iy to Rx
      vm.stack[args[ x ]] = args[ y ]
    case oLD:
      // Load the value at Mx to Ry
      vm.stack[args[ y ]] = vm.memory[tx.addr][vm.stack[args[ x ]]]
    case oLT:
      cmp := Big(vm.stack[args[ x ]]).Cmp( Big(vm.stack[args[ y ]]) )
      // Set the result as "boolean" value to Rz
      if  cmp < 0 { // a < b
        vm.stack[args[ z ]] = "1"
      } else {
        vm.stack[args[ z ]] = "0"
      }
    case oJMP:
      // Set the instruction pointer to the value at Rx
      ptr, _ := strconv.Atoi( vm.stack[args[ x ]] )
      nptr = ptr
    case oJMPI:
      // Set the instruction pointer to the value at Ry if Rx yields true
      if vm.stack[args[ x ]] != "0" {
        ptr, _ := strconv.Atoi( vm.stack[args[ y ]] )
        nptr = ptr
      }
    default:
      fmt.Println("Error op", op)
      break
    }

    if stPtr == nptr {
      stPtr++
    } else {
      stPtr = nptr
      fmt.Println("... JMP", nptr, "...")
    }
  }
  fmt.Println("# finished processing Tx\n")
}
