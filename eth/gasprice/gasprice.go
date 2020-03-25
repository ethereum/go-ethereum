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
	"errors"
	"math/big"
	"sort"
	"sync"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/rpc"
)

const sampleNumber = 3 // Number of transactions sampled in a block

var (
	DefaultMaxPrice = big.NewInt(500 * params.GWei)

	maxPrice   = big.NewInt(500 * params.GWei)
	maxPremium = big.NewInt(500 * params.GWei)
	maxFeeCap  = big.NewInt(1000 * params.GWei)

	errEIP1559IsFinalized    = errors.New("past EIP1559 finalization, GasPrice cannot be set")
	errEIP1559IsNotActivated = errors.New("before EIP1559 activation, GasPremium and FeeCap cannot be set")
)

type Config struct {
	Blocks            int
	Percentile        int
	Default           *big.Int `toml:",omitempty"`
	DefaultGasPremium *big.Int `toml:",omitempty"`
	DefaultFeeCap     *big.Int `toml:",omitempty"`
	MaxPrice          *big.Int `toml:",omitempty"`
}

// OracleBackend includes all necessary background APIs for oracle.
type OracleBackend interface {
	HeaderByNumber(ctx context.Context, number rpc.BlockNumber) (*types.Header, error)
	BlockByNumber(ctx context.Context, number rpc.BlockNumber) (*types.Block, error)
	ChainConfig() *params.ChainConfig
}

// Oracle recommends gas prices based on the content of recent
// blocks. Suitable for both light and full clients.
type Oracle struct {
	backend     OracleBackend
	lastHead    common.Hash
	lastPrice   *big.Int
	lastPremium *big.Int
	lastCap     *big.Int
	lastBaseFee *big.Int
	maxPrice    *big.Int
	cacheLock   sync.RWMutex
	fetchLock   sync.Mutex

	checkBlocks int
	percentile  int
}

// NewOracle returns a new gasprice oracle which can recommend suitable
// gasprice for newly created transaction.
func NewOracle(backend OracleBackend, params Config) *Oracle {
	blocks := params.Blocks
	if blocks < 1 {
		blocks = 1
		log.Warn("Sanitizing invalid gasprice oracle sample blocks", "provided", params.Blocks, "updated", blocks)
	}
	percent := params.Percentile
	if percent < 0 {
		percent = 0
		log.Warn("Sanitizing invalid gasprice oracle sample percentile", "provided", params.Percentile, "updated", percent)
	}
	if percent > 100 {
		percent = 100
		log.Warn("Sanitizing invalid gasprice oracle sample percentile", "provided", params.Percentile, "updated", percent)
	}
	maxPrice := params.MaxPrice
	if maxPrice == nil || maxPrice.Int64() <= 0 {
		maxPrice = DefaultMaxPrice
		log.Warn("Sanitizing invalid gasprice oracle price cap", "provided", params.MaxPrice, "updated", maxPrice)
	}
	return &Oracle{
		backend:     backend,
		lastPrice:   params.Default,
		lastPremium: params.DefaultGasPremium,
		lastCap:     params.DefaultFeeCap,
		maxPrice:    maxPrice,
		checkBlocks: blocks,
		percentile:  percent,
	}
}

