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
	"github.com/XinFinOrg/XDPoSChain/core/rawdb"
	"io/ioutil"
	"math/big"
	"os"
	"path/filepath"
	"testing"

	"github.com/XinFinOrg/XDPoSChain/common"
	"github.com/XinFinOrg/XDPoSChain/core"
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

var daoGenesisHash = common.HexToHash("29a4b5d743bfbda3a7461974d49c62bf23ba5df9c8b01de8256e2ac2a9ae1cd8")
var daoGenesisForkBlock = big.NewInt(314)

// TestDAOForkBlockNewChain tests that the DAO hard-fork number and the nodes support/opposition is correctly
// set in the database after various initialization procedures and invocations.
func TestDAOForkBlockNewChain(t *testing.T) {
	for i, arg := range []struct {
		genesis     string
		expectBlock *big.Int
		expectVote  bool
	}{
		// Test DAO Default Mainnet
		// {"", params.XDCMainnetChainConfig.DAOForkBlock, false},
		// test DAO Init Old Privnet
		{daoOldGenesis, nil, false},
		// test DAO Default No Fork Privnet
		{daoNoForkGenesis, daoGenesisForkBlock, false},
		// test DAO Default Pro Fork Privnet
		{daoProForkGenesis, daoGenesisForkBlock, true},
	} {
		testDAOForkBlockNewChain(t, i, arg.genesis, arg.expectBlock, arg.expectVote)
	}
}

func testDAOForkBlockNewChain(t *testing.T, test int, genesis string, expectBlock *big.Int, expectVote bool) {
	// Create a temporary data directory to use and inspect later
	datadir := tmpdir(t)
	defer os.RemoveAll(datadir)

	// Start a XDC instance with the requested flags set and immediately terminate
	if genesis != "" {
		json := filepath.Join(datadir, "genesis.json")
		if err := ioutil.WriteFile(json, []byte(genesis), 0600); err != nil {
			t.Fatalf("test %d: failed to write genesis file: %v", test, err)
		}
		runXDC(t, "--datadir", datadir, "init", json).WaitExit()
	} else {
		// Force chain initialization
		args := []string{"--port", "0", "--maxpeers", "0", "--nodiscover", "--nat", "none", "--ipcdisable", "--datadir", datadir}
		XDC := runXDC(t, append(args, []string{"--exec", "2+2", "console"}...)...)
		XDC.WaitExit()
	}
	// Retrieve the DAO config flag from the database
	path := filepath.Join(datadir, "XDC", "chaindata")
	db, err := rawdb.NewLevelDBDatabase(path, 0, 0, "")
	if err != nil {
		t.Fatalf("test %d: failed to open test database: %v", test, err)
	}
	defer db.Close()

	genesisHash := common.HexToHash("8d13370621558f4ed0da587934473c0404729f28b0ff1d50e5fdd840457a2f17")
	if genesis != "" {
		genesisHash = daoGenesisHash
	}
	config, err := core.GetChainConfig(db, genesisHash)
	if err != nil {
		t.Errorf("test %d: failed to retrieve chain config: %v", test, err)
		return // we want to return here, the other checks can't make it past this point (nil panic).
	}
	// Validate the DAO hard-fork block number against the expected value
	if config.DAOForkBlock == nil {
		if expectBlock != nil {
			t.Errorf("test %d: dao hard-fork block mismatch: have nil, want %v", test, expectBlock)
		}
	} else if expectBlock == nil {
		t.Errorf("test %d: dao hard-fork block mismatch: have %v, want nil", test, config.DAOForkBlock)
	} else if config.DAOForkBlock.Cmp(expectBlock) != 0 {
		t.Errorf("test %d: dao hard-fork block mismatch: have %v, want %v", test, config.DAOForkBlock, expectBlock)
	}
	if config.DAOForkSupport != expectVote {
		t.Errorf("test %d: dao hard-fork support mismatch: have %v, want %v", test, config.DAOForkSupport, expectVote)
	}
}
