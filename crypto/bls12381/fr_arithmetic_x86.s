// +build amd64,!generic

#include "textflag.h"
#include "funcdata.h"


// func addFR(c *[4]uint64, a *[4]uint64, b *[4]uint64)
TEXT ·addFR(SB), NOSPLIT, $0-24
	// |
	MOVQ a+8(FP), DI
	MOVQ b+16(FP), SI

	// |
	MOVQ (DI), CX
	MOVQ 8(DI), DX
	MOVQ 16(DI), R8
	MOVQ 24(DI), R9
	ADDQ (SI), CX
	ADCQ 8(SI), DX
	ADCQ 16(SI), R8
	ADCQ 24(SI), R9

	// |
	MOVQ CX, R10
	MOVQ DX, R11
	MOVQ R8, R12
	MOVQ R9, R13
	SUBQ ·q+0(SB), R10
	SBBQ ·q+8(SB), R11
	SBBQ ·q+16(SB), R12
	SBBQ ·q+24(SB), R13

	// |
	MOVQ    c+0(FP), DI
	CMOVQCC R10, CX
	CMOVQCC R11, DX
	CMOVQCC R12, R8
	CMOVQCC R13, R9
	MOVQ    CX, (DI)
	MOVQ    DX, 8(DI)
	MOVQ    R8, 16(DI)
	MOVQ    R9, 24(DI)
	RET
/* end                                     */


// func laddAssignFR(a *[4]uint64, b *[4]uint64)
TEXT ·laddAssignFR(SB), NOSPLIT, $0-16
	// |
	MOVQ a+0(FP), DI
	MOVQ b+8(FP), SI

	// |
	MOVQ (DI), CX
	MOVQ 8(DI), DX
	MOVQ 16(DI), R8
	MOVQ 24(DI), R9
	ADDQ (SI), CX
	ADCQ 8(SI), DX
	ADCQ 16(SI), R8
	ADCQ 24(SI), R9
	MOVQ    CX, (DI)
	MOVQ    DX, 8(DI)
	MOVQ    R8, 16(DI)
	MOVQ    R9, 24(DI)
	RET
/* end                                     */


// func doubleFR(c *[4]uint64, a *[4]uint64)
TEXT ·doubleFR(SB), NOSPLIT, $0-16
	// |
	MOVQ a+8(FP), DI

	MOVQ (DI), CX
	MOVQ 8(DI), DX
	MOVQ 16(DI), SI
	MOVQ 24(DI), R8
	ADDQ CX, CX
	ADCQ DX, DX
	ADCQ SI, SI
	ADCQ R8, R8

	// |
	MOVQ CX, R9
	MOVQ DX, R10
	MOVQ SI, R11
	MOVQ R8, R12
	SUBQ ·q+0(SB), R9
	SBBQ ·q+8(SB), R10
	SBBQ ·q+16(SB), R11
	SBBQ ·q+24(SB), R12

	// |
	MOVQ    c+0(FP), DI
	CMOVQCC R9, CX
	CMOVQCC R10, DX
	CMOVQCC R11, SI
	CMOVQCC R12, R8
	MOVQ    CX, (DI)
	MOVQ    DX, 8(DI)
	MOVQ    SI, 16(DI)
	MOVQ    R8, 24(DI)
	RET
/* end                                     */


// func subFR(c *[4]uint64, a *[4]uint64, b *[4]uint64)
TEXT ·subFR(SB), NOSPLIT, $0-24
	// |
	MOVQ a+8(FP), DI
	MOVQ b+16(FP), SI
	XORQ AX, AX

	MOVQ (DI), CX
	MOVQ 8(DI), DX
	MOVQ 16(DI), R8
	MOVQ 24(DI), R9
	SUBQ (SI), CX
	SBBQ 8(SI), DX
	SBBQ 16(SI), R8
	SBBQ 24(SI), R9

	// |
	MOVQ    ·q+0(SB), SI
	MOVQ    ·q+8(SB), R10
	MOVQ    ·q+16(SB), R11
	MOVQ    ·q+24(SB), R12
	CMOVQCC AX, SI
	CMOVQCC AX, R10
	CMOVQCC AX, R11
	CMOVQCC AX, R12

	// |
	ADDQ SI, CX
	ADCQ R10, DX
	ADCQ R11, R8
	ADCQ R12, R9

	MOVQ c+0(FP), DI
	MOVQ CX, (DI)
	MOVQ DX, 8(DI)
	MOVQ R8, 16(DI)
	MOVQ R9, 24(DI)
	RET
