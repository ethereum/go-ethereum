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

type Prices interface {
	Price(interface{}) *Price
}

type Price struct {
	Value   int64 //Positive if sender pays, negative if receiver pays
	PerByte bool  //True if the price is per byte or for unit
}

func (p *Price) ForSender(size uint32) int64 {
	price := p.Value
	if p.PerByte {
		price *= int64(size)
	}
	return price
}

func (p *Price) ForReceiver(size uint32) int64 {
	price := p.Value
	if p.PerByte {
		price *= int64(size)
	}
	return 0 - price
}

type Balance interface {
	//Adds amount to the local balance with remote node `peer`;
	//positive amount = credit local node
	//negative amount = debit local node
	Add(amount int64, peer *Peer) error
}

type Accounting struct {
	Balance
	Prices
}

func NewAccounting(mgr Balance, po Prices) *Accounting {
	ah := &Accounting{
		Prices:  po,
		Balance: mgr,
	}
	return ah
}

func (ah *Accounting) Send(peer *Peer, size uint32, msg interface{}) error {
	price := ah.Price(msg)
	if price == nil {
		return nil
	}
	finalPrice := price.ForSender(size)
	err := ah.Add(finalPrice, peer)
	ah.doMetrics(finalPrice, size, err)
	return err
}

func (ah *Accounting) Receive(peer *Peer, size uint32, msg interface{}) error {
	price := ah.Price(msg)
	if price == nil {
		return nil
	}
	finalPrice := price.ForReceiver(size)
	err := ah.Add(finalPrice, peer)
	ah.doMetrics(finalPrice, size, err)
	return err
}

func (ah *Accounting) doMetrics(price int64, size uint32, err error) {
	if price > 0 {
		mBalanceCredit.Inc(int64(price))
		mBytesCredit.Inc(int64(size))
		mMsgCredit.Inc(1)
		if err != nil {
			mPeerDrops.Inc(1)
		}
	} else {
		mBalanceDebit.Inc(int64(price))
		mBytesDebit.Inc(int64(size))
		mMsgDebit.Inc(1)
		if err != nil {
			mSelfDrops.Inc(1)
		}
	}
}
