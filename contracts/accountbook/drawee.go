// Copyright 2019 The go-ethereum Authors
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

package accountbook

import (
	"context"
	"math/big"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/contracts/accountbook/contract"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/event"
	"github.com/ethereum/go-ethereum/log"
)

// ChequeDrawee represents the payment drawee in a off-chain payment channel.
type ChequeDrawee struct {
	address  common.Address
	cdb      *chequeDB
	book     *AccountBook
	opts     *bind.TransactOpts
	cBackend bind.ContractBackend
	dBackend bind.DeployBackend
}

// NewChequeDrawee creates a payment drawee and deploys the contract if necessary.
func NewChequeDrawee(selfAddr common.Address, opts *bind.TransactOpts, contractBackend bind.ContractBackend, deployBackend bind.DeployBackend, db ethdb.Database) (*ChequeDrawee, error) {
	cdb := newChequeDB(db)
	var chanAddr common.Address
	stored := cdb.readContractAddr(selfAddr)
	if stored == nil {
		addr, err := deployAccountBook(opts, contractBackend, deployBackend)
		if err != nil {
			return nil, err
		}
		cdb.writeContractAddr(selfAddr, addr)
		chanAddr = addr
	} else {
		chanAddr = *stored
	}
	book, err := newAccountBook(chanAddr, contractBackend)
	if err != nil {
		return nil, err
	}
	drawee := &ChequeDrawee{
		address:  selfAddr,
		cdb:      cdb,
		book:     book,
		opts:     opts,
		cBackend: contractBackend,
		dBackend: deployBackend,
	}
	return drawee, nil
}

// ContractAddr returns the address of deployed accountbook contract.
func (drawee *ChequeDrawee) ContractAddr() common.Address {
	return drawee.book.address
}

// AddCheque receives a cheque from drawer, check the validity and store
// it locally.
//
// In the mean time, this function will return the cumulative uncash amount
// for auto cash triggering.
func (drawee *ChequeDrawee) AddCheque(c *Cheque) (*big.Int, *big.Int, error) {
	// Ensure the cheque is signed properly
	if err := c.validate(drawee.ContractAddr()); err != nil {
		return nil, nil, err
	}
	// Ensure the drawer has enough balance to cover the expense.
	unspent, err := drawee.Unspent(c.Drawer)
	if err != nil {
		return nil, nil, err
	}
	// Figure out the net amount of this cheque.
	lastReceived := drawee.cdb.readCheque(drawee.book.address, c.Drawer)
	var net *big.Int
	if lastReceived == nil {
		net = c.Amount
	} else if lastReceived.Amount.Cmp(c.Amount) >= 0 {
		// There are many cases can lead to this situation:
		// * Drawer passes a stale cheque deliberately
		// * Drawer's chequedb is broken, it loses all payment history
		// In order to help drawer to recover the payment history,
		// return an evidence here.
		return nil, nil, &StaleChequeError{Msg: "stale cheque", Evidence: lastReceived}
	} else {
		net = new(big.Int).Sub(c.Amount, lastReceived.Amount)
	}
	if unspent.Cmp(net) < 0 {
		return nil, nil, ErrNotEnoughDeposit
	}
	// Calculate uncashed amount from this drawer.
	paid, err := drawee.book.contract.Paids(nil, c.Drawer)
	if err != nil {
		return nil, nil, err
	}
	// Pass the validation, save it into disk.
	drawee.cdb.writeCheque(drawee.book.address, c.Drawer, c)
	return net, new(big.Int).Sub(c.Amount, paid), nil
}

//             +-------------   Deposit  -----------+
//             |                                    |
//             +-----------+-----------+------------+
//             |   Spent   |  Unspent  | Withdrawed |
// +-----------+-----------+-----------+------------+
// |   Paid    |   Unpaid  |
// +-----------+-----------+
// |                       |
// +---  Total Issued   ---+

