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
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// bintrieCLIGenesis funds one EOA and one contract with code and storage on
// both sides of the header boundary (slots 1 and 64).
const bintrieCLIGenesis = `{
	"config": {
		"chainId": 1337,
		"homesteadBlock": 0,
		"eip150Block": 0,
		"eip155Block": 0,
		"eip158Block": 0,
		"byzantiumBlock": 0,
		"constantinopleBlock": 0,
		"petersburgBlock": 0,
		"istanbulBlock": 0,
		"berlinBlock": 0,
		"londonBlock": 0
	},
	"difficulty": "1",
	"gasLimit": "8000000",
	"alloc": {
		"0x1000000000000000000000000000000000000001": {"balance": "1000000"},
		"0x2000000000000000000000000000000000000002": {
			"balance": "1",
			"code": "0x5b5b5b5b5b5b5b5b5b5b5b5b5b5b5b5b5b5b5b5b",
			"storage": {
				"0x0000000000000000000000000000000000000000000000000000000000000001": "0x0000000000000000000000000000000000000000000000000000000000000011",
				"0x0000000000000000000000000000000000000000000000000000000000000040": "0x0000000000000000000000000000000000000000000000000000000000000022"
			}
		}
	}
}`

// TestBintrieConvertCLI drives the real command against a real datadir: the
// only test that runs the CLI action, and the only configuration in which
// the wipe meets an ancient directory and a node-layout triedb path.
func TestBintrieConvertCLI(t *testing.T) {
	t.Parallel()
	datadir := t.TempDir()
	json := filepath.Join(datadir, "genesis.json")
	if err := os.WriteFile(json, []byte(bintrieCLIGenesis), 0600); err != nil {
		t.Fatal(err)
	}
	runCmd := func(expectFail bool, args ...string) string {
		t.Helper()
		geth := runGeth(t, append([]string{"--datadir", datadir}, args...)...)
		geth.WaitExit()
		if failed := geth.ExitStatus() != 0; failed != expectFail {
			t.Fatalf("geth %v: exit status %d, expected failure %v\nstderr:\n%s",
				args, geth.ExitStatus(), expectFail, geth.StderrText())
		}
		return geth.StderrText()
	}

	// Init with preimages: the converter cannot run without them.
	runCmd(false, "--cache.preimages", "init", json)

	// A full conversion with both artifacts.
	outdir := t.TempDir()
	snapPath := filepath.Join(outdir, "snapshot.bin")
	prePath := filepath.Join(outdir, "preimages.bin")
	out := runCmd(false, "bintrie", "convert", "--snapshot-out", snapPath, "--preimages-out", prePath)
	for _, want := range []string{"Verified converted tree", "Verified converted flat state", "Conversion complete"} {
		if !strings.Contains(out, want) {
			t.Fatalf("conversion output lacks %q:\n%s", want, out)
		}
	}
	if fi, err := os.Stat(snapPath); err != nil || fi.Size() <= snapshotHeaderSize {
		t.Fatalf("snapshot artifact missing or empty: %v", err)
	}
	if fi, err := os.Stat(prePath); err != nil || fi.Size() == 0 {
		t.Fatalf("preimage file missing or empty: %v", err)
	}

	// A second run must refuse the converted namespace.
	out = runCmd(true, "bintrie", "convert")
	if !strings.Contains(out, "--force") {
		t.Fatalf("refusal does not mention --force:\n%s", out)
	}

	// --force wipes and reconverts; on a real datadir this is the one path
	// where the wipe resets the PBT freezers and removes the journal file.
	runCmd(false, "bintrie", "convert", "--force")

	// And once more, dropping the source afterwards.
	out = runCmd(false, "bintrie", "convert", "--force", "--delete-source")
	if !strings.Contains(out, "Source MPT data deleted") {
		t.Fatalf("deletion did not report completion:\n%s", out)
	}
}
