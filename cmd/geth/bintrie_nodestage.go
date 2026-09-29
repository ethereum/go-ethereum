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
	"path/filepath"
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
	depths map[int]*stagedDepth
	head   [2 * binary.MaxVarintLen64]byte // a record's length prefix
}

type stagedDepth struct {
	f *os.File
	w *bufio.Writer
}

// newNodeStager stages its files in a fresh directory under tmpDir.
func newNodeStager(tmpDir string) (*nodeStager, error) {
	dir, err := os.MkdirTemp(tmpDir, "bintrie-nodes-")
	if err != nil {
		return nil, err
	}
	return &nodeStager{dir: dir, depths: make(map[int]*stagedDepth)}, nil
}

// nodeDepth is the bit count encodePath leads a node's path with. The root's
// empty path sorts before every other one and gets depth -1.
func nodeDepth(path []byte) int {
	if len(path) < 2 {
		return -1
	}
	return int(binary.BigEndian.Uint16(path))
}

// add appends one node to its depth's file, path and blob length-prefixed.
// Neither needs to outlive the call.
func (s *nodeStager) add(path, blob []byte) error {
	depth := nodeDepth(path)
	d := s.depths[depth]
	if d == nil {
		f, err := os.Create(filepath.Join(s.dir, fmt.Sprintf("depth-%d", depth)))
		if err != nil {
			return err
		}
		d = &stagedDepth{f: f, w: bufio.NewWriterSize(f, 256<<10)}
		s.depths[depth] = d
	}
	n := binary.PutUvarint(s.head[:], uint64(len(path)))
	n += binary.PutUvarint(s.head[n:], uint64(len(blob)))
	if _, err := d.w.Write(s.head[:n]); err != nil {
		return err
	}
	if _, err := d.w.Write(path); err != nil {
		return err
	}
	_, err := d.w.Write(blob)
	return err
}

// replay feeds every staged node to write in database key order: depth by
// depth, each depth in the order its nodes were added. path and blob are only
// valid for the call.
func (s *nodeStager) replay(write func(path, blob []byte) error) error {
	var (
		r   = bufio.NewReaderSize(nil, 1<<20)
		buf []byte
	)
	for _, depth := range slices.Sorted(maps.Keys(s.depths)) {
		d := s.depths[depth]
		if err := d.w.Flush(); err != nil {
			return err
		}
		if _, err := d.f.Seek(0, io.SeekStart); err != nil {
			return err
		}
		r.Reset(d.f)
		for {
			plen, err := binary.ReadUvarint(r)
			if err == io.EOF {
				break
			}
			if err != nil {
				return err
			}
			blen, err := binary.ReadUvarint(r)
			if err != nil {
				return err
			}
			buf = slices.Grow(buf[:0], int(plen+blen))[:plen+blen]
			if _, err := io.ReadFull(r, buf); err != nil {
				return err
			}
			if err := write(buf[:plen], buf[plen:]); err != nil {
				return err
			}
		}
	}
	return nil
}

// close removes the staging files.
func (s *nodeStager) close() {
	for _, d := range s.depths {
		d.f.Close()
	}
	os.RemoveAll(s.dir)
}
