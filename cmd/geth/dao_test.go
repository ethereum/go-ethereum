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

// Genesis block for nodes which don't care about the DAO fork (i.e. not configured)
var daoOldGenesis = `{
	"alloc"      : {},
	"coinbase"   : "0x0000000000000000000000000000000000000000",
	"difficulty" : "0x20000",
	"extraData"  : "",
	"gasLimit"   : "0x2fefd8",
	"nonce"      : "0x0000000000000042",
	"mixhash"    : "0x0000000000000000000000000000000000000000000000000000000000000000",
	"parentHash" : "0x0000000000000000000000000000000000000000000000000000000000000000",
	"timestamp"  : "0x00",
	"config"     : {}
}`

// Genesis block for nodes which actively oppose the DAO fork
var daoNoForkGenesis = `{
	"alloc"      : {},
	"coinbase"   : "0x0000000000000000000000000000000000000000",
	"difficulty" : "0x20000",
	"extraData"  : "",
	"gasLimit"   : "0x2fefd8",
	"nonce"      : "0x0000000000000042",
	"mixhash"    : "0x0000000000000000000000000000000000000000000000000000000000000000",
	"parentHash" : "0x0000000000000000000000000000000000000000000000000000000000000000",
	"timestamp"  : "0x00",
	"config"     : {
		"daoForkBlock"   : 314,
		"daoForkSupport" : false
	}
}`

// Genesis block for nodes which actively support the DAO fork
var daoProForkGenesis = `{
	"alloc"      : {},
	"coinbase"   : "0x0000000000000000000000000000000000000000",
	"difficulty" : "0x20000",
	"extraData"  : "",
	"gasLimit"   : "0x2fefd8",
	"nonce"      : "0x0000000000000042",
	"mixhash"    : "0x0000000000000000000000000000000000000000000000000000000000000000",
	"parentHash" : "0x0000000000000000000000000000000000000000000000000000000000000000",
	"timestamp"  : "0x00",
	"config"     : {
		"daoForkBlock"   : 314,
		"daoForkSupport" : true
	}
}`

var daoGenesisHash = common.HexToHash("5e1fc79cb4ffa4739177b5408045cd5d51c6cf766133f23f7cd72ee1f8d790e0")
var daoGenesisForkBlock = big.NewInt(314)

// Tests that the DAO hard-fork number and the nodes support/opposition is correctly
// set in the database after various initialization procedures and invocations.
func TestDAODefaultMainnet(t *testing.T) {
	testDAOForkBlockNewChain(t, false, "", [][2]bool{{false, false}}, params.MainNetDAOForkBlock, true)
}
func TestDAOSupportMainnet(t *testing.T) {
	testDAOForkBlockNewChain(t, false, "", [][2]bool{{true, false}}, params.MainNetDAOForkBlock, true)
}
func TestDAOOpposeMainnet(t *testing.T) {
	testDAOForkBlockNewChain(t, false, "", [][2]bool{{false, true}}, params.MainNetDAOForkBlock, false)
}
func TestDAOSwitchToSupportMainnet(t *testing.T) {
	testDAOForkBlockNewChain(t, false, "", [][2]bool{{false, true}, {true, false}}, params.MainNetDAOForkBlock, true)
}
func TestDAOSwitchToOpposeMainnet(t *testing.T) {
	testDAOForkBlockNewChain(t, false, "", [][2]bool{{true, false}, {false, true}}, params.MainNetDAOForkBlock, false)
}
func TestDAODefaultTestnet(t *testing.T) {
	testDAOForkBlockNewChain(t, true, "", [][2]bool{{false, false}}, params.TestNetDAOForkBlock, true)
}
func TestDAOSupportTestnet(t *testing.T) {
	testDAOForkBlockNewChain(t, true, "", [][2]bool{{true, false}}, params.TestNetDAOForkBlock, true)
}
func TestDAOOpposeTestnet(t *testing.T) {
	testDAOForkBlockNewChain(t, true, "", [][2]bool{{false, true}}, params.TestNetDAOForkBlock, false)
}
func TestDAOSwitchToSupportTestnet(t *testing.T) {
	testDAOForkBlockNewChain(t, true, "", [][2]bool{{false, true}, {true, false}}, params.TestNetDAOForkBlock, true)
}
func TestDAOSwitchToOpposeTestnet(t *testing.T) {
	testDAOForkBlockNewChain(t, true, "", [][2]bool{{true, false}, {false, true}}, params.TestNetDAOForkBlock, false)
}
func TestDAOInitOldPrivnet(t *testing.T) {
	testDAOForkBlockNewChain(t, false, daoOldGenesis, [][2]bool{}, nil, false)
}
func TestDAODefaultOldPrivnet(t *testing.T) {
	testDAOForkBlockNewChain(t, false, daoOldGenesis, [][2]bool{{false, false}}, params.MainNetDAOForkBlock, true)
}
func TestDAOSupportOldPrivnet(t *testing.T) {
	testDAOForkBlockNewChain(t, false, daoOldGenesis, [][2]bool{{true, false}}, params.MainNetDAOForkBlock, true)
}
func TestDAOOpposeOldPrivnet(t *testing.T) {
	testDAOForkBlockNewChain(t, false, daoOldGenesis, [][2]bool{{false, true}}, params.MainNetDAOForkBlock, false)
}
func TestDAOSwitchToSupportOldPrivnet(t *testing.T) {
	testDAOForkBlockNewChain(t, false, daoOldGenesis, [][2]bool{{false, true}, {true, false}}, params.MainNetDAOForkBlock, true)
}
func TestDAOSwitchToOpposeOldPrivnet(t *testing.T) {
	testDAOForkBlockNewChain(t, false, daoOldGenesis, [][2]bool{{true, false}, {false, true}}, params.MainNetDAOForkBlock, false)
}
func TestDAOInitNoForkPrivnet(t *testing.T) {
	testDAOForkBlockNewChain(t, false, daoNoForkGenesis, [][2]bool{}, daoGenesisForkBlock, false)
}
func TestDAODefaultNoForkPrivnet(t *testing.T) {
	testDAOForkBlockNewChain(t, false, daoNoForkGenesis, [][2]bool{{false, false}}, daoGenesisForkBlock, false)
}
func TestDAOSupportNoForkPrivnet(t *testing.T) {
	testDAOForkBlockNewChain(t, false, daoNoForkGenesis, [][2]bool{{true, false}}, daoGenesisForkBlock, true)
}
func TestDAOOpposeNoForkPrivnet(t *testing.T) {
	testDAOForkBlockNewChain(t, false, daoNoForkGenesis, [][2]bool{{false, true}}, daoGenesisForkBlock, false)
}
func TestDAOSwitchToSupportNoForkPrivnet(t *testing.T) {
	testDAOForkBlockNewChain(t, false, daoNoForkGenesis, [][2]bool{{false, true}, {true, false}}, daoGenesisForkBlock, true)
}
func TestDAOSwitchToOpposeNoForkPrivnet(t *testing.T) {
	testDAOForkBlockNewChain(t, false, daoNoForkGenesis, [][2]bool{{true, false}, {false, true}}, daoGenesisForkBlock, false)
}
func TestDAOInitProForkPrivnet(t *testing.T) {
	testDAOForkBlockNewChain(t, false, daoProForkGenesis, [][2]bool{}, daoGenesisForkBlock, true)
}
func TestDAODefaultProForkPrivnet(t *testing.T) {
	testDAOForkBlockNewChain(t, false, daoProForkGenesis, [][2]bool{{false, false}}, daoGenesisForkBlock, true)
}
func TestDAOSupportProForkPrivnet(t *testing.T) {
	testDAOForkBlockNewChain(t, false, daoProForkGenesis, [][2]bool{{true, false}}, daoGenesisForkBlock, true)
}
func TestDAOOpposeProForkPrivnet(t *testing.T) {
	testDAOForkBlockNewChain(t, false, daoProForkGenesis, [][2]bool{{false, true}}, daoGenesisForkBlock, false)
}
func TestDAOSwitchToSupportProForkPrivnet(t *testing.T) {
	testDAOForkBlockNewChain(t, false, daoProForkGenesis, [][2]bool{{false, true}, {true, false}}, daoGenesisForkBlock, true)
}
func TestDAOSwitchToOpposeProForkPrivnet(t *testing.T) {
	testDAOForkBlockNewChain(t, false, daoProForkGenesis, [][2]bool{{true, false}, {false, true}}, daoGenesisForkBlock, false)
}

