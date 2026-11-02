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
	"github.com/ethereum/go-ethereum/core/types/bal"
	"github.com/ethereum/go-ethereum/trie"
)

// receiptDigest holds the two values the block validator needs from the
// receipts of a block.
type receiptDigest struct {
	bloom types.Bloom // bloom filter of the block, the receipt blooms merged
	root  common.Hash // root of the receipt trie
}

// accessListDigest holds the two values the block validator needs from the
// access list of a block.
type accessListDigest struct {
	list *bal.BlockAccessList // access list in its encoding form
	hash common.Hash          // hash of the encoded access list
}

type stage[In, Out any] struct {
	in   chan In
	out  Out // only valid once done is closed
	done chan struct{}
}

// wait blocks until the result is ready and returns it.
func (s *stage[In, Out]) wait() Out {
	<-s.done
	return s.out
}

// digestPipeline turns the receipts and the access list of a block into their
// digests on a goroutine of its own.
type digestPipeline struct {
	receipts   stage[*types.Receipt, receiptDigest]
	accessList stage[*bal.ConstructionBlockAccessList, accessListDigest]

	// bloomed tells the pipeline that the receipts arrive with their bloom
	// filter already computed.
	bloomed bool
}

// newDigestPipeline starts the pipeline for a block of txs transactions. Set
// bloomed when the receipts already carry their bloom filter, the pipeline then
// uses it rather than hashing the logs again.
func newDigestPipeline(txs int, bloomed bool) *digestPipeline {
	p := &digestPipeline{
		receipts: stage[*types.Receipt, receiptDigest]{
			in:   make(chan *types.Receipt, txs),
			done: make(chan struct{}),
		},
		accessList: stage[*bal.ConstructionBlockAccessList, accessListDigest]{
			in:   make(chan *bal.ConstructionBlockAccessList, 1),
			done: make(chan struct{}),
		},
		bloomed: bloomed,
	}
	go p.run(txs)
	return p
}

// run consumes the receipts of the block and computes their digest, then does
// the same for the access list once it arrives.
func (p *digestPipeline) run(txs int) {
	var (
		bloom    types.Bloom
		receipts = make(types.Receipts, 0, txs)
		stream   = types.NewListHashStream(trie.NewStackTrie(nil))
	)
	for range txs {
		// The input closes early only when the block was abandoned, nobody
		// reads the digest then.
		receipt, ok := <-p.receipts.in
		if !ok {
			return
		}
		if !p.bloomed {
			receipt.Bloom = types.CreateBloom(receipt)
		}
		if len(receipt.Logs) != 0 {
			bitutil.ORBytes(bloom[:], bloom[:], receipt.Bloom[:])
		}
		receipts = append(receipts, receipt)
		stream.Update(receipts)
	}
	p.receipts.out = receiptDigest{
		bloom: bloom,
		root:  stream.Hash(),
	}
	close(p.receipts.done)

	// The access list arrives last, once the block is finalized. The stage
	// closes empty when the block was abandoned, nobody reads the digest then.
	if list, ok := <-p.accessList.in; ok {
		enc := list.ToEncodingObj()
		p.accessList.out = accessListDigest{
			list: enc,
			hash: enc.Hash(),
		}
	}
	close(p.accessList.done)
}

// feedReceipt hands a receipt over. It must not be touched again until the
// receipts have been joined, the pipeline fills in its bloom filter.
func (p *digestPipeline) feedReceipt(receipt *types.Receipt) {
	p.receipts.in <- receipt
}

// feedBAL gives the pipeline the final access list, so it can encode and hash
// it while the block is validated. It must not be mutated afterwards and must
// be fed only once.
func (p *digestPipeline) feedBAL(list *bal.ConstructionBlockAccessList) {
	p.accessList.in <- list
}

// abandon ends both inputs, so the goroutine runs out and does not linger when
// the block is dropped half way through. It must be called only once, after
// the last feed.
func (p *digestPipeline) abandon() {
	close(p.receipts.in)
	close(p.accessList.in)
}

// joinReceipts waits for the receipts to be digested and returns the result.
// Afterwards the receipts carry their blooms and are safe to hand back.
func (p *digestPipeline) joinReceipts() receiptDigest {
	return p.receipts.wait()
}

// joinAccessList waits for the access list to be digested and returns the
// result. The list must have been fed first.
func (p *digestPipeline) joinAccessList() accessListDigest {
	return p.accessList.wait()
}

// blockBloom returns the bloom filter of the processed block, merged alongside
// execution if a pipeline ran, and from the receipts here if none did.
func (r *ProcessResult) blockBloom() types.Bloom {
	if r.pipeline != nil {
		return r.pipeline.joinReceipts().bloom
	}
	return types.MergeBloom(r.Receipts)
}

// receiptRoot returns the root of the receipt trie of the processed block,
// built alongside execution if a pipeline ran, and from the receipts here if
// none did.
func (r *ProcessResult) receiptRoot() common.Hash {
	if r.pipeline != nil {
		return r.pipeline.joinReceipts().root
	}
	return types.DeriveSha(r.Receipts, trie.NewStackTrie(nil))
}

// encodedAccessList returns the block access list in its encoding form along
// with its hash, converted alongside validation. Both are empty for a result
// assembled without a pipeline.
func (r *ProcessResult) encodedAccessList() (*bal.BlockAccessList, common.Hash) {
	if r.pipeline == nil {
		return nil, common.Hash{}
	}
	digest := r.pipeline.joinAccessList()
	return digest.list, digest.hash
}
