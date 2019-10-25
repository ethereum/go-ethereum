// Copyright 2019 The go-ethereum Authors
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
	"github.com/ethereum/go-ethereum/lescdn"
	"github.com/ethereum/go-ethereum/light"
)

func TestHTTPRequest(t *testing.T) {
	config := light.TestServerIndexerConfig

	waitIndexers := func(cIndexer, bIndexer, btIndexer *core.ChainIndexer) {
		for {
			cs, _, _ := cIndexer.Sections()
			bts, _, _ := btIndexer.Sections()
			if cs >= 1 && bts >= 1 {
				break
			}
			time.Sleep(10 * time.Millisecond)
		}
	}
	// Generate 512+4 blocks (totally 1 CHT sections)
	server, client, tearDown := newClientServerEnv(t, int(config.ChtSize+config.ChtConfirms), 3, waitIndexers, nil, 0, false, true)
	defer tearDown()

	// Start a CDN service for request handling
	lesCDN := lescdn.New(server.backend.Blockchain())
	lesCDN.Start(nil)

	// Prepare CDN requests
	blockOne := server.backend.Blockchain().GetBlockByNumber(1)
	headerSection := server.backend.Blockchain().GetHeaderByNumber(config.BloomTrieSize - 1)
	chtRoot := light.GetChtRoot(server.db, 0, headerSection.Hash())
	bloomTrieRoot := light.GetBloomTrieRoot(server.db, 0, headerSection.Hash())

	// Prepare txstatus requests
	var hashes []common.Hash
	txs := blockOne.Transactions()
	for _, tx := range txs {
		hashes = append(hashes, tx.Hash())
	}

	// Prepare state reqeusts
	blockHead := server.backend.Blockchain().CurrentBlock()
	state, _ := server.backend.Blockchain().State()
	codeHash := state.GetCodeHash(registrarAddr)

	stateTrieID := &light.TrieID{
		BlockHash:   blockHead.Hash(),
		BlockNumber: blockHead.NumberU64(),
	}
	accTrieID := &light.TrieID{
		BlockHash:   blockHead.Hash(),
		BlockNumber: blockHead.NumberU64(),
		AccKey:      registrarAddr.Bytes(),
	}
	var requests = []light.OdrRequest{
		&light.BlockRequest{Hash: blockOne.Hash(), Number: blockOne.NumberU64()},
		&light.ReceiptsRequest{Hash: blockOne.Hash(), Number: blockOne.NumberU64(), Header: blockOne.Header()},
		&light.TrieRequest{Id: stateTrieID, MissNodeHash: blockHead.Root()},
		&light.CodeRequest{Hash: codeHash, Id: accTrieID},
		&light.ChtRequest{ChtRoot: chtRoot, ChtNum: 0, Config: server.handler.server.iConfig, BlockNum: headerSection.Number.Uint64()},
		&light.BloomRequest{SectionList: []uint64{0}, BitIndex: 0, Config: server.handler.server.iConfig, BloomTrieNum: 0, BloomTrieRoot: bloomTrieRoot},
		&light.TxStatusRequest{Hashes: hashes},
	}
	for _, request := range requests {
		req := LesRequest(request)
		err := req.RequestByHTTP(context.Background(), "http://localhost:8548", client.db)
		if err != nil {
			t.Fatalf("Failed to retrieve data via CDN: %v", err)
		}
	}
}
