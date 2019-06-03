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
	"time"

	"github.com/ethereum/go-ethereum/metrics"
)

// define some metrics
var (
	// All metrics are cumulative

	// total amount of units credited
	mBalanceCredit = metrics.NewRegisteredCounterForced("account.balance.credit", metrics.AccountingRegistry)
	// total amount of units debited
	mBalanceDebit = metrics.NewRegisteredCounterForced("account.balance.debit", metrics.AccountingRegistry)
	// total amount of bytes credited
	mBytesCredit = metrics.NewRegisteredCounterForced("account.bytes.credit", metrics.AccountingRegistry)
	// total amount of bytes debited
	mBytesDebit = metrics.NewRegisteredCounterForced("account.bytes.debit", metrics.AccountingRegistry)
	// total amount of credited messages
	mMsgCredit = metrics.NewRegisteredCounterForced("account.msg.credit", metrics.AccountingRegistry)
	// total amount of debited messages
	mMsgDebit = metrics.NewRegisteredCounterForced("account.msg.debit", metrics.AccountingRegistry)
	// how many times local node had to drop remote peers
	mPeerDrops = metrics.NewRegisteredCounterForced("account.peerdrops", metrics.AccountingRegistry)
	// how many times local node overdrafted and dropped
	mSelfDrops = metrics.NewRegisteredCounterForced("account.selfdrops", metrics.AccountingRegistry)
)

// Prices defines how prices are being passed on to the accounting instance
type Prices interface {
	// Return the Price for a message
	Price(interface{}) *Price
}

type Payer bool

const (
	Sender   = Payer(true)
	Receiver = Payer(false)
)

// Price represents the costs of a message
type Price struct {
	Value   uint64
	PerByte bool // True if the price is per byte or for unit
	Payer   Payer
}

// For gives back the price for a message
// A protocol provides the message price in absolute value
// This method then returns the correct signed amount,
// depending on who pays, which is identified by the `payer` argument:
// `Send` will pass a `Sender` payer, `Receive` will pass the `Receiver` argument.
// Thus: If Sending and sender pays, amount positive, otherwise negative
// If Receiving, and receiver pays, amount positive, otherwise negative
func (p *Price) For(payer Payer, size uint32) int64 {
	price := p.Value
	if p.PerByte {
		price *= uint64(size)
	}
	if p.Payer == payer {
		return 0 - int64(price)
	}
	return int64(price)
}

// Balance is the actual accounting instance
// Balance defines the operations needed for accounting
// Implementations internally maintain the balance for every peer
type Balance interface {
	// Adds amount to the local balance with remote node `peer`;
	// positive amount = credit local node
	// negative amount = debit local node
	Add(amount int64, peer *Peer) error
}

// Accounting implements the Hook interface
// It interfaces to the balances through the Balance interface,
// while interfacing with protocols and its prices through the Prices interface
type Accounting struct {
	Balance // interface to accounting logic
	Prices  // interface to prices logic
}

func NewAccounting(balance Balance, po Prices) *Accounting {
	ah := &Accounting{
		Prices:  po,
		Balance: balance,
	}
	return ah
}

// SetupAccountingMetrics uses a separate registry for p2p accounting metrics;
// this registry should be independent of any other metrics as it persists at different endpoints.
// It also starts the persisting go-routine which
// at the passed interval writes the metrics to a LevelDB
func SetupAccountingMetrics(reportInterval time.Duration, path string) *AccountingMetrics {
	// create the DB and start persisting
	return NewAccountingMetrics(metrics.AccountingRegistry, reportInterval, path)
}

// Send takes a peer, a size and a msg and
//   - calculates the cost for the local node sending a msg of size to peer using the Prices interface
//   - credits/debits local node using balance interface
func (ah *Accounting) Send(peer *Peer, size uint32, msg interface{}) error {
	// get the price for a message (through the protocol spec)
	price := ah.Price(msg)
	// this message doesn't need accounting
	if price == nil {
		return nil
	}
	// evaluate the price for sending messages
	costToLocalNode := price.For(Sender, size)
	// do the accounting
	err := ah.Add(costToLocalNode, peer)
	// record metrics: just increase counters for user-facing metrics
	ah.doMetrics(costToLocalNode, size, err)
	return err
}

// Receive takes a peer, a size and a msg and
//   - calculates the cost for the local node receiving a msg of size from peer using the Prices interface
//   - credits/debits local node using balance interface
func (ah *Accounting) Receive(peer *Peer, size uint32, msg interface{}) error {
	// get the price for a message (through the protocol spec)
	price := ah.Price(msg)
	// this message doesn't need accounting
	if price == nil {
		return nil
	}
	// evaluate the price for receiving messages
	costToLocalNode := price.For(Receiver, size)
	// do the accounting
	err := ah.Add(costToLocalNode, peer)
	// record metrics: just increase counters for user-facing metrics
	ah.doMetrics(costToLocalNode, size, err)
	return err
}

// record some metrics
// this is not an error handling. `err` is returned by both `Send` and `Receive`
// `err` will only be non-nil if a limit has been violated (overdraft), in which case the peer has been dropped.
// if the limit has been violated and `err` is thus not nil:
//   * if the price is positive, local node has been credited; thus `err` implicitly signals the REMOTE has been dropped
//   * if the price is negative, local node has been debited, thus `err` implicitly signals LOCAL node "overdraft"
func (ah *Accounting) doMetrics(price int64, size uint32, err error) {
	if price > 0 {
		mBalanceCredit.Inc(price)
		mBytesCredit.Inc(int64(size))
		mMsgCredit.Inc(1)
		if err != nil {
			// increase the number of times a remote node has been dropped due to "overdraft"
			mPeerDrops.Inc(1)
		}
	} else {
		mBalanceDebit.Inc(price)
		mBytesDebit.Inc(int64(size))
		mMsgDebit.Inc(1)
		if err != nil {
			// increase the number of times the local node has done an "overdraft" in respect to other nodes
			mSelfDrops.Inc(1)
		}
	}
}
