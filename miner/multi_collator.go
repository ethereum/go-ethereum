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
	"math/big"
    "sync/atomic"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/log"
)
type AddTransactionsResultFunc func(error, []*types.Receipt) bool

// BlockState represents a block-to-be-mined, which is being assembled.
// A collator can add transactions by calling AddTransactions.
// When the collator is done adding transactions to a block, it calls Commit.
// after-which, no more transactions can be added to the block
type BlockState interface {
	// AddTransactions adds the sequence of transactions to the blockstate. Either all
	// transactions are added, or none of them. In the latter case, the error
	// describes the reason why the txs could not be included.
	// if ErrRecommit, the collator should not attempt to add more transactions to the
	// block and submit the block for sealing.
	// If ErrAbort is returned, the collator should immediately abort and return a
	// value (true) from CollateBlock which indicates to the miner to discard the
	// block
	AddTransactions(sequence types.Transactions, cb AddTransactionsResultFunc) error
    Commit()
    Copy() BlockState
	Gas() (remaining uint64)
	Coinbase() common.Address
	BaseFee() *big.Int
	Signer() types.Signer
}

type collatorWork struct {
    env *environment
    counter uint64
    interrupt *int32
}

// Pool is an interface to the transaction pool
type Pool interface {
        Pending(bool) (map[common.Address]types.Transactions, error)
        Locals() []common.Address
}

var (
    ErrAbort = errors.New("abort sealing current work (resubmit/newHead interrupt)")
    ErrUnsupportedEIP155Tx = errors.New("replay-protected tx when EIP155 not enabled")
)

type collatorBlockState struct {
    work collatorWork
    c *collator
    done bool
}

func (w *collatorWork) Copy() collatorWork {
    newEnv := w.env.copy()
    return collatorWork{
        env: newEnv,
        counter: w.counter,
        interrupt: w.interrupt,
    }
}

func (bs *collatorBlockState) AddTransactions(sequence types.Transactions, cb AddTransactionsResultFunc) error {
	var (
		interrupt   = bs.work.interrupt
		header = bs.work.env.header
        gasPool = bs.work.env.gasPool
        signer = bs.work.env.signer
        chainConfig = bs.c.chainConfig
        chain = bs.c.chain
        state = bs.work.env.state
		snap        = state.Snapshot()
        curProfit = big.NewInt(0)
		coinbaseBalanceBefore = state.GetBalance(bs.work.env.header.Coinbase)
		tcount      = bs.work.env.tcount
		err         error
		logs        []*types.Log
		startTCount = bs.work.env.tcount
	)
	if bs.done {
		return ErrAbort
	}

	for _, tx := range sequence {
		if interrupt != nil && atomic.LoadInt32(interrupt) != commitInterruptNone {
            bs.done = true
			break
		}
		if gasPool.Gas() < params.TxGas {
			log.Trace("Not enough gas for further transactions", "have", gasPool, "want", params.TxGas)
			err = core.ErrGasLimitReached
			break
		}
		from, _ := types.Sender(signer, tx)
		// Check whether the tx is replay protected. If we're not in the EIP155 hf
		// phase, start ignoring the sender until we do.
		if tx.Protected() && !chainConfig.IsEIP155(header.Number) {
			log.Trace("encountered replay-protected transaction when chain doesn't support replay protection", "hash", tx.Hash(), "eip155", chainConfig.EIP155Block)
			err = ErrUnsupportedEIP155Tx
			break
		}
        gasPrice, err := tx.EffectiveGasTip(bs.work.env.header.BaseFee)
        if err != nil {
            break
        }
		// Start executing the transaction
		state.Prepare(tx.Hash(), bs.work.env.tcount)

		var txLogs []*types.Log
		txLogs, err = commitTransaction(chain, chainConfig, bs.work.env, tx, bs.Coinbase())
		if err == nil {
			logs = append(logs, txLogs...)
            gasUsed := new(big.Int).SetUint64(bs.work.env.receipts[len(bs.work.env.receipts) - 1].GasUsed)
            curProfit.Add(curProfit, gasUsed.Mul(gasUsed, gasPrice))
			tcount++
		} else {
			log.Trace("Tx block inclusion failed", "sender", from, "nonce", tx.Nonce(),
				"type", tx.Type(), "hash", tx.Hash(), "err", err)
			break
		}
	}
	var txReceipts []*types.Receipt = nil
	if err == nil {
		txReceipts = bs.work.env.receipts[startTCount:tcount]
	}
	// TODO: deep copy the tx receipts here or add a disclaimer to implementors not to modify them?
	shouldRevert := cb(err, txReceipts)

	if err != nil || shouldRevert {
		state.RevertToSnapshot(snap)

		// remove the txs and receipts that were added
		for i := startTCount; i < tcount; i++ {
			bs.work.env.txs[i] = nil
			bs.work.env.receipts[i] = nil
		}
		bs.work.env.txs = bs.work.env.txs[:startTCount]
		bs.work.env.receipts = bs.work.env.receipts[:startTCount]
	} else {
		coinbaseBalanceAfter := bs.work.env.state.GetBalance(bs.work.env.header.Coinbase)
        coinbaseTransfer := big.NewInt(0).Sub(coinbaseBalanceAfter, coinbaseBalanceBefore)
        curProfit.Add(curProfit, coinbaseTransfer)
		bs.work.env.logs = append(bs.work.env.logs, logs...)
        bs.env.profit = curProfit
		bs.env.tcount = tcount
	}
}

