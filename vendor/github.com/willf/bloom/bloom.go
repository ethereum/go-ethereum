/*
Package bloom provides data structures and methods for creating Bloom filters.

A Bloom filter is a representation of a set of _n_ items, where the main
requirement is to make membership queries; _i.e._, whether an item is a
member of a set.

A Bloom filter has two parameters: _m_, a maximum size (typically a reasonably large
multiple of the cardinality of the set to represent) and _k_, the number of hashing
functions on elements of the set. (The actual hashing functions are important, too,
but this is not a parameter for this implementation). A Bloom filter is backed by
a BitSet; a key is represented in the filter by setting the bits at each value of the
hashing functions (modulo _m_). Set membership is done by _testing_ whether the
bits at each value of the hashing functions (again, modulo _m_) are set. If so,
the item is in the set. If the item is actually in the set, a Bloom filter will
never fail (the true positive rate is 1.0); but it is susceptible to false
positives. The art is to choose _k_ and _m_ correctly.

In this implementation, the hashing functions used is murmurhash,
a non-cryptographic hashing function.

This implementation accepts keys for setting as testing as []byte. Thus, to
add a string item, "Love":

    uint n = 1000
    filter := bloom.New(20*n, 5) // load of 20, 5 keys
    filter.Add([]byte("Love"))

Similarly, to test if "Love" is in bloom:

    if filter.Test([]byte("Love"))

For numeric data, I recommend that you look into the binary/encoding library. But,
for example, to add a uint32 to the filter:

    i := uint32(100)
    n1 := make([]byte,4)
    binary.BigEndian.PutUint32(n1,i)
    f.Add(n1)

Finally, there is a method to estimate the false positive rate of a particular
Bloom filter for a set of size _n_:

    if filter.EstimateFalsePositiveRate(1000) > 0.001

Given the particular hashing scheme, it's best to be empirical about this. Note
that estimating the FP rate will clear the Bloom filter.
*/
package bloom

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"math"

	"github.com/spaolacci/murmur3"
	"github.com/willf/bitset"
)

// A BloomFilter is a representation of a set of _n_ items, where the main
// requirement is to make membership queries; _i.e._, whether an item is a
// member of a set.
type BloomFilter struct {
	m uint
	k uint
	b *bitset.BitSet
}

func max(x, y uint) uint {
	if x > y {
		return x
	}
	return y
}

// New creates a new Bloom filter with _m_ bits and _k_ hashing functions
// We force _m_ and _k_ to be at least one to avoid panics.
func New(m uint, k uint) *BloomFilter {
	return &BloomFilter{max(1, m), max(1, k), bitset.New(m)}
}

// From creates a new Bloom filter with len(_data_) * 64 bits and _k_ hashing
// functions. The data slice is not going to be reset.
func From(data []uint64, k uint) *BloomFilter {
	m := uint(len(data) * 64)
	return &BloomFilter{m, k, bitset.From(data)}
}

// baseHashes returns the four hash values of data that are used to create k
// hashes
func baseHashes(data []byte) [4]uint64 {
	a1 := []byte{1} // to grab another bit of data
	hasher := murmur3.New128()
	hasher.Write(data) // #nosec
	v1, v2 := hasher.Sum128()
	hasher.Write(a1) // #nosec
	v3, v4 := hasher.Sum128()
	return [4]uint64{
		v1, v2, v3, v4,
	}
}

// location returns the ith hashed location using the four base hash values
func location(h [4]uint64, i uint) uint64 {
	ii := uint64(i)
	return h[ii%2] + ii*h[2+(((ii+(ii%2))%4)/2)]
}

// location returns the ith hashed location using the four base hash values
func (f *BloomFilter) location(h [4]uint64, i uint) uint {
	return uint(location(h, i) % uint64(f.m))
}

