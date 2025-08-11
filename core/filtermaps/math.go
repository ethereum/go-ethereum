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
	"math"
	"sort"

	"github.com/ethereum/go-ethereum/common"
)

// Params defines the basic parameters of the log index structure.
type Params struct {
	logMapHeight    uint // The number of bits required to represent the map height
	logMapWidth     uint // The number of bits required to represent the map width
	logMapsPerEpoch uint // The number of bits required to represent the number of maps per epoch
	logValuesPerMap uint // The number of bits required to represent the number of log values per map

	// baseRowLengthRatio represents the ratio of base row length
	// to the average row length.
	baseRowLengthRatio uint

	// logLayerDiff defines the logarithmic growth factor (base 2) of
	// the maximum row length per layer. It indicates how much the maximum
	// row length increases as the layer depth increases.
	//
	// Specifically:
	// - the row length in base layer (layer == 0) is baseRowLength
	// - the row length in layer x is baseRowLength << (logLayerDiff * x)
	logLayerDiff uint

	// These fields can be derived with the information above
	mapHeight     uint32 // The number of rows in the filter map
	mapsPerEpoch  uint32 // The number of maps in an epoch
	valuesPerMap  uint64 // The number of log values marked on each filter map
	baseRowLength uint32 // maximum number of log values per row on layer 0

	// baseRowGroupSize defines the number of base row entries grouped together
	// as a single database entry in the local database to optimize storage
	// and retrieval efficiency.
	//
	// This value can be configured based on the specific implementation.
	baseRowGroupSize uint32
}

// DefaultParams is the set of parameters used on mainnet.
var DefaultParams = Params{
	logMapHeight:       16,
	logMapWidth:        24,
	logMapsPerEpoch:    10,
	logValuesPerMap:    16,
	baseRowGroupSize:   32,
	baseRowLengthRatio: 8,
	logLayerDiff:       4,
}

// RangeTestParams puts one log value per epoch, ensuring block exact tail unindexing for testing
var RangeTestParams = Params{
	logMapHeight:       4,
	logMapWidth:        24,
	logMapsPerEpoch:    0,
	logValuesPerMap:    0,
	baseRowGroupSize:   32,
	baseRowLengthRatio: 16, // baseRowLength >= 1
	logLayerDiff:       4,
}

// deriveFields calculates the derived fields of the parameter set.
func (p *Params) deriveFields() {
	p.mapHeight = uint32(1) << p.logMapHeight
	p.mapsPerEpoch = uint32(1) << p.logMapsPerEpoch
	p.valuesPerMap = uint64(1) << p.logValuesPerMap
	p.baseRowLength = uint32(p.valuesPerMap * uint64(p.baseRowLengthRatio) / uint64(p.mapHeight))
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
	if p.baseRowGroupSize == 0 || (p.baseRowGroupSize&(p.baseRowGroupSize-1)) != 0 {
		return fmt.Errorf("invalid configuration: baseRowGroupSize (%d) must be a power of 2", p.baseRowGroupSize)
	}
	return nil
}

// mapGroupIndex returns the start index of the base row group that contains the
// given map index. Assumes baseRowGroupSize is a power of 2.
func (p *Params) mapGroupIndex(index uint32) uint32 {
	return index & ^(p.baseRowGroupSize - 1)
}

// mapGroupOffset returns the offset of the given map index within its base row group.
func (p *Params) mapGroupOffset(index uint32) uint32 {
	return index & (p.baseRowGroupSize - 1)
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

// maxRowLength returns the maximum length filter rows are populated up to when
// using the given mapping layer.
//
// A log value can be marked on the map according to a given mapping layer if
// the row mapping on that layer points to a row that has not yet reached the
// maxRowLength belonging to that layer.
//
// This means that a row that is considered full on a given layer may still be
// extended further on a higher order layer.
//
// Each value is marked on the lowest order layer possible, assuming that marks
// are added in ascending log value index order. When searching for a log value
// one should consider all layers and process corresponding rows up until the
// first one where the row mapped to the given layer is not full.
func (p *Params) maxRowLength(layerIndex uint32) uint32 {
	logLayerDiff := uint(layerIndex) * p.logLayerDiff
	if logLayerDiff > p.logMapsPerEpoch {
		logLayerDiff = p.logMapsPerEpoch
	}
	return p.baseRowLength << logLayerDiff
}

// maskedMapIndex returns the index used for row mapping calculation on the
// given layer. On layer zero the mapping changes once per epoch, then the
// frequency of re-mapping increases with every new layer until it reaches
// the frequency where it is different for every mapIndex.
func (p *Params) maskedMapIndex(mapIndex, layerIndex uint32) uint32 {
	logLayerDiff := uint(layerIndex) * p.logLayerDiff
	if logLayerDiff > p.logMapsPerEpoch {
		logLayerDiff = p.logMapsPerEpoch
	}
	return mapIndex & (uint32(math.MaxUint32) << (p.logMapsPerEpoch - logLayerDiff))
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
		rowLen, maxLen := len(row), int(p.maxRowLength(uint32(i)))
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
	sort.Sort(results)
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
// potentialMatches implements sort.Interface.
// Note that nil is used as a wildcard and therefore means that all log value
// indices in the filter map range are potential matches. If there are no
// potential matches in the given map's range then an empty slice should be used.
type potentialMatches []uint64

func (p potentialMatches) Len() int           { return len(p) }
func (p potentialMatches) Less(i, j int) bool { return p[i] < p[j] }
func (p potentialMatches) Swap(i, j int)      { p[i], p[j] = p[j], p[i] }
