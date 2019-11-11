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
	"errors"
	"math/big"
	"sync/atomic"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/log"
)

// ChequeDrawer represents the payment drawer in a off-chain payment channel.
type ChequeDrawer struct {
	withdrawRequest uint64
	cdb             *chequeDB
	contractBackend bind.ContractBackend
	deployBackend   bind.DeployBackend
	book            *AccountBook
	selfAddr        common.Address
	txSigner        *bind.TransactOpts

	keySigner    func(data []byte) ([]byte, error) // Used for testing, cheque signer
	chequeSigner func(data []byte) ([]byte, error) // Used for production environment, cheque signer
}

func NewChequeDrawer(txSigner *bind.TransactOpts, chequeSigner func(data []byte) ([]byte, error), selfAddr common.Address, contractAddr common.Address, contractBackend bind.ContractBackend, deployBackend bind.DeployBackend, db ethdb.Database) (*ChequeDrawer, error) {
	if contractAddr == (common.Address{}) {
		return nil, errors.New("empty contract address")
	}
	book, err := newAccountBook(contractAddr, contractBackend)
	if err != nil {
		return nil, err
	}
	drawer := &ChequeDrawer{
		cdb:             newChequeDB(db),
		contractBackend: contractBackend,
		deployBackend:   deployBackend,
		selfAddr:        selfAddr,
		book:            book,
		txSigner:        txSigner,
		chequeSigner:    chequeSigner,
	}
	return drawer, nil
}

// ContractAddr returns the address of deployed accountbook contract.
func (drawer *ChequeDrawer) ContractAddr() common.Address {
	return drawer.book.address
}

// Deposit transfers the given amount wei into the contract. If the sync is true,
// this function will wait until the transaction is included and return the actual
// transaction status.
func (drawer *ChequeDrawer) Deposit(context context.Context, amount *big.Int) (bool, error) {
	// Create an independent auth opt
	depositOpt := &bind.TransactOpts{
		From:   drawer.selfAddr,
		Signer: drawer.txSigner.Signer,
		Value:  amount,
	}
	tx, err := drawer.book.contract.Deposit(depositOpt)
	if err != nil {
		return false, err
	}
	receipt, err := bind.WaitMined(context, drawer.deployBackend, tx)
	if err != nil {
		return false, err
	}
	return receipt.Status == types.ReceiptStatusSuccessful, nil
}

//             +-------------  Deposit  ------------+
//             |                                    |
//             +-----------+-----------+------------+
//             |   Spent   |  Unspent  | Withdrawed |
// +-----------+-----------+-----------+------------+
// |   Paid    |   Unpaid  |
// +-----------+-----------+
// |                       |
// +---  Total Issued   ---+

// Unpaid returns unpaid amount in the channel. The calculation method is using
// total_issued minus paid. However the total_issued record can be missing due
// to db corrupt. If this happen, we can reset the total issued amount by repair.
// If the returned error is nil, the returned value should always be non-nil.
//
// todo(rjl493456442) it's too expensive for light client to retrieve paid amount
// every time.
func (drawer *ChequeDrawer) Unpaid() (*big.Int, error) {
	// Check how much we have already cashed
	paid, err := drawer.book.contract.Paids(nil, drawer.selfAddr)
	if err != nil {
		return nil, err
	}
	lastIssued := drawer.cdb.readLastIssued(drawer.selfAddr, drawer.book.address)
	// We never issue any cheque from this address or local db is corrupt
	if lastIssued == nil {
		// Drawee has already cashed a few cheques, local db must be corrupt,
		// repair it. But we still have no clue how much we have issued which
		// is not cashed by drawee yet.
		if paid.Uint64() != 0 {
			drawer.cdb.writeLastIssued(drawer.selfAddr, drawer.book.address, paid)
		}
		return big.NewInt(0), nil
	}
	// Drawee has already cashed a few cheques, but total_issued is lower than
	// the cashed amount, db is corrupt. But we still have no clue how much we
	// have issued which is not cashed by drawee yet.
	if lastIssued.Cmp(paid) < 0 {
		drawer.cdb.writeLastIssued(drawer.selfAddr, drawer.book.address, paid)
		return big.NewInt(0), nil
	}
	return new(big.Int).Sub(lastIssued, paid), nil
}

// Unspent returns all unspent balance of ourselves in the channel. It can
// happen that we get a larger value then real unspent amount due to the
// data loss.
// If the returned error is nil, the returned balance should always be non-nil.
func (drawer *ChequeDrawer) Unspent() (*big.Int, error) {
	// Fetch deposit balance from the contract.
	balance, err := drawer.book.contract.Deposits(nil, drawer.selfAddr)
	if err != nil {
		return nil, err
	}
	unpaid, err := drawer.Unpaid()
	if err != nil {
		return nil, err
	}
	// If no spendable balance or even worse we already spend the
	// money exceeds all deposit.
	if unpaid.Cmp(balance) > 0 {
		return big.NewInt(0), ErrNotEnoughDeposit
	}
	remaining := new(big.Int).Sub(balance, unpaid)
	req, err := drawer.book.contract.WithdrawRequests(nil, drawer.selfAddr)
	if err != nil {
		return nil, err
	}
	// Drawer has a opened withdrawal request, no matter it passes the challenge
	// period or not, minus this part.
	if req.Amount.Uint64() != 0 && remaining.Cmp(req.Amount) < 0 {
		return big.NewInt(0), ErrNotEnoughDeposit
	}
	return new(big.Int).Sub(remaining, req.Amount), nil
}

