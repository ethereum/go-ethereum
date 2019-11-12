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

package les

import (
	"encoding/binary"
	"math/big"
	"math/rand"
	"testing"
	"time"

	"github.com/maticnetwork/bor/common"
	"github.com/maticnetwork/bor/common/mclock"
	"github.com/maticnetwork/bor/consensus/ethash"
	"github.com/maticnetwork/bor/core"
	"github.com/maticnetwork/bor/core/rawdb"
	"github.com/maticnetwork/bor/core/types"
	"github.com/maticnetwork/bor/crypto"
	"github.com/maticnetwork/bor/eth/downloader"
	"github.com/maticnetwork/bor/light"
	"github.com/maticnetwork/bor/p2p"
	"github.com/maticnetwork/bor/params"
	"github.com/maticnetwork/bor/rlp"
	"github.com/maticnetwork/bor/trie"
)

func expectResponse(r p2p.MsgReader, msgcode, reqID, bv uint64, data interface{}) error {
	type resp struct {
		ReqID, BV uint64
		Data      interface{}
	}
	return p2p.ExpectMsg(r, msgcode, resp{reqID, bv, data})
}

// Tests that block headers can be retrieved from a remote chain based on user queries.
func TestGetBlockHeadersLes2(t *testing.T) { testGetBlockHeaders(t, 2) }

