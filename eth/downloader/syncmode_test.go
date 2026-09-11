// Copyright 2026 The go-ethereum Authors
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
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/state/snapshot"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/eth/ethconfig"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/ethereum/go-ethereum/triedb"
)

type mockSyncChain struct {
	head     *types.Header
	hasState bool
}

func (m *mockSyncChain) HasHeader(common.Hash, uint64) bool                     { return false }
func (m *mockSyncChain) HasState(root common.Hash) bool                         { return m.hasState }
func (m *mockSyncChain) GetHeaderByHash(common.Hash) *types.Header              { return nil }
func (m *mockSyncChain) CurrentHeader() *types.Header                           { return m.head }
func (m *mockSyncChain) SetHead(uint64) error                                   { return nil }
func (m *mockSyncChain) HasBlock(common.Hash, uint64) bool                      { return false }
func (m *mockSyncChain) HasFastBlock(common.Hash, uint64) bool                  { return false }
func (m *mockSyncChain) GetCanonicalHash(uint64) common.Hash                    { return common.Hash{} }
func (m *mockSyncChain) GetBlockByHash(common.Hash) *types.Block                { return nil }
func (m *mockSyncChain) CurrentBlock() *types.Header                            { return m.head }
func (m *mockSyncChain) CurrentSnapBlock() *types.Header                        { return m.head }
func (m *mockSyncChain) SnapSyncStart() error                                   { return nil }
func (m *mockSyncChain) SnapSyncComplete(hash common.Hash, isSnapV2 bool) error { return nil }
func (m *mockSyncChain) InsertHeadersBeforeCutoff([]*types.Header) (int, error) { return 0, nil }
func (m *mockSyncChain) InsertChain(types.Blocks) (int, error)                  { return 0, nil }
func (m *mockSyncChain) InterruptInsert(on bool)                                {}
func (m *mockSyncChain) InsertReceiptChain(types.Blocks, []rlp.RawValue, uint64) (int, error) {
	return 0, nil
}
func (m *mockSyncChain) Snapshots() *snapshot.Tree                   { return nil }
func (m *mockSyncChain) TrieDB() *triedb.Database                    { return nil }
func (m *mockSyncChain) HistoryPruningCutoff() (uint64, common.Hash) { return 0, common.Hash{} }

func TestSyncModeGenesisTransition(t *testing.T) {
	db := rawdb.NewMemoryDatabase()

	genesisHeader := &types.Header{
		Number: big.NewInt(0),
		Root:   common.HexToHash("0x1"),
	}

	chain := &mockSyncChain{
		head:     genesisHeader,
		hasState: true,
	}

	// Initialize syncModer with default SnapSync at genesis block 0.
	moder := newSyncModer(ethconfig.SnapSync, chain, db)

	// At block 0, even though state is available, sync mode stays SnapSync so network snap sync can be performed.
	if mode := moder.get(false); mode != ethconfig.SnapSync {
		t.Fatalf("expected SnapSync at genesis block 0, got %v", mode)
	}

	// Now advance chain to block 1 with state available.
	block1Header := &types.Header{
		Number: big.NewInt(1),
		Root:   common.HexToHash("0x2"),
	}
	chain.head = block1Header

	// get() should now automatically detect that chain has progressed past genesis with state and flip to FullSync.
	if mode := moder.get(false); mode != ethconfig.FullSync {
		t.Fatalf("expected FullSync after advancing past genesis block 0, got %v", mode)
	}
}