/* end                                     */


// func lsubAssignFR(a *[4]uint64, b *[4]uint64)
TEXT ·lsubAssignFR(SB), NOSPLIT, $0-16
	// |
	MOVQ a+0(FP), DI
	MOVQ b+8(FP), SI

	MOVQ (DI), CX
	MOVQ 8(DI), DX
	MOVQ 16(DI), R8
	MOVQ 24(DI), R9
	SUBQ (SI), CX
	SBBQ 8(SI), DX
	SBBQ 16(SI), R8
	SBBQ 24(SI), R9
	MOVQ CX, (DI)
	MOVQ DX, 8(DI)
	MOVQ R8, 16(DI)
	MOVQ R9, 24(DI)
	RET
/* end                                     */


// func _negFR(c *[4]uint64, a *[4]uint64)
TEXT ·_negFR(SB), NOSPLIT, $0-16
	// |
	MOVQ a+8(FP), DI

	// |
	MOVQ ·q+0(SB), CX
	SUBQ (DI), CX
	MOVQ ·q+8(SB), DX
	SBBQ 8(DI), DX
	MOVQ ·q+16(SB), SI
	SBBQ 16(DI), SI
	MOVQ ·q+24(SB), R8
	SBBQ 24(DI), R8

	// |
	MOVQ c+0(FP), DI
	MOVQ CX, (DI)
	MOVQ DX, 8(DI)
	MOVQ SI, 16(DI)
	MOVQ R8, 24(DI)
	RET
/* end                                     */


// func mulFR(c *[4]uint64, a *[4]uint64, b *[4]uint64)
TEXT ·mulNoADXFR(SB), NOSPLIT, $0-24
	// | 

/* inputs                                  */

	MOVQ a+8(FP), DI
	MOVQ b+16(FP), SI
	MOVQ $0x00, R10
	MOVQ $0x00, R11
	MOVQ $0x00, R12
	MOVQ $0x00, R13
	MOVQ $0x00, R14

	// | 

/* i = 0                                   */

	// | a0 @ CX
	MOVQ (DI), CX

	// | a0 * b0 
	MOVQ (SI), AX
	MULQ CX
	MOVQ AX, R8
	MOVQ DX, R9

	// | a0 * b1 
	MOVQ 8(SI), AX
	MULQ CX
	ADDQ AX, R9
	ADCQ DX, R10

	// | a0 * b2 
	MOVQ 16(SI), AX
	MULQ CX
	ADDQ AX, R10
	ADCQ DX, R11

	// | a0 * b3 
	MOVQ 24(SI), AX
	MULQ CX
	ADDQ AX, R11
	ADCQ DX, R12

	// | 

/* i = 1                                   */

	// | a1 @ CX
	MOVQ 8(DI), CX
	MOVQ $0x00, BX

	// | a1 * b0 
	MOVQ (SI), AX
	MULQ CX
	ADDQ AX, R9
	ADCQ DX, R10
	ADCQ $0x00, R11
	ADCQ $0x00, BX

	// | a1 * b1 
	MOVQ 8(SI), AX
	MULQ CX
	ADDQ AX, R10
	ADCQ DX, R11
	ADCQ BX, R12
	MOVQ $0x00, BX
	ADCQ $0x00, BX

	// | a1 * b2 
	MOVQ 16(SI), AX
	MULQ CX
	ADDQ AX, R11
	ADCQ DX, R12
	ADCQ BX, R13

	// | a1 * b3 
	MOVQ 24(SI), AX
	MULQ CX
	ADDQ AX, R12
	ADCQ DX, R13

	// | 

/* i = 2                                   */

	// | a2 @ CX
	MOVQ 16(DI), CX
	MOVQ $0x00, BX

	// | a2 * b0 
	MOVQ (SI), AX
	MULQ CX
	ADDQ AX, R10
	ADCQ DX, R11
	ADCQ $0x00, R12
	ADCQ $0x00, BX

	// | a2 * b1 
	MOVQ 8(SI), AX
	MULQ CX
	ADDQ AX, R11
	ADCQ DX, R12
	ADCQ BX, R13
	MOVQ $0x00, BX
	ADCQ $0x00, BX

	// | a2 * b2 
	MOVQ 16(SI), AX
	MULQ CX
	ADDQ AX, R12
	ADCQ DX, R13
	ADCQ BX, R14

	// | a2 * b3 
	MOVQ 24(SI), AX
	MULQ CX
	ADDQ AX, R13
	ADCQ DX, R14

	// | 

