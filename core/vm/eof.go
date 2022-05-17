// Copyright 2021 The go-ethereum Authors
// This file is part of go-ethereum.
//
// go-ethereum is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// go-ethereum is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with go-ethereum. If not, see <http://www.gnu.org/licenses/>.

package vm

import (
	"bytes"
	"encoding/binary"
	"reflect"
)

const (
	eofFormatByte         byte = 0xEF
	eofMagicLen           int  = 2
	eof1Version           byte = 1
	sectionKindTerminator byte = 0
	sectionKindCode       byte = 1
	sectionKindData       byte = 2
)

type EOF1Header struct {
	codeSize uint16 // Size of code section. Cannot be 0 for EOF1 code. Equals 0 for legacy code.
	dataSize uint16 // Size of data section. Equals 0 if data section is absent in EOF1 code. Equals 0 for legacy code.
}

func getEofMagic() []byte {
	return []byte{0xEF, 0x00}
}

// hasFormatByte returns true if code starts with 0xEF byte
func hasFormatByte(code []byte) bool {
	return len(code) != 0 && code[0] == eofFormatByte
}

// hasEOFMagic returns true if code starts with magic defined by EIP-3540
func hasEOFMagic(code []byte) bool {
	return len(code) >= eofMagicLen && bytes.Equal(getEofMagic(), code[0:eofMagicLen])
}

// readEOF1Header parses EOF1-formatted code header
func readEOF1Header(code []byte) (EOF1Header, error) {
	codeLen := len(code)

	i := eofMagicLen
	if i >= codeLen || code[i] != eof1Version {
		return EOF1Header{}, ErrEOF1InvalidVersion
	}
	i += 1

	var header EOF1Header
sectionLoop:
	for i < codeLen {
		sectionKind := code[i]
		i += 1
		switch sectionKind {
		case sectionKindTerminator:
			break sectionLoop
		case sectionKindCode:
			// Only 1 code section is allowed.
			if header.codeSize != 0 {
				return EOF1Header{}, ErrEOF1MultipleCodeSections
			}
			// Code section size must be present.
			if i+2 > codeLen {
				return EOF1Header{}, ErrEOF1CodeSectionSizeMissing
			}
			header.codeSize = binary.BigEndian.Uint16(code[i : i+2])
			// Code section size must not be 0.
			if header.codeSize == 0 {
				return EOF1Header{}, ErrEOF1EmptyCodeSection
			}
			i += 2
		case sectionKindData:
			// Data section is allowed only after code section.
			if header.codeSize == 0 {
				return EOF1Header{}, ErrEOF1DataSectionBeforeCodeSection
			}
			// Only 1 data section is allowed.
			if header.dataSize != 0 {
				return EOF1Header{}, ErrEOF1MultipleDataSections
			}
			// Data section size must be present.
			if i+2 > codeLen {
				return EOF1Header{}, ErrEOF1DataSectionSizeMissing
			}
			header.dataSize = binary.BigEndian.Uint16(code[i : i+2])
			// Data section size must not be 0.
			if header.dataSize == 0 {
				return EOF1Header{}, ErrEOF1EmptyDataSection
			}
			i += 2
		default:
			return EOF1Header{}, ErrEOF1UnknownSection
		}
	}
	// 1 code section is required.
	if header.codeSize == 0 {
		return EOF1Header{}, ErrEOF1CodeSectionMissing
	}
	// Declared section sizes must correspond to real size (trailing bytes are not allowed.)
	if i+int(header.codeSize)+int(header.dataSize) != codeLen {
		return EOF1Header{}, ErrEOF1InvalidTotalSize
	}

	return header, nil
}

// validateInstructions checks that there're no undefined instructions and code ends with a terminating instruction
func validateInstructions(code []byte, header *EOF1Header, jumpTable *JumpTable) error {
	i := header.CodeBeginOffset()
	var opcode OpCode
	for i < header.CodeEndOffset() {
		opcode = OpCode(code[i])
		if reflect.ValueOf(jumpTable[opcode].execute).Pointer() == reflect.ValueOf(opUndefined).Pointer() {
			return ErrEOF1UndefinedInstruction
		}
		if opcode >= PUSH1 && opcode <= PUSH32 {
			i += uint64(opcode) - uint64(PUSH1) + 1
		}
		i += 1
	}

	if !opcode.isTerminating() {
		return ErrEOF1TerminatingInstructionMissing
	}

	return nil
}

// validateEOF returns true if code has valid format and code section
func validateEOF(code []byte, jumpTable *JumpTable) (EOF1Header, error) {
	header, err := readEOF1Header(code)
	if err != nil {
		return EOF1Header{}, err
	}
	err = validateInstructions(code, &header, jumpTable)
	if err != nil {
		return EOF1Header{}, err
	}
	return header, nil
}

// readValidEOF1Header parses EOF1-formatted code header, assuming that it is already validated
func readValidEOF1Header(code []byte) EOF1Header {
	var header EOF1Header
	codeSizeOffset := 2 + eofMagicLen
	header.codeSize = binary.BigEndian.Uint16(code[codeSizeOffset : codeSizeOffset+2])
	if code[codeSizeOffset+2] == 2 {
		dataSizeOffset := codeSizeOffset + 3
		header.dataSize = binary.BigEndian.Uint16(code[dataSizeOffset : dataSizeOffset+2])
	}
	return header
}

// CodeBeginOffset returns starting offset of the code section
func (header *EOF1Header) CodeBeginOffset() uint64 {
	if header.dataSize == 0 {
		// len(magic) + version + code_section_id + code_section_size + terminator
		return uint64(5 + eofMagicLen)
	}
	// len(magic) + version + code_section_id + code_section_size + data_section_id + data_section_size + terminator
	return uint64(8 + eofMagicLen)
}

// CodeEndOffset returns offset of the code section end
func (header *EOF1Header) CodeEndOffset() uint64 {
	return header.CodeBeginOffset() + uint64(header.codeSize)
}
