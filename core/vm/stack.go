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
	"reflect"
	"sync"
	"unsafe"

	"github.com/ethereum/go-ethereum/params"
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

func (st *Stack) len() int {
	return len(st.data)
}

func (st *Stack) growUnsafeByN(n uint) {
	sh := (*reflect.SliceHeader)(unsafe.Pointer(&st.data))
	sh.Len += int(n)
}

func (st *Stack) push(d *uint256.Int) error {
	if uint64(st.len()) >= params.StackLimit {
		return ErrStackOverflow{st.len(), int(params.StackLimit)}
	}
	st.data = append(st.data, *d)
	return nil
}

func (st *Stack) pop(peekLastN uint) (*uint256.Int, error) {
	const requiredDepth = 1
	depth := st.len()
	if depth < requiredDepth {
		return nil, ErrStackUnderflow{st.len(), requiredDepth}
	}
	stack := st.data
	st.data = st.data[:depth-requiredDepth]
	st.growUnsafeByN(peekLastN)
	return &stack[depth-1], nil
}

func (st *Stack) pop2(peekLastN uint) (*uint256.Int, *uint256.Int, error) {
	const requiredDepth = 2
	depth := st.len()
	if depth < requiredDepth {
		return nil, nil, ErrStackUnderflow{st.len(), requiredDepth}
	}
	stack := st.data
	st.data = st.data[:depth-requiredDepth]
	st.growUnsafeByN(peekLastN)
	return &stack[depth-1], &stack[depth-2], nil
}

func (st *Stack) pop3(peekLastN uint) (*uint256.Int, *uint256.Int, *uint256.Int, error) {
	const requiredDepth = 3
	depth := st.len()
	if depth < requiredDepth {
		return nil, nil, nil, ErrStackUnderflow{st.len(), requiredDepth}
	}
	stack := st.data
	st.data = st.data[:depth-requiredDepth]
	st.growUnsafeByN(peekLastN)
	return &stack[depth-1], &stack[depth-2], &stack[depth-3], nil
}

func (st *Stack) pop4(peekLastN uint) (*uint256.Int, *uint256.Int, *uint256.Int, *uint256.Int, error) {
	const requiredDepth = 4
	depth := st.len()
	if depth < requiredDepth {
		return nil, nil, nil, nil, ErrStackUnderflow{st.len(), requiredDepth}
	}
	stack := st.data
	st.data = st.data[:depth-requiredDepth]
	st.growUnsafeByN(peekLastN)
	return &stack[depth-1], &stack[depth-2], &stack[depth-3], &stack[depth-4], nil
}

func (st *Stack) pop6(peekLastN uint) (*uint256.Int, *uint256.Int, *uint256.Int, *uint256.Int, *uint256.Int, *uint256.Int, error) {
	const requiredDepth = 6
	depth := st.len()
	if depth < requiredDepth {
		return nil, nil, nil, nil, nil, nil, ErrStackUnderflow{st.len(), requiredDepth}
	}
	stack := st.data
	st.data = st.data[:depth-requiredDepth]
	st.growUnsafeByN(peekLastN)
	return &stack[depth-1], &stack[depth-2], &stack[depth-3], &stack[depth-4], &stack[depth-5], &stack[depth-6], nil
}

func (st *Stack) pop7(peekLastN uint) (*uint256.Int, *uint256.Int, *uint256.Int, *uint256.Int, *uint256.Int, *uint256.Int, *uint256.Int, error) {
	const requiredDepth = 7
	depth := st.len()
	if depth < requiredDepth {
		return nil, nil, nil, nil, nil, nil, nil, ErrStackUnderflow{st.len(), requiredDepth}
	}
	stack := st.data
	st.data = st.data[:depth-requiredDepth]
	st.growUnsafeByN(peekLastN)
	return &stack[depth-1], &stack[depth-2], &stack[depth-3], &stack[depth-4], &stack[depth-5], &stack[depth-6], &stack[depth-7], nil
}

func (st *Stack) swap1() error {
	const requiredDepth = 2
	depth := st.len()
	if depth < requiredDepth {
		return ErrStackUnderflow{st.len(), requiredDepth}
	}
	topOfStack := st.data[depth-requiredDepth:]
	topOfStack[0], topOfStack[len(topOfStack)-1] = topOfStack[len(topOfStack)-1], topOfStack[0]
	return nil
}

func (st *Stack) swap2() error {
	const requiredDepth = 3
	depth := st.len()
	if depth < requiredDepth {
		return ErrStackUnderflow{st.len(), requiredDepth}
	}
	topOfStack := st.data[depth-requiredDepth:]
	topOfStack[0], topOfStack[len(topOfStack)-1] = topOfStack[len(topOfStack)-1], topOfStack[0]
	return nil
}

