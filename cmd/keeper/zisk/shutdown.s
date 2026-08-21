//go:build tamago && riscv64 && zisk

#include "textflag.h"

#define ARCH_ID_ZISK 0xFFFEEEE
#define QEMU_EXIT_ADDR 0x100000
#define QEMU_EXIT_CODE 0x5555

// Shutdown triggers ZisK exit via ecall with a7=93
TEXT ·Shutdown(SB),NOSPLIT|NOFRAME,$0
	// Read marchid CSR into A0 (CSRRS x10, 0xF12, x0)
	WORD	$0xF1202573
	MOV	$ARCH_ID_ZISK, A1	// Load ZisK arch ID
	BNE	A0, A1, qemu_exit	// If not, jump to qemu_exit

	MOV	$93, A7		// CAUSE_EXIT = 93
	ECALL			// System call to exit
	RET			// Should never reach here

qemu_exit:
	MOV	$QEMU_EXIT_CODE, T0	// Load the exit code for QEMU
	MOV	$QEMU_EXIT_ADDR, T1	// Load the exit address for QEMU
	ECALL			// System call to exit QEMU
	RET			// Should never reach here
