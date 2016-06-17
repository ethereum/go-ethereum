// Copyright 2015 The go-ethereum Authors
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

package downloader

import (
	"errors"
	"fmt"
	"math/big"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/event"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/trie"
)

var (
	testdb, _   = ethdb.NewMemDatabase()
	testKey, _  = crypto.HexToECDSA("b71c71a67e1177ad4e901695e1b4b9ee17ae16c6668d313eac2f96dbcda3f291")
	testAddress = crypto.PubkeyToAddress(testKey.PublicKey)
	genesis     = core.GenesisBlockForTesting(testdb, testAddress, big.NewInt(1000000000))
)

// Reduce some of the parameters to make the tester faster.
func init() {
	MaxForkAncestry = uint64(10000)
	blockCacheLimit = 1024
}

// makeChain creates a chain of n blocks starting at and including parent.
// the returned hash chain is ordered head->parent. In addition, every 3rd block
// contains a transaction and every 5th an uncle to allow testing correct block
// reassembly.
func makeChain(n int, seed byte, parent *types.Block, parentReceipts types.Receipts, heavy bool) ([]common.Hash, map[common.Hash]*types.Header, map[common.Hash]*types.Block, map[common.Hash]types.Receipts) {
	// Generate the block chain
	blocks, receipts := core.GenerateChain(parent, testdb, n, func(i int, block *core.BlockGen) {
		block.SetCoinbase(common.Address{seed})

		// If a heavy chain is requested, delay blocks to raise difficulty
		if heavy {
			block.OffsetTime(-1)
		}
		// If the block number is multiple of 3, send a bonus transaction to the miner
		if parent == genesis && i%3 == 0 {
			tx, err := types.NewTransaction(block.TxNonce(testAddress), common.Address{seed}, big.NewInt(1000), params.TxGas, nil, nil).SignECDSA(testKey)
			if err != nil {
				panic(err)
			}
			block.AddTx(tx)
		}
		// If the block number is a multiple of 5, add a bonus uncle to the block
		if i > 0 && i%5 == 0 {
			block.AddUncle(&types.Header{
				ParentHash: block.PrevBlock(i - 1).Hash(),
				Number:     big.NewInt(block.Number().Int64() - 1),
			})
		}
	})
	// Convert the block-chain into a hash-chain and header/block maps
	hashes := make([]common.Hash, n+1)
	hashes[len(hashes)-1] = parent.Hash()

	headerm := make(map[common.Hash]*types.Header, n+1)
	headerm[parent.Hash()] = parent.Header()

	blockm := make(map[common.Hash]*types.Block, n+1)
	blockm[parent.Hash()] = parent

	receiptm := make(map[common.Hash]types.Receipts, n+1)
	receiptm[parent.Hash()] = parentReceipts

	for i, b := range blocks {
		hashes[len(hashes)-i-2] = b.Hash()
		headerm[b.Hash()] = b.Header()
		blockm[b.Hash()] = b
		receiptm[b.Hash()] = receipts[i]
	}
	return hashes, headerm, blockm, receiptm
}

// makeChainFork creates two chains of length n, such that h1[:f] and
// h2[:f] are different but have a common suffix of length n-f.
func makeChainFork(n, f int, parent *types.Block, parentReceipts types.Receipts, balanced bool) ([]common.Hash, []common.Hash, map[common.Hash]*types.Header, map[common.Hash]*types.Header, map[common.Hash]*types.Block, map[common.Hash]*types.Block, map[common.Hash]types.Receipts, map[common.Hash]types.Receipts) {
	// Create the common suffix
	hashes, headers, blocks, receipts := makeChain(n-f, 0, parent, parentReceipts, false)

	// Create the forks, making the second heavyer if non balanced forks were requested
	hashes1, headers1, blocks1, receipts1 := makeChain(f, 1, blocks[hashes[0]], receipts[hashes[0]], false)
	hashes1 = append(hashes1, hashes[1:]...)

	heavy := false
	if !balanced {
		heavy = true
	}
	hashes2, headers2, blocks2, receipts2 := makeChain(f, 2, blocks[hashes[0]], receipts[hashes[0]], heavy)
	hashes2 = append(hashes2, hashes[1:]...)

	for hash, header := range headers {
		headers1[hash] = header
		headers2[hash] = header
	}
	for hash, block := range blocks {
		blocks1[hash] = block
		blocks2[hash] = block
	}
	for hash, receipt := range receipts {
		receipts1[hash] = receipt
		receipts2[hash] = receipt
	}
	return hashes1, hashes2, headers1, headers2, blocks1, blocks2, receipts1, receipts2
}

// downloadTester is a test simulator for mocking out local block chain.
type downloadTester struct {
	stateDb    ethdb.Database
	downloader *Downloader

	ownHashes   []common.Hash                  // Hash chain belonging to the tester
	ownHeaders  map[common.Hash]*types.Header  // Headers belonging to the tester
	ownBlocks   map[common.Hash]*types.Block   // Blocks belonging to the tester
	ownReceipts map[common.Hash]types.Receipts // Receipts belonging to the tester
	ownChainTd  map[common.Hash]*big.Int       // Total difficulties of the blocks in the local chain

	peerHashes   map[string][]common.Hash                  // Hash chain belonging to different test peers
	peerHeaders  map[string]map[common.Hash]*types.Header  // Headers belonging to different test peers
	peerBlocks   map[string]map[common.Hash]*types.Block   // Blocks belonging to different test peers
	peerReceipts map[string]map[common.Hash]types.Receipts // Receipts belonging to different test peers
	peerChainTds map[string]map[common.Hash]*big.Int       // Total difficulties of the blocks in the peer chains

	peerMissingStates map[string]map[common.Hash]bool // State entries that fast sync should not return

	lock sync.RWMutex
}

// newTester creates a new downloader test mocker.
func newTester() *downloadTester {
	tester := &downloadTester{
		ownHashes:         []common.Hash{genesis.Hash()},
		ownHeaders:        map[common.Hash]*types.Header{genesis.Hash(): genesis.Header()},
		ownBlocks:         map[common.Hash]*types.Block{genesis.Hash(): genesis},
		ownReceipts:       map[common.Hash]types.Receipts{genesis.Hash(): nil},
		ownChainTd:        map[common.Hash]*big.Int{genesis.Hash(): genesis.Difficulty()},
		peerHashes:        make(map[string][]common.Hash),
		peerHeaders:       make(map[string]map[common.Hash]*types.Header),
		peerBlocks:        make(map[string]map[common.Hash]*types.Block),
		peerReceipts:      make(map[string]map[common.Hash]types.Receipts),
		peerChainTds:      make(map[string]map[common.Hash]*big.Int),
		peerMissingStates: make(map[string]map[common.Hash]bool),
	}
	tester.stateDb, _ = ethdb.NewMemDatabase()
	tester.stateDb.Put(genesis.Root().Bytes(), []byte{0x00})

	tester.downloader = New(tester.stateDb, new(event.TypeMux), tester.hasHeader, tester.hasBlock, tester.getHeader,
		tester.getBlock, tester.headHeader, tester.headBlock, tester.headFastBlock, tester.commitHeadBlock, tester.getTd,
		tester.insertHeaders, tester.insertBlocks, tester.insertReceipts, tester.rollback, tester.dropPeer)

	return tester
}

// terminate aborts any operations on the embedded downloader and releases all
// held resources.
func (dl *downloadTester) terminate() {
	dl.downloader.Terminate()
}

// sync starts synchronizing with a remote peer, blocking until it completes.
func (dl *downloadTester) sync(id string, td *big.Int, mode SyncMode) error {
	dl.lock.RLock()
	hash := dl.peerHashes[id][0]
	// If no particular TD was requested, load from the peer's blockchain
	if td == nil {
		td = big.NewInt(1)
		if diff, ok := dl.peerChainTds[id][hash]; ok {
			td = diff
		}
	}
	dl.lock.RUnlock()

	// Synchronise with the chosen peer and ensure proper cleanup afterwards
	err := dl.downloader.synchronise(id, hash, td, mode)
	select {
	case <-dl.downloader.cancelCh:
		// Ok, downloader fully cancelled after sync cycle
	default:
		// Downloader is still accepting packets, can block a peer up
		panic("downloader active post sync cycle") // panic will be caught by tester
	}
	return err
}

// hasHeader checks if a header is present in the testers canonical chain.
func (dl *downloadTester) hasHeader(hash common.Hash) bool {
	return dl.getHeader(hash) != nil
}

// hasBlock checks if a block and associated state is present in the testers canonical chain.
func (dl *downloadTester) hasBlock(hash common.Hash) bool {
	block := dl.getBlock(hash)
	if block == nil {
		return false
	}
	_, err := dl.stateDb.Get(block.Root().Bytes())
	return err == nil
}

// getHeader retrieves a header from the testers canonical chain.
func (dl *downloadTester) getHeader(hash common.Hash) *types.Header {
	dl.lock.RLock()
	defer dl.lock.RUnlock()

	return dl.ownHeaders[hash]
}

// getBlock retrieves a block from the testers canonical chain.
func (dl *downloadTester) getBlock(hash common.Hash) *types.Block {
	dl.lock.RLock()
	defer dl.lock.RUnlock()

	return dl.ownBlocks[hash]
}

// headHeader retrieves the current head header from the canonical chain.
func (dl *downloadTester) headHeader() *types.Header {
	dl.lock.RLock()
	defer dl.lock.RUnlock()

	for i := len(dl.ownHashes) - 1; i >= 0; i-- {
		if header := dl.ownHeaders[dl.ownHashes[i]]; header != nil {
			return header
		}
	}
	return genesis.Header()
}

// headBlock retrieves the current head block from the canonical chain.
func (dl *downloadTester) headBlock() *types.Block {
	dl.lock.RLock()
	defer dl.lock.RUnlock()

	for i := len(dl.ownHashes) - 1; i >= 0; i-- {
		if block := dl.ownBlocks[dl.ownHashes[i]]; block != nil {
			if _, err := dl.stateDb.Get(block.Root().Bytes()); err == nil {
				return block
			}
		}
	}
	return genesis
}

// headFastBlock retrieves the current head fast-sync block from the canonical chain.
func (dl *downloadTester) headFastBlock() *types.Block {
	dl.lock.RLock()
	defer dl.lock.RUnlock()

	for i := len(dl.ownHashes) - 1; i >= 0; i-- {
		if block := dl.ownBlocks[dl.ownHashes[i]]; block != nil {
			return block
		}
	}
	return genesis
}

// commitHeadBlock manually sets the head block to a given hash.
func (dl *downloadTester) commitHeadBlock(hash common.Hash) error {
	// For now only check that the state trie is correct
	if block := dl.getBlock(hash); block != nil {
		_, err := trie.NewSecure(block.Root(), dl.stateDb)
		return err
	}
	return fmt.Errorf("non existent block: %x", hash[:4])
}

