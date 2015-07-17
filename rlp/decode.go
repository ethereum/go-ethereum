// Copyright 2014 The go-ethereum Authors
// This file is part of go-ethereum.
//
// go-ethereum is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// go-ethereum is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with go-ethereum.  If not, see <http://www.gnu.org/licenses/>.

package rlp

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"math/big"
	"reflect"
	"strings"
)

var (
	errNoPointer     = errors.New("rlp: interface given to Decode must be a pointer")
	errDecodeIntoNil = errors.New("rlp: pointer given to Decode must not be nil")
)

// Decoder is implemented by types that require custom RLP
// decoding rules or need to decode into private fields.
//
// The DecodeRLP method should read one value from the given
// Stream. It is not forbidden to read less or more, but it might
// be confusing.
type Decoder interface {
	DecodeRLP(*Stream) error
}

// Decode parses RLP-encoded data from r and stores the result in the
// value pointed to by val. Val must be a non-nil pointer. If r does
// not implement ByteReader, Decode will do its own buffering.
//
// Decode uses the following type-dependent decoding rules:
//
// If the type implements the Decoder interface, decode calls
// DecodeRLP.
//
// To decode into a pointer, Decode will decode into the value pointed
// to. If the pointer is nil, a new value of the pointer's element
// type is allocated. If the pointer is non-nil, the existing value
// will reused.
//
// To decode into a struct, Decode expects the input to be an RLP
// list. The decoded elements of the list are assigned to each public
// field in the order given by the struct's definition. The input list
// must contain an element for each decoded field. Decode returns an
// error if there are too few or too many elements.
//
// The decoding of struct fields honours one particular struct tag,
// "nil". This tag applies to pointer-typed fields and changes the
// decoding rules for the field such that input values of size zero
// decode as a nil pointer. This tag can be useful when decoding recursive
// types.
//
//     type StructWithEmptyOK struct {
//         Foo *[20]byte `rlp:"nil"`
//     }
//
// To decode into a slice, the input must be a list and the resulting
// slice will contain the input elements in order. For byte slices,
// the input must be an RLP string. Array types decode similarly, with
// the additional restriction that the number of input elements (or
// bytes) must match the array's length.
//
// To decode into a Go string, the input must be an RLP string. The
// input bytes are taken as-is and will not necessarily be valid UTF-8.
//
// To decode into an unsigned integer type, the input must also be an RLP
// string. The bytes are interpreted as a big endian representation of
// the integer. If the RLP string is larger than the bit size of the
// type, Decode will return an error. Decode also supports *big.Int.
// There is no size limit for big integers.
//
// To decode into an interface value, Decode stores one of these
// in the value:
//
//	  []interface{}, for RLP lists
//	  []byte, for RLP strings
//
// Non-empty interface types are not supported, nor are booleans,
// signed integers, floating point numbers, maps, channels and
// functions.
//
// Note that Decode does not set an input limit for all readers
// and may be vulnerable to panics cause by huge value sizes. If
// you need an input limit, use
//
//     NewStream(r, limit).Decode(val)
func Decode(r io.Reader, val interface{}) error {
	// TODO: this could use a Stream from a pool.
	return NewStream(r, 0).Decode(val)
}

// DecodeBytes parses RLP data from b into val.
// Please see the documentation of Decode for the decoding rules.
// The input must contain exactly one value and no trailing data.
func DecodeBytes(b []byte, val interface{}) error {
	// TODO: this could use a Stream from a pool.
	r := bytes.NewReader(b)
	if err := NewStream(r, uint64(len(b))).Decode(val); err != nil {
		return err
	}
	if r.Len() > 0 {
		return ErrMoreThanOneValue
	}
	return nil
}

type decodeError struct {
	msg string
	typ reflect.Type
	ctx []string
}

func (err *decodeError) Error() string {
	ctx := ""
	if len(err.ctx) > 0 {
		ctx = ", decoding into "
		for i := len(err.ctx) - 1; i >= 0; i-- {
			ctx += err.ctx[i]
		}
	}
	return fmt.Sprintf("rlp: %s for %v%s", err.msg, err.typ, ctx)
}

