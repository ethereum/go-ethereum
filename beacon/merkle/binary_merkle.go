// Copyright 2022 The go-ethereum Authors
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

package merkle

import (
	"math/bits"
	"reflect"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/log"
	"github.com/minio/sha256-simd"
)

// Value represents either a 32 byte value or hash node in a binary merkle tree/partial proof
type (
	Value  [32]byte
	Values []Value
)

var ValueT = reflect.TypeOf(Value{})

// UnmarshalJSON parses a merkle value in hex syntax.
func (m *Value) UnmarshalJSON(input []byte) error {
	return hexutil.UnmarshalFixedJSON(ValueT, input, m[:])
}

// VerifySingleProof verifies a Merkle proof branch for a single value in a
// binary Merkle tree (index is a generalized tree index).
func VerifySingleProof(proof Values, index uint64, value Value) (common.Hash, bool) {
	hasher := sha256.New()
	for _, proofHash := range proof {
		hasher.Reset()
		if index&1 == 0 {
			hasher.Write(value[:])
			hasher.Write(proofHash[:])
		} else {
			hasher.Write(proofHash[:])
			hasher.Write(value[:])
		}
		hasher.Sum(value[:0])
		index /= 2
		if index == 0 {
			return common.Hash{}, false
		}
	}
	if index != 1 {
		return common.Hash{}, false
	}
	return common.Hash(value), true
}

// ProofFormat defines the shape of a partial proof and allows traversing a subset of a tree
type ProofFormat interface {
	Children() (left, right ProofFormat) // either both or neither should be nil
}

// IsEqual returns true if the two formats are the same
func IsEqual(a, b ProofFormat) bool {
	al, ar := a.Children()
	bl, br := b.Children()
	if al == nil || bl == nil {
		return al == nil && bl == nil
	}
	return IsEqual(al, bl) && IsEqual(ar, br)
}

// ProofReader allows traversing and reading a tree structure or a subset of it.
// Note: the hash of each traversed node is always requested. If the internal
// hash is not available then subtrees are always traversed (first left, then right).
// If internal hash is available then subtrees are only traversed if needed by the writer.
type ProofReader interface {
	Children() (left, right ProofReader) // subtrees accessible if not nil
	ReadNode() (Value, bool)             // hash should be available if children are nil (leaf node), optional otherwise (internal node)
}

// ProofWriter allow collecting data for a partial proof while a subset of a tree is traversed.
type ProofWriter interface {
	Children() (left, right ProofWriter) // all non-nil subtrees are traversed
	WriteNode(Value)                     // called for every traversed tree node (both leaf and internal)
}

// TraverseProof traverses a reader and a writer defined on the same tree
// simultaneously, copies data from the reader to the writer (if writer is not nil)
// and returns the root hash. At least the shape defined by the writer is traversed;
// subtrees not required by the writer are only traversed (with writer == nil)
// if the hash of the internal tree node is not provided by the reader.
func TraverseProof(reader ProofReader, writer ProofWriter) (common.Hash, bool) {
	var (
		wl ProofWriter
		wr ProofWriter
	)
	if writer != nil {
		wl, wr = writer.Children()
	}
	node, nodeAvailable := reader.ReadNode()
	if nodeAvailable && wl == nil {
		if writer != nil {
			writer.WriteNode(node)
		}
		return common.Hash(node), true
	}
	rl, rr := reader.Children()
	if rl == nil {
		return common.Hash{}, false
	}
	lhash, ok := TraverseProof(rl, wl)
	if !ok {
		return common.Hash{}, false
	}
	rhash, ok := TraverseProof(rr, wr)
	if !ok {
		return common.Hash{}, false
	}
	if !nodeAvailable {
		hasher := sha256.New()
		hasher.Write(lhash[:])
		hasher.Write(rhash[:])
		hasher.Sum(node[:0])
	}
	if writer != nil {
		writer.WriteNode(node)
	}
	return common.Hash(node), true
}

// MultiProof stores a partial Merkle tree proof
type MultiProof struct {
	Format ProofFormat
	Values Values
}