/* i = 3                                   */

	// | a3 @ CX
	MOVQ 24(DI), CX
	MOVQ $0x00, BX

	// | a3 * b0 
	MOVQ (SI), AX
	MULQ CX
	ADDQ AX, R11
	ADCQ DX, R12
	ADCQ $0x00, R13
	ADCQ $0x00, BX

	// | a3 * b1 
	MOVQ 8(SI), AX
	MULQ CX
	ADDQ AX, R12
	ADCQ DX, R13
	ADCQ BX, R14
	MOVQ $0x00, BX
	ADCQ $0x00, BX

	// | a3 * b2 
	MOVQ 16(SI), AX
	MULQ CX
	ADDQ AX, R13
	ADCQ DX, R14
	ADCQ $0x00, BX

	// | a3 * b3 
	MOVQ 24(SI), AX
	MULQ CX
	ADDQ AX, R14
	ADCQ DX, BX

	// | 

/* 			                                     */

	// | 
	// | W
	// | 0   R8        | 1   R9        | 2   R10       | 3   R11       
	// | 4   R12       | 5   R13       | 6   R14       | 7   BX        


	// | 

/* montgomery reduction                    */

	// | 

/* i = 0                                   */

	// | 
	// | W
	// | 0   R8        | 1   R9        | 2   R10       | 3   R11       
	// | 4   R12       | 5   R13       | 6   R14       | 7   BX        


	// | | u0 = w0 * inp
	MOVQ R8, AX
	MULQ ·qinp+0(SB)
	MOVQ AX, DI
	MOVQ $0x00, CX

	// | 

/*                                         */

	// | j0

	// | w0 @ R8
	MOVQ ·q+0(SB), AX
	MULQ DI
	ADDQ AX, R8
	ADCQ DX, CX

	// | j1

	// | w1 @ R9
	MOVQ ·q+8(SB), AX
	MULQ DI
	ADDQ AX, R9
	ADCQ $0x00, DX
	ADDQ CX, R9
	MOVQ $0x00, CX
	ADCQ DX, CX

	// | j2

	// | w2 @ R10
	MOVQ ·q+16(SB), AX
	MULQ DI
	ADDQ AX, R10
	ADCQ $0x00, DX
	ADDQ CX, R10
	MOVQ $0x00, CX
	ADCQ DX, CX

	// | j3

	// | w3 @ R11
	MOVQ ·q+24(SB), AX
	MULQ DI
	ADDQ AX, R11
	ADCQ $0x00, DX
	ADDQ CX, R11

	// | w4 @ R12
	ADCQ DX, R12
	ADCQ $0x00, R8

	// | 

/* i = 1                                   */

	// | 
	// | W
	// | 0   -         | 1   R9        | 2   R10       | 3   R11       
	// | 4   R12       | 5   R13       | 6   R14       | 7   BX        


	// | | u1 = w1 * inp
	MOVQ R9, AX
	MULQ ·qinp+0(SB)
	MOVQ AX, DI
	MOVQ $0x00, CX

	// | 

/*                                         */

	// | j0

	// | w1 @ R9
	MOVQ ·q+0(SB), AX
	MULQ DI
	ADDQ AX, R9
	ADCQ DX, CX

	// | j1

	// | w2 @ R10
	MOVQ ·q+8(SB), AX
	MULQ DI
	ADDQ AX, R10
	ADCQ $0x00, DX
	ADDQ CX, R10
	MOVQ $0x00, CX
	ADCQ DX, CX

	// | j2

	// | w3 @ R11
	MOVQ ·q+16(SB), AX
	MULQ DI
	ADDQ AX, R11
	ADCQ $0x00, DX
	ADDQ CX, R11
	MOVQ $0x00, CX
	ADCQ DX, CX

	// | j3

	// | w4 @ R12
	MOVQ ·q+24(SB), AX
	MULQ DI
	ADDQ AX, R12
	ADCQ DX, R8
	ADDQ CX, R12

	// | w5 @ R13
	ADCQ R8, R13
	MOVQ $0x00, R8
	ADCQ $0x00, R8

	// | 

/* i = 2                                   */

	// | 
	// | W
	// | 0   -         | 1   -         | 2   R10       | 3   R11       
	// | 4   R12       | 5   R13       | 6   R14       | 7   BX        


	// | | u2 = w2 * inp
	MOVQ R10, AX
	MULQ ·qinp+0(SB)
	MOVQ AX, DI
	MOVQ $0x00, CX

	// | 