// Unpaid returns unpaid amount of the specified drawer in the channel.
// The calculation method is using total_received(total_issued) minus cashed(paid).
// However the total_received record can be missing due to db corrupt. If this happen,
// we can only adjust the total_received to paid amount and drawer can double-spend
// the uncashed part.
// If the returned error is nil, the returned value should always be non-nil.
func (drawee *ChequeDrawee) Unpaid(addr common.Address) (*big.Int, error) {
	// Check how much we have already cashed
	paid, err := drawee.book.contract.Paids(nil, addr)
	if err != nil {
		return nil, err
	}
	lastReceived := drawee.cdb.readCheque(drawee.book.address, addr)
	// We never receive the cheque from this address or local db is corrupt
	if lastReceived == nil {
		// We have cashing record in contract, but total_received is missing,
		// db is corrupt.
		if paid.Uint64() != 0 {
			// Write a cheque without signature and drawer, we only need the issued amount.
			drawee.cdb.writeCheque(drawee.book.address, addr, &Cheque{Amount: paid, ContractAddr: drawee.ContractAddr()})
		}
		return big.NewInt(0), nil
	}
	// We have cashing record in contract, but total_received is lower than
	// the cashed amount, db is corrupt.
	if lastReceived.Amount.Cmp(paid) < 0 {
		// Write a cheque without signature and drawer, we only need the issued amount.
		drawee.cdb.writeCheque(drawee.book.address, addr, &Cheque{Amount: paid, ContractAddr: drawee.ContractAddr()})
		return big.NewInt(0), nil
	}
	return new(big.Int).Sub(lastReceived.Amount, paid), nil
}

// Unspent returns all unspent balance of the specified drawer in the channel.
// According to the diagram of balance, we can see unspent part is: deposit-withdrawed-unpaid.
// If the returned error is nil, the returned balance should always be non-nil.
func (drawee *ChequeDrawee) Unspent(addr common.Address) (*big.Int, error) {
	// Fetch deposit balance from the contract.
	balance, err := drawee.book.contract.Deposits(nil, addr)
	if err != nil {
		return nil, err
	}
	unpaid, err := drawee.Unpaid(addr)
	if err != nil {
		return nil, err
	}
	// If no spendable balance or even worse the drawer already spends the
	// money exceeds all deposit.
	if unpaid.Cmp(balance) > 0 {
		return nil, ErrNotEnoughDeposit
	}
	remaining := new(big.Int).Sub(balance, unpaid)
	req, err := drawee.book.contract.WithdrawRequests(nil, addr)
	if err != nil {
		return nil, err
	}
	// Drawer has a opened withdrawal request, no matter it passes the challenge
	// period or not, minus this part.
	if remaining.Cmp(req.Amount) < 0 {
		return nil, ErrNotEnoughDeposit
	}
	return new(big.Int).Sub(remaining, req.Amount), nil
}

// Cash cashes all unpaid payment by given drawer.
func (drawee *ChequeDrawee) Cash(context context.Context, drawer common.Address, sync bool) error {
	unpaid, err := drawee.Unpaid(drawer)
	if err != nil {
		return err
	}
	if unpaid.Uint64() == 0 {
		return nil // Nothing to cash
	}
	lastReceived := drawee.cdb.readCheque(drawee.book.address, drawer) // Can't be nil here.
	tx, err := drawee.book.contract.Cash(drawee.opts, drawer, lastReceived.Amount, lastReceived.Sig[64], common.BytesToHash(lastReceived.Sig[:32]), common.BytesToHash(lastReceived.Sig[32:64]))
	if err != nil {
		return err
	}
	if sync {
		_, err := bind.WaitMined(context, drawee.dBackend, tx)
		if err != nil {
			return err
		}
	}
	log.Info("Cashed cheque", "drawer", drawer, "amount", unpaid, "cumulative", lastReceived.Amount)
	return nil
}

// ListenWithdraw watches new withdraw event triggered by drawers.
func (drawee *ChequeDrawee) ListenWithdraw() (event.Subscription, chan *contract.AccountBookWithdrawEvent, error) {
	sink := make(chan *contract.AccountBookWithdrawEvent)
	sub, err := drawee.book.contract.WatchWithdrawEvent(nil, sink, nil)
	if err != nil {
		return nil, nil, err
	}
	return sub, sink, nil
}

// ListCheques returns all cheques drawee received.
func (drawee *ChequeDrawee) ListCheques() []*Cheque {
	return drawee.cdb.allCheques(drawee.ContractAddr())
}
