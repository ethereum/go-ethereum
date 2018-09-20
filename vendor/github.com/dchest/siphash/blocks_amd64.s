// +build amd64,!appengine,!gccgo

#define ROUND(v0, v1, v2, v3) \
	ADDQ v1, v0; \
	RORQ $51, v1; \
	ADDQ v3, v2; \
	XORQ v0, v1; \
	RORQ $48, v3; \
	RORQ $32, v0; \
	XORQ v2, v3; \
	ADDQ v1, v2; \
	ADDQ v3, v0; \
	RORQ $43, v3; \
	RORQ $47, v1; \
	XORQ v0, v3; \
	XORQ v2, v1; \
	RORQ $32, v2

// blocks(d *digest, data []uint8)
TEXT ·blocks(SB),4,$0-32
	MOVQ d+0(FP), BX
	MOVQ 0(BX), R9		// R9 = v0
	MOVQ 8(BX), R10		// R10 = v1
	MOVQ 16(BX), R11	// R11 = v2
	MOVQ 24(BX), R12	// R12 = v3
	MOVQ p_base+8(FP), DI	// DI = *uint64
	MOVQ p_len+16(FP), SI	// SI = nblocks
	XORL DX, DX		// DX = index (0)
	SHRQ $3, SI 		// SI /= 8
body:
	CMPQ DX, SI
	JGE  end
	MOVQ 0(DI)(DX*8), CX	// CX = m
	XORQ CX, R12
	ROUND(R9, R10, R11, R12)
	ROUND(R9, R10, R11, R12)
	XORQ CX, R9
	ADDQ $1, DX
	JMP  body
end:
	MOVQ R9, 0(BX)
	MOVQ R10, 8(BX)
	MOVQ R11, 16(BX)
	MOVQ R12, 24(BX)
	RET

// once(d *digest)
TEXT ·once(SB),4,$0-8
	MOVQ d+0(FP), BX
	MOVQ 0(BX), R9		// R9 = v0
	MOVQ 8(BX), R10		// R10 = v1
	MOVQ 16(BX), R11	// R11 = v2
	MOVQ 24(BX), R12	// R12 = v3
	MOVQ 48(BX), CX		// CX = d.x[:]
	XORQ CX, R12
	ROUND(R9, R10, R11, R12)
	ROUND(R9, R10, R11, R12)
	XORQ CX, R9
	MOVQ R9, 0(BX)
	MOVQ R10, 8(BX)
	MOVQ R11, 16(BX)
	MOVQ R12, 24(BX)
	RET

// finalize(d *digest) uint64
TEXT ·finalize(SB),4,$0-16
	MOVQ d+0(FP), BX
	MOVQ 0(BX), R9		// R9 = v0
	MOVQ 8(BX), R10		// R10 = v1
	MOVQ 16(BX), R11	// R11 = v2
	MOVQ 24(BX), R12	// R12 = v3
	MOVQ 48(BX), CX		// CX = d.x[:]
	XORQ CX, R12
	ROUND(R9, R10, R11, R12)
	ROUND(R9, R10, R11, R12)
	XORQ CX, R9
	NOTB R11
	ROUND(R9, R10, R11, R12)
	ROUND(R9, R10, R11, R12)
	ROUND(R9, R10, R11, R12)
	ROUND(R9, R10, R11, R12)
	XORQ R12, R11
	XORQ R10, R9
	XORQ R11, R9
	MOVQ R9, ret+8(FP)
	RET
