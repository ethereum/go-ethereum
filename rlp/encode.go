// Copyright 2014 The go-ethereum Authors
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
	"math/big"
	"reflect"
	"sync"
)

var (
	// Common encoded values.
	// These are useful when implementing EncodeRLP.
	EmptyString = []byte{0x80}
	EmptyList   = []byte{0xC0}
)

// Encoder is implemented by types that require custom
// encoding rules or want to encode private fields.
type Encoder interface {
	// EncodeRLP should write the RLP encoding of its receiver to w.
	// If the implementation is a pointer method, it may also be
	// called for nil pointers.
	//
	// Implementations should generate valid RLP. The data written is
	// not verified at the moment, but a future version might. It is
	// recommended to write only a single value but writing multiple
	// values or no value at all is also permitted.
	EncodeRLP(io.Writer) error
}

// Encode writes the RLP encoding of val to w. Note that Encode may
// perform many small writes in some cases. Consider making w
// buffered.
//
// Please see package-level documentation of encoding rules.
func Encode(w io.Writer, val interface{}) error {
	if outer, ok := w.(*encbuf); ok {
		// Encode was called by some type's EncodeRLP.
		// Avoid copying by writing to the outer encbuf directly.
		return outer.encode(val)
	}
	eb := encbufPool.Get().(*encbuf)
	defer encbufPool.Put(eb)
	eb.reset()
	if err := eb.encode(val); err != nil {
		return err
	}
	return eb.toWriter(w)
}

// EncodeToBytes returns the RLP encoding of val.
// Please see package-level documentation for the encoding rules.
func EncodeToBytes(val interface{}) ([]byte, error) {
	eb := encbufPool.Get().(*encbuf)
	defer encbufPool.Put(eb)
	eb.reset()
	if err := eb.encode(val); err != nil {
		return nil, err
	}
	return eb.toBytes(), nil
}

// EncodeToReader returns a reader from which the RLP encoding of val
// can be read. The returned size is the total size of the encoded
// data.
//
// Please see the documentation of Encode for the encoding rules.
func EncodeToReader(val interface{}) (size int, r io.Reader, err error) {
	eb := encbufPool.Get().(*encbuf)
	eb.reset()
	if err := eb.encode(val); err != nil {
		return 0, nil, err
	}
	return eb.size(), &encReader{buf: eb}, nil
}

type listhead struct {
	offset int // index of this header in string data
	size   int // total size of encoded data (including list headers)
}

// encode writes head to the given buffer, which must be at least
// 9 bytes long. It returns the encoded bytes.
func (head *listhead) encode(buf []byte) []byte {
	return buf[:puthead(buf, 0xC0, 0xF7, uint64(head.size))]
}

// headsize returns the size of a list or string header
// for a value of the given size.
func headsize(size uint64) int {
	if size < 56 {
		return 1
	}
	return 1 + intsize(size)
}

// puthead writes a list or string header to buf.
// buf must be at least 9 bytes long.
func puthead(buf []byte, smalltag, largetag byte, size uint64) int {
	if size < 56 {
		buf[0] = smalltag + byte(size)
		return 1
	}
	sizesize := putint(buf[1:], size)
	buf[0] = largetag + byte(sizesize)
	return sizesize + 1
}

type encbuf struct {
	str     []byte     // string data, contains everything except list headers
	lheads  []listhead // all list headers
	lhsize  int        // sum of sizes of all encoded list headers
	sizebuf [9]byte    // auxiliary buffer for uint encoding
}

// encbufs are pooled.
var encbufPool = sync.Pool{
	New: func() interface{} { return new(encbuf) },
}

func (w *encbuf) reset() {
	w.lhsize = 0
	w.str = w.str[:0]
	w.lheads = w.lheads[:0]
}

// encbuf implements io.Writer so it can be passed it into EncodeRLP.
func (w *encbuf) Write(b []byte) (int, error) {
	w.str = append(w.str, b...)
	return len(b), nil
}

func (w *encbuf) encode(val interface{}) error {
	rval := reflect.ValueOf(val)
	writer, err := cachedWriter(rval.Type())
	if err != nil {
		return err
	}
	return writer(rval, w)
}

