// +build amd64,!appengine,!gccgo

#include "gfp.h"
#include "mul.h"
#include "mul_bmi2.h"

TEXT ·gfpNeg(SB),0,$0-16
	MOVQ ·p2+0(SB), R8
	MOVQ ·p2+8(SB), R9
	MOVQ ·p2+16(SB), R10
	MOVQ ·p2+24(SB), R11

	MOVQ a+8(FP), DI
	SUBQ 0(DI), R8
	SBBQ 8(DI), R9
	SBBQ 16(DI), R10
	SBBQ 24(DI), R11

	MOVQ $0, AX
	gfpCarry(R8,R9,R10,R11,AX, R12,R13,R14,R15,BX)

	MOVQ c+0(FP), DI
	storeBlock(R8,R9,R10,R11, 0(DI))
	RET

TEXT ·gfpAdd(SB),0,$0-24
	MOVQ a+8(FP), DI
	MOVQ b+16(FP), SI

	loadBlock(0(DI), R8,R9,R10,R11)
	MOVQ $0, R12

	ADDQ  0(SI), R8
	ADCQ  8(SI), R9
	ADCQ 16(SI), R10
	ADCQ 24(SI), R11
	ADCQ $0, R12

	gfpCarry(R8,R9,R10,R11,R12, R13,R14,R15,AX,BX)

	MOVQ c+0(FP), DI
	storeBlock(R8,R9,R10,R11, 0(DI))
	RET

TEXT ·gfpSub(SB),0,$0-24
	MOVQ a+8(FP), DI
	MOVQ b+16(FP), SI

	loadBlock(0(DI), R8,R9,R10,R11)

	MOVQ ·p2+0(SB), R12
	MOVQ ·p2+8(SB), R13
	MOVQ ·p2+16(SB), R14
	MOVQ ·p2+24(SB), R15
	MOVQ $0, AX

	SUBQ  0(SI), R8
	SBBQ  8(SI), R9
	SBBQ 16(SI), R10
	SBBQ 24(SI), R11

	CMOVQCC AX, R12
	CMOVQCC AX, R13
	CMOVQCC AX, R14
	CMOVQCC AX, R15

	ADDQ R12, R8
	ADCQ R13, R9
	ADCQ R14, R10
	ADCQ R15, R11

	MOVQ c+0(FP), DI
	storeBlock(R8,R9,R10,R11, 0(DI))
	RET

TEXT ·gfpMul(SB),0,$160-24
	MOVQ a+8(FP), DI
	MOVQ b+16(FP), SI

	// Jump to a slightly different implementation if MULX isn't supported.
	CMPB runtime·support_bmi2(SB), $0
	JE   nobmi2Mul

	mulBMI2(0(DI),8(DI),16(DI),24(DI), 0(SI))
	storeBlock( R8, R9,R10,R11,  0(SP))
	storeBlock(R12,R13,R14,R15, 32(SP))
	gfpReduceBMI2()
	JMP end

nobmi2Mul:
	mul(0(DI),8(DI),16(DI),24(DI), 0(SI), 0(SP))
	gfpReduce(0(SP))

end:
	MOVQ c+0(FP), DI
	storeBlock(R12,R13,R14,R15, 0(DI))
	RET