func testGetBlockHeaders(t *testing.T, protocol int) {
	server, tearDown := newServerEnv(t, downloader.MaxHashFetch+15, protocol, nil)
	defer tearDown()
	bc := server.pm.blockchain.(*core.BlockChain)

	// Create a "random" unknown hash for testing
	var unknown common.Hash
	for i := range unknown {
		unknown[i] = byte(i)
	}
	// Create a batch of tests for various scenarios
	limit := uint64(MaxHeaderFetch)
	tests := []struct {
		query  *getBlockHeadersData // The query to execute for header retrieval
		expect []common.Hash        // The hashes of the block whose headers are expected
	}{
		// A single random block should be retrievable by hash and number too
		{
			&getBlockHeadersData{Origin: hashOrNumber{Hash: bc.GetBlockByNumber(limit / 2).Hash()}, Amount: 1},
			[]common.Hash{bc.GetBlockByNumber(limit / 2).Hash()},
		}, {
			&getBlockHeadersData{Origin: hashOrNumber{Number: limit / 2}, Amount: 1},
			[]common.Hash{bc.GetBlockByNumber(limit / 2).Hash()},
		},
		// Multiple headers should be retrievable in both directions
		{
			&getBlockHeadersData{Origin: hashOrNumber{Number: limit / 2}, Amount: 3},
			[]common.Hash{
				bc.GetBlockByNumber(limit / 2).Hash(),
				bc.GetBlockByNumber(limit/2 + 1).Hash(),
				bc.GetBlockByNumber(limit/2 + 2).Hash(),
			},
		}, {
			&getBlockHeadersData{Origin: hashOrNumber{Number: limit / 2}, Amount: 3, Reverse: true},
			[]common.Hash{
				bc.GetBlockByNumber(limit / 2).Hash(),
				bc.GetBlockByNumber(limit/2 - 1).Hash(),
				bc.GetBlockByNumber(limit/2 - 2).Hash(),
			},
		},
		// Multiple headers with skip lists should be retrievable
		{
			&getBlockHeadersData{Origin: hashOrNumber{Number: limit / 2}, Skip: 3, Amount: 3},
			[]common.Hash{
				bc.GetBlockByNumber(limit / 2).Hash(),
				bc.GetBlockByNumber(limit/2 + 4).Hash(),
				bc.GetBlockByNumber(limit/2 + 8).Hash(),
			},
		}, {
			&getBlockHeadersData{Origin: hashOrNumber{Number: limit / 2}, Skip: 3, Amount: 3, Reverse: true},
			[]common.Hash{
				bc.GetBlockByNumber(limit / 2).Hash(),
				bc.GetBlockByNumber(limit/2 - 4).Hash(),
				bc.GetBlockByNumber(limit/2 - 8).Hash(),
			},
		},
		// The chain endpoints should be retrievable
		{
			&getBlockHeadersData{Origin: hashOrNumber{Number: 0}, Amount: 1},
			[]common.Hash{bc.GetBlockByNumber(0).Hash()},
		}, {
			&getBlockHeadersData{Origin: hashOrNumber{Number: bc.CurrentBlock().NumberU64()}, Amount: 1},
			[]common.Hash{bc.CurrentBlock().Hash()},
		},
		// Ensure protocol limits are honored
		/*{
			&getBlockHeadersData{Origin: hashOrNumber{Number: bc.CurrentBlock().NumberU64() - 1}, Amount: limit + 10, Reverse: true},
			bc.GetBlockHashesFromHash(bc.CurrentBlock().Hash(), limit),
		},*/
		// Check that requesting more than available is handled gracefully
		{
			&getBlockHeadersData{Origin: hashOrNumber{Number: bc.CurrentBlock().NumberU64() - 4}, Skip: 3, Amount: 3},
			[]common.Hash{
				bc.GetBlockByNumber(bc.CurrentBlock().NumberU64() - 4).Hash(),
				bc.GetBlockByNumber(bc.CurrentBlock().NumberU64()).Hash(),
			},
		}, {
			&getBlockHeadersData{Origin: hashOrNumber{Number: 4}, Skip: 3, Amount: 3, Reverse: true},
			[]common.Hash{
				bc.GetBlockByNumber(4).Hash(),
				bc.GetBlockByNumber(0).Hash(),
			},
		},
		// Check that requesting more than available is handled gracefully, even if mid skip
		{
			&getBlockHeadersData{Origin: hashOrNumber{Number: bc.CurrentBlock().NumberU64() - 4}, Skip: 2, Amount: 3},
			[]common.Hash{
				bc.GetBlockByNumber(bc.CurrentBlock().NumberU64() - 4).Hash(),
				bc.GetBlockByNumber(bc.CurrentBlock().NumberU64() - 1).Hash(),
			},
		}, {
			&getBlockHeadersData{Origin: hashOrNumber{Number: 4}, Skip: 2, Amount: 3, Reverse: true},
			[]common.Hash{
				bc.GetBlockByNumber(4).Hash(),
				bc.GetBlockByNumber(1).Hash(),
			},
		},
		// Check that non existing headers aren't returned
		{
			&getBlockHeadersData{Origin: hashOrNumber{Hash: unknown}, Amount: 1},
			[]common.Hash{},
		}, {
			&getBlockHeadersData{Origin: hashOrNumber{Number: bc.CurrentBlock().NumberU64() + 1}, Amount: 1},
			[]common.Hash{},
		},
	}
	// Run each of the tests and verify the results against the chain
	var reqID uint64
	for i, tt := range tests {
		// Collect the headers to expect in the response
		var headers []*types.Header
		for _, hash := range tt.expect {
			headers = append(headers, bc.GetHeaderByHash(hash))
		}
		// Send the hash request and verify the response
		reqID++
		cost := server.tPeer.GetRequestCost(GetBlockHeadersMsg, int(tt.query.Amount))
		sendRequest(server.tPeer.app, GetBlockHeadersMsg, reqID, cost, tt.query)
		if err := expectResponse(server.tPeer.app, BlockHeadersMsg, reqID, testBufLimit, headers); err != nil {
			t.Errorf("test %d: headers mismatch: %v", i, err)
		}
	}
}

// Tests that block contents can be retrieved from a remote chain based on their hashes.
func TestGetBlockBodiesLes2(t *testing.T) { testGetBlockBodies(t, 2) }

