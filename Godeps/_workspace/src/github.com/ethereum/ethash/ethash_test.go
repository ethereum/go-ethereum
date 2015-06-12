package ethash

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"log"
	"math/big"
	"os"
	"sync"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

func init() {
	// glog.SetV(6)
	// glog.SetToStderr(true)
}

type testBlock struct {
	difficulty  *big.Int
	hashNoNonce common.Hash
	nonce       uint64
	mixDigest   common.Hash
	number      uint64
}

func (b *testBlock) Difficulty() *big.Int     { return b.difficulty }
func (b *testBlock) HashNoNonce() common.Hash { return b.hashNoNonce }
func (b *testBlock) Nonce() uint64            { return b.nonce }
func (b *testBlock) MixDigest() common.Hash   { return b.mixDigest }
func (b *testBlock) NumberU64() uint64        { return b.number }

var validBlocks = []*testBlock{
	// from proof of concept nine testnet, epoch 0
	{
		number:      22,
		hashNoNonce: common.HexToHash("372eca2454ead349c3df0ab5d00b0b706b23e49d469387db91811cee0358fc6d"),
		difficulty:  big.NewInt(132416),
		nonce:       0x495732e0ed7a801c,
		mixDigest:   common.HexToHash("2f74cdeb198af0b9abe65d22d372e22fb2d474371774a9583c1cc427a07939f5"),
	},
	// from proof of concept nine testnet, epoch 1
	{
		number:      30001,
		hashNoNonce: common.HexToHash("7e44356ee3441623bc72a683fd3708fdf75e971bbe294f33e539eedad4b92b34"),
		difficulty:  big.NewInt(1532671),
		nonce:       0x318df1c8adef7e5e,
		mixDigest:   common.HexToHash("144b180aad09ae3c81fb07be92c8e6351b5646dda80e6844ae1b697e55ddde84"),
	},
	// from proof of concept nine testnet, epoch 2
	{
		number:      60000,
		hashNoNonce: common.HexToHash("5fc898f16035bf5ac9c6d9077ae1e3d5fc1ecc3c9fd5bee8bb00e810fdacbaa0"),
		difficulty:  big.NewInt(2467358),
		nonce:       0x50377003e5d830ca,
		mixDigest:   common.HexToHash("ab546a5b73c452ae86dadd36f0ed83a6745226717d3798832d1b20b489e82063"),
	},
}

var invalidZeroDiffBlock = testBlock{
	number:      61440000,
	hashNoNonce: crypto.Sha3Hash([]byte("foo")),
	difficulty:  big.NewInt(0),
	nonce:       0xcafebabec00000fe,
	mixDigest:   crypto.Sha3Hash([]byte("bar")),
}

func TestEthashVerifyValid(t *testing.T) {
	eth := New()
	for i, block := range validBlocks {
		if !eth.Verify(block) {
			t.Errorf("block %d (%x) did not validate.", i, block.hashNoNonce[:6])
		}
	}
}

func TestEthashVerifyInvalid(t *testing.T) {
	eth := New()
	if eth.Verify(&invalidZeroDiffBlock) {
		t.Errorf("should not validate - we just ensure it does not panic on this block")
	}
}

func TestEthashConcurrentVerify(t *testing.T) {
	eth, err := NewForTesting()
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(eth.Full.Dir)

	block := &testBlock{difficulty: big.NewInt(10)}
	nonce, md := eth.Search(block, nil, 0)
	block.nonce = nonce
	block.mixDigest = common.BytesToHash(md)

	// Verify the block concurrently to check for data races.
	var wg sync.WaitGroup
	wg.Add(100)
	for i := 0; i < 100; i++ {
		go func() {
			if !eth.Verify(block) {
				t.Error("Block could not be verified")
			}
			wg.Done()
		}()
	}
	wg.Wait()
}

func TestEthashConcurrentSearch(t *testing.T) {
	eth, err := NewForTesting()
	if err != nil {
		t.Fatal(err)
	}
	eth.Turbo(true)
	defer os.RemoveAll(eth.Full.Dir)

	type searchRes struct {
		n  uint64
		md []byte
	}

	var (
		block   = &testBlock{difficulty: big.NewInt(35000)}
		nsearch = 10
		wg      = new(sync.WaitGroup)
		found   = make(chan searchRes)
		stop    = make(chan struct{})
	)
	rand.Read(block.hashNoNonce[:])
	wg.Add(nsearch)
	// launch n searches concurrently.
	for i := 0; i < nsearch; i++ {
		go func() {
			nonce, md := eth.Search(block, stop, 0)
			select {
			case found <- searchRes{n: nonce, md: md}:
			case <-stop:
			}
			wg.Done()
		}()
	}

	// wait for one of them to find the nonce
	res := <-found
	// stop the others
	close(stop)
	wg.Wait()

	block.nonce = res.n
	block.mixDigest = common.BytesToHash(res.md)
	if !eth.Verify(block) {
		t.Error("Block could not be verified")
	}
}

func TestEthashSearchAcrossEpoch(t *testing.T) {
	eth, err := NewForTesting()
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(eth.Full.Dir)

	for i := epochLength - 40; i < epochLength+40; i++ {
		block := &testBlock{number: i, difficulty: big.NewInt(90)}
		rand.Read(block.hashNoNonce[:])
		nonce, md := eth.Search(block, nil, 0)
		block.nonce = nonce
		block.mixDigest = common.BytesToHash(md)
		if !eth.Verify(block) {
			t.Fatalf("Block could not be verified")
		}
	}
}

func TestGetSeedHash(t *testing.T) {
	seed0, err := GetSeedHash(0)
	if err != nil {
		t.Errorf("Failed to get seedHash for block 0: %v", err)
	}
	if bytes.Compare(seed0, make([]byte, 32)) != 0 {
		log.Printf("seedHash for block 0 should be 0s, was: %v\n", seed0)
	}
	seed1, err := GetSeedHash(30000)
	if err != nil {
		t.Error(err)
	}

	// From python:
	// > from pyethash import get_seedhash
	// > get_seedhash(30000)
	expectedSeed1, err := hex.DecodeString("290decd9548b62a8d60345a988386fc84ba6bc95484008f6362f93160ef3e563")
	if err != nil {
		t.Error(err)
	}

	if bytes.Compare(seed1, expectedSeed1) != 0 {
		log.Printf("seedHash for block 1 should be: %v,\nactual value: %v\n", expectedSeed1, seed1)
	}

}
