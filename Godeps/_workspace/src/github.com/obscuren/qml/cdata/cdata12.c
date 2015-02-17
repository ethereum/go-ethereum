// +build !go1.4

#include "runtime.h"

void ·Ref(uintptr ref) {
	ref = (uintptr)g->m;
	FLUSH(&ref);
}

void runtime·main(void);
void main·main(void);

void ·Addrs(uintptr rmain, uintptr mmain) {
	rmain = (uintptr)runtime·main;
	mmain = (uintptr)main·main;
	FLUSH(&rmain);
	FLUSH(&mmain);
}
