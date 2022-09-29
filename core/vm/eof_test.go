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
	{"EF0001010002006000", 2, 0},
	{"EF0001010002020001006000AA", 2, 1},
	{"EF0001010002020004006000AABBCCDD", 2, 4},
	{"EF00010100040200020060006001AABB", 4, 2},
	{"EF000101000602000400600060016002AABBCCDD", 6, 4},
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
	for _, test := range eof1ValidTests {
		if !validateEOF(common.Hex2Bytes(test.code)) {
			t.Errorf("code %v expected to be valid", test.code)
		}
	}

	for _, test := range eof1InvalidTests {
		if validateEOF(common.Hex2Bytes(test.code)) {
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
