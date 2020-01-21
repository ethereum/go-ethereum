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
	"bytes"
	"fmt"
	"io/ioutil"
	"math/big"
	"math/rand"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
)

// Tests that ethash works correctly in test mode.
func TestTestMode(t *testing.T) {
	header := &types.Header{Number: big.NewInt(1), Difficulty: big.NewInt(100)}

	ethash := NewTester(nil, false)
	defer ethash.Close()

	results := make(chan *types.Block)
	err := ethash.Seal(nil, types.NewBlockWithHeader(header), results, nil)
	if err != nil {
		t.Fatalf("failed to seal block: %v", err)
	}
	select {
	case block := <-results:
		header.Nonce = types.EncodeNonce(block.Nonce())
		header.MixDigest = block.MixDigest()
		if err := ethash.VerifySeal(nil, header); err != nil {
			t.Fatalf("unexpected verification error: %v", err)
		}
	case <-time.NewTimer(2 * time.Second).C:
		t.Error("sealing result timeout")
	}
}

// This test checks that cache lru logic doesn't crash under load.
// It reproduces https://github.com/ethereum/go-ethereum/issues/14943
func TestCacheFileEvict(t *testing.T) {
	tmpdir, err := ioutil.TempDir("", "ethash-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpdir)
	e := New(Config{CachesInMem: 3, CachesOnDisk: 10, CacheDir: tmpdir, PowMode: ModeTest}, nil, false)
	defer e.Close()

	workers := 8
	epochs := 100
	var wg sync.WaitGroup
	wg.Add(workers)
	for i := 0; i < workers; i++ {
		go verifyTest(&wg, e, i, epochs)
	}
	wg.Wait()
}

func verifyTest(wg *sync.WaitGroup, e *Ethash, workerIndex, epochs int) {
	defer wg.Done()

	const wiggle = 4 * epochLength
	r := rand.New(rand.NewSource(int64(workerIndex)))
	for epoch := 0; epoch < epochs; epoch++ {
		block := int64(epoch)*epochLength - wiggle/2 + r.Int63n(wiggle)
		if block < 0 {
			block = 0
		}
		header := &types.Header{Number: big.NewInt(block), Difficulty: big.NewInt(100)}
		e.VerifySeal(nil, header)
	}
}

func TestRemoteSealer(t *testing.T) {
	ethash := NewTester(nil, false)
	defer ethash.Close()

	api := &API{ethash}
	if _, err := api.GetWork(); err != errNoMiningWork {
		t.Error("expect to return an error indicate there is no mining work")
	}
	header := &types.Header{Number: big.NewInt(1), Difficulty: big.NewInt(100)}
	block := types.NewBlockWithHeader(header)
	sealhash := ethash.SealHash(header)

	// Push new work.
	results := make(chan *types.Block)
	ethash.Seal(nil, block, results, nil)

	var (
		work [4]string
		err  error
	)
	if work, err = api.GetWork(); err != nil || work[0] != sealhash.Hex() {
		t.Error("expect to return a mining work has same hash")
	}

	if res := api.SubmitWork(types.BlockNonce{}, sealhash, common.Hash{}); res {
		t.Error("expect to return false when submit a fake solution")
	}
	// Push new block with same block number to replace the original one.
	header = &types.Header{Number: big.NewInt(1), Difficulty: big.NewInt(1000)}
	block = types.NewBlockWithHeader(header)
	sealhash = ethash.SealHash(header)
	ethash.Seal(nil, block, results, nil)

	if work, err = api.GetWork(); err != nil || work[0] != sealhash.Hex() {
		t.Error("expect to return the latest pushed work")
	}
}

func TestHashRate(t *testing.T) {
	var (
		hashrate = []hexutil.Uint64{100, 200, 300}
		expect   uint64
		ids      = []common.Hash{common.HexToHash("a"), common.HexToHash("b"), common.HexToHash("c")}
	)
	ethash := NewTester(nil, false)
	defer ethash.Close()

	if tot := ethash.Hashrate(); tot != 0 {
		t.Error("expect the result should be zero")
	}

	api := &API{ethash}
	for i := 0; i < len(hashrate); i += 1 {
		if res := api.SubmitHashRate(hashrate[i], ids[i]); !res {
			t.Error("remote miner submit hashrate failed")
		}
		expect += uint64(hashrate[i])
	}
	if tot := ethash.Hashrate(); tot != float64(expect) {
		t.Error("expect total hashrate should be same")
	}
}

func TestClosedRemoteSealer(t *testing.T) {
	ethash := NewTester(nil, false)
	time.Sleep(1 * time.Second) // ensure exit channel is listening
	ethash.Close()

	api := &API{ethash}
	if _, err := api.GetWork(); err != errEthashStopped {
		t.Error("expect to return an error to indicate ethash is stopped")
	}

	if res := api.SubmitHashRate(hexutil.Uint64(100), common.HexToHash("a")); res {
		t.Error("expect to return false when submit hashrate to a stopped ethash")
	}
}

func TestEthashVerification(t *testing.T) {

	/*
			ERROR[01-20|14:19:41.708] Invalid mix digest
		number=5901768 datasetSize=2717907328 hdr.nonce=8774555798626709531
		sealHash="[69 201 230 234 206 75 24 9 247 242 180 231 191 184 43 24 242 195 86 93 114 177 250 81 31 11 16 38 7 157 51 241]"
		result="[103 158 62 246 38 246 6 82 71 33 210 238 177 37 178 13 191 241 61 186 140 226 136 148 83 111 108 166 252 245 249 55]"
		digest="[243 53 19 163 179 107 241 143 214 217 69 240 230 214 186 30 147 91 37 133 63 119 50 135 143 2 34 91 56 44 57 133]"
		hdr.digest="[7 230 161 170 212 130 52 40 152 14 130 41 51 80 133 198 228 85 153 243 214 89 81 206 124 121 106 216 112 129 235 114]"
	*/
	engine := New(Config{
		CacheDir:       "",
		CachesInMem:    2,
		CachesOnDisk:   3,
		DatasetDir:     "",
		DatasetsInMem:  1,
		DatasetsOnDisk: 2,
	}, nil, false)

	var (
		number   = uint64(5901768)
		nonce    = uint64(8774555798626709531)
		sealHash = []byte{69, 201, 230, 234, 206, 75, 24, 9, 247, 242, 180, 231, 191, 184,
			43, 24, 242, 195, 86, 93, 114, 177, 250, 81, 31, 11, 16, 38, 7, 157, 51, 241}
	)
	cache := engine.cache(number)
	size := datasetSize(number)
	fmt.Printf("number: %d\n", number)
	fmt.Printf("datasetSize: %d\n", size)
	digest, result := hashimotoLight(size, cache.cache, sealHash, nonce)
	fmt.Printf("nonce: %v\n", nonce)
	fmt.Printf("sealHash: %x\n", sealHash)
	fmt.Printf("result: %x\n", result)
	fmt.Println()
	fmt.Printf("digest: %x\n", digest)
	expDigest, _ := hexutil.Decode("07e6a1aad4823428980e8229335085c6e45599f3d65951ce7c796ad87081eb72")
	if bytes.Equal(expDigest, digest) {
		t.Errorf("Mix digest wrong!, got %x exp %x", digest, expDigest)
	}
	sum := uint32(0)
	for _, val := range cache.cache {
		sum = sum ^ val
	}
	fmt.Printf("xor sum of cache contents: %x\n", sum)
}
