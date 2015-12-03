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

func (self *testInPayment) Receive(promise Promise) (*big.Int, error) {
	p := promise.(*testPromise)
	self.received = append(self.received, p)
	return p.amount, nil
}

func (self *testInPayment) AutoCash(interval time.Duration, limit *big.Int) {
	self.autocashInterval = interval
	self.autocashLimit = limit
}

func (self *testInPayment) Cash() (string, error) { return "", nil }

func (self *testInPayment) Stop() {}

type testOutPayment struct {
	deposits             []*big.Int
	autodepositInterval  time.Duration
	autodepositThreshold *big.Int
	autodepositBuffer    *big.Int
}

func (self *testOutPayment) Issue(amount *big.Int) (promise Promise, err error) {
	return &testPromise{amount}, nil
}

func (self *testOutPayment) Deposit(amount *big.Int) (string, error) {
	self.deposits = append(self.deposits, amount)
	return "", nil
}

func (self *testOutPayment) AutoDeposit(interval time.Duration, threshold, buffer *big.Int) {
	self.autodepositInterval = interval
	self.autodepositThreshold = threshold
	self.autodepositBuffer = buffer
}

func (self *testOutPayment) Stop() {}

type testProtocol struct {
	drop     bool
	amounts  []int
	promises []*testPromise
}

func (self *testProtocol) Drop() {
	self.drop = true
}

func (self *testProtocol) String() string {
	return ""
}

func (self *testProtocol) Pay(amount int, promise Promise) {
	p := promise.(*testPromise)
	self.promises = append(self.promises, p)
	self.amounts = append(self.amounts, amount)
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
		t.Fatalf("autocash interval not properly set, expect %v, got ", strategy.AutoCashInterval, in.autocashInterval)
	}
	if out.autodepositInterval != strategy.AutoDepositInterval {
		t.Fatalf("autodeposit interval not properly set, expect %v, got ", strategy.AutoDepositInterval, out.autodepositInterval)
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
		t.Fatalf("expected payment for %v units, got %v", proto.amounts[0])
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
