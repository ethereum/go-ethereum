// Copyright 2025 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.
// inspired by: https://github.com/golang/go/blob/4a3cef2036097d323b6cc0bbe90fc4d8c7588660/src/crypto/internal/fips140/subtle/xor_arm64.s

//go:build !purego

#include "textflag.h"

// func testBytesASM(p *byte, n int) bool
TEXT ·testBytesASM(SB), NOSPLIT|NOFRAME, $0
	MOVD	p+0(FP), R0
	MOVD	n+8(FP), R1
	CMP	$64, R1
	BLT	tail
loop_64:
	VLD1.P	64(R0), [V0.B16, V1.B16, V2.B16, V3.B16]
	// OR all vectors together to check if any byte is non-zero
	VORR	V0.B16, V1.B16, V4.B16
	VORR	V2.B16, V3.B16, V5.B16
	VORR	V4.B16, V5.B16, V6.B16
	// Check if any byte in V6 is non-zero by checking both 64-bit halves
	VMOV	V6.D[0], R2
	VMOV	V6.D[1], R3
	ORR	R2, R3, R2
	CBNZ	R2, found
	SUBS	$64, R1
	CMP	$64, R1
	BGE	loop_64
tail:
	// quick end
	CBZ	R1, not_found
	TBZ	$5, R1, less_than32
	VLD1.P	32(R0), [V0.B16, V1.B16]
	VORR	V0.B16, V1.B16, V2.B16
	VMOV	V2.D[0], R2
	VMOV	V2.D[1], R3
	ORR	R2, R3, R2
	CBNZ	R2, found
less_than32:
	TBZ	$4, R1, less_than16
	LDP.P	16(R0), (R11, R12)
	ORR	R11, R12, R2
	CBNZ	R2, found
less_than16:
	TBZ	$3, R1, less_than8
	MOVD.P	8(R0), R11
	CBNZ	R11, found
less_than8:
	TBZ	$2, R1, less_than4
	MOVWU.P	4(R0), R11
	CBNZ	R11, found
less_than4:
	TBZ	$1, R1, less_than2
	MOVHU.P	2(R0), R11
	CBNZ	R11, found
less_than2:
	TBZ	$0, R1, not_found
	MOVBU	(R0), R11
	CBNZ	R11, found
not_found:
	MOVD	$0, R0
	MOVB	R0, ret+16(FP)
	RET
found:
	MOVD	$1, R0
	MOVB	R0, ret+16(FP)
	RET
