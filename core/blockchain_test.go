// Copyright 2014 The go-ethereum Authors
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

package core

import (
	"fmt"
	"math/big"
	"math/rand"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"testing"
	"time"

	"github.com/ethereum/ethash"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/event"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/pow"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/hashicorp/golang-lru"
)

func init() {
	runtime.GOMAXPROCS(runtime.NumCPU())
}

func thePow() pow.PoW {
	pow, _ := ethash.NewForTesting()
	return pow
}

func theBlockChain(db ethdb.Database, t *testing.T) *BlockChain {
	var eventMux event.TypeMux
	WriteTestNetGenesisBlock(db)
	blockchain, err := NewBlockChain(db, thePow(), &eventMux)
	if err != nil {
		t.Error("failed creating blockchain:", err)
		t.FailNow()
		return nil
	}

	return blockchain
}

// Test fork of length N starting from block i
func testFork(t *testing.T, blockchain *BlockChain, i, n int, full bool, comparator func(td1, td2 *big.Int)) {
	// Copy old chain up to #i into a new db
	db, blockchain2, err := newCanonical(i, full)
	if err != nil {
		t.Fatal("could not make new canonical in testFork", err)
	}
	// Assert the chains have the same header/block at #i
	var hash1, hash2 common.Hash
	if full {
		hash1 = blockchain.GetBlockByNumber(uint64(i)).Hash()
		hash2 = blockchain2.GetBlockByNumber(uint64(i)).Hash()
	} else {
		hash1 = blockchain.GetHeaderByNumber(uint64(i)).Hash()
		hash2 = blockchain2.GetHeaderByNumber(uint64(i)).Hash()
	}
	if hash1 != hash2 {
		t.Errorf("chain content mismatch at %d: have hash %v, want hash %v", i, hash2, hash1)
	}
	// Extend the newly created chain
	var (
		blockChainB  []*types.Block
		headerChainB []*types.Header
	)
	if full {
		blockChainB = makeBlockChain(blockchain2.CurrentBlock(), n, db, forkSeed)
		if _, err := blockchain2.InsertChain(blockChainB); err != nil {
			t.Fatalf("failed to insert forking chain: %v", err)
		}
	} else {
		headerChainB = makeHeaderChain(blockchain2.CurrentHeader(), n, db, forkSeed)
		if _, err := blockchain2.InsertHeaderChain(headerChainB, 1); err != nil {
			t.Fatalf("failed to insert forking chain: %v", err)
		}
	}
	// Sanity check that the forked chain can be imported into the original
	var tdPre, tdPost *big.Int

	if full {
		tdPre = blockchain.GetTd(blockchain.CurrentBlock().Hash())
		if err := testBlockChainImport(blockChainB, blockchain); err != nil {
			t.Fatalf("failed to import forked block chain: %v", err)
		}
		tdPost = blockchain.GetTd(blockChainB[len(blockChainB)-1].Hash())
	} else {
		tdPre = blockchain.GetTd(blockchain.CurrentHeader().Hash())
		if err := testHeaderChainImport(headerChainB, blockchain); err != nil {
			t.Fatalf("failed to import forked header chain: %v", err)
		}
		tdPost = blockchain.GetTd(headerChainB[len(headerChainB)-1].Hash())
	}
	// Compare the total difficulties of the chains
	comparator(tdPre, tdPost)
}

func printChain(bc *BlockChain) {
	for i := bc.CurrentBlock().Number().Uint64(); i > 0; i-- {
		b := bc.GetBlockByNumber(uint64(i))
		fmt.Printf("\t%x %v\n", b.Hash(), b.Difficulty())
	}
}

// testBlockChainImport tries to process a chain of blocks, writing them into
// the database if successful.
func testBlockChainImport(chain types.Blocks, blockchain *BlockChain) error {
	for _, block := range chain {
		// Try and process the block
		err := blockchain.Validator().ValidateBlock(block)
		if err != nil {
			if IsKnownBlockErr(err) {
				continue
			}
			return err
		}
		statedb, err := state.New(blockchain.GetBlock(block.ParentHash()).Root(), blockchain.chainDb)
		if err != nil {
			return err
		}
		receipts, _, usedGas, err := blockchain.Processor().Process(block, statedb)
		if err != nil {
			reportBlock(block, err)
			return err
		}
		err = blockchain.Validator().ValidateState(block, blockchain.GetBlock(block.ParentHash()), statedb, receipts, usedGas)
		if err != nil {
			reportBlock(block, err)
			return err
		}
		blockchain.mu.Lock()
		WriteTd(blockchain.chainDb, block.Hash(), new(big.Int).Add(block.Difficulty(), blockchain.GetTd(block.ParentHash())))
		WriteBlock(blockchain.chainDb, block)
		statedb.Commit()
		blockchain.mu.Unlock()
	}
	return nil
}

// testHeaderChainImport tries to process a chain of header, writing them into
// the database if successful.
func testHeaderChainImport(chain []*types.Header, blockchain *BlockChain) error {
	for _, header := range chain {
		// Try and validate the header
		if err := blockchain.Validator().ValidateHeader(header, blockchain.GetHeader(header.ParentHash), false); err != nil {
			return err
		}
		// Manually insert the header into the database, but don't reorganise (allows subsequent testing)
		blockchain.mu.Lock()
		WriteTd(blockchain.chainDb, header.Hash(), new(big.Int).Add(header.Difficulty, blockchain.GetTd(header.ParentHash)))
		WriteHeader(blockchain.chainDb, header)
		blockchain.mu.Unlock()
	}
	return nil
}

