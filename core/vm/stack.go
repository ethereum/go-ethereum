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
	"sync"

	"github.com/holiman/uint256"
)

var stackPool = sync.Pool{
	New: func() interface{} {
		return &Stack{data: make([]uint256.Int, 0, 16)}
	},
}

// Stack is an object for basic stack operations. Items popped to the stack are
// expected to be changed and modified. stack does not take care of adding newly
// initialized objects.
type Stack struct {
	data []uint256.Int
}

func Newstack() *Stack {
	return stackPool.Get().(*Stack)
}

func returnStack(s *Stack) {
	s.data = s.data[:0]
	stackPool.Put(s)
}

// Data returns the underlying uint256.Int array.
func (st *Stack) Data() []uint256.Int {
	return st.data
}

func (st *Stack) Push(d *uint256.Int) {
	// NOTE push limit (1024) is checked in baseCheck
	st.data = append(st.data, *d)
}

func (st *Stack) Pop() (ret uint256.Int) {
	ret = st.data[len(st.data)-1]
	st.data = st.data[:len(st.data)-1]
	return
}

func (st *Stack) Len() int {
	return len(st.data)
}

func (st *Stack) swap1() {
	st.data[st.Len()-2], st.data[st.Len()-1] = st.data[st.Len()-1], st.data[st.Len()-2]
}
func (st *Stack) swap2() {
	st.data[st.Len()-3], st.data[st.Len()-1] = st.data[st.Len()-1], st.data[st.Len()-3]
}
func (st *Stack) swap3() {
	st.data[st.Len()-4], st.data[st.Len()-1] = st.data[st.Len()-1], st.data[st.Len()-4]
}
func (st *Stack) swap4() {
	st.data[st.Len()-5], st.data[st.Len()-1] = st.data[st.Len()-1], st.data[st.Len()-5]
}
func (st *Stack) swap5() {
	st.data[st.Len()-6], st.data[st.Len()-1] = st.data[st.Len()-1], st.data[st.Len()-6]
}
func (st *Stack) swap6() {
	st.data[st.Len()-7], st.data[st.Len()-1] = st.data[st.Len()-1], st.data[st.Len()-7]
}
func (st *Stack) swap7() {
	st.data[st.Len()-8], st.data[st.Len()-1] = st.data[st.Len()-1], st.data[st.Len()-8]
}
func (st *Stack) swap8() {
	st.data[st.Len()-9], st.data[st.Len()-1] = st.data[st.Len()-1], st.data[st.Len()-9]
}
func (st *Stack) swap9() {
	st.data[st.Len()-10], st.data[st.Len()-1] = st.data[st.Len()-1], st.data[st.Len()-10]
}
func (st *Stack) swap10() {
	st.data[st.Len()-11], st.data[st.Len()-1] = st.data[st.Len()-1], st.data[st.Len()-11]
}
func (st *Stack) swap11() {
	st.data[st.Len()-12], st.data[st.Len()-1] = st.data[st.Len()-1], st.data[st.Len()-12]
}
func (st *Stack) swap12() {
	st.data[st.Len()-13], st.data[st.Len()-1] = st.data[st.Len()-1], st.data[st.Len()-13]
}
func (st *Stack) swap13() {
	st.data[st.Len()-14], st.data[st.Len()-1] = st.data[st.Len()-1], st.data[st.Len()-14]
}
func (st *Stack) swap14() {
	st.data[st.Len()-15], st.data[st.Len()-1] = st.data[st.Len()-1], st.data[st.Len()-15]
}
func (st *Stack) swap15() {
	st.data[st.Len()-16], st.data[st.Len()-1] = st.data[st.Len()-1], st.data[st.Len()-16]
}
func (st *Stack) swap16() {
	st.data[st.Len()-17], st.data[st.Len()-1] = st.data[st.Len()-1], st.data[st.Len()-17]
}

func (st *Stack) dup(n int) {
	st.Push(&st.data[st.Len()-n])
}

func (st *Stack) Peek() *uint256.Int {
	return &st.data[st.Len()-1]
}

// Back returns the n'th item in stack
func (st *Stack) Back(n int) *uint256.Int {
	return &st.data[st.Len()-n-1]
}
