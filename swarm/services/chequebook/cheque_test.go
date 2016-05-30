package chequebook

import (
	"crypto/ecdsa"
	"math/big"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/accounts/abi/bind/backends"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/swarm/services/chequebook/contract"
)

var (
	key0, _ = crypto.HexToECDSA("b71c71a67e1177ad4e901695e1b4b9ee17ae16c6668d313eac2f96dbcda3f291")
	key1, _ = crypto.HexToECDSA("8a1f9a8f95be41cd7ccb6168179afb4504aefe388d1e14474d32c45c72ce7b7a")
	key2, _ = crypto.HexToECDSA("49a7b37aa6f6645917e7b807e9d1c00d4fa71f18343b0d4122a4d2df64dd6fee")
	addr0   = crypto.PubkeyToAddress(key0.PublicKey)
	addr1   = crypto.PubkeyToAddress(key1.PublicKey)
	addr2   = crypto.PubkeyToAddress(key2.PublicKey)
)

type testBackend struct {
	*backends.SimulatedBackend
}

func accounts() []core.GenesisAccount {

	return []core.GenesisAccount{
		core.GenesisAccount{addr0, big.NewInt(1000000000)},
		core.GenesisAccount{addr1, big.NewInt(1000000000)},
		core.GenesisAccount{addr2, big.NewInt(1000000000)},
	}
}

func newTestBackend() *testBackend {
	accs := accounts()
	return &testBackend{SimulatedBackend: backends.NewSimulatedBackend(accs...)}
}

func (b *testBackend) GetTxReceipt(txhash common.Hash) (map[string]interface{}, error) {
	return nil, nil
}

func (b *testBackend) CodeAt(address common.Address) (string, error) {
	return "", nil
}

func (b *testBackend) BalanceAt(address common.Address) (*big.Int, error) {
	return big.NewInt(0), nil
}

func deploy(prvKey *ecdsa.PrivateKey, amount *big.Int, backend *backends.SimulatedBackend) (common.Address, error) {
	deployTransactor := bind.NewKeyedTransactor(prvKey)
	deployTransactor.Value = amount
	addr, _, _, err := contract.DeployChequebook(deployTransactor, backend)
	if err != nil {
		return common.Address{}, err
	}
	backend.Commit()
	return addr, nil
}

func TestIssueAndReceive(t *testing.T) {
	path := "/tmp/checkbook.json"
	backend := newTestBackend()
	addr0, err := deploy(key0, big.NewInt(0), backend.SimulatedBackend)
	if err != nil {
		t.Fatalf("deploy contract: expected no error, got %v", err)
	}
	chbook, err := NewChequebook(path, addr0, key0, backend)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	chbook.sent[addr1] = new(big.Int).SetUint64(42)
	amount := common.Big1
	ch, err := chbook.Issue(addr1, amount)
	if err == nil {
		t.Fatalf("expected insufficient funds error, got none")
	}

	chbook.balance = new(big.Int).Set(common.Big1)
	if chbook.Balance().Cmp(common.Big1) != 0 {
		t.Fatalf("expected: %v, got %v", "0", chbook.Balance())
	}

	ch, err = chbook.Issue(addr1, amount)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if chbook.Balance().Cmp(common.Big0) != 0 {
		t.Errorf("expected: %v, got %v", "0", chbook.Balance())
	}

	chbox, err := NewInbox(key1, addr0, addr1, &key0.PublicKey, backend)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	received, err := chbox.Receive(ch)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if received.Cmp(big.NewInt(43)) != 0 {
		t.Errorf("expected: %v, got %v", "43", received)
	}

}

