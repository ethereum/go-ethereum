package goja

import (
	"errors"
	"io"
	"math"
	"regexp"
	"strconv"
	"unicode/utf16"
	"unicode/utf8"
)

var (
	parseFloatRegexp = regexp.MustCompile(`^([+-]?(?:Infinity|[0-9]*\.?[0-9]*(?:[eE][+-]?[0-9]+)?))`)
)

func (r *Runtime) builtin_isNaN(call FunctionCall) Value {
	if math.IsNaN(call.Argument(0).ToFloat()) {
		return valueTrue
	} else {
		return valueFalse
	}
}

func (r *Runtime) builtin_parseInt(call FunctionCall) Value {
	str := call.Argument(0).ToString().toTrimmedUTF8()
	radix := int(toInt32(call.Argument(1)))
	v, _ := parseInt(str, radix)
	return v
}

func (r *Runtime) builtin_parseFloat(call FunctionCall) Value {
	m := parseFloatRegexp.FindStringSubmatch(call.Argument(0).ToString().toTrimmedUTF8())
	if len(m) == 2 {
		if s := m[1]; s != "" && s != "+" && s != "-" {
			switch s {
			case "+", "-":
			case "Infinity", "+Infinity":
				return _positiveInf
			case "-Infinity":
				return _negativeInf
			default:
				f, err := strconv.ParseFloat(s, 64)
				if err == nil || isRangeErr(err) {
					return floatToValue(f)
				}
			}
		}
	}
	return _NaN
}

func (r *Runtime) builtin_isFinite(call FunctionCall) Value {
	f := call.Argument(0).ToFloat()
	if math.IsNaN(f) || math.IsInf(f, 0) {
		return valueFalse
	}
	return valueTrue
}

func (r *Runtime) _encode(uriString valueString, unescaped *[256]bool) valueString {
	reader := uriString.reader(0)
	utf8Buf := make([]byte, utf8.UTFMax)
	needed := false
	l := 0
	for {
		rn, _, err := reader.ReadRune()
		if err != nil {
			if err != io.EOF {
				panic(r.newError(r.global.URIError, "Malformed URI"))
			}
			break
		}

		if rn >= utf8.RuneSelf {
			needed = true
			l += utf8.EncodeRune(utf8Buf, rn) * 3
		} else if !unescaped[rn] {
			needed = true
			l += 3
		} else {
			l++
		}
	}

	if !needed {
		return uriString
	}

	buf := make([]byte, l)
	i := 0
	reader = uriString.reader(0)
	for {
		rn, _, err := reader.ReadRune()
		if err != nil {
			break
		}

		if rn >= utf8.RuneSelf {
			n := utf8.EncodeRune(utf8Buf, rn)
			for _, b := range utf8Buf[:n] {
				buf[i] = '%'
				buf[i+1] = "0123456789ABCDEF"[b>>4]
				buf[i+2] = "0123456789ABCDEF"[b&15]
				i += 3
			}
		} else if !unescaped[rn] {
			buf[i] = '%'
			buf[i+1] = "0123456789ABCDEF"[rn>>4]
			buf[i+2] = "0123456789ABCDEF"[rn&15]
			i += 3
		} else {
			buf[i] = byte(rn)
			i++
		}
	}
	return asciiString(string(buf))
}

