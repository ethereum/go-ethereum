// Copyright 2020 The go-ethereum Authors
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

package threadpool

import (
	"fmt"
	"sync"
	"testing"
)

func TestThreadPool(t *testing.T) {
	tp := NewThreadPool(10)
	a := tp.Get(5)
	b := tp.Get(4)
	c := tp.Get(5)
	d := tp.Get(1)
	e := tp.Get(0)
	f := tp.Get(0)
	g := tp.Get(0)
	tp.Put(1)
	tp.Get(0)
	fmt.Printf("%v %v %v %v %v %v %v", a, b, c, d, e, f, g)
}

func TestMaxTasks(t *testing.T) {
	tp := NewThreadPool(10)
	a := tp.Get(1)
	if a != 1 {
		t.Fail()
	}
}

func TestThreadPoolRandom(t *testing.T) {
	tp := NewThreadPool(10)
	wg := sync.WaitGroup{}
	wg.Add(1000)
	for i := 0; i < 1000; i++ {
		go func(i int) {
			a := tp.Get(0)
			fmt.Printf("%v has %v threads\n", i, a)
			tp.Put(a)
			wg.Done()
		}(i)
	}
	wg.Wait()
}

func BenchmarkThreadPool(t *testing.B) {
	tp := NewThreadPool(10)
	wg := sync.WaitGroup{}
	wg.Add(t.N)
	for i := 0; i < t.N; i++ {
		go func(i int) {
			a := tp.Get(0)
			tp.Put(a)
			wg.Done()
		}(i)
	}
	wg.Wait()
}