func loadChain(fn string, t *testing.T) (types.Blocks, error) {
	fh, err := os.OpenFile(filepath.Join("..", "_data", fn), os.O_RDONLY, os.ModePerm)
	if err != nil {
		return nil, err
	}
	defer fh.Close()

	var chain types.Blocks
	if err := rlp.Decode(fh, &chain); err != nil {
		return nil, err
	}

	return chain, nil
}

func insertChain(done chan bool, blockchain *BlockChain, chain types.Blocks, t *testing.T) {
	_, err := blockchain.InsertChain(chain)
	if err != nil {
		fmt.Println(err)
		t.FailNow()
	}
	done <- true
}

func TestLastBlock(t *testing.T) {
	db, _ := ethdb.NewMemDatabase()

	bchain := theBlockChain(db, t)
	block := makeBlockChain(bchain.CurrentBlock(), 1, db, 0)[0]
	bchain.insert(block)
	if block.Hash() != GetHeadBlockHash(db) {
		t.Errorf("Write/Get HeadBlockHash failed")
	}
}

// Tests that given a starting canonical chain of a given size, it can be extended
// with various length chains.
func TestExtendCanonicalHeaders(t *testing.T) { testExtendCanonical(t, false) }
func TestExtendCanonicalBlocks(t *testing.T)  { testExtendCanonical(t, true) }

func testExtendCanonical(t *testing.T, full bool) {
	length := 5

	// Make first chain starting from genesis
	_, processor, err := newCanonical(length, full)
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
	testFork(t, processor, length, 1, full, better)
	testFork(t, processor, length, 2, full, better)
	testFork(t, processor, length, 5, full, better)
	testFork(t, processor, length, 10, full, better)
}

// Tests that given a starting canonical chain of a given size, creating shorter
// forks do not take canonical ownership.
func TestShorterForkHeaders(t *testing.T) { testShorterFork(t, false) }
func TestShorterForkBlocks(t *testing.T)  { testShorterFork(t, true) }

func testShorterFork(t *testing.T, full bool) {
	length := 10

	// Make first chain starting from genesis
	_, processor, err := newCanonical(length, full)
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
	testFork(t, processor, 0, 3, full, worse)
	testFork(t, processor, 0, 7, full, worse)
	testFork(t, processor, 1, 1, full, worse)
	testFork(t, processor, 1, 7, full, worse)
	testFork(t, processor, 5, 3, full, worse)
	testFork(t, processor, 5, 4, full, worse)
}

// Tests that given a starting canonical chain of a given size, creating longer
// forks do take canonical ownership.
func TestLongerForkHeaders(t *testing.T) { testLongerFork(t, false) }
func TestLongerForkBlocks(t *testing.T)  { testLongerFork(t, true) }

func testLongerFork(t *testing.T, full bool) {
	length := 10

	// Make first chain starting from genesis
	_, processor, err := newCanonical(length, full)
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
	testFork(t, processor, 0, 11, full, better)
	testFork(t, processor, 0, 15, full, better)
	testFork(t, processor, 1, 10, full, better)
	testFork(t, processor, 1, 12, full, better)
	testFork(t, processor, 5, 6, full, better)
	testFork(t, processor, 5, 8, full, better)
}

// Tests that given a starting canonical chain of a given size, creating equal
// forks do take canonical ownership.
func TestEqualForkHeaders(t *testing.T) { testEqualFork(t, false) }
func TestEqualForkBlocks(t *testing.T)  { testEqualFork(t, true) }

func testEqualFork(t *testing.T, full bool) {
	length := 10

	// Make first chain starting from genesis
	_, processor, err := newCanonical(length, full)
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
	testFork(t, processor, 0, 10, full, equal)
	testFork(t, processor, 1, 9, full, equal)
	testFork(t, processor, 2, 8, full, equal)
	testFork(t, processor, 5, 5, full, equal)
	testFork(t, processor, 6, 4, full, equal)
	testFork(t, processor, 9, 1, full, equal)
}

// Tests that chains missing links do not get accepted by the processor.
func TestBrokenHeaderChain(t *testing.T) { testBrokenChain(t, false) }
func TestBrokenBlockChain(t *testing.T)  { testBrokenChain(t, true) }

func testBrokenChain(t *testing.T, full bool) {
	// Make chain starting from genesis
	db, blockchain, err := newCanonical(10, full)
	if err != nil {
		t.Fatalf("failed to make new canonical chain: %v", err)
	}
	// Create a forked chain, and try to insert with a missing link
	if full {
		chain := makeBlockChain(blockchain.CurrentBlock(), 5, db, forkSeed)[1:]
		if err := testBlockChainImport(chain, blockchain); err == nil {
			t.Errorf("broken block chain not reported")
		}
	} else {
		chain := makeHeaderChain(blockchain.CurrentHeader(), 5, db, forkSeed)[1:]
		if err := testHeaderChainImport(chain, blockchain); err == nil {
			t.Errorf("broken header chain not reported")
		}
	}
}

