package ethtest

import (
	"github.com/ethereum/go-ethereum/common"
	"golang.org/x/exp/rand"
)

// PseudoRand extends [rand.Rand] (*not* crypto/rand).
type PseudoRand struct {
	*rand.Rand
}

// NewPseudoRand returns a new PseudoRand with the given seed.
func NewPseudoRand(seed uint64) *PseudoRand {
	return &PseudoRand{rand.New(rand.NewSource(seed))}
}

// Address returns a pseudorandom address.
func (r *PseudoRand) Address() (a common.Address) {
	r.Read(a[:])
	return a
}

// AddressPtr returns a pointer to a pseudorandom address.
func (r *PseudoRand) AddressPtr() *common.Address {
	a := r.Address()
	return &a
}

// Hash returns a pseudorandom hash.
func (r *PseudoRand) Hash() (h common.Hash) {
	r.Read(h[:])
	return h
}

// Bytes returns `n` pseudorandom bytes.
func (r *PseudoRand) Bytes(n uint) []byte {
	b := make([]byte, n)
	r.Read(b)
	return b
}
