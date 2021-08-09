type AddTransactionsResultFunc func(error, []*types.Receipt) bool

// BlockState represents a block-to-be-mined, which is being assembled.
// A collator can add transactions by calling AddTransactions
type BlockState interface {
	// AddTransactions adds the sequence of transactions to the blockstate. Either all
	// transactions are added, or none of them. In the latter case, the error
	// describes the reason why the txs could not be included.
	// if ErrRecommit, the collator should not attempt to add more transactions to the
	// block and submit the block for sealing.
	// If ErrAbort is returned, the collator should immediately abort and return a
	// value (true) from CollateBlock which indicates to the miner to discard the
	// block
	AddTransactions(sequence types.Transactions, cb AddTransactionsResultFunc)
	Gas() (remaining uint64)
	Coinbase() common.Address
	BaseFee() *big.Int
	Signer() types.Signer
}

type collatorWork {
    env *environment
    counter uint64
    interrupt *int32
}

type collatorBlockState struct {
    work collatorWork
    c *collator
    done bool
}

func (w *collatorWork) Copy() collatorWork {
    newEnv := w.copy()
    return collatorWork{
        env: newEnv,
        counter: w.counter,
        interrupt: w.interrupt,
    }
}

func (bs *collatorBlockState) AddTransactions(sequence types.Transactions, cb AddTransactionsResultFunc) {
	var (
		interrupt   = bs.work.interrupt
		header = bs.work.env.header
        gasPool = bs.work.env.gasPool
        signer = bs.work.env.signer
        chainConfig = bs.c.chainConfig
        chain = bs.c.chain
        state = bs.work.env.state
		snap        = state.Snapshot()
// ---------------
		w           = bs.worker
		err         error
		logs        []*types.Log
		tcount      = w.current.tcount
		startTCount = w.current.tcount
	)
	if bs.done {
		cb(ErrRecommit, nil)
		return
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
		if tx.Protected() && !w.chainConfig.IsEIP155(header.Number) {
			log.Trace("encountered replay-protected transaction when chain doesn't support replay protection", "hash", tx.Hash(), "eip155", w.chainConfig.EIP155Block)
			err = ErrUnsupportedEIP155Tx
			break
		}
		// Start executing the transaction
		state.Prepare(tx.Hash(), w.current.tcount)

		var txLogs []*types.Log
		txLogs, err = commitTransaction(chain, chainConfig, tx, bs.Coinbase())
		if err == nil {
			//logs = append(logs, txLogs...)
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
		//bs.logs = append(bs.logs, logs...)
		w.current.tcount = tcount
	}
}

func (bs *collatorBlockState) Commit() {
    bs.done = true
    bs.c.workResultch <- bs.work
}

type collator struct {
    newWorkCh chan<- collatorWork
    workResultCh chan<-collatorWork
    // channel signalling collator loop should exit
    exitCh chan<-interface{}
    newHeadCh chan<-types.Header
    collateBlockImpl BlockCollator
    chainConfig *params.ChainConfig
    chain *core.BlockChain
}

func (c *collator) mainLoop() {
    for {
        select {
        case newWork := <-c.newWorkCh:
            // pass a wrapped CollatorBlockState object to the collator
            // implementation.  collator calls Commit() to flush new work to the
            // result channel.
            c.collateBlockImpl(collatorBlockState{newWork, c})

            // signal to the exitCh that the collator is done
            // computing this work.
            c.workResultCh <- collatorWork{nil, newWork.counter}
        case <-c.exitCh:
            // TODO any cleanup needed?
            return
        case newHead := <-newHeadCh:
            // TODO call hook here
            fallthrough
        default:
            fallthrough
        }
    }
}

type MultiCollator struct {
    workResultCh
    counter uint64
    responsiveCollatorCount uint
    collators []collator
}

func (m *MultiCollator) Start() {
    for c := range m.collators {
        go c.mainLoop()
    }
}

func (m *MultiCollator) Stop() {
    for c := range m.collators {
        select {
        case c.exitCh<-true:
            fallthrough
        default:
            continue
        }
    }
}

func (m *MultiCollator) CollateBlock(work *environment, interrupt *int32) {
    if m.counter == math.Uint64Max {
        m.counter = 0
    } else {
        m.counter++
    }
    m.responsiveCollatorCount = 0
    m.interrupt = interrupt
    for c := range m.collators {
        select {
        case c.newWorkCh <- collatorWork{env: work.copy(), counter: m.counter, interrupt: interrupt}
            m.responsiveCollatorCount++
        }
    }
}

type WorkResult func(environment)

func (m *MultiCollator) Collect(work *environment, cb WorkResult) {
    finishedCollators := []uint{}
    shouldAdjustRecommitDown := true

    for {
        if finishedCollators == m.responsiveCollatorcount {
            break
        }

        if interrupt != nil && atomic.LoadInt32(interrupt) != commitInterruptNone {
            // TODO: if the interrupt was recommit, signal to the worker to adjust up
            shouldAdjustRecommitDown = false
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
                fallthrough
            }
        }
    }

    if shouldAdjustRecommitDown {
        // TODO signal to worker to adjust recommit interval down
    }
}

/*
TODO implement and hook these into the miner
func (m *MultiCollator) NewHeadHook() {

}

func (m *Multicollator) SideChainHook() {

}
/*