// EstimateParameters estimates requirements for m and k.
// Based on https://bitbucket.org/ww/bloom/src/829aa19d01d9/bloom.go
// used with permission.
func EstimateParameters(n uint, p float64) (m uint, k uint) {
	m = uint(math.Ceil(-1 * float64(n) * math.Log(p) / math.Pow(math.Log(2), 2)))
	k = uint(math.Ceil(math.Log(2) * float64(m) / float64(n)))
	return
}

// NewWithEstimates creates a new Bloom filter for about n items with fp
// false positive rate
func NewWithEstimates(n uint, fp float64) *BloomFilter {
	m, k := EstimateParameters(n, fp)
	return New(m, k)
}

// Cap returns the capacity, _m_, of a Bloom filter
func (f *BloomFilter) Cap() uint {
	return f.m
}

// K returns the number of hash functions used in the BloomFilter
func (f *BloomFilter) K() uint {
	return f.k
}

// Add data to the Bloom Filter. Returns the filter (allows chaining)
func (f *BloomFilter) Add(data []byte) *BloomFilter {
	h := baseHashes(data)
	for i := uint(0); i < f.k; i++ {
		f.b.Set(f.location(h, i))
	}
	return f
}

// Merge the data from two Bloom Filters.
func (f *BloomFilter) Merge(g *BloomFilter) error {
	// Make sure the m's and k's are the same, otherwise merging has no real use.
	if f.m != g.m {
		return fmt.Errorf("m's don't match: %d != %d", f.m, g.m)
	}

	if f.k != g.k {
		return fmt.Errorf("k's don't match: %d != %d", f.m, g.m)
	}

	f.b.InPlaceUnion(g.b)
	return nil
}

// Copy creates a copy of a Bloom filter.
func (f *BloomFilter) Copy() *BloomFilter {
	fc := New(f.m, f.k)
	fc.Merge(f) // #nosec
	return fc
}

// AddString to the Bloom Filter. Returns the filter (allows chaining)
func (f *BloomFilter) AddString(data string) *BloomFilter {
	return f.Add([]byte(data))
}

// Test returns true if the data is in the BloomFilter, false otherwise.
// If true, the result might be a false positive. If false, the data
// is definitely not in the set.
func (f *BloomFilter) Test(data []byte) bool {
	h := baseHashes(data)
	for i := uint(0); i < f.k; i++ {
		if !f.b.Test(f.location(h, i)) {
			return false
		}
	}
	return true
}

// TestString returns true if the string is in the BloomFilter, false otherwise.
// If true, the result might be a false positive. If false, the data
// is definitely not in the set.
func (f *BloomFilter) TestString(data string) bool {
	return f.Test([]byte(data))
}

// TestLocations returns true if all locations are set in the BloomFilter, false
// otherwise.
func (f *BloomFilter) TestLocations(locs []uint64) bool {
	for i := 0; i < len(locs); i++ {
		if !f.b.Test(uint(locs[i] % uint64(f.m))) {
			return false
		}
	}
	return true
}

// TestAndAdd is the equivalent to calling Test(data) then Add(data).
// Returns the result of Test.
func (f *BloomFilter) TestAndAdd(data []byte) bool {
	present := true
	h := baseHashes(data)
	for i := uint(0); i < f.k; i++ {
		l := f.location(h, i)
		if !f.b.Test(l) {
			present = false
		}
		f.b.Set(l)
	}
	return present
}

// TestAndAddString is the equivalent to calling Test(string) then Add(string).
// Returns the result of Test.
func (f *BloomFilter) TestAndAddString(data string) bool {
	return f.TestAndAdd([]byte(data))
}

// ClearAll clears all the data in a Bloom filter, removing all keys
func (f *BloomFilter) ClearAll() *BloomFilter {
	f.b.ClearAll()
	return f
}

