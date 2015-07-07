// Copyright 2014 The go-ethereum Authors
// This file is part of go-ethereum.
//
// go-ethereum is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// go-ethereum is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with go-ethereum.  If not, see <http://www.gnu.org/licenses/>.

package vm

import (
	"fmt"
	"math/big"
)

func newstack() *stack {
	return &stack{}
}

type stack struct {
	data []*big.Int
	ptr  int
}

func (st *stack) Data() []*big.Int {
	return st.data[:st.ptr]
}

func (st *stack) push(d *big.Int) {
	// NOTE push limit (1024) is checked in baseCheck
	stackItem := new(big.Int).Set(d)
	if len(st.data) > st.ptr {
		st.data[st.ptr] = stackItem
	} else {
		st.data = append(st.data, stackItem)
	}
	st.ptr++
}

func (st *stack) pop() (ret *big.Int) {
	st.ptr--
	ret = st.data[st.ptr]
	return
}

func (st *stack) len() int {
	return st.ptr
}

func (st *stack) swap(n int) {
	st.data[st.len()-n], st.data[st.len()-1] = st.data[st.len()-1], st.data[st.len()-n]
}

func (st *stack) dup(n int) {
	st.push(st.data[st.len()-n])
}

func (st *stack) peek() *big.Int {
	return st.data[st.len()-1]
}

func (st *stack) require(n int) error {
	if st.len() < n {
		return fmt.Errorf("stack underflow (%d <=> %d)", len(st.data), n)
	}
	return nil
}

func (st *stack) Print() {
	fmt.Println("### stack ###")
	if len(st.data) > 0 {
		for i, val := range st.data {
			fmt.Printf("%-3d  %v\n", i, val)
		}
	} else {
		fmt.Println("-- empty --")
	}
	fmt.Println("#############")
}
