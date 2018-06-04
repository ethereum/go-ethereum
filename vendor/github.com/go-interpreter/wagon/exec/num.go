// Copyright 2017 The go-interpreter Authors.  All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package exec

import (
	"math"
	"math/bits"
)

// int32 operators

func (vm *VM) i32Clz() {
	vm.pushUint64(uint64(bits.LeadingZeros32(vm.popUint32())))
}

func (vm *VM) i32Ctz() {
	vm.pushUint64(uint64(bits.TrailingZeros32(vm.popUint32())))
}

func (vm *VM) i32Popcnt() {
	vm.pushUint64(uint64(bits.OnesCount32(vm.popUint32())))
}

func (vm *VM) i32Add() {
	vm.pushUint32(vm.popUint32() + vm.popUint32())
}

func (vm *VM) i32Mul() {
	vm.pushUint32(vm.popUint32() * vm.popUint32())
}

func (vm *VM) i32DivS() {
	v2 := vm.popInt32()
	v1 := vm.popInt32()
	vm.pushInt32(v1 / v2)
}

func (vm *VM) i32DivU() {
	v2 := vm.popUint32()
	v1 := vm.popUint32()
	vm.pushUint32(v1 / v2)
}

func (vm *VM) i32RemS() {
	v2 := vm.popInt32()
	v1 := vm.popInt32()
	vm.pushInt32(v1 % v2)
}

func (vm *VM) i32RemU() {
	v2 := vm.popUint32()
	v1 := vm.popUint32()
	vm.pushUint32(v1 % v2)
}

func (vm *VM) i32Sub() {
	v2 := vm.popUint32()
	v1 := vm.popUint32()
	vm.pushUint32(v1 - v2)
}

func (vm *VM) i32And() {
	vm.pushUint32(vm.popUint32() & vm.popUint32())
}

func (vm *VM) i32Or() {
	vm.pushUint32(vm.popUint32() | vm.popUint32())
}

func (vm *VM) i32Xor() {
	vm.pushUint32(vm.popUint32() ^ vm.popUint32())
}

func (vm *VM) i32Shl() {
	v2 := vm.popUint32()
	v1 := vm.popUint32()
	vm.pushUint32(v1 << v2)
}

func (vm *VM) i32ShrU() {
	v2 := vm.popUint32()
	v1 := vm.popUint32()
	vm.pushUint32(v1 >> v2)
}

func (vm *VM) i32ShrS() {
	v2 := vm.popUint32()
	v1 := vm.popInt32()
	vm.pushInt32(v1 >> v2)
}

func (vm *VM) i32Rotl() {
	v2 := vm.popUint32()
	v1 := vm.popUint32()
	vm.pushUint32(bits.RotateLeft32(v1, int(v2)))
}

func (vm *VM) i32Rotr() {
	v2 := vm.popUint32()
	v1 := vm.popUint32()
	vm.pushUint32(bits.RotateLeft32(v1, -int(v2)))
}

func (vm *VM) i32LeS() {
	v2 := vm.popInt32()
	v1 := vm.popInt32()
	vm.pushBool(v1 <= v2)
}

func (vm *VM) i32LeU() {
	v2 := vm.popUint32()
	v1 := vm.popUint32()
	vm.pushBool(v1 <= v2)
}

func (vm *VM) i32LtS() {
	v2 := vm.popInt32()
	v1 := vm.popInt32()
	vm.pushBool(v1 < v2)
}

func (vm *VM) i32LtU() {
	v2 := vm.popUint32()
	v1 := vm.popUint32()
	vm.pushBool(v1 < v2)
}

func (vm *VM) i32GtS() {
	v2 := vm.popInt32()
	v1 := vm.popInt32()
	vm.pushBool(v1 > v2)
}

func (vm *VM) i32GeS() {
	v2 := vm.popInt32()
	v1 := vm.popInt32()
	vm.pushBool(v1 >= v2)
}

func (vm *VM) i32GtU() {
	v2 := vm.popUint32()
	v1 := vm.popUint32()
	vm.pushBool(v1 > v2)
}

func (vm *VM) i32GeU() {
	v2 := vm.popUint32()
	v1 := vm.popUint32()
	vm.pushBool(v1 >= v2)
}

func (vm *VM) i32Eqz() {
	vm.pushBool(vm.popUint32() == 0)
}

func (vm *VM) i32Eq() {
	vm.pushBool(vm.popUint32() == vm.popUint32())
}

func (vm *VM) i32Ne() {
	vm.pushBool(vm.popUint32() != vm.popUint32())
}

// int64 operators

func (vm *VM) i64Clz() {
	vm.pushUint64(uint64(bits.LeadingZeros64(vm.popUint64())))
}

func (vm *VM) i64Ctz() {
	vm.pushUint64(uint64(bits.TrailingZeros64(vm.popUint64())))
}

