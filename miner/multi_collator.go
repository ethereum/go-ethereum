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

package miner

import (
	"errors"
    "math"
	"math/big"
	"sync/atomic"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"
)

// called by AddTransactions.  If the provided error is not nil, none of the trasactions were added.
// If the error is nil, the receipts of all executed transactions are provided
type AddTransactionsResultFunc func(error, *types.Receipt) bool

// BlockState represents a block-to-be-mined, which is being assembled.
// A collator can add transactions by calling AddTransactions.
// When the collator is done adding transactions to a block, it calls Commit.
// after-which, no more transactions can be added to the block
type BlockState interface {
	// AddTransactions adds the sequence of transactions to the blockstate. Either all
	// transactions are added, or none of them. In the latter case, the error
	// describes the reason why the txs could not be included.
	// If all transactions were successfully executed, the return value of the callback
	// determines if the transactions are kept (true) or all reverted (false)
	AddTransactions(sequence types.Transactions, cb AddTransactionsResultFunc)
	// Commit is called when the collator is done adding transactions to a block
	// and wants to suggest it for sealing
	Commit()
	// deep copy of a blockState
	Copy() BlockState
	Gas() (remaining uint64)
	Coinbase() common.Address
	BaseFee() *big.Int
	Signer() types.Signer
    Profit() *big.Int
}

// collatorWork is provided by the CollatorPool to each collator goroutine
// when new work is being generated
type collatorWork struct {
	env       *environment
	counter   uint64
	interrupt *int32
}

func (w *collatorWork) Copy() collatorWork {
	newEnv := w.env.copy()
	return collatorWork{
		env:       newEnv,
		counter:   w.counter,
		interrupt: w.interrupt,
	}
}

// Pool is an interface to the transaction pool
type Pool interface {
	Pending(bool) (map[common.Address]types.Transactions, error)
	Locals() []common.Address
}

var (
	ErrAbort               = errors.New("abort sealing current work (resubmit/newHead interrupt)")
	ErrUnsupportedEIP155Tx = errors.New("replay-protected tx when EIP155 not enabled")
)

// collatorBlockState is an implementation of BlockState
type collatorBlockState struct {
	work collatorWork
	c    *collator
	done bool
}

func (c *collatorBlockState) Copy() BlockState {
	return &collatorBlockState{
		work: c.work.Copy(),
		c:    c.c,
		done: c.done,
	}
}

