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
  "STOP":        "0",
  "ADD":         "1",
  "MUL":         "2",
  "SUB":         "3",
  "DIV":         "4",
  "SDIV":        "5",
  "MOD":         "6",
  "SMOD":        "7",
  "EXP":         "8",
  "NEG":         "9",
  "LT":         "10",
  "LE":         "11",
  "GT":         "12",
  "GE":         "13",
  "EQ":         "14",
  "NOT":        "15",
  "MYADDRESS":  "16",
  "TXSENDER":   "17",


  "PUSH":       "48",
  "POP":        "49",
  "LOAD":       "54",
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