func TestChainInsertions(t *testing.T) {
	t.Skip("Skipped: outdated test files")

	db, _ := ethdb.NewMemDatabase()

	chain1, err := loadChain("valid1", t)
	if err != nil {
		fmt.Println(err)
		t.FailNow()
	}

	chain2, err := loadChain("valid2", t)
	if err != nil {
		fmt.Println(err)
		t.FailNow()
	}

	blockchain := theBlockChain(db, t)

	const max = 2
	done := make(chan bool, max)

	go insertChain(done, blockchain, chain1, t)
	go insertChain(done, blockchain, chain2, t)

	for i := 0; i < max; i++ {
		<-done
	}

	if chain2[len(chain2)-1].Hash() != blockchain.CurrentBlock().Hash() {
		t.Error("chain2 is canonical and shouldn't be")
	}

	if chain1[len(chain1)-1].Hash() != blockchain.CurrentBlock().Hash() {
		t.Error("chain1 isn't canonical and should be")
	}
}

func TestChainMultipleInsertions(t *testing.T) {
	t.Skip("Skipped: outdated test files")

	db, _ := ethdb.NewMemDatabase()

	const max = 4
	chains := make([]types.Blocks, max)
	var longest int
	for i := 0; i < max; i++ {
		var err error
		name := "valid" + strconv.Itoa(i+1)
		chains[i], err = loadChain(name, t)
		if len(chains[i]) >= len(chains[longest]) {
			longest = i
		}
		fmt.Println("loaded", name, "with a length of", len(chains[i]))
		if err != nil {
			fmt.Println(err)
			t.FailNow()
		}
	}

	blockchain := theBlockChain(db, t)

	done := make(chan bool, max)
	for i, chain := range chains {
		// XXX the go routine would otherwise reference the same (chain[3]) variable and fail
		i := i
		chain := chain
		go func() {
			insertChain(done, blockchain, chain, t)
			fmt.Println(i, "done")
		}()
	}

	for i := 0; i < max; i++ {
		<-done
	}

	if chains[longest][len(chains[longest])-1].Hash() != blockchain.CurrentBlock().Hash() {
		t.Error("Invalid canonical chain")
	}
}

type bproc struct{}

func (bproc) ValidateBlock(*types.Block) error                        { return nil }
func (bproc) ValidateHeader(*types.Header, *types.Header, bool) error { return nil }
func (bproc) ValidateState(block, parent *types.Block, state *state.StateDB, receipts types.Receipts, usedGas *big.Int) error {
	return nil
}
func (bproc) Process(block *types.Block, statedb *state.StateDB) (types.Receipts, vm.Logs, *big.Int, error) {
	return nil, nil, nil, nil
}

func makeHeaderChainWithDiff(genesis *types.Block, d []int, seed byte) []*types.Header {
	blocks := makeBlockChainWithDiff(genesis, d, seed)
	headers := make([]*types.Header, len(blocks))
	for i, block := range blocks {
		headers[i] = block.Header()
	}
	return headers
}

func makeBlockChainWithDiff(genesis *types.Block, d []int, seed byte) []*types.Block {
	var chain []*types.Block
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
		block := types.NewBlockWithHeader(header)
		chain = append(chain, block)
	}
	return chain
}

func chm(genesis *types.Block, db ethdb.Database) *BlockChain {
	var eventMux event.TypeMux
	bc := &BlockChain{
		chainDb: db,
		genesisBlock: genesis,
		eventMux: &eventMux,
		pow: FakePow{},
	}
	valFn := func() HeaderValidator { return bc.Validator() }
	bc.hc, _ = NewHeaderChain(db, valFn, bc.getProcInterrupt)
	bc.bodyCache, _ = lru.New(100)
	bc.bodyRLPCache, _ = lru.New(100)
	bc.blockCache, _ = lru.New(100)
	bc.futureBlocks, _ = lru.New(100)
	bc.SetValidator(bproc{})
	bc.SetProcessor(bproc{})
	bc.ResetWithGenesisBlock(genesis)

	return bc
}

// Tests that reorganising a long difficult chain after a short easy one
// overwrites the canonical numbers and links in the database.
func TestReorgLongHeaders(t *testing.T) { testReorgLong(t, false) }
func TestReorgLongBlocks(t *testing.T)  { testReorgLong(t, true) }

func testReorgLong(t *testing.T, full bool) {
	testReorg(t, []int{1, 2, 4}, []int{1, 2, 3, 4}, 10, full)
}

// Tests that reorganising a short difficult chain after a long easy one
// overwrites the canonical numbers and links in the database.
func TestReorgShortHeaders(t *testing.T) { testReorgShort(t, false) }
func TestReorgShortBlocks(t *testing.T)  { testReorgShort(t, true) }

func testReorgShort(t *testing.T, full bool) {
	testReorg(t, []int{1, 2, 3, 4}, []int{1, 10}, 11, full)
}

