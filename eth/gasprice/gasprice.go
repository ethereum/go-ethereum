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
	"fmt"
	"math/big"
	"math/rand"
	"sync"

	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/event"
	"github.com/ethereum/go-ethereum/log"
)

const (
	gpoProcessPastBlocks = 100

	// for testing
	gpoDefaultBaseCorrectionFactor = 110
	gpoDefaultMinGasPrice          = 10000000000000
)

type blockPriceInfo struct {
	baseGasPrice *big.Int
}

type GpoParams struct {
	GpoMinGasPrice          *big.Int
	GpoMaxGasPrice          *big.Int
	GpoFullBlockRatio       int
	GpobaseStepDown         int
	GpobaseStepUp           int
	GpobaseCorrectionFactor int
}

// GasPriceOracle recommends gas prices based on the content of recent
// blocks.
type GasPriceOracle struct {
	chain         *core.BlockChain
	db            ethdb.Database
	evmux         *event.TypeMux
	params        *GpoParams
	initOnce      sync.Once
	minPrice      *big.Int
	lastBaseMutex sync.Mutex
	lastBase      *big.Int

	// state of listenLoop
	blocks                        map[uint64]*blockPriceInfo
	firstProcessed, lastProcessed uint64
	minBase                       *big.Int
}

// NewGasPriceOracle returns a new oracle.
func NewGasPriceOracle(chain *core.BlockChain, db ethdb.Database, evmux *event.TypeMux, params *GpoParams) *GasPriceOracle {
	minprice := params.GpoMinGasPrice
	if minprice == nil {
		minprice = big.NewInt(gpoDefaultMinGasPrice)
	}
	minbase := new(big.Int).Mul(minprice, big.NewInt(100))
	if params.GpobaseCorrectionFactor > 0 {
		minbase = minbase.Div(minbase, big.NewInt(int64(params.GpobaseCorrectionFactor)))
	}
	return &GasPriceOracle{
		chain:    chain,
		db:       db,
		evmux:    evmux,
		params:   params,
		blocks:   make(map[uint64]*blockPriceInfo),
		minBase:  minbase,
		minPrice: minprice,
		lastBase: minprice,
	}
}

func (gpo *GasPriceOracle) init() {
	gpo.initOnce.Do(func() {
		gpo.processPastBlocks()
		go gpo.listenLoop()
	})
}

func (self *GasPriceOracle) processPastBlocks() {
	last := int64(-1)
	cblock := self.chain.CurrentBlock()
	if cblock != nil {
		last = int64(cblock.NumberU64())
	}
	first := int64(0)
	if last > gpoProcessPastBlocks {
		first = last - gpoProcessPastBlocks
	}
	self.firstProcessed = uint64(first)
	for i := first; i <= last; i++ {
		block := self.chain.GetBlockByNumber(uint64(i))
		if block != nil {
			self.processBlock(block)
		}
	}

}

func (self *GasPriceOracle) listenLoop() {
	events := self.evmux.Subscribe(core.ChainEvent{}, core.ChainSplitEvent{})
	defer events.Unsubscribe()

	for event := range events.Chan() {
		switch event := event.Data.(type) {
		case core.ChainEvent:
			self.processBlock(event.Block)
		case core.ChainSplitEvent:
			self.processBlock(event.Block)
		}
	}
}

func (self *GasPriceOracle) processBlock(block *types.Block) {
	i := block.NumberU64()
	if i > self.lastProcessed {
		self.lastProcessed = i
	}

	lastBase := self.minPrice
	bpl := self.blocks[i-1]
	if bpl != nil {
		lastBase = bpl.baseGasPrice
	}
	if lastBase == nil {
		return
	}

	var corr int
	lp := self.lowestPrice(block)
	if lp == nil {
		return
	}

	if lastBase.Cmp(lp) < 0 {
		corr = self.params.GpobaseStepUp
	} else {
		corr = -self.params.GpobaseStepDown
	}

	crand := int64(corr * (900 + rand.Intn(201)))
	newBase := new(big.Int).Mul(lastBase, big.NewInt(1000000+crand))
	newBase.Div(newBase, big.NewInt(1000000))

	if newBase.Cmp(self.minBase) < 0 {
		newBase = self.minBase
	}

	bpi := self.blocks[i]
	if bpi == nil {
		bpi = &blockPriceInfo{}
		self.blocks[i] = bpi
	}
	bpi.baseGasPrice = newBase
	self.lastBaseMutex.Lock()
	self.lastBase = newBase
	self.lastBaseMutex.Unlock()

	log.Trace(fmt.Sprintf("Processed block #%v, base price is %v\n", i, newBase.Int64()))
}

// returns the lowers possible price with which a tx was or could have been included
func (self *GasPriceOracle) lowestPrice(block *types.Block) *big.Int {
	gasUsed := big.NewInt(0)

	receipts := core.GetBlockReceipts(self.db, block.Hash(), block.NumberU64())
	if len(receipts) > 0 {
		if cgu := receipts[len(receipts)-1].CumulativeGasUsed; cgu != nil {
			gasUsed = receipts[len(receipts)-1].CumulativeGasUsed
		}
	}

	if new(big.Int).Mul(gasUsed, big.NewInt(100)).Cmp(new(big.Int).Mul(block.GasLimit(),
		big.NewInt(int64(self.params.GpoFullBlockRatio)))) < 0 {
		// block is not full, could have posted a tx with MinGasPrice
		return big.NewInt(0)
	}

	txs := block.Transactions()
	if len(txs) == 0 {
		return big.NewInt(0)
	}
	// block is full, find smallest gasPrice
	minPrice := txs[0].GasPrice()
	for i := 1; i < len(txs); i++ {
		price := txs[i].GasPrice()
		if price.Cmp(minPrice) < 0 {
			minPrice = price
		}
	}
	return minPrice
}

// SuggestPrice returns the recommended gas price.
func (self *GasPriceOracle) SuggestPrice() *big.Int {
	self.init()
	self.lastBaseMutex.Lock()
	price := new(big.Int).Set(self.lastBase)
	self.lastBaseMutex.Unlock()

	price.Mul(price, big.NewInt(int64(self.params.GpobaseCorrectionFactor)))
	price.Div(price, big.NewInt(100))
	if price.Cmp(self.minPrice) < 0 {
		price.Set(self.minPrice)
	} else if self.params.GpoMaxGasPrice != nil && price.Cmp(self.params.GpoMaxGasPrice) > 0 {
		price.Set(self.params.GpoMaxGasPrice)
	}
	return price
}
