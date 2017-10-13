// Copyright 2017 The go-ethereum Authors
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

package downloader

import (
	"fmt"
	"math/big"
	"sync/atomic"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/ethereum/go-ethereum/trie"
	"github.com/golang/snappy"
)

// FakePeer is a mock downloader peer that operates on a local database instance
// instead of being an actual live node. It's useful for testing and to implement
// sync commands from an xisting local database.
type FakePeer struct {
	id string
	db ethdb.Database
	hc *core.HeaderChain
	dl *Downloader
}

// NewFakePeer creates a new mock downloader peer with the given data sources.
func NewFakePeer(id string, db ethdb.Database, hc *core.HeaderChain, dl *Downloader) *FakePeer {
	return &FakePeer{id: id, db: db, hc: hc, dl: dl}
}

// Head implements downloader.Peer, returning the current head hash and number
// of the best known header.
func (p *FakePeer) Head() (common.Hash, *big.Int) {
	header := p.hc.CurrentHeader()
	return header.Hash(), header.Number
}

// RequestHeadersByHash implements downloader.Peer, returning a batch of headers
// defined by the origin hash and the associaed query parameters.
func (p *FakePeer) RequestHeadersByHash(hash common.Hash, amount int, skip int, reverse bool) error {
	var (
		headers []*types.Header
		unknown bool
	)
	for !unknown && len(headers) < amount {
		origin := p.hc.GetHeaderByHash(hash)
		if origin == nil {
			break
		}
		number := origin.Number.Uint64()
		headers = append(headers, origin)
		if reverse {
			for i := 0; i < int(skip)+1; i++ {
				if header := p.hc.GetHeader(hash, number); header != nil {
					hash = header.ParentHash
					number--
				} else {
					unknown = true
					break
				}
			}
		} else {
			var (
				current = origin.Number.Uint64()
				next    = current + uint64(skip) + 1
			)
			if header := p.hc.GetHeaderByNumber(next); header != nil {
				if p.hc.GetBlockHashesFromHash(header.Hash(), uint64(skip+1))[skip] == hash {
					hash = header.Hash()
				} else {
					unknown = true
				}
			} else {
				unknown = true
			}
		}
	}
	p.dl.DeliverHeaders(p.id, headers)
	return nil
}

// RequestHeadersByNumber implements downloader.Peer, returning a batch of headers
// defined by the origin number and the associaed query parameters.
func (p *FakePeer) RequestHeadersByNumber(number uint64, amount int, skip int, reverse bool) error {
	var (
		headers []*types.Header
		unknown bool
	)
	for !unknown && len(headers) < amount {
		origin := p.hc.GetHeaderByNumber(number)
		if origin == nil {
			break
		}
		if reverse {
			if number >= uint64(skip+1) {
				number -= uint64(skip + 1)
			} else {
				unknown = true
			}
		} else {
			number += uint64(skip + 1)
		}
		headers = append(headers, origin)
	}
	p.dl.DeliverHeaders(p.id, headers)
	return nil
}

// RequestBodies implements downloader.Peer, returning a batch of block bodies
// corresponding to the specified block hashes.
func (p *FakePeer) RequestBodies(hashes []common.Hash) error {
	var (
		txs    [][]*types.Transaction
		uncles [][]*types.Header
	)
	for _, hash := range hashes {
		block := core.GetBlock(p.db, hash, p.hc.GetBlockNumber(hash))

		txs = append(txs, block.Transactions())
		uncles = append(uncles, block.Uncles())
	}
	p.dl.DeliverBodies(p.id, txs, uncles)
	return nil
}

// RequestReceipts implements downloader.Peer, returning a batch of transaction
// receipts corresponding to the specified block hashes.
func (p *FakePeer) RequestReceipts(hashes []common.Hash) error {
	var receipts [][]*types.Receipt
	for _, hash := range hashes {
		receipts = append(receipts, core.GetBlockReceipts(p.db, hash, p.hc.GetBlockNumber(hash)))
	}
	p.dl.DeliverReceipts(p.id, receipts)
	return nil
}

var cnt int32
var siz int32

func init() {
	go func() {
		for {
			time.Sleep(time.Second)
			//fmt.Println("Packets:", atomic.LoadInt32(&cnt), "Bytes (snappy):", atomic.LoadInt32(&siz))
		}
	}()
}

// RequestNodeData implements downloader.Peer, returning a batch of state trie
// nodes corresponding to the specified trie hashes.
func (p *FakePeer) RequestNodeData(hashes []common.Hash) error {
	var data [][]byte
	for _, hash := range hashes {
		if entry, err := p.db.Get(hash.Bytes()); err == nil {
			data = append(data, entry)
		}
	}

	atomic.AddInt32(&cnt, 1)
	blob, _ := rlp.EncodeToBytes(data)
	atomic.AddInt32(&siz, int32(len(snappy.Encode(nil, blob))))

	p.dl.DeliverNodeData(p.id, data)
	return nil
}

// RequestTries implements downloader.Peer, returning the (potentially partial)
// leaves of a state sub-trie.
func (p *FakePeer) RequestTries(roots []common.Hash, limit common.StorageSize) error {
	start := time.Now()

	t, err := trie.New(common.Hash{}, p.db)
	if err != nil {
		return err
	}
	results := make([]*trie.SyncResult, 0, len(roots))
	for _, root := range roots {
		res, size, err := t.FetchData(root, limit, time.Second-time.Since(start))
		if err != nil {
			return err
		}
		results = append(results, res)

		if size >= limit || time.Since(start) > time.Second {
			break
		}
		limit -= size
	}

	atomic.AddInt32(&cnt, 1)
	blob, _ := rlp.EncodeToBytes(results)
	size := len(snappy.Encode(nil, blob))
	atomic.AddInt32(&siz, int32(size))

	fmt.Printf("Packet #%d: %d bytes (%d total) in %v\n", atomic.LoadInt32(&cnt), size, atomic.LoadInt32(&siz), time.Since(start))

	p.dl.DeliverTries(p.id, results)
	return nil
}