func (w *encbuf) encodeStringHeader(size int) {
	if size < 56 {
		w.str = append(w.str, 0x80+byte(size))
	} else {
		sizesize := putint(w.sizebuf[1:], uint64(size))
		w.sizebuf[0] = 0xB7 + byte(sizesize)
		w.str = append(w.str, w.sizebuf[:sizesize+1]...)
	}
}

func (w *encbuf) encodeString(b []byte) {
	if len(b) == 1 && b[0] <= 0x7F {
		// fits single byte, no string header
		w.str = append(w.str, b[0])
	} else {
		w.encodeStringHeader(len(b))
		w.str = append(w.str, b...)
	}
}

func (w *encbuf) encodeUint(i uint64) {
	if i == 0 {
		w.str = append(w.str, 0x80)
	} else if i < 128 {
		// fits single byte
		w.str = append(w.str, byte(i))
	} else {
		s := putint(w.sizebuf[1:], i)
		w.sizebuf[0] = 0x80 + byte(s)
		w.str = append(w.str, w.sizebuf[:s+1]...)
	}
}

// list adds a new list header to the header stack. It returns the index
// of the header. The caller must call listEnd with this index after encoding
// the content of the list.
func (w *encbuf) list() int {
	w.lheads = append(w.lheads, listhead{offset: len(w.str), size: w.lhsize})
	return len(w.lheads) - 1
}

func (w *encbuf) listEnd(index int) {
	lh := &w.lheads[index]
	lh.size = w.size() - lh.offset - lh.size
	if lh.size < 56 {
		w.lhsize++ // length encoded into kind tag
	} else {
		w.lhsize += 1 + intsize(uint64(lh.size))
	}
}

func (w *encbuf) size() int {
	return len(w.str) + w.lhsize
}

func (w *encbuf) toBytes() []byte {
	out := make([]byte, w.size())
	strpos := 0
	pos := 0
	for _, head := range w.lheads {
		// write string data before header
		n := copy(out[pos:], w.str[strpos:head.offset])
		pos += n
		strpos += n
		// write the header
		enc := head.encode(out[pos:])
		pos += len(enc)
	}
	// copy string data after the last list header
	copy(out[pos:], w.str[strpos:])
	return out
}

func (w *encbuf) toWriter(out io.Writer) (err error) {
	strpos := 0
	for _, head := range w.lheads {
		// write string data before header
		if head.offset-strpos > 0 {
			n, err := out.Write(w.str[strpos:head.offset])
			strpos += n
			if err != nil {
				return err
			}
		}
		// write the header
		enc := head.encode(w.sizebuf[:])
		if _, err = out.Write(enc); err != nil {
			return err
		}
	}
	if strpos < len(w.str) {
		// write string data after the last list header
		_, err = out.Write(w.str[strpos:])
	}
	return err
}

// encReader is the io.Reader returned by EncodeToReader.
// It releases its encbuf at EOF.
type encReader struct {
	buf    *encbuf // the buffer we're reading from. this is nil when we're at EOF.
	lhpos  int     // index of list header that we're reading
	strpos int     // current position in string buffer
	piece  []byte  // next piece to be read
}

func (r *encReader) Read(b []byte) (n int, err error) {
	for {
		if r.piece = r.next(); r.piece == nil {
			// Put the encode buffer back into the pool at EOF when it
			// is first encountered. Subsequent calls still return EOF
			// as the error but the buffer is no longer valid.
			if r.buf != nil {
				encbufPool.Put(r.buf)
				r.buf = nil
			}
			return n, io.EOF
		}
		nn := copy(b[n:], r.piece)
		n += nn
		if nn < len(r.piece) {
			// piece didn't fit, see you next time.
			r.piece = r.piece[nn:]
			return n, nil
		}
		r.piece = nil
	}
}

