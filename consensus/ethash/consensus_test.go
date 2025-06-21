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
	crand "crypto/rand"
	"encoding/binary"
	"encoding/json"
	"math/big"
	"math/rand"
	"os"
	"path/filepath"
	"testing"

	"github.com/ethereum/go-ethereum/common"
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

func randSlice(min, max uint32) []byte {
	var b = make([]byte, 4)
	crand.Read(b)
	a := binary.LittleEndian.Uint32(b)
	size := min + a%(max-min)
	out := make([]byte, size)
	crand.Read(out)
	return out
}

func TestDifficultyCalculators(t *testing.T) {
	for i := 0; i < 5000; i++ {
		// 1 to 300 seconds diff
		var timeDelta = uint64(1 + rand.Uint32()%3000)
		diffBig := new(big.Int).SetBytes(randSlice(2, 10))
		if diffBig.Cmp(params.MinimumDifficulty) < 0 {
			diffBig.Set(params.MinimumDifficulty)
		}
		//rand.Read(difficulty)
		header := &types.Header{
			Difficulty: diffBig,
			Number:     new(big.Int).SetUint64(rand.Uint64() % 50_000_000),
			Time:       rand.Uint64() - timeDelta,
		}
		if rand.Uint32()&1 == 0 {
			header.UncleHash = types.EmptyUncleHash
		}
		bombDelay := new(big.Int).SetUint64(rand.Uint64() % 50_000_000)
		for i, pair := range []struct {
			bigFn  func(time uint64, parent *types.Header) *big.Int
			u256Fn func(time uint64, parent *types.Header) *big.Int
		}{
			{FrontierDifficultyCalculator, CalcDifficultyFrontierU256},
			{HomesteadDifficultyCalculator, CalcDifficultyHomesteadU256},
			{DynamicDifficultyCalculator(bombDelay), MakeDifficultyCalculatorU256(bombDelay)},
		} {
			time := header.Time + timeDelta
			want := pair.bigFn(time, header)
			have := pair.u256Fn(time, header)
			if want.BitLen() > 256 {
				continue
			}
			if want.Cmp(have) != 0 {
				t.Fatalf("pair %d: want %x have %x\nparent.Number: %x\np.Time: %x\nc.Time: %x\nBombdelay: %v\n", i, want, have,
					header.Number, header.Time, time, bombDelay)
			}
		}
	}
}

func BenchmarkDifficultyCalculator(b *testing.B) {
	x1 := makeDifficultyCalculator(big.NewInt(1000000))
	x2 := MakeDifficultyCalculatorU256(big.NewInt(1000000))
	h := &types.Header{
		ParentHash: common.Hash{},
		UncleHash:  types.EmptyUncleHash,
		Difficulty: big.NewInt(0xffffff),
		Number:     big.NewInt(500000),
		Time:       1000000,
	}
	b.Run("big-frontier", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			calcDifficultyFrontier(1000014, h)
		}
	})
	b.Run("u256-frontier", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			CalcDifficultyFrontierU256(1000014, h)
		}
	})
	b.Run("big-homestead", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			calcDifficultyHomestead(1000014, h)
		}
	})
	b.Run("u256-homestead", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			CalcDifficultyHomesteadU256(1000014, h)
		}
	})
	b.Run("big-generic", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			x1(1000014, h)
		}
	})
	b.Run("u256-generic", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			x2(1000014, h)
		}
	})
}

type testChainHeaderReader struct{}

func (t testChainHeaderReader) Config() *params.ChainConfig {
	return &params.ChainConfig{}
}
func (t testChainHeaderReader) CurrentHeader() *types.Header {
	return &types.Header{}
}
func (t testChainHeaderReader) GetHeader(hash common.Hash, number uint64) *types.Header {
	var rueck *types.Header = nil
	if number == 22431047 {
		rueck = &types.Header{}
		rueck.Number = new(big.Int).SetUint64(number)
		rueck.Time = 99
		rueck.Difficulty = big.NewInt(1)
		rueck.GasLimit = 5000
	}
	return rueck
}
func (t testChainHeaderReader) GetHeaderByHash(hash common.Hash) *types.Header {
	return &types.Header{}
}
func (t testChainHeaderReader) GetHeaderByNumber(number uint64) *types.Header {
	return &types.Header{}
}

