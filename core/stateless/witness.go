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
	"bytes"
	"errors"
	"fmt"
	"maps"
	"slices"
	"sync"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rlp"
)

// HeaderReader is an interface to pull in headers in place of block hashes for
// the witness.
type HeaderReader interface {
	// GetHeader retrieves a block header from the database by hash and number,
	GetHeader(hash common.Hash, number uint64) *types.Header
}

// Witness encompasses a block, state and any other chain data required to apply
// a set of transactions and derive a post state/receipt root.
type Witness struct {
	Block   *types.Block        // Current block with rootHash and receiptHash zeroed out
	Headers []*types.Header     // Past headers in reverse order (0=parent, 1=parent's-parent, etc). First *must* be set.
	Codes   map[string]struct{} // Set of bytecodes ran or accessed
	State   map[string]struct{} // Set of MPT state trie nodes (account and storage together)

	chain HeaderReader // Chain reader to convert block hash ops to header proofs
	lock  sync.Mutex   // Lock to allow concurrent state insertions
}

// NewWitness creates an empty witness ready for population.
func NewWitness(chain HeaderReader, block *types.Block) (*Witness, error) {
	// Zero out the result fields to avoid accidentally sending them to the verifier
	header := block.Header()
	header.Root = common.Hash{}
	header.ReceiptHash = common.Hash{}

	// Retrieve the parent header, which will *always* be included to act as a
	// trustless pre-root hash container
	parent := chain.GetHeader(block.ParentHash(), block.NumberU64()-1)
	if parent == nil {
		return nil, errors.New("failed to retrieve parent header")
	}
	// Create the wtness with a reconstructed gutted out block
	return &Witness{
		Block:   types.NewBlockWithHeader(header).WithBody(*block.Body()),
		Codes:   make(map[string]struct{}),
		State:   make(map[string]struct{}),
		Headers: []*types.Header{parent},
		chain:   chain,
	}, nil
}

// AddBlockHash adds a "blockhash" to the witness with the designated offset from
// chain head. Under the hood, this method actually pulls in enough headers from
// the chain to cover the block being added.
func (w *Witness) AddBlockHash(number uint64) {
	// Keep pulling in headers until this hash is populated
	for int(w.Block.NumberU64()-number) > len(w.Headers) {
		tail := w.Block.Header()
		if len(w.Headers) > 0 {
			tail = w.Headers[len(w.Headers)-1]
		}
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
	return &Witness{
		Block:   w.Block,
		Headers: slices.Clone(w.Headers),
		Codes:   maps.Clone(w.Codes),
		State:   maps.Clone(w.State),
	}
}

// String prints a human-readable summary containing the total size of the
// witness and the sizes of the underlying components
func (w *Witness) String() string {
	blob, _ := rlp.EncodeToBytes(w)
	bytesTotal := len(blob)

	blob, _ = rlp.EncodeToBytes(w.Block)
	bytesBlock := len(blob)

	bytesHeaders := 0
	for _, header := range w.Headers {
		blob, _ = rlp.EncodeToBytes(header)
		bytesHeaders += len(blob)
	}
	bytesCodes := 0
	for code := range w.Codes {
		bytesCodes += len(code)
	}
	bytesState := 0
	for node := range w.State {
		bytesState += len(node)
	}
	buf := new(bytes.Buffer)

	fmt.Fprintf(buf, "Witness #%d: %v\n", w.Block.Number(), common.StorageSize(bytesTotal))
	fmt.Fprintf(buf, "     block (%4d txs):  %10v\n", len(w.Block.Transactions()), common.StorageSize(bytesBlock))
	fmt.Fprintf(buf, "%4d headers:      %10v\n", len(w.Headers), common.StorageSize(bytesHeaders))
	fmt.Fprintf(buf, "%4d trie nodes:   %10v\n", len(w.State), common.StorageSize(bytesState))
	fmt.Fprintf(buf, "%4d codes:        %10v\n", len(w.Codes), common.StorageSize(bytesCodes))

	return buf.String()
}

// Root returns the pre-state root from the first header.
//
// Note, this method will panic in case of a bad witness (but RLP decoding will
// sanitize it and fail before that).
func (w *Witness) Root() common.Hash {
	return w.Headers[0].Root
}