func testGetBlockBodies(t *testing.T, protocol int) {
	server, tearDown := newServerEnv(t, downloader.MaxBlockFetch+15, protocol, nil)
	defer tearDown()
	bc := server.pm.blockchain.(*core.BlockChain)

	// Create a batch of tests for various scenarios
	limit := MaxBodyFetch
	tests := []struct {
		random    int           // Number of blocks to fetch randomly from the chain
		explicit  []common.Hash // Explicitly requested blocks
		available []bool        // Availability of explicitly requested blocks
		expected  int           // Total number of existing blocks to expect
	}{
		{1, nil, nil, 1},         // A single random block should be retrievable
		{10, nil, nil, 10},       // Multiple random blocks should be retrievable
		{limit, nil, nil, limit}, // The maximum possible blocks should be retrievable
		//{limit + 1, nil, nil, limit},                                  // No more than the possible block count should be returned
		{0, []common.Hash{bc.Genesis().Hash()}, []bool{true}, 1},      // The genesis block should be retrievable
		{0, []common.Hash{bc.CurrentBlock().Hash()}, []bool{true}, 1}, // The chains head block should be retrievable
		{0, []common.Hash{{}}, []bool{false}, 0},                      // A non existent block should not be returned

		// Existing and non-existing blocks interleaved should not cause problems
		{0, []common.Hash{
			{},
			bc.GetBlockByNumber(1).Hash(),
			{},
			bc.GetBlockByNumber(10).Hash(),
			{},
			bc.GetBlockByNumber(100).Hash(),
			{},
		}, []bool{false, true, false, true, false, true, false}, 3},
	}
	// Run each of the tests and verify the results against the chain
	var reqID uint64
	for i, tt := range tests {
		// Collect the hashes to request, and the response to expect
		var hashes []common.Hash
		seen := make(map[int64]bool)
		var bodies []*types.Body

		for j := 0; j < tt.random; j++ {
			for {
				num := rand.Int63n(int64(bc.CurrentBlock().NumberU64()))
				if !seen[num] {
					seen[num] = true

					block := bc.GetBlockByNumber(uint64(num))
					hashes = append(hashes, block.Hash())
					if len(bodies) < tt.expected {
						bodies = append(bodies, &types.Body{Transactions: block.Transactions(), Uncles: block.Uncles()})
					}
					break
				}
			}
		}
		for j, hash := range tt.explicit {
			hashes = append(hashes, hash)
			if tt.available[j] && len(bodies) < tt.expected {
				block := bc.GetBlockByHash(hash)
				bodies = append(bodies, &types.Body{Transactions: block.Transactions(), Uncles: block.Uncles()})
			}
		}
		reqID++
		// Send the hash request and verify the response
		cost := server.tPeer.GetRequestCost(GetBlockBodiesMsg, len(hashes))
		sendRequest(server.tPeer.app, GetBlockBodiesMsg, reqID, cost, hashes)
		if err := expectResponse(server.tPeer.app, BlockBodiesMsg, reqID, testBufLimit, bodies); err != nil {
			t.Errorf("test %d: bodies mismatch: %v", i, err)
		}
	}
}

// Tests that the contract codes can be retrieved based on account addresses.
func TestGetCodeLes2(t *testing.T) { testGetCode(t, 2) }

func testGetCode(t *testing.T, protocol int) {
	// Assemble the test environment
	server, tearDown := newServerEnv(t, 4, protocol, nil)
	defer tearDown()
	bc := server.pm.blockchain.(*core.BlockChain)

	var codereqs []*CodeReq
	var codes [][]byte
	for i := uint64(0); i <= bc.CurrentBlock().NumberU64(); i++ {
		header := bc.GetHeaderByNumber(i)
		req := &CodeReq{
			BHash:  header.Hash(),
			AccKey: crypto.Keccak256(testContractAddr[:]),
		}
		codereqs = append(codereqs, req)
		if i >= testContractDeployed {
			codes = append(codes, testContractCodeDeployed)
		}
	}

	cost := server.tPeer.GetRequestCost(GetCodeMsg, len(codereqs))
	sendRequest(server.tPeer.app, GetCodeMsg, 42, cost, codereqs)
	if err := expectResponse(server.tPeer.app, CodeMsg, 42, testBufLimit, codes); err != nil {
		t.Errorf("codes mismatch: %v", err)
	}
}

// Tests that the stale contract codes can't be retrieved based on account addresses.
func TestGetStaleCodeLes2(t *testing.T) { testGetStaleCode(t, 2) }
func TestGetStaleCodeLes3(t *testing.T) { testGetStaleCode(t, 3) }

