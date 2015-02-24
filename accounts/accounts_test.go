package accounts

import (
	"testing"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/crypto/randentropy"
)

func TestAccountManager(t *testing.T) {

	// Get AccountManager
	ks := crypto.NewKeyStorePlain(crypto.DefaultDataDir() + "/testaccounts")
	am := NewAccountManager(ks)

	// Create a new account
	pass := "" // not used but required by API
	a1, err := am.NewAccount(pass)

	// Sign junk
	toSign := randentropy.GetEntropyCSPRNG(32)
	_, err = am.Sign(a1, pass, toSign)
	if err != nil {
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
