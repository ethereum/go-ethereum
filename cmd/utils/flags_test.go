// Copyright 2019 The go-ethereum Authors
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

// Package utils contains internal helper functions for go-ethereum commands.
package utils

import (
	"flag"
	"reflect"
	"testing"

	"github.com/ethereum/go-ethereum/eth/ethconfig"
	"github.com/urfave/cli/v2"
)

func Test_SplitTagsFlag(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		args string
		want map[string]string
	}{
		{
			"2 tags case",
			"host=localhost,bzzkey=123",
			map[string]string{
				"host":   "localhost",
				"bzzkey": "123",
			},
		},
		{
			"1 tag case",
			"host=localhost123",
			map[string]string{
				"host": "localhost123",
			},
		},
		{
			"empty case",
			"",
			map[string]string{},
		},
		{
			"garbage",
			"smth=smthelse=123",
			map[string]string{
				"smth": "smthelse=123",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := SplitTagsFlag(tt.args); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("splitTagsFlag() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsNetworkPresetUsesFlagValue(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		args []string
		want bool
	}{
		{
			name: "unset",
			want: false,
		},
		{
			name: "enabled",
			args: []string{"--sepolia"},
			want: true,
		},
		{
			name: "explicit false",
			args: []string{"--sepolia=false"},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctx := newTestContext(t, tt.args, NetworkFlags...)
			if got := IsNetworkPreset(ctx); got != tt.want {
				t.Fatalf("IsNetworkPreset() = %v, want %v", got, tt.want)
			}
		})
	}
}

// Tests that archive nodes index the entire chain by default, while honoring an
// explicitly configured transaction history.
func TestSetTransactionHistory(t *testing.T) {
	t.Parallel()

	var (
		def     = ethconfig.Defaults.TransactionHistory
		uint64p = func(n uint64) *uint64 { return &n }
	)
	tests := []struct {
		name    string
		archive bool
		toml    *uint64 // TransactionHistory from the config file
		legacy  uint64  // TxLookupLimit from the config file
		args    []string
		want    uint64
	}{
		{name: "full node, unset", want: def},
		{name: "full node, flag", args: []string{"--history.transactions=1000"}, want: 1000},
		{name: "full node, config file", toml: uint64p(1000), want: 1000},

		{name: "archive, unset", archive: true, want: 0},
		{name: "archive, flag", archive: true, args: []string{"--history.transactions=250000"}, want: 250000},
		{name: "archive, flag zero", archive: true, args: []string{"--history.transactions=0"}, want: 0},
		{name: "archive, flag default value", archive: true, args: []string{"--history.transactions=2350000"}, want: def},
		{name: "archive, config file", archive: true, toml: uint64p(250000), want: 250000},
		{name: "archive, config file zero", archive: true, toml: uint64p(0), want: 0},
		{name: "archive, flag overrides config file", archive: true, toml: uint64p(250000), args: []string{"--history.transactions=1000"}, want: 1000},
		{name: "archive, legacy config file", archive: true, legacy: 5000, want: 5000},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			cfg := ethconfig.Defaults
			cfg.NoPruning = tt.archive
			if tt.toml != nil {
				cfg.TransactionHistory = *tt.toml
			}
			if tt.legacy != 0 {
				cfg.TxLookupLimit = tt.legacy
			}
			setTransactionHistory(newTestContext(t, tt.args, TransactionHistoryFlag), &cfg)
			if cfg.TransactionHistory != tt.want {
				t.Fatalf("TransactionHistory = %d, want %d", cfg.TransactionHistory, tt.want)
			}
		})
	}
}

func newTestContext(t *testing.T, args []string, flags ...cli.Flag) *cli.Context {
	t.Helper()

	set := flag.NewFlagSet("test", flag.ContinueOnError)
	for _, f := range flags {
		if err := f.Apply(set); err != nil {
			t.Fatal(err)
		}
	}
	if err := set.Parse(args); err != nil {
		t.Fatal(err)
	}
	return cli.NewContext(nil, set, nil)
}
