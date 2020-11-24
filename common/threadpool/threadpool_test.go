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
	"testing"
)

func TestThreadPool(t *testing.T) {
	tp := NewThreadPool(10)
	a := tp.Get()
	b := tp.Get()
	c := tp.Get()
	d := tp.Get()
	e := tp.Get()
	f := tp.Get()
	g := tp.Get()
	tp.Put(1)
	tp.Get()
	panic(fmt.Sprintf("%v %v %v %v %v %v %v", a, b, c, d, e, f, g))
}
