package miner

import (
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/consensus"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/event"
	"github.com/ethereum/go-ethereum/params"
)

type multiWorker struct {
	regularWorker   *worker
	flashbotsWorker *worker
}

func (w *multiWorker) stop() {
	w.regularWorker.stop()
	w.flashbotsWorker.stop()
}

func (w *multiWorker) start() {
	w.regularWorker.start()
	w.flashbotsWorker.start()
}

func (w *multiWorker) close() {
	w.regularWorker.close()
	w.flashbotsWorker.close()
}

func (w *multiWorker) isRunning() bool {
	return w.regularWorker.isRunning() || w.flashbotsWorker.isRunning()
}

func (w *multiWorker) setExtra(extra []byte) {
	w.regularWorker.setExtra(extra)
	w.flashbotsWorker.setExtra(extra)
}

func (w *multiWorker) setRecommitInterval(interval time.Duration) {
	w.regularWorker.setRecommitInterval(interval)
	w.flashbotsWorker.setRecommitInterval(interval)
}

func (w *multiWorker) setEtherbase(addr common.Address) {
	w.regularWorker.setEtherbase(addr)
	w.flashbotsWorker.setEtherbase(addr)
}

func (w *multiWorker) enablePreseal() {
	w.regularWorker.enablePreseal()
	w.flashbotsWorker.enablePreseal()
}

func (w *multiWorker) disablePreseal() {
	w.regularWorker.disablePreseal()
	w.flashbotsWorker.disablePreseal()
}

func newMultiWorker(config *Config, chainConfig *params.ChainConfig, engine consensus.Engine, eth Backend, mux *event.TypeMux, isLocalBlock func(*types.Block) bool, init bool) *multiWorker {
	queue := make(chan *task)

	return &multiWorker{
		regularWorker: newWorker(config, chainConfig, engine, eth, mux, isLocalBlock, init, &flashbotsData{
			isFlashbots: false,
			queue:       queue,
		}),
		flashbotsWorker: newWorker(config, chainConfig, engine, eth, mux, isLocalBlock, init, &flashbotsData{
			isFlashbots: true,
			queue:       queue,
		}),
	}
}

type flashbotsData struct {
	isFlashbots bool
	queue       chan *task
}