func wrapStreamError(err error, typ reflect.Type) error {
	switch err {
	case ErrCanonInt:
		return &decodeError{msg: "non-canonical integer (leading zero bytes)", typ: typ}
	case ErrCanonSize:
		return &decodeError{msg: "non-canonical size information", typ: typ}
	case ErrExpectedList:
		return &decodeError{msg: "expected input list", typ: typ}
	case ErrExpectedString:
		return &decodeError{msg: "expected input string or byte", typ: typ}
	case errUintOverflow:
		return &decodeError{msg: "input string too long", typ: typ}
	case errNotAtEOL:
		return &decodeError{msg: "input list has too many elements", typ: typ}
	}
	return err
}

func addErrorContext(err error, ctx string) error {
	if decErr, ok := err.(*decodeError); ok {
		decErr.ctx = append(decErr.ctx, ctx)
	}
	return err
}

var (
	decoderInterface = reflect.TypeOf(new(Decoder)).Elem()
	bigInt           = reflect.TypeOf(big.Int{})
)

func makeDecoder(typ reflect.Type, tags tags) (dec decoder, err error) {
	kind := typ.Kind()
	switch {
	case typ.Implements(decoderInterface):
		return decodeDecoder, nil
	case kind != reflect.Ptr && reflect.PtrTo(typ).Implements(decoderInterface):
		return decodeDecoderNoPtr, nil
	case typ.AssignableTo(reflect.PtrTo(bigInt)):
		return decodeBigInt, nil
	case typ.AssignableTo(bigInt):
		return decodeBigIntNoPtr, nil
	case isUint(kind):
		return decodeUint, nil
	case kind == reflect.String:
		return decodeString, nil
	case kind == reflect.Slice || kind == reflect.Array:
		return makeListDecoder(typ)
	case kind == reflect.Struct:
		return makeStructDecoder(typ)
	case kind == reflect.Ptr:
		if tags.nilOK {
			return makeOptionalPtrDecoder(typ)
		}
		return makePtrDecoder(typ)
	case kind == reflect.Interface:
		return decodeInterface, nil
	default:
		return nil, fmt.Errorf("rlp: type %v is not RLP-serializable", typ)
	}
}

func decodeUint(s *Stream, val reflect.Value) error {
	typ := val.Type()
	num, err := s.uint(typ.Bits())
	if err != nil {
		return wrapStreamError(err, val.Type())
	}
	val.SetUint(num)
	return nil
}

func decodeString(s *Stream, val reflect.Value) error {
	b, err := s.Bytes()
	if err != nil {
		return wrapStreamError(err, val.Type())
	}
	val.SetString(string(b))
	return nil
}

func decodeBigIntNoPtr(s *Stream, val reflect.Value) error {
	return decodeBigInt(s, val.Addr())
}

func decodeBigInt(s *Stream, val reflect.Value) error {
	b, err := s.Bytes()
	if err != nil {
		return wrapStreamError(err, val.Type())
	}
	i := val.Interface().(*big.Int)
	if i == nil {
		i = new(big.Int)
		val.Set(reflect.ValueOf(i))
	}
	// Reject leading zero bytes
	if len(b) > 0 && b[0] == 0 {
		return wrapStreamError(ErrCanonInt, val.Type())
	}
	i.SetBytes(b)
	return nil
}

func makeListDecoder(typ reflect.Type) (decoder, error) {
	etype := typ.Elem()
	if etype.Kind() == reflect.Uint8 && !reflect.PtrTo(etype).Implements(decoderInterface) {
		if typ.Kind() == reflect.Array {
			return decodeByteArray, nil
		} else {
			return decodeByteSlice, nil
		}
	}
	etypeinfo, err := cachedTypeInfo1(etype, tags{})
	if err != nil {
		return nil, err
	}

	isArray := typ.Kind() == reflect.Array
	return func(s *Stream, val reflect.Value) error {
		if isArray {
			return decodeListArray(s, val, etypeinfo.decoder)
		} else {
			return decodeListSlice(s, val, etypeinfo.decoder)
		}
	}, nil
}

