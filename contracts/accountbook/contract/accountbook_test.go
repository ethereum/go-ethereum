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

package contract

import (
	"context"
	"crypto/ecdsa"
	"fmt"
	"math/big"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/accounts/abi/bind/backends"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/crypto"
)

var (
	testChallengeTimeWindow = big.NewInt(4)
)

type Account struct {
	key  *ecdsa.PrivateKey
	addr common.Address
}

type testAccountBook struct {
	sim      *backends.SimulatedBackend
	contract *AccountBook
	address  common.Address
	owner    Account
	customer Account
}

func newAccount() Account {
	key, _ := crypto.GenerateKey()
	return Account{addr: crypto.PubkeyToAddress(key.PublicKey), key: key}
}

func newTestContract(t *testing.T) *testAccountBook {
	owner, customer := newAccount(), newAccount()

	sim := backends.NewSimulatedBackend(core.GenesisAlloc{owner.addr: {Balance: big.NewInt(1000000000)}, customer.addr: {Balance: big.NewInt(1000000000)}}, 10000000)
	transactOpts := bind.NewKeyedTransactor(owner.key)

	addr, _, c, err := DeployAccountBook(transactOpts, sim, testChallengeTimeWindow.Uint64())
	if err != nil {
		t.Error("Failed to deploy registrar contract", err)
	}
	sim.Commit()

	return &testAccountBook{
		sim:      sim,
		contract: c,
		address:  addr,
		owner:    owner,
		customer: customer,
	}
}

func (tester *testAccountBook) teardown() {
	tester.sim.Close()
}

func (tester *testAccountBook) listenBalanceChangeEvent(res chan bool, callback func(new *big.Int, old *big.Int) bool) {
	sink := make(chan *AccountBookBalanceChangedEvent)
	sub, err := tester.contract.WatchBalanceChangedEvent(nil, sink, nil)
	if err != nil {
		res <- false
		return
	}
	defer sub.Unsubscribe()

	// Check whether we receive the desired event
	select {
	case ev := <-sink:
		if !callback(ev.NewBalance, ev.OldBalance) {
			res <- false
			return
		}
	case <-time.NewTimer(time.Second).C:
		res <- false // Timeout
		return
	}
	// Ensure no more additional event receive
	select {
	case <-sink:
		res <- false
		return
	case <-time.NewTimer(100 * time.Millisecond).C:
		res <- true
		return
	}
}

func (tester *testAccountBook) listenWithdrawEvent(res chan bool, callback func(addr common.Address, amount *big.Int) bool) {
	sink := make(chan *AccountBookWithdrawEvent)
	sub, err := tester.contract.WatchWithdrawEvent(nil, sink, nil)
	if err != nil {
		res <- false
		return
	}
	defer sub.Unsubscribe()

	// Check whether we receive the desired event
	select {
	case ev := <-sink:
		if !callback(ev.Addr, ev.Amount) {
			res <- false
			return
		}
	case <-time.NewTimer(time.Second).C:
		res <- false // Timeout
		return
	}
	// Ensure no more additional event receive
	select {
	case <-sink:
		res <- false
		return
	case <-time.NewTimer(100 * time.Millisecond).C:
		res <- true
		return
	}
}

func (tester *testAccountBook) issueCheque(amount *big.Int) []byte {
	buf := make([]byte, 32)
	copy(buf[32-len(amount.Bytes()):], amount.Bytes())
	data := append([]byte{0x19, 0x00}, append(tester.address.Bytes(), buf...)...)
	sig, _ := crypto.Sign(crypto.Keccak256(data), tester.customer.key)
	sig[64] += 27 // Transform V from 0/1 to 27/28 according to the yellow paper
	return sig
}

func (tester *testAccountBook) commitEmptyBlocks(number int) {
	for i := 0; i < number; i++ {
		tester.sim.Commit()
	}
}

func (tester *testAccountBook) balance(address common.Address) *big.Int {
	balance, err := tester.sim.BalanceAt(context.Background(), address, nil)
	if err != nil {
		return nil
	}
	return balance
}

func (tester *testAccountBook) contractBalance() *big.Int {
	return tester.balance(tester.address)
}

