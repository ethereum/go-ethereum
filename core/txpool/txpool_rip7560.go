package txpool

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

// SubmitRip7560Bundle inserts the entire bundle of Type 4 transactions into the relevant pool.
func (p *TxPool) SubmitRip7560Bundle(bundle *types.ExternallyReceivedBundle) error {
	// todo: we cannot 'filter-out' the AA pool so just passing to all pools - only AA pool has code in SubmitBundle
	for _, subpool := range p.subpools {
		err := subpool.SubmitRip7560Bundle(bundle)
		if err != nil {
			return err
		}
	}
	return nil
}

func (p *TxPool) GetRip7560BundleStatus(hash common.Hash) (*types.BundleReceipt, error) {
	// todo: we cannot 'filter-out' the AA pool so just passing to all pools - only AA pool has code in SubmitBundle
	for _, subpool := range p.subpools {
		bundleStats, err := subpool.GetRip7560BundleStatus(hash)
		if err != nil {
			return nil, err
		}
		if bundleStats != nil {
			return bundleStats, nil
		}
	}
	return nil, nil
}

func (p *TxPool) PendingRip7560Bundle() (*types.ExternallyReceivedBundle, error) {
	// todo: we cannot 'filter-out' the AA pool so just passing to all pools - only AA pool has code in PendingBundle
	for _, subpool := range p.subpools {
		pendingBundle, err := subpool.PendingRip7560Bundle()
		if err != nil {
			return nil, err
		}
		if pendingBundle != nil {
			return pendingBundle, nil
		}
	}
	return nil, nil
}
