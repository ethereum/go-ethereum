// Copyright 2017 The go-ethereum Authors
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

package ubqhash

import (
	// "encoding/json"
	"math/big"
	// "os"
	// "path/filepath"
	"testing"

	// "github.com/ubiq/go-ubiq/common/math"
	// "github.com/ubiq/go-ubiq/core"
	// "github.com/ubiq/go-ubiq/core/types"
	// "github.com/ubiq/go-ubiq/core/vm"
	// "github.com/ubiq/go-ubiq/ethdb"
	"github.com/ubiq/go-ubiq/params"
)

// TODO: write new difficulty tests
/*
type diffTest struct {
	ParentTimestamp    uint64
	ParentDifficulty   *big.Int
	CurrentTimestamp   uint64
	CurrentBlocknumber *big.Int
	CurrentDifficulty  *big.Int
}

func (d *diffTest) UnmarshalJSON(b []byte) (err error) {
	var ext struct {
		ParentTimestamp    string
		ParentDifficulty   string
		CurrentTimestamp   string
		CurrentBlocknumber string
		CurrentDifficulty  string
	}
	if err := json.Unmarshal(b, &ext); err != nil {
		return err
	}

	d.ParentTimestamp = math.MustParseUint64(ext.ParentTimestamp)
	d.ParentDifficulty = math.MustParseBig256(ext.ParentDifficulty)
	d.CurrentTimestamp = math.MustParseUint64(ext.CurrentTimestamp)
	d.CurrentBlocknumber = math.MustParseBig256(ext.CurrentBlocknumber)
	d.CurrentDifficulty = math.MustParseBig256(ext.CurrentDifficulty)

	return nil
}

func TestCalcDifficulty(t *testing.T) {
	file, err := os.Open(filepath.Join("..", "..", "tests", "testdata", "BasicTests", "difficulty.json"))
	if err != nil {
		t.Skip(err)
	}
	defer file.Close()

	tests := make(map[string]diffTest)
	err = json.NewDecoder(file).Decode(&tests)
	if err != nil {
		t.Fatal(err)
	}

	config := &params.ChainConfig{HomesteadBlock: big.NewInt(1150000)}

	var (
		testdb    = ethdb.NewMemDatabase()
		gspec     = &core.Genesis{Config: config}
		genesis   = gspec.MustCommit(testdb)
		blocks, _ = core.GenerateChain(config, genesis, NewFaker(), testdb, 88, nil)
	)

	headers := make([]*types.Header, len(blocks))
	for i, block := range blocks {
		headers[i] = block.Header()
	}
	// Run the header checker for blocks one-by-one, checking for both valid and invalid nonces
	chain, _ := core.NewBlockChain(testdb, nil, config, NewFaker(), vm.Config{}, nil)
	defer chain.Stop()

	for name, test := range tests {
		number := new(big.Int).Sub(test.CurrentBlocknumber, big.NewInt(1))
		diff := CalcDifficulty(chain, test.CurrentTimestamp, &types.Header{
			Number:     number,
			Time:       test.ParentTimestamp,
			Difficulty: test.ParentDifficulty,
		})
		if diff.Cmp(test.CurrentDifficulty) != 0 {
			t.Error(name, "failed. Expected", test.CurrentDifficulty, "and calculated", diff)
		}
	}
}*/

