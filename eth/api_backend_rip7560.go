package eth

import (
	"context"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

func (b *EthAPIBackend) SubmitRip7560Bundle(bundle *types.ExternallyReceivedBundle) error {
	return b.eth.txPool.SubmitRip7560Bundle(bundle)
}

func (b *EthAPIBackend) GetRip7560BundleStatus(ctx context.Context, hash common.Hash) (*types.BundleReceipt, error) {
	return b.eth.txPool.GetRip7560BundleStatus(hash)
}
