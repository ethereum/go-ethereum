// Copyright 2024 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

package vm_test

import (
	"encoding/binary"
	"errors"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/params"
)

func TestValidateCode(t *testing.T) {
	for i, test := range []struct {
		code     []byte
		section  int
		metadata []*vm.FunctionMetadata
		err      error
	}{
		{
			code: []byte{
				byte(vm.CALLER),
				byte(vm.POP),
				byte(vm.STOP),
			},
			section:  0,
			metadata: []*vm.FunctionMetadata{{Inputs: 0, Outputs: 0x80, MaxStackHeight: 1}},
		},
		{
			code: []byte{
				byte(vm.CALLF), 0x00, 0x00,
				byte(vm.RETF),
			},
			section:  0,
			metadata: []*vm.FunctionMetadata{{Inputs: 0, Outputs: 0, MaxStackHeight: 0}},
		},
		{
			code: []byte{
				byte(vm.ADDRESS),
				byte(vm.CALLF), 0x00, 0x00,
				byte(vm.POP),
				byte(vm.RETF),
			},
			section:  0,
			metadata: []*vm.FunctionMetadata{{Inputs: 0, Outputs: 0, MaxStackHeight: 1}},
		},
		{
			code: []byte{
				byte(vm.CALLER),
				byte(vm.POP),
			},
			section:  0,
			metadata: []*vm.FunctionMetadata{{Inputs: 0, Outputs: 0x80, MaxStackHeight: 1}},
			err:      vm.ErrInvalidCodeTermination,
		},
		{
			code: []byte{
				byte(vm.RJUMP),
				byte(0x00),
				byte(0x01),
				byte(vm.CALLER),
				byte(vm.STOP),
			},
			section:  0,
			metadata: []*vm.FunctionMetadata{{Inputs: 0, Outputs: 0x80, MaxStackHeight: 0}},
			err:      vm.ErrUnreachableCode,
		},
		{
			code: []byte{
				byte(vm.PUSH1),
				byte(0x42),
				byte(vm.ADD),
				byte(vm.STOP),
			},
			section:  0,
			metadata: []*vm.FunctionMetadata{{Inputs: 0, Outputs: 0x80, MaxStackHeight: 1}},
			err:      vm.ErrStackUnderflow{StackLen: 1, Required: 2},
		},
		{
			code: []byte{
				byte(vm.PUSH1),
				byte(0x42),
				byte(vm.POP),
				byte(vm.STOP),
			},
			section:  0,
			metadata: []*vm.FunctionMetadata{{Inputs: 0, Outputs: 0x80, MaxStackHeight: 2}},
			err:      vm.ErrInvalidMaxStackHeight,
		},
		{
			code: []byte{
				byte(vm.PUSH0),
				byte(vm.RJUMPI),
				byte(0x00),
				byte(0x01),
				byte(vm.PUSH1),
				byte(0x42), // jumps to here
				byte(vm.POP),
				byte(vm.STOP),
			},
			section:  0,
			metadata: []*vm.FunctionMetadata{{Inputs: 0, Outputs: 0x80, MaxStackHeight: 1}},
			err:      vm.ErrInvalidJumpDest,
		},
		{
			code: []byte{
				byte(vm.PUSH0),
				byte(vm.RJUMPV),
				byte(0x01),
				byte(0x00),
				byte(0x01),
				byte(0x00),
				byte(0x02),
				byte(vm.PUSH1),
				byte(0x42),   // jumps to here
				byte(vm.POP), // and here
				byte(vm.STOP),
			},
			section:  0,
			metadata: []*vm.FunctionMetadata{{Inputs: 0, Outputs: 0x80, MaxStackHeight: 1}},
			err:      vm.ErrInvalidJumpDest,
		},
		{
			code: []byte{
				byte(vm.PUSH0),
				byte(vm.RJUMPV),
				byte(0x00),
				byte(vm.STOP),
			},
			section:  0,
			metadata: []*vm.FunctionMetadata{{Inputs: 0, Outputs: 0x80, MaxStackHeight: 1}},
			err:      vm.ErrTruncatedImmediate,
		},
		{
			code: []byte{
				byte(vm.RJUMP), 0x00, 0x03,
				byte(vm.JUMPDEST), // this code is unreachable to forward jumps alone
				byte(vm.JUMPDEST),
				byte(vm.RETURN),
				byte(vm.PUSH1), 20,
				byte(vm.PUSH1), 39,
				byte(vm.PUSH1), 0x00,
				byte(vm.DATACOPY),
				byte(vm.PUSH1), 20,
				byte(vm.PUSH1), 0x00,
				byte(vm.RJUMP), 0xff, 0xef,
			},
			section:  0,
			metadata: []*vm.FunctionMetadata{{Inputs: 0, Outputs: 0x80, MaxStackHeight: 3}},
			err:      vm.ErrUnreachableCode,
		},
		{
			code: []byte{
				byte(vm.PUSH1), 1,
				byte(vm.RJUMPI), 0x00, 0x03,
				byte(vm.JUMPDEST),
				byte(vm.JUMPDEST),
				byte(vm.STOP),
				byte(vm.PUSH1), 20,
				byte(vm.PUSH1), 39,
				byte(vm.PUSH1), 0x00,
				byte(vm.DATACOPY),
				byte(vm.PUSH1), 20,
				byte(vm.PUSH1), 0x00,
				byte(vm.RETURN),
			},
			section:  0,
			metadata: []*vm.FunctionMetadata{{Inputs: 0, Outputs: 0x80, MaxStackHeight: 3}},
		},
		{
			code: []byte{
				byte(vm.PUSH1), 1,
				byte(vm.RJUMPV), 0x01, 0x00, 0x03, 0xff, 0xf8,
				byte(vm.JUMPDEST),
				byte(vm.JUMPDEST),
				byte(vm.STOP),
				byte(vm.PUSH1), 20,
				byte(vm.PUSH1), 39,
				byte(vm.PUSH1), 0x00,
				byte(vm.DATACOPY),
				byte(vm.PUSH1), 20,
				byte(vm.PUSH1), 0x00,
				byte(vm.RETURN),
			},
			section:  0,
			metadata: []*vm.FunctionMetadata{{Inputs: 0, Outputs: 0x80, MaxStackHeight: 3}},
		},
		{
			code: []byte{
				byte(vm.STOP),
				byte(vm.STOP),
				byte(vm.INVALID),
			},
			section:  0,
			metadata: []*vm.FunctionMetadata{{Inputs: 0, Outputs: 0x80, MaxStackHeight: 0}},
			err:      vm.ErrUnreachableCode,
		},
		{
			code: []byte{
				byte(vm.RETF),
			},
			section:  0,
			metadata: []*vm.FunctionMetadata{{Inputs: 0, Outputs: 1, MaxStackHeight: 0}},
			err:      vm.ErrInvalidOutputs,
		},
		{
			code: []byte{
				byte(vm.RETF),
			},
			section:  0,
			metadata: []*vm.FunctionMetadata{{Inputs: 3, Outputs: 3, MaxStackHeight: 3}},
		},
		{
			code: []byte{
				byte(vm.CALLF), 0x00, 0x01,
				byte(vm.POP),
				byte(vm.STOP),
			},
			section:  0,
			metadata: []*vm.FunctionMetadata{{Inputs: 0, Outputs: 0x80, MaxStackHeight: 1}, {Inputs: 0, Outputs: 1, MaxStackHeight: 0}},
		},
		{
			code: []byte{
				byte(vm.ORIGIN),
				byte(vm.ORIGIN),
				byte(vm.CALLF), 0x00, 0x01,
				byte(vm.POP),
				byte(vm.RETF),
			},
			section:  0,
			metadata: []*vm.FunctionMetadata{{Inputs: 0, Outputs: 0, MaxStackHeight: 2}, {Inputs: 2, Outputs: 1, MaxStackHeight: 2}},
		},
	} {
		container := &vm.Container{
			Types:         test.metadata,
			Data:          make([]byte, 0),
			SubContainers: make([]*vm.Container, 0),
		}
		_, err := vm.ValidateCode(test.code, test.section, container, &vm.EofInstructionSet, false)
		if !errors.Is(err, test.err) {
			t.Errorf("test %d (%s): unexpected error (want: %v, got: %v)", i, common.Bytes2Hex(test.code), test.err, err)
		}
	}
}