// getTd retrieves the block's total difficulty from the canonical chain.
func (dl *downloadTester) getTd(hash common.Hash) *big.Int {
	dl.lock.RLock()
	defer dl.lock.RUnlock()

	return dl.ownChainTd[hash]
}

// insertHeaders injects a new batch of headers into the simulated chain.
func (dl *downloadTester) insertHeaders(headers []*types.Header, checkFreq int) (int, error) {
	dl.lock.Lock()
	defer dl.lock.Unlock()

	// Do a quick check, as the blockchain.InsertHeaderChain doesn't insert anything in case of errors
	if _, ok := dl.ownHeaders[headers[0].ParentHash]; !ok {
		return 0, errors.New("unknown parent")
	}
	for i := 1; i < len(headers); i++ {
		if headers[i].ParentHash != headers[i-1].Hash() {
			return i, errors.New("unknown parent")
		}
	}
	// Do a full insert if pre-checks passed
	for i, header := range headers {
		if _, ok := dl.ownHeaders[header.Hash()]; ok {
			continue
		}
		if _, ok := dl.ownHeaders[header.ParentHash]; !ok {
			return i, errors.New("unknown parent")
		}
		dl.ownHashes = append(dl.ownHashes, header.Hash())
		dl.ownHeaders[header.Hash()] = header
		dl.ownChainTd[header.Hash()] = new(big.Int).Add(dl.ownChainTd[header.ParentHash], header.Difficulty)
	}
	return len(headers), nil
}

// insertBlocks injects a new batch of blocks into the simulated chain.
func (dl *downloadTester) insertBlocks(blocks types.Blocks) (int, error) {
	dl.lock.Lock()
	defer dl.lock.Unlock()

	for i, block := range blocks {
		if parent, ok := dl.ownBlocks[block.ParentHash()]; !ok {
			return i, errors.New("unknown parent")
		} else if _, err := dl.stateDb.Get(parent.Root().Bytes()); err != nil {
			return i, fmt.Errorf("unknown parent state %x: %v", parent.Root(), err)
		}
		if _, ok := dl.ownHeaders[block.Hash()]; !ok {
			dl.ownHashes = append(dl.ownHashes, block.Hash())
			dl.ownHeaders[block.Hash()] = block.Header()
		}
		dl.ownBlocks[block.Hash()] = block
		dl.stateDb.Put(block.Root().Bytes(), []byte{0x00})
		dl.ownChainTd[block.Hash()] = new(big.Int).Add(dl.ownChainTd[block.ParentHash()], block.Difficulty())
	}
	return len(blocks), nil
}

// insertReceipts injects a new batch of blocks into the simulated chain.
func (dl *downloadTester) insertReceipts(blocks types.Blocks, receipts []types.Receipts) (int, error) {
	dl.lock.Lock()
	defer dl.lock.Unlock()

	for i := 0; i < len(blocks) && i < len(receipts); i++ {
		if _, ok := dl.ownHeaders[blocks[i].Hash()]; !ok {
			return i, errors.New("unknown owner")
		}
		if _, ok := dl.ownBlocks[blocks[i].ParentHash()]; !ok {
			return i, errors.New("unknown parent")
		}
		dl.ownBlocks[blocks[i].Hash()] = blocks[i]
		dl.ownReceipts[blocks[i].Hash()] = receipts[i]
	}
	return len(blocks), nil
}

// rollback removes some recently added elements from the chain.
func (dl *downloadTester) rollback(hashes []common.Hash) {
	dl.lock.Lock()
	defer dl.lock.Unlock()

	for i := len(hashes) - 1; i >= 0; i-- {
		if dl.ownHashes[len(dl.ownHashes)-1] == hashes[i] {
			dl.ownHashes = dl.ownHashes[:len(dl.ownHashes)-1]
		}
		delete(dl.ownChainTd, hashes[i])
		delete(dl.ownHeaders, hashes[i])
		delete(dl.ownReceipts, hashes[i])
		delete(dl.ownBlocks, hashes[i])
	}
}

// newPeer registers a new block download source into the downloader.
func (dl *downloadTester) newPeer(id string, version int, hashes []common.Hash, headers map[common.Hash]*types.Header, blocks map[common.Hash]*types.Block, receipts map[common.Hash]types.Receipts) error {
	return dl.newSlowPeer(id, version, hashes, headers, blocks, receipts, 0)
}

// newSlowPeer registers a new block download source into the downloader, with a
// specific delay time on processing the network packets sent to it, simulating
// potentially slow network IO.
func (dl *downloadTester) newSlowPeer(id string, version int, hashes []common.Hash, headers map[common.Hash]*types.Header, blocks map[common.Hash]*types.Block, receipts map[common.Hash]types.Receipts, delay time.Duration) error {
	dl.lock.Lock()
	defer dl.lock.Unlock()

	var err error
	switch version {
	case 61:
		err = dl.downloader.RegisterPeer(id, version, hashes[0], dl.peerGetRelHashesFn(id, delay), dl.peerGetAbsHashesFn(id, delay), dl.peerGetBlocksFn(id, delay), nil, nil, nil, nil, nil)
	case 62:
		err = dl.downloader.RegisterPeer(id, version, hashes[0], nil, nil, nil, dl.peerGetRelHeadersFn(id, delay), dl.peerGetAbsHeadersFn(id, delay), dl.peerGetBodiesFn(id, delay), nil, nil)
	case 63:
		err = dl.downloader.RegisterPeer(id, version, hashes[0], nil, nil, nil, dl.peerGetRelHeadersFn(id, delay), dl.peerGetAbsHeadersFn(id, delay), dl.peerGetBodiesFn(id, delay), dl.peerGetReceiptsFn(id, delay), dl.peerGetNodeDataFn(id, delay))
	case 64:
		err = dl.downloader.RegisterPeer(id, version, hashes[0], nil, nil, nil, dl.peerGetRelHeadersFn(id, delay), dl.peerGetAbsHeadersFn(id, delay), dl.peerGetBodiesFn(id, delay), dl.peerGetReceiptsFn(id, delay), dl.peerGetNodeDataFn(id, delay))
	}
	if err == nil {
		// Assign the owned hashes, headers and blocks to the peer (deep copy)
		dl.peerHashes[id] = make([]common.Hash, len(hashes))
		copy(dl.peerHashes[id], hashes)

		dl.peerHeaders[id] = make(map[common.Hash]*types.Header)
		dl.peerBlocks[id] = make(map[common.Hash]*types.Block)
		dl.peerReceipts[id] = make(map[common.Hash]types.Receipts)
		dl.peerChainTds[id] = make(map[common.Hash]*big.Int)
		dl.peerMissingStates[id] = make(map[common.Hash]bool)

		genesis := hashes[len(hashes)-1]
		if header := headers[genesis]; header != nil {
			dl.peerHeaders[id][genesis] = header
			dl.peerChainTds[id][genesis] = header.Difficulty
		}
		if block := blocks[genesis]; block != nil {
			dl.peerBlocks[id][genesis] = block
			dl.peerChainTds[id][genesis] = block.Difficulty()
		}

		for i := len(hashes) - 2; i >= 0; i-- {
			hash := hashes[i]

			if header, ok := headers[hash]; ok {
				dl.peerHeaders[id][hash] = header
				if _, ok := dl.peerHeaders[id][header.ParentHash]; ok {
					dl.peerChainTds[id][hash] = new(big.Int).Add(header.Difficulty, dl.peerChainTds[id][header.ParentHash])
				}
			}
			if block, ok := blocks[hash]; ok {
				dl.peerBlocks[id][hash] = block
				if _, ok := dl.peerBlocks[id][block.ParentHash()]; ok {
					dl.peerChainTds[id][hash] = new(big.Int).Add(block.Difficulty(), dl.peerChainTds[id][block.ParentHash()])
				}
			}
			if receipt, ok := receipts[hash]; ok {
				dl.peerReceipts[id][hash] = receipt
			}
		}
	}
	return err
}

// dropPeer simulates a hard peer removal from the connection pool.
func (dl *downloadTester) dropPeer(id string) {
	dl.lock.Lock()
	defer dl.lock.Unlock()

	delete(dl.peerHashes, id)
	delete(dl.peerHeaders, id)
	delete(dl.peerBlocks, id)
	delete(dl.peerChainTds, id)

	dl.downloader.UnregisterPeer(id)
}

// peerGetRelHashesFn constructs a GetHashes function associated with a specific
// peer in the download tester. The returned function can be used to retrieve
// batches of hashes from the particularly requested peer.
func (dl *downloadTester) peerGetRelHashesFn(id string, delay time.Duration) func(head common.Hash) error {
	return func(head common.Hash) error {
		time.Sleep(delay)

		dl.lock.RLock()
		defer dl.lock.RUnlock()

		// Gather the next batch of hashes
		hashes := dl.peerHashes[id]
		result := make([]common.Hash, 0, MaxHashFetch)
		for i, hash := range hashes {
			if hash == head {
				i++
				for len(result) < cap(result) && i < len(hashes) {
					result = append(result, hashes[i])
					i++
				}
				break
			}
		}
		// Delay delivery a bit to allow attacks to unfold
		go func() {
			time.Sleep(time.Millisecond)
			dl.downloader.DeliverHashes(id, result)
		}()
		return nil
	}
}

// peerGetAbsHashesFn constructs a GetHashesFromNumber function associated with
// a particular peer in the download tester. The returned function can be used to
// retrieve batches of hashes from the particularly requested peer.
func (dl *downloadTester) peerGetAbsHashesFn(id string, delay time.Duration) func(uint64, int) error {
	return func(head uint64, count int) error {
		time.Sleep(delay)

		dl.lock.RLock()
		defer dl.lock.RUnlock()

		// Gather the next batch of hashes
		hashes := dl.peerHashes[id]
		result := make([]common.Hash, 0, count)
		for i := 0; i < count && len(hashes)-int(head)-1-i >= 0; i++ {
			result = append(result, hashes[len(hashes)-int(head)-1-i])
		}
		// Delay delivery a bit to allow attacks to unfold
		go func() {
			time.Sleep(time.Millisecond)
			dl.downloader.DeliverHashes(id, result)
		}()
		return nil
	}
}

// peerGetBlocksFn constructs a getBlocks function associated with a particular
// peer in the download tester. The returned function can be used to retrieve
// batches of blocks from the particularly requested peer.
func (dl *downloadTester) peerGetBlocksFn(id string, delay time.Duration) func([]common.Hash) error {
	return func(hashes []common.Hash) error {
		time.Sleep(delay)

		dl.lock.RLock()
		defer dl.lock.RUnlock()

		blocks := dl.peerBlocks[id]
		result := make([]*types.Block, 0, len(hashes))
		for _, hash := range hashes {
			if block, ok := blocks[hash]; ok {
				result = append(result, block)
			}
		}
		go dl.downloader.DeliverBlocks(id, result)

		return nil
	}
}