func (st *Stack) swap3() error {
	const requiredDepth = 4
	depth := st.len()
	if depth < requiredDepth {
		return ErrStackUnderflow{st.len(), requiredDepth}
	}
	topOfStack := st.data[depth-requiredDepth:]
	topOfStack[0], topOfStack[len(topOfStack)-1] = topOfStack[len(topOfStack)-1], topOfStack[0]
	return nil
}

func (st *Stack) swap4() error {
	const requiredDepth = 5
	depth := st.len()
	if depth < requiredDepth {
		return ErrStackUnderflow{st.len(), requiredDepth}
	}
	topOfStack := st.data[depth-requiredDepth:]
	topOfStack[0], topOfStack[len(topOfStack)-1] = topOfStack[len(topOfStack)-1], topOfStack[0]
	return nil
}

func (st *Stack) swap5() error {
	const requiredDepth = 6
	depth := st.len()
	if depth < requiredDepth {
		return ErrStackUnderflow{st.len(), requiredDepth}
	}
	topOfStack := st.data[depth-requiredDepth:]
	topOfStack[0], topOfStack[len(topOfStack)-1] = topOfStack[len(topOfStack)-1], topOfStack[0]
	return nil
}

func (st *Stack) swap6() error {
	const requiredDepth = 7
	depth := st.len()
	if depth < requiredDepth {
		return ErrStackUnderflow{st.len(), requiredDepth}
	}
	topOfStack := st.data[depth-requiredDepth:]
	topOfStack[0], topOfStack[len(topOfStack)-1] = topOfStack[len(topOfStack)-1], topOfStack[0]
	return nil
}

func (st *Stack) swap7() error {
	const requiredDepth = 8
	depth := st.len()
	if depth < requiredDepth {
		return ErrStackUnderflow{st.len(), requiredDepth}
	}
	topOfStack := st.data[depth-requiredDepth:]
	topOfStack[0], topOfStack[len(topOfStack)-1] = topOfStack[len(topOfStack)-1], topOfStack[0]
	return nil
}

func (st *Stack) swap8() error {
	const requiredDepth = 9
	depth := st.len()
	if depth < requiredDepth {
		return ErrStackUnderflow{st.len(), requiredDepth}
	}
	topOfStack := st.data[depth-requiredDepth:]
	topOfStack[0], topOfStack[len(topOfStack)-1] = topOfStack[len(topOfStack)-1], topOfStack[0]
	return nil
}

func (st *Stack) swap9() error {
	const requiredDepth = 10
	depth := st.len()
	if depth < requiredDepth {
		return ErrStackUnderflow{st.len(), requiredDepth}
	}
	topOfStack := st.data[depth-requiredDepth:]
	topOfStack[0], topOfStack[len(topOfStack)-1] = topOfStack[len(topOfStack)-1], topOfStack[0]
	return nil
}

func (st *Stack) swap10() error {
	const requiredDepth = 11
	depth := st.len()
	if depth < requiredDepth {
		return ErrStackUnderflow{st.len(), requiredDepth}
	}
	topOfStack := st.data[depth-requiredDepth:]
	topOfStack[0], topOfStack[len(topOfStack)-1] = topOfStack[len(topOfStack)-1], topOfStack[0]
	return nil
}

func (st *Stack) swap11() error {
	const requiredDepth = 12
	depth := st.len()
	if depth < requiredDepth {
		return ErrStackUnderflow{st.len(), requiredDepth}
	}
	topOfStack := st.data[depth-requiredDepth:]
	topOfStack[0], topOfStack[len(topOfStack)-1] = topOfStack[len(topOfStack)-1], topOfStack[0]
	return nil
}

func (st *Stack) swap12() error {
	const requiredDepth = 13
	depth := st.len()
	if depth < requiredDepth {
		return ErrStackUnderflow{st.len(), requiredDepth}
	}
	topOfStack := st.data[depth-requiredDepth:]
	topOfStack[0], topOfStack[len(topOfStack)-1] = topOfStack[len(topOfStack)-1], topOfStack[0]
	return nil
}

func (st *Stack) swap13() error {
	const requiredDepth = 14
	depth := st.len()
	if depth < requiredDepth {
		return ErrStackUnderflow{st.len(), requiredDepth}
	}
	topOfStack := st.data[depth-requiredDepth:]
	topOfStack[0], topOfStack[len(topOfStack)-1] = topOfStack[len(topOfStack)-1], topOfStack[0]
	return nil
}

func (st *Stack) swap14() error {
	const requiredDepth = 15
	depth := st.len()
	if depth < requiredDepth {
		return ErrStackUnderflow{st.len(), requiredDepth}
	}
	topOfStack := st.data[depth-requiredDepth:]
	topOfStack[0], topOfStack[len(topOfStack)-1] = topOfStack[len(topOfStack)-1], topOfStack[0]
	return nil
}