// multiProofReader implements ProofReader based on a MultiProof and also allows
// attaching further subtree readers at certain indices
// Note: valuePtr is stored and copied as a reference because child readers read
// from the same value list as the tree is traversed
type multiProofReader struct {
	format   ProofFormat              // corresponding proof format
	values   Values                   // proof values
	valuePtr *int                     // next index to be read from values
	index    uint64                   // generalized tree index
	subtrees func(uint64) ProofReader // attached subtrees
}

// children implements ProofReader
func (mpr multiProofReader) Children() (left, right ProofReader) {
	lf, rf := mpr.format.Children()
	if lf == nil {
		if mpr.subtrees != nil {
			if subtree := mpr.subtrees(mpr.index); subtree != nil {
				return subtree.Children()
			}
		}
		return nil, nil
	}
	return multiProofReader{format: lf, values: mpr.values, valuePtr: mpr.valuePtr, index: mpr.index * 2, subtrees: mpr.subtrees},
		multiProofReader{format: rf, values: mpr.values, valuePtr: mpr.valuePtr, index: mpr.index*2 + 1, subtrees: mpr.subtrees}
}

// readNode implements ProofReader
func (mpr multiProofReader) ReadNode() (Value, bool) {
	if l, _ := mpr.format.Children(); l == nil && len(mpr.values) > *mpr.valuePtr {
		hash := mpr.values[*mpr.valuePtr]
		(*mpr.valuePtr)++
		return hash, true
	}
	return Value{}, false
}

// Reader creates a multiProofReader for the given proof; if subtrees != nil
// then also attaches subtree readers at indices where the function returns a
// non-nil reader.
// Note that the reader can only be traversed once as the values slice is
// sequentially consumed.
func (mp MultiProof) Reader(subtrees func(uint64) ProofReader) multiProofReader {
	return multiProofReader{format: mp.Format, values: mp.Values, valuePtr: new(int), index: 1, subtrees: subtrees}
}

// Finished returns true if all values have been consumed by the traversal.
// Should be checked after TraverseProof if received from an untrusted source in
// order to prevent DoS attacks by excess proof values.
func (mpr multiProofReader) Finished() bool {
	return len(mpr.values) == *mpr.valuePtr
}

// rootHash returns the root hash of the proven structure.
func (mp MultiProof) RootHash() common.Hash {
	reader := mp.Reader(nil)
	hash, ok := TraverseProof(reader, nil)
	if !ok || !reader.Finished() {
		log.Error("MultiProof.rootHash: invalid proof format")
	}
	return hash
}

// multiProofWriter implements ProofWriter and creates a MultiProof with the
// previously specified format. Also allows attaching further subtree writers at
// certain indices.
// Note: values is stored and copied as a reference because child writers append
// to the same value list as the tree is traversed
type multiProofWriter struct {
	format   ProofFormat              // target proof format
	values   *Values                  // target proof value list
	index    uint64                   // generalized tree index
	subtrees func(uint64) ProofWriter // attached subtrees
}

// NewMultiProofWriter creates a new multiproof writer with the specified format.
// If subtrees != nil then further subtree writers are attached at indices where
// the function returns a non-nil writer.
// Note that the specified format should not include these attached subtrees;
// they should be attached at leaf indices of the given format.
// Also note that target can be nil in which case the nodes specified by the format
// are traversed but not stored; subtree writers might still store tree data.
func NewMultiProofWriter(format ProofFormat, target *Values, subtrees func(uint64) ProofWriter) multiProofWriter {
	return multiProofWriter{format: format, values: target, index: 1, subtrees: subtrees}
}

// children implements ProofWriter
func (mpw multiProofWriter) Children() (left, right ProofWriter) {
	if mpw.subtrees != nil {
		if subtree := mpw.subtrees(mpw.index); subtree != nil {
			return subtree.Children()
		}
	}
	lf, rf := mpw.format.Children()
	if lf == nil {
		return nil, nil
	}
	return multiProofWriter{format: lf, values: mpw.values, index: mpw.index * 2, subtrees: mpw.subtrees},
		multiProofWriter{format: rf, values: mpw.values, index: mpw.index*2 + 1, subtrees: mpw.subtrees}
}

