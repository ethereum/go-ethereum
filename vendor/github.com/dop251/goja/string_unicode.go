package goja

import (
	"errors"
	"fmt"
	"github.com/dop251/goja/parser"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
	"io"
	"math"
	"reflect"
	"regexp"
	"strings"
	"unicode/utf16"
	"unicode/utf8"
)

type unicodeString []uint16

type unicodeRuneReader struct {
	s   unicodeString
	pos int
}

type runeReaderReplace struct {
	wrapped io.RuneReader
}

var (
	InvalidRuneError = errors.New("Invalid rune")
)

var (
	unicodeTrimRegexp = regexp.MustCompile("^[" + parser.WhitespaceChars + "]*(.*?)[" + parser.WhitespaceChars + "]*$")
)

func (rr runeReaderReplace) ReadRune() (r rune, size int, err error) {
	r, size, err = rr.wrapped.ReadRune()
	if err == InvalidRuneError {
		err = nil
		r = utf8.RuneError
	}
	return
}

func (rr *unicodeRuneReader) ReadRune() (r rune, size int, err error) {
	if rr.pos < len(rr.s) {
		r = rune(rr.s[rr.pos])
		if r != utf8.RuneError {
			if utf16.IsSurrogate(r) {
				if rr.pos+1 < len(rr.s) {
					r1 := utf16.DecodeRune(r, rune(rr.s[rr.pos+1]))
					size++
					rr.pos++
					if r1 == utf8.RuneError {
						err = InvalidRuneError
					} else {
						r = r1
					}
				} else {
					err = InvalidRuneError
				}
			}
		}
		size++
		rr.pos++
	} else {
		err = io.EOF
	}
	return
}

func (s unicodeString) reader(start int) io.RuneReader {
	return &unicodeRuneReader{
		s: s[start:],
	}
}

func (s unicodeString) ToInteger() int64 {
	return 0
}

func (s unicodeString) ToString() valueString {
	return s
}

func (s unicodeString) ToFloat() float64 {
	return math.NaN()
}

func (s unicodeString) ToBoolean() bool {
	return len(s) > 0
}

func (s unicodeString) toTrimmedUTF8() string {
	if len(s) == 0 {
		return ""
	}
	return unicodeTrimRegexp.FindStringSubmatch(s.String())[1]
}

func (s unicodeString) ToNumber() Value {
	return asciiString(s.toTrimmedUTF8()).ToNumber()
}

func (s unicodeString) ToObject(r *Runtime) *Object {
	return r._newString(s)
}

func (s unicodeString) equals(other unicodeString) bool {
	if len(s) != len(other) {
		return false
	}
	for i, r := range s {
		if r != other[i] {
			return false
		}
	}
	return true
}

func (s unicodeString) SameAs(other Value) bool {
	if otherStr, ok := other.(unicodeString); ok {
		return s.equals(otherStr)
	}

	return false
}

func (s unicodeString) Equals(other Value) bool {
	if s.SameAs(other) {
		return true
	}

	if _, ok := other.assertInt(); ok {
		return false
	}

	if _, ok := other.assertFloat(); ok {
		return false
	}

	if _, ok := other.(valueBool); ok {
		return false
	}

	if o, ok := other.(*Object); ok {
		return s.Equals(o.self.toPrimitive())
	}
	return false
}

func (s unicodeString) StrictEquals(other Value) bool {
	return s.SameAs(other)
}

func (s unicodeString) assertInt() (int64, bool) {
	return 0, false
}

func (s unicodeString) assertFloat() (float64, bool) {
	return 0, false
}

func (s unicodeString) assertString() (valueString, bool) {
	return s, true
}

func (s unicodeString) baseObject(r *Runtime) *Object {
	ss := r.stringSingleton
	ss.value = s
	ss.setLength()
	return ss.val
}

func (s unicodeString) charAt(idx int64) rune {
	return rune(s[idx])
}

func (s unicodeString) length() int64 {
	return int64(len(s))
}

func (s unicodeString) concat(other valueString) valueString {
	switch other := other.(type) {
	case unicodeString:
		return unicodeString(append(s, other...))
	case asciiString:
		b := make([]uint16, len(s)+len(other))
		copy(b, s)
		b1 := b[len(s):]
		for i := 0; i < len(other); i++ {
			b1[i] = uint16(other[i])
		}
		return unicodeString(b)
	default:
		panic(fmt.Errorf("Unknown string type: %T", other))
	}
}

func (s unicodeString) substring(start, end int64) valueString {
	ss := s[start:end]
	for _, c := range ss {
		if c >= utf8.RuneSelf {
			return unicodeString(ss)
		}
	}
	as := make([]byte, end-start)
	for i, c := range ss {
		as[i] = byte(c)
	}
	return asciiString(as)
}

func (s unicodeString) String() string {
	return string(utf16.Decode(s))
}

func (s unicodeString) compareTo(other valueString) int {
	return strings.Compare(s.String(), other.String())
}

func (s unicodeString) index(substr valueString, start int64) int64 {
	var ss []uint16
	switch substr := substr.(type) {
	case unicodeString:
		ss = substr
	case asciiString:
		ss = make([]uint16, len(substr))
		for i := 0; i < len(substr); i++ {
			ss[i] = uint16(substr[i])
		}
	default:
		panic(fmt.Errorf("Unknown string type: %T", substr))
	}

	// TODO: optimise
	end := int64(len(s) - len(ss))
	for start <= end {
		for i := int64(0); i < int64(len(ss)); i++ {
			if s[start+i] != ss[i] {
				goto nomatch
			}
		}

		return start
	nomatch:
		start++
	}
	return -1
}

func (s unicodeString) lastIndex(substr valueString, start int64) int64 {
	var ss []uint16
	switch substr := substr.(type) {
	case unicodeString:
		ss = substr
	case asciiString:
		ss = make([]uint16, len(substr))
		for i := 0; i < len(substr); i++ {
			ss[i] = uint16(substr[i])
		}
	default:
		panic(fmt.Errorf("Unknown string type: %T", substr))
	}

	if maxStart := int64(len(s) - len(ss)); start > maxStart {
		start = maxStart
	}
	// TODO: optimise
	for start >= 0 {
		for i := int64(0); i < int64(len(ss)); i++ {
			if s[start+i] != ss[i] {
				goto nomatch
			}
		}

		return start
	nomatch:
		start--
	}
	return -1
}

func (s unicodeString) toLower() valueString {
	caser := cases.Lower(language.Und)
	r := []rune(caser.String(s.String()))
	// Workaround
	ascii := true
	for i := 0; i < len(r)-1; i++ {
		if (i == 0 || r[i-1] != 0x3b1) && r[i] == 0x345 && r[i+1] == 0x3c2 {
			i++
			r[i] = 0x3c3
		}
		if r[i] >= utf8.RuneSelf {
			ascii = false
		}
	}
	if ascii {
		ascii = r[len(r)-1] < utf8.RuneSelf
	}
	if ascii {
		return asciiString(r)
	}
	return unicodeString(utf16.Encode(r))
}

func (s unicodeString) toUpper() valueString {
	caser := cases.Upper(language.Und)
	return newStringValue(caser.String(s.String()))
}

func (s unicodeString) Export() interface{} {
	return s.String()
}

func (s unicodeString) ExportType() reflect.Type {
	return reflectTypeString
}
