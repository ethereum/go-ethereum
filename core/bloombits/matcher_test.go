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

package bloombits

import (
	"math/rand"
	"sync/atomic"
	"testing"
	"time"
)

const testSectionSize = 4096

// Tests the matcher pipeline on a single continuous workflow without interrupts.
func TestMatcherContinuous(t *testing.T) {
	testMatcherDiffBatches(t, [][]bloomIndexes{{{10, 20, 30}}}, 100000, false, 75)
	testMatcherDiffBatches(t, [][]bloomIndexes{{{32, 3125, 100}}, {{40, 50, 10}}}, 100000, false, 81)
	testMatcherDiffBatches(t, [][]bloomIndexes{{{4, 8, 11}, {7, 8, 17}}, {{9, 9, 12}, {15, 20, 13}}, {{18, 15, 15}, {12, 10, 4}}}, 10000, false, 36)
}

// Tests the matcher pipeline on a constantly interrupted and resumed work pattern
// with the aim of ensuring data items are requested only once.
func TestMatcherIntermittent(t *testing.T) {
	testMatcherDiffBatches(t, [][]bloomIndexes{{{10, 20, 30}}}, 100000, true, 75)
	testMatcherDiffBatches(t, [][]bloomIndexes{{{32, 3125, 100}}, {{40, 50, 10}}}, 100000, true, 81)
	testMatcherDiffBatches(t, [][]bloomIndexes{{{4, 8, 11}, {7, 8, 17}}, {{9, 9, 12}, {15, 20, 13}}, {{18, 15, 15}, {12, 10, 4}}}, 10000, true, 36)
}

// Tests the matcher pipeline on random input to hopefully catch anomalies.
func TestMatcherRandom(t *testing.T) {
	for i := 0; i < 10; i++ {
		testMatcherBothModes(t, makeRandomIndexes([]int{1}, 50), 10000, 0)
		testMatcherBothModes(t, makeRandomIndexes([]int{3}, 50), 10000, 0)
		testMatcherBothModes(t, makeRandomIndexes([]int{2, 2, 2}, 20), 10000, 0)
		testMatcherBothModes(t, makeRandomIndexes([]int{5, 5, 5}, 50), 10000, 0)
		testMatcherBothModes(t, makeRandomIndexes([]int{4, 4, 4}, 20), 10000, 0)
	}
}

// makeRandomIndexes generates a random filter system, composed on multiple filter
// criteria, each having one bloom list component for the address and arbitrarilly
// many topic bloom list components.
func makeRandomIndexes(lengths []int, max int) [][]bloomIndexes {
	res := make([][]bloomIndexes, len(lengths))
	for i, topics := range lengths {
		res[i] = make([]bloomIndexes, topics)
		for j := 0; j < topics; j++ {
			for k := 0; k < len(res[i][j]); k++ {
				res[i][j][k] = uint(rand.Intn(max-1) + 2)
			}
		}
	}
	return res
}

// testMatcherDiffBatches runs the given matches test in single-delivery and also
// in batches delivery mode, verifying that all kinds of deliveries are handled
// correctly withn.
func testMatcherDiffBatches(t *testing.T, filter [][]bloomIndexes, blocks uint64, intermittent bool, retrievals uint32) {
	singleton := testMatcher(t, filter, blocks, intermittent, retrievals, 1)
	batched := testMatcher(t, filter, blocks, intermittent, retrievals, 16)

	if singleton != batched {
		t.Errorf("filter = %v blocks = %v intermittent = %v: request count mismatch, %v in signleton vs. %v in batched mode", filter, blocks, intermittent, singleton, batched)
	}
}

// testMatcherBothModes runs the given matcher test in both continuous as well as
// in intermittent mode, verifying that the request counts match each other.
func testMatcherBothModes(t *testing.T, filter [][]bloomIndexes, blocks uint64, retrievals uint32) {
	continuous := testMatcher(t, filter, blocks, false, retrievals, 16)
	intermittent := testMatcher(t, filter, blocks, true, retrievals, 16)

	if continuous != intermittent {
		t.Errorf("filter = %v blocks = %v: request count mismatch, %v in continuous vs. %v in intermittent mode", filter, blocks, continuous, intermittent)
	}
}

