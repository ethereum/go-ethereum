// Copyright 2017 The go-interpreter Authors.  All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package operators

import (
	"github.com/go-interpreter/wagon/wasm"
)

var (
	I32ReinterpretF32 = newOp(0xbc, "i32.reinterpret/f32", []wasm.ValueType{wasm.ValueTypeF32}, wasm.ValueTypeI32)
	I64ReinterpretF64 = newOp(0xbd, "i64.reinterpret/f64", []wasm.ValueType{wasm.ValueTypeF64}, wasm.ValueTypeI64)
	F32ReinterpretI32 = newOp(0xbe, "f32.reinterpret/i32", []wasm.ValueType{wasm.ValueTypeI32}, wasm.ValueTypeF32)
	F64ReinterpretI64 = newOp(0xbf, "f64.reinterpret/i64", []wasm.ValueType{wasm.ValueTypeI64}, wasm.ValueTypeF64)
)
