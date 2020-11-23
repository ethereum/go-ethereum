// Copyright 2018 The go-ethereum Authors
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

// Package bmt provides a binary merkle tree implementation used for swarm chunk hash
package bmt

import (
	"fmt"
	"hash"
	"strings"
	"sync"
	"sync/atomic"
)

/*
Binary Merkle Tree Hash is a hash function over arbitrary datachunks of limited size.
It is defined as the root hash of the binary merkle tree built over fixed size segments
of the underlying chunk using any base hash function (e.g., keccak 256 SHA3).
Chunks with data shorter than the fixed size are hashed as if they had zero padding.

BMT hash is used as the chunk hash function in swarm which in turn is the basis for the
128 branching swarm hash http://swarm-guide.readthedocs.io/en/latest/architecture.html#swarm-hash

The BMT is optimal for providing compact inclusion proofs, i.e. prove that a
segment is a substring of a chunk starting at a particular offset.
The size of the underlying segments is fixed to the size of the base hash (called the resolution
of the BMT hash), Using Keccak256 SHA3 hash is 32 bytes, the EVM word size to optimize for on-chain BMT verification
as well as the hash size optimal for inclusion proofs in the merkle tree of the swarm hash.

Two implementations are provided:

* RefHasher is optimized for code simplicity and meant as a reference implementation
  that is simple to understand
* Hasher is optimized for speed taking advantage of concurrency with minimalistic
  control structure to coordinate the concurrent routines

  BMT Hasher implements the following interfaces
	* standard golang hash.Hash - synchronous, reusable
	* SwarmHash - SumWithSpan provided
	* io.Writer - synchronous left-to-right datawriter
	* AsyncWriter - concurrent section writes and asynchronous Sum call
*/

const (
	// SegmentCount is the maximum number of segments of the underlying chunk
	// Should be equal to max-chunk-data-size / hash-size
	SegmentCount = 128
	// PoolSize is the maximum number of bmt trees used by the hashers, i.e,
	// the maximum number of concurrent BMT hashing operations performed by the same hasher
	PoolSize = 8
)

// BaseHasherFunc is a hash.Hash constructor function used for the base hash of the BMT.
// implemented by Keccak256 SHA3 sha3.NewKeccak256
type BaseHasherFunc func() hash.Hash

// Hasher a reusable hasher for fixed maximum size chunks representing a BMT
// - implements the hash.Hash interface
// - reuses a pool of trees for amortised memory allocation and resource control
// - supports order-agnostic concurrent segment writes and section (double segment) writes
//   as well as sequential read and write
// - the same hasher instance must not be called concurrently on more than one chunk
// - the same hasher instance is synchronously reuseable
// - Sum gives back the tree to the pool and guaranteed to leave
//   the tree and itself in a state reusable for hashing a new chunk
// - generates and verifies segment inclusion proofs (TODO:)
type Hasher struct {
	pool *TreePool // BMT resource pool
	bmt  *tree     // prebuilt BMT resource for flowcontrol and proofs
}

// New creates a reusable BMT Hasher that
// pulls a new tree from a resource pool for hashing each chunk
func New(p *TreePool) *Hasher {
	return &Hasher{
		pool: p,
	}
}

// TreePool provides a pool of trees used as resources by the BMT Hasher.
// A tree popped from the pool is guaranteed to have a clean state ready
// for hashing a new chunk.
type TreePool struct {
	lock         sync.Mutex
	c            chan *tree     // the channel to obtain a resource from the pool
	hasher       BaseHasherFunc // base hasher to use for the BMT levels
	SegmentSize  int            // size of leaf segments, stipulated to be = hash size
	SegmentCount int            // the number of segments on the base level of the BMT
	Capacity     int            // pool capacity, controls concurrency
	Depth        int            // depth of the bmt trees = int(log2(segmentCount))+1
	Size         int            // the total length of the data (count * size)
	count        int            // current count of (ever) allocated resources
	zerohashes   [][]byte       // lookup table for predictable padding subtrees for all levels
}

