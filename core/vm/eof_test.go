// Copyright 2021 The go-ethereum Authors
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
	"testing"

	"github.com/ethereum/go-ethereum/common"
)

type eof1Test struct {
	code     string
	codeSize uint16
	dataSize uint16
}

var eof1ValidTests = []eof1Test{
	{"EF00010100010000", 1, 0},
	{"EF0001010002000000", 2, 0},
	{"EF0001010002020001000000AA", 2, 1},
	{"EF0001010002020004000000AABBCCDD", 2, 4},
	{"EF0001010005020002006000600100AABB", 5, 2},
	{"EF00010100070200040060006001600200AABBCCDD", 7, 4},
	{"EF000101000100FE", 1, 0},         // INVALID is defined and can be terminating
	{"EF00010100050060006000F3", 5, 0}, // terminating with RETURN
	{"EF00010100050060006000FD", 5, 0}, // terminating with REVERT
	{"EF0001010003006000FF", 3, 0},     // terminating with SELFDESTRUCT
	{"EF0001010022007F000000000000000000000000000000000000000000000000000000000000000000", 34, 0},     // PUSH32
	{"EF0001010022007F0C0D0E0F1E1F2122232425262728292A2B2C2D2E2F494A4B4C4D4E4F5C5D5E5F00", 34, 0},     // undefined instructions inside push data
	{"EF000101000102002000000C0D0E0F1E1F2122232425262728292A2B2C2D2E2F494A4B4C4D4E4F5C5D5E5F", 1, 32}, // undefined instructions inside data section
}

type eof1InvalidTest struct {
	code  string
	error string
}

// Codes starting with something else other than magic
var notEOFTests = []string{
	// valid: "EF0001010002020004006000AABBCCDD",
	"",
	"FE",                               // invalid first byte
	"FE0001010002020004006000AABBCCDD", // valid except first byte of magic
	"EF",                               // incomplete magic
	"EF01",                             // not correct magic
	"EF0101010002020004006000AABBCCDD", // valid except second byte of magic
}

// Codes starting with magic, but the rest is invalid
var eof1InvalidTests = []eof1InvalidTest{
	// valid: {"EF0001010002020004006000AABBCCDD", nil},
	{"EF00", ErrEOF1InvalidVersion.Error()},                                                 // no version
	{"EF0000", ErrEOF1InvalidVersion.Error()},                                               // invalid version
	{"EF0002", ErrEOF1InvalidVersion.Error()},                                               // invalid version
	{"EF0000010002020004006000AABBCCDD", ErrEOF1InvalidVersion.Error()},                     // valid except version
	{"EF0001", ErrEOF1CodeSectionMissing.Error()},                                           // no header
	{"EF000100", ErrEOF1CodeSectionMissing.Error()},                                         // no code section
	{"EF000101", ErrEOF1CodeSectionSizeMissing.Error()},                                     // no code section size
	{"EF00010100", ErrEOF1CodeSectionSizeMissing.Error()},                                   // code section size incomplete
	{"EF0001010002", ErrEOF1InvalidTotalSize.Error()},                                       // no section terminator
	{"EF000101000200", ErrEOF1InvalidTotalSize.Error()},                                     // no code section contents
	{"EF00010100020060", ErrEOF1InvalidTotalSize.Error()},                                   // not complete code section contents
	{"EF0001010002006000DEADBEEF", ErrEOF1InvalidTotalSize.Error()},                         // trailing bytes after code
	{"EF00010100020100020060006000", ErrEOF1MultipleCodeSections.Error()},                   // two code sections
	{"EF000101000000", ErrEOF1EmptyCodeSection.Error()},                                     // 0 size code section
	{"EF000101000002000200AABB", ErrEOF1EmptyCodeSection.Error()},                           // 0 size code section, with non-0 data section
	{"EF000102000401000200AABBCCDD6000", ErrEOF1DataSectionBeforeCodeSection.Error()},       // data section before code section
	{"EF0001020004AABBCCDD", ErrEOF1DataSectionBeforeCodeSection.Error()},                   // data section without code section
	{"EF000101000202", ErrEOF1DataSectionSizeMissing.Error()},                               // no data section size
	{"EF00010100020200", ErrEOF1DataSectionSizeMissing.Error()},                             // data section size incomplete
	{"EF0001010002020004", ErrEOF1InvalidTotalSize.Error()},                                 // no section terminator
	{"EF0001010002020004006000", ErrEOF1InvalidTotalSize.Error()},                           // no data section contents
	{"EF0001010002020004006000AABBCC", ErrEOF1InvalidTotalSize.Error()},                     // not complete data section contents
	{"EF0001010002020004006000AABBCCDDEE", ErrEOF1InvalidTotalSize.Error()},                 // trailing bytes after data
	{"EF0001010002020000006000", ErrEOF1EmptyDataSection.Error()},                           // 0 size data section
	{"EF0001010002020004020004006000AABBCCDDAABBCCDD", ErrEOF1MultipleDataSections.Error()}, // two data sections
	{"EF0001010002030004006000AABBCCDD", ErrEOF1UnknownSection.Error()},                     // section id = 3
}

