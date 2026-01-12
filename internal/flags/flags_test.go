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

package flags

import (
	"flag"
	"math/big"
	"runtime"
	"testing"

	"github.com/urfave/cli/v2"
)

func TestPathExpansion(t *testing.T) {
	home := HomeDir()
	var tests map[string]string

	if runtime.GOOS == "windows" {
		tests = map[string]string{
			`/home/someuser/tmp`:        `\home\someuser\tmp`,
			`~/tmp`:                     home + `\tmp`,
			`~thisOtherUser/b/`:         `~thisOtherUser\b`,
			`$DDDXXX/a/b`:               `\tmp\a\b`,
			`/a/b/`:                     `\a\b`,
			`C:\Documents\Newsletters\`: `C:\Documents\Newsletters`,
			`C:\`:                       `C:\`,
			`\\.\pipe\\pipe\geth621383`: `\\.\pipe\\pipe\geth621383`,
		}
	} else {
		tests = map[string]string{
			`/home/someuser/tmp`:        `/home/someuser/tmp`,
			`~/tmp`:                     home + `/tmp`,
			`~thisOtherUser/b/`:         `~thisOtherUser/b`,
			`$DDDXXX/a/b`:               `/tmp/a/b`,
			`/a/b/`:                     `/a/b`,
			`C:\Documents\Newsletters\`: `C:\Documents\Newsletters\`,
			`C:\`:                       `C:\`,
			`\\.\pipe\\pipe\geth621383`: `\\.\pipe\\pipe\geth621383`,
		}
	}

	t.Setenv(`DDDXXX`, `/tmp`)
	for test, expected := range tests {
		t.Run(test, func(t *testing.T) {
			t.Parallel()

			got := expandPath(test)
			if got != expected {
				t.Errorf(`test %s, got %s, expected %s\n`, test, got, expected)
			}
		})
	}
}

func TestBigFlagEnvValuePreserved(t *testing.T) {
	// Prepare a BigFlag with a non-zero default and an environment override.
	const envVar = "GETH_TEST_BIGFLAG"
	initialDefault := big.NewInt(123)
	bf := &BigFlag{
		Name:    "test.bigflag",
		Value:   new(big.Int).Set(initialDefault),
		EnvVars: []string{envVar},
	}

	t.Setenv(envVar, "0x10")

	fs := flag.NewFlagSet("test", flag.ContinueOnError)
	if err := bf.Apply(fs); err != nil {
		t.Fatalf("Apply failed: %v", err)
	}

	app := cli.NewApp()
	ctx := cli.NewContext(app, fs, nil)

	got := GlobalBig(ctx, bf.Name)
	if got == nil {
		t.Fatalf("GlobalBig returned nil")
	}
	expected := big.NewInt(16)
	if got.Cmp(expected) != 0 {
		t.Fatalf("GlobalBig = %v, want %v", got, expected)
	}
	if !bf.HasBeenSet {
		t.Fatalf("BigFlag.HasBeenSet = false, want true")
	}
	if bf.defaultValue == nil || bf.defaultValue.Cmp(initialDefault) != 0 {
		t.Fatalf("defaultValue = %v, want %v", bf.defaultValue, initialDefault)
	}
}
