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
// initialised objects.
type Stack struct {
	data []uint256.Int
}

func newstack() *Stack {
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

func (st *Stack) push(d *uint256.Int) {
	// NOTE push limit (1024) is checked in baseCheck
	st.data = append(st.data, *d)
}

func (st *Stack) pop() (ret uint256.Int) {
	ret = st.data[len(st.data)-1]
	st.data = st.data[:len(st.data)-1]
	return
}

func (st *Stack) len() int {
	return len(st.data)
}

func (st *Stack) swap(n int) {
	st.data[st.len()-n], st.data[st.len()-1] = st.data[st.len()-1], st.data[st.len()-n]
}

func (s *Stack) swap1() { s.data[s.len()-2], s.data[s.len()-1] = s.data[s.len()-1], s.data[s.len()-2] }
func (s *Stack) swap2() { s.data[s.len()-3], s.data[s.len()-1] = s.data[s.len()-1], s.data[s.len()-3] }
func (s *Stack) swap3() { s.data[s.len()-4], s.data[s.len()-1] = s.data[s.len()-1], s.data[s.len()-4] }
func (s *Stack) swap4() { s.data[s.len()-5], s.data[s.len()-1] = s.data[s.len()-1], s.data[s.len()-5] }
func (s *Stack) swap5() { s.data[s.len()-6], s.data[s.len()-1] = s.data[s.len()-1], s.data[s.len()-6] }
func (s *Stack) swap6() { s.data[s.len()-7], s.data[s.len()-1] = s.data[s.len()-1], s.data[s.len()-7] }
func (s *Stack) swap7() { s.data[s.len()-8], s.data[s.len()-1] = s.data[s.len()-1], s.data[s.len()-8] }
func (s *Stack) swap8() { s.data[s.len()-9], s.data[s.len()-1] = s.data[s.len()-1], s.data[s.len()-9] }
func (s *Stack) swap9() {
	s.data[s.len()-10], s.data[s.len()-1] = s.data[s.len()-1], s.data[s.len()-10]
}
func (s *Stack) swap10() {
	s.data[s.len()-11], s.data[s.len()-1] = s.data[s.len()-1], s.data[s.len()-11]
}
func (s *Stack) swap11() {
	s.data[s.len()-12], s.data[s.len()-1] = s.data[s.len()-1], s.data[s.len()-12]
}
func (s *Stack) swap12() {
	s.data[s.len()-13], s.data[s.len()-1] = s.data[s.len()-1], s.data[s.len()-13]
}
func (s *Stack) swap13() {
	s.data[s.len()-14], s.data[s.len()-1] = s.data[s.len()-1], s.data[s.len()-14]
}
func (s *Stack) swap14() {
	s.data[s.len()-15], s.data[s.len()-1] = s.data[s.len()-1], s.data[s.len()-15]
}
func (s *Stack) swap15() {
	s.data[s.len()-16], s.data[s.len()-1] = s.data[s.len()-1], s.data[s.len()-16]
}
func (s *Stack) swap16() {
	s.data[s.len()-17], s.data[s.len()-1] = s.data[s.len()-1], s.data[s.len()-17]
}

func (st *Stack) dup(n int) {
	st.push(&st.data[st.len()-n])
}

func (st *Stack) peek() *uint256.Int {
	return &st.data[st.len()-1]
}

// Back returns the n'th item in stack
func (st *Stack) Back(n int) *uint256.Int {
	return &st.data[st.len()-n-1]
}
