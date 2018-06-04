// Copyright 2017 The go-interpreter Authors.  All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package exec

import (
	ops "github.com/go-interpreter/wagon/wasm/operators"
)

func (vm *VM) newFuncTable() {
	vm.funcTable[ops.I32Clz] = vm.i32Clz
	vm.funcTable[ops.I32Ctz] = vm.i32Ctz
	vm.funcTable[ops.I32Popcnt] = vm.i32Popcnt
	vm.funcTable[ops.I32Add] = vm.i32Add
	vm.funcTable[ops.I32Sub] = vm.i32Sub
	vm.funcTable[ops.I32Mul] = vm.i32Mul
	vm.funcTable[ops.I32DivS] = vm.i32DivS
	vm.funcTable[ops.I32DivU] = vm.i32DivU
	vm.funcTable[ops.I32RemS] = vm.i32RemS
	vm.funcTable[ops.I32RemU] = vm.i32RemU
	vm.funcTable[ops.I32And] = vm.i32And
	vm.funcTable[ops.I32Or] = vm.i32Or
	vm.funcTable[ops.I32Xor] = vm.i32Xor
	vm.funcTable[ops.I32Shl] = vm.i32Shl
	vm.funcTable[ops.I32ShrS] = vm.i32ShrS
	vm.funcTable[ops.I32ShrU] = vm.i32ShrU
	vm.funcTable[ops.I32Rotl] = vm.i32Rotl
	vm.funcTable[ops.I32Rotr] = vm.i32Rotr
	vm.funcTable[ops.I32Eqz] = vm.i32Eqz
	vm.funcTable[ops.I32Eq] = vm.i32Eq
	vm.funcTable[ops.I32Ne] = vm.i32Ne
	vm.funcTable[ops.I32LtS] = vm.i32LtS
	vm.funcTable[ops.I32LtU] = vm.i32LtU
	vm.funcTable[ops.I32GtS] = vm.i32GtS
	vm.funcTable[ops.I32GtU] = vm.i32GtU
	vm.funcTable[ops.I32LeS] = vm.i32LeS
	vm.funcTable[ops.I32LeU] = vm.i32LeU
	vm.funcTable[ops.I32GeS] = vm.i32GeS
	vm.funcTable[ops.I32GeU] = vm.i32GeU

	vm.funcTable[ops.I64Clz] = vm.i64Clz
	vm.funcTable[ops.I64Ctz] = vm.i64Ctz
	vm.funcTable[ops.I64Popcnt] = vm.i64Popcnt
	vm.funcTable[ops.I64Add] = vm.i64Add
	vm.funcTable[ops.I64Sub] = vm.i64Sub
	vm.funcTable[ops.I64Mul] = vm.i64Mul
	vm.funcTable[ops.I64DivS] = vm.i64DivS
	vm.funcTable[ops.I64DivU] = vm.i64DivU
	vm.funcTable[ops.I64RemS] = vm.i64RemS
	vm.funcTable[ops.I64RemU] = vm.i64RemU
	vm.funcTable[ops.I64And] = vm.i64And
	vm.funcTable[ops.I64Or] = vm.i64Or
	vm.funcTable[ops.I64Xor] = vm.i64Xor
	vm.funcTable[ops.I64Shl] = vm.i64Shl
	vm.funcTable[ops.I64ShrS] = vm.i64ShrS
	vm.funcTable[ops.I64ShrU] = vm.i64ShrU
	vm.funcTable[ops.I64Rotl] = vm.i64Rotl
	vm.funcTable[ops.I64Rotr] = vm.i64Rotr
	vm.funcTable[ops.I64Eqz] = vm.i64Eqz
	vm.funcTable[ops.I64Eq] = vm.i64Eq
	vm.funcTable[ops.I64Ne] = vm.i64Ne
	vm.funcTable[ops.I64LtS] = vm.i64LtS
	vm.funcTable[ops.I64LtU] = vm.i64LtU
	vm.funcTable[ops.I64GtS] = vm.i64GtS
	vm.funcTable[ops.I64GtU] = vm.i64GtU
	vm.funcTable[ops.I64LeS] = vm.i64LeS
	vm.funcTable[ops.I64LeU] = vm.i64LeU
	vm.funcTable[ops.I64GeS] = vm.i64GeS
	vm.funcTable[ops.I64GeU] = vm.i64GeU

	vm.funcTable[ops.F32Eq] = vm.f32Eq
	vm.funcTable[ops.F32Ne] = vm.f32Ne
	vm.funcTable[ops.F32Lt] = vm.f32Lt
	vm.funcTable[ops.F32Gt] = vm.f32Gt
	vm.funcTable[ops.F32Le] = vm.f32Le
	vm.funcTable[ops.F32Ge] = vm.f32Ge
	vm.funcTable[ops.F32Abs] = vm.f32Abs
	vm.funcTable[ops.F32Neg] = vm.f32Neg
	vm.funcTable[ops.F32Ceil] = vm.f32Ceil
	vm.funcTable[ops.F32Floor] = vm.f32Floor
	vm.funcTable[ops.F32Trunc] = vm.f32Trunc
	vm.funcTable[ops.F32Nearest] = vm.f32Nearest
	vm.funcTable[ops.F32Sqrt] = vm.f32Sqrt
	vm.funcTable[ops.F32Add] = vm.f32Add
	vm.funcTable[ops.F32Sub] = vm.f32Sub
	vm.funcTable[ops.F32Mul] = vm.f32Mul
	vm.funcTable[ops.F32Div] = vm.f32Div
	vm.funcTable[ops.F32Min] = vm.f32Min
	vm.funcTable[ops.F32Max] = vm.f32Max
	vm.funcTable[ops.F32Copysign] = vm.f32Copysign

	vm.funcTable[ops.F64Eq] = vm.f64Eq
	vm.funcTable[ops.F64Ne] = vm.f64Ne
	vm.funcTable[ops.F64Lt] = vm.f64Lt
	vm.funcTable[ops.F64Gt] = vm.f64Gt
	vm.funcTable[ops.F64Le] = vm.f64Le
	vm.funcTable[ops.F64Ge] = vm.f64Ge
	vm.funcTable[ops.F64Abs] = vm.f64Abs
	vm.funcTable[ops.F64Neg] = vm.f64Neg
	vm.funcTable[ops.F64Ceil] = vm.f64Ceil
	vm.funcTable[ops.F64Floor] = vm.f64Floor
	vm.funcTable[ops.F64Trunc] = vm.f64Trunc
	vm.funcTable[ops.F64Nearest] = vm.f64Nearest
	vm.funcTable[ops.F64Sqrt] = vm.f64Sqrt
	vm.funcTable[ops.F64Add] = vm.f64Add
	vm.funcTable[ops.F64Sub] = vm.f64Sub
	vm.funcTable[ops.F64Mul] = vm.f64Mul
	vm.funcTable[ops.F64Div] = vm.f64Div
	vm.funcTable[ops.F64Min] = vm.f64Min
	vm.funcTable[ops.F64Max] = vm.f64Max
	vm.funcTable[ops.F64Copysign] = vm.f64Copysign

	vm.funcTable[ops.I32Const] = vm.i32Const
	vm.funcTable[ops.I64Const] = vm.i64Const
	vm.funcTable[ops.F32Const] = vm.f32Const
	vm.funcTable[ops.F64Const] = vm.f64Const

	vm.funcTable[ops.I32ReinterpretF32] = vm.i32ReinterpretF32
	vm.funcTable[ops.I64ReinterpretF64] = vm.i64ReinterpretF64
	vm.funcTable[ops.F32ReinterpretI32] = vm.f32ReinterpretI32
	vm.funcTable[ops.F64ReinterpretI64] = vm.f64ReinterpretI64

	vm.funcTable[ops.I32WrapI64] = vm.i32Wrapi64
	vm.funcTable[ops.I32TruncSF32] = vm.i32TruncSF32
	vm.funcTable[ops.I32TruncUF32] = vm.i32TruncUF32
	vm.funcTable[ops.I32TruncSF64] = vm.i32TruncSF64
	vm.funcTable[ops.I32TruncUF64] = vm.i32TruncUF64
	vm.funcTable[ops.I64ExtendSI32] = vm.i64ExtendSI32
	vm.funcTable[ops.I64ExtendUI32] = vm.i64ExtendUI32
	vm.funcTable[ops.I64TruncSF32] = vm.i64TruncSF32
	vm.funcTable[ops.I64TruncUF32] = vm.i64TruncUF32
	vm.funcTable[ops.I64TruncSF64] = vm.i64TruncSF64
	vm.funcTable[ops.I64TruncUF64] = vm.i64TruncUF64
	vm.funcTable[ops.F32ConvertSI32] = vm.f32ConvertSI32
	vm.funcTable[ops.F32ConvertUI32] = vm.f32ConvertUI32
	vm.funcTable[ops.F32ConvertSI64] = vm.f32ConvertSI64
	vm.funcTable[ops.F32ConvertUI64] = vm.f32ConvertUI64
	vm.funcTable[ops.F32DemoteF64] = vm.f32DemoteF64
	vm.funcTable[ops.F64ConvertSI32] = vm.f64ConvertSI32
	vm.funcTable[ops.F64ConvertUI32] = vm.f64ConvertUI32
	vm.funcTable[ops.F64ConvertSI64] = vm.f64ConvertSI64
	vm.funcTable[ops.F64ConvertUI64] = vm.f64ConvertUI64
	vm.funcTable[ops.F64PromoteF32] = vm.f64PromoteF32

	vm.funcTable[ops.I32Load] = vm.i32Load
	vm.funcTable[ops.I64Load] = vm.i64Load
	vm.funcTable[ops.F32Load] = vm.f32Load
	vm.funcTable[ops.F64Load] = vm.f64Load
	vm.funcTable[ops.I32Load8s] = vm.i32Load8s
	vm.funcTable[ops.I32Load8u] = vm.i32Load8u
	vm.funcTable[ops.I32Load16s] = vm.i32Load16s
	vm.funcTable[ops.I32Load16u] = vm.i32Load16u
	vm.funcTable[ops.I64Load8s] = vm.i64Load8s
	vm.funcTable[ops.I64Load8u] = vm.i64Load8u
	vm.funcTable[ops.I64Load16s] = vm.i64Load16s
	vm.funcTable[ops.I64Load16u] = vm.i64Load16u
	vm.funcTable[ops.I64Load32s] = vm.i64Load32s
	vm.funcTable[ops.I64Load32u] = vm.i64Load32u
	vm.funcTable[ops.I32Store] = vm.i32Store
	vm.funcTable[ops.I64Store] = vm.i64Store
	vm.funcTable[ops.F32Store] = vm.f32Store
	vm.funcTable[ops.F64Store] = vm.f64Store
	vm.funcTable[ops.I32Store8] = vm.i32Store8
	vm.funcTable[ops.I32Store16] = vm.i32Store16
	vm.funcTable[ops.I64Store8] = vm.i64Store8
	vm.funcTable[ops.I64Store16] = vm.i64Store16
	vm.funcTable[ops.I64Store32] = vm.i64Store32
	vm.funcTable[ops.CurrentMemory] = vm.currentMemory
	vm.funcTable[ops.GrowMemory] = vm.growMemory

	vm.funcTable[ops.Drop] = vm.drop
	vm.funcTable[ops.Select] = vm.selectOp

	vm.funcTable[ops.GetLocal] = vm.getLocal
	vm.funcTable[ops.SetLocal] = vm.setLocal
	vm.funcTable[ops.TeeLocal] = vm.teeLocal
	vm.funcTable[ops.GetGlobal] = vm.getGlobal
	vm.funcTable[ops.SetGlobal] = vm.setGlobal

	vm.funcTable[ops.Unreachable] = vm.unreachable
	vm.funcTable[ops.Nop] = vm.nop

	vm.funcTable[ops.Call] = vm.call
	vm.funcTable[ops.CallIndirect] = vm.callIndirect
}
