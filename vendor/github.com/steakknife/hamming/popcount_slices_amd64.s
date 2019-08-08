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

// type SliceHeader struct {
//         Data uintptr 0
//         Len  int     8
//         Cap  int     16
// }

//  0 x.Data
//  8 x.Len
// 16 x.Cap
// 24 ret

// type StringHeader struct {
//        Data uintptr 0 
//        Len  int     8
// }

//  0 x.Data
//  8 x.Len
// 16 ret

// func CountBitsInt8sPopCnt(x []int8) (ret int)
TEXT ·CountBitsInt8sPopCnt(SB),NOSPLIT,$0
	JMP        ·CountBitsUint8sPopCnt(SB)

// func CountBitsInt16sPopCnt(x []int16) (ret int)
TEXT ·CountBitsInt16sPopCnt(SB),NOSPLIT,$0
	JMP        ·CountBitsUint16sPopCnt(SB)

// func CountBitsInt32sPopCnt(x []int32) (ret int)
TEXT ·CountBitsInt32sPopCnt(SB),NOSPLIT,$0
	JMP        ·CountBitsUint32sPopCnt(SB)

// func CountBitsInt64sPopCnt(x []int64) (ret int)
TEXT ·CountBitsInt64sPopCnt(SB),NOSPLIT,$0
	JMP        ·CountBitsUint64sPopCnt(SB)

// func CountBitsUint8sPopCnt(x []uint8) (ret int)
TEXT ·CountBitsUint8sPopCnt(SB),NOSPLIT,$0
	XORQ       AX, AX // ret = 0
	MOVQ       x+8(FP), CX // x.Len -> CX

test_negative_slice_len:
	MOVQ       CX, BX // x.Len < 0 ---> x.Len[63] != 0
	SHRQ       $63, BX
	JNZ        done

	MOVQ       x+0(FP), DI // x.Data -> DI

	CMPQ       CX, $32 // x.Len >= 32
	JL         unrolled_loop_skip

unrolled_loop_setup:
	XORQ       R9, R9
	XORQ       BX, BX
	XORQ       DX, DX

unrolled_loop: // 4 unrolled loops of POPCNTQ (4 quad words at a time)
	SUBQ       $32, CX

	POPCNTQ    0(DI), R10
	ADDQ       R10, R9
	POPCNTQ    8(DI), R11
	ADDQ       R11, AX
	POPCNTQ    16(DI), R12
	ADDQ       R12, BX
	POPCNTQ    24(DI), R13
	ADDQ       R13, DX

	ADDQ       $32, DI
	CMPQ       CX, $32 // x.Len >= 32
	JGE        unrolled_loop

unrolled_loop_done:
	ADDQ       R9, AX
	ADDQ       BX, DX
	ADDQ       DX, AX

	XORQ       BX, BX

unrolled_loop_skip:
	CMPQ       CX, $0
	JZ         done

	XORQ       DX, DX

remainder_loop:
	MOVB       0(DI), DL
	POPCNTQ    DX, BX
	ADDQ       BX, AX
	
	INCQ       DI
	DECQ       CX
	JNZ        remainder_loop

done:
	MOVQ       AX, ret+24(FP)
	RET

// func CountBitsUint16sPopCnt(x []uint16) (ret int)
TEXT ·CountBitsUint16sPopCnt(SB),NOSPLIT,$0
	XORQ       AX, AX // ret = 0
	MOVQ       x+8(FP), CX // x.Len -> CX

test_negative_slice_len:
	MOVQ       CX, BX // x.Len*2 < 0 ---> x.Len[63:62] != 0
	SHLQ       $1, CX
	SHRQ       $62, BX
	JNZ        done

	MOVQ       x+0(FP), DI // x.Data -> DI


	CMPQ       CX, $32 // x.Len*2 >= 32
	JL         unrolled_loop_skip

unrolled_loop_setup:
	XORQ       R9, R9
	XORQ       BX, BX
	XORQ       DX, DX

unrolled_loop: // 4 unrolled loops of POPCNTQ (4 quad words at a time)
	SUBQ       $32, CX

	POPCNTQ    0(DI), R10
	ADDQ       R10, R9
	POPCNTQ    8(DI), R11
	ADDQ       R11, AX
	POPCNTQ    16(DI), R12
	ADDQ       R12, BX
	POPCNTQ    24(DI), R13
	ADDQ       R13, DX

	ADDQ       $32, DI
	CMPQ       CX, $32 // x.Len*2 >= 32
	JGE        unrolled_loop

unrolled_loop_done:
	ADDQ       R9, AX
	ADDQ       BX, DX
	ADDQ       DX, AX

	XORQ       BX, BX

unrolled_loop_skip:
	CMPQ       CX, $0
	JZ         done

	XORQ       DX, DX

remainder_loop:
	MOVW       0(DI), DX
	POPCNTQ    DX, BX
	ADDQ       BX, AX
	
	ADDQ       $2, DI
	SUBQ       $2, CX
	JNZ        remainder_loop

done:
	MOVQ       AX, ret+24(FP)
	RET

// func CountBitsUint32sPopCnt(x []uint32) (ret int)
TEXT ·CountBitsUint32sPopCnt(SB),NOSPLIT,$0
	XORQ       AX, AX // ret = 0
	MOVQ       x+8(FP), CX // x.Len -> CX
	MOVQ       CX,      BX 
	MOVQ       x+0(FP), DI // x.Data -> DI

test_negative_slice_len:
	SHLQ       $2, CX // x.Len*4 < 0 ---> x.Len[63:61] != 0
	SHRQ       $61, BX
	JNZ        done



	CMPQ       CX, $32 // x.Len*4 >= 32
	JL         unrolled_loop_skip