func TestCalcBaseBlockReward(t *testing.T) {
	config := *params.MainnetChainConfig
	_, reward := CalcBaseBlockReward(config.Ubqhash, big.NewInt(1))
	if reward.Cmp(big.NewInt(8e+18)) != 0 {
		t.Error("TestCalcBaseBlockReward 8 (start)", "failed. Expected", big.NewInt(8e+18), "and calculated", reward)
	}
	_, reward = CalcBaseBlockReward(config.Ubqhash, big.NewInt(358363))
	if reward.Cmp(big.NewInt(8e+18)) != 0 {
		t.Error("TestCalcBaseBlockReward 8 (end)", "failed. Expected", big.NewInt(8e+18), "and calculated", reward)
	}

	_, reward = CalcBaseBlockReward(config.Ubqhash, big.NewInt(358364))
	if reward.Cmp(big.NewInt(7e+18)) != 0 {
		t.Error("TestCalcBaseBlockReward 7 (start)", "failed. Expected", big.NewInt(7e+18), "and calculated", reward)
	}
	_, reward = CalcBaseBlockReward(config.Ubqhash, big.NewInt(716727))
	if reward.Cmp(big.NewInt(7e+18)) != 0 {
		t.Error("TestCalcBaseBlockReward 7 (end)", "failed. Expected", big.NewInt(7e+18), "and calculated", reward)
	}

	_, reward = CalcBaseBlockReward(config.Ubqhash, big.NewInt(716728))
	if reward.Cmp(big.NewInt(6e+18)) != 0 {
		t.Error("TestCalcBaseBlockReward 6 (start)", "failed. Expected", big.NewInt(6e+18), "and calculated", reward)
	}
	_, reward = CalcBaseBlockReward(config.Ubqhash, big.NewInt(1075090))
	if reward.Cmp(big.NewInt(6e+18)) != 0 {
		t.Error("TestCalcBaseBlockReward 6 (end)", "failed. Expected", big.NewInt(6e+18), "and calculated", reward)
	}

	_, reward = CalcBaseBlockReward(config.Ubqhash, big.NewInt(1075091))
	if reward.Cmp(big.NewInt(5e+18)) != 0 {
		t.Error("TestCalcBaseBlockReward 5 (start)", "failed. Expected", big.NewInt(5e+18), "and calculated", reward)
	}
	_, reward = CalcBaseBlockReward(config.Ubqhash, big.NewInt(1433454))
	if reward.Cmp(big.NewInt(5e+18)) != 0 {
		t.Error("TestCalcBaseBlockReward 5 (end)", "failed. Expected", big.NewInt(5e+18), "and calculated", reward)
	}

	_, reward = CalcBaseBlockReward(config.Ubqhash, big.NewInt(1433455))
	if reward.Cmp(big.NewInt(4e+18)) != 0 {
		t.Error("TestCalcBaseBlockReward 4 (start)", "failed. Expected", big.NewInt(4e+18), "and calculated", reward)
	}
	_, reward = CalcBaseBlockReward(config.Ubqhash, big.NewInt(1791818))
	if reward.Cmp(big.NewInt(4e+18)) != 0 {
		t.Error("TestCalcBaseBlockReward 4 (end)", "failed. Expected", big.NewInt(4e+18), "and calculated", reward)
	}

	_, reward = CalcBaseBlockReward(config.Ubqhash, big.NewInt(1791819))
	if reward.Cmp(big.NewInt(3e+18)) != 0 {
		t.Error("TestCalcBaseBlockReward 3 (start)", "failed. Expected", big.NewInt(3e+18), "and calculated", reward)
	}
	_, reward = CalcBaseBlockReward(config.Ubqhash, big.NewInt(2150181))
	if reward.Cmp(big.NewInt(3e+18)) != 0 {
		t.Error("TestCalcBaseBlockReward 3 (end)", "failed. Expected", big.NewInt(3e+18), "and calculated", reward)
	}

	_, reward = CalcBaseBlockReward(config.Ubqhash, big.NewInt(2150182))
	if reward.Cmp(big.NewInt(2e+18)) != 0 {
		t.Error("TestCalcBaseBlockReward 2 (start)", "failed. Expected", big.NewInt(2e+18), "and calculated", reward)
	}
	_, reward = CalcBaseBlockReward(config.Ubqhash, big.NewInt(2508545))
	if reward.Cmp(big.NewInt(2e+18)) != 0 {
		t.Error("TestCalcBaseBlockReward 2 (end)", "failed. Expected", big.NewInt(2e+18), "and calculated", reward)
	}

	_, reward = CalcBaseBlockReward(config.Ubqhash, big.NewInt(2508546))
	if reward.Cmp(big.NewInt(1e+18)) != 0 {
		t.Error("TestCalcBaseBlockReward 1 (start)", "failed. Expected", big.NewInt(1e+18), "and calculated", reward)
	}
}

func TestCalcUncleBlockReward(t *testing.T) {
	config := params.MainnetChainConfig
	reward := big.NewInt(8e+18)
	// depth 1
	u := CalcUncleBlockReward(config, big.NewInt(5), big.NewInt(4), reward)
	if u.Cmp(big.NewInt(4e+18)) != 0 {
		t.Error("TestCalcUncleBlockReward 8", "failed. Expected", big.NewInt(4e+18), "and calculated", u)
	}

	// depth 2
	u = CalcUncleBlockReward(config, big.NewInt(8), big.NewInt(6), reward)
	if u.Cmp(big.NewInt(0)) != 0 {
		t.Error("TestCalcUncleBlockReward 8", "failed. Expected", big.NewInt(0), "and calculated", u)
	}

	// depth 3 (before negative fix)
	u = CalcUncleBlockReward(config, big.NewInt(8), big.NewInt(5), reward)
	if u.Cmp(big.NewInt(-4e+18)) != 0 {
		t.Error("TestCalcUncleBlockReward 8", "failed. Expected", big.NewInt(-4e+18), "and calculated", u)
	}

	// depth 3 (after negative fix)
	u = CalcUncleBlockReward(config, big.NewInt(10), big.NewInt(7), reward)
	if u.Cmp(big.NewInt(0)) != 0 {
		t.Error("TestCalcUncleBlockReward 8", "failed. Expected", big.NewInt(0), "and calculated", u)
	}

	_, reward = CalcBaseBlockReward(config.Ubqhash, big.NewInt(358364))
	expected := big.NewInt(35e+17)
	// depth 1 (after stepdown)
	u = CalcUncleBlockReward(config, big.NewInt(8), big.NewInt(7), reward)
	if u.Cmp(expected) != 0 {
		t.Error("TestCalcUncleBlockReward 7", "failed. Expected", expected, "and calculated", u)
	}

	_, reward = CalcBaseBlockReward(config.Ubqhash, big.NewInt(1075091))
	expected = big.NewInt(25e+17)
	// depth 1 (after stepdown)
	u = CalcUncleBlockReward(config, big.NewInt(8), big.NewInt(7), reward)
	if u.Cmp(expected) != 0 {
		t.Error("TestCalcUncleBlockReward 5", "failed. Expected", expected, "and calculated", u)
	}

}
