// Copyright 2013 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package encoding defines an interface for character encodings, such as Shift
// JIS and Windows 1252, that can convert to and from UTF-8.
//
// To convert the bytes of an io.Reader r from the encoding e to UTF-8:
//	rInUTF8 := transform.NewReader(r, e.NewDecoder())
// and to convert from UTF-8 to the encoding e:
//	wInUTF8 := transform.NewWriter(w, e.NewEncoder())
// In both cases, import "golang.org/x/text/transform".
//
// Encoding implementations are provided in other packages, such as
// golang.org/x/text/encoding/charmap and
// golang.org/x/text/encoding/japanese.
package encoding

import (
	"errors"
	"unicode/utf8"

	"golang.org/x/text/transform"
)

// Encoding is a character set encoding that can be transformed to and from
// UTF-8.
type Encoding interface {
	// NewDecoder returns a transformer that converts to UTF-8.
	//
	// Transforming source bytes that are not of that encoding will not
	// result in an error per se. Each byte that cannot be transcoded will
	// be represented in the output by the UTF-8 encoding of '\uFFFD', the
	// replacement rune.
	NewDecoder() transform.Transformer

	// NewEncoder returns a transformer that converts from UTF-8.
	//
	// Transforming source bytes that are not valid UTF-8 will not result in
	// an error per se. Each rune that cannot be transcoded will be
	// represented in the output by an encoding-specific replacement such as
	// "\x1a" (the ASCII substitute character) or "\xff\xfd". To return
	// early with error instead, use transform.Chain to preprocess the data
	// with a UTF8Validator.
	NewEncoder() transform.Transformer
}

// ASCIISub is the ASCII substitute character, as recommended by
// http://unicode.org/reports/tr36/#Text_Comparison
const ASCIISub = '\x1a'

// Nop is the nop encoding. Its transformed bytes are the same as the source
// bytes; it does not replace invalid UTF-8 sequences.
var Nop Encoding = nop{}

type nop struct{}

func (nop) NewDecoder() transform.Transformer {
	return transform.Nop
}

func (nop) NewEncoder() transform.Transformer {
	return transform.Nop
}

// Replacement is the replacement encoding. Decoding from the replacement
// encoding yields a single '\uFFFD' replacement rune. Encoding from UTF-8 to
// the replacement encoding yields the same as the source bytes except that
// invalid UTF-8 is converted to '\uFFFD'.
//
// It is defined at http://encoding.spec.whatwg.org/#replacement
var Replacement Encoding = replacement{}

type replacement struct{}

func (replacement) NewDecoder() transform.Transformer {
	return replacementDecoder{}
}

func (replacement) NewEncoder() transform.Transformer {
	return replacementEncoder{}
}

type replacementDecoder struct{ transform.NopResetter }

func (replacementDecoder) Transform(dst, src []byte, atEOF bool) (nDst, nSrc int, err error) {
	if len(dst) < 3 {
		return 0, 0, transform.ErrShortDst
	}
	if atEOF {
		const fffd = "\ufffd"
		dst[0] = fffd[0]
		dst[1] = fffd[1]
		dst[2] = fffd[2]
		nDst = 3
	}
	return nDst, len(src), nil
}

type replacementEncoder struct{ transform.NopResetter }

func (replacementEncoder) Transform(dst, src []byte, atEOF bool) (nDst, nSrc int, err error) {
	r, size := rune(0), 0

	for ; nSrc < len(src); nSrc += size {
		r = rune(src[nSrc])

		// Decode a 1-byte rune.
		if r < utf8.RuneSelf {
			size = 1

		} else {
			// Decode a multi-byte rune.
			r, size = utf8.DecodeRune(src[nSrc:])
			if size == 1 {
				// All valid runes of size 1 (those below utf8.RuneSelf) were
				// handled above. We have invalid UTF-8 or we haven't seen the
				// full character yet.
				if !atEOF && !utf8.FullRune(src[nSrc:]) {
					err = transform.ErrShortSrc
					break
				}
				r = '\ufffd'
			}
		}

		if nDst+utf8.RuneLen(r) > len(dst) {
			err = transform.ErrShortDst
			break
		}
		nDst += utf8.EncodeRune(dst[nDst:], r)
	}
	return nDst, nSrc, err
}

// ErrInvalidUTF8 means that a transformer encountered invalid UTF-8.
var ErrInvalidUTF8 = errors.New("encoding: invalid UTF-8")

// UTF8Validator is a transformer that returns ErrInvalidUTF8 on the first
// input byte that is not valid UTF-8.
var UTF8Validator transform.Transformer = utf8Validator{}

type utf8Validator struct{ transform.NopResetter }

func (utf8Validator) Transform(dst, src []byte, atEOF bool) (nDst, nSrc int, err error) {
	n := len(src)
	if n > len(dst) {
		n = len(dst)
	}
	for i := 0; i < n; {
		if c := src[i]; c < utf8.RuneSelf {
			dst[i] = c
			i++
			continue
		}
		_, size := utf8.DecodeRune(src[i:])
		if size == 1 {
			// All valid runes of size 1 (those below utf8.RuneSelf) were
			// handled above. We have invalid UTF-8 or we haven't seen the
			// full character yet.
			err = ErrInvalidUTF8
			if !atEOF && !utf8.FullRune(src[i:]) {
				err = transform.ErrShortSrc
			}
			return i, i, err
		}
		if i+size > len(dst) {
			return i, i, transform.ErrShortDst
		}
		for ; size > 0; size-- {
			dst[i] = src[i]
			i++
		}
	}
	if len(src) > len(dst) {
		err = transform.ErrShortDst
	}
	return n, n, err
}
