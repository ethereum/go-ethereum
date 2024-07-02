// Copyright 2016 Tom Thorogood. All rights reserved.
// Use of this source code is governed by a
// Modified BSD License license that can be found in
// the LICENSE file.
//
// Copyright 2005-2016, Wojciech Muła. All rights reserved.
// Use of this source code is governed by a
// Simplified BSD License license that can be found in
// the LICENSE file.
//
// This file is auto-generated - do not modify

// +build amd64,!gccgo,!appengine

#include "textflag.h"

DATA decodeBase<>+0x00(SB)/8, $0x3030303030303030
DATA decodeBase<>+0x08(SB)/8, $0x3030303030303030
DATA decodeBase<>+0x10(SB)/8, $0x2727272727272727
DATA decodeBase<>+0x18(SB)/8, $0x2727272727272727
GLOBL decodeBase<>(SB),RODATA,$32

DATA decodeToLower<>+0x00(SB)/8, $0x2020202020202020
DATA decodeToLower<>+0x08(SB)/8, $0x2020202020202020
GLOBL decodeToLower<>(SB),RODATA,$16

DATA decodeHigh<>+0x00(SB)/8, $0x0e0c0a0806040200
DATA decodeHigh<>+0x08(SB)/8, $0xffffffffffffffff
GLOBL decodeHigh<>(SB),RODATA,$16

DATA decodeLow<>+0x00(SB)/8, $0x0f0d0b0907050301
DATA decodeLow<>+0x08(SB)/8, $0xffffffffffffffff
GLOBL decodeLow<>(SB),RODATA,$16

DATA decodeValid<>+0x00(SB)/8, $0xb0b0b0b0b0b0b0b0
DATA decodeValid<>+0x08(SB)/8, $0xb0b0b0b0b0b0b0b0
DATA decodeValid<>+0x10(SB)/8, $0xb9b9b9b9b9b9b9b9
DATA decodeValid<>+0x18(SB)/8, $0xb9b9b9b9b9b9b9b9
DATA decodeValid<>+0x20(SB)/8, $0xe1e1e1e1e1e1e1e1
DATA decodeValid<>+0x28(SB)/8, $0xe1e1e1e1e1e1e1e1
DATA decodeValid<>+0x30(SB)/8, $0xe6e6e6e6e6e6e6e6
DATA decodeValid<>+0x38(SB)/8, $0xe6e6e6e6e6e6e6e6
GLOBL decodeValid<>(SB),RODATA,$64

DATA decodeToSigned<>+0x00(SB)/8, $0x8080808080808080
DATA decodeToSigned<>+0x08(SB)/8, $0x8080808080808080
GLOBL decodeToSigned<>(SB),RODATA,$16

TEXT ·decodeAVX(SB),NOSPLIT,$0
	MOVQ dst+0(FP), DI
	MOVQ src+8(FP), SI
	MOVQ len+16(FP), BX
	MOVQ SI, R15
	MOVOU decodeValid<>(SB), X14
	MOVOU decodeValid<>+0x20(SB), X15
	MOVW $65535, DX
	CMPQ BX, $16
	JB tail
bigloop:
	MOVOU (SI), X0
	VPXOR decodeToSigned<>(SB), X0, X1
	POR decodeToLower<>(SB), X0
	VPXOR decodeToSigned<>(SB), X0, X2
	VPCMPGTB X1, X14, X3
	PCMPGTB decodeValid<>+0x10(SB), X1
	VPCMPGTB X2, X15, X4
	PCMPGTB decodeValid<>+0x30(SB), X2
	PAND X4, X1
	POR X2, X3
	POR X1, X3
	PMOVMSKB X3, AX
	TESTW AX, DX
	JNZ invalid
	PSUBB decodeBase<>(SB), X0
	PANDN decodeBase<>+0x10(SB), X4
	PSUBB X4, X0
	VPSHUFB decodeLow<>(SB), X0, X3
	PSHUFB decodeHigh<>(SB), X0
	PSLLW $4, X0
	POR X3, X0
	MOVQ X0, (DI)
	SUBQ $16, BX
	JZ ret
	ADDQ $16, SI
	ADDQ $8, DI
	CMPQ BX, $16
	JAE bigloop
tail:
	MOVQ $16, CX
	SUBQ BX, CX
	SHRW CX, DX
	CMPQ BX, $4
	JB tail_in_2
	JE tail_in_4
	CMPQ BX, $8
	JB tail_in_6
	JE tail_in_8
	CMPQ BX, $12
	JB tail_in_10
	JE tail_in_12
tail_in_14:
	PINSRW $6, 12(SI), X0
tail_in_12:
	PINSRW $5, 10(SI), X0
tail_in_10:
	PINSRW $4, 8(SI), X0
tail_in_8:
	PINSRQ $0, (SI), X0
	JMP tail_conv
tail_in_6:
	PINSRW $2, 4(SI), X0
tail_in_4:
	PINSRW $1, 2(SI), X0
tail_in_2:
	PINSRW $0, (SI), X0
tail_conv:
	VPXOR decodeToSigned<>(SB), X0, X1
	POR decodeToLower<>(SB), X0
	VPXOR decodeToSigned<>(SB), X0, X2
	VPCMPGTB X1, X14, X3
	PCMPGTB decodeValid<>+0x10(SB), X1
	VPCMPGTB X2, X15, X4
	PCMPGTB decodeValid<>+0x30(SB), X2
	PAND X4, X1
	POR X2, X3
	POR X1, X3
	PMOVMSKB X3, AX
	TESTW AX, DX
	JNZ invalid
	PSUBB decodeBase<>(SB), X0
	PANDN decodeBase<>+0x10(SB), X4
	PSUBB X4, X0
	VPSHUFB decodeLow<>(SB), X0, X3
	PSHUFB decodeHigh<>(SB), X0
	PSLLW $4, X0
	POR X3, X0
	CMPQ BX, $4
	JB tail_out_2
	JE tail_out_4
	CMPQ BX, $8
	JB tail_out_6
	JE tail_out_8
	CMPQ BX, $12
	JB tail_out_10
	JE tail_out_12