func (bs *collatorBlockState) AddTransactions(sequence types.Transactions, cb AddTransactionsResultFunc) {
	var (
		interrupt             = bs.work.interrupt
		header                = bs.work.env.header
		gasPool               = bs.work.env.gasPool
		signer                = bs.work.env.signer
		chainConfig           = bs.c.chainConfig
		chain                 = bs.c.chain
		state                 = bs.work.env.state
		snap                  = state.Snapshot()
		curProfit             = new(big.Int).Set(bs.work.env.profit)
        startProfit           = new(big.Int).Set(bs.work.env.profit)
		tcount                = bs.work.env.tcount
		err                   error
		logs                  []*types.Log
		startTCount           = bs.work.env.tcount
        shouldRevert bool
	)
	if bs.done {
		err = ErrAbort
		cb(err, nil)
		return
	}

	for _, tx := range sequence {
		if interrupt != nil && atomic.LoadInt32(interrupt) != commitInterruptNone {
			bs.done = true
            shouldRevert = true
            cb(ErrAbort, nil)
			break
		}
		if gasPool.Gas() < params.TxGas {
			log.Trace("Not enough gas for further transactions", "have", gasPool, "want", params.TxGas)
			err = core.ErrGasLimitReached
            cb(err, nil)
            shouldRevert = true
			break
		}
		from, _ := types.Sender(signer, tx)
		// Check whether the tx is replay protected. If we're not in the EIP155 hf
		// phase, start ignoring the sender until we do.
		if tx.Protected() && !chainConfig.IsEIP155(header.Number) {
			log.Trace("encountered replay-protected transaction when chain doesn't support replay protection", "hash", tx.Hash(), "eip155", chainConfig.EIP155Block)
			err = ErrUnsupportedEIP155Tx
            cb(err, nil)
			break
		}
		gasPrice, err := tx.EffectiveGasTip(bs.work.env.header.BaseFee)
		if err != nil {
            shouldRevert = true
            cb(err, nil)
			break
		}
		// Start executing the transaction
		state.Prepare(tx.Hash(), tcount)

		var txLogs []*types.Log
		txLogs, err = commitTransaction(chain, chainConfig, bs.work.env, tx, bs.Coinbase())
		if err == nil {
			logs = append(logs, txLogs...)
			gasUsed := new(big.Int).SetUint64(bs.work.env.receipts[len(bs.work.env.receipts)-1].GasUsed)
            // TODO remove this allocation once things are working
			curProfit.Add(curProfit, gasUsed.Mul(gasUsed, gasPrice))
            bs.work.env.profit.Set(curProfit)
            if cb(nil, bs.work.env.receipts[len(bs.work.env.receipts) - 1]) {
                shouldRevert = true
                break
            } else {
			    tcount++
            }
		} else {
            cb(err, nil)
            shouldRevert = true
			log.Trace("Tx block inclusion failed", "sender", from, "nonce", tx.Nonce(),
				"type", tx.Type(), "hash", tx.Hash(), "err", err)
			break
		}
	}
	if shouldRevert {
		state.RevertToSnapshot(snap)
		bs.work.env.txs = bs.work.env.txs[:startTCount]
		bs.work.env.receipts = bs.work.env.receipts[:startTCount]
		bs.work.env.profit.Set(startProfit)
	} else {
		bs.work.env.logs = append(bs.work.env.logs, logs...)
		bs.work.env.tcount = tcount
	}
	return
}

func (bs *collatorBlockState) Commit() {
	if !bs.done {
		bs.done = true
		bs.c.workResultCh <- bs.work
	}
}

func (bs *collatorBlockState) Gas() uint64 {
	return bs.work.env.gasPool.Gas()
}

func (bs *collatorBlockState) Coinbase() common.Address {
	resultCopy := common.Address{}
	copy(resultCopy[:], bs.work.env.coinbase[:])
	return resultCopy
}

func (bs *collatorBlockState) BaseFee() *big.Int {
	return new(big.Int).Set(bs.work.env.header.BaseFee)
}

func (bs *collatorBlockState) Signer() types.Signer {
	return bs.work.env.signer
}

func (bs *collatorBlockState) Profit() *big.Int {
    return new(big.Int).Set(bs.work.env.profit)
}

// BlockCollator is the publicly-exposed interface
// for implementing custom block collation strategies
type BlockCollator interface {
	CollateBlock(bs BlockState, pool Pool)
	/*
	   // TODO implement these
	   SideChainHook(header *types.Header)
	   NewHeadHook(header *types.Header)
	*/
}

type collator struct {
	newWorkCh    chan collatorWork
	workResultCh chan collatorWork
	// channel signalling collator loop should exit
	exitCh            chan struct{}
	newHeadCh         chan types.Header
	sideChainCh       chan types.Header
	blockCollatorImpl BlockCollator

	chainConfig *params.ChainConfig
	chain       *core.BlockChain
	pool        Pool
}

// each active collator runs mainLoop() in a goroutine.
// It receives new work from the miner and listens for new blocks built from CollateBlock
// calling Commit() on the provided collatorBlockState
func (c *collator) mainLoop() {
	for {
		select {
		case newWork := <-c.newWorkCh:
			c.blockCollatorImpl.CollateBlock(&collatorBlockState{work: newWork, c: c, done: false}, c.pool)
			// signal to the exitCh that the collator is done
			// computing this work.
			c.workResultCh <- collatorWork{env: nil, interrupt: nil, counter: newWork.counter}
		case <-c.exitCh:
			// TODO any cleanup needed?
			return
		case newHead := <-c.newHeadCh:
			// TODO call hook here
			_ = newHead
		case sideHeader := <-c.sideChainCh:
			_ = sideHeader
			// TODO call hook here
		default:
		}
	}
}

