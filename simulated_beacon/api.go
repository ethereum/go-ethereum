package simulated_beacon

import (
	"context"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

type API struct {
	simBeacon *SimulatedBeacon
}

func (api *API) AddWithdrawal(ctx context.Context, withdrawal *types.Withdrawal) error {
	api.simBeacon.mu.Lock()
	defer api.simBeacon.mu.Unlock()
	return api.simBeacon.withdrawals.add(withdrawal)
}

func (api *API) SetFeeRecipient(ctx context.Context, feeRecipient *common.Address) {
	api.simBeacon.mu.Lock()
	api.simBeacon.feeRecipient = *feeRecipient
	api.simBeacon.mu.Unlock()
}
