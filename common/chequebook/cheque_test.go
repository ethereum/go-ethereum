package chequebook

import (
	"math/big"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
)

type testBackend struct {
	calls []string
	errs  []error
	txs   []string
}

func newTestBackend() *testBackend {
	return &testBackend{}
}

func (b *testBackend) Transact(fromStr, toStr, nonceStr, valueStr, gasStr, gasPriceStr, codeStr string) (string, error) {
	txhash := string(crypto.Sha3([]byte(codeStr)))
	b.txs = append(b.txs, txhash)
	return txhash, nil
}

func (b *testBackend) Call(fromStr, toStr, valueStr, gasStr, gasPriceStr, codeStr string) (string, string, error) {
	if len(b.calls) == 0 {
		panic("test backend called too many times")
	}
	res := b.calls[0]
	err := b.errs[0]
	b.calls = b.calls[1:]
	b.errs = b.errs[1:]
	return res, "", err
}

func (b *testBackend) GetTxReceipt(txhash common.Hash) *types.Receipt {
	return nil
}

func (b *testBackend) CodeAt(address string) string {
	return ""
}

func genAddr() common.Address {
	prvKey, _ := crypto.GenerateKey()
	return crypto.PubkeyToAddress(prvKey.PublicKey)
}

func TestIssueAndReceive(t *testing.T) {
	prvKey, _ := crypto.GenerateKey()
	sender := genAddr()
	path := "/tmp/checkbook.json"
	chbook, err := NewChequebook(path, sender, prvKey, nil)
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	recipient := genAddr()
	chbook.sent[recipient] = new(big.Int).SetUint64(42)
	amount := common.Big1
	ch, err := chbook.Issue(recipient, amount)
	if err == nil {
		t.Errorf("expected insufficient funds error, got none")
	}

	chbook.balance = new(big.Int).Set(common.Big1)
	if chbook.Balance().Cmp(common.Big1) != 0 {
		t.Errorf("expected: %v, got %v", "0", chbook.Balance())
	}

	ch, err = chbook.Issue(recipient, amount)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if chbook.Balance().Cmp(common.Big0) != 0 {
		t.Errorf("expected: %v, got %v", "0", chbook.Balance())
	}

	backend := newTestBackend()
	backend.calls = []string{common.ToHex(big.NewInt(42).Bytes())}
	backend.errs = []error{nil}
	chbox, err := NewInbox(sender, recipient, recipient, &prvKey.PublicKey, backend)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	received, err := chbox.Receive(ch)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if received.Cmp(common.Big1) != 0 {
		t.Errorf("expected: %v, got %v", "1", received)
	}

}

