//go:build tamago && riscv64

#include "textflag.h"

// Entry point - this is where execution starts
TEXT cpuinit(SB),NOSPLIT|NOFRAME,$0
	// Clear A0/A1 registers so runtime does not think we have
	// argc/argv arguments
	MOV	$0, A0
	MOV	$0, A1
	// Jump to tamago runtime for RISC-V
	JMP	runtime·rt0_riscv64_tamago(SB)

// hwinit0 is called by the runtime during initialization
TEXT runtime·hwinit0(SB),NOSPLIT|NOFRAME,$0
	// Hardware initialization before runtime setup
	// For zkVM, nothing special needed here
	RET