// peerGetRelHeadersFn constructs a GetBlockHeaders function based on a hashed
// origin; associated with a particular peer in the download tester. The returned
// function can be used to retrieve batches of headers from the particular peer.
func (dl *downloadTester) peerGetRelHeadersFn(id string, delay time.Duration) func(common.Hash, int, int, bool) error {
	return func(origin common.Hash, amount int, skip int, reverse bool) error {
		// Find the canonical number of the hash
		dl.lock.RLock()
		number := uint64(0)
		for num, hash := range dl.peerHashes[id] {
			if hash == origin {
				number = uint64(len(dl.peerHashes[id]) - num - 1)
				break
			}
		}
		dl.lock.RUnlock()

		// Use the absolute header fetcher to satisfy the query
		return dl.peerGetAbsHeadersFn(id, delay)(number, amount, skip, reverse)
	}
}

// peerGetAbsHeadersFn constructs a GetBlockHeaders function based on a numbered
// origin; associated with a particular peer in the download tester. The returned
// function can be used to retrieve batches of headers from the particular peer.
func (dl *downloadTester) peerGetAbsHeadersFn(id string, delay time.Duration) func(uint64, int, int, bool) error {
	return func(origin uint64, amount int, skip int, reverse bool) error {
		time.Sleep(delay)

		dl.lock.RLock()
		defer dl.lock.RUnlock()

		// Gather the next batch of headers
		hashes := dl.peerHashes[id]
		headers := dl.peerHeaders[id]
		result := make([]*types.Header, 0, amount)
		for i := 0; i < amount && len(hashes)-int(origin)-1-i*(skip+1) >= 0; i++ {
			if header, ok := headers[hashes[len(hashes)-int(origin)-1-i*(skip+1)]]; ok {
				result = append(result, header)
			}
		}
		// Delay delivery a bit to allow attacks to unfold
		go func() {
			time.Sleep(time.Millisecond)
			dl.downloader.DeliverHeaders(id, result)
		}()
		return nil
	}
}

// peerGetBodiesFn constructs a getBlockBodies method associated with a particular
// peer in the download tester. The returned function can be used to retrieve
// batches of block bodies from the particularly requested peer.
func (dl *downloadTester) peerGetBodiesFn(id string, delay time.Duration) func([]common.Hash) error {
	return func(hashes []common.Hash) error {
		time.Sleep(delay)

		dl.lock.RLock()
		defer dl.lock.RUnlock()

		blocks := dl.peerBlocks[id]

		transactions := make([][]*types.Transaction, 0, len(hashes))
		uncles := make([][]*types.Header, 0, len(hashes))

		for _, hash := range hashes {
			if block, ok := blocks[hash]; ok {
				transactions = append(transactions, block.Transactions())
				uncles = append(uncles, block.Uncles())
			}
		}
		go dl.downloader.DeliverBodies(id, transactions, uncles)

		return nil
	}
}

// peerGetReceiptsFn constructs a getReceipts method associated with a particular
// peer in the download tester. The returned function can be used to retrieve
// batches of block receipts from the particularly requested peer.
func (dl *downloadTester) peerGetReceiptsFn(id string, delay time.Duration) func([]common.Hash) error {
	return func(hashes []common.Hash) error {
		time.Sleep(delay)

		dl.lock.RLock()
		defer dl.lock.RUnlock()

		receipts := dl.peerReceipts[id]

		results := make([][]*types.Receipt, 0, len(hashes))
		for _, hash := range hashes {
			if receipt, ok := receipts[hash]; ok {
				results = append(results, receipt)
			}
		}
		go dl.downloader.DeliverReceipts(id, results)

		return nil
	}
}

// peerGetNodeDataFn constructs a getNodeData method associated with a particular
// peer in the download tester. The returned function can be used to retrieve
// batches of node state data from the particularly requested peer.
func (dl *downloadTester) peerGetNodeDataFn(id string, delay time.Duration) func([]common.Hash) error {
	return func(hashes []common.Hash) error {
		time.Sleep(delay)

		dl.lock.RLock()
		defer dl.lock.RUnlock()

		results := make([][]byte, 0, len(hashes))
		for _, hash := range hashes {
			if data, err := testdb.Get(hash.Bytes()); err == nil {
				if !dl.peerMissingStates[id][hash] {
					results = append(results, data)
				}
			}
		}
		go dl.downloader.DeliverNodeData(id, results)

		return nil
	}
}

// assertOwnChain checks if the local chain contains the correct number of items
// of the various chain components.
func assertOwnChain(t *testing.T, tester *downloadTester, length int) {
	assertOwnForkedChain(t, tester, 1, []int{length})
}

// assertOwnForkedChain checks if the local forked chain contains the correct
// number of items of the various chain components.
func assertOwnForkedChain(t *testing.T, tester *downloadTester, common int, lengths []int) {
	// Initialize the counters for the first fork
	headers, blocks := lengths[0], lengths[0]

	minReceipts, maxReceipts := lengths[0]-fsMinFullBlocks-fsPivotInterval, lengths[0]-fsMinFullBlocks
	if minReceipts < 0 {
		minReceipts = 1
	}
	if maxReceipts < 0 {
		maxReceipts = 1
	}
	// Update the counters for each subsequent fork
	for _, length := range lengths[1:] {
		headers += length - common
		blocks += length - common

		minReceipts += length - common - fsMinFullBlocks - fsPivotInterval
		maxReceipts += length - common - fsMinFullBlocks
	}
	switch tester.downloader.mode {
	case FullSync:
		minReceipts, maxReceipts = 1, 1
	case LightSync:
		blocks, minReceipts, maxReceipts = 1, 1, 1
	}
	if hs := len(tester.ownHeaders); hs != headers {
		t.Fatalf("synchronised headers mismatch: have %v, want %v", hs, headers)
	}
	if bs := len(tester.ownBlocks); bs != blocks {
		t.Fatalf("synchronised blocks mismatch: have %v, want %v", bs, blocks)
	}
	if rs := len(tester.ownReceipts); rs < minReceipts || rs > maxReceipts {
		t.Fatalf("synchronised receipts mismatch: have %v, want between [%v, %v]", rs, minReceipts, maxReceipts)
	}
	// Verify the state trie too for fast syncs
	if tester.downloader.mode == FastSync {
		index := 0
		if pivot := int(tester.downloader.queue.fastSyncPivot); pivot < common {
			index = pivot
		} else {
			index = len(tester.ownHashes) - lengths[len(lengths)-1] + int(tester.downloader.queue.fastSyncPivot)
		}
		if index > 0 {
			if statedb, err := state.New(tester.ownHeaders[tester.ownHashes[index]].Root, tester.stateDb); statedb == nil || err != nil {
				t.Fatalf("state reconstruction failed: %v", err)
			}
		}
	}
}

// Tests that simple synchronization against a canonical chain works correctly.
// In this test common ancestor lookup should be short circuited and not require
// binary searching.
func TestCanonicalSynchronisation61(t *testing.T)      { testCanonicalSynchronisation(t, 61, FullSync) }
func TestCanonicalSynchronisation62(t *testing.T)      { testCanonicalSynchronisation(t, 62, FullSync) }
func TestCanonicalSynchronisation63Full(t *testing.T)  { testCanonicalSynchronisation(t, 63, FullSync) }
func TestCanonicalSynchronisation63Fast(t *testing.T)  { testCanonicalSynchronisation(t, 63, FastSync) }
func TestCanonicalSynchronisation64Full(t *testing.T)  { testCanonicalSynchronisation(t, 64, FullSync) }
func TestCanonicalSynchronisation64Fast(t *testing.T)  { testCanonicalSynchronisation(t, 64, FastSync) }
func TestCanonicalSynchronisation64Light(t *testing.T) { testCanonicalSynchronisation(t, 64, LightSync) }

func testCanonicalSynchronisation(t *testing.T, protocol int, mode SyncMode) {
	t.Parallel()

	// Create a small enough block chain to download
	targetBlocks := blockCacheLimit - 15
	hashes, headers, blocks, receipts := makeChain(targetBlocks, 0, genesis, nil, false)

	tester := newTester()
	defer tester.terminate()

	tester.newPeer("peer", protocol, hashes, headers, blocks, receipts)

	// Synchronise with the peer and make sure all relevant data was retrieved
	if err := tester.sync("peer", nil, mode); err != nil {
		t.Fatalf("failed to synchronise blocks: %v", err)
	}
	assertOwnChain(t, tester, targetBlocks+1)
}

// Tests that if a large batch of blocks are being downloaded, it is throttled
// until the cached blocks are retrieved.
func TestThrottling61(t *testing.T)     { testThrottling(t, 61, FullSync) }
func TestThrottling62(t *testing.T)     { testThrottling(t, 62, FullSync) }
func TestThrottling63Full(t *testing.T) { testThrottling(t, 63, FullSync) }
func TestThrottling63Fast(t *testing.T) { testThrottling(t, 63, FastSync) }
func TestThrottling64Full(t *testing.T) { testThrottling(t, 64, FullSync) }
func TestThrottling64Fast(t *testing.T) { testThrottling(t, 64, FastSync) }

func testThrottling(t *testing.T, protocol int, mode SyncMode) {
	// Create a long block chain to download and the tester
	targetBlocks := 8 * blockCacheLimit
	hashes, headers, blocks, receipts := makeChain(targetBlocks, 0, genesis, nil, false)

	tester := newTester()
	defer tester.terminate()

	tester.newPeer("peer", protocol, hashes, headers, blocks, receipts)

	// Wrap the importer to allow stepping
	blocked, proceed := uint32(0), make(chan struct{})
	tester.downloader.chainInsertHook = func(results []*fetchResult) {
		atomic.StoreUint32(&blocked, uint32(len(results)))
		<-proceed
	}
	// Start a synchronisation concurrently
	errc := make(chan error)
	go func() {
		errc <- tester.sync("peer", nil, mode)
	}()
	// Iteratively take some blocks, always checking the retrieval count
	for {
		// Check the retrieval count synchronously (! reason for this ugly block)
		tester.lock.RLock()
		retrieved := len(tester.ownBlocks)
		tester.lock.RUnlock()
		if retrieved >= targetBlocks+1 {
			break
		}
		// Wait a bit for sync to throttle itself
		var cached, frozen int
		for start := time.Now(); time.Since(start) < 3*time.Second; {
			time.Sleep(25 * time.Millisecond)

			tester.lock.Lock()
			tester.downloader.queue.lock.Lock()
			cached = len(tester.downloader.queue.blockDonePool)
			if mode == FastSync {
				if receipts := len(tester.downloader.queue.receiptDonePool); receipts < cached {
					if tester.downloader.queue.resultCache[receipts].Header.Number.Uint64() < tester.downloader.queue.fastSyncPivot {
						cached = receipts
					}
				}
			}
			frozen = int(atomic.LoadUint32(&blocked))
			retrieved = len(tester.ownBlocks)
			tester.downloader.queue.lock.Unlock()
			tester.lock.Unlock()

			if cached == blockCacheLimit || retrieved+cached+frozen == targetBlocks+1 {
				break
			}
		}
		// Make sure we filled up the cache, then exhaust it
		time.Sleep(25 * time.Millisecond) // give it a chance to screw up

		tester.lock.RLock()
		retrieved = len(tester.ownBlocks)
		tester.lock.RUnlock()
		if cached != blockCacheLimit && retrieved+cached+frozen != targetBlocks+1 {
			t.Fatalf("block count mismatch: have %v, want %v (owned %v, blocked %v, target %v)", cached, blockCacheLimit, retrieved, frozen, targetBlocks+1)
		}
		// Permit the blocked blocks to import
		if atomic.LoadUint32(&blocked) > 0 {
			atomic.StoreUint32(&blocked, uint32(0))
			proceed <- struct{}{}
		}
	}
	// Check that we haven't pulled more blocks than available
	assertOwnChain(t, tester, targetBlocks+1)
	if err := <-errc; err != nil {
		t.Fatalf("block synchronization failed: %v", err)
	}
}

