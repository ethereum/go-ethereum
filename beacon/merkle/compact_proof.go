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
	"errors"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
)

// CompactProofFormat is a binary compact proof format, see description here:
// https://github.com/ChainSafe/consensus-specs/blob/feat/multiproof/ssz/merkle-proofs.md#compact-multiproofs
type CompactProofFormat struct {
	Format                 []byte
	firstBit, afterLastBit int
}

// Children implements ProofFormat
func (c CompactProofFormat) Children() (left, right ProofFormat) {
	if bit, ok := c.readFirstBit(); !ok {
		log.Error("Invalid compact proof format")
	} else if bit {
		return
	}
	l, r := c, c
	if !r.skipSubtree() {
		log.Error("Invalid compact proof format")
	}
	return l, r
}

// ValueCount returns the number of merkle values required for this proof
func (c CompactProofFormat) ValueCount() int {
	return (c.afterLastBit + 1 - c.firstBit) / 2
}

// EncodeCompactProofFormat encodes a ProofFormat into a binary compact proof format.
func EncodeCompactProofFormat(format ProofFormat) (c CompactProofFormat) {
	c.encodeFormatSubtree(format)
	return
}

// readFirstBit reads the first bit of the bit vector and moves the first bit
// pointer one bit ahead.
func (c *CompactProofFormat) readFirstBit() (bit, ok bool) {
	if c.firstBit >= c.afterLastBit {
		return false, false
	}
	bit = c.Format[c.firstBit>>3]&(byte(128)>>(c.firstBit&7)) != 0
	c.firstBit++
	return bit, true
}

// skipSubtree moves the first bit pointer beyond the subtree it was pointing at
// before and returns true if successful.
func (c *CompactProofFormat) skipSubtree() bool {
	if bit, ok := c.readFirstBit(); !ok {
		return false
	} else if bit {
		return true
	}
	return c.skipSubtree() && c.skipSubtree()
}

// appendBit adds a bit at the end of the bit vector.
func (c *CompactProofFormat) appendBit(bit bool) {
	bytePtr := c.afterLastBit >> 3
	if bytePtr == len(c.Format) {
		c.Format = append(c.Format, byte(0))
	}
	if bit {
		c.Format[bytePtr] += byte(128) >> (c.afterLastBit & 7)
	}
	c.afterLastBit++
}

// encodeFormatSubtree encodes a ProofFormat subtree at the end of the bit vector.
func (c *CompactProofFormat) encodeFormatSubtree(format ProofFormat) {
	if left, right := format.Children(); left == nil {
		c.appendBit(true)
	} else {
		c.appendBit(false)
		c.encodeFormatSubtree(left)
		c.encodeFormatSubtree(right)
	}
}

// encodeProofFormatSubtree recursively encodes a subtree of a proof format into
// binary compact format.
func encodeProofFormatSubtree(format ProofFormat, target *[]byte, bitLength *int) {
	bytePtr, bitMask := *bitLength>>3, byte(128)>>(*bitLength&7)
	*bitLength++
	if bytePtr == len(*target) {
		*target = append(*target, byte(0))
	}
	if left, right := format.Children(); left == nil {
		(*target)[bytePtr] += bitMask
	} else {
		encodeProofFormatSubtree(left, target, bitLength)
		encodeProofFormatSubtree(right, target, bitLength)
	}
}

// MultiProof stores a partial Merkle tree proof
type MultiProof struct {
	Format CompactProofFormat
	Values Values
}

// Encode encodes a MultiProof into a byte slice
func (m *MultiProof) Encode() []byte {
	lf := len(m.Format.Format)
	enc := make([]byte, lf+32*len(m.Values))
	copy(enc[:lf], m.Format.Format)
	for i, value := range m.Values {
		copy(enc[lf+i*32:lf+(i+1)*32], value[:])
	}
	return enc
}

// Decode decodes a MultiProof from a byte slice
func (m *MultiProof) Decode(enc []byte) error {
	valueCount := len(enc) * 4 / 129
	lf := (valueCount + 3) / 4
	if len(enc) != lf+32*valueCount {
		return errors.New("Invalid length for encoded MultiProof")
	}
	format := CompactProofFormat{
		Format:       make([]byte, lf),
		afterLastBit: valueCount*2 - 1,
	}
	copy(format.Format, enc[:lf])
	if f := format; !f.skipSubtree() || f.firstBit != f.afterLastBit {
		log.Error("Invalid compact proof format")
	}
	m.Format, m.Values = format, make(Values, valueCount)
	for i := range m.Values {
		copy(m.Values[i][:], enc[lf+i*32:lf+(i+1)*32])
	}
	return nil
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