func testReorg(t *testing.T, first, second []int, td int64, full bool) {
	// Create a pristine block chain
	db, _ := ethdb.NewMemDatabase()
	genesis, _ := WriteTestNetGenesisBlock(db)
	bc := chm(genesis, db)

	// Insert an easy and a difficult chain afterwards
	if full {
		bc.InsertChain(makeBlockChainWithDiff(genesis, first, 11))
		bc.InsertChain(makeBlockChainWithDiff(genesis, second, 22))
	} else {
		bc.InsertHeaderChain(makeHeaderChainWithDiff(genesis, first, 11), 1)
		bc.InsertHeaderChain(makeHeaderChainWithDiff(genesis, second, 22), 1)
	}
	// Check that the chain is valid number and link wise
	if full {
		prev := bc.CurrentBlock()
		for block := bc.GetBlockByNumber(bc.CurrentBlock().NumberU64() - 1); block.NumberU64() != 0; prev, block = block, bc.GetBlockByNumber(block.NumberU64()-1) {
			if prev.ParentHash() != block.Hash() {
				t.Errorf("parent block hash mismatch: have %x, want %x", prev.ParentHash(), block.Hash())
			}
		}
	} else {
		prev := bc.CurrentHeader()
		for header := bc.GetHeaderByNumber(bc.CurrentHeader().Number.Uint64() - 1); header.Number.Uint64() != 0; prev, header = header, bc.GetHeaderByNumber(header.Number.Uint64()-1) {
			if prev.ParentHash != header.Hash() {
				t.Errorf("parent header hash mismatch: have %x, want %x", prev.ParentHash, header.Hash())
			}
		}
	}
	// Make sure the chain total difficulty is the correct one
	want := new(big.Int).Add(genesis.Difficulty(), big.NewInt(td))
	if full {
		if have := bc.GetTd(bc.CurrentBlock().Hash()); have.Cmp(want) != 0 {
			t.Errorf("total difficulty mismatch: have %v, want %v", have, want)
		}
	} else {
		if have := bc.GetTd(bc.CurrentHeader().Hash()); have.Cmp(want) != 0 {
			t.Errorf("total difficulty mismatch: have %v, want %v", have, want)
		}
	}
}

// Tests that the insertion functions detect banned hashes.
func TestBadHeaderHashes(t *testing.T) { testBadHashes(t, false) }
func TestBadBlockHashes(t *testing.T)  { testBadHashes(t, true) }

func testBadHashes(t *testing.T, full bool) {
	// Create a pristine block chain
	db, _ := ethdb.NewMemDatabase()
	genesis, _ := WriteTestNetGenesisBlock(db)
	bc := chm(genesis, db)

	// Create a chain, ban a hash and try to import
	var err error
	if full {
		blocks := makeBlockChainWithDiff(genesis, []int{1, 2, 4}, 10)
		BadHashes[blocks[2].Header().Hash()] = true
		_, err = bc.InsertChain(blocks)
	} else {
		headers := makeHeaderChainWithDiff(genesis, []int{1, 2, 4}, 10)
		BadHashes[headers[2].Hash()] = true
		_, err = bc.InsertHeaderChain(headers, 1)
	}
	if !IsBadHashError(err) {
		t.Errorf("error mismatch: want: BadHashError, have: %v", err)
	}
}

// Tests that bad hashes are detected on boot, and the chain rolled back to a
// good state prior to the bad hash.
func TestReorgBadHeaderHashes(t *testing.T) { testReorgBadHashes(t, false) }
func TestReorgBadBlockHashes(t *testing.T)  { testReorgBadHashes(t, true) }

func testReorgBadHashes(t *testing.T, full bool) {
	// Create a pristine block chain
	db, _ := ethdb.NewMemDatabase()
	genesis, _ := WriteTestNetGenesisBlock(db)
	bc := chm(genesis, db)

	// Create a chain, import and ban afterwards
	headers := makeHeaderChainWithDiff(genesis, []int{1, 2, 3, 4}, 10)
	blocks := makeBlockChainWithDiff(genesis, []int{1, 2, 3, 4}, 10)

	if full {
		if _, err := bc.InsertChain(blocks); err != nil {
			t.Fatalf("failed to import blocks: %v", err)
		}
		if bc.CurrentBlock().Hash() != blocks[3].Hash() {
			t.Errorf("last block hash mismatch: have: %x, want %x", bc.CurrentBlock().Hash(), blocks[3].Header().Hash())
		}
		BadHashes[blocks[3].Header().Hash()] = true
		defer func() { delete(BadHashes, blocks[3].Header().Hash()) }()
	} else {
		if _, err := bc.InsertHeaderChain(headers, 1); err != nil {
			t.Fatalf("failed to import headers: %v", err)
		}
		if bc.CurrentHeader().Hash() != headers[3].Hash() {
			t.Errorf("last header hash mismatch: have: %x, want %x", bc.CurrentHeader().Hash(), headers[3].Hash())
		}
		BadHashes[headers[3].Hash()] = true
		defer func() { delete(BadHashes, headers[3].Hash()) }()
	}
	// Create a new chain manager and check it rolled back the state
	ncm, err := NewBlockChain(db, FakePow{}, new(event.TypeMux))
	if err != nil {
		t.Fatalf("failed to create new chain manager: %v", err)
	}
	if full {
		if ncm.CurrentBlock().Hash() != blocks[2].Header().Hash() {
			t.Errorf("last block hash mismatch: have: %x, want %x", ncm.CurrentBlock().Hash(), blocks[2].Header().Hash())
		}
		if blocks[2].Header().GasLimit.Cmp(ncm.GasLimit()) != 0 {
			t.Errorf("last  block gasLimit mismatch: have: %x, want %x", ncm.GasLimit(), blocks[2].Header().GasLimit)
		}
	} else {
		if ncm.CurrentHeader().Hash() != headers[2].Hash() {
			t.Errorf("last header hash mismatch: have: %x, want %x", ncm.CurrentHeader().Hash(), headers[2].Hash())
		}
	}
}

// Tests chain insertions in the face of one entity containing an invalid nonce.
func TestHeadersInsertNonceError(t *testing.T) { testInsertNonceError(t, false) }
func TestBlocksInsertNonceError(t *testing.T)  { testInsertNonceError(t, true) }

