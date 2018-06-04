// Copyright 2017 The go-interpreter Authors.  All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package exec

import "errors"

var (
	// ErrSignatureMismatch is the error value used while trapping the VM when
	// a signature mismatch between the table entry and the type entry is found
	// in a call_indirect operation.
	ErrSignatureMismatch = errors.New("exec: signature mismatch in call_indirect")
	// ErrUndefinedElementIndex is the error value used while trapping the VM when
	// an invalid index to the module's table space is used as an operand to
	// call_indirect
	ErrUndefinedElementIndex = errors.New("exec: undefined element index")
)

func (vm *VM) call() {
	index := vm.fetchUint32()

	vm.funcs[index].call(vm, int64(index))
}

func (vm *VM) callIndirect() {
	index := vm.fetchUint32()
	fnExpect := vm.module.Types.Entries[index]
	_ = vm.fetchUint32() // reserved (https://github.com/WebAssembly/design/blob/27ac254c854994103c24834a994be16f74f54186/BinaryEncoding.md#call-operators-described-here)
	tableIndex := vm.popUint32()
	if int(tableIndex) >= len(vm.module.TableIndexSpace[0]) {
		panic(ErrUndefinedElementIndex)
	}
	elemIndex := vm.module.TableIndexSpace[0][tableIndex]
	fnActual := vm.module.FunctionIndexSpace[elemIndex]

	if len(fnExpect.ParamTypes) != len(fnActual.Sig.ParamTypes) {
		panic(ErrSignatureMismatch)
	}
	if len(fnExpect.ReturnTypes) != len(fnActual.Sig.ReturnTypes) {
		panic(ErrSignatureMismatch)
	}

	for i := range fnExpect.ParamTypes {
		if fnExpect.ParamTypes[i] != fnActual.Sig.ParamTypes[i] {
			panic(ErrSignatureMismatch)
		}
	}

	for i := range fnExpect.ReturnTypes {
		if fnExpect.ReturnTypes[i] != fnActual.Sig.ReturnTypes[i] {
			panic(ErrSignatureMismatch)
		}
	}

	vm.funcs[elemIndex].call(vm, int64(elemIndex))
}
