// Copyright 2023 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

package simulatedbeacon

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
