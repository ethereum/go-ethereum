// Copyright 2016 The go-ethereum Authors
// This file is part of go-ethereum.
//
// go-ethereum is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// go-ethereum is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with go-ethereum. If not, see <http://www.gnu.org/licenses/>.

package main

import (
	"io/ioutil"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/cespare/cp"
)

// These tests are 'smoke tests' for the account related
// subcommands and flags.
//
// For most tests, the test files from package accounts
// are copied into a temporary keystore directory.

func tmpDatadirWithKeystore(t *testing.T) string {
	datadir := tmpdir(t)
	keystore := filepath.Join(datadir, "keystore")
	source := filepath.Join("..", "..", "accounts", "keystore", "testdata", "keystore")
	if err := cp.CopyAll(keystore, source); err != nil {
		t.Fatal(err)
	}
	return datadir
}

func TestAccountListEmpty(t *testing.T) {
	gexp := runGeth(t, "account", "list")
	gexp.ExpectExit()
}

func TestAccountList(t *testing.T) {
	datadir := tmpDatadirWithKeystore(t)
	gexp := runGeth(t, "account", "list", "--datadir", datadir)
	defer gexp.ExpectExit()
	if runtime.GOOS == "windows" {
		gexp.Expect(`
Account #0: {7ef5a6135f1fd6a02593eedc869c6d41d934aef8} keystore://{{.Datadir}}\keystore\UTC--2016-03-22T12-57-55.920751759Z--7ef5a6135f1fd6a02593eedc869c6d41d934aef8
Account #1: {f466859ead1932d743d622cb74fc058882e8648a} keystore://{{.Datadir}}\keystore\aaa
Account #2: {289d485d9771714cce91d3393d764e1311907acc} keystore://{{.Datadir}}\keystore\zzz
`)
	} else {
		gexp.Expect(`
Account #0: {7ef5a6135f1fd6a02593eedc869c6d41d934aef8} keystore://{{.Datadir}}/keystore/UTC--2016-03-22T12-57-55.920751759Z--7ef5a6135f1fd6a02593eedc869c6d41d934aef8
Account #1: {f466859ead1932d743d622cb74fc058882e8648a} keystore://{{.Datadir}}/keystore/aaa
Account #2: {289d485d9771714cce91d3393d764e1311907acc} keystore://{{.Datadir}}/keystore/zzz
`)
	}
}

func TestAccountNew(t *testing.T) {
	gexp := runGeth(t, "account", "new", "--lightkdf")
	defer gexp.ExpectExit()
	gexp.Expect(`
Your new account is locked with a password. Please give a password. Do not forget this password.
!! Unsupported terminal, password will be echoed.
Passphrase: {{.InputLine "foobar"}}
Repeat passphrase: {{.InputLine "foobar"}}
`)
	gexp.ExpectRegexp(`Address: \{[0-9a-f]{40}\}\n`)
}

func TestAccountNewBadRepeat(t *testing.T) {
	gexp := runGeth(t, "account", "new", "--lightkdf")
	defer gexp.ExpectExit()
	gexp.Expect(`
Your new account is locked with a password. Please give a password. Do not forget this password.
!! Unsupported terminal, password will be echoed.
Passphrase: {{.InputLine "something"}}
Repeat passphrase: {{.InputLine "something else"}}
Fatal: Passphrases do not match
`)
}

func TestAccountUpdate(t *testing.T) {
	datadir := tmpDatadirWithKeystore(t)
	gexp := runGeth(t, "account", "update",
		"--datadir", datadir, "--lightkdf",
		"f466859ead1932d743d622cb74fc058882e8648a")

	defer gexp.ExpectExit()
	gexp.Expect(`
Unlocking account f466859ead1932d743d622cb74fc058882e8648a | Attempt 1/3
!! Unsupported terminal, password will be echoed.
Passphrase: {{.InputLine "foobar"}}
Please give a new password. Do not forget this password.
Passphrase: {{.InputLine "foobar2"}}
Repeat passphrase: {{.InputLine "foobar2"}}
`)
}