func TestVerifyHeader(t *testing.T) {
	ethash := Ethash{}
	chain := testChainHeaderReader{}
	header := types.Header{}
	header.Number = big.NewInt(22431048)
	header.Time = 100
	header.Difficulty = new(big.Int)
	header.Difficulty.SetString("6739986666787659948666753771754907668409286105635143120275902693376", 10)
	header.GasLimit = 5000

	ret := ethash.VerifyHeader(chain, &header)
	if ret != nil {
		t.Errorf("VerifyHeader not successful")
	}
}

func TestVerifyHeaders(t *testing.T) {
	ethash := Ethash{}
	chain := testChainHeaderReader{}
	var liste []*types.Header
	liste = append(liste, &types.Header{})
	liste = append(liste, &types.Header{})
	liste[0].Number = big.NewInt(22431048)
	liste[0].Time = 100
	liste[0].Difficulty = new(big.Int)
	liste[0].Difficulty.SetString("6739986666787659948666753771754907668409286105635143120275902693376", 10)
	liste[0].GasLimit = 5000
	liste[1].Number = big.NewInt(22431049)
	liste[1].Time = 101
	liste[1].Difficulty = liste[0].Difficulty
	liste[1].GasLimit = 5000
	_, c2 := ethash.VerifyHeaders(chain, liste)
	r2 := <-c2
	if r2 != nil {
		t.Errorf("VerifyHeaders not successful")
	}
}

type testChainReader struct {
}

func (t testChainReader) GetBlock(hash common.Hash, number uint64) *types.Block {
	return &types.Block{}
}
func (t testChainReader) Config() *params.ChainConfig {
	return &params.ChainConfig{}
}
func (t testChainReader) CurrentHeader() *types.Header {
	return &types.Header{}
}
func (t testChainReader) GetHeader(hash common.Hash, number uint64) *types.Header {
	var rueck *types.Header = nil
	if number == 22431047 {
		rueck = &types.Header{}
		rueck.Number = new(big.Int).SetUint64(number)
		rueck.Time = 99
		rueck.Difficulty = big.NewInt(1)
		rueck.GasLimit = 5000
		rueck.ParentHash = common.Hash{4}
	} else if number == 22431046 {
		rueck = &types.Header{}
		rueck.Number = new(big.Int).SetUint64(number)
		rueck.Time = 98
		rueck.Difficulty = big.NewInt(1)
		rueck.GasLimit = 5000
	}
	return rueck
}
func (t testChainReader) GetHeaderByHash(hash common.Hash) *types.Header {
	return &types.Header{}
}
func (t testChainReader) GetHeaderByNumber(number uint64) *types.Header {
	return &types.Header{}
}

type testTrieHasher struct{}

func (t testTrieHasher) Reset()                      {}
func (t testTrieHasher) Update([]byte, []byte) error { return nil }
func (t testTrieHasher) Hash() common.Hash {
	ret := common.Hash{}
	return ret
}

func TestVerifyUncles(t *testing.T) {
	ethash := Ethash{}
	chain := testChainReader{}

	header := types.Header{}
	header.Number = big.NewInt(22431048)
	header.Time = 101
	header.Difficulty = new(big.Int)
	header.Difficulty.SetString("6739986666787659948666753771754907668409286105635143120275902693376", 10)
	header.GasLimit = 5000
	header.ParentHash = common.Hash{5}

	body := types.Body{}
	var uncles []*types.Header
	uncles = append(uncles, &types.Header{})
	uncles[0].Number = new(big.Int).SetUint64(22431047)
	uncles[0].Time = 100
	uncles[0].Difficulty = header.Difficulty
	uncles[0].GasLimit = 5000
	uncles[0].ParentHash = common.Hash{4}
	body.Uncles = uncles

	block := types.NewBlock(&header, &body, nil, testTrieHasher{})

	ret := ethash.VerifyUncles(chain, block)
	if ret != nil {
		t.Errorf("VerifyUncles not successful")
	}
}

func TestPrepare(t *testing.T) {
	ethash := Ethash{}
	chain := testChainReader{}
	header := types.Header{}
	header.Number = big.NewInt(22431048)
	header.Time = 100
	header.Difficulty = big.NewInt(131072)
	header.GasLimit = 5000
	ret := ethash.Prepare(chain, &header)
	if ret != nil {
		t.Errorf("Prepare not successful")
	}
}

func TestSealHash(t *testing.T) {
	ethash := Ethash{}
	header := types.Header{}
	hash := ethash.SealHash(&header)
	if len(hash) != 32 {
		t.Errorf("SealHash not successful")
	}
}
