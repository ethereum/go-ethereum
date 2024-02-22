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
	for _, test := range []struct {
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
		tt := test
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var (
				b = bytes.NewBuffer(nil)
				w = NewWriter(b)
			)
			for _, e := range tt.entries {
				if _, err := w.Write(e.Type, e.Value); err != nil {
					t.Fatalf("encoding error: %v", err)
				}
			}
			if want, have := common.FromHex(tt.want), b.Bytes(); !bytes.Equal(want, have) {
				t.Fatalf("encoding mismatch (want %x, have %x", want, have)
			}
			r := NewReader(bytes.NewReader(b.Bytes()))
			for _, want := range tt.entries {
				have, err := r.Read()
				if err != nil {
					t.Fatalf("decoding error: %v", err)
				}
				if have.Type != want.Type {
					t.Fatalf("decoded entry does type mismatch (want %v, got %v)", want.Type, have.Type)
				}
				if !bytes.Equal(have.Value, want.Value) {
					t.Fatalf("decoded entry does not match (want %#x, got %#x)", want.Value, have.Value)
				}
			}
		})
	}
}

func TestDecode(t *testing.T) {
	for i, tt := range []struct {
		have string
		err  error
	}{
		{ // basic valid decoding
			have: "ffff000000000000",
		},
		{ // basic invalid decoding
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
		r := NewReader(bytes.NewReader(common.FromHex(tt.have)))
		if tt.err != nil {
			_, err := r.Read()
			if err == nil && tt.err != nil {
				t.Fatalf("test %d, expected error, got none", i)
			}
			if err != nil && tt.err == nil {
				t.Fatalf("test %d, expected no error, got %v", i, err)
			}
			if err != nil && tt.err != nil && err.Error() != tt.err.Error() {
				t.Fatalf("expected error %v, got %v", tt.err, err)
			}
			continue
		}
	}
}

func FuzzCodec(f *testing.F) {
	f.Fuzz(func(t *testing.T, input []byte) {
		r := NewReader(bytes.NewReader(input))
		entry, err := r.Read()
		if err != nil {
			return
		}
		var (
			b = bytes.NewBuffer(nil)
			w = NewWriter(b)
		)
		w.Write(entry.Type, entry.Value)
		output := b.Bytes()
		// Only care about the input that was actually consumed
		input = input[:r.offset]
		if !bytes.Equal(input, output) {
			t.Fatalf("decode-encode mismatch, input %#x output %#x", input, output)
		}
	})
}
