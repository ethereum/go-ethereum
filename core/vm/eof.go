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

// Container is and EOF container object.
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
	b := make([]byte, 2)
	copy(b, eofMagic)
	b = append(b, eof1Version)
	b = append(b, kindTypes)
	b = binary.BigEndian.AppendUint16(b, uint16(len(c.Types)*4))
	//b = appendUint16(b, uint16(len(c.Types)*4))
	b = append(b, kindCode)
	b = binary.BigEndian.AppendUint16(b, uint16(len(c.Code)))
	for _, code := range c.Code {
		b = binary.BigEndian.AppendUint16(b, uint16(len(code)))
	}
	b = append(b, kindData)
	b = binary.BigEndian.AppendUint16(b, uint16(len(c.Data)))
	b = append(b, 0) // terminator

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
		return fmt.Errorf("invalid magic")
	}
	if !isEOFVersion1(b) {
		return fmt.Errorf("invalid eof version")
	}

	if len(b) < 15 {
		return fmt.Errorf("container size less than minimum valid size")
	}

	var (
		typesSize, dataSize int
		codeSizes           []int
		kind                int
		err                 error
	)

	// Parse types size.
	if kind, typesSize, err = parseSection(b, offsetTypesKind); err != nil {
		return err
	} else if kind != kindTypes {
		return fmt.Errorf("expected kind types")
	}
	if typesSize < 4 || typesSize%4 != 0 {
		return fmt.Errorf("type section size invalid")
	}
	if typesSize > 4*1024 {
		return fmt.Errorf("number of code sections must not exceed 1024 (got %d)", typesSize)
	}

	// Parse code sizes.
	if kind, codeSizes, err = parseSectionList(b, offsetCodeKind); err != nil {
		return fmt.Errorf("failed to parse section list: %v", err)
	} else if kind != kindCode {
		return fmt.Errorf("expected kind code")
	}
	if len(codeSizes) != typesSize/4 {
		return fmt.Errorf("mismatch of code sections count and type signatures (types %d, code %d)", typesSize/3, len(codeSizes))
	}

	// Parse data size.
	offsetDataKind := offsetCodeKind + 2 + 2*len(codeSizes) + 1
	if len(b) < offsetDataKind+2 {
		return fmt.Errorf("container size invalid")
	}
	if kind, dataSize, err = parseSection(b, offsetDataKind); err != nil {
		return err
	} else if kind != kindData {
		return fmt.Errorf("expected kind data")
	}
	offsetTerminator := offsetDataKind + 3
	if check(b, offsetTerminator, 0) {
		return fmt.Errorf("expected terminator")
	}

	// Check for terminator.
	expectedSize := offsetTerminator + typesSize + sum(codeSizes) + dataSize + 1
	if len(b) != expectedSize {
		return fmt.Errorf("invalid container size (want %d, got %d)", expectedSize, len(b))
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
		if sig.Output > 127 || sig.Input > 127 {
			return fmt.Errorf("type annotation %d inputs and outputs must not exceed 127", i)
		}
		if sig.MaxStackHeight > 1023 {
			return fmt.Errorf("type annotation %d max stack height must not exceed 1023", i)
		}
		types = append(types, sig)
	}
	if types[0].Input != 0 || types[0].Output != 0 {
		return fmt.Errorf("input and output of first code section must be 0")
	}
	c.Types = types

	// Parse code sections.
	idx += typesSize
	code := make([][]byte, len(codeSizes))
	for i, size := range codeSizes {
		if size == 0 {
			return fmt.Errorf("code section %d size must not be 0", i)
		}
		code[i] = b[idx : idx+size]
		idx += size
	}
	c.Code = code

	// Parse data section.
	c.Data = b[idx : idx+dataSize]

	return nil
}

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

// check returns if b[idx] == want after performing a bounds check.
func check(b []byte, idx int, want byte) bool {
	if len(b) < idx {
		return false
	}
	return b[idx] != want
}

func sum(list []int) (s int) {
	for _, n := range list {
		s += n
	}
	return
}
