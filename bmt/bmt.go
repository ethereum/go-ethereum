// Copyright 2017 The go-ethereum Authors
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

// Package bmt provides a binary merkle tree implementation
package bmt

import (
	"fmt"
	"hash"
	"io"
	"strings"
	"sync"
	"sync/atomic"
)

/*
Binary Merkle Tree Hash is a hash function over arbitrary datachunks of limited size
It is defined as the root hash of the binary merkle tree built over fixed size segments
of the underlying chunk using any base hash function (e.g keccak 256 SHA3)

It is used as the chunk hash function in swarm which in turn is the basis for the
128 branching swarm hash http://swarm-guide.readthedocs.io/en/latest/architecture.html#swarm-hash

The BMT is optimal for providing compact inclusion proofs, i.e. prove that a
segment is a substring of a chunk starting at a particular offset
The size of the underlying segments is fixed at 32 bytes (called the resolution
of the BMT hash), the EVM word size to optimize for on-chain BMT verification
as well as the hash size optimal for inclusion proofs in the merkle tree of the swarm hash.

Two implementations are provided:

* RefHasher is optimized for code simplicity and meant as a reference implementation
* Hasher is optimized for speed taking advantage of concurrency with minimalistic
  control structure to coordinate the concurrent routines
  It implements the ChunkHash interface as well as the go standard hash.Hash interface

*/

const (
	// DefaultSegmentCount is the maximum number of segments of the underlying chunk
	DefaultSegmentCount = 128 // Should be equal to storage.DefaultBranches
	// DefaultPoolSize is the maximum number of bmt trees used by the hashers, i.e,
	// the maximum number of concurrent BMT hashing operations performed by the same hasher
	DefaultPoolSize = 8
)

// BaseHasher is a hash.Hash constructor function used for the base hash of the  BMT.
type BaseHasher func() hash.Hash

// Hasher a reusable hasher for fixed maximum size chunks representing a BMT
// implements the hash.Hash interface
// reuse pool of Tree-s for amortised memory allocation and resource control
// supports order-agnostic concurrent segment writes
// as well as sequential read and write
// can not be called concurrently on more than one chunk
// can be further appended after Sum
// Reset gives back the Tree to the pool and guaranteed to leave
// the tree and itself in a state reusable for hashing a new chunk
type Hasher struct {
	pool        *TreePool   // BMT resource pool
	bmt         *Tree       // prebuilt BMT resource for flowcontrol and proofs
	blocksize   int         // segment size (size of hash) also for hash.Hash
	count       int         // segment count
	size        int         // for hash.Hash same as hashsize
	cur         int         // cursor position for rightmost currently open chunk
	segment     []byte      // the rightmost open segment (not complete)
	depth       int         // index of last level
	result      chan []byte // result channel
	hash        []byte      // to record the result
	max         int32       // max segments for SegmentWriter interface
	blockLength []byte      // The block length that needes to be added in Sum
}

// New creates a reusable Hasher
// implements the hash.Hash interface
// pulls a new Tree from a resource pool for hashing each chunk
func New(p *TreePool) *Hasher {
	return &Hasher{
		pool:      p,
		depth:     depth(p.SegmentCount),
		size:      p.SegmentSize,
		blocksize: p.SegmentSize,
		count:     p.SegmentCount,
		result:    make(chan []byte),
	}
}

// Node is a reuseable segment hasher representing a node in a BMT
// it allows for continued writes after a Sum
// and is left in completely reusable state after Reset
type Node struct {
	level, index int   // position of node for information/logging only
	initial      bool  // first and last node
	root         bool  // whether the node is root to a smaller BMT
	isLeft       bool  // whether it is left side of the parent double segment
	unbalanced   bool  // indicates if a node has only the left segment
	parent       *Node // BMT connections
	state        int32 // atomic increment impl concurrent boolean toggle
	left, right  []byte
}

