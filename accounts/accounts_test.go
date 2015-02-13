package accounts

import (
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethutil"
	"testing"
)

func init() {
	ethutil.ReadConfig("/tmp/ethtest", "/tmp", "ETH")
}

func TestAccountManager(t *testing.T) {
	ks := crypto.NewKeyStorePlain(crypto.DefaultDataDir())
	am, _ := NewAccountManager(ks)
	pass := "" // not used but required by API
	a1, err := am.NewAccount(pass)
	toSign := crypto.GetEntropyCSPRNG(32)
	_, err = am.Sign(a1, pass, toSign)
	if err != nil {
		t.Fatal(err)
	}
}
