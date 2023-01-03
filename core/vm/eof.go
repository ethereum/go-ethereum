// Copyright 2022 The go-ethereum Authors
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
	"bytes"
	"encoding/binary"
	"fmt"
	"io"

	"github.com/ethereum/go-ethereum/common"
)

const (
	offsetVersion   = 2
	offsetTypesKind = 3
	offsetCodeKind  = 6

	kindTypes = 1
	kindCode  = 2
	kindData  = 3

	eofFormatByte = 0xef
	eof1Version   = 1

	maxInputItems  = 127
	maxOutputItems = 127
	maxStackHeight = 1023
)

var eofMagic = []byte{0xef, 0x00}

// hasEOFByte returns true if code starts with 0xEF byte
func hasEOFByte(code []byte) bool {
	return len(code) != 0 && code[0] == eofFormatByte
}

// hasEOFMagic returns true if code starts with magic defined by EIP-3540
func hasEOFMagic(code []byte) bool {
	return len(eofMagic) <= len(code) && bytes.Equal(eofMagic, code[0:len(eofMagic)])
}

// isEOFVersion1 returns true if the code's version byte equals eof1Version. It
// does not verify the EOF magic is valid.
func isEOFVersion1(code []byte) bool {
	return 2 < len(code) && code[2] == byte(eof1Version)
}

// Container is an EOF container object.
type Container struct {
	Types []*FunctionMetadata
	Code  [][]byte
	Data  []byte
}

// FunctionMetadata is an EOF function signature.
type FunctionMetadata struct {
	Input          uint8
	Output         uint8
	MaxStackHeight uint16
}

// MarshalBinary encodes an EOF container into binary format.
func (c *Container) MarshalBinary() []byte {
	// Build EOF prefix.
	b := make([]byte, 2)
	copy(b, eofMagic)
	b = append(b, eof1Version)

	// Write section headers.
	b = append(b, kindTypes)
	b = binary.BigEndian.AppendUint16(b, uint16(len(c.Types)*4))
	b = append(b, kindCode)
	b = binary.BigEndian.AppendUint16(b, uint16(len(c.Code)))
	for _, code := range c.Code {
		b = binary.BigEndian.AppendUint16(b, uint16(len(code)))
	}
	b = append(b, kindData)
	b = binary.BigEndian.AppendUint16(b, uint16(len(c.Data)))
	b = append(b, 0) // terminator

	// Write section contents.
	for _, ty := range c.Types {
		b = append(b, []byte{ty.Input, ty.Output, byte(ty.MaxStackHeight >> 8), byte(ty.MaxStackHeight & 0x00ff)}...)
	}
	for _, code := range c.Code {
		b = append(b, code...)
	}
	b = append(b, c.Data...)

	return b
}