// SuggestPrice returns a gasprice so that newly created transaction can
// have a very high chance to be included in the following blocks.
func (gpo *Oracle) SuggestPrice(ctx context.Context) (*big.Int, error) {
	head, _ := gpo.backend.HeaderByNumber(ctx, rpc.LatestBlockNumber)
	if gpo.backend.ChainConfig().IsEIP1559Finalized(head.Number) {
		return nil, errEIP1559IsFinalized
	}
	headHash := head.Hash()

	// If the latest gasprice is still available, return it.
	gpo.cacheLock.RLock()
	lastHead, lastPrice := gpo.lastHead, gpo.lastPrice
	gpo.cacheLock.RUnlock()
	if headHash == lastHead {
		return lastPrice, nil
	}
	gpo.fetchLock.Lock()
	defer gpo.fetchLock.Unlock()

	// Try checking the cache again, maybe the last fetch fetched what we need
	gpo.cacheLock.RLock()
	lastHead, lastPrice = gpo.lastHead, gpo.lastPrice
	gpo.cacheLock.RUnlock()
	if headHash == lastHead {
		return lastPrice, nil
	}
	var (
		sent, exp int
		number    = head.Number.Uint64()
		result    = make(chan getBlockPricesResult, gpo.checkBlocks)
		quit      = make(chan struct{})
		txPrices  []*big.Int
	)
	for sent < gpo.checkBlocks && number > 0 {
		go gpo.getBlockPrices(ctx, types.MakeSigner(gpo.backend.ChainConfig(), big.NewInt(int64(number))), number, sampleNumber, result, quit)
		sent++
		exp++
		number--
	}
	for exp > 0 {
		res := <-result
		if res.err != nil {
			close(quit)
			return lastPrice, res.err
		}
		exp--
		// Nothing returned. There are two special cases here:
		// - The block is empty
		// - All the transactions included are sent by the miner itself.
		// In these cases, use the latest calculated price for samping.
		if len(res.prices) == 0 {
			res.prices = []*big.Int{lastPrice}
		}
		// Besides, in order to collect enough data for sampling, if nothing
		// meaningful returned, try to query more blocks. But the maximum
		// is 2*checkBlocks.
		if len(res.prices) == 1 && len(txPrices)+1+exp < gpo.checkBlocks*2 && number > 0 {
			go gpo.getBlockPrices(ctx, types.MakeSigner(gpo.backend.ChainConfig(), big.NewInt(int64(number))), number, sampleNumber, result, quit)
			sent++
			exp++
			number--
		}
		txPrices = append(txPrices, res.prices...)
	}
	price := lastPrice
	if len(txPrices) > 0 {
		sort.Sort(bigIntArray(txPrices))
		price = txPrices[(len(txPrices)-1)*gpo.percentile/100]
	}
	if price.Cmp(gpo.maxPrice) > 0 {
		price = new(big.Int).Set(gpo.maxPrice)
	}
	gpo.cacheLock.Lock()
	gpo.lastHead = headHash
	gpo.lastPrice = price
	gpo.cacheLock.Unlock()
	return price, nil
}

// SuggestPremium returns the recommended gas premium.
func (gpo *Oracle) SuggestPremium(ctx context.Context) (*big.Int, error) {
	gpo.cacheLock.RLock()
	lastHead := gpo.lastHead
	lastPremium := gpo.lastPremium
	gpo.cacheLock.RUnlock()

	head, _ := gpo.backend.HeaderByNumber(ctx, rpc.LatestBlockNumber)
	if !gpo.backend.ChainConfig().IsEIP1559(head.Number) {
		return nil, errEIP1559IsNotActivated
	}
	headHash := head.Hash()
	if headHash == lastHead {
		return lastPremium, nil
	}

	gpo.fetchLock.Lock()
	defer gpo.fetchLock.Unlock()

	// try checking the cache again, maybe the last fetch fetched what we need
	gpo.cacheLock.RLock()
	lastHead = gpo.lastHead
	lastPremium = gpo.lastPremium
	gpo.cacheLock.RUnlock()
	if headHash == lastHead {
		return lastPremium, nil
	}

	blockNum := head.Number.Uint64()
	ch := make(chan getBlockPremiumsResult, gpo.checkBlocks)
	sent := 0
	exp := 0
	var blockPremiums []*big.Int
	for sent < gpo.checkBlocks && blockNum > 0 {
		go gpo.getBlockPremiums(ctx, types.MakeSigner(gpo.backend.ChainConfig(), big.NewInt(int64(blockNum))), blockNum, ch)
		sent++
		exp++
		blockNum--
	}
	maxEmpty := gpo.maxEmpty
	for exp > 0 {
		res := <-ch
		if res.err != nil {
			return lastPremium, res.err
		}
		exp--
		if res.premium != nil {
			blockPremiums = append(blockPremiums, res.premium)
			continue
		}
		if maxEmpty > 0 {
			maxEmpty--
			continue
		}
		if blockNum > 0 && sent < gpo.maxBlocks {
			go gpo.getBlockPremiums(ctx, types.MakeSigner(gpo.backend.ChainConfig(), big.NewInt(int64(blockNum))), blockNum, ch)
			sent++
			exp++
			blockNum--
		}
	}
	premium := lastPremium
	if len(blockPremiums) > 0 {
		sort.Sort(bigIntArray(blockPremiums))
		premium = blockPremiums[(len(blockPremiums)-1)*gpo.percentile/100]
	}
	if premium.Cmp(maxPremium) > 0 {
		premium = new(big.Int).Set(maxPremium)
	}

	gpo.cacheLock.Lock()
	gpo.lastHead = headHash
	gpo.lastPremium = premium
	gpo.cacheLock.Unlock()
	return premium, nil
}