func testGetStaleCode(t *testing.T, protocol int) {
	server, tearDown := newServerEnv(t, core.TriesInMemory+4, protocol, nil)
	defer tearDown()
	bc := server.pm.blockchain.(*core.BlockChain)

	check := func(number uint64, expected [][]byte) {
		req := &CodeReq{
			BHash:  bc.GetHeaderByNumber(number).Hash(),
			AccKey: crypto.Keccak256(testContractAddr[:]),
		}
		cost := server.tPeer.GetRequestCost(GetCodeMsg, 1)
		sendRequest(server.tPeer.app, GetCodeMsg, 42, cost, []*CodeReq{req})
		if err := expectResponse(server.tPeer.app, CodeMsg, 42, testBufLimit, expected); err != nil {
			t.Errorf("codes mismatch: %v", err)
		}
	}
	check(0, [][]byte{})                                                          // Non-exist contract
	check(testContractDeployed, [][]byte{})                                       // Stale contract
	check(bc.CurrentHeader().Number.Uint64(), [][]byte{testContractCodeDeployed}) // Fresh contract
}

// Tests that the transaction receipts can be retrieved based on hashes.
func TestGetReceiptLes2(t *testing.T) { testGetReceipt(t, 2) }

func testGetReceipt(t *testing.T, protocol int) {
	// Assemble the test environment
	server, tearDown := newServerEnv(t, 4, protocol, nil)
	defer tearDown()
	bc := server.pm.blockchain.(*core.BlockChain)

	// Collect the hashes to request, and the response to expect
	var receipts []types.Receipts
	var hashes []common.Hash
	for i := uint64(0); i <= bc.CurrentBlock().NumberU64(); i++ {
		block := bc.GetBlockByNumber(i)

		hashes = append(hashes, block.Hash())
		receipts = append(receipts, rawdb.ReadRawReceipts(server.db, block.Hash(), block.NumberU64()))
	}
	// Send the hash request and verify the response
	cost := server.tPeer.GetRequestCost(GetReceiptsMsg, len(hashes))
	sendRequest(server.tPeer.app, GetReceiptsMsg, 42, cost, hashes)
	if err := expectResponse(server.tPeer.app, ReceiptsMsg, 42, testBufLimit, receipts); err != nil {
		t.Errorf("receipts mismatch: %v", err)
	}
}

// Tests that trie merkle proofs can be retrieved
func TestGetProofsLes2(t *testing.T) { testGetProofs(t, 2) }

func testGetProofs(t *testing.T, protocol int) {
	// Assemble the test environment
	server, tearDown := newServerEnv(t, 4, protocol, nil)
	defer tearDown()
	bc := server.pm.blockchain.(*core.BlockChain)

	var proofreqs []ProofReq
	proofsV2 := light.NewNodeSet()

	accounts := []common.Address{bankAddr, userAddr1, userAddr2, {}}
	for i := uint64(0); i <= bc.CurrentBlock().NumberU64(); i++ {
		header := bc.GetHeaderByNumber(i)
		trie, _ := trie.New(header.Root, trie.NewDatabase(server.db))

		for _, acc := range accounts {
			req := ProofReq{
				BHash: header.Hash(),
				Key:   crypto.Keccak256(acc[:]),
			}
			proofreqs = append(proofreqs, req)
			trie.Prove(crypto.Keccak256(acc[:]), 0, proofsV2)
		}
	}
	// Send the proof request and verify the response
	cost := server.tPeer.GetRequestCost(GetProofsV2Msg, len(proofreqs))
	sendRequest(server.tPeer.app, GetProofsV2Msg, 42, cost, proofreqs)
	if err := expectResponse(server.tPeer.app, ProofsV2Msg, 42, testBufLimit, proofsV2.NodeList()); err != nil {
		t.Errorf("proofs mismatch: %v", err)
	}
}

// Tests that the stale contract codes can't be retrieved based on account addresses.
func TestGetStaleProofLes2(t *testing.T) { testGetStaleProof(t, 2) }
func TestGetStaleProofLes3(t *testing.T) { testGetStaleProof(t, 3) }

