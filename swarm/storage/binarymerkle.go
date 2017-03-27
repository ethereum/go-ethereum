package storage

// provides a binary merkle tree implementation.

import (
	"bytes"
	_ "crypto/sha256"
	"encoding/binary"
	"errors"
	"fmt"
	"hash"
	"io"
	"sync"

	"github.com/ethereum/go-ethereum/crypto/sha3"
)

var hashFunc Hasher = sha3.NewKeccak256 //default hasher

type state struct {
	btree BTree
	root  Root
}

const (
	// max number of leafs on 4096 chunk len in 32 bytes segments
	mprocessors = 128
)

// A merkle tree for a user that stores the entire tree
// Specifically this tree is left a leaning balanced binary tree
// Where each node holds the hash of its leaves
// And the rootHash is the root node hashed with the count
// This tree is immutable
type BTree struct {
	count    uint64
	root     *node
	rootHash []byte
	//hashFunc Hasher
}

type node struct {
	label    []byte
	children [2]*node // if all nil, leaf node

	// Representation invariants:
	// if children[0] is nil, children[1] is nil
	// if both children non nil:
	//      label is hash of (children[0].label + children[1].label)
	// if leaf: label is arbitrary data
	// else if children[1] is nil, label=hash(children[0].label)
}

type jobparam struct {
	data   [][]byte
	height int
	n0     *node
	n1     *node
	id     int
}

type jobresult struct {
	n    *node
	data [][]byte
	id   int
}

func (t BTree) Count() uint64 {
	return t.count
}

// The hash/root of an empty BTree does not matter
func (t BTree) Root() []byte {
	return t.rootHash
}

// All trees should pass , unless they are invalid, which should only happen
// if incorrectly built or modified.
// Checks the rep invariants
func (t BTree) Validate() error {
	count, height, err := t.root.validate()
	if err != nil {
		return err
	}
	if count != t.count {
		return fmt.Errorf("Incorrect count. Was %d, should be %d", t.count, count)
	}
	if height != GetHeight(count) {
		return fmt.Errorf("Incorrect height. Was %d, should be %d", height, GetHeight(count))
	}

	rootLabel := make([]byte, 0)
	if height > 0 {
		rootLabel = t.root.label
	}
	h := rootHash(count, rootLabel)
	if !bytes.Equal(t.rootHash, h) {
		return fmt.Errorf("Incorrect rootHash")
	}
	return nil
}

// Checks the rep invariants
func (t *node) validate() (count uint64, height int, err error) {
	if t == nil {
		return 0, 0, nil
	}
	if t.children[0] == nil {
		if t.children[1] != nil {
			return 0, 0, fmt.Errorf("Invalid Node: Node missing first child, but has second")
		}
		// Leaf node
		return 1, 1, nil
	}

	// Not a leaf node
	count, height, err = t.children[0].validate()
	if err != nil {
		return
	}
	if t.children[1] != nil {
		count2, height2, err2 := t.children[1].validate()
		count += count2
		if err2 != nil {
			return count, height, err2
		}
		if height2 != height {
			return count, height, fmt.Errorf("Invalid Node: height mismatch between children")
		}
	}
	h := makeHash(t.children[0], t.children[1])
	if !bytes.Equal(h, t.label) {
		return 0, 0, fmt.Errorf("Invalid Node: Node hash mismatch")
	}

	height++
	return
}

func rootHash(count uint64, data []byte) []byte {
	h := hashFunc()
	h.Reset()
	h.Write(data)
	binary.Write(h, binary.LittleEndian, count)
	return h.Sum(nil)
}

func makeHash(left, right *node) []byte {
	h := hashFunc()
	h.Reset()
	if left != nil {
		h.Write(left.label)
		if right != nil {
			h.Write(right.label)
		}
	}
	return h.Sum(make([]byte, 0))
}

// Returns the height of the tree containing count leaf nodes.
// This the number of nodes (including the final leaf) from the root to
// any leaf.
func GetHeight(count uint64) int {
	if count == 0 {
		return 0
	}
	height := 0
	for count > (1 << uint(height)) {
		height++
	}
	return height + 1
}

// Build Binary Merkle Tree over data segments of segmentsize len with a specific hash func
// Return
// BMT - The BMT Representation of the data
// ROOT - BMT Root
// Count - Numers of leafs at the BMT
// error -
func BuildBMT(h Hasher, data []byte, segmentsize int) (bmt *BTree, roor *Root, count int, err error) {
	hashFunc = h
	leafcount := len(data) / segmentsize
	if len(data)%segmentsize != 0 {
		leafcount++
	}
	tree := BuildParralel(data, segmentsize)
	err = tree.Validate()
	if err != nil {
		return nil, nil, 0, errors.New("Validation error")
	}
	if tree.Count() != uint64(leafcount) {
		return nil, nil, 0, errors.New("Validation count error")
	}

	return tree, &Root{uint64(leafcount), tree.Root()}, leafcount, nil

}