func TestCheckbookFile(t *testing.T) {
	path := "/tmp/checkbook.json"
	backend := newTestBackend()
	chbook, err := NewChequebook(path, addr0, key0, backend)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	chbook.sent[addr1] = new(big.Int).SetUint64(42)
	chbook.balance = new(big.Int).Set(common.Big1)

	chbook.Save()

	chbook, err = LoadChequebook(path, key0, backend, false)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if chbook.Balance().Cmp(common.Big1) != 0 {
		t.Errorf("expected: %v, got %v", "0", chbook.Balance())
	}

	ch, err := chbook.Issue(addr1, common.Big1)
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
	path0 := "/tmp/checkbook0.json"
	backend := newTestBackend()
	contr0, err := deploy(key0, common.Big2, backend.SimulatedBackend)
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	chbook0, err := NewChequebook(path0, contr0, key0, backend)
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	path1 := "/tmp/checkbook1.json"
	contr1, err := deploy(key1, common.Big2, backend.SimulatedBackend)
	chbook1, err := NewChequebook(path1, contr1, key1, backend)
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	chbook0.sent[addr1] = new(big.Int).SetUint64(42)
	chbook0.balance = new(big.Int).Set(common.Big2)
	chbook1.balance = new(big.Int).Set(common.Big1)
	amount := common.Big1
	ch0, err := chbook0.Issue(addr1, amount)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	time.Sleep(5)
	chbox, err := NewInbox(key1, contr0, addr1, &key0.PublicKey, backend)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	received, err := chbox.Receive(ch0)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if received.Cmp(big.NewInt(43)) != 0 {
		t.Errorf("expected: %v, got %v", "43", received)
	}

	ch1, err := chbook0.Issue(addr2, amount)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	received, err = chbox.Receive(ch1)
	t.Logf("correct error: %v", err)
	if err == nil {
		t.Fatalf("expected receiver error, got none")
	}

	ch2, err := chbook1.Issue(addr1, amount)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	received, err = chbox.Receive(ch2)
	t.Logf("correct error: %v", err)
	if err == nil {
		t.Fatalf("expected sender error, got none")
	}

	_, err = chbook1.Issue(addr1, new(big.Int).SetInt64(-1))
	t.Logf("correct error: %v", err)
	if err == nil {
		t.Fatalf("expected incorrect amount error, got none")
	}

	received, err = chbox.Receive(ch0)
	t.Logf("correct error: %v", err)
	if err == nil {
		t.Fatalf("expected incorrect amount error, got none")
	}

}

func TestDeposit(t *testing.T) {
	path0 := "/tmp/checkbook0.json"
	backend := newTestBackend()
	contr0, err := deploy(key0, common.Big2, backend.SimulatedBackend)
	chbook, err := NewChequebook(path0, contr0, key0, backend)
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	balance := new(big.Int).SetUint64(42)
	chbook.Deposit(balance)
	backend.Commit()
	if chbook.balance.Cmp(balance) != 0 {
		t.Fatalf("expected balance %v, got %v", balance, chbook.balance)
	}

	amount := common.Big1
	_, err = chbook.Issue(addr1, amount)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	backend.Commit()
	exp := new(big.Int).SetUint64(41)
	if chbook.balance.Cmp(exp) != 0 {
		t.Fatalf("expected balance %v, got %v", exp, chbook.balance)
	}

	// autodeposit on each issue
	chbook.AutoDeposit(0, balance, balance)
	_, err = chbook.Issue(addr1, amount)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	backend.Commit()
	_, err = chbook.Issue(addr1, amount)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	backend.Commit()
	if chbook.balance.Cmp(balance) != 0 {
		t.Fatalf("expected balance %v, got %v", balance, chbook.balance)
	}

	// autodeposit off
	chbook.AutoDeposit(0, common.Big0, balance)
	_, err = chbook.Issue(addr1, amount)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	backend.Commit()
	_, err = chbook.Issue(addr1, amount)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	backend.Commit()

	exp = new(big.Int).SetUint64(40)
	if chbook.balance.Cmp(exp) != 0 {
		t.Fatalf("expected balance %v, got %v", exp, chbook.balance)
	}

	// autodeposit every 10ms if new cheque issued
	interval := 30 * time.Millisecond
	chbook.AutoDeposit(interval, common.Big1, balance)
	_, err = chbook.Issue(addr1, amount)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	backend.Commit()
	_, err = chbook.Issue(addr1, amount)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	backend.Commit()

	exp = new(big.Int).SetUint64(38)
	if chbook.balance.Cmp(exp) != 0 {
		t.Fatalf("expected balance %v, got %v", exp, chbook.balance)
	}

	time.Sleep(3 * interval)
	backend.Commit()
	if chbook.balance.Cmp(balance) != 0 {
		t.Fatalf("expected balance %v, got %v", balance, chbook.balance)
	}

	exp = new(big.Int).SetUint64(40)
	chbook.AutoDeposit(4*interval, exp, balance)
	_, err = chbook.Issue(addr1, amount)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	backend.Commit()
	_, err = chbook.Issue(addr1, amount)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	time.Sleep(3 * interval)
	backend.Commit()
	if chbook.balance.Cmp(exp) != 0 {
		t.Fatalf("expected balance %v, got %v", exp, chbook.balance)
	}

	_, err = chbook.Issue(addr1, amount)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	time.Sleep(1 * interval)
	backend.Commit()

	if chbook.balance.Cmp(balance) != 0 {
		t.Fatalf("expected balance %v, got %v", balance, chbook.balance)
	}

	chbook.AutoDeposit(1*interval, common.Big0, balance)
	chbook.Stop()

	_, err = chbook.Issue(addr1, common.Big1)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	backend.Commit()

	_, err = chbook.Issue(addr1, common.Big2)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	time.Sleep(1 * interval)
	backend.Commit()

	exp = new(big.Int).SetUint64(39)
	if chbook.balance.Cmp(exp) != 0 {
		t.Fatalf("expected balance %v, got %v", exp, chbook.balance)
	}

}

