package clmock

import (
	"context"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

type API struct {
	mock *CLMock
}

func (api *API) AddWithdrawal(ctx context.Context, withdrawal *types.Withdrawal) error {
	return api.mock.addWithdrawal(*withdrawal)
}

func (api *API) SetFeeRecipient(ctx context.Context, feeRecipient *common.Address) {
	api.mock.mu.Lock()
	api.mock.feeRecipient = *feeRecipient
	api.mock.mu.Unlock()
}
