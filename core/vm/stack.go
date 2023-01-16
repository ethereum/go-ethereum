// Copyright 2014 The go-ethereum Authors
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
	"fmt"
	"sync"

	"github.com/holiman/uint256"
)

var stackPool = sync.Pool{
	New: func() interface{} {
		return &Stack{Data: make([]uint256.Int, 0, 16)}
	},
}

// Stack is an object for basic stack operations. Items popped to the stack are
// expected to be changed and modified. stack does not take care of adding newly
// initialised objects.
type Stack struct {
	Data []uint256.Int
}

func NewStack() (*Stack, error) {
	stack, ok := stackPool.Get().(*Stack)
	if !ok {
		return nil, fmt.Errorf("Type assertion failure: cannot get Stack pointer from stackPool")
	}
	return stack, nil
}

func (st *Stack) Push(d *uint256.Int) {
	// NOTE push limit (1024) is checked in baseCheck
	st.Data = append(st.Data, *d)
}

func (st *Stack) PushN(ds ...uint256.Int) {
	// FIXME: Is there a way to pass args by pointers.
	st.Data = append(st.Data, ds...)
}

func (st *Stack) Pop() (ret uint256.Int) {
	ret = st.Data[len(st.Data)-1]
	st.Data = st.Data[:len(st.Data)-1]
	return
}

func (st *Stack) Cap() int {
	return cap(st.Data)
}

func (st *Stack) Swap(n int) {
	st.Data[st.Len()-n], st.Data[st.Len()-1] = st.Data[st.Len()-1], st.Data[st.Len()-n]
}

func (st *Stack) Dup(n int) {
	st.Push(&st.Data[st.Len()-n])
}

func (st *Stack) Peek() *uint256.Int {
	return &st.Data[st.Len()-1]
}

func (st *Stack) Back(n int) *uint256.Int {
	return &st.Data[st.Len()-n-1]
}

func (st *Stack) Reset() {
	st.Data = st.Data[:0]
}

func (st *Stack) Len() int {
	return len(st.Data)
}

// Print dumps the content of the stack
func (st *Stack) Print() {
	fmt.Println("### stack ###")
	if len(st.Data) > 0 {
		for i, val := range st.Data {
			fmt.Printf("%-3d  %v\n", i, val)
		}
	} else {
		fmt.Println("-- empty --")
	}
	fmt.Println("#############")
}

func ReturnNormalStack(s *Stack) {
	s.Data = s.Data[:0]
	stackPool.Put(s)
}

var rStackPool = sync.Pool{
	New: func() interface{} {
		return &ReturnStack{data: make([]uint32, 0, 10)}
	},
}

func ReturnRStack(rs *ReturnStack) {
	rs.data = rs.data[:0]
	rStackPool.Put(rs)
}

// ReturnStack is an object for basic return stack operations.
type ReturnStack struct {
	data []uint32
}

func NewReturnStack() (*ReturnStack, error) {
	rStack, ok := rStackPool.Get().(*ReturnStack)
	if !ok {
		return nil, fmt.Errorf("Type assertion failure: cannot get ReturnStack pointer from rStackPool")
	}
	return rStack, nil
}

func (st *ReturnStack) Push(d uint32) {
	st.data = append(st.data, d)
}

// Pop A uint32 is sufficient as for code below 4.2G
func (st *ReturnStack) Pop() (ret uint32) {
	ret = st.data[len(st.data)-1]
	st.data = st.data[:len(st.data)-1]
	return
}

func (st *ReturnStack) Data() []uint32 {
	return st.data
}
