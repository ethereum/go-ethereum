//go:build tamago && riscv64

#include "textflag.h"

#define INPUT_ADDR  0x90000000
#define OUTPUT_ADDR 0xa0010000

// hwinit1 is called after basic runtime initialization
// We set A0/A1 here instead of in the emulator
TEXT runtime·hwinit1(SB),NOSPLIT|NOFRAME,$0
	// Set A0 to INPUT_ADDR
	MOV	$INPUT_ADDR, A0
	
	// Set A1 to OUTPUT_ADDR  
	MOV	$OUTPUT_ADDR, A1
	
	RET
