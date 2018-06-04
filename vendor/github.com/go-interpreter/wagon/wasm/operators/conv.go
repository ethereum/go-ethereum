// Copyright 2017 The go-interpreter Authors.  All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package operators

import (
	"regexp"

	"github.com/go-interpreter/wagon/wasm"
)

var reCvrtOp = regexp.MustCompile(`(.+)\.(?:[a-z]|\_)+\/(.+)`)

func valType(s string) wasm.ValueType {
	switch s {
	case "i32":
		return wasm.ValueTypeI32
	case "i64":
		return wasm.ValueTypeI64
	case "f32":
		return wasm.ValueTypeF32
	case "f64":
		return wasm.ValueTypeF64
	default:
		panic("Invalid value type string: " + s)
	}
}

func newConversionOp(code byte, name string) byte {
	matches := reCvrtOp.FindStringSubmatch(name)
	if len(matches) == 0 {
		panic(name + " is not a conversion operator")
	}

	returns := valType(matches[1])
	param := valType(matches[2])

	return newOp(code, name, []wasm.ValueType{param}, returns)
}

var (
	I32WrapI64     = newConversionOp(0xa7, "i32.wrap/i64")
	I32TruncSF32   = newConversionOp(0xa8, "i32.trunc_s/f32")
	I32TruncUF32   = newConversionOp(0xa9, "i32.trunc_u/f32")
	I32TruncSF64   = newConversionOp(0xaa, "i32.trunc_s/f64")
	I32TruncUF64   = newConversionOp(0xab, "i32.trunc_u/f64")
	I64ExtendSI32  = newConversionOp(0xac, "i64.extend_s/i32")
	I64ExtendUI32  = newConversionOp(0xad, "i64.extend_u/i32")
	I64TruncSF32   = newConversionOp(0xae, "i64.trunc_s/f32")
	I64TruncUF32   = newConversionOp(0xaf, "i64.trunc_u/f32")
	I64TruncSF64   = newConversionOp(0xb0, "i64.trunc_s/f64")
	I64TruncUF64   = newConversionOp(0xb1, "i64.trunc_u/f64")
	F32ConvertSI32 = newConversionOp(0xb2, "f32.convert_s/i32")
	F32ConvertUI32 = newConversionOp(0xb3, "f32.convert_u/i32")
	F32ConvertSI64 = newConversionOp(0xb4, "f32.convert_s/i64")
	F32ConvertUI64 = newConversionOp(0xb5, "f32.convert_u/i64")
	F32DemoteF64   = newConversionOp(0xb6, "f32.demote/f64")
	F64ConvertSI32 = newConversionOp(0xb7, "f64.convert_s/i32")
	F64ConvertUI32 = newConversionOp(0xb8, "f64.convert_u/i32")
	F64ConvertSI64 = newConversionOp(0xb9, "f64.convert_s/i64")
	F64ConvertUI64 = newConversionOp(0xba, "f64.convert_u/i64")
	F64PromoteF32  = newConversionOp(0xbb, "f64.promote/f32")
)
