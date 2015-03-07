package accounts

import (
	"testing"

	"time"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/crypto/randentropy"
	"github.com/ethereum/go-ethereum/ethutil"
)

func TestAccountManager(t *testing.T) {
	ks := crypto.NewKeyStorePlain(ethutil.DefaultDataDir() + "/testaccounts")
	am := NewAccountManager(ks, 100*time.Millisecond)
	pass := "" // not used but required by API
	a1, err := am.NewAccount(pass)
	toSign := randentropy.GetEntropyCSPRNG(32)
	_, err = am.SignLocked(a1, pass, toSign)
	if err != nil {
		t.Fatal(err)
	}

	// Cleanup
	time.Sleep(150 * time.Millisecond) // wait for locking

	accounts, err := am.Accounts()
	if err != nil {
		t.Fatal(err)
	}
	for _, account := range accounts {
		err := am.DeleteAccount(account.Address, pass)
		if err != nil {
			t.Fatal(err)
		}
	}
}

func TestAccountManagerLocking(t *testing.T) {
	ks := crypto.NewKeyStorePassphrase(ethutil.DefaultDataDir() + "/testaccounts")
	am := NewAccountManager(ks, 200*time.Millisecond)
	pass := "foo"
	a1, err := am.NewAccount(pass)
	toSign := randentropy.GetEntropyCSPRNG(32)

	// Signing without passphrase fails because account is locked
	_, err = am.Sign(a1, toSign)
	if err != ErrLocked {
		t.Fatal(err)
	}

	// Signing with passphrase works
	_, err = am.SignLocked(a1, pass, toSign)
	if err != nil {
		t.Fatal(err)
	}

	// Signing without passphrase works because account is temp unlocked
	_, err = am.Sign(a1, toSign)
	if err != nil {
		t.Fatal(err)
	}

	// Signing without passphrase fails after automatic locking
	time.Sleep(250 * time.Millisecond)

	_, err = am.Sign(a1, toSign)
	if err != ErrLocked {
		t.Fatal(err)
	}

	// Cleanup
	accounts, err := am.Accounts()
	if err != nil {
		t.Fatal(err)
	}
	for _, account := range accounts {
		err := am.DeleteAccount(account.Address, pass)
		if err != nil {
			t.Fatal(err)
		}
	}
}
