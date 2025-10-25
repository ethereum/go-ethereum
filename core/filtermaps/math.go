// Copyright 2024 The go-ethereum Authors
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

package filtermaps

import (
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"hash/fnv"
	"iter"
	"math"
	"slices"
	"sort"

	"github.com/ethereum/go-ethereum/common"
)

// FilterRow encodes a single row of a filter map as a list of column indices.
// Note that the values are always stored in the same order as they were added
// and if the same column index is added twice, it is also stored twice.
// Order of column indices and potential duplications do not matter when searching
// for a value but leaving the original order makes reverting to a previous state
// simpler.
type FilterRow []uint32

// Params defines the basic parameters of the log index structure.
type Params struct {
	logMapHeight        uint // The number of bits required to represent the map height
	logMapWidth         uint // The number of bits required to represent the map width
	logMapsPerEpoch     uint // The number of bits required to represent the number of maps per epoch
	logValuesPerMap     uint // The number of bits required to represent the number of log values per map
	logMappingFrequency []uint
	maxRowLength        []uint32

	// These fields can be derived with the information above
	mapHeight    uint32 // The number of rows in the filter map
	mapsPerEpoch uint32 // The number of maps in an epoch
	valuesPerMap uint64 // The number of log values marked on each filter map

	rowGroupSize []uint32
}

// DefaultParams is the set of parameters used on mainnet.
var DefaultParams = Params{
	logMapHeight:        16,
	logMapWidth:         24,
	logMapsPerEpoch:     10,
	logValuesPerMap:     16,
	logMappingFrequency: []uint{10, 6, 2, 0},
	maxRowLength:        []uint32{8, 168, 2728, 10920},
	rowGroupSize:        []uint32{256, 16, 1, 1},
}

// RangeTestParams puts one log value per epoch, ensuring block exact tail unindexing for testing
var RangeTestParams = Params{
	logMapHeight:        4,
	logMapWidth:         24,
	logMapsPerEpoch:     0,
	logValuesPerMap:     0,
	logMappingFrequency: []uint{10, 6, 2, 0},
	maxRowLength:        []uint32{8, 168, 2728, 10920},
	rowGroupSize:        []uint32{16, 4, 1, 1},
}

// deriveFields calculates the derived fields of the parameter set.
func (p *Params) deriveFields() {
	p.mapHeight = uint32(1) << p.logMapHeight
	p.mapsPerEpoch = uint32(1) << p.logMapsPerEpoch
	p.valuesPerMap = uint64(1) << p.logValuesPerMap
}

// mapRowIndex calculates the unified storage index where the given row of the
// given map is stored. Note that this indexing scheme is the same as the one
// proposed in EIP-7745 for tree-hashing the filter map structure and for the
// same data proximity reasons it is also suitable for database representation.
// See also:
// https://eips.ethereum.org/EIPS/eip-7745#hash-tree-structure
func (p *Params) mapRowIndex(mapIndex, rowIndex uint32) uint64 {
	epochIndex, mapSubIndex := mapIndex>>p.logMapsPerEpoch, mapIndex&(p.mapsPerEpoch-1)
	return (uint64(epochIndex)<<p.logMapHeight+uint64(rowIndex))<<p.logMapsPerEpoch + uint64(mapSubIndex)
}

func (p *Params) getLogMappingFrequency(layerIndex uint32) uint {
	return p.logMappingFrequency[min(layerIndex, uint32(len(p.logMappingFrequency)-1))]
}

func (p *Params) getMaxRowLength(layerIndex uint32) uint32 {
	return p.maxRowLength[min(layerIndex, uint32(len(p.maxRowLength)-1))]
}

// addressValue returns the log value hash of a log emitting address.
func addressValue(address common.Address) common.Hash {
	var result common.Hash
	hasher := sha256.New()
	hasher.Write(address[:])
	hasher.Sum(result[:0])
	return result
}

// topicValue returns the log value hash of a log topic.
func topicValue(topic common.Hash) common.Hash {
	var result common.Hash
	hasher := sha256.New()
	hasher.Write(topic[:])
	hasher.Sum(result[:0])
	return result
}

// sanitize derives any missing fields and validates the parameter values.
func (p *Params) sanitize() error {
	p.deriveFields()
	if p.logMapWidth%8 != 0 {
		return fmt.Errorf("invalid configuration: logMapWidth (%d) must be a multiple of 8", p.logMapWidth)
	}
	if p.logMapWidth > 32 { // column index stored as uint32
		return fmt.Errorf("invalid configuration: logMapWidth (%d) should not exceed 32", p.logMapWidth)
	}
	if p.logMapHeight > 16 { // row index stored as uint16 in finishedMap
		return fmt.Errorf("invalid configuration: logMapHeight (%d) should not exceed 32", p.logMapHeight)
	}
	for _, maxRowLength := range p.maxRowLength {
		if maxRowLength >= 0x10000 { // index wrap-around issue in finishedMap
			return fmt.Errorf("invalid configuration: maxRowLength entry (%d) should not exceed 2**16", maxRowLength)
		}
	}
	for _, groupSize := range p.rowGroupSize {
		if groupSize == 0 || (groupSize&(groupSize-1)) != 0 {
			return fmt.Errorf("invalid configuration: rowGroupSize entry (%d) must be a power of 2", groupSize)
		}
	}
	if len(p.maxRowLength) != len(p.rowGroupSize) {
		return fmt.Errorf("invalid configuration: length of maxRowLength (%d) and rowGroupSize entry (%d) should be equal", len(p.maxRowLength), len(p.rowGroupSize))
	}
	return nil
}