/*                                         */

	// | j0

	// | w2 @ R10
	MOVQ ·q+0(SB), AX
	MULQ DI
	ADDQ AX, R10
	ADCQ DX, CX

	// | j1

	// | w3 @ R11
	MOVQ ·q+8(SB), AX
	MULQ DI
	ADDQ AX, R11
	ADCQ $0x00, DX
	ADDQ CX, R11
	MOVQ $0x00, CX
	ADCQ DX, CX

	// | j2

	// | w4 @ R12
	MOVQ ·q+16(SB), AX
	MULQ DI
	ADDQ AX, R12
	ADCQ $0x00, DX
	ADDQ CX, R12
	MOVQ $0x00, CX
	ADCQ DX, CX

	// | j3

	// | w5 @ R13
	MOVQ ·q+24(SB), AX
	MULQ DI
	ADDQ AX, R13
	ADCQ DX, R8
	ADDQ CX, R13

	// | w6 @ R14
	ADCQ R8, R14
	MOVQ $0x00, R8
	ADCQ $0x00, R8

	// | 

/* i = 3                                   */

	// | 
	// | W
	// | 0   -         | 1   -         | 2   -         | 3   R11       
	// | 4   R12       | 5   R13       | 6   R14       | 7   BX        


	// | | u3 = w3 * inp
	MOVQ R11, AX
	MULQ ·qinp+0(SB)
	MOVQ AX, DI
	MOVQ $0x00, CX

	// | 

/*                                         */

	// | j0

	// | w3 @ R11
	MOVQ ·q+0(SB), AX
	MULQ DI
	ADDQ AX, R11
	ADCQ DX, CX

	// | j1

	// | w4 @ R12
	MOVQ ·q+8(SB), AX
	MULQ DI
	ADDQ AX, R12
	ADCQ $0x00, DX
	ADDQ CX, R12
	MOVQ $0x00, CX
	ADCQ DX, CX

	// | j2

	// | w5 @ R13
	MOVQ ·q+16(SB), AX
	MULQ DI
	ADDQ AX, R13
	ADCQ $0x00, DX
	ADDQ CX, R13
	MOVQ $0x00, CX
	ADCQ DX, CX

	// | j3

	// | w6 @ R14
	MOVQ ·q+24(SB), AX
	MULQ DI
	ADDQ AX, R14
	ADCQ DX, R8
	ADDQ CX, R14

	// | w-1 @ BX
	ADCQ R8, BX
	MOVQ $0x00, R8
	ADCQ $0x00, R8

	// | 
	// | W montgomerry reduction ends
	// | 0   -         | 1   -         | 2   -         | 3   -         
	// | 4   R12       | 5   R13       | 6   R14       | 7   BX        


	// | 

/* modular reduction                       */

	MOVQ R12, SI
	SUBQ ·q+0(SB), SI
	MOVQ R13, R9
	SBBQ ·q+8(SB), R9
	MOVQ R14, R10
	SBBQ ·q+16(SB), R10
	MOVQ BX, R11
	SBBQ ·q+24(SB), R11
	SBBQ $0x00, R8

	// | 

/* out                                     */

	MOVQ    c+0(FP), R8
	CMOVQCC SI, R12
	MOVQ    R12, (R8)
	CMOVQCC R9, R13
	MOVQ    R13, 8(R8)
	CMOVQCC R10, R14
	MOVQ    R14, 16(R8)
	CMOVQCC R11, BX
	MOVQ    BX, 24(R8)
	RET
/* end                                     */


// func mulFR(c *[4]uint64, a *[4]uint64, b *[4]uint64)
TEXT ·mulADXFR(SB), NOSPLIT, $0-24
	// | 

/* inputs                                  */

	MOVQ a+8(FP), DI
	MOVQ b+16(FP), SI
	XORQ AX, AX

	// | 

/* i = 0                                   */

	// | a0 @ DX
	MOVQ (DI), DX

	// | a0 * b0 
	MULXQ (SI), CX, R8

	// | a0 * b1 
	MULXQ 8(SI), AX, R9
	ADCXQ AX, R8

	// | a0 * b2 
	MULXQ 16(SI), AX, R10
	ADCXQ AX, R9

	// | a0 * b3 
	MULXQ 24(SI), AX, R11
	ADCXQ AX, R10
	ADCQ  $0x00, R11

	// | 