// NewNode constructor for segment hasher nodes in the BMT
func NewNode(level, index int, parent *Node) *Node {
	return &Node{
		parent:  parent,
		level:   level,
		index:   index,
		initial: index == 0,
		isLeft:  index%2 == 0,
	}
}

// TreePool provides a pool of Trees used as resources by Hasher
// a Tree popped from the pool is guaranteed to have clean state
// for hashing a new chunk
// Hasher Reset releases the Tree to the pool
type TreePool struct {
	lock         sync.Mutex
	c            chan *Tree
	hasher       BaseHasher
	SegmentSize  int
	SegmentCount int
	Capacity     int
	count        int
}

// NewTreePool creates a Tree pool with hasher, segment size, segment count and capacity
// on GetTree it reuses free Trees or creates a new one if size is not reached
func NewTreePool(hasher BaseHasher, segmentCount, capacity int) *TreePool {
	return &TreePool{
		c:            make(chan *Tree, capacity),
		hasher:       hasher,
		SegmentSize:  hasher().Size(),
		SegmentCount: segmentCount,
		Capacity:     capacity,
	}
}

// Drain drains the pool until it has no more than n resources
func (p *TreePool) Drain(n int) {
	p.lock.Lock()
	defer p.lock.Unlock()
	for len(p.c) > n {
		<-p.c
		p.count--
	}
}

// Reserve is blocking until it returns an available Tree
// it reuses free Trees or creates a new one if size is not reached
func (p *TreePool) Reserve() *Tree {
	p.lock.Lock()
	defer p.lock.Unlock()
	var t *Tree
	if p.count == p.Capacity {
		return <-p.c
	}
	select {
	case t = <-p.c:
	default:
		t = NewTree(p.hasher, p.SegmentSize, p.SegmentCount)
		p.count++
	}
	return t
}

// Release gives back a Tree to the pool.
// This Tree is guaranteed to be in reusable state
// does not need locking
func (p *TreePool) Release(t *Tree) {
	p.c <- t // can never fail but...
}

// Tree is a reusable control structure representing a BMT
// organised in a binary tree
// Hasher uses a TreePool to pick one for each chunk hash
// the Tree is 'locked' while not in the pool
type Tree struct {
	leaves []*Node
}

// Draw draws the BMT (badly)
func (t *Tree) Draw(hash []byte, d int) string {
	var left, right []string
	var anc []*Node
	for i, n := range t.leaves {
		left = append(left, fmt.Sprintf("%v", hashstr(n.left)))
		if i%2 == 0 {
			anc = append(anc, n.parent)
		}
		right = append(right, fmt.Sprintf("%v", hashstr(n.right)))
	}
	anc = t.leaves
	var hashes [][]string
	for l := 0; len(anc) > 0; l++ {
		var nodes []*Node
		hash := []string{""}
		for i, n := range anc {
			hash = append(hash, fmt.Sprintf("%v|%v", hashstr(n.left), hashstr(n.right)))
			if i%2 == 0 && n.parent != nil {
				nodes = append(nodes, n.parent)
			}
		}
		hash = append(hash, "")
		hashes = append(hashes, hash)
		anc = nodes
	}
	hashes = append(hashes, []string{"", fmt.Sprintf("%v", hashstr(hash)), ""})
	total := 60
	del := "                             "
	var rows []string
	for i := len(hashes) - 1; i >= 0; i-- {
		var textlen int
		hash := hashes[i]
		for _, s := range hash {
			textlen += len(s)
		}
		if total < textlen {
			total = textlen + len(hash)
		}
		delsize := (total - textlen) / (len(hash) - 1)
		if delsize > len(del) {
			delsize = len(del)
		}
		row := fmt.Sprintf("%v: %v", len(hashes)-i-1, strings.Join(hash, del[:delsize]))
		rows = append(rows, row)

	}
	rows = append(rows, strings.Join(left, "  "))
	rows = append(rows, strings.Join(right, "  "))
	return strings.Join(rows, "\n") + "\n"
}