// next returns the next piece of data to be read.
// it returns nil at EOF.
func (r *encReader) next() []byte {
	switch {
	case r.buf == nil:
		return nil

	case r.piece != nil:
		// There is still data available for reading.
		return r.piece

	case r.lhpos < len(r.buf.lheads):
		// We're before the last list header.
		head := r.buf.lheads[r.lhpos]
		sizebefore := head.offset - r.strpos
		if sizebefore > 0 {
			// String data before header.
			p := r.buf.str[r.strpos:head.offset]
			r.strpos += sizebefore
			return p
		}
		r.lhpos++
		return head.encode(r.buf.sizebuf[:])

	case r.strpos < len(r.buf.str):
		// String data at the end, after all list headers.
		p := r.buf.str[r.strpos:]
		r.strpos = len(r.buf.str)
		return p

	default:
		return nil
	}
}

var encoderInterface = reflect.TypeOf(new(Encoder)).Elem()

// makeWriter creates a writer function for the given type.
func makeWriter(typ reflect.Type, ts tags) (writer, error) {
	kind := typ.Kind()
	switch {
	case typ == rawValueType:
		return writeRawValue, nil
	case typ.AssignableTo(reflect.PtrTo(bigInt)):
		return writeBigIntPtr, nil
	case typ.AssignableTo(bigInt):
		return writeBigIntNoPtr, nil
	case kind == reflect.Ptr:
		return makePtrWriter(typ, ts)
	case reflect.PtrTo(typ).Implements(encoderInterface):
		return makeEncoderWriter(typ), nil
	case isUint(kind):
		return writeUint, nil
	case kind == reflect.Bool:
		return writeBool, nil
	case kind == reflect.String:
		return writeString, nil
	case kind == reflect.Slice && isByte(typ.Elem()):
		return writeBytes, nil
	case kind == reflect.Array && isByte(typ.Elem()):
		return makeByteArrayWriter(typ), nil
	case kind == reflect.Slice || kind == reflect.Array:
		return makeSliceWriter(typ, ts)
	case kind == reflect.Struct:
		return makeStructWriter(typ)
	case kind == reflect.Interface:
		return writeInterface, nil
	default:
		return nil, fmt.Errorf("rlp: type %v is not RLP-serializable", typ)
	}
}

func writeRawValue(val reflect.Value, w *encbuf) error {
	w.str = append(w.str, val.Bytes()...)
	return nil
}

func writeUint(val reflect.Value, w *encbuf) error {
	w.encodeUint(val.Uint())
	return nil
}

func writeBool(val reflect.Value, w *encbuf) error {
	if val.Bool() {
		w.str = append(w.str, 0x01)
	} else {
		w.str = append(w.str, 0x80)
	}
	return nil
}

func writeBigIntPtr(val reflect.Value, w *encbuf) error {
	ptr := val.Interface().(*big.Int)
	if ptr == nil {
		w.str = append(w.str, 0x80)
		return nil
	}
	return writeBigInt(ptr, w)
}

func writeBigIntNoPtr(val reflect.Value, w *encbuf) error {
	i := val.Interface().(big.Int)
	return writeBigInt(&i, w)
}

// wordBytes is the number of bytes in a big.Word
const wordBytes = (32 << (uint64(^big.Word(0)) >> 63)) / 8

func writeBigInt(i *big.Int, w *encbuf) error {
	if i.Sign() == -1 {
		return fmt.Errorf("rlp: cannot encode negative *big.Int")
	}
	bitlen := i.BitLen()
	if bitlen <= 64 {
		w.encodeUint(i.Uint64())
		return nil
	}
	// Integer is larger than 64 bits, encode from i.Bits().
	// The minimal byte length is bitlen rounded up to the next
	// multiple of 8, divided by 8.
	length := ((bitlen + 7) & -8) >> 3
	w.encodeStringHeader(length)
	w.str = append(w.str, make([]byte, length)...)
	index := length
	buf := w.str[len(w.str)-length:]
	for _, d := range i.Bits() {
		for j := 0; j < wordBytes && index > 0; j++ {
			index--
			buf[index] = byte(d)
			d >>= 8
		}
	}
	return nil
}

func writeBytes(val reflect.Value, w *encbuf) error {
	w.encodeString(val.Bytes())
	return nil
}