func TestWalletImport(t *testing.T) {
	gexp := runGeth(t, "wallet", "import", "--lightkdf", "testdata/guswallet.json")
	defer gexp.ExpectExit()
	gexp.Expect(`
!! Unsupported terminal, password will be echoed.
Passphrase: {{.InputLine "foo"}}
Address: {d4584b5f6229b7be90727b0fc8c6b91bb427821f}
`)

	files, err := ioutil.ReadDir(filepath.Join(gexp.Datadir, "keystore"))
	if len(files) != 1 {
		t.Errorf("expected one key file in keystore directory, found %d files (error: %v)", len(files), err)
	}
}

func TestWalletImportBadPassword(t *testing.T) {
	gexp := runGeth(t, "wallet", "import", "--lightkdf", "testdata/guswallet.json")
	defer gexp.ExpectExit()
	gexp.Expect(`
!! Unsupported terminal, password will be echoed.
Passphrase: {{.InputLine "wrong"}}
Fatal: could not decrypt key with given passphrase
`)
}

func TestUnlockFlag(t *testing.T) {
	datadir := tmpDatadirWithKeystore(t)
	gexp := runGeth(t,
		"--datadir", datadir, "--nat", "none", "--nodiscover", "--dev",
		"--unlock", "f466859ead1932d743d622cb74fc058882e8648a",
		"js", "testdata/empty.js")
	gexp.Expect(`
Unlocking account f466859ead1932d743d622cb74fc058882e8648a | Attempt 1/3
!! Unsupported terminal, password will be echoed.
Passphrase: {{.InputLine "foobar"}}
`)
	gexp.ExpectExit()

	wantMessages := []string{
		"Unlocked account",
		"=0xf466859eAD1932D743d622CB74FC058882E8648A",
	}
	for _, m := range wantMessages {
		if !strings.Contains(gexp.StderrText(), m) {
			t.Errorf("stderr text does not contain %q", m)
		}
	}
}

func TestUnlockFlagWrongPassword(t *testing.T) {
	datadir := tmpDatadirWithKeystore(t)
	gexp := runGeth(t,
		"--datadir", datadir, "--nat", "none", "--nodiscover", "--dev",
		"--unlock", "f466859ead1932d743d622cb74fc058882e8648a")
	defer gexp.ExpectExit()
	gexp.Expect(`
Unlocking account f466859ead1932d743d622cb74fc058882e8648a | Attempt 1/3
!! Unsupported terminal, password will be echoed.
Passphrase: {{.InputLine "wrong1"}}
Unlocking account f466859ead1932d743d622cb74fc058882e8648a | Attempt 2/3
Passphrase: {{.InputLine "wrong2"}}
Unlocking account f466859ead1932d743d622cb74fc058882e8648a | Attempt 3/3
Passphrase: {{.InputLine "wrong3"}}
Fatal: Failed to unlock account f466859ead1932d743d622cb74fc058882e8648a (could not decrypt key with given passphrase)
`)
}

// https://github.com/expanse-org/go-expanse/issues/1785
func TestUnlockFlagMultiIndex(t *testing.T) {
	datadir := tmpDatadirWithKeystore(t)
	gexp := runGeth(t,
		"--datadir", datadir, "--nat", "none", "--nodiscover", "--dev",
		"--unlock", "0,2",
		"js", "testdata/empty.js")
	gexp.Expect(`
Unlocking account 0 | Attempt 1/3
!! Unsupported terminal, password will be echoed.
Passphrase: {{.InputLine "foobar"}}
Unlocking account 2 | Attempt 1/3
Passphrase: {{.InputLine "foobar"}}
`)
	gexp.ExpectExit()

	wantMessages := []string{
		"Unlocked account",
		"=0x7EF5A6135f1FD6a02593eEdC869c6D41D934aef8",
		"=0x289d485D9771714CCe91D3393D764E1311907ACc",
	}
	for _, m := range wantMessages {

		if !strings.Contains(gexp.StderrText(), m) {
			t.Errorf("stderr text does not contain %q", m)
		}
	}
}