// NewTree initialises the Tree by building up the nodes of a BMT
// segment size is stipulated to be the size of the hash
// segmentCount needs to be positive integer and does not need to be
// a power of two and can even be an odd number
// segmentSize * segmentCount determines the maximum chunk size
// hashed using the tree
func NewTree(hasher BaseHasher, segmentSize, segmentCount int) *Tree {
	n := NewNode(0, 0, nil)
	n.root = true
	prevlevel := []*Node{n}
	// iterate over levels and creates 2^level nodes
	level := 1
	count := 2
	for d := 1; d <= depth(segmentCount); d++ {
		nodes := make([]*Node, count)
		for i := 0; i < len(nodes); i++ {
			parent := prevlevel[i/2]
			t := NewNode(level, i, parent)
			nodes[i] = t
		}
		prevlevel = nodes
		level++
		count *= 2
	}
	// the datanode level is the nodes on the last level where
	return &Tree{
		leaves: prevlevel,
	}
}

// methods needed by hash.Hash

// Size returns the size
func (h *Hasher) Size() int {
	return h.size
}

// BlockSize returns the block size
func (h *Hasher) BlockSize() int {
	return h.blocksize
}

// Sum returns the hash of the buffer
// hash.Hash interface Sum method appends the byte slice to the underlying
// data before it calculates and returns the hash of the chunk
func (h *Hasher) Sum(b []byte) (r []byte) {
	t := h.bmt
	i := h.cur
	n := t.leaves[i]
	j := i
	// must run strictly before all nodes calculate
	// datanodes are guaranteed to have a parent
	if len(h.segment) > h.size && i > 0 && n.parent != nil {
		n = n.parent
	} else {
		i *= 2
	}
	d := h.finalise(n, i)
	h.writeSegment(j, h.segment, d)
	c := <-h.result
	h.releaseTree()

	// sha3(length + BMT(pure_chunk))
	if h.blockLength == nil {
		return c
	}
	res := h.pool.hasher()
	res.Reset()
	res.Write(h.blockLength)
	res.Write(c)
	return res.Sum(nil)
}

// Hasher implements the SwarmHash interface

// Hash waits for the hasher result and returns it
// caller must call this on a BMT Hasher being written to
func (h *Hasher) Hash() []byte {
	return <-h.result
}

// Hasher implements the io.Writer interface

// Write fills the buffer to hash
// with every full segment complete launches a hasher go routine
// that shoots up the BMT
func (h *Hasher) Write(b []byte) (int, error) {
	l := len(b)
	if l <= 0 {
		return 0, nil
	}
	s := h.segment
	i := h.cur
	count := (h.count + 1) / 2
	need := h.count*h.size - h.cur*2*h.size
	size := h.size
	if need > size {
		size *= 2
	}
	if l < need {
		need = l
	}
	// calculate missing bit to complete current open segment
	rest := size - len(s)
	if need < rest {
		rest = need
	}
	s = append(s, b[:rest]...)
	need -= rest
	// read full segments and the last possibly partial segment
	for need > 0 && i < count-1 {
		// push all finished chunks we read
		h.writeSegment(i, s, h.depth)
		need -= size
		if need < 0 {
			size += need
		}
		s = b[rest : rest+size]
		rest += size
		i++
	}
	h.segment = s
	h.cur = i
	// otherwise, we can assume len(s) == 0, so all buffer is read and chunk is not yet full
	return l, nil
}

// Hasher implements the io.ReaderFrom interface

// ReadFrom reads from io.Reader and appends to the data to hash using Write
// it reads so that chunk to hash is maximum length or reader reaches EOF
// caller must Reset the hasher prior to call
func (h *Hasher) ReadFrom(r io.Reader) (m int64, err error) {
	bufsize := h.size*h.count - h.size*h.cur - len(h.segment)
	buf := make([]byte, bufsize)
	var read int
	for {
		var n int
		n, err = r.Read(buf)
		read += n
		if err == io.EOF || read == len(buf) {
			hash := h.Sum(buf[:n])
			if read == len(buf) {
				err = NewEOC(hash)
			}
			break
		}
		if err != nil {
			break
		}
		n, err = h.Write(buf[:n])
		if err != nil {
			break
		}
	}
	return int64(read), err
}

