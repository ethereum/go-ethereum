// Copyright 2026 the libevm authors.
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

package eth

import (
	"github.com/ava-labs/libevm/core/bloombits"
	"github.com/ava-labs/libevm/ethdb"
)

const (
	// BloomFilterThreads is the number of goroutines used locally per filter to
	// multiplex requests onto the global servicing goroutines.
	BloomFilterThreads = bloomFilterThreads

	// BloomRetrievalBatch is the maximum number of bloom bit retrievals to
	// service in a single batch.
	BloomRetrievalBatch = bloomRetrievalBatch

	// BloomRetrievalWait is the maximum time to wait for enough bloom bit
	// requests to accumulate request an entire batch (avoiding hysteresis).
	BloomRetrievalWait = bloomRetrievalWait
)

// StartBloomHandlers starts a batch of goroutines to serve data for
// [bloombits.Retrieval] requests from any number of filters. This is identical
// to [Ethereum.startBloomHandlers], but exposed for independent use.
func StartBloomHandlers(db ethdb.Database, sectionSize uint64) *BloomHandlers {
	bh := &BloomHandlers{
		Requests: make(chan chan *bloombits.Retrieval),
		quit:     make(chan struct{}),
	}
	eth := &Ethereum{
		bloomRequests:     bh.Requests,
		closeBloomHandler: bh.quit,
		chainDb:           db,
	}
	eth.startBloomHandlers(sectionSize)
	return bh
}

// BloomHandlers serve data for [bloombits.Retrieval] requests from any number
// of filters. [BloomHandlers.Close] MUST be called to release goroutines, after
// which a send on the requests channel will block indefinitely.
type BloomHandlers struct {
	Requests chan chan *bloombits.Retrieval
	quit     chan struct{}
}

// Close releases resources in use by the [BloomHandlers]; repeated calls will
// panic.
func (bh *BloomHandlers) Close() {
	close(bh.quit)
}