func decodeListSlice(s *Stream, val reflect.Value, elemdec decoder) error {
	size, err := s.List()
	if err != nil {
		return wrapStreamError(err, val.Type())
	}
	if size == 0 {
		val.Set(reflect.MakeSlice(val.Type(), 0, 0))
		return s.ListEnd()
	}

	i := 0
	for ; ; i++ {
		// grow slice if necessary
		if i >= val.Cap() {
			newcap := val.Cap() + val.Cap()/2
			if newcap < 4 {
				newcap = 4
			}
			newv := reflect.MakeSlice(val.Type(), val.Len(), newcap)
			reflect.Copy(newv, val)
			val.Set(newv)
		}
		if i >= val.Len() {
			val.SetLen(i + 1)
		}
		// decode into element
		if err := elemdec(s, val.Index(i)); err == EOL {
			break
		} else if err != nil {
			return addErrorContext(err, fmt.Sprint("[", i, "]"))
		}
	}
	if i < val.Len() {
		val.SetLen(i)
	}
	return s.ListEnd()
}

func decodeListArray(s *Stream, val reflect.Value, elemdec decoder) error {
	_, err := s.List()
	if err != nil {
		return wrapStreamError(err, val.Type())
	}
	vlen := val.Len()
	i := 0
	for ; i < vlen; i++ {
		if err := elemdec(s, val.Index(i)); err == EOL {
			break
		} else if err != nil {
			return addErrorContext(err, fmt.Sprint("[", i, "]"))
		}
	}
	if i < vlen {
		return &decodeError{msg: "input list has too few elements", typ: val.Type()}
	}
	return wrapStreamError(s.ListEnd(), val.Type())
}

func decodeByteSlice(s *Stream, val reflect.Value) error {
	b, err := s.Bytes()
	if err != nil {
		return wrapStreamError(err, val.Type())
	}
	val.SetBytes(b)
	return nil
}

func decodeByteArray(s *Stream, val reflect.Value) error {
	kind, size, err := s.Kind()
	if err != nil {
		return err
	}
	vlen := val.Len()
	switch kind {
	case Byte:
		if vlen == 0 {
			return &decodeError{msg: "input string too long", typ: val.Type()}
		}
		if vlen > 1 {
			return &decodeError{msg: "input string too short", typ: val.Type()}
		}
		bv, _ := s.Uint()
		val.Index(0).SetUint(bv)
	case String:
		if uint64(vlen) < size {
			return &decodeError{msg: "input string too long", typ: val.Type()}
		}
		if uint64(vlen) > size {
			return &decodeError{msg: "input string too short", typ: val.Type()}
		}
		slice := val.Slice(0, vlen).Interface().([]byte)
		if err := s.readFull(slice); err != nil {
			return err
		}
		// Reject cases where single byte encoding should have been used.
		if size == 1 && slice[0] < 56 {
			return wrapStreamError(ErrCanonSize, val.Type())
		}
	case List:
		return wrapStreamError(ErrExpectedString, val.Type())
	}
	return nil
}

func makeStructDecoder(typ reflect.Type) (decoder, error) {
	fields, err := structFields(typ)
	if err != nil {
		return nil, err
	}
	dec := func(s *Stream, val reflect.Value) (err error) {
		if _, err = s.List(); err != nil {
			return wrapStreamError(err, typ)
		}
		for _, f := range fields {
			err = f.info.decoder(s, val.Field(f.index))
			if err == EOL {
				return &decodeError{msg: "too few elements", typ: typ}
			} else if err != nil {
				return addErrorContext(err, "."+typ.Field(f.index).Name)
			}
		}
		return wrapStreamError(s.ListEnd(), typ)
	}
	return dec, nil
}

// makePtrDecoder creates a decoder that decodes into
// the pointer's element type.
func makePtrDecoder(typ reflect.Type) (decoder, error) {
	etype := typ.Elem()
	etypeinfo, err := cachedTypeInfo1(etype, tags{})
	if err != nil {
		return nil, err
	}
	dec := func(s *Stream, val reflect.Value) (err error) {
		newval := val
		if val.IsNil() {
			newval = reflect.New(etype)
		}
		if err = etypeinfo.decoder(s, newval.Elem()); err == nil {
			val.Set(newval)
		}
		return err
	}
	return dec, nil
}

