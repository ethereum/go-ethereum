// Copyright 2026 go-ethereum Authors
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

package archive

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/ethereum/go-ethereum/rlp"
)

// ResolverFn is a callback to resolve expired nodes from an archive file.
// Given an offset and size, it returns the serialized node data from the archive.
type ResolverFn func(offset, size uint64) ([]*Record, error)

// OffsetSize is the size of the file offset in bytes.
const OffsetSize = 8

var (
	EmptyArchiveRecord = errors.New("empty record")                             // The archive contained a size-zero record.
	ErrNoResolver      = errors.New("no archive resolver set for expired node") // An expired node is accessed without a resolver.
)

// Record contains an archive file record. It is not the most optimal
// structure, since any modification to it will need to be overwritten.
type Record struct {
	Path  []byte
	Value []byte
}

// ArchivedNodeResolver takes a buffer containing the archive data
// held by an expiring node (an offset and a size) and returns a
// list of records, which is a list of serialized leaf nodes. The
// caller knows the context (MPT, binary trie) and is responsible
// for decoding the nodes.
func ArchivedNodeResolver(offset, size uint64) ([]*Record, error) {
	file, err := os.Open("nodearchive")
	if err != nil {
		return nil, fmt.Errorf("error opening archive file: %w", err)
	}
	defer file.Close()

	o, err := file.Seek(int64(offset), io.SeekStart)
	if err != nil {
		return nil, fmt.Errorf("error seeking into archive file: %w", err)
	}
	if uint64(o) != offset {
		return nil, fmt.Errorf("invalid offset: want %d, got %d", offset, o)
	}

	data := make([]byte, size)
	if _, err := io.ReadFull(file, data); err != nil {
		return nil, fmt.Errorf("error reading data from archive: %w", err)
	}

	var records []*Record
	stream := rlp.NewStream(bytes.NewReader(data), uint64(len(data)))
	for len(data) > 0 {
		_, size, err := stream.Kind()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("error getting rlp kind from archive data: %w", err)
		}
		var record Record
		err = stream.Decode(&record)
		if err != nil {
			return nil, fmt.Errorf("error decoding rlp record from archive data: %w", err)
		}
		records = append(records, &record)
	}
	return records, nil
}
