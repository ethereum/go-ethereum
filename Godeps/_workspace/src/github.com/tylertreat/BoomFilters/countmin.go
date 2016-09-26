package boom

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"hash"
	"hash/fnv"
	"io"
	"math"
)

// CountMinSketch implements a Count-Min Sketch as described by Cormode and
// Muthukrishnan in An Improved Data Stream Summary: The Count-Min Sketch and
// its Applications:
//
// http://dimacs.rutgers.edu/~graham/pubs/papers/cm-full.pdf
//
// A Count-Min Sketch (CMS) is a probabilistic data structure which
// approximates the frequency of events in a data stream. Unlike a hash map, a
// CMS uses sub-linear space at the expense of a configurable error factor.
// Similar to Counting Bloom filters, items are hashed to a series of buckets,
// which increment a counter. The frequency of an item is estimated by taking
// the minimum of each of the item's respective counter values.
//
// Count-Min Sketches are useful for counting the frequency of events in
// massive data sets or unbounded streams online. In these situations, storing
// the entire data set or allocating counters for every event in memory is
// impractical. It may be possible for offline processing, but real-time
// processing requires fast, space-efficient solutions like the CMS. For
// approximating set cardinality, refer to the HyperLogLog.
type CountMinSketch struct {
	matrix  [][]uint64  // count matrix
	width   uint        // matrix width
	depth   uint        // matrix depth
	count   uint64      // number of items added
	epsilon float64     // relative-accuracy factor
	delta   float64     // relative-accuracy probability
	hash    hash.Hash64 // hash function (kernel for all depth functions)
}

// NewCountMinSketch creates a new Count-Min Sketch whose relative accuracy is
// within a factor of epsilon with probability delta. Both of these parameters
// affect the space and time complexity.
func NewCountMinSketch(epsilon, delta float64) *CountMinSketch {
	var (
		width  = uint(math.Ceil(math.E / epsilon))
		depth  = uint(math.Ceil(math.Log(1 / delta)))
		matrix = make([][]uint64, depth)
	)

	for i := uint(0); i < depth; i++ {
		matrix[i] = make([]uint64, width)
	}

	return &CountMinSketch{
		matrix:  matrix,
		width:   width,
		depth:   depth,
		epsilon: epsilon,
		delta:   delta,
		hash:    fnv.New64(),
	}
}

// Epsilon returns the relative-accuracy factor, epsilon.
func (c *CountMinSketch) Epsilon() float64 {
	return c.epsilon
}

// Delta returns the relative-accuracy probability, delta.
func (c *CountMinSketch) Delta() float64 {
	return c.delta
}

// TotalCount returns the number of items added to the sketch.
func (c *CountMinSketch) TotalCount() uint64 {
	return c.count
}

// Add will add the data to the set. Returns the CountMinSketch to allow for
// chaining.
func (c *CountMinSketch) Add(data []byte) *CountMinSketch {
	lower, upper := hashKernel(data, c.hash)

	// Increment count in each row.
	for i := uint(0); i < c.depth; i++ {
		c.matrix[i][(uint(lower)+uint(upper)*i)%c.width]++
	}

	c.count++
	return c
}

// Count returns the approximate count for the specified item, correct within
// epsilon * total count with a probability of delta.
func (c *CountMinSketch) Count(data []byte) uint64 {
	var (
		lower, upper = hashKernel(data, c.hash)
		count        = uint64(math.MaxUint64)
	)

	for i := uint(0); i < c.depth; i++ {
		count = uint64(math.Min(float64(count),
			float64(c.matrix[i][(uint(lower)+uint(upper)*i)%c.width])))
	}

	return count
}

// Merge combines this CountMinSketch with another. Returns an error if the
// matrix width and depth are not equal.
func (c *CountMinSketch) Merge(other *CountMinSketch) error {
	if c.depth != other.depth {
		return errors.New("matrix depth must match")
	}

	if c.width != other.width {
		return errors.New("matrix width must match")
	}

	for i := uint(0); i < c.depth; i++ {
		for j := uint(0); j < c.width; j++ {
			c.matrix[i][j] += other.matrix[i][j]
		}
	}

	c.count += other.count
	return nil
}

// Reset restores the CountMinSketch to its original state. It returns itself
// to allow for chaining.
func (c *CountMinSketch) Reset() *CountMinSketch {
	matrix := make([][]uint64, c.depth)
	for i := uint(0); i < c.depth; i++ {
		matrix[i] = make([]uint64, c.width)
	}

	c.matrix = matrix
	c.count = 0
	return c
}

// SetHash sets the hashing function used.
func (c *CountMinSketch) SetHash(h hash.Hash64) {
	c.hash = h
}

// WriteDataTo writes a binary representation of the CMS data to
// an io stream. It returns the number of bytes written and error
func (c *CountMinSketch) WriteDataTo(stream io.Writer) (int, error) {
	buf := new(bytes.Buffer)
	// serialize epsilon and delta as cms configuration check
	err := binary.Write(buf, binary.LittleEndian, c.epsilon)
	if err != nil {
		return 0, err
	}
	err = binary.Write(buf, binary.LittleEndian, c.delta)
	if err != nil {
		return 0, err
	}
	err = binary.Write(buf, binary.LittleEndian, c.count)
	if err != nil {
		return 0, err
	}
	// encode matrix
	for i := range c.matrix {
		err = binary.Write(buf, binary.LittleEndian, c.matrix[i])
		if err != nil {
			return 0, err
		}
	}

	return stream.Write(buf.Bytes())
}

// ReadDataFrom reads a binary representation of the CMS data written
// by WriteDataTo() from io stream. It returns the number of bytes read
// and error
// If serialized CMS configuration is different it returns error with expected params
func (c *CountMinSketch) ReadDataFrom(stream io.Reader) (int, error) {
	var (
		count          uint64
		epsilon, delta float64
	)

	err := binary.Read(stream, binary.LittleEndian, &epsilon)
	if err != nil {
		return 0, err
	}
	err = binary.Read(stream, binary.LittleEndian, &delta)
	if err != nil {
		return 0, err
	}

	// check if serialized and target cms configurations are same
	if c.epsilon != epsilon || c.delta != delta {
		return 0, fmt.Errorf("expected cms values for epsilon %f and delta %f", epsilon, delta)
	}

	err = binary.Read(stream, binary.LittleEndian, &count)
	if err != nil {
		return 0, err
	}

	for i := uint(0); i < uint(c.depth); i++ {
		err = binary.Read(stream, binary.LittleEndian, c.matrix[i])
	}
	// count size of matrix and count
	size := int(c.depth*c.width)*binary.Size(uint64(0)) + binary.Size(count) + 2*binary.Size(float64(0))

	c.count = count

	return size, err
}
