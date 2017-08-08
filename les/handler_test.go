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
	"math/rand"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/eth/downloader"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/ethereum/go-ethereum/trie"
)

func expectResponse(r p2p.MsgReader, msgcode, reqID, bv uint64, data interface{}) error {
	type resp struct {
		ReqID, BV uint64
		Data      interface{}
	}
	return p2p.ExpectMsg(r, msgcode, resp{reqID, bv, data})
}

// Tests that block headers can be retrieved from a remote chain based on user queries.
func TestGetBlockHeadersLes1(t *testing.T) { testGetBlockHeaders(t, 1) }

func testGetBlockHeaders(t *testing.T, protocol int) {
	db, _ := ethdb.NewMemDatabase()
	pm := newTestProtocolManagerMust(t, false, downloader.MaxHashFetch+15, nil, nil, nil, db)
	bc := pm.blockchain.(*core.BlockChain)
	peer, _ := newTestPeer(t, "peer", protocol, pm, true)
	defer peer.close()

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
		headers := []*types.Header{}
		for _, hash := range tt.expect {
			headers = append(headers, bc.GetHeaderByHash(hash))
		}
		// Send the hash request and verify the response
		reqID++
		cost := peer.GetRequestCost(GetBlockHeadersMsg, int(tt.query.Amount))
		sendRequest(peer.app, GetBlockHeadersMsg, reqID, cost, tt.query)
		if err := expectResponse(peer.app, BlockHeadersMsg, reqID, testBufLimit, headers); err != nil {
			t.Errorf("test %d: headers mismatch: %v", i, err)
		}
	}
}

// Tests that block contents can be retrieved from a remote chain based on their hashes.
func TestGetBlockBodiesLes1(t *testing.T) { testGetBlockBodies(t, 1) }

func testGetBlockBodies(t *testing.T, protocol int) {
	db, _ := ethdb.NewMemDatabase()
	pm := newTestProtocolManagerMust(t, false, downloader.MaxBlockFetch+15, nil, nil, nil, db)
	bc := pm.blockchain.(*core.BlockChain)
	peer, _ := newTestPeer(t, "peer", protocol, pm, true)
	defer peer.close()

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
		hashes, seen := []common.Hash{}, make(map[int64]bool)
		bodies := []*types.Body{}

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
		cost := peer.GetRequestCost(GetBlockBodiesMsg, len(hashes))
		sendRequest(peer.app, GetBlockBodiesMsg, reqID, cost, hashes)
		if err := expectResponse(peer.app, BlockBodiesMsg, reqID, testBufLimit, bodies); err != nil {
			t.Errorf("test %d: bodies mismatch: %v", i, err)
		}
	}
}

// Tests that the contract codes can be retrieved based on account addresses.
func TestGetCodeLes1(t *testing.T) { testGetCode(t, 1) }

func testGetCode(t *testing.T, protocol int) {
	// Assemble the test environment
	db, _ := ethdb.NewMemDatabase()
	pm := newTestProtocolManagerMust(t, false, 4, testChainGen, nil, nil, db)
	bc := pm.blockchain.(*core.BlockChain)
	peer, _ := newTestPeer(t, "peer", protocol, pm, true)
	defer peer.close()

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

	cost := peer.GetRequestCost(GetCodeMsg, len(codereqs))
	sendRequest(peer.app, GetCodeMsg, 42, cost, codereqs)
	if err := expectResponse(peer.app, CodeMsg, 42, testBufLimit, codes); err != nil {
		t.Errorf("codes mismatch: %v", err)
	}
}

// Tests that the transaction receipts can be retrieved based on hashes.
func TestGetReceiptLes1(t *testing.T) { testGetReceipt(t, 1) }

func testGetReceipt(t *testing.T, protocol int) {
	// Assemble the test environment
	db, _ := ethdb.NewMemDatabase()
	pm := newTestProtocolManagerMust(t, false, 4, testChainGen, nil, nil, db)
	bc := pm.blockchain.(*core.BlockChain)
	peer, _ := newTestPeer(t, "peer", protocol, pm, true)
	defer peer.close()

	// Collect the hashes to request, and the response to expect
	hashes, receipts := []common.Hash{}, []types.Receipts{}
	for i := uint64(0); i <= bc.CurrentBlock().NumberU64(); i++ {
		block := bc.GetBlockByNumber(i)

		hashes = append(hashes, block.Hash())
		receipts = append(receipts, core.GetBlockReceipts(db, block.Hash(), block.NumberU64()))
	}
	// Send the hash request and verify the response
	cost := peer.GetRequestCost(GetReceiptsMsg, len(hashes))
	sendRequest(peer.app, GetReceiptsMsg, 42, cost, hashes)
	if err := expectResponse(peer.app, ReceiptsMsg, 42, testBufLimit, receipts); err != nil {
		t.Errorf("receipts mismatch: %v", err)
	}
}

// Tests that trie merkle proofs can be retrieved
func TestGetProofsLes1(t *testing.T) { testGetProofs(t, 1) }

func testGetProofs(t *testing.T, protocol int) {
	// Assemble the test environment
	db, _ := ethdb.NewMemDatabase()
	pm := newTestProtocolManagerMust(t, false, 4, testChainGen, nil, nil, db)
	bc := pm.blockchain.(*core.BlockChain)
	peer, _ := newTestPeer(t, "peer", protocol, pm, true)
	defer peer.close()

	var proofreqs []ProofReq
	var proofs [][]rlp.RawValue

	accounts := []common.Address{testBankAddress, acc1Addr, acc2Addr, {}}
	for i := uint64(0); i <= bc.CurrentBlock().NumberU64(); i++ {
		header := bc.GetHeaderByNumber(i)
		root := header.Root
		trie, _ := trie.New(root, db)

		for _, acc := range accounts {
			req := ProofReq{
				BHash: header.Hash(),
				Key:   crypto.Keccak256(acc[:]),
			}
			proofreqs = append(proofreqs, req)

			proof := trie.Prove(crypto.Keccak256(acc[:]))
			proofs = append(proofs, proof)
		}
	}
	// Send the proof request and verify the response
	cost := peer.GetRequestCost(GetProofsMsg, len(proofreqs))
	sendRequest(peer.app, GetProofsMsg, 42, cost, proofreqs)
	if err := expectResponse(peer.app, ProofsMsg, 42, testBufLimit, proofs); err != nil {
		t.Errorf("proofs mismatch: %v", err)
	}
}
