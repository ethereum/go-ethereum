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
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/crypto"
)

func (env *testEnv) commitEmptyBlocks(n int) {
	for i := 0; i < n; i++ {
		env.backend.Commit()
	}
}
func (env *testEnv) issueAndCheck(t *testing.T, drawer *ChequeDrawer, amount *big.Int, expectErr error, expectCum *big.Int, expectUnspent *big.Int) *Cheque {
	cheque, unspent, err := drawer.IssueCheque(amount)
	if expectErr != nil {
		if err.Error() != expectErr.Error() {
			t.Fatalf("Error mismatch, want: %v, got: %v", expectErr, err)
		}
		return nil
	}
	if unspent.Cmp(expectUnspent) != 0 {
		t.Fatalf("Unspent amount mismatch, want: %v, got: %v", expectUnspent, unspent)
	}
	if cheque.Amount.Cmp(expectCum) != 0 {
		t.Fatalf("Cumulative spent amount mismatch, want: %v, got: %v", expectCum, cheque.Amount)
	}
	return cheque
}

func TestIssueCheque(t *testing.T) {
	env := newTestEnv(t)
	defer env.close()

	var exit = make(chan struct{})
	defer close(exit)

	// Start the automatic blockchain.
	go func() {
		ticker := time.NewTicker(time.Millisecond * 100)
		for {
			select {
			case <-ticker.C:
				env.backend.Commit()
			case <-exit:
				return
			}
		}
	}()
	// Deploy the contract if missing
	drawee, err := NewChequeDrawee(env.draweeAddr, bind.NewKeyedTransactor(env.draweeKey), env.backend, env.backend, env.db)
	if err != nil {
		t.Fatalf("Faield to deploy contract, err: %v", err)
	}
	drawer, err := NewChequeDrawer(bind.NewKeyedTransactor(env.drawerKey), nil, env.drawerAddr, drawee.ContractAddr(), env.backend, env.backend, env.db)
	if err != nil {
		t.Fatalf("Faield to create drawer, err: %v", err)
	}
	drawer.keySigner = func(data []byte) ([]byte, error) {
		sig, _ := crypto.Sign(data, env.drawerKey)
		return sig, nil
	}
	// Reject all invalid issue operation
	env.issueAndCheck(t, drawer, big.NewInt(0), errors.New("invalid issue amount"), nil, nil)
	env.issueAndCheck(t, drawer, big.NewInt(100), ErrNotEnoughDeposit, nil, nil)

	// Deposit some funds into the contract
	drawer.Deposit(context.Background(), big.NewInt(200))
	env.issueAndCheck(t, drawer, big.NewInt(50), nil, big.NewInt(50), big.NewInt(150))
	lastIssued := env.issueAndCheck(t, drawer, big.NewInt(50), nil, big.NewInt(100), big.NewInt(100))

	// Cash all payments
	tx, err := drawer.book.contract.Cash(bind.NewKeyedTransactor(env.draweeKey), env.drawerAddr, lastIssued.Amount, lastIssued.Sig[64], common.BytesToHash(lastIssued.Sig[:32]), common.BytesToHash(lastIssued.Sig[32:64]))
	if err != nil {
		t.Fatalf("Failed to cash payment, err: %v", err)
	}
	bind.WaitMined(context.Background(), env.backend, tx)

	// Remove chequedb explictly
	drawer.cdb = newChequeDB(rawdb.NewMemoryDatabase())
	// Ensure we can repair the broken db!
	env.issueAndCheck(t, drawer, big.NewInt(50), nil, big.NewInt(150), big.NewInt(50))
	lastIssued = env.issueAndCheck(t, drawer, big.NewInt(50), nil, big.NewInt(200), big.NewInt(0))
	env.issueAndCheck(t, drawer, big.NewInt(50), ErrNotEnoughDeposit, nil, nil)

	// Remove chequedb explictly again!
	drawer.cdb = newChequeDB(rawdb.NewMemoryDatabase())
	// Amend the broken chequedb
	if err := drawer.Amend(lastIssued); err != nil {
		t.Fatalf("Failed to amend broken chequedb, err: %v", err)
	}
	unspent, err := drawer.Unspent()
	if err != nil {
		t.Fatalf("Failed to retrieve unspent part, err: %v", err)
	}
	if unspent.Uint64() != 0 {
		t.Fatalf("Failed to ament the chequedb")
	}
}

func TestWithdrawAndClaim(t *testing.T) {
	env := newTestEnv(t)
	defer env.close()

	var exit = make(chan struct{})
	defer close(exit)

	// Start the automatic blockchain.
	go func() {
		ticker := time.NewTicker(time.Millisecond * 100)
		for {
			select {
			case <-ticker.C:
				env.backend.Commit()
			case <-exit:
				return
			}
		}
	}()
	// Deploy the contract if missing
	drawee, err := NewChequeDrawee(env.draweeAddr, bind.NewKeyedTransactor(env.draweeKey), env.backend, env.backend, env.db)
	if err != nil {
		t.Fatalf("Faield to deploy contract, err: %v", err)
	}
	drawer, err := NewChequeDrawer(bind.NewKeyedTransactor(env.drawerKey), nil, env.drawerAddr, drawee.ContractAddr(), env.backend, env.backend, env.db)
	if err != nil {
		t.Fatalf("Faield to create drawer, err: %v", err)
	}
	drawer.keySigner = func(data []byte) ([]byte, error) {
		sig, _ := crypto.Sign(data, env.drawerKey)
		return sig, nil
	}
	// Try to withdraw money, but we have no deposit.
	if err := drawer.Withdraw(context.Background()); err == nil {
		t.Fatal("Failed to reject invalid withdrawal request")
	}
	drawer.Deposit(context.Background(), big.NewInt(200))

	if err := drawer.Withdraw(context.Background()); err != nil {
		t.Fatalf("Failed to open withdrawal request, err: %v", err)
	}
	if err := drawer.Withdraw(context.Background()); err == nil {
		t.Fatal("Duplicated withdrawal request should be rejected")
	}
	number := drawer.WithdrawalRecord()
	if number == 0 {
		t.Fatal("Withdrawal record should be set")
	}
	// After opening the withdrawal request, we can spend this part
	env.issueAndCheck(t, drawer, big.NewInt(100), ErrNotEnoughDeposit, nil, nil)
}
