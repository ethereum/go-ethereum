// Copyright 2016 The go-ethereum Authors
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
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"testing"
	"time"

	"github.com/cespare/cp"
	"github.com/davecgh/go-spew/spew"
	"github.com/ethereum/go-ethereum/common"
)

var (
	cachetestDir, _   = filepath.Abs(filepath.Join("testdata", "keystore"))
	cachetestAccounts = []Account{
		{
			Address: common.HexToAddress("7ef5a6135f1fd6a02593eedc869c6d41d934aef8"),
			File:    filepath.Join(cachetestDir, "UTC--2016-03-22T12-57-55.920751759Z--7ef5a6135f1fd6a02593eedc869c6d41d934aef8"),
		},
		{
			Address: common.HexToAddress("f466859ead1932d743d622cb74fc058882e8648a"),
			File:    filepath.Join(cachetestDir, "aaa"),
		},
		{
			Address: common.HexToAddress("289d485d9771714cce91d3393d764e1311907acc"),
			File:    filepath.Join(cachetestDir, "zzz"),
		},
	}
)

func TestWatchNewFile(t *testing.T) {
	t.Parallel()

	dir, am := tmpManager(t, false)
	defer os.RemoveAll(dir)

	// Ensure the watcher is started before adding any files.
	am.Accounts()
	time.Sleep(200 * time.Millisecond)

	// Move in the files.
	wantAccounts := make([]Account, len(cachetestAccounts))
	for i := range cachetestAccounts {
		a := cachetestAccounts[i]
		a.File = filepath.Join(dir, filepath.Base(a.File))
		wantAccounts[i] = a
		if err := cp.CopyFile(a.File, cachetestAccounts[i].File); err != nil {
			t.Fatal(err)
		}
	}

	// am should see the accounts.
	var list []Account
	for d := 200 * time.Millisecond; d < 5*time.Second; d *= 2 {
		list = am.Accounts()
		if reflect.DeepEqual(list, wantAccounts) {
			return
		}
		time.Sleep(d)
	}
	t.Errorf("got %s, want %s", spew.Sdump(list), spew.Sdump(wantAccounts))
}

func TestWatchNoDir(t *testing.T) {
	t.Parallel()

	// Create am but not the directory that it watches.
	rand.Seed(time.Now().UnixNano())
	dir := filepath.Join(os.TempDir(), fmt.Sprintf("eth-keystore-watch-test-%d-%d", os.Getpid(), rand.Int()))
	am := NewManager(dir, LightScryptN, LightScryptP)

	list := am.Accounts()
	if len(list) > 0 {
		t.Error("initial account list not empty:", list)
	}
	time.Sleep(100 * time.Millisecond)

	// Create the directory and copy a key file into it.
	os.MkdirAll(dir, 0700)
	defer os.RemoveAll(dir)
	file := filepath.Join(dir, "aaa")
	if err := cp.CopyFile(file, cachetestAccounts[0].File); err != nil {
		t.Fatal(err)
	}

	// am should see the account.
	wantAccounts := []Account{cachetestAccounts[0]}
	wantAccounts[0].File = file
	for d := 200 * time.Millisecond; d < 8*time.Second; d *= 2 {
		list = am.Accounts()
		if reflect.DeepEqual(list, wantAccounts) {
			return
		}
		time.Sleep(d)
	}
	t.Errorf("\ngot  %v\nwant %v", list, wantAccounts)
}

func TestCacheInitialReload(t *testing.T) {
	cache := newAddrCache(cachetestDir)
	accounts := cache.accounts()
	if !reflect.DeepEqual(accounts, cachetestAccounts) {
		t.Fatalf("got initial accounts: %swant %s", spew.Sdump(accounts), spew.Sdump(cachetestAccounts))
	}
}