tail_out_14:
	PEXTRB $6, X0, 6(DI)
tail_out_12:
	PEXTRB $5, X0, 5(DI)
tail_out_10:
	PEXTRB $4, X0, 4(DI)
tail_out_8:
	MOVL X0, (DI)
	JMP ret
tail_out_6:
	PEXTRB $2, X0, 2(DI)
tail_out_4:
	PEXTRB $1, X0, 1(DI)
tail_out_2:
	PEXTRB $0, X0, (DI)
ret:
	MOVB $1, ok+32(FP)
	RET
invalid:
	BSFW AX, AX
	SUBQ R15, SI
	ADDQ SI, AX
	MOVQ AX, n+24(FP)
	MOVB $0, ok+32(FP)
	RET

TEXT ·decodeSSE(SB),NOSPLIT,$0
	MOVQ dst+0(FP), DI
	MOVQ src+8(FP), SI
	MOVQ len+16(FP), BX
	MOVQ SI, R15
	MOVOU decodeValid<>(SB), X14
	MOVOU decodeValid<>+0x20(SB), X15
	MOVW $65535, DX
	CMPQ BX, $16
	JB tail
bigloop:
	MOVOU (SI), X0
	MOVOU X0, X1
	PXOR decodeToSigned<>(SB), X1
	POR decodeToLower<>(SB), X0
	MOVOU X0, X2
	PXOR decodeToSigned<>(SB), X2
	MOVOU X14, X3
	PCMPGTB X1, X3
	PCMPGTB decodeValid<>+0x10(SB), X1
	MOVOU X15, X4
	PCMPGTB X2, X4
	PCMPGTB decodeValid<>+0x30(SB), X2
	PAND X4, X1
	POR X2, X3
	POR X1, X3
	PMOVMSKB X3, AX
	TESTW AX, DX
	JNZ invalid
	PSUBB decodeBase<>(SB), X0
	PANDN decodeBase<>+0x10(SB), X4
	PSUBB X4, X0
	MOVOU X0, X3
	PSHUFB decodeLow<>(SB), X3
	PSHUFB decodeHigh<>(SB), X0
	PSLLW $4, X0
	POR X3, X0
	MOVQ X0, (DI)
	SUBQ $16, BX
	JZ ret
	ADDQ $16, SI
	ADDQ $8, DI
	CMPQ BX, $16
	JAE bigloop
tail:
	MOVQ $16, CX
	SUBQ BX, CX
	SHRW CX, DX
	CMPQ BX, $4
	JB tail_in_2
	JE tail_in_4
	CMPQ BX, $8
	JB tail_in_6
	JE tail_in_8
	CMPQ BX, $12
	JB tail_in_10
	JE tail_in_12
tail_in_14:
	PINSRW $6, 12(SI), X0
tail_in_12:
	PINSRW $5, 10(SI), X0
tail_in_10:
	PINSRW $4, 8(SI), X0
tail_in_8:
	PINSRQ $0, (SI), X0
	JMP tail_conv
tail_in_6:
	PINSRW $2, 4(SI), X0
tail_in_4:
	PINSRW $1, 2(SI), X0
tail_in_2:
	PINSRW $0, (SI), X0
tail_conv:
	MOVOU X0, X1
	PXOR decodeToSigned<>(SB), X1
	POR decodeToLower<>(SB), X0
	MOVOU X0, X2
	PXOR decodeToSigned<>(SB), X2
	MOVOU X14, X3
	PCMPGTB X1, X3
	PCMPGTB decodeValid<>+0x10(SB), X1
	MOVOU X15, X4
	PCMPGTB X2, X4
	PCMPGTB decodeValid<>+0x30(SB), X2
	PAND X4, X1
	POR X2, X3
	POR X1, X3
	PMOVMSKB X3, AX
	TESTW AX, DX
	JNZ invalid
	PSUBB decodeBase<>(SB), X0
	PANDN decodeBase<>+0x10(SB), X4
	PSUBB X4, X0
	MOVOU X0, X3
	PSHUFB decodeLow<>(SB), X3
	PSHUFB decodeHigh<>(SB), X0
	PSLLW $4, X0
	POR X3, X0
	CMPQ BX, $4
	JB tail_out_2
	JE tail_out_4
	CMPQ BX, $8
	JB tail_out_6
	JE tail_out_8
	CMPQ BX, $12
	JB tail_out_10
	JE tail_out_12
tail_out_14:
	PEXTRB $6, X0, 6(DI)
tail_out_12:
	PEXTRB $5, X0, 5(DI)
tail_out_10:
	PEXTRB $4, X0, 4(DI)
tail_out_8:
	MOVL X0, (DI)
	JMP ret
tail_out_6:
	PEXTRB $2, X0, 2(DI)
tail_out_4:
	PEXTRB $1, X0, 1(DI)
tail_out_2:
	PEXTRB $0, X0, (DI)
ret:
	MOVB $1, ok+32(FP)
	RET
invalid:
	BSFW AX, AX
	SUBQ R15, SI
	ADDQ SI, AX
	MOVQ AX, n+24(FP)
	MOVB $0, ok+32(FP)
	RET
