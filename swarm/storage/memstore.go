// Copyright 2016 The go-ethereum Authors
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

// memory storage layer for the package blockhash

package storage

import (
	"sync"
)

const (
	memTreeLW              = 2  // log2(subtree count) of the subtrees
	memTreeFLW             = 14 // log2(subtree count) of the root layer
	dbForceUpdateAccessCnt = 1000
	defaultCacheCapacity   = 5000
)

type MemStore struct {
	memtree            *memTree
	entryCnt, capacity uint   // stored entries
	accessCnt          uint64 // access counter; oldest is thrown away when full
	dbAccessCnt        uint64
	dbStore            *DbStore
	lock               sync.Mutex
}

/*
a hash prefix subtree containing subtrees or one storage entry (but never both)

- access[0] stores the smallest (oldest) access count value in this subtree
- if it contains more subtrees and its subtree count is at least 4, access[1:2]
  stores the smallest access count in the first and second halves of subtrees
  (so that access[0] = min(access[1], access[2])
- likewise, if subtree count is at least 8,
  access[1] = min(access[3], access[4])
  access[2] = min(access[5], access[6])
  (access[] is a binary tree inside the multi-bit leveled hash tree)
*/

func NewMemStore(d *DbStore, capacity uint) (m *MemStore) {
	m = &MemStore{}
	m.memtree = newMemTree(memTreeFLW, nil, 0)
	m.dbStore = d
	m.setCapacity(capacity)
	return
}

type memTree struct {
	subtree   []*memTree
	parent    *memTree
	parentIdx uint

	bits  uint // log2(subtree count)
	width uint // subtree count

	entry        *Chunk // if subtrees are present, entry should be nil
	lastDBaccess uint64
	access       []uint64
}

func newMemTree(b uint, parent *memTree, pidx uint) (node *memTree) {
	node = new(memTree)
	node.bits = b
	node.width = 1 << uint(b)
	node.subtree = make([]*memTree, node.width)
	node.access = make([]uint64, node.width-1)
	node.parent = parent
	node.parentIdx = pidx
	if parent != nil {
		parent.subtree[pidx] = node
	}

	return node
}

func (node *memTree) updateAccess(a uint64) {
	aidx := uint(0)
	var aa uint64
	oa := node.access[0]
	for node.access[aidx] == oa {
		node.access[aidx] = a
		if aidx > 0 {
			aa = node.access[((aidx-1)^1)+1]
			aidx = (aidx - 1) >> 1
		} else {
			pidx := node.parentIdx
			node = node.parent
			if node == nil {
				return
			}
			nn := node.subtree[pidx^1]
			if nn != nil {
				aa = nn.access[0]
			} else {
				aa = 0
			}
			aidx = (node.width + pidx - 2) >> 1
		}

		if (aa != 0) && (aa < a) {
			a = aa
		}
	}
}

func (s *MemStore) setCapacity(c uint) {
	s.lock.Lock()
	defer s.lock.Unlock()

	for c < s.entryCnt {
		s.removeOldest()
	}
	s.capacity = c
}

func (s *MemStore) getEntryCnt() uint {
	return s.entryCnt
}

// entry (not its copy) is going to be in MemStore
func (s *MemStore) Put(entry *Chunk) {
	if s.capacity == 0 {
		return
	}

	s.lock.Lock()
	defer s.lock.Unlock()

	if s.entryCnt >= s.capacity {
		s.removeOldest()
	}

	s.accessCnt++

	node := s.memtree
	bitpos := uint(0)
	for node.entry == nil {
		l := entry.Key.bits(bitpos, node.bits)
		st := node.subtree[l]
		if st == nil {
			st = newMemTree(memTreeLW, node, l)
			bitpos += node.bits
			node = st
			break
		}
		bitpos += node.bits
		node = st
	}

	if node.entry != nil {

		if node.entry.Key.isEqual(entry.Key) {
			node.updateAccess(s.accessCnt)
			if entry.SData == nil {
				entry.Size = node.entry.Size
				entry.SData = node.entry.SData
			}
			if entry.Req == nil {
				entry.Req = node.entry.Req
			}
			entry.C = node.entry.C
			node.entry = entry
			return
		}

		for node.entry != nil {

			l := node.entry.Key.bits(bitpos, node.bits)
			st := node.subtree[l]
			if st == nil {
				st = newMemTree(memTreeLW, node, l)
			}
			st.entry = node.entry
			node.entry = nil
			st.updateAccess(node.access[0])

			l = entry.Key.bits(bitpos, node.bits)
			st = node.subtree[l]
			if st == nil {
				st = newMemTree(memTreeLW, node, l)
			}
			bitpos += node.bits
			node = st

		}
	}

	node.entry = entry
	node.lastDBaccess = s.dbAccessCnt
	node.updateAccess(s.accessCnt)
	s.entryCnt++

	return
}

func (s *MemStore) Get(hash Key) (chunk *Chunk, err error) {
	s.lock.Lock()
	defer s.lock.Unlock()

	node := s.memtree
	bitpos := uint(0)
	for node.entry == nil {
		l := hash.bits(bitpos, node.bits)
		st := node.subtree[l]
		if st == nil {
			return nil, notFound
		}
		bitpos += node.bits
		node = st
	}

	if node.entry.Key.isEqual(hash) {
		s.accessCnt++
		node.updateAccess(s.accessCnt)
		chunk = node.entry
		if s.dbAccessCnt-node.lastDBaccess > dbForceUpdateAccessCnt {
			s.dbAccessCnt++
			node.lastDBaccess = s.dbAccessCnt
			if s.dbStore != nil {
				s.dbStore.updateAccessCnt(hash)
			}
		}
	} else {
		err = notFound
	}

	return
}

func (s *MemStore) removeOldest() {
	node := s.memtree

	for node.entry == nil {

		aidx := uint(0)
		av := node.access[aidx]

		for aidx < node.width/2-1 {
			if av == node.access[aidx*2+1] {
				node.access[aidx] = node.access[aidx*2+2]
				aidx = aidx*2 + 1
			} else if av == node.access[aidx*2+2] {
				node.access[aidx] = node.access[aidx*2+1]
				aidx = aidx*2 + 2
			} else {
				panic(nil)
			}
		}
		pidx := aidx*2 + 2 - node.width
		if (node.subtree[pidx] != nil) && (av == node.subtree[pidx].access[0]) {
			if node.subtree[pidx+1] != nil {
				node.access[aidx] = node.subtree[pidx+1].access[0]
			} else {
				node.access[aidx] = 0
			}
		} else if (node.subtree[pidx+1] != nil) && (av == node.subtree[pidx+1].access[0]) {
			if node.subtree[pidx] != nil {
				node.access[aidx] = node.subtree[pidx].access[0]
			} else {
				node.access[aidx] = 0
			}
			pidx++
		} else {
			panic(nil)
		}

		//fmt.Println(pidx)
		node = node.subtree[pidx]

	}

	if node.entry.dbStored != nil {
		<-node.entry.dbStored
	}

	if node.entry.SData != nil {
		node.entry = nil
		s.entryCnt--
	}

	node.access[0] = 0

	//---

	aidx := uint(0)
	for {
		aa := node.access[aidx]
		if aidx > 0 {
			aidx = (aidx - 1) >> 1
		} else {
			pidx := node.parentIdx
			node = node.parent
			if node == nil {
				return
			}
			aidx = (node.width + pidx - 2) >> 1
		}
		if (aa != 0) && ((aa < node.access[aidx]) || (node.access[aidx] == 0)) {
			node.access[aidx] = aa
		}
	}
}
