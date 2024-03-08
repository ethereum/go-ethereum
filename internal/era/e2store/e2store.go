// Copyright 2023 The go-ethereum Authors
// This file is part of go-ethereum.
//
// go-ethereum is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// go-ethereum is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with go-ethereum. If not, see <http://www.gnu.org/licenses/>.

package e2store

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
)

const (
	headerSize     = 8
	valueSizeLimit = 1024 * 1024 * 50
)

// Entry is a variable-length-data record in an e2store.
type Entry struct {
	Type  uint16
	Value []byte
}

// Writer writes entries using e2store encoding.
// For more information on this format, see:
// https://github.com/status-im/nimbus-eth2/blob/stable/docs/e2store.md
type Writer struct {
	w io.Writer
}

// NewWriter returns a new Writer that writes to w.
func NewWriter(w io.Writer) *Writer {
	return &Writer{w}
}

// Write writes a single e2store entry to w.
// An entry is encoded in a type-length-value format. The first 8 bytes of the
// record store the type (2 bytes), the length (4 bytes), and some reserved
// data (2 bytes). The remaining bytes store b.
func (w *Writer) Write(typ uint16, b []byte) (int, error) {
	buf := make([]byte, headerSize)
	binary.LittleEndian.PutUint16(buf, typ)
	binary.LittleEndian.PutUint32(buf[2:], uint32(len(b)))

	// Write header.
	if n, err := w.w.Write(buf); err != nil {
		return n, err
	}
	// Write value, return combined write size.
	n, err := w.w.Write(b)
	return n + headerSize, err
}

// A Reader reads entries from an e2store-encoded file.
// For more information on this format, see
// https://github.com/status-im/nimbus-eth2/blob/stable/docs/e2store.md
type Reader struct {
	r      io.ReaderAt
	offset int64
}

// NewReader returns a new Reader that reads from r.
func NewReader(r io.ReaderAt) *Reader {
	return &Reader{r, 0}
}

// Read reads one Entry from r.
func (r *Reader) Read() (*Entry, error) {
	var e Entry
	n, err := r.ReadAt(&e, r.offset)
	if err != nil {
		return nil, err
	}
	r.offset += int64(n)
	return &e, nil
}

// ReadAt reads one Entry from r at the specified offset.
func (r *Reader) ReadAt(entry *Entry, off int64) (int, error) {
	typ, length, err := r.ReadMetadataAt(off)
	if err != nil {
		return 0, err
	}
	entry.Type = typ

	// Check length bounds.
	if length > valueSizeLimit {
		return headerSize, fmt.Errorf("item larger than item size limit %d: have %d", valueSizeLimit, length)
	}
	if length == 0 {
		return headerSize, nil
	}

	// Read value.
	val := make([]byte, length)
	if n, err := r.r.ReadAt(val, off+headerSize); err != nil {
		n += headerSize
		// An entry with a non-zero length should not return EOF when
		// reading the value.
		if err == io.EOF {
			return n, io.ErrUnexpectedEOF
		}
		return n, err
	}
	entry.Value = val
	return int(headerSize + length), nil
}

// ReaderAt returns an io.Reader delivering value data for the entry at
// the specified offset. If the entry type does not match the expected type, an
// error is returned.
func (r *Reader) ReaderAt(expectedType uint16, off int64) (io.Reader, int, error) {
	// problem = need to return length+headerSize not just value length via section reader
	typ, length, err := r.ReadMetadataAt(off)
	if err != nil {
		return nil, headerSize, err
	}
	if typ != expectedType {
		return nil, headerSize, fmt.Errorf("wrong type, want %d have %d", expectedType, typ)
	}
	if length > valueSizeLimit {
		return nil, headerSize, fmt.Errorf("item larger than item size limit %d: have %d", valueSizeLimit, length)
	}
	return io.NewSectionReader(r.r, off+headerSize, int64(length)), headerSize + int(length), nil
}

// LengthAt reads the header at off and returns the total length of the entry,
// including header.
func (r *Reader) LengthAt(off int64) (int64, error) {
	_, length, err := r.ReadMetadataAt(off)
	if err != nil {
		return 0, err
	}
	return int64(length) + headerSize, nil
}

// ReadMetadataAt reads the header metadata at the given offset.
func (r *Reader) ReadMetadataAt(off int64) (typ uint16, length uint32, err error) {
	b := make([]byte, headerSize)
	if n, err := r.r.ReadAt(b, off); err != nil {
		if err == io.EOF && n > 0 {
			return 0, 0, io.ErrUnexpectedEOF
		}
		return 0, 0, err
	}
	typ = binary.LittleEndian.Uint16(b)
	length = binary.LittleEndian.Uint32(b[2:])

	// Check reserved bytes of header.
	if b[6] != 0 || b[7] != 0 {
		return 0, 0, errors.New("reserved bytes are non-zero")
	}

	return typ, length, nil
}

// Find returns the first entry with the matching type.
func (r *Reader) Find(want uint16) (*Entry, error) {
	var (
		off    int64
		typ    uint16
		length uint32
		err    error
	)
	for {
		typ, length, err = r.ReadMetadataAt(off)
		if err == io.EOF {
			return nil, io.EOF
		} else if err != nil {
			return nil, err
		}
		if typ == want {
			var e Entry
			if _, err := r.ReadAt(&e, off); err != nil {
				return nil, err
			}
			return &e, nil
		}
		off += int64(headerSize + length)
	}
}

// FindAll returns all entries with the matching type.
func (r *Reader) FindAll(want uint16) ([]*Entry, error) {
	var (
		off     int64
		typ     uint16
		length  uint32
		entries []*Entry
		err     error
	)
	for {
		typ, length, err = r.ReadMetadataAt(off)
		if err == io.EOF {
			return entries, nil
		} else if err != nil {
			return entries, err
		}
		if typ == want {
			e := new(Entry)
			if _, err := r.ReadAt(e, off); err != nil {
				return entries, err
			}
			entries = append(entries, e)
		}
		off += int64(headerSize + length)
	}
}
