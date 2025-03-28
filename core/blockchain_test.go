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
	"errors"
	"fmt"
	"math/big"
	"math/rand"
	"sync"
	"testing"
	"time"

	"github.com/XinFinOrg/XDPoSChain/common"
	"github.com/XinFinOrg/XDPoSChain/consensus"
	"github.com/XinFinOrg/XDPoSChain/consensus/ethash"
	"github.com/XinFinOrg/XDPoSChain/core/rawdb"
	"github.com/XinFinOrg/XDPoSChain/core/state"
	"github.com/XinFinOrg/XDPoSChain/core/types"
	"github.com/XinFinOrg/XDPoSChain/core/vm"
	"github.com/XinFinOrg/XDPoSChain/crypto"
	"github.com/XinFinOrg/XDPoSChain/ethdb"
	"github.com/XinFinOrg/XDPoSChain/params"
	"github.com/XinFinOrg/XDPoSChain/trie"
)

// newCanonical creates a chain database, and injects a deterministic canonical
// chain. Depending on the full flag, it creates either a full block chain or a
// header only chain. The database and genesis specification for block generation
// are also returned in case more test blocks are needed later.
func newCanonical(engine consensus.Engine, n int, full bool) (ethdb.Database, *BlockChain, error) {
	var (
		gspec = &Genesis{
			BaseFee: big.NewInt(params.InitialBaseFee),
			Config:  params.AllEthashProtocolChanges,
		}
	)
	db := rawdb.NewMemoryDatabase()
	genesis := gspec.MustCommit(db)

	// Initialize a fresh chain with only a genesis block
	blockchain, _ := NewBlockChain(db, nil, params.AllEthashProtocolChanges, engine, vm.Config{})

	// Create and inject the requested chain
	if n == 0 {
		return db, blockchain, nil
	}
	if full {
		// Full block-chain requested
		blocks := makeBlockChain(genesis, n, engine, db, canonicalSeed)
		_, err := blockchain.InsertChain(blocks)
		return db, blockchain, err
	}
	// Header-only chain requested
	headers := makeHeaderChain(genesis.Header(), n, engine, db, canonicalSeed)
	_, err := blockchain.InsertHeaderChain(headers, 1)
	return db, blockchain, err
}

// Test fork of length N starting from block i
func testFork(t *testing.T, blockchain *BlockChain, i, n int, full bool, comparator func(td1, td2 *big.Int)) {
	// Copy old chain up to #i into a new db
	db, blockchain2, err := newCanonical(ethash.NewFaker(), i, full)
	if err != nil {
		t.Fatal("could not make new canonical in testFork", err)
	}
	defer blockchain2.Stop()

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
		blockChainB = makeBlockChain(blockchain2.CurrentBlock(), n, ethash.NewFaker(), db, forkSeed)
		if _, err := blockchain2.InsertChain(blockChainB); err != nil {
			t.Fatalf("failed to insert forking chain: %v", err)
		}
	} else {
		headerChainB = makeHeaderChain(blockchain2.CurrentHeader(), n, ethash.NewFaker(), db, forkSeed)
		if _, err := blockchain2.InsertHeaderChain(headerChainB, 1); err != nil {
			t.Fatalf("failed to insert forking chain: %v", err)
		}
	}
	// Sanity check that the forked chain can be imported into the original
	var tdPre, tdPost *big.Int

	if full {
		tdPre = blockchain.GetTdByHash(blockchain.CurrentBlock().Hash())
		if err := testBlockChainImport(blockChainB, blockchain); err != nil {
			t.Fatalf("failed to import forked block chain: %v", err)
		}
		tdPost = blockchain.GetTdByHash(blockChainB[len(blockChainB)-1].Hash())
	} else {
		tdPre = blockchain.GetTdByHash(blockchain.CurrentHeader().Hash())
		if err := testHeaderChainImport(headerChainB, blockchain); err != nil {
			t.Fatalf("failed to import forked header chain: %v", err)
		}
		tdPost = blockchain.GetTdByHash(headerChainB[len(headerChainB)-1].Hash())
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
		err := blockchain.engine.VerifyHeader(blockchain, block.Header(), true)
		if err == nil {
			err = blockchain.validator.ValidateBody(block)
		}
		if err != nil {
			if err == ErrKnownBlock {
				continue
			}
			return err
		}
		statedb, err := state.New(blockchain.GetBlockByHash(block.ParentHash()).Root(), blockchain.stateCache)
		if err != nil {
			return err
		}
		receipts, _, usedGas, err := blockchain.Processor().Process(block, statedb, nil, vm.Config{}, map[common.Address]*big.Int{})
		if err != nil {
			blockchain.reportBlock(block, receipts, err)
			return err
		}
		err = blockchain.validator.ValidateState(block, blockchain.GetBlockByHash(block.ParentHash()), statedb, receipts, usedGas)
		if err != nil {
			blockchain.reportBlock(block, receipts, err)
			return err
		}
		blockchain.chainmu.MustLock()
		rawdb.WriteTd(blockchain.db, block.Hash(), block.NumberU64(), new(big.Int).Add(block.Difficulty(), blockchain.GetTdByHash(block.ParentHash())))
		rawdb.WriteBlock(blockchain.db, block)
		statedb.Commit(true)
		blockchain.chainmu.Unlock()
	}
	return nil
}

// testHeaderChainImport tries to process a chain of header, writing them into
// the database if successful.
func testHeaderChainImport(chain []*types.Header, blockchain *BlockChain) error {
	for _, header := range chain {
		// Try and validate the header
		if err := blockchain.engine.VerifyHeader(blockchain, header, false); err != nil {
			return err
		}
		// Manually insert the header into the database, but don't reorganise (allows subsequent testing)
		blockchain.chainmu.MustLock()
		rawdb.WriteTd(blockchain.db, header.Hash(), header.Number.Uint64(), new(big.Int).Add(header.Difficulty, blockchain.GetTdByHash(header.ParentHash)))
		rawdb.WriteHeader(blockchain.db, header)
		blockchain.chainmu.Unlock()
	}
	return nil
}

func insertChain(done chan bool, blockchain *BlockChain, chain types.Blocks, t *testing.T) {
	_, err := blockchain.InsertChain(chain)
	if err != nil {
		t.Fatal(err)
	}
	done <- true
}

func TestLastBlock(t *testing.T) {
	_, blockchain, err := newCanonical(ethash.NewFaker(), 0, true)
	if err != nil {
		t.Fatalf("failed to create pristine chain: %v", err)
	}
	defer blockchain.Stop()

	blocks := makeBlockChain(blockchain.CurrentBlock(), 1, ethash.NewFullFaker(), blockchain.db, 0)
	if _, err := blockchain.InsertChain(blocks); err != nil {
		t.Fatalf("Failed to insert block: %v", err)
	}
	if blocks[len(blocks)-1].Hash() != rawdb.ReadHeadBlockHash(blockchain.db) {
		t.Fatalf("Write/Get HeadBlockHash failed")
	}
}

// Tests that given a starting canonical chain of a given size, it can be extended
// with various length chains.
func TestExtendCanonicalHeaders(t *testing.T) { testExtendCanonical(t, false) }
func TestExtendCanonicalBlocks(t *testing.T)  { testExtendCanonical(t, true) }

func testExtendCanonical(t *testing.T, full bool) {
	length := 5

	// Make first chain starting from genesis
	_, processor, err := newCanonical(ethash.NewFaker(), length, full)
	if err != nil {
		t.Fatalf("failed to make new canonical chain: %v", err)
	}
	defer processor.Stop()

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
	_, processor, err := newCanonical(ethash.NewFaker(), length, full)
	if err != nil {
		t.Fatalf("failed to make new canonical chain: %v", err)
	}
	defer processor.Stop()

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
	_, processor, err := newCanonical(ethash.NewFaker(), length, full)
	if err != nil {
		t.Fatalf("failed to make new canonical chain: %v", err)
	}
	defer processor.Stop()

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
	_, processor, err := newCanonical(ethash.NewFaker(), length, full)
	if err != nil {
		t.Fatalf("failed to make new canonical chain: %v", err)
	}
	defer processor.Stop()

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
	db, blockchain, err := newCanonical(ethash.NewFaker(), 10, full)
	if err != nil {
		t.Fatalf("failed to make new canonical chain: %v", err)
	}
	defer blockchain.Stop()

	// Create a forked chain, and try to insert with a missing link
	if full {
		chain := makeBlockChain(blockchain.CurrentBlock(), 5, ethash.NewFaker(), db, forkSeed)[1:]
		if err := testBlockChainImport(chain, blockchain); err == nil {
			t.Errorf("broken block chain not reported")
		}
	} else {
		chain := makeHeaderChain(blockchain.CurrentHeader(), 5, ethash.NewFaker(), db, forkSeed)[1:]
		if err := testHeaderChainImport(chain, blockchain); err == nil {
			t.Errorf("broken header chain not reported")
		}
	}
}

// Tests that reorganising a long difficult chain after a short easy one
// overwrites the canonical numbers and links in the database.
func TestReorgLongHeaders(t *testing.T) { testReorgLong(t, false) }
func TestReorgLongBlocks(t *testing.T)  { testReorgLong(t, true) }