// mapGroupIndex returns the start index of the base row group that contains the
// given map index. Assumes baseRowGroupSize is a power of 2.
func (p *Params) mapGroupIndex(index, dbLayer uint32) uint32 {
	return index & ^(p.rowGroupSize[dbLayer] - 1)
}

// mapGroupOffset returns the offset of the given map index within its base row group.
func (p *Params) mapGroupOffset(index, dbLayer uint32) uint32 {
	return index & (p.rowGroupSize[dbLayer] - 1)
}

// mapEpoch returns the epoch number that the given map index belongs to.
func (p *Params) mapEpoch(index uint32) uint32 {
	return index >> p.logMapsPerEpoch
}

// firstEpochMap returns the index of the first map in the specified epoch.
func (p *Params) firstEpochMap(epoch uint32) uint32 {
	return epoch << p.logMapsPerEpoch
}

// lastEpochMap returns the index of the last map in the specified epoch.
func (p *Params) lastEpochMap(epoch uint32) uint32 {
	return (epoch+1)<<p.logMapsPerEpoch - 1
}

// rowIndex returns the row index in which the given log value should be marked
// on the given map and mapping layer. Note that row assignments are re-shuffled
// with a different frequency on each mapping layer, allowing efficient disk
// access and Merkle proofs for long sections of short rows on lower order
// layers while avoiding putting too many heavy rows next to each other on
// higher order layers.
func (p *Params) rowIndex(mapIndex, layerIndex uint32, logValue common.Hash) uint32 {
	hasher := sha256.New()
	hasher.Write(logValue[:])
	var indexEnc [8]byte
	binary.LittleEndian.PutUint32(indexEnc[0:4], p.maskedMapIndex(mapIndex, layerIndex))
	binary.LittleEndian.PutUint32(indexEnc[4:8], layerIndex)
	hasher.Write(indexEnc[:])
	var hash common.Hash
	hasher.Sum(hash[:0])
	return binary.LittleEndian.Uint32(hash[:4]) % p.mapHeight
}

// columnIndex returns the column index where the given log value at the given
// position should be marked.
func (p *Params) columnIndex(lvIndex uint64, logValue *common.Hash) uint32 {
	var indexEnc [8]byte
	binary.LittleEndian.PutUint64(indexEnc[:], lvIndex)
	// Note: reusing the hasher brings practically no performance gain and would
	// require passing it through the entire matcher logic because of multi-thread
	// matching
	hasher := fnv.New64a()
	hasher.Write(indexEnc[:])
	hasher.Write(logValue[:])
	hash := hasher.Sum64()
	hashBits := p.logMapWidth - p.logValuesPerMap
	return uint32(lvIndex%p.valuesPerMap)<<hashBits + (uint32(hash>>(64-hashBits)) ^ uint32(hash)>>(32-hashBits))
}

// maskedMapIndex returns the index used for row mapping calculation on the
// given layer. On layer zero the mapping changes once per epoch, then the
// frequency of re-mapping increases with every new layer until it reaches
// the frequency where it is different for every mapIndex.
func (p *Params) maskedMapIndex(mapIndex, layerIndex uint32) uint32 {
	return mapIndex & (uint32(math.MaxUint32) << p.getLogMappingFrequency(layerIndex))
}

// potentialMatches returns the list of log value indices potentially matching
// the given log value hash in the range of the filter map the row belongs to.
// Note that the list of indices is always sorted and potential duplicates are
// removed. Though the column indices are stored in the same order they were
// added and therefore the true matches are automatically reverse transformed
// in the right order, false positives can ruin this property. Since these can
// only be separated from true matches after the combined pattern matching of the
// outputs of individual log value matchers and this pattern matcher assumes a
// sorted and duplicate-free list of indices, we should ensure these properties
// here.
func (p *Params) potentialMatches(rows []FilterRow, mapIndex uint32, logValue common.Hash) potentialMatches {
	results := make(potentialMatches, 0, 8)
	mapFirst := uint64(mapIndex) << p.logValuesPerMap
	for i, row := range rows {
		rowLen, maxLen := len(row), int(p.getMaxRowLength(uint32(i)))
		if rowLen > maxLen {
			rowLen = maxLen // any additional entries are generated by another log value on a higher mapping layer
		}
		for i := 0; i < rowLen; i++ {
			if potentialMatch := mapFirst + uint64(row[i]>>(p.logMapWidth-p.logValuesPerMap)); row[i] == p.columnIndex(potentialMatch, &logValue) {
				results = append(results, potentialMatch)
			}
		}
		if rowLen < maxLen {
			break
		}
		if i == len(rows)-1 {
			panic("potentialMatches: insufficient list of row alternatives")
		}
	}
	slices.Sort(results)
	// remove duplicates
	j := 0
	for i, match := range results {
		if i == 0 || match != results[i-1] {
			results[j] = results[i]
			j++
		}
	}
	return results[:j]
}