// makeOptionalPtrDecoder creates a decoder that decodes empty values
// as nil. Non-empty values are decoded into a value of the element type,
// just like makePtrDecoder does.
//
// This decoder is used for pointer-typed struct fields with struct tag "nil".
func makeOptionalPtrDecoder(typ reflect.Type) (decoder, error) {
	etype := typ.Elem()
	etypeinfo, err := cachedTypeInfo1(etype, tags{})
	if err != nil {
		return nil, err
	}
	dec := func(s *Stream, val reflect.Value) (err error) {
		kind, size, err := s.Kind()
		if err != nil || size == 0 && kind != Byte {
			// rearm s.Kind. This is important because the input
			// position must advance to the next value even though
			// we don't read anything.
			s.kind = -1
			// set the pointer to nil.
			val.Set(reflect.Zero(typ))
			return err
		}
		newval := val
		if val.IsNil() {
			newval = reflect.New(etype)
		}
		if err = etypeinfo.decoder(s, newval.Elem()); err == nil {
			val.Set(newval)
		}
		return err
	}
	return dec, nil
}

var ifsliceType = reflect.TypeOf([]interface{}{})

func decodeInterface(s *Stream, val reflect.Value) error {
	if val.Type().NumMethod() != 0 {
		return fmt.Errorf("rlp: type %v is not RLP-serializable", val.Type())
	}
	kind, _, err := s.Kind()
	if err != nil {
		return err
	}
	if kind == List {
		slice := reflect.New(ifsliceType).Elem()
		if err := decodeListSlice(s, slice, decodeInterface); err != nil {
			return err
		}
		val.Set(slice)
	} else {
		b, err := s.Bytes()
		if err != nil {
			return err
		}
		val.Set(reflect.ValueOf(b))
	}
	return nil
}

// This decoder is used for non-pointer values of types
// that implement the Decoder interface using a pointer receiver.
func decodeDecoderNoPtr(s *Stream, val reflect.Value) error {
	return val.Addr().Interface().(Decoder).DecodeRLP(s)
}

func decodeDecoder(s *Stream, val reflect.Value) error {
	// Decoder instances are not handled using the pointer rule if the type
	// implements Decoder with pointer receiver (i.e. always)
	// because it might handle empty values specially.
	// We need to allocate one here in this case, like makePtrDecoder does.
	if val.Kind() == reflect.Ptr && val.IsNil() {
		val.Set(reflect.New(val.Type().Elem()))
	}
	return val.Interface().(Decoder).DecodeRLP(s)
}

// Kind represents the kind of value contained in an RLP stream.
type Kind int

const (
	Byte Kind = iota
	String
	List
)

func (k Kind) String() string {
	switch k {
	case Byte:
		return "Byte"
	case String:
		return "String"
	case List:
		return "List"
	default:
		return fmt.Sprintf("Unknown(%d)", k)
	}
}

var (
	// EOL is returned when the end of the current list
	// has been reached during streaming.
	EOL = errors.New("rlp: end of list")

	// Actual Errors
	ErrExpectedString = errors.New("rlp: expected String or Byte")
	ErrExpectedList   = errors.New("rlp: expected List")
	ErrCanonInt       = errors.New("rlp: non-canonical integer format")
	ErrCanonSize      = errors.New("rlp: non-canonical size information")
	ErrElemTooLarge   = errors.New("rlp: element is larger than containing list")
	ErrValueTooLarge  = errors.New("rlp: value size exceeds available input length")

	// This error is reported by DecodeBytes if the slice contains
	// additional data after the first RLP value.
	ErrMoreThanOneValue = errors.New("rlp: input contains more than one value")

	// internal errors
	errNotInList    = errors.New("rlp: call of ListEnd outside of any list")
	errNotAtEOL     = errors.New("rlp: call of ListEnd not positioned at EOL")
	errUintOverflow = errors.New("rlp: uint overflow")
)

// ByteReader must be implemented by any input reader for a Stream. It
// is implemented by e.g. bufio.Reader and bytes.Reader.
type ByteReader interface {
	io.Reader
	io.ByteReader
}

