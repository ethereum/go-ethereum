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

package vm

import (
	"encoding/binary"
	"errors"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/params"
)

func TestValidateCode(t *testing.T) {
	for i, test := range []struct {
		code     []byte
		section  int
		metadata []*functionMetadata
		err      error
	}{
		{
			code: []byte{
				byte(CALLER),
				byte(POP),
				byte(STOP),
			},
			section:  0,
			metadata: []*functionMetadata{{inputs: 0, outputs: 0x80, maxStackHeight: 1}},
		},
		{
			code: []byte{
				byte(CALLF), 0x00, 0x00,
				byte(RETF),
			},
			section:  0,
			metadata: []*functionMetadata{{inputs: 0, outputs: 0, maxStackHeight: 0}},
		},
		{
			code: []byte{
				byte(ADDRESS),
				byte(CALLF), 0x00, 0x00,
				byte(POP),
				byte(RETF),
			},
			section:  0,
			metadata: []*functionMetadata{{inputs: 0, outputs: 0, maxStackHeight: 1}},
		},
		{
			code: []byte{
				byte(CALLER),
				byte(POP),
			},
			section:  0,
			metadata: []*functionMetadata{{inputs: 0, outputs: 0x80, maxStackHeight: 1}},
			err:      errInvalidCodeTermination,
		},
		{
			code: []byte{
				byte(RJUMP),
				byte(0x00),
				byte(0x01),
				byte(CALLER),
				byte(STOP),
			},
			section:  0,
			metadata: []*functionMetadata{{inputs: 0, outputs: 0x80, maxStackHeight: 0}},
			err:      errUnreachableCode,
		},
		{
			code: []byte{
				byte(PUSH1),
				byte(0x42),
				byte(ADD),
				byte(STOP),
			},
			section:  0,
			metadata: []*functionMetadata{{inputs: 0, outputs: 0x80, maxStackHeight: 1}},
			err:      ErrStackUnderflow{stackLen: 1, required: 2},
		},
		{
			code: []byte{
				byte(PUSH1),
				byte(0x42),
				byte(POP),
				byte(STOP),
			},
			section:  0,
			metadata: []*functionMetadata{{inputs: 0, outputs: 0x80, maxStackHeight: 2}},
			err:      errInvalidMaxStackHeight,
		},
		{
			code: []byte{
				byte(PUSH0),
				byte(RJUMPI),
				byte(0x00),
				byte(0x01),
				byte(PUSH1),
				byte(0x42), // jumps to here
				byte(POP),
				byte(STOP),
			},
			section:  0,
			metadata: []*functionMetadata{{inputs: 0, outputs: 0x80, maxStackHeight: 1}},
			err:      errInvalidJumpDest,
		},
		{
			code: []byte{
				byte(PUSH0),
				byte(RJUMPV),
				byte(0x01),
				byte(0x00),
				byte(0x01),
				byte(0x00),
				byte(0x02),
				byte(PUSH1),
				byte(0x42), // jumps to here
				byte(POP),  // and here
				byte(STOP),
			},
			section:  0,
			metadata: []*functionMetadata{{inputs: 0, outputs: 0x80, maxStackHeight: 1}},
			err:      errInvalidJumpDest,
		},
		{
			code: []byte{
				byte(PUSH0),
				byte(RJUMPV),
				byte(0x00),
				byte(STOP),
			},
			section:  0,
			metadata: []*functionMetadata{{inputs: 0, outputs: 0x80, maxStackHeight: 1}},
			err:      errTruncatedImmediate,
		},
		{
			code: []byte{
				byte(RJUMP), 0x00, 0x03,
				byte(JUMPDEST), // this code is unreachable to forward jumps alone
				byte(JUMPDEST),
				byte(RETURN),
				byte(PUSH1), 20,
				byte(PUSH1), 39,
				byte(PUSH1), 0x00,
				byte(DATACOPY),
				byte(PUSH1), 20,
				byte(PUSH1), 0x00,
				byte(RJUMP), 0xff, 0xef,
			},
			section:  0,
			metadata: []*functionMetadata{{inputs: 0, outputs: 0x80, maxStackHeight: 3}},
			err:      errUnreachableCode,
		},
		{
			code: []byte{
				byte(PUSH1), 1,
				byte(RJUMPI), 0x00, 0x03,
				byte(JUMPDEST),
				byte(JUMPDEST),
				byte(STOP),
				byte(PUSH1), 20,
				byte(PUSH1), 39,
				byte(PUSH1), 0x00,
				byte(DATACOPY),
				byte(PUSH1), 20,
				byte(PUSH1), 0x00,
				byte(RETURN),
			},
			section:  0,
			metadata: []*functionMetadata{{inputs: 0, outputs: 0x80, maxStackHeight: 3}},
		},
		{
			code: []byte{
				byte(PUSH1), 1,
				byte(RJUMPV), 0x01, 0x00, 0x03, 0xff, 0xf8,
				byte(JUMPDEST),
				byte(JUMPDEST),
				byte(STOP),
				byte(PUSH1), 20,
				byte(PUSH1), 39,
				byte(PUSH1), 0x00,
				byte(DATACOPY),
				byte(PUSH1), 20,
				byte(PUSH1), 0x00,
				byte(RETURN),
			},
			section:  0,
			metadata: []*functionMetadata{{inputs: 0, outputs: 0x80, maxStackHeight: 3}},
		},
		{
			code: []byte{
				byte(STOP),
				byte(STOP),
				byte(INVALID),
			},
			section:  0,
			metadata: []*functionMetadata{{inputs: 0, outputs: 0x80, maxStackHeight: 0}},
			err:      errUnreachableCode,
		},
		{
			code: []byte{
				byte(RETF),
			},
			section:  0,
			metadata: []*functionMetadata{{inputs: 0, outputs: 1, maxStackHeight: 0}},
			err:      errInvalidOutputs,
		},
		{
			code: []byte{
				byte(RETF),
			},
			section:  0,
			metadata: []*functionMetadata{{inputs: 3, outputs: 3, maxStackHeight: 3}},
		},
		{
			code: []byte{
				byte(CALLF), 0x00, 0x01,
				byte(POP),
				byte(STOP),
			},
			section:  0,
			metadata: []*functionMetadata{{inputs: 0, outputs: 0x80, maxStackHeight: 1}, {inputs: 0, outputs: 1, maxStackHeight: 0}},
		},
		{
			code: []byte{
				byte(ORIGIN),
				byte(ORIGIN),
				byte(CALLF), 0x00, 0x01,
				byte(POP),
				byte(RETF),
			},
			section:  0,
			metadata: []*functionMetadata{{inputs: 0, outputs: 0, maxStackHeight: 2}, {inputs: 2, outputs: 1, maxStackHeight: 2}},
		},
	} {
		container := &Container{
			types:         test.metadata,
			data:          make([]byte, 0),
			subContainers: make([]*Container, 0),
		}
		_, err := validateCode(test.code, test.section, container, &pragueEOFInstructionSet, false)
		if !errors.Is(err, test.err) {
			t.Errorf("test %d (%s): unexpected error (want: %v, got: %v)", i, common.Bytes2Hex(test.code), test.err, err)
		}
	}
}