// UnmarshalBinary decodes an EOF container.
func (c *Container) UnmarshalBinary(b []byte) error {
	if !hasEOFMagic(b) {
		return fmt.Errorf("invalid magic: have %s, want %s", common.Bytes2Hex(b[:min(2, len(b))]), common.Bytes2Hex(eofMagic))
	}
	if !isEOFVersion1(b) {
		have := "<nil>"
		if len(b) > 3 {
			have = fmt.Sprintf("%d", int(b[2]))
		}
		return fmt.Errorf("invalid eof version: have %s, want %d", have, eof1Version)
	}

	var (
		kind, typesSize, dataSize int
		codeSizes                 []int
		err                       error
	)

	// Parse type section header.
	kind, typesSize, err = parseSection(b, offsetTypesKind)
	if err != nil {
		return err
	}
	if kind != kindTypes {
		return fmt.Errorf("expected kind types 0x01: have %x", kind)
	}
	if typesSize < 4 || typesSize%4 != 0 {
		return fmt.Errorf("type section size must be divisible by 4: have %d", typesSize)
	}
	if typesSize/4 > 1024 {
		return fmt.Errorf("number of code sections must not exceed 1024: got %d", typesSize/4)
	}

	// Parse code section header.
	kind, codeSizes, err = parseSectionList(b, offsetCodeKind)
	if err != nil {
		return fmt.Errorf("failed to parse section list: %v", err)
	}
	if kind != kindCode {
		return fmt.Errorf("expected kind code 0x02: have %x", kind)
	}
	if len(codeSizes) != typesSize/4 {
		return fmt.Errorf("mismatch of code sections count and type signatures: types %d, code %d)", typesSize/4, len(codeSizes))
	}

	// Parse data section header.
	offsetDataKind := offsetCodeKind + 2 + 2*len(codeSizes) + 1
	kind, dataSize, err = parseSection(b, offsetDataKind)
	if err != nil {
		return err
	}
	if kind != kindData {
		return fmt.Errorf("expected kind data 0x03: have %x", kind)
	}

	// Check for terminator.
	offsetTerminator := offsetDataKind + 3
	if len(b) < offsetTerminator || b[offsetTerminator] != 0 {
		return fmt.Errorf("expected terminator")
	}

	// Verify overall container size.
	expectedSize := offsetTerminator + typesSize + sum(codeSizes) + dataSize + 1
	if len(b) != expectedSize {
		return fmt.Errorf("invalid container size: have %d, want %d", len(b), expectedSize)
	}

	// Parse types section.
	idx := offsetTerminator + 1
	var types []*FunctionMetadata
	for i := 0; i < typesSize/4; i++ {
		sig := &FunctionMetadata{
			Input:          b[idx+i*4],
			Output:         b[idx+i*4+1],
			MaxStackHeight: uint16(binary.BigEndian.Uint16(b[idx+i*4+2:])),
		}
		if sig.Input > maxInputItems {
			return fmt.Errorf("invalid type annotation at index %d: inputs must not exceed %d: have %d", i, maxInputItems, sig.Input)
		}
		if sig.Output > maxOutputItems {
			return fmt.Errorf("invalid type annotation at index %d: inputs and outputs must not exceed %d: have %d", i, maxOutputItems, sig.Output)
		}
		if sig.MaxStackHeight > maxStackHeight {
			return fmt.Errorf("invalid type annotation at index %d: max stack height must not exceed %d: have %d", i, maxStackHeight, sig.MaxStackHeight)
		}
		types = append(types, sig)
	}
	if types[0].Input != 0 || types[0].Output != 0 {
		return fmt.Errorf("invalid type annotation at index 0: input and output must be 0: have (%d, %d)", types[0].Input, types[0].Output)
	}
	c.Types = types

	// Parse code sections.
	idx += typesSize
	code := make([][]byte, len(codeSizes))
	for i, size := range codeSizes {
		if size == 0 {
			return fmt.Errorf("invalid code section %d: size must not be 0", i)
		}
		code[i] = b[idx : idx+size]
		idx += size
	}
	c.Code = code

	// Parse data section.
	c.Data = b[idx : idx+dataSize]

	return nil
}

// ValidateCode validates each code section of the container against the EOF v1
// rule set.
func (c *Container) ValidateCode(jt *JumpTable) error {
	for i, code := range c.Code {
		if err := validateCode(code, i, c.Types, jt); err != nil {
			return err
		}
	}
	return nil
}

// parseSection decodes a (kind, size) pair from an EOF header.
func parseSection(b []byte, idx int) (kind, size int, err error) {
	if idx+3 >= len(b) {
		return 0, 0, io.ErrUnexpectedEOF
	}
	kind = int(b[idx])
	size = int(binary.BigEndian.Uint16(b[idx+1:]))
	return kind, size, nil
}

// parseSectionList decodes a (kind, len, []codeSize) section list from an EOF
// header.
func parseSectionList(b []byte, idx int) (kind int, list []int, err error) {
	if idx >= len(b) {
		return 0, nil, io.ErrUnexpectedEOF
	}
	kind = int(b[idx])
	list, err = parseList(b, idx+1)
	if err != nil {
		return 0, nil, err
	}
	return kind, list, nil
}

// parseList decodes a list of uint16..
func parseList(b []byte, idx int) ([]int, error) {
	if len(b) < idx+2 {
		return nil, io.ErrUnexpectedEOF
	}
	count := binary.BigEndian.Uint16(b[idx:])
	if len(b) <= idx+2+int(count)*2 {
		return nil, io.ErrUnexpectedEOF

	}
	list := make([]int, count)
	for i := 0; i < int(count); i++ {
		list[i] = int(binary.BigEndian.Uint16(b[idx+2+2*i:]))
	}
	return list, nil
}

// parseUint16 parses a 16 bit unsigned integer.
func parseUint16(b []byte) (int, error) {
	if len(b) < 2 {
		return 0, io.ErrUnexpectedEOF
	}
	return int(binary.BigEndian.Uint16(b)), nil
}

// parseInt16 parses a 16 bit signed integer.
func parseInt16(b []byte) int {
	return int(int16(b[1]) | int16(b[0])<<8)
}

// max returns the maximum of a and b.
func max(a, b int) int {
	if a < b {
		return b
	}
	return a
}

// min returns the minimum value between x and y.
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// sum computes the sum of a slice.
func sum(list []int) (s int) {
	for _, n := range list {
		s += n
	}
	return
}
