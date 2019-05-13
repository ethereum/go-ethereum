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
	"math/big"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
)

type testInPayment struct {
	received         []*testPromise
	autocashInterval time.Duration
	autocashLimit    *big.Int
}

type testPromise struct {
	amount *big.Int
}

func (test *testInPayment) Receive(promise Promise) (*big.Int, error) {
	p := promise.(*testPromise)
	test.received = append(test.received, p)
	return p.amount, nil
}

func (test *testInPayment) AutoCash(interval time.Duration, limit *big.Int) {
	test.autocashInterval = interval
	test.autocashLimit = limit
}

func (test *testInPayment) Cash() (string, error) { return "", nil }

func (test *testInPayment) Stop() {}

type testOutPayment struct {
	deposits             []*big.Int
	autodepositInterval  time.Duration
	autodepositThreshold *big.Int
	autodepositBuffer    *big.Int
}

func (test *testOutPayment) Issue(amount *big.Int) (promise Promise, err error) {
	return &testPromise{amount}, nil
}

func (test *testOutPayment) Deposit(amount *big.Int) (string, error) {
	test.deposits = append(test.deposits, amount)
	return "", nil
}

func (test *testOutPayment) AutoDeposit(interval time.Duration, threshold, buffer *big.Int) {
	test.autodepositInterval = interval
	test.autodepositThreshold = threshold
	test.autodepositBuffer = buffer
}

func (test *testOutPayment) Stop() {}

type testProtocol struct {
	drop     bool
	amounts  []int
	promises []*testPromise
}

func (test *testProtocol) Drop() {
	test.drop = true
}

func (test *testProtocol) String() string {
	return ""
}

func (test *testProtocol) Pay(amount int, promise Promise) {
	p := promise.(*testPromise)
	test.promises = append(test.promises, p)
	test.amounts = append(test.amounts, amount)
}

func TestSwap(t *testing.T) {

	strategy := &Strategy{
		AutoCashInterval:     1 * time.Second,
		AutoCashThreshold:    big.NewInt(20),
		AutoDepositInterval:  1 * time.Second,
		AutoDepositThreshold: big.NewInt(20),
		AutoDepositBuffer:    big.NewInt(40),
	}

	local := &Params{
		Profile: &Profile{
			PayAt:  5,
			DropAt: 10,
			BuyAt:  common.Big3,
			SellAt: common.Big2,
		},
		Strategy: strategy,
	}

	in := &testInPayment{}
	out := &testOutPayment{}
	proto := &testProtocol{}

	swap, _ := New(local, Payment{In: in, Out: out, Buys: true, Sells: true}, proto)

	if in.autocashInterval != strategy.AutoCashInterval {
		t.Fatalf("autocash interval not properly set, expect %v, got %v", strategy.AutoCashInterval, in.autocashInterval)
	}
	if out.autodepositInterval != strategy.AutoDepositInterval {
		t.Fatalf("autodeposit interval not properly set, expect %v, got %v", strategy.AutoDepositInterval, out.autodepositInterval)
	}

	remote := &Profile{
		PayAt:  3,
		DropAt: 10,
		BuyAt:  common.Big2,
		SellAt: common.Big3,
	}
	swap.SetRemote(remote)

	swap.Add(9)
	if proto.drop {
		t.Fatalf("not expected peer to be dropped")
	}
	swap.Add(1)
	if !proto.drop {
		t.Fatalf("expected peer to be dropped")
	}
	if !proto.drop {
		t.Fatalf("expected peer to be dropped")
	}
	proto.drop = false

	swap.Receive(10, &testPromise{big.NewInt(20)})
	if swap.balance != 0 {
		t.Fatalf("expected zero balance, got %v", swap.balance)
	}

	if len(proto.amounts) != 0 {
		t.Fatalf("expected zero balance, got %v", swap.balance)
	}

	swap.Add(-2)
	if len(proto.amounts) > 0 {
		t.Fatalf("expected no payments yet, got %v", proto.amounts)
	}

	swap.Add(-1)
	if len(proto.amounts) != 1 {
		t.Fatalf("expected one payment, got %v", len(proto.amounts))
	}

	if proto.amounts[0] != 3 {
		t.Fatalf("expected payment for %v units, got %v", proto.amounts[0], 3)
	}

	exp := new(big.Int).Mul(big.NewInt(int64(proto.amounts[0])), remote.SellAt)
	if proto.promises[0].amount.Cmp(exp) != 0 {
		t.Fatalf("expected payment amount %v, got %v", exp, proto.promises[0].amount)
	}

	swap.SetParams(&Params{
		Profile: &Profile{
			PayAt:  5,
			DropAt: 10,
			BuyAt:  common.Big3,
			SellAt: common.Big2,
		},
		Strategy: &Strategy{
			AutoCashInterval:     2 * time.Second,
			AutoCashThreshold:    big.NewInt(40),
			AutoDepositInterval:  2 * time.Second,
			AutoDepositThreshold: big.NewInt(40),
			AutoDepositBuffer:    big.NewInt(60),
		},
	})

}
