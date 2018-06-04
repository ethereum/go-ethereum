// Copyright 2017 The go-interpreter Authors.  All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package exec

func (vm *VM) getLocal() {
	index := vm.fetchUint32()
	vm.pushUint64(vm.ctx.locals[int(index)])
}

func (vm *VM) setLocal() {
	index := vm.fetchUint32()
	vm.ctx.locals[int(index)] = vm.popUint64()
}

func (vm *VM) teeLocal() {
	index := vm.fetchUint32()
	val := vm.ctx.stack[len(vm.ctx.stack)-1]
	vm.ctx.locals[int(index)] = val
}

func (vm *VM) getGlobal() {
	index := vm.fetchUint32()
	vm.pushUint64(vm.globals[int(index)])
}

func (vm *VM) setGlobal() {
	index := vm.fetchUint32()
	vm.globals[int(index)] = vm.popUint64()
}