func testReorgLong(t *testing.T, full bool) {
	testReorg(t, []int64{0, 0, -9}, []int64{0, 0, 0, -9}, 393280, full)
}

// Tests that reorganising a short difficult chain after a long easy one
// overwrites the canonical numbers and links in the database.
func TestReorgShortHeaders(t *testing.T) { testReorgShort(t, false) }
func TestReorgShortBlocks(t *testing.T)  { testReorgShort(t, true) }

func testReorgShort(t *testing.T, full bool) {
	// Create a long easy chain vs. a short heavy one. Due to difficulty adjustment
	// we need a fairly long chain of blocks with different difficulties for a short
	// one to become heavyer than a long one. The 96 is an empirical value.
	easy := make([]int64, 96)
	for i := 0; i < len(easy); i++ {
		easy[i] = 60
	}
	diff := make([]int64, len(easy)-1)
	for i := 0; i < len(diff); i++ {
		diff[i] = -9
	}
	if full {
		testReorg(t, easy, diff, 12451840, full)
	} else {
		testReorg(t, easy, diff, 12615120, full)
	}
}

func testReorg(t *testing.T, first, second []int64, td int64, full bool) {
	// Create a pristine chain and database
	db, blockchain, err := newCanonical(ethash.NewFaker(), 0, full)
	if err != nil {
		t.Fatalf("failed to create pristine chain: %v", err)
	}
	defer blockchain.Stop()

	// Insert an easy and a difficult chain afterwards
	easyBlocks, _ := GenerateChain(params.TestChainConfig, blockchain.CurrentBlock(), ethash.NewFaker(), db, len(first), func(i int, b *BlockGen) {
		b.OffsetTime(first[i])
	})
	diffBlocks, _ := GenerateChain(params.TestChainConfig, blockchain.CurrentBlock(), ethash.NewFaker(), db, len(second), func(i int, b *BlockGen) {
		b.OffsetTime(second[i])
	})
	if full {
		if _, err := blockchain.InsertChain(easyBlocks); err != nil {
			t.Fatalf("failed to insert easy chain: %v", err)
		}
		if _, err := blockchain.InsertChain(diffBlocks); err != nil {
			t.Fatalf("failed to insert difficult chain: %v", err)
		}
	} else {
		easyHeaders := make([]*types.Header, len(easyBlocks))
		for i, block := range easyBlocks {
			easyHeaders[i] = block.Header()
		}
		diffHeaders := make([]*types.Header, len(diffBlocks))
		for i, block := range diffBlocks {
			diffHeaders[i] = block.Header()
		}
		if _, err := blockchain.InsertHeaderChain(easyHeaders, 1); err != nil {
			t.Fatalf("failed to insert easy chain: %v", err)
		}
		if _, err := blockchain.InsertHeaderChain(diffHeaders, 1); err != nil {
			t.Fatalf("failed to insert difficult chain: %v", err)
		}
	}
	// Check that the chain is valid number and link wise
	if full {
		prev := blockchain.CurrentBlock()
		for block := blockchain.GetBlockByNumber(blockchain.CurrentBlock().NumberU64() - 1); block.NumberU64() != 0; prev, block = block, blockchain.GetBlockByNumber(block.NumberU64()-1) {
			if prev.ParentHash() != block.Hash() {
				t.Errorf("parent block hash mismatch: have %x, want %x", prev.ParentHash(), block.Hash())
			}
		}
	} else {
		prev := blockchain.CurrentHeader()
		for header := blockchain.GetHeaderByNumber(blockchain.CurrentHeader().Number.Uint64() - 1); header.Number.Uint64() != 0; prev, header = header, blockchain.GetHeaderByNumber(header.Number.Uint64()-1) {
			if prev.ParentHash != header.Hash() {
				t.Errorf("parent header hash mismatch: have %x, want %x", prev.ParentHash, header.Hash())
			}
		}
	}
}

// Tests that the insertion functions detect banned hashes.
func TestBadHeaderHashes(t *testing.T) { testBadHashes(t, false) }
func TestBadBlockHashes(t *testing.T)  { testBadHashes(t, true) }

func testBadHashes(t *testing.T, full bool) {
	// Create a pristine chain and database
	db, blockchain, err := newCanonical(ethash.NewFaker(), 0, full)
	if err != nil {
		t.Fatalf("failed to create pristine chain: %v", err)
	}
	defer blockchain.Stop()

	// Create a chain, ban a hash and try to import
	if full {
		blocks := makeBlockChain(blockchain.CurrentBlock(), 3, ethash.NewFaker(), db, 10)

		BadHashes[blocks[2].Header().Hash()] = true
		defer func() { delete(BadHashes, blocks[2].Header().Hash()) }()

		_, err = blockchain.InsertChain(blocks)
	} else {
		headers := makeHeaderChain(blockchain.CurrentHeader(), 3, ethash.NewFaker(), db, 10)

		BadHashes[headers[2].Hash()] = true
		defer func() { delete(BadHashes, headers[2].Hash()) }()

		_, err = blockchain.InsertHeaderChain(headers, 1)
	}
	if !errors.Is(err, ErrBlacklistedHash) {
		t.Errorf("error mismatch: have: %v, want: %v", err, ErrBlacklistedHash)
	}
}

// Tests that bad hashes are detected on boot, and the chain rolled back to a
// good state prior to the bad hash.
func TestReorgBadHeaderHashes(t *testing.T) { testReorgBadHashes(t, false) }
func TestReorgBadBlockHashes(t *testing.T)  { testReorgBadHashes(t, true) }

func testReorgBadHashes(t *testing.T, full bool) {
	// Create a pristine chain and database
	db, blockchain, err := newCanonical(ethash.NewFaker(), 0, full)
	if err != nil {
		t.Fatalf("failed to create pristine chain: %v", err)
	}
	// Create a chain, import and ban afterwards
	headers := makeHeaderChain(blockchain.CurrentHeader(), 4, ethash.NewFaker(), db, 10)
	blocks := makeBlockChain(blockchain.CurrentBlock(), 4, ethash.NewFaker(), db, 10)

	if full {
		if _, err = blockchain.InsertChain(blocks); err != nil {
			t.Errorf("failed to import blocks: %v", err)
		}
		if blockchain.CurrentBlock().Hash() != blocks[3].Hash() {
			t.Errorf("last block hash mismatch: have: %x, want %x", blockchain.CurrentBlock().Hash(), blocks[3].Header().Hash())
		}
		BadHashes[blocks[3].Header().Hash()] = true
		defer func() { delete(BadHashes, blocks[3].Header().Hash()) }()
	} else {
		if _, err = blockchain.InsertHeaderChain(headers, 1); err != nil {
			t.Errorf("failed to import headers: %v", err)
		}
		if blockchain.CurrentHeader().Hash() != headers[3].Hash() {
			t.Errorf("last header hash mismatch: have: %x, want %x", blockchain.CurrentHeader().Hash(), headers[3].Hash())
		}
		BadHashes[headers[3].Hash()] = true
		defer func() { delete(BadHashes, headers[3].Hash()) }()
	}
	blockchain.Stop()

	// Create a new BlockChain and check that it rolled back the state.
	ncm, err := NewBlockChain(blockchain.db, nil, blockchain.chainConfig, ethash.NewFaker(), vm.Config{})
	if err != nil {
		t.Fatalf("failed to create new chain manager: %v", err)
	}
	if full {
		if ncm.CurrentBlock().Hash() != blocks[2].Header().Hash() {
			t.Errorf("last block hash mismatch: have: %x, want %x", ncm.CurrentBlock().Hash(), blocks[2].Header().Hash())
		}
		if blocks[2].Header().GasLimit != ncm.GasLimit() {
			t.Errorf("last  block gasLimit mismatch: have: %d, want %d", ncm.GasLimit(), blocks[2].Header().GasLimit)
		}
	} else {
		if ncm.CurrentHeader().Hash() != headers[2].Hash() {
			t.Errorf("last header hash mismatch: have: %x, want %x", ncm.CurrentHeader().Hash(), headers[2].Hash())
		}
	}
	ncm.Stop()
}

// Tests chain insertions in the face of one entity containing an invalid nonce.
func TestHeadersInsertNonceError(t *testing.T) { testInsertNonceError(t, false) }
func TestBlocksInsertNonceError(t *testing.T)  { testInsertNonceError(t, true) }

