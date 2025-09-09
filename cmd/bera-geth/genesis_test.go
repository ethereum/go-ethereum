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
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

var customGenesisTests = []struct {
	genesis string
	query   string
	result  string
}{
	// Genesis file with a mostly-empty chain configuration (ensure missing fields work)
	{
		genesis: `{
			"alloc"      : {},
			"coinbase"   : "0x0000000000000000000000000000000000000000",
			"difficulty" : "0x20000",
			"extraData"  : "",
			"gasLimit"   : "0x2fefd8",
			"nonce"      : "0x0000000000001338",
			"mixhash"    : "0x0000000000000000000000000000000000000000000000000000000000000000",
			"parentHash" : "0x0000000000000000000000000000000000000000000000000000000000000000",
			"timestamp"  : "0x00",
			"config": {
				"terminalTotalDifficulty": 0
			}
		}`,
		query:  "eth.getBlock(0).nonce",
		result: "0x0000000000001338",
	},
	// Genesis file with specific chain configurations
	{
		genesis: `{
			"alloc"      : {},
			"coinbase"   : "0x0000000000000000000000000000000000000000",
			"difficulty" : "0x20000",
			"extraData"  : "",
			"gasLimit"   : "0x2fefd8",
			"nonce"      : "0x0000000000001339",
			"mixhash"    : "0x0000000000000000000000000000000000000000000000000000000000000000",
			"parentHash" : "0x0000000000000000000000000000000000000000000000000000000000000000",
			"timestamp"  : "0x00",
			"config"     : {
				"homesteadBlock"                : 42,
				"daoForkBlock"                  : 141,
				"daoForkSupport"                : true,
				"terminalTotalDifficulty": 0
			}
		}`,
		query:  "eth.getBlock(0).nonce",
		result: "0x0000000000001339",
	},
}

// Tests that initializing Geth with a custom genesis block and chain definitions
// work properly.
func TestCustomGenesis(t *testing.T) {
	t.Parallel()
	for i, tt := range customGenesisTests {
		// Create a temporary data directory to use and inspect later
		datadir := t.TempDir()

		// Initialize the data directory with the custom genesis block
		json := filepath.Join(datadir, "genesis.json")
		if err := os.WriteFile(json, []byte(tt.genesis), 0600); err != nil {
			t.Fatalf("test %d: failed to write genesis file: %v", i, err)
		}
		runGeth(t, "--datadir", datadir, "init", json).WaitExit()

		// Query the custom genesis block
		geth := runGeth(t, "--networkid", "1337", "--syncmode=full", "--cache", "16",
			"--datadir", datadir, "--maxpeers", "0", "--port", "0", "--authrpc.port", "0",
			"--nodiscover", "--nat", "none", "--ipcdisable",
			"--exec", tt.query, "console")
		geth.ExpectRegexp(tt.result)
		geth.ExpectExit()
	}
}

// TestPBSSRepeatInitNoTrieOpen verifies that running init twice on a PBSS (path-scheme)
// database does not re-initialize the trie database and safely short-circuits.
func TestPBSSRepeatInitNoTrieOpen(t *testing.T) {
	t.Parallel()

	// Minimal genesis with TTD=0 to keep execution simple
	genesis := `{
        "alloc": {"0x0000000000000000000000000000000000000001": {"balance": "0x1"}},
        "coinbase": "0x0000000000000000000000000000000000000000",
        "difficulty": "0x1",
        "gasLimit": "0x2fefd8",
        "nonce": "0x0000000000000000",
        "mixhash": "0x0000000000000000000000000000000000000000000000000000000000000000",
        "parentHash": "0x0000000000000000000000000000000000000000000000000000000000000000",
        "timestamp": "0x00",
        "config": { "terminalTotalDifficulty": 0 }
    }`

	datadir := t.TempDir()
	jsonPath := filepath.Join(datadir, "genesis.json")
	if err := os.WriteFile(jsonPath, []byte(genesis), 0600); err != nil {
		t.Fatalf("failed to write genesis file: %v", err)
	}

	// First init should perform normal initialization
	gethInit1 := runGeth(t, "--datadir", datadir, "init", jsonPath)
	gethInit1.WaitExit()
	if gethInit1.Err != nil {
		t.Fatalf("first init failed: %v", gethInit1.Err)
	}

	// Second init should skip trie db initialization and not truncate freezer
	gethInit2 := runGeth(t, "--datadir", datadir, "init", jsonPath)
	gethInit2.WaitExit()
	if gethInit2.Err != nil {
		t.Fatalf("second init failed: %v", gethInit2.Err)
	}
	stderr := gethInit2.StderrText()
	if !strings.Contains(stderr, "PBSS db already initialized with genesis, skipping trie db initialization") {
		t.Fatalf("expected skip-triedb log not found in stderr. got:\n%s", stderr)
	}
}

