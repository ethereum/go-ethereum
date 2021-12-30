// Copyright 2016 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// +build amd64,!gccgo,!appengine

#include "textflag.h"

DATA ·iv0<>+0x00(SB)/8, $0x6a09e667f3bcc908
DATA ·iv0<>+0x08(SB)/8, $0xbb67ae8584caa73b
GLOBL ·iv0<>(SB), (NOPTR+RODATA), $16

DATA ·iv1<>+0x00(SB)/8, $0x3c6ef372fe94f82b
DATA ·iv1<>+0x08(SB)/8, $0xa54ff53a5f1d36f1
GLOBL ·iv1<>(SB), (NOPTR+RODATA), $16

DATA ·iv2<>+0x00(SB)/8, $0x510e527fade682d1
DATA ·iv2<>+0x08(SB)/8, $0x9b05688c2b3e6c1f
GLOBL ·iv2<>(SB), (NOPTR+RODATA), $16

DATA ·iv3<>+0x00(SB)/8, $0x1f83d9abfb41bd6b
DATA ·iv3<>+0x08(SB)/8, $0x5be0cd19137e2179
GLOBL ·iv3<>(SB), (NOPTR+RODATA), $16

DATA ·c40<>+0x00(SB)/8, $0x0201000706050403
DATA ·c40<>+0x08(SB)/8, $0x0a09080f0e0d0c0b
GLOBL ·c40<>(SB), (NOPTR+RODATA), $16

DATA ·c48<>+0x00(SB)/8, $0x0100070605040302
DATA ·c48<>+0x08(SB)/8, $0x09080f0e0d0c0b0a
GLOBL ·c48<>(SB), (NOPTR+RODATA), $16

#define SHUFFLE(v2, v3, v4, v5, v6, v7, t1, t2) \
	MOVO       v4, t1; \
	MOVO       v5, v4; \
	MOVO       t1, v5; \
	MOVO       v6, t1; \
	PUNPCKLQDQ v6, t2; \
	PUNPCKHQDQ v7, v6; \
	PUNPCKHQDQ t2, v6; \
	PUNPCKLQDQ v7, t2; \
	MOVO       t1, v7; \
	MOVO       v2, t1; \
	PUNPCKHQDQ t2, v7; \
	PUNPCKLQDQ v3, t2; \
	PUNPCKHQDQ t2, v2; \
	PUNPCKLQDQ t1, t2; \
	PUNPCKHQDQ t2, v3

#define SHUFFLE_INV(v2, v3, v4, v5, v6, v7, t1, t2) \
	MOVO       v4, t1; \
	MOVO       v5, v4; \
	MOVO       t1, v5; \
	MOVO       v2, t1; \
	PUNPCKLQDQ v2, t2; \
	PUNPCKHQDQ v3, v2; \
	PUNPCKHQDQ t2, v2; \
	PUNPCKLQDQ v3, t2; \
	MOVO       t1, v3; \
	MOVO       v6, t1; \
	PUNPCKHQDQ t2, v3; \
	PUNPCKLQDQ v7, t2; \
	PUNPCKHQDQ t2, v6; \
	PUNPCKLQDQ t1, t2; \
	PUNPCKHQDQ t2, v7

#define HALF_ROUND(v0, v1, v2, v3, v4, v5, v6, v7, m0, m1, m2, m3, t0, c40, c48) \
	PADDQ  m0, v0;        \
	PADDQ  m1, v1;        \
	PADDQ  v2, v0;        \
	PADDQ  v3, v1;        \
	PXOR   v0, v6;        \
	PXOR   v1, v7;        \
	PSHUFD $0xB1, v6, v6; \
	PSHUFD $0xB1, v7, v7; \
	PADDQ  v6, v4;        \
	PADDQ  v7, v5;        \
	PXOR   v4, v2;        \
	PXOR   v5, v3;        \
	PSHUFB c40, v2;       \
	PSHUFB c40, v3;       \
	PADDQ  m2, v0;        \
	PADDQ  m3, v1;        \
	PADDQ  v2, v0;        \
	PADDQ  v3, v1;        \
	PXOR   v0, v6;        \
	PXOR   v1, v7;        \
	PSHUFB c48, v6;       \
	PSHUFB c48, v7;       \
	PADDQ  v6, v4;        \
	PADDQ  v7, v5;        \
	PXOR   v4, v2;        \
	PXOR   v5, v3;        \
	MOVOU  v2, t0;        \
	PADDQ  v2, t0;        \
	PSRLQ  $63, v2;       \
	PXOR   t0, v2;        \
	MOVOU  v3, t0;        \
	PADDQ  v3, t0;        \
	PSRLQ  $63, v3;       \
	PXOR   t0, v3