func testInsertNonceError(t *testing.T, full bool) {
	for i := 1; i < 25 && !t.Failed(); i++ {
		// Create a pristine chain and database
		db, blockchain, err := newCanonical(ethash.NewFaker(), 0, full)
		if err != nil {
			t.Fatalf("failed to create pristine chain: %v", err)
		}
		defer blockchain.Stop()

		// Create and insert a chain with a failing nonce
		var (
			failAt  int
			failRes int
			failNum uint64
		)
		headers := makeHeaderChain(blockchain.CurrentHeader(), i, ethash.NewFaker(), db, 0)

		failAt = rand.Int() % len(headers)
		failNum = headers[failAt].Number.Uint64()

		blockchain.engine = ethash.NewFakeFailer(failNum)
		blockchain.hc.engine = blockchain.engine
		failRes, _ = blockchain.InsertHeaderChain(headers, 1)
		// Check that the returned error indicates the failure.
		if failRes != failAt {
			t.Errorf("test %d: failure index mismatch: have %d, want %d", i, failRes, failAt)
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
		gendb   = rawdb.NewMemoryDatabase()
		key, _  = crypto.HexToECDSA("b71c71a67e1177ad4e901695e1b4b9ee17ae16c6668d313eac2f96dbcda3f291")
		address = crypto.PubkeyToAddress(key.PublicKey)
		funds   = big.NewInt(1000000000000000)
		gspec   = &Genesis{
			Config:  params.TestChainConfig,
			Alloc:   types.GenesisAlloc{address: {Balance: funds}},
			BaseFee: big.NewInt(params.InitialBaseFee),
		}
		genesis = gspec.MustCommit(gendb)
		signer  = types.LatestSigner(gspec.Config)
	)
	blocks, receipts := GenerateChain(gspec.Config, genesis, ethash.NewFaker(), gendb, 1024, func(i int, block *BlockGen) {
		block.SetCoinbase(common.Address{0x00})

		// If the block number is multiple of 3, send a few bonus transactions to the miner
		if i%3 == 2 {
			for j := 0; j < i%4+1; j++ {
				tx, err := types.SignTx(types.NewTransaction(block.TxNonce(address), common.Address{0x00}, big.NewInt(1000), params.TxGas, block.header.BaseFee, nil), signer, key)
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
	archiveDb := rawdb.NewMemoryDatabase()
	gspec.MustCommit(archiveDb)
	archive, _ := NewBlockChain(archiveDb, nil, gspec.Config, ethash.NewFaker(), vm.Config{})
	defer archive.Stop()

	if n, err := archive.InsertChain(blocks); err != nil {
		t.Fatalf("failed to process block %d: %v", n, err)
	}
	// Fast import the chain as a non-archive node to test
	fastDb := rawdb.NewMemoryDatabase()
	gspec.MustCommit(fastDb)
	fast, _ := NewBlockChain(fastDb, nil, gspec.Config, ethash.NewFaker(), vm.Config{})
	defer fast.Stop()

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

		if ftd, atd := fast.GetTdByHash(hash), archive.GetTdByHash(hash); ftd.Cmp(atd) != 0 {
			t.Errorf("block #%d [%x]: td mismatch: have %v, want %v", num, hash, ftd, atd)
		}
		if fheader, aheader := fast.GetHeaderByHash(hash), archive.GetHeaderByHash(hash); fheader.Hash() != aheader.Hash() {
			t.Errorf("block #%d [%x]: header mismatch: have %v, want %v", num, hash, fheader, aheader)
		}
		if fblock, ablock := fast.GetBlockByHash(hash), archive.GetBlockByHash(hash); fblock.Hash() != ablock.Hash() {
			t.Errorf("block #%d [%x]: block mismatch: have %v, want %v", num, hash, fblock, ablock)
		} else if types.DeriveSha(fblock.Transactions(), new(trie.Trie)) != types.DeriveSha(ablock.Transactions(), new(trie.Trie)) {
			t.Errorf("block #%d [%x]: transactions mismatch: have %v, want %v", num, hash, fblock.Transactions(), ablock.Transactions())
		} else if types.CalcUncleHash(fblock.Uncles()) != types.CalcUncleHash(ablock.Uncles()) {
			t.Errorf("block #%d [%x]: uncles mismatch: have %v, want %v", num, hash, fblock.Uncles(), ablock.Uncles())
		}
		if freceipts, areceipts := rawdb.ReadReceipts(fastDb, hash, *rawdb.ReadHeaderNumber(fastDb, hash), fast.Config()), rawdb.ReadReceipts(archiveDb, hash, *rawdb.ReadHeaderNumber(archiveDb, hash), fast.Config()); types.DeriveSha(freceipts, new(trie.Trie)) != types.DeriveSha(areceipts, new(trie.Trie)) {
			t.Errorf("block #%d [%x]: receipts mismatch: have %v, want %v", num, hash, freceipts, areceipts)
		}
	}
	// Check that the canonical chains are the same between the databases
	for i := 0; i < len(blocks)+1; i++ {
		if fhash, ahash := rawdb.ReadCanonicalHash(fastDb, uint64(i)), rawdb.ReadCanonicalHash(archiveDb, uint64(i)); fhash != ahash {
			t.Errorf("block #%d: canonical hash mismatch: have %v, want %v", i, fhash, ahash)
		}
	}
}

// Tests that various import methods move the chain head pointers to the correct
// positions.
func TestLightVsFastVsFullChainHeads(t *testing.T) {
	// Configure and generate a sample block chain
	var (
		gendb   = rawdb.NewMemoryDatabase()
		key, _  = crypto.HexToECDSA("b71c71a67e1177ad4e901695e1b4b9ee17ae16c6668d313eac2f96dbcda3f291")
		address = crypto.PubkeyToAddress(key.PublicKey)
		funds   = big.NewInt(1000000000000000)
		gspec   = &Genesis{
			Config:  params.TestChainConfig,
			Alloc:   types.GenesisAlloc{address: {Balance: funds}},
			BaseFee: big.NewInt(params.InitialBaseFee),
		}
		genesis = gspec.MustCommit(gendb)
	)
	height := uint64(1024)
	blocks, receipts := GenerateChain(gspec.Config, genesis, ethash.NewFaker(), gendb, int(height), nil)

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
	archiveDb := rawdb.NewMemoryDatabase()
	gspec.MustCommit(archiveDb)

	archive, _ := NewBlockChain(archiveDb, nil, gspec.Config, ethash.NewFaker(), vm.Config{})
	if n, err := archive.InsertChain(blocks); err != nil {
		t.Fatalf("failed to process block %d: %v", n, err)
	}
	defer archive.Stop()

	assert(t, "archive", archive, height, height, height)
	archive.Rollback(remove)
	assert(t, "archive", archive, height/2, height/2, height/2)

	// Import the chain as a non-archive node and ensure all pointers are updated
	fastDb := rawdb.NewMemoryDatabase()
	gspec.MustCommit(fastDb)
	fast, _ := NewBlockChain(fastDb, nil, gspec.Config, ethash.NewFaker(), vm.Config{})
	defer fast.Stop()

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
	lightDb := rawdb.NewMemoryDatabase()
	gspec.MustCommit(lightDb)

	light, _ := NewBlockChain(lightDb, nil, gspec.Config, ethash.NewFaker(), vm.Config{})
	if n, err := light.InsertHeaderChain(headers, 1); err != nil {
		t.Fatalf("failed to insert header %d: %v", n, err)
	}
	defer light.Stop()

	assert(t, "light", light, height, 0, 0)
	light.Rollback(remove)
	assert(t, "light", light, height/2, 0, 0)
}

// Tests that chain reorganisations handle transaction removals and reinsertions.
func TestChainTxReorgs(t *testing.T) {
	var (
		key1, _ = crypto.HexToECDSA("b71c71a67e1177ad4e901695e1b4b9ee17ae16c6668d313eac2f96dbcda3f291")
		key2, _ = crypto.HexToECDSA("8a1f9a8f95be41cd7ccb6168179afb4504aefe388d1e14474d32c45c72ce7b7a")
		key3, _ = crypto.HexToECDSA("49a7b37aa6f6645917e7b807e9d1c00d4fa71f18343b0d4122a4d2df64dd6fee")
		addr1   = crypto.PubkeyToAddress(key1.PublicKey)
		addr2   = crypto.PubkeyToAddress(key2.PublicKey)
		addr3   = crypto.PubkeyToAddress(key3.PublicKey)
		db      = rawdb.NewMemoryDatabase()
		gspec   = &Genesis{
			Config:   params.TestChainConfig,
			GasLimit: 3141592,
			Alloc: types.GenesisAlloc{
				addr1: {Balance: big.NewInt(1000000000000000)},
				addr2: {Balance: big.NewInt(1000000000000000)},
				addr3: {Balance: big.NewInt(1000000000000000)},
			},
		}
		genesis = gspec.MustCommit(db)
		signer  = types.LatestSigner(gspec.Config)
	)

	// Create two transactions shared between the chains:
	//  - postponed: transaction included at a later block in the forked chain
	//  - swapped: transaction included at the same block number in the forked chain
	postponed, _ := types.SignTx(types.NewTransaction(0, addr1, big.NewInt(1000), params.TxGas, big.NewInt(params.InitialBaseFee), nil), signer, key1)
	swapped, _ := types.SignTx(types.NewTransaction(1, addr1, big.NewInt(1000), params.TxGas, big.NewInt(params.InitialBaseFee), nil), signer, key1)

	// Create two transactions that will be dropped by the forked chain:
	//  - pastDrop: transaction dropped retroactively from a past block
	//  - freshDrop: transaction dropped exactly at the block where the reorg is detected
	var pastDrop, freshDrop *types.Transaction

	// Create three transactions that will be added in the forked chain:
	//  - pastAdd:   transaction added before the reorganization is detected
	//  - freshAdd:  transaction added at the exact block the reorg is detected
	//  - futureAdd: transaction added after the reorg has already finished
	var pastAdd, freshAdd, futureAdd *types.Transaction

	chain, _ := GenerateChain(gspec.Config, genesis, ethash.NewFaker(), db, 3, func(i int, gen *BlockGen) {
		switch i {
		case 0:
			pastDrop, _ = types.SignTx(types.NewTransaction(gen.TxNonce(addr2), addr2, big.NewInt(1000), params.TxGas, gen.header.BaseFee, nil), signer, key2)

			gen.AddTx(pastDrop)  // This transaction will be dropped in the fork from below the split point
			gen.AddTx(postponed) // This transaction will be postponed till block #3 in the fork

		case 2:
			freshDrop, _ = types.SignTx(types.NewTransaction(gen.TxNonce(addr2), addr2, big.NewInt(1000), params.TxGas, gen.header.BaseFee, nil), signer, key2)

			gen.AddTx(freshDrop) // This transaction will be dropped in the fork from exactly at the split point
			gen.AddTx(swapped)   // This transaction will be swapped out at the exact height

			gen.OffsetTime(9) // Lower the block difficulty to simulate a weaker chain
		}
	})
	// Import the chain. This runs all block validation rules.
	blockchain, _ := NewBlockChain(db, nil, gspec.Config, ethash.NewFaker(), vm.Config{})
	if i, err := blockchain.InsertChain(chain); err != nil {
		t.Fatalf("failed to insert original chain[%d]: %v", i, err)
	}
	defer blockchain.Stop()

	// overwrite the old chain
	chain, _ = GenerateChain(gspec.Config, genesis, ethash.NewFaker(), db, 5, func(i int, gen *BlockGen) {
		switch i {
		case 0:
			pastAdd, _ = types.SignTx(types.NewTransaction(gen.TxNonce(addr3), addr3, big.NewInt(1000), params.TxGas, gen.header.BaseFee, nil), signer, key3)
			gen.AddTx(pastAdd) // This transaction needs to be injected during reorg

		case 2:
			gen.AddTx(postponed) // This transaction was postponed from block #1 in the original chain
			gen.AddTx(swapped)   // This transaction was swapped from the exact current spot in the original chain

			freshAdd, _ = types.SignTx(types.NewTransaction(gen.TxNonce(addr3), addr3, big.NewInt(1000), params.TxGas, gen.header.BaseFee, nil), signer, key3)
			gen.AddTx(freshAdd) // This transaction will be added exactly at reorg time

		case 3:
			futureAdd, _ = types.SignTx(types.NewTransaction(gen.TxNonce(addr3), addr3, big.NewInt(1000), params.TxGas, gen.header.BaseFee, nil), signer, key3)
			gen.AddTx(futureAdd) // This transaction will be added after a full reorg
		}
	})
	if _, err := blockchain.InsertChain(chain); err != nil {
		t.Fatalf("failed to insert forked chain: %v", err)
	}

	// removed tx
	for i, tx := range (types.Transactions{pastDrop, freshDrop}) {
		if txn, _, _, _ := rawdb.ReadTransaction(db, tx.Hash()); txn != nil {
			t.Errorf("drop %d: tx %v found while shouldn't have been", i, txn)
		}
		if rcpt, _, _, _ := rawdb.ReadReceipt(db, tx.Hash(), blockchain.Config()); rcpt != nil {
			t.Errorf("drop %d: receipt %v found while shouldn't have been", i, rcpt)
		}
	}
	// added tx
	for i, tx := range (types.Transactions{pastAdd, freshAdd, futureAdd}) {
		if txn, _, _, _ := rawdb.ReadTransaction(db, tx.Hash()); txn == nil {
			t.Errorf("add %d: expected tx to be found", i)
		}
		if rcpt, _, _, _ := rawdb.ReadReceipt(db, tx.Hash(), blockchain.Config()); rcpt == nil {
			t.Errorf("add %d: expected receipt to be found", i)
		}
	}
	// shared tx
	for i, tx := range (types.Transactions{postponed, swapped}) {
		if txn, _, _, _ := rawdb.ReadTransaction(db, tx.Hash()); txn == nil {
			t.Errorf("share %d: expected tx to be found", i)
		}
		if rcpt, _, _, _ := rawdb.ReadReceipt(db, tx.Hash(), blockchain.Config()); rcpt == nil {
			t.Errorf("share %d: expected receipt to be found", i)
		}
	}
}

func TestLogReorgs(t *testing.T) {

	var (
		key1, _ = crypto.HexToECDSA("b71c71a67e1177ad4e901695e1b4b9ee17ae16c6668d313eac2f96dbcda3f291")
		addr1   = crypto.PubkeyToAddress(key1.PublicKey)
		db      = rawdb.NewMemoryDatabase()
		// this code generates a log
		code    = common.Hex2Bytes("60606040525b7f24ec1d3ff24c2f6ff210738839dbc339cd45a5294d85c79361016243157aae7b60405180905060405180910390a15b600a8060416000396000f360606040526008565b00")
		gspec   = &Genesis{Config: params.TestChainConfig, Alloc: types.GenesisAlloc{addr1: {Balance: big.NewInt(10000000000000000)}}}
		genesis = gspec.MustCommit(db)
		signer  = types.LatestSigner(gspec.Config)
	)

	blockchain, _ := NewBlockChain(db, nil, gspec.Config, ethash.NewFaker(), vm.Config{})
	defer blockchain.Stop()

	rmLogsCh := make(chan RemovedLogsEvent)
	blockchain.SubscribeRemovedLogsEvent(rmLogsCh)
	chain, _ := GenerateChain(params.TestChainConfig, genesis, ethash.NewFaker(), db, 2, func(i int, gen *BlockGen) {
		if i == 1 {
			tx, err := types.SignTx(types.NewContractCreation(gen.TxNonce(addr1), new(big.Int), 1000000, gen.header.BaseFee, code), signer, key1)
			if err != nil {
				t.Fatalf("failed to create tx: %v", err)
			}
			gen.AddTx(tx)
		}
	})
	if _, err := blockchain.InsertChain(chain); err != nil {
		t.Fatalf("failed to insert chain: %v", err)
	}

	chain, _ = GenerateChain(params.TestChainConfig, genesis, ethash.NewFaker(), db, 3, func(i int, gen *BlockGen) {})
	if _, err := blockchain.InsertChain(chain); err != nil {
		t.Fatalf("failed to insert forked chain: %v", err)
	}

	timeout := time.NewTimer(1 * time.Second)
	select {
	case ev := <-rmLogsCh:
		if len(ev.Logs) == 0 {
			t.Error("expected logs")
		}
	case <-timeout.C:
		t.Fatal("Timeout. There is no RemovedLogsEvent has been sent.")
	}
}

//func TestReorgSideEvent(t *testing.T) {
//	var (
//		db, _   = rawdb.NewMemoryDatabase()
//		key1, _ = crypto.HexToECDSA("b71c71a67e1177ad4e901695e1b4b9ee17ae16c6668d313eac2f96dbcda3f291")
//		addr1   = crypto.PubkeyToAddress(key1.PublicKey)
//		gspec   = &Genesis{
//			Config: params.TestChainConfig,
//			Alloc:  GenesisAlloc{addr1: {Balance: big.NewInt(10000000000000000)}},
//		}
//		genesis = gspec.MustCommit(db)
//		signer  = types.LatestSigner(gspec.Config)
//	)
//
//	blockchain, _ := NewBlockChain(db, nil, gspec.Config, ethash.NewFaker(), vm.Config{})
//	defer blockchain.Stop()
//
//	chain, _ := GenerateChain(gspec.Config, genesis, ethash.NewFaker(), db, 3, func(i int, gen *BlockGen) {})
//	if _, err := blockchain.InsertChain(chain); err != nil {
//		t.Fatalf("failed to insert chain: %v", err)
//	}
//
//	replacementBlocks, _ := GenerateChain(gspec.Config, genesis, ethash.NewFaker(), db, 4, func(i int, gen *BlockGen) {
//		tx, err := types.SignTx(types.NewContractCreation(gen.TxNonce(addr1), new(big.Int), 1000000, gen.header.BaseFee, nil), signer, key1)
//		if i == 2 {
//			gen.OffsetTime(-9)
//		}
//		if err != nil {
//			t.Fatalf("failed to create tx: %v", err)
//		}
//		gen.AddTx(tx)
//	})
//	chainSideCh := make(chan ChainSideEvent, 64)
//	blockchain.SubscribeChainSideEvent(chainSideCh)
//	if _, err := blockchain.InsertChain(replacementBlocks); err != nil {
//		t.Fatalf("failed to insert chain: %v", err)
//	}
//
//	// first two block of the secondary chain are for a brief moment considered
//	// side chains because up to that point the first one is considered the
//	// heavier chain.
//	expectedSideHashes := map[common.Hash]bool{
//		replacementBlocks[0].Hash(): true,
//		replacementBlocks[1].Hash(): true,
//		chain[0].Hash():             true,
//		chain[1].Hash():             true,
//		chain[2].Hash():             true,
//	}
//
//	i := 0
//
//	const timeoutDura = 10 * time.Second
//	timeout := time.NewTimer(timeoutDura)
//done:
//	for {
//		select {
//		case ev := <-chainSideCh:
//			block := ev.Block
//			if _, ok := expectedSideHashes[block.Hash()]; !ok {
//				t.Errorf("%d: didn't expect %x to be in side chain", i, block.Hash())
//			}
//			i++
//
//			if i == len(expectedSideHashes) {
//				timeout.Stop()
//
//				break done
//			}
//			timeout.Reset(timeoutDura)
//
//		case <-timeout.C:
//			t.Fatal("Timeout. Possibly not all blocks were triggered for sideevent")
//		}
//	}
//
//	// make sure no more events are fired
//	select {
//	case e := <-chainSideCh:
//		t.Errorf("unexpected event fired: %v", e)
//	case <-time.After(250 * time.Millisecond):
//	}
//
//}

// Tests if the canonical block can be fetched from the database during chain insertion.
func TestCanonicalBlockRetrieval(t *testing.T) {
	_, blockchain, err := newCanonical(ethash.NewFaker(), 0, true)
	if err != nil {
		t.Fatalf("failed to create pristine chain: %v", err)
	}
	defer blockchain.Stop()

	chain, _ := GenerateChain(blockchain.chainConfig, blockchain.genesisBlock, ethash.NewFaker(), blockchain.db, 10, func(i int, gen *BlockGen) {})

	var pend sync.WaitGroup
	pend.Add(len(chain))

	for i := range chain {
		go func(block *types.Block) {
			defer pend.Done()

			// try to retrieve a block by its canonical hash and see if the block data can be retrieved.
			for {
				ch := rawdb.ReadCanonicalHash(blockchain.db, block.NumberU64())
				if ch == (common.Hash{}) {
					continue // busy wait for canonical hash to be written
				}
				if ch != block.Hash() {
					t.Errorf("unknown canonical hash, want %s, got %s", block.Hash().Hex(), ch.Hex())
					return
				}
				fb := rawdb.ReadBlock(blockchain.db, ch, block.NumberU64())
				if fb == nil {
					t.Errorf("unable to retrieve block %d for canonical hash: %s", block.NumberU64(), ch.Hex())
					return
				}
				if fb.Hash() != block.Hash() {
					t.Errorf("invalid block hash for block %d, want %s, got %s", block.NumberU64(), block.Hash().Hex(), fb.Hash().Hex())
					return
				}
				return
			}
		}(chain[i])

		if _, err := blockchain.InsertChain(types.Blocks{chain[i]}); err != nil {
			t.Fatalf("failed to insert block %d: %v", i, err)
		}
	}
	pend.Wait()
}

func TestEIP155Transition(t *testing.T) {
	// Configure and generate a sample block chain
	var (
		db         = rawdb.NewMemoryDatabase()
		key, _     = crypto.HexToECDSA("b71c71a67e1177ad4e901695e1b4b9ee17ae16c6668d313eac2f96dbcda3f291")
		address    = crypto.PubkeyToAddress(key.PublicKey)
		funds      = big.NewInt(1000000000)
		deleteAddr = common.Address{1}
		gspec      = &Genesis{
			Config: &params.ChainConfig{ChainId: big.NewInt(1), EIP150Block: big.NewInt(0), EIP155Block: big.NewInt(2), HomesteadBlock: new(big.Int)},
			Alloc:  types.GenesisAlloc{address: {Balance: funds}, deleteAddr: {Balance: new(big.Int)}},
		}
		genesis = gspec.MustCommit(db)
	)

	blockchain, _ := NewBlockChain(db, nil, gspec.Config, ethash.NewFaker(), vm.Config{})
	defer blockchain.Stop()

	blocks, _ := GenerateChain(gspec.Config, genesis, ethash.NewFaker(), db, 4, func(i int, block *BlockGen) {
		var (
			tx      *types.Transaction
			err     error
			basicTx = func(signer types.Signer) (*types.Transaction, error) {
				return types.SignTx(types.NewTransaction(block.TxNonce(address), common.Address{}, new(big.Int), 21000, new(big.Int), nil), signer, key)
			}
		)
		switch i {
		case 0:
			tx, err = basicTx(types.HomesteadSigner{})
			if err != nil {
				t.Fatal(err)
			}
			block.AddTx(tx)
		case 2:
			tx, err = basicTx(types.HomesteadSigner{})
			if err != nil {
				t.Fatal(err)
			}
			block.AddTx(tx)

			tx, err = basicTx(types.LatestSigner(gspec.Config))
			if err != nil {
				t.Fatal(err)
			}
			block.AddTx(tx)
		case 3:
			tx, err = basicTx(types.HomesteadSigner{})
			if err != nil {
				t.Fatal(err)
			}
			block.AddTx(tx)

			tx, err = basicTx(types.LatestSigner(gspec.Config))
			if err != nil {
				t.Fatal(err)
			}
			block.AddTx(tx)
		}
	})

	if _, err := blockchain.InsertChain(blocks); err != nil {
		t.Fatal(err)
	}
	block := blockchain.GetBlockByNumber(1)
	if block.Transactions()[0].Protected() {
		t.Error("Expected block[0].txs[0] to not be replay protected")
	}

	block = blockchain.GetBlockByNumber(3)
	if block.Transactions()[0].Protected() {
		t.Error("Expected block[3].txs[0] to not be replay protected")
	}
	if !block.Transactions()[1].Protected() {
		t.Error("Expected block[3].txs[1] to be replay protected")
	}
	if _, err := blockchain.InsertChain(blocks[4:]); err != nil {
		t.Fatal(err)
	}

	// generate an invalid chain id transaction
	config := &params.ChainConfig{ChainId: big.NewInt(2), EIP150Block: big.NewInt(0), EIP155Block: big.NewInt(2), HomesteadBlock: new(big.Int)}
	blocks, _ = GenerateChain(config, blocks[len(blocks)-1], ethash.NewFaker(), db, 4, func(i int, block *BlockGen) {
		var (
			tx      *types.Transaction
			err     error
			basicTx = func(signer types.Signer) (*types.Transaction, error) {
				return types.SignTx(types.NewTransaction(block.TxNonce(address), common.Address{}, new(big.Int), 21000, new(big.Int), nil), signer, key)
			}
		)
		if i == 0 {
			tx, err = basicTx(types.LatestSigner(config))
			if err != nil {
				t.Fatal(err)
			}
			block.AddTx(tx)
		}
	})
	_, err := blockchain.InsertChain(blocks)
	if have, want := err, types.ErrInvalidChainId; !errors.Is(have, want) {
		t.Errorf("have %v, want %v", have, want)
	}
}

func TestEIP161AccountRemoval(t *testing.T) {
	// Configure and generate a sample block chain
	var (
		db      = rawdb.NewMemoryDatabase()
		key, _  = crypto.HexToECDSA("b71c71a67e1177ad4e901695e1b4b9ee17ae16c6668d313eac2f96dbcda3f291")
		address = crypto.PubkeyToAddress(key.PublicKey)
		funds   = big.NewInt(1000000000)
		theAddr = common.Address{1}
		gspec   = &Genesis{
			Config: &params.ChainConfig{
				ChainId:        big.NewInt(1),
				HomesteadBlock: new(big.Int),
				EIP155Block:    new(big.Int),
				EIP158Block:    big.NewInt(2),
			},
			Alloc: types.GenesisAlloc{address: {Balance: funds}},
		}
		genesis = gspec.MustCommit(db)
	)
	blockchain, _ := NewBlockChain(db, nil, gspec.Config, ethash.NewFaker(), vm.Config{})
	defer blockchain.Stop()

	blocks, _ := GenerateChain(gspec.Config, genesis, ethash.NewFaker(), db, 3, func(i int, block *BlockGen) {
		var (
			tx     *types.Transaction
			err    error
			signer = types.LatestSigner(gspec.Config)
		)
		switch i {
		case 0:
			tx, err = types.SignTx(types.NewTransaction(block.TxNonce(address), theAddr, new(big.Int), 21000, new(big.Int), nil), signer, key)
		case 1:
			tx, err = types.SignTx(types.NewTransaction(block.TxNonce(address), theAddr, new(big.Int), 21000, new(big.Int), nil), signer, key)
		case 2:
			tx, err = types.SignTx(types.NewTransaction(block.TxNonce(address), theAddr, new(big.Int), 21000, new(big.Int), nil), signer, key)
		}
		if err != nil {
			t.Fatal(err)
		}
		block.AddTx(tx)
	})
	// account must exist pre eip 161
	if _, err := blockchain.InsertChain(types.Blocks{blocks[0]}); err != nil {
		t.Fatal(err)
	}
	if st, _ := blockchain.State(); !st.Exist(theAddr) {
		t.Error("expected account to exist")
	}

	// account needs to be deleted post eip 161
	if _, err := blockchain.InsertChain(types.Blocks{blocks[1]}); err != nil {
		t.Fatal(err)
	}
	if st, _ := blockchain.State(); st.Exist(theAddr) {
		t.Error("account should not exist")
	}

	// account mustn't be created post eip 161
	if _, err := blockchain.InsertChain(types.Blocks{blocks[2]}); err != nil {
		t.Fatal(err)
	}
	if st, _ := blockchain.State(); st.Exist(theAddr) {
		t.Error("account should not exist")
	}
}

// This is a regression test (i.e. as weird as it is, don't delete it ever), which
// tests that under weird reorg conditions the blockchain and its internal header-
// chain return the same latest block/header.
//
// https://github.com/XinFinOrg/XDPoSChain/pull/15941how Source Control
func TestBlockchainHeaderchainReorgConsistency(t *testing.T) {
	// Generate a canonical chain to act as the main dataset
	engine := ethash.NewFaker()

	db := rawdb.NewMemoryDatabase()
	gspec := &Genesis{
		Config:  params.TestChainConfig,
		BaseFee: big.NewInt(params.InitialBaseFee),
	}
	genesis := gspec.MustCommit(db)
	blocks, _ := GenerateChain(params.TestChainConfig, genesis, engine, db, 64, func(i int, b *BlockGen) { b.SetCoinbase(common.Address{1}) })

	// Generate a bunch of fork blocks, each side forking from the canonical chain
	forks := make([]*types.Block, len(blocks))
	for i := 0; i < len(forks); i++ {
		parent := genesis
		if i > 0 {
			parent = blocks[i-1]
		}
		fork, _ := GenerateChain(params.TestChainConfig, parent, engine, db, 1, func(i int, b *BlockGen) { b.SetCoinbase(common.Address{2}) })
		forks[i] = fork[0]
	}
	// Import the canonical and fork chain side by side, verifying the current block
	// and current header consistency
	diskdb := rawdb.NewMemoryDatabase()
	gspec.MustCommit(diskdb)

	chain, err := NewBlockChain(diskdb, nil, params.TestChainConfig, engine, vm.Config{})
	if err != nil {
		t.Fatalf("failed to create tester chain: %v", err)
	}
	for i := 0; i < len(blocks); i++ {
		if _, err := chain.InsertChain(blocks[i : i+1]); err != nil {
			t.Fatalf("block %d: failed to insert into chain: %v", i, err)
		}
		if chain.CurrentBlock().Hash() != chain.CurrentHeader().Hash() {
			t.Errorf("block %d: current block/header mismatch: block #%d [%x..], header #%d [%x..]", i, chain.CurrentBlock().Number(), chain.CurrentBlock().Hash().Bytes()[:4], chain.CurrentHeader().Number, chain.CurrentHeader().Hash().Bytes()[:4])
		}
		if _, err := chain.InsertChain(forks[i : i+1]); err != nil {
			t.Fatalf(" fork %d: failed to insert into chain: %v", i, err)
		}
		if chain.CurrentBlock().Hash() != chain.CurrentHeader().Hash() {
			t.Errorf(" fork %d: current block/header mismatch: block #%d [%x..], header #%d [%x..]", i, chain.CurrentBlock().Number(), chain.CurrentBlock().Hash().Bytes()[:4], chain.CurrentHeader().Number, chain.CurrentHeader().Hash().Bytes()[:4])
		}
	}
}

// Tests that importing small side forks doesn't leave junk in the trie database
// cache (which would eventually cause memory issues).
func TestTrieForkGC(t *testing.T) {
	// Generate a canonical chain to act as the main dataset
	engine := ethash.NewFaker()

	db := rawdb.NewMemoryDatabase()
	genesis := (&Genesis{
		Config:  params.TestChainConfig,
		BaseFee: big.NewInt(params.InitialBaseFee),
	}).MustCommit(db)
	blocks, _ := GenerateChain(params.TestChainConfig, genesis, engine, db, 2*triesInMemory, func(i int, b *BlockGen) { b.SetCoinbase(common.Address{1}) })

	// Generate a bunch of fork blocks, each side forking from the canonical chain
	forks := make([]*types.Block, len(blocks))
	for i := 0; i < len(forks); i++ {
		parent := genesis
		if i > 0 {
			parent = blocks[i-1]
		}
		fork, _ := GenerateChain(params.TestChainConfig, parent, engine, db, 1, func(i int, b *BlockGen) { b.SetCoinbase(common.Address{2}) })
		forks[i] = fork[0]
	}
	// Import the canonical and fork chain side by side, forcing the trie cache to cache both
	diskdb := rawdb.NewMemoryDatabase()
	(&Genesis{
		Config:  params.TestChainConfig,
		BaseFee: big.NewInt(params.InitialBaseFee),
	}).MustCommit(diskdb)

	chain, err := NewBlockChain(diskdb, nil, params.TestChainConfig, engine, vm.Config{})
	if err != nil {
		t.Fatalf("failed to create tester chain: %v", err)
	}
	for i := 0; i < len(blocks); i++ {
		if _, err := chain.InsertChain(blocks[i : i+1]); err != nil {
			t.Fatalf("block %d: failed to insert into chain: %v", i, err)
		}
		if _, err := chain.InsertChain(forks[i : i+1]); err != nil {
			t.Fatalf("fork %d: failed to insert into chain: %v", i, err)
		}
	}
	// Dereference all the recent tries and ensure no past trie is left in
	for i := 0; i < triesInMemory; i++ {
		chain.stateCache.TrieDB().Dereference(blocks[len(blocks)-1-i].Root())
		chain.stateCache.TrieDB().Dereference(forks[len(blocks)-1-i].Root())
	}
	if len(chain.stateCache.TrieDB().Nodes()) > 0 {
		t.Fatalf("stale tries still alive after garbase collection")
	}
}

// Tests that doing large reorgs works even if the state associated with the
// forking point is not available any more.
func TestLargeReorgTrieGC(t *testing.T) {
	// Generate the original common chain segment and the two competing forks
	engine := ethash.NewFaker()
	gspec := &Genesis{
		Config:  params.TestChainConfig,
		BaseFee: big.NewInt(params.InitialBaseFee),
	}

	db := rawdb.NewMemoryDatabase()
	genesis := gspec.MustCommit(db)

	shared, _ := GenerateChain(params.TestChainConfig, genesis, engine, db, 64, func(i int, b *BlockGen) { b.SetCoinbase(common.Address{1}) })
	original, _ := GenerateChain(params.TestChainConfig, shared[len(shared)-1], engine, db, 2*triesInMemory, func(i int, b *BlockGen) { b.SetCoinbase(common.Address{2}) })
	competitor, _ := GenerateChain(params.TestChainConfig, shared[len(shared)-1], engine, db, 2*triesInMemory+1, func(i int, b *BlockGen) { b.SetCoinbase(common.Address{3}) })

	// Import the shared chain and the original canonical one
	diskdb := rawdb.NewMemoryDatabase()
	gspec.MustCommit(diskdb)

	chain, err := NewBlockChain(diskdb, nil, params.TestChainConfig, engine, vm.Config{})
	if err != nil {
		t.Fatalf("failed to create tester chain: %v", err)
	}
	if _, err := chain.InsertChain(shared); err != nil {
		t.Fatalf("failed to insert shared chain: %v", err)
	}
	if _, err := chain.InsertChain(original); err != nil {
		t.Fatalf("failed to insert shared chain: %v", err)
	}
	// Ensure that the state associated with the forking point is pruned away
	if node, _ := chain.stateCache.TrieDB().Node(shared[len(shared)-1].Root()); node != nil {
		t.Fatalf("common-but-old ancestor still cache")
	}
	// Import the competitor chain without exceeding the canonical's TD and ensure
	// we have not processed any of the blocks (protection against malicious blocks)
	if _, err := chain.InsertChain(competitor[:len(competitor)-2]); err != nil {
		t.Fatalf("failed to insert competitor chain: %v", err)
	}
	for i, block := range competitor[:len(competitor)-2] {
		if node, _ := chain.stateCache.TrieDB().Node(block.Root()); node != nil {
			t.Fatalf("competitor %d: low TD chain became processed", i)
		}
	}
	// Import the head of the competitor chain, triggering the reorg and ensure we
	// successfully reprocess all the stashed away blocks.
	if _, err := chain.InsertChain(competitor[len(competitor)-2:]); err != nil {
		t.Fatalf("failed to finalize competitor chain: %v", err)
	}
	for i, block := range competitor[:len(competitor)-triesInMemory] {
		if node, _ := chain.stateCache.TrieDB().Node(block.Root()); node != nil {
			t.Fatalf("competitor %d: competing chain state missing", i)
		}
	}
}

/*
	Collection test for BlocksHashCache
	cases
		1. When init new chain
		2. when insertChain
		3. when insertFork
		4. When adding new block by mining
		5. When adding new block by syncing with other nodes

*/

func TestBlocksHashCacheUpdate(t *testing.T) {
	_, chain, err := newCanonical(ethash.NewFaker(), 0, true)
	if err != nil {
		t.Fatalf("failed to make new canonical chain: %v", err)
	}
	defer chain.Stop()
	if err != nil {
		t.Fatalf("failed to create tester chain: %v", err)
	}

	t.Run("Expect BlocksHashCache blank after initialized", func(t *testing.T) {
		if len(chain.blocksHashCache.Keys()) != 0 {
			t.Error("BlocksHashCache is not initialized correctly ")
		}
	})

	t.Run("Expect BlocksHashCache has 4 cached keys after concat a 4-length-chain", func(t *testing.T) {
		concatedChain := makeBlockChain(chain.CurrentBlock(), 4, ethash.NewFullFaker(), chain.db, 0)
		if _, err := chain.InsertChain(concatedChain); err != nil {
			t.Fatalf("failed to insert shared chain: %v", err)
		}

		if len(chain.blocksHashCache.Keys()) != 4 {
			t.Error("BlocksHashCache doesn't add new cache after concating new chain ")
		}
	})

	t.Run("Expect BlocksHashCache caches work for fork case", func(t *testing.T) {
		concatedChain := makeBlockChain(chain.GetBlockByNumber(uint64(2)), 3, ethash.NewFullFaker(), chain.db, 3)
		if _, err := chain.InsertChain(concatedChain); err != nil {
			t.Fatalf("failed to insert forked chain: %v", err)
		}
		cachedAt, _ := chain.blocksHashCache.Get(uint64(3))

		if len(cachedAt) != 2 {
			t.Error("BlocksHashCache doesn't add new cache after concating new fork ")
		}
	})

	t.Run("Expect BlocksHashCache caches when inserting block on syncing", func(t *testing.T) {
		currentCachedLength := len(chain.blocksHashCache.Keys())
		singleBlockChain := makeBlockChain(chain.CurrentBlock(), 1, ethash.NewFaker(), chain.db, 0)
		chain.insertBlock(singleBlockChain[0])

		if len(chain.blocksHashCache.Keys()) != currentCachedLength+1 {
			t.Error("BlocksHashCache doesn't work when inserting block solely")
		}
	})

}

func TestAreTwoBlocksSamePath(t *testing.T) {
	_, chain, err := newCanonical(ethash.NewFaker(), 10, true)
	if err != nil {
		t.Fatalf("failed to make new canonical chain: %v", err)
	}
	defer chain.Stop()
	if err != nil {
		t.Fatalf("failed to create tester chain: %v", err)
	}

	t.Run("Expect return true with two canonical blocks", func(t *testing.T) {
		if !chain.AreTwoBlockSamePath(chain.CurrentBlock().Hash(), chain.GetBlockByNumber(uint64(2)).Hash()) {
			t.Error("Failed")
		}
	})

	t.Run("Expect return fail with canonical-fork paths", func(t *testing.T) {
		concatedChain := makeBlockChain(chain.GetBlockByNumber(uint64(2)), 3, ethash.NewFullFaker(), chain.db, 3)
		if _, err := chain.InsertChain(concatedChain); err != nil {
			t.Fatalf("failed to insert forked chain: %v", err)
		}
		if chain.AreTwoBlockSamePath(concatedChain[len(concatedChain)-1].Hash(), chain.GetBlockByNumber(uint64(3)).Hash()) {
			t.Error("Failed")
		}
	})

}

// TestEIP2718Transition tests that an EIP-2718 transaction will be accepted
// after the fork block has passed. This is verified by sending an EIP-2930
// access list transaction, which specifies a single slot access, and then
// checking that the gas usage of a hot SLOAD and a cold SLOAD are calculated
// correctly.
func TestEIP2718Transition(t *testing.T) {
	var (
		aa = common.HexToAddress("0x000000000000000000000000000000000000aaaa")

		// Generate a canonical chain to act as the main dataset
		engine = ethash.NewFaker()
		db     = rawdb.NewMemoryDatabase()

		// A sender who makes transactions, has some funds
		key, _  = crypto.HexToECDSA("b71c71a67e1177ad4e901695e1b4b9ee17ae16c6668d313eac2f96dbcda3f291")
		address = crypto.PubkeyToAddress(key.PublicKey)
		// funds   = big.NewInt(math.MaxInt64)
		funds = big.NewInt(1000000000000000)
		gspec = &Genesis{
			Config: &params.ChainConfig{
				ChainId:             new(big.Int).SetBytes([]byte("eip1559")),
				HomesteadBlock:      big.NewInt(0),
				DAOForkBlock:        nil,
				DAOForkSupport:      true,
				EIP150Block:         big.NewInt(0),
				EIP155Block:         big.NewInt(0),
				EIP158Block:         big.NewInt(0),
				ByzantiumBlock:      big.NewInt(0),
				ConstantinopleBlock: big.NewInt(0),
				PetersburgBlock:     big.NewInt(0),
				IstanbulBlock:       big.NewInt(0),
				Eip1559Block:        big.NewInt(0),
			},
			Alloc: types.GenesisAlloc{
				address: {Balance: funds},
				// The address 0xAAAA sloads 0x00 and 0x01
				aa: {
					Code: []byte{
						byte(vm.PC),
						byte(vm.PC),
						byte(vm.SLOAD),
						byte(vm.SLOAD),
					},
					Nonce:   0,
					Balance: big.NewInt(50000000000),
				},
			},
		}
		genesis = gspec.MustCommit(db)
	)

	blocks, _ := GenerateChain(gspec.Config, genesis, engine, db, 1, func(i int, b *BlockGen) {
		b.SetCoinbase(common.Address{1})

		// One transaction to 0xAAAA
		signer := types.LatestSigner(gspec.Config)
		tx, _ := types.SignNewTx(key, signer, &types.AccessListTx{
			ChainID:  gspec.Config.ChainId,
			Nonce:    0,
			To:       &aa,
			Gas:      30000,
			GasPrice: b.header.BaseFee,
			AccessList: types.AccessList{{
				Address:     aa,
				StorageKeys: []common.Hash{{0}},
			}},
		})
		b.AddTx(tx)
	})

	// Import the canonical chain
	diskdb := rawdb.NewMemoryDatabase()
	gspec.MustCommit(diskdb)

	chain, err := NewBlockChain(diskdb, nil, gspec.Config, engine, vm.Config{})
	if err != nil {
		t.Fatalf("failed to create tester chain: %v", err)
	}
	if n, err := chain.InsertChain(blocks); err != nil {
		t.Fatalf("block %d: failed to insert into chain: %v", n, err)
	}

	block := chain.GetBlockByNumber(1)

	// Expected gas is intrinsic + 2 * pc + hot load + cold load, since only one load is in the access list
	expected := params.TxGas + params.TxAccessListAddressGas + params.TxAccessListStorageKeyGas +
		vm.GasQuickStep*2 + params.WarmStorageReadCostEIP2929 + params.ColdSloadCostEIP2929
	if block.GasUsed() != expected {
		t.Fatalf("incorrect amount of gas spent: expected %d, got %d", expected, block.GasUsed())
	}
}

// TestTransientStorageReset ensures the transient storage is wiped correctly
// between transactions.
func TestTransientStorageReset(t *testing.T) {
	var (
		engine      = ethash.NewFaker()
		key, _      = crypto.HexToECDSA("b71c71a67e1177ad4e901695e1b4b9ee17ae16c6668d313eac2f96dbcda3f291")
		address     = crypto.PubkeyToAddress(key.PublicKey)
		destAddress = crypto.CreateAddress(address, 0)
		funds       = big.NewInt(1000000000000000)
		vmConfig    = vm.Config{
			ExtraEips: []int{1153}, // Enable transient storage EIP
		}
	)
	code := append([]byte{
		// TLoad value with location 1
		byte(vm.PUSH1), 0x1,
		byte(vm.TLOAD),

		// PUSH location
		byte(vm.PUSH1), 0x1,

		// SStore location:value
		byte(vm.SSTORE),
	}, make([]byte, 32-6)...)
	initCode := []byte{
		// TSTORE 1:1
		byte(vm.PUSH1), 0x1,
		byte(vm.PUSH1), 0x1,
		byte(vm.TSTORE),

		// Get the runtime-code on the stack
		byte(vm.PUSH32)}
	initCode = append(initCode, code...)
	initCode = append(initCode, []byte{
		byte(vm.PUSH1), 0x0, // offset
		byte(vm.MSTORE),
		byte(vm.PUSH1), 0x6, // size
		byte(vm.PUSH1), 0x0, // offset
		byte(vm.RETURN), // return 6 bytes of zero-code
	}...)
	gspec := &Genesis{
		Config: params.TestChainConfig,
		Alloc: types.GenesisAlloc{
			address: {Balance: funds},
		},
	}
	nonce := uint64(0)
	signer := types.HomesteadSigner{}
	_, blocks, _ := GenerateChainWithGenesis(gspec, engine, 1, func(i int, b *BlockGen) {
		fee := big.NewInt(1)
		if b.header.BaseFee != nil {
			fee = b.header.BaseFee
		}
		b.SetCoinbase(common.Address{1})
		tx, _ := types.SignNewTx(key, signer, &types.LegacyTx{
			Nonce:    nonce,
			GasPrice: new(big.Int).Set(fee),
			Gas:      100000,
			Data:     initCode,
		})
		nonce++
		b.AddTxWithVMConfig(tx, vmConfig)

		tx, _ = types.SignNewTx(key, signer, &types.LegacyTx{
			Nonce:    nonce,
			GasPrice: new(big.Int).Set(fee),
			Gas:      100000,
			To:       &destAddress,
		})
		b.AddTxWithVMConfig(tx, vmConfig)
		nonce++
	})

	diskdb := rawdb.NewMemoryDatabase()
	gspec.MustCommit(diskdb)

	// Initialize the blockchain with 1153 enabled.
	chain, err := NewBlockChain(diskdb, nil, gspec.Config, engine, vmConfig)
	if err != nil {
		t.Fatalf("failed to create tester chain: %v", err)
	}
	// Import the blocks
	if _, err := chain.InsertChain(blocks); err != nil {
		t.Fatalf("failed to insert into chain: %v", err)
	}
	// Check the storage
	state, err := chain.StateAt(chain.CurrentHeader().Root)
	if err != nil {
		t.Fatalf("Failed to load state %v", err)
	}
	loc := common.BytesToHash([]byte{1})
	slot := state.GetState(destAddress, loc)
	if slot != (common.Hash{}) {
		t.Fatalf("Unexpected dirty storage slot")
	}
}

func TestEIP3651(t *testing.T) {
	var (
		ConstantinopleBlockReward = big.NewInt(3e+18) // Block reward in wei for successfully mining a block upward from Constantinople

		aa     = common.HexToAddress("0x000000000000000000000000000000000000aaaa")
		bb     = common.HexToAddress("0x000000000000000000000000000000000000bbbb")
		engine = ethash.NewFaker()

		// A sender who makes transactions, has some funds
		key1, _ = crypto.HexToECDSA("b71c71a67e1177ad4e901695e1b4b9ee17ae16c6668d313eac2f96dbcda3f291")
		key2, _ = crypto.HexToECDSA("8a1f9a8f95be41cd7ccb6168179afb4504aefe388d1e14474d32c45c72ce7b7a")
		addr1   = crypto.PubkeyToAddress(key1.PublicKey)
		addr2   = crypto.PubkeyToAddress(key2.PublicKey)
		funds   = big.NewInt(8000000000000000)
		gspec   = &Genesis{
			Config: params.TestChainConfig,
			Alloc: types.GenesisAlloc{
				addr1: {Balance: funds},
				addr2: {Balance: funds},
				// The address 0xAAAA sloads 0x00 and 0x01
				aa: {
					Code: []byte{
						byte(vm.PC),
						byte(vm.PC),
						byte(vm.SLOAD),
						byte(vm.SLOAD),
					},
					Nonce:   0,
					Balance: big.NewInt(0),
				},
				// The address 0xBBBB calls 0xAAAA
				bb: {
					Code: []byte{
						byte(vm.PUSH1), 0, // out size
						byte(vm.DUP1),  // out offset
						byte(vm.DUP1),  // out insize
						byte(vm.DUP1),  // in offset
						byte(vm.PUSH2), // address
						byte(0xaa),
						byte(0xaa),
						byte(vm.GAS), // gas
						byte(vm.DELEGATECALL),
					},
					Nonce:   0,
					Balance: big.NewInt(0),
				},
			},
		}
	)

	gspec.Config.Eip1559Block = common.Big0
	signer := types.LatestSigner(gspec.Config)

	_, blocks, _ := GenerateChainWithGenesis(gspec, engine, 1, func(i int, b *BlockGen) {
		b.SetCoinbase(aa)
		// One transaction to Coinbase
		txdata := &types.DynamicFeeTx{
			ChainID:    gspec.Config.ChainId,
			Nonce:      0,
			To:         &bb,
			Gas:        500000,
			GasFeeCap:  big.NewInt(12500000000),
			GasTipCap:  big.NewInt(0),
			AccessList: nil,
			Data:       []byte{},
		}
		tx := types.NewTx(txdata)
		tx, _ = types.SignTx(tx, signer, key1)

		b.AddTx(tx)
	})

	diskdb := rawdb.NewMemoryDatabase()
	gspec.MustCommit(diskdb)

	chain, err := NewBlockChain(diskdb, nil, gspec.Config, engine, vm.Config{})
	if err != nil {
		t.Fatalf("failed to create tester chain: %v", err)
	}
	if n, err := chain.InsertChain(blocks); err != nil {
		t.Fatalf("block %d: failed to insert into chain: %v", n, err)
	}

	block := chain.GetBlockByNumber(1)

	// 1+2: Ensure EIP-1559 access lists are accounted for via gas usage.
	innerGas := vm.GasQuickStep*2 + params.ColdSloadCostEIP2929*2
	expectedGas := params.TxGas + 5*vm.GasFastestStep + vm.GasQuickStep + 100 + innerGas // 100 because 0xaaaa is in access list
	if block.GasUsed() != expectedGas {
		t.Fatalf("incorrect amount of gas spent: expected %d, got %d", expectedGas, block.GasUsed())
	}

	state, _ := chain.State()

	// 3: Ensure that miner received only the tx's tip.
	actual := state.GetBalance(block.Coinbase())
	expected := new(big.Int).Add(
		new(big.Int).SetUint64(block.GasUsed()*block.Transactions()[0].GasTipCap().Uint64()),
		ConstantinopleBlockReward,
	)
	if actual.Cmp(expected) != 0 {
		t.Fatalf("miner balance incorrect: expected %d, got %d", expected, actual)
	}

	// 4: Ensure the tx sender paid for the gasUsed * (tip + block baseFee).
	actual = new(big.Int).Sub(funds, state.GetBalance(addr1))
	expected = new(big.Int).SetUint64(block.GasUsed() * (block.Transactions()[0].GasTipCap().Uint64() + block.BaseFee().Uint64()))
	if actual.Cmp(expected) != 0 {
		t.Fatalf("sender balance incorrect: expected %d, got %d", expected, actual)
	}
}

// TestDeleteCreateRevert tests a weird state transition corner case that we hit
// while changing the internals of statedb. The workflow is that a contract is
// self destructed, then in a followup transaction (but same block) it's created
// again and the transaction reverted.
//
// The original statedb implementation flushed dirty objects to the tries after
// each transaction, so this works ok. The rework accumulated writes in memory
// first, but the journal wiped the entire state object on create-revert.
func TestDeleteCreateRevert(t *testing.T) {
	var (
		aa = common.HexToAddress("0x000000000000000000000000000000000000aaaa")
		bb = common.HexToAddress("0x000000000000000000000000000000000000bbbb")
		// Generate a canonical chain to act as the main dataset
		engine = ethash.NewFaker()
		db     = rawdb.NewMemoryDatabase()

		// A sender who makes transactions, has some funds
		key, _  = crypto.HexToECDSA("b71c71a67e1177ad4e901695e1b4b9ee17ae16c6668d313eac2f96dbcda3f291")
		address = crypto.PubkeyToAddress(key.PublicKey)
		funds   = big.NewInt(10000000000000000)
		gspec   = &Genesis{
			Config: params.TestChainConfig,
			Alloc: GenesisAlloc{
				address: {Balance: funds},
				// The address 0xAAAAA selfdestructs if called
				aa: {
					// Code needs to just selfdestruct
					Code:    []byte{byte(vm.PC), 0xFF},
					Nonce:   1,
					Balance: big.NewInt(0),
				},
				// The address 0xBBBB send 1 wei to 0xAAAA, then reverts
				bb: {
					Code: []byte{
						byte(vm.PC),          // [0]
						byte(vm.DUP1),        // [0,0]
						byte(vm.DUP1),        // [0,0,0]
						byte(vm.DUP1),        // [0,0,0,0]
						byte(vm.PUSH1), 0x01, // [0,0,0,0,1] (value)
						byte(vm.PUSH2), 0xaa, 0xaa, // [0,0,0,0,1, 0xaaaa]
						byte(vm.GAS),
						byte(vm.CALL),
						byte(vm.REVERT),
					},
					Balance: big.NewInt(1),
				},
			},
		}
		genesis = gspec.MustCommit(db)
	)

	blocks, _ := GenerateChain(params.TestChainConfig, genesis, engine, db, 1, func(i int, b *BlockGen) {
		b.SetCoinbase(common.Address{1})
		// One transaction to AAAA
		tx, _ := types.SignTx(types.NewTransaction(0, aa,
			big.NewInt(0), 50000, b.header.BaseFee, nil), types.HomesteadSigner{}, key)
		b.AddTx(tx)
		// One transaction to BBBB
		tx, _ = types.SignTx(types.NewTransaction(1, bb,
			big.NewInt(0), 100000, b.header.BaseFee, nil), types.HomesteadSigner{}, key)
		b.AddTx(tx)
	})
	// Import the canonical chain
	diskdb := rawdb.NewMemoryDatabase()
	gspec.MustCommit(diskdb)

	chain, err := NewBlockChain(diskdb, nil, params.TestChainConfig, engine, vm.Config{})
	if err != nil {
		t.Fatalf("failed to create tester chain: %v", err)
	}
	if n, err := chain.InsertChain(blocks); err != nil {
		t.Fatalf("block %d: failed to insert into chain: %v", n, err)
	}
}