func testGetStaleProof(t *testing.T, protocol int) {
	server, tearDown := newServerEnv(t, core.TriesInMemory+4, protocol, nil)
	defer tearDown()
	bc := server.pm.blockchain.(*core.BlockChain)

	check := func(number uint64, wantOK bool) {
		var (
			header  = bc.GetHeaderByNumber(number)
			account = crypto.Keccak256(userAddr1.Bytes())
		)
		req := &ProofReq{
			BHash: header.Hash(),
			Key:   account,
		}
		cost := server.tPeer.GetRequestCost(GetProofsV2Msg, 1)
		sendRequest(server.tPeer.app, GetProofsV2Msg, 42, cost, []*ProofReq{req})

		var expected []rlp.RawValue
		if wantOK {
			proofsV2 := light.NewNodeSet()
			t, _ := trie.New(header.Root, trie.NewDatabase(server.db))
			t.Prove(account, 0, proofsV2)
			expected = proofsV2.NodeList()
		}
		if err := expectResponse(server.tPeer.app, ProofsV2Msg, 42, testBufLimit, expected); err != nil {
			t.Errorf("codes mismatch: %v", err)
		}
	}
	check(0, false)                                 // Non-exist proof
	check(2, false)                                 // Stale proof
	check(bc.CurrentHeader().Number.Uint64(), true) // Fresh proof
}

// Tests that CHT proofs can be correctly retrieved.
func TestGetCHTProofsLes2(t *testing.T) { testGetCHTProofs(t, 2) }

func testGetCHTProofs(t *testing.T, protocol int) {
	config := light.TestServerIndexerConfig

	waitIndexers := func(cIndexer, bIndexer, btIndexer *core.ChainIndexer) {
		for {
			cs, _, _ := cIndexer.Sections()
			if cs >= 1 {
				break
			}
			time.Sleep(10 * time.Millisecond)
		}
	}
	server, tearDown := newServerEnv(t, int(config.ChtSize+config.ChtConfirms), protocol, waitIndexers)
	defer tearDown()
	bc := server.pm.blockchain.(*core.BlockChain)

	// Assemble the proofs from the different protocols
	header := bc.GetHeaderByNumber(config.ChtSize - 1)
	rlp, _ := rlp.EncodeToBytes(header)

	key := make([]byte, 8)
	binary.BigEndian.PutUint64(key, config.ChtSize-1)

	proofsV2 := HelperTrieResps{
		AuxData: [][]byte{rlp},
	}
	root := light.GetChtRoot(server.db, 0, bc.GetHeaderByNumber(config.ChtSize-1).Hash())
	trie, _ := trie.New(root, trie.NewDatabase(rawdb.NewTable(server.db, light.ChtTablePrefix)))
	trie.Prove(key, 0, &proofsV2.Proofs)
	// Assemble the requests for the different protocols
	requestsV2 := []HelperTrieReq{{
		Type:    htCanonical,
		TrieIdx: 0,
		Key:     key,
		AuxReq:  auxHeader,
	}}
	// Send the proof request and verify the response
	cost := server.tPeer.GetRequestCost(GetHelperTrieProofsMsg, len(requestsV2))
	sendRequest(server.tPeer.app, GetHelperTrieProofsMsg, 42, cost, requestsV2)
	if err := expectResponse(server.tPeer.app, HelperTrieProofsMsg, 42, testBufLimit, proofsV2); err != nil {
		t.Errorf("proofs mismatch: %v", err)
	}
}