// SuggestCap returns the recommended fee cap.
func (gpo *Oracle) SuggestCap(ctx context.Context) (*big.Int, error) {
	gpo.cacheLock.RLock()
	lastHead := gpo.lastHead
	lastCap := gpo.lastCap
	gpo.cacheLock.RUnlock()

	head, _ := gpo.backend.HeaderByNumber(ctx, rpc.LatestBlockNumber)
	if !gpo.backend.ChainConfig().IsEIP1559(head.Number) {
		return nil, errEIP1559IsNotActivated
	}
	headHash := head.Hash()
	if headHash == lastHead {
		return lastCap, nil
	}

	gpo.fetchLock.Lock()
	defer gpo.fetchLock.Unlock()

	// try checking the cache again, maybe the last fetch fetched what we need
	gpo.cacheLock.RLock()
	lastHead = gpo.lastHead
	lastCap = gpo.lastCap
	gpo.cacheLock.RUnlock()
	if headHash == lastHead {
		return lastCap, nil
	}

	blockNum := head.Number.Uint64()
	ch := make(chan getBlockCapsResult, gpo.checkBlocks)
	sent := 0
	exp := 0
	var blockCaps []*big.Int
	for sent < gpo.checkBlocks && blockNum > 0 {
		go gpo.getBlockCaps(ctx, types.MakeSigner(gpo.backend.ChainConfig(), big.NewInt(int64(blockNum))), blockNum, ch)
		sent++
		exp++
		blockNum--
	}
	maxEmpty := gpo.maxEmpty
	for exp > 0 {
		res := <-ch
		if res.err != nil {
			return lastCap, res.err
		}
		exp--
		if res.cap != nil {
			blockCaps = append(blockCaps, res.cap)
			continue
		}
		if maxEmpty > 0 {
			maxEmpty--
			continue
		}
		if blockNum > 0 && sent < gpo.maxBlocks {
			go gpo.getBlockCaps(ctx, types.MakeSigner(gpo.backend.ChainConfig(), big.NewInt(int64(blockNum))), blockNum, ch)
			sent++
			exp++
			blockNum--
		}
	}
	cap := lastCap
	if len(blockCaps) > 0 {
		sort.Sort(bigIntArray(blockCaps))
		cap = blockCaps[(len(blockCaps)-1)*gpo.percentile/100]
	}
	if cap.Cmp(maxFeeCap) > 0 {
		cap = new(big.Int).Set(maxFeeCap)
	}

	gpo.cacheLock.Lock()
	gpo.lastHead = headHash
	gpo.lastCap = cap
	gpo.cacheLock.Unlock()
	return cap, nil
}

type getBlockPricesResult struct {
	prices []*big.Int
	err    error
}

type getBlockPremiumsResult struct {
	premium *big.Int
	err     error
}

type getBlockCapsResult struct {
	cap *big.Int
	err error
}

type transactionsByGasPrice struct {
	txs     []*types.Transaction
	baseFee *big.Int
}

func (t *transactionsByGasPrice) Len() int      { return len(t.txs) }
func (t *transactionsByGasPrice) Swap(i, j int) { t.txs[i], t.txs[j] = t.txs[j], t.txs[i] }
func (t *transactionsByGasPrice) Less(i, j int) bool {
	iPrice := t.txs[i].GasPrice()
	jPrice := t.txs[j].GasPrice()
	if iPrice == nil {
		iPrice = new(big.Int).Add(t.baseFee, t.txs[i].GasPremium())
		if iPrice.Cmp(t.txs[i].FeeCap()) > 0 {
			iPrice.Set(t.txs[i].FeeCap())
		}
	}
	if jPrice == nil {
		jPrice = new(big.Int).Add(t.baseFee, t.txs[j].GasPremium())
		if jPrice.Cmp(t.txs[j].FeeCap()) > 0 {
			jPrice.Set(t.txs[j].FeeCap())
		}
	}
	return iPrice.Cmp(jPrice) < 0
}

type transactionsByGasPremium struct {
	txs     []*types.Transaction
	baseFee *big.Int
}

func (t *transactionsByGasPremium) Len() int      { return len(t.txs) }
func (t *transactionsByGasPremium) Swap(i, j int) { t.txs[i], t.txs[j] = t.txs[j], t.txs[i] }
func (t *transactionsByGasPremium) Less(i, j int) bool {
	iPremium := t.txs[i].GasPremium()
	jPremium := t.txs[j].GasPremium()
	if iPremium == nil {
		iPremium = new(big.Int).Sub(t.txs[i].GasPrice(), t.baseFee)
		if iPremium.Cmp(common.Big0) < 0 {
			iPremium.Set(common.Big0)
		}
	}
	if jPremium == nil {
		jPremium = new(big.Int).Sub(t.txs[j].GasPrice(), t.baseFee)
		if jPremium.Cmp(common.Big0) < 0 {
			jPremium.Set(common.Big0)
		}
	}
	return iPremium.Cmp(jPremium) < 0
}

type transactionsByFeeCap []*types.Transaction