// Tests that simple synchronization against a forked chain works correctly. In
// this test common ancestor lookup should *not* be short circuited, and a full
// binary search should be executed.
func TestForkedSync61(t *testing.T)      { testForkedSync(t, 61, FullSync) }
func TestForkedSync62(t *testing.T)      { testForkedSync(t, 62, FullSync) }
func TestForkedSync63Full(t *testing.T)  { testForkedSync(t, 63, FullSync) }
func TestForkedSync63Fast(t *testing.T)  { testForkedSync(t, 63, FastSync) }
func TestForkedSync64Full(t *testing.T)  { testForkedSync(t, 64, FullSync) }
func TestForkedSync64Fast(t *testing.T)  { testForkedSync(t, 64, FastSync) }
func TestForkedSync64Light(t *testing.T) { testForkedSync(t, 64, LightSync) }

func testForkedSync(t *testing.T, protocol int, mode SyncMode) {
	t.Parallel()

	// Create a long enough forked chain
	common, fork := MaxHashFetch, 2*MaxHashFetch
	hashesA, hashesB, headersA, headersB, blocksA, blocksB, receiptsA, receiptsB := makeChainFork(common+fork, fork, genesis, nil, true)

	tester := newTester()
	defer tester.terminate()

	tester.newPeer("fork A", protocol, hashesA, headersA, blocksA, receiptsA)
	tester.newPeer("fork B", protocol, hashesB, headersB, blocksB, receiptsB)

	// Synchronise with the peer and make sure all blocks were retrieved
	if err := tester.sync("fork A", nil, mode); err != nil {
		t.Fatalf("failed to synchronise blocks: %v", err)
	}
	assertOwnChain(t, tester, common+fork+1)

	// Synchronise with the second peer and make sure that fork is pulled too
	if err := tester.sync("fork B", nil, mode); err != nil {
		t.Fatalf("failed to synchronise blocks: %v", err)
	}
	assertOwnForkedChain(t, tester, common+1, []int{common + fork + 1, common + fork + 1})
}

// Tests that synchronising against a much shorter but much heavyer fork works
// corrently and is not dropped.
func TestHeavyForkedSync61(t *testing.T)      { testHeavyForkedSync(t, 61, FullSync) }
func TestHeavyForkedSync62(t *testing.T)      { testHeavyForkedSync(t, 62, FullSync) }
func TestHeavyForkedSync63Full(t *testing.T)  { testHeavyForkedSync(t, 63, FullSync) }
func TestHeavyForkedSync63Fast(t *testing.T)  { testHeavyForkedSync(t, 63, FastSync) }
func TestHeavyForkedSync64Full(t *testing.T)  { testHeavyForkedSync(t, 64, FullSync) }
func TestHeavyForkedSync64Fast(t *testing.T)  { testHeavyForkedSync(t, 64, FastSync) }
func TestHeavyForkedSync64Light(t *testing.T) { testHeavyForkedSync(t, 64, LightSync) }

func testHeavyForkedSync(t *testing.T, protocol int, mode SyncMode) {
	t.Parallel()

	// Create a long enough forked chain
	common, fork := MaxHashFetch, 4*MaxHashFetch
	hashesA, hashesB, headersA, headersB, blocksA, blocksB, receiptsA, receiptsB := makeChainFork(common+fork, fork, genesis, nil, false)

	tester := newTester()
	defer tester.terminate()

	tester.newPeer("light", protocol, hashesA, headersA, blocksA, receiptsA)
	tester.newPeer("heavy", protocol, hashesB[fork/2:], headersB, blocksB, receiptsB)

	// Synchronise with the peer and make sure all blocks were retrieved
	if err := tester.sync("light", nil, mode); err != nil {
		t.Fatalf("failed to synchronise blocks: %v", err)
	}
	assertOwnChain(t, tester, common+fork+1)

	// Synchronise with the second peer and make sure that fork is pulled too
	if err := tester.sync("heavy", nil, mode); err != nil {
		t.Fatalf("failed to synchronise blocks: %v", err)
	}
	assertOwnForkedChain(t, tester, common+1, []int{common + fork + 1, common + fork/2 + 1})
}

// Tests that an inactive downloader will not accept incoming hashes and blocks.
func TestInactiveDownloader61(t *testing.T) {
	t.Parallel()
	tester := newTester()

	// Check that neither hashes nor blocks are accepted
	if err := tester.downloader.DeliverHashes("bad peer", []common.Hash{}); err != errNoSyncActive {
		t.Errorf("error mismatch: have %v, want %v", err, errNoSyncActive)
	}
	if err := tester.downloader.DeliverBlocks("bad peer", []*types.Block{}); err != errNoSyncActive {
		t.Errorf("error mismatch: have %v, want %v", err, errNoSyncActive)
	}
}

// Tests that chain forks are contained within a certain interval of the current
// chain head, ensuring that malicious peers cannot waste resources by feeding
// long dead chains.
func TestBoundedForkedSync61(t *testing.T)      { testBoundedForkedSync(t, 61, FullSync) }
func TestBoundedForkedSync62(t *testing.T)      { testBoundedForkedSync(t, 62, FullSync) }
func TestBoundedForkedSync63Full(t *testing.T)  { testBoundedForkedSync(t, 63, FullSync) }
func TestBoundedForkedSync63Fast(t *testing.T)  { testBoundedForkedSync(t, 63, FastSync) }
func TestBoundedForkedSync64Full(t *testing.T)  { testBoundedForkedSync(t, 64, FullSync) }
func TestBoundedForkedSync64Fast(t *testing.T)  { testBoundedForkedSync(t, 64, FastSync) }
func TestBoundedForkedSync64Light(t *testing.T) { testBoundedForkedSync(t, 64, LightSync) }

func testBoundedForkedSync(t *testing.T, protocol int, mode SyncMode) {
	t.Parallel()

	// Create a long enough forked chain
	common, fork := 13, int(MaxForkAncestry+17)
	hashesA, hashesB, headersA, headersB, blocksA, blocksB, receiptsA, receiptsB := makeChainFork(common+fork, fork, genesis, nil, true)

	tester := newTester()
	defer tester.terminate()

	tester.newPeer("original", protocol, hashesA, headersA, blocksA, receiptsA)
	tester.newPeer("rewriter", protocol, hashesB, headersB, blocksB, receiptsB)

	// Synchronise with the peer and make sure all blocks were retrieved
	if err := tester.sync("original", nil, mode); err != nil {
		t.Fatalf("failed to synchronise blocks: %v", err)
	}
	assertOwnChain(t, tester, common+fork+1)

	// Synchronise with the second peer and ensure that the fork is rejected to being too old
	if err := tester.sync("rewriter", nil, mode); err != errInvalidAncestor {
		t.Fatalf("sync failure mismatch: have %v, want %v", err, errInvalidAncestor)
	}
}

// Tests that chain forks are contained within a certain interval of the current
// chain head for short but heavy forks too. These are a bit special because they
// take different ancestor lookup paths.
func TestBoundedHeavyForkedSync61(t *testing.T)      { testBoundedHeavyForkedSync(t, 61, FullSync) }
func TestBoundedHeavyForkedSync62(t *testing.T)      { testBoundedHeavyForkedSync(t, 62, FullSync) }
func TestBoundedHeavyForkedSync63Full(t *testing.T)  { testBoundedHeavyForkedSync(t, 63, FullSync) }
func TestBoundedHeavyForkedSync63Fast(t *testing.T)  { testBoundedHeavyForkedSync(t, 63, FastSync) }
func TestBoundedHeavyForkedSync64Full(t *testing.T)  { testBoundedHeavyForkedSync(t, 64, FullSync) }
func TestBoundedHeavyForkedSync64Fast(t *testing.T)  { testBoundedHeavyForkedSync(t, 64, FastSync) }
func TestBoundedHeavyForkedSync64Light(t *testing.T) { testBoundedHeavyForkedSync(t, 64, LightSync) }

func testBoundedHeavyForkedSync(t *testing.T, protocol int, mode SyncMode) {
	t.Parallel()

	// Create a long enough forked chain
	common, fork := 13, int(MaxForkAncestry+17)
	hashesA, hashesB, headersA, headersB, blocksA, blocksB, receiptsA, receiptsB := makeChainFork(common+fork, fork, genesis, nil, false)

	tester := newTester()
	defer tester.terminate()

	tester.newPeer("original", protocol, hashesA, headersA, blocksA, receiptsA)
	tester.newPeer("heavy-rewriter", protocol, hashesB[MaxForkAncestry-17:], headersB, blocksB, receiptsB) // Root the fork below the ancestor limit

	// Synchronise with the peer and make sure all blocks were retrieved
	if err := tester.sync("original", nil, mode); err != nil {
		t.Fatalf("failed to synchronise blocks: %v", err)
	}
	assertOwnChain(t, tester, common+fork+1)

	// Synchronise with the second peer and ensure that the fork is rejected to being too old
	if err := tester.sync("heavy-rewriter", nil, mode); err != errInvalidAncestor {
		t.Fatalf("sync failure mismatch: have %v, want %v", err, errInvalidAncestor)
	}
}

// Tests that an inactive downloader will not accept incoming block headers and
// bodies.
func TestInactiveDownloader62(t *testing.T) {
	t.Parallel()

	tester := newTester()
	defer tester.terminate()

	// Check that neither block headers nor bodies are accepted
	if err := tester.downloader.DeliverHeaders("bad peer", []*types.Header{}); err != errNoSyncActive {
		t.Errorf("error mismatch: have %v, want %v", err, errNoSyncActive)
	}
	if err := tester.downloader.DeliverBodies("bad peer", [][]*types.Transaction{}, [][]*types.Header{}); err != errNoSyncActive {
		t.Errorf("error mismatch: have %v, want %v", err, errNoSyncActive)
	}
}

