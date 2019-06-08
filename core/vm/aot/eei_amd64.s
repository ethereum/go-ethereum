// Copyright 2019 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

#include "textflag.h"

TEXT ·ImportGetCallDataSize(SB),$0-8
	LEAQ	ethereumGetCallDataSize<>(SB), AX
	MOVQ	AX, ret+0(FP)
	RET

TEXT ethereumGetCallDataSize<>(SB),NOSPLIT,$0
	// Get pointer to the contract info area into r13
	MOVQ    -0x20(R15), R13
	MOVQ	0x10(R13), AX
	RET

TEXT ·ImportUseGas(SB),$0-8
	LEAQ	ethereumUseGas<>(SB), AX
	MOVQ	AX, ret+0(FP)
	RET

TEXT ethereumUseGas<>(SB),NOSPLIT,$0
	// Get pointer to the contract info area into r13
	MOVQ    -0x20(R15), R13

	// Disable gas accounting as it is currently implemented
	// before calling the precompile.
	//MOVQ    8(SP), AX		// Gas required
	//MOVQ	0x8(R13), CX	// Gas left
	//CMPQ	AX, CX
	//JA		oog
	
	//SUBQ	AX, CX
	//MOVQ	CX, 0x8(R13)
	XORQ	AX, AX
	XORQ	CX, CX
	RET

	oog:
	// Set gas value to -1
	XORQ	AX, AX
	NOTQ	AX
	MOVQ	AX, 0x8(R13)
	// Recover the saved value of the stack
	MOVQ    -0x20(R15), SI
	MOVQ	(SI), SP
	RET

TEXT ·ImportCallDataCopy(SB),$0-8
	LEAQ	ethereumCallDataCopy<>(SB), AX
	MOVQ	AX, ret+0(FP)
	RET

TEXT ethereumCallDataCopy<>(SB),NOSPLIT,$0
	// Get pointer to the contract info area into r13
	MOVQ    -0x20(R15), R13

	// Get pointer to input data
	MOVQ	R13, SI
	ADDQ	$0x18, SI		// start of input buffer
	MOVQ	0x10(SP), AX	// rax = input data offset
	ADDQ	AX, SI

	// Load and check the size of data to be
	// copied to the destination buffer
	MOVQ	0x8(SP), CX		// rcx = number of bytes
	ADDQ	CX, AX			// rax = input buffer + nbytes
	MOVQ	0x10(R13), R12	// r12 = max size
	CMPQ	AX, R12
	JA		eei_error

	// Skip copy loop if there are no data to copy
	CMPQ	AX, $0
	JZ		calldatacopy_end

	// Load address of the destination buffer
	MOVQ	0x18(SP), DI
	ADDQ	R14, DI

	copy:
	MOVB	(SI), AX
	MOVB	AX, (DI)
	ADDQ	$1, SI
	ADDQ	$1, DI
	LOOP	copy
calldatacopy_end:
	RET

	eei_error:
	// Recover the saved value of the stack
	MOVQ    -0x20(R15), SI
	MOVQ	(SI), SP
	RET

TEXT ·ImportFinish(SB),$0-8
	LEAQ	ethereumFinish<>(SB), AX
	MOVQ	AX, ret+0(FP)
	RET

// Voir si je peux mettre $0-16 et ret+0(FP), ret+8(FP)
TEXT ethereumFinish<>(SB),NOSPLIT,$0
	// Get both arguments from the stack, before it
	// is changed back to its previous value
	MOVQ	8(SP), AX
	MOVQ	16(SP), CX

	// Recover the saved value of the stack
	MOVQ    -0x20(R15), SI
	MOVQ	(SI), SP

	// Store the buffer addresses and size at
	// the location where go expects both parameters
	// to be stored, on the initial stack.
	MOVQ	CX, 0x28(SP)
	MOVQ	AX, 0x30(SP)
	RET

TEXT ·ImportRevert(SB),$0-8
	LEAQ	ethereumRevert<>(SB), AX
	MOVQ	AX, ret+0(FP)
	RET

TEXT ethereumRevert<>(SB),NOSPLIT,$0
	// Get both arguments from the stack, before it
	// is changed back to its previous value
	MOVQ	8(SP), AX
	MOVQ	16(SP), CX

	// Recover the saved value of the stack
	MOVQ    -0x20(R15), SI
	MOVQ	(SI), SP

	// Store the buffer addresses and size at
	// the location where go expects both parameters
	// to be stored, on the initial stack.
	MOVQ	CX, 0x28(SP)
	MOVQ	AX, 0x30(SP)
	RET

TEXT ·ImportGrowMemoryHandler(SB),$0-8
	LEAQ	growMemoryHandler<>(SB), AX
	MOVQ	AX, ret+0(FP)
	RET

TEXT growMemoryHandler<>(SB),NOSPLIT,$0-8
	//CALL main·GrowMemory+0(FP)
	RET
