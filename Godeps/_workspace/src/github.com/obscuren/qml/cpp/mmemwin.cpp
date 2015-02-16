#include <windows.h>

#define protREAD  1
#define protWRITE 2
#define protEXEC  4

extern "C" {

int mprotect(void *addr, size_t len, int prot)
{
	DWORD wprot = 0;
	if (prot & protWRITE) {
		wprot = PAGE_READWRITE;
	} else if (prot & protREAD) {
		wprot = PAGE_READONLY;
	}
	if (prot & protEXEC) {
		wprot <<= 4;
	}
	DWORD oldwprot;
	if (!VirtualProtect(addr, len, wprot, &oldwprot)) {
		return -1;
	}
	return 0;
}

} // extern "C"
