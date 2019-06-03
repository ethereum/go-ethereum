// Copyright 2019 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

// Copyright (c) 2018 Timo Savola. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package vm

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"log"
	"os"
	"reflect"
	"syscall"
	"unsafe"

	"github.com/ethereum/go-ethereum/core/vm/aot"

	"github.com/tsavola/wag"
	"github.com/tsavola/wag/buffer"
	"github.com/tsavola/wag/compile"
	"github.com/tsavola/wag/object/debug/dump"
	"github.com/tsavola/wag/wa"
)

const linearMemoryAddressSpace = 8 * 1024 * 1024 * 1024
const signalStackReserve = 8192
const goStackVectorOffset = -4 * 8

type importFunc struct {
	index  int
	params int
}

func makeMem(size int, prot, extraFlags int) (mem []byte, err error) {
	if size > 0 {
		mem, err = syscall.Mmap(-1, 0, size, prot, syscall.MAP_PRIVATE|syscall.MAP_ANONYMOUS|extraFlags)
	}
	return
}

func memAddr(mem []byte) uintptr {
	return (*reflect.SliceHeader)(unsafe.Pointer(&mem)).Data
}

func alignSize(size, alignment int) int {
	return (size + (alignment - 1)) &^ (alignment - 1)
}

func GrowMemory(size int32) {

}

// AoTContract
type AoTContract struct {
	Code         []byte
	Num          int
	ImportVector []byte
	ImportFuncs  map[string]importFunc
}

func NewAoTContract(code []byte, num int) *AoTContract {
	ret := &AoTContract{Code: code, Num: num}
	ret.ImportFuncs = make(map[string]importFunc)
	ret.ImportVector = make([]byte, (5+4)*8)

	binary.LittleEndian.PutUint64(ret.ImportVector[0:], aot.ImportUseGas())
	binary.LittleEndian.PutUint64(ret.ImportVector[8:], aot.ImportCallDataCopy())
	binary.LittleEndian.PutUint64(ret.ImportVector[16:], aot.ImportGetCallDataSize())
	binary.LittleEndian.PutUint64(ret.ImportVector[24:], aot.ImportFinish())
	binary.LittleEndian.PutUint64(ret.ImportVector[32:], aot.ImportRevert())
	ret.ImportFuncs["useGas"] = importFunc{-9, 1}
	ret.ImportFuncs["callDataCopy"] = importFunc{-8, 3}
	ret.ImportFuncs["getCallDataSize"] = importFunc{-7, 0}
	ret.ImportFuncs["finish"] = importFunc{-6, 2}
	ret.ImportFuncs["revert"] = importFunc{-5, 2}

	binary.LittleEndian.PutUint64(ret.ImportVector[56:], aot.ImportGrowMemoryHandler())

	return ret
}

func (ac *AoTContract) ResolveFunc(module, field string, sig wa.FuncType) (index int, err error) {
	if module != "ethereum" {
		err = fmt.Errorf("import function's module is unknown: %s %s", module, field)
		return
	}

	i := ac.ImportFuncs[field]
	if i.index == 0 {
		err = fmt.Errorf("import function not supported: %s", field)
		return
	}
	if len(sig.Params) != i.params {
		err = fmt.Errorf("%s: import function has wrong number of parameters: import signature has %d, syscall wrapper has %d", field, len(sig.Params), i.params)
		return
	}

	index = i.index
	return
}

func (ac *AoTContract) ResolveGlobal(module, field string, t wa.Type) (init uint64, err error) {
	err = fmt.Errorf("imported global not supported: %s %s", module, field)
	return
}

func (ac *AoTContract) RequiredGas(code []byte) uint64 {
	return 0
}

