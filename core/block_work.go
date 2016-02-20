package core

import (
	"math"

	"github.com/ethereum/go-ethereum/balancer"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/pow"
)

type nonceResult struct {
	index int
	valid bool
}

func balanceBlockWork(b *balancer.Balancer, blocks []*types.Block, checker pow.PoW) (chan nonceResult, chan struct{}) {
	const workSize = 64

	var (
		nonceResults = make(chan nonceResult, len(blocks))                                      // the nonce result channel (buffered)
		errch        = make(chan error, int(math.Ceil(float64(len(blocks))/float64(workSize)))) // error channel (buffered)
	)
	for i := 0; i < len(blocks); i += workSize {
		max := int(math.Min(float64(i+workSize), float64(len(blocks)))) // get max size...
		batch := blocks[i:max]                                          // ...and create batch

		batchNo := i // batch number for task
		// create new tasks
		task := balancer.NewTask(func() error {
			for i := 0; i < max-batchNo; i++ {
				nonceResults <- nonceResult{batchNo + i, checker.Verify(batch[i])}
			}
			return nil
		}, errch)

		b.Push(task)
	}

	donech := make(chan struct{})
	// we aren't at all interested in the errors
	// since we handle errors ourself.
	go func() {
		<-donech // wait for parent proc to finish
		for i := 0; i < cap(errch); i++ {
			<-errch
		}
		close(errch)
	}()

	return nonceResults, donech
}