func makeByteArrayWriter(typ reflect.Type) writer {
	switch typ.Len() {
	case 0:
		return writeLengthZeroByteArray
	case 1:
		return writeLengthOneByteArray
	default:
		length := typ.Len()
		return func(val reflect.Value, w *encbuf) error {
			if !val.CanAddr() {
				// Getting the byte slice of val requires it to be addressable. Make it
				// addressable by copying.
				copy := reflect.New(val.Type()).Elem()
				copy.Set(val)
				val = copy
			}
			slice := byteArrayBytes(val, length)
			w.encodeStringHeader(len(slice))
			w.str = append(w.str, slice...)
			return nil
		}
	}
}

func writeLengthZeroByteArray(val reflect.Value, w *encbuf) error {
	w.str = append(w.str, 0x80)
	return nil
}

func writeLengthOneByteArray(val reflect.Value, w *encbuf) error {
	b := byte(val.Index(0).Uint())
	if b <= 0x7f {
		w.str = append(w.str, b)
	} else {
		w.str = append(w.str, 0x81, b)
	}
	return nil
}

func writeString(val reflect.Value, w *encbuf) error {
	s := val.String()
	if len(s) == 1 && s[0] <= 0x7f {
		// fits single byte, no string header
		w.str = append(w.str, s[0])
	} else {
		w.encodeStringHeader(len(s))
		w.str = append(w.str, s...)
	}
	return nil
}

func writeInterface(val reflect.Value, w *encbuf) error {
	if val.IsNil() {
		// Write empty list. This is consistent with the previous RLP
		// encoder that we had and should therefore avoid any
		// problems.
		w.str = append(w.str, 0xC0)
		return nil
	}
	eval := val.Elem()
	writer, err := cachedWriter(eval.Type())
	if err != nil {
		return err
	}
	return writer(eval, w)
}

func makeSliceWriter(typ reflect.Type, ts tags) (writer, error) {
	etypeinfo := theTC.infoWhileGenerating(typ.Elem(), tags{})
	if etypeinfo.writerErr != nil {
		return nil, etypeinfo.writerErr
	}

	var wfn writer
	if ts.tail {
		// This is for struct tail slices.
		// w.list is not called for them.
		wfn = func(val reflect.Value, w *encbuf) error {
			vlen := val.Len()
			for i := 0; i < vlen; i++ {
				if err := etypeinfo.writer(val.Index(i), w); err != nil {
					return err
				}
			}
			return nil
		}
	} else {
		// This is for regular slices and arrays.
		wfn = func(val reflect.Value, w *encbuf) error {
			vlen := val.Len()
			if vlen == 0 {
				w.str = append(w.str, 0xC0)
				return nil
			}
			listOffset := w.list()
			for i := 0; i < vlen; i++ {
				if err := etypeinfo.writer(val.Index(i), w); err != nil {
					return err
				}
			}
			w.listEnd(listOffset)
			return nil
		}
	}
	return wfn, nil
}

func makeStructWriter(typ reflect.Type) (writer, error) {
	fields, err := structFields(typ)
	if err != nil {
		return nil, err
	}
	for _, f := range fields {
		if f.info.writerErr != nil {
			return nil, structFieldError{typ, f.index, f.info.writerErr}
		}
	}

	var writer writer
	firstOptionalField := firstOptionalField(fields)
	if firstOptionalField == len(fields) {
		// This is the writer function for structs without any optional fields.
		writer = func(val reflect.Value, w *encbuf) error {
			lh := w.list()
			for _, f := range fields {
				if err := f.info.writer(val.Field(f.index), w); err != nil {
					return err
				}
			}
			w.listEnd(lh)
			return nil
		}
	} else {
		// If there are any "optional" fields, the writer needs to perform additional
		// checks to determine the output list length.
		writer = func(val reflect.Value, w *encbuf) error {
			lastField := len(fields) - 1
			for ; lastField >= firstOptionalField; lastField-- {
				if !val.Field(fields[lastField].index).IsZero() {
					break
				}
			}
			lh := w.list()
			for i := 0; i <= lastField; i++ {
				if err := fields[i].info.writer(val.Field(fields[i].index), w); err != nil {
					return err
				}
			}
			w.listEnd(lh)
			return nil
		}
	}
	return writer, nil
}

