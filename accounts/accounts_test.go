// Copyright 2015 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// go-ethereum is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

package accounts

import (
	"io/ioutil"
	"os"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/crypto"
)

var testSigData = make([]byte, 32)

func TestSign(t *testing.T) {
	dir, ks := tmpKeyStore(t, crypto.NewKeyStorePlain)
	defer os.RemoveAll(dir)

	am := NewManager(ks)
	pass := "" // not used but required by API
	a1, err := am.NewAccount(pass)
	am.Unlock(a1.Address, "")

	_, err = am.Sign(a1, testSigData)
	if err != nil {
		t.Fatal(err)
	}
}

func TestTimedUnlock(t *testing.T) {
	dir, ks := tmpKeyStore(t, crypto.NewKeyStorePlain)
	defer os.RemoveAll(dir)

	am := NewManager(ks)
	pass := "foo"
	a1, err := am.NewAccount(pass)

	// Signing without passphrase fails because account is locked
	_, err = am.Sign(a1, testSigData)
	if err != ErrLocked {
		t.Fatal("Signing should've failed with ErrLocked before unlocking, got ", err)
	}

	// Signing with passphrase works
	if err = am.TimedUnlock(a1.Address, pass, 100*time.Millisecond); err != nil {
		t.Fatal(err)
	}

	// Signing without passphrase works because account is temp unlocked
	_, err = am.Sign(a1, testSigData)
	if err != nil {
		t.Fatal("Signing shouldn't return an error after unlocking, got ", err)
	}

	// Signing fails again after automatic locking
	time.Sleep(150 * time.Millisecond)
	_, err = am.Sign(a1, testSigData)
	if err != ErrLocked {
		t.Fatal("Signing should've failed with ErrLocked timeout expired, got ", err)
	}
}

func TestOverrideUnlock(t *testing.T) {
	dir, ks := tmpKeyStore(t, crypto.NewKeyStorePlain)
	defer os.RemoveAll(dir)

	am := NewManager(ks)
	pass := "foo"
	a1, err := am.NewAccount(pass)

	// Unlock indefinitely
	if err = am.Unlock(a1.Address, pass); err != nil {
		t.Fatal(err)
	}

	// Signing without passphrase works because account is temp unlocked
	_, err = am.Sign(a1, testSigData)
	if err != nil {
		t.Fatal("Signing shouldn't return an error after unlocking, got ", err)
	}

	// reset unlock to a shorter period, invalidates the previous unlock
	if err = am.TimedUnlock(a1.Address, pass, 100*time.Millisecond); err != nil {
		t.Fatal(err)
	}

	// Signing without passphrase still works because account is temp unlocked
	_, err = am.Sign(a1, testSigData)
	if err != nil {
		t.Fatal("Signing shouldn't return an error after unlocking, got ", err)
	}

	// Signing fails again after automatic locking
	time.Sleep(150 * time.Millisecond)
	_, err = am.Sign(a1, testSigData)
	if err != ErrLocked {
		t.Fatal("Signing should've failed with ErrLocked timeout expired, got ", err)
	}
}

// This test should fail under -race if signing races the expiration goroutine.
func TestSignRace(t *testing.T) {
	dir, ks := tmpKeyStore(t, crypto.NewKeyStorePlain)
	defer os.RemoveAll(dir)

	// Create a test account.
	am := NewManager(ks)
	a1, err := am.NewAccount("")
	if err != nil {
		t.Fatal("could not create the test account", err)
	}

	if err := am.TimedUnlock(a1.Address, "", 15*time.Millisecond); err != nil {
		t.Fatalf("could not unlock the test account", err)
	}
	end := time.Now().Add(80 * time.Millisecond)
	for time.Now().Before(end) {
		if _, err := am.Sign(a1, testSigData); err == ErrLocked {
			return
		} else if err != nil {
			t.Errorf("Sign error: %v", err)
			return
		}
	}
	t.Errorf("Account did not lock within the timeout")
}

func tmpKeyStore(t *testing.T, new func(string) crypto.KeyStore) (string, crypto.KeyStore) {
	d, err := ioutil.TempDir("", "eth-keystore-test")
	if err != nil {
		t.Fatal(err)
	}
	return d, new(d)
}
