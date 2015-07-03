package eth

import (
	"math/big"
	"math/rand"
	"sync"

	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/event"
	"github.com/ethereum/go-ethereum/logger"
	"github.com/ethereum/go-ethereum/logger/glog"
)

const gpoProcessPastBlocks = 100

type blockPriceInfo struct {
	baseGasPrice *big.Int
}

type GasPriceOracle struct {
	eth                           *Ethereum
	chain                         *core.ChainManager
	pool                          *core.TxPool
	events                        event.Subscription
	blocks                        map[uint64]*blockPriceInfo
	firstProcessed, lastProcessed uint64
	lastBaseMutex                 sync.Mutex
	lastBase                      *big.Int
}

func NewGasPriceOracle(eth *Ethereum) (self *GasPriceOracle) {
	self = &GasPriceOracle{}
	self.blocks = make(map[uint64]*blockPriceInfo)
	self.eth = eth
	self.chain = eth.chainManager
	self.pool = eth.txPool
	self.events = eth.EventMux().Subscribe(
		core.ChainEvent{},
		core.ChainSplitEvent{},
		core.TxPreEvent{},
		core.TxPostEvent{},
	)
	self.processPastBlocks()
	go self.listenLoop()
	return
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
	for {
		ev, isopen := <-self.events.Chan()
		if !isopen {
			break
		}
		switch ev := ev.(type) {
		case core.ChainEvent:
			self.processBlock(ev.Block)
		case core.ChainSplitEvent:
			self.processBlock(ev.Block)
		case core.TxPreEvent:
		case core.TxPostEvent:
		}
	}
	self.events.Unsubscribe()
}

func (self *GasPriceOracle) processBlock(block *types.Block) {
	i := block.NumberU64()
	if i > self.lastProcessed {
		self.lastProcessed = i
	}

	lastBase := self.eth.GpoMinGasPrice
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
		corr = self.eth.GpobaseStepUp
	} else {
		corr = -self.eth.GpobaseStepDown
	}

	crand := int64(corr * (900 + rand.Intn(201)))
	newBase := new(big.Int).Mul(lastBase, big.NewInt(1000000+crand))
	newBase.Div(newBase, big.NewInt(1000000))

	bpi := self.blocks[i]
	if bpi == nil {
		bpi = &blockPriceInfo{}
		self.blocks[i] = bpi
	}
	bpi.baseGasPrice = newBase
	self.lastBaseMutex.Lock()
	self.lastBase = newBase
	self.lastBaseMutex.Unlock()

	glog.V(logger.Detail).Infof("Processed block #%v, base price is %v\n", block.NumberU64(), newBase.Int64())
}

// returns the lowers possible price with which a tx was or could have been included
func (self *GasPriceOracle) lowestPrice(block *types.Block) *big.Int {
	gasUsed := new(big.Int)
	recepits, err := self.eth.BlockProcessor().GetBlockReceipts(block.Hash())
	if err != nil {
		return self.eth.GpoMinGasPrice
	}

	if len(recepits) > 0 {
		gasUsed = recepits[len(recepits)-1].CumulativeGasUsed
	}

	if new(big.Int).Mul(gasUsed, big.NewInt(100)).Cmp(new(big.Int).Mul(block.GasLimit(),
		big.NewInt(int64(self.eth.GpoFullBlockRatio)))) < 0 {
		// block is not full, could have posted a tx with MinGasPrice
		return self.eth.GpoMinGasPrice
	}

	txs := block.Transactions()
	if len(txs) == 0 {
		return self.eth.GpoMinGasPrice
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

func (self *GasPriceOracle) SuggestPrice() *big.Int {
	self.lastBaseMutex.Lock()
	base := self.lastBase
	self.lastBaseMutex.Unlock()

	if base == nil {
		base = self.eth.GpoMinGasPrice
	}
	if base == nil {
		return big.NewInt(10000000000000) // apparently MinGasPrice is not initialized during some tests
	}

	baseCorr := new(big.Int).Mul(base, big.NewInt(int64(self.eth.GpobaseCorrectionFactor)))
	baseCorr.Div(baseCorr, big.NewInt(100))

	if baseCorr.Cmp(self.eth.GpoMinGasPrice) < 0 {
		return self.eth.GpoMinGasPrice
	}

	if baseCorr.Cmp(self.eth.GpoMaxGasPrice) > 0 {
		return self.eth.GpoMaxGasPrice
	}

	return baseCorr
}