func testInsertNonceError(t *testing.T, full bool) {
	for i := 1; i < 25 && !t.Failed(); i++ {
		// Create a pristine chain and database
		db, blockchain, err := newCanonical(0, full)
		if err != nil {
			t.Fatalf("failed to create pristine chain: %v", err)
		}
		// Create and insert a chain with a failing nonce
		var (
			failAt   int
			failRes  int
			failNum  uint64
			failHash common.Hash
		)
		if full {
			blocks := makeBlockChain(blockchain.CurrentBlock(), i, db, 0)

			failAt = rand.Int() % len(blocks)
			failNum = blocks[failAt].NumberU64()
			failHash = blocks[failAt].Hash()

			blockchain.pow = failPow{failNum}

			failRes, err = blockchain.InsertChain(blocks)
		} else {
			headers := makeHeaderChain(blockchain.CurrentHeader(), i, db, 0)

			failAt = rand.Int() % len(headers)
			failNum = headers[failAt].Number.Uint64()
			failHash = headers[failAt].Hash()

			blockchain.pow = failPow{failNum}
			blockchain.validator = NewBlockValidator(blockchain, failPow{failNum})

			failRes, err = blockchain.InsertHeaderChain(headers, 1)
		}
		// Check that the returned error indicates the nonce failure.
		if failRes != failAt {
			t.Errorf("test %d: failure index mismatch: have %d, want %d", i, failRes, failAt)
		}
		if !IsBlockNonceErr(err) {
			t.Fatalf("test %d: error mismatch: have %v, want nonce error %T", i, err, err)
		}
		nerr := err.(*BlockNonceErr)
		if nerr.Number.Uint64() != failNum {
			t.Errorf("test %d: number mismatch: have %v, want %v", i, nerr.Number, failNum)
		}
		if nerr.Hash != failHash {
			t.Errorf("test %d: hash mismatch: have %x, want %x", i, nerr.Hash[:4], failHash[:4])
		}
		// Check that all no blocks after the failing block have been inserted.
		for j := 0; j < i-failAt; j++ {
			if full {
				if block := blockchain.GetBlockByNumber(failNum + uint64(j)); block != nil {
					t.Errorf("test %d: invalid block in chain: %v", i, block)
				}
			} else {
				if header := blockchain.GetHeaderByNumber(failNum + uint64(j)); header != nil {
					t.Errorf("test %d: invalid header in chain: %v", i, header)
				}
			}
		}
	}
}

// Tests that fast importing a block chain produces the same chain data as the
// classical full block processing.
func TestFastVsFullChains(t *testing.T) {
	// Configure and generate a sample block chain
	var (
		gendb, _ = ethdb.NewMemDatabase()
		key, _   = crypto.HexToECDSA("b71c71a67e1177ad4e901695e1b4b9ee17ae16c6668d313eac2f96dbcda3f291")
		address  = crypto.PubkeyToAddress(key.PublicKey)
		funds    = big.NewInt(1000000000)
		genesis  = GenesisBlockForTesting(gendb, address, funds)
	)
	blocks, receipts := GenerateChain(genesis, gendb, 1024, func(i int, block *BlockGen) {
		block.SetCoinbase(common.Address{0x00})

		// If the block number is multiple of 3, send a few bonus transactions to the miner
		if i%3 == 2 {
			for j := 0; j < i%4+1; j++ {
				tx, err := types.NewTransaction(block.TxNonce(address), common.Address{0x00}, big.NewInt(1000), params.TxGas, nil, nil).SignECDSA(key)
				if err != nil {
					panic(err)
				}
				block.AddTx(tx)
			}
		}
		// If the block number is a multiple of 5, add a few bonus uncles to the block
		if i%5 == 5 {
			block.AddUncle(&types.Header{ParentHash: block.PrevBlock(i - 1).Hash(), Number: big.NewInt(int64(i - 1))})
		}
	})
	// Import the chain as an archive node for the comparison baseline
	archiveDb, _ := ethdb.NewMemDatabase()
	WriteGenesisBlockForTesting(archiveDb, GenesisAccount{address, funds})

	archive, _ := NewBlockChain(archiveDb, FakePow{}, new(event.TypeMux))

	if n, err := archive.InsertChain(blocks); err != nil {
		t.Fatalf("failed to process block %d: %v", n, err)
	}
	// Fast import the chain as a non-archive node to test
	fastDb, _ := ethdb.NewMemDatabase()
	WriteGenesisBlockForTesting(fastDb, GenesisAccount{address, funds})
	fast, _ := NewBlockChain(fastDb, FakePow{}, new(event.TypeMux))

	headers := make([]*types.Header, len(blocks))
	for i, block := range blocks {
		headers[i] = block.Header()
	}
	if n, err := fast.InsertHeaderChain(headers, 1); err != nil {
		t.Fatalf("failed to insert header %d: %v", n, err)
	}
	if n, err := fast.InsertReceiptChain(blocks, receipts); err != nil {
		t.Fatalf("failed to insert receipt %d: %v", n, err)
	}
	// Iterate over all chain data components, and cross reference
	for i := 0; i < len(blocks); i++ {
		num, hash := blocks[i].NumberU64(), blocks[i].Hash()

		if ftd, atd := fast.GetTd(hash), archive.GetTd(hash); ftd.Cmp(atd) != 0 {
			t.Errorf("block #%d [%x]: td mismatch: have %v, want %v", num, hash, ftd, atd)
		}
		if fheader, aheader := fast.GetHeader(hash), archive.GetHeader(hash); fheader.Hash() != aheader.Hash() {
			t.Errorf("block #%d [%x]: header mismatch: have %v, want %v", num, hash, fheader, aheader)
		}
		if fblock, ablock := fast.GetBlock(hash), archive.GetBlock(hash); fblock.Hash() != ablock.Hash() {
			t.Errorf("block #%d [%x]: block mismatch: have %v, want %v", num, hash, fblock, ablock)
		} else if types.DeriveSha(fblock.Transactions()) != types.DeriveSha(ablock.Transactions()) {
			t.Errorf("block #%d [%x]: transactions mismatch: have %v, want %v", num, hash, fblock.Transactions(), ablock.Transactions())
		} else if types.CalcUncleHash(fblock.Uncles()) != types.CalcUncleHash(ablock.Uncles()) {
			t.Errorf("block #%d [%x]: uncles mismatch: have %v, want %v", num, hash, fblock.Uncles(), ablock.Uncles())
		}
		if freceipts, areceipts := GetBlockReceipts(fastDb, hash), GetBlockReceipts(archiveDb, hash); types.DeriveSha(freceipts) != types.DeriveSha(areceipts) {
			t.Errorf("block #%d [%x]: receipts mismatch: have %v, want %v", num, hash, freceipts, areceipts)
		}
	}
	// Check that the canonical chains are the same between the databases
	for i := 0; i < len(blocks)+1; i++ {
		if fhash, ahash := GetCanonicalHash(fastDb, uint64(i)), GetCanonicalHash(archiveDb, uint64(i)); fhash != ahash {
			t.Errorf("block #%d: canonical hash mismatch: have %v, want %v", i, fhash, ahash)
		}
	}
}

