package crypto

import (
	"fmt"
	"reflect"
	"testing"
)

func TestKeyStorePlain(t *testing.T) {
	ks := NewKeyStorePlain(DefaultDataDir())
	pass := "" // not used but required by API
	k1, err := ks.GenerateNewKey(pass)
	if err != nil {
		t.Error(err)
		t.FailNow()
	}

	k2 := new(Key)
	k2, err = ks.GetKey(k1.Id, pass)
	if err != nil {
		t.Error(err)
		t.FailNow()
	}

	if !reflect.DeepEqual(k1.Id, k2.Id) {
		fmt.Println("key Id mismatch")
		t.FailNow()
	}

	if k1.Flags != k2.Flags {
		fmt.Println("key Flags mismatch")
		t.FailNow()
	}

	if !reflect.DeepEqual(k1.PrivateKey, k2.PrivateKey) {
		fmt.Println("key PrivateKey mismatch")
		t.FailNow()
	}

	err = ks.DeleteKey(k2.Id, pass)
	if err != nil {
		t.Error(err)
		t.FailNow()
	}
}

func TestKeyStorePassphrase(t *testing.T) {
	ks := NewKeyStorePassphrase(DefaultDataDir())
	pass := "foo"
	k1, err := ks.GenerateNewKey(pass)
	if err != nil {
		t.Error(err)
		t.FailNow()
	}

	k2 := new(Key)
	k2, err = ks.GetKey(k1.Id, pass)
	if err != nil {
		t.Error(err)
		t.FailNow()
	}

	if !reflect.DeepEqual(k1.Id, k2.Id) {
		fmt.Println("key Id mismatch")
		t.FailNow()
	}

	if k1.Flags != k2.Flags {
		fmt.Println("key Flags mismatch")
		t.FailNow()
	}

	if !reflect.DeepEqual(k1.PrivateKey, k2.PrivateKey) {
		fmt.Println("key PrivateKey mismatch")
		t.FailNow()
	}

	err = ks.DeleteKey(k2.Id, pass) // also to clean up created files
	if err != nil {
		t.Error(err)
		t.FailNow()
	}
}

func TestKeyStorePassphraseDecryptionFail(t *testing.T) {
	ks := NewKeyStorePassphrase(DefaultDataDir())
	pass := "foo"
	k1, err := ks.GenerateNewKey(pass)
	if err != nil {
		t.Error(err)
		t.FailNow()
	}

	_, err = ks.GetKey(k1.Id, "bar") // wrong passphrase
	// t.Error(err)
	if err == nil {
		t.FailNow()
	}

	err = ks.DeleteKey(k1.Id, "bar") // wrong passphrase
	if err == nil {
		t.Error(err)
		t.FailNow()
	}

	err = ks.DeleteKey(k1.Id, pass) // to clean up
	if err != nil {
		t.Error(err)
		t.FailNow()
	}
}

func TestKeyMixedEntropy(t *testing.T) {
	GetEntropyTinFoilHat()
}
