// CookieJar - A contestant's algorithm toolbox
// Copyright (c) 2013 Peter Szilagyi. All rights reserved.
//
// CookieJar is dual licensed: you can redistribute it and/or modify it under
// the terms of the GNU General Public License as published by the Free Software
// Foundation, either version 3 of the License, or (at your option) any later
// version.
//
// The toolbox is distributed in the hope that it will be useful, but WITHOUT
// ANY WARRANTY; without even the implied warranty of MERCHANTABILITY or
// FITNESS FOR A PARTICULAR PURPOSE.  See the GNU General Public License for
// more details.
//
// Alternatively, the CookieJar toolbox may be used in accordance with the terms
// and conditions contained in a signed written agreement between you and the
// author(s).

package prque

import (
	"math/rand"
	"testing"
)

func TestPrque(t *testing.T) {
	// Generate a batch of random data and a specific priority order
	size := 16 * blockSize
	prio := rand.Perm(size)
	data := make([]int, size)
	for i := 0; i < size; i++ {
		data[i] = rand.Int()
	}
	queue := New()
	for rep := 0; rep < 2; rep++ {
		// Fill a priority queue with the above data
		for i := 0; i < size; i++ {
			queue.Push(data[i], float32(prio[i]))
			if queue.Size() != i+1 {
				t.Errorf("queue size mismatch: have %v, want %v.", queue.Size(), i+1)
			}
		}
		// Create a map the values to the priorities for easier verification
		dict := make(map[float32]int)
		for i := 0; i < size; i++ {
			dict[float32(prio[i])] = data[i]
		}
		// Pop out the elements in priority order and verify them
		prevPrio := float32(size + 1)
		for !queue.Empty() {
			val, prio := queue.Pop()
			if prio > prevPrio {
				t.Errorf("invalid priority order: %v after %v.", prio, prevPrio)
			}
			prevPrio = prio
			if val != dict[prio] {
				t.Errorf("push/pop mismatch: have %v, want %v.", val, dict[prio])
			}
			delete(dict, prio)
		}
	}
}

func TestReset(t *testing.T) {
	// Fill the queue with some random data
	size := 16 * blockSize
	queue := New()
	for i := 0; i < size; i++ {
		queue.Push(rand.Int(), rand.Float32())
	}
	// Reset and ensure it's empty
	queue.Reset()
	if !queue.Empty() {
		t.Errorf("priority queue not empty after reset: %v", queue)
	}
}

func BenchmarkPush(b *testing.B) {
	// Create some initial data
	data := make([]int, b.N)
	prio := make([]float32, b.N)
	for i := 0; i < len(data); i++ {
		data[i] = rand.Int()
		prio[i] = rand.Float32()
	}
	// Execute the benchmark
	b.ResetTimer()
	queue := New()
	for i := 0; i < len(data); i++ {
		queue.Push(data[i], prio[i])
	}
}

func BenchmarkPop(b *testing.B) {
	// Create some initial data
	data := make([]int, b.N)
	prio := make([]float32, b.N)
	for i := 0; i < len(data); i++ {
		data[i] = rand.Int()
		prio[i] = rand.Float32()
	}
	queue := New()
	for i := 0; i < len(data); i++ {
		queue.Push(data[i], prio[i])
	}
	// Execute the benchmark
	b.ResetTimer()
	for !queue.Empty() {
		queue.Pop()
	}
}
