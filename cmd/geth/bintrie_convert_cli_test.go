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
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/ethdb/pebble"
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

// bintrieCLIEmptyGenesis is bintrieCLIGenesis with nothing allocated.
var bintrieCLIEmptyGenesis = bintrieCLIGenesis[:strings.Index(bintrieCLIGenesis, `"alloc"`)] + `"alloc": {}}`

// TestBintrieExportCLI drives the producer's command against a real datadir:
// it writes both artifacts and leaves the node's database without binary
// tree state, needs both outputs, and refuses a head past the fork.
func TestBintrieExportCLI(t *testing.T) {
	t.Parallel()
	datadir := t.TempDir()
	json := filepath.Join(datadir, "genesis.json")
	if err := os.WriteFile(json, []byte(bintrieCLIGenesis), 0600); err != nil {
		t.Fatal(err)
	}
	run := gethRunner(t)
	run(datadir, false, "--cache.preimages", "init", json)

	outdir := t.TempDir()
	snapPath := filepath.Join(outdir, "snapshot.bin")
	prePath := filepath.Join(outdir, "preimages.bin")
	out := run(datadir, false, "bintrie", "export", "--snapshot-out", snapPath, "--preimages-out", prePath)
	if !strings.Contains(out, "Export complete") {
		t.Fatalf("export output lacks completion:\n%s", out)
	}
	if fi, err := os.Stat(snapPath); err != nil || fi.Size() <= snapshotOverhead {
		t.Fatalf("snapshot artifact missing or empty: %v", err)
	}
	if fi, err := os.Stat(prePath); err != nil || fi.Size() == 0 {
		t.Fatalf("preimage file missing or empty: %v", err)
	}
	chaindb := mustOpenChainDB(t, datadir)
	if dirty, err := hasBinaryTrieState(rawdb.NewDatabase(chaindb)); err != nil || dirty {
		t.Fatalf("the export wrote binary tree state into the node (err %v)", err)
	}
	if err := chaindb.Close(); err != nil {
		t.Fatal(err)
	}
	if out := run(datadir, true, "bintrie", "export", "--snapshot-out", snapPath); !strings.Contains(out, "are required") {
		t.Fatalf("an export without a preimage file output was not refused:\n%s", out)
	}

	// A head past the fork commits the binary tree: no merkle state to export.
	chaindb = mustOpenChainDB(t, datadir)
	db := rawdb.NewDatabase(chaindb)
	ghash := rawdb.ReadCanonicalHash(db, 0)
	cfg := rawdb.ReadChainConfig(db, ghash)
	cfg.BinaryTrieTime = new(uint64)
	rawdb.WriteChainConfig(db, ghash, cfg)
	if err := chaindb.Close(); err != nil {
		t.Fatal(err)
	}
	if out := run(datadir, true, "bintrie", "export", "--snapshot-out", snapPath, "--preimages-out", prePath); !strings.Contains(out, "commits the binary tree") {
		t.Fatalf("export not refused past the fork:\n%s", out)
	}
}

// mustOpenChainDB opens chaindata directly; the CLI cannot write the marker.
func mustOpenChainDB(t *testing.T, datadir string) ethdb.KeyValueStore {
	t.Helper()
	db, err := pebble.New(filepath.Join(datadir, "geth", "chaindata"), 0, 0, "", false)
	if err != nil {
		t.Fatalf("failed to open chaindata: %v", err)
	}
	return db
}

// TestBintrieImportCLI drives the consumer story end to end through the real
// command: artifacts produced on one datadir, verified and imported on a
// second that holds headers but no state and no preimages, anchored at the
// genesis block resolved from its own header chain.
func TestBintrieImportCLI(t *testing.T) {
	t.Parallel()
	producer := t.TempDir()
	json := filepath.Join(producer, "genesis.json")
	if err := os.WriteFile(json, []byte(bintrieCLIGenesis), 0600); err != nil {
		t.Fatal(err)
	}
	run := gethRunner(t)

	// The producer exports.
	outdir := t.TempDir()
	snapPath := filepath.Join(outdir, "snapshot.bin")
	prePath := filepath.Join(outdir, "preimages.bin")
	run(producer, false, "--cache.preimages", "init", json)
	run(producer, false, "bintrie", "export", "--snapshot-out", snapPath, "--preimages-out", prePath)

	// The consumer holds only headers: same genesis, no preimages.
	consumer := t.TempDir()
	run(consumer, false, "init", json)

	// Verify first, writing nothing; then import; then refuse a re-import.
	out := run(consumer, false, "bintrie", "import", "--verify-only", snapPath, prePath, "0")
	if !strings.Contains(out, "nothing written") {
		t.Fatalf("verify-only did not report itself:\n%s", out)
	}
	out = run(consumer, false, "bintrie", "import", snapPath, prePath, "0")
	if !strings.Contains(out, "Import complete") {
		t.Fatalf("import did not complete:\n%s", out)
	}
	out = run(consumer, true, "bintrie", "import", snapPath, prePath, "0")
	if !strings.Contains(out, "--force") {
		t.Fatalf("re-import refusal does not mention --force:\n%s", out)
	}
	// --force wipes and re-imports; on a real datadir this is the one path
	// where the wipe meets an ancient directory and a node-layout journal.
	run(consumer, false, "bintrie", "import", "--force", snapPath, prePath, "0")

	// Past the fork the namespace is the live state: import --force is refused.
	chaindb := mustOpenChainDB(t, consumer)
	db := rawdb.NewDatabase(chaindb)
	ghash := rawdb.ReadCanonicalHash(db, 0)
	cfg := rawdb.ReadChainConfig(db, ghash)
	cfg.BinaryTrieTime = new(uint64)
	rawdb.WriteChainConfig(db, ghash, cfg)
	if err := chaindb.Close(); err != nil {
		t.Fatal(err)
	}
	out = run(consumer, true, "bintrie", "import", "--force", snapPath, prePath, "0")
	if !strings.Contains(out, "commits the binary tree") {
		t.Fatalf("import --force not refused past the fork:\n%s", out)
	}
	after := mustOpenChainDB(t, consumer)
	defer after.Close()
	if !rawdb.ReadPBTFlatState(rawdb.NewTable(rawdb.NewDatabase(after), string(rawdb.PBTPrefix))) {
		t.Fatal("refused --force wiped the imported namespace")
	}
}

