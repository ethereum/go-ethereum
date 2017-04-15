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

// provides a binary merkle tree implementation
package bmt

import (
	"fmt"
	"hash"
	"io"
	"strings"
	"sync"
	"sync/atomic"
)

const (
	DefaultSegmentCount int = 128 // Should be equal to storage.DefaultBranches
	DefaultPoolSize         = 8
)

type Hasher func() hash.Hash

type EOC struct {
	Hash []byte // read the hash of the chunk off the error
}

// a reusable hasher for fixed maximum size chunks representing a BMT
// implements the hash.Hash interface
// reuse pool of BMTree-s for amortised memory allocation and resource control
// supports order-agnostic concurrent segment writes
// as well as sequential read and write
// can not be called concurrently on more than one chunk
// can be further appended after Sum
// Reset gives back the BMTree to the pool and guaranteed to leave
// the tree and itself in a state reusable for hashing a new chunk
type BMTHasher struct {
	pool        *BMTreePool // BMT resource pool
	bmt         *BMTree     // prebuilt BMT resource for flowcontrol and proofs
	blocksize   int         // segment size (size of hash) also for hash.Hash
	count       int         // segment count
	size        int         // for hash.Hash same as hashsize
	cur         int         // cursor position for righmost currently open chunk
	segment     []byte      // the rightmost open segment (not complete)
	depth       int         // index of last level
	result      chan []byte // result channel
	hash        []byte      // to record the result
	max         int32       // max segments for SegmentWriter interface
	blockLength []byte      // The block length that needes to be added in Sum
}

// creates a reusable BMTHasher
// implements the hash.Hash interface
// pulls a new BMTree from a resource pool for hashing each chunk
func NewBMTHasher(p *BMTreePool) *BMTHasher {
	return &BMTHasher{
		pool:      p,
		depth:     depth(p.SegmentCount),
		size:      p.SegmentSize,
		blocksize: p.SegmentSize,
		count:     p.SegmentCount,
		result:    make(chan []byte),
	}
}

// a reuseable segment hasher representing a node in a BMT
// it allows for continued writes after a Sum
// and is left in completely reusable state after Reset
type BMTNode struct {
	level, index int      // position of node for information/logging only
	initial      bool     // first and last node
	root         bool     // whether the node is root to a smaller BMT
	isLeft       bool     // whether it is left side of the parent double segment
	unbalanced   bool     // indicates if a node has only the left segment
	parent       *BMTNode // BMT connections
	state        int32    // atomic increment impl concurrent boolean toggle
	left, right  []byte
}

// constructor for segment hasher nodes in the BMT
func NewBMTNode(level, index int, parent *BMTNode) *BMTNode {
	self := &BMTNode{
		parent:  parent,
		level:   level,
		index:   index,
		initial: index == 0,
		isLeft:  index%2 == 0,
	}
	return self
}

// provides a pool of BMTrees used as resources by BMTHasher
// a BMTree popped from the pool is guaranteed to have clean state
// for hashing a new chunk
// BMTHasher Reset releases the BMTree to the pool
type BMTreePool struct {
	lock         sync.Mutex
	c            chan *BMTree
	hasher       Hasher
	SegmentSize  int
	SegmentCount int
	Capacity     int
	count        int
}

// create a BMTree pool with hasher, segment size, segment count and capacity
// on GetBMTree it reuses free BMTrees or creates a new one if size is not reached
func NewBMTreePool(hasher Hasher, segmentCount, capacity int) *BMTreePool {
	segmentSize := hasher().Size()
	return &BMTreePool{
		c:            make(chan *BMTree, capacity),
		hasher:       hasher,
		SegmentSize:  segmentSize,
		SegmentCount: segmentCount,
		Capacity:     capacity,
	}
}

// drains the pool uptil it has no more than n resources
func (self *BMTreePool) Drain(n int) {
	self.lock.Lock()
	defer self.lock.Unlock()
	for len(self.c) > n {
		select {
		case <-self.c:
			self.count--
		default:
		}
	}
}

// blocks until it returns an available BMTree
// it reuses free BMTrees or creates a new one if size is not reached
func (self *BMTreePool) Reserve() *BMTree {
	self.lock.Lock()
	defer self.lock.Unlock()
	var t *BMTree
	if self.count == self.Capacity {
		t = <-self.c
	} else {
		select {
		case t = <-self.c:
		default:
			t = NewBMTree(self.hasher, self.SegmentSize, self.SegmentCount)
			self.count++
		}
	}
	return t
}

// Releases a BMTree to the pool. the BMTree is guaranteed to be in reusable state
// does not need locking
func (self *BMTreePool) Release(t *BMTree) {
	self.c <- t // can never fail but...
}

// reusable control structure representing a BMT
// organised in a binary tree
// BMTHasher uses a BMTreePool to pick one for each chunk hash
// the BMTree is 'locked' while not in the pool
type BMTree struct {
	leaves []*BMTNode
}

