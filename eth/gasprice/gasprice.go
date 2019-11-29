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
	"context"
	"math/big"
	"sort"
	"sync"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/internal/ethapi"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/rpc"
)

const (
	fullSampleNumber  = 1 // Number of transactions sampled in a block for full node
	lightSampleNumber = 3 // Number of transactions sampled in a block for light client
)

var maxPrice = big.NewInt(500 * params.GWei)

type Config struct {
	Blocks     int
	Percentile int
	Default    *big.Int `toml:",omitempty"`
}

// Oracle recommends gas prices based on the content of recent
// blocks. Suitable for both light and full clients.
type Oracle struct {
	backend      ethapi.Backend
	lastHead     common.Hash
	defaultPrice *big.Int
	lastPrice    *big.Int
	cacheLock    sync.RWMutex
	fetchLock    sync.Mutex

	checkBlocks, maxInvalid, maxBlocks int
	sampleNumber, percentile           int
}

// newOracle returns a new gasprice oracle which can recommend suitable
// gasprice for newly created transaction.
func newOracle(backend ethapi.Backend, sampleNumber int, params Config) *Oracle {
	blocks := params.Blocks
	if blocks < 1 {
		blocks = 1
	}
	percent := params.Percentile
	if percent < 0 {
		percent = 0
	}
	if percent > 100 {
		percent = 100
	}
	return &Oracle{
		backend:      backend,
		defaultPrice: params.Default,
		lastPrice:    params.Default,
		checkBlocks:  blocks,
		maxInvalid:   blocks / 2,
		maxBlocks:    blocks * 5,
		sampleNumber: sampleNumber,
		percentile:   percent,
	}
}

// NewFullOracle returns a gasprice oracle for full node which has
// avaiblable recent blocks in local db. FullOracle has higher
// recommendation accuracy.
func NewFullOracle(backend ethapi.Backend, params Config) *Oracle {
	return newOracle(backend, fullSampleNumber, params)
}

// NewLightOracle returns a gasprice oracle for light client which doesn't
// has recent block locally. LightOracle is much cheaper than FullOracle
// however the corresponding recommendation accuracy is lower.
func NewLightOracle(backend ethapi.Backend, params Config) *Oracle {
	return newOracle(backend, lightSampleNumber, params)
}

// getLatest returns a recommended gas price which is suggested last time
// but still suitable now.
func (gpo *Oracle) getLatest(headHash common.Hash) *big.Int {
	gpo.cacheLock.RLock()
	lastHead := gpo.lastHead
	lastPrice := gpo.lastPrice
	gpo.cacheLock.RUnlock()
	if headHash == lastHead {
		return lastPrice
	}
	return nil
}

// SuggesstPrice returns a gasprice so that newly created transaction can
// has very high chance to be included in the following blocks.
func (gpo *Oracle) SuggestPrice(ctx context.Context) (*big.Int, error) {
	head, _ := gpo.backend.HeaderByNumber(ctx, rpc.LatestBlockNumber)
	headHash := head.Hash()
	// Firstly check whether there is available gasprice for recommendation.
	if price := gpo.getLatest(headHash); price != nil {
		return price, nil
	}
	gpo.fetchLock.Lock()
	defer gpo.fetchLock.Unlock()
	// Try checking the cache again, maybe the last fetch fetched what we need
	if price := gpo.getLatest(headHash); price != nil {
		return price, nil
	}
	var (
		sent, exp int
		number    = head.Number.Uint64()
		ch        = make(chan getBlockPricesResult, gpo.checkBlocks)
		txPrices  []*big.Int
	)
	for sent < gpo.checkBlocks && number > 0 {
		go gpo.getBlockPrices(ctx, types.MakeSigner(gpo.backend.ChainConfig(), big.NewInt(int64(number))), number, gpo.sampleNumber, ch)
		sent++
		exp++
		number--
	}
	maxInvalid := gpo.maxInvalid
	for exp > 0 {
		res := <-ch
		if res.err != nil {
			return gpo.lastPrice, res.err
		}
		exp--
		if res.prices != nil {
			txPrices = append(txPrices, res.prices...)
			continue
		}
		if maxInvalid > 0 {
			maxInvalid--
			continue
		}
		if number > 0 && sent < gpo.maxBlocks {
			go gpo.getBlockPrices(ctx, types.MakeSigner(gpo.backend.ChainConfig(), big.NewInt(int64(number))), number, gpo.sampleNumber, ch)
			sent++
			exp++
			number--
		}
	}
	price := gpo.lastPrice
	if len(txPrices) > 0 {
		sort.Sort(bigIntArray(txPrices))
		price = txPrices[(len(txPrices)-1)*gpo.percentile/100]
	}
	if price.Cmp(maxPrice) > 0 {
		price = new(big.Int).Set(maxPrice)
	}
	gpo.cacheLock.Lock()
	gpo.lastHead = headHash
	gpo.lastPrice = price
	gpo.cacheLock.Unlock()
	return price, nil
}

type getBlockPricesResult struct {
	prices []*big.Int
	err    error
}

type transactionsByGasPrice []*types.Transaction

func (t transactionsByGasPrice) Len() int           { return len(t) }
func (t transactionsByGasPrice) Swap(i, j int)      { t[i], t[j] = t[j], t[i] }
func (t transactionsByGasPrice) Less(i, j int) bool { return t[i].GasPriceCmp(t[j]) < 0 }

// getBlockPrices calculates the lowest transaction gas price in a given block
// and sends it to the result channel. If the block is empty, price is nil.
func (gpo *Oracle) getBlockPrices(ctx context.Context, signer types.Signer, blockNum uint64, limit int, ch chan getBlockPricesResult) {
	block, err := gpo.backend.BlockByNumber(ctx, rpc.BlockNumber(blockNum))
	if block == nil {
		ch <- getBlockPricesResult{nil, err}
		return
	}
	blockTxs := block.Transactions()
	// If the block is empty, it means the lowest gas price is
	// enough to let our transaction to be included.
	//
	// There is a corner case that some miners choose to not include
	// any transaction. If so, the recommended gas price is too low.
	// However for this case, node can query enough recent blocks.
	// In theory, it's very unlikely for all recent miners to intentionally
	// choose to not include any transaction.
	if len(blockTxs) == 0 {
		ch <- getBlockPricesResult{[]*big.Int{gpo.defaultPrice}, nil}
		return
	}
	txs := make([]*types.Transaction, len(blockTxs))
	copy(txs, blockTxs)
	sort.Sort(transactionsByGasPrice(txs))

	var result []*big.Int
	for _, tx := range txs {
		sender, err := types.Sender(signer, tx)
		if err == nil && sender != block.Coinbase() {
			result = append(result, tx.GasPrice())
			if len(result) >= limit {
				break
			}
		}
	}
	ch <- getBlockPricesResult{result, nil}
}

type bigIntArray []*big.Int

func (s bigIntArray) Len() int           { return len(s) }
func (s bigIntArray) Less(i, j int) bool { return s[i].Cmp(s[j]) < 0 }
func (s bigIntArray) Swap(i, j int)      { s[i], s[j] = s[j], s[i] }