var (
	// collator
	workResultChSize = 10
	newWorkChSize    = 10
	newHeadChSize    = 10
	sideChainChSize  = 10
)

// MultiCollator manages multiple active collators
type MultiCollator struct {
	counter                 uint64
	responsiveCollatorCount int
	collators               []collator
	pool                    Pool
	interrupt               *int32
}

func NewMultiCollator(chainConfig *params.ChainConfig, chain *core.BlockChain, pool Pool, strategies []BlockCollator) MultiCollator {
	collators := []collator{}
	for _, s := range strategies {
		collators = append(collators, collator{
			newWorkCh:         make(chan collatorWork, newWorkChSize),
			workResultCh:      make(chan collatorWork, workResultChSize),
			exitCh:            make(chan struct{}),
			newHeadCh:         make(chan types.Header, newHeadChSize),
			blockCollatorImpl: s,
			chainConfig:       chainConfig,
			chain:             chain,
			pool:              pool,
		})
	}
	return MultiCollator{
		counter:                 0,
		responsiveCollatorCount: 0,
		collators:               collators,
		interrupt:               nil,
	}
}

func (m *MultiCollator) Start() {
	for _, c := range m.collators {
		go c.mainLoop()
	}
}

func (m *MultiCollator) Close() {
	for _, c := range m.collators {
		select {
		case c.exitCh <- struct{}{}:
		default:
		}
	}
}

// SuggestBlock sends a new empty block to each active collator.
// collators whose receiving channels are full are noted as "unresponsive"
// for the purpose of not expecting a response back (for this round) during
// polling performed by Collect
func (m *MultiCollator) SuggestBlock(work *environment, interrupt *int32) {
	if m.counter == math.MaxUint64 {
		m.counter = 0
	} else {
		m.counter++
	}
	m.responsiveCollatorCount = 0
	m.interrupt = interrupt
	for _, c := range m.collators {
		select {
		case c.newWorkCh <- collatorWork{env: work.copy(), counter: m.counter, interrupt: interrupt}:
			m.responsiveCollatorCount++
        default:
		}
	}
}

type WorkResult func(environment)

// Collect retrieves filled blocks returned by active collators in response to the block suggested by the previous call to SuggestBlock.
// It blocks until all responsive collators (ones which accepted the block from SuggestBlock) signal that they are done
// or the interrupt provided in SetBlock is set.
func (m *MultiCollator) Collect(cb WorkResult) {
	finishedCollators := []int{}
	for {
		if len(finishedCollators) == m.responsiveCollatorCount {
			break
		}
		if m.interrupt != nil && atomic.LoadInt32(m.interrupt) != commitInterruptNone {
			break
		}
		for i, c := range m.collators {
			select {
			case response := <-c.workResultCh:
				// ignore collators responding from old work rounds
				if response.counter != m.counter {
					break
				}
				// ignore responses from collators that have already signalled they are done
				shouldIgnore := false
				for _, finishedCollator := range finishedCollators {
					if i == finishedCollator {
						shouldIgnore = true
                        break
					}
				}
				if shouldIgnore {
					break
				}
				// nil for work signals the collator won't send back any more blocks for this round
				if response.env == nil {
					finishedCollators = append(finishedCollators, i)
				} else {
					cb(*response.env)
				}
			default:
			}
		}
	}
	return
}

/*
TODO implement and hook these into the miner
func (m *MultiCollator) NewHeadHook() {

}

func (m *Multicollator) SideChainHook() {

}
*/
