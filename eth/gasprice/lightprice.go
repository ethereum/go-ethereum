// Copyright 2015 The go-ethereum Authors
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

package gasprice

import (
	"math/big"
	"sort"
	"sync"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/internal/ethapi"
	"github.com/ethereum/go-ethereum/rpc"
	"golang.org/x/net/context"
)

const (
	LpoAvgCount     = 5
	LpoMinCount     = 3
	LpoMaxBlocks    = 20
	LpoSelect       = 50
	LpoDefaultPrice = 20000000000
)

// LightPriceOracle recommends gas prices based on the content of recent
// blocks. Suitable for both light and full clients.
type LightPriceOracle struct {
	backend   ethapi.Backend
	lastHead  common.Hash
	lastPrice *big.Int
	cacheLock sync.RWMutex
	fetchLock sync.Mutex
}

// NewLightPriceOracle returns a new oracle.
func NewLightPriceOracle(backend ethapi.Backend) *LightPriceOracle {
	return &LightPriceOracle{
		backend:   backend,
		lastPrice: big.NewInt(LpoDefaultPrice),
	}
}

// SuggestPrice returns the recommended gas price.
func (self *LightPriceOracle) SuggestPrice(ctx context.Context) (*big.Int, error) {
	self.cacheLock.RLock()
	lastHead := self.lastHead
	lastPrice := self.lastPrice
	self.cacheLock.RUnlock()

	head, _ := self.backend.HeaderByNumber(ctx, rpc.LatestBlockNumber)
	headHash := head.Hash()
	if headHash == lastHead {
		return lastPrice, nil
	}

	self.fetchLock.Lock()
	defer self.fetchLock.Unlock()

	// try checking the cache again, maybe the last fetch fetched what we need
	self.cacheLock.RLock()
	lastHead = self.lastHead
	lastPrice = self.lastPrice
	self.cacheLock.RUnlock()
	if headHash == lastHead {
		return lastPrice, nil
	}

	blockNum := head.Number.Uint64()
	chn := make(chan lpResult, LpoMaxBlocks)
	sent := 0
	exp := 0
	var lps bigIntArray
	for sent < LpoAvgCount && blockNum > 0 {
		go self.getLowestPrice(ctx, blockNum, chn)
		sent++
		exp++
		blockNum--
	}
	maxEmpty := LpoAvgCount - LpoMinCount
	for exp > 0 {
		res := <-chn
		if res.err != nil {
			return nil, res.err
		}
		exp--
		if res.price != nil {
			lps = append(lps, res.price)
		} else {
			if maxEmpty > 0 {
				maxEmpty--
			} else {
				if blockNum > 0 && sent < LpoMaxBlocks {
					go self.getLowestPrice(ctx, blockNum, chn)
					sent++
					exp++
					blockNum--
				}
			}
		}
	}
	price := lastPrice
	if len(lps) > 0 {
		sort.Sort(lps)
		price = lps[(len(lps)-1)*LpoSelect/100]
	}

	self.cacheLock.Lock()
	self.lastHead = headHash
	self.lastPrice = price
	self.cacheLock.Unlock()
	return price, nil
}

type lpResult struct {
	price *big.Int
	err   error
}

// getLowestPrice calculates the lowest transaction gas price in a given block
// and sends it to the result channel. If the block is empty, price is nil.
func (self *LightPriceOracle) getLowestPrice(ctx context.Context, blockNum uint64, chn chan lpResult) {
	block, err := self.backend.BlockByNumber(ctx, rpc.BlockNumber(blockNum))
	if block == nil {
		chn <- lpResult{nil, err}
		return
	}
	txs := block.Transactions()
	if len(txs) == 0 {
		chn <- lpResult{nil, nil}
		return
	}
	// find smallest gasPrice
	minPrice := txs[0].GasPrice()
	for i := 1; i < len(txs); i++ {
		price := txs[i].GasPrice()
		if price.Cmp(minPrice) < 0 {
			minPrice = price
		}
	}
	chn <- lpResult{minPrice, nil}
}

type bigIntArray []*big.Int

func (s bigIntArray) Len() int           { return len(s) }
func (s bigIntArray) Less(i, j int) bool { return s[i].Cmp(s[j]) < 0 }
func (s bigIntArray) Swap(i, j int)      { s[i], s[j] = s[j], s[i] }
