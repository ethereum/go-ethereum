package rlp

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"math/big"
	"reflect"
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

// Decode parses RLP-encoded data from r and stores the result
// in the value pointed to by val. Val must be a non-nil pointer.
//
// Decode uses the following type-dependent decoding rules:
//
// If the type implements the Decoder interface, decode calls
// DecodeRLP.
//
// To decode into a pointer, Decode will set the pointer to nil if the
// input has size zero or the input is a single byte with value zero.
// If the input has nonzero size, Decode will allocate a new value of
// the type being pointed to.
//
// To decode into a struct, Decode expects the input to be an RLP
// list. The decoded elements of the list are assigned to each public
// field in the order given by the struct's definition. If the input
// list has too few elements, no error is returned and the remaining
// fields will have the zero value.
// Recursive struct types are supported.
//
// To decode into a slice, the input must be a list and the resulting
// slice will contain the input elements in order.
// As a special case, if the slice has a byte-size element type, the input
// can also be an RLP string.
//
// To decode into a Go string, the input must be an RLP string. The
// bytes are taken as-is and will not necessarily be valid UTF-8.
//
// To decode into an integer type, the input must also be an RLP
// string. The bytes are interpreted as a big endian representation of
// the integer. If the RLP string is larger than the bit size of the
// type, Decode will return an error. Decode also supports *big.Int.
// There is no size limit for big integers.
//
// To decode into an interface value, Decode stores one of these
// in the value:
//
//	[]interface{}, for RLP lists
//	[]byte, for RLP strings
//
// Non-empty interface types are not supported, nor are bool, float32,
// float64, maps, channel types and functions.
func Decode(r ByteReader, val interface{}) error {
	return NewStream(r).Decode(val)
}

func makeNumDecoder(typ reflect.Type) decoder {
	kind := typ.Kind()
	switch {
	case kind <= reflect.Int64:
		return decodeInt
	case kind <= reflect.Uint64:
		return decodeUint
	default:
		panic("fallthrough")
	}
}

func decodeInt(s *Stream, val reflect.Value) error {
	num, err := s.uint(val.Type().Bits())
	if err != nil {
		return err
	}
	val.SetInt(int64(num))
	return nil
}

func decodeUint(s *Stream, val reflect.Value) error {
	num, err := s.uint(val.Type().Bits())
	if err != nil {
		return err
	}
	val.SetUint(num)
	return nil
}

func decodeString(s *Stream, val reflect.Value) error {
	b, err := s.Bytes()
	if err != nil {
		return err
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
		return err
	}
	i := val.Interface().(*big.Int)
	if i == nil {
		i = new(big.Int)
		val.Set(reflect.ValueOf(i))
	}
	i.SetBytes(b)
	return nil
}

const maxInt = int(^uint(0) >> 1)

func makeListDecoder(typ reflect.Type) (decoder, error) {
	etype := typ.Elem()
	if etype.Kind() == reflect.Uint8 && !reflect.PtrTo(etype).Implements(decoderInterface) {
		if typ.Kind() == reflect.Array {
			return decodeByteArray, nil
		} else {
			return decodeByteSlice, nil
		}
	}
	etypeinfo, err := cachedTypeInfo1(etype)
	if err != nil {
		return nil, err
	}
	var maxLen = maxInt
	if typ.Kind() == reflect.Array {
		maxLen = typ.Len()
	}
	dec := func(s *Stream, val reflect.Value) error {
		return decodeList(s, val, etypeinfo.decoder, maxLen)
	}
	return dec, nil
}

// decodeList decodes RLP list elements into slices and arrays.
//
// The approach here is stolen from package json, although we differ
// in the semantics for arrays. package json discards remaining
// elements that would not fit into the array. We generate an error in
// this case because we'd be losing information.
func decodeList(s *Stream, val reflect.Value, elemdec decoder, maxelem int) error {
	size, err := s.List()
	if err != nil {
		return err
	}
	if size == 0 {
		if val.Kind() == reflect.Slice {
			val.Set(reflect.MakeSlice(val.Type(), 0, 0))
		} else {
			zero(val, 0)
		}
		return s.ListEnd()
	}

	i := 0
	for {
		if i > maxelem {
			return fmt.Errorf("rlp: input List has more than %d elements", maxelem)
		}
		if val.Kind() == reflect.Slice {
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
		}
		// decode into element
		if err := elemdec(s, val.Index(i)); err == EOL {
			break
		} else if err != nil {
			return err
		}
		i++
	}
	if i < val.Len() {
		if val.Kind() == reflect.Array {
			// zero the rest of the array.
			zero(val, i)
		} else {
			val.SetLen(i)
		}
	}
	return s.ListEnd()
}

func decodeByteSlice(s *Stream, val reflect.Value) error {
	kind, _, err := s.Kind()
	if err != nil {
		return err
	}
	if kind == List {
		return decodeList(s, val, decodeUint, maxInt)
	}
	b, err := s.Bytes()
	if err == nil {
		val.SetBytes(b)
	}
	return err
}

var errStringDoesntFitArray = errors.New("rlp: string value doesn't fit into target array")