// NewTreePool creates a tree pool with hasher, segment size, segment count and capacity
// on Hasher.getTree it reuses free trees or creates a new one if capacity is not reached
func NewTreePool(hasher BaseHasherFunc, segmentCount, capacity int) *TreePool {
	// initialises the zerohashes lookup table
	depth := calculateDepthFor(segmentCount)
	segmentSize := hasher().Size()
	zerohashes := make([][]byte, depth+1)
	zeros := make([]byte, segmentSize)
	zerohashes[0] = zeros
	h := hasher()
	for i := 1; i < depth+1; i++ {
		zeros = doSum(h, nil, zeros, zeros)
		zerohashes[i] = zeros
	}
	return &TreePool{
		c:            make(chan *tree, capacity),
		hasher:       hasher,
		SegmentSize:  segmentSize,
		SegmentCount: segmentCount,
		Capacity:     capacity,
		Size:         segmentCount * segmentSize,
		Depth:        depth,
		zerohashes:   zerohashes,
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

// Reserve is blocking until it returns an available tree
// it reuses free trees or creates a new one if size is not reached
// TODO: should use a context here
func (p *TreePool) reserve() *tree {
	p.lock.Lock()
	defer p.lock.Unlock()
	var t *tree
	if p.count == p.Capacity {
		return <-p.c
	}
	select {
	case t = <-p.c:
	default:
		t = newTree(p.SegmentSize, p.Depth, p.hasher)
		p.count++
	}
	return t
}

// release gives back a tree to the pool.
// this tree is guaranteed to be in reusable state
func (p *TreePool) release(t *tree) {
	p.c <- t // can never fail ...
}

// tree is a reusable control structure representing a BMT
// organised in a binary tree
// Hasher uses a TreePool to obtain a tree for each chunk hash
// the tree is 'locked' while not in the pool
type tree struct {
	leaves  []*node     // leaf nodes of the tree, other nodes accessible via parent links
	cursor  int         // index of rightmost currently open segment
	offset  int         // offset (cursor position) within currently open segment
	section []byte      // the rightmost open section (double segment)
	result  chan []byte // result channel
	span    []byte      // The span of the data subsumed under the chunk
}

// node is a reuseable segment hasher representing a node in a BMT
type node struct {
	isLeft      bool      // whether it is left side of the parent double segment
	parent      *node     // pointer to parent node in the BMT
	state       int32     // atomic increment impl concurrent boolean toggle
	left, right []byte    // this is where the two children sections are written
	hasher      hash.Hash // preconstructed hasher on nodes
}

// newNode constructs a segment hasher node in the BMT (used by newTree)
func newNode(index int, parent *node, hasher hash.Hash) *node {
	return &node{
		parent: parent,
		isLeft: index%2 == 0,
		hasher: hasher,
	}
}

// Draw draws the BMT (badly)
func (t *tree) draw(hash []byte) string {
	var left, right []string
	var anc []*node
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
		var nodes []*node
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

// newTree initialises a tree by building up the nodes of a BMT
// - segment size is stipulated to be the size of the hash
func newTree(segmentSize, depth int, hashfunc func() hash.Hash) *tree {
	n := newNode(0, nil, hashfunc())
	prevlevel := []*node{n}
	// iterate over levels and creates 2^(depth-level) nodes
	// the 0 level is on double segment sections so we start at depth - 2 since
	count := 2
	for level := depth - 2; level >= 0; level-- {
		nodes := make([]*node, count)
		for i := 0; i < count; i++ {
			parent := prevlevel[i/2]
			var hasher hash.Hash
			if level == 0 {
				hasher = hashfunc()
			}
			nodes[i] = newNode(i, parent, hasher)
		}
		prevlevel = nodes
		count *= 2
	}
	// the datanode level is the nodes on the last level
	return &tree{
		leaves:  prevlevel,
		result:  make(chan []byte),
		section: make([]byte, 2*segmentSize),
	}
}

// methods needed to implement hash.Hash

// Size returns the size
func (h *Hasher) Size() int {
	return h.pool.SegmentSize
}

// BlockSize returns the block size
func (h *Hasher) BlockSize() int {
	return 2 * h.pool.SegmentSize
}

// Sum returns the BMT root hash of the buffer
// using Sum presupposes sequential synchronous writes (io.Writer interface)
// hash.Hash interface Sum method appends the byte slice to the underlying
// data before it calculates and returns the hash of the chunk
// caller must make sure Sum is not called concurrently with Write, writeSection
func (h *Hasher) Sum(b []byte) (s []byte) {
	t := h.getTree()
	// write the last section with final flag set to true
	go h.writeSection(t.cursor, t.section, true, true)
	// wait for the result
	s = <-t.result
	span := t.span
	// release the tree resource back to the pool
	h.releaseTree()
	// b + sha3(span + BMT(pure_chunk))
	if len(span) == 0 {
		return append(b, s...)
	}
	return doSum(h.pool.hasher(), b, span, s)
}

// methods needed to implement the SwarmHash and the io.Writer interfaces

// Write calls sequentially add to the buffer to be hashed,
// with every full segment calls writeSection in a go routine
func (h *Hasher) Write(b []byte) (int, error) {
	l := len(b)
	if l == 0 {
		return 0, nil
	}
	t := h.getTree()
	secsize := 2 * h.pool.SegmentSize
	// calculate length of missing bit to complete current open section
	smax := secsize - t.offset
	// if at the beginning of chunk or middle of the section
	if t.offset < secsize {
		// fill up current segment from buffer
		copy(t.section[t.offset:], b)
		// if input buffer consumed and open section not complete, then
		// advance offset and return
		if smax == 0 {
			smax = secsize
		}
		if l <= smax {
			t.offset += l
			return l, nil
		}
	} else {
		// if end of a section
		if t.cursor == h.pool.SegmentCount*2 {
			return 0, nil
		}
	}
	// read full sections and the last possibly partial section from the input buffer
	for smax < l {
		// section complete; push to tree asynchronously
		go h.writeSection(t.cursor, t.section, true, false)
		// reset section
		t.section = make([]byte, secsize)
		// copy from input buffer at smax to right half of section
		copy(t.section, b[smax:])
		// advance cursor
		t.cursor++
		// smax here represents successive offsets in the input buffer
		smax += secsize
	}
	t.offset = l - smax + secsize
	return l, nil
}

// Reset needs to be called before writing to the hasher
func (h *Hasher) Reset() {
	h.releaseTree()
}

// methods needed to implement the SwarmHash interface

// ResetWithLength needs to be called before writing to the hasher
// the argument is supposed to be the byte slice binary representation of
// the length of the data subsumed under the hash, i.e., span
func (h *Hasher) ResetWithLength(span []byte) {
	h.Reset()
	h.getTree().span = span
}

// releaseTree gives back the Tree to the pool whereby it unlocks
// it resets tree, segment and index
func (h *Hasher) releaseTree() {
	t := h.bmt
	if t == nil {
		return
	}
	h.bmt = nil
	go func() {
		t.cursor = 0
		t.offset = 0
		t.span = nil
		t.section = make([]byte, h.pool.SegmentSize*2)
		select {
		case <-t.result:
		default:
		}
		h.pool.release(t)
	}()
}

// NewAsyncWriter extends Hasher with an interface for concurrent segment/section writes
func (h *Hasher) NewAsyncWriter(double bool) *AsyncHasher {
	secsize := h.pool.SegmentSize
	if double {
		secsize *= 2
	}
	write := func(i int, section []byte, final bool) {
		h.writeSection(i, section, double, final)
	}
	return &AsyncHasher{
		Hasher:  h,
		double:  double,
		secsize: secsize,
		write:   write,
	}
}

// SectionWriter is an asynchronous segment/section writer interface
type SectionWriter interface {
	Reset()                                       // standard init to be called before reuse
	Write(index int, data []byte)                 // write into section of index
	Sum(b []byte, length int, span []byte) []byte // returns the hash of the buffer
	SectionSize() int                             // size of the async section unit to use
}

// AsyncHasher extends BMT Hasher with an asynchronous segment/section writer interface
// AsyncHasher is unsafe and does not check indexes and section data lengths
// it must be used with the right indexes and length and the right number of sections
//
// behaviour is undefined if
// * non-final sections are shorter or longer than secsize
// * if final section does not match length
// * write a section with index that is higher than length/secsize
// * set length in Sum call when length/secsize < maxsec
//
// * if Sum() is not called on a Hasher that is fully written
//   a process will block, can be terminated with Reset
// * it will not leak processes if not all sections are written but it blocks
//   and keeps the resource which can be released calling Reset()
type AsyncHasher struct {
	*Hasher            // extends the Hasher
	mtx     sync.Mutex // to lock the cursor access
	double  bool       // whether to use double segments (call Hasher.writeSection)
	secsize int        // size of base section (size of hash or double)
	write   func(i int, section []byte, final bool)
}

// methods needed to implement AsyncWriter

// SectionSize returns the size of async section unit to use
func (sw *AsyncHasher) SectionSize() int {
	return sw.secsize
}

// Write writes the i-th section of the BMT base
// this function can and is meant to be called concurrently
// it sets max segment threadsafely
func (sw *AsyncHasher) Write(i int, section []byte) {
	sw.mtx.Lock()
	defer sw.mtx.Unlock()
	t := sw.getTree()
	// cursor keeps track of the rightmost section written so far
	// if index is lower than cursor then just write non-final section as is
	if i < t.cursor {
		// if index is not the rightmost, safe to write section
		go sw.write(i, section, false)
		return
	}
	// if there is a previous rightmost section safe to write section
	if t.offset > 0 {
		if i == t.cursor {
			// i==cursor implies cursor was set by Hash call so we can write section as final one
			// since it can be shorter, first we copy it to the padded buffer
			t.section = make([]byte, sw.secsize)
			copy(t.section, section)
			go sw.write(i, t.section, true)
			return
		}
		// the rightmost section just changed, so we write the previous one as non-final
		go sw.write(t.cursor, t.section, false)
	}
	// set i as the index of the righmost section written so far
	// set t.offset to cursor*secsize+1
	t.cursor = i
	t.offset = i*sw.secsize + 1
	t.section = make([]byte, sw.secsize)
	copy(t.section, section)
}

// Sum can be called any time once the length and the span is known
// potentially even before all segments have been written
// in such cases Sum will block until all segments are present and
// the hash for the length can be calculated.
//
// b: digest is appended to b
// length: known length of the input (unsafe; undefined if out of range)
// meta: metadata to hash together with BMT root for the final digest
//   e.g., span for protection against existential forgery
func (sw *AsyncHasher) Sum(b []byte, length int, meta []byte) (s []byte) {
	sw.mtx.Lock()
	t := sw.getTree()
	if length == 0 {
		sw.mtx.Unlock()
		s = sw.pool.zerohashes[sw.pool.Depth]
	} else {
		// for non-zero input the rightmost section is written to the tree asynchronously
		// if the actual last section has been written (t.cursor == length/t.secsize)
		maxsec := (length - 1) / sw.secsize
		if t.offset > 0 {
			go sw.write(t.cursor, t.section, maxsec == t.cursor)
		}
		// set cursor to maxsec so final section is written when it arrives
		t.cursor = maxsec
		t.offset = length
		result := t.result
		sw.mtx.Unlock()
		// wait for the result or reset
		s = <-result
	}
	// relesase the tree back to the pool
	sw.releaseTree()
	// if no meta is given just append digest to b
	if len(meta) == 0 {
		return append(b, s...)
	}
	// hash together meta and BMT root hash using the pools
	return doSum(sw.pool.hasher(), b, meta, s)
}

// writeSection writes the hash of i-th section into level 1 node of the BMT tree
func (h *Hasher) writeSection(i int, section []byte, double bool, final bool) {
	// select the leaf node for the section
	var n *node
	var isLeft bool
	var hasher hash.Hash
	var level int
	t := h.getTree()
	if double {
		level++
		n = t.leaves[i]
		hasher = n.hasher
		isLeft = n.isLeft
		n = n.parent
		// hash the section
		section = doSum(hasher, nil, section)
	} else {
		n = t.leaves[i/2]
		hasher = n.hasher
		isLeft = i%2 == 0
	}
	// write hash into parent node
	if final {
		// for the last segment use writeFinalNode
		h.writeFinalNode(level, n, hasher, isLeft, section)
	} else {
		h.writeNode(n, hasher, isLeft, section)
	}
}

// writeNode pushes the data to the node
// if it is the first of 2 sisters written, the routine terminates
// if it is the second, it calculates the hash and writes it
// to the parent node recursively
// since hashing the parent is synchronous the same hasher can be used
func (h *Hasher) writeNode(n *node, bh hash.Hash, isLeft bool, s []byte) {
	level := 1
	for {
		// at the root of the bmt just write the result to the result channel
		if n == nil {
			h.getTree().result <- s
			return
		}
		// otherwise assign child hash to left or right segment
		if isLeft {
			n.left = s
		} else {
			n.right = s
		}
		// the child-thread first arriving will terminate
		if n.toggle() {
			return
		}
		// the thread coming second now can be sure both left and right children are written
		// so it calculates the hash of left|right and pushes it to the parent
		s = doSum(bh, nil, n.left, n.right)
		isLeft = n.isLeft
		n = n.parent
		level++
	}
}

// writeFinalNode is following the path starting from the final datasegment to the
// BMT root via parents
// for unbalanced trees it fills in the missing right sister nodes using
// the pool's lookup table for BMT subtree root hashes for all-zero sections
// otherwise behaves like `writeNode`
func (h *Hasher) writeFinalNode(level int, n *node, bh hash.Hash, isLeft bool, s []byte) {

	for {
		// at the root of the bmt just write the result to the result channel
		if n == nil {
			if s != nil {
				h.getTree().result <- s
			}
			return
		}
		var noHash bool
		if isLeft {
			// coming from left sister branch
			// when the final section's path is going via left child node
			// we include an all-zero subtree hash for the right level and toggle the node.
			n.right = h.pool.zerohashes[level]
			if s != nil {
				n.left = s
				// if a left final node carries a hash, it must be the first (and only thread)
				// so the toggle is already in passive state no need no call
				// yet thread needs to carry on pushing hash to parent
				noHash = false
			} else {
				// if again first thread then propagate nil and calculate no hash
				noHash = n.toggle()
			}
		} else {
			// right sister branch
			if s != nil {
				// if hash was pushed from right child node, write right segment change state
				n.right = s
				// if toggle is true, we arrived first so no hashing just push nil to parent
				noHash = n.toggle()

			} else {
				// if s is nil, then thread arrived first at previous node and here there will be two,
				// so no need to do anything and keep s = nil for parent
				noHash = true
			}
		}
		// the child-thread first arriving will just continue resetting s to nil
		// the second thread now can be sure both left and right children are written
		// it calculates the hash of left|right and pushes it to the parent
		if noHash {
			s = nil
		} else {
			s = doSum(bh, nil, n.left, n.right)
		}
		// iterate to parent
		isLeft = n.isLeft
		n = n.parent
		level++
	}
}

// getTree obtains a BMT resource by reserving one from the pool and assigns it to the bmt field
func (h *Hasher) getTree() *tree {
	if h.bmt != nil {
		return h.bmt
	}
	t := h.pool.reserve()
	h.bmt = t
	return t
}

// atomic bool toggle implementing a concurrent reusable 2-state object
// atomic addint with %2 implements atomic bool toggle
// it returns true if the toggler just put it in the active/waiting state
func (n *node) toggle() bool {
	return atomic.AddInt32(&n.state, 1)%2 == 1
}

// calculates the hash of the data using hash.Hash
func doSum(h hash.Hash, b []byte, data ...[]byte) []byte {
	h.Reset()
	for _, v := range data {
		h.Write(v)
	}
	return h.Sum(b)
}

// hashstr is a pretty printer for bytes used in tree.draw
func hashstr(b []byte) string {
	end := len(b)
	if end > 4 {
		end = 4
	}
	return fmt.Sprintf("%x", b[:end])
}

// calculateDepthFor calculates the depth (number of levels) in the BMT tree
func calculateDepthFor(n int) (d int) {
	c := 2
	for ; c < n; c *= 2 {
		d++
	}
	return d + 1
}