func (bs *collatorBlockState) Commit() {
    if !bs.done {
        bs.done = true
        bs.c.workResultCh <- bs.work
    }
}

func (bs *collatorBlockState) Gas() uint64 {
    return *bs.work.env.gasPool
}

func (bs *collatorBlockState) Coinbase() common.Address {
    // TODO should clone this but I'm feeling lazy rn
    return bs.work.env.header.Coinbase
}

func (bs *collatorBlockState) BaseFee() *big.Int {
    return new(big.Int).SetInt(bs.work.env.header.BaseFee)
}

func (bs *collatorBlockState) Signer() types.Signer {
    return bs.work.env.signer
}

type BlockCollator interface {
    CollateBlock(bs BlockState, pool Pool)
    SideChainHook(header *types.Header)
    NewHeadHook(header *types.Header)
}

type collator struct {
    newWorkCh chan<- collatorWork
    workResultCh chan<-collatorWork
    // channel signalling collator loop should exit
    exitCh chan<-interface{}
    newHeadCh chan<-types.Header
    sideChainCh chan<-types.Header
    blockCollatorImpl BlockCollator
    chainConfig *params.ChainConfig
    chain *core.BlockChain
}

// mainLoop runs in a separate goroutine and handles the lifecycle of an active collator.
// TODO more explanation
func (c *collator) mainLoop() {
    for {
        select {
        case newWork := <-c.newWorkCh:
            c.collateBlockImpl(collatorBlockState{work: newWork, c: c, done: false})
            // signal to the exitCh that the collator is done
            // computing this work.
            c.workResultCh <- collatorWork{nil, newWork.counter}
        case <-c.exitCh:
            // TODO any cleanup needed?
            return
        case newHead := <-c.newHeadCh:
            // TODO call hook here
        case sideHeader := <-c.sideChainCh:
            // TODO call hook here
        default:
        }
    }
}

var (
    // MultiCollator
    workResultChSize = 10

    // collator 
    newWorkChSize = 10
    newHeadChSize = 10
    sideChainChSize = 10
)

type MultiCollator struct {
    workResultCh chan<- collatorWork
    counter uint64
    responsiveCollatorCount uint
    collators []collator
}

func NewMultiCollator(chainConfig *params.ChainConfig, chain *core.BlockChain, strategies []BlockCollator) MultiCollator {
    workResultCh := make(chan collatorWork, workResultChSize)
    collators := []collator{}
    for _, s := range strategies {
        collators = append(collators, collator{
            newWorkCh: make(chan collatorWork, newWorkChSize),
            workResultCh: workResultCh,
            exitCh: make(chan struct{}),
            newHeadCh: make(chan types.Header, newHeadChSize),
            blockCollatorImpl: s,
            chainConfig: chainConfig,
            chain: chain,
        })
    }

    m := MultiCollator {
        counter: 0,
        responsiveCollatorCount: 0,
        collators: collators,
        workResultCh: workResultCh,
    }
}

func (m *MultiCollator) Start() {
    for c := range m.collators {
        go c.mainLoop()
    }
}

func (m *MultiCollator) Close() {
    for c := range m.collators {
        select {
        case c.exitCh<-true:
        default:
        }
    }
}

// SuggestBlock sends a new empty block to each active collator.
// collators whose receiving channels are full are noted as "unresponsive"
// for the purpose of not expecting a response back (for this round) during 
// polling performed by Collect
func (m *MultiCollator) SuggestBlock(work *environment, interrupt *int32) {
    if m.counter == math.Uint64Max {
        m.counter = 0
    } else {
        m.counter++
    }
    m.responsiveCollatorCount = 0
    m.interrupt = interrupt
    for c := range m.collators {
        select {
        case c.newWorkCh <- collatorWork{env: work.copy(), counter: m.counter, interrupt: interrupt}:
            m.responsiveCollatorCount++
        }
    }
}

type WorkResult func(environment)

// Collect retrieves filled blocks returned by active collators based on the block suggested by the previous call to SuggestedBlock.
// It blocks until all responsive collators (ones which accepted the block from SuggestBlock) signal that they are done
// or the provided interrupt is set.
func (m *MultiCollator) Collect(cb WorkResult) {
    finishedCollators := []uint{}
    for {
        if finishedCollators == m.responsiveCollatorcount {
            break
        }
        if m.interrupt != nil && atomic.LoadInt32(m.interrupt) != commitInterruptNone {
            break
        }
        for i, c := range m.collators {
            select {
            case response := m.workResultCh:
                // ignore collators responding from old work rounds
                if response.counter != m.counter {
                    break
                }
                // ignore responses from collators that have already signalled they are done
                shouldIgnore := false
                for _, finishedCollator := range finishedCollators {
                    if i == finishedCollator {
                        shouldIgnore = true
                    }
                }
                if shouldIgnore {
                    break
                }
                // nil for work signals the collator won't send back any more blocks for this round
                if response.work == nil {
                    finishedCollators = append(finishedCollators, i)
                } else {
                    cb(response.work)
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
