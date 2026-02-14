// Copyright 2015 The go-ethereum Authors
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

package rlp

import (
	"fmt"
	"io"
	"reflect"
	"slices"
)

// RawValue represents an encoded RLP value and can be used to delay
// RLP decoding or to precompute an encoding. Note that the decoder does
// not verify whether the content of RawValues is valid RLP.
type RawValue []byte

var rawValueType = reflect.TypeFor[RawValue]()

// RawList represents an encoded RLP list.
type RawList[T any] struct {
	// The list is stored in encoded form.
	// Note this buffer has some special properties:
	//
	//   - if the buffer is nil, it's the zero value, representing
	//     an empty list.
	//   - if the buffer is non-nil, it must have a length of at least
	//     9 bytes, which is reserved padding for the encoded list header.
	//     The remaining bytes, enc[9:], store the content bytes of the list.
	//
	// The implementation code mostly works with the Content method because it
	// returns something valid either way.
	enc []byte

	// length holds the number of items in the list.
	length int
}

// Content returns the RLP-encoded data of the list.
// This does not include the list-header.
// The return value is a direct reference to the internal buffer, not a copy.
func (r *RawList[T]) Content() []byte {
	if r.enc == nil {
		return nil
	} else {
		return r.enc[9:]
	}
}

// EncodeRLP writes the encoded list to the writer.
func (r RawList[T]) EncodeRLP(w io.Writer) error {
	_, err := w.Write(r.Bytes())
	return err
}

// Bytes returns the RLP encoding of the list.
// Note the return value aliases the internal buffer.
func (r *RawList[T]) Bytes() []byte {
	if r == nil || r.enc == nil {
		return []byte{0xC0} // zero value encodes as empty list
	}
	n := puthead(r.enc, 0xC0, 0xF7, uint64(len(r.Content())))
	copy(r.enc[9-n:], r.enc[:n])
	return r.enc[9-n:]
}

// DecodeRLP decodes the list. This does not perform validation of the items!
func (r *RawList[T]) DecodeRLP(s *Stream) error {
	k, size, err := s.Kind()
	if err != nil {
		return err
	}
	if k != List {
		return fmt.Errorf("%w for %T", ErrExpectedList, r)
	}
	enc := make([]byte, 9+size)
	if err := s.readFull(enc[9:]); err != nil {
		return err
	}
	n, err := CountValues(enc[9:])
	if err != nil {
		if err == ErrValueTooLarge {
			return ErrElemTooLarge
		}
		return err
	}
	*r = RawList[T]{enc: enc, length: n}
	return nil
}

// Items decodes and returns all items in the list.
func (r *RawList[T]) Items() ([]T, error) {
	items := make([]T, r.Len())
	it := r.ContentIterator()
	for i := 0; it.Next(); i++ {
		if err := DecodeBytes(it.Value(), &items[i]); err != nil {
			return items[:i], err
		}
	}
	return items, nil
}

// Len returns the number of items in the list.
func (r *RawList[T]) Len() int {
	return r.length
}

// Size returns the encoded size of the list.
func (r *RawList[T]) Size() uint64 {
	return ListSize(uint64(len(r.Content())))
}

// ContentIterator returns an iterator over the content of the list.
// Note the offsets returned by iterator.Offset are relative to the
// Content bytes of the list.
func (r *RawList[T]) ContentIterator() Iterator {
	return newIterator(r.Content(), 0)
}

// Append adds an item to the end of the list.
func (r *RawList[T]) Append(item T) error {
	if r.enc == nil {
		r.enc = make([]byte, 9)
	}

	eb := getEncBuffer()
	defer encBufferPool.Put(eb)

	if err := eb.encode(item); err != nil {
		return err
	}
	prevEnd := len(r.enc)
	end := prevEnd + eb.size()
	r.enc = slices.Grow(r.enc, eb.size())[:end]
	eb.copyTo(r.enc[prevEnd:end])
	r.length++
	return nil
}

// AppendRaw adds an encoded item to the list.
// The given byte slice must contain exactly one RLP value.
func (r *RawList[T]) AppendRaw(b []byte) error {
	_, tagsize, contentsize, err := readKind(b)
	if err != nil {
		return err
	}
	if tagsize+contentsize != uint64(len(b)) {
		return fmt.Errorf("rlp: input has trailing bytes in AppendRaw")
	}
	if r.enc == nil {
		r.enc = make([]byte, 9)
	}
	r.enc = append(r.enc, b...)
	r.length++
	return nil
}

