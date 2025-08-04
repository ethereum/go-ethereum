package core

import (
	"runtime"
	"sync/atomic"

	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"golang.org/x/sync/errgroup"
)

// balPrefetcher implements a state prefetcher for block access lists.
// State present in the block access list is retrieved in parallel for
// each account.
type balPrefetcher struct{}

// Prefetch retrieves state contained in the block's access list to warm state
// caches before executing the block.
func (p *balPrefetcher) Prefetch(block *types.Block, db *state.StateDB, interrupt *atomic.Bool) {
	al := block.Body().AccessList

	var workers errgroup.Group

	workers.SetLimit(runtime.NumCPU() / 2)

	for _, accesses := range al.Accesses {
		statedb := db.Copy()
		workers.Go(func() error {
			statedb.GetBalance(accesses.Address)
			for _, storageAccess := range accesses.StorageWrites {
				if interrupt != nil && interrupt.Load() {
					return nil
				}
				statedb.GetState(accesses.Address, storageAccess.Slot)
			}
			for _, storageRead := range accesses.StorageReads {
				if interrupt != nil && interrupt.Load() {
					return nil
				}
				statedb.GetState(accesses.Address, storageRead)
			}
			if interrupt != nil && interrupt.Load() {
				return nil
			}
			statedb.GetCode(accesses.Address)
			return nil
		})
	}
	workers.Wait()
}
