// Copyright 2017 The go-interpreter Authors.  All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package operators

import (
	"github.com/go-interpreter/wagon/wasm"
)

var (
	I32Const = newOp(0x41, "i32.const", nil, wasm.ValueTypeI32)
	I64Const = newOp(0x42, "i64.const", nil, wasm.ValueTypeI64)
	F32Const = newOp(0x43, "f32.const", nil, wasm.ValueTypeF32)
	F64Const = newOp(0x44, "f64.const", nil, wasm.ValueTypeF64)
)
