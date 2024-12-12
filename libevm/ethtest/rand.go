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

	"github.com/holiman/uint256"
	"golang.org/x/exp/rand"

	"github.com/ava-labs/libevm/common"
	"github.com/ava-labs/libevm/core/types"
)

// PseudoRand extends [rand.Rand] (*not* crypto/rand).
type PseudoRand struct {
	*rand.Rand
}

// NewPseudoRand returns a new PseudoRand with the given seed.
func NewPseudoRand(seed uint64) *PseudoRand {
	return &PseudoRand{rand.New(rand.NewSource(seed))}
}

// Read is equivalent to [rand.Rand.Read] except that it doesn't return an error
// because it is guaranteed to be nil.
func (r *PseudoRand) Read(p []byte) int {
	n, _ := r.Rand.Read(p) // Guaranteed nil error
	return n
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

// HashPtr returns a pointer to a pseudorandom hash.
func (r *PseudoRand) HashPtr() *common.Hash {
	h := r.Hash()
	return &h
}

// Bytes returns `n` pseudorandom bytes.
func (r *PseudoRand) Bytes(n uint) []byte {
	b := make([]byte, n)
	r.Read(b)
	return b
}

// Big returns [rand.Rand.Uint64] as a [big.Int].
func (r *PseudoRand) BigUint64() *big.Int {
	return new(big.Int).SetUint64(r.Uint64())
}

// Uint64Ptr returns a pointer to a pseudorandom uint64.
func (r *PseudoRand) Uint64Ptr() *uint64 {
	u := r.Uint64()
	return &u
}

// Uint256 returns a random 256-bit unsigned int.
func (r *PseudoRand) Uint256() *uint256.Int {
	return new(uint256.Int).SetBytes(r.Bytes(32))
}

// Bloom returns a pseudorandom Bloom.
func (r *PseudoRand) Bloom() (b types.Bloom) {
	r.Read(b[:])
	return b
}

// BlockNonce returns a pseudorandom BlockNonce.
func (r *PseudoRand) BlockNonce() (n types.BlockNonce) {
	r.Read(n[:])
	return n
}
