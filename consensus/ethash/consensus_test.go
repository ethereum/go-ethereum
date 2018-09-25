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

package ethash

import (
	"encoding/json"
	"math/big"
	"os"
	"path/filepath"
	"testing"

	"github.com/ethereum/go-ethereum/common/math"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/params"
)

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

	for name, test := range tests {
		number := new(big.Int).Sub(test.CurrentBlocknumber, big.NewInt(1))
		diff := CalcDifficulty(config, test.CurrentTimestamp, &types.Header{
			Number:     number,
			Time:       test.ParentTimestamp,
			Difficulty: test.ParentDifficulty,
		})
		if diff.Cmp(test.CurrentDifficulty) != 0 {
			t.Error(name, "failed. Expected", test.CurrentDifficulty, "and calculated", diff)
		}
	}
}

//func TestTransitionToProgpow(t *testing.T) {
//	fn := filepath.Join("..", "..", "tests", "hashi_to_pp_at_5.rlp.gz")
//	fh, err := os.Open(fn)
//	if err != nil {
//		t.Skip(err)
//	}
//	defer fh.Close()
//
//	var reader io.Reader = fh
//	if strings.HasSuffix(fn, ".gz") {
//		if reader, err = gzip.NewReader(reader); err != nil {
//			t.Skip(err)
//		}
//	}
//	stream := rlp.NewStream(reader, 0)
//	config := &params.ChainConfig{
//		HomesteadBlock: big.NewInt(1),
//		EIP150Block:    big.NewInt(2),
//		EIP155Block:    big.NewInt(3),
//		EIP158Block:    big.NewInt(3),
//		ProgpowBlock:   big.NewInt(5),
//	}
//	genesis := core.Genesis{Config: config,
//		GasLimit:  0x47b760,
//		Alloc:     core.GenesisAlloc{},
//		Timestamp: 0x59a4e76d,
//		ExtraData: hexutil.MustDecode("0x0000000000000000000000000000000000000000000000000000000000000000"),
//	}
//	db := ethdb.NewMemDatabase()
//	genesis.MustCommit(db)
//
//	engine := New(Config{
//		CacheDir:           "",
//		CachesInMem:        1,
//		CachesOnDisk:       1,
//		DatasetDir:         "",
//		DatasetsInMem:      1,
//		DatasetsOnDisk:     1,
//		ProgpowBlockNumber: config.ProgpowBlock,
//	}, nil, false)
//	bc, err := core.NewBlockChain(db, nil, config, engine, vm.Config{}, nil)
//	//fmt.Printf("Genesis hash %x\n", bc.Genesis().Hash())
//	if err != nil {
//		t.Skip(err)
//	}
//	blocks := make(types.Blocks, 100)
//	n := 0
//	for batch := 0; ; batch++ {
//		// Load a batch of RLP blocks.
//		i := 0
//		for ; i < 100; i++ {
//			var b types.Block
//			if err := stream.Decode(&b); err == io.EOF {
//				break
//			} else if err != nil {
//				t.Errorf("at block %d: %v", n, err)
//			}
//			// don't import first block
//			if b.NumberU64() == 0 {
//				i--
//				continue
//			}
//			blocks[i] = &b
//			n++
//		}
//		if i == 0 {
//			break
//		}
//		if _, err := bc.InsertChain(blocks[:i]); err != nil {
//			t.Fatalf("invalid block %d: %v", n, err)
//		}
//	}
//	if bc.CurrentBlock().Number().Cmp(big.NewInt(1054)) != 0 {
//		t.Errorf("Expected to import 1054 blocks, got %v", bc.CurrentBlock().Number())
//
//	}
//}
