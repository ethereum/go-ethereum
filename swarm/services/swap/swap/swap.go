// Copyright 2016 The go-ethereum Authors
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

package swap

import (
	"fmt"
	"math/big"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/log"
)

// SwAP Swarm Accounting Protocol with
//      Swift Automatic  Payments
// a peer to peer micropayment system

// Profile is the public swap profile, public parameters for SWAP,
// and serializable config struct passed in handshake.
type Profile struct {
	BuyAt  *big.Int // accepted max price for chunk
	SellAt *big.Int // offered sale price for chunk
	PayAt  uint     // threshold that triggers payment request
	DropAt uint     // threshold that triggers disconnect
}

// Strategy encapsulates parameters relating to
// automatic deposit and automatic cashing
type Strategy struct {
	AutoCashInterval     time.Duration // default interval for autocash
	AutoCashThreshold    *big.Int      // threshold that triggers autocash (wei)
	AutoDepositInterval  time.Duration // default interval for autocash
	AutoDepositThreshold *big.Int      // threshold that triggers autodeposit (wei)
	AutoDepositBuffer    *big.Int      // buffer that is surplus for fork protection etc (wei)
}

// Params extends the public profile with private parameters relating to
// automatic deposit and automatic cashing
type Params struct {
	*Profile
	*Strategy
}

// Promise is  a 3rd party Provable Promise of Payment issued by outPayment
// serialisable to send with Protocol.
type Promise interface{}

// Protocol is the interface for the peer protocol for testing or external 
// alternative payment.
type Protocol interface {
	Pay(int, Promise) // units, payment proof
	Drop()
	String() string
}

// OutPayment is the interface for the (delayed) ougoing payment system with autodeposit
type OutPayment interface {
	Issue(amount *big.Int) (promise Promise, err error)
	AutoDeposit(interval time.Duration, threshold, buffer *big.Int)
	Stop()
}

// InPayment is the interface for the (delayed) incoming payment system with autocash
type InPayment interface {
	Receive(promise Promise) (*big.Int, error)
	AutoCash(cashInterval time.Duration, maxUncashed *big.Int)
	Stop()
}

// Swap is the swarm accounting protocol instance
// * pairwise accounting and payments
type Swap struct {
	lock    sync.Mutex // mutex for balance access
	balance int        // units of chunk/retrieval request
	local   *Params    // local peer's swap parameters
	remote  *Profile   // remote peer's swap profile
	proto   Protocol   // peer communication protocol
	Payment
}

type Payment struct {
	Out         OutPayment // outgoing payment handler
	In          InPayment  // incoming  payment handler
	Buys, Sells bool
}

// New is the swap constructor.
func New(local *Params, pm Payment, proto Protocol) (self *Swap, err error) {

	self = &Swap{
		local:   local,
		Payment: pm,
		proto:   proto,
	}

	self.SetParams(local)

	return
}

// SetRemote is the entry point for setting remote swap profile 
// (e.g from handshake or other message).
func (s *Swap) SetRemote(remote *Profile) {
	defer s.lock.Unlock()
	s.lock.Lock()

	s.remote = remote
	if s.Sells && (remote.BuyAt.Sign() <= 0 || s.local.SellAt.Sign() <= 0 || remote.BuyAt.Cmp(s.local.SellAt) < 0) {
		s.Out.Stop()
		s.Sells = false
	}
	if s.Buys && (remote.SellAt.Sign() <= 0 || s.local.BuyAt.Sign() <= 0 || s.local.BuyAt.Cmp(s.remote.SellAt) < 0) {
		s.In.Stop()
		s.Buys = false
	}

	log.Debug(fmt.Sprintf("<%v> remote profile set: pay at: %v, drop at: %v, buy at: %v, sell at: %v", s.proto, remote.PayAt, remote.DropAt, remote.BuyAt, remote.SellAt))
}

// SetParams is used to set strategy dynamically.
func (s *Swap) SetParams(local *Params) {
	defer s.lock.Unlock()
	s.lock.Lock()
	s.local = local
	s.setParams(local)
}

// caller holds the lock

