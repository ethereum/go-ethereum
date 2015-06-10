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