// Stream can be used for piecemeal decoding of an input stream. This
// is useful if the input is very large or if the decoding rules for a
// type depend on the input structure. Stream does not keep an
// internal buffer. After decoding a value, the input reader will be
// positioned just before the type information for the next value.
//
// When decoding a list and the input position reaches the declared
// length of the list, all operations will return error EOL.
// The end of the list must be acknowledged using ListEnd to continue
// reading the enclosing list.
//
// Stream is not safe for concurrent use.
type Stream struct {
	r ByteReader

	// number of bytes remaining to be read from r.
	remaining uint64
	limited   bool

	// auxiliary buffer for integer decoding
	uintbuf []byte

	kind    Kind   // kind of value ahead
	size    uint64 // size of value ahead
	byteval byte   // value of single byte in type tag
	kinderr error  // error from last readKind
	stack   []listpos
}

type listpos struct{ pos, size uint64 }

// NewStream creates a new decoding stream reading from r.
//
// If r implements the ByteReader interface, Stream will
// not introduce any buffering.
//
// For non-toplevel values, Stream returns ErrElemTooLarge
// for values that do not fit into the enclosing list.
//
// Stream supports an optional input limit. If a limit is set, the
// size of any toplevel value will be checked against the remaining
// input length. Stream operations that encounter a value exceeding
// the remaining input length will return ErrValueTooLarge. The limit
// can be set by passing a non-zero value for inputLimit.
//
// If r is a bytes.Reader or strings.Reader, the input limit is set to
// the length of r's underlying data unless an explicit limit is
// provided.
func NewStream(r io.Reader, inputLimit uint64) *Stream {
	s := new(Stream)
	s.Reset(r, inputLimit)
	return s
}

// NewListStream creates a new stream that pretends to be positioned
// at an encoded list of the given length.
func NewListStream(r io.Reader, len uint64) *Stream {
	s := new(Stream)
	s.Reset(r, len)
	s.kind = List
	s.size = len
	return s
}

// Bytes reads an RLP string and returns its contents as a byte slice.
// If the input does not contain an RLP string, the returned
// error will be ErrExpectedString.
func (s *Stream) Bytes() ([]byte, error) {
	kind, size, err := s.Kind()
	if err != nil {
		return nil, err
	}
	switch kind {
	case Byte:
		s.kind = -1 // rearm Kind
		return []byte{s.byteval}, nil
	case String:
		b := make([]byte, size)
		if err = s.readFull(b); err != nil {
			return nil, err
		}
		if size == 1 && b[0] < 56 {
			return nil, ErrCanonSize
		}
		return b, nil
	default:
		return nil, ErrExpectedString
	}
}

// Raw reads a raw encoded value including RLP type information.
func (s *Stream) Raw() ([]byte, error) {
	kind, size, err := s.Kind()
	if err != nil {
		return nil, err
	}
	if kind == Byte {
		s.kind = -1 // rearm Kind
		return []byte{s.byteval}, nil
	}
	// the original header has already been read and is no longer
	// available. read content and put a new header in front of it.
	start := headsize(size)
	buf := make([]byte, uint64(start)+size)
	if err := s.readFull(buf[start:]); err != nil {
		return nil, err
	}
	if kind == String {
		puthead(buf, 0x80, 0xB8, size)
	} else {
		puthead(buf, 0xC0, 0xF7, size)
	}
	return buf, nil
}

// Uint reads an RLP string of up to 8 bytes and returns its contents
// as an unsigned integer. If the input does not contain an RLP string, the
// returned error will be ErrExpectedString.
func (s *Stream) Uint() (uint64, error) {
	return s.uint(64)
}

func (s *Stream) uint(maxbits int) (uint64, error) {
	kind, size, err := s.Kind()
	if err != nil {
		return 0, err
	}
	switch kind {
	case Byte:
		if s.byteval == 0 {
			return 0, ErrCanonInt
		}
		s.kind = -1 // rearm Kind
		return uint64(s.byteval), nil
	case String:
		if size > uint64(maxbits/8) {
			return 0, errUintOverflow
		}
		v, err := s.readUint(byte(size))
		switch {
		case err == ErrCanonSize:
			// Adjust error because we're not reading a size right now.
			return 0, ErrCanonInt
		case err != nil:
			return 0, err
		case size > 0 && v < 56:
			return 0, ErrCanonSize
		default:
			return v, nil
		}
	default:
		return 0, ErrExpectedString
	}
}

