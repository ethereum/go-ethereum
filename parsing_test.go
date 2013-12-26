package main

import (
  "testing"
  "math"
)

func TestCompile(t *testing.T) {
  instr, err := CompileInstr("SET 10 1")

  if err != nil {
    t.Error("Failed compiling instruction")
  }

  calc := (67 + 10 * 256 + 1 * int64(math.Pow(256,2)))
  if Big(instr).Int64() != calc {
    t.Error("Expected", calc, ", got:", instr)
  }
}

func TestValidInstr(t *testing.T) {
  op, args, err := Instr("68163")
  if err != nil {
    t.Error("Error decoding instruction")
  }

  if op != oSET {
    t.Error("Expected op to be 43, got:", op)
  }

  if args[0] != "10" {
    t.Error("Expect args[0] to be 10, got:", args[0])
  }

  if args[1] != "1" {
    t.Error("Expected args[1] to be 1, got:", args[1])
  }
}

func TestInvalidInstr(t *testing.T) {
}

