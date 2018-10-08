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
)

type PriceOracle interface {
	Price(uint32, interface{}) (EntryDirection, uint64)
	Accountable(interface{}) bool
}

type BalanceManager interface {
	//Credit is crediting the peer, charging local node
	Credit(peer *Peer, amount uint64, size uint32) error
	//Debit is crediting the local node, charging the remote peer
	Debit(peer *Peer, amount uint64, size uint32) error
}

type EntryDirection bool

const (
	ChargeSender   EntryDirection = true
	ChargeReceiver EntryDirection = false
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
	if !ah.PriceOracle.Accountable(msg) {
		return nil
	}
	direction, price := ah.PriceOracle.Price(size, msg)
	if direction == ChargeSender {
		err = ah.BalanceManager.Debit(peer, price, size)
	} else {
		err = ah.BalanceManager.Credit(peer, price, size)
	}
	return err
}

func (ah *AccountingHook) Receive(peer *Peer, size uint32, msg interface{}) error {
	ah.lock.Lock()
	defer ah.lock.Unlock()
	var err error
	if !ah.PriceOracle.Accountable(msg) {
		return nil
	}
	direction, price := ah.PriceOracle.Price(size, msg)
	if direction == ChargeReceiver {
		err = ah.BalanceManager.Debit(peer, price, size)
	} else {
		err = ah.BalanceManager.Credit(peer, price, size)
	}
	return err
}