func (r *Runtime) _decode(sv valueString, reservedSet *[256]bool) valueString {
	s := sv.String()
	hexCount := 0
	for i := 0; i < len(s); {
		switch s[i] {
		case '%':
			if i+2 >= len(s) || !ishex(s[i+1]) || !ishex(s[i+2]) {
				panic(r.newError(r.global.URIError, "Malformed URI"))
			}
			c := unhex(s[i+1])<<4 | unhex(s[i+2])
			if !reservedSet[c] {
				hexCount++
			}
			i += 3
		default:
			i++
		}
	}

	if hexCount == 0 {
		return sv
	}

	t := make([]byte, len(s)-hexCount*2)
	j := 0
	isUnicode := false
	for i := 0; i < len(s); {
		ch := s[i]
		switch ch {
		case '%':
			c := unhex(s[i+1])<<4 | unhex(s[i+2])
			if reservedSet[c] {
				t[j] = s[i]
				t[j+1] = s[i+1]
				t[j+2] = s[i+2]
				j += 3
			} else {
				t[j] = c
				if c >= utf8.RuneSelf {
					isUnicode = true
				}
				j++
			}
			i += 3
		default:
			if ch >= utf8.RuneSelf {
				isUnicode = true
			}
			t[j] = ch
			j++
			i++
		}
	}

	if !isUnicode {
		return asciiString(t)
	}

	us := make([]rune, 0, len(s))
	for len(t) > 0 {
		rn, size := utf8.DecodeRune(t)
		if rn == utf8.RuneError {
			if size != 3 || t[0] != 0xef || t[1] != 0xbf || t[2] != 0xbd {
				panic(r.newError(r.global.URIError, "Malformed URI"))
			}
		}
		us = append(us, rn)
		t = t[size:]
	}
	return unicodeString(utf16.Encode(us))
}

func ishex(c byte) bool {
	switch {
	case '0' <= c && c <= '9':
		return true
	case 'a' <= c && c <= 'f':
		return true
	case 'A' <= c && c <= 'F':
		return true
	}
	return false
}

func unhex(c byte) byte {
	switch {
	case '0' <= c && c <= '9':
		return c - '0'
	case 'a' <= c && c <= 'f':
		return c - 'a' + 10
	case 'A' <= c && c <= 'F':
		return c - 'A' + 10
	}
	return 0
}

func (r *Runtime) builtin_decodeURI(call FunctionCall) Value {
	uriString := call.Argument(0).ToString()
	return r._decode(uriString, &uriReservedHash)
}

func (r *Runtime) builtin_decodeURIComponent(call FunctionCall) Value {
	uriString := call.Argument(0).ToString()
	return r._decode(uriString, &emptyEscapeSet)
}

func (r *Runtime) builtin_encodeURI(call FunctionCall) Value {
	uriString := call.Argument(0).ToString()
	return r._encode(uriString, &uriReservedUnescapedHash)
}

func (r *Runtime) builtin_encodeURIComponent(call FunctionCall) Value {
	uriString := call.Argument(0).ToString()
	return r._encode(uriString, &uriUnescaped)
}

func (r *Runtime) initGlobalObject() {
	o := r.globalObject.self
	o._putProp("NaN", _NaN, false, false, false)
	o._putProp("undefined", _undefined, false, false, false)
	o._putProp("Infinity", _positiveInf, false, false, false)

	o._putProp("isNaN", r.newNativeFunc(r.builtin_isNaN, nil, "isNaN", nil, 1), true, false, true)
	o._putProp("parseInt", r.newNativeFunc(r.builtin_parseInt, nil, "parseInt", nil, 2), true, false, true)
	o._putProp("parseFloat", r.newNativeFunc(r.builtin_parseFloat, nil, "parseFloat", nil, 1), true, false, true)
	o._putProp("isFinite", r.newNativeFunc(r.builtin_isFinite, nil, "isFinite", nil, 1), true, false, true)
	o._putProp("decodeURI", r.newNativeFunc(r.builtin_decodeURI, nil, "decodeURI", nil, 1), true, false, true)
	o._putProp("decodeURIComponent", r.newNativeFunc(r.builtin_decodeURIComponent, nil, "decodeURIComponent", nil, 1), true, false, true)
	o._putProp("encodeURI", r.newNativeFunc(r.builtin_encodeURI, nil, "encodeURI", nil, 1), true, false, true)
	o._putProp("encodeURIComponent", r.newNativeFunc(r.builtin_encodeURIComponent, nil, "encodeURIComponent", nil, 1), true, false, true)

	o._putProp("toString", r.newNativeFunc(func(FunctionCall) Value {
		return stringGlobalObject
	}, nil, "toString", nil, 0), false, false, false)

	// TODO: Annex B

}

