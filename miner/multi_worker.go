package miner

import (
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/consensus"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/event"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"
)

type multiWorker struct {
	workers       []*worker
	regularWorker *worker
}

func (w *multiWorker) stop() {
	for _, worker := range w.workers {
		worker.stop()
	}
}

func (w *multiWorker) start() {
	for _, worker := range w.workers {
		worker.start()
	}
}

func (w *multiWorker) close() {
	for _, worker := range w.workers {
		worker.close()
	}
}

func (w *multiWorker) isRunning() bool {
	for _, worker := range w.workers {
		if worker.isRunning() {
			return true
		}
	}
	return false
}

// pendingBlockAndReceipts returns pending block and corresponding receipts from the `regularWorker`
func (w *multiWorker) pendingBlockAndReceipts() (*types.Block, types.Receipts) {
	// return a snapshot to avoid contention on currentMu mutex
	return w.regularWorker.pendingBlockAndReceipts()
}

func (w *multiWorker) setGasCeil(ceil uint64) {
	for _, worker := range w.workers {
		worker.setGasCeil(ceil)
	}
}

func (w *multiWorker) setExtra(extra []byte) {
	for _, worker := range w.workers {
		worker.setExtra(extra)
	}
}

func (w *multiWorker) setRecommitInterval(interval time.Duration) {
	for _, worker := range w.workers {
		worker.setRecommitInterval(interval)
	}
}

func (w *multiWorker) setEtherbase(addr common.Address) {
	for _, worker := range w.workers {
		worker.setEtherbase(addr)
	}
}

func (w *multiWorker) enablePreseal() {
	for _, worker := range w.workers {
		worker.enablePreseal()
	}
}

func (w *multiWorker) disablePreseal() {
	for _, worker := range w.workers {
		worker.disablePreseal()
	}
}

func newMultiWorker(config *Config, chainConfig *params.ChainConfig, engine consensus.Engine, eth Backend, mux *event.TypeMux, isLocalBlock func(*types.Block) bool, init bool) *multiWorker {
	queue := make(chan *task)

	regularWorker := newWorker(config, chainConfig, engine, eth, mux, isLocalBlock, init, &flashbotsData{
		isFlashbots: false,
		queue:       queue,
	})

	workers := []*worker{regularWorker}

	for i := 1; i <= config.MaxMergedBundles; i++ {
		workers = append(workers,
			newWorker(config, chainConfig, engine, eth, mux, isLocalBlock, init, &flashbotsData{
				isFlashbots:        true,
				isMegabundleWorker: false,
				queue:              queue,
				maxMergedBundles:   i,
			}))
	}

	for i := 0; i < len(config.TrustedRelays); i++ {
		workers = append(workers,
			newWorker(config, chainConfig, engine, eth, mux, isLocalBlock, init, &flashbotsData{
				isFlashbots:        true,
				isMegabundleWorker: true,
				queue:              queue,
				relayAddr:          config.TrustedRelays[i],
			}))
	}

	log.Info("creating multi worker", "config.MaxMergedBundles", config.MaxMergedBundles, "config.TrustedRelays", config.TrustedRelays, "worker", len(workers))
	return &multiWorker{
		regularWorker: regularWorker,
		workers:       workers,
	}
}

type flashbotsData struct {
	isFlashbots        bool
	isMegabundleWorker bool
	queue              chan *task
	maxMergedBundles   int
	relayAddr          common.Address
}
