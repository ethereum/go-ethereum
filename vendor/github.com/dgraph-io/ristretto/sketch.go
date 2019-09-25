/*
 * Copyright 2019 Dgraph Labs, Inc. and Contributors
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

// This package includes multiple probabalistic data structures needed for
// admission/eviction metadata. Most are Counting Bloom Filter variations, but
// a caching-specific feature that is also required is a "freshness" mechanism,
// which basically serves as a "lifetime" process. This freshness mechanism
// was described in the original TinyLFU paper [1], but other mechanisms may
// be better suited for certain data distributions.
//
// [1]: https://arxiv.org/abs/1512.00727
package ristretto

import (
	"fmt"
)

// cmSketch is a Count-Min sketch implementation with 4-bit counters, heavily
// based on Damian Gryski's CM4 [1].
//
// [1]: https://github.com/dgryski/go-tinylfu/blob/master/cm4.go
type cmSketch struct {
	rows [cmDepth]cmRow
	mask uint32
}

const (
	// cmDepth is the number of counter copies to store (think of it as rows)
	cmDepth = 4
)

func newCmSketch(numCounters int64) *cmSketch {
	if numCounters == 0 {
		panic("cmSketch: bad numCounters")
	}
	// get the next power of 2 for better cache performance
	numCounters = next2Power(numCounters)
	// sketch with FNV-64a hashing algorithm
	sketch := &cmSketch{
		mask: uint32(numCounters - 1),
	}
	// initialize rows of counters
	for i := 0; i < cmDepth; i++ {
		sketch.rows[i] = newCmRow(numCounters)
	}
	return sketch
}

// Increment increments the count(ers) for the specified key.
func (s *cmSketch) Increment(hashed uint64) {
	l, r := uint32(hashed), uint32(hashed>>32)
	for i := range s.rows {
		// increment the counter on each row
		s.rows[i].increment((l + uint32(i)*r) & s.mask)
	}
}

// Estimate returns the value of the specified key.
func (s *cmSketch) Estimate(hashed uint64) int64 {
	l, r := uint32(hashed), uint32(hashed>>32)
	min := byte(255)
	for i := range s.rows {
		// find the smallest counter value from all the rows
		if v := s.rows[i].get((l + uint32(i)*r) & s.mask); v < min {
			min = v
		}
	}
	return int64(min)
}

// Reset halves all counter values.
func (s *cmSketch) Reset() {
	for _, r := range s.rows {
		r.reset()
	}
}

func (s *cmSketch) string() string {
	var state string
	for i := range s.rows {
		state += "  [ "
		state += s.rows[i].string()
		state += " ]\n"
	}
	return state
}

// cmRow is a row of bytes, with each byte holding two counters
type cmRow []byte

func newCmRow(numCounters int64) cmRow {
	return make(cmRow, numCounters/2)
}

func (r cmRow) get(n uint32) byte {
	return byte(r[n/2]>>((n&1)*4)) & 0x0f
}

func (r cmRow) increment(n uint32) {
	// index of the counter
	i := n / 2
	// shift distance (even 0, odd 4)
	s := (n & 1) * 4
	// counter value
	v := (r[i] >> s) & 0x0f
	// only increment if not max value (overflow wrap is bad for LFU)
	if v < 15 {
		r[i] += 1 << s
	}
}

func (r cmRow) reset() {
	// halve each counter
	for i := range r {
		r[i] = (r[i] >> 1) & 0x77
	}
}

func (r cmRow) string() string {
	var state string
	for i := uint64(0); i < uint64(len(r)*2); i++ {
		state += fmt.Sprintf("%02d ", (r[(i/2)]>>((i&1)*4))&0x0f)
	}
	state = state[:len(state)-1]
	return state
}

// next2Power rounds x up to the next power of 2, if it's not already one.
func next2Power(x int64) int64 {
	x--
	x |= x >> 1
	x |= x >> 2
	x |= x >> 4
	x |= x >> 8
	x |= x >> 16
	x |= x >> 32
	x++
	return x
}

/*
// TODO
//
// Fingerprint Counting Bloom Filter (FP-CBF): lower false positive rates than
// basic CBF with little added complexity.
//
// https://doi.org/10.1016/j.ipl.2015.11.002
type FPCBF struct {
}

func (c *FPCBF) Push(keys []ring.Element)      {}
func (c *FPCBF) Estimate(hashed int64) int64 { return 0 }

// TODO
//
// d-left Counting Bloom Filter: based on d-left hashing which allows for much
// better space efficiency (usually saving a factor of 2 or more).
//
// https://link.springer.com/chapter/10.1007/11841036_61
type DLCBF struct {
}

func (c *DLCBF) Push(keys []ring.Element)      {}
func (c *DLCBF) Estimate(hashed int64) int64 { return 0 }

// TODO
//
// Bloom Clock: this might be a good route for keeping track of LRU information
// in a space efficient, probabilistic manner.
//
// https://arxiv.org/abs/1905.13064
type BC struct{}

func (c *BC) Push(keys []ring.Element)      {}
func (c *BC) Estimate(hashed int64) int64 { return 0 }
*/
