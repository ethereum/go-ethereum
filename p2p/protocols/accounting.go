// Copyright 2018 The go-ethereum Authors
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

package protocols

import (
	"sync"

	"github.com/ethereum/go-ethereum/metrics"
)

var (
	//NOTE: these metrics just define the interfaces and are currently *NOT persisted* over sessions
	mBalanceCredit   = metrics.NewRegisteredCounterForced("account.balance.credit", nil)
	mBalanceDebit    = metrics.NewRegisteredCounterForced("account.balance.debit", nil)
	mBytesCredit     = metrics.NewRegisteredCounterForced("account.bytes.credit", nil)
	mBytesDebit      = metrics.NewRegisteredCounterForced("account.bytes.debit", nil)
	mMsgCredit       = metrics.NewRegisteredCounterForced("account.msg.credit", nil)
	mMsgDebit        = metrics.NewRegisteredCounterForced("account.msg.debit", nil)
	mPeerDrops       = metrics.NewRegisteredCounterForced("account.peerdrops", nil)
	mSelfDrops       = metrics.NewRegisteredCounterForced("account.selfdrops", nil)
	mChequesIssued   = metrics.NewRegisteredCounterForced("account.cheques.issued", nil)
	mChequesReceived = metrics.NewRegisteredCounterForced("account.cheques.received", nil)
)

type PriceOracle interface {
	Price(uint32, interface{}) (EntryDirection, uint64)
}

type BalanceManager interface {
	//Credit is crediting the peer, charging local node
	Credit(peer *Peer, amount uint64) error
	//Debit is crediting the local node, charging the remote peer
	Debit(peer *Peer, amount uint64) error
}

type EntryDirection uint

const (
	ChargeSender   EntryDirection = 1
	ChargeReceiver EntryDirection = 2
	ChargeNone     EntryDirection = 3
)

type AccountingHook struct {
	BalanceManager
	PriceOracle
	lock sync.RWMutex //lock the balances
}

func NewAccountingHook(mgr BalanceManager, po PriceOracle) *AccountingHook {
	ah := &AccountingHook{
		PriceOracle:    po,
		BalanceManager: mgr,
	}
	return ah
}

func (ah *AccountingHook) Send(peer *Peer, size uint32, msg interface{}) error {
	ah.lock.Lock()
	defer ah.lock.Unlock()
	var err error
	direction, price := ah.PriceOracle.Price(size, msg)
	if direction == ChargeSender {
		err = ah.BalanceManager.Debit(peer, price)
		ah.debitMetrics(price, size, err)
	} else if direction == ChargeReceiver {
		err = ah.BalanceManager.Credit(peer, price)
		ah.creditMetrics(price, size, err)
	} else if direction == ChargeNone {
		return nil
	}
	return err
}

func (ah *AccountingHook) Receive(peer *Peer, size uint32, msg interface{}) error {
	ah.lock.Lock()
	defer ah.lock.Unlock()
	var err error
	direction, price := ah.PriceOracle.Price(size, msg)
	if direction == ChargeReceiver {
		err = ah.BalanceManager.Debit(peer, price)
		ah.debitMetrics(price, size, err)
	} else if direction == ChargeSender {
		err = ah.BalanceManager.Credit(peer, price)
		ah.creditMetrics(price, size, err)
	} else if direction == ChargeNone {
		return nil
	}
	return err
}

func (ah *AccountingHook) debitMetrics(price uint64, size uint32, err error) {
	mBalanceDebit.Inc(int64(price))
	mBytesDebit.Inc(int64(size))
	mMsgDebit.Inc(1)
	if err != nil {
		mSelfDrops.Inc(1)
	}
}

func (ah *AccountingHook) creditMetrics(price uint64, size uint32, err error) {
	mBalanceCredit.Inc(int64(price))
	mBytesCredit.Inc(int64(size))
	mMsgCredit.Inc(1)
	if err != nil {
		mPeerDrops.Inc(1)
	}
}
