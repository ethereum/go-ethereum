// Copyright 2017 The go-interpreter Authors.  All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package wasm

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"math"

	"github.com/go-interpreter/wagon/wasm/leb128"
)

const (
	i32Const  byte = 0x41
	i64Const  byte = 0x42
	f32Const  byte = 0x43
	f64Const  byte = 0x44
	getGlobal byte = 0x23
	end       byte = 0x0b
)

var ErrEmptyInitExpr = errors.New("wasm: Initializer expression produces no value")

type InvalidInitExprOpError byte

func (e InvalidInitExprOpError) Error() string {
	return fmt.Sprintf("wasm: Invalid opcode in initializer expression: %#x", byte(e))
}

type InvalidGlobalIndexError uint32

func (e InvalidGlobalIndexError) Error() string {
	return fmt.Sprintf("wasm: Invalid index to global index space: %#x", uint32(e))
}

func readInitExpr(r io.Reader) ([]byte, error) {
	b := make([]byte, 1)
	buf := new(bytes.Buffer)
	r = io.TeeReader(r, buf)

outer:
	for {
		_, err := io.ReadFull(r, b)
		if err != nil {
			return nil, err
		}
		switch b[0] {
		case i32Const:
			_, err := leb128.ReadVarint32(r)
			if err != nil {
				return nil, err
			}
		case i64Const:
			_, err := leb128.ReadVarint64(r)
			if err != nil {
				return nil, err
			}
		case f32Const:
			if _, err := readU32(r); err != nil {
				return nil, err
			}
		case f64Const:
			if _, err := readU64(r); err != nil {
				return nil, err
			}
		case getGlobal:
			_, err := leb128.ReadVarUint32(r)
			if err != nil {
				return nil, err
			}
		case end:
			break outer
		default:
			return nil, InvalidInitExprOpError(b[0])
		}
	}

	if buf.Len() == 0 {
		return nil, ErrEmptyInitExpr
	}

	return buf.Bytes(), nil
}

// ExecInitExpr executes an initializer expression and returns an interface{} value
// which can either be int32, int64, float32 or float64.
// It returns an error if the expression is invalid, and nil when the expression
// yields no value.
func (m *Module) ExecInitExpr(expr []byte) (interface{}, error) {
	var stack []uint64
	var lastVal ValueType
	r := bytes.NewReader(expr)

	if r.Len() == 0 {
		return nil, ErrEmptyInitExpr
	}

	for {
		b, err := r.ReadByte()
		if err == io.EOF {
			break
		} else if err != nil {
			return nil, err
		}
		switch b {
		case i32Const:
			i, err := leb128.ReadVarint32(r)
			if err != nil {
				return nil, err
			}
			stack = append(stack, uint64(i))
			lastVal = ValueTypeI32
		case i64Const:
			i, err := leb128.ReadVarint64(r)
			if err != nil {
				return nil, err
			}
			stack = append(stack, uint64(i))
			lastVal = ValueTypeI64
		case f32Const:
			i, err := readU32(r)
			if err != nil {
				return nil, err
			}
			stack = append(stack, uint64(i))
			lastVal = ValueTypeF32
		case f64Const:
			i, err := readU64(r)
			if err != nil {
				return nil, err
			}
			stack = append(stack, i)
			lastVal = ValueTypeF64
		case getGlobal:
			index, err := leb128.ReadVarUint32(r)
			if err != nil {
				return nil, err
			}
			globalVar := m.GetGlobal(int(index))
			if globalVar == nil {
				return nil, InvalidGlobalIndexError(index)
			}
			lastVal = globalVar.Type.Type
		case end:
			break
		default:
			return nil, InvalidInitExprOpError(b)
		}
	}

	if len(stack) == 0 {
		return nil, nil
	}

	v := stack[len(stack)-1]
	switch lastVal {
	case ValueTypeI32:
		return int32(v), nil
	case ValueTypeI64:
		return int64(v), nil
	case ValueTypeF32:
		return math.Float32frombits(uint32(v)), nil
	case ValueTypeF64:
		return math.Float64frombits(uint64(v)), nil
	default:
		panic(fmt.Sprintf("Invalid value type produced by initializer expression: %d", int8(lastVal)))
	}
}
