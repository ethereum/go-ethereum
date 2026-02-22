// Copyright 2025 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.
// inspired by: https://github.com/golang/go/blob/4a3cef2036097d323b6cc0bbe90fc4d8c7588660/src/crypto/internal/fips140/subtle/xor_amd64.s

//go:build !purego

#include "textflag.h"

// func testBytesASM(p *byte, n int) bool
TEXT ·testBytesASM(SB), NOSPLIT, $0
	MOVQ  p+0(FP), SI
	MOVQ  n+8(FP), DX
	TESTQ DX, DX            // if len is 0, return false
	JZ    not_found
	TESTQ $15, DX            // AND 15 & len, if not zero jump to not_aligned.
	JNZ   not_aligned

aligned:
	MOVQ $0, AX // position in slice

	PCALIGN $16
loop16b:
	MOVOU (SI)(AX*1), X0   // Load 16 bytes
	PTEST X0, X0           // Test if all bits are zero (ZF=1 if all zero)
	JNZ   found            // If any bit is set (ZF=0), jump to found
	ADDQ  $16, AX
	CMPQ  DX, AX
	JNE   loop16b
	JMP   not_found

	PCALIGN $16
loop_1b:
	SUBQ  $1, DX           // Test 1 byte backwards.
	MOVB  (SI)(DX*1), DI
	TESTB DI, DI           // Test if byte is non-zero
	JNZ   found
	TESTQ $7, DX           // AND 7 & len, if not zero jump to loop_1b.
	JNZ   loop_1b
	CMPQ  DX, $0           // if len is 0, ret.
	JE    not_found
	TESTQ $15, DX          // AND 15 & len, if zero jump to aligned.
	JZ    aligned

not_aligned:
	TESTQ $7, DX           // AND $7 & len, if not zero jump to loop_1b.
	JNE   loop_1b
	SUBQ  $8, DX           // Test 8 bytes backwards.
	MOVQ  (SI)(DX*1), DI
	TESTQ DI, DI           // Test if 8 bytes are non-zero
	JNZ   found
	CMPQ  DX, $16          // if len is greater or equal 16 here, it must be aligned.
	JGE   aligned
	JMP   not_found

not_found:
	MOVB $0, ret+16(FP)
	RET

found:
	MOVB $1, ret+16(FP)
	RET