// IssueCheque creates a cheque for issuing specified amount money for payee.
//
// Whenever the drawer creates a cheque and sends it to drawee, drawee has
// the permission to cash the deposit of drawer in the contract.
//
// Because of the possible data loss, we can issue some double-spend cheques,
// they will be rejected by drawee.
//
// In the mean time, this function will also return the remaining unspent
// to trigger auto deposit.
func (drawer *ChequeDrawer) IssueCheque(amount *big.Int) (*Cheque, *big.Int, error) {
	if amount == nil || amount.Uint64() == 0 {
		return nil, nil, errors.New("invalid issue amount")
	}
	unspent, err := drawer.Unspent()
	if err != nil {
		return nil, nil, err
	}
	if unspent.Cmp(amount) < 0 {
		return nil, nil, ErrNotEnoughDeposit
	}
	var newAmount *big.Int
	// If local chequedb is broken, the new amount maybe is a invalid stale number.
	// Finally drawee will show us the evidence which we signed before, we can repair
	// broken db.
	lastIssued := drawer.cdb.readLastIssued(drawer.selfAddr, drawer.book.address)
	if lastIssued == nil {
		newAmount = amount
	} else {
		newAmount = new(big.Int).Add(lastIssued, amount)
	}
	// Assmeble the cheque and sign it.
	cheque := &Cheque{
		Drawer:       drawer.selfAddr,
		Amount:       newAmount,
		ContractAddr: drawer.book.address,
	}
	// Uses keySigner if we are testing.
	if drawer.keySigner != nil {
		if err := cheque.signWithKey(drawer.keySigner); err != nil {
			return nil, nil, err
		}
	} else {
		if err := cheque.sign(drawer.chequeSigner); err != nil {
			return nil, nil, err
		}
	}
	drawer.cdb.writeLastIssued(drawer.selfAddr, drawer.book.address, newAmount)
	return cheque, new(big.Int).Sub(unspent, amount), nil
}

// Withdraw submits a on-chain transaction to open the withdrawal request to
// withdraw all deposit.
func (drawer *ChequeDrawer) Withdraw(context context.Context) error {
	// In the channel contract, we only allow one withdrawal operation
	// at the same time. Ensure there is no opened withdrawl request.
	req, err := drawer.book.contract.WithdrawRequests(nil, drawer.selfAddr)
	if err != nil {
		return err
	}
	if req.Amount.Uint64() != 0 {
		atomic.StoreUint64(&drawer.withdrawRequest, req.CreatedAt.Uint64())
		log.Info("Ongoing withdrawal operation", "amount", req.Amount, "createAt", req.CreatedAt)
		return errors.New("duplicate withdraw operation")
	}
	unspent, err := drawer.Unspent()
	if err != nil {
		return err
	}
	if unspent.Uint64() == 0 {
		return errors.New("no withdrawable balance")
	}
	// todo(rjl493456442) add a threshold checking, ignore small balance.
	tx, err := drawer.book.contract.Withdraw(drawer.txSigner, unspent)
	if err != nil {
		return err
	}
	receipt, err := bind.WaitMined(context, drawer.deployBackend, tx)
	if err != nil {
		return err
	}
	if receipt.Status == types.ReceiptStatusSuccessful {
		ret, err := drawer.book.contract.WithdrawRequests(nil, drawer.selfAddr)
		if err != nil {
			return err
		}
		atomic.StoreUint64(&drawer.withdrawRequest, ret.CreatedAt.Uint64())
	}
	return nil
}

// WithdrawalRecord returns the create block number of ongoing withdrawal request.
func (drawer *ChequeDrawer) WithdrawalRecord() uint64 {
	return atomic.LoadUint64(&drawer.withdrawRequest)
}

// ResetWithdrawlRecord resets withdrawal record when we submit the cash transaction.
func (drawer *ChequeDrawer) ResetWithdrawlRecord() {
	atomic.StoreUint64(&drawer.withdrawRequest, 0)
}

// Claim submits a on-chain transaction to claim all claimable balance.
func (drawer *ChequeDrawer) Claim(context context.Context) (bool, error) {
	tx, err := drawer.book.contract.Claim(drawer.txSigner)
	if err != nil {
		return false, err
	}
	receipt, err := bind.WaitMined(context, drawer.deployBackend, tx)
	if err != nil {
		return false, err
	}
	return receipt.Status == types.ReceiptStatusSuccessful, nil
}

// CodeHash returns the code hash of payment channel.
func (drawer *ChequeDrawer) CodeHash(context context.Context) (common.Hash, error) {
	code, err := drawer.deployBackend.CodeAt(context, drawer.book.address, nil)
	if err != nil {
		return common.Hash{}, err
	}
	return crypto.Keccak256Hash(code), nil
}

// Amend amends the local cheque db with externally provided cheque which is issued
// by ourselves.
func (drawer *ChequeDrawer) Amend(cheque *Cheque) error {
	if err := cheque.validate(drawer.book.address); err != nil {
		return err
	}
	if cheque.Drawer != drawer.selfAddr {
		return errors.New("invalid evidence")
	}
	// If local chequedb is corrupt, we can lose some payment records.
	// Since the amount of cheque is cumulative, so we need the evidence
	// from drawee to amend the local db.
	lastIssued := drawer.cdb.readLastIssued(drawer.selfAddr, drawer.book.address)
	if lastIssued == nil || lastIssued.Cmp(cheque.Amount) < 0 {
		drawer.cdb.writeLastIssued(drawer.selfAddr, drawer.book.address, cheque.Amount)
	}
	return nil
}

// Payed returns the total payed amount in this channel.
func (drawer *ChequeDrawer) Payed() *big.Int {
	return drawer.cdb.readLastIssued(drawer.selfAddr, drawer.ContractAddr())
}