func (st *Stack) swap15() error {
	const requiredDepth = 16
	depth := st.len()
	if depth < requiredDepth {
		return ErrStackUnderflow{st.len(), requiredDepth}
	}
	topOfStack := st.data[depth-requiredDepth:]
	topOfStack[0], topOfStack[len(topOfStack)-1] = topOfStack[len(topOfStack)-1], topOfStack[0]
	return nil
}

func (st *Stack) swap16() error {
	const requiredDepth = 17
	depth := st.len()
	if depth < requiredDepth {
		return ErrStackUnderflow{st.len(), requiredDepth}
	}
	topOfStack := st.data[depth-requiredDepth:]
	topOfStack[0], topOfStack[len(topOfStack)-1] = topOfStack[len(topOfStack)-1], topOfStack[0]
	return nil
}

func (st *Stack) dup1() error {
	const requiredDepth = 1
	depth := st.len()
	if depth < requiredDepth {
		return ErrStackUnderflow{st.len(), requiredDepth}
	}
	return st.push(&st.data[depth-requiredDepth])
}

func (st *Stack) dup2() error {
	const requiredDepth = 2
	depth := st.len()
	if depth < requiredDepth {
		return ErrStackUnderflow{st.len(), requiredDepth}
	}
	return st.push(&st.data[depth-requiredDepth])
}

func (st *Stack) dup3() error {
	const requiredDepth = 3
	depth := st.len()
	if depth < requiredDepth {
		return ErrStackUnderflow{st.len(), requiredDepth}
	}
	return st.push(&st.data[depth-requiredDepth])
}

func (st *Stack) dup4() error {
	const requiredDepth = 4
	depth := st.len()
	if depth < requiredDepth {
		return ErrStackUnderflow{st.len(), requiredDepth}
	}
	return st.push(&st.data[depth-requiredDepth])
}

func (st *Stack) dup5() error {
	const requiredDepth = 5
	depth := st.len()
	if depth < requiredDepth {
		return ErrStackUnderflow{st.len(), requiredDepth}
	}
	return st.push(&st.data[depth-requiredDepth])
}

func (st *Stack) dup6() error {
	const requiredDepth = 6
	depth := st.len()
	if depth < requiredDepth {
		return ErrStackUnderflow{st.len(), requiredDepth}
	}
	return st.push(&st.data[depth-requiredDepth])
}

func (st *Stack) dup7() error {
	const requiredDepth = 7
	depth := st.len()
	if depth < requiredDepth {
		return ErrStackUnderflow{st.len(), requiredDepth}
	}
	return st.push(&st.data[depth-requiredDepth])
}

func (st *Stack) dup8() error {
	const requiredDepth = 8
	depth := st.len()
	if depth < requiredDepth {
		return ErrStackUnderflow{st.len(), requiredDepth}
	}
	return st.push(&st.data[depth-requiredDepth])
}

func (st *Stack) dup9() error {
	const requiredDepth = 9
	depth := st.len()
	if depth < requiredDepth {
		return ErrStackUnderflow{st.len(), requiredDepth}
	}
	return st.push(&st.data[depth-requiredDepth])
}

func (st *Stack) dup10() error {
	const requiredDepth = 10
	depth := st.len()
	if depth < requiredDepth {
		return ErrStackUnderflow{st.len(), requiredDepth}
	}
	return st.push(&st.data[depth-requiredDepth])
}

func (st *Stack) dup11() error {
	const requiredDepth = 11
	depth := st.len()
	if depth < requiredDepth {
		return ErrStackUnderflow{st.len(), requiredDepth}
	}
	return st.push(&st.data[depth-requiredDepth])
}

func (st *Stack) dup12() error {
	const requiredDepth = 12
	depth := st.len()
	if depth < requiredDepth {
		return ErrStackUnderflow{st.len(), requiredDepth}
	}
	return st.push(&st.data[depth-requiredDepth])
}

func (st *Stack) dup13() error {
	const requiredDepth = 13
	depth := st.len()
	if depth < requiredDepth {
		return ErrStackUnderflow{st.len(), requiredDepth}
	}
	return st.push(&st.data[depth-requiredDepth])
}

func (st *Stack) dup14() error {
	const requiredDepth = 14
	depth := st.len()
	if depth < requiredDepth {
		return ErrStackUnderflow{st.len(), requiredDepth}
	}
	return st.push(&st.data[depth-requiredDepth])
}

func (st *Stack) dup15() error {
	const requiredDepth = 15
	depth := st.len()
	if depth < requiredDepth {
		return ErrStackUnderflow{st.len(), requiredDepth}
	}
	return st.push(&st.data[depth-requiredDepth])
}

func (st *Stack) dup16() error {
	const requiredDepth = 16
	depth := st.len()
	if depth < requiredDepth {
		return ErrStackUnderflow{st.len(), requiredDepth}
	}
	return st.push(&st.data[depth-requiredDepth])
}
