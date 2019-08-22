//
// hamming distance calculations in Go
//
// https://github.com/steakknife/hamming
//
// Copyright © 2014, 2015, 2016 Barry Allard
//
// MIT license
//

#include "textflag.h"

TEXT ·CountBitsInt8PopCnt(SB),NOSPLIT,$0
	JMP        ·CountBitsBytePopCnt(SB)

TEXT ·CountBitsInt16PopCnt(SB),NOSPLIT,$0
	JMP        ·CountBitsUint16PopCnt(SB)

TEXT ·CountBitsInt32PopCnt(SB),NOSPLIT,$0
	JMP        ·CountBitsUint32PopCnt(SB)

TEXT ·CountBitsInt64PopCnt(SB),NOSPLIT,$0
	JMP        ·CountBitsUint64PopCnt(SB)

TEXT ·CountBitsBytePopCnt(SB),NOSPLIT,$0
	JMP        ·CountBitsUint8PopCnt(SB)

TEXT ·CountBitsRunePopCnt(SB),NOSPLIT,$0
	JMP        ·CountBitsUint32PopCnt(SB)

TEXT ·CountBitsUint8PopCnt(SB),NOSPLIT,$0
	XORQ       AX, AX
	MOVB       x+0(FP), AX
	POPCNTQ    AX, AX	
	MOVQ       AX, ret+8(FP)
	RET

TEXT ·CountBitsUint16PopCnt(SB),NOSPLIT,$0
	XORQ       AX, AX
	MOVW       x+0(FP), AX
	POPCNTQ    AX, AX	
	MOVQ       AX, ret+8(FP)
	RET

TEXT ·CountBitsUint32PopCnt(SB),NOSPLIT,$0
	XORQ       AX, AX
	MOVL       x+0(FP), AX
	POPCNTQ    AX, AX	
	MOVQ       AX, ret+8(FP)
	RET

TEXT ·CountBitsUint64PopCnt(SB),NOSPLIT,$0
	POPCNTQ    x+0(FP), AX
	MOVQ       AX, ret+8(FP)
	RET

// func hasPopCnt() (ret bool) 
TEXT ·HasPopCnt(SB),NOSPLIT,$0
	MOVL       $1, AX
	CPUID
	SHRL       $23, CX  // bit 23: Advanced Bit Manipulation Bit (ABM) -> POPCNTQ
	ANDL       $1, CX
	MOVB       CX, ret+0(FP)
	RET
