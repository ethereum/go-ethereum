// Copyright 2024 the libevm authors.
//
// The libevm additions to go-ethereum are free software: you can redistribute
// them and/or modify them under the terms of the GNU Lesser General Public License
// as published by the Free Software Foundation, either version 3 of the License,
// or (at your option) any later version.
//
// The libevm additions are distributed in the hope that they will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the GNU Lesser
// General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see
// <http://www.gnu.org/licenses/>.
package ethtest

import (
	"math/big"

	"golang.org/x/exp/rand"

	"github.com/ethereum/go-ethereum/common"
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
	r.Read(a[:]) //nolint:gosec,errcheck // Guaranteed nil error
	return a
}

// AddressPtr returns a pointer to a pseudorandom address.
func (r *PseudoRand) AddressPtr() *common.Address {
	a := r.Address()
	return &a
}

// Hash returns a pseudorandom hash.
func (r *PseudoRand) Hash() (h common.Hash) {
	r.Read(h[:]) //nolint:gosec,errcheck // Guaranteed nil error
	return h
}

// Bytes returns `n` pseudorandom bytes.
func (r *PseudoRand) Bytes(n uint) []byte {
	b := make([]byte, n)
	r.Read(b) //nolint:gosec,errcheck // Guaranteed nil error
	return b
}

// Big returns [rand.Rand.Uint64] as a [big.Int].
func (r *PseudoRand) BigUint64() *big.Int {
	return new(big.Int).SetUint64(r.Uint64())
}
