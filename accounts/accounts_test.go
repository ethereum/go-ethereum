package accounts

import (
	"github.com/ethereum/go-ethereum/crypto"
	"testing"
)

func TestAccountManager(t *testing.T) {
	ks := crypto.NewKeyStorePlain(crypto.DefaultDataDir())
	am := NewAccountManager(ks)
	pass := "" // not used but required by API
	a1, err := am.NewAccount(pass)
	toSign := make([]byte, 4, 4)
	toSign = []byte{0, 1, 2, 3}
	_, err = am.Sign(a1.Addr, pass, toSign)
	if err != nil {
		t.Fatal(err)
	}
}