// TestPBSSConfigUpdate verifies that a repeat init on PBSS writes updated chain
// configuration without touching the trie database when the genesis block hash
// is unchanged.
func TestPBSSConfigUpdate(t *testing.T) {
	t.Parallel()

	// Base genesis (no Prague1 yet). Include all prior timestamp forks explicitly
	// with timestamp 0 to satisfy fork ordering: Shanghai -> Cancun -> Prague -> Prague1.
	baseGenesis := `{
        "alloc": {"0x0000000000000000000000000000000000000001": {"balance": "0x1"}},
        "coinbase": "0x0000000000000000000000000000000000000000",
        "difficulty": "0x1",
        "gasLimit": "0x2fefd8",
        "nonce": "0x0000000000000000",
        "mixhash": "0x0000000000000000000000000000000000000000000000000000000000000000",
        "parentHash": "0x0000000000000000000000000000000000000000000000000000000000000000",
        "timestamp": "0x00",
        "config": { 
            "terminalTotalDifficulty": 0,
			"homesteadBlock": 0,
			"eip150Block": 0,
			"eip155Block": 0,
			"eip158Block": 0,
			"byzantiumBlock": 0,
			"constantinopleBlock": 0,
			"petersburgBlock": 0,
			"istanbulBlock": 0,
			"berlinBlock": 0,
			"londonBlock": 0,
            "shanghaiTime": 0
        }
    }`

	// Updated config: enable Berachain Prague1 while keeping genesis state identical.
	// Include all prior timestamp forks to maintain correct ordering.
	updatedGenesis := `{
        "alloc": {"0x0000000000000000000000000000000000000001": {"balance": "0x1"}},
        "coinbase": "0x0000000000000000000000000000000000000000",
        "difficulty": "0x1",
        "gasLimit": "0x2fefd8",
        "nonce": "0x0000000000000000",
        "mixhash": "0x0000000000000000000000000000000000000000000000000000000000000000",
        "parentHash": "0x0000000000000000000000000000000000000000000000000000000000000000",
        "timestamp": "0x00",
        "config": {
            "terminalTotalDifficulty": 0,
			"homesteadBlock": 0,
			"eip150Block": 0,
			"eip155Block": 0,
			"eip158Block": 0,
			"byzantiumBlock": 0,
			"constantinopleBlock": 0,
			"petersburgBlock": 0,
			"istanbulBlock": 0,
			"berlinBlock": 0,
			"londonBlock": 0,
            "shanghaiTime": 0,
            "berachain": {
                "prague1": {
                    "time": 1000,
                    "baseFeeChangeDenominator": 48,
                    "poLDistributorAddress": "0x1111111111111111111111111111111111111111"
                }
            }
        }
    }`

	datadir := t.TempDir()
	jsonPath := filepath.Join(datadir, "genesis.json")
	if err := os.WriteFile(jsonPath, []byte(baseGenesis), 0600); err != nil {
		t.Fatalf("failed to write base genesis file: %v", err)
	}

	// First init with base config
	runGeth(t, "--datadir", datadir, "init", jsonPath).WaitExit()

	// Overwrite with updated config (same genesis state, different chain config)
	if err := os.WriteFile(jsonPath, []byte(updatedGenesis), 0600); err != nil {
		t.Fatalf("failed to write updated genesis file: %v", err)
	}

	// Second init should skip trie db init and write the new chain config
	geth := runGeth(t, "--datadir", datadir, "init", jsonPath)
	geth.WaitExit()
	stderr := geth.StderrText()
	if !strings.Contains(stderr, "PBSS db already initialized with genesis, skipping trie db initialization") {
		t.Fatalf("expected PBSS skip log not found in stderr. got:\n%s", stderr)
	}
	if !strings.Contains(stderr, "Writing new chain config") {
		t.Fatalf("expected config-update log not found in stderr. got:\n%s", stderr)
	}
}