func (tester *testAccountBook) ownerBalance() *big.Int {
	return tester.balance(tester.owner.addr)
}

func (tester *testAccountBook) customerBalance() *big.Int {
	return tester.balance(tester.customer.addr)
}

func TestDeposit(t *testing.T) {
	tester := newTestContract(t)
	defer tester.teardown()

	eventCh := make(chan bool, 1)
	go tester.listenBalanceChangeEvent(eventCh, func(new *big.Int, old *big.Int) bool {
		return new.Cmp(big.NewInt(10000)) == 0 && old.Cmp(big.NewInt(0)) == 0
	})

	// Deposit 10,000 wei
	opt := bind.NewKeyedTransactor(tester.customer.key)
	opt.Value = big.NewInt(10000)
	tester.contract.Deposit(opt)
	tester.sim.Commit()

	balance, err := tester.contract.Deposits(nil, tester.customer.addr)
	if err != nil {
		t.Fatalf("Failed to retrieve balanace: %v", err)
	}
	want := big.NewInt(10000)
	if balance.Cmp(want) != 0 {
		t.Fatalf("Balance mismtach, want: %d, got: %d", want, balance)
	}
	if !<-eventCh {
		t.Fatalf("Failed for balance change event")
	}
	if balance := tester.contractBalance(); balance == nil || balance.Cmp(want) != 0 {
		t.Fatalf("Contract balance mismatch, want: %d, got: %d", want, balance)
	}

	// Deposit 0 wei, we don't accept empty deposit
	opt.Value = nil
	_, err = tester.contract.Deposit(opt)
	if err == nil {
		t.Fatalf("Zero deposit should be rejected")
	}
}

func TestWithdraw(t *testing.T) {
	tester := newTestContract(t)
	defer tester.teardown()

	// Deposit 10,000 wei
	opt := bind.NewKeyedTransactor(tester.customer.key)
	opt.Value = big.NewInt(10000)
	tester.contract.Deposit(opt)
	tester.sim.Commit()

	eventCh := make(chan bool, 1)
	go tester.listenWithdrawEvent(eventCh, func(addr common.Address, amount *big.Int) bool {
		return addr == tester.customer.addr && amount.Cmp(big.NewInt(10000)) == 0
	})

	// Open the request for withdrawal
	opt.Value = nil
	tester.contract.Withdraw(opt, big.NewInt(10000)) // Withdraw all money
	tester.sim.Commit()
	if !<-eventCh {
		t.Fatalf("Failed for withdraw event")
	}

	// Check there is a open withdrawal request
	request, err := tester.contract.WithdrawRequests(nil, tester.customer.addr)
	if err != nil {
		t.Fatalf("Failed to retrieve withdraw request: %v", err)
	}
	if request.Amount.Cmp(big.NewInt(10000)) != 0 {
		t.Fatalf("Withdrawal amount mismatch, want: %d, got: %d", 10000, request.Amount)
	}

	// Try to claim the deposit before challenge
	tester.contract.Claim(opt)
	tester.sim.Commit()
	balance, err := tester.contract.Deposits(nil, tester.customer.addr)
	if err != nil {
		t.Fatalf("Failed to retrieve balanace: %v", err)
	}
	if balance.Cmp(big.NewInt(10000)) != 0 {
		t.Fatalf("Deposit can't be withdraw during the challenge period")
	}

	// Pass the challenge period and withdraw
	tester.commitEmptyBlocks(int(testChallengeTimeWindow.Int64()))

	go tester.listenBalanceChangeEvent(eventCh, func(new *big.Int, old *big.Int) bool {
		return old.Cmp(big.NewInt(10000)) == 0 && new.Cmp(big.NewInt(0)) == 0
	})
	tester.contract.Claim(opt)
	tester.sim.Commit()

	balance, err = tester.contract.Deposits(nil, tester.customer.addr)
	if err != nil {
		t.Fatalf("Failed to retrieve balanace: %v", err)
	}
	want := big.NewInt(0)
	if balance.Cmp(want) != 0 {
		t.Fatalf("Balance mismtach, want: %d, got: %d", want, balance)
	}
	if !<-eventCh {
		t.Fatalf("Failed for balance change event")
	}
	if balance := tester.contractBalance(); balance == nil || balance.Cmp(want) != 0 {
		t.Fatalf("Contract balance mismatch, want: %d, got: %d", want, balance)
	}
}