var eof1InvalidInstructionsTests = []eof1InvalidTest{
	// 0C is undefined instruction
	{"EF0001010001000C", ErrEOF1UndefinedInstruction.Error()},
	// EF is undefined instruction
	{"EF000101000100EF", ErrEOF1UndefinedInstruction.Error()},
	// ADDRESS is not a terminating instruction
	{"EF00010100010030", ErrEOF1TerminatingInstructionMissing.Error()},
	// PUSH1 without data
	{"EF00010100010060", ErrEOF1TerminatingInstructionMissing.Error()},
	// PUSH32 with 31 bytes of data
	{"EF0001010020007F00000000000000000000000000000000000000000000000000000000000000", ErrEOF1TerminatingInstructionMissing.Error()},
	// PUSH32 with 32 bytes of data and no terminating instruction
	{"EF0001010021007F0000000000000000000000000000000000000000000000000000000000000000", ErrEOF1TerminatingInstructionMissing.Error()},
}

func TestHasEOFMagic(t *testing.T) {
	for _, test := range notEOFTests {
		if hasEOFMagic(common.Hex2Bytes(test)) {
			t.Errorf("code %v expected to be not EOF", test)
		}
	}

	for _, test := range eof1ValidTests {
		if !hasEOFMagic(common.Hex2Bytes(test.code)) {
			t.Errorf("code %v expected to be EOF", test.code)
		}
	}

	// invalid but still EOF
	for _, test := range eof1InvalidTests {
		if !hasEOFMagic(common.Hex2Bytes(test.code)) {
			t.Errorf("code %v expected to be EOF", test.code)
		}
	}
}

func TestReadEOF1Header(t *testing.T) {
	for _, test := range eof1ValidTests {
		header, err := readEOF1Header(common.Hex2Bytes(test.code))
		if err != nil {
			t.Errorf("code %v validation failure, error: %v", test.code, err)
		}
		if header.codeSize != test.codeSize {
			t.Errorf("code %v codeSize expected %v, got %v", test.code, test.codeSize, header.codeSize)
		}
		if header.dataSize != test.dataSize {
			t.Errorf("code %v dataSize expected %v, got %v", test.code, test.dataSize, header.dataSize)
		}
	}

	for _, test := range eof1InvalidTests {
		_, err := readEOF1Header(common.Hex2Bytes(test.code))
		if err == nil {
			t.Errorf("code %v expected to be invalid", test.code)
		} else if err.Error() != test.error {
			t.Errorf("code %v expected error: \"%v\" got error: \"%v\"", test.code, test.error, err.Error())
		}
	}
}

func TestValidateEOF(t *testing.T) {
	jt := &mergeInstructionSet
	for _, test := range eof1ValidTests {
		_, err := validateEOF(common.Hex2Bytes(test.code), jt)
		if err != nil {
			t.Errorf("code %v expected to be valid", test.code)
		}
	}

	for _, test := range eof1InvalidTests {
		_, err := validateEOF(common.Hex2Bytes(test.code), jt)
		if err == nil {
			t.Errorf("code %v expected to be invalid", test.code)
		}
	}
}

func TestReadValidEOF1Header(t *testing.T) {
	for _, test := range eof1ValidTests {
		header := readValidEOF1Header(common.Hex2Bytes(test.code))
		if header.codeSize != test.codeSize {
			t.Errorf("code %v codeSize expected %v, got %v", test.code, test.codeSize, header.codeSize)
		}
		if header.dataSize != test.dataSize {
			t.Errorf("code %v dataSize expected %v, got %v", test.code, test.dataSize, header.dataSize)
		}
	}
}

func TestValidateInstructions(t *testing.T) {
	jt := &londonInstructionSet
	for _, test := range eof1ValidTests {
		code := common.Hex2Bytes(test.code)
		header, err := readEOF1Header(code)
		if err != nil {
			t.Errorf("code %v header validation failure, error: %v", test.code, err)
		}

		err = validateInstructions(code, &header, jt)
		if err != nil {
			t.Errorf("code %v instruction validation failure, error: %v", test.code, err)
		}
	}

	for _, test := range eof1InvalidInstructionsTests {
		code := common.Hex2Bytes(test.code)
		header, err := readEOF1Header(code)
		if err != nil {
			t.Errorf("code %v header validation failure, error: %v", test.code, err)
		}

		err = validateInstructions(code, &header, jt)
		if err == nil {
			t.Errorf("code %v expected to be invalid", test.code)
		} else if err.Error() != test.error {
			t.Errorf("code %v expected error: \"%v\" got error: \"%v\"", test.code, test.error, err.Error())
		}
	}
}