func (vm *VM) i64Popcnt() {
	vm.pushUint64(uint64(bits.OnesCount64(vm.popUint64())))
}

func (vm *VM) i64Add() {
	vm.pushUint64(vm.popUint64() + vm.popUint64())
}

func (vm *VM) i64Sub() {
	v2 := vm.popUint64()
	v1 := vm.popUint64()
	vm.pushUint64(v1 - v2)
}

func (vm *VM) i64Mul() {
	vm.pushUint64(vm.popUint64() * vm.popUint64())
}

func (vm *VM) i64DivS() {
	v2 := vm.popInt64()
	v1 := vm.popInt64()
	vm.pushInt64(v1 / v2)
}

func (vm *VM) i64DivU() {
	v2 := vm.popUint64()
	v1 := vm.popUint64()
	vm.pushUint64(v1 / v2)
}

func (vm *VM) i64RemS() {
	v2 := vm.popInt64()
	v1 := vm.popInt64()
	vm.pushInt64(v1 % v2)
}

func (vm *VM) i64RemU() {
	v2 := vm.popUint64()
	v1 := vm.popUint64()
	vm.pushUint64(v1 % v2)
}

func (vm *VM) i64And() {
	vm.pushUint64(vm.popUint64() & vm.popUint64())
}

func (vm *VM) i64Or() {
	vm.pushUint64(vm.popUint64() | vm.popUint64())
}

func (vm *VM) i64Xor() {
	vm.pushUint64(vm.popUint64() ^ vm.popUint64())
}

func (vm *VM) i64Shl() {
	v2 := vm.popUint64()
	v1 := vm.popUint64()
	vm.pushUint64(v1 << v2)
}

func (vm *VM) i64ShrS() {
	v2 := vm.popUint64()
	v1 := vm.popInt64()
	vm.pushInt64(v1 >> v2)
}

func (vm *VM) i64ShrU() {
	v2 := vm.popUint64()
	v1 := vm.popUint64()
	vm.pushUint64(v1 >> v2)
}

func (vm *VM) i64Rotl() {
	v2 := vm.popInt64()
	v1 := vm.popUint64()
	vm.pushUint64(bits.RotateLeft64(v1, int(v2)))
}

func (vm *VM) i64Rotr() {
	v2 := vm.popInt64()
	v1 := vm.popUint64()
	vm.pushUint64(bits.RotateLeft64(v1, -int(v2)))
}

func (vm *VM) i64Eq() {
	vm.pushBool(vm.popUint64() == vm.popUint64())
}

func (vm *VM) i64Eqz() {
	vm.pushBool(vm.popUint64() == 0)
}

func (vm *VM) i64Ne() {
	vm.pushBool(vm.popUint64() != vm.popUint64())
}

func (vm *VM) i64LtS() {
	v2 := vm.popInt64()
	v1 := vm.popInt64()
	vm.pushBool(v1 < v2)
}

func (vm *VM) i64LtU() {
	v2 := vm.popUint64()
	v1 := vm.popUint64()
	vm.pushBool(v1 < v2)
}

func (vm *VM) i64GtS() {
	v2 := vm.popInt64()
	v1 := vm.popInt64()
	vm.pushBool(v1 > v2)
}

func (vm *VM) i64GtU() {
	v2 := vm.popUint64()
	v1 := vm.popUint64()
	vm.pushBool(v1 > v2)
}

func (vm *VM) i64LeU() {
	v2 := vm.popUint64()
	v1 := vm.popUint64()
	vm.pushBool(v1 <= v2)
}

func (vm *VM) i64LeS() {
	v2 := vm.popInt64()
	v1 := vm.popInt64()
	vm.pushBool(v1 <= v2)
}

func (vm *VM) i64GeS() {
	v2 := vm.popInt64()
	v1 := vm.popInt64()
	vm.pushBool(v1 >= v2)
}

func (vm *VM) i64GeU() {
	v2 := vm.popUint64()
	v1 := vm.popUint64()
	vm.pushBool(v1 >= v2)
}

// float32 operators

func (vm *VM) f32Abs() {
	vm.pushFloat32(float32(math.Abs(float64(vm.popFloat32()))))
}

func (vm *VM) f32Neg() {
	vm.pushFloat32(-vm.popFloat32())
}

func (vm *VM) f32Ceil() {
	vm.pushFloat32(float32(math.Ceil(float64(vm.popFloat32()))))
}

func (vm *VM) f32Floor() {
	vm.pushFloat32(float32(math.Floor(float64(vm.popFloat32()))))
}

func (vm *VM) f32Trunc() {
	vm.pushFloat32(float32(math.Trunc(float64(vm.popFloat32()))))
}

func (vm *VM) f32Nearest() {
	f := vm.popFloat32()
	vm.pushFloat32(float32(int32(f + float32(math.Copysign(0.5, float64(f))))))
}