func TestCacheAddDeleteOrder(t *testing.T) {
	cache := newAddrCache("testdata/no-such-dir")
	cache.watcher.running = true // prevent unexpected reloads

	accounts := []Account{
		{
			Address: common.HexToAddress("095e7baea6a6c7c4c2dfeb977efac326af552d87"),
			File:    "-309830980",
		},
		{
			Address: common.HexToAddress("2cac1adea150210703ba75ed097ddfe24e14f213"),
			File:    "ggg",
		},
		{
			Address: common.HexToAddress("8bda78331c916a08481428e4b07c96d3e916d165"),
			File:    "zzzzzz-the-very-last-one.keyXXX",
		},
		{
			Address: common.HexToAddress("d49ff4eeb0b2686ed89c0fc0f2b6ea533ddbbd5e"),
			File:    "SOMETHING.key",
		},
		{
			Address: common.HexToAddress("7ef5a6135f1fd6a02593eedc869c6d41d934aef8"),
			File:    "UTC--2016-03-22T12-57-55.920751759Z--7ef5a6135f1fd6a02593eedc869c6d41d934aef8",
		},
		{
			Address: common.HexToAddress("f466859ead1932d743d622cb74fc058882e8648a"),
			File:    "aaa",
		},
		{
			Address: common.HexToAddress("289d485d9771714cce91d3393d764e1311907acc"),
			File:    "zzz",
		},
	}
	for _, a := range accounts {
		cache.add(a)
	}
	// Add some of them twice to check that they don't get reinserted.
	cache.add(accounts[0])
	cache.add(accounts[2])

	// Check that the account list is sorted by filename.
	wantAccounts := make([]Account, len(accounts))
	copy(wantAccounts, accounts)
	sort.Sort(accountsByFile(wantAccounts))
	list := cache.accounts()
	if !reflect.DeepEqual(list, wantAccounts) {
		t.Fatalf("got accounts: %s\nwant %s", spew.Sdump(accounts), spew.Sdump(wantAccounts))
	}
	for _, a := range accounts {
		if !cache.hasAddress(a.Address) {
			t.Errorf("expected hasAccount(%x) to return true", a.Address)
		}
	}
	if cache.hasAddress(common.HexToAddress("fd9bd350f08ee3c0c19b85a8e16114a11a60aa4e")) {
		t.Errorf("expected hasAccount(%x) to return false", common.HexToAddress("fd9bd350f08ee3c0c19b85a8e16114a11a60aa4e"))
	}

	// Delete a few keys from the cache.
	for i := 0; i < len(accounts); i += 2 {
		cache.delete(wantAccounts[i])
	}
	cache.delete(Account{Address: common.HexToAddress("fd9bd350f08ee3c0c19b85a8e16114a11a60aa4e"), File: "something"})

	// Check content again after deletion.
	wantAccountsAfterDelete := []Account{
		wantAccounts[1],
		wantAccounts[3],
		wantAccounts[5],
	}
	list = cache.accounts()
	if !reflect.DeepEqual(list, wantAccountsAfterDelete) {
		t.Fatalf("got accounts after delete: %s\nwant %s", spew.Sdump(list), spew.Sdump(wantAccountsAfterDelete))
	}
	for _, a := range wantAccountsAfterDelete {
		if !cache.hasAddress(a.Address) {
			t.Errorf("expected hasAccount(%x) to return true", a.Address)
		}
	}
	if cache.hasAddress(wantAccounts[0].Address) {
		t.Errorf("expected hasAccount(%x) to return false", wantAccounts[0].Address)
	}
}

func TestCacheFind(t *testing.T) {
	dir := filepath.Join("testdata", "dir")
	cache := newAddrCache(dir)
	cache.watcher.running = true // prevent unexpected reloads

	accounts := []Account{
		{
			Address: common.HexToAddress("095e7baea6a6c7c4c2dfeb977efac326af552d87"),
			File:    filepath.Join(dir, "a.key"),
		},
		{
			Address: common.HexToAddress("2cac1adea150210703ba75ed097ddfe24e14f213"),
			File:    filepath.Join(dir, "b.key"),
		},
		{
			Address: common.HexToAddress("d49ff4eeb0b2686ed89c0fc0f2b6ea533ddbbd5e"),
			File:    filepath.Join(dir, "c.key"),
		},
		{
			Address: common.HexToAddress("d49ff4eeb0b2686ed89c0fc0f2b6ea533ddbbd5e"),
			File:    filepath.Join(dir, "c2.key"),
		},
	}
	for _, a := range accounts {
		cache.add(a)
	}

	nomatchAccount := Account{
		Address: common.HexToAddress("f466859ead1932d743d622cb74fc058882e8648a"),
		File:    filepath.Join(dir, "something"),
	}
	tests := []struct {
		Query      Account
		WantResult Account
		WantError  error
	}{
		// by address
		{Query: Account{Address: accounts[0].Address}, WantResult: accounts[0]},
		// by file
		{Query: Account{File: accounts[0].File}, WantResult: accounts[0]},
		// by basename
		{Query: Account{File: filepath.Base(accounts[0].File)}, WantResult: accounts[0]},
		// by file and address
		{Query: accounts[0], WantResult: accounts[0]},
		// ambiguous address, tie resolved by file
		{Query: accounts[2], WantResult: accounts[2]},
		// ambiguous address error
		{
			Query: Account{Address: accounts[2].Address},
			WantError: &AmbiguousAddrError{
				Addr:    accounts[2].Address,
				Matches: []Account{accounts[2], accounts[3]},
			},
		},
		// no match error
		{Query: nomatchAccount, WantError: ErrNoMatch},
		{Query: Account{File: nomatchAccount.File}, WantError: ErrNoMatch},
		{Query: Account{File: filepath.Base(nomatchAccount.File)}, WantError: ErrNoMatch},
		{Query: Account{Address: nomatchAccount.Address}, WantError: ErrNoMatch},
	}
	for i, test := range tests {
		a, err := cache.find(test.Query)
		if !reflect.DeepEqual(err, test.WantError) {
			t.Errorf("test %d: error mismatch for query %v\ngot %q\nwant %q", i, test.Query, err, test.WantError)
			continue
		}
		if a != test.WantResult {
			t.Errorf("test %d: result mismatch for query %v\ngot %v\nwant %v", i, test.Query, a, test.WantResult)
			continue
		}
	}
}