#define LOAD_MSG(m0, m1, m2, m3, i0, i1, i2, i3, i4, i5, i6, i7) \
	MOVQ   i0*8(SI), m0;     \
	PINSRQ $1, i1*8(SI), m0; \
	MOVQ   i2*8(SI), m1;     \
	PINSRQ $1, i3*8(SI), m1; \
	MOVQ   i4*8(SI), m2;     \
	PINSRQ $1, i5*8(SI), m2; \
	MOVQ   i6*8(SI), m3;     \
	PINSRQ $1, i7*8(SI), m3

// func fSSE4(h *[8]uint64, m *[16]uint64, c0, c1 uint64, flag uint64, rounds uint64)
TEXT ·fSSE4(SB), 4, $24-48 // frame size = 8 + 16 byte alignment
	MOVQ h+0(FP), AX
	MOVQ m+8(FP), SI
	MOVQ c0+16(FP), R8
	MOVQ c1+24(FP), R9
	MOVQ flag+32(FP), CX
	MOVQ rounds+40(FP), BX

	MOVQ SP, BP
	MOVQ SP, R10
	ADDQ $15, R10
	ANDQ $~15, R10
	MOVQ R10, SP

	MOVOU ·iv3<>(SB), X0
	MOVO  X0, 0(SP)
	XORQ  CX, 0(SP)     // 0(SP) = ·iv3 ^ (CX || 0)

	MOVOU ·c40<>(SB), X13
	MOVOU ·c48<>(SB), X14

	MOVOU 0(AX), X12
	MOVOU 16(AX), X15

	MOVQ R8, X8
	PINSRQ $1, R9, X8

	MOVO X12, X0
	MOVO X15, X1
	MOVOU 32(AX), X2
	MOVOU 48(AX), X3
	MOVOU ·iv0<>(SB), X4
	MOVOU ·iv1<>(SB), X5
	MOVOU ·iv2<>(SB), X6

	PXOR X8, X6
	MOVO 0(SP), X7