func (vm *VM) f32Sqrt() {
	vm.pushFloat32(float32(math.Sqrt(float64(vm.popFloat32()))))
}

func (vm *VM) f32Add() {
	vm.pushFloat32(vm.popFloat32() + vm.popFloat32())
}

func (vm *VM) f32Sub() {
	v2 := vm.popFloat32()
	v1 := vm.popFloat32()
	vm.pushFloat32(v1 - v2)
}

func (vm *VM) f32Mul() {
	vm.pushFloat32(vm.popFloat32() * vm.popFloat32())
}

func (vm *VM) f32Div() {
	v2 := vm.popFloat32()
	v1 := vm.popFloat32()
	vm.pushFloat32(v1 / v2)
}

func (vm *VM) f32Min() {
	vm.pushFloat32(float32(math.Min(float64(vm.popFloat32()), float64(vm.popFloat32()))))
}

func (vm *VM) f32Max() {
	vm.pushFloat32(float32(math.Max(float64(vm.popFloat32()), float64(vm.popFloat32()))))
}

func (vm *VM) f32Copysign() {
	vm.pushFloat32(float32(math.Copysign(float64(vm.popFloat32()), float64(vm.popFloat32()))))
}

func (vm *VM) f32Eq() {
	vm.pushBool(vm.popFloat32() == vm.popFloat32())
}

func (vm *VM) f32Ne() {
	vm.pushBool(vm.popFloat32() != vm.popFloat32())
}

func (vm *VM) f32Lt() {
	v2 := vm.popFloat32()
	v1 := vm.popFloat32()
	vm.pushBool(v1 < v2)
}

func (vm *VM) f32Gt() {
	v2 := vm.popFloat32()
	v1 := vm.popFloat32()
	vm.pushBool(v1 > v2)
}

func (vm *VM) f32Le() {
	v2 := vm.popFloat32()
	v1 := vm.popFloat32()
	vm.pushBool(v1 <= v2)
}

func (vm *VM) f32Ge() {
	v2 := vm.popFloat32()
	v1 := vm.popFloat32()
	vm.pushBool(v1 >= v2)
}

// float64 operators

func (vm *VM) f64Abs() {
	vm.pushFloat64(math.Abs(vm.popFloat64()))
}

func (vm *VM) f64Neg() {
	vm.pushFloat64(-vm.popFloat64())
}

func (vm *VM) f64Ceil() {
	vm.pushFloat64(math.Ceil(vm.popFloat64()))
}

func (vm *VM) f64Floor() {
	vm.pushFloat64(math.Floor(vm.popFloat64()))
}

func (vm *VM) f64Trunc() {
	vm.pushFloat64(math.Trunc(vm.popFloat64()))
}

func (vm *VM) f64Nearest() {
	f := vm.popFloat64()
	vm.pushFloat64(float64(int64(f + math.Copysign(0.5, f))))
}

func (vm *VM) f64Sqrt() {
	vm.pushFloat64(math.Sqrt(vm.popFloat64()))
}

func (vm *VM) f64Add() {
	vm.pushFloat64(vm.popFloat64() + vm.popFloat64())
}

func (vm *VM) f64Sub() {
	v2 := vm.popFloat64()
	v1 := vm.popFloat64()
	vm.pushFloat64(v1 - v2)
}

func (vm *VM) f64Mul() {
	vm.pushFloat64(vm.popFloat64() * vm.popFloat64())
}

func (vm *VM) f64Div() {
	v2 := vm.popFloat64()
	v1 := vm.popFloat64()
	vm.pushFloat64(v1 / v2)
}

func (vm *VM) f64Min() {
	vm.pushFloat64(math.Min(vm.popFloat64(), vm.popFloat64()))
}

func (vm *VM) f64Max() {
	vm.pushFloat64(math.Max(vm.popFloat64(), vm.popFloat64()))
}

func (vm *VM) f64Copysign() {
	vm.pushFloat64(math.Copysign(vm.popFloat64(), vm.popFloat64()))
}

func (vm *VM) f64Eq() {
	vm.pushBool(vm.popFloat64() == vm.popFloat64())
}

func (vm *VM) f64Ne() {
	vm.pushBool(vm.popFloat64() != vm.popFloat64())
}

func (vm *VM) f64Lt() {
	v2 := vm.popFloat64()
	v1 := vm.popFloat64()
	vm.pushBool(v1 < v2)
}

func (vm *VM) f64Gt() {
	v2 := vm.popFloat64()
	v1 := vm.popFloat64()
	vm.pushBool(v1 > v2)
}

func (vm *VM) f64Le() {
	v2 := vm.popFloat64()
	v1 := vm.popFloat64()
	vm.pushBool(v1 <= v2)
}

func (vm *VM) f64Ge() {
	v2 := vm.popFloat64()
	v1 := vm.popFloat64()
	vm.pushBool(v1 >= v2)
}