// StringSize returns the encoded size of a string.
func StringSize(s string) uint64 {
	switch n := len(s); n {
	case 0:
		return 1
	case 1:
		if s[0] <= 0x7f {
			return 1
		} else {
			return 2
		}
	default:
		return uint64(headsize(uint64(n)) + n)
	}
}

// BytesSize returns the encoded size of a byte slice.
func BytesSize(b []byte) uint64 {
	switch n := len(b); n {
	case 0:
		return 1
	case 1:
		if b[0] <= 0x7f {
			return 1
		} else {
			return 2
		}
	default:
		return uint64(headsize(uint64(n)) + n)
	}
}

// ListSize returns the encoded size of an RLP list with the given
// content size.
func ListSize(contentSize uint64) uint64 {
	return uint64(headsize(contentSize)) + contentSize
}

// IntSize returns the encoded size of the integer x. Note: The return type of this
// function is 'int' for backwards-compatibility reasons. The result is always positive.
func IntSize(x uint64) int {
	if x < 0x80 {
		return 1
	}
	return 1 + intsize(x)
}

// Split returns the content of first RLP value and any
// bytes after the value as subslices of b.
func Split(b []byte) (k Kind, content, rest []byte, err error) {
	k, ts, cs, err := readKind(b)
	if err != nil {
		return 0, nil, b, err
	}
	return k, b[ts : ts+cs], b[ts+cs:], nil
}

// SplitString splits b into the content of an RLP string
// and any remaining bytes after the string.
func SplitString(b []byte) (content, rest []byte, err error) {
	k, content, rest, err := Split(b)
	if err != nil {
		return nil, b, err
	}
	if k == List {
		return nil, b, ErrExpectedString
	}
	return content, rest, nil
}

// SplitUint64 decodes an integer at the beginning of b.
// It also returns the remaining data after the integer in 'rest'.
func SplitUint64(b []byte) (x uint64, rest []byte, err error) {
	content, rest, err := SplitString(b)
	if err != nil {
		return 0, b, err
	}
	switch n := len(content); n {
	case 0:
		return 0, rest, nil
	case 1:
		if content[0] == 0 {
			return 0, b, ErrCanonInt
		}
		return uint64(content[0]), rest, nil
	default:
		if n > 8 {
			return 0, b, errUintOverflow
		}

		x, err = readSize(content, byte(n))
		if err != nil {
			return 0, b, ErrCanonInt
		}
		return x, rest, nil
	}
}

// SplitList splits b into the content of a list and any remaining
// bytes after the list.
func SplitList(b []byte) (content, rest []byte, err error) {
	k, content, rest, err := Split(b)
	if err != nil {
		return nil, b, err
	}
	if k != List {
		return nil, b, ErrExpectedList
	}
	return content, rest, nil
}

// CountValues counts the number of encoded values in b.
func CountValues(b []byte) (int, error) {
	i := 0
	for ; len(b) > 0; i++ {
		_, tagsize, size, err := readKind(b)
		if err != nil {
			return i + 1, err
		}
		b = b[tagsize+size:]
	}
	return i, nil
}

// SplitListValues extracts the raw elements from the list RLP-encoding blob.
//
// Note: the returned slice must not be modified, as it shares the same
// backing array as the original slice. It's acceptable to deep-copy the elements
// out if necessary, but let's stick with this approach for less allocation
// overhead.
func SplitListValues(b []byte) ([][]byte, error) {
	b, _, err := SplitList(b)
	if err != nil {
		return nil, err
	}
	n, err := CountValues(b)
	if err != nil {
		return nil, err
	}
	var elements = make([][]byte, 0, n)

	for len(b) > 0 {
		_, tagsize, size, err := readKind(b)
		if err != nil {
			return nil, err
		}
		elements = append(elements, b[:tagsize+size])
		b = b[tagsize+size:]
	}
	return elements, nil
}

// MergeListValues takes a list of raw elements and rlp-encodes them as list.
func MergeListValues(elems [][]byte) ([]byte, error) {
	w := NewEncoderBuffer(nil)
	offset := w.List()
	for _, elem := range elems {
		w.Write(elem)
	}
	w.ListEnd(offset)
	return w.ToBytes(), nil
}

