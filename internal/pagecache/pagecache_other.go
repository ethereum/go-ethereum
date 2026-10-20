// Copyright 2026 The go-ethereum Authors
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

//go:build !linux

package pagecache

import "os"

// Writer is the file itself: only Linux has the calls to write behind.
type Writer = os.File

// NewWriter returns f.
func NewWriter(f *os.File) *Writer { return f }

// Reader is the file itself: only Linux has the calls to drop behind.
type Reader = os.File

// NewReader returns f.
func NewReader(f *os.File) *Reader { return f }

// NewConsumingReader returns f: only Linux has the call to punch the file
// out behind the reader.
func NewConsumingReader(f *os.File) *Reader { return f }
