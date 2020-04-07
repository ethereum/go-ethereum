// Copyright 2017 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
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

package keystore

import (
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/cmd/clef/dbutil"
	"github.com/ethereum/go-ethereum/common"
	_ "github.com/mattn/go-sqlite3"
)

const foo = "foo"

func TestKeyStoreDB(t *testing.T) {
	ks := tmpKeyStoreDB(t)

	acc, err := ks.NewAccount(foo)
	if err != nil {
		t.Fatal(err)
	}
	if !ks.HasAddress(acc.Address) {
		t.Errorf("HasAccount(%x) should've returned true", acc.Address)
	}
	if err := ks.Update(acc, foo, "bar"); err != nil {
		t.Errorf("Update error: %v", err)
	}
	if err := ks.Delete(acc, "bar"); err != nil {
		t.Errorf("Delete error: %v", err)
	}
	if ks.HasAddress(acc.Address) {
		t.Errorf("HasAccount(%x) should've returned true after Delete", acc.Address)
	}
}

func TestSignDB(t *testing.T) {
	ks := tmpKeyStoreDB(t)

	pass := "" // not used by required by API
	acc, err := ks.NewAccount(pass)
	if err != nil {
		t.Fatal(err)
	}
	if err := ks.Unlock(acc, ""); err != nil {
		t.Fatal(err)
	}
	if _, err := ks.SignHash(accounts.Account{Address: acc.Address}, testSigData); err != nil {
		t.Fatal(err)
	}
}

func TestSignWithPassphraseDB(t *testing.T) {
	ks := tmpKeyStoreDB(t)

	pass := "passwd"
	acc, err := ks.NewAccount(pass)
	if err != nil {
		t.Fatal(err)
	}

	if _, unlocked := ks.unlocked[acc.Address]; unlocked {
		t.Fatal("expected account to be locked")
	}

	_, err = ks.SignHashWithPassphrase(acc, pass, testSigData)
	if err != nil {
		t.Fatal(err)
	}

	if _, unlocked := ks.unlocked[acc.Address]; unlocked {
		t.Fatal("expected account to be locked")
	}

	if _, err = ks.SignHashWithPassphrase(acc, "invalid passwd", testSigData); err == nil {
		t.Fatal("expected SignHashWithPassphrase to fail with invalid password")
	}
}

func TestTimedUnlockDB(t *testing.T) {
	ks := tmpKeyStoreDB(t)

	pass := foo
	acc, err := ks.NewAccount(pass)
	if err != nil {
		t.Fatal(err)
	}

	// Signing without passphrase fails because account is locked
	_, err = ks.SignHash(accounts.Account{Address: acc.Address}, testSigData)
	if err != ErrLocked {
		t.Fatal("Signing should've failed with ErrLocked before unlocking, got ", err)
	}

	// Signing with passphrase works
	if err = ks.TimedUnlock(acc, pass, 100*time.Millisecond); err != nil {
		t.Fatal(err)
	}

	// Signing without passphrase works because account is temp unlocked
	_, err = ks.SignHash(accounts.Account{Address: acc.Address}, testSigData)
	if err != nil {
		t.Fatal("Signing shouldn't return an error after unlocking, got ", err)
	}

	// Signing fails again after automatic locking
	time.Sleep(250 * time.Millisecond)
	_, err = ks.SignHash(accounts.Account{Address: acc.Address}, testSigData)
	if err != ErrLocked {
		t.Fatal("Signing should've failed with ErrLocked timeout expired, got ", err)
	}
}

func TestOverrideUnlockDB(t *testing.T) {
	ks := tmpKeyStoreDB(t)

	pass := foo
	acc, err := ks.NewAccount(pass)
	if err != nil {
		t.Fatal(err)
	}

	// Unlock indefinitely.
	if err = ks.TimedUnlock(acc, pass, 5*time.Minute); err != nil {
		t.Fatal(err)
	}

	// Signing without passphrase works because account is temp unlocked
	_, err = ks.SignHash(accounts.Account{Address: acc.Address}, testSigData)
	if err != nil {
		t.Fatal("Signing shouldn't return an error after unlocking, got ", err)
	}

	// reset unlock to a shorter period, invalidates the previous unlock
	if err = ks.TimedUnlock(acc, pass, 100*time.Millisecond); err != nil {
		t.Fatal(err)
	}

	// Signing without passphrase still works because account is temp unlocked
	_, err = ks.SignHash(accounts.Account{Address: acc.Address}, testSigData)
	if err != nil {
		t.Fatal("Signing shouldn't return an error after unlocking, got ", err)
	}

	// Signing fails again after automatic locking
	time.Sleep(250 * time.Millisecond)
	_, err = ks.SignHash(accounts.Account{Address: acc.Address}, testSigData)
	if err != ErrLocked {
		t.Fatal("Signing should've failed with ErrLocked timeout expired, got ", err)
	}
}

// This test should fail under -race if signing races the expiration goroutine.
func TestSignRaceDB(t *testing.T) {
	ks := tmpKeyStoreDB(t)

	pass := ""
	acc, err := ks.NewAccount(pass)
	if err != nil {
		t.Fatal(err)
	}

	if err := ks.TimedUnlock(acc, "", 15*time.Millisecond); err != nil {
		t.Fatal("could not unlock the test account", err)
	}
	end := time.Now().Add(500 * time.Millisecond)
	for time.Now().Before(end) {
		if _, err := ks.SignHash(accounts.Account{Address: acc.Address}, testSigData); err == ErrLocked {
			return
		} else if err != nil {
			t.Errorf("Sign error: %v", err)
			return
		}
		time.Sleep(1 * time.Millisecond)
	}
	t.Errorf("Account did not lock within the timeout")
}

func tmpKeyStoreDB(t *testing.T) *keyStoreDB {
	// ks, err := NewKeyStoreDB("./testdata/db_keystore_test.yaml", "testTable", veryLightScryptN, veryLightScryptP)
	kvstore, err := dbutil.NewKVStore("./testdata/db_keystore_test.yaml", "testTable")
	if err != nil {
		t.Fatal(err)
	}
	storage := &keyStorePassphraseDB{kvstore, veryLightScryptN, veryLightScryptP, false}
	ks := &keyStoreDB{storage: storage, unlocked: make(map[common.Address]*unlocked)}
	if err != nil {
		t.Fatal("Cannot initiate database keystore: ", err)
	}
	return ks
}