// List starts decoding an RLP list. If the input does not contain a
// list, the returned error will be ErrExpectedList. When the list's
// end has been reached, any Stream operation will return EOL.
func (s *Stream) List() (size uint64, err error) {
	kind, size, err := s.Kind()
	if err != nil {
		return 0, err
	}
	if kind != List {
		return 0, ErrExpectedList
	}
	s.stack = append(s.stack, listpos{0, size})
	s.kind = -1
	s.size = 0
	return size, nil
}

// ListEnd returns to the enclosing list.
// The input reader must be positioned at the end of a list.
func (s *Stream) ListEnd() error {
	if len(s.stack) == 0 {
		return errNotInList
	}
	tos := s.stack[len(s.stack)-1]
	if tos.pos != tos.size {
		return errNotAtEOL
	}
	s.stack = s.stack[:len(s.stack)-1] // pop
	if len(s.stack) > 0 {
		s.stack[len(s.stack)-1].pos += tos.size
	}
	s.kind = -1
	s.size = 0
	return nil
}

// Decode decodes a value and stores the result in the value pointed
// to by val. Please see the documentation for the Decode function
// to learn about the decoding rules.
func (s *Stream) Decode(val interface{}) error {
	if val == nil {
		return errDecodeIntoNil
	}
	rval := reflect.ValueOf(val)
	rtyp := rval.Type()
	if rtyp.Kind() != reflect.Ptr {
		return errNoPointer
	}
	if rval.IsNil() {
		return errDecodeIntoNil
	}
	info, err := cachedTypeInfo(rtyp.Elem(), tags{})
	if err != nil {
		return err
	}

	err = info.decoder(s, rval.Elem())
	if decErr, ok := err.(*decodeError); ok && len(decErr.ctx) > 0 {
		// add decode target type to error so context has more meaning
		decErr.ctx = append(decErr.ctx, fmt.Sprint("(", rtyp.Elem(), ")"))
	}
	return err
}

// Reset discards any information about the current decoding context
// and starts reading from r. This method is meant to facilitate reuse
// of a preallocated Stream across many decoding operations.
//
// If r does not also implement ByteReader, Stream will do its own
// buffering.
func (s *Stream) Reset(r io.Reader, inputLimit uint64) {
	if inputLimit > 0 {
		s.remaining = inputLimit
		s.limited = true
	} else {
		// Attempt to automatically discover
		// the limit when reading from a byte slice.
		switch br := r.(type) {
		case *bytes.Reader:
			s.remaining = uint64(br.Len())
			s.limited = true
		case *strings.Reader:
			s.remaining = uint64(br.Len())
			s.limited = true
		default:
			s.limited = false
		}
	}
	// Wrap r with a buffer if it doesn't have one.
	bufr, ok := r.(ByteReader)
	if !ok {
		bufr = bufio.NewReader(r)
	}
	s.r = bufr
	// Reset the decoding context.
	s.stack = s.stack[:0]
	s.size = 0
	s.kind = -1
	s.kinderr = nil
	if s.uintbuf == nil {
		s.uintbuf = make([]byte, 8)
	}
}

// Kind returns the kind and size of the next value in the
// input stream.
//
// The returned size is the number of bytes that make up the value.
// For kind == Byte, the size is zero because the value is
// contained in the type tag.
//
// The first call to Kind will read size information from the input
// reader and leave it positioned at the start of the actual bytes of
// the value. Subsequent calls to Kind (until the value is decoded)
// will not advance the input reader and return cached information.
func (s *Stream) Kind() (kind Kind, size uint64, err error) {
	var tos *listpos
	if len(s.stack) > 0 {
		tos = &s.stack[len(s.stack)-1]
	}
	if s.kind < 0 {
		s.kinderr = nil
		// Don't read further if we're at the end of the
		// innermost list.
		if tos != nil && tos.pos == tos.size {
			return 0, 0, EOL
		}
		s.kind, s.size, s.kinderr = s.readKind()
		if s.kinderr == nil {
			if tos == nil {
				// At toplevel, check that the value is smaller
				// than the remaining input length.
				if s.limited && s.size > s.remaining {
					s.kinderr = ErrValueTooLarge
				}
			} else {
				// Inside a list, check that the value doesn't overflow the list.
				if s.size > tos.size-tos.pos {
					s.kinderr = ErrElemTooLarge
				}
			}
		}
	}
	// Note: this might return a sticky error generated
	// by an earlier call to readKind.
	return s.kind, s.size, s.kinderr
}