func TestValidateUndefinedInstructions(t *testing.T) {
	jt := &londonInstructionSet
	code := common.Hex2Bytes("EF0001010002000C00")
	instrByte := &code[7]
	for opcode := uint16(0); opcode <= 0xff; opcode++ {
		if OpCode(opcode) >= PUSH1 && OpCode(opcode) <= PUSH32 {
			continue
		}

		*instrByte = byte(opcode)
		header, err := readEOF1Header(code)
		if err != nil {
			t.Errorf("code %v header validation failure, error: %v", common.Bytes2Hex(code), err)
		}

		err = validateInstructions(code, &header, jt)
		if jt[opcode].undefined {
			if err == nil {
				t.Errorf("opcode %v expected to be invalid", opcode)
			} else if err != ErrEOF1UndefinedInstruction {
				t.Errorf("opcode %v unxpected error: \"%v\"", opcode, err.Error())
			}
		} else {
			if err != nil {
				t.Errorf("code %v instruction validation failure, error: %v", common.Bytes2Hex(code), err)
			}
		}
	}
}

func TestValidateTerminatingInstructions(t *testing.T) {
	jt := &londonInstructionSet
	code := common.Hex2Bytes("EF0001010001000C")
	instrByte := &code[7]
	for opcodeValue := uint16(0); opcodeValue <= 0xff; opcodeValue++ {
		opcode := OpCode(opcodeValue)
		if opcode >= PUSH1 && opcode <= PUSH32 {
			continue
		}
		if jt[opcode].undefined {
			continue
		}
		*instrByte = byte(opcode)
		header, err := readEOF1Header(code)
		if err != nil {
			t.Errorf("code %v header validation failure, error: %v", common.Bytes2Hex(code), err)
		}
		err = validateInstructions(code, &header, jt)

		if opcode == STOP || opcode == RETURN || opcode == REVERT || opcode == INVALID || opcode == SELFDESTRUCT {
			if err != nil {
				t.Errorf("opcode %v expected to be valid terminating instruction", opcode)
			}
		} else {
			if err == nil {
				t.Errorf("opcode %v expected to be invalid terminating instruction", opcode)
			} else if err != ErrEOF1TerminatingInstructionMissing {
				t.Errorf("opcode %v unexpected error: \"%v\"", opcode, err.Error())
			}
		}
	}
}

func TestValidateTruncatedPush(t *testing.T) {
	jt := &londonInstructionSet
	zeroes := [33]byte{}
	code := common.Hex2Bytes("EF0001010001000C")
	for opcode := PUSH1; opcode <= PUSH32; opcode++ {
		requiredBytes := opcode - PUSH1 + 1

		// make code with truncated PUSH data
		codeTruncatedPush := append(code, zeroes[:requiredBytes-1]...)
		codeTruncatedPush[5] = byte(len(codeTruncatedPush) - 7)
		codeTruncatedPush[7] = byte(opcode)

		header, err := readEOF1Header(codeTruncatedPush)
		if err != nil {
			t.Errorf("code %v header validation failure, error: %v", common.Bytes2Hex(code), err)
		}
		err = validateInstructions(codeTruncatedPush, &header, jt)
		if err == nil {
			t.Errorf("code %v has truncated PUSH, expected to be invalid", common.Bytes2Hex(codeTruncatedPush))
		} else if err != ErrEOF1TerminatingInstructionMissing {
			t.Errorf("code %v unexpected validation error: %v", common.Bytes2Hex(codeTruncatedPush), err)
		}

		// make code with full PUSH data but no terminating instruction in the end
		codeNotTerminated := append(code, zeroes[:requiredBytes]...)
		codeNotTerminated[5] = byte(len(codeNotTerminated) - 7)
		codeNotTerminated[7] = byte(opcode)

		header, err = readEOF1Header(codeNotTerminated)
		if err != nil {
			t.Errorf("code %v header validation failure, error: %v", common.Bytes2Hex(codeNotTerminated), err)
		}
		err = validateInstructions(codeTruncatedPush, &header, jt)
		if err == nil {
			t.Errorf("code %v does not have terminating instruction, expected to be invalid", common.Bytes2Hex(codeNotTerminated))
		} else if err != ErrEOF1TerminatingInstructionMissing {
			t.Errorf("code %v unexpected validation error: %v", common.Bytes2Hex(codeNotTerminated), err)
		}

		// make valid code
		codeValid := append(code, zeroes[:requiredBytes+1]...) // + 1 for terminating STOP
		codeValid[5] = byte(len(codeValid) - 7)
		codeValid[7] = byte(opcode)

		header, err = readEOF1Header(codeValid)
		if err != nil {
			t.Errorf("code %v header validation failure, error: %v", common.Bytes2Hex(code), err)
		}
		err = validateInstructions(codeValid, &header, jt)
		if err != nil {
			t.Errorf("code %v instruction validation failure, error: %v", common.Bytes2Hex(code), err)
		}
	}
}
