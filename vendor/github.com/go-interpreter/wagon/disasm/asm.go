// Copyright 2018 The go-interpreter Authors.  All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package disasm

import (
	"bytes"
	"encoding/binary"
	"math"

	"github.com/go-interpreter/wagon/wasm/leb128"
	ops "github.com/go-interpreter/wagon/wasm/operators"
)

// Assemble encodes a set of instructions into binary representation.
func Assemble(instr []Instr) ([]byte, error) {
	body := new(bytes.Buffer)
	for _, ins := range instr {
		body.WriteByte(ins.Op.Code)
		switch op := ins.Op.Code; op {
		case ops.Block, ops.Loop, ops.If:
			leb128.WriteVarint64(body, int64(ins.Block.Signature))
		case ops.Br, ops.BrIf:
			leb128.WriteVarUint32(body, ins.Immediates[0].(uint32))
		case ops.BrTable:
			cnt := ins.Immediates[0].(uint32)
			leb128.WriteVarUint32(body, cnt)
			for i := uint32(0); i < cnt; i++ {
				leb128.WriteVarUint32(body, ins.Immediates[i+1].(uint32))
			}
			leb128.WriteVarUint32(body, ins.Immediates[1+cnt].(uint32))
		case ops.Call, ops.CallIndirect:
			leb128.WriteVarUint32(body, ins.Immediates[0].(uint32))
			if op == ops.CallIndirect {
				leb128.WriteVarUint32(body, ins.Immediates[1].(uint32))
			}
		case ops.GetLocal, ops.SetLocal, ops.TeeLocal, ops.GetGlobal, ops.SetGlobal:
			leb128.WriteVarUint32(body, ins.Immediates[0].(uint32))
		case ops.I32Const:
			leb128.WriteVarint64(body, int64(ins.Immediates[0].(int32)))
		case ops.I64Const:
			leb128.WriteVarint64(body, ins.Immediates[0].(int64))
		case ops.F32Const:
			f := ins.Immediates[0].(float32)
			var b [4]byte
			binary.LittleEndian.PutUint32(b[:], math.Float32bits(f))
			body.Write(b[:])
		case ops.F64Const:
			f := ins.Immediates[0].(float64)
			var b [8]byte
			binary.LittleEndian.PutUint64(b[:], math.Float64bits(f))
			body.Write(b[:])
		case ops.I32Load, ops.I64Load, ops.F32Load, ops.F64Load, ops.I32Load8s, ops.I32Load8u, ops.I32Load16s, ops.I32Load16u, ops.I64Load8s, ops.I64Load8u, ops.I64Load16s, ops.I64Load16u, ops.I64Load32s, ops.I64Load32u, ops.I32Store, ops.I64Store, ops.F32Store, ops.F64Store, ops.I32Store8, ops.I32Store16, ops.I64Store8, ops.I64Store16, ops.I64Store32:
			leb128.WriteVarUint32(body, ins.Immediates[0].(uint32))
			leb128.WriteVarUint32(body, ins.Immediates[1].(uint32))
		case ops.CurrentMemory, ops.GrowMemory:
			leb128.WriteVarUint32(body, uint32(ins.Immediates[0].(uint8)))
		}
	}
	return body.Bytes(), nil
}
