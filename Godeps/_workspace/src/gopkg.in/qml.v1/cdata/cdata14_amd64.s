// +build go1.4

#include "textflag.h"

TEXT ·Ref(SB),NOSPLIT,$8-8
	CALL runtime·acquirem(SB)
	MOVQ 0(SP), AX
	MOVQ AX, ret+0(FP)
	CALL runtime·releasem(SB)
	RET

TEXT ·Addrs(SB),NOSPLIT,$0-16
	MOVQ	$runtime·main(SB), AX
	MOVQ	AX, ret+0(FP)
	MOVQ	$runtime·main_main(SB), AX
	MOVQ	AX, ret+8(FP)
	RET
