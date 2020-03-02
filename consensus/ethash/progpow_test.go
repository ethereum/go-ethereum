// Copyright 2018 The go-ethereum Authors
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

package ethash

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"strconv"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"golang.org/x/crypto/sha3"
)

func TestRandomMerge(t *testing.T) {

	type test struct {
		a   uint32
		b   uint32
		exp uint32
	}
	for i, tt := range []test{
		{1000000, 101, 33000101},
		{2000000, 102, 66003366},
		{3000000, 103, 6000103},
		{4000000, 104, 2000104},
		{1000000, 0, 33000000},
		{2000000, 0, 66000000},
		{3000000, 0, 6000000},
		{4000000, 0, 2000000},
	} {
		res := tt.a
		merge(&res, tt.b, uint32(i))
		if res != tt.exp {
			t.Errorf("test %d, expected %d, got %d", i, tt.exp, res)
		}
	}

}

func TestProgpowChanges(t *testing.T) {
	headerHash := common.HexToHash("ffeeddccbbaa9988776655443322110000112233445566778899aabbccddeeff")
	nonce := uint64(0x123456789abcdef0)
	blocknum := uint64(30000)
	seed := seedHash(blocknum)
	fmt.Printf("seedHash %x\n", seed)
	//seed =  common.FromHex("ee304846ddd0a47b")
	expCdag0_to_15 := []uint32{
		0xb3e35467, 0xae7402e3, 0x8522a782, 0xa2d8353b,
		0xff4723bd, 0xbfbc05ee, 0xde6944de, 0xf0d2b5b8,
		0xc74cbad3, 0xb100f797, 0x05bc60be, 0x4f40840b,
		0x35e47268, 0x9cd6f993, 0x6a0e4659, 0xb838e46e,
	}
	expCdag4080_to_4095 := []uint32{
		0xbde0c650, 0x57cba482, 0x54877c9d, 0xf9fdc423,
		0xfb65141b, 0x55074ca4, 0xc7dd116e, 0xbc1737d1,
		0x126e8847, 0xb16983b2, 0xf80c058e, 0xe0ad53b5,
		0xd5f3e840, 0xff1bdd89, 0x35660a19, 0x73244193,
	}
	epoch := blocknum / epochLength
	size := cacheSize(blocknum)
	cache := make([]uint32, size/4)
	generateCache(cache, epoch, seed)
	cDag := make([]uint32, progpowCacheWords)
	generateCDag(cDag, cache, epoch)

	for i := 0; i < 15; i++ {
		if exp := expCdag0_to_15[i]; exp != cDag[i] {
			t.Errorf("test %d, exp %x != %x", i, exp, cDag[i])

		}
		if exp := expCdag4080_to_4095[i]; exp != cDag[4080+i] {
			t.Errorf("test %d (+4080), exp %x != %x", i, exp, cDag[4080+i])
		}
	}
	mixHash, finalHash, _ := hashForBlock(blocknum, nonce, headerHash)
	fmt.Printf("mixHash %x\n", mixHash)
	fmt.Printf("finalHash %x\n", finalHash)
	expMix := common.FromHex("6018c151b0f9895ebe44a4ca6ce2829e5ba6ae1a68a4ccd05a67ac01219655c1")
	expHash := common.FromHex("34d8436444aa5c61761ce0bcce0f11401df2eace77f5c14ba7039b86b5800c08")
	if !bytes.Equal(expMix, mixHash) {
		t.Errorf("mixhash err, expected %x, got %x", expMix, mixHash)
	}
	if !bytes.Equal(expHash, finalHash) {
		t.Errorf("finhash err, expected %x, got %x", expHash, finalHash)
	}
	//digest: 7d9a5f6b1407796497f16b091e5dcbbcd711d025634b505fae496611c0d6f57d
	//result (top 64 bits): 6cf196600abd663e
}

func TestCDag(t *testing.T) {
	size := cacheSize(0)
	cache := make([]uint32, size/4)
	seed := seedHash(0)
	generateCache(cache, 0, seed)
	cDag := make([]uint32, progpowCacheWords)
	generateCDag(cDag, cache, 0)
	//fmt.Printf("Cdag: %d \n", cDag[:20])
	expect := []uint32{690150178, 1181503948, 2248155602, 2118233073, 2193871115,
		1791778428, 1067701239, 724807309, 530799275, 3480325829, 3899029234,
		1998124059, 2541974622, 1100859971, 1297211151, 3268320000, 2217813733,
		2690422980, 3172863319, 2651064309}
	for i, v := range cDag[:20] {
		if expect[i] != v {
			t.Errorf("cdag err, index %d, expected %d, got %d", i, expect[i], v)
		}
	}
}

func TestRandomMath(t *testing.T) {

	type test struct {
		a   uint32
		b   uint32
		exp uint32
	}
	for i, tt := range []test{
		{20, 22, 42},
		{70000, 80000, 1305032704},
		{70000, 80000, 1},
		{1, 2, 1},
		{3, 10000, 196608},
		{3, 0, 3},
		{3, 6, 2},
		{3, 6, 7},
		{3, 6, 5},
		{0, 0xffffffff, 32},
		{3 << 13, 1 << 5, 3},
		{22, 20, 42},
		{80000, 70000, 1305032704},
		{80000, 70000, 1},
		{2, 1, 1},
		{10000, 3, 80000},
		{0, 3, 0},
		{6, 3, 2},
		{6, 3, 7},
		{6, 3, 5},
		{0, 0xffffffff, 32},
		{3 << 13, 1 << 5, 3},
	} {
		res := progpowMath(tt.a, tt.b, uint32(i))
		if res != tt.exp {
			t.Errorf("test %d, expected %d, got %d", i, tt.exp, res)
		}
	}
}

