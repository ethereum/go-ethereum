// Copyright 2026 The go-ethereum Authors
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

package main

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"io"
	"maps"
	"os"
	"slices"
)

// nodeStager holds the tree nodes StackBuilder emits until the tree is built,
// then hands them back in database key order. A node's key sorts by depth
// first (encodePath leads with the path's bit count) and the builder emits
// each depth in ascending order, but children before parents, so consecutive
// nodes jump between depths. Written as they come, every flushed table spans
// the node keys of every depth and overlaps the tables before it, so the
// database rewrites the nodes at each level on the way down. One append-only
// file per depth, replayed in depth order, writes them in ascending key order
// instead.
type nodeStager struct {
	dir    string
	depths map[int]*spillFile
}

// newNodeStager stages its files in a fresh directory under tmpDir.
func newNodeStager(tmpDir string) (*nodeStager, error) {
	dir, err := os.MkdirTemp(tmpDir, "bintrie-nodes-")
	if err != nil {
		return nil, err
	}
	return &nodeStager{dir: dir, depths: make(map[int]*spillFile)}, nil
}

// nodeDepth is the bit count encodePath leads a node's path with. The root's
// empty path sorts before every other one and gets depth -1.
func nodeDepth(path []byte) int {
	if len(path) < 2 {
		return -1
	}
	return int(binary.BigEndian.Uint16(path))
}

// add appends one node to its depth's file. Neither path nor blob needs to
// outlive the call.
func (s *nodeStager) add(path, blob []byte) error {
	depth := nodeDepth(path)
	d := s.depths[depth]
	if d == nil {
		var err error
		if d, err = newSpillFile(s.dir, fmt.Sprintf("depth-%d-*", depth)); err != nil {
			return err
		}
		s.depths[depth] = d
	}
	return d.add(path, blob)
}

// replay feeds every staged node to write in database key order: depth by
// depth, each depth in the order its nodes were added. path and blob are only
// valid for the call.
func (s *nodeStager) replay(write func(path, blob []byte) error) error {
	r := bufio.NewReaderSize(nil, 1<<20)
	for _, depth := range slices.Sorted(maps.Keys(s.depths)) {
		if err := s.depths[depth].replay(r, write); err != nil {
			return err
		}
	}
	return nil
}

// close removes the staging files.
func (s *nodeStager) close() {
	for _, d := range s.depths {
		d.close()
	}
	os.RemoveAll(s.dir)
}

// spillFile is an append-only temp file of two-field records, read back in
// the order they were added.
type spillFile struct {
	f    *os.File
	w    *bufio.Writer
	head [2 * binary.MaxVarintLen64]byte // a record's length prefix
}

// newSpillFile creates a spill file in dir, named from pattern as
// os.CreateTemp does.
func newSpillFile(dir, pattern string) (*spillFile, error) {
	f, err := os.CreateTemp(dir, pattern)
	if err != nil {
		return nil, err
	}
	return &spillFile{f: f, w: bufio.NewWriterSize(f, 256<<10)}, nil
}

// add appends one record, both fields length-prefixed. Neither needs to
// outlive the call.
func (s *spillFile) add(a, b []byte) error {
	n := binary.PutUvarint(s.head[:], uint64(len(a)))
	n += binary.PutUvarint(s.head[n:], uint64(len(b)))
	if _, err := s.w.Write(s.head[:n]); err != nil {
		return err
	}
	if _, err := s.w.Write(a); err != nil {
		return err
	}
	_, err := s.w.Write(b)
	return err
}

// replay feeds every record to fn in the order added, reading through r. a
// and b are only valid for the call.
func (s *spillFile) replay(r *bufio.Reader, fn func(a, b []byte) error) error {
	if err := s.w.Flush(); err != nil {
		return err
	}
	if _, err := s.f.Seek(0, io.SeekStart); err != nil {
		return err
	}
	r.Reset(s.f)
	var buf []byte
	for {
		alen, err := binary.ReadUvarint(r)
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
		blen, err := binary.ReadUvarint(r)
		if err != nil {
			return err
		}
		buf = slices.Grow(buf[:0], int(alen+blen))[:alen+blen]
		if _, err := io.ReadFull(r, buf); err != nil {
			return err
		}
		if err := fn(buf[:alen], buf[alen:]); err != nil {
			return err
		}
	}
}

// close removes the file.
func (s *spillFile) close() {
	s.f.Close()
	os.Remove(s.f.Name())
}
