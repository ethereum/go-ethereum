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
)

var eofFormatByte byte = 0xEF
var eofMagic = [...]byte{0xCA, 0xFE}
var eof1Version byte = 1

type eof1Header struct {
	codeSize uint16 // Size of code section. Cannot be 0 for EOF1 code. Equals 0 for legacy code.
	dataSize uint16 // Size of data section. Equals 0 if data section is absent in EOF1 code. Equals 0 for legacy code.
}

// hasFormatByte returns true if code starts with FORMAT byte
func hasFormatByte(code []byte) bool {
	return len(code) != 0 && code[0] == eofFormatByte
}

// hasEOFMagic returns true if code contains magic defined by EIP-3540
func hasEOFMagic(code []byte) bool {
	magicLen := len(eofMagic)
	codeLen := len(code)

	return 1+magicLen <= codeLen && bytes.Equal(code[1:1+magicLen], eofMagic[:])
}

// isEOFCode returns true if code starts with valid FORMAT byte + EOF magic
func isEOFCode(code []byte) bool {
	return hasFormatByte(code) && hasEOFMagic(code)
}

// readEOF1Header parses EOF1-formatted code header
func readEOF1Header(code []byte) (eof1Header, error) {
	if !hasFormatByte(code) {
		return eof1Header{}, ErrEOF1InvalidFormatByte
	}

	if !hasEOFMagic(code) {
		return eof1Header{}, ErrEOF1InvalidMagic
	}

	codeLen := len(code)

	i := 1 + len(eofMagic)
	if i >= codeLen || code[i] != eof1Version {
		return eof1Header{}, ErrEOF1InvalidVersion
	}
	i += 1

	var header eof1Header
sectionLoop:
	for i < codeLen {
		sectionKind := code[i]
		i += 1
		switch sectionKind {
		case 0:
			break sectionLoop
		case 1:
			// Only 1 code section is allowed.
			if header.codeSize != 0 {
				return eof1Header{}, ErrEOF1MultipleCodeSections
			}
			// Code section size must be present.
			if i+2 > codeLen {
				return eof1Header{}, ErrEOF1CodeSectionSizeMissing
			}
			header.codeSize = binary.BigEndian.Uint16(code[i : i+2])
			// Code section size must not be 0.
			if header.codeSize == 0 {
				return eof1Header{}, ErrEOF1EmptyCodeSection
			}
			i += 2
		case 2:
			// Data section is allowed only after code section.
			if header.codeSize == 0 {
				return eof1Header{}, ErrEOF1DataSectionBeforeCodeSection
			}
			// Only 1 data section is allowed.
			if header.dataSize != 0 {
				return eof1Header{}, ErrEOF1MultipleDataSections
			}
			// Data section size must be present.
			if i+2 > codeLen {
				return eof1Header{}, ErrEOF1DataSectionSizeMissing
			}
			header.dataSize = binary.BigEndian.Uint16(code[i : i+2])
			// Data section size must not be 0.
			if header.dataSize == 0 {
				return eof1Header{}, ErrEOF1EmptyDataSection
			}
			i += 2
		default:
			return eof1Header{}, ErrEOF1UnknownSection
		}
	}
	// 1 code section is required.
	if header.codeSize == 0 {
		return eof1Header{}, ErrEOF1CodeSectionMissing
	}
	// Declared section sizes must correspond to real size (trailing bytes are not allowed.)
	if i+int(header.codeSize)+int(header.dataSize) != codeLen {
		return eof1Header{}, ErrEOF1InvalidTotalSize
	}

	return header, nil
}

// validateEOF returns true if code has valid format
func validateEOF(code []byte) bool {
	_, err := readEOF1Header(code)
	return err == nil
}

// readValidEOF1Header parses EOF1-formatted code header, assuming that it is already validated
func readValidEOF1Header(code []byte) eof1Header {
	var header eof1Header
	codeSizeOffset := 3 + len(eofMagic)
	header.codeSize = binary.BigEndian.Uint16(code[codeSizeOffset : codeSizeOffset+2])
	if code[codeSizeOffset+2] == 2 {
		dataSizeOffset := codeSizeOffset + 3
		header.dataSize = binary.BigEndian.Uint16(code[dataSizeOffset : dataSizeOffset+2])
	}
	return header
}
