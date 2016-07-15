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
	"io/ioutil"
	"math/big"
	"os"
	"path/filepath"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/params"
)

var daoNoForkGenesis = `{
	"alloc"      : {},
	"coinbase"   : "0x0000000000000000000000000000000000000000",
	"difficulty" : "0x20000",
	"extraData"  : "",
	"gasLimit"   : "0x2fefd8",
	"nonce"      : "0x0000000000000042",
	"mixhash"    : "0x0000000000000000000000000000000000000000000000000000000000000000",
	"parentHash" : "0x0000000000000000000000000000000000000000000000000000000000000000",
	"timestamp"  : "0x00"
}`
var daoNoForkGenesisHash = common.HexToHash("5e1fc79cb4ffa4739177b5408045cd5d51c6cf766133f23f7cd72ee1f8d790e0")

var daoProForkGenesis = `{
	"alloc"      : {},
	"coinbase"   : "0x0000000000000000000000000000000000000000",
	"difficulty" : "0x20000",
	"extraData"  : "",
	"gasLimit"   : "0x2fefd8",
	"nonce"      : "0x0000000000000043",
	"mixhash"    : "0x0000000000000000000000000000000000000000000000000000000000000000",
	"parentHash" : "0x0000000000000000000000000000000000000000000000000000000000000000",
	"timestamp"  : "0x00",
	"config"     : {
		"daoForkBlock": 314
	}
}`
var daoProForkGenesisHash = common.HexToHash("c80f3c1c3d81ae6d8ea59edf35d3e4b723e4c8684ec71fdb6d4715e3f8add237")
var daoProForkBlock = big.NewInt(314)

// Tests that creating a new node to with or without the DAO fork flag will correctly
// set the genesis block but with DAO support explicitly set or unset in the chain
// config in the database.
func TestDAOSupportMainnet(t *testing.T) {
	testDAOForkBlockNewChain(t, false, "", true, params.MainNetDAOForkBlock)
}
func TestDAOSupportTestnet(t *testing.T) {
	testDAOForkBlockNewChain(t, true, "", true, params.TestNetDAOForkBlock)
}
func TestDAOSupportPrivnet(t *testing.T) {
	testDAOForkBlockNewChain(t, false, daoProForkGenesis, false, daoProForkBlock)
}
func TestDAONoSupportMainnet(t *testing.T) {
	testDAOForkBlockNewChain(t, false, "", false, nil)
}
func TestDAONoSupportTestnet(t *testing.T) {
	testDAOForkBlockNewChain(t, true, "", false, nil)
}
func TestDAONoSupportPrivnet(t *testing.T) {
	testDAOForkBlockNewChain(t, false, daoNoForkGenesis, false, nil)
}

func testDAOForkBlockNewChain(t *testing.T, testnet bool, genesis string, fork bool, expect *big.Int) {
	// Create a temporary data directory to use and inspect later
	datadir := tmpdir(t)
	defer os.RemoveAll(datadir)

	// Start a Geth instance with the requested flags set and immediately terminate
	if genesis != "" {
		json := filepath.Join(datadir, "genesis.json")
		if err := ioutil.WriteFile(json, []byte(genesis), 0600); err != nil {
			t.Fatalf("failed to write genesis file: %v", err)
		}
		runGeth(t, "--datadir", datadir, "init", json).cmd.Wait()
	}
	execDAOGeth(t, datadir, testnet, fork, false)

	// Retrieve the DAO config flag from the database
	path := filepath.Join(datadir, "chaindata")
	if testnet {
		path = filepath.Join(datadir, "testnet", "chaindata")
	}
	db, err := ethdb.NewLDBDatabase(path, 0, 0)
	if err != nil {
		t.Fatalf("failed to open test database: %v", err)
	}
	defer db.Close()

	genesisHash := common.HexToHash("0xd4e56740f876aef8c010b86a40d5f56745a118d0906a34e69aec8c0db1cb8fa3")
	if testnet {
		genesisHash = common.HexToHash("0x0cd786a2425d16f152c658316c423e6ce1181e15c3295826d7c9904cba9ce303")
	} else if genesis == daoNoForkGenesis {
		genesisHash = daoNoForkGenesisHash
	} else if genesis == daoProForkGenesis {
		genesisHash = daoProForkGenesisHash
	}
	config, err := core.GetChainConfig(db, genesisHash)
	if err != nil {
		t.Fatalf("failed to retrieve chain config: %v", err)
	}
	// Validate the DAO hard-fork block number against the expected value
	if config.DAOForkBlock == nil {
		if expect != nil {
			t.Fatalf("dao hard-fork block mismatch: have nil, want %v", expect)
		}
	} else if config.DAOForkBlock.Cmp(expect) != 0 {
		t.Fatalf("dao hard-fork block mismatch: have %v, want %v", config.DAOForkBlock, expect)
	}
}

