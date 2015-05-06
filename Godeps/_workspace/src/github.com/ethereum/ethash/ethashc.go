package ethash

/*
#cgo CFLAGS: -std=gnu99 -Wall
#cgo windows CFLAGS: -mno-stack-arg-probe
#cgo LDFLAGS: -lm

#include "src/libethash/internal.c"
#include "src/libethash/sha3.c"
#include "src/libethash/io.c"

#ifdef _WIN32
#	include "src/libethash/util_win32.c"
#	include "src/libethash/io_win32.c"
#	include "src/libethash/mmap_win32.c"
#else
#	include "src/libethash/io_posix.c"
#endif

// 'gateway function' for calling back into go.
extern int ethashGoCallback(unsigned);
int ethashGoCallback_cgo(unsigned percent) { return ethashGoCallback(percent); }

*/
import "C"
