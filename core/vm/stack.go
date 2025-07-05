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
	"slices"
	"sync"

	"github.com/holiman/uint256"
)

// stackArena is an arena which actual evm stacks use for data storage
type stackArena struct {
	data []uint256.Int
	top  int // first free slot
}

func newArena() *stackArena {
	return stackPool.New().(*stackArena)
}

var stackPool = sync.Pool{
	New: func() any {
		return &stackArena{
			data: make([]uint256.Int, 1025),
		}
	},
}

func returnStack(arena *stackArena) {
	stackPool.Put(arena)
}

// stack returns an instance of a stack which uses the underlying arena. The instance
// must be released by invoking the (*Stack).release() method
func (sa *stackArena) stack() *Stack {
	// make sure every substack has at least 1024 elements
	if len(sa.data) <= sa.top+1024 {
		// we need to grow the arena
		sa.data = slices.Grow(sa.data, 1024)
		sa.data = sa.data[:cap(sa.data)]
	}
	return &Stack{
		bottom: sa.top,
		size:   0,
		inner:  sa,
	}
}

// newStackForTesting is meant to be used solely for testing. It creates a stack
// backed by a newly allocated arena.
func newStackForTesting() *Stack {
	arena := &stackArena{
		data: make([]uint256.Int, 1025),
	}
	return arena.stack()
}

// Stack is an object for basic stack operations. Items popped to the stack are
// expected to be changed and modified. stack does not take care of adding newly
// initialized objects.
type Stack struct {
	bottom int // bottom is the index of the first element of this stack
	size   int // size is the number of elements in this stack
	inner  *stackArena
}

// release un-claims the area of the arena which was claimed by the stack.
func (s *Stack) release() {
	// When the stack is returned, need to notify the arena that the new 'top' is
	// the returned stack's bottom.
	s.inner.top = s.bottom
}

// Data returns the underlying uint256.Int array.
func (s *Stack) Data() []uint256.Int {
	return s.inner.data[s.bottom : s.bottom+s.size]
}

func (s *Stack) push(d *uint256.Int) {
	s.inner.data[s.inner.top] = *d
	s.inner.top++
	s.size++
}

// get returns a pointer to a newly created element
// on top of the stack
func (s *Stack) get() *uint256.Int {
	elem := &s.inner.data[s.inner.top]
	s.inner.top++
	s.size++
	return elem
}

func (s *Stack) pop() uint256.Int {
	s.inner.top--
	s.size--
	return s.inner.data[s.inner.top]
}

func (s *Stack) len() int {
	return s.size
}

func (s *Stack) swap1() {
	s.inner.data[s.bottom+s.size-2], s.inner.data[s.bottom+s.size-1] = s.inner.data[s.bottom+s.size-1], s.inner.data[s.bottom+s.size-2]
}
func (s *Stack) swap2() {
	s.inner.data[s.bottom+s.size-3], s.inner.data[s.bottom+s.size-1] = s.inner.data[s.bottom+s.size-1], s.inner.data[s.bottom+s.size-3]
}
func (s *Stack) swap3() {
	s.inner.data[s.bottom+s.size-4], s.inner.data[s.bottom+s.size-1] = s.inner.data[s.bottom+s.size-1], s.inner.data[s.bottom+s.size-4]
}
func (s *Stack) swap4() {
	s.inner.data[s.bottom+s.size-5], s.inner.data[s.bottom+s.size-1] = s.inner.data[s.bottom+s.size-1], s.inner.data[s.bottom+s.size-5]
}
func (s *Stack) swap5() {
	s.inner.data[s.bottom+s.size-6], s.inner.data[s.bottom+s.size-1] = s.inner.data[s.bottom+s.size-1], s.inner.data[s.bottom+s.size-6]
}
func (s *Stack) swap6() {
	s.inner.data[s.bottom+s.size-7], s.inner.data[s.bottom+s.size-1] = s.inner.data[s.bottom+s.size-1], s.inner.data[s.bottom+s.size-7]
}
func (s *Stack) swap7() {
	s.inner.data[s.bottom+s.size-8], s.inner.data[s.bottom+s.size-1] = s.inner.data[s.bottom+s.size-1], s.inner.data[s.bottom+s.size-8]
}
func (s *Stack) swap8() {
	s.inner.data[s.bottom+s.size-9], s.inner.data[s.bottom+s.size-1] = s.inner.data[s.bottom+s.size-1], s.inner.data[s.bottom+s.size-9]
}
func (s *Stack) swap9() {
	s.inner.data[s.bottom+s.size-10], s.inner.data[s.bottom+s.size-1] = s.inner.data[s.bottom+s.size-1], s.inner.data[s.bottom+s.size-10]
}
func (s *Stack) swap10() {
	s.inner.data[s.bottom+s.size-11], s.inner.data[s.bottom+s.size-1] = s.inner.data[s.bottom+s.size-1], s.inner.data[s.bottom+s.size-11]
}
func (s *Stack) swap11() {
	s.inner.data[s.bottom+s.size-12], s.inner.data[s.bottom+s.size-1] = s.inner.data[s.bottom+s.size-1], s.inner.data[s.bottom+s.size-12]
}
func (s *Stack) swap12() {
	s.inner.data[s.bottom+s.size-13], s.inner.data[s.bottom+s.size-1] = s.inner.data[s.bottom+s.size-1], s.inner.data[s.bottom+s.size-13]
}
func (s *Stack) swap13() {
	s.inner.data[s.bottom+s.size-14], s.inner.data[s.bottom+s.size-1] = s.inner.data[s.bottom+s.size-1], s.inner.data[s.bottom+s.size-14]
}
func (s *Stack) swap14() {
	s.inner.data[s.bottom+s.size-15], s.inner.data[s.bottom+s.size-1] = s.inner.data[s.bottom+s.size-1], s.inner.data[s.bottom+s.size-15]
}
func (s *Stack) swap15() {
	s.inner.data[s.bottom+s.size-16], s.inner.data[s.bottom+s.size-1] = s.inner.data[s.bottom+s.size-1], s.inner.data[s.bottom+s.size-16]
}
func (s *Stack) swap16() {
	s.inner.data[s.bottom+s.size-17], s.inner.data[s.bottom+s.size-1] = s.inner.data[s.bottom+s.size-1], s.inner.data[s.bottom+s.size-17]
}

func (s *Stack) dup(n int) {
	s.inner.data[s.bottom+s.size] = s.inner.data[s.bottom+s.size-n]
	s.size++
	s.inner.top++
}

func (s *Stack) peek() *uint256.Int {
	return &s.inner.data[s.bottom+s.size-1]
}

// Back returns the n'th item in stack
func (s *Stack) Back(n int) *uint256.Int {
	return &s.inner.data[s.bottom+s.size-n-1]
}
