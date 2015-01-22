package crypto

import (
	crand "crypto/rand"
	"reflect"
	"testing"
)

func TestKeyStorePlain(t *testing.T) {
	ks := NewKeyStorePlain(DefaultDataDir())
	pass := "" // not used but required by API
	k1, err := ks.GenerateNewKey(crand.Reader, pass)
	if err != nil {
		t.Fatal(err)
	}

	k2 := new(Key)
	k2, err = ks.GetKey(k1.Id, pass)
	if err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(k1.Id, k2.Id) {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(k1.PrivateKey, k2.PrivateKey) {
		t.Fatal(err)
	}

	err = ks.DeleteKey(k2.Id, pass)
	if err != nil {
		t.Fatal(err)
	}
}

func TestKeyStorePassphrase(t *testing.T) {
	ks := NewKeyStorePassphrase(DefaultDataDir())
	pass := "foo"
	k1, err := ks.GenerateNewKey(crand.Reader, pass)
	if err != nil {
		t.Fatal(err)
	}
	k2 := new(Key)
	k2, err = ks.GetKey(k1.Id, pass)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(k1.Id, k2.Id) {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(k1.PrivateKey, k2.PrivateKey) {
		t.Fatal(err)
	}

	err = ks.DeleteKey(k2.Id, pass) // also to clean up created files
	if err != nil {
		t.Fatal(err)
	}
}

func TestKeyStorePassphraseDecryptionFail(t *testing.T) {
	ks := NewKeyStorePassphrase(DefaultDataDir())
	pass := "foo"
	k1, err := ks.GenerateNewKey(crand.Reader, pass)
	if err != nil {
		t.Fatal(err)
	}

	_, err = ks.GetKey(k1.Id, "bar") // wrong passphrase
	if err == nil {
		t.Fatal(err)
	}

	err = ks.DeleteKey(k1.Id, "bar") // wrong passphrase
	if err == nil {
		t.Fatal(err)
	}

	err = ks.DeleteKey(k1.Id, pass) // to clean up
	if err != nil {
		t.Fatal(err)
	}
}