func readKind(buf []byte) (k Kind, tagsize, contentsize uint64, err error) {
	if len(buf) == 0 {
		return 0, 0, 0, io.ErrUnexpectedEOF
	}
	b := buf[0]
	switch {
	case b < 0x80:
		k = Byte
		tagsize = 0
		contentsize = 1
	case b < 0xB8:
		k = String
		tagsize = 1
		contentsize = uint64(b - 0x80)
		// Reject strings that should've been single bytes.
		if contentsize == 1 && len(buf) > 1 && buf[1] < 128 {
			return 0, 0, 0, ErrCanonSize
		}
	case b < 0xC0:
		k = String
		tagsize = uint64(b-0xB7) + 1
		contentsize, err = readSize(buf[1:], b-0xB7)
	case b < 0xF8:
		k = List
		tagsize = 1
		contentsize = uint64(b - 0xC0)
	default:
		k = List
		tagsize = uint64(b-0xF7) + 1
		contentsize, err = readSize(buf[1:], b-0xF7)
	}
	if err != nil {
		return 0, 0, 0, err
	}
	// Reject values larger than the input slice.
	if contentsize > uint64(len(buf))-tagsize {
		return 0, 0, 0, ErrValueTooLarge
	}
	return k, tagsize, contentsize, err
}

func readSize(b []byte, slen byte) (uint64, error) {
	if int(slen) > len(b) {
		return 0, io.ErrUnexpectedEOF
	}
	var s uint64
	switch slen {
	case 1:
		s = uint64(b[0])
	case 2:
		s = uint64(b[0])<<8 | uint64(b[1])
	case 3:
		s = uint64(b[0])<<16 | uint64(b[1])<<8 | uint64(b[2])
	case 4:
		s = uint64(b[0])<<24 | uint64(b[1])<<16 | uint64(b[2])<<8 | uint64(b[3])
	case 5:
		s = uint64(b[0])<<32 | uint64(b[1])<<24 | uint64(b[2])<<16 | uint64(b[3])<<8 | uint64(b[4])
	case 6:
		s = uint64(b[0])<<40 | uint64(b[1])<<32 | uint64(b[2])<<24 | uint64(b[3])<<16 | uint64(b[4])<<8 | uint64(b[5])
	case 7:
		s = uint64(b[0])<<48 | uint64(b[1])<<40 | uint64(b[2])<<32 | uint64(b[3])<<24 | uint64(b[4])<<16 | uint64(b[5])<<8 | uint64(b[6])
	case 8:
		s = uint64(b[0])<<56 | uint64(b[1])<<48 | uint64(b[2])<<40 | uint64(b[3])<<32 | uint64(b[4])<<24 | uint64(b[5])<<16 | uint64(b[6])<<8 | uint64(b[7])
	}
	// Reject sizes < 56 (shouldn't have separate size) and sizes with
	// leading zero bytes.
	if s < 56 || b[0] == 0 {
		return 0, ErrCanonSize
	}
	return s, nil
}

// AppendUint64 appends the RLP encoding of i to b, and returns the resulting slice.
func AppendUint64(b []byte, i uint64) []byte {
	if i == 0 {
		return append(b, 0x80)
	} else if i < 128 {
		return append(b, byte(i))
	}
	switch {
	case i < (1 << 8):
		return append(b, 0x81, byte(i))
	case i < (1 << 16):
		return append(b, 0x82,
			byte(i>>8),
			byte(i),
		)
	case i < (1 << 24):
		return append(b, 0x83,
			byte(i>>16),
			byte(i>>8),
			byte(i),
		)
	case i < (1 << 32):
		return append(b, 0x84,
			byte(i>>24),
			byte(i>>16),
			byte(i>>8),
			byte(i),
		)
	case i < (1 << 40):
		return append(b, 0x85,
			byte(i>>32),
			byte(i>>24),
			byte(i>>16),
			byte(i>>8),
			byte(i),
		)

	case i < (1 << 48):
		return append(b, 0x86,
			byte(i>>40),
			byte(i>>32),
			byte(i>>24),
			byte(i>>16),
			byte(i>>8),
			byte(i),
		)
	case i < (1 << 56):
		return append(b, 0x87,
			byte(i>>48),
			byte(i>>40),
			byte(i>>32),
			byte(i>>24),
			byte(i>>16),
			byte(i>>8),
			byte(i),
		)

	default:
		return append(b, 0x88,
			byte(i>>56),
			byte(i>>48),
			byte(i>>40),
			byte(i>>32),
			byte(i>>24),
			byte(i>>16),
			byte(i>>8),
			byte(i),
		)
	}
}
