package filtermaps

import (
	"crypto/sha256"
	"encoding/binary"
	"sort"

	"github.com/ethereum/go-ethereum/common"
)

type Params struct {
	logMapHeight    uint // log2(mapHeight)
	logMapsPerEpoch uint // log2(mmapsPerEpochapsPerEpoch)
	logValuesPerMap uint // log2(logValuesPerMap)
	// derived fields
	mapHeight    uint32 // filter map height (number of rows)
	mapsPerEpoch uint32 // number of maps in an epoch
	valuesPerMap uint64 // number of log values marked on each filter map
}

var DefaultParams = Params{
	logMapHeight:    12,
	logMapsPerEpoch: 6,
	logValuesPerMap: 16,
}

func (p *Params) deriveFields() {
	p.mapHeight = uint32(1) << p.logMapHeight
	p.mapsPerEpoch = uint32(1) << p.logMapsPerEpoch
	p.valuesPerMap = uint64(1) << p.logValuesPerMap
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

// rowIndex returns the row index in which the given log value should be marked
// during the given epoch. Note that row assignments are re-shuffled in every
// epoch in order to ensure that even though there are always a few more heavily
// used rows due to very popular addresses and topics, these will not make search
// for other log values very expensive. Even if certain values are occasionally
// sorted into these heavy rows, in most of the epochs they are placed in average
// length rows.
func (p *Params) rowIndex(epochIndex uint32, logValue common.Hash) uint32 {
	hasher := sha256.New()
	hasher.Write(logValue[:])
	var indexEnc [4]byte
	binary.LittleEndian.PutUint32(indexEnc[:], epochIndex)
	hasher.Write(indexEnc[:])
	var hash common.Hash
	hasher.Sum(hash[:0])
	return binary.LittleEndian.Uint32(hash[:4]) % p.mapHeight
}

// columnIndex returns the column index that should be added to the appropriate
// row in order to place a mark for the next log value.
func (p *Params) columnIndex(lvIndex uint64, logValue common.Hash) uint32 {
	x := uint32(lvIndex % p.valuesPerMap) // log value sub-index
	transformHash := transformHash(uint32(lvIndex/p.valuesPerMap), logValue)
	// apply column index transformation function
	x += binary.LittleEndian.Uint32(transformHash[0:4])
	x *= binary.LittleEndian.Uint32(transformHash[4:8])*2 + 1
	x ^= binary.LittleEndian.Uint32(transformHash[8:12])
	x *= binary.LittleEndian.Uint32(transformHash[12:16])*2 + 1
	x += binary.LittleEndian.Uint32(transformHash[16:20])
	x *= binary.LittleEndian.Uint32(transformHash[20:24])*2 + 1
	x ^= binary.LittleEndian.Uint32(transformHash[24:28])
	x *= binary.LittleEndian.Uint32(transformHash[28:32])*2 + 1
	return x
}

// transformHash calculates a hash specific to a given map and log value hash
// that defines a bijective function on the uint32 range. This function is used
// to transform the log value sub-index (distance from the first index of the map)
// into a 32 bit column index, then applied in reverse when searching for potential
// matches for a given log value.
func transformHash(mapIndex uint32, logValue common.Hash) (result common.Hash) {
	hasher := sha256.New()
	hasher.Write(logValue[:])
	var indexEnc [4]byte
	binary.LittleEndian.PutUint32(indexEnc[:], mapIndex)
	hasher.Write(indexEnc[:])
	hasher.Sum(result[:0])
	return
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
func (p *Params) potentialMatches(row FilterRow, mapIndex uint32, logValue common.Hash) potentialMatches {
	results := make(potentialMatches, 0, 8)
	transformHash := transformHash(mapIndex, logValue)
	sub1 := binary.LittleEndian.Uint32(transformHash[0:4])
	mul1 := uint32ModInverse(binary.LittleEndian.Uint32(transformHash[4:8])*2 + 1)
	xor1 := binary.LittleEndian.Uint32(transformHash[8:12])
	mul2 := uint32ModInverse(binary.LittleEndian.Uint32(transformHash[12:16])*2 + 1)
	sub2 := binary.LittleEndian.Uint32(transformHash[16:20])
	mul3 := uint32ModInverse(binary.LittleEndian.Uint32(transformHash[20:24])*2 + 1)
	xor2 := binary.LittleEndian.Uint32(transformHash[24:28])
	mul4 := uint32ModInverse(binary.LittleEndian.Uint32(transformHash[28:32])*2 + 1)
	// perform reverse column index transformation on all column indices of the row.
	// if a column index was added by the searched log value then the reverse
	// transform will yield a valid log value sub-index of the given map.
	// Column index is 32 bits long while there are 2**16 valid log value indices
	// in the map's range, so this can also happen by accident with 1 in 2**16
	// chance, in which case we have a false positive.
	for _, columnIndex := range row {
		if potentialSubIndex := (((((((columnIndex * mul4) ^ xor2) * mul3) - sub2) * mul2) ^ xor1) * mul1) - sub1; potentialSubIndex < uint32(p.valuesPerMap) {
			results = append(results, uint64(mapIndex)<<p.logValuesPerMap+uint64(potentialSubIndex))
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
// Note that nil is used as a wildcard and therefore means that all log value
// indices in the filter map range are potential matches. If there are no
// potential matches in the given map's range then an empty slice should be used.
type potentialMatches []uint64

// noMatches means there are no potential matches in a given filter map's range.
var noMatches = potentialMatches{}

func (p potentialMatches) Len() int           { return len(p) }
func (p potentialMatches) Less(i, j int) bool { return p[i] < p[j] }
func (p potentialMatches) Swap(i, j int)      { p[i], p[j] = p[j], p[i] }

// uint32ModInverse takes an odd 32 bit number and returns its modular
// multiplicative inverse (mod 2**32), meaning that for any uint32 x and odd y
// x * y *  uint32ModInverse(y) == 1.
func uint32ModInverse(v uint32) uint32 {
	if v&1 == 0 {
		panic("uint32ModInverse called with even argument")
	}
	m := int64(1) << 32
	m0 := m
	a := int64(v)
	x, y := int64(1), int64(0)
	for a > 1 {
		q := a / m
		m, a = a%m, m
		x, y = y, x-q*y
	}
	if x < 0 {
		x += m0
	}
	return uint32(x)
}
