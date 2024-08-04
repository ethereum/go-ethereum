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

package catalyst

import (
	"context"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
)

// simulatedBeaconAPI provides a RPC API for SimulatedBeacon.
type simulatedBeaconAPI struct {
	sim      *SimulatedBeacon
	doCommit chan struct{}
}

// loop is the main loop for the API when it's running in period = 0 mode. It
// ensures that block production is triggered as soon as a new withdrawal or
// transaction is received.
func (a *simulatedBeaconAPI) loop() {
	var (
		newTxs    = make(chan core.NewTxsEvent)
		newWxs    = make(chan newWithdrawalsEvent)
		newTxsSub = a.sim.eth.TxPool().SubscribeTransactions(newTxs, true)
		newWxsSub = a.sim.withdrawals.Subscribe(newWxs)
	)
	defer newTxsSub.Unsubscribe()
	defer newWxsSub.Unsubscribe()

	go a.worker()

	for {
		select {
		case <-a.sim.shutdownCh:
			return
		case <-newWxs:
			a.commit()
		case <-newTxs:
			a.commit()
		}
	}
}

// commit is a non-blocking method to initate Commit() on the simulator.
func (a *simulatedBeaconAPI) commit() {
	select {
	case a.doCommit <- struct{}{}:
	default:
	}
}

// worker runs in the background and signals to the simulator when to commit
// based on messages over doCommit.
func (a *simulatedBeaconAPI) worker() {
	for {
		select {
		case <-a.sim.shutdownCh:
			return
		case <-a.doCommit:
			a.sim.Commit()
			a.sim.eth.TxPool().Sync()
			executable, _ := a.sim.eth.TxPool().Stats()
			if executable != 0 {
				a.commit()
			}
		}
	}
}

// AddWithdrawal adds a withdrawal to the pending queue.
func (a *simulatedBeaconAPI) AddWithdrawal(ctx context.Context, withdrawal *types.Withdrawal) error {
	return a.sim.withdrawals.Add(withdrawal)
}

// SetFeeRecipient sets the fee recipient for block building purposes.
func (a *simulatedBeaconAPI) SetFeeRecipient(ctx context.Context, feeRecipient common.Address) {
	a.sim.setFeeRecipient(feeRecipient)
}
