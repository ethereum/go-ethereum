package ethchain

import (
	"fmt"
	"math"
	"math/big"
)

type OpType int

const (
	tNorm = iota
	tData
	tExtro
	tCrypto
)

type TxCallback func(opType OpType) bool

// Simple push/pop stack mechanism
type Stack struct {
	data []*big.Int
}

func NewStack() *Stack {
	return &Stack{}
}

func (st *Stack) Data() []*big.Int {
	return st.data
}

func (st *Stack) Len() int {
	return len(st.data)
}

func (st *Stack) Pop() *big.Int {
	str := st.data[len(st.data)-1]

	copy(st.data[:len(st.data)-1], st.data[:len(st.data)-1])
	st.data = st.data[:len(st.data)-1]

	return str
}

func (st *Stack) Popn() (*big.Int, *big.Int) {
	ints := st.data[len(st.data)-2:]

	copy(st.data[:len(st.data)-2], st.data[:len(st.data)-2])
	st.data = st.data[:len(st.data)-2]

	return ints[0], ints[1]
}

func (st *Stack) Peek() *big.Int {
	str := st.data[len(st.data)-1]

	return str
}

func (st *Stack) Peekn() (*big.Int, *big.Int) {
	ints := st.data[:2]

	return ints[0], ints[1]
}

func (st *Stack) Push(d *big.Int) {
	st.data = append(st.data, new(big.Int).Set(d))
}

func (st *Stack) Get(amount *big.Int) []*big.Int {
	// offset + size <= len(data)
	length := big.NewInt(int64(len(st.data)))
	if amount.Cmp(length) <= 0 {
		start := new(big.Int).Sub(length, amount)
		return st.data[start.Int64():length.Int64()]
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

type Memory struct {
	store []byte
}

func (m *Memory) Set(offset, size int64, value []byte) {
	totSize := offset + size
	lenSize := int64(len(m.store) - 1)
	if totSize > lenSize {
		// Calculate the diff between the sizes
		diff := totSize - lenSize
		if diff > 0 {
			// Create a new empty slice and append it
			newSlice := make([]byte, diff-1)
			// Resize slice
			m.store = append(m.store, newSlice...)
		}
	}
	copy(m.store[offset:offset+size], value)
}

func (m *Memory) Resize(size uint64) {
	if uint64(m.Len()) < size {
		m.store = append(m.store, make([]byte, size-uint64(m.Len()))...)
	}
}

func (m *Memory) Get(offset, size int64) []byte {
	if len(m.store) > int(offset) {
		end := int(math.Min(float64(len(m.store)), float64(offset+size)))

		return m.store[offset:end]
	}

	return nil
}

func (m *Memory) Len() int {
	return len(m.store)
}

func (m *Memory) Data() []byte {
	return m.store
}

func (m *Memory) Print() {
	fmt.Printf("### mem %d bytes ###\n", len(m.store))
	if len(m.store) > 0 {
		addr := 0
		for i := 0; i+32 <= len(m.store); i += 32 {
			fmt.Printf("%03d: % x\n", addr, m.store[i:i+32])
			addr++
		}
	} else {
		fmt.Println("-- empty --")
	}
	fmt.Println("####################")
}
