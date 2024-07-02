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
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
)

type api struct {
	sim *SimulatedBeacon
}

func (a *api) loop() {
	var (
		newTxs   = make(chan core.NewTxsEvent)
		sub      = a.sim.eth.TxPool().SubscribeTransactions(newTxs, true)
		commitMu = sync.Mutex{}
	)
	defer sub.Unsubscribe()

	for {
		select {
		case <-a.sim.shutdownCh:
			return
		case w := <-a.sim.withdrawals.pending:
			go func() {
				commitMu.Lock()
				defer commitMu.Unlock()
				// When the beacon chain is ran by a simulator, then transaction insertion,
				// block insertion and block production will happen without any timing
				// delay between them. This will cause flaky simulator executions due to
				// the transaction pool running its internal reset operation on a back-
				// ground thread. To avoid the racey behavior - in simulator mode - the
				// pool will be explicitly blocked on its reset before continuing to the
				// block production below.
				if err := a.sim.eth.TxPool().Sync(); err != nil {
					log.Error("Failed to sync transaction pool", "err", err)
					return
				}
				withdrawals := append(a.sim.withdrawals.gatherPending(9), w)
				if err := a.sim.sealBlock(withdrawals, uint64(time.Now().Unix())); err != nil {
					log.Warn("Error performing sealing work", "err", err)
				}
			}()
		case <-newTxs:
			go func() {
				commitMu.Lock()
				defer commitMu.Unlock()

				if err := a.sim.eth.TxPool().Sync(); err != nil {
					log.Error("Failed to sync transaction pool", "err", err)
					return
				}
				a.sim.Commit()
			}()
		}
	}
}

func (a *api) AddWithdrawal(ctx context.Context, withdrawal *types.Withdrawal) error {
	return a.sim.withdrawals.add(withdrawal)
}

func (a *api) SetFeeRecipient(ctx context.Context, feeRecipient common.Address) {
	a.sim.setFeeRecipient(feeRecipient)
}
