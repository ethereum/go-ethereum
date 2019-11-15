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

	"github.com/maticnetwork/bor/common"
	"github.com/maticnetwork/bor/core/rawdb"
	"github.com/maticnetwork/bor/crypto"
	"github.com/maticnetwork/bor/ethdb"
	"github.com/maticnetwork/bor/light"
)

var testBankSecureTrieKey = secAddr(bankAddr)

func secAddr(addr common.Address) []byte {
	return crypto.Keccak256(addr[:])
}

type accessTestFn func(db ethdb.Database, bhash common.Hash, number uint64) light.OdrRequest

func TestBlockAccessLes2(t *testing.T) { testAccess(t, 2, tfBlockAccess) }

func tfBlockAccess(db ethdb.Database, bhash common.Hash, number uint64) light.OdrRequest {
	return &light.BlockRequest{Hash: bhash, Number: number}
}

func TestReceiptsAccessLes2(t *testing.T) { testAccess(t, 2, tfReceiptsAccess) }

func tfReceiptsAccess(db ethdb.Database, bhash common.Hash, number uint64) light.OdrRequest {
	return &light.ReceiptsRequest{Hash: bhash, Number: number}
}

func TestTrieEntryAccessLes2(t *testing.T) { testAccess(t, 2, tfTrieEntryAccess) }

func tfTrieEntryAccess(db ethdb.Database, bhash common.Hash, number uint64) light.OdrRequest {
	if number := rawdb.ReadHeaderNumber(db, bhash); number != nil {
		return &light.TrieRequest{Id: light.StateTrieID(rawdb.ReadHeader(db, bhash, *number)), Key: testBankSecureTrieKey}
	}
	return nil
}

func TestCodeAccessLes2(t *testing.T) { testAccess(t, 2, tfCodeAccess) }

func tfCodeAccess(db ethdb.Database, bhash common.Hash, num uint64) light.OdrRequest {
	number := rawdb.ReadHeaderNumber(db, bhash)
	if number != nil {
		return nil
	}
	header := rawdb.ReadHeader(db, bhash, *number)
	if header.Number.Uint64() < testContractDeployed {
		return nil
	}
	sti := light.StateTrieID(header)
	ci := light.StorageTrieID(sti, crypto.Keccak256Hash(testContractAddr[:]), common.Hash{})
	return &light.CodeRequest{Id: ci, Hash: crypto.Keccak256Hash(testContractCodeDeployed)}
}

func testAccess(t *testing.T, protocol int, fn accessTestFn) {
	// Assemble the test environment
	server, client, tearDown := newClientServerEnv(t, 4, protocol, nil, true)
	defer tearDown()
	client.pm.synchronise(client.rPeer)

	test := func(expFail uint64) {
		for i := uint64(0); i <= server.pm.blockchain.CurrentHeader().Number.Uint64(); i++ {
			bhash := rawdb.ReadCanonicalHash(server.db, i)
			if req := fn(client.db, bhash, i); req != nil {
				ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
				defer cancel()
				err := client.pm.odr.Retrieve(ctx, req)
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
	client.peers.Unregister(client.rPeer.id)
	time.Sleep(time.Millisecond * 10) // ensure that all peerSetNotify callbacks are executed
	// expect retrievals to fail (except genesis block) without a les peer
	test(0)

	client.peers.Register(client.rPeer)
	time.Sleep(time.Millisecond * 10) // ensure that all peerSetNotify callbacks are executed
	client.rPeer.lock.Lock()
	client.rPeer.hasBlock = func(common.Hash, uint64, bool) bool { return true }
	client.rPeer.lock.Unlock()
	// expect all retrievals to pass
	test(5)
}