// Tests that starting up an already existing node with various DAO fork override
// flags correctly changes the chain configs in the database.
func TestDAODefaultMainnet(t *testing.T) {
	testDAOForkBlockOldChain(t, false, "", false, false, false, false, nil)
}
func TestDAOStartSupportMainnet(t *testing.T) {
	testDAOForkBlockOldChain(t, false, "", false, true, false, false, params.MainNetDAOForkBlock)
}
func TestDAOContinueExplicitSupportMainnet(t *testing.T) {
	testDAOForkBlockOldChain(t, false, "", true, true, false, false, params.MainNetDAOForkBlock)
}
func TestDAOContinueImplicitSupportMainnet(t *testing.T) {
	testDAOForkBlockOldChain(t, false, "", true, false, false, false, params.MainNetDAOForkBlock)
}
func TestDAOSwitchSupportMainnet(t *testing.T) {
	testDAOForkBlockOldChain(t, false, "", false, true, true, false, params.MainNetDAOForkBlock)
}
func TestDAOStartOpposeMainnet(t *testing.T) {
	testDAOForkBlockOldChain(t, false, "", false, false, false, true, nil)
}
func TestDAOContinueExplicitOpposeMainnet(t *testing.T) {
	testDAOForkBlockOldChain(t, false, "", false, false, true, true, nil)
}
func TestDAOContinueImplicitOpposeMainnet(t *testing.T) {
	testDAOForkBlockOldChain(t, false, "", false, false, true, false, nil)
}
func TestDAOSwitchOpposeMainnet(t *testing.T) {
	testDAOForkBlockOldChain(t, false, "", true, false, false, true, nil)
}
func TestDAODefaultTestnet(t *testing.T) {
	testDAOForkBlockOldChain(t, true, "", false, false, false, false, nil)
}
func TestDAOStartSupportTestnet(t *testing.T) {
	testDAOForkBlockOldChain(t, true, "", false, true, false, false, params.TestNetDAOForkBlock)
}
func TestDAOContinueExplicitSupportTestnet(t *testing.T) {
	testDAOForkBlockOldChain(t, true, "", true, true, false, false, params.TestNetDAOForkBlock)
}
func TestDAOContinueImplicitSupportTestnet(t *testing.T) {
	testDAOForkBlockOldChain(t, true, "", true, false, false, false, params.TestNetDAOForkBlock)
}
func TestDAOSwitchSupportTestnet(t *testing.T) {
	testDAOForkBlockOldChain(t, true, "", false, true, true, false, params.TestNetDAOForkBlock)
}
func TestDAOStartOpposeTestnet(t *testing.T) {
	testDAOForkBlockOldChain(t, true, "", false, false, false, true, nil)
}
func TestDAOContinueExplicitOpposeTestnet(t *testing.T) {
	testDAOForkBlockOldChain(t, true, "", false, false, true, true, nil)
}
func TestDAOContinueImplicitOpposeTestnet(t *testing.T) {
	testDAOForkBlockOldChain(t, true, "", false, false, true, false, nil)
}
func TestDAOSwitchOpposeTestnet(t *testing.T) {
	testDAOForkBlockOldChain(t, true, "", true, false, false, true, nil)
}
func TestDAODefaultPrivnet(t *testing.T) {
	testDAOForkBlockOldChain(t, false, daoNoForkGenesis, false, false, false, false, nil)
}
func TestDAOStartSupportConPrivnet(t *testing.T) {
	testDAOForkBlockOldChain(t, false, daoNoForkGenesis, false, true, false, false, params.MainNetDAOForkBlock)
}
func TestDAOContinueExplicitSupportConPrivnet(t *testing.T) {
	testDAOForkBlockOldChain(t, false, daoNoForkGenesis, true, true, false, false, params.MainNetDAOForkBlock)
}
func TestDAOContinueImplicitSupportConPrivnet(t *testing.T) {
	testDAOForkBlockOldChain(t, false, daoNoForkGenesis, true, false, false, false, params.MainNetDAOForkBlock)
}
func TestDAOSwitchSupportConPrivnet(t *testing.T) {
	testDAOForkBlockOldChain(t, false, daoNoForkGenesis, false, true, true, false, params.MainNetDAOForkBlock)
}
func TestDAOStartOpposeConPrivnet(t *testing.T) {
	testDAOForkBlockOldChain(t, false, daoNoForkGenesis, false, false, false, true, nil)
}
func TestDAOContinueExplicitOpposeConPrivnet(t *testing.T) {
	testDAOForkBlockOldChain(t, false, daoNoForkGenesis, false, false, true, true, nil)
}
func TestDAOContinueImplicitOpposeConPrivnet(t *testing.T) {
	testDAOForkBlockOldChain(t, false, daoNoForkGenesis, false, false, true, false, nil)
}
func TestDAOSwitchOpposeConPrivnet(t *testing.T) {
	testDAOForkBlockOldChain(t, false, daoNoForkGenesis, true, false, false, true, nil)
}
func TestDAODefaultProPrivnet(t *testing.T) {
	testDAOForkBlockOldChain(t, false, daoProForkGenesis, false, false, false, false, daoProForkBlock)
}
func TestDAOStartSupportProPrivnet(t *testing.T) {
	testDAOForkBlockOldChain(t, false, daoProForkGenesis, false, true, false, false, daoProForkBlock)
}
func TestDAOContinueExplicitSupportProPrivnet(t *testing.T) {
	testDAOForkBlockOldChain(t, false, daoProForkGenesis, true, true, false, false, daoProForkBlock)
}
func TestDAOContinueImplicitSupportProPrivnet(t *testing.T) {
	testDAOForkBlockOldChain(t, false, daoProForkGenesis, true, false, false, false, daoProForkBlock)
}
func TestDAOSwitchSupportProPrivnet(t *testing.T) {
	testDAOForkBlockOldChain(t, false, daoProForkGenesis, false, true, true, false, params.MainNetDAOForkBlock)
}
func TestDAOStartOpposeProPrivnet(t *testing.T) {
	testDAOForkBlockOldChain(t, false, daoProForkGenesis, false, false, false, true, nil)
}
func TestDAOContinueExplicitOpposeProPrivnet(t *testing.T) {
	testDAOForkBlockOldChain(t, false, daoProForkGenesis, false, false, true, true, nil)
}
func TestDAOContinueImplicitOpposeProPrivnet(t *testing.T) {
	testDAOForkBlockOldChain(t, false, daoProForkGenesis, false, false, true, false, nil)
}
func TestDAOSwitchOpposeProPrivnet(t *testing.T) {
	testDAOForkBlockOldChain(t, false, daoProForkGenesis, true, false, false, true, nil)
}