func testDAOForkBlockNewChain(t *testing.T, testnet bool, genesis string, votes [][2]bool, expectBlock *big.Int, expectVote bool) {
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
	for _, vote := range votes {
		args := []string{"--port", "0", "--maxpeers", "0", "--nodiscover", "--nat", "none", "--ipcdisable", "--datadir", datadir}
		if testnet {
			args = append(args, "--testnet")
		}
		if vote[0] {
			args = append(args, "--support-dao-fork")
		}
		if vote[1] {
			args = append(args, "--oppose-dao-fork")
		}
		geth := runGeth(t, append(args, []string{"--exec", "2+2", "console"}...)...)
		geth.cmd.Wait()
	}
	// Retrieve the DAO config flag from the database
	path := filepath.Join(datadir, "geth", "chaindata")
	if testnet && genesis == "" {
		path = filepath.Join(datadir, "testnet", "geth", "chaindata")
	}
	db, err := ethdb.NewLDBDatabase(path, 0, 0)
	if err != nil {
		t.Fatalf("failed to open test database: %v", err)
	}
	defer db.Close()

	genesisHash := common.HexToHash("0xd4e56740f876aef8c010b86a40d5f56745a118d0906a34e69aec8c0db1cb8fa3")
	if testnet {
		genesisHash = common.HexToHash("0x0cd786a2425d16f152c658316c423e6ce1181e15c3295826d7c9904cba9ce303")
	}
	if genesis != "" {
		genesisHash = daoGenesisHash
	}
	config, err := core.GetChainConfig(db, genesisHash)
	if err != nil {
		t.Fatalf("failed to retrieve chain config: %v", err)
	}
	// Validate the DAO hard-fork block number against the expected value
	if config.DAOForkBlock == nil {
		if expectBlock != nil {
			t.Errorf("dao hard-fork block mismatch: have nil, want %v", expectBlock)
		}
	} else if expectBlock == nil {
		t.Errorf("dao hard-fork block mismatch: have %v, want nil", config.DAOForkBlock)
	} else if config.DAOForkBlock.Cmp(expectBlock) != 0 {
		t.Errorf("dao hard-fork block mismatch: have %v, want %v", config.DAOForkBlock, expectBlock)
	}
	if config.DAOForkSupport != expectVote {
		t.Errorf("dao hard-fork support mismatch: have %v, want %v", config.DAOForkSupport, expectVote)
	}
}
