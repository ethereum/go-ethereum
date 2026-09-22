// Copyright 2026 The go-ethereum Authors
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

package core

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/bitutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/trie"
)

// receiptDigest holds the two values the block validator needs from the
// receipts of a block.
type receiptDigest struct {
	bloom types.Bloom // bloom filter of the block, the receipt blooms merged
	root  common.Hash // root of the receipt trie
}

// receiptPipeline turns the receipts of a block into their digest on a
// goroutine of its own, fed as each transaction finishes so the hashing
// overlaps the transactions still to run. Only the goroutine processing the
// block may drive it.
type receiptPipeline struct {
	feed   chan *types.Receipt
	done   chan struct{}
	closed bool

	// bloomed tells the pipeline that the receipts arrive with their bloom
	// filter already computed.
	bloomed bool

	// digest is the result. It is only valid once done is closed.
	digest receiptDigest
}

// newReceiptPipeline starts the pipeline for a block of txs transactions. Set
// bloomed when the receipts already carry their bloom filter, the pipeline then
// uses it rather than hashing the logs again.
func newReceiptPipeline(txs int, bloomed bool) *receiptPipeline {
	p := &receiptPipeline{
		feed:    make(chan *types.Receipt, txs),
		done:    make(chan struct{}),
		bloomed: bloomed,
	}
	go p.run(txs)
	return p
}

// run consumes the receipts of the block and computes their digest.
func (p *receiptPipeline) run(txs int) {
	defer close(p.done)

	var (
		bloom    types.Bloom
		receipts = make(types.Receipts, 0, txs)
		stream   = types.NewListHashStream(trie.NewStackTrie(nil))
	)
	for receipt := range p.feed {
		if !p.bloomed {
			receipt.Bloom = types.CreateBloom(receipt)
		}
		if len(receipt.Logs) != 0 {
			bitutil.ORBytes(bloom[:], bloom[:], receipt.Bloom[:])
		}
		// The receipt encoding covers the bloom, so the trie is fed after it.
		receipts = append(receipts, receipt)
		stream.Update(receipts)
	}
	p.digest = receiptDigest{
		bloom: bloom,
		root:  stream.Hash(),
	}
}

// add hands a receipt over. It must not be touched again until the pipeline
// has been joined, the pipeline fills in its bloom filter.
func (p *receiptPipeline) add(receipt *types.Receipt) {
	p.feed <- receipt
}

// close tells the pipeline that no more receipts are coming, so it can finish
// the trie while the processor wraps the block up. Calling it twice is fine,
// both joining and abandoning a block go through here.
func (p *receiptPipeline) close() {
	if !p.closed {
		p.closed = true
		close(p.feed)
	}
}

// join waits for the pipeline to drain and returns the digest of the receipts
// it was given.
func (p *receiptPipeline) join() receiptDigest {
	p.close()
	<-p.done
	return p.digest
}

// blockBloom returns the bloom filter of the processed block, merged alongside
// execution if a pipeline ran, and from the receipts here if none did.
func (r *ProcessResult) blockBloom() types.Bloom {
	if r.digest != nil {
		return r.digest.bloom
	}
	return types.MergeBloom(r.Receipts)
}

// receiptRoot returns the root of the receipt trie of the processed block,
// built alongside execution if a pipeline ran, and from the receipts here if
// none did.
func (r *ProcessResult) receiptRoot() common.Hash {
	if r.digest != nil {
		return r.digest.root
	}
	return types.DeriveSha(r.Receipts, trie.NewStackTrie(nil))
}