/* i = 1                                   */

	// | a1 @ DX
	MOVQ 8(DI), DX
	XORQ R12, R12

	// | a1 * b0 
	MULXQ (SI), AX, BX
	ADOXQ AX, R8
	ADCXQ BX, R9

	// | a1 * b1 
	MULXQ 8(SI), AX, BX
	ADOXQ AX, R9
	ADCXQ BX, R10

	// | a1 * b2 
	MULXQ 16(SI), AX, BX
	ADOXQ AX, R10
	ADCXQ BX, R11

	// | a1 * b3 
	MULXQ 24(SI), AX, BX
	ADOXQ AX, R11
	ADOXQ R12, R12
	ADCXQ BX, R12

	// | 

/* i = 2                                   */

	// | a2 @ DX
	MOVQ 16(DI), DX
	XORQ R13, R13

	// | a2 * b0 
	MULXQ (SI), AX, BX
	ADOXQ AX, R9
	ADCXQ BX, R10

	// | a2 * b1 
	MULXQ 8(SI), AX, BX
	ADOXQ AX, R10
	ADCXQ BX, R11

	// | a2 * b2 
	MULXQ 16(SI), AX, BX
	ADOXQ AX, R11
	ADCXQ BX, R12

	// | a2 * b3 
	MULXQ 24(SI), AX, BX
	ADOXQ AX, R12
	ADOXQ R13, R13
	ADCXQ BX, R13

	// | 

/* i = 3                                   */

	// | a3 @ DX
	MOVQ 24(DI), DX
	XORQ DI, DI

	// | a3 * b0 
	MULXQ (SI), AX, BX
	ADOXQ AX, R10
	ADCXQ BX, R11

	// | a3 * b1 
	MULXQ 8(SI), AX, BX
	ADOXQ AX, R11
	ADCXQ BX, R12

	// | a3 * b2 
	MULXQ 16(SI), AX, BX
	ADOXQ AX, R12
	ADCXQ BX, R13

	// | a3 * b3 
	MULXQ 24(SI), AX, BX
	ADOXQ AX, R13
	ADOXQ BX, DI
	ADCQ  $0x00, DI

	// | 

/* 			                                     */

	// | 
	// | W
	// | 0   CX        | 1   R8        | 2   R9        | 3   R10       
	// | 4   R11       | 5   R12       | 6   R13       | 7   DI        


	// | 
	// | W ready to mont
	// | 0   CX        | 1   R8        | 2   R9        | 3   R10       
	// | 4   R11       | 5   R12       | 6   R13       | 7   DI        


	// | 

/* montgomery reduction                    */

	// | clear flags
	XORQ AX, AX

	// | 

/* i = 0                                   */

	// | 
	// | W
	// | 0   CX        | 1   R8        | 2   R9        | 3   R10       
	// | 4   R11       | 5   R12       | 6   R13       | 7   DI        


	// | | u0 = w0 * inp
	MOVQ  CX, DX
	MULXQ ·qinp+0(SB), DX, BX

	// | 

/*                                         */

	// | j0

	// | w0 @ CX
	MULXQ ·q+0(SB), AX, BX
	ADOXQ AX, CX
	ADCXQ BX, R8

	// | j1

	// | w1 @ R8
	MULXQ ·q+8(SB), AX, BX
	ADOXQ AX, R8
	ADCXQ BX, R9

	// | j2

	// | w2 @ R9
	MULXQ ·q+16(SB), AX, BX
	ADOXQ AX, R9
	ADCXQ BX, R10

	// | j3

	// | w3 @ R10
	MULXQ ·q+24(SB), AX, BX
	ADOXQ AX, R10
	ADCXQ BX, R11
	ADOXQ CX, R11
	ADCXQ CX, CX
	MOVQ  $0x00, AX
	ADOXQ AX, CX

	// | clear flags
	XORQ AX, AX

	// | 

/* i = 1                                   */

	// | 
	// | W
	// | 0   -         | 1   R8        | 2   R9        | 3   R10       
	// | 4   R11       | 5   R12       | 6   R13       | 7   DI        


	// | | u1 = w1 * inp
	MOVQ  R8, DX
	MULXQ ·qinp+0(SB), DX, BX

	// | 

