package vm

import (
	"fmt"
	"math/big"
)

func newStack() *Stack {
	return &Stack{}
}

type Stack struct {
	data []*big.Int
	ptr  int
}

func (st *Stack) Data() []*big.Int {
	return st.data
}

func (st *Stack) push(d *big.Int) {
	// NOTE push limit (1024) is checked in baseCheck
	stackItem := new(big.Int).Set(d)
	if len(st.data) > st.ptr {
		st.data[st.ptr] = stackItem
	} else {
		st.data = append(st.data, stackItem)
	}
	st.ptr++
}

func (st *Stack) pop() (ret *big.Int) {
	st.ptr--
	ret = st.data[st.ptr]
	return
}

func (st *Stack) len() int {
	return st.ptr
}

func (st *Stack) swap(n int) {
	st.data[st.len()-n], st.data[st.len()-1] = st.data[st.len()-1], st.data[st.len()-n]
}

func (st *Stack) dup(n int) {
	st.push(st.data[st.len()-n])
}

func (st *Stack) peek() *big.Int {
	return st.data[st.len()-1]
}

func (st *Stack) require(n int) error {
	if st.len() < n {
		return fmt.Errorf("stack underflow (%d <=> %d)", len(st.data), n)
	}
	return nil
}

func (st *Stack) Print() {
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