// draws the BMT (badly)
func (self *BMTree) Draw(hash []byte, d int) string {
	var left, right []string
	var anc []*BMTNode
	for i, n := range self.leaves {
		left = append(left, fmt.Sprintf("%v", hashstr(n.left)))
		if i%2 == 0 {
			anc = append(anc, n.parent)
		}
		right = append(right, fmt.Sprintf("%v", hashstr(n.right)))
	}
	anc = self.leaves
	var hashes [][]string
	for l := 0; len(anc) > 0; l++ {
		var nodes []*BMTNode
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

// initialises the BMTree by building up the nodes of a BMT
// segment size is stipulated to be the size of the hash
// segmentCount needs to be positive integer and does not need to be
// a power of two and can even be an odd number
// segmentSize * segmentCount determines the maximum chunk size
// hashed using the tree
func NewBMTree(hasher Hasher, segmentSize, segmentCount int) *BMTree {
	n := NewBMTNode(0, 0, nil)
	n.root = true
	prevlevel := []*BMTNode{n}
	// iterate over levels and creates 2^level nodes
	level := 1
	count := 2
	for d := 1; d <= depth(segmentCount); d++ {
		nodes := make([]*BMTNode, count)
		for i, _ := range nodes {
			var parent *BMTNode
			parent = prevlevel[i/2]
			t := NewBMTNode(level, i, parent)
			// fmt.Printf("created node level %v, index: %v/%v\n", level, i, count)
			nodes[i] = t
		}
		prevlevel = nodes
		level++
		count *= 2
	}
	// the datanode level is the nodes on the last level where
	return &BMTree{
		leaves: prevlevel,
	}
}

// methods needed by hash.Hash

// returns the size
func (self *BMTHasher) Size() int {
	return self.size
}

// returns the block size
func (self *BMTHasher) BlockSize() int {
	return self.blocksize
}

// hash.Hash interface Sum method appends the byte slice to the underlying
// data before it calculates and returns the hash of the chunk
func (self *BMTHasher) Sum(b []byte) (r []byte) {
	t := self.bmt
	i := self.cur
	//fmt.Printf("finalise for node index %v (leaves: %v)\n", i, len(t.leaves))
	n := t.leaves[i]
	j := i
	// must run strictly before all nodes calculate
	// datanodes are guaranteed to have a parent
	if len(self.segment) > self.size && i > 0 && n.parent != nil {
		n = n.parent
	} else {
		i *= 2
	}
	d := self.finalise(n, i)
	//fmt.Printf("finalise for node level %v index %v depth %v, %v, %v\n", n.level, i, d, j, len(self.segment))
	self.writeSegment(j, self.segment, d)
	c := <-self.result
	self.releaseTree()

	// sha3(length + BMT(pure_chunk))
	if self.blockLength == nil {
		return c
	}
	res := self.pool.hasher()
	res.Reset()
	res.Write(self.blockLength)
	res.Write(c)
	return res.Sum(nil)
}

// BMTHasher implements the io.Writer interface
// Write fills the buffer to hash
// with every full segment complete launches a hasher go routine
// that shoots up the BMT
func (self *BMTHasher) Write(b []byte) (int, error) {
	l := len(b)
	if l <= 0 {
		return 0, nil
	}
	s := self.segment
	i := self.cur
	count := (self.count + 1) / 2
	need := self.count*self.size - self.cur*2*self.size
	size := self.size
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
	//fmt.Printf("l: %v, s: %x, need: %v, size: %v, index: %v\n", l, s, need, size, self.cur)
	// read full segments and the last possibly partial segment
	for need > 0 && i < count-1 {
		// push all finished chunks we read
		self.writeSegment(i, s, self.depth)
		need -= size
		if need < 0 {
			size += need
		}
		s = b[rest : rest+size]
		rest += size
		i++
	}
	//fmt.Printf("open segment %v len %v\n (data: %v)\n", i, len(s), hashstr(s))
	self.segment = s
	self.cur = i
	// otherwise, we can assume len(s) == 0, so all buffer is read and chunk is not yet full
	return l, nil
}

// reads from io.Reader and appends to the data to hash using Write
// it reads so that chunk to hash is maximum length or reader reaches if EOF
func (self *BMTHasher) ReadFrom(r io.Reader) (m int64, err error) {
	self.getTree()
	bufsize := self.size*self.count - self.size*self.cur - len(self.segment)
	buf := make([]byte, bufsize)
	var read int
	for {
		var n int
		n, err = r.Read(buf)
		read += n
		if err == io.EOF || read == len(buf) {
			hash := self.Sum(buf[:n])
			if read == len(buf) {
				err = NewEOC(hash)
			}
			break
		}
		if err != nil {
			break
		}
		n, err = self.Write(buf[:n])
		if err != nil {
			break
		}
	}
	return int64(read), err
}

func (self *BMTHasher) Reset() {
	self.getTree()
	self.blockLength = nil
}

// It implements the swarmHash interface
func (self *BMTHasher) ResetWithLength(l []byte) {
	self.Reset()
	self.blockLength = l

}

// Release gives back the BMTree to the pool whereby it unlocks
// it resets tree, segment and index
func (self *BMTHasher) releaseTree() {
	if self.bmt != nil {
		n := self.bmt.leaves[self.cur]
		for ; n != nil; n = n.parent {
			n.unbalanced = false
			if n.parent != nil {
				n.root = false
			}
		}
		self.pool.Release(self.bmt)
		self.bmt = nil

	}
	self.cur = 0
	self.segment = nil
}

type SegmentWriter interface {
	Init(int) // initialises the segment writer with max int chunks
	WriteSegment(int, []byte) error
	Hash() []byte
}

func (self *BMTHasher) Hash() []byte {
	return <-self.result
}

func (self *BMTHasher) Init(i int) {
	self.getTree()
	self.max = int32(i)
}

// implements the SegmentWriter interface ie allows for segments
// to be written concurrently
func (self *BMTHasher) WriteSegment(i int, s []byte) (err error) {
	max := atomic.LoadInt32(&self.max)
	if int(max) <= i {
		return NewEOC(nil)
	}
	rightmost := i == int(max-1)
	last := atomic.AddInt32(&self.max, 1) == max
	if rightmost {
		self.segment = s
	} else {
		self.writeSegment(i, s, self.depth)
		if !last {
			return
		}
	}
	n := self.bmt.leaves[int(self.max-1)/2]
	d := self.finalise(n, i)
	self.writeSegment(i, self.segment, d)
	return
}

func (self *BMTHasher) writeSegment(i int, s []byte, d int) {
	h := self.pool.hasher()
	n := self.bmt.leaves[i]

	if len(s) > self.size && n.parent != nil {
		go func() {
			h.Reset()
			h.Write(s)
			s = h.Sum(nil)

			if n.root {
				self.result <- s
				return
			}
			self.run(n.parent, h, d, n.index, s)
		}()
		return
	}
	go self.run(n, h, d, i*2, s)
}

func (self *BMTHasher) run(n *BMTNode, h hash.Hash, d int, i int, s []byte) {
	isLeft := i%2 == 0
	for {
		if isLeft {
			n.left = s
			// fmt.Printf("->%v/%v left %v\n", n.level, n.index, hashstr(s))
		} else {
			n.right = s
			// fmt.Printf("->%v/%v right %v\n", n.level, n.index, hashstr(s))
		}
		if !n.unbalanced && n.toggle() {
			return
		}
		if !n.unbalanced || !isLeft || i == 0 && d == 0 {
			h.Reset()
			h.Write(n.left)
			h.Write(n.right)
			s = h.Sum(nil)

		} else {
			s = append(n.left, n.right...)
		}

		self.hash = s
		if n.root {
			// fmt.Printf("%v/%v depth: %v, root hash %v\n", n.level, n.index, d, hashstr(s))
			self.result <- s
			return
		}

		// fmt.Printf("%v/%v->%v/%v/%v %v\n", n.level, i, n.parent.level, i/2, i%2, hashstr(s))
		isLeft = n.isLeft
		n = n.parent
		i++
	}
}

// obtains a BMT resource by reserving one from the pool
func (self *BMTHasher) getTree() *BMTree {
	if self.bmt != nil {
		return self.bmt
	}
	t := self.pool.Reserve()
	self.bmt = t
	return t
}

// atomic bool toggle implementing a concurrent reusable bi-state object
// atomic addint with %2 implements atomic bool toggle
// it returns true if the toggler just put it in the active/waiting state
func (self *BMTNode) toggle() bool {
	return atomic.AddInt32(&self.state, 1)%2 == 1
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

// it is following the zigzags on the tree belonging
// to the final datasegment
func (self *BMTHasher) finalise(n *BMTNode, i int) (d int) {
	isLeft := i%2 == 0
	for {
		// when the final segment's path is going via left segments
		// the incoming data is pushed to the parent upon pulling the left
		// we do not need toogle the state since this condition is
		// detectable
		n.unbalanced = isLeft
		n.right = nil
		// fmt.Printf("%v/%v unbalanced %v\n", n.level, n.index, n.unbalanced)
		if n.initial {
			// fmt.Printf("%v/%v initial node found, depth: %v\n", n.level, n.index, d)
			n.root = true
			return d
		}
		isLeft = n.isLeft
		n = n.parent
		d++
	}
}

// EOC implements the error interface
func (self *EOC) Error() string {
	return fmt.Sprintf("hasher limit reached, chunk hash: %x", self.Hash)
}

// creates new end of chunk error with the hash
func NewEOC(hash []byte) *EOC {
	return &EOC{hash}
}
