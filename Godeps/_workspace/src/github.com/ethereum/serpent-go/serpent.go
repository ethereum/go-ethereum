package serpent

// #cgo CXXFLAGS: -I. -Ilangs/ -std=c++0x -Wall -fno-strict-aliasing
// #cgo LDFLAGS: -lstdc++
//
// #include "cpp/api.h"
//
import "C"

import (
	"encoding/hex"
	"errors"
	"unsafe"
)

func Compile(str string) ([]byte, error) {
	var err C.int
	out := C.GoString(C.compileGo(C.CString(str), (*C.int)(unsafe.Pointer(&err))))

	if err == C.int(1) {
		return nil, errors.New(out)
	}

	bytes, _ := hex.DecodeString(out)

	return bytes, nil
}