// TestCustomBackend that the backend selection and detection (leveldb vs pebble) works properly.
func TestCustomBackend(t *testing.T) {
	t.Parallel()
	// Test pebble, but only on 64-bit platforms
	if strconv.IntSize != 64 {
		t.Skip("Custom backends are only available on 64-bit platform")
	}
	genesis := `{
		"alloc"      : {},
		"coinbase"   : "0x0000000000000000000000000000000000000000",
			"difficulty" : "0x20000",
			"extraData"  : "",
			"gasLimit"   : "0x2fefd8",
			"nonce"      : "0x0000000000001338",
			"mixhash"    : "0x0000000000000000000000000000000000000000000000000000000000000000",
			"parentHash" : "0x0000000000000000000000000000000000000000000000000000000000000000",
			"timestamp"  : "0x00",
			"config": {
				"terminalTotalDifficulty": 0
			}
		}`
	type backendTest struct {
		initArgs   []string
		initExpect string
		execArgs   []string
		execExpect string
	}
	testfunc := func(t *testing.T, tt backendTest) error {
		// Create a temporary data directory to use and inspect later
		datadir := t.TempDir()

		// Initialize the data directory with the custom genesis block
		json := filepath.Join(datadir, "genesis.json")
		if err := os.WriteFile(json, []byte(genesis), 0600); err != nil {
			return fmt.Errorf("failed to write genesis file: %v", err)
		}
		{ // Init
			args := append(tt.initArgs, "--datadir", datadir, "init", json)
			geth := runGeth(t, args...)
			geth.ExpectRegexp(tt.initExpect)
			geth.ExpectExit()
		}
		{ // Exec + query
			args := append(tt.execArgs, "--networkid", "1337", "--syncmode=full", "--cache", "16",
				"--datadir", datadir, "--maxpeers", "0", "--port", "0", "--authrpc.port", "0",
				"--nodiscover", "--nat", "none", "--ipcdisable",
				"--exec", "eth.getBlock(0).nonce", "console")
			geth := runGeth(t, args...)
			geth.ExpectRegexp(tt.execExpect)
			geth.ExpectExit()
		}
		return nil
	}
	for i, tt := range []backendTest{
		{ // When not specified, it should default to pebble
			execArgs:   []string{"--db.engine", "pebble"},
			execExpect: "0x0000000000001338",
		},
		{ // Explicit leveldb
			initArgs:   []string{"--db.engine", "leveldb"},
			execArgs:   []string{"--db.engine", "leveldb"},
			execExpect: "0x0000000000001338",
		},
		{ // Explicit leveldb first, then autodiscover
			initArgs:   []string{"--db.engine", "leveldb"},
			execExpect: "0x0000000000001338",
		},
		{ // Explicit pebble
			initArgs:   []string{"--db.engine", "pebble"},
			execArgs:   []string{"--db.engine", "pebble"},
			execExpect: "0x0000000000001338",
		},
		{ // Explicit pebble, then auto-discover
			initArgs:   []string{"--db.engine", "pebble"},
			execExpect: "0x0000000000001338",
		},
		{ // Can't start pebble on top of leveldb
			initArgs:   []string{"--db.engine", "leveldb"},
			execArgs:   []string{"--db.engine", "pebble"},
			execExpect: `Fatal: Failed to register the Ethereum service: db.engine choice was pebble but found pre-existing leveldb database in specified data directory`,
		},
		{ // Can't start leveldb on top of pebble
			initArgs:   []string{"--db.engine", "pebble"},
			execArgs:   []string{"--db.engine", "leveldb"},
			execExpect: `Fatal: Failed to register the Ethereum service: db.engine choice was leveldb but found pre-existing pebble database in specified data directory`,
		},
		{ // Reject invalid backend choice
			initArgs:   []string{"--db.engine", "mssql"},
			initExpect: `Fatal: Invalid choice for db.engine 'mssql', allowed 'leveldb' or 'pebble'`,
			// Since the init fails, this will return the (default) berachain mainnet genesis
			// block nonce
			execExpect: `0x0000000000001234`,
		},
	} {
		if err := testfunc(t, tt); err != nil {
			t.Fatalf("test %d-leveldb: %v", i, err)
		}
	}
}