/*                                         */

	// | j0

	// | w1 @ R8
	MULXQ ·q+0(SB), AX, BX
	ADOXQ AX, R8
	ADCXQ BX, R9

	// | j1

	// | w2 @ R9
	MULXQ ·q+8(SB), AX, BX
	ADOXQ AX, R9
	ADCXQ BX, R10

	// | j2

	// | w3 @ R10
	MULXQ ·q+16(SB), AX, BX
	ADOXQ AX, R10
	ADCXQ BX, R11

	// | j3

	// | w4 @ R11
	MULXQ ·q+24(SB), AX, BX
	ADOXQ AX, R11
	ADCXQ BX, R12
	ADOXQ CX, R12
	ADCXQ R8, R8
	MOVQ  $0x00, AX
	ADOXQ AX, R8

	// | clear flags
	XORQ AX, AX

	// | 

/* i = 2                                   */

	// | 
	// | W
	// | 0   -         | 1   -         | 2   R9        | 3   R10       
	// | 4   R11       | 5   R12       | 6   R13       | 7   DI        


	// | | u2 = w2 * inp
	MOVQ  R9, DX
	MULXQ ·qinp+0(SB), DX, BX

	// | 

/*                                         */

	// | j0

	// | w2 @ R9
	MULXQ ·q+0(SB), AX, BX
	ADOXQ AX, R9
	ADCXQ BX, R10

	// | j1

	// | w3 @ R10
	MULXQ ·q+8(SB), AX, BX
	ADOXQ AX, R10
	ADCXQ BX, R11

	// | j2

	// | w4 @ R11
	MULXQ ·q+16(SB), AX, BX
	ADOXQ AX, R11
	ADCXQ BX, R12

	// | j3

	// | w5 @ R12
	MULXQ ·q+24(SB), AX, BX
	ADOXQ AX, R12
	ADCXQ BX, R13
	ADOXQ R8, R13
	ADCXQ R9, R9
	MOVQ  $0x00, AX
	ADOXQ AX, R9

	// | clear flags
	XORQ AX, AX

	// | 

/* i = 3                                   */

	// | 
	// | W
	// | 0   -         | 1   -         | 2   -         | 3   R10       
	// | 4   R11       | 5   R12       | 6   R13       | 7   DI        


	// | | u3 = w3 * inp
	MOVQ  R10, DX
	MULXQ ·qinp+0(SB), DX, BX

	// | 

/*                                         */

	// | j0

	// | w3 @ R10
	MULXQ ·q+0(SB), AX, BX
	ADOXQ AX, R10
	ADCXQ BX, R11

	// | j1

	// | w4 @ R11
	MULXQ ·q+8(SB), AX, BX
	ADOXQ AX, R11
	ADCXQ BX, R12

	// | j2

	// | w5 @ R12
	MULXQ ·q+16(SB), AX, BX
	ADOXQ AX, R12
	ADCXQ BX, R13

	// | j3

	// | w6 @ R13
	MULXQ ·q+24(SB), AX, BX
	ADOXQ AX, R13
	ADCXQ BX, DI
	ADOXQ R9, DI
	ADCXQ R10, R10
	MOVQ  $0x00, AX
	ADOXQ AX, R10

	// | 
	// | W montgomery reduction ends
	// | 0   -         | 1   -         | 2   -         | 3   -         
	// | 4   R11       | 5   R12       | 6   R13       | 7   DI        


	// | 

/* modular reduction                       */

	MOVQ R11, CX
	SUBQ ·q+0(SB), CX
	MOVQ R12, AX
	SBBQ ·q+8(SB), AX
	MOVQ R13, BX
	SBBQ ·q+16(SB), BX
	MOVQ DI, SI
	SBBQ ·q+24(SB), SI
	SBBQ $0x00, R10

	// | 

/* out                                     */

	MOVQ    c+0(FP), R10
	CMOVQCC CX, R11
	MOVQ    R11, (R10)
	CMOVQCC AX, R12
	MOVQ    R12, 8(R10)
	CMOVQCC BX, R13
	MOVQ    R13, 16(R10)
	CMOVQCC SI, DI
	MOVQ    DI, 24(R10)
	RET
/* end                                     */


