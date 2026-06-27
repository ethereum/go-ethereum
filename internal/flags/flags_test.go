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
)

func TestBigFlagApplyPreservesDefaultValue(t *testing.T) {
	defaultVal := big.NewInt(12345)
	f := &BigFlag{
		Name:  "test-big",
		Value: defaultVal,
	}
	set := flag.NewFlagSet("test", flag.ContinueOnError)
	if err := f.Apply(set); err != nil {
		t.Fatalf("Apply failed: %v", err)
	}
	if f.Value.Cmp(defaultVal) != 0 {
		t.Fatalf("Value after Apply = %s, want %s", f.Value, defaultVal)
	}
}

func TestBigFlagApplyPreservesEnvValue(t *testing.T) {
	t.Setenv("TEST_BIG_FLAG", "999")
	f := &BigFlag{
		Name:    "test-big",
		Value:   big.NewInt(1),
		EnvVars: []string{"TEST_BIG_FLAG"},
	}
	set := flag.NewFlagSet("test", flag.ContinueOnError)
	if err := f.Apply(set); err != nil {
		t.Fatalf("Apply failed: %v", err)
	}
	want := big.NewInt(999)
	if f.Value.Cmp(want) != 0 {
		t.Fatalf("Value after Apply = %s, want %s", f.Value, want)
	}
	if !f.HasBeenSet {
		t.Fatal("HasBeenSet should be true when env var is set")
	}
}

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
