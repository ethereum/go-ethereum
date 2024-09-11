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
	"encoding/hex"
	"errors"
	"fmt"
	"io"

	"github.com/ethereum/go-ethereum/params"
)

const (
	offsetVersion   = 2
	offsetTypesKind = 3
	offsetCodeKind  = 6

	kindTypes     = 1
	kindCode      = 2
	kindContainer = 3
	kindData      = 4

	eofFormatByte = 0xef
	eof1Version   = 1

	maxInputItems        = 127
	maxOutputItems       = 128
	maxStackHeight       = 1023
	maxContainerSections = 256
)

var eofMagic = []byte{0xef, 0x00}

// HasEOFByte returns true if code starts with 0xEF byte
func HasEOFByte(code []byte) bool {
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
	types             []*functionMetadata
	codeSections      [][]byte
	subContainers     []*Container
	subContainerCodes [][]byte
	data              []byte
	dataSize          int // might be more than len(data)
}

// functionMetadata is an EOF function signature.
type functionMetadata struct {
	inputs         uint8
	outputs        uint8
	maxStackHeight uint16
}

// MarshalBinary encodes an EOF container into binary format.
func (c *Container) MarshalBinary() []byte {
	// Build EOF prefix.
	b := make([]byte, 2)
	copy(b, eofMagic)
	b = append(b, eof1Version)

	// Write section headers.
	b = append(b, kindTypes)
	b = binary.BigEndian.AppendUint16(b, uint16(len(c.types)*4))
	b = append(b, kindCode)
	b = binary.BigEndian.AppendUint16(b, uint16(len(c.codeSections)))
	for _, codeSection := range c.codeSections {
		b = binary.BigEndian.AppendUint16(b, uint16(len(codeSection)))
	}
	var encodedContainer [][]byte
	if len(c.subContainers) != 0 {
		b = append(b, kindContainer)
		b = binary.BigEndian.AppendUint16(b, uint16(len(c.subContainers)))
		for _, section := range c.subContainers {
			encoded := section.MarshalBinary()
			b = binary.BigEndian.AppendUint16(b, uint16(len(encoded)))
			encodedContainer = append(encodedContainer, encoded)
		}
	}
	b = append(b, kindData)
	b = binary.BigEndian.AppendUint16(b, uint16(c.dataSize))
	b = append(b, 0) // terminator

	// Write section contents.
	for _, ty := range c.types {
		b = append(b, []byte{ty.inputs, ty.outputs, byte(ty.maxStackHeight >> 8), byte(ty.maxStackHeight & 0x00ff)}...)
	}
	for _, code := range c.codeSections {
		b = append(b, code...)
	}
	for _, section := range encodedContainer {
		b = append(b, section...)
	}
	b = append(b, c.data...)

	return b
}

// UnmarshalBinary decodes an EOF container.
func (c *Container) UnmarshalBinary(b []byte, isInitcode bool) error {
	return c.unmarshalSubContainer(b, isInitcode, true)
}

