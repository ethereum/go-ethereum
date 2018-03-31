// Package mpool provides a sync.Pool equivalent that buckets incoming
// requests to one of 32 sub-pools, one for each power of 2, 0-32.
//
//	import "github.com/libp2p/go-msgio/mpool"
//	var p mpool.Pool
//
//	small := make([]byte, 1024)
//	large := make([]byte, 4194304)
//	p.Put(1024, small)
//	p.Put(4194304, large)
//
//	small2 := p.Get(1024).([]byte)
//	large2 := p.Get(4194304).([]byte)
//	fmt.Println("small2 len:", len(small2))
//	fmt.Println("large2 len:", len(large2))
//
//	// Output:
//	// small2 len: 1024
//	// large2 len: 4194304
//
package mpool

import (
	"fmt"
	"sync"
)

// ByteSlicePool is a static Pool for reusing byteslices of various sizes.
var ByteSlicePool = &Pool{
	New: func(length int) interface{} {
		return make([]byte, length)
	},
}

// MaxLength is the maximum length of an element that can be added to the Pool.
const MaxLength = 1 << 32

// Pool is a pool to handle cases of reusing elements of varying sizes.
// It maintains up to  32 internal pools, for each power of 2 in 0-32.
type Pool struct {
	pools [32]sync.Pool // a list of singlePools

	// New is a function that constructs a new element in the pool, with given len
	New func(len int) interface{}
}

func (p *Pool) getPool(idx uint32) *sync.Pool {
	if idx > uint32(len(p.pools)) {
		panic(fmt.Errorf("index too large: %d", idx))
	}
	return &p.pools[idx]
}

// Get selects an arbitrary item from the Pool, removes it from the Pool,
// and returns it to the caller. Get may choose to ignore the pool and
// treat it as empty. Callers should not assume any relation between values
// passed to Put and the values returned by Get.
//
// If Get would otherwise return nil and p.New is non-nil, Get returns the
// result of calling p.New.
func (p *Pool) Get(length uint32) interface{} {
	idx := nextPowerOfTwo(length)
	sp := p.getPool(idx)
	// fmt.Printf("Get(%d) idx(%d)\n", length, idx)
	val := sp.Get()
	if val == nil && p.New != nil {
		val = p.New(0x1 << idx)
	}
	return val
}

// Put adds x to the pool.
func (p *Pool) Put(length uint32, val interface{}) {
	idx := prevPowerOfTwo(length)
	// fmt.Printf("Put(%d, -) idx(%d)\n", length, idx)
	sp := p.getPool(idx)
	sp.Put(val)
}

func nextPowerOfTwo(v uint32) uint32 {
	// fmt.Printf("nextPowerOfTwo(%d) ", v)
	v--
	v |= v >> 1
	v |= v >> 2
	v |= v >> 4
	v |= v >> 8
	v |= v >> 16
	v++

	// fmt.Printf("-> %d", v)

	i := uint32(0)
	for ; v > 1; i++ {
		v = v >> 1
	}

	// fmt.Printf("-> %d\n", i)
	return i
}

func prevPowerOfTwo(num uint32) uint32 {
	next := nextPowerOfTwo(num)
	// fmt.Printf("prevPowerOfTwo(%d) next: %d", num, next)
	switch {
	case num == (1 << next): // num is a power of 2
	case next == 0:
	default:
		next = next - 1 // smaller
	}
	// fmt.Printf(" = %d\n", next)
	return next
}
