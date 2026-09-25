// Copyright 2026 The go-ethereum Authors
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
	"strings"
	"testing"
)

// TestDumpConfigForkOverrides checks that the fork override flags end up in
// the config written by dumpconfig.
func TestDumpConfigForkOverrides(t *testing.T) {
	t.Parallel()
	geth := runGeth(t, "--sepolia", "--datadir", t.TempDir(),
		"--override.osaka", "1700000000", "--override.amsterdam", "1800000000", "dumpconfig")
	out := string(geth.Output())
	geth.WaitExit()
	for _, want := range []string{"OverrideOsaka = 1700000000", "OverrideAmsterdam = 1800000000"} {
		if !strings.Contains(out, want) {
			t.Errorf("dumpconfig output missing %q", want)
		}
	}
}