loop:
	SUBQ $1, BX; JCS done
	LOAD_MSG(X8, X9, X10, X11, 0, 2, 4, 6, 1, 3, 5, 7)
	HALF_ROUND(X0, X1, X2, X3, X4, X5, X6, X7, X8, X9, X10, X11, X11, X13, X14)
	SHUFFLE(X2, X3, X4, X5, X6, X7, X8, X9)
	LOAD_MSG(X8, X9, X10, X11, 8, 10, 12, 14, 9, 11, 13, 15)
	HALF_ROUND(X0, X1, X2, X3, X4, X5, X6, X7, X8, X9, X10, X11, X11, X13, X14)
	SHUFFLE_INV(X2, X3, X4, X5, X6, X7, X8, X9)

	SUBQ $1, BX; JCS done
	LOAD_MSG(X8, X9, X10, X11, 14, 4, 9, 13, 10, 8, 15, 6)
	HALF_ROUND(X0, X1, X2, X3, X4, X5, X6, X7, X8, X9, X10, X11, X11, X13, X14)
	SHUFFLE(X2, X3, X4, X5, X6, X7, X8, X9)
	LOAD_MSG(X8, X9, X10, X11, 1, 0, 11, 5, 12, 2, 7, 3)
	HALF_ROUND(X0, X1, X2, X3, X4, X5, X6, X7, X8, X9, X10, X11, X11, X13, X14)
	SHUFFLE_INV(X2, X3, X4, X5, X6, X7, X8, X9)

	SUBQ $1, BX; JCS done
	LOAD_MSG(X8, X9, X10, X11, 11, 12, 5, 15, 8, 0, 2, 13)
	HALF_ROUND(X0, X1, X2, X3, X4, X5, X6, X7, X8, X9, X10, X11, X11, X13, X14)
	SHUFFLE(X2, X3, X4, X5, X6, X7, X8, X9)
	LOAD_MSG(X8, X9, X10, X11, 10, 3, 7, 9, 14, 6, 1, 4)
	HALF_ROUND(X0, X1, X2, X3, X4, X5, X6, X7, X8, X9, X10, X11, X11, X13, X14)
	SHUFFLE_INV(X2, X3, X4, X5, X6, X7, X8, X9)

	SUBQ $1, BX; JCS done
	LOAD_MSG(X8, X9, X10, X11, 7, 3, 13, 11, 9, 1, 12, 14)
	HALF_ROUND(X0, X1, X2, X3, X4, X5, X6, X7, X8, X9, X10, X11, X11, X13, X14)
	SHUFFLE(X2, X3, X4, X5, X6, X7, X8, X9)
	LOAD_MSG(X8, X9, X10, X11, 2, 5, 4, 15, 6, 10, 0, 8)
	HALF_ROUND(X0, X1, X2, X3, X4, X5, X6, X7, X8, X9, X10, X11, X11, X13, X14)
	SHUFFLE_INV(X2, X3, X4, X5, X6, X7, X8, X9)

	SUBQ $1, BX; JCS done
	LOAD_MSG(X8, X9, X10, X11, 9, 5, 2, 10, 0, 7, 4, 15)
	HALF_ROUND(X0, X1, X2, X3, X4, X5, X6, X7, X8, X9, X10, X11, X11, X13, X14)
	SHUFFLE(X2, X3, X4, X5, X6, X7, X8, X9)
	LOAD_MSG(X8, X9, X10, X11, 14, 11, 6, 3, 1, 12, 8, 13)
	HALF_ROUND(X0, X1, X2, X3, X4, X5, X6, X7, X8, X9, X10, X11, X11, X13, X14)
	SHUFFLE_INV(X2, X3, X4, X5, X6, X7, X8, X9)

	SUBQ $1, BX; JCS done
	LOAD_MSG(X8, X9, X10, X11, 2, 6, 0, 8, 12, 10, 11, 3)
	HALF_ROUND(X0, X1, X2, X3, X4, X5, X6, X7, X8, X9, X10, X11, X11, X13, X14)
	SHUFFLE(X2, X3, X4, X5, X6, X7, X8, X9)
	LOAD_MSG(X8, X9, X10, X11, 4, 7, 15, 1, 13, 5, 14, 9)
	HALF_ROUND(X0, X1, X2, X3, X4, X5, X6, X7, X8, X9, X10, X11, X11, X13, X14)
	SHUFFLE_INV(X2, X3, X4, X5, X6, X7, X8, X9)

	SUBQ $1, BX; JCS done
	LOAD_MSG(X8, X9, X10, X11, 12, 1, 14, 4, 5, 15, 13, 10)
	HALF_ROUND(X0, X1, X2, X3, X4, X5, X6, X7, X8, X9, X10, X11, X11, X13, X14)
	SHUFFLE(X2, X3, X4, X5, X6, X7, X8, X9)
	LOAD_MSG(X8, X9, X10, X11, 0, 6, 9, 8, 7, 3, 2, 11)
	HALF_ROUND(X0, X1, X2, X3, X4, X5, X6, X7, X8, X9, X10, X11, X11, X13, X14)
	SHUFFLE_INV(X2, X3, X4, X5, X6, X7, X8, X9)

	SUBQ $1, BX; JCS done
	LOAD_MSG(X8, X9, X10, X11, 13, 7, 12, 3, 11, 14, 1, 9)
	HALF_ROUND(X0, X1, X2, X3, X4, X5, X6, X7, X8, X9, X10, X11, X11, X13, X14)
	SHUFFLE(X2, X3, X4, X5, X6, X7, X8, X9)
	LOAD_MSG(X8, X9, X10, X11, 5, 15, 8, 2, 0, 4, 6, 10)
	HALF_ROUND(X0, X1, X2, X3, X4, X5, X6, X7, X8, X9, X10, X11, X11, X13, X14)
	SHUFFLE_INV(X2, X3, X4, X5, X6, X7, X8, X9)

	SUBQ $1, BX; JCS done
	LOAD_MSG(X8, X9, X10, X11, 6, 14, 11, 0, 15, 9, 3, 8)
	HALF_ROUND(X0, X1, X2, X3, X4, X5, X6, X7, X8, X9, X10, X11, X11, X13, X14)
	SHUFFLE(X2, X3, X4, X5, X6, X7, X8, X9)
	LOAD_MSG(X8, X9, X10, X11, 12, 13, 1, 10, 2, 7, 4, 5)
	HALF_ROUND(X0, X1, X2, X3, X4, X5, X6, X7, X8, X9, X10, X11, X11, X13, X14)
	SHUFFLE_INV(X2, X3, X4, X5, X6, X7, X8, X9)

	SUBQ $1, BX; JCS done
	LOAD_MSG(X8, X9, X10, X11, 10, 8, 7, 1, 2, 4, 6, 5)
	HALF_ROUND(X0, X1, X2, X3, X4, X5, X6, X7, X8, X9, X10, X11, X11, X13, X14)
	SHUFFLE(X2, X3, X4, X5, X6, X7, X8, X9)
	LOAD_MSG(X8, X9, X10, X11, 15, 9, 3, 13, 11, 14, 12, 0)
	HALF_ROUND(X0, X1, X2, X3, X4, X5, X6, X7, X8, X9, X10, X11, X11, X13, X14)
	SHUFFLE_INV(X2, X3, X4, X5, X6, X7, X8, X9)

	JMP loop

done:
	MOVOU 32(AX), X10
	MOVOU 48(AX), X11
	PXOR  X0, X12
	PXOR  X1, X15
	PXOR  X2, X10
	PXOR  X3, X11
	PXOR  X4, X12
	PXOR  X5, X15
	PXOR  X6, X10
	PXOR  X7, X11
	MOVOU X10, 32(AX)
	MOVOU X11, 48(AX)

	MOVOU X12, 0(AX)
	MOVOU X15, 16(AX)

	MOVQ BP, SP
	RET