// nilEncoding returns the encoded value of a nil pointer.
func nilEncoding(typ reflect.Type, ts tags) uint8 {
	var nilKind Kind
	if ts.nilOK {
		nilKind = ts.nilKind // use struct tag if provided
	} else {
		nilKind = defaultNilKind(typ.Elem())
	}

	switch nilKind {
	case String:
		return 0x80
	case List:
		return 0xC0
	default:
		panic(fmt.Errorf("rlp: invalid nil kind %d", nilKind))
	}
}

func makePtrWriter(typ reflect.Type, ts tags) (writer, error) {
	etypeinfo := theTC.infoWhileGenerating(typ.Elem(), tags{})
	if etypeinfo.writerErr != nil {
		return nil, etypeinfo.writerErr
	}
	nilEncoding := nilEncoding(typ, ts)

	writer := func(val reflect.Value, w *encbuf) error {
		if ev := val.Elem(); ev.IsValid() {
			return etypeinfo.writer(ev, w)
		}
		w.str = append(w.str, nilEncoding)
		return nil
	}
	return writer, nil
}

func makeEncoderWriter(typ reflect.Type) writer {
	if typ.Implements(encoderInterface) {
		return func(val reflect.Value, w *encbuf) error {
			return val.Interface().(Encoder).EncodeRLP(w)
		}
	}
	w := func(val reflect.Value, w *encbuf) error {
		if !val.CanAddr() {
			// package json simply doesn't call MarshalJSON for this case, but encodes the
			// value as if it didn't implement the interface. We don't want to handle it that
			// way.
			return fmt.Errorf("rlp: unadressable value of type %v, EncodeRLP is pointer method", val.Type())
		}
		return val.Addr().Interface().(Encoder).EncodeRLP(w)
	}
	return w
}

// putint writes i to the beginning of b in big endian byte
// order, using the least number of bytes needed to represent i.
func putint(b []byte, i uint64) (size int) {
	switch {
	case i < (1 << 8):
		b[0] = byte(i)
		return 1
	case i < (1 << 16):
		b[0] = byte(i >> 8)
		b[1] = byte(i)
		return 2
	case i < (1 << 24):
		b[0] = byte(i >> 16)
		b[1] = byte(i >> 8)
		b[2] = byte(i)
		return 3
	case i < (1 << 32):
		b[0] = byte(i >> 24)
		b[1] = byte(i >> 16)
		b[2] = byte(i >> 8)
		b[3] = byte(i)
		return 4
	case i < (1 << 40):
		b[0] = byte(i >> 32)
		b[1] = byte(i >> 24)
		b[2] = byte(i >> 16)
		b[3] = byte(i >> 8)
		b[4] = byte(i)
		return 5
	case i < (1 << 48):
		b[0] = byte(i >> 40)
		b[1] = byte(i >> 32)
		b[2] = byte(i >> 24)
		b[3] = byte(i >> 16)
		b[4] = byte(i >> 8)
		b[5] = byte(i)
		return 6
	case i < (1 << 56):
		b[0] = byte(i >> 48)
		b[1] = byte(i >> 40)
		b[2] = byte(i >> 32)
		b[3] = byte(i >> 24)
		b[4] = byte(i >> 16)
		b[5] = byte(i >> 8)
		b[6] = byte(i)
		return 7
	default:
		b[0] = byte(i >> 56)
		b[1] = byte(i >> 48)
		b[2] = byte(i >> 40)
		b[3] = byte(i >> 32)
		b[4] = byte(i >> 24)
		b[5] = byte(i >> 16)
		b[6] = byte(i >> 8)
		b[7] = byte(i)
		return 8
	}
}

// intsize computes the minimum number of bytes required to store i.
func intsize(i uint64) (size int) {
	for size = 1; ; size++ {
		if i >>= 8; i == 0 {
			return size
		}
	}
}
