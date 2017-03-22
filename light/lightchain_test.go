// Copyright 2016 The go-ethereum Authors
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

package light

import (
	"context"
	"fmt"
	"math/big"
	"runtime"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/event"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/pow"
	"github.com/hashicorp/golang-lru"
)

// So we can deterministically seed different blockchains
var (
	canonicalSeed = 1
	forkSeed      = 2
)

// makeHeaderChain creates a deterministic chain of headers rooted at parent.
func makeHeaderChain(parent *types.Header, n int, db ethdb.Database, seed int) []*types.Header {
	blocks, _ := core.GenerateChain(params.TestChainConfig, types.NewBlockWithHeader(parent), db, n, func(i int, b *core.BlockGen) {
		b.SetCoinbase(common.Address{0: byte(seed), 19: byte(i)})
	})
	headers := make([]*types.Header, len(blocks))
	for i, block := range blocks {
		headers[i] = block.Header()
	}
	return headers
}

func testChainConfig() *params.ChainConfig {
	return &params.ChainConfig{HomesteadBlock: big.NewInt(0)}
}

// newCanonical creates a chain database, and injects a deterministic canonical
// chain. Depending on the full flag, if creates either a full block chain or a
// header only chain.
func newCanonical(n int) (ethdb.Database, *LightChain, error) {
	// Create te new chain database
	db, _ := ethdb.NewMemDatabase()
	evmux := &event.TypeMux{}

	// Initialize a fresh chain with only a genesis block
	genesis, _ := core.WriteTestNetGenesisBlock(db)

	blockchain, _ := NewLightChain(&dummyOdr{db: db}, testChainConfig(), pow.FakePow{}, evmux)
	// Create and inject the requested chain
	if n == 0 {
		return db, blockchain, nil
	}
	// Header-only chain requested
	headers := makeHeaderChain(genesis.Header(), n, db, canonicalSeed)
	_, err := blockchain.InsertHeaderChain(headers, 1)
	return db, blockchain, err
}

func init() {
	runtime.GOMAXPROCS(runtime.NumCPU())
}

func theLightChain(db ethdb.Database, t *testing.T) *LightChain {
	var eventMux event.TypeMux
	core.WriteTestNetGenesisBlock(db)
	LightChain, err := NewLightChain(&dummyOdr{db: db}, testChainConfig(), pow.NewTestEthash(), &eventMux)
	if err != nil {
		t.Error("failed creating LightChain:", err)
		t.FailNow()
		return nil
	}

	return LightChain
}

// Test fork of length N starting from block i
func testFork(t *testing.T, LightChain *LightChain, i, n int, comparator func(td1, td2 *big.Int)) {
	// Copy old chain up to #i into a new db
	db, LightChain2, err := newCanonical(i)
	if err != nil {
		t.Fatal("could not make new canonical in testFork", err)
	}
	// Assert the chains have the same header/block at #i
	var hash1, hash2 common.Hash
	hash1 = LightChain.GetHeaderByNumber(uint64(i)).Hash()
	hash2 = LightChain2.GetHeaderByNumber(uint64(i)).Hash()
	if hash1 != hash2 {
		t.Errorf("chain content mismatch at %d: have hash %v, want hash %v", i, hash2, hash1)
	}
	// Extend the newly created chain
	var (
		headerChainB []*types.Header
	)
	headerChainB = makeHeaderChain(LightChain2.CurrentHeader(), n, db, forkSeed)
	if _, err := LightChain2.InsertHeaderChain(headerChainB, 1); err != nil {
		t.Fatalf("failed to insert forking chain: %v", err)
	}
	// Sanity check that the forked chain can be imported into the original
	var tdPre, tdPost *big.Int

	tdPre = LightChain.GetTdByHash(LightChain.CurrentHeader().Hash())
	if err := testHeaderChainImport(headerChainB, LightChain); err != nil {
		t.Fatalf("failed to import forked header chain: %v", err)
	}
	tdPost = LightChain.GetTdByHash(headerChainB[len(headerChainB)-1].Hash())
	// Compare the total difficulties of the chains
	comparator(tdPre, tdPost)
}

func printChain(bc *LightChain) {
	for i := bc.CurrentHeader().Number.Uint64(); i > 0; i-- {
		b := bc.GetHeaderByNumber(uint64(i))
		fmt.Printf("\t%x %v\n", b.Hash(), b.Difficulty)
	}
}

// testHeaderChainImport tries to process a chain of header, writing them into
// the database if successful.
func testHeaderChainImport(chain []*types.Header, LightChain *LightChain) error {
	for _, header := range chain {
		// Try and validate the header
		if err := LightChain.Validator().ValidateHeader(header, LightChain.GetHeaderByHash(header.ParentHash), false); err != nil {
			return err
		}
		// Manually insert the header into the database, but don't reorganize (allows subsequent testing)
		LightChain.mu.Lock()
		core.WriteTd(LightChain.chainDb, header.Hash(), header.Number.Uint64(), new(big.Int).Add(header.Difficulty, LightChain.GetTdByHash(header.ParentHash)))
		core.WriteHeader(LightChain.chainDb, header)
		LightChain.mu.Unlock()
	}
	return nil
}

