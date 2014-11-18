/*
	This file is part of go-ethereum

	go-ethereum is free software: you can redistribute it and/or modify
	it under the terms of the GNU Lesser General Public License as published by
	the Free Software Foundation, either version 3 of the License, or
	(at your option) any later version.

	go-ethereum is distributed in the hope that it will be useful,
	but WITHOUT ANY WARRANTY; without even the implied warranty of
	MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
	GNU General Public License for more details.

	You should have received a copy of the GNU Lesser General Public License
	along with go-ethereum.  If not, see <http://www.gnu.org/licenses/>.
*/
/**
 * @authors
 * 	Jeffrey Wilcke <i@jev.io>
 * @date 2014
 *
 */

package miner

import (
	"math/big"
	"sort"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/ethutil"

	"github.com/ethereum/go-ethereum/chain"
	"github.com/ethereum/go-ethereum/chain/types"
	"github.com/ethereum/go-ethereum/event"
	"github.com/ethereum/go-ethereum/logger"
	"github.com/ethereum/go-ethereum/wire"
)

type LocalTx struct {
	To       []byte `json:"to"`
	Data     []byte `json:"data"`
	Gas      string `json:"gas"`
	GasPrice string `json:"gasPrice"`
	Value    string `json:"value"`
}

func (self *LocalTx) Sign(key []byte) *types.Transaction {
	return nil
}

var minerlogger = logger.NewLogger("MINER")

type Miner struct {
	eth    *eth.Ethereum
	events event.Subscription

	uncles    types.Blocks
	localTxs  map[int]*LocalTx
	localTxId int

	pow       chain.PoW
	quitCh    chan struct{}
	powQuitCh chan struct{}

	Coinbase []byte

	mining bool

	MinAcceptedGasPrice *big.Int
}

func New(coinbase []byte, eth *eth.Ethereum) *Miner {
	return &Miner{
		eth:                 eth,
		powQuitCh:           make(chan struct{}),
		pow:                 &chain.EasyPow{},
		mining:              false,
		localTxs:            make(map[int]*LocalTx),
		MinAcceptedGasPrice: big.NewInt(10000000000000),
		Coinbase:            coinbase,
	}
}

func (self *Miner) GetPow() chain.PoW {
	return self.pow
}

func (self *Miner) AddLocalTx(tx *LocalTx) int {
	minerlogger.Infof("Added local tx (%x %v / %v)\n", tx.To[0:4], tx.GasPrice, tx.Value)

	self.localTxId++
	self.localTxs[self.localTxId] = tx
	self.eth.EventMux().Post(tx)

	return self.localTxId
}

func (self *Miner) RemoveLocalTx(id int) {
	if tx := self.localTxs[id]; tx != nil {
		minerlogger.Infof("Removed local tx (%x %v / %v)\n", tx.To[0:4], tx.GasPrice, tx.Value)
	}
	self.eth.EventMux().Post(&LocalTx{})

	delete(self.localTxs, id)
}

func (self *Miner) Start() {
	if self.mining {
		return
	}

	minerlogger.Infoln("Starting mining operations")
	self.mining = true
	self.quitCh = make(chan struct{})
	self.powQuitCh = make(chan struct{})

	mux := self.eth.EventMux()
	self.events = mux.Subscribe(chain.NewBlockEvent{}, chain.TxPreEvent{}, &LocalTx{})

	go self.update()
	go self.mine()
}

func (self *Miner) Stop() {
	if !self.mining {
		return
	}

	self.mining = false

	minerlogger.Infoln("Stopping mining operations")

	self.events.Unsubscribe()

	close(self.quitCh)
	close(self.powQuitCh)
}

func (self *Miner) Mining() bool {
	return self.mining
}

func (self *Miner) update() {
out:
	for {
		select {
		case event := <-self.events.Chan():
			switch event := event.(type) {
			case chain.NewBlockEvent:
				block := event.Block
				if self.eth.ChainManager().HasBlock(block.Hash()) {
					self.reset()
					self.eth.TxPool().RemoveSet(block.Transactions())
					go self.mine()
				} else if true {
					// do uncle stuff
				}
			case chain.TxPreEvent, *LocalTx:
				self.reset()
				go self.mine()
			}
		case <-self.quitCh:
			break out
		}
	}
}

func (self *Miner) reset() {
	println("reset")
	close(self.powQuitCh)
	self.powQuitCh = make(chan struct{})
}

func (self *Miner) mine() {
	var (
		blockManager = self.eth.BlockManager()
		chainMan     = self.eth.ChainManager()
		block        = chainMan.NewBlock(self.Coinbase)
	)
	block.MinGasPrice = self.MinAcceptedGasPrice

	// Apply uncles
	if len(self.uncles) > 0 {
		block.SetUncles(self.uncles)
	}

	parent := chainMan.GetBlock(block.PrevHash)
	coinbase := block.State().GetOrNewStateObject(block.Coinbase)
	coinbase.SetGasPool(block.CalcGasLimit(parent))

	transactions := self.finiliseTxs()

	// Accumulate all valid transactions and apply them to the new state
	// Error may be ignored. It's not important during mining
	receipts, txs, _, erroneous, err := blockManager.ProcessTransactions(coinbase, block.State(), block, block, transactions)
	if err != nil {
		minerlogger.Debugln(err)
	}
	self.eth.TxPool().RemoveSet(erroneous)

	block.SetTransactions(txs)
	block.SetReceipts(receipts)

	// Accumulate the rewards included for this block
	blockManager.AccumelateRewards(block.State(), block, parent)

	block.State().Update()

	minerlogger.Infof("Mining on block. Includes %v transactions", len(transactions))

	// Find a valid nonce
	nonce := self.pow.Search(block, self.powQuitCh)
	if nonce != nil {
		block.Nonce = nonce
		lchain := chain.NewChain(types.Blocks{block})
		_, err := chainMan.TestChain(lchain)
		if err != nil {
			minerlogger.Infoln(err)
		} else {
			chainMan.InsertChain(lchain)
			//self.eth.EventMux().Post(chain.NewBlockEvent{block})
			self.eth.Broadcast(wire.MsgBlockTy, []interface{}{block.Value().Val})

			minerlogger.Infof("🔨  Mined block %x\n", block.Hash())
			minerlogger.Infoln(block)
		}

		go self.mine()
	}
}

func (self *Miner) finiliseTxs() types.Transactions {
	// Sort the transactions by nonce in case of odd network propagation
	var txs types.Transactions

	state := self.eth.BlockManager().TransState()
	// XXX This has to change. Coinbase is, for new, same as key.
	key := self.eth.KeyManager()
	for _, ltx := range self.localTxs {
		tx := types.NewTransactionMessage(ltx.To, ethutil.Big(ltx.Value), ethutil.Big(ltx.Gas), ethutil.Big(ltx.GasPrice), ltx.Data)
		tx.Nonce = state.GetNonce(self.Coinbase)
		state.SetNonce(self.Coinbase, tx.Nonce+1)

		tx.Sign(key.PrivateKey())

		txs = append(txs, tx)
	}

	txs = append(txs, self.eth.TxPool().CurrentTransactions()...)
	sort.Sort(types.TxByNonce{txs})

	return txs
}