func (c *Container) unmarshalSubContainer(b []byte, isInitcode bool, topLevel bool) error {
	if !hasEOFMagic(b) {
		return fmt.Errorf("%w: want %x", ErrInvalidMagic, eofMagic)
	}
	if len(b) < 14 {
		return io.ErrUnexpectedEOF
	}
	if len(b) > params.MaxInitCodeSize {
		return ErrMaxInitCodeSizeExceeded
	}
	if !isEOFVersion1(b) {
		return fmt.Errorf("%w: have %d, want %d", ErrInvalidVersion, b[2], eof1Version)
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
		return fmt.Errorf("%w: found section kind %x instead", ErrMissingTypeHeader, kind)
	}
	if typesSize < 4 || typesSize%4 != 0 {
		return fmt.Errorf("%w: type section size must be divisible by 4, have %d", ErrInvalidTypeSize, typesSize)
	}
	if typesSize/4 > 1024 {
		return fmt.Errorf("%w: type section must not exceed 4*1024, have %d", ErrInvalidTypeSize, typesSize)
	}

	// Parse code section header.
	kind, codeSizes, err = parseSectionList(b, offsetCodeKind)
	if err != nil {
		return err
	}
	if kind != kindCode {
		return fmt.Errorf("%w: found section kind %x instead", ErrMissingCodeHeader, kind)
	}
	if len(codeSizes) != typesSize/4 {
		return fmt.Errorf("%w: mismatch of code sections found and type signatures, types %d, code %d", ErrInvalidCodeSize, typesSize/4, len(codeSizes))
	}

	// Parse (optional) container section header.
	var containerSizes []int
	offset := offsetCodeKind + 2 + 2*len(codeSizes) + 1
	if offset < len(b) && b[offset] == kindContainer {
		kind, containerSizes, err = parseSectionList(b, offset)
		if err != nil {
			return err
		}
		if kind != kindContainer {
			panic("somethings wrong")
		}
		if len(containerSizes) == 0 {
			return fmt.Errorf("%w: total container count must not be zero", ErrInvalidContainerSectionSize)
		}
		offset = offset + 2 + 2*len(containerSizes) + 1
	}

	// Parse data section header.
	kind, dataSize, err = parseSection(b, offset)
	if err != nil {
		return err
	}
	if kind != kindData {
		return fmt.Errorf("%w: found section %x instead", ErrMissingDataHeader, kind)
	}
	c.dataSize = dataSize

	// Check for terminator.
	offsetTerminator := offset + 3
	if len(b) < offsetTerminator {
		return fmt.Errorf("%w: invalid offset terminator", io.ErrUnexpectedEOF)
	}
	if b[offsetTerminator] != 0 {
		return fmt.Errorf("%w: have %x", ErrMissingTerminator, b[offsetTerminator])
	}

	// Verify overall container size.
	expectedSize := offsetTerminator + typesSize + sum(codeSizes) + dataSize + 1
	if len(containerSizes) != 0 {
		expectedSize += sum(containerSizes)
	}
	if len(b) < expectedSize-dataSize {
		return fmt.Errorf("%w: have %d, want %d", ErrInvalidContainerSize, len(b), expectedSize)
	}
	// Only check that the expected size is not exceed on non-initcode
	if !isInitcode && len(b) > expectedSize {
		return fmt.Errorf("%w: have %d, want %d", ErrInvalidContainerSize, len(b), expectedSize)
	}

	// Parse types section.
	idx := offsetTerminator + 1
	var types = make([]*functionMetadata, 0, typesSize/4)
	for i := 0; i < typesSize/4; i++ {
		sig := &functionMetadata{
			inputs:         b[idx+i*4],
			outputs:        b[idx+i*4+1],
			maxStackHeight: binary.BigEndian.Uint16(b[idx+i*4+2:]),
		}
		if sig.inputs > maxInputItems {
			return fmt.Errorf("%w for section %d: have %d", ErrTooManyInputs, i, sig.inputs)
		}
		if sig.outputs > maxOutputItems {
			return fmt.Errorf("%w for section %d: have %d", ErrTooManyOutputs, i, sig.outputs)
		}
		if sig.maxStackHeight > maxStackHeight {
			return fmt.Errorf("%w for section %d: have %d", ErrTooLargeMaxStackHeight, i, sig.maxStackHeight)
		}
		types = append(types, sig)
	}
	if types[0].inputs != 0 || types[0].outputs != 0x80 {
		return fmt.Errorf("%w: have %d, %d", ErrInvalidSection0Type, types[0].inputs, types[0].outputs)
	}
	c.types = types

	// Parse code sections.
	idx += typesSize
	codeSections := make([][]byte, len(codeSizes))
	for i, size := range codeSizes {
		if size == 0 {
			return fmt.Errorf("%w for section %d: size must not be 0", ErrInvalidCodeSize, i)
		}
		codeSections[i] = b[idx : idx+size]
		idx += size
	}
	c.codeSections = codeSections
	// Parse the optional container sizes.
	if len(containerSizes) != 0 {
		if len(containerSizes) > maxContainerSections {
			return fmt.Errorf("%w number of container section exceed: %v: have %v", ErrInvalidContainerSectionSize, maxContainerSections, len(containerSizes))
		}
		subContainerCodes := make([][]byte, 0, len(containerSizes))
		subContainers := make([]*Container, 0, len(containerSizes))
		for i, size := range containerSizes {
			if size == 0 || idx+size > len(b) {
				return fmt.Errorf("%w for section %d: size must not be 0", ErrInvalidContainerSectionSize, i)
			}
			subC := new(Container)
			end := min(idx+size, len(b))
			if err := subC.unmarshalSubContainer(b[idx:end], isInitcode, false); err != nil {
				if topLevel {
					return fmt.Errorf("%w in sub container %d", err, i)
				}
				return err
			}
			subContainers = append(subContainers, subC)
			subContainerCodes = append(subContainerCodes, b[idx:end])

			idx += size
		}
		c.subContainers = subContainers
		c.subContainerCodes = subContainerCodes
	}

	//Parse data section.
	end := len(b)
	if !isInitcode {
		end = min(idx+dataSize, len(b))
	}
	if topLevel && len(b) != idx+dataSize {
		return ErrTruncatedTopLevelContainer
	}
	c.data = b[idx:end]

	return nil
}