// BenchmarkRJUMPI tries to benchmark the RJUMPI opcode validation
// For this we do a bunch of RJUMPIs that jump backwards (in a potential infinite loop).
func BenchmarkRJUMPI(b *testing.B) {
	snippet := []byte{
		byte(vm.PUSH0),
		byte(vm.RJUMPI), 0xFF, 0xFC,
	}
	code := []byte{}
	for i := 0; i < params.MaxCodeSize/len(snippet)-1; i++ {
		code = append(code, snippet...)
	}
	code = append(code, byte(vm.STOP))
	container := &vm.Container{
		Types:         []*vm.FunctionMetadata{{Inputs: 0, Outputs: 0x80, MaxStackHeight: 1}},
		Data:          make([]byte, 0),
		SubContainers: make([]*vm.Container, 0),
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := vm.ValidateCode(code, 0, container, &vm.EofInstructionSet, false)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkRJUMPV tries to benchmark the validation of the RJUMPV opcode
// for this we set up as many RJUMPV opcodes with a full jumptable (containing 0s) as possible.
func BenchmarkRJUMPV(b *testing.B) {
	snippet := []byte{
		byte(vm.PUSH0),
		byte(vm.RJUMPV),
		0xff, // count
		0x00, 0x00,
	}
	for i := 0; i < 255; i++ {
		snippet = append(snippet, []byte{0x00, 0x00}...)
	}
	code := []byte{}
	for i := 0; i < 24576/len(snippet)-1; i++ {
		code = append(code, snippet...)
	}
	code = append(code, byte(vm.PUSH0))
	code = append(code, byte(vm.STOP))
	container := &vm.Container{
		Types:         []*vm.FunctionMetadata{{Inputs: 0, Outputs: 0x80, MaxStackHeight: 1}},
		Data:          make([]byte, 0),
		SubContainers: make([]*vm.Container, 0),
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := vm.ValidateCode(code, 0, container, &vm.PragueInstructionSet, false)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkEOFValidation tries to benchmark the code validation for the CALLF/RETF operation.
// For this we set up code that calls into 1024 code sections which can either
// - just contain a RETF opcode
// - or code to again call into 1024 code sections.
// We can't have all code sections calling each other, otherwise we would exceed 48KB.
func BenchmarkEOFValidation(b *testing.B) {
	var container vm.Container
	var code []byte
	maxSections := 1024
	for i := 0; i < maxSections; i++ {
		code = append(code, byte(vm.CALLF))
		code = binary.BigEndian.AppendUint16(code, uint16(i%(maxSections-1))+1)
	}
	// First container
	container.CodeSections = append(container.CodeSections, append(code, byte(vm.STOP)))
	container.Types = append(container.Types, &vm.FunctionMetadata{Inputs: 0, Outputs: 0x80, MaxStackHeight: 0})

	inner := []byte{
		byte(vm.RETF),
	}

	for i := 0; i < 1023; i++ {
		container.CodeSections = append(container.CodeSections, inner)
		container.Types = append(container.Types, &vm.FunctionMetadata{Inputs: 0, Outputs: 0, MaxStackHeight: 0})
	}

	for i := 0; i < 12; i++ {
		container.CodeSections[i+1] = append(code, byte(vm.RETF))
	}

	bin := container.MarshalBinary()
	if len(bin) > 48*1024 {
		b.Fatal("Exceeds 48Kb")
	}

	var container2 vm.Container
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if err := container2.UnmarshalBinary(bin, true); err != nil {
			b.Fatal(err)
		}
		if err := container2.ValidateCode(&vm.PragueInstructionSet, false); err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkEOFValidation2 tries to benchmark the code validation for the CALLF/RETF operation.
// For this we set up code that calls into 1024 code sections which
// - contain calls to some other code sections.
// We can't have all code sections calling each other, otherwise we would exceed 48KB.
func BenchmarkEOFValidation2(b *testing.B) {
	var container vm.Container
	var code []byte
	maxSections := 1024
	for i := 0; i < maxSections; i++ {
		code = append(code, byte(vm.CALLF))
		code = binary.BigEndian.AppendUint16(code, uint16(i%(maxSections-1))+1)
	}
	code = append(code, byte(vm.STOP))
	// First container
	container.CodeSections = append(container.CodeSections, code)
	container.Types = append(container.Types, &vm.FunctionMetadata{Inputs: 0, Outputs: 0x80, MaxStackHeight: 0})

	inner := []byte{
		byte(vm.CALLF), 0x03, 0xE8,
		byte(vm.CALLF), 0x03, 0xE9,
		byte(vm.CALLF), 0x03, 0xF0,
		byte(vm.CALLF), 0x03, 0xF1,
		byte(vm.CALLF), 0x03, 0xF2,
		byte(vm.CALLF), 0x03, 0xF3,
		byte(vm.CALLF), 0x03, 0xF4,
		byte(vm.CALLF), 0x03, 0xF5,
		byte(vm.CALLF), 0x03, 0xF6,
		byte(vm.CALLF), 0x03, 0xF7,
		byte(vm.CALLF), 0x03, 0xF8,
		byte(vm.CALLF), 0x03, 0xF,
		byte(vm.RETF),
	}

	for i := 0; i < 1023; i++ {
		container.CodeSections = append(container.CodeSections, inner)
		container.Types = append(container.Types, &vm.FunctionMetadata{Inputs: 0, Outputs: 0, MaxStackHeight: 0})
	}

	bin := container.MarshalBinary()
	if len(bin) > 48*1024 {
		b.Fatal("Exceeds 48Kb")
	}

	var container2 vm.Container
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if err := container2.UnmarshalBinary(bin, true); err != nil {
			b.Fatal(err)
		}
		if err := container2.ValidateCode(&vm.PragueInstructionSet, false); err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkEOFValidation3 tries to benchmark the code validation for the CALLF/RETF and RJUMPI/V operations.
// For this we set up code that calls into 1024 code sections which either
// - contain an RJUMP opcode
// - contain calls to other code sections
// We can't have all code sections calling each other, otherwise we would exceed 48KB.
func BenchmarkEOFValidation3(b *testing.B) {
	var container vm.Container
	var code []byte
	snippet := []byte{
		byte(vm.PUSH0),
		byte(vm.RJUMPV),
		0xff, // count
		0x00, 0x00,
	}
	for i := 0; i < 255; i++ {
		snippet = append(snippet, []byte{0x00, 0x00}...)
	}
	code = append(code, snippet...)
	// First container, calls into all other containers
	maxSections := 1024
	for i := 0; i < maxSections; i++ {
		code = append(code, byte(vm.CALLF))
		code = binary.BigEndian.AppendUint16(code, uint16(i%(maxSections-1))+1)
	}
	code = append(code, byte(vm.STOP))
	container.CodeSections = append(container.CodeSections, code)
	container.Types = append(container.Types, &vm.FunctionMetadata{Inputs: 0, Outputs: 0x80, MaxStackHeight: 1})

	// Other containers
	for i := 0; i < 1023; i++ {
		container.CodeSections = append(container.CodeSections, []byte{byte(vm.RJUMP), 0x00, 0x00, byte(vm.RETF)})
		container.Types = append(container.Types, &vm.FunctionMetadata{Inputs: 0, Outputs: 0, MaxStackHeight: 0})
	}
	// Other containers
	for i := 0; i < 68; i++ {
		container.CodeSections[i+1] = append(snippet, byte(vm.RETF))
		container.Types[i+1] = &vm.FunctionMetadata{Inputs: 0, Outputs: 0, MaxStackHeight: 1}
	}
	bin := container.MarshalBinary()
	if len(bin) > 48*1024 {
		b.Fatal("Exceeds 48Kb")
	}
	b.ResetTimer()
	b.ReportMetric(float64(len(bin)), "bytes")
	for i := 0; i < b.N; i++ {
		for k := 0; k < 40; k++ {
			var container2 vm.Container
			if err := container2.UnmarshalBinary(bin, true); err != nil {
				b.Fatal(err)
			}
			if err := container2.ValidateCode(&vm.PragueInstructionSet, false); err != nil {
				b.Fatal(err)
			}
		}
	}
}

func BenchmarkRJUMPI_2(b *testing.B) {
	code := []byte{
		byte(vm.PUSH0),
		byte(vm.RJUMPI), 0xFF, 0xFC,
	}
	for i := 0; i < params.MaxCodeSize/4-1; i++ {
		code = append(code, byte(vm.PUSH0))
		x := -4 * i
		code = append(code, byte(vm.RJUMPI))
		code = binary.BigEndian.AppendUint16(code, uint16(x))
	}
	code = append(code, byte(vm.STOP))
	container := &vm.Container{
		Types:         []*vm.FunctionMetadata{{Inputs: 0, Outputs: 0x80, MaxStackHeight: 1}},
		Data:          make([]byte, 0),
		SubContainers: make([]*vm.Container, 0),
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := vm.ValidateCode(code, 0, container, &vm.PragueInstructionSet, false)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func FuzzUnmarshalBinary(f *testing.F) {
	f.Fuzz(func(_ *testing.T, input []byte) {
		var container vm.Container
		container.UnmarshalBinary(input, true)
	})
}

func FuzzValidate(f *testing.F) {
	f.Fuzz(func(_ *testing.T, code []byte, maxStack uint16) {
		var container vm.Container
		container.Types = append(container.Types, &vm.FunctionMetadata{Inputs: 0, Outputs: 0x80, MaxStackHeight: maxStack})
		vm.ValidateCode(code, 0, &container, &vm.PragueInstructionSet, true)
	})
}