func (t transactionsByFeeCap) Len() int      { return len(t) }
func (t transactionsByFeeCap) Swap(i, j int) { t[i], t[j] = t[j], t[i] }
func (t transactionsByFeeCap) Less(i, j int) bool {
	iCap := t[i].FeeCap()
	jCap := t[j].FeeCap()
	if iCap == nil {
		iCap = t[i].GasPrice()
	}
	if jCap == nil {
		jCap = t[j].GasPrice()
	}
	return iCap.Cmp(jCap) < 0
}

// getBlockPrices calculates the lowest transaction gas price in a given block
// and sends it to the result channel. If the block is empty or all transactions
// are sent by the miner itself(it doesn't make any sense to include this kind of
// transaction prices for sampling), nil gasprice is returned.
func (gpo *Oracle) getBlockPrices(ctx context.Context, signer types.Signer, blockNum uint64, limit int, result chan getBlockPricesResult, quit chan struct{}) {
	block, err := gpo.backend.BlockByNumber(ctx, rpc.BlockNumber(blockNum))
	if block == nil {
		select {
		case result <- getBlockPricesResult{nil, err}:
		case <-quit:
		}
		return
	}
	blockTxs := block.Transactions()
	txs := new(transactionsByGasPrice)
	txs.txs = make([]*types.Transaction, len(blockTxs))
	copy(txs.txs, blockTxs)
	txs.baseFee = block.BaseFee()
	sort.Sort(txs)

	var prices []*big.Int
	for _, tx := range txs.txs {
		sender, err := types.Sender(signer, tx)
		if err != nil || sender == block.Coinbase() {
			continue
		}
		price := tx.GasPrice()
		if price == nil {
			price = new(big.Int).Add(block.BaseFee(), tx.GasPremium())
			if price.Cmp(tx.FeeCap()) > 0 {
				price.Set(tx.FeeCap())
			}
		}
		prices = append(prices, price)
		if len(prices) >= limit {
			break
		}
	}
	select {
	case result <- getBlockPricesResult{prices, nil}:
	case <-quit:
	}
}

// getBlockPremiums calculates the lowest transaction gas premium in a given block
// and sends it to the result channel. If the block is empty, price is nil.
func (gpo *Oracle) getBlockPremiums(ctx context.Context, signer types.Signer, blockNum uint64, ch chan getBlockPremiumsResult) {
	block, err := gpo.backend.BlockByNumber(ctx, rpc.BlockNumber(blockNum))
	if block == nil {
		ch <- getBlockPremiumsResult{nil, err}
		return
	}

	blockTxs := block.Transactions()
	txs := new(transactionsByGasPremium)
	txs.txs = make([]*types.Transaction, len(blockTxs))
	copy(txs.txs, blockTxs)
	txs.baseFee = block.BaseFee()
	sort.Sort(txs)

	for _, tx := range txs.txs {
		sender, err := types.Sender(signer, tx)
		if err != nil || sender == block.Coinbase() {
			continue
		}
		premium := tx.GasPremium()
		if premium == nil {
			premium = new(big.Int).Sub(tx.GasPrice(), block.BaseFee())
			if premium.Cmp(common.Big0) < 0 {
				premium.Set(common.Big0)
			}
		}
		ch <- getBlockPremiumsResult{premium, nil}
		return
	}
	ch <- getBlockPremiumsResult{nil, nil}
}

// getBlockCaps calculates the lowest transaction fee cap in a given block
// and sends it to the result channel. If the block is empty, price is nil.
func (gpo *Oracle) getBlockCaps(ctx context.Context, signer types.Signer, blockNum uint64, ch chan getBlockCapsResult) {
	block, err := gpo.backend.BlockByNumber(ctx, rpc.BlockNumber(blockNum))
	if block == nil {
		ch <- getBlockCapsResult{nil, err}
		return
	}

	blockTxs := block.Transactions()
	txs := make([]*types.Transaction, len(blockTxs))
	copy(txs, blockTxs)
	sort.Sort(transactionsByFeeCap(txs))

	for _, tx := range txs {
		sender, err := types.Sender(signer, tx)
		if err != nil || sender == block.Coinbase() {
			continue
		}
		cap := tx.FeeCap()
		if cap == nil {
			cap = tx.GasPrice()
		}
		ch <- getBlockCapsResult{cap, nil}
		return
	}
	ch <- getBlockCapsResult{nil, nil}
}

type bigIntArray []*big.Int

func (s bigIntArray) Len() int           { return len(s) }
func (s bigIntArray) Less(i, j int) bool { return s[i].Cmp(s[j]) < 0 }
func (s bigIntArray) Swap(i, j int)      { s[i], s[j] = s[j], s[i] }