unrolled_loop_setup:
	XORQ       R9, R9
	XORQ       BX, BX
	XORQ       DX, DX

unrolled_loop: // 4 unrolled loops of POPCNTQ (4 quad words at a time)
	SUBQ       $32, CX

	POPCNTQ    0(DI), R10 // r9 += popcntq(QW DI+0)
	ADDQ       R10, R9
	POPCNTQ    8(DI), R11 // ax += popcntq(QW DI+8)
	ADDQ       R11, AX
	POPCNTQ    16(DI), R12 // bx += popcntq(QW DI+16)
	ADDQ       R12, BX
	POPCNTQ    24(DI), R13 // dx += popcntq(QW DI+24)
	ADDQ       R13, DX

	ADDQ       $32, DI
	CMPQ       CX, $32 // x.Len*4 >= 32
	JGE        unrolled_loop

unrolled_loop_done:
	ADDQ       R9, AX // ax = (ax + r9) + (bx + dx)
	ADDQ       BX, DX
	ADDQ       DX, AX

	XORQ       BX, BX

unrolled_loop_skip:
	CMPQ       CX, $0
	JZ         done

	XORQ       DX, DX
remainder_loop:
	MOVB       (DI), DX // ax += popcnt(DB 0(DI))
	POPCNTQ    DX, BX
	ADDQ       BX, AX
	
	INCQ       DI
	DECQ       CX
	JNZ        remainder_loop

done:
	MOVQ       AX, ret+24(FP)
	RET

// func CountBitsUint64sPopCnt(x []uint64) (ret int)
TEXT ·CountBitsUint64sPopCnt(SB),NOSPLIT,$0
	XORQ       AX, AX // ret = 0
	MOVQ       x+8(FP), CX // x.Len -> CX

test_negative_slice_len:
	MOVQ       CX, BX // x.Len*8 < 0 ---> x.Len[63:60] != 0
	SHLQ       $3, CX
	SHRQ       $60, BX
	JNZ        done

	MOVQ       x+0(FP), DI // x.Data -> DI


	CMPQ       CX, $32 // x.Len*8 >= 32
	JL         unrolled_loop_skip

unrolled_loop_setup:
	XORQ       R9, R9
	XORQ       BX, BX
	XORQ       DX, DX

unrolled_loop: // 4 unrolled loops of POPCNTQ (4 quad words at a time)
	SUBQ       $32, CX

	POPCNTQ    0(DI), R10
	ADDQ       R10, R9
	POPCNTQ    8(DI), R11
	ADDQ       R11, AX
	POPCNTQ    16(DI), R12
	ADDQ       R12, BX
	POPCNTQ    24(DI), R13
	ADDQ       R13, DX

	ADDQ       $32, DI
	CMPQ       CX, $32 // x.Len*4 >= 32
	JGE        unrolled_loop

unrolled_loop_done:
	ADDQ       R9, AX
	ADDQ       BX, DX
	ADDQ       DX, AX

	XORQ       BX, BX

unrolled_loop_skip:
	CMPQ       CX, $0
	JZ         done

	XORQ       DX, DX

remainder_loop:
	MOVQ       0(DI), DX
	POPCNTQ    DX, BX
	ADDQ       BX, AX
	
	ADDQ       $8, DI
	SUBQ       $8, CX
	JNZ        remainder_loop

done:
	MOVQ       AX, ret+24(FP)
	RET

// func CountBitsBytesPopCnt(x []byte) (ret int)
TEXT ·CountBitsBytesPopCnt(SB),NOSPLIT,$0
	JMP        ·CountBitsUint8sPopCnt(SB)

// func CountBitsRunesPopCnt(x []rune) (ret int)
TEXT ·CountBitsRunesPopCnt(SB),NOSPLIT,$0
	JMP        ·CountBitsUint32sPopCnt(SB)

// func CountBitsStringPopCnt(s string) (ret int)
TEXT ·CountBitsStringPopCnt(SB),NOSPLIT,$0
	XORQ       AX, AX // ret = 0
	MOVQ       x+8(FP), CX // x.Len -> CX

test_negative_slice_len:
	MOVQ       CX, BX // x.Len < 0 ---> x.Len[63] != 0
	SHRQ       $63, BX
	JNZ        done

	MOVQ       x+0(FP), DI // x.Data -> DI

	CMPQ       CX, $32 // x.Len >= 32
	JL         unrolled_loop_skip

unrolled_loop_setup:
	XORQ       R9, R9
	XORQ       BX, BX
	XORQ       DX, DX

unrolled_loop: // 4 unrolled loops of POPCNTQ (4 quad words at a time)
	SUBQ       $32, CX

	POPCNTQ    0(DI), R10
	ADDQ       R10, R9
	POPCNTQ    8(DI), R11
	ADDQ       R11, AX
	POPCNTQ    16(DI), R12
	ADDQ       R12, BX
	POPCNTQ    24(DI), R13
	ADDQ       R13, DX

	ADDQ       $32, DI
	CMPQ       CX, $32 // x.Len >= 32
	JGE        unrolled_loop

unrolled_loop_done:
	ADDQ       R9, AX
	ADDQ       BX, DX
	ADDQ       DX, AX

	XORQ       BX, BX

unrolled_loop_skip:
	CMPQ       CX, $0
	JZ         done

	XORQ       DX, DX

remainder_loop:
	MOVB       0(DI), DL
	POPCNTQ    DX, BX
	ADDQ       BX, AX
	
	INCQ       DI
	DECQ       CX
	JNZ        remainder_loop

done:
	MOVQ       AX, ret+16(FP)
	RET