func TestCash(t *testing.T) {
	tester := newTestContract(t)
	defer tester.teardown()

	// Deposit 10,000 wei
	customerOpt := bind.NewKeyedTransactor(tester.customer.key)
	customerOpt.Value = big.NewInt(10000)
	tester.contract.Deposit(customerOpt)
	tester.sim.Commit()

	// Customer issues a cheque with amount 1000
	sig := tester.issueCheque(big.NewInt(1000))
	ownerOpt := bind.NewKeyedTransactor(tester.owner.key)
	_, err := tester.contract.Cash(ownerOpt, tester.customer.addr, big.NewInt(1000), sig[64], common.BytesToHash(sig[:32]), common.BytesToHash(sig[32:64]))
	if err != nil {
		t.Fatalf("Failed to cash the signed cheque: %v", err)
	}
	tester.sim.Commit()
	balance, err := tester.contract.Deposits(nil, tester.customer.addr)
	if err != nil {
		t.Fatalf("Failed to retrieve balanace: %v", err)
	}
	want := big.NewInt(9000)
	if balance.Cmp(want) != 0 {
		t.Fatalf("Balance mismtach, want: %d, got: %d", want, balance)
	}
	// The stored money in contract should also changed
	if balance := tester.contractBalance(); balance == nil || balance.Cmp(want) != 0 {
		t.Fatalf("Contract balance mismatch, want: %d, got: %d", want, balance)
	}

	// Try to double-cash, prevent it.
	_, err = tester.contract.Cash(ownerOpt, tester.customer.addr, big.NewInt(1000), sig[64], common.BytesToHash(sig[:32]), common.BytesToHash(sig[32:64]))
	if err == nil {
		t.Fatalf("Double-cash should be prevent")
	}
}

func TestChallenge(t *testing.T) {
	tester := newTestContract(t)
	defer tester.teardown()

	// Deposit 10,000 wei
	customerOpt := bind.NewKeyedTransactor(tester.customer.key)
	customerOpt.Value = big.NewInt(10000)
	tester.contract.Deposit(customerOpt)
	tester.sim.Commit()

	// Customer issues a cheque with amount 1000
	sig := tester.issueCheque(big.NewInt(1000))

	// Customer tries to withdraw all money including the spent part
	customerOpt.Value = nil
	_, err := tester.contract.Withdraw(customerOpt, big.NewInt(10000))
	if err != nil {
		t.Fatalf("Failed to open withdraw request: %v", err)
	}
	tester.sim.Commit()

	// Owner sumbits the evidence to cash the "spent" money
	ownerOpt := bind.NewKeyedTransactor(tester.owner.key)
	_, err = tester.contract.Cash(ownerOpt, tester.customer.addr, big.NewInt(1000), sig[64], common.BytesToHash(sig[:32]), common.BytesToHash(sig[32:64]))
	if err != nil {
		t.Fatalf("Failed to cash the signed cheque: %v", err)
	}
	tester.sim.Commit()

	tester.commitEmptyBlocks(int(testChallengeTimeWindow.Int64()))

	eventCh := make(chan bool, 1)
	go tester.listenBalanceChangeEvent(eventCh, func(new *big.Int, old *big.Int) bool {
		return old.Cmp(big.NewInt(9000)) == 0 && new.Cmp(big.NewInt(0)) == 0
	})
	tester.contract.Claim(customerOpt)
	tester.sim.Commit()
	if !<-eventCh {
		t.Fatalf("Failed for balance change event")
	}

	want := big.NewInt(0)
	if balance := tester.contractBalance(); balance == nil || balance.Cmp(want) != 0 {
		t.Fatalf("Contract balance mismatch, want: %d, got: %d", want, balance)
	}
}

