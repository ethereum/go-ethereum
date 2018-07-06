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

	"github.com/ethereum/go-ethereum/swarm/log"
)

// SwAP Swarm Accounting Protocol with
//      Swift Automatic  Payments
// a peer to peer micropayment system

// Profile - public swap profile
// public parameters for SWAP, serializable config struct passed in handshake
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

// Promise - 3rd party Provable Promise of Payment
// issued by outPayment
// serializable to send with Protocol
type Promise interface{}

// Protocol interface for the peer protocol for testing or external alternative payment
type Protocol interface {
	Pay(int, Promise) // units, payment proof
	Drop()
	String() string
}

// OutPayment interface for the (delayed) outgoing payment system with auto-deposit
type OutPayment interface {
	Issue(amount *big.Int) (promise Promise, err error)
	AutoDeposit(interval time.Duration, threshold, buffer *big.Int)
	Stop()
}

// InPayment interface for the (delayed) incoming payment system with autocash
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

// Payment handlers
type Payment struct {
	Out         OutPayment // outgoing payment handler
	In          InPayment  // incoming  payment handler
	Buys, Sells bool
}

// New - swap constructor
func New(local *Params, pm Payment, proto Protocol) (swap *Swap, err error) {

	swap = &Swap{
		local:   local,
		Payment: pm,
		proto:   proto,
	}

	swap.SetParams(local)

	return
}

// SetRemote - entry point for setting remote swap profile (e.g from handshake or other message)
func (swap *Swap) SetRemote(remote *Profile) {
	defer swap.lock.Unlock()
	swap.lock.Lock()

	swap.remote = remote
	if swap.Sells && (remote.BuyAt.Sign() <= 0 || swap.local.SellAt.Sign() <= 0 || remote.BuyAt.Cmp(swap.local.SellAt) < 0) {
		swap.Out.Stop()
		swap.Sells = false
	}
	if swap.Buys && (remote.SellAt.Sign() <= 0 || swap.local.BuyAt.Sign() <= 0 || swap.local.BuyAt.Cmp(swap.remote.SellAt) < 0) {
		swap.In.Stop()
		swap.Buys = false
	}

	log.Debug(fmt.Sprintf("<%v> remote profile set: pay at: %v, drop at: %v, buy at: %v, sell at: %v", swap.proto, remote.PayAt, remote.DropAt, remote.BuyAt, remote.SellAt))

}

// SetParams - to set strategy dynamically
func (swap *Swap) SetParams(local *Params) {
	defer swap.lock.Unlock()
	swap.lock.Lock()
	swap.local = local
	swap.setParams(local)
}

// setParams - caller holds the lock
func (swap *Swap) setParams(local *Params) {

	if swap.Sells {
		swap.In.AutoCash(local.AutoCashInterval, local.AutoCashThreshold)
		log.Info(fmt.Sprintf("<%v> set autocash to every %v, max uncashed limit: %v", swap.proto, local.AutoCashInterval, local.AutoCashThreshold))
	} else {
		log.Info(fmt.Sprintf("<%v> autocash off (not selling)", swap.proto))
	}
	if swap.Buys {
		swap.Out.AutoDeposit(local.AutoDepositInterval, local.AutoDepositThreshold, local.AutoDepositBuffer)
		log.Info(fmt.Sprintf("<%v> set autodeposit to every %v, pay at: %v, buffer: %v", swap.proto, local.AutoDepositInterval, local.AutoDepositThreshold, local.AutoDepositBuffer))
	} else {
		log.Info(fmt.Sprintf("<%v> autodeposit off (not buying)", swap.proto))
	}
}

// Add (n)
// n > 0 called when promised/provided n units of service
// n < 0 called when used/requested n units of service
func (swap *Swap) Add(n int) error {
	defer swap.lock.Unlock()
	swap.lock.Lock()
	swap.balance += n
	if !swap.Sells && swap.balance > 0 {
		log.Trace(fmt.Sprintf("<%v> remote peer cannot have debt (balance: %v)", swap.proto, swap.balance))
		swap.proto.Drop()
		return fmt.Errorf("[SWAP] <%v> remote peer cannot have debt (balance: %v)", swap.proto, swap.balance)
	}
	if !swap.Buys && swap.balance < 0 {
		log.Trace(fmt.Sprintf("<%v> we cannot have debt (balance: %v)", swap.proto, swap.balance))
		return fmt.Errorf("[SWAP] <%v> we cannot have debt (balance: %v)", swap.proto, swap.balance)
	}
	if swap.balance >= int(swap.local.DropAt) {
		log.Trace(fmt.Sprintf("<%v> remote peer has too much debt (balance: %v, disconnect threshold: %v)", swap.proto, swap.balance, swap.local.DropAt))
		swap.proto.Drop()
		return fmt.Errorf("[SWAP] <%v> remote peer has too much debt (balance: %v, disconnect threshold: %v)", swap.proto, swap.balance, swap.local.DropAt)
	} else if swap.balance <= -int(swap.remote.PayAt) {
		swap.send()
	}
	return nil
}

// Balance accessor
func (swap *Swap) Balance() int {
	defer swap.lock.Unlock()
	swap.lock.Lock()
	return swap.balance
}

// send (units) is called when payment is due
// In case of insolvency no promise is issued and sent, safe against fraud
// No return value: no error = payment is opportunistic = hang in till dropped
func (swap *Swap) send() {
	if swap.local.BuyAt != nil && swap.balance < 0 {
		amount := big.NewInt(int64(-swap.balance))
		amount.Mul(amount, swap.remote.SellAt)
		promise, err := swap.Out.Issue(amount)
		if err != nil {
			log.Warn(fmt.Sprintf("<%v> cannot issue cheque (amount: %v, channel: %v): %v", swap.proto, amount, swap.Out, err))
		} else {
			log.Warn(fmt.Sprintf("<%v> cheque issued (amount: %v, channel: %v)", swap.proto, amount, swap.Out))
			swap.proto.Pay(-swap.balance, promise)
			swap.balance = 0
		}
	}
}

// Receive (units, promise) is called by the protocol when a payment msg is received
// returns error if promise is invalid.
func (swap *Swap) Receive(units int, promise Promise) error {
	if units <= 0 {
		return fmt.Errorf("invalid units: %v <= 0", units)
	}

	price := new(big.Int).SetInt64(int64(units))
	price.Mul(price, swap.local.SellAt)

	amount, err := swap.In.Receive(promise)

	if err != nil {
		err = fmt.Errorf("invalid promise: %v", err)
	} else if price.Cmp(amount) != 0 {
		// verify amount = units * unit sale price
		return fmt.Errorf("invalid amount: %v = %v * %v (units sent in msg * agreed sale unit price) != %v (signed in cheque)", price, units, swap.local.SellAt, amount)
	}
	if err != nil {
		log.Trace(fmt.Sprintf("<%v> invalid promise (amount: %v, channel: %v): %v", swap.proto, amount, swap.In, err))
		return err
	}

	// credit remote peer with units
	swap.Add(-units)
	log.Trace(fmt.Sprintf("<%v> received promise (amount: %v, channel: %v): %v", swap.proto, amount, swap.In, promise))

	return nil
}

// Stop causes autocash loop to terminate.
// Called after protocol handle loop terminates.
func (swap *Swap) Stop() {
	defer swap.lock.Unlock()
	swap.lock.Lock()
	if swap.Buys {
		swap.Out.Stop()
	}
	if swap.Sells {
		swap.In.Stop()
	}
}