// Tests that an inactive downloader will not accept incoming block headers,
// bodies and receipts.
func TestInactiveDownloader63(t *testing.T) {
	t.Parallel()

	tester := newTester()
	defer tester.terminate()

	// Check that neither block headers nor bodies are accepted
	if err := tester.downloader.DeliverHeaders("bad peer", []*types.Header{}); err != errNoSyncActive {
		t.Errorf("error mismatch: have %v, want %v", err, errNoSyncActive)
	}
	if err := tester.downloader.DeliverBodies("bad peer", [][]*types.Transaction{}, [][]*types.Header{}); err != errNoSyncActive {
		t.Errorf("error mismatch: have %v, want %v", err, errNoSyncActive)
	}
	if err := tester.downloader.DeliverReceipts("bad peer", [][]*types.Receipt{}); err != errNoSyncActive {
		t.Errorf("error mismatch: have %v, want %v", err, errNoSyncActive)
	}
}

// Tests that a canceled download wipes all previously accumulated state.
func TestCancel61(t *testing.T)      { testCancel(t, 61, FullSync) }
func TestCancel62(t *testing.T)      { testCancel(t, 62, FullSync) }
func TestCancel63Full(t *testing.T)  { testCancel(t, 63, FullSync) }
func TestCancel63Fast(t *testing.T)  { testCancel(t, 63, FastSync) }
func TestCancel64Full(t *testing.T)  { testCancel(t, 64, FullSync) }
func TestCancel64Fast(t *testing.T)  { testCancel(t, 64, FastSync) }
func TestCancel64Light(t *testing.T) { testCancel(t, 64, LightSync) }

func testCancel(t *testing.T, protocol int, mode SyncMode) {
	t.Parallel()

	// Create a small enough block chain to download and the tester
	targetBlocks := blockCacheLimit - 15
	if targetBlocks >= MaxHashFetch {
		targetBlocks = MaxHashFetch - 15
	}
	if targetBlocks >= MaxHeaderFetch {
		targetBlocks = MaxHeaderFetch - 15
	}
	hashes, headers, blocks, receipts := makeChain(targetBlocks, 0, genesis, nil, false)

	tester := newTester()
	defer tester.terminate()

	tester.newPeer("peer", protocol, hashes, headers, blocks, receipts)

	// Make sure canceling works with a pristine downloader
	tester.downloader.cancel()
	if !tester.downloader.queue.Idle() {
		t.Errorf("download queue not idle")
	}
	// Synchronise with the peer, but cancel afterwards
	if err := tester.sync("peer", nil, mode); err != nil {
		t.Fatalf("failed to synchronise blocks: %v", err)
	}
	tester.downloader.cancel()
	if !tester.downloader.queue.Idle() {
		t.Errorf("download queue not idle")
	}
}

// Tests that synchronisation from multiple peers works as intended (multi thread sanity test).
func TestMultiSynchronisation61(t *testing.T)      { testMultiSynchronisation(t, 61, FullSync) }
func TestMultiSynchronisation62(t *testing.T)      { testMultiSynchronisation(t, 62, FullSync) }
func TestMultiSynchronisation63Full(t *testing.T)  { testMultiSynchronisation(t, 63, FullSync) }
func TestMultiSynchronisation63Fast(t *testing.T)  { testMultiSynchronisation(t, 63, FastSync) }
func TestMultiSynchronisation64Full(t *testing.T)  { testMultiSynchronisation(t, 64, FullSync) }
func TestMultiSynchronisation64Fast(t *testing.T)  { testMultiSynchronisation(t, 64, FastSync) }
func TestMultiSynchronisation64Light(t *testing.T) { testMultiSynchronisation(t, 64, LightSync) }

func testMultiSynchronisation(t *testing.T, protocol int, mode SyncMode) {
	t.Parallel()

	// Create various peers with various parts of the chain
	targetPeers := 8
	targetBlocks := targetPeers*blockCacheLimit - 15
	hashes, headers, blocks, receipts := makeChain(targetBlocks, 0, genesis, nil, false)

	tester := newTester()
	defer tester.terminate()

	for i := 0; i < targetPeers; i++ {
		id := fmt.Sprintf("peer #%d", i)
		tester.newPeer(id, protocol, hashes[i*blockCacheLimit:], headers, blocks, receipts)
	}
	if err := tester.sync("peer #0", nil, mode); err != nil {
		t.Fatalf("failed to synchronise blocks: %v", err)
	}
	assertOwnChain(t, tester, targetBlocks+1)
}

// Tests that synchronisations behave well in multi-version protocol environments
// and not wreak havoc on other nodes in the network.
func TestMultiProtoSynchronisation61(t *testing.T)      { testMultiProtoSync(t, 61, FullSync) }
func TestMultiProtoSynchronisation62(t *testing.T)      { testMultiProtoSync(t, 62, FullSync) }
func TestMultiProtoSynchronisation63Full(t *testing.T)  { testMultiProtoSync(t, 63, FullSync) }
func TestMultiProtoSynchronisation63Fast(t *testing.T)  { testMultiProtoSync(t, 63, FastSync) }
func TestMultiProtoSynchronisation64Full(t *testing.T)  { testMultiProtoSync(t, 64, FullSync) }
func TestMultiProtoSynchronisation64Fast(t *testing.T)  { testMultiProtoSync(t, 64, FastSync) }
func TestMultiProtoSynchronisation64Light(t *testing.T) { testMultiProtoSync(t, 64, LightSync) }

func testMultiProtoSync(t *testing.T, protocol int, mode SyncMode) {
	t.Parallel()

	// Create a small enough block chain to download
	targetBlocks := blockCacheLimit - 15
	hashes, headers, blocks, receipts := makeChain(targetBlocks, 0, genesis, nil, false)

	// Create peers of every type
	tester := newTester()
	defer tester.terminate()

	tester.newPeer("peer 61", 61, hashes, nil, blocks, nil)
	tester.newPeer("peer 62", 62, hashes, headers, blocks, nil)
	tester.newPeer("peer 63", 63, hashes, headers, blocks, receipts)
	tester.newPeer("peer 64", 64, hashes, headers, blocks, receipts)

	// Synchronise with the requested peer and make sure all blocks were retrieved
	if err := tester.sync(fmt.Sprintf("peer %d", protocol), nil, mode); err != nil {
		t.Fatalf("failed to synchronise blocks: %v", err)
	}
	assertOwnChain(t, tester, targetBlocks+1)

	// Check that no peers have been dropped off
	for _, version := range []int{61, 62, 63, 64} {
		peer := fmt.Sprintf("peer %d", version)
		if _, ok := tester.peerHashes[peer]; !ok {
			t.Errorf("%s dropped", peer)
		}
	}
}

// Tests that if a block is empty (e.g. header only), no body request should be
// made, and instead the header should be assembled into a whole block in itself.
func TestEmptyShortCircuit62(t *testing.T)      { testEmptyShortCircuit(t, 62, FullSync) }
func TestEmptyShortCircuit63Full(t *testing.T)  { testEmptyShortCircuit(t, 63, FullSync) }
func TestEmptyShortCircuit63Fast(t *testing.T)  { testEmptyShortCircuit(t, 63, FastSync) }
func TestEmptyShortCircuit64Full(t *testing.T)  { testEmptyShortCircuit(t, 64, FullSync) }
func TestEmptyShortCircuit64Fast(t *testing.T)  { testEmptyShortCircuit(t, 64, FastSync) }
func TestEmptyShortCircuit64Light(t *testing.T) { testEmptyShortCircuit(t, 64, LightSync) }

func testEmptyShortCircuit(t *testing.T, protocol int, mode SyncMode) {
	t.Parallel()

	// Create a block chain to download
	targetBlocks := 2*blockCacheLimit - 15
	hashes, headers, blocks, receipts := makeChain(targetBlocks, 0, genesis, nil, false)

	tester := newTester()
	defer tester.terminate()

	tester.newPeer("peer", protocol, hashes, headers, blocks, receipts)

	// Instrument the downloader to signal body requests
	bodiesHave, receiptsHave := int32(0), int32(0)
	tester.downloader.bodyFetchHook = func(headers []*types.Header) {
		atomic.AddInt32(&bodiesHave, int32(len(headers)))
	}
	tester.downloader.receiptFetchHook = func(headers []*types.Header) {
		atomic.AddInt32(&receiptsHave, int32(len(headers)))
	}
	// Synchronise with the peer and make sure all blocks were retrieved
	if err := tester.sync("peer", nil, mode); err != nil {
		t.Fatalf("failed to synchronise blocks: %v", err)
	}
	assertOwnChain(t, tester, targetBlocks+1)

	// Validate the number of block bodies that should have been requested
	bodiesNeeded, receiptsNeeded := 0, 0
	for _, block := range blocks {
		if mode != LightSync && block != genesis && (len(block.Transactions()) > 0 || len(block.Uncles()) > 0) {
			bodiesNeeded++
		}
	}
	for hash, receipt := range receipts {
		if mode == FastSync && len(receipt) > 0 && headers[hash].Number.Uint64() <= tester.downloader.queue.fastSyncPivot {
			receiptsNeeded++
		}
	}
	if int(bodiesHave) != bodiesNeeded {
		t.Errorf("body retrieval count mismatch: have %v, want %v", bodiesHave, bodiesNeeded)
	}
	if int(receiptsHave) != receiptsNeeded {
		t.Errorf("receipt retrieval count mismatch: have %v, want %v", receiptsHave, receiptsNeeded)
	}
}

// Tests that headers are enqueued continuously, preventing malicious nodes from
// stalling the downloader by feeding gapped header chains.
func TestMissingHeaderAttack62(t *testing.T)      { testMissingHeaderAttack(t, 62, FullSync) }
func TestMissingHeaderAttack63Full(t *testing.T)  { testMissingHeaderAttack(t, 63, FullSync) }
func TestMissingHeaderAttack63Fast(t *testing.T)  { testMissingHeaderAttack(t, 63, FastSync) }
func TestMissingHeaderAttack64Full(t *testing.T)  { testMissingHeaderAttack(t, 64, FullSync) }
func TestMissingHeaderAttack64Fast(t *testing.T)  { testMissingHeaderAttack(t, 64, FastSync) }
func TestMissingHeaderAttack64Light(t *testing.T) { testMissingHeaderAttack(t, 64, LightSync) }

func testMissingHeaderAttack(t *testing.T, protocol int, mode SyncMode) {
	t.Parallel()

	// Create a small enough block chain to download
	targetBlocks := blockCacheLimit - 15
	hashes, headers, blocks, receipts := makeChain(targetBlocks, 0, genesis, nil, false)

	tester := newTester()
	defer tester.terminate()

	// Attempt a full sync with an attacker feeding gapped headers
	tester.newPeer("attack", protocol, hashes, headers, blocks, receipts)
	missing := targetBlocks / 2
	delete(tester.peerHeaders["attack"], hashes[missing])

	if err := tester.sync("attack", nil, mode); err == nil {
		t.Fatalf("succeeded attacker synchronisation")
	}
	// Synchronise with the valid peer and make sure sync succeeds
	tester.newPeer("valid", protocol, hashes, headers, blocks, receipts)
	if err := tester.sync("valid", nil, mode); err != nil {
		t.Fatalf("failed to synchronise blocks: %v", err)
	}
	assertOwnChain(t, tester, targetBlocks+1)
}

