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
	"encoding/binary"
	"fmt"
	"hash/crc32"
	"io/ioutil"
	"math/big"
	"math/rand"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/assert"
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
		if err := ethash.verifySeal(nil, header, false); err != nil {
			t.Fatalf("unexpected verification error: %v", err)
		}
	case <-time.NewTimer(4 * time.Second).C:
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

	config := Config{
		CachesInMem:  3,
		CachesOnDisk: 10,
		CacheDir:     tmpdir,
		PowMode:      ModeTest,
	}
	e := New(config, nil, false)
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
		e.verifySeal(nil, header, false)
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

func TestHashrate(t *testing.T) {
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
		if res := api.SubmitHashrate(hashrate[i], ids[i]); !res {
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

	if res := api.SubmitHashrate(hexutil.Uint64(100), common.HexToHash("a")); res {
		t.Error("expect to return false when submit hashrate to a stopped ethash")
	}
}

func TestMemoryMap(t *testing.T) {
	dummyData := []uint32{9343423, 3723123, 885, 4314324, 482853252}

	tests := map[string]struct {
		storedChecksum  uint32
		storedDumpMagic []uint32
		storedData      []uint32
		expectedErr     error
	}{
		"checksum mismatches": {
			storedChecksum:  123,
			storedDumpMagic: dumpMagic,
			storedData:      dummyData,
			expectedErr:     errInvalidCacheFileChecksum,
		},
		"invalid dumpMagic": {
			storedChecksum:  crc32.ChecksumIEEE(uintsToBytes([]uint32{99, 98}, dummyData)),
			storedDumpMagic: []uint32{99, 98},
			storedData:      dummyData,
			expectedErr:     ErrInvalidDumpMagic,
		},
		"valid checksum and dumpMagic": {
			storedChecksum:  crc32.ChecksumIEEE(uintsToBytes(dumpMagic, dummyData)),
			storedDumpMagic: dumpMagic,
			storedData:      dummyData,
			expectedErr:     nil,
		},
		"malformed small file": {
			storedChecksum:  0,
			storedDumpMagic: []uint32{},
			storedData:      []uint32{},
			expectedErr:     errMalformedCacheFile,
		},
	}

	for ttName, tt := range tests {
		t.Run(ttName, func(t *testing.T) {
			cacheFile, err := ioutil.TempFile("", "ethash-test")
			if err != nil {
				t.Fatal(err)
			}
			defer os.Remove(cacheFile.Name())

			binary.Write(cacheFile, systemByteOrder, tt.storedChecksum)
			binary.Write(cacheFile, systemByteOrder, tt.storedDumpMagic)
			binary.Write(cacheFile, systemByteOrder, tt.storedData)
			cacheFile.Close()

			_, mmap, data, err := memoryMap(cacheFile.Name(), false)
			assert.Equal(t, tt.expectedErr, err)
			if tt.expectedErr == nil {
				expectedMMap := uintsToBytes(
					[]uint32{tt.storedChecksum},
					tt.storedDumpMagic,
					tt.storedData,
				)
				assert.EqualValues(t, expectedMMap, mmap)
				assert.Equal(t, tt.storedData, data)
			} else {
				assert.Nil(t, mmap)
				assert.Nil(t, data)
			}
		})
	}
}

func uintsToBytes(input ...[]uint32) []byte {
	flattenInput := make([]uint32, 0)
	for _, arr := range input {
		flattenInput = append(flattenInput, arr...)
	}
	res := make([]byte, len(flattenInput)*4)
	for i, el := range flattenInput {
		systemByteOrder.PutUint32(res[i*4:], el)
	}
	return res
}

func TestMemoryMapAndGenerate(t *testing.T) {
	t.Run("it puts checksum to the cache file itself", func(t *testing.T) {
		tmpdir, err := ioutil.TempDir("", "ethash-test")
		if err != nil {
			t.Fatal(err)
		}
		defer os.RemoveAll(tmpdir)

		rand.Seed(time.Now().UnixNano())
		fakeData := make([]uint32, 20)
		for i := range fakeData {
			fakeData[i] = rand.Uint32()
		}
		targetPath := filepath.Join(tmpdir, fmt.Sprintf("generate-test-%d", time.Now().UnixNano()))
		generator := func(buffer []uint32) {
			copy(buffer, fakeData)
		}
		dataSizeInBytes := uint64(len(fakeData) * 4)

		actualFile, mmap, data, err := memoryMapAndGenerate(targetPath, dataSizeInBytes, true, generator)
		assert.Nil(t, err)
		defer actualFile.Close()

		assert.Equal(t, fakeData, data)
		expectedChecksum := crc32.ChecksumIEEE(uintsToBytes(dumpMagic, fakeData))
		expectedMMap := uintsToBytes(
			[]uint32{expectedChecksum},
			dumpMagic,
			fakeData,
		)
		assert.EqualValues(t, expectedMMap, mmap)

		fileContent, err := ioutil.ReadAll(actualFile)
		if err != nil {
			t.Fatal(err)
		}
		assert.Equal(t, fileContent, expectedMMap)
	})
}

func TestCachedFilesChecksums(t *testing.T) {
	var endian string
	if !isLittleEndian() {
		endian = ".be"
	}
	// hardcoded test settings in ethash.go
	csize := 1024
	dsize := 32 * 1024

	t.Run("dataset ignores files with invalid checksums and generates new data", func(t *testing.T) {
		tmpdir, err := ioutil.TempDir("", "ethash-test")
		if err != nil {
			t.Fatal(err)
		}
		defer os.RemoveAll(tmpdir)

		epoch := uint64(2)
		seed := seedHash(epoch*epochLength + 1)

		expectedCache := make([]uint32, csize/4)
		generateCache(expectedCache, epoch, seed)
		expectedDataset := make([]uint32, dsize/4)
		generateDataset(expectedDataset, epoch, expectedCache)
		expectedChecksum := crc32.ChecksumIEEE(uintsToBytes(dumpMagic, expectedDataset))

		targetFilepath := filepath.Join(tmpdir, fmt.Sprintf("full-R%d-%x%s", algorithmRevision, seed[:8], endian))
		defer os.Remove(targetFilepath)
		file, err := os.Create(targetFilepath)
		if err != nil {
			t.Fatal(err)
		}

		binary.Write(file, systemByteOrder, expectedChecksum)
		binary.Write(file, systemByteOrder, dumpMagic)
		// data loss
		binary.Write(file, systemByteOrder, expectedDataset[:len(expectedDataset)-2])
		file.Close()

		testDataset := newDataset(epoch).(*dataset)
		testDataset.generate(tmpdir, 5, true, true)

		assert.Equal(t, expectedDataset, testDataset.dataset)
		expectedRawData := uintsToBytes(
			[]uint32{expectedChecksum},
			dumpMagic,
			expectedDataset,
		)
		assert.EqualValues(t, expectedRawData, testDataset.mmap)

		fileData, err := ioutil.ReadFile(targetFilepath)
		if err != nil {
			t.Fatal(err)
		}
		assert.Equal(t, expectedRawData, fileData)
	})

	t.Run("cache ignores files with invalid checksums and generates new data", func(t *testing.T) {
		tmpdir, err := ioutil.TempDir("", "ethash-test")
		if err != nil {
			t.Fatal(err)
		}
		defer os.RemoveAll(tmpdir)

		epoch := uint64(2)
		seed := seedHash(epoch*epochLength + 1)

		expectedCache := make([]uint32, csize/4)
		generateCache(expectedCache, epoch, seed)
		expectedChecksum := crc32.ChecksumIEEE(uintsToBytes(dumpMagic, expectedCache))

		targetFilepath := filepath.Join(tmpdir, fmt.Sprintf("cache-R%d-%x%s", algorithmRevision, seed[:8], endian))
		defer os.Remove(targetFilepath)
		file, err := os.Create(targetFilepath)
		if err != nil {
			t.Fatal(err)
		}

		binary.Write(file, systemByteOrder, expectedChecksum)
		binary.Write(file, systemByteOrder, dumpMagic)
		// data loss
		binary.Write(file, systemByteOrder, expectedCache[:len(expectedCache)-2])
		file.Close()

		testCache := newCache(epoch).(*cache)
		testCache.generate(tmpdir, 5, true, true)

		assert.Equal(t, expectedCache, testCache.cache)
		expectedRawData := uintsToBytes(
			[]uint32{expectedChecksum},
			dumpMagic,
			expectedCache,
		)
		assert.EqualValues(t, expectedRawData, testCache.mmap)

		fileData, err := ioutil.ReadFile(targetFilepath)
		if err != nil {
			t.Fatal(err)
		}
		assert.Equal(t, expectedRawData, fileData)
	})
}