func decodeByteArray(s *Stream, val reflect.Value) error {
	kind, size, err := s.Kind()
	if err != nil {
		return err
	}
	switch kind {
	case Byte:
		if val.Len() == 0 {
			return errStringDoesntFitArray
		}
		bv, _ := s.Uint()
		val.Index(0).SetUint(bv)
		zero(val, 1)
	case String:
		if uint64(val.Len()) < size {
			return errStringDoesntFitArray
		}
		slice := val.Slice(0, int(size)).Interface().([]byte)
		if err := s.readFull(slice); err != nil {
			return err
		}
		zero(val, int(size))
	case List:
		return decodeList(s, val, decodeUint, val.Len())
	}
	return nil
}

func zero(val reflect.Value, start int) {
	z := reflect.Zero(val.Type().Elem())
	for i := start; i < val.Len(); i++ {
		val.Index(i).Set(z)
	}
}

type field struct {
	index int
	info  *typeinfo
}

func makeStructDecoder(typ reflect.Type) (decoder, error) {
	var fields []field
	for i := 0; i < typ.NumField(); i++ {
		if f := typ.Field(i); f.PkgPath == "" { // exported
			info, err := cachedTypeInfo1(f.Type)
			if err != nil {
				return nil, err
			}
			fields = append(fields, field{i, info})
		}
	}
	dec := func(s *Stream, val reflect.Value) (err error) {
		if _, err = s.List(); err != nil {
			return err
		}
		for _, f := range fields {
			err = f.info.decoder(s, val.Field(f.index))
			if err == EOL {
				// too few elements. leave the rest at their zero value.
				break
			} else if err != nil {
				return err
			}
		}
		if err = s.ListEnd(); err == errNotAtEOL {
			err = errors.New("rlp: input List has too many elements")
		}
		return err
	}
	return dec, nil
}

func makePtrDecoder(typ reflect.Type) (decoder, error) {
	etype := typ.Elem()
	etypeinfo, err := cachedTypeInfo1(etype)
	if err != nil {
		return nil, err
	}
	dec := func(s *Stream, val reflect.Value) (err error) {
		_, size, err := s.Kind()
		if err != nil || size == 0 && s.byteval == 0 {
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
	kind, _, err := s.Kind()
	if err != nil {
		return err
	}
	if kind == List {
		slice := reflect.New(ifsliceType).Elem()
		if err := decodeList(s, slice, decodeInterface, maxInt); err != nil {
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

	// Other errors
	ErrExpectedString = errors.New("rlp: expected String or Byte")
	ErrExpectedList   = errors.New("rlp: expected List")
	ErrElemTooLarge   = errors.New("rlp: element is larger than containing list")

	// internal errors
	errNotInList = errors.New("rlp: call of ListEnd outside of any list")
	errNotAtEOL  = errors.New("rlp: call of ListEnd not positioned at EOL")
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
	r       ByteReader
	uintbuf []byte

	kind    Kind   // kind of value ahead
	size    uint64 // size of value ahead
	byteval byte   // value of single byte in type tag
	stack   []listpos
}

type listpos struct{ pos, size uint64 }

func NewStream(r ByteReader) *Stream {
	return &Stream{r: r, uintbuf: make([]byte, 8), kind: -1}
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
		return b, nil
	default:
		return nil, ErrExpectedString
	}
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
		s.kind = -1 // rearm Kind
		return uint64(s.byteval), nil
	case String:
		if size > uint64(maxbits/8) {
			return 0, fmt.Errorf("rlp: string is larger than %d bits", maxbits)
		}
		return s.readUint(byte(size))
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
	info, err := cachedTypeInfo(rtyp.Elem())
	if err != nil {
		return err
	}
	return info.decoder(s, rval.Elem())
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
		if tos != nil && tos.pos == tos.size {
			return 0, 0, EOL
		}
		kind, size, err = s.readKind()
		if err != nil {
			return 0, 0, err
		}
		s.kind, s.size = kind, size
	}
	if tos != nil && tos.pos+s.size > tos.size {
		return 0, 0, ErrElemTooLarge
	}
	return s.kind, s.size, nil
}

func (s *Stream) readKind() (kind Kind, size uint64, err error) {
	b, err := s.readByte()
	if err != nil {
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
		return List, size, err
	}
}

func (s *Stream) readUint(size byte) (uint64, error) {
	if size == 1 {
		b, err := s.readByte()
		if err == io.EOF {
			err = io.ErrUnexpectedEOF
		}
		return uint64(b), err
	}
	start := int(8 - size)
	for i := 0; i < start; i++ {
		s.uintbuf[i] = 0
	}
	err := s.readFull(s.uintbuf[start:])
	return binary.BigEndian.Uint64(s.uintbuf), err
}

func (s *Stream) readFull(buf []byte) (err error) {
	s.willRead(uint64(len(buf)))
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
	s.willRead(1)
	b, err := s.r.ReadByte()
	if len(s.stack) > 0 && err == io.EOF {
		err = io.ErrUnexpectedEOF
	}
	return b, err
}

func (s *Stream) willRead(n uint64) {
	s.kind = -1 // rearm Kind
	if len(s.stack) > 0 {
		s.stack[len(s.stack)-1].pos += n
	}
}
