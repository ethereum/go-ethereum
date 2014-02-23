package ethutil

import (
	"math"
	"testing"
)

func TestCompile(t *testing.T) {
	instr, err := CompileInstr("PUSH")

	if err != nil {
		t.Error("Failed compiling instruction")
	}

	calc := (48 + 0*256 + 0*int64(math.Pow(256, 2)))
	if BigD(instr).Int64() != calc {
		t.Error("Expected", calc, ", got:", instr)
	}
}

func TestValidInstr(t *testing.T) {
	/*
	  op, args, err := Instr("68163")
	  if err != nil {
	    t.Error("Error decoding instruction")
	  }
	*/

}

func TestInvalidInstr(t *testing.T) {
}