// Tests that various import methods move the chain head pointers to the correct
// positions.
func TestLightVsFastVsFullChainHeads(t *testing.T) {
	// Configure and generate a sample block chain
	var (
		gendb, _ = ethdb.NewMemDatabase()
		key, _   = crypto.HexToECDSA("b71c71a67e1177ad4e901695e1b4b9ee17ae16c6668d313eac2f96dbcda3f291")
		address  = crypto.PubkeyToAddress(key.PublicKey)
		funds    = big.NewInt(1000000000)
		genesis  = GenesisBlockForTesting(gendb, address, funds)
	)
	height := uint64(1024)
	blocks, receipts := GenerateChain(genesis, gendb, int(height), nil)

	// Configure a subchain to roll back
	remove := []common.Hash{}
	for _, block := range blocks[height/2:] {
		remove = append(remove, block.Hash())
	}
	// Create a small assertion method to check the three heads
	assert := func(t *testing.T, kind string, chain *BlockChain, header uint64, fast uint64, block uint64) {
		if num := chain.CurrentBlock().NumberU64(); num != block {
			t.Errorf("%s head block mismatch: have #%v, want #%v", kind, num, block)
		}
		if num := chain.CurrentFastBlock().NumberU64(); num != fast {
			t.Errorf("%s head fast-block mismatch: have #%v, want #%v", kind, num, fast)
		}
		if num := chain.CurrentHeader().Number.Uint64(); num != header {
			t.Errorf("%s head header mismatch: have #%v, want #%v", kind, num, header)
		}
	}
	// Import the chain as an archive node and ensure all pointers are updated
	archiveDb, _ := ethdb.NewMemDatabase()
	WriteGenesisBlockForTesting(archiveDb, GenesisAccount{address, funds})

	archive, _ := NewBlockChain(archiveDb, FakePow{}, new(event.TypeMux))

	if n, err := archive.InsertChain(blocks); err != nil {
		t.Fatalf("failed to process block %d: %v", n, err)
	}
	assert(t, "archive", archive, height, height, height)
	archive.Rollback(remove)
	assert(t, "archive", archive, height/2, height/2, height/2)

	// Import the chain as a non-archive node and ensure all pointers are updated
	fastDb, _ := ethdb.NewMemDatabase()
	WriteGenesisBlockForTesting(fastDb, GenesisAccount{address, funds})
	fast, _ := NewBlockChain(fastDb, FakePow{}, new(event.TypeMux))

	headers := make([]*types.Header, len(blocks))
	for i, block := range blocks {
		headers[i] = block.Header()
	}
	if n, err := fast.InsertHeaderChain(headers, 1); err != nil {
		t.Fatalf("failed to insert header %d: %v", n, err)
	}
	if n, err := fast.InsertReceiptChain(blocks, receipts); err != nil {
		t.Fatalf("failed to insert receipt %d: %v", n, err)
	}
	assert(t, "fast", fast, height, height, 0)
	fast.Rollback(remove)
	assert(t, "fast", fast, height/2, height/2, 0)

	// Import the chain as a light node and ensure all pointers are updated
	lightDb, _ := ethdb.NewMemDatabase()
	WriteGenesisBlockForTesting(lightDb, GenesisAccount{address, funds})
	light, _ := NewBlockChain(lightDb, FakePow{}, new(event.TypeMux))

	if n, err := light.InsertHeaderChain(headers, 1); err != nil {
		t.Fatalf("failed to insert header %d: %v", n, err)
	}
	assert(t, "light", light, height, 0, 0)
	light.Rollback(remove)
	assert(t, "light", light, height/2, 0, 0)
}

