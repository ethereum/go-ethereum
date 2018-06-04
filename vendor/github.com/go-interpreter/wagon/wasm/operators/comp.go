// Copyright 2017 The go-interpreter Authors.  All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package operators

import (
	"github.com/go-interpreter/wagon/wasm"
)

var (
	I32Eqz = newOp(0x45, "i32.eqz", []wasm.ValueType{wasm.ValueTypeI32}, wasm.ValueTypeI32)
	I32Eq  = newOp(0x46, "i32.eq", []wasm.ValueType{wasm.ValueTypeI32, wasm.ValueTypeI32}, wasm.ValueTypeI32)
	I32Ne  = newOp(0x47, "i32.ne", []wasm.ValueType{wasm.ValueTypeI32, wasm.ValueTypeI32}, wasm.ValueTypeI32)
	I32LtS = newOp(0x48, "i32.lt_s", []wasm.ValueType{wasm.ValueTypeI32, wasm.ValueTypeI32}, wasm.ValueTypeI32)
	I32LtU = newOp(0x49, "i32.lt_u", []wasm.ValueType{wasm.ValueTypeI32, wasm.ValueTypeI32}, wasm.ValueTypeI32)
	I32GtS = newOp(0x4a, "i32.gt_s", []wasm.ValueType{wasm.ValueTypeI32, wasm.ValueTypeI32}, wasm.ValueTypeI32)
	I32GtU = newOp(0x4b, "i32.gt_u", []wasm.ValueType{wasm.ValueTypeI32, wasm.ValueTypeI32}, wasm.ValueTypeI32)
	I32LeS = newOp(0x4c, "i32.le_s", []wasm.ValueType{wasm.ValueTypeI32, wasm.ValueTypeI32}, wasm.ValueTypeI32)
	I32LeU = newOp(0x4d, "i32.le_u", []wasm.ValueType{wasm.ValueTypeI32, wasm.ValueTypeI32}, wasm.ValueTypeI32)
	I32GeS = newOp(0x4e, "i32.ge_s", []wasm.ValueType{wasm.ValueTypeI32, wasm.ValueTypeI32}, wasm.ValueTypeI32)
	I32GeU = newOp(0x4f, "i32.ge_u", []wasm.ValueType{wasm.ValueTypeI32, wasm.ValueTypeI32}, wasm.ValueTypeI32)
	I64Eqz = newOp(0x50, "i64.eqz", []wasm.ValueType{wasm.ValueTypeI64}, wasm.ValueTypeI32)
	I64Eq  = newOp(0x51, "i64.eq", []wasm.ValueType{wasm.ValueTypeI64, wasm.ValueTypeI64}, wasm.ValueTypeI32)
	I64Ne  = newOp(0x52, "i64.ne", []wasm.ValueType{wasm.ValueTypeI64, wasm.ValueTypeI64}, wasm.ValueTypeI32)
	I64LtS = newOp(0x53, "i64.lt_s", []wasm.ValueType{wasm.ValueTypeI64, wasm.ValueTypeI64}, wasm.ValueTypeI32)
	I64LtU = newOp(0x54, "i64.lt_u", []wasm.ValueType{wasm.ValueTypeI64, wasm.ValueTypeI64}, wasm.ValueTypeI32)
	I64GtS = newOp(0x55, "i64.gt_s", []wasm.ValueType{wasm.ValueTypeI64, wasm.ValueTypeI64}, wasm.ValueTypeI32)
	I64GtU = newOp(0x56, "i64.gt_u", []wasm.ValueType{wasm.ValueTypeI64, wasm.ValueTypeI64}, wasm.ValueTypeI32)
	I64LeS = newOp(0x57, "i64.le_s", []wasm.ValueType{wasm.ValueTypeI64, wasm.ValueTypeI64}, wasm.ValueTypeI32)
	I64LeU = newOp(0x58, "i64.le_u", []wasm.ValueType{wasm.ValueTypeI64, wasm.ValueTypeI64}, wasm.ValueTypeI32)
	I64GeS = newOp(0x59, "i64.ge_s", []wasm.ValueType{wasm.ValueTypeI64, wasm.ValueTypeI64}, wasm.ValueTypeI32)
	I64GeU = newOp(0x5a, "i64.ge_u", []wasm.ValueType{wasm.ValueTypeI64, wasm.ValueTypeI64}, wasm.ValueTypeI32)
	F32Eq  = newOp(0x5b, "f32.eq", []wasm.ValueType{wasm.ValueTypeF32, wasm.ValueTypeF32}, wasm.ValueTypeI32)
	F32Ne  = newOp(0x5c, "f32.ne", []wasm.ValueType{wasm.ValueTypeF32, wasm.ValueTypeF32}, wasm.ValueTypeI32)
	F32Lt  = newOp(0x5d, "f32.lt", []wasm.ValueType{wasm.ValueTypeF32, wasm.ValueTypeF32}, wasm.ValueTypeI32)
	F32Gt  = newOp(0x5e, "f32.gt", []wasm.ValueType{wasm.ValueTypeF32, wasm.ValueTypeF32}, wasm.ValueTypeI32)
	F32Le  = newOp(0x5f, "f32.le", []wasm.ValueType{wasm.ValueTypeF32, wasm.ValueTypeF32}, wasm.ValueTypeI32)
	F32Ge  = newOp(0x60, "f32.ge", []wasm.ValueType{wasm.ValueTypeF32, wasm.ValueTypeF32}, wasm.ValueTypeI32)
	F64Eq  = newOp(0x61, "f64.eq", []wasm.ValueType{wasm.ValueTypeF64, wasm.ValueTypeF64}, wasm.ValueTypeI32)
	F64Ne  = newOp(0x62, "f64.ne", []wasm.ValueType{wasm.ValueTypeF64, wasm.ValueTypeF64}, wasm.ValueTypeI32)
	F64Lt  = newOp(0x63, "f64.lt", []wasm.ValueType{wasm.ValueTypeF64, wasm.ValueTypeF64}, wasm.ValueTypeI32)
	F64Gt  = newOp(0x64, "f64.gt", []wasm.ValueType{wasm.ValueTypeF64, wasm.ValueTypeF64}, wasm.ValueTypeI32)
	F64Le  = newOp(0x65, "f64.le", []wasm.ValueType{wasm.ValueTypeF64, wasm.ValueTypeF64}, wasm.ValueTypeI32)
	F64Ge  = newOp(0x66, "f64.ge", []wasm.ValueType{wasm.ValueTypeF64, wasm.ValueTypeF64}, wasm.ValueTypeI32)
)