// TestBintrieEmptyStateCLI: an anchor with no accounts exports to the empty
// snapshot, a zero root and three zero counts, beside an empty preimage file,
// and the pair verifies against the same empty anchor.
func TestBintrieEmptyStateCLI(t *testing.T) {
	t.Parallel()
	producer, consumer, outdir := t.TempDir(), t.TempDir(), t.TempDir()
	json := filepath.Join(outdir, "genesis.json")
	if err := os.WriteFile(json, []byte(bintrieCLIEmptyGenesis), 0600); err != nil {
		t.Fatal(err)
	}
	run := func(datadir string, args ...string) string {
		t.Helper()
		geth := runGeth(t, append([]string{"--datadir", datadir}, args...)...)
		geth.WaitExit()
		if geth.ExitStatus() != 0 {
			t.Fatalf("geth %v: exit status %d\nstderr:\n%s", args, geth.ExitStatus(), geth.StderrText())
		}
		return geth.StderrText()
	}
	snapPath, prePath := filepath.Join(outdir, "snapshot.bin"), filepath.Join(outdir, "preimages.bin")
	run(producer, "--cache.preimages", "init", json)
	run(producer, "bintrie", "export", "--snapshot-out", snapPath, "--preimages-out", prePath)
	if fi, err := os.Stat(snapPath); err != nil || fi.Size() != snapshotOverhead {
		t.Fatalf("empty snapshot: %v, want %d bytes", fi, snapshotOverhead)
	}
	if fi, err := os.Stat(prePath); err != nil || fi.Size() != 0 {
		t.Fatalf("empty preimage file: %v, want 0 bytes", fi)
	}
	run(consumer, "init", json)
	if out := run(consumer, "bintrie", "import", "--verify-only", snapPath, prePath, "0"); !strings.Contains(out, "nothing written") {
		t.Fatalf("verify-only did not accept the empty pair:\n%s", out)
	}
}

// TestBintrieExportFromPreimageFileCLI: a node that never kept preimages
// exports by taking its keys from a preimage file of the same state, and
// writes the producer's artifacts byte for byte.
func TestBintrieExportFromPreimageFileCLI(t *testing.T) {
	t.Parallel()
	producer, node, outdir := t.TempDir(), t.TempDir(), t.TempDir()
	json := filepath.Join(outdir, "genesis.json")
	if err := os.WriteFile(json, []byte(bintrieCLIGenesis), 0600); err != nil {
		t.Fatal(err)
	}
	run := gethRunner(t)
	snapPath, prePath := filepath.Join(outdir, "snapshot.bin"), filepath.Join(outdir, "preimages.bin")
	run(producer, false, "--cache.preimages", "init", json)
	run(producer, false, "bintrie", "export", "--snapshot-out", snapPath, "--preimages-out", prePath)

	run(node, false, "init", json)
	nodeSnap, nodePre := filepath.Join(outdir, "node-snapshot.bin"), filepath.Join(outdir, "node-preimages.bin")
	if out := run(node, true, "bintrie", "export", "--snapshot-out", nodeSnap, "--preimages-out", nodePre); !strings.Contains(out, "no preimages") {
		t.Fatalf("an export with no preimages was not refused:\n%s", out)
	}
	// Another spelling of the input as an output would truncate it unread.
	if out := run(node, true, "bintrie", "export", "--preimages", prePath, "--snapshot-out", nodeSnap, "--preimages-out", outdir+"/./preimages.bin"); !strings.Contains(out, "must not name an output file") {
		t.Fatalf("an output aliasing the input was not refused:\n%s", out)
	}
	run(node, false, "bintrie", "export", "--preimages", prePath, "--snapshot-out", nodeSnap, "--preimages-out", nodePre)
	for _, pair := range [][2]string{{snapPath, nodeSnap}, {prePath, nodePre}} {
		want, err := os.ReadFile(pair[0])
		if err != nil {
			t.Fatal(err)
		}
		got, err := os.ReadFile(pair[1])
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(got, want) {
			t.Fatalf("%s from the preimage file differs from the producer's %s", pair[1], pair[0])
		}
	}
}

// gethRunner returns a runner of geth on a datadir that fails t unless the
// exit status matches expectFail, and returns stderr.
func gethRunner(t *testing.T) func(datadir string, expectFail bool, args ...string) string {
	return func(datadir string, expectFail bool, args ...string) string {
		t.Helper()
		geth := runGeth(t, append([]string{"--datadir", datadir}, args...)...)
		geth.WaitExit()
		if failed := geth.ExitStatus() != 0; failed != expectFail {
			t.Fatalf("geth %v: exit status %d, expected failure %v\nstderr:\n%s",
				args, geth.ExitStatus(), expectFail, geth.StderrText())
		}
		return geth.StderrText()
	}
}
