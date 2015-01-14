package vm

import "math/big"
import "github.com/ethereum/go-ethereum/crypto"

/*
#include "../evmjit/libevmjit/interface.h"

#cgo LDFLAGS: -L/home/chfast/go/src/github.com/ethereum/go-ethereum/evmjit/build/libevmjit -levmjit
*/
import "C"

import "unsafe"
import "fmt"
import "reflect"

type JitVm struct {
	env Environment
	backup *DebugVm
}

func NewJitVm(env Environment) *JitVm {
	backupVm := NewDebugVm(env)
	return &JitVm{env: env, backup: backupVm}
}

func (self *JitVm) Run(me, caller ContextRef, code []byte, value, gas, price *big.Int, callData []byte) (ret []byte, err error) {
	r := C.evmjit_run();
	fmt.Printf("JIT result: %d", r);
	
	return self.backup.Run(me, caller, code, value, gas, price, callData)
}

func (self *JitVm) Printf(format string, v ...interface{}) VirtualMachine {
	return self.backup.Printf(format, v)
}

func (self *JitVm) Endl() VirtualMachine {
	return self.backup.Endl()
}

func (self *JitVm) Env() Environment {
	return self.env
}

//export env_sha3
func env_sha3(dataPtr unsafe.Pointer, length uint64, hashPtr unsafe.Pointer) {
	fmt.Printf("env_sha3(%p, %d, %p)\n", dataPtr, length, hashPtr);
	
	dataHdr := reflect.SliceHeader{
		Data: uintptr(dataPtr),
		Len:  int(length),
		Cap:  int(length),
	}
	data := *(*[]byte)(unsafe.Pointer(&dataHdr))
	fmt.Printf("\tdata: %x\n", data)
	
	hash := crypto.Sha3(data);
	fmt.Printf("\thash: %x\n", hash)
	
	hashHdr := reflect.SliceHeader{
		Data: uintptr(hashPtr),
		Len:  32,
		Cap:  32,
	}
	oHash := *(*[]byte)(unsafe.Pointer(&hashHdr))
	fmt.Printf("\tout0: %x\n", oHash)
	
	copy(oHash, hash)
	fmt.Printf("\tout1: %x\n", oHash)
}

//go is nice