package dashboard

import (
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/event"
	"github.com/ethereum/go-ethereum/log"
)

type block struct {
	Number int64  `json:"number,omitempty"`
	Time   uint64 `json:"timestamp,omitempty"`
}

func (db *Dashboard) collectChainData() {
	defer db.wg.Done()

	var (
		currentBlock *block
		chainCh      chan core.ChainHeadEvent
		chainSub     event.Subscription
	)
	switch {
	case db.ethServ != nil:
		chain := db.ethServ.BlockChain()
		currentBlock = &block{
			Number: chain.CurrentHeader().Number.Int64(),
			Time:   chain.CurrentHeader().Time,
		}
		chainCh = make(chan core.ChainHeadEvent)
		chainSub = chain.SubscribeChainHeadEvent(chainCh)
	case db.lesServ != nil:
		chain := db.lesServ.BlockChain()
		currentBlock = &block{
			Number: chain.CurrentHeader().Number.Int64(),
			Time:   chain.CurrentHeader().Time,
		}
		chainCh = make(chan core.ChainHeadEvent)
		chainSub = chain.SubscribeChainHeadEvent(chainCh)
	default:
		errc := <-db.quit
		errc <- nil
		return
	}
	defer chainSub.Unsubscribe()

	db.chainLock.Lock()
	db.history.Chain = &ChainMessage{
		CurrentBlock: currentBlock,
	}
	db.chainLock.Unlock()
	db.sendToAll(&Message{Chain: &ChainMessage{CurrentBlock: currentBlock}})

	for {
		select {
		case e := <-chainCh:
			currentBlock := &block{
				Number: e.Block.Number().Int64(),
				Time:   e.Block.Time(),
			}
			db.chainLock.Lock()
			db.history.Chain = &ChainMessage{
				CurrentBlock: currentBlock,
			}
			db.chainLock.Unlock()

			db.sendToAll(&Message{Chain: &ChainMessage{CurrentBlock: currentBlock}})
		case err := <-chainSub.Err():
			log.Warn("Chain subscription error", "err", err)
			errc := <-db.quit
			errc <- nil
			return
		case errc := <-db.quit:
			errc <- nil
			return
		}
	}
}