// Tests that given a starting canonical chain of a given size, it can be extended
// with various length chains.
func TestExtendCanonicalHeaders(t *testing.T) {
	length := 5

	// Make first chain starting from genesis
	_, processor, err := newCanonical(length)
	if err != nil {
		t.Fatalf("failed to make new canonical chain: %v", err)
	}
	// Define the difficulty comparator
	better := func(td1, td2 *big.Int) {
		if td2.Cmp(td1) <= 0 {
			t.Errorf("total difficulty mismatch: have %v, expected more than %v", td2, td1)
		}
	}
	// Start fork from current height
	testFork(t, processor, length, 1, better)
	testFork(t, processor, length, 2, better)
	testFork(t, processor, length, 5, better)
	testFork(t, processor, length, 10, better)
}

// Tests that given a starting canonical chain of a given size, creating shorter
// forks do not take canonical ownership.
func TestShorterForkHeaders(t *testing.T) {
	length := 10

	// Make first chain starting from genesis
	_, processor, err := newCanonical(length)
	if err != nil {
		t.Fatalf("failed to make new canonical chain: %v", err)
	}
	// Define the difficulty comparator
	worse := func(td1, td2 *big.Int) {
		if td2.Cmp(td1) >= 0 {
			t.Errorf("total difficulty mismatch: have %v, expected less than %v", td2, td1)
		}
	}
	// Sum of numbers must be less than `length` for this to be a shorter fork
	testFork(t, processor, 0, 3, worse)
	testFork(t, processor, 0, 7, worse)
	testFork(t, processor, 1, 1, worse)
	testFork(t, processor, 1, 7, worse)
	testFork(t, processor, 5, 3, worse)
	testFork(t, processor, 5, 4, worse)
}

// Tests that given a starting canonical chain of a given size, creating longer
// forks do take canonical ownership.
func TestLongerForkHeaders(t *testing.T) {
	length := 10

	// Make first chain starting from genesis
	_, processor, err := newCanonical(length)
	if err != nil {
		t.Fatalf("failed to make new canonical chain: %v", err)
	}
	// Define the difficulty comparator
	better := func(td1, td2 *big.Int) {
		if td2.Cmp(td1) <= 0 {
			t.Errorf("total difficulty mismatch: have %v, expected more than %v", td2, td1)
		}
	}
	// Sum of numbers must be greater than `length` for this to be a longer fork
	testFork(t, processor, 0, 11, better)
	testFork(t, processor, 0, 15, better)
	testFork(t, processor, 1, 10, better)
	testFork(t, processor, 1, 12, better)
	testFork(t, processor, 5, 6, better)
	testFork(t, processor, 5, 8, better)
}

// Tests that given a starting canonical chain of a given size, creating equal
// forks do take canonical ownership.
func TestEqualForkHeaders(t *testing.T) {
	length := 10

	// Make first chain starting from genesis
	_, processor, err := newCanonical(length)
	if err != nil {
		t.Fatalf("failed to make new canonical chain: %v", err)
	}
	// Define the difficulty comparator
	equal := func(td1, td2 *big.Int) {
		if td2.Cmp(td1) != 0 {
			t.Errorf("total difficulty mismatch: have %v, want %v", td2, td1)
		}
	}
	// Sum of numbers must be equal to `length` for this to be an equal fork
	testFork(t, processor, 0, 10, equal)
	testFork(t, processor, 1, 9, equal)
	testFork(t, processor, 2, 8, equal)
	testFork(t, processor, 5, 5, equal)
	testFork(t, processor, 6, 4, equal)
	testFork(t, processor, 9, 1, equal)
}

// Tests that chains missing links do not get accepted by the processor.
func TestBrokenHeaderChain(t *testing.T) {
	// Make chain starting from genesis
	db, LightChain, err := newCanonical(10)
	if err != nil {
		t.Fatalf("failed to make new canonical chain: %v", err)
	}
	// Create a forked chain, and try to insert with a missing link
	chain := makeHeaderChain(LightChain.CurrentHeader(), 5, db, forkSeed)[1:]
	if err := testHeaderChainImport(chain, LightChain); err == nil {
		t.Errorf("broken header chain not reported")
	}
}

type bproc struct{}

func (bproc) ValidateHeader(*types.Header, *types.Header, bool) error { return nil }

func makeHeaderChainWithDiff(genesis *types.Block, d []int, seed byte) []*types.Header {
	var chain []*types.Header
	for i, difficulty := range d {
		header := &types.Header{
			Coinbase:    common.Address{seed},
			Number:      big.NewInt(int64(i + 1)),
			Difficulty:  big.NewInt(int64(difficulty)),
			UncleHash:   types.EmptyUncleHash,
			TxHash:      types.EmptyRootHash,
			ReceiptHash: types.EmptyRootHash,
		}
		if i == 0 {
			header.ParentHash = genesis.Hash()
		} else {
			header.ParentHash = chain[i-1].Hash()
		}
		chain = append(chain, types.CopyHeader(header))
	}
	return chain
}

