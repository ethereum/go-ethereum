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
	"context"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/light"
)

var testBankSecureTrieKey = secAddr(testBankAddress)

func secAddr(addr common.Address) []byte {
	return crypto.Keccak256(addr[:])
}

type accessTestFn func(db ethdb.Database, bhash common.Hash, number uint64) light.OdrRequest

func TestBlockAccessLes1(t *testing.T) { testAccess(t, 1, tfBlockAccess) }

func tfBlockAccess(db ethdb.Database, bhash common.Hash, number uint64) light.OdrRequest {
	return &light.BlockRequest{Hash: bhash, Number: number}
}

func TestReceiptsAccessLes1(t *testing.T) { testAccess(t, 1, tfReceiptsAccess) }

func tfReceiptsAccess(db ethdb.Database, bhash common.Hash, number uint64) light.OdrRequest {
	return &light.ReceiptsRequest{Hash: bhash, Number: number}
}

func TestTrieEntryAccessLes1(t *testing.T) { testAccess(t, 1, tfTrieEntryAccess) }

func tfTrieEntryAccess(db ethdb.Database, bhash common.Hash, number uint64) light.OdrRequest {
	return &light.TrieRequest{Id: light.StateTrieID(core.GetHeader(db, bhash, core.GetBlockNumber(db, bhash))), Key: testBankSecureTrieKey}
}

func TestCodeAccessLes1(t *testing.T) { testAccess(t, 1, tfCodeAccess) }

func tfCodeAccess(db ethdb.Database, bhash common.Hash, number uint64) light.OdrRequest {
	header := core.GetHeader(db, bhash, core.GetBlockNumber(db, bhash))
	if header.Number.Uint64() < testContractDeployed {
		return nil
	}
	sti := light.StateTrieID(header)
	ci := light.StorageTrieID(sti, testContractAddr, common.Hash{})
	return &light.CodeRequest{Id: ci, Hash: crypto.Keccak256Hash(testContractCodeDeployed)}
}

func testAccess(t *testing.T, protocol int, fn accessTestFn) {
	// Assemble the test environment
	pm, db, _ := newTestProtocolManagerMust(t, false, 4, testChainGen)
	lpm, ldb, odr := newTestProtocolManagerMust(t, true, 0, nil)
	_, err1, lpeer, err2 := newTestPeerPair("peer", protocol, pm, lpm)
	pool := &testServerPool{}
	lpm.reqDist = newRequestDistributor(pool.getAllPeers, lpm.quitSync)
	odr.reqDist = lpm.reqDist
	pool.setPeer(lpeer)
	odr.serverPool = pool
	lpeer.hasBlock = func(common.Hash, uint64) bool { return true }
	select {
	case <-time.After(time.Millisecond * 100):
	case err := <-err1:
		t.Fatalf("peer 1 handshake error: %v", err)
	case err := <-err2:
		t.Fatalf("peer 1 handshake error: %v", err)
	}

	lpm.synchronise(lpeer)

	test := func(expFail uint64) {
		for i := uint64(0); i <= pm.blockchain.CurrentHeader().Number.Uint64(); i++ {
			bhash := core.GetCanonicalHash(db, i)
			if req := fn(ldb, bhash, i); req != nil {
				ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
				defer cancel()

				err := odr.Retrieve(ctx, req)
				got := err == nil
				exp := i < expFail
				if exp && !got {
					t.Errorf("object retrieval failed")
				}
				if !exp && got {
					t.Errorf("unexpected object retrieval success")
				}
			}
		}
	}

	// temporarily remove peer to test odr fails
	pool.setPeer(nil)
	// expect retrievals to fail (except genesis block) without a les peer
	test(0)
	pool.setPeer(lpeer)
	// expect all retrievals to pass
	test(5)
}
