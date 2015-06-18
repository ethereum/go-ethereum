package accounts

import (
	"io/ioutil"
	"os"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/crypto/randentropy"
)

func TestSign(t *testing.T) {
	dir, ks := tmpKeyStore(t, crypto.NewKeyStorePlain)
	defer os.RemoveAll(dir)

	am := NewManager(ks)
	pass := "" // not used but required by API
	a1, err := am.NewAccount(pass)
	toSign := randentropy.GetEntropyCSPRNG(32)
	am.Unlock(a1.Address, "")

	_, err = am.Sign(a1, toSign)
	if err != nil {
		t.Fatal(err)
	}
}

func TestTimedUnlock(t *testing.T) {
	dir, ks := tmpKeyStore(t, crypto.NewKeyStorePassphrase)
	defer os.RemoveAll(dir)

	am := NewManager(ks)
	pass := "foo"
	a1, err := am.NewAccount(pass)
	toSign := randentropy.GetEntropyCSPRNG(32)

	// Signing without passphrase fails because account is locked
	_, err = am.Sign(a1, toSign)
	if err != ErrLocked {
		t.Fatal("Signing should've failed with ErrLocked before unlocking, got ", err)
	}

	// Signing with passphrase works
	if err = am.TimedUnlock(a1.Address, pass, 100*time.Millisecond); err != nil {
		t.Fatal(err)
	}

	// Signing without passphrase works because account is temp unlocked
	_, err = am.Sign(a1, toSign)
	if err != nil {
		t.Fatal("Signing shouldn't return an error after unlocking, got ", err)
	}

	// Signing fails again after automatic locking
	time.Sleep(150 * time.Millisecond)
	_, err = am.Sign(a1, toSign)
	if err != ErrLocked {
		t.Fatal("Signing should've failed with ErrLocked timeout expired, got ", err)
	}

}

func TestOverrideUnlock(t *testing.T) {
	dir, ks := tmpKeyStore(t, crypto.NewKeyStorePassphrase)
	defer os.RemoveAll(dir)

	am := NewManager(ks)
	pass := "foo"
	a1, err := am.NewAccount(pass)
	toSign := randentropy.GetEntropyCSPRNG(32)

	// Unlock indefinitely
	if err = am.Unlock(a1.Address, pass); err != nil {
		t.Fatal(err)
	}

	// Signing without passphrase works because account is temp unlocked
	_, err = am.Sign(a1, toSign)
	if err != nil {
		t.Fatal("Signing shouldn't return an error after unlocking, got ", err)
	}

	// reset unlock to a shorter period, invalidates the previous unlock
	if err = am.TimedUnlock(a1.Address, pass, 100*time.Millisecond); err != nil {
		t.Fatal(err)
	}

	// Signing without passphrase still works because account is temp unlocked
	_, err = am.Sign(a1, toSign)
	if err != nil {
		t.Fatal("Signing shouldn't return an error after unlocking, got ", err)
	}

	// Signing fails again after automatic locking
	time.Sleep(150 * time.Millisecond)
	_, err = am.Sign(a1, toSign)
	if err != ErrLocked {
		t.Fatal("Signing should've failed with ErrLocked timeout expired, got ", err)
	}

}

func tmpKeyStore(t *testing.T, new func(string) crypto.KeyStore2) (string, crypto.KeyStore2) {
	d, err := ioutil.TempDir("", "eth-keystore-test")
	if err != nil {
		t.Fatal(err)
	}
	return d, new(d)
}