// Tests that if requested headers are shifted (i.e. first is missing), the queue
// detects the invalid numbering.
func TestShiftedHeaderAttack62(t *testing.T)      { testShiftedHeaderAttack(t, 62, FullSync) }
func TestShiftedHeaderAttack63Full(t *testing.T)  { testShiftedHeaderAttack(t, 63, FullSync) }
func TestShiftedHeaderAttack63Fast(t *testing.T)  { testShiftedHeaderAttack(t, 63, FastSync) }
func TestShiftedHeaderAttack64Full(t *testing.T)  { testShiftedHeaderAttack(t, 64, FullSync) }
func TestShiftedHeaderAttack64Fast(t *testing.T)  { testShiftedHeaderAttack(t, 64, FastSync) }
func TestShiftedHeaderAttack64Light(t *testing.T) { testShiftedHeaderAttack(t, 64, LightSync) }

func testShiftedHeaderAttack(t *testing.T, protocol int, mode SyncMode) {
	// Create a small enough block chain to download
	targetBlocks := blockCacheLimit - 15
	hashes, headers, blocks, receipts := makeChain(targetBlocks, 0, genesis, nil, false)

	tester := newTester()
	defer tester.terminate()

	// Attempt a full sync with an attacker feeding shifted headers
	tester.newPeer("attack", protocol, hashes, headers, blocks, receipts)
	delete(tester.peerHeaders["attack"], hashes[len(hashes)-2])
	delete(tester.peerBlocks["attack"], hashes[len(hashes)-2])
	delete(tester.peerReceipts["attack"], hashes[len(hashes)-2])

	if err := tester.sync("attack", nil, mode); err == nil {
		t.Fatalf("succeeded attacker synchronisation")
	}
	// Synchronise with the valid peer and make sure sync succeeds
	tester.newPeer("valid", protocol, hashes, headers, blocks, receipts)
	if err := tester.sync("valid", nil, mode); err != nil {
		t.Fatalf("failed to synchronise blocks: %v", err)
	}
	assertOwnChain(t, tester, targetBlocks+1)
}

// Tests that upon detecting an invalid header, the recent ones are rolled back
// for various failure scenarios. Afterwards a full sync is attempted to make
// sure no state was corrupted.
func TestInvalidHeaderRollback63Fast(t *testing.T)  { testInvalidHeaderRollback(t, 63, FastSync) }
func TestInvalidHeaderRollback64Fast(t *testing.T)  { testInvalidHeaderRollback(t, 64, FastSync) }
func TestInvalidHeaderRollback64Light(t *testing.T) { testInvalidHeaderRollback(t, 64, LightSync) }

func testInvalidHeaderRollback(t *testing.T, protocol int, mode SyncMode) {
	// Create a small enough block chain to download
	targetBlocks := 3*fsHeaderSafetyNet + fsMinFullBlocks
	hashes, headers, blocks, receipts := makeChain(targetBlocks, 0, genesis, nil, false)

	tester := newTester()
	defer tester.terminate()

	// Attempt to sync with an attacker that feeds junk during the fast sync phase.
	// This should result in the last fsHeaderSafetyNet headers being rolled back.
	tester.newPeer("fast-attack", protocol, hashes, headers, blocks, receipts)
	missing := fsHeaderSafetyNet + MaxHeaderFetch + 1
	delete(tester.peerHeaders["fast-attack"], hashes[len(hashes)-missing])

	if err := tester.sync("fast-attack", nil, mode); err == nil {
		t.Fatalf("succeeded fast attacker synchronisation")
	}
	if head := tester.headHeader().Number.Int64(); int(head) > MaxHeaderFetch {
		t.Errorf("rollback head mismatch: have %v, want at most %v", head, MaxHeaderFetch)
	}
	// Attempt to sync with an attacker that feeds junk during the block import phase.
	// This should result in both the last fsHeaderSafetyNet number of headers being
	// rolled back, and also the pivot point being reverted to a non-block status.
	tester.newPeer("block-attack", protocol, hashes, headers, blocks, receipts)
	missing = 3*fsHeaderSafetyNet + MaxHeaderFetch + 1
	delete(tester.peerHeaders["fast-attack"], hashes[len(hashes)-missing]) // Make sure the fast-attacker doesn't fill in
	delete(tester.peerHeaders["block-attack"], hashes[len(hashes)-missing])

	if err := tester.sync("block-attack", nil, mode); err == nil {
		t.Fatalf("succeeded block attacker synchronisation")
	}
	if head := tester.headHeader().Number.Int64(); int(head) > 2*fsHeaderSafetyNet+MaxHeaderFetch {
		t.Errorf("rollback head mismatch: have %v, want at most %v", head, 2*fsHeaderSafetyNet+MaxHeaderFetch)
	}
	if mode == FastSync {
		if head := tester.headBlock().NumberU64(); head != 0 {
			t.Errorf("fast sync pivot block #%d not rolled back", head)
		}
	}
	// Attempt to sync with an attacker that withholds promised blocks after the
	// fast sync pivot point. This could be a trial to leave the node with a bad
	// but already imported pivot block.
	tester.newPeer("withhold-attack", protocol, hashes, headers, blocks, receipts)
	missing = 3*fsHeaderSafetyNet + MaxHeaderFetch + 1

	tester.downloader.fsPivotFails = 0
	tester.downloader.syncInitHook = func(uint64, uint64) {
		for i := missing; i <= len(hashes); i++ {
			delete(tester.peerHeaders["withhold-attack"], hashes[len(hashes)-i])
		}
		tester.downloader.syncInitHook = nil
	}

	if err := tester.sync("withhold-attack", nil, mode); err == nil {
		t.Fatalf("succeeded withholding attacker synchronisation")
	}
	if head := tester.headHeader().Number.Int64(); int(head) > 2*fsHeaderSafetyNet+MaxHeaderFetch {
		t.Errorf("rollback head mismatch: have %v, want at most %v", head, 2*fsHeaderSafetyNet+MaxHeaderFetch)
	}
	if mode == FastSync {
		if head := tester.headBlock().NumberU64(); head != 0 {
			t.Errorf("fast sync pivot block #%d not rolled back", head)
		}
	}
	tester.downloader.fsPivotFails = fsCriticalTrials

	// Synchronise with the valid peer and make sure sync succeeds. Since the last
	// rollback should also disable fast syncing for this process, verify that we
	// did a fresh full sync. Note, we can't assert anything about the receipts
	// since we won't purge the database of them, hence we can't use assertOwnChain.
	tester.newPeer("valid", protocol, hashes, headers, blocks, receipts)
	if err := tester.sync("valid", nil, mode); err != nil {
		t.Fatalf("failed to synchronise blocks: %v", err)
	}
	if hs := len(tester.ownHeaders); hs != len(headers) {
		t.Fatalf("synchronised headers mismatch: have %v, want %v", hs, len(headers))
	}
	if mode != LightSync {
		if bs := len(tester.ownBlocks); bs != len(blocks) {
			t.Fatalf("synchronised blocks mismatch: have %v, want %v", bs, len(blocks))
		}
	}
}

// Tests that a peer advertising an high TD doesn't get to stall the downloader
// afterwards by not sending any useful hashes.
func TestHighTDStarvationAttack61(t *testing.T)      { testHighTDStarvationAttack(t, 61, FullSync) }
func TestHighTDStarvationAttack62(t *testing.T)      { testHighTDStarvationAttack(t, 62, FullSync) }
func TestHighTDStarvationAttack63Full(t *testing.T)  { testHighTDStarvationAttack(t, 63, FullSync) }
func TestHighTDStarvationAttack63Fast(t *testing.T)  { testHighTDStarvationAttack(t, 63, FastSync) }
func TestHighTDStarvationAttack64Full(t *testing.T)  { testHighTDStarvationAttack(t, 64, FullSync) }
func TestHighTDStarvationAttack64Fast(t *testing.T)  { testHighTDStarvationAttack(t, 64, FastSync) }
func TestHighTDStarvationAttack64Light(t *testing.T) { testHighTDStarvationAttack(t, 64, LightSync) }

func testHighTDStarvationAttack(t *testing.T, protocol int, mode SyncMode) {
	t.Parallel()

	tester := newTester()
	defer tester.terminate()

	hashes, headers, blocks, receipts := makeChain(0, 0, genesis, nil, false)
	tester.newPeer("attack", protocol, []common.Hash{hashes[0]}, headers, blocks, receipts)

	if err := tester.sync("attack", big.NewInt(1000000), mode); err != errStallingPeer {
		t.Fatalf("synchronisation error mismatch: have %v, want %v", err, errStallingPeer)
	}
}

// Tests that misbehaving peers are disconnected, whilst behaving ones are not.
func TestBlockHeaderAttackerDropping61(t *testing.T) { testBlockHeaderAttackerDropping(t, 61) }
func TestBlockHeaderAttackerDropping62(t *testing.T) { testBlockHeaderAttackerDropping(t, 62) }
func TestBlockHeaderAttackerDropping63(t *testing.T) { testBlockHeaderAttackerDropping(t, 63) }
func TestBlockHeaderAttackerDropping64(t *testing.T) { testBlockHeaderAttackerDropping(t, 64) }

