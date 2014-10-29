package ethminer

import (
	"bytes"
	"sort"

	"github.com/ethereum/go-ethereum/ethchain"
	"github.com/ethereum/go-ethereum/ethlog"
	"github.com/ethereum/go-ethereum/ethwire"
	"github.com/ethereum/go-ethereum/event"
)

var logger = ethlog.NewLogger("MINER")

type Miner struct {
	pow      ethchain.PoW
	ethereum ethchain.EthManager
	coinbase []byte
	txs      ethchain.Transactions
	uncles   []*ethchain.Block
	block    *ethchain.Block

	events      event.Subscription
	powQuitChan chan struct{}
	powDone     chan struct{}

	turbo bool
}

const (
	Started = iota
	Stopped
)

type Event struct {
	Type  int // Started || Stopped
	Miner *Miner
}

func (self *Miner) GetPow() ethchain.PoW {
	return self.pow
}

func NewDefaultMiner(coinbase []byte, ethereum ethchain.EthManager) *Miner {
	miner := Miner{
		pow:      &ethchain.EasyPow{},
		ethereum: ethereum,
		coinbase: coinbase,
	}

	return &miner
}

func (self *Miner) ToggleTurbo() {
	self.turbo = !self.turbo

	self.pow.Turbo(self.turbo)
}

func (miner *Miner) Start() {

	// Insert initial TXs in our little miner 'pool'
	miner.txs = miner.ethereum.TxPool().Flush()
	miner.block = miner.ethereum.ChainManager().NewBlock(miner.coinbase)

	mux := miner.ethereum.EventMux()
	miner.events = mux.Subscribe(ethchain.NewBlockEvent{}, ethchain.TxEvent{})

	// Prepare inital block
	//miner.ethereum.StateManager().Prepare(miner.block.State(), miner.block.State())
	go miner.listener()

	logger.Infoln("Started")
	mux.Post(Event{Started, miner})
}

func (miner *Miner) Stop() {
	logger.Infoln("Stopping...")
	miner.events.Unsubscribe()
	miner.ethereum.EventMux().Post(Event{Stopped, miner})
}

func (miner *Miner) listener() {
	miner.startMining()

	for {
		select {
		case event, isopen := <-miner.events.Chan():
			if !isopen {
				return
			}

			switch event := event.(type) {
			case ethchain.NewBlockEvent:
				miner.stopMining()

				block := event.Block
				//logger.Infoln("Got new block via Reactor")
				if bytes.Compare(miner.ethereum.ChainManager().CurrentBlock.Hash(), block.Hash()) == 0 {
					// TODO: Perhaps continue mining to get some uncle rewards
					//logger.Infoln("New top block found resetting state")

					// Filter out which Transactions we have that were not in this block
					var newtxs []*ethchain.Transaction
					for _, tx := range miner.txs {
						found := false
						for _, othertx := range block.Transactions() {
							if bytes.Compare(tx.Hash(), othertx.Hash()) == 0 {
								found = true
							}
						}
						if found == false {
							newtxs = append(newtxs, tx)
						}
					}
					miner.txs = newtxs

					// Setup a fresh state to mine on
					//miner.block = miner.ethereum.ChainManager().NewBlock(miner.coinbase, miner.txs)

				} else {
					if bytes.Compare(block.PrevHash, miner.ethereum.ChainManager().CurrentBlock.PrevHash) == 0 {
						logger.Infoln("Adding uncle block")
						miner.uncles = append(miner.uncles, block)
					}
				}

			case ethchain.TxEvent:
				if event.Type == ethchain.TxPre {
					miner.stopMining()

					found := false
					for _, ctx := range miner.txs {
						if found = bytes.Compare(ctx.Hash(), event.Tx.Hash()) == 0; found {
							break
						}
					}
					if found == false {
						// Undo all previous commits
						miner.block.Undo()
						// Apply new transactions
						miner.txs = append(miner.txs, event.Tx)
					}
				}
			}

		case <-miner.powDone:
			miner.startMining()
		}
	}
}

func (miner *Miner) startMining() {
	if miner.powDone == nil {
		miner.powDone = make(chan struct{})
	}
	miner.powQuitChan = make(chan struct{})
	go miner.mineNewBlock()
}

func (miner *Miner) stopMining() {
	close(miner.powQuitChan)
	<-miner.powDone
}

func (self *Miner) mineNewBlock() {
	stateManager := self.ethereum.StateManager()

	self.block = self.ethereum.ChainManager().NewBlock(self.coinbase)

	// Apply uncles
	if len(self.uncles) > 0 {
		self.block.SetUncles(self.uncles)
	}

	// Sort the transactions by nonce in case of odd network propagation
	sort.Sort(ethchain.TxByNonce{self.txs})

	// Accumulate all valid transactions and apply them to the new state
	// Error may be ignored. It's not important during mining
	parent := self.ethereum.ChainManager().GetBlock(self.block.PrevHash)
	coinbase := self.block.State().GetOrNewStateObject(self.block.Coinbase)
	coinbase.SetGasPool(self.block.CalcGasLimit(parent))
	receipts, txs, unhandledTxs, erroneous, err := stateManager.ProcessTransactions(coinbase, self.block.State(), self.block, self.block, self.txs)
	if err != nil {
		logger.Debugln(err)
	}
	self.ethereum.TxPool().RemoveSet(erroneous)
	self.txs = append(txs, unhandledTxs...)
	self.block.SetReceiptHash(receipts)

	// Set the transactions to the block so the new SHA3 can be calculated
	self.block.SetReceipts(receipts, txs)

	// Accumulate the rewards included for this block
	stateManager.AccumelateRewards(self.block.State(), self.block, parent)

	self.block.State().Update()

	logger.Infof("Mining on block. Includes %v transactions", len(self.txs))

	// Find a valid nonce
	nonce := self.pow.Search(self.block, self.powQuitChan)
	if nonce != nil {
		self.block.Nonce = nonce
		err := self.ethereum.StateManager().Process(self.block, false)
		if err != nil {
			logger.Infoln(err)
		} else {
			self.ethereum.Broadcast(ethwire.MsgBlockTy, []interface{}{self.block.Value().Val})
			logger.Infof("🔨  Mined block %x\n", self.block.Hash())
			logger.Infoln(self.block)
			// Gather the new batch of transactions currently in the tx pool
			self.txs = self.ethereum.TxPool().CurrentTransactions()
		}
	}
	self.powDone <- struct{}{}
}