TEXT ·waddFR(SB), NOSPLIT, $0-16
	// |
	MOVQ a+0(FP), DI
	MOVQ b+8(FP), SI

	// |
	MOVQ (DI), R8
	MOVQ 8(DI), R9
	MOVQ 16(DI), R10
	MOVQ 24(DI), R11
	MOVQ 32(DI), R12
	MOVQ 40(DI), R13
	MOVQ 48(DI), R14
	MOVQ 56(DI), R15

	// |
	ADDQ (SI), R8
	ADCQ 8(SI), R9
	ADCQ 16(SI), R10
	ADCQ 24(SI), R11
	ADCQ 32(SI), R12
	ADCQ 40(SI), R13
	ADCQ 48(SI), R14
	ADCQ 56(SI), R15

	// |
	MOVQ R8, (DI)
	MOVQ R9, 8(DI)
	MOVQ R10, 16(DI)
	MOVQ R11, 24(DI)
	MOVQ R12, 32(DI)
	MOVQ R13, 40(DI)
	MOVQ R14, 48(DI)
	MOVQ R15, 56(DI)
	RET
/* end                                     */

// func wmulNoADXFR(c *[8]uint64, a *[4]uint64, b *[4]uint64)
TEXT ·wmulNoADXFR(SB), NOSPLIT, $0-24
	// | 

/* inputs                                  */

	MOVQ a+8(FP), DI
	MOVQ b+16(FP), SI
	MOVQ $0x00, R10
	MOVQ $0x00, R11
	MOVQ $0x00, R12
	MOVQ $0x00, R13
	MOVQ $0x00, R14

	// | 

/* i = 0                                   */

	// | a0 @ CX
	MOVQ (DI), CX

	// | a0 * b0 
	MOVQ (SI), AX
	MULQ CX
	MOVQ AX, R8
	MOVQ DX, R9

	// | a0 * b1 
	MOVQ 8(SI), AX
	MULQ CX
	ADDQ AX, R9
	ADCQ DX, R10

	// | a0 * b2 
	MOVQ 16(SI), AX
	MULQ CX
	ADDQ AX, R10
	ADCQ DX, R11

	// | a0 * b3 
	MOVQ 24(SI), AX
	MULQ CX
	ADDQ AX, R11
	ADCQ DX, R12

	// | 

/* i = 1                                   */

	// | a1 @ CX
	MOVQ 8(DI), CX
	MOVQ $0x00, BX

	// | a1 * b0 
	MOVQ (SI), AX
	MULQ CX
	ADDQ AX, R9
	ADCQ DX, R10
	ADCQ $0x00, R11
	ADCQ $0x00, BX

	// | a1 * b1 
	MOVQ 8(SI), AX
	MULQ CX
	ADDQ AX, R10
	ADCQ DX, R11
	ADCQ BX, R12
	MOVQ $0x00, BX
	ADCQ $0x00, BX

	// | a1 * b2 
	MOVQ 16(SI), AX
	MULQ CX
	ADDQ AX, R11
	ADCQ DX, R12
	ADCQ BX, R13

	// | a1 * b3 
	MOVQ 24(SI), AX
	MULQ CX
	ADDQ AX, R12
	ADCQ DX, R13

	// | 

/* i = 2                                   */

	// | a2 @ CX
	MOVQ 16(DI), CX
	MOVQ $0x00, BX

	// | a2 * b0 
	MOVQ (SI), AX
	MULQ CX
	ADDQ AX, R10
	ADCQ DX, R11
	ADCQ $0x00, R12
	ADCQ $0x00, BX

	// | a2 * b1 
	MOVQ 8(SI), AX
	MULQ CX
	ADDQ AX, R11
	ADCQ DX, R12
	ADCQ BX, R13
	MOVQ $0x00, BX
	ADCQ $0x00, BX

	// | a2 * b2 
	MOVQ 16(SI), AX
	MULQ CX
	ADDQ AX, R12
	ADCQ DX, R13
	ADCQ BX, R14

	// | a2 * b3 
	MOVQ 24(SI), AX
	MULQ CX
	ADDQ AX, R13
	ADCQ DX, R14

	// | 

/* i = 3                                   */

	// | a3 @ CX
	MOVQ 24(DI), CX
	MOVQ $0x00, BX

	// | a3 * b0 
	MOVQ (SI), AX
	MULQ CX
	ADDQ AX, R11
	ADCQ DX, R12
	ADCQ $0x00, R13
	ADCQ $0x00, BX

	// | a3 * b1 
	MOVQ 8(SI), AX
	MULQ CX
	ADDQ AX, R12
	ADCQ DX, R13
	ADCQ BX, R14
	MOVQ $0x00, BX
	ADCQ $0x00, BX

	// | a3 * b2 
	MOVQ 16(SI), AX
	MULQ CX
	ADDQ AX, R13
	ADCQ DX, R14
	ADCQ $0x00, BX

	// | a3 * b3 
	MOVQ 24(SI), AX
	MULQ CX
	ADDQ AX, R14
	ADCQ DX, BX

	// | 