// writeNode implements ProofWriter
func (mpw multiProofWriter) WriteNode(node Value) {
	if mpw.values != nil {
		if lf, _ := mpw.format.Children(); lf == nil {
			*mpw.values = append(*mpw.values, node)
		}
	}
	if mpw.subtrees != nil {
		if subtree := mpw.subtrees(mpw.index); subtree != nil {
			subtree.WriteNode(node)
		}
	}
}

// ProofFormatIndexMap creates a generalized tree index -> MultiProof value
// slice index association map based on the given proof format.
func ProofFormatIndexMap(f ProofFormat) map[uint64]int {
	var (
		m   = make(map[uint64]int)
		pos int
	)
	addToIndexMap(m, f, &pos, 1)
	return m
}

// addToIndexMap recursively creates index associations for a given proof format subtree.
func addToIndexMap(m map[uint64]int, f ProofFormat, pos *int, index uint64) {
	l, r := f.Children()
	if l == nil {
		m[index] = *pos
		(*pos)++
	} else {
		addToIndexMap(m, l, pos, index*2)
		addToIndexMap(m, r, pos, index*2+1)
	}
}

// ChildIndex returns the generalized tree index of a subtree node in terms of
// the main tree where a is the main tree index of the subtree root and b is the
// subtree index of the node in question.
func ChildIndex(a, b uint64) uint64 {
	return (a-1)<<(63-bits.LeadingZeros64(b)) + b
}

// IndexMapFormat implements ProofFormat based on an index map filled with
// AddLeaf calls. Subtree formats can also be attached at certain indices.
type IndexMapFormat struct {
	leaves map[uint64]ProofFormat
	index  uint64
}

// NewIndexMapFormat returns an empty format.
func NewIndexMapFormat() IndexMapFormat {
	return IndexMapFormat{leaves: make(map[uint64]ProofFormat), index: 1}
}

// AddLeaf adds either a single leaf or attaches a subtree at the given tree index.
func (f IndexMapFormat) AddLeaf(index uint64, subtree ProofFormat) IndexMapFormat {
	if subtree != nil {
		f.leaves[index] = subtree
	}
	for index > 1 {
		index /= 2
		f.leaves[index] = nil
	}
	return f
}

// children implements ProofFormat
func (f IndexMapFormat) Children() (left, right ProofFormat) {
	if st, ok := f.leaves[f.index]; ok {
		if st != nil {
			return st.Children()
		}
		return IndexMapFormat{leaves: f.leaves, index: f.index * 2}, IndexMapFormat{leaves: f.leaves, index: f.index*2 + 1}
	}
	return nil, nil
}

// rangeFormat defined a proof format with a continuous range of leaf indices.
// Attaching subtree formats is also possible.
type rangeFormat struct {
	begin, end, index uint64 // begin and end should be on the same level
	subtree           func(uint64) ProofFormat
}

// NewRangeFormat creates a new rangeFormat with leafs in the begin..end range.
// If subtrees != nil then further subtree formats are attached at indices where
// the function returns a non-nil format.
func NewRangeFormat(begin, end uint64, subtree func(uint64) ProofFormat) rangeFormat {
	return rangeFormat{
		begin:   begin,
		end:     end,
		index:   1,
		subtree: subtree,
	}
}

// children implements ProofFormat
func (rf rangeFormat) Children() (left, right ProofFormat) {
	var (
		lzr = bits.LeadingZeros64(rf.begin)
		lzi = bits.LeadingZeros64(rf.index)
	)
	if lzi < lzr {
		return nil, nil
	}
	if lzi == lzr {
		if rf.subtree != nil && rf.index >= rf.begin && rf.index <= rf.end {
			if st := rf.subtree(rf.index); st != nil {
				return st.Children()
			}
		}
		return nil, nil
	}
	var (
		// i1..i2 are the descendants of rf.index at the tree level where begin and end are located
		i1 = rf.index << (lzi - lzr)
		i2 = ((rf.index + 1) << (lzi - lzr)) - 1
	)
	if i1 <= rf.end && i2 >= rf.begin {
		// Return child formats if there is an overlap (rf.index has any descendants
		// in the begin..end range).
		// Note that if begin..end only touches one of the returned child subtrees,
		// we still return a rangeFormat for both branches and the other one will
		// not have any further children (that child of rf.index will be stored
		// in the proof as a single sibling node).
		return rangeFormat{begin: rf.begin, end: rf.end, index: rf.index * 2, subtree: rf.subtree},
			rangeFormat{begin: rf.begin, end: rf.end, index: rf.index*2 + 1, subtree: rf.subtree}
	}
	return nil, nil
}