func (s *Stream) readKind() (kind Kind, size uint64, err error) {
	b, err := s.readByte()
	if err != nil {
		if len(s.stack) == 0 {
			// At toplevel, Adjust the error to actual EOF. io.EOF is
			// used by callers to determine when to stop decoding.
			switch err {
			case io.ErrUnexpectedEOF:
				err = io.EOF
			case ErrValueTooLarge:
				err = io.EOF
			}
		}
		return 0, 0, err
	}
	s.byteval = 0
	switch {
	case b < 0x80:
		// For a single byte whose value is in the [0x00, 0x7F] range, that byte
		// is its own RLP encoding.
		s.byteval = b
		return Byte, 0, nil
	case b < 0xB8:
		// Otherwise, if a string is 0-55 bytes long,
		// the RLP encoding consists of a single byte with value 0x80 plus the
		// length of the string followed by the string. The range of the first
		// byte is thus [0x80, 0xB7].
		return String, uint64(b - 0x80), nil
	case b < 0xC0:
		// If a string is more than 55 bytes long, the
		// RLP encoding consists of a single byte with value 0xB7 plus the length
		// of the length of the string in binary form, followed by the length of
		// the string, followed by the string. For example, a length-1024 string
		// would be encoded as 0xB90400 followed by the string. The range of
		// the first byte is thus [0xB8, 0xBF].
		size, err = s.readUint(b - 0xB7)
		if err == nil && size < 56 {
			err = ErrCanonSize
		}
		return String, size, err
	case b < 0xF8:
		// If the total payload of a list
		// (i.e. the combined length of all its items) is 0-55 bytes long, the
		// RLP encoding consists of a single byte with value 0xC0 plus the length
		// of the list followed by the concatenation of the RLP encodings of the
		// items. The range of the first byte is thus [0xC0, 0xF7].
		return List, uint64(b - 0xC0), nil
	default:
		// If the total payload of a list is more than 55 bytes long,
		// the RLP encoding consists of a single byte with value 0xF7
		// plus the length of the length of the payload in binary
		// form, followed by the length of the payload, followed by
		// the concatenation of the RLP encodings of the items. The
		// range of the first byte is thus [0xF8, 0xFF].
		size, err = s.readUint(b - 0xF7)
		if err == nil && size < 56 {
			err = ErrCanonSize
		}
		return List, size, err
	}
}

func (s *Stream) readUint(size byte) (uint64, error) {
	switch size {
	case 0:
		s.kind = -1 // rearm Kind
		return 0, nil
	case 1:
		b, err := s.readByte()
		return uint64(b), err
	default:
		start := int(8 - size)
		for i := 0; i < start; i++ {
			s.uintbuf[i] = 0
		}
		if err := s.readFull(s.uintbuf[start:]); err != nil {
			return 0, err
		}
		if s.uintbuf[start] == 0 {
			// Note: readUint is also used to decode integer
			// values. The error needs to be adjusted to become
			// ErrCanonInt in this case.
			return 0, ErrCanonSize
		}
		return binary.BigEndian.Uint64(s.uintbuf), nil
	}
}

func (s *Stream) readFull(buf []byte) (err error) {
	if err := s.willRead(uint64(len(buf))); err != nil {
		return err
	}
	var nn, n int
	for n < len(buf) && err == nil {
		nn, err = s.r.Read(buf[n:])
		n += nn
	}
	if err == io.EOF {
		err = io.ErrUnexpectedEOF
	}
	return err
}

func (s *Stream) readByte() (byte, error) {
	if err := s.willRead(1); err != nil {
		return 0, err
	}
	b, err := s.r.ReadByte()
	if err == io.EOF {
		err = io.ErrUnexpectedEOF
	}
	return b, err
}

func (s *Stream) willRead(n uint64) error {
	s.kind = -1 // rearm Kind

	if len(s.stack) > 0 {
		// check list overflow
		tos := s.stack[len(s.stack)-1]
		if n > tos.size-tos.pos {
			return ErrElemTooLarge
		}
		s.stack[len(s.stack)-1].pos += n
	}
	if s.limited {
		if n > s.remaining {
			return ErrValueTooLarge
		}
		s.remaining -= n
	}
	return nil
}