// potentialMatches is a strictly monotonically increasing list of log value
// indices in the range of a filter map that are potential matches for certain
// filter criteria.
// Note that nil is used as a wildcard and therefore means that all log value
// indices in the filter map range are potential matches. If there are no
// potential matches in the given map's range then an empty slice should be used.
type potentialMatches []uint64

func (p potentialMatches) Len() int           { return len(p) }
func (p potentialMatches) Less(i, j int) bool { return p[i] < p[j] }
func (p potentialMatches) Swap(i, j int)      { p[i], p[j] = p[j], p[i] }

type rangeSet[T uint32 | uint64] []common.Range[T]

func (a rangeSet[T]) includes(v T) bool {
	for _, r := range a {
		if r.Includes(v) {
			return true
		}
	}
	return false
}

func (a rangeSet[T]) closestLte(v T) (last T, found bool) {
	for _, r := range a {
		if r.First() > v {
			return
		}
		if r.AfterLast() > v {
			return v, true
		}
		last, found = r.Last(), true
	}
	return
}

func (a rangeSet[T]) closestGte(v T) (last T, found bool) {
	for _, r := range a {
		if r.First() > v {
			return r.First(), true
		}
		if r.AfterLast() > v {
			return v, true
		}
	}
	return
}

type rangeBoundary[T uint32 | uint64] struct {
	v T
	d int
}

type rangeBoundaries[T uint32 | uint64] []rangeBoundary[T]

func (rb *rangeBoundaries[T]) add(r common.Range[T], d int) {
	*rb = append((*rb), rangeBoundary[T]{v: r.First(), d: d}, rangeBoundary[T]{v: r.AfterLast(), d: -d})
}

func (rb rangeBoundaries[T]) makeSet(threshold int) rangeSet[T] {
	res := make(rangeSet[T], 0, len(rb)/2)
	sort.Slice(rb, func(i, j int) bool {
		return rb[i].v < rb[j].v
	})
	var (
		sum     int
		lastCmp bool
		start   T
	)
	for i, r := range rb {
		sum += r.d
		cmp := sum >= threshold
		if cmp != lastCmp && (i == len(rb)-1 || rb[i+1].v != r.v) {
			if cmp {
				start = r.v
			} else {
				res = append(res, common.NewRange[T](start, r.v-start))
			}
			lastCmp = cmp
		}
	}
	return res
}

func (a rangeSet[T]) intersection(b rangeSet[T]) rangeSet[T] {
	if len(a) == 0 || len(b) == 0 {
		return nil
	}
	rb := make(rangeBoundaries[T], 0, (len(a)+len(b))*2)
	for _, r := range a {
		rb.add(r, 1)
	}
	for _, r := range b {
		rb.add(r, 1)
	}
	return rb.makeSet(2)
}

func (a rangeSet[T]) exclude(b rangeSet[T]) rangeSet[T] {
	if len(a) == 0 {
		return nil
	}
	if len(b) == 0 {
		return a
	}
	rb := make(rangeBoundaries[T], 0, (len(a)+len(b))*2)
	for _, r := range a {
		rb.add(r, 1)
	}
	for _, r := range b {
		rb.add(r, -1)
	}
	return rb.makeSet(1)
}

func (a rangeSet[T]) union(b rangeSet[T]) rangeSet[T] {
	if len(a) == 0 {
		return b
	}
	if len(b) == 0 {
		return a
	}
	rb := make(rangeBoundaries[T], 0, (len(a)+len(b))*2)
	for _, r := range a {
		rb.add(r, 1)
	}
	for _, r := range b {
		rb.add(r, 1)
	}
	return rb.makeSet(1)
}

// iter iterates all integers in the range set.
func (r rangeSet[T]) iter() iter.Seq[T] {
	return func(yield func(T) bool) {
		for _, rr := range r {
			for i := range rr.Iter() {
				if !yield(i) {
					break
				}
			}
		}
	}
}

func (a rangeSet[T]) count() T {
	var count T
	for _, r := range a {
		count += r.Count()
	}
	return count
}

func (a rangeSet[T]) singleRange() common.Range[T] {
	if len(a) > 1 {
		panic("singleRange called for non-continuous rangeSet")
	}
	if len(a) == 1 {
		return a[0]
	}
	return common.NewRange[T](0, 0)
}

func singleRangeSet[T uint32 | uint64](r common.Range[T]) rangeSet[T] {
	if r.IsEmpty() {
		return nil
	}
	return rangeSet[T]{r}
}

func (a rangeSet[T]) equal(b rangeSet[T]) bool {
	if len(a) != len(b) {
		return false
	}
	for i, r := range a {
		if b[i] != r {
			return false
		}
	}
	return true
}