func TestProgpowKeccak256(t *testing.T) {
	result := make([]uint32, 8)
	header := make([]byte, 32)
	hash := keccakF800Long(header, 0, result)
	exp := "5dd431e5fbc604f499bfa0232f45f8f142d0ff5178f539e5a7800bf0643697af"
	if !bytes.Equal(hash, common.FromHex(exp)) {
		t.Errorf("expected %s, got %x", exp, hash)
	}
}
func TestProgpowKeccak64(t *testing.T) {
	result := make([]uint32, 8)
	header := make([]byte, 32)
	hash := keccakF800Short(header, 0, result)
	exp := uint64(0x5dd431e5fbc604f4)
	if exp != hash {
		t.Errorf("expected %x, got %x", exp, hash)
	}
}

func hashForBlock(blocknum uint64, nonce uint64, headerHash common.Hash) ([]byte, []byte, error) {
	return speedyHashForBlock(&periodContext{}, blocknum, nonce, headerHash)
}

type periodContext struct {
	cDag        []uint32
	cache       []uint32
	datasetSize uint64
	blockNum    uint64
}

// speedyHashForBlock reuses the context, if possible
func speedyHashForBlock(ctx *periodContext, blocknum uint64, nonce uint64, headerHash common.Hash) ([]byte, []byte, error) {
	if blocknum == 0 || ctx.blockNum/epochLength != blocknum/epochLength {
		size := cacheSize(blocknum)
		cache := make([]uint32, size/4)
		seed := seedHash(blocknum)
		epoch := blocknum / epochLength
		generateCache(cache, epoch, seed)
		cDag := make([]uint32, progpowCacheWords)
		generateCDag(cDag, cache, epoch)
		ctx.cache = cache
		ctx.cDag = cDag
		ctx.datasetSize = datasetSize(blocknum)
		ctx.blockNum = blocknum

	}
	keccak512 := makeHasher(sha3.NewLegacyKeccak512())
	lookup := func(index uint32) []byte {
		x := generateDatasetItem(ctx.cache, index/16, keccak512)
		//fmt.Printf("lookup(%d) : %x\n", index/16, x)
		return x
	}
	mixhash, final := progpow(headerHash.Bytes(), nonce, ctx.datasetSize, blocknum, ctx.cDag, lookup)
	return mixhash, final, nil
}

func TestProgpowHash(t *testing.T) {
	mixHash, finalHash, _ := hashForBlock(0, 0, common.Hash{})
	expHash := common.FromHex("b3bad9ca6f7c566cf0377d1f8cce29d6516a96562c122d924626281ec948ef02")
	expMix := common.FromHex("f4ac202715ded4136e72887c39e63a4738331c57fd9eb79f6ec421c281aa8743")
	if !bytes.Equal(mixHash, expMix) {
		t.Errorf("mixhash err, got %x expected %x", mixHash, expMix)
	}
	if !bytes.Equal(finalHash, expHash) {
		t.Errorf("sealhash err, got %x expected %x", finalHash, expHash)
	}
}

type progpowHashTestcase struct {
	blockNum   int
	headerHash string
	nonce      string
	mixHash    string
	finalHash  string
}

func (n *progpowHashTestcase) UnmarshalJSON(buf []byte) error {
	tmp := []interface{}{&n.blockNum, &n.headerHash, &n.nonce, &n.mixHash, &n.finalHash}
	wantLen := len(tmp)
	if err := json.Unmarshal(buf, &tmp); err != nil {
		return err
	}
	if g, e := len(tmp), wantLen; g != e {
		return fmt.Errorf("wrong number of fields in testcase: %d != %d", g, e)
	}
	return nil
}
func TestProgpowHashes(t *testing.T) {
	data, err := ioutil.ReadFile(filepath.Join(".", "testdata", "progpow_testvectors.json"))
	if err != nil {
		t.Fatal(err)
	}
	var tests []progpowHashTestcase
	if err = json.Unmarshal(data, &tests); err != nil {
		t.Fatal(err)
	}
	var ctx periodContext
	for i, tt := range tests {
		nonce, err := strconv.ParseInt(tt.nonce, 16, 64)
		if err != nil {
			t.Errorf("test %d, nonce err: %v", i, err)
		}
		mixhash, final, err := speedyHashForBlock(&ctx,
			uint64(tt.blockNum),
			uint64(nonce),
			common.BytesToHash(common.FromHex(tt.headerHash)))
		if err != nil {
			t.Errorf("test %d, err: %v", i, err)
		}
		expectFinalHash := common.FromHex(tt.finalHash)
		expectMixHash := common.FromHex(tt.mixHash)
		if !bytes.Equal(final, expectFinalHash) {
			t.Errorf("test %d (blocknum %d), sealhash err, got %x expected %x", i, tt.blockNum, final, expectFinalHash)
		}
		if !bytes.Equal(mixhash, expectMixHash) {
			t.Fatalf("test %d (blocknum %d), mixhash err, got %x expected %x", i, tt.blockNum, mixhash, expectMixHash)
		}
		//fmt.Printf("test %d ok!\n", i)
	}
}