func TestCheckbookFile(t *testing.T) {
	prvKey, _ := crypto.GenerateKey()
	sender := genAddr()
	path := "/tmp/checkbook.json"
	chbook, err := NewChequebook(path, sender, prvKey, nil)
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	recipient := genAddr()
	chbook.sent[recipient] = new(big.Int).SetUint64(42)
	chbook.balance = new(big.Int).Set(common.Big1)

	chbook.Save()

	chbook, err = LoadChequebook(path, prvKey, nil)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if chbook.Balance().Cmp(common.Big1) != 0 {
		t.Errorf("expected: %v, got %v", "0", chbook.Balance())
	}

	ch, err := chbook.Issue(recipient, common.Big1)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if ch.Amount.Cmp(new(big.Int).SetUint64(43)) != 0 {
		t.Errorf("expected: %v, got %v", "0", ch.Amount)
	}

	err = chbook.Save()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestVerifyErrors(t *testing.T) {
	prvKey, _ := crypto.GenerateKey()
	sender0 := genAddr()
	sender1 := genAddr()
	path0 := "/tmp/checkbook0.json"
	chbook0, err := NewChequebook(path0, sender0, prvKey, nil)
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	path1 := "/tmp/checkbook1.json"
	chbook1, err := NewChequebook(path1, sender1, prvKey, nil)
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	recipient0 := genAddr()
	recipient1 := genAddr()
	chbook0.balance = new(big.Int).Set(common.Big2)
	chbook1.balance = new(big.Int).Set(common.Big1)
	chbook0.sent[recipient0] = new(big.Int).SetUint64(42)
	amount := common.Big1
	ch0, err := chbook0.Issue(recipient0, amount)

	backend := newTestBackend()
	backend.calls = []string{common.ToHex(big.NewInt(42).Bytes())}
	backend.errs = []error{nil}
	chbox, err := NewInbox(sender0, recipient0, recipient0, &prvKey.PublicKey, backend)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	received, err := chbox.Receive(ch0)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if received.Cmp(common.Big1) != 0 {
		t.Errorf("expected: %v, got %v", "1", received)
	}

	ch1, err := chbook0.Issue(recipient1, amount)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	received, err = chbox.Receive(ch1)
	t.Log(err)
	if err == nil {
		t.Fatalf("expected receiver error, got none")
	}

	ch2, err := chbook1.Issue(recipient0, amount)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	received, err = chbox.Receive(ch2)
	t.Log(err)
	if err == nil {
		t.Fatalf("expected sender error, got none")
	}

	_, err = chbook1.Issue(recipient0, new(big.Int).SetInt64(-1))
	t.Log(err)
	if err == nil {
		t.Fatalf("expected incorrect amount error, got none")
	}

	received, err = chbox.Receive(ch0)
	t.Log(err)
	if err == nil {
		t.Fatalf("expected incorrect amount error, got none")
	}

}

func TestDeposit(t *testing.T) {
	prvKey, _ := crypto.GenerateKey()
	sender := genAddr()
	path := "/tmp/checkbook.json"

	backend := newTestBackend()
	chbook, err := NewChequebook(path, sender, prvKey, backend)
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	balance := new(big.Int).SetUint64(42)
	chbook.Deposit(balance)
	if len(backend.txs) != 1 {
		t.Fatalf("expected 1 txs to send, got %v", len(backend.txs))
	}
	if chbook.balance.Cmp(balance) != 0 {
		t.Fatalf("expected balance %v, got %v", balance, chbook.balance)
	}

	recipient := genAddr()
	amount := common.Big1
	_, err = chbook.Issue(recipient, amount)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	exp := new(big.Int).SetUint64(41)
	if chbook.balance.Cmp(exp) != 0 {
		t.Fatalf("expected balance %v, got %v", exp, chbook.balance)
	}

	// autodeposit on each issue
	chbook.AutoDeposit(0, balance, balance)
	_, err = chbook.Issue(recipient, amount)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	_, err = chbook.Issue(recipient, amount)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(backend.txs) != 3 {
		t.Fatalf("expected 3 txs to send, got %v", len(backend.txs))
	}
	if chbook.balance.Cmp(balance) != 0 {
		t.Fatalf("expected balance %v, got %v", balance, chbook.balance)
	}

	// autodeposit off
	chbook.AutoDeposit(0, common.Big0, balance)
	_, err = chbook.Issue(recipient, amount)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	_, err = chbook.Issue(recipient, amount)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(backend.txs) != 3 {
		t.Fatalf("expected 3 txs to send, got %v", len(backend.txs))
	}
	exp = new(big.Int).SetUint64(40)
	if chbook.balance.Cmp(exp) != 0 {
		t.Fatalf("expected balance %v, got %v", exp, chbook.balance)
	}

	// autodeposit every 10ms if new cheque issued
	interval := 30 * time.Millisecond
	chbook.AutoDeposit(interval, common.Big1, balance)
	_, err = chbook.Issue(recipient, amount)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	_, err = chbook.Issue(recipient, amount)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(backend.txs) != 3 {
		t.Fatalf("expected 3 txs to send, got %v", len(backend.txs))
	}
	exp = new(big.Int).SetUint64(38)
	if chbook.balance.Cmp(exp) != 0 {
		t.Fatalf("expected balance %v, got %v", exp, chbook.balance)
	}

	time.Sleep(3 * interval)
	if len(backend.txs) != 4 {
		t.Fatalf("expected 4 txs to send, got %v", len(backend.txs))
	}
	if chbook.balance.Cmp(balance) != 0 {
		t.Fatalf("expected balance %v, got %v", balance, chbook.balance)
	}

	exp = new(big.Int).SetUint64(40)
	chbook.AutoDeposit(4*interval, exp, balance)
	_, err = chbook.Issue(recipient, amount)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	_, err = chbook.Issue(recipient, amount)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	time.Sleep(3 * interval)
	if len(backend.txs) != 4 {
		t.Fatalf("expected 4 txs to send, got %v", len(backend.txs))
	}
	if chbook.balance.Cmp(exp) != 0 {
		t.Fatalf("expected balance %v, got %v", exp, chbook.balance)
	}

	_, err = chbook.Issue(recipient, amount)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	time.Sleep(1 * interval)

	if len(backend.txs) != 5 {
		t.Fatalf("expected 5 txs to send, got %v", len(backend.txs))
	}
	if chbook.balance.Cmp(balance) != 0 {
		t.Fatalf("expected balance %v, got %v", balance, chbook.balance)
	}

	chbook.AutoDeposit(1*interval, common.Big0, balance)
	chbook.Stop()

	_, err = chbook.Issue(recipient, common.Big1)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	_, err = chbook.Issue(recipient, common.Big2)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	time.Sleep(1 * interval)

	if len(backend.txs) != 5 {
		t.Fatalf("expected 5 txs to send, got %v", len(backend.txs))
	}
	exp = new(big.Int).SetUint64(39)
	if chbook.balance.Cmp(exp) != 0 {
		t.Fatalf("expected balance %v, got %v", exp, chbook.balance)
	}

}

func TestCash(t *testing.T) {
	prvKey, _ := crypto.GenerateKey()
	sender := genAddr()
	path := "/tmp/checkbook.json"
	chbook, err := NewChequebook(path, sender, prvKey, nil)
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	recipient := genAddr()
	chbook.sent[recipient] = new(big.Int).SetUint64(42)
	amount := common.Big1
	chbook.balance = new(big.Int).Set(common.Big1)
	ch, err := chbook.Issue(recipient, amount)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	backend := newTestBackend()
	backend.calls = []string{common.ToHex(big.NewInt(42).Bytes())}
	backend.errs = []error{nil}
	chbox, err := NewInbox(sender, recipient, recipient, &prvKey.PublicKey, backend)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// cashing latest cheque
	_, err = chbox.Receive(ch)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	_, err = ch.Cash(recipient, backend)
	if len(backend.txs) != 1 {
		t.Fatalf("expected 1 txs to send, got %v", len(backend.txs))
	}

	chbook.balance = new(big.Int).Set(common.Big3)
	ch0, err := chbook.Issue(recipient, amount)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	ch1, err := chbook.Issue(recipient, amount)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	interval := 10 * time.Millisecond
	// setting autocash with interval of 10ms
	chbox.AutoCash(interval, nil)
	_, err = chbox.Receive(ch0)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	_, err = chbox.Receive(ch1)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	// after < interval time and 2 cheques received, no new cashing tx is sent
	if len(backend.txs) != 1 {
		t.Fatalf("expected 1 txs to send, got %v", len(backend.txs))
	}
	// after 3x interval time and 2 cheques received, exactly one cashing tx is sent
	time.Sleep(4 * interval)
	if len(backend.txs) != 2 {
		t.Fatalf("expected 2 txs to send, got %v", len(backend.txs))
	}

	// after stopping autocash no more tx are sent
	ch2, err := chbook.Issue(recipient, amount)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	chbox.Stop()
	time.Sleep(interval) // make sure loop stops
	_, err = chbox.Receive(ch2)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	time.Sleep(2 * interval)
	if len(backend.txs) != 2 {
		t.Fatalf("expected 2 txs to send, got %v", len(backend.txs))
	}

	// autocash below 1
	chbook.balance = new(big.Int).Set(common.Big2)
	chbox.AutoCash(0, common.Big1)

	ch3, err := chbook.Issue(recipient, amount)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	ch4, err := chbook.Issue(recipient, amount)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	_, err = chbox.Receive(ch3)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	_, err = chbox.Receive(ch4)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// 2 checks of amount 1 received, exactly 1 tx is sent
	if len(backend.txs) != 3 {
		t.Fatalf("expected 3 txs to send, got %v", len(backend.txs))
	}

	// autochash on receipt when maxUncashed is 0
	chbook.balance = new(big.Int).Set(common.Big2)
	chbox.AutoCash(0, common.Big0)

	ch5, err := chbook.Issue(recipient, amount)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	ch6, err := chbook.Issue(recipient, amount)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	_, err = chbox.Receive(ch5)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	_, err = chbox.Receive(ch6)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(backend.txs) != 5 {
		t.Fatalf("expected 5 txs to send, got %v", len(backend.txs))
	}

}