func (s *Swap) setParams(local *Params) {

	if s.Sells {
		s.In.AutoCash(local.AutoCashInterval, local.AutoCashThreshold)
		log.Info(fmt.Sprintf("<%v> set autocash to every %v, max uncashed limit: %v", s.proto, local.AutoCashInterval, local.AutoCashThreshold))
	} else {
		log.Info(fmt.Sprintf("<%v> autocash off (not selling)", s.proto))
	}
	if s.Buys {
		s.Out.AutoDeposit(local.AutoDepositInterval, local.AutoDepositThreshold, local.AutoDepositBuffer)
		log.Info(fmt.Sprintf("<%v> set autodeposit to every %v, pay at: %v, buffer: %v", s.proto, local.AutoDepositInterval, local.AutoDepositThreshold, local.AutoDepositBuffer))
	} else {
		log.Info(fmt.Sprintf("<%v> autodeposit off (not buying)", s.proto))
	}
}

// Add with param n, n > 0 called when promised/provided n units of service,
// n < 0 called when used/requested n units of service.
func (s *Swap) Add(n int) error {
	defer s.lock.Unlock()
	s.lock.Lock()
	s.balance += n
	if !s.Sells && s.balance > 0 {
		log.Trace(fmt.Sprintf("<%v> remote peer cannot have debt (balance: %v)", s.proto, s.balance))
		s.proto.Drop()
		return fmt.Errorf("[SWAP] <%v> remote peer cannot have debt (balance: %v)", s.proto, s.balance)
	}
	if !s.Buys && s.balance < 0 {
		log.Trace(fmt.Sprintf("<%v> we cannot have debt (balance: %v)", s.proto, s.balance))
		return fmt.Errorf("[SWAP] <%v> we cannot have debt (balance: %v)", s.proto, s.balance)
	}
	if s.balance >= int(s.local.DropAt) {
		log.Trace(fmt.Sprintf("<%v> remote peer has too much debt (balance: %v, disconnect threshold: %v)", s.proto, s.balance, s.local.DropAt))
		s.proto.Drop()
		return fmt.Errorf("[SWAP] <%v> remote peer has too much debt (balance: %v, disconnect threshold: %v)", s.proto, s.balance, s.local.DropAt)
	} else if s.balance <= -int(s.remote.PayAt) {
		s.send()
	}
	return nil
}

func (s *Swap) Balance() int {
	defer s.lock.Unlock()
	s.lock.Lock()
	return s.balance
}

// send(units) is called when payment is due
// In case of insolvency no promise is issued and sent, safe against fraud
// No return value: no error = payment is opportunistic = hang in till dropped
func (s *Swap) send() {
	if s.local.BuyAt != nil && s.balance < 0 {
		amount := big.NewInt(int64(-s.balance))
		amount.Mul(amount, s.remote.SellAt)
		promise, err := s.Out.Issue(amount)
		if err != nil {
			log.Warn(fmt.Sprintf("<%v> cannot issue cheque (amount: %v, channel: %v): %v", s.proto, amount, s.Out, err))
		} else {
			log.Warn(fmt.Sprintf("<%v> cheque issued (amount: %v, channel: %v)", s.proto, amount, s.Out))
			s.proto.Pay(-s.balance, promise)
			s.balance = 0
		}
	}
}

// Receive with params (units, promise) is called by the protocol when a payment msg is received
// returns error if promise is invalid.
func (s *Swap) Receive(units int, promise Promise) error {
	if units <= 0 {
		return fmt.Errorf("invalid units: %v <= 0", units)
	}

	price := new(big.Int).SetInt64(int64(units))
	price.Mul(price, s.local.SellAt)

	amount, err := s.In.Receive(promise)

	if err != nil {
		err = fmt.Errorf("invalid promise: %v", err)
	} else if price.Cmp(amount) != 0 {
		// verify amount = units * unit sale price
		return fmt.Errorf("invalid amount: %v = %v * %v (units sent in msg * agreed sale unit price) != %v (signed in cheque)", price, units, s.local.SellAt, amount)
	}
	if err != nil {
		log.Trace(fmt.Sprintf("<%v> invalid promise (amount: %v, channel: %v): %v", s.proto, amount, s.In, err))
		return err
	}

	// credit remote peer with units
	s.Add(-units)
	log.Trace(fmt.Sprintf("<%v> received promise (amount: %v, channel: %v): %v", s.proto, amount, s.In, promise))

	return nil
}

// Stop causes autocash loop to terminate.
// Called after protocol handle loop terminates.
func (s *Swap) Stop() {
	defer s.lock.Unlock()
	s.lock.Lock()
	if s.Buys {
		s.Out.Stop()
	}
	if s.Sells {
		s.In.Stop()
	}
}