// testMatcher is a generic tester to run the given matcher test and return the
// number of requests made for cross validation between different modes.
func testMatcher(t *testing.T, filter [][]bloomIndexes, blocks uint64, intermittent bool, retrievals uint32, maxReqCount int) uint32 {
	// Create a new matcher an simulate our explicit random bitsets
	matcher := NewMatcher(testSectionSize, nil, nil)

	matcher.addresses = filter[0]
	matcher.topics = filter[1:]

	for _, rule := range filter {
		for _, topic := range rule {
			for _, bit := range topic {
				matcher.addScheduler(bit)
			}
		}
	}
	// Track the number of retrieval requests made
	var requested uint32

	// Start the matching session for the filter and the retriver goroutines
	quit := make(chan struct{})
	matches := make(chan uint64, 16)

	session, err := matcher.Start(0, blocks-1, matches)
	if err != nil {
		t.Fatalf("failed to stat matcher session: %v", err)
	}
	startRetrievers(session, quit, &requested, maxReqCount)

	// Iterate over all the blocks and verify that the pipeline produces the correct matches
	for i := uint64(0); i < blocks; i++ {
		if expMatch3(filter, i) {
			match, ok := <-matches
			if !ok {
				t.Errorf("filter = %v  blocks = %v  intermittent = %v: expected #%v, results channel closed", filter, blocks, intermittent, i)
				return 0
			}
			if match != i {
				t.Errorf("filter = %v  blocks = %v  intermittent = %v: expected #%v, got #%v", filter, blocks, intermittent, i, match)
			}
			// If we're testing intermittent mode, abort and restart the pipeline
			if intermittent {
				session.Close(time.Second)
				close(quit)

				quit = make(chan struct{})
				matches = make(chan uint64, 16)

				session, err = matcher.Start(i+1, blocks-1, matches)
				if err != nil {
					t.Fatalf("failed to stat matcher session: %v", err)
				}
				startRetrievers(session, quit, &requested, maxReqCount)
			}
		}
	}
	// Ensure the result channel is torn down after the last block
	match, ok := <-matches
	if ok {
		t.Errorf("filter = %v  blocks = %v  intermittent = %v: expected closed channel, got #%v", filter, blocks, intermittent, match)
	}
	// Clean up the session and ensure we match the expected retrieval count
	session.Close(time.Second)
	close(quit)

	if retrievals != 0 && requested != retrievals {
		t.Errorf("filter = %v  blocks = %v  intermittent = %v: request count mismatch, have #%v, want #%v", filter, blocks, intermittent, requested, retrievals)
	}
	return requested
}

// startRetrievers starts a batch of goroutines listening for section requests
// and serving them.
func startRetrievers(session *MatcherSession, quit chan struct{}, retrievals *uint32, batch int) {
	requests := make(chan chan *Retrieval)

	for i := 0; i < 10; i++ {
		// Start a multiplexer to test multiple threaded execution
		go session.Multiplex(batch, 100*time.Microsecond, requests)

		// Start a services to match the above multiplexer
		go func() {
			for {
				// Wait for a service request or a shutdown
				select {
				case <-quit:
					return

				case request := <-requests:
					task := <-request

					task.Bitsets = make([][]byte, len(task.Sections))
					for i, section := range task.Sections {
						if rand.Int()%4 != 0 { // Handle occasional missing deliveries
							task.Bitsets[i] = generateBitset(task.Bit, section)
							atomic.AddUint32(retrievals, 1)
						}
					}
					request <- task
				}
			}
		}()
	}
}

// generateBitset generates the rotated bitset for the given bloom bit and section
// numbers.
func generateBitset(bit uint, section uint64) []byte {
	bitset := make([]byte, testSectionSize/8)
	for i := 0; i < len(bitset); i++ {
		for b := 0; b < 8; b++ {
			blockIdx := section*testSectionSize + uint64(i*8+b)
			bitset[i] += bitset[i]
			if (blockIdx % uint64(bit)) == 0 {
				bitset[i]++
			}
		}
	}
	return bitset
}

func expMatch1(filter bloomIndexes, i uint64) bool {
	for _, ii := range filter {
		if (i % uint64(ii)) != 0 {
			return false
		}
	}
	return true
}

func expMatch2(filter []bloomIndexes, i uint64) bool {
	for _, ii := range filter {
		if expMatch1(ii, i) {
			return true
		}
	}
	return false
}

func expMatch3(filter [][]bloomIndexes, i uint64) bool {
	for _, ii := range filter {
		if !expMatch2(ii, i) {
			return false
		}
	}
	return true
}