// ValidateCode validates each code section of the container against the EOF v1
// rule set.
func (c *Container) ValidateCode(jt *JumpTable, isInitCode bool) error {
	refBy := notRefByEither
	if isInitCode {
		refBy = refByEOFCreate
	}
	return c.validateSubContainer(jt, refBy)
}

func (c *Container) validateSubContainer(jt *JumpTable, refBy int) error {
	visited := make(map[int]struct{})
	subContainerVisited := make(map[int]int)
	toVisit := []int{0}
	for len(toVisit) > 0 {
		// TODO check if this can be used as a DOS
		// Theres and edge case here where we mark something as visited that we visit before,
		// This should not trigger a re-visit
		// e.g. 0 -> 1, 2, 3
		// 1 -> 2, 3
		// should not mean 2 and 3 should be visited twice
		var (
			index = toVisit[0]
			code  = c.codeSections[index]
		)
		if _, ok := visited[index]; !ok {
			res, err := validateCode(code, index, c, jt, refBy == refByEOFCreate)
			if err != nil {
				return err
			}
			visited[index] = struct{}{}
			// Mark all sections that can be visited from here.
			for idx := range res.visitedCode {
				if _, ok := visited[idx]; !ok {
					toVisit = append(toVisit, idx)
				}
			}
			// Mark all subcontainer that can be visited from here.
			for idx, reference := range res.visitedSubContainers {
				// Make sure subcontainers are only ever referenced by either EOFCreate or ReturnContract
				if ref, ok := subContainerVisited[idx]; ok && ref != reference {
					return errors.New("section referenced by both EOFCreate and ReturnContract")
				}
				subContainerVisited[idx] = reference
			}
			if refBy == refByReturnContract && res.isInitCode {
				return ErrIncompatibleContainerKind
			}
			if refBy == refByEOFCreate && res.isRuntime {
				return ErrIncompatibleContainerKind
			}
		}
		toVisit = toVisit[1:]
	}
	// Make sure every code section is visited at least once.
	if len(visited) != len(c.codeSections) {
		return ErrUnreachableCode
	}
	for idx, container := range c.subContainers {
		reference, ok := subContainerVisited[idx]
		if !ok {
			return ErrOrphanedSubcontainer
		}
		if err := container.validateSubContainer(jt, reference); err != nil {
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

// sum computes the sum of a slice.
func sum(list []int) (s int) {
	for _, n := range list {
		s += n
	}
	return
}

func (c *Container) String() string {
	var result string
	result += "Header\n"
	result += "-----------\n"
	result += fmt.Sprintf("EOFMagic: %02x\n", eofMagic)
	result += fmt.Sprintf("EOFVersion: %02x\n", eof1Version)
	result += fmt.Sprintf("KindType: %02x\n", kindTypes)
	result += fmt.Sprintf("TypesSize: %04x\n", len(c.types)*4)
	result += fmt.Sprintf("KindCode: %02x\n", kindCode)
	result += fmt.Sprintf("CodeSize: %04x\n", len(c.codeSections))
	for i, code := range c.codeSections {
		result += fmt.Sprintf("Code %v length: %04x\n", i, len(code))
	}
	if len(c.subContainers) != 0 {
		result += fmt.Sprintf("KindContainer: %02x\n", kindContainer)
		result += fmt.Sprintf("ContainerSize: %04x\n", len(c.subContainers))
		for i, section := range c.subContainers {
			result += fmt.Sprintf("Container %v length: %04x\n", i, len(section.MarshalBinary()))
		}
	}
	result += fmt.Sprintf("KindData: %02x\n", kindData)
	result += fmt.Sprintf("DataSize: %04x\n", len(c.data))
	result += fmt.Sprintf("Terminator: %02x\n", 0x0)
	result += "-----------\n"
	result += "Body\n"
	result += "-----------\n"
	for i, typ := range c.types {
		result += fmt.Sprintf("Type %v: %v\n", i, hex.EncodeToString([]byte{typ.inputs, typ.outputs, byte(typ.maxStackHeight >> 8), byte(typ.maxStackHeight & 0x00ff)}))
	}
	for i, code := range c.codeSections {
		result += fmt.Sprintf("Code %v: %v\n", i, hex.EncodeToString(code))
	}
	for i, section := range c.subContainers {
		result += fmt.Sprintf("Section %v: %v\n", i, hex.EncodeToString(section.MarshalBinary()))
	}
	result += fmt.Sprintf("Data: %v\n", hex.EncodeToString(c.data))
	return result
}
