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
	"crypto/ecdsa"
	"math/big"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/accounts/abi/bind/backends"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethdb"
)

type testEnv struct {
	db         ethdb.Database
	draweeKey  *ecdsa.PrivateKey
	draweeAddr common.Address
	drawerKey  *ecdsa.PrivateKey
	drawerAddr common.Address
	backend    *backends.SimulatedBackend
}

func newTestEnv(t *testing.T) *testEnv {
	db := rawdb.NewMemoryDatabase()
	key, _ := crypto.GenerateKey()
	key2, _ := crypto.GenerateKey()
	addr := crypto.PubkeyToAddress(key.PublicKey)
	addr2 := crypto.PubkeyToAddress(key2.PublicKey)
	sim := backends.NewSimulatedBackend(core.GenesisAlloc{addr: {Balance: big.NewInt(1000000000)}, addr2: {Balance: big.NewInt(1000000000)}}, 10000000)
	return &testEnv{
		db:         db,
		draweeKey:  key,
		draweeAddr: addr,
		drawerKey:  key2,
		drawerAddr: addr2,
		backend:    sim,
	}
}

func (env *testEnv) close() { env.backend.Close() }
func (env *testEnv) issueCheque(amount *big.Int, chanAddr common.Address) *Cheque {
	cheque := &Cheque{
		Drawer:       env.drawerAddr,
		ContractAddr: chanAddr,
		Amount:       amount,
	}
	cheque.signWithKey(func(digestHash []byte) ([]byte, error) {
		sig, _ := crypto.Sign(digestHash, env.drawerKey)
		return sig, nil
	})
	return cheque
}

func (env *testEnv) spendAndCheck(t *testing.T, drawee *ChequeDrawee, amount *big.Int, expectErr error, expectNet *big.Int, expectUnpaid *big.Int, expectUnspent *big.Int) {
	cheque := env.issueCheque(amount, drawee.ContractAddr())
	net, unpaid, err := drawee.AddCheque(cheque)
	if expectErr != nil {
		if err.Error() != expectErr.Error() {
			t.Fatalf("Error mismatch, want: %v, got: %v", expectErr, err)
		}
		return
	}
	if net.Cmp(expectNet) != 0 {
		t.Fatalf("Net amount mismatch, want: %v, got: %v", expectNet, net)
	}
	if unpaid.Cmp(expectUnpaid) != 0 {
		t.Fatalf("Unpaid amount mismatch, want: %v, got: %v", expectUnpaid, unpaid)
	}
	unspent, _ := drawee.Unspent(env.drawerAddr)
	if unspent.Cmp(expectUnspent) != 0 {
		t.Fatalf("Unspent amount mismatch, want: %v, got: %v", expectUnspent, unspent)
	}
}

func TestDeployment(t *testing.T) {
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
	addr := drawee.cdb.readContractAddr(env.draweeAddr)
	if addr == nil {
		t.Fatalf("Failed to deploy contract")
	}
	if *addr != drawee.book.address {
		t.Fatalf("Contract address mismatch, want: %v, got: %v", drawee.book.address, *addr)
	}
	// Restart, no deploy needed
	drawee, _ = NewChequeDrawee(env.draweeAddr, bind.NewKeyedTransactor(env.draweeKey), env.backend, env.backend, env.db)
	addr2 := drawee.cdb.readContractAddr(env.draweeAddr)
	if addr2 == nil {
		t.Fatalf("Failed to reload contract")
	}
	if *addr2 != *addr {
		t.Fatalf("Contract address mismatch, want: %v, got: %v", *addr, *addr2)
	}
	// Remove the db explicitly
	newdb := rawdb.NewMemoryDatabase()
	drawee, _ = NewChequeDrawee(env.draweeAddr, bind.NewKeyedTransactor(env.draweeKey), env.backend, env.backend, newdb)
	addr3 := drawee.cdb.readContractAddr(env.draweeAddr)
	if addr3 == nil {
		t.Fatalf("Failed to re-deploy contract")
	}
	if *addr3 == *addr {
		t.Fatalf("New contract address expected")
	}
	// Drawee changes the key, new contract should be deployed
	drawee, _ = NewChequeDrawee(env.drawerAddr, bind.NewKeyedTransactor(env.drawerKey), env.backend, env.backend, newdb)
	addr4 := drawee.cdb.readContractAddr(env.drawerAddr)
	if addr4 == nil {
		t.Fatalf("Failed to re-deploy contract")
	}
	if *addr4 == *addr3 {
		t.Fatalf("New contract address expected")
	}
}

