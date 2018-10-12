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

//define some metrics
var (
	//NOTE: these metrics just define the interfaces and are currently *NOT persisted* over sessions
	//All metrics are cumulative

	//total amount of units credited
	mBalanceCredit = metrics.NewRegisteredCounterForced("account.balance.credit", nil)
	//total amount of units debited
	mBalanceDebit = metrics.NewRegisteredCounterForced("account.balance.debit", nil)
	//total amount of bytes credited
	mBytesCredit = metrics.NewRegisteredCounterForced("account.bytes.credit", nil)
	//total amount of bytes debited
	mBytesDebit = metrics.NewRegisteredCounterForced("account.bytes.debit", nil)
	//total amount of credited messages
	mMsgCredit = metrics.NewRegisteredCounterForced("account.msg.credit", nil)
	//total amount of debited messages
	mMsgDebit = metrics.NewRegisteredCounterForced("account.msg.debit", nil)
	//how many times local node had to drop remote peers
	mPeerDrops = metrics.NewRegisteredCounterForced("account.peerdrops", nil)
	//how many times local node overdrafted and dropped
	mSelfDrops = metrics.NewRegisteredCounterForced("account.selfdrops", nil)
	//how many cheques have been issued
	//mChequesIssued = metrics.NewRegisteredCounterForced("account.cheques.issued", nil)
	//how many cheques have been received
	//mChequesReceived = metrics.NewRegisteredCounterForced("account.cheques.received", nil)
)

//Prices defines how prices are being passed on to the accounting instance
type Prices interface {
	//Return the Price for a message
	Price(interface{}) *Price
}

//Price represents the costs of a message
type Price struct {
	Value   int64 //Positive if sender pays, negative if receiver pays
	PerByte bool  //True if the price is per byte or for unit
}

//Price can be different depending on if it is the sender or receiver who pays
//ForSender gives back the price for sending a message
func (p *Price) ForSender(size uint32) int64 {
	price := p.Value
	if p.PerByte {
		price *= int64(size)
	}
	return price
}

//Price can be different depending on if it is the sender or receiver who pays
//ForReceiver gives back the price for  receiving a message
func (p *Price) ForReceiver(size uint32) int64 {
	price := p.Value
	if p.PerByte {
		price *= int64(size)
	}
	return 0 - price
}

//Balance is the actual accounting instance
//Balance defines the operations needed for accounting
//Implementations internally maintain the balance for every peer
type Balance interface {
	//Adds amount to the local balance with remote node `peer`;
	//positive amount = credit local node
	//negative amount = debit local node
	Add(amount int64, peer *Peer) error
}

//Accounting implements the Hook interface
//It interfaces to the balances through the Balance interface,
//while interfacing with protocols and its prices through the Prices interface
type Accounting struct {
	Balance //interface to accounting logic
	Prices  //interface to prices logic
}

func NewAccounting(mgr Balance, po Prices) *Accounting {
	ah := &Accounting{
		Prices:  po,
		Balance: mgr,
	}
	return ah
}

//Implement Hook.Send
func (ah *Accounting) Send(peer *Peer, size uint32, msg interface{}) error {
	//get the price for a message (through the protocol spec)
	price := ah.Price(msg)
	//this message doesn't need accounting
	if price == nil {
		return nil
	}
	//evaluate the price for sending messages
	finalPrice := price.ForSender(size)
	//do the accounting
	err := ah.Add(finalPrice, peer)
	//record metrics
	ah.doMetrics(finalPrice, size, err)
	return err
}

//Implement Hook.Receive
func (ah *Accounting) Receive(peer *Peer, size uint32, msg interface{}) error {
	//get the price for a message (through the protocol spec)
	price := ah.Price(msg)
	//this message doesn't need accounting
	if price == nil {
		return nil
	}
	//evaluate the price for receiving messages
	finalPrice := price.ForReceiver(size)
	//do the accounting
	err := ah.Add(finalPrice, peer)
	//record metrics
	ah.doMetrics(finalPrice, size, err)
	return err
}

//record some metrics
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