// Tests that chain reorganisations handle transaction removals and reinsertions.
func TestChainTxReorgs(t *testing.T) {
	params.MinGasLimit = big.NewInt(125000)      // Minimum the gas limit may ever be.
	params.GenesisGasLimit = big.NewInt(3141592) // Gas limit of the Genesis block.

	var (
		key1, _ = crypto.HexToECDSA("b71c71a67e1177ad4e901695e1b4b9ee17ae16c6668d313eac2f96dbcda3f291")
		key2, _ = crypto.HexToECDSA("8a1f9a8f95be41cd7ccb6168179afb4504aefe388d1e14474d32c45c72ce7b7a")
		key3, _ = crypto.HexToECDSA("49a7b37aa6f6645917e7b807e9d1c00d4fa71f18343b0d4122a4d2df64dd6fee")
		addr1   = crypto.PubkeyToAddress(key1.PublicKey)
		addr2   = crypto.PubkeyToAddress(key2.PublicKey)
		addr3   = crypto.PubkeyToAddress(key3.PublicKey)
		db, _   = ethdb.NewMemDatabase()
	)
	genesis := WriteGenesisBlockForTesting(db,
		GenesisAccount{addr1, big.NewInt(1000000)},
		GenesisAccount{addr2, big.NewInt(1000000)},
		GenesisAccount{addr3, big.NewInt(1000000)},
	)
	// Create two transactions shared between the chains:
	//  - postponed: transaction included at a later block in the forked chain
	//  - swapped: transaction included at the same block number in the forked chain
	postponed, _ := types.NewTransaction(0, addr1, big.NewInt(1000), params.TxGas, nil, nil).SignECDSA(key1)
	swapped, _ := types.NewTransaction(1, addr1, big.NewInt(1000), params.TxGas, nil, nil).SignECDSA(key1)

	// Create two transactions that will be dropped by the forked chain:
	//  - pastDrop: transaction dropped retroactively from a past block
	//  - freshDrop: transaction dropped exactly at the block where the reorg is detected
	var pastDrop, freshDrop *types.Transaction

	// Create three transactions that will be added in the forked chain:
	//  - pastAdd:   transaction added before the reorganization is detected
	//  - freshAdd:  transaction added at the exact block the reorg is detected
	//  - futureAdd: transaction added after the reorg has already finished
	var pastAdd, freshAdd, futureAdd *types.Transaction

	chain, _ := GenerateChain(genesis, db, 3, func(i int, gen *BlockGen) {
		switch i {
		case 0:
			pastDrop, _ = types.NewTransaction(gen.TxNonce(addr2), addr2, big.NewInt(1000), params.TxGas, nil, nil).SignECDSA(key2)

			gen.AddTx(pastDrop)  // This transaction will be dropped in the fork from below the split point
			gen.AddTx(postponed) // This transaction will be postponed till block #3 in the fork

		case 2:
			freshDrop, _ = types.NewTransaction(gen.TxNonce(addr2), addr2, big.NewInt(1000), params.TxGas, nil, nil).SignECDSA(key2)

			gen.AddTx(freshDrop) // This transaction will be dropped in the fork from exactly at the split point
			gen.AddTx(swapped)   // This transaction will be swapped out at the exact height

			gen.OffsetTime(9) // Lower the block difficulty to simulate a weaker chain
		}
	})
	// Import the chain. This runs all block validation rules.
	evmux := &event.TypeMux{}
	blockchain, _ := NewBlockChain(db, FakePow{}, evmux)
	if i, err := blockchain.InsertChain(chain); err != nil {
		t.Fatalf("failed to insert original chain[%d]: %v", i, err)
	}

	// overwrite the old chain
	chain, _ = GenerateChain(genesis, db, 5, func(i int, gen *BlockGen) {
		switch i {
		case 0:
			pastAdd, _ = types.NewTransaction(gen.TxNonce(addr3), addr3, big.NewInt(1000), params.TxGas, nil, nil).SignECDSA(key3)
			gen.AddTx(pastAdd) // This transaction needs to be injected during reorg

		case 2:
			gen.AddTx(postponed) // This transaction was postponed from block #1 in the original chain
			gen.AddTx(swapped)   // This transaction was swapped from the exact current spot in the original chain

			freshAdd, _ = types.NewTransaction(gen.TxNonce(addr3), addr3, big.NewInt(1000), params.TxGas, nil, nil).SignECDSA(key3)
			gen.AddTx(freshAdd) // This transaction will be added exactly at reorg time

		case 3:
			futureAdd, _ = types.NewTransaction(gen.TxNonce(addr3), addr3, big.NewInt(1000), params.TxGas, nil, nil).SignECDSA(key3)
			gen.AddTx(futureAdd) // This transaction will be added after a full reorg
		}
	})
	if _, err := blockchain.InsertChain(chain); err != nil {
		t.Fatalf("failed to insert forked chain: %v", err)
	}

	// removed tx
	for i, tx := range (types.Transactions{pastDrop, freshDrop}) {
		if txn, _, _, _ := GetTransaction(db, tx.Hash()); txn != nil {
			t.Errorf("drop %d: tx %v found while shouldn't have been", i, txn)
		}
		if GetReceipt(db, tx.Hash()) != nil {
			t.Errorf("drop %d: receipt found while shouldn't have been", i)
		}
	}
	// added tx
	for i, tx := range (types.Transactions{pastAdd, freshAdd, futureAdd}) {
		if txn, _, _, _ := GetTransaction(db, tx.Hash()); txn == nil {
			t.Errorf("add %d: expected tx to be found", i)
		}
		if GetReceipt(db, tx.Hash()) == nil {
			t.Errorf("add %d: expected receipt to be found", i)
		}
	}
	// shared tx
	for i, tx := range (types.Transactions{postponed, swapped}) {
		if txn, _, _, _ := GetTransaction(db, tx.Hash()); txn == nil {
			t.Errorf("share %d: expected tx to be found", i)
		}
		if GetReceipt(db, tx.Hash()) == nil {
			t.Errorf("share %d: expected receipt to be found", i)
		}
	}
}