/* 			                                   */

	// | 
	// | W
	// | 0   R8        | 1   R9        | 2   R10       | 3   R11       
	// | 4   R12       | 5   R13       | 6   R14       | 7   BX      

	MOVQ c+0(FP), AX
	MOVQ R8, (AX)
	MOVQ R9, 8(AX)
	MOVQ R10, 16(AX)
	MOVQ R11, 24(AX)
	MOVQ R12, 32(AX)
	MOVQ R13, 40(AX)
	MOVQ R14, 48(AX)
	MOVQ BX, 56(AX)

	RET
/* end 			                                 */


// func wmulADXFR(c *[8]uint64, a *[4]uint64, b *[4]uint64)
TEXT ·wmulADXFR(SB), NOSPLIT, $0-24
	// | 

/* inputs                                  */

	MOVQ a+8(FP), DI
	MOVQ b+16(FP), SI
	XORQ AX, AX

	// | 

/* i = 0                                   */

	// | a0 @ DX
	MOVQ (DI), DX

	// | a0 * b0 
	MULXQ (SI), CX, R8

	// | a0 * b1 
	MULXQ 8(SI), AX, R9
	ADCXQ AX, R8

	// | a0 * b2 
	MULXQ 16(SI), AX, R10
	ADCXQ AX, R9

	// | a0 * b3 
	MULXQ 24(SI), AX, R11
	ADCXQ AX, R10
	ADCQ  $0x00, R11

	// | 

/* i = 1                                   */

	// | a1 @ DX
	MOVQ 8(DI), DX
	XORQ R12, R12

	// | a1 * b0 
	MULXQ (SI), AX, BX
	ADOXQ AX, R8
	ADCXQ BX, R9

	// | a1 * b1 
	MULXQ 8(SI), AX, BX
	ADOXQ AX, R9
	ADCXQ BX, R10

	// | a1 * b2 
	MULXQ 16(SI), AX, BX
	ADOXQ AX, R10
	ADCXQ BX, R11

	// | a1 * b3 
	MULXQ 24(SI), AX, BX
	ADOXQ AX, R11
	ADOXQ R12, R12
	ADCXQ BX, R12

	// | 

/* i = 2                                   */

	// | a2 @ DX
	MOVQ 16(DI), DX
	XORQ R13, R13

	// | a2 * b0 
	MULXQ (SI), AX, BX
	ADOXQ AX, R9
	ADCXQ BX, R10

	// | a2 * b1 
	MULXQ 8(SI), AX, BX
	ADOXQ AX, R10
	ADCXQ BX, R11

	// | a2 * b2 
	MULXQ 16(SI), AX, BX
	ADOXQ AX, R11
	ADCXQ BX, R12

	// | a2 * b3 
	MULXQ 24(SI), AX, BX
	ADOXQ AX, R12
	ADOXQ R13, R13
	ADCXQ BX, R13

	// | 

/* i = 3                                   */

	// | a3 @ DX
	MOVQ 24(DI), DX
	XORQ DI, DI

	// | a3 * b0 
	MULXQ (SI), AX, BX
	ADOXQ AX, R10
	ADCXQ BX, R11

	// | a3 * b1 
	MULXQ 8(SI), AX, BX
	ADOXQ AX, R11
	ADCXQ BX, R12

	// | a3 * b2 
	MULXQ 16(SI), AX, BX
	ADOXQ AX, R12
	ADCXQ BX, R13

	// | a3 * b3 
	MULXQ 24(SI), AX, BX
	ADOXQ AX, R13
	ADOXQ BX, DI
	ADCQ  $0x00, DI

	// | 

/* 			                                   */

	// | 
	// | W
	// | 0   CX        | 1   R8        | 2   R9        | 3   R10       
	// | 4   R11       | 5   R12       | 6   R13       | 7   DI        


	MOVQ c+0(FP), AX
	MOVQ CX, (AX)
	MOVQ R8, 8(AX)
	MOVQ R9, 16(AX)
	MOVQ R10, 24(AX)
	MOVQ R11, 32(AX)
	MOVQ R12, 40(AX)
	MOVQ R13, 48(AX)
	MOVQ DI, 56(AX)

	RET

/* end                                     */