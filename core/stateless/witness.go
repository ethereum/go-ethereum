// Copyright 2024 The go-ethereum Authors
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

package stateless

import (
	"errors"
	"maps"
	"slices"
	"sync"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

// HeaderReader is an interface to pull in headers in place of block hashes for
// the witness.
type HeaderReader interface {
	// GetHeader retrieves a block header from the database by hash and number,
	GetHeader(hash common.Hash, number uint64) *types.Header
}

// Witness encompasses the state required to apply a set of transactions and
// derive a post state/receipt root.
type Witness struct {
	context *types.Header // Header to which this witness belongs to, with rootHash and receiptHash zeroed out

	Headers []*types.Header     // Past headers in reverse order (0=parent, 1=parent's-parent, etc). First *must* be set.
	Codes   map[string]struct{} // Set of bytecodes ran or accessed
	State   map[string]struct{} // Set of MPT state trie nodes (account and storage together)

	chain HeaderReader // Chain reader to convert block hash ops to header proofs
	lock  sync.Mutex   // Lock to allow concurrent state insertions
}

// NewWitness creates an empty witness ready for population.
func NewWitness(context *types.Header, chain HeaderReader) (*Witness, error) {
	// When building witnesses, retrieve the parent header, which will *always*
	// be included to act as a trustless pre-root hash container
	var headers []*types.Header
	if chain != nil {
		parent := chain.GetHeader(context.ParentHash, context.Number.Uint64()-1)
		if parent == nil {
			return nil, errors.New("failed to retrieve parent header")
		}
		headers = append(headers, parent)
	}
	// Create the wtness with a reconstructed gutted out block
	return &Witness{
		context: context,
		Headers: headers,
		Codes:   make(map[string]struct{}),
		State:   make(map[string]struct{}),
		chain:   chain,
	}, nil
}

// AddBlockHash adds a "blockhash" to the witness with the designated offset from
// chain head. Under the hood, this method actually pulls in enough headers from
// the chain to cover the block being added.
func (w *Witness) AddBlockHash(number uint64) {
	// Keep pulling in headers until this hash is populated
	for int(w.context.Number.Uint64()-number) > len(w.Headers) {
		tail := w.Headers[len(w.Headers)-1]
		w.Headers = append(w.Headers, w.chain.GetHeader(tail.ParentHash, tail.Number.Uint64()-1))
	}
}

// AddCode adds a bytecode blob to the witness.
func (w *Witness) AddCode(code []byte) {
	if len(code) == 0 {
		return
	}
	w.Codes[string(code)] = struct{}{}
}

// AddState inserts a batch of MPT trie nodes into the witness.
func (w *Witness) AddState(nodes map[string]struct{}) {
	if len(nodes) == 0 {
		return
	}
	w.lock.Lock()
	defer w.lock.Unlock()

	maps.Copy(w.State, nodes)
}

// Copy deep-copies the witness object.  Witness.Block isn't deep-copied as it
// is never mutated by Witness
func (w *Witness) Copy() *Witness {
	cpy := &Witness{
		Headers: slices.Clone(w.Headers),
		Codes:   maps.Clone(w.Codes),
		State:   maps.Clone(w.State),
		chain:   w.chain,
	}
	if w.context != nil {
		cpy.context = types.CopyHeader(w.context)
	}
	return cpy
}

// Root returns the pre-state root from the first header.
//
// Note, this method will panic in case of a bad witness (but RLP decoding will
// sanitize it and fail before that).
func (w *Witness) Root() common.Hash {
	return w.Headers[0].Root
}