// Reset needs to be called before writing to the hasher
func (h *Hasher) Reset() {
	h.getTree()
	h.blockLength = nil
}

// Hasher implements the SwarmHash interface

// ResetWithLength needs to be called before writing to the hasher
// the argument is supposed to be the byte slice binary representation of
// the length of the data subsumed under the hash
func (h *Hasher) ResetWithLength(l []byte) {
	h.Reset()
	h.blockLength = l
}

// Release gives back the Tree to the pool whereby it unlocks
// it resets tree, segment and index
func (h *Hasher) releaseTree() {
	if h.bmt != nil {
		n := h.bmt.leaves[h.cur]
		for ; n != nil; n = n.parent {
			n.unbalanced = false
			if n.parent != nil {
				n.root = false
			}
		}
		h.pool.Release(h.bmt)
		h.bmt = nil

	}
	h.cur = 0
	h.segment = nil
}

func (h *Hasher) writeSegment(i int, s []byte, d int) {
	hash := h.pool.hasher()
	n := h.bmt.leaves[i]

	if len(s) > h.size && n.parent != nil {
		go func() {
			hash.Reset()
			hash.Write(s)
			s = hash.Sum(nil)

			if n.root {
				h.result <- s
				return
			}
			h.run(n.parent, hash, d, n.index, s)
		}()
		return
	}
	go h.run(n, hash, d, i*2, s)
}

func (h *Hasher) run(n *Node, hash hash.Hash, d int, i int, s []byte) {
	isLeft := i%2 == 0
	for {
		if isLeft {
			n.left = s
		} else {
			n.right = s
		}
		if !n.unbalanced && n.toggle() {
			return
		}
		if !n.unbalanced || !isLeft || i == 0 && d == 0 {
			hash.Reset()
			hash.Write(n.left)
			hash.Write(n.right)
			s = hash.Sum(nil)

		} else {
			s = append(n.left, n.right...)
		}

		h.hash = s
		if n.root {
			h.result <- s
			return
		}

		isLeft = n.isLeft
		n = n.parent
		i++
	}
}

// getTree obtains a BMT resource by reserving one from the pool
func (h *Hasher) getTree() *Tree {
	if h.bmt != nil {
		return h.bmt
	}
	t := h.pool.Reserve()
	h.bmt = t
	return t
}

// atomic bool toggle implementing a concurrent reusable 2-state object
// atomic addint with %2 implements atomic bool toggle
// it returns true if the toggler just put it in the active/waiting state
func (n *Node) toggle() bool {
	return atomic.AddInt32(&n.state, 1)%2 == 1
}

func hashstr(b []byte) string {
	end := len(b)
	if end > 4 {
		end = 4
	}
	return fmt.Sprintf("%x", b[:end])
}

func depth(n int) (d int) {
	for l := (n - 1) / 2; l > 0; l /= 2 {
		d++
	}
	return d
}

// finalise is following the zigzags on the tree belonging
// to the final datasegment
func (h *Hasher) finalise(n *Node, i int) (d int) {
	isLeft := i%2 == 0
	for {
		// when the final segment's path is going via left segments
		// the incoming data is pushed to the parent upon pulling the left
		// we do not need toggle the state since this condition is
		// detectable
		n.unbalanced = isLeft
		n.right = nil
		if n.initial {
			n.root = true
			return d
		}
		isLeft = n.isLeft
		n = n.parent
		d++
	}
}

// EOC (end of chunk) implements the error interface
type EOC struct {
	Hash []byte // read the hash of the chunk off the error
}

// Error returns the error string
func (e *EOC) Error() string {
	return fmt.Sprintf("hasher limit reached, chunk hash: %x", e.Hash)
}

// NewEOC creates new end of chunk error with the hash
func NewEOC(hash []byte) *EOC {
	return &EOC{hash}
}
