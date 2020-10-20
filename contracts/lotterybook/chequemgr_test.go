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
	"math/big"
	"reflect"
	"testing"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/math"
	"github.com/ethereum/go-ethereum/contracts/lotterybook/contract"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/crypto"
)

func TestChequeManagement(t *testing.T) {
	env := newTestEnv(t)
	defer env.close()

	_, _, c, err := contract.DeployLotteryBook(bind.NewKeyedTransactor(env.drawerKey), env.backend)
	if err != nil {
		t.Fatalf("Failed to deploy contract: %v", err)
	}
	env.backend.Commit()
	cdb := newChequeDB(rawdb.NewMemoryDatabase())

	claim := make(chan struct{}, 1)
	mgr := newChequeManager(env.draweeAddr, env.backend.Blockchain(), c, cdb, func(ctx context.Context, cheque *Cheque) error {
		claim <- struct{}{}
		return nil
	})
	defer mgr.close()

	lottery, cheques, _, _ := env.newRawLottery([]common.Address{env.draweeAddr}, []uint64{128}, 5)

	var signed = cheques[0]
	signed.signWithKey(func(digestHash []byte) ([]byte, error) {
		return crypto.Sign(digestHash, env.drawerKey)
	})
	// Set the additional fields for the receiver.
	signed.RevealNumber = lottery.RevealNumber
	signed.Amount = lottery.Amount

	var payed = signed.copy()
	payed.RevealRange = []byte{0xff, 0xff, 0xff, 0xff}
	payed.SignedRange = math.MaxUint32
	payed.signWithKey(func(digestHash []byte) ([]byte, error) {
		return crypto.Sign(digestHash, env.drawerKey)
	})

	var cases = []struct {
		testFn func()
		expect []*Cheque
	}{
		// Track empty cheque
		{func() { mgr.trackCheque(signed) }, []*Cheque{signed}},

		// Re-track cheque with higher allowance, de-duplicated is expected
		{func() { mgr.trackCheque(payed) }, []*Cheque{payed}},

		// Lottery is revealed, but wait more confirmations
		{func() { env.commitEmptyBlocks(5) }, []*Cheque{payed}},

		// Time to claim the lottery
		{func() { env.commitEmptyBlocks(lotteryProcessConfirms); <-claim }, nil},
	}
	for _, c := range cases {
		c.testFn()
		got := mgr.activeCheques()
		if !reflect.DeepEqual(got, c.expect) {
			t.Fatal("Active cheques mismatch")
		}
	}
}

func TestChequeRecovery(t *testing.T) {
	env := newTestEnv(t)
	defer env.close()

	_, _, contract, err := contract.DeployLotteryBook(bind.NewKeyedTransactor(env.drawerKey), env.backend)
	if err != nil {
		t.Fatalf("Failed to deploy contract: %v", err)
	}
	env.backend.Commit()
	cdb := newChequeDB(rawdb.NewMemoryDatabase())

	current := env.backend.Blockchain().CurrentHeader().Number.Uint64()
	lottery, cheques, salt, _ := env.newRawLottery([]common.Address{env.draweeAddr}, []uint64{128}, 5)

	var signed = cheques[0]
	signed.RevealRange = []byte{0xff, 0xff, 0xff, 0xff}
	signed.SignedRange = math.MaxUint32
	signed.signWithKey(func(digestHash []byte) ([]byte, error) {
		return crypto.Sign(digestHash, env.drawerKey)
	})
	// Set the additional fields for the receiver.
	signed.RevealNumber = lottery.RevealNumber
	signed.Amount = lottery.Amount
	cdb.writeCheque(env.draweeAddr, env.drawerAddr, signed, false)

	opt := bind.NewKeyedTransactor(env.drawerKey)
	opt.Value = big.NewInt(128)
	contract.NewLottery(opt, lottery.Id, current+5, salt)
	env.backend.Commit()

	var cases = []struct {
		testFn func()
		claim  bool
		expect []*Cheque
	}{
		{func() {}, false, []*Cheque{signed}},
		{func() { env.commitEmptyUntil(current + 5) }, false, []*Cheque{signed}},
		{func() { env.commitEmptyUntil(current + 5 + lotteryProcessConfirms - 1) }, false, []*Cheque{signed}},
		{func() { env.commitEmptyUntil(current + 5 + lotteryProcessConfirms) }, true, nil},
	}
	for _, c := range cases {
		claim := make(chan struct{}, 1)
		mgr := newChequeManager(env.draweeAddr, env.backend.Blockchain(), contract, cdb, func(ctx context.Context, cheque *Cheque) error {
			claim <- struct{}{}
			return nil
		})
		c.testFn()
		if c.claim {
			<-claim
		}
		got := mgr.activeCheques()
		if !reflect.DeepEqual(got, c.expect) {
			t.Fatal("Active cheques mismatch")
		}
		mgr.close()
	}
}