func TestCash(t *testing.T) {
	path := "/tmp/checkbook.json"
	backend := newTestBackend()
	contr0, err := deploy(key0, common.Big2, backend.SimulatedBackend)
	chbook, err := NewChequebook(path, contr0, key0, backend)
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	chbook.sent[addr1] = new(big.Int).SetUint64(42)
	amount := common.Big1
	chbook.balance = new(big.Int).Set(common.Big1)
	ch, err := chbook.Issue(addr1, amount)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	backend.Commit()
	chbox, err := NewInbox(key1, contr0, addr1, &key0.PublicKey, backend)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// cashing latest cheque
	_, err = chbox.Receive(ch)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	_, err = ch.Cash(chbook.session)
	backend.Commit()

	chbook.balance = new(big.Int).Set(common.Big3)
	ch0, err := chbook.Issue(addr1, amount)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	backend.Commit()
	ch1, err := chbook.Issue(addr1, amount)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	backend.Commit()

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
	backend.Commit()
	// expBalance := big.NewInt(2)
	// gotBalance := backend.BalanceAt(addr1)
	// if gotBalance.Cmp(expBalance) != 0 {
	// 	t.Fatalf("expected beneficiary balance %v, got %v", expBalance, gotBalance)
	// }
	// after 3x interval time and 2 cheques received, exactly one cashing tx is sent
	time.Sleep(4 * interval)
	backend.Commit()

	// expBalance = big.NewInt(4)
	// gotBalance = backend.BalanceAt(addr1)
	// if gotBalance.Cmp(expBalance) != 0 {
	// 	t.Fatalf("expected beneficiary balance %v, got %v", expBalance, gotBalance)
	// }

	// after stopping autocash no more tx are sent
	ch2, err := chbook.Issue(addr1, amount)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	chbox.Stop()
	_, err = chbox.Receive(ch2)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	time.Sleep(2 * interval)
	backend.Commit()
	// expBalance = big.NewInt(4)
	// gotBalance = backend.BalanceAt(addr1)
	// if gotBalance.Cmp(expBalance) != 0 {
	// 	t.Fatalf("expected beneficiary balance %v, got %v", expBalance, gotBalance)
	// }

	// autocash below 1
	chbook.balance = big.NewInt(2)
	chbox.AutoCash(0, common.Big1)

	ch3, err := chbook.Issue(addr1, amount)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	backend.Commit()
	// expBalance = big.NewInt(4)
	// gotBalance = backend.BalanceAt(addr1)
	// if gotBalance.Cmp(expBalance) != 0 {
	// 	t.Fatalf("expected beneficiary balance %v, got %v", expBalance, gotBalance)
	// }

	ch4, err := chbook.Issue(addr1, amount)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	backend.Commit()

	_, err = chbox.Receive(ch3)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	backend.Commit()
	_, err = chbox.Receive(ch4)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	backend.Commit()

	// 2 checks of amount 1 received, exactly 1 tx is sent
	// expBalance = big.NewInt(6)
	// gotBalance = backend.BalanceAt(addr1)
	// if gotBalance.Cmp(expBalance) != 0 {
	// 	t.Fatalf("expected beneficiary balance %v, got %v", expBalance, gotBalance)
	// }

	// autochash on receipt when maxUncashed is 0
	chbook.balance = new(big.Int).Set(common.Big2)
	chbox.AutoCash(0, common.Big0)

	ch5, err := chbook.Issue(addr1, amount)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	backend.Commit()
	// expBalance = big.NewInt(5)
	// gotBalance = backend.BalanceAt(addr1)
	// if gotBalance.Cmp(expBalance) != 0 {
	// 	t.Fatalf("expected beneficiary balance %v, got %v", expBalance, gotBalance)
	// }

	ch6, err := chbook.Issue(addr1, amount)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	_, err = chbox.Receive(ch5)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	backend.Commit()
	// expBalance = big.NewInt(4)
	// gotBalance = backend.BalanceAt(addr1)
	// if gotBalance.Cmp(expBalance) != 0 {
	// 	t.Fatalf("expected beneficiary balance %v, got %v", expBalance, gotBalance)
	// }

	_, err = chbox.Receive(ch6)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	backend.Commit()
	// expBalance = big.NewInt(6)
	// gotBalance = backend.BalanceAt(addr1)
	// if gotBalance.Cmp(expBalance) != 0 {
	// 	t.Fatalf("expected beneficiary balance %v, got %v", expBalance, gotBalance)
	// }

}