// Tests that bloombits proofs can be correctly retrieved.
func TestGetBloombitsProofs(t *testing.T) {
	config := light.TestServerIndexerConfig

	waitIndexers := func(cIndexer, bIndexer, btIndexer *core.ChainIndexer) {
		for {
			bts, _, _ := btIndexer.Sections()
			if bts >= 1 {
				break
			}
			time.Sleep(10 * time.Millisecond)
		}
	}
	server, tearDown := newServerEnv(t, int(config.BloomTrieSize+config.BloomTrieConfirms), 2, waitIndexers)
	defer tearDown()
	bc := server.pm.blockchain.(*core.BlockChain)

	// Request and verify each bit of the bloom bits proofs
	for bit := 0; bit < 2048; bit++ {
		// Assemble the request and proofs for the bloombits
		key := make([]byte, 10)

		binary.BigEndian.PutUint16(key[:2], uint16(bit))
		// Only the first bloom section has data.
		binary.BigEndian.PutUint64(key[2:], 0)

		requests := []HelperTrieReq{{
			Type:    htBloomBits,
			TrieIdx: 0,
			Key:     key,
		}}
		var proofs HelperTrieResps

		root := light.GetBloomTrieRoot(server.db, 0, bc.GetHeaderByNumber(config.BloomTrieSize-1).Hash())
		trie, _ := trie.New(root, trie.NewDatabase(rawdb.NewTable(server.db, light.BloomTrieTablePrefix)))
		trie.Prove(key, 0, &proofs.Proofs)

		// Send the proof request and verify the response
		cost := server.tPeer.GetRequestCost(GetHelperTrieProofsMsg, len(requests))
		sendRequest(server.tPeer.app, GetHelperTrieProofsMsg, 42, cost, requests)
		if err := expectResponse(server.tPeer.app, HelperTrieProofsMsg, 42, testBufLimit, proofs); err != nil {
			t.Errorf("bit %d: proofs mismatch: %v", bit, err)
		}
	}
}

