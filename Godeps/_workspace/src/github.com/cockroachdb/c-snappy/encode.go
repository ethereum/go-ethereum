// Copyright 2011 The Snappy-Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package snappy

import (
	"io"
	"unsafe"
)

// #include <snappy-c.h>
import "C"

// Encode returns the encoded form of src. The returned slice may be a sub-
// slice of dst if dst was large enough to hold the entire encoded block.
// Otherwise, a newly allocated slice will be returned.
// It is valid to pass a nil dst.
func Encode(dst, src []byte) ([]byte, error) {
	if n := MaxEncodedLen(len(src)); len(dst) < n {
		dst = make([]byte, n)
	}

	var srcPtr unsafe.Pointer
	if len(src) != 0 {
		srcPtr = unsafe.Pointer(&src[0])
	}

	dLen := C.size_t(len(dst))
	status := C.snappy_compress((*C.char)(srcPtr), C.size_t(len(src)),
		(*C.char)(unsafe.Pointer(&dst[0])), &dLen)
	if status != C.SNAPPY_OK {
		return nil, ErrCorrupt
	}

	return dst[:dLen], nil
}

// MaxEncodedLen returns the maximum length of a snappy block, given its
// uncompressed length.
func MaxEncodedLen(srcLen int) int {
	// Compressed data can be defined as:
	//    compressed := item* literal*
	//    item       := literal* copy
	//
	// The trailing literal sequence has a space blowup of at most 62/60
	// since a literal of length 60 needs one tag byte + one extra byte
	// for length information.
	//
	// Item blowup is trickier to measure. Suppose the "copy" op copies
	// 4 bytes of data. Because of a special check in the encoding code,
	// we produce a 4-byte copy only if the offset is < 65536. Therefore
	// the copy op takes 3 bytes to encode, and this type of item leads
	// to at most the 62/60 blowup for representing literals.
	//
	// Suppose the "copy" op copies 5 bytes of data. If the offset is big
	// enough, it will take 5 bytes to encode the copy op. Therefore the
	// worst case here is a one-byte literal followed by a five-byte copy.
	// That is, 6 bytes of input turn into 7 bytes of "compressed" data.
	//
	// This last factor dominates the blowup, so the final estimate is:
	return 32 + srcLen + srcLen/6
}

// NewWriter returns a new Writer that compresses to w, using the framing
// format described at
// https://code.google.com/p/snappy/source/browse/trunk/framing_format.txt
func NewWriter(w io.Writer) *Writer {
	return &Writer{
		w:   w,
		enc: make([]byte, MaxEncodedLen(maxUncompressedChunkLen)),
	}
}

// Writer is an io.Writer than can write Snappy-compressed bytes.
type Writer struct {
	w           io.Writer
	err         error
	enc         []byte
	buf         [checksumSize + chunkHeaderSize]byte
	wroteHeader bool
}

// Reset discards the writer's state and switches the Snappy writer to write to
// w. This permits reusing a Writer rather than allocating a new one.
func (w *Writer) Reset(writer io.Writer) {
	w.w = writer
	w.err = nil
	w.wroteHeader = false
}

// Write satisfies the io.Writer interface.
func (w *Writer) Write(p []byte) (n int, errRet error) {
	if w.err != nil {
		return 0, w.err
	}
	if !w.wroteHeader {
		copy(w.enc, magicChunk)
		if _, err := w.w.Write(w.enc[:len(magicChunk)]); err != nil {
			w.err = err
			return n, err
		}
		w.wroteHeader = true
	}
	for len(p) > 0 {
		var uncompressed []byte
		if len(p) > maxUncompressedChunkLen {
			uncompressed, p = p[:maxUncompressedChunkLen], p[maxUncompressedChunkLen:]
		} else {
			uncompressed, p = p, nil
		}
		checksum := crc(uncompressed)

		// Compress the buffer, discarding the result if the improvement
		// isn't at least 12.5%.
		chunkType := uint8(chunkTypeCompressedData)
		chunkBody, err := Encode(w.enc, uncompressed)
		if err != nil {
			w.err = err
			return n, err
		}
		if len(chunkBody) >= len(uncompressed)-len(uncompressed)/8 {
			chunkType, chunkBody = chunkTypeUncompressedData, uncompressed
		}

		chunkLen := 4 + len(chunkBody)
		w.buf[0] = chunkType
		w.buf[1] = uint8(chunkLen >> 0)
		w.buf[2] = uint8(chunkLen >> 8)
		w.buf[3] = uint8(chunkLen >> 16)
		w.buf[4] = uint8(checksum >> 0)
		w.buf[5] = uint8(checksum >> 8)
		w.buf[6] = uint8(checksum >> 16)
		w.buf[7] = uint8(checksum >> 24)
		if _, err = w.w.Write(w.buf[:]); err != nil {
			w.err = err
			return n, err
		}
		if _, err = w.w.Write(chunkBody); err != nil {
			w.err = err
			return n, err
		}
		n += len(uncompressed)
	}
	return n, nil
}
