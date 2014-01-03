package main

import (
  "fmt"
  "strings"
  "errors"
  "math/big"
  "strconv"
)

// Op codes
var OpCodes = map[string]string{
  "STOP":       "0",
  "PUSH":       "48",  // 0x30
  "POP":        "49",  // 0x31
  "LOAD":       "54",  // 0x36

  /* OLD VM OPCODES
  "ADD":        "16",  // 0x10
  "SUB":        "17",  // 0x11
  "MUL":        "18",  // 0x12
  "DIV":        "19",  // 0x13
  "SDIV":       "20",  // 0x14
  "MOD":        "21",  // 0x15
  "SMOD":       "22",  // 0x16
  "EXP":        "23",  // 0x17
  "NEG":        "24",  // 0x18
  "LT":         "32",  // 0x20
  "LE":         "33",  // 0x21
  "GT":         "34",  // 0x22
  "GE":         "35",  // 0x23
  "EQ":         "36",  // 0x24
  "NOT":        "37",  // 0x25
  "SHA256":     "48",  // 0x30
  "RIPEMD160":  "49",  // 0x31
  "ECMUL":      "50",  // 0x32
  "ECADD":      "51",  // 0x33
  "SIGN":       "52",  // 0x34
  "RECOVER":    "53",  // 0x35
  "COPY":       "64",  // 0x40
  "ST":         "65",  // 0x41
  "LD":         "66",  // 0x42
  "SET":        "67",  // 0x43
  "JMP":        "80",  // 0x50
  "JMPI":       "81",  // 0x51
  "IND":        "82",  // 0x52
  "EXTRO":      "96",  // 0x60
  "BALANCE":    "97",  // 0x61
  "MKTX":       "112", // 0x70
  "DATA":       "128", // 0x80
  "DATAN":      "129", // 0x81
  "MYADDRESS":  "144", // 0x90
  "BLKHASH":    "145", // 0x91
  "COINBASE":   "146", // 0x92
  "SUICIDE":    "255", // 0xff
  */
}


func CompileInstr(s string) (string, error) {
  tokens := strings.Split(s, " ")
  if OpCodes[tokens[0]] == "" {
    return s, errors.New(fmt.Sprintf("OP not found: %s", tokens[0]))
  }

  code := OpCodes[tokens[0]] // Replace op codes with the proper numerical equivalent
  op := new(big.Int)
  op.SetString(code, 0)

  args := make([]*big.Int, 6)
  for i, val := range tokens[1:len(tokens)] {
    num := new(big.Int)
    num.SetString(val, 0)
    args[i] = num
  }

  // Big int equation = op + x * 256 + y * 256**2 + z * 256**3 + a * 256**4 + b * 256**5 + c * 256**6
  base := new(big.Int)
  x := new(big.Int)
  y := new(big.Int)
  z := new(big.Int)
  a := new(big.Int)
  b := new(big.Int)
  c := new(big.Int)

  if args[0] != nil { x.Mul(args[0], big.NewInt(256)) }
  if args[1] != nil { y.Mul(args[1], BigPow(256, 2)) }
  if args[2] != nil { z.Mul(args[2], BigPow(256, 3)) }
  if args[3] != nil { a.Mul(args[3], BigPow(256, 4)) }
  if args[4] != nil { b.Mul(args[4], BigPow(256, 5)) }
  if args[5] != nil { c.Mul(args[5], BigPow(256, 6)) }

  base.Add(op, x)
  base.Add(base, y)
  base.Add(base, z)
  base.Add(base, a)
  base.Add(base, b)
  base.Add(base, c)

  return base.String(), nil
}

func Instr(instr string) (int, []string, error) {
  base := new(big.Int)
  base.SetString(instr, 0)

  args := make([]string, 7)
  for i := 0; i < 7; i++ {
    // int(int(val) / int(math.Pow(256,float64(i)))) % 256
    exp := BigPow(256, i)
    num := new(big.Int)
    num.Div(base, exp)

    args[i] = num.Mod(num, big.NewInt(256)).String()
  }
  op, _ := strconv.Atoi(args[0])

  return op, args[1:7], nil
}
