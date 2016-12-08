// Copyright 2015 The go-ethereum Authors
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

package accounts

import (
	"io/ioutil"
	"os"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
)

var testSigData = make([]byte, 32)

func TestManager(t *testing.T) {
	dir, am := tmpManager(t, true)
	defer os.RemoveAll(dir)

	a, err := am.NewAccount("foo")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(a.File, dir) {
		t.Errorf("account file %s doesn't have dir prefix", a.File)
	}
	stat, err := os.Stat(a.File)
	if err != nil {
		t.Fatalf("account file %s doesn't exist (%v)", a.File, err)
	}
	if runtime.GOOS != "windows" && stat.Mode() != 0600 {
		t.Fatalf("account file has wrong mode: got %o, want %o", stat.Mode(), 0600)
	}
	if !am.HasAddress(a.Address) {
		t.Errorf("HasAccount(%x) should've returned true", a.Address)
	}
	if err := am.Update(a, "foo", "bar"); err != nil {
		t.Errorf("Update error: %v", err)
	}
	if err := am.DeleteAccount(a, "bar"); err != nil {
		t.Errorf("DeleteAccount error: %v", err)
	}
	if common.FileExist(a.File) {
		t.Errorf("account file %s should be gone after DeleteAccount", a.File)
	}
	if am.HasAddress(a.Address) {
		t.Errorf("HasAccount(%x) should've returned true after DeleteAccount", a.Address)
	}
}

func TestSign(t *testing.T) {
	dir, am := tmpManager(t, true)
	defer os.RemoveAll(dir)

	pass := "" // not used but required by API
	a1, err := am.NewAccount(pass)
	if err != nil {
		t.Fatal(err)
	}
	if err := am.Unlock(a1, ""); err != nil {
		t.Fatal(err)
	}
	if _, err := am.Sign(a1.Address, testSigData); err != nil {
		t.Fatal(err)
	}
}

func TestSignWithPassphrase(t *testing.T) {
	dir, am := tmpManager(t, true)
	defer os.RemoveAll(dir)

	pass := "passwd"
	acc, err := am.NewAccount(pass)
	if err != nil {
		t.Fatal(err)
	}

	if _, unlocked := am.unlocked[acc.Address]; unlocked {
		t.Fatal("expected account to be locked")
	}

	_, err = am.SignWithPassphrase(acc, pass, testSigData)
	if err != nil {
		t.Fatal(err)
	}

	if _, unlocked := am.unlocked[acc.Address]; unlocked {
		t.Fatal("expected account to be locked")
	}

	if _, err = am.SignWithPassphrase(acc, "invalid passwd", testSigData); err == nil {
		t.Fatal("expected SignHash to fail with invalid password")
	}
}

func TestTimedUnlock(t *testing.T) {
	dir, am := tmpManager(t, true)
	defer os.RemoveAll(dir)

	pass := "foo"
	a1, err := am.NewAccount(pass)

	// Signing without passphrase fails because account is locked
	_, err = am.Sign(a1.Address, testSigData)
	if err != ErrLocked {
		t.Fatal("Signing should've failed with ErrLocked before unlocking, got ", err)
	}

	// Signing with passphrase works
	if err = am.TimedUnlock(a1, pass, 100*time.Millisecond); err != nil {
		t.Fatal(err)
	}

	// Signing without passphrase works because account is temp unlocked
	_, err = am.Sign(a1.Address, testSigData)
	if err != nil {
		t.Fatal("Signing shouldn't return an error after unlocking, got ", err)
	}

	// Signing fails again after automatic locking
	time.Sleep(250 * time.Millisecond)
	_, err = am.Sign(a1.Address, testSigData)
	if err != ErrLocked {
		t.Fatal("Signing should've failed with ErrLocked timeout expired, got ", err)
	}
}

func TestOverrideUnlock(t *testing.T) {
	dir, am := tmpManager(t, false)
	defer os.RemoveAll(dir)

	pass := "foo"
	a1, err := am.NewAccount(pass)

	// Unlock indefinitely.
	if err = am.TimedUnlock(a1, pass, 5*time.Minute); err != nil {
		t.Fatal(err)
	}

	// Signing without passphrase works because account is temp unlocked
	_, err = am.Sign(a1.Address, testSigData)
	if err != nil {
		t.Fatal("Signing shouldn't return an error after unlocking, got ", err)
	}

	// reset unlock to a shorter period, invalidates the previous unlock
	if err = am.TimedUnlock(a1, pass, 100*time.Millisecond); err != nil {
		t.Fatal(err)
	}

	// Signing without passphrase still works because account is temp unlocked
	_, err = am.Sign(a1.Address, testSigData)
	if err != nil {
		t.Fatal("Signing shouldn't return an error after unlocking, got ", err)
	}

	// Signing fails again after automatic locking
	time.Sleep(250 * time.Millisecond)
	_, err = am.Sign(a1.Address, testSigData)
	if err != ErrLocked {
		t.Fatal("Signing should've failed with ErrLocked timeout expired, got ", err)
	}
}

// This test should fail under -race if signing races the expiration goroutine.
func TestSignRace(t *testing.T) {
	dir, am := tmpManager(t, false)
	defer os.RemoveAll(dir)

	// Create a test account.
	a1, err := am.NewAccount("")
	if err != nil {
		t.Fatal("could not create the test account", err)
	}

	if err := am.TimedUnlock(a1, "", 15*time.Millisecond); err != nil {
		t.Fatal("could not unlock the test account", err)
	}
	end := time.Now().Add(500 * time.Millisecond)
	for time.Now().Before(end) {
		if _, err := am.Sign(a1.Address, testSigData); err == ErrLocked {
			return
		} else if err != nil {
			t.Errorf("Sign error: %v", err)
			return
		}
		time.Sleep(1 * time.Millisecond)
	}
	t.Errorf("Account did not lock within the timeout")
}

func tmpManager(t *testing.T, encrypted bool) (string, *Manager) {
	d, err := ioutil.TempDir("", "eth-keystore-test")
	if err != nil {
		t.Fatal(err)
	}
	new := NewPlaintextManager
	if encrypted {
		new = func(kd string) *Manager { return NewManager(kd, veryLightScryptN, veryLightScryptP) }
	}
	return d, new(d)
}