type dummyOdr struct {
	OdrBackend
	db ethdb.Database
}

func (odr *dummyOdr) Database() ethdb.Database {
	return odr.db
}

func (odr *dummyOdr) Retrieve(ctx context.Context, req OdrRequest) error {
	return nil
}

func chm(genesis *types.Block, db ethdb.Database) *LightChain {
	odr := &dummyOdr{db: db}
	var eventMux event.TypeMux
	bc := &LightChain{odr: odr, chainDb: db, genesisBlock: genesis, eventMux: &eventMux, pow: pow.FakePow{}}
	bc.hc, _ = core.NewHeaderChain(db, testChainConfig(), bc.Validator, bc.getProcInterrupt)
	bc.bodyCache, _ = lru.New(100)
	bc.bodyRLPCache, _ = lru.New(100)
	bc.blockCache, _ = lru.New(100)
	bc.SetValidator(bproc{})
	bc.ResetWithGenesisBlock(genesis)

	return bc
}

// Tests that reorganizing a long difficult chain after a short easy one
// overwrites the canonical numbers and links in the database.
func TestReorgLongHeaders(t *testing.T) {
	testReorg(t, []int{1, 2, 4}, []int{1, 2, 3, 4}, 10)
}

// Tests that reorganizing a short difficult chain after a long easy one
// overwrites the canonical numbers and links in the database.
func TestReorgShortHeaders(t *testing.T) {
	testReorg(t, []int{1, 2, 3, 4}, []int{1, 10}, 11)
}

func testReorg(t *testing.T, first, second []int, td int64) {
	// Create a pristine block chain
	db, _ := ethdb.NewMemDatabase()
	genesis, _ := core.WriteTestNetGenesisBlock(db)
	bc := chm(genesis, db)

	// Insert an easy and a difficult chain afterwards
	bc.InsertHeaderChain(makeHeaderChainWithDiff(genesis, first, 11), 1)
	bc.InsertHeaderChain(makeHeaderChainWithDiff(genesis, second, 22), 1)
	// Check that the chain is valid number and link wise
	prev := bc.CurrentHeader()
	for header := bc.GetHeaderByNumber(bc.CurrentHeader().Number.Uint64() - 1); header.Number.Uint64() != 0; prev, header = header, bc.GetHeaderByNumber(header.Number.Uint64()-1) {
		if prev.ParentHash != header.Hash() {
			t.Errorf("parent header hash mismatch: have %x, want %x", prev.ParentHash, header.Hash())
		}
	}
	// Make sure the chain total difficulty is the correct one
	want := new(big.Int).Add(genesis.Difficulty(), big.NewInt(td))
	if have := bc.GetTdByHash(bc.CurrentHeader().Hash()); have.Cmp(want) != 0 {
		t.Errorf("total difficulty mismatch: have %v, want %v", have, want)
	}
}

// Tests that the insertion functions detect banned hashes.
func TestBadHeaderHashes(t *testing.T) {
	// Create a pristine block chain
	db, _ := ethdb.NewMemDatabase()
	genesis, _ := core.WriteTestNetGenesisBlock(db)
	bc := chm(genesis, db)

	// Create a chain, ban a hash and try to import
	var err error
	headers := makeHeaderChainWithDiff(genesis, []int{1, 2, 4}, 10)
	core.BadHashes[headers[2].Hash()] = true
	_, err = bc.InsertHeaderChain(headers, 1)
	if !core.IsBadHashError(err) {
		t.Errorf("error mismatch: want: BadHashError, have: %v", err)
	}
}

// Tests that bad hashes are detected on boot, and the chan rolled back to a
// good state prior to the bad hash.
func TestReorgBadHeaderHashes(t *testing.T) {
	// Create a pristine block chain
	db, _ := ethdb.NewMemDatabase()
	genesis, _ := core.WriteTestNetGenesisBlock(db)
	bc := chm(genesis, db)

	// Create a chain, import and ban aferwards
	headers := makeHeaderChainWithDiff(genesis, []int{1, 2, 3, 4}, 10)

	if _, err := bc.InsertHeaderChain(headers, 1); err != nil {
		t.Fatalf("failed to import headers: %v", err)
	}
	if bc.CurrentHeader().Hash() != headers[3].Hash() {
		t.Errorf("last header hash mismatch: have: %x, want %x", bc.CurrentHeader().Hash(), headers[3].Hash())
	}
	core.BadHashes[headers[3].Hash()] = true
	defer func() { delete(core.BadHashes, headers[3].Hash()) }()
	// Create a new chain manager and check it rolled back the state
	ncm, err := NewLightChain(&dummyOdr{db: db}, testChainConfig(), pow.FakePow{}, new(event.TypeMux))
	if err != nil {
		t.Fatalf("failed to create new chain manager: %v", err)
	}
	if ncm.CurrentHeader().Hash() != headers[2].Hash() {
		t.Errorf("last header hash mismatch: have: %x, want %x", ncm.CurrentHeader().Hash(), headers[2].Hash())
	}
}