// MergedFormat implements ProofFormat and realizes the union of the included
// individual formats.
type MergedFormat []ProofFormat

// children implements ProofFormat
func (m MergedFormat) Children() (left, right ProofFormat) {
	var (
		l = make(MergedFormat, 0, len(m))
		r = make(MergedFormat, 0, len(m))
	)
	for _, f := range m {
		if left, right := f.Children(); left != nil {
			l = append(l, left)
			r = append(r, right)
		}
	}
	if len(l) > 0 {
		return l, r
	}
	return nil, nil
}

// MergedReader implements ProofReader and realizes the union of the included
// individual readers.
// Note that the readers belonging to the same structure (having the same root)
// is not checked by MergedReader.
// Also note that fully consuming underlying sequential readers is not guaranteed
// (MultiProofReader.Finalized will not necessarily return true so if necessary
// then the well-formedness of individual multiproofs should be checked separately).
type MergedReader []ProofReader

// children implements ProofReader
func (m MergedReader) Children() (left, right ProofReader) {
	var (
		l = make(MergedReader, 0, len(m))
		r = make(MergedReader, 0, len(m))
	)
	for _, reader := range m {
		if left, right := reader.Children(); left != nil {
			l = append(l, left)
			r = append(r, right)
		}
	}
	if len(l) > 0 {
		return l, r
	}
	return nil, nil
}

// readNode implements ProofReader
func (m MergedReader) ReadNode() (value Value, ok bool) {
	var hasChildren bool
	for _, reader := range m {
		if left, _ := reader.Children(); left != nil {
			// ensure that all readers are fully traversed
			hasChildren = true
		}
		if v, o := reader.ReadNode(); o {
			value, ok = v, o
		}
	}
	if hasChildren {
		return Value{}, false
	}
	return
}

// MergedWriter implements ProofWriter and realizes the union of the included
// individual writers. The shape traversed by MergedWriter is the union of the
// shapes traversed by individual writers.
type MergedWriter []ProofWriter

// children implements ProofWriter
func (m MergedWriter) Children() (left, right ProofWriter) {
	var (
		l = make(MergedWriter, 0, len(m))
		r = make(MergedWriter, 0, len(m))
	)
	for _, w := range m {
		if left, right := w.Children(); left != nil {
			l = append(l, left)
			r = append(r, right)
		}
	}
	if len(l) > 0 {
		return l, r
	}
	return nil, nil
}

// writeNode implements ProofWriter
func (m MergedWriter) WriteNode(value Value) {
	for _, w := range m {
		w.WriteNode(value)
	}
}

// callbackWriter implements ProofWriter with a simple callback mechanism
type callbackWriter struct {
	format        ProofFormat
	index         uint64
	storeCallback func(uint64, Value)
}

// NewCallbackWriter creates a callbackWriter that traverses the tree subset
// defined by the given proof format and calls callbackWriter for each traversed node
func NewCallbackWriter(format ProofFormat, storeCallback func(uint64, Value)) callbackWriter {
	return callbackWriter{format: format, index: 1, storeCallback: storeCallback}
}

// children implements ProofWriter
func (cw callbackWriter) Children() (left, right ProofWriter) {
	lf, rf := cw.format.Children()
	if lf == nil {
		return nil, nil
	}
	return callbackWriter{format: lf, index: cw.index * 2, storeCallback: cw.storeCallback},
		callbackWriter{format: rf, index: cw.index*2 + 1, storeCallback: cw.storeCallback}
}

// writeNode implements ProofWriter
func (cw callbackWriter) WriteNode(node Value) {
	cw.storeCallback(cw.index, node)
}
