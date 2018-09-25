package ethash

import (
	"bytes"
	"encoding/json"
	"fmt"
	"golang.org/x/crypto/sha3"
	"io/ioutil"
	"path/filepath"
	"strconv"
	"testing"

	"github.com/ethereum/go-ethereum/common"
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
		{3000000, 103, 2999975},
		{4000000, 104, 4000104},
		{1000000, 0, 33000000},
		{2000000, 0, 66000000},
		{3000000, 0, 3000000},
		{4000000, 0, 4000000},
	} {
		res := tt.a
		merge(&res, tt.b, uint32(i))
		if res != tt.exp {
			t.Errorf("test %d, expected %d, got %d", i, tt.exp, res)
		}
	}

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
		return generateDatasetItem(ctx.cache, index/16, keccak512)
	}
	mixhash, final := progpow(headerHash.Bytes(), nonce, ctx.datasetSize, blocknum, ctx.cDag, lookup)

	return mixhash, final, nil
}

func TestProgpowHash(t *testing.T) {
	mixHash, finalHash, _ := hashForBlock(0, 0, common.Hash{})
	expHash := common.FromHex("5391770a00140cfab1202df86ab47fb86bb299fe4386e6d593d4416b9414df92")
	expMix := common.FromHex("d46c7c0a927acead9f943bee6ed95bba40dfbe6c24b232af3e7764f6c8849d41")
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
	data, err := ioutil.ReadFile(filepath.Join("..", "..", "tests", "progpow_testvectors.json"))
	if err != nil {
		t.Fatal(err)
	}
	var tests []progpowHashTestcase
	if err = json.Unmarshal(data, &tests); err != nil {
		t.Fatal(err)
	}
	var ctx periodContext
	for i, tt := range tests {
		// Only run test 0,1,49,50,51,99,100, 101 .. etc
		if !(i+1%50 == 0 || i%50 == 0 || i-1%50 == 0) {
			continue
		}
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
