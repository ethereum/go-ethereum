package eth

import (
	"context"
	"errors"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

func (b *EthAPIBackend) SubmitRip7560Bundle(bundle *types.ExternallyReceivedBundle) error {
	if !b.rip7560AcceptPush {
		return errors.New("illegal call to eth_sendRip7560TransactionsBundle: Config.Eth.Rip7560AcceptPush is not set")
	}
	return b.eth.txPool.SubmitRip7560Bundle(bundle)
}

func (b *EthAPIBackend) GetRip7560BundleStatus(ctx context.Context, hash common.Hash) (*types.BundleReceipt, error) {
	return b.eth.txPool.GetRip7560BundleStatus(hash)
}