func TestTransactionStatusLes2(t *testing.T) {
	server, tearDown := newServerEnv(t, 0, 2, nil)
	defer tearDown()

	chain := server.pm.blockchain.(*core.BlockChain)
	config := core.DefaultTxPoolConfig
	config.Journal = ""
	txpool := core.NewTxPool(config, params.TestChainConfig, chain)
	server.pm.txpool = txpool
	peer, _ := newTestPeer(t, "peer", 2, server.pm, true, 0)
	defer peer.close()

	var reqID uint64

	test := func(tx *types.Transaction, send bool, expStatus light.TxStatus) {
		reqID++
		if send {
			cost := server.tPeer.GetRequestCost(SendTxV2Msg, 1)
			sendRequest(server.tPeer.app, SendTxV2Msg, reqID, cost, types.Transactions{tx})
		} else {
			cost := server.tPeer.GetRequestCost(GetTxStatusMsg, 1)
			sendRequest(server.tPeer.app, GetTxStatusMsg, reqID, cost, []common.Hash{tx.Hash()})
		}
		if err := expectResponse(server.tPeer.app, TxStatusMsg, reqID, testBufLimit, []light.TxStatus{expStatus}); err != nil {
			t.Errorf("transaction status mismatch")
		}
	}

	signer := types.HomesteadSigner{}

	// test error status by sending an underpriced transaction
	tx0, _ := types.SignTx(types.NewTransaction(0, userAddr1, big.NewInt(10000), params.TxGas, nil, nil), signer, bankKey)
	test(tx0, true, light.TxStatus{Status: core.TxStatusUnknown, Error: core.ErrUnderpriced.Error()})

	tx1, _ := types.SignTx(types.NewTransaction(0, userAddr1, big.NewInt(10000), params.TxGas, big.NewInt(100000000000), nil), signer, bankKey)
	test(tx1, false, light.TxStatus{Status: core.TxStatusUnknown}) // query before sending, should be unknown
	test(tx1, true, light.TxStatus{Status: core.TxStatusPending})  // send valid processable tx, should return pending
	test(tx1, true, light.TxStatus{Status: core.TxStatusPending})  // adding it again should not return an error

	tx2, _ := types.SignTx(types.NewTransaction(1, userAddr1, big.NewInt(10000), params.TxGas, big.NewInt(100000000000), nil), signer, bankKey)
	tx3, _ := types.SignTx(types.NewTransaction(2, userAddr1, big.NewInt(10000), params.TxGas, big.NewInt(100000000000), nil), signer, bankKey)
	// send transactions in the wrong order, tx3 should be queued
	test(tx3, true, light.TxStatus{Status: core.TxStatusQueued})
	test(tx2, true, light.TxStatus{Status: core.TxStatusPending})
	// query again, now tx3 should be pending too
	test(tx3, false, light.TxStatus{Status: core.TxStatusPending})

	// generate and add a block with tx1 and tx2 included
	gchain, _ := core.GenerateChain(params.TestChainConfig, chain.GetBlockByNumber(0), ethash.NewFaker(), server.db, 1, func(i int, block *core.BlockGen) {
		block.AddTx(tx1)
		block.AddTx(tx2)
	})
	if _, err := chain.InsertChain(gchain); err != nil {
		panic(err)
	}
	// wait until TxPool processes the inserted block
	for i := 0; i < 10; i++ {
		if pending, _ := txpool.Stats(); pending == 1 {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}
	if pending, _ := txpool.Stats(); pending != 1 {
		t.Fatalf("pending count mismatch: have %d, want 1", pending)
	}

	// check if their status is included now
	block1hash := rawdb.ReadCanonicalHash(server.db, 1)
	test(tx1, false, light.TxStatus{Status: core.TxStatusIncluded, Lookup: &rawdb.LegacyTxLookupEntry{BlockHash: block1hash, BlockIndex: 1, Index: 0}})
	test(tx2, false, light.TxStatus{Status: core.TxStatusIncluded, Lookup: &rawdb.LegacyTxLookupEntry{BlockHash: block1hash, BlockIndex: 1, Index: 1}})

	// create a reorg that rolls them back
	gchain, _ = core.GenerateChain(params.TestChainConfig, chain.GetBlockByNumber(0), ethash.NewFaker(), server.db, 2, func(i int, block *core.BlockGen) {})
	if _, err := chain.InsertChain(gchain); err != nil {
		panic(err)
	}
	// wait until TxPool processes the reorg
	for i := 0; i < 10; i++ {
		if pending, _ := txpool.Stats(); pending == 3 {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}
	if pending, _ := txpool.Stats(); pending != 3 {
		t.Fatalf("pending count mismatch: have %d, want 3", pending)
	}
	// check if their status is pending again
	test(tx1, false, light.TxStatus{Status: core.TxStatusPending})
	test(tx2, false, light.TxStatus{Status: core.TxStatusPending})
}

func TestStopResumeLes3(t *testing.T) {
	db := rawdb.NewMemoryDatabase()
	clock := &mclock.Simulated{}
	testCost := testBufLimit / 10
	pm, _, err := newTestProtocolManager(false, 0, nil, nil, nil, db, nil, 0, testCost, clock)
	if err != nil {
		t.Fatalf("Failed to create protocol manager: %v", err)
	}
	peer, _ := newTestPeer(t, "peer", 3, pm, true, testCost)
	defer peer.close()

	expBuf := testBufLimit
	var reqID uint64

	header := pm.blockchain.CurrentHeader()
	req := func() {
		reqID++
		sendRequest(peer.app, GetBlockHeadersMsg, reqID, testCost, &getBlockHeadersData{Origin: hashOrNumber{Hash: header.Hash()}, Amount: 1})
	}

	for i := 1; i <= 5; i++ {
		// send requests while we still have enough buffer and expect a response
		for expBuf >= testCost {
			req()
			expBuf -= testCost
			if err := expectResponse(peer.app, BlockHeadersMsg, reqID, expBuf, []*types.Header{header}); err != nil {
				t.Fatalf("expected response and failed: %v", err)
			}
		}
		// send some more requests in excess and expect a single StopMsg
		c := i
		for c > 0 {
			req()
			c--
		}
		if err := p2p.ExpectMsg(peer.app, StopMsg, nil); err != nil {
			t.Errorf("expected StopMsg and failed: %v", err)
		}
		// wait until the buffer is recharged by half of the limit
		wait := testBufLimit / testBufRecharge / 2
		clock.Run(time.Millisecond * time.Duration(wait))
		// expect a ResumeMsg with the partially recharged buffer value
		expBuf += testBufRecharge * wait
		if err := p2p.ExpectMsg(peer.app, ResumeMsg, expBuf); err != nil {
			t.Errorf("expected ResumeMsg and failed: %v", err)
		}
	}
}