func processor(pend *sync.WaitGroup, id int, tasks <-chan jobparam, results chan<- jobresult) {

	defer pend.Done()
	for task := range tasks {

		var res jobresult

		switch task.height {
		case 0:
			res = jobresult{nil, task.data, task.id}
		case 1:
			res = jobresult{&node{label: task.data[0]}, nil, task.id}
		default:
			hash := makeHash(task.n0, task.n1)
			res = jobresult{&node{label: hash, children: [2]*node{task.n0, task.n1}}, task.data, task.id}
		}
		results <- res
		pend.Done()
	}

}

// Build a tree
func Build(data [][]byte) *BTree {
	count := uint64(len(data))
	height := GetHeight(count)

	node, leftOverData := buildNode(data, height)
	if len(leftOverData) != 0 {
		panic("Build failed to consume all data")
	}
	rootLabel := make([]byte, 0)
	if height > 0 {
		rootLabel = node.label
	}
	hash := rootHash(count, rootLabel)
	t := BTree{count, node, hash}
	return &t
}

// Build a tree parralel computation
// We build it level by level ..starting from the depper one and get up..
func BuildParralel(data []byte, segmentsize int) *BTree {
	count := len(data) / segmentsize
	var height int

	if len(data)%segmentsize != 0 {
		height = GetHeight(uint64(count + 1))

	} else {
		height = GetHeight(uint64(count))
	}

	jobs := make(chan jobparam, mprocessors)
	results := make(chan jobresult, mprocessors)

	pend := new(sync.WaitGroup)

	for w := 1; w <= mprocessors; w++ {
		pend.Add(1)
		go processor(pend, 2, jobs, results)
	}
	/*build leafs*/
	var leaflevelcount = 1 << uint(height-1)
	for k := 0; k < leaflevelcount; k++ {
		datasegment := make([][]byte, 0, 1)
		pend.Add(1)
		if k*segmentsize+segmentsize > len(data) {
			if (k*segmentsize + len(data)%segmentsize) > len(data) {
				jobs <- jobparam{data: nil, height: 0, n0: nil, n1: nil, id: k}
			} else {
				datasegment = append(datasegment, data[k*segmentsize:k*segmentsize+len(data)%segmentsize])
				jobs <- jobparam{data: datasegment, height: 1, n0: nil, n1: nil, id: k}
			}
		} else {
			datasegment = append(datasegment, data[k*segmentsize:(k+1)*segmentsize])
			jobs <- jobparam{data: datasegment, height: 1, n0: nil, n1: nil, id: k}
		}
	}
	fmt.Println("1")
	close(jobs)
	pend.Wait()

	if len(data)%segmentsize != 0 {
		count++
	}
	restmp := make([]jobresult, leaflevelcount)
	fmt.Println("2", leaflevelcount)
	//Get computation results
	for i := 0; i < leaflevelcount; i++ {
		tres := <-results
		restmp[tres.id] = tres
	}
	fmt.Println("3", height)
	for i := 1; i < height; i++ {
		jobs := make(chan jobparam, mprocessors)
		for w := 1; w <= mprocessors; w++ {
			pend.Add(1)
			go processor(pend, 2, jobs, results)
		}
		var indexres = 0
		for j := 0; j < (1<<uint(height-1))>>uint(i); j++ {
			pend.Add(1)
			jobs <- jobparam{n0: restmp[indexres].n, n1: restmp[indexres+1].n, data: restmp[indexres+1].data, height: 2, id: j}
			indexres += 2
		}
		close(jobs)
		pend.Wait()
		if i < height-1 {
			for l := 0; l < (1<<uint(height-1))>>uint(i); l++ {
				tmpres := <-results
				restmp[tmpres.id] = tmpres
			}
		}
	}
	rootLabel := make([]byte, 0)
	var res jobresult
	if leaflevelcount == 1 {
		res = restmp[0]
	} else {
		res = <-results
	}
	if height > 0 {
		rootLabel = res.n.label
	}
	hash := rootHash(uint64(count), rootLabel)
	t := BTree{uint64(count), res.n, hash}
	return &t
}

// returns a node and the left over data not used by it
func buildNode(data [][]byte, height int) (*node, [][]byte) {
	if height == 0 || len(data) == 0 {
		return nil, data
	}
	if height == 1 {
		// leaf
		return &node{label: data[0]}, data[1:]
	}
	n0, data := buildNode(data, height-1)
	n1, data := buildNode(data, height-1)

	hash := makeHash(n0, n1)
	return &node{label: hash, children: [2]*node{n0, n1}}, data
}

