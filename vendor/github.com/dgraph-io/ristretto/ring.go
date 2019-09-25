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

package ristretto

import (
	"sync"
	"sync/atomic"
	"time"
)

const (
	ringLossy byte = iota
	ringLossless
)

// ringConsumer is the user-defined object responsible for receiving and
// processing items in batches when buffers are drained.
type ringConsumer interface {
	Push([]uint64) bool
}

// ringStripe is a singular ring buffer that is not concurrent safe.
type ringStripe struct {
	consumer ringConsumer
	data     []uint64
	capacity int
	busy     int32
}

func newRingStripe(config *ringConfig) *ringStripe {
	return &ringStripe{
		consumer: config.Consumer,
		data:     make([]uint64, 0, config.Capacity),
		capacity: int(config.Capacity),
	}
}

// Push appends an item in the ring buffer and drains (copies items and
// sends to Consumer) if full.
func (s *ringStripe) Push(item uint64) {
	s.data = append(s.data, item)
	// if we should drain
	if len(s.data) >= s.capacity {
		// Send elements to consumer. Create a new one.
		if s.consumer.Push(s.data) {
			s.data = make([]uint64, 0, s.capacity)
		} else {
			s.data = s.data[:0]
		}
	}
}

// ringConfig is passed to newRingBuffer with parameters.
type ringConfig struct {
	Consumer ringConsumer
	Stripes  int64
	Capacity int64
}

// ringBuffer stores multiple buffers (stripes) and distributes Pushed items
// between them to lower contention.
//
// This implements the "batching" process described in the BP-Wrapper paper
// (section III part A).
type ringBuffer struct {
	stripes []*ringStripe
	pool    *sync.Pool
	push    func(*ringBuffer, uint64)
	rand    int
	mask    int
}

// newRingBuffer returns a striped ring buffer. The Type can be either LOSSY or
// LOSSLESS. LOSSY should provide better performance. The Consumer in ringConfig
// will be called when individual stripes are full and need to drain their
// elements.
func newRingBuffer(ringType byte, config *ringConfig) *ringBuffer {
	if ringType == ringLossy {
		// LOSSY buffers use a very simple sync.Pool for concurrently reusing
		// stripes. We do lose some stripes due to GC (unheld items in sync.Pool
		// are cleared), but the performance gains generally outweigh the small
		// percentage of elements lost. The performance primarily comes from
		// low-level runtime functions used in the standard library that aren't
		// available to us (such as runtime_procPin()).
		return &ringBuffer{
			pool: &sync.Pool{
				New: func() interface{} { return newRingStripe(config) },
			},
			push: pushLossy,
		}
	}
	// begin LOSSLESS buffer handling
	//
	// unlike lossy, lossless manually handles all stripes
	stripes := make([]*ringStripe, config.Stripes)
	for i := range stripes {
		stripes[i] = newRingStripe(config)
	}
	return &ringBuffer{
		stripes: stripes,
		mask:    int(config.Stripes - 1),
		rand:    int(time.Now().UnixNano()), // random seed for picking stripes
		push:    pushLossless,
	}
}

// Push adds an element to one of the internal stripes and possibly drains if
// the stripe becomes full.
func (b *ringBuffer) Push(item uint64) {
	b.push(b, item)
}

func pushLossy(b *ringBuffer, item uint64) {
	// reuse or create a new stripe
	stripe := b.pool.Get().(*ringStripe)
	stripe.Push(item)
	b.pool.Put(stripe)
}

func pushLossless(b *ringBuffer, item uint64) {
	// try to find an available stripe
	for i := 0; ; i = (i + 1) & b.mask {
		if atomic.CompareAndSwapInt32(&b.stripes[i].busy, 0, 1) {
			// try to get exclusive lock on the stripe
			b.stripes[i].Push(item)
			// unlock
			atomic.StoreInt32(&b.stripes[i].busy, 0)
			return
		}
	}
}
