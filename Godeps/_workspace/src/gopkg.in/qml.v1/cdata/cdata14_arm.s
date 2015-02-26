// +build go1.4

#include "textflag.h"

TEXT ·Ref(SB),NOSPLIT,$4-4
	BL runtime·acquirem(SB)
	MOVW 4(R13), R0
	MOVW R0, ret+0(FP)
	MOVW R0, 4(R13)
	BL runtime·releasem(SB)
	RET

TEXT ·Addrs(SB),NOSPLIT,$0-8
	MOVW	$runtime·main(SB), R0
	MOVW	R0, ret+0(FP)
	MOVW	$runtime·main_main(SB), R0
	MOVW	R0, ret+4(FP)
	RET