func splitData(data []byte, size int) [][]byte {
	/* Splits data into an array of slices of len(size) */
	count := len(data) / size
	blocks := make([][]byte, 0, count)
	for i := 0; i < count; i++ {
		block := data[i*size : (i+1)*size]
		blocks = append(blocks, block)
	}
	if len(data)%size != 0 {
		blocks = append(blocks, data[len(blocks)*size:])
	}
	return blocks
}

// Return a [][]byte needed to prove the gkf of the item at the passed index
// The payload of the item at index is the first value in the proof
func (t *BTree) InclusionProof(index int) [][]byte {
	if uint64(index) >= t.count {
		panic("Invalid index: too large")
	}
	if index < 0 {
		panic("Invalid index: negative")
	}
	h := GetHeight(t.count)
	fmt.Println(h)
	return proveNode(h, t.root, index)
}

func proveNode(height int, n *node, index int) [][]byte {
	if height == 1 {
		if index != 0 {
			panic("Invalid index: non 0 for final node")
		}
		return [][]byte{n.label}
	}
	childIndex := index >> uint(height-2)
	nextIndex := index & (^(1 << uint(height-2)))
	b := proveNode(height-1, n.children[childIndex], nextIndex)
	otherChildIndex := (childIndex + 1) % 2
	if n.children[otherChildIndex] != nil {
		b = append(b, n.children[otherChildIndex].label)
	}
	return b
}

// The Root of a merkle tree for a client that does not store the tree
type Root struct {
	Count uint64
	Base  []byte
}

// Proves theof an element at the given index with the value thats the first entry in proof
func (r *Root) CheckProof(h Hasher, proof [][]byte, index int) (bool, error) {
	hashFunc = h
	theight := GetHeight(r.Count)
	var root, ok, err = checkNode(theight, proof, uint64(index), r.Count)
	base := rootHash(r.Count, root)
	return ok && bytes.Equal(r.Base, base), err
}

func checkNode(height int, proof [][]byte, index uint64, count uint64) (hash []byte, ok bool, err error) {
	if len(proof) == 0 {
		return nil, false, errors.New("checkNode : proof is empty")
	}
	if count <= index {
		fmt.Println("bad count", count, index)
		return nil, false, fmt.Errorf("bad count %d at index %d", count, index)
	}

	if height == 1 {
		if index != 0 || len(proof) != 1 {
			fmt.Println("BAD", index, proof)
			return nil, false, fmt.Errorf("BAD %d %d", index, proof)
		}
		return proof[0], true, nil
	}

	childIndex := index >> uint(height-2)
	mask := uint64(^(1 << uint(height-2)))
	nextIndex := index & mask

	var data []byte
	//var ok bool

	h := hashFunc()
	h.Reset()
	//	h:=hashFunc.New()
	var nextCount uint64
	last := len(proof) - 1
	if childIndex == 1 {
		nextCount = count & mask
		h.Write(proof[last])
		data, ok, err = checkNode(height-1, proof[:last], nextIndex, nextCount)
		h.Write(data)
	} else {
		nextCount = count
		if count > ^mask {
			nextCount = ^mask
		}
		if count == nextCount {
			data, ok, err = checkNode(height-1, proof, nextIndex, nextCount)
			h.Write(data)
		} else {
			data, ok, err = checkNode(height-1, proof[:last], nextIndex, nextCount)
			h.Write(data)
			h.Write(proof[last])
		}
	}

	hash = h.Sum(make([]byte, 0))
	return hash, ok, nil
}

// BMTHash defines the interface to hash functions that
type BMTHash interface {
	// Write absorbs more data into the hash's state. It panics if input is
	// written to it after output has been read from it.
	io.Writer

	// Read reads more output from the hash; reading affects the hash's
	// state.
	// It never returns an error.
	io.Reader

	// Clone returns a copy of the BMTHash in its current state.
	Clone() BMTHash

	// Reset resets the BMTHash to its initial state.
	Reset()
}

// Reset clears the internal state
func (d *state) Reset() {
	d.root = Root{Count: 0, Base: nil}
	d.btree = BTree{count: 0, root: nil, rootHash: nil}
}

// Write absorbs more data into the hash's state.
func (d *state) Write(p []byte) (written int, err error) {
	tree, r, count, err1 := BuildBMT(hashFunc, p, 32)
	d.btree = *tree
	d.root = *r

	if err1 != nil {
		err = errors.New("bmt write error")
	}

	return count, err
}

// Sum return the root hash of the BMT
func (d *state) Sum(in []byte) []byte {
	return d.root.Base
}

// BlockSize returns the rate of sponge underlying this hash function.
func (d *state) BlockSize() int { return 0 }

// Size returns the output size of the hash function in bytes.
func (d *state) Size() int { return 32 }

// NewBMTSHA3 creates a new BMT hash
func NewBMTSHA3() hash.Hash {
	tmpbtree := BTree{count: 0, root: nil, rootHash: nil}
	troot := Root{Count: 0, Base: nil}
	return &state{btree: tmpbtree, root: troot}
}