func testBlockHeaderAttackerDropping(t *testing.T, protocol int) {
	// Define the disconnection requirement for individual hash fetch errors
	tests := []struct {
		result error
		drop   bool
	}{
		{nil, false},                        // Sync succeeded, all is well
		{errBusy, false},                    // Sync is already in progress, no problem
		{errUnknownPeer, false},             // Peer is unknown, was already dropped, don't double drop
		{errBadPeer, true},                  // Peer was deemed bad for some reason, drop it
		{errStallingPeer, true},             // Peer was detected to be stalling, drop it
		{errNoPeers, false},                 // No peers to download from, soft race, no issue
		{errTimeout, true},                  // No hashes received in due time, drop the peer
		{errEmptyHashSet, true},             // No hashes were returned as a response, drop as it's a dead end
		{errEmptyHeaderSet, true},           // No headers were returned as a response, drop as it's a dead end
		{errPeersUnavailable, true},         // Nobody had the advertised blocks, drop the advertiser
		{errInvalidAncestor, true},          // Agreed upon ancestor is not acceptable, drop the chain rewriter
		{errInvalidChain, true},             // Hash chain was detected as invalid, definitely drop
		{errInvalidBlock, false},            // A bad peer was detected, but not the sync origin
		{errInvalidBody, false},             // A bad peer was detected, but not the sync origin
		{errInvalidReceipt, false},          // A bad peer was detected, but not the sync origin
		{errCancelHashFetch, false},         // Synchronisation was canceled, origin may be innocent, don't drop
		{errCancelBlockFetch, false},        // Synchronisation was canceled, origin may be innocent, don't drop
		{errCancelHeaderFetch, false},       // Synchronisation was canceled, origin may be innocent, don't drop
		{errCancelBodyFetch, false},         // Synchronisation was canceled, origin may be innocent, don't drop
		{errCancelReceiptFetch, false},      // Synchronisation was canceled, origin may be innocent, don't drop
		{errCancelHeaderProcessing, false},  // Synchronisation was canceled, origin may be innocent, don't drop
		{errCancelContentProcessing, false}, // Synchronisation was canceled, origin may be innocent, don't drop
	}
	// Run the tests and check disconnection status
	tester := newTester()
	defer tester.terminate()

	for i, tt := range tests {
		// Register a new peer and ensure it's presence
		id := fmt.Sprintf("test %d", i)
		if err := tester.newPeer(id, protocol, []common.Hash{genesis.Hash()}, nil, nil, nil); err != nil {
			t.Fatalf("test %d: failed to register new peer: %v", i, err)
		}
		if _, ok := tester.peerHashes[id]; !ok {
			t.Fatalf("test %d: registered peer not found", i)
		}
		// Simulate a synchronisation and check the required result
		tester.downloader.synchroniseMock = func(string, common.Hash) error { return tt.result }

		tester.downloader.Synchronise(id, genesis.Hash(), big.NewInt(1000), FullSync)
		if _, ok := tester.peerHashes[id]; !ok != tt.drop {
			t.Errorf("test %d: peer drop mismatch for %v: have %v, want %v", i, tt.result, !ok, tt.drop)
		}
	}
}

// Tests that synchronisation progress (origin block number, current block number
// and highest block number) is tracked and updated correctly.
func TestSyncProgress61(t *testing.T)      { testSyncProgress(t, 61, FullSync) }
func TestSyncProgress62(t *testing.T)      { testSyncProgress(t, 62, FullSync) }
func TestSyncProgress63Full(t *testing.T)  { testSyncProgress(t, 63, FullSync) }
func TestSyncProgress63Fast(t *testing.T)  { testSyncProgress(t, 63, FastSync) }
func TestSyncProgress64Full(t *testing.T)  { testSyncProgress(t, 64, FullSync) }
func TestSyncProgress64Fast(t *testing.T)  { testSyncProgress(t, 64, FastSync) }
func TestSyncProgress64Light(t *testing.T) { testSyncProgress(t, 64, LightSync) }

func testSyncProgress(t *testing.T, protocol int, mode SyncMode) {
	t.Parallel()

	// Create a small enough block chain to download
	targetBlocks := blockCacheLimit - 15
	hashes, headers, blocks, receipts := makeChain(targetBlocks, 0, genesis, nil, false)

	// Set a sync init hook to catch progress changes
	starting := make(chan struct{})
	progress := make(chan struct{})

	tester := newTester()
	defer tester.terminate()

	tester.downloader.syncInitHook = func(origin, latest uint64) {
		starting <- struct{}{}
		<-progress
	}
	// Retrieve the sync progress and ensure they are zero (pristine sync)
	if origin, current, latest, _, _ := tester.downloader.Progress(); origin != 0 || current != 0 || latest != 0 {
		t.Fatalf("Pristine progress mismatch: have %v/%v/%v, want %v/%v/%v", origin, current, latest, 0, 0, 0)
	}
	// Synchronise half the blocks and check initial progress
	tester.newPeer("peer-half", protocol, hashes[targetBlocks/2:], headers, blocks, receipts)
	pending := new(sync.WaitGroup)
	pending.Add(1)

	go func() {
		defer pending.Done()
		if err := tester.sync("peer-half", nil, mode); err != nil {
			t.Fatalf("failed to synchronise blocks: %v", err)
		}
	}()
	<-starting
	if origin, current, latest, _, _ := tester.downloader.Progress(); origin != 0 || current != 0 || latest != uint64(targetBlocks/2+1) {
		t.Fatalf("Initial progress mismatch: have %v/%v/%v, want %v/%v/%v", origin, current, latest, 0, 0, targetBlocks/2+1)
	}
	progress <- struct{}{}
	pending.Wait()

	// Synchronise all the blocks and check continuation progress
	tester.newPeer("peer-full", protocol, hashes, headers, blocks, receipts)
	pending.Add(1)

	go func() {
		defer pending.Done()
		if err := tester.sync("peer-full", nil, mode); err != nil {
			t.Fatalf("failed to synchronise blocks: %v", err)
		}
	}()
	<-starting
	if origin, current, latest, _, _ := tester.downloader.Progress(); origin != uint64(targetBlocks/2+1) || current != uint64(targetBlocks/2+1) || latest != uint64(targetBlocks) {
		t.Fatalf("Completing progress mismatch: have %v/%v/%v, want %v/%v/%v", origin, current, latest, targetBlocks/2+1, targetBlocks/2+1, targetBlocks)
	}
	progress <- struct{}{}
	pending.Wait()

	// Check final progress after successful sync
	if origin, current, latest, _, _ := tester.downloader.Progress(); origin != uint64(targetBlocks/2+1) || current != uint64(targetBlocks) || latest != uint64(targetBlocks) {
		t.Fatalf("Final progress mismatch: have %v/%v/%v, want %v/%v/%v", origin, current, latest, targetBlocks/2+1, targetBlocks, targetBlocks)
	}
}

// Tests that synchronisation progress (origin block number and highest block
// number) is tracked and updated correctly in case of a fork (or manual head
// revertal).
func TestForkedSyncProgress61(t *testing.T)      { testForkedSyncProgress(t, 61, FullSync) }
func TestForkedSyncProgress62(t *testing.T)      { testForkedSyncProgress(t, 62, FullSync) }
func TestForkedSyncProgress63Full(t *testing.T)  { testForkedSyncProgress(t, 63, FullSync) }
func TestForkedSyncProgress63Fast(t *testing.T)  { testForkedSyncProgress(t, 63, FastSync) }
func TestForkedSyncProgress64Full(t *testing.T)  { testForkedSyncProgress(t, 64, FullSync) }
func TestForkedSyncProgress64Fast(t *testing.T)  { testForkedSyncProgress(t, 64, FastSync) }
func TestForkedSyncProgress64Light(t *testing.T) { testForkedSyncProgress(t, 64, LightSync) }

func testForkedSyncProgress(t *testing.T, protocol int, mode SyncMode) {
	t.Parallel()

	// Create a forked chain to simulate origin revertal
	common, fork := MaxHashFetch, 2*MaxHashFetch
	hashesA, hashesB, headersA, headersB, blocksA, blocksB, receiptsA, receiptsB := makeChainFork(common+fork, fork, genesis, nil, true)

	// Set a sync init hook to catch progress changes
	starting := make(chan struct{})
	progress := make(chan struct{})

	tester := newTester()
	defer tester.terminate()

	tester.downloader.syncInitHook = func(origin, latest uint64) {
		starting <- struct{}{}
		<-progress
	}
	// Retrieve the sync progress and ensure they are zero (pristine sync)
	if origin, current, latest, _, _ := tester.downloader.Progress(); origin != 0 || current != 0 || latest != 0 {
		t.Fatalf("Pristine progress mismatch: have %v/%v/%v, want %v/%v/%v", origin, current, latest, 0, 0, 0)
	}
	// Synchronise with one of the forks and check progress
	tester.newPeer("fork A", protocol, hashesA, headersA, blocksA, receiptsA)
	pending := new(sync.WaitGroup)
	pending.Add(1)

	go func() {
		defer pending.Done()
		if err := tester.sync("fork A", nil, mode); err != nil {
			t.Fatalf("failed to synchronise blocks: %v", err)
		}
	}()
	<-starting
	if origin, current, latest, _, _ := tester.downloader.Progress(); origin != 0 || current != 0 || latest != uint64(len(hashesA)-1) {
		t.Fatalf("Initial progress mismatch: have %v/%v/%v, want %v/%v/%v", origin, current, latest, 0, 0, len(hashesA)-1)
	}
	progress <- struct{}{}
	pending.Wait()

	// Simulate a successful sync above the fork
	tester.downloader.syncStatsChainOrigin = tester.downloader.syncStatsChainHeight

	// Synchronise with the second fork and check progress resets
	tester.newPeer("fork B", protocol, hashesB, headersB, blocksB, receiptsB)
	pending.Add(1)

	go func() {
		defer pending.Done()
		if err := tester.sync("fork B", nil, mode); err != nil {
			t.Fatalf("failed to synchronise blocks: %v", err)
		}
	}()
	<-starting
	if origin, current, latest, _, _ := tester.downloader.Progress(); origin != uint64(common) || current != uint64(len(hashesA)-1) || latest != uint64(len(hashesB)-1) {
		t.Fatalf("Forking progress mismatch: have %v/%v/%v, want %v/%v/%v", origin, current, latest, common, len(hashesA)-1, len(hashesB)-1)
	}
	progress <- struct{}{}
	pending.Wait()

	// Check final progress after successful sync
	if origin, current, latest, _, _ := tester.downloader.Progress(); origin != uint64(common) || current != uint64(len(hashesB)-1) || latest != uint64(len(hashesB)-1) {
		t.Fatalf("Final progress mismatch: have %v/%v/%v, want %v/%v/%v", origin, current, latest, common, len(hashesB)-1, len(hashesB)-1)
	}
}

// Tests that if synchronisation is aborted due to some failure, then the progress
// origin is not updated in the next sync cycle, as it should be considered the
// continuation of the previous sync and not a new instance.
func TestFailedSyncProgress61(t *testing.T)      { testFailedSyncProgress(t, 61, FullSync) }
func TestFailedSyncProgress62(t *testing.T)      { testFailedSyncProgress(t, 62, FullSync) }
func TestFailedSyncProgress63Full(t *testing.T)  { testFailedSyncProgress(t, 63, FullSync) }
func TestFailedSyncProgress63Fast(t *testing.T)  { testFailedSyncProgress(t, 63, FastSync) }
func TestFailedSyncProgress64Full(t *testing.T)  { testFailedSyncProgress(t, 64, FullSync) }
func TestFailedSyncProgress64Fast(t *testing.T)  { testFailedSyncProgress(t, 64, FastSync) }
func TestFailedSyncProgress64Light(t *testing.T) { testFailedSyncProgress(t, 64, LightSync) }

