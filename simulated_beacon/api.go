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
	return api.simBeacon.withdrawals.add(withdrawal)
}

func (api *API) SetFeeRecipient(ctx context.Context, feeRecipient *common.Address) {
	api.simBeacon.setFeeRecipient(feeRecipient)
}