func TestLogReorgs(t *testing.T) {
	params.MinGasLimit = big.NewInt(125000)      // Minimum the gas limit may ever be.
	params.GenesisGasLimit = big.NewInt(3141592) // Gas limit of the Genesis block.

	var (
		key1, _ = crypto.HexToECDSA("b71c71a67e1177ad4e901695e1b4b9ee17ae16c6668d313eac2f96dbcda3f291")
		addr1   = crypto.PubkeyToAddress(key1.PublicKey)
		db, _   = ethdb.NewMemDatabase()
		// this code generates a log
		code = common.Hex2Bytes("60606040525b7f24ec1d3ff24c2f6ff210738839dbc339cd45a5294d85c79361016243157aae7b60405180905060405180910390a15b600a8060416000396000f360606040526008565b00")
	)
	genesis := WriteGenesisBlockForTesting(db,
		GenesisAccount{addr1, big.NewInt(10000000000000)},
	)

	evmux := &event.TypeMux{}
	blockchain, _ := NewBlockChain(db, FakePow{}, evmux)

	subs := evmux.Subscribe(RemovedLogsEvent{})
	chain, _ := GenerateChain(genesis, db, 2, func(i int, gen *BlockGen) {
		if i == 1 {
			tx, err := types.NewContractCreation(gen.TxNonce(addr1), new(big.Int), big.NewInt(1000000), new(big.Int), code).SignECDSA(key1)
			if err != nil {
				t.Fatalf("failed to create tx: %v", err)
			}
			gen.AddTx(tx)
		}
	})
	if _, err := blockchain.InsertChain(chain); err != nil {
		t.Fatalf("failed to insert chain: %v", err)
	}

	chain, _ = GenerateChain(genesis, db, 3, func(i int, gen *BlockGen) {})
	if _, err := blockchain.InsertChain(chain); err != nil {
		t.Fatalf("failed to insert forked chain: %v", err)
	}

	ev := <-subs.Chan()
	if len(ev.Data.(RemovedLogsEvent).Logs) == 0 {
		t.Error("expected logs")
	}
}

func TestReorgSideEvent(t *testing.T) {
	var (
		db, _   = ethdb.NewMemDatabase()
		key1, _ = crypto.HexToECDSA("b71c71a67e1177ad4e901695e1b4b9ee17ae16c6668d313eac2f96dbcda3f291")
		addr1   = crypto.PubkeyToAddress(key1.PublicKey)
		genesis = WriteGenesisBlockForTesting(db, GenesisAccount{addr1, big.NewInt(10000000000000)})
	)

	evmux := &event.TypeMux{}
	blockchain, _ := NewBlockChain(db, FakePow{}, evmux)

	chain, _ := GenerateChain(genesis, db, 3, func(i int, gen *BlockGen) {
		if i == 2 {
			gen.OffsetTime(9)
		}
	})
	if _, err := blockchain.InsertChain(chain); err != nil {
		t.Fatalf("failed to insert chain: %v", err)
	}

	replacementBlocks, _ := GenerateChain(genesis, db, 4, func(i int, gen *BlockGen) {
		tx, err := types.NewContractCreation(gen.TxNonce(addr1), new(big.Int), big.NewInt(1000000), new(big.Int), nil).SignECDSA(key1)
		if err != nil {
			t.Fatalf("failed to create tx: %v", err)
		}
		gen.AddTx(tx)
	})

	subs := evmux.Subscribe(ChainSideEvent{})
	if _, err := blockchain.InsertChain(replacementBlocks); err != nil {
		t.Fatalf("failed to insert chain: %v", err)
	}

	// first two block of the secondary chain are for a brief moment considered
	// side chains because up to that point the first one is considered the
	// heavier chain.
	expectedSideHashes := map[common.Hash]bool{
		replacementBlocks[0].Hash(): true,
		replacementBlocks[1].Hash(): true,
		chain[0].Hash():             true,
		chain[1].Hash():             true,
		chain[2].Hash():             true,
	}

	i := 0

	const timeoutDura = 10 * time.Second
	timeout := time.NewTimer(timeoutDura)
done:
	for {
		select {
		case ev := <-subs.Chan():
			block := ev.Data.(ChainSideEvent).Block
			if _, ok := expectedSideHashes[block.Hash()]; !ok {
				t.Errorf("%d: didn't expect %x to be in side chain", i, block.Hash())
			}
			i++

			if i == len(expectedSideHashes) {
				timeout.Stop()

				break done
			}
			timeout.Reset(timeoutDura)

		case <-timeout.C:
			t.Fatal("Timeout. Possibly not all blocks were triggered for sideevent")
		}
	}

	// make sure no more events are fired
	select {
	case e := <-subs.Chan():
		t.Errorf("unexpected event fired: %v", e)
	case <-time.After(250 * time.Millisecond):
	}

}