// EstimateFalsePositiveRate returns, for a BloomFilter with a estimate of m bits
// and k hash functions, what the false positive rate will be
// while storing n entries; runs 100,000 tests. This is an empirical
// test using integers as keys. As a side-effect, it clears the BloomFilter.
func (f *BloomFilter) EstimateFalsePositiveRate(n uint) (fpRate float64) {
	rounds := uint32(100000)
	f.ClearAll()
	n1 := make([]byte, 4)
	for i := uint32(0); i < uint32(n); i++ {
		binary.BigEndian.PutUint32(n1, i)
		f.Add(n1)
	}
	fp := 0
	// test for number of rounds
	for i := uint32(0); i < rounds; i++ {
		binary.BigEndian.PutUint32(n1, i+uint32(n)+1)
		if f.Test(n1) {
			//fmt.Printf("%v failed.\n", i+uint32(n)+1)
			fp++
		}
	}
	fpRate = float64(fp) / (float64(rounds))
	f.ClearAll()
	return
}

// bloomFilterJSON is an unexported type for marshaling/unmarshaling BloomFilter struct.
type bloomFilterJSON struct {
	M uint           `json:"m"`
	K uint           `json:"k"`
	B *bitset.BitSet `json:"b"`
}

// MarshalJSON implements json.Marshaler interface.
func (f *BloomFilter) MarshalJSON() ([]byte, error) {
	return json.Marshal(bloomFilterJSON{f.m, f.k, f.b})
}

// UnmarshalJSON implements json.Unmarshaler interface.
func (f *BloomFilter) UnmarshalJSON(data []byte) error {
	var j bloomFilterJSON
	err := json.Unmarshal(data, &j)
	if err != nil {
		return err
	}
	f.m = j.M
	f.k = j.K
	f.b = j.B
	return nil
}

// WriteTo writes a binary representation of the BloomFilter to an i/o stream.
// It returns the number of bytes written.
func (f *BloomFilter) WriteTo(stream io.Writer) (int64, error) {
	err := binary.Write(stream, binary.BigEndian, uint64(f.m))
	if err != nil {
		return 0, err
	}
	err = binary.Write(stream, binary.BigEndian, uint64(f.k))
	if err != nil {
		return 0, err
	}
	numBytes, err := f.b.WriteTo(stream)
	return numBytes + int64(2*binary.Size(uint64(0))), err
}

// ReadFrom reads a binary representation of the BloomFilter (such as might
// have been written by WriteTo()) from an i/o stream. It returns the number
// of bytes read.
func (f *BloomFilter) ReadFrom(stream io.Reader) (int64, error) {
	var m, k uint64
	err := binary.Read(stream, binary.BigEndian, &m)
	if err != nil {
		return 0, err
	}
	err = binary.Read(stream, binary.BigEndian, &k)
	if err != nil {
		return 0, err
	}
	b := &bitset.BitSet{}
	numBytes, err := b.ReadFrom(stream)
	if err != nil {
		return 0, err
	}
	f.m = uint(m)
	f.k = uint(k)
	f.b = b
	return numBytes + int64(2*binary.Size(uint64(0))), nil
}

// GobEncode implements gob.GobEncoder interface.
func (f *BloomFilter) GobEncode() ([]byte, error) {
	var buf bytes.Buffer
	_, err := f.WriteTo(&buf)
	if err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

// GobDecode implements gob.GobDecoder interface.
func (f *BloomFilter) GobDecode(data []byte) error {
	buf := bytes.NewBuffer(data)
	_, err := f.ReadFrom(buf)

	return err
}

// Equal tests for the equality of two Bloom filters
func (f *BloomFilter) Equal(g *BloomFilter) bool {
	return f.m == g.m && f.k == g.k && f.b.Equal(g.b)
}

// Locations returns a list of hash locations representing a data item.
func Locations(data []byte, k uint) []uint64 {
	locs := make([]uint64, k)

	// calculate locations
	h := baseHashes(data)
	for i := uint(0); i < k; i++ {
		locs[i] = location(h, i)
	}

	return locs
}