func TestChallengeOutOfWindow(t *testing.T) {
	tester := newTestContract(t)
	defer tester.teardown()

	// Deposit 10,000 wei
	customerOpt := bind.NewKeyedTransactor(tester.customer.key)
	customerOpt.Value = big.NewInt(10000)
	tester.contract.Deposit(customerOpt)
	tester.sim.Commit()

	// Customer issues a cheque with amount 1000
	sig := tester.issueCheque(big.NewInt(1000))

	// Customer tries to withdraw all money including the spent part
	customerOpt.Value = nil
	_, err := tester.contract.Withdraw(customerOpt, big.NewInt(10000))
	if err != nil {
		t.Fatalf("Failed to open withdraw request: %v", err)
	}
	tester.sim.Commit()
	tester.commitEmptyBlocks(int(testChallengeTimeWindow.Int64()))

	// Now all the money has been claimed.
	tester.contract.Claim(customerOpt)
	tester.sim.Commit()

	// Owner sumbits the evidence to cash the "spent" money, but it's
	// too late.
	ownerBalanceOld := tester.ownerBalance()
	ownerOpt := bind.NewKeyedTransactor(tester.owner.key)
	ownerOpt.GasPrice = big.NewInt(0)
	_, err = tester.contract.Cash(ownerOpt, tester.customer.addr, big.NewInt(1000), sig[64], common.BytesToHash(sig[:32]), common.BytesToHash(sig[32:64]))
	if err != nil {
		t.Fatalf("Failed to cash the signed cheque: %v", err)
	}
	tester.sim.Commit()

	ownerBalanceNew := tester.ownerBalance()
	if ownerBalanceNew.Cmp(ownerBalanceOld) != 0 {
		t.Fatalf("All claimed money should be cashed")
	}
	paid, err := tester.contract.Paids(nil, tester.customer.addr)
	if err != nil {
		t.Fatalf("Failed to retrieve paid amount: %v", err)
	}
	if paid.Cmp(big.NewInt(1000)) != 0 {
		t.Fatal("The paid amount should be set to 1000 even the challenge time window is out")
	}
}

func TestGasUsed(t *testing.T) {
	tester := newTestContract(t)
	defer tester.teardown()

	// Deposit 10,000 wei
	customerOpt := bind.NewKeyedTransactor(tester.customer.key)
	customerOpt.Value = big.NewInt(10000)
	tx, _ := tester.contract.Deposit(customerOpt)
	tester.sim.Commit()

	r, _ := tester.sim.TransactionReceipt(context.Background(), tx.Hash())
	fmt.Println("Deposit => gas used:", r.GasUsed)

	// Second deposit
	customerOpt.Value = big.NewInt(10000)
	tx, _ = tester.contract.Deposit(customerOpt)
	tester.sim.Commit()

	r, _ = tester.sim.TransactionReceipt(context.Background(), tx.Hash())
	fmt.Println("Deposit 2 => gas used:", r.GasUsed)

	// Customer issues a cheque with amount 1000
	sig := tester.issueCheque(big.NewInt(1000))

	// Customer tries to withdraw all money including the spent part
	customerOpt.Value = nil
	tx, _ = tester.contract.Withdraw(customerOpt, big.NewInt(9000))
	tester.sim.Commit()

	r, _ = tester.sim.TransactionReceipt(context.Background(), tx.Hash())
	fmt.Println("Withdraw request => gas used:", r.GasUsed)

	tester.commitEmptyBlocks(int(testChallengeTimeWindow.Int64()))

	// Now all the money has been claimed.
	tx, _ = tester.contract.Claim(customerOpt)
	tester.sim.Commit()
	r, _ = tester.sim.TransactionReceipt(context.Background(), tx.Hash())
	fmt.Println("Deposit claim => gas used:", r.GasUsed)

	//
	ownerOpt := bind.NewKeyedTransactor(tester.owner.key)
	tx, _ = tester.contract.Cash(ownerOpt, tester.customer.addr, big.NewInt(1000), sig[64], common.BytesToHash(sig[:32]), common.BytesToHash(sig[32:64]))
	tester.sim.Commit()
	r, _ = tester.sim.TransactionReceipt(context.Background(), tx.Hash())
	fmt.Println("Cash cheque => gas used:", r.GasUsed)
}