func (ac *AoTContract) Run(input []byte, contract *Contract) ([]byte, error) {
	var (
		textSize  = compile.DefaultMaxTextSize
		stackSize = wa.PageSize
		entry     = "main"
		dumpText  = false
	)

	progReader := bytes.NewReader(ac.Code)

	vecSize := alignSize(len(ac.ImportVector), os.Getpagesize())

	vecTextMem, err := makeMem(vecSize+textSize, syscall.PROT_READ|syscall.PROT_WRITE, 0)
	if err != nil {
		log.Fatal("error allocating vector+text memory: ", err)
	}
	defer func() {
		err = syscall.Munmap(vecTextMem)
		if err != nil {
			log.Fatal("error freeing vector+text memory", err)
		}
	}()

	vecMem := vecTextMem[:vecSize]
	copy(vecMem[vecSize-len(ac.ImportVector):], ac.ImportVector)

	contractData := make([]byte, 8+8+8+len(input) /* original rsp + gas + size + data */)
	binary.LittleEndian.PutUint64(contractData[8:], contract.Gas)
	binary.LittleEndian.PutUint64(contractData[16:], uint64(len(input)))
	copy(contractData[24:], input)
	cdAddr := uint64(memAddr(contractData))
	binary.LittleEndian.PutUint64(vecTextMem[vecSize-4*8:], cdAddr)

	textMem := vecTextMem[vecSize:]
	textAddr := memAddr(textMem)
	textBuf := buffer.NewStatic(textMem[:0], len(textMem))

	config := &wag.Config{
		Text:            textBuf,
		MemoryAlignment: os.Getpagesize(),
		Entry:           entry,
	}
	obj, err := wag.Compile(config, progReader, ac)
	if dumpText && len(obj.Text) > 0 {
		e := dump.Text(os.Stdout, obj.Text, textAddr, obj.FuncAddrs, &obj.Names)
		if err == nil {
			err = e
		}
	}
	if err != nil {
		log.Fatal("Error compiling the program:", err)
	}

	binary.LittleEndian.PutUint64(ac.ImportVector[40:], uint64(obj.InitialMemorySize))

	globalsMemory, err := makeMem(obj.MemoryOffset+linearMemoryAddressSpace, syscall.PROT_NONE, 0)
	if err != nil {
		log.Fatal("error allocating memory for globals:", err)
	}
	defer func() {
		err = syscall.Munmap(globalsMemory)
		if err != nil {
			log.Fatal("error freeing globals:", err)
		}
	}()

	err = syscall.Mprotect(globalsMemory[:obj.MemoryOffset+obj.InitialMemorySize], syscall.PROT_READ|syscall.PROT_WRITE)
	if err != nil {
		log.Fatal("error changing globals mem protection:", err)
	}

	copy(globalsMemory, obj.GlobalsMemory)

	memoryAddr := memAddr(globalsMemory) + uintptr(obj.MemoryOffset)

	if err := syscall.Mprotect(vecMem, syscall.PROT_READ); err != nil {
		log.Fatal("error changing protection for vector memory:", err)
	}

	if err := syscall.Mprotect(textMem, syscall.PROT_READ|syscall.PROT_EXEC); err != nil {
		log.Fatal("error changing text segment protection: ", err)
	}

	stackMem, err := makeMem(stackSize, syscall.PROT_READ|syscall.PROT_WRITE, syscall.MAP_STACK)
	if err != nil {
		log.Fatal("error allocating memory for the contract stack: ", err)
	}
	defer func() {
		err = syscall.Munmap(stackMem)
		if err != nil {
			log.Fatal("error freeing contract stack:", err)
		}
	}()
	stackOffset := stackSize - len(obj.StackFrame)
	copy(stackMem[stackOffset:], obj.StackFrame)

	stackAddr := memAddr(stackMem)
	stackLimit := stackAddr + signalStackReserve
	stackPtr := stackAddr + uintptr(stackOffset)

	if stackLimit >= stackPtr {
		log.Fatal("stack is too small for starting program")
	}

	retaddr, retsize := aot.Exec(textAddr, stackLimit, memoryAddr, stackPtr)

	gasLeft := binary.LittleEndian.Uint64(contractData[8:])
	if gasLeft == 0xffffffffffffffff {
		fmt.Println("Out of gas")
		return nil, ErrOutOfGas
	} else {
		fmt.Println("gas left: ", gasLeft, "result: ", globalsMemory[retaddr+obj.MemoryOffset:retaddr+obj.MemoryOffset+retsize])
		contract.Gas = gasLeft
		if retsize == 0 {
			return nil, nil
		}

		retData := make([]byte, retsize)
		copy(retData, globalsMemory[retaddr+obj.MemoryOffset:retaddr+obj.MemoryOffset+retsize])
		return retData, nil
	}

	// TODO cleanup all protected memory areas
}