func digitVal(d byte) int {
	var v byte
	switch {
	case '0' <= d && d <= '9':
		v = d - '0'
	case 'a' <= d && d <= 'z':
		v = d - 'a' + 10
	case 'A' <= d && d <= 'Z':
		v = d - 'A' + 10
	default:
		return 36
	}
	return int(v)
}

// ECMAScript compatible version of strconv.ParseInt
func parseInt(s string, base int) (Value, error) {
	var n int64
	var err error
	var cutoff, maxVal int64
	var sign bool
	i := 0

	if len(s) < 1 {
		err = strconv.ErrSyntax
		goto Error
	}

	switch s[0] {
	case '-':
		sign = true
		s = s[1:]
	case '+':
		s = s[1:]
	}

	if len(s) < 1 {
		err = strconv.ErrSyntax
		goto Error
	}

	// Look for hex prefix.
	if s[0] == '0' && len(s) > 1 && (s[1] == 'x' || s[1] == 'X') {
		if base == 0 || base == 16 {
			base = 16
			s = s[2:]
		}
	}

	switch {
	case len(s) < 1:
		err = strconv.ErrSyntax
		goto Error

	case 2 <= base && base <= 36:
	// valid base; nothing to do

	case base == 0:
		// Look for hex prefix.
		switch {
		case s[0] == '0' && len(s) > 1 && (s[1] == 'x' || s[1] == 'X'):
			if len(s) < 3 {
				err = strconv.ErrSyntax
				goto Error
			}
			base = 16
			s = s[2:]
		default:
			base = 10
		}

	default:
		err = errors.New("invalid base " + strconv.Itoa(base))
		goto Error
	}

	// Cutoff is the smallest number such that cutoff*base > maxInt64.
	// Use compile-time constants for common cases.
	switch base {
	case 10:
		cutoff = math.MaxInt64/10 + 1
	case 16:
		cutoff = math.MaxInt64/16 + 1
	default:
		cutoff = math.MaxInt64/int64(base) + 1
	}

	maxVal = math.MaxInt64
	for ; i < len(s); i++ {
		if n >= cutoff {
			// n*base overflows
			return parseLargeInt(float64(n), s[i:], base, sign)
		}
		v := digitVal(s[i])
		if v >= base {
			break
		}
		n *= int64(base)

		n1 := n + int64(v)
		if n1 < n || n1 > maxVal {
			// n+v overflows
			return parseLargeInt(float64(n)+float64(v), s[i+1:], base, sign)
		}
		n = n1
	}

	if i == 0 {
		err = strconv.ErrSyntax
		goto Error
	}

	if sign {
		n = -n
	}
	return intToValue(n), nil

Error:
	return _NaN, err
}

func parseLargeInt(n float64, s string, base int, sign bool) (Value, error) {
	i := 0
	b := float64(base)
	for ; i < len(s); i++ {
		v := digitVal(s[i])
		if v >= base {
			break
		}
		n = n*b + float64(v)
	}
	if sign {
		n = -n
	}
	// We know it can't be represented as int, so use valueFloat instead of floatToValue
	return valueFloat(n), nil
}

var (
	uriUnescaped             [256]bool
	uriReserved              [256]bool
	uriReservedHash          [256]bool
	uriReservedUnescapedHash [256]bool
	emptyEscapeSet           [256]bool
)

func init() {
	for _, c := range "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789-_.!~*'()" {
		uriUnescaped[c] = true
	}

	for _, c := range ";/?:@&=+$," {
		uriReserved[c] = true
	}

	for i := 0; i < 256; i++ {
		if uriUnescaped[i] || uriReserved[i] {
			uriReservedUnescapedHash[i] = true
		}
		uriReservedHash[i] = uriReserved[i]
	}
	uriReservedUnescapedHash['#'] = true
	uriReservedHash['#'] = true
}