// BenchmarkRJUMPI tries to benchmark the RJUMPI opcode validation
// For this we do a bunch of RJUMPIs that jump backwards (in a potential infinite loop).
func BenchmarkRJUMPI(b *testing.B) {
	snippet := []byte{
		byte(PUSH0),
		byte(RJUMPI), 0xFF, 0xFC,
	}
	code := []byte{}
	for i := 0; i < params.MaxCodeSize/len(snippet)-1; i++ {
		code = append(code, snippet...)
	}
	code = append(code, byte(STOP))
	container := &Container{
		types:         []*functionMetadata{{inputs: 0, outputs: 0x80, maxStackHeight: 1}},
		data:          make([]byte, 0),
		subContainers: make([]*Container, 0),
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := validateCode(code, 0, container, &pragueEOFInstructionSet, false)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkRJUMPV tries to benchmark the validation of the RJUMPV opcode
// for this we set up as many RJUMPV opcodes with a full jumptable (containing 0s) as possible.
func BenchmarkRJUMPV(b *testing.B) {
	snippet := []byte{
		byte(PUSH0),
		byte(RJUMPV),
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
	code = append(code, byte(PUSH0))
	code = append(code, byte(STOP))
	container := &Container{
		types:         []*functionMetadata{{inputs: 0, outputs: 0x80, maxStackHeight: 1}},
		data:          make([]byte, 0),
		subContainers: make([]*Container, 0),
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := validateCode(code, 0, container, &pragueEOFInstructionSet, false)
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
	var container Container
	var code []byte
	maxSections := 1024
	for i := 0; i < maxSections; i++ {
		code = append(code, byte(CALLF))
		code = binary.BigEndian.AppendUint16(code, uint16(i%(maxSections-1))+1)
	}
	// First container
	container.codeSections = append(container.codeSections, append(code, byte(STOP)))
	container.types = append(container.types, &functionMetadata{inputs: 0, outputs: 0x80, maxStackHeight: 0})

	inner := []byte{
		byte(RETF),
	}

	for i := 0; i < 1023; i++ {
		container.codeSections = append(container.codeSections, inner)
		container.types = append(container.types, &functionMetadata{inputs: 0, outputs: 0, maxStackHeight: 0})
	}

	for i := 0; i < 12; i++ {
		container.codeSections[i+1] = append(code, byte(RETF))
	}

	bin := container.MarshalBinary()
	if len(bin) > 48*1024 {
		b.Fatal("Exceeds 48Kb")
	}

	var container2 Container
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if err := container2.UnmarshalBinary(bin, true); err != nil {
			b.Fatal(err)
		}
		if err := container2.ValidateCode(&pragueEOFInstructionSet, false); err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkEOFValidation tries to benchmark the code validation for the CALLF/RETF operation.
// For this we set up code that calls into 1024 code sections which
// - contain calls to some other code sections.
// We can't have all code sections calling each other, otherwise we would exceed 48KB.
func BenchmarkEOFValidation2(b *testing.B) {
	var container Container
	var code []byte
	maxSections := 1024
	for i := 0; i < maxSections; i++ {
		code = append(code, byte(CALLF))
		code = binary.BigEndian.AppendUint16(code, uint16(i%(maxSections-1))+1)
	}
	code = append(code, byte(STOP))
	// First container
	container.codeSections = append(container.codeSections, code)
	container.types = append(container.types, &functionMetadata{inputs: 0, outputs: 0x80, maxStackHeight: 0})

	inner := []byte{
		byte(CALLF), 0x03, 0xE8,
		byte(CALLF), 0x03, 0xE9,
		byte(CALLF), 0x03, 0xF0,
		byte(CALLF), 0x03, 0xF1,
		byte(CALLF), 0x03, 0xF2,
		byte(CALLF), 0x03, 0xF3,
		byte(CALLF), 0x03, 0xF4,
		byte(CALLF), 0x03, 0xF5,
		byte(CALLF), 0x03, 0xF6,
		byte(CALLF), 0x03, 0xF7,
		byte(CALLF), 0x03, 0xF8,
		byte(CALLF), 0x03, 0xF,
		byte(RETF),
	}

	for i := 0; i < 1023; i++ {
		container.codeSections = append(container.codeSections, inner)
		container.types = append(container.types, &functionMetadata{inputs: 0, outputs: 0, maxStackHeight: 0})
	}

	bin := container.MarshalBinary()
	if len(bin) > 48*1024 {
		b.Fatal("Exceeds 48Kb")
	}

	var container2 Container
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if err := container2.UnmarshalBinary(bin, true); err != nil {
			b.Fatal(err)
		}
		if err := container2.ValidateCode(&pragueEOFInstructionSet, false); err != nil {
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
	var container Container
	var code []byte
	snippet := []byte{
		byte(PUSH0),
		byte(RJUMPV),
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
		code = append(code, byte(CALLF))
		code = binary.BigEndian.AppendUint16(code, uint16(i%(maxSections-1))+1)
	}
	code = append(code, byte(STOP))
	container.codeSections = append(container.codeSections, code)
	container.types = append(container.types, &functionMetadata{inputs: 0, outputs: 0x80, maxStackHeight: 1})

	// Other containers
	for i := 0; i < 1023; i++ {
		container.codeSections = append(container.codeSections, []byte{byte(RJUMP), 0x00, 0x00, byte(RETF)})
		container.types = append(container.types, &functionMetadata{inputs: 0, outputs: 0, maxStackHeight: 0})
	}
	// Other containers
	for i := 0; i < 68; i++ {
		container.codeSections[i+1] = append(snippet, byte(RETF))
		container.types[i+1] = &functionMetadata{inputs: 0, outputs: 0, maxStackHeight: 1}
	}
	bin := container.MarshalBinary()
	if len(bin) > 48*1024 {
		b.Fatal("Exceeds 48Kb")
	}
	b.ResetTimer()
	b.ReportMetric(float64(len(bin)), "bytes")
	for i := 0; i < b.N; i++ {
		for k := 0; k < 40; k++ {
			var container2 Container
			if err := container2.UnmarshalBinary(bin, true); err != nil {
				b.Fatal(err)
			}
			if err := container2.ValidateCode(&pragueEOFInstructionSet, false); err != nil {
				b.Fatal(err)
			}
		}
	}
}

func BenchmarkRJUMPI_2(b *testing.B) {
	code := []byte{
		byte(PUSH0),
		byte(RJUMPI), 0xFF, 0xFC,
	}
	for i := 0; i < params.MaxCodeSize/4-1; i++ {
		code = append(code, byte(PUSH0))
		x := -4 * i
		code = append(code, byte(RJUMPI))
		code = binary.BigEndian.AppendUint16(code, uint16(x))
	}
	code = append(code, byte(STOP))
	container := &Container{
		types:         []*functionMetadata{{inputs: 0, outputs: 0x80, maxStackHeight: 1}},
		data:          make([]byte, 0),
		subContainers: make([]*Container, 0),
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := validateCode(code, 0, container, &pragueEOFInstructionSet, false)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func FuzzUnmarshalBinary(f *testing.F) {
	f.Fuzz(func(_ *testing.T, input []byte) {
		var container Container
		container.UnmarshalBinary(input, true)
	})
}

func FuzzValidate(f *testing.F) {
	f.Fuzz(func(_ *testing.T, code []byte, maxStack uint16) {
		var container Container
		container.types = append(container.types, &functionMetadata{inputs: 0, outputs: 0x80, maxStackHeight: maxStack})
		validateCode(code, 0, &container, &pragueEOFInstructionSet, true)
	})
}