func TestAddCheque(t *testing.T) {
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
	// Ensure we can reject all cheques which doesn't enough fund backup
	env.spendAndCheck(t, drawee, big.NewInt(100), ErrNotEnoughDeposit, nil, nil, nil)

	// Deposit enough money for drawer
	opt := bind.NewKeyedTransactor(env.drawerKey)
	opt.Value = big.NewInt(200)
	tx, _ := drawee.book.contract.Deposit(opt)
	bind.WaitMined(context.Background(), env.backend, tx)
	unspent, _ := drawee.Unspent(env.drawerAddr)
	if unspent.Uint64() != 200 {
		t.Fatalf("Balance mismatch")
	}
	// Spend 100, ensure it's successful
	env.spendAndCheck(t, drawee, big.NewInt(100), nil, big.NewInt(100), big.NewInt(100), big.NewInt(100))

	// Spend another 100, ensure it's also successful
	env.spendAndCheck(t, drawee, big.NewInt(200), nil, big.NewInt(100), big.NewInt(200), big.NewInt(0))

	// Remove the cheque db explicitly
	drawee.cdb = newChequeDB(rawdb.NewMemoryDatabase())

	// Drawer can double-spend the part which we haven't cashed.
	env.spendAndCheck(t, drawee, big.NewInt(100), nil, big.NewInt(100), big.NewInt(100), big.NewInt(100))
	env.spendAndCheck(t, drawee, big.NewInt(150), nil, big.NewInt(50), big.NewInt(150), big.NewInt(50))

	// Cash all received payments
	drawee.Cash(context.Background(), env.drawerAddr, true)
	// Remove the cheque db explicitly again
	drawee.cdb = newChequeDB(rawdb.NewMemoryDatabase())

	// We can repair the broken db and reject stale cheque
	env.spendAndCheck(t, drawee, big.NewInt(100), &StaleChequeError{Msg: "stale cheque"}, nil, nil, nil)
	env.spendAndCheck(t, drawee, big.NewInt(150), &StaleChequeError{Msg: "stale cheque"}, nil, nil, nil)
	env.spendAndCheck(t, drawee, big.NewInt(200), nil, big.NewInt(50), big.NewInt(50), big.NewInt(0))
	env.spendAndCheck(t, drawee, big.NewInt(250), ErrNotEnoughDeposit, nil, nil, nil)
}

// This function tests a special scenario:
// drawer deposits some money and then opens the withdrawal request quickly.
// Then it wants to spend some withdrawed money.
func TestWithdrawInAdvance(t *testing.T) {
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
	// Deposit enough money for drawer
	opt := bind.NewKeyedTransactor(env.drawerKey)
	opt.Value = big.NewInt(200)
	tx, _ := drawee.book.contract.Deposit(opt)
	bind.WaitMined(context.Background(), env.backend, tx)

	// Open withdrawal request immediately
	opt = bind.NewKeyedTransactor(env.drawerKey)
	tx, _ = drawee.book.contract.Withdraw(opt, big.NewInt(150))
	bind.WaitMined(context.Background(), env.backend, tx)

	// Ensure all withdrawed part can't be double spend
	env.spendAndCheck(t, drawee, big.NewInt(100), ErrNotEnoughDeposit, nil, nil, nil)
	env.spendAndCheck(t, drawee, big.NewInt(50), nil, big.NewInt(50), big.NewInt(50), big.NewInt(0))
}
