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
	"bytes"
	"fmt"
	"io"
	"testing"

	"github.com/ethereum/go-ethereum/common"
)

func TestEncode(t *testing.T) {
	for _, tt := range []struct {
		entries []Entry
		want    string
		name    string
	}{
		{
			name:    "emptyEntry",
			entries: []Entry{{0xffff, nil}},
			want:    "ffff000000000000",
		},
		{
			name:    "beef",
			entries: []Entry{{42, common.Hex2Bytes("beef")}},
			want:    "2a00020000000000beef",
		},
		{
			name: "twoEntries",
			entries: []Entry{
				{42, common.Hex2Bytes("beef")},
				{9, common.Hex2Bytes("abcdabcd")},
			},
			want: "2a00020000000000beef0900040000000000abcdabcd",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			var (
				b = NewWriteSeeker()
				w = NewWriter(b)
			)
			for _, e := range tt.entries {
				if _, err := w.Write(e.Type, e.Value); err != nil {
					t.Fatalf("encoding error: %v", err)
				}
			}
			if want, got := common.Hex2Bytes(tt.want), b.Bytes(); !bytes.Equal(want, got) {
				t.Fatalf("encoding mismatch (want %s, got %s", common.Bytes2Hex(want), common.Bytes2Hex(got))
			}
			r := NewReader(bytes.NewReader(b.Bytes()))
			for _, want := range tt.entries {
				if got, err := r.Read(); err != nil {
					t.Fatalf("decoding error: %v", err)
				} else if got.Type != want.Type || !bytes.Equal(got.Value, want.Value) {
					t.Fatalf("decoded entry does not match (want %v, got %v)", want, got)
				}
			}
		})
	}
}

func TestDecode(t *testing.T) {
	for i, tt := range []struct {
		have string
		want []Entry
		err  error
	}{
		{ // basic valid decoding
			have: "ffff000000000000",
			want: []Entry{{0xffff, nil}},
		},
		{ // basic valid decoding
			have: "ffff000000000001",
			err:  fmt.Errorf("reserved bytes are non-zero"),
		},
		{ // no more entries to read, returns EOF
			have: "",
			err:  io.EOF,
		},
		{ // malformed type
			have: "bad",
			err:  io.ErrUnexpectedEOF,
		},
		{ // malformed length
			have: "badbeef",
			err:  io.ErrUnexpectedEOF,
		},
		{ // specified length longer than actual value
			have: "beef010000000000",
			err:  io.ErrUnexpectedEOF,
		},
	} {
		r := NewReader(bytes.NewReader(common.Hex2Bytes(tt.have)))
		if tt.err != nil {
			if _, err := r.Read(); err != nil && tt.err != nil && err.Error() != tt.err.Error() {
				t.Fatalf("expected error %v, got %v", tt.err, err)
			}
			continue
		}
		for _, want := range tt.want {
			if got, err := r.Read(); err != nil {
				t.Fatalf("test %d: decoding error: %v", i, err)
			} else if got.Type != want.Type || !bytes.Equal(got.Value, want.Value) {
				t.Fatalf("test %d: decoded entry does not match (want %v, got %v)", i, want, got)
			}
		}
	}
}

// WriteSeeker is an in-memory io.Writer and io.Seeker implementation.
type WriteSeeker struct {
	pos int64
	buf []byte
}

func NewWriteSeeker() *WriteSeeker {
	return &WriteSeeker{}
}

func (w *WriteSeeker) Write(p []byte) (n int, err error) {
	if len(w.buf) != int(w.pos) {
		return 0, fmt.Errorf("writing after seek not supported")
	}
	w.buf = append(w.buf, p...)
	w.pos += int64(len(p))
	return len(p), nil
}

func (w *WriteSeeker) Seek(offset int64, whence int) (int64, error) {
	switch whence {
	case io.SeekStart:
		w.pos = offset
	case io.SeekCurrent:
		w.pos = w.pos + offset
	case io.SeekEnd:
		w.pos = int64(len(w.buf)) + offset
	default:
		return 0, fmt.Errorf("unknown seek whence %d", whence)
	}
	if w.pos < 0 {
		w.pos = 0
	}
	return w.pos, nil
}

func (w *WriteSeeker) Bytes() []byte {
	return w.buf
}
