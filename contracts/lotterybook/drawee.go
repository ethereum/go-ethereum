// Copyright 2020 The go-ethereum Authors
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

package lotterybook

import (
	"context"
	"errors"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/log"
)

// ChequeDrawee represents the payment drawee in a off-chain payment channel.
// In chequeDrawee the most basic functions related to payment are defined
// here like: AddCheque.
//
// ChequeDrawee is self-contained and stateful. It will track all received
// cheques and claim the lottery if it's the lucky winner. Only AddCheque
// is exposed and needed by external caller.
//
// In addition, the structure depends on the blockchain state of the local node.
// In order to avoid inconsistency, you need to ensure that the local node has
// completed synchronization before using drawee. Otherwise these following
// scenarios can happen:
// - Accept cheques based on the revealed lottery
// - Accept cheques based on the claimed/resetted lottery
// - Reject valid cheques with the NEW lottery(just submitted)
type ChequeDrawee struct {
	address  common.Address       // Address used by chequeDrawee to accept payment
	cdb      *chequeDB            // Database which saves all received payments
	book     *LotteryBook         // Shared lottery contract used to verify deposit and claim payment
	opts     *bind.TransactOpts   // Signing handler for transaction signing
	cmgr     *chequeManager       // The manager for all received cheques management
	cBackend bind.ContractBackend // Blockchain backend for contract interaction
	dBackend bind.DeployBackend   // Blockchain backend for contract interaction
	chain    Blockchain           // Backend for local blockchain accessing

	// Testing hooks
	onClaimedHook func(common.Hash) // onClaimedHook is called if a lottery is successfully claimed
}

// NewChequeDrawee creates a payment drawee instance which handles all payments.
func NewChequeDrawee(opts *bind.TransactOpts, address, contractAddr common.Address, chain Blockchain, cBackend bind.ContractBackend, dBackend bind.DeployBackend, db ethdb.Database) (*ChequeDrawee, error) {
	if contractAddr == (common.Address{}) {
		return nil, errors.New("empty contract address")
	}
	book, err := newLotteryBook(contractAddr, cBackend)
	if err != nil {
		return nil, err
	}
	cdb := newChequeDB(db)
	drawee := &ChequeDrawee{
		address:  address,
		cdb:      cdb,
		book:     book,
		opts:     opts,
		cBackend: cBackend,
		dBackend: dBackend,
		chain:    chain,
	}
	drawee.cmgr = newChequeManager(address, chain, book.contract, cdb, drawee.claim)
	return drawee, nil
}

func (drawee *ChequeDrawee) Close() {
	drawee.cmgr.close()
}

// AddCheque receives a cheque from the specified drawer, checks the validity
// and stores it locally. Besides, this function will return the net amount of
// this cheque.
func (drawee *ChequeDrawee) AddCheque(drawer common.Address, c *Cheque) (uint64, error) {
	if err := validateCheque(c, drawer, drawee.address, drawee.book.address); err != nil {
		return 0, err
	}
	var revealNumber, amount uint64
	current := drawee.chain.CurrentHeader().Number.Uint64()
	stored := drawee.cdb.readCheque(drawee.address, c.Signer(), c.LotteryId, false)
	if stored == nil {
		// It's the first time the receiver gets the cheque, resolve
		// the lottery info from the contract.
		l, err := drawee.book.contract.Lotteries(nil, c.LotteryId)
		if err != nil {
			return 0, err
		}
		// TODO what if the sender is deliberately attacking us
		// via sending cheques without deposit? Read status from
		// contract is not trivial.
		if l.Amount == 0 {
			return 0, errors.New("empty lottery") // not submitted, claimed, resetted
		}
		revealNumber, amount = l.RevealNumber, l.Amount
	} else {
		// The lottery info is already saved in the cheque, don't
		// bother the contract.
		revealNumber, amount = stored.RevealNumber, stored.Amount
	}
	if current >= revealNumber+lotterySafetyThreshold {
		invalidChequeMeter.Mark(1)
		return 0, errors.New("expired lottery")
	}
	// It's an almost expired lottery, but it's may not a deliberate operation
	// e.g. the sender is a bit out of sync. Don't return error here(which will
	// lead to drop the connection).
	if current+lotterySafetyMargin >= revealNumber {
		log.Debug("Reject almost expired lottery", "current", current, "reveal", revealNumber, "sender", drawer)
		return 0, nil
	}
	var diff uint64
	if stored != nil {
		if stored.SignedRange >= c.SignedRange {
			// There are many cases can lead to this situation:
			// * Drawer sends a stale cheque deliberately
			// * Drawer's chequedb is broken, it loses all payment history
			// No matter which reason, reject the cheque here.
			staleChequeMeter.Mark(1)
			return 0, errors.New("stale cheque")
		}
		// Figure out the net newly signed reveal range
		diff = c.SignedRange - stored.SignedRange
	} else {
		// Reject cheque if the paid amount is zero.
		if c.SignedRange == maxSignedRange {
			invalidChequeMeter.Mark(1)
			return 0, errors.New("invalid payment amount")
		}
		diff = c.SignedRange - c.LowerLimit + 1
	}
	// It may lose precision but it's ok.
	assigned := amount >> (len(c.Witness) - 1)

	// Note the following calculation may lose precision, but it's okish.
	//
	// In theory diff/interval WON't be very small. So it's the best choice
	// to calculate percentage first. Otherwise the calculation may overflow.
	diffAmount := uint64((float64(diff) / float64(c.UpperLimit-c.LowerLimit+1)) * float64(assigned))
	if diffAmount == 0 {
		invalidChequeMeter.Mark(1)
		return 0, errors.New("invalid payment amount")
	}
	// Tag the additional information(lottery) into the cheque. We have the
	// assumption that ALL cheques maintained in the receiver side have these
	// additional fields.
	c.RevealNumber, c.Amount = revealNumber, amount
	drawee.cdb.writeCheque(drawee.address, drawer, c, false)
	drawee.cmgr.trackCheque(c)
	return diffAmount, nil
}

// claim sends an on-chain transaction to claim the specified lottery.
// Note this function will block until the transaction is mined! Please
// don't call this in the main thread.
func (drawee *ChequeDrawee) claim(context context.Context, cheque *Cheque) error {
	var proofslice [][32]byte
	for i := 1; i < len(cheque.Witness); i++ {
		var p [32]byte
		copy(p[:], cheque.Witness[i].Bytes())
		proofslice = append(proofslice, p)
	}
	start := time.Now()
	if len(cheque.RevealRange) != 4 {
		return errors.New("invalid cheque")
	}
	var revealRange [4]byte
	copy(revealRange[:], cheque.RevealRange)
	tx, err := drawee.book.contract.Claim(drawee.opts, cheque.LotteryId, revealRange, cheque.Sig[64], common.BytesToHash(cheque.Sig[:common.HashLength]), common.BytesToHash(cheque.Sig[common.HashLength:2*common.HashLength]), cheque.ReceiverSalt, proofslice)
	if err != nil {
		return err
	}
	receipt, err := bind.WaitMined(context, drawee.dBackend, tx)
	if err != nil {
		return err
	}
	if receipt.Status != types.ReceiptStatusSuccessful {
		return ErrTransactionFailed
	}
	if drawee.onClaimedHook != nil {
		drawee.onClaimedHook(cheque.LotteryId)
	}
	claimDurationTimer.UpdateSince(start)
	log.Debug("Claimed lottery", "id", cheque.LotteryId)
	return nil
}

// Cheques returns all active cheques locally received.
func (drawee *ChequeDrawee) Cheques() []*Cheque {
	return drawee.cmgr.activeCheques()
}
