package qml

// #include <stdlib.h>
// int mprotect(void *addr, size_t len, int prot);
import "C"

import (
	"bytes"
	"encoding/binary"
	"gopkg.in/qml.v1/cdata"
	"reflect"
	"unsafe"
)

const pageSize = 4096

func qmain() {
	Run(func() error { tmain(); return nil })
}

func tmain() { tstub() }
func tstub() { tstub() }

func SetupTesting() {
	ptr := func(f func()) uintptr { return reflect.ValueOf(f).Pointer() }
	rmain, mmain := cdata.Addrs()
	fset(rmain, mmain, ptr(qmain))
	fset(ptr(tmain), ptr(tstub), mmain)
}

const (
	protREAD  = 1
	protWRITE = 2
	protEXEC  = 4
)

func fset(target, old, new uintptr) {
	pageOffset := target % pageSize
	pageAddr := target - pageOffset

	var mem []byte
	memh := (*reflect.SliceHeader)(unsafe.Pointer(&mem))
	memh.Data = pageAddr
	memh.Len = pageSize * 2
	memh.Cap = pageSize * 2

	oldAddr := make([]byte, 8)
	newAddr := make([]byte, 8)

	binary.LittleEndian.PutUint64(oldAddr, uint64(old))
	binary.LittleEndian.PutUint64(newAddr, uint64(new))

	// BSD's syscall package misses Mprotect. Use cgo instead.
	C.mprotect(unsafe.Pointer(pageAddr), C.size_t(len(mem)), protEXEC|protREAD|protWRITE)
	defer C.mprotect(unsafe.Pointer(pageAddr), C.size_t(len(mem)), protEXEC|protREAD)

	delta := make([]byte, 4)
	for i, c := range mem[pageOffset:] {
		if c == 0xe8 && int(pageOffset)+i+5 < len(mem) {
			instrAddr := pageAddr + pageOffset + uintptr(i)
			binary.LittleEndian.PutUint32(delta, uint32(old-instrAddr-5))
			if bytes.Equal(mem[int(pageOffset)+i+1:int(pageOffset)+i+5], delta) {
				binary.LittleEndian.PutUint32(mem[int(pageOffset)+i+1:], uint32(new-instrAddr-5))
				return
			}
		}
	}
	panic("cannot setup qml package for testing")
}