func TestUnlockFlagPasswordFile(t *testing.T) {
	datadir := tmpDatadirWithKeystore(t)
	gexp := runGeth(t,
		"--datadir", datadir, "--nat", "none", "--nodiscover", "--dev",
		"--password", "testdata/passwords.txt", "--unlock", "0,2",
		"js", "testdata/empty.js")
	gexp.ExpectExit()

	wantMessages := []string{
		"Unlocked account",
		"=0x7EF5A6135f1FD6a02593eEdC869c6D41D934aef8",
		"=0x289d485D9771714CCe91D3393D764E1311907ACc",
	}
	for _, m := range wantMessages {
		if !strings.Contains(gexp.StderrText(), m) {
			t.Errorf("stderr text does not contain %q", m)
		}
	}
}

func TestUnlockFlagPasswordFileWrongPassword(t *testing.T) {
	datadir := tmpDatadirWithKeystore(t)
	gexp := runGeth(t,
		"--datadir", datadir, "--nat", "none", "--nodiscover", "--dev",
		"--password", "testdata/wrong-passwords.txt", "--unlock", "0,2")
	defer gexp.ExpectExit()
	gexp.Expect(`
Fatal: Failed to unlock account 0 (could not decrypt key with given passphrase)
`)
}

func TestUnlockFlagAmbiguous(t *testing.T) {
	store := filepath.Join("..", "..", "accounts", "keystore", "testdata", "dupes")
	gexp := runGeth(t,
		"--keystore", store, "--nat", "none", "--nodiscover", "--dev",
		"--unlock", "f466859ead1932d743d622cb74fc058882e8648a",
		"js", "testdata/empty.js")
	defer gexp.ExpectExit()

	// Helper for the expect template, returns absolute keystore path.
	gexp.SetTemplateFunc("keypath", func(file string) string {
		abs, _ := filepath.Abs(filepath.Join(store, file))
		return abs
	})
	gexp.Expect(`
Unlocking account f466859ead1932d743d622cb74fc058882e8648a | Attempt 1/3
!! Unsupported terminal, password will be echoed.
Passphrase: {{.InputLine "foobar"}}
Multiple key files exist for address f466859ead1932d743d622cb74fc058882e8648a:
   keystore://{{keypath "1"}}
   keystore://{{keypath "2"}}
Testing your passphrase against all of them...
Your passphrase unlocked keystore://{{keypath "1"}}
In order to avoid this warning, you need to remove the following duplicate key files:
   keystore://{{keypath "2"}}
`)
	gexp.ExpectExit()

	wantMessages := []string{
		"Unlocked account",
		"=0xf466859eAD1932D743d622CB74FC058882E8648A",
	}
	for _, m := range wantMessages {
		if !strings.Contains(gexp.StderrText(), m) {
			t.Errorf("stderr text does not contain %q", m)
		}
	}
}

func TestUnlockFlagAmbiguousWrongPassword(t *testing.T) {
	store := filepath.Join("..", "..", "accounts", "keystore", "testdata", "dupes")
	gexp := runGeth(t,
		"--keystore", store, "--nat", "none", "--nodiscover", "--dev",
		"--unlock", "f466859ead1932d743d622cb74fc058882e8648a")
	defer gexp.ExpectExit()

	// Helper for the expect template, returns absolute keystore path.
	gexp.SetTemplateFunc("keypath", func(file string) string {
		abs, _ := filepath.Abs(filepath.Join(store, file))
		return abs
	})
	gexp.Expect(`
Unlocking account f466859ead1932d743d622cb74fc058882e8648a | Attempt 1/3
!! Unsupported terminal, password will be echoed.
Passphrase: {{.InputLine "wrong"}}
Multiple key files exist for address f466859ead1932d743d622cb74fc058882e8648a:
   keystore://{{keypath "1"}}
   keystore://{{keypath "2"}}
Testing your passphrase against all of them...
Fatal: None of the listed files could be unlocked.
`)
	gexp.ExpectExit()
}
