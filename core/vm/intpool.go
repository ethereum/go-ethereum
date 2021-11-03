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

package vm

import (
	"math/big"
	"sync"
)

var checkVal = big.NewInt(-42)

const poolLimit = 256

// intPool is a pool of big integers that
// can be reused for all big.Int operations.
type intPool struct {
	pool *Stack
}

func newIntPool() *intPool {
	return &intPool{pool: newstack()}
}

// get retrieves a big int from the pool, allocating one if the pool is empty.
// Note, the returned int's value is arbitrary and will not be zeroed!
func (p *intPool) get() *big.Int {
	if p.pool.len() > 0 {
		return p.pool.pop()
	}
	return new(big.Int)
}

// getZero retrieves a big int from the pool, setting it to zero or allocating
// a new one if the pool is empty.
func (p *intPool) getZero() *big.Int {
	if p.pool.len() > 0 {
		return p.pool.pop().SetUint64(0)
	}
	return new(big.Int)
}

// putOne returns an allocated big int to the pool to be later reused by get calls.
// Note, the values as saved as is; neither put nor get zeroes the ints out!
// As opposed to 'put' with variadic args, this method becomes inlined by the
// go compiler
func (p *intPool) putOne(i *big.Int) {
	if len(p.pool.data) > poolLimit {
		return
	}
	p.pool.push(i)
}

// put returns an allocated big int to the pool to be later reused by get calls.
// Note, the values as saved as is; neither put nor get zeroes the ints out!
func (p *intPool) put(is ...*big.Int) {
	if len(p.pool.data) > poolLimit {
		return
	}
	for _, i := range is {
		// verifyPool is a build flag. Pool verification makes sure the integrity
		// of the integer pool by comparing values to a default value.
		if verifyPool {
			i.Set(checkVal)
		}
		p.pool.push(i)
	}
}

// The intPool pool's default capacity
const poolDefaultCap = 25

// intPoolPool manages a pool of intPools.
type intPoolPool struct {
	pools []*intPool
	lock  sync.Mutex
}

var poolOfIntPools = &intPoolPool{
	pools: make([]*intPool, 0, poolDefaultCap),
}

// get is looking for an available pool to return.
func (ipp *intPoolPool) get() *intPool {
	ipp.lock.Lock()
	defer ipp.lock.Unlock()

	if len(poolOfIntPools.pools) > 0 {
		ip := ipp.pools[len(ipp.pools)-1]
		ipp.pools = ipp.pools[:len(ipp.pools)-1]
		return ip
	}
	return newIntPool()
}

// put a pool that has been allocated with get.
func (ipp *intPoolPool) put(ip *intPool) {
	ipp.lock.Lock()
	defer ipp.lock.Unlock()

	if len(ipp.pools) < cap(ipp.pools) {
		ipp.pools = append(ipp.pools, ip)
	}
}
