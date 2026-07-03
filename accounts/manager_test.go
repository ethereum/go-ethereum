// Copyright 2026 The go-ethereum Authors
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

import "testing"

type managerTestWallet struct {
	Wallet
	url URL
}

func (w managerTestWallet) URL() URL {
	return w.url
}

func TestDropMissingWallet(t *testing.T) {
	t.Parallel()

	wallets := []Wallet{
		managerTestWallet{url: URL{Scheme: "test", Path: "a"}},
		managerTestWallet{url: URL{Scheme: "test", Path: "c"}},
	}
	dropped := drop(wallets, managerTestWallet{url: URL{Scheme: "test", Path: "b"}})

	if len(dropped) != len(wallets) {
		t.Fatalf("drop removed wallet for missing URL: got %d wallets, want %d", len(dropped), len(wallets))
	}
	for i := range dropped {
		if got, want := dropped[i].URL(), wallets[i].URL(); got != want {
			t.Fatalf("wallet %d mismatch: got %v, want %v", i, got, want)
		}
	}
}
