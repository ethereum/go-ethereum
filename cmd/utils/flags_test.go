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
			map[string]string{},
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

func TestSetEthConfigAllowsDisablingStateSizeTracking(t *testing.T) {
	ctx := newTestFlagContext(t, []cli.Flag{
		CacheFlag,
		CacheDatabaseFlag,
		CacheGCFlag,
		CacheSnapshotFlag,
		CacheTrieFlag,
		CryptoKZGFlag,
		FDLimitFlag,
		GCModeFlag,
		SnapshotFlag,
		StateSizeTrackingFlag,
	}, "--state.size-tracking=false")

	cfg := ethconfig.Defaults
	cfg.EnableStateSizeTracking = true
	SetEthConfig(ctx, nil, &cfg)

	if cfg.EnableStateSizeTracking {
		t.Fatal("state size tracking should be disabled by explicit CLI flag")
	}
}

func newTestFlagContext(t *testing.T, flags []cli.Flag, args ...string) *cli.Context {
	t.Helper()

	set := flag.NewFlagSet(t.Name(), flag.ContinueOnError)
	for _, flag := range flags {
		if err := flag.Apply(set); err != nil {
			t.Fatal(err)
		}
	}
	if err := set.Parse(args); err != nil {
		t.Fatal(err)
	}
	return cli.NewContext(cli.NewApp(), set, nil)
}