func testDAOForkBlockOldChain(t *testing.T, testnet bool, genesis string, oldSupport, newSupport, oldOppose, newOppose bool, expect *big.Int) {
	// Create a temporary data directory to use and inspect later
	datadir := tmpdir(t)
	defer os.RemoveAll(datadir)

	// Cycle two Geth instances, possibly changing fork support in between
	if genesis != "" {
		json := filepath.Join(datadir, "genesis.json")
		if err := ioutil.WriteFile(json, []byte(genesis), 0600); err != nil {
			t.Fatalf("failed to write genesis file: %v", err)
		}
		runGeth(t, "--datadir", datadir, "init", json).cmd.Wait()
	}
	execDAOGeth(t, datadir, testnet, oldSupport, oldOppose)
	execDAOGeth(t, datadir, testnet, newSupport, newOppose)

	// Retrieve the DAO config flag from the database
	path := filepath.Join(datadir, "chaindata")
	if testnet {
		path = filepath.Join(datadir, "testnet", "chaindata")
	}
	db, err := ethdb.NewLDBDatabase(path, 0, 0)
	if err != nil {
		t.Fatalf("failed to open test database: %v", err)
	}
	defer db.Close()

	genesisHash := common.HexToHash("0xd4e56740f876aef8c010b86a40d5f56745a118d0906a34e69aec8c0db1cb8fa3")
	if testnet {
		genesisHash = common.HexToHash("0x0cd786a2425d16f152c658316c423e6ce1181e15c3295826d7c9904cba9ce303")
	} else if genesis == daoNoForkGenesis {
		genesisHash = daoNoForkGenesisHash
	} else if genesis == daoProForkGenesis {
		genesisHash = daoProForkGenesisHash
	}
	config, err := core.GetChainConfig(db, genesisHash)
	if err != nil {
		t.Fatalf("failed to retrieve chain config: %v", err)
	}
	// Validate the DAO hard-fork block number against the expected value
	if config.DAOForkBlock == nil {
		if expect != nil {
			t.Fatalf("dao hard-fork block mismatch: have nil, want %v", expect)
		}
	} else if config.DAOForkBlock.Cmp(expect) != 0 {
		t.Fatalf("dao hard-fork block mismatch: have %v, want %v", config.DAOForkBlock, expect)
	}
}

// execDAOGeth starts a Geth instance with some DAO forks set and terminates.
func execDAOGeth(t *testing.T, datadir string, testnet bool, supportFork bool, opposeFork bool) {
	args := []string{"--port", "0", "--maxpeers", "0", "--nodiscover", "--nat", "none", "--ipcdisable", "--datadir", datadir}
	if testnet {
		args = append(args, "--testnet")
	}
	if supportFork {
		args = append(args, "--support-dao-fork")
	}
	if opposeFork {
		args = append(args, "--oppose-dao-fork")
	}
	geth := runGeth(t, append(args, []string{"--exec", "2+2", "console"}...)...)
	geth.cmd.Wait()
}
