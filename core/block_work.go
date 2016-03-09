package core

import (
	"fmt"

	"github.com/ethereum/go-ethereum/balancer"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/pow"
)

// nonceResult returns whether the nonce was valid or not.
type nonceResult struct {
	index int
	err   error
}

// CreateBlockTasks creates new work for the load balancer and returns a
// channel on which the results of each block is written.
func CreateBlockTasks(b *balancer.Balancer, blocks []*types.Block, checker pow.PoW) chan nonceResult {
	if len(blocks) == 0 {
		return nil
	}

	var (
		// Nonce result channel used to post the result of each nonce
		// checked by the task.
		nonceResults = make(chan nonceResult, len(blocks))
		// Error channel (ignored, see below)
		errch = make(chan error, len(blocks))
	)
	for i, block := range blocks {
		i := i
		task := balancer.NewTask(func() error {
			var err error
			// verify the block's nonce...
			if !checker.Verify(block) {
				err = &BlockNonceErr{Hash: block.Hash(), Number: block.Number(), Nonce: block.Nonce()}
			}

			// ...verify the block uncle's nonces...
			for _, u := range block.Uncles() {
				if !checker.Verify(types.NewBlockWithHeader(u)) {
					err = fmt.Errorf("uncle: %v", BlockNonceErr{Hash: u.Hash(), Number: u.Number, Nonce: block.Nonce()})
					break
				}
			}
			// ...and write the result on the results chan
			nonceResults <- nonceResult{i, err}
			// return nil, ignore error handling by the balancer
			return nil
		}, errch)
		// push task to the balancer
		b.Push(task)
		// create transaction tasks for this block
		CreateTxWork(b, block.Transactions())
	}

	// we aren't at all interested in the errors
	// since we handle errors ourself.
	go func() {
		// empty out the error channel
		for i := 0; i < len(blocks); i++ {
			<-errch
		}
		close(errch)
	}()
	return nonceResults
}

// CreateTxWork creates new tasks for the load balancer and derives the public
// key for each transaction.
func CreateTxWork(b *balancer.Balancer, txs types.Transactions) {
	if len(txs) == 0 {
		return
	}

	// error channel (ignored, see below)
	errch := make(chan error, len(txs))
	// create a new tasks for each transaction and derive the public
	// key from the sender signature.
	for i := 0; i < len(txs); i++ {
		i := i
		// create new tasks
		task := balancer.NewTask(func() error {
			txs[i].FromFrontier()
			return nil
		}, errch)
		// push task to the balancer
		b.Push(task)
	}

	// we aren't at all interested in the errors
	go func() {
		for i := 0; i < cap(errch); i++ {
			<-errch
		}
		close(errch)
	}()
}