func testFailedSyncProgress(t *testing.T, protocol int, mode SyncMode) {
	t.Parallel()

	// Create a small enough block chain to download
	targetBlocks := blockCacheLimit - 15
	hashes, headers, blocks, receipts := makeChain(targetBlocks, 0, genesis, nil, false)

	// Set a sync init hook to catch progress changes
	starting := make(chan struct{})
	progress := make(chan struct{})

	tester := newTester()
	defer tester.terminate()

	tester.downloader.syncInitHook = func(origin, latest uint64) {
		starting <- struct{}{}
		<-progress
	}
	// Retrieve the sync progress and ensure they are zero (pristine sync)
	if origin, current, latest, _, _ := tester.downloader.Progress(); origin != 0 || current != 0 || latest != 0 {
		t.Fatalf("Pristine progress mismatch: have %v/%v/%v, want %v/%v/%v", origin, current, latest, 0, 0, 0)
	}
	// Attempt a full sync with a faulty peer
	tester.newPeer("faulty", protocol, hashes, headers, blocks, receipts)
	missing := targetBlocks / 2
	delete(tester.peerHeaders["faulty"], hashes[missing])
	delete(tester.peerBlocks["faulty"], hashes[missing])
	delete(tester.peerReceipts["faulty"], hashes[missing])

	pending := new(sync.WaitGroup)
	pending.Add(1)

	go func() {
		defer pending.Done()
		if err := tester.sync("faulty", nil, mode); err == nil {
			t.Fatalf("succeeded faulty synchronisation")
		}
	}()
	<-starting
	if origin, current, latest, _, _ := tester.downloader.Progress(); origin != 0 || current != 0 || latest != uint64(targetBlocks) {
		t.Fatalf("Initial progress mismatch: have %v/%v/%v, want %v/%v/%v", origin, current, latest, 0, 0, targetBlocks)
	}
	progress <- struct{}{}
	pending.Wait()

	// Synchronise with a good peer and check that the progress origin remind the same after a failure
	tester.newPeer("valid", protocol, hashes, headers, blocks, receipts)
	pending.Add(1)

	go func() {
		defer pending.Done()
		if err := tester.sync("valid", nil, mode); err != nil {
			t.Fatalf("failed to synchronise blocks: %v", err)
		}
	}()
	<-starting
	if origin, current, latest, _, _ := tester.downloader.Progress(); origin != 0 || current > uint64(targetBlocks/2) || latest != uint64(targetBlocks) {
		t.Fatalf("Completing progress mismatch: have %v/%v/%v, want %v/0-%v/%v", origin, current, latest, 0, targetBlocks/2, targetBlocks)
	}
	progress <- struct{}{}
	pending.Wait()

	// Check final progress after successful sync
	if origin, current, latest, _, _ := tester.downloader.Progress(); origin > uint64(targetBlocks/2) || current != uint64(targetBlocks) || latest != uint64(targetBlocks) {
		t.Fatalf("Final progress mismatch: have %v/%v/%v, want 0-%v/%v/%v", origin, current, latest, targetBlocks/2, targetBlocks, targetBlocks)
	}
}

// Tests that if an attacker fakes a chain height, after the attack is detected,
// the progress height is successfully reduced at the next sync invocation.
func TestFakedSyncProgress61(t *testing.T)      { testFakedSyncProgress(t, 61, FullSync) }
func TestFakedSyncProgress62(t *testing.T)      { testFakedSyncProgress(t, 62, FullSync) }
func TestFakedSyncProgress63Full(t *testing.T)  { testFakedSyncProgress(t, 63, FullSync) }
func TestFakedSyncProgress63Fast(t *testing.T)  { testFakedSyncProgress(t, 63, FastSync) }
func TestFakedSyncProgress64Full(t *testing.T)  { testFakedSyncProgress(t, 64, FullSync) }
func TestFakedSyncProgress64Fast(t *testing.T)  { testFakedSyncProgress(t, 64, FastSync) }
func TestFakedSyncProgress64Light(t *testing.T) { testFakedSyncProgress(t, 64, LightSync) }

func testFakedSyncProgress(t *testing.T, protocol int, mode SyncMode) {
	t.Parallel()

	// Create a small block chain
	targetBlocks := blockCacheLimit - 15
	hashes, headers, blocks, receipts := makeChain(targetBlocks+3, 0, genesis, nil, false)

	// Set a sync init hook to catch progress changes
	starting := make(chan struct{})
	progress := make(chan struct{})

	tester := newTester()
	defer tester.terminate()

	tester.downloader.syncInitHook = func(origin, latest uint64) {
		starting <- struct{}{}
		<-progress
	}
	// Retrieve the sync progress and ensure they are zero (pristine sync)
	if origin, current, latest, _, _ := tester.downloader.Progress(); origin != 0 || current != 0 || latest != 0 {
		t.Fatalf("Pristine progress mismatch: have %v/%v/%v, want %v/%v/%v", origin, current, latest, 0, 0, 0)
	}
	//  Create and sync with an attacker that promises a higher chain than available
	tester.newPeer("attack", protocol, hashes, headers, blocks, receipts)
	for i := 1; i < 3; i++ {
		delete(tester.peerHeaders["attack"], hashes[i])
		delete(tester.peerBlocks["attack"], hashes[i])
		delete(tester.peerReceipts["attack"], hashes[i])
	}

	pending := new(sync.WaitGroup)
	pending.Add(1)

	go func() {
		defer pending.Done()
		if err := tester.sync("attack", nil, mode); err == nil {
			t.Fatalf("succeeded attacker synchronisation")
		}
	}()
	<-starting
	if origin, current, latest, _, _ := tester.downloader.Progress(); origin != 0 || current != 0 || latest != uint64(targetBlocks+3) {
		t.Fatalf("Initial progress mismatch: have %v/%v/%v, want %v/%v/%v", origin, current, latest, 0, 0, targetBlocks+3)
	}
	progress <- struct{}{}
	pending.Wait()

	// Synchronise with a good peer and check that the progress height has been reduced to the true value
	tester.newPeer("valid", protocol, hashes[3:], headers, blocks, receipts)
	pending.Add(1)

	go func() {
		defer pending.Done()
		if err := tester.sync("valid", nil, mode); err != nil {
			t.Fatalf("failed to synchronise blocks: %v", err)
		}
	}()
	<-starting
	if origin, current, latest, _, _ := tester.downloader.Progress(); origin != 0 || current > uint64(targetBlocks) || latest != uint64(targetBlocks) {
		t.Fatalf("Completing progress mismatch: have %v/%v/%v, want %v/0-%v/%v", origin, current, latest, 0, targetBlocks, targetBlocks)
	}
	progress <- struct{}{}
	pending.Wait()

	// Check final progress after successful sync
	if origin, current, latest, _, _ := tester.downloader.Progress(); origin > uint64(targetBlocks) || current != uint64(targetBlocks) || latest != uint64(targetBlocks) {
		t.Fatalf("Final progress mismatch: have %v/%v/%v, want 0-%v/%v/%v", origin, current, latest, targetBlocks, targetBlocks, targetBlocks)
	}
}

// This test reproduces an issue where unexpected deliveries would
// block indefinitely if they arrived at the right time.
func TestDeliverHeadersHang62(t *testing.T)      { testDeliverHeadersHang(t, 62, FullSync) }
func TestDeliverHeadersHang63Full(t *testing.T)  { testDeliverHeadersHang(t, 63, FullSync) }
func TestDeliverHeadersHang63Fast(t *testing.T)  { testDeliverHeadersHang(t, 63, FastSync) }
func TestDeliverHeadersHang64Full(t *testing.T)  { testDeliverHeadersHang(t, 64, FullSync) }
func TestDeliverHeadersHang64Fast(t *testing.T)  { testDeliverHeadersHang(t, 64, FastSync) }
func TestDeliverHeadersHang64Light(t *testing.T) { testDeliverHeadersHang(t, 64, LightSync) }

func testDeliverHeadersHang(t *testing.T, protocol int, mode SyncMode) {
	t.Parallel()
	hashes, headers, blocks, receipts := makeChain(5, 0, genesis, nil, false)
	fakeHeads := []*types.Header{{}, {}, {}, {}}
	for i := 0; i < 200; i++ {
		tester := newTester()
		tester.newPeer("peer", protocol, hashes, headers, blocks, receipts)
		// Whenever the downloader requests headers, flood it with
		// a lot of unrequested header deliveries.
		tester.downloader.peers.peers["peer"].getAbsHeaders = func(from uint64, count, skip int, reverse bool) error {
			deliveriesDone := make(chan struct{}, 500)
			for i := 0; i < cap(deliveriesDone); i++ {
				peer := fmt.Sprintf("fake-peer%d", i)
				go func() {
					tester.downloader.DeliverHeaders(peer, fakeHeads)
					deliveriesDone <- struct{}{}
				}()
			}
			// Deliver the actual requested headers.
			impl := tester.peerGetAbsHeadersFn("peer", 0)
			go impl(from, count, skip, reverse)
			// None of the extra deliveries should block.
			timeout := time.After(15 * time.Second)
			for i := 0; i < cap(deliveriesDone); i++ {
				select {
				case <-deliveriesDone:
				case <-timeout:
					panic("blocked")
				}
			}
			return nil
		}
		if err := tester.sync("peer", nil, mode); err != nil {
			t.Errorf("sync failed: %v", err)
		}
		tester.terminate()
	}
}

// Tests that if fast sync aborts in the critical section, it can restart a few
// times before giving up.
func TestFastCriticalRestarts63(t *testing.T) { testFastCriticalRestarts(t, 63) }
func TestFastCriticalRestarts64(t *testing.T) { testFastCriticalRestarts(t, 64) }

func testFastCriticalRestarts(t *testing.T, protocol int) {
	t.Parallel()

	// Create a large enough blockchin to actually fast sync on
	targetBlocks := fsMinFullBlocks + 2*fsPivotInterval - 15
	hashes, headers, blocks, receipts := makeChain(targetBlocks, 0, genesis, nil, false)

	// Create a tester peer with the critical section state roots missing (force failures)
	tester := newTester()
	defer tester.terminate()

	tester.newPeer("peer", protocol, hashes, headers, blocks, receipts)
	for i := 0; i < fsPivotInterval; i++ {
		tester.peerMissingStates["peer"][headers[hashes[fsMinFullBlocks+i]].Root] = true
	}
	tester.downloader.dropPeer = func(id string) {} // We reuse the same "faulty" peer throughout the test

	// Synchronise with the peer a few times and make sure they fail until the retry limit
	for i := 0; i < fsCriticalTrials; i++ {
		// Attempt a sync and ensure it fails properly
		if err := tester.sync("peer", nil, FastSync); err == nil {
			t.Fatalf("failing fast sync succeeded: %v", err)
		}
		time.Sleep(100 * time.Millisecond) // Make sure no in-flight requests remain

		// If it's the first failure, pivot should be locked => reenable all others to detect pivot changes
		if i == 0 {
			tester.lock.Lock()
			tester.peerMissingStates["peer"] = map[common.Hash]bool{tester.downloader.fsPivotLock.Root: true}
			tester.lock.Unlock()
		}
	}
	// Retry limit exhausted, downloader will switch to full sync, should succeed
	if err := tester.sync("peer", nil, FastSync); err != nil {
		t.Fatalf("failed to synchronise blocks in slow sync: %v", err)
	}
	assertOwnChain(t, tester, targetBlocks+1)
}
