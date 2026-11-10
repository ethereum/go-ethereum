// Copyright 2026-2027 The quarkchain Authors
// This file is part of the quarkchain library.
//
// The quarkchain library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The quarkchain library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the quarkchain library. If not, see <http://www.gnu.org/licenses/>.

package shard

import (
	"errors"
	"math/big"
	"reflect"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/params"
	qkctypes "github.com/ethereum/go-ethereum/qkc/types"
	"github.com/holiman/uint256"
)

// testXShardChain is a root chain and the deposits a shard received, in memory.
type testXShardChain struct {
	roots    map[common.Hash]*qkctypes.RootBlock
	deposits map[common.Hash][]*types.DepositTx
}

func (c *testXShardChain) RootBlock(hash common.Hash) *qkctypes.RootBlock {
	return c.roots[hash]
}

func (c *testXShardChain) RootBlockByNumber(tip common.Hash, number uint64) *qkctypes.RootBlock {
	for root := c.roots[tip]; root != nil; root = c.roots[root.ParentHash()] {
		if root.NumberU64() == number {
			return root
		}
		if root.NumberU64() < number {
			return nil
		}
	}
	return nil
}

func (c *testXShardChain) XShardDeposits(hash common.Hash) []*types.DepositTx {
	return c.deposits[hash]
}

// sourceBlock returns a block header of the given shard, unique by number.
func sourceBlock(shard int64, number int64) *types.Header {
	return &types.Header{
		Number:     big.NewInt(number),
		Difficulty: new(big.Int),
		Extra:      (&types.QKCExtra{ChainID: big.NewInt(shard)}).Encode(),
	}
}

// testDeposit returns a deposit sent by the given source block.
func testDeposit(source *types.Header, index uint64) *types.DepositTx {
	to := common.Address{byte(index)}
	return &types.DepositTx{
		Version:         types.DepositTxVersion,
		SourceBlockHash: source.Hash(),
		SourceLogIndex:  index,
		From:            common.Address{0xf0},
		To:              &to,
		Value:           uint256.NewInt(index + 1),
		GasPrice:        uint256.NewInt(1),
		Gas:             params.XShardSendGas,
		IsMove:          true,
	}
}

// cursorAt returns a shard block header with the given cursor.
func cursorAt(height, shard, deposit uint64) *types.Header {
	return &types.Header{
		Number:     new(big.Int),
		Difficulty: new(big.Int),
		Extra:      (&types.QKCExtra{Cursor: types.XShardCursor{RootBlockHeight: height, ShardBlockIndex: shard, XShardDepositIndex: deposit}}).Encode(),
	}
}

// testRootChain builds the root chain R0 <- R1 <- R2 <- R3, where R1 confirms
// a block A of shard 0, a block B of shard 1 and a block C of shard 2, R2 a
// block D of shard 2, and R3 nothing. Shard 1 received two deposits from A,
// none from C and one from D, and shard 0 one from B and one from D.
func testRootChain() (roots []*qkctypes.RootBlock, a, b, c, d *types.Header, toShard0, toShard1 *testXShardChain) {
	a, b, c, d = sourceBlock(10001, 1), sourceBlock(10002, 1), sourceBlock(10003, 1), sourceBlock(10003, 2)
	confirmed := [][]*types.Header{nil, {a, b, c}, {d}, nil}
	toShard0 = &testXShardChain{roots: make(map[common.Hash]*qkctypes.RootBlock), deposits: map[common.Hash][]*types.DepositTx{
		b.Hash(): {testDeposit(b, 0)},
		c.Hash(): {},
		d.Hash(): {testDeposit(d, 5)},
	}}
	toShard1 = &testXShardChain{roots: toShard0.roots, deposits: map[common.Hash][]*types.DepositTx{
		a.Hash(): {testDeposit(a, 0), testDeposit(a, 3)},
		c.Hash(): {},
		d.Hash(): {testDeposit(d, 2)},
	}}
	var parent common.Hash
	for number, headers := range confirmed {
		root := qkctypes.NewRootBlockWithHeader(&qkctypes.RootBlockHeader{
			Number:         uint32(number),
			ParentHash:     parent,
			Coinbase:       common.Address{0xc0, byte(number)},
			CoinbaseAmount: *uint256.NewInt(uint64(100 + number)),
		}).WithBody(headers, nil)
		toShard0.roots[root.Hash()] = root
		roots = append(roots, root)
		parent = root.Hash()
	}
	return roots, a, b, c, d, toShard0, toShard1
}

// pos returns a cross-shard cursor.
func pos(height, shard, deposit uint64) types.XShardCursor {
	return types.XShardCursor{RootBlockHeight: height, ShardBlockIndex: shard, XShardDepositIndex: deposit}
}

func TestConfiguredRootCoinbaseChain(t *testing.T) {
	roots, _, _, _, _, _, _ := testRootChain()
	id := new(big.Int).Lsh(big.NewInt(1), 128)
	config := params.NewTestQuarkChainConfig(10002)
	config.ChainID = new(big.Int).Set(id)
	config.QuarkChain.ChainIDs = []*big.Int{new(big.Int).Set(id), big.NewInt(10001)}
	cursor := &xShardCursor{config: config, root: roots[1], depositIndex: 1}
	if got := cursor.current(); !reflect.DeepEqual(got, rootCoinbaseDeposit(roots[1])) {
		t.Fatalf("configured reward chain did not receive root coinbase: %v", got)
	}
	config.ChainID = big.NewInt(10001)
	if got := cursor.current(); got != nil {
		t.Fatal("root reward assigned to a chain other than the first entry")
	}
	config.QuarkChain.ChainIDs[0], config.QuarkChain.ChainIDs[1] = config.QuarkChain.ChainIDs[1], config.QuarkChain.ChainIDs[0]
	if got := cursor.current(); !reflect.DeepEqual(got, rootCoinbaseDeposit(roots[1])) {
		t.Fatalf("root reward did not follow the first entry after reordering: %v", got)
	}
}

// depositAt is a deposit walked, with the cursor on it.
type depositAt struct {
	deposit *types.DepositTx
	cursor  types.XShardCursor
}

func TestXShardInboxDeposits(t *testing.T) {
	roots, a, b, _, d, toShard0, toShard1 := testRootChain()
	var (
		shard0 = NewXShardInbox(params.NewTestQuarkChainConfig(10001, 10001, 10002, 10003), toShard0)
		shard1 = NewXShardInbox(params.NewTestQuarkChainConfig(10002, 10001, 10002, 10003), toShard1)
		eof    = pos(4, 0, 0)
	)
	for _, tt := range []struct {
		name  string
		inbox *XShardInbox
		start types.XShardCursor
		prev  *qkctypes.RootBlock
		want  []depositAt
		end   types.XShardCursor
	}{
		{
			// B is shard 1 itself, C sent nothing, and R1 to R3 pay shard 0.
			name: "shard 1 from R1 to R3", inbox: shard1, start: pos(1, 0, 0), prev: roots[3],
			want: []depositAt{{testDeposit(a, 0), pos(1, 1, 0)}, {testDeposit(a, 3), pos(1, 1, 1)}, {testDeposit(d, 2), pos(2, 1, 0)}},
			end:  eof,
		},
		{
			name: "shard 0 gets the root coinbases", inbox: shard0, start: pos(1, 0, 0), prev: roots[3],
			want: []depositAt{
				{rootCoinbaseDeposit(roots[1]), pos(1, 0, 1)}, {testDeposit(b, 0), pos(1, 2, 0)},
				{rootCoinbaseDeposit(roots[2]), pos(2, 0, 1)}, {testDeposit(d, 5), pos(2, 1, 0)},
				{rootCoinbaseDeposit(roots[3]), pos(3, 0, 1)},
			},
			end: eof,
		},
		{
			name: "up to the previous root block", inbox: shard1, start: pos(1, 0, 0), prev: roots[1],
			want: []depositAt{{testDeposit(a, 0), pos(1, 1, 0)}, {testDeposit(a, 3), pos(1, 1, 1)}},
			end:  pos(2, 0, 0),
		},
		{
			name: "resuming past the previous end", inbox: shard1, start: pos(2, 0, 0), prev: roots[3],
			want: []depositAt{{testDeposit(d, 2), pos(2, 1, 0)}},
			end:  eof,
		},
		{
			name: "resuming in a source block", inbox: shard1, start: pos(1, 1, 0), prev: roots[3],
			want: []depositAt{{testDeposit(a, 3), pos(1, 1, 1)}, {testDeposit(d, 2), pos(2, 1, 0)}},
			end:  eof,
		},
		{
			name: "resuming after a source block", inbox: shard1, start: pos(1, 1, 1), prev: roots[3],
			want: []depositAt{{testDeposit(d, 2), pos(2, 1, 0)}},
			end:  eof,
		},
		{
			name: "at the end", inbox: shard1, start: eof, prev: roots[3],
			end: eof,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			deposits, err := tt.inbox.Deposits(cursorAt(tt.start.RootBlockHeight, tt.start.ShardBlockIndex, tt.start.XShardDepositIndex), tt.prev.Hash())
			if err != nil {
				t.Fatalf("failed to open the deposits: %v", err)
			}
			if cursor := deposits.Cursor(); cursor != tt.start {
				t.Errorf("cursor %+v before the walk, want %+v", cursor, tt.start)
			}
			for i, want := range tt.want {
				deposit, err := deposits.Next()
				if err != nil {
					t.Fatalf("deposit %d: %v", i, err)
				}
				if !reflect.DeepEqual(deposit, want.deposit) {
					t.Fatalf("deposit %d: %+v, want %+v", i, deposit, want.deposit)
				}
				if cursor := deposits.Cursor(); cursor != want.cursor {
					t.Errorf("deposit %d: cursor %+v, want %+v", i, cursor, want.cursor)
				}
			}
			for range 2 {
				if deposit, err := deposits.Next(); deposit != nil || err != nil {
					t.Fatalf("deposit %+v (err %v) after the end", deposit, err)
				}
				if cursor := deposits.Cursor(); cursor != tt.end {
					t.Errorf("cursor %+v at the end, want %+v", cursor, tt.end)
				}
			}
		})
	}
}

func TestXShardInboxErrors(t *testing.T) {
	roots, a, _, _, _, _, toShard1 := testRootChain()
	shard1 := NewXShardInbox(params.NewTestQuarkChainConfig(10002, 10001, 10002, 10003), toShard1)

	if _, err := shard1.Deposits(cursorAt(1, 0, 0), common.Hash{0xff}); !errors.Is(err, ErrUnknownCursorRoot) {
		t.Errorf("unknown previous root block: error %v, want %v", err, ErrUnknownCursorRoot)
	}
	if _, err := shard1.Deposits(&types.Header{Number: new(big.Int), Difficulty: new(big.Int)}, roots[3].Hash()); err == nil {
		t.Error("walked from a parent without a cursor")
	}
	for _, cursor := range []types.XShardCursor{pos(5, 0, 0), pos(4, 1, 0), pos(4, 0, 1)} {
		if _, err := shard1.Deposits(cursorAt(cursor.RootBlockHeight, cursor.ShardBlockIndex, cursor.XShardDepositIndex), roots[3].Hash()); !errors.Is(err, ErrInconsistentClusterState) {
			t.Errorf("invalid EOF cursor %+v: error %v, want %v", cursor, err, ErrInconsistentClusterState)
		}
	}
	invalidShardIndex := uint64(len(roots[1].ShardBlockHeaders())) + 1
	if _, err := shard1.Deposits(cursorAt(1, invalidShardIndex, 0), roots[3].Hash()); !errors.Is(err, ErrInconsistentClusterState) {
		t.Errorf("cursor source index out of range: error %v, want %v", err, ErrInconsistentClusterState)
	}
	delete(toShard1.deposits, a.Hash())
	deposits, err := shard1.Deposits(cursorAt(1, 0, 0), roots[3].Hash())
	if err != nil {
		t.Fatalf("failed to open the deposits: %v", err)
	}
	if _, err := deposits.Next(); !errors.Is(err, ErrXShardDepositsMissing) {
		t.Errorf("deposits not received: error %v, want %v", err, ErrXShardDepositsMissing)
	}
	if _, err := shard1.Deposits(cursorAt(1, 1, 0), roots[3].Hash()); !errors.Is(err, ErrInconsistentClusterState) {
		t.Errorf("deposits missing at the parent cursor: error %v, want %v", err, ErrInconsistentClusterState)
	}
}

func TestXShardInboxRootReference(t *testing.T) {
	roots, _, _, _, _, _, chain := testRootChain()
	fork := newRoot(roots[0], 99)
	forkChild := newRoot(fork, 100)
	chain.roots[fork.Hash()], chain.roots[forkChild.Hash()] = fork, forkChild
	inbox := NewXShardInbox(params.NewTestQuarkChainConfig(10002), chain)
	for _, tt := range []struct {
		name       string
		parentRoot common.Hash
		root       common.Hash
		want       error
	}{
		{"same root", roots[1].Hash(), roots[1].Hash(), nil},
		{"descendant", roots[1].Hash(), roots[3].Hash(), nil},
		{"fork after parent root", roots[0].Hash(), forkChild.Hash(), nil},
		{"backward", roots[1].Hash(), roots[0].Hash(), ErrRootNotDescendant},
		{"same height fork", roots[1].Hash(), fork.Hash(), ErrRootNotDescendant},
		{"higher fork", roots[1].Hash(), forkChild.Hash(), ErrRootNotDescendant},
		{"unknown root", roots[1].Hash(), common.Hash{0xff}, ErrUnknownCursorRoot},
		{"unknown parent root", common.Hash{0xff}, roots[3].Hash(), ErrInconsistentClusterState},
		{"empty non-genesis root", common.Hash{}, roots[3].Hash(), ErrInconsistentClusterState},
	} {
		t.Run(tt.name, func(t *testing.T) {
			parent := &types.Header{
				Number: big.NewInt(1),
				Extra:  (&types.QKCExtra{PrevRootBlock: tt.parentRoot, Cursor: pos(1, 0, 0)}).Encode(),
			}
			if _, err := inbox.Deposits(parent, tt.root); !errors.Is(err, tt.want) {
				t.Fatalf("deposits error %v, want %v", err, tt.want)
			}
		})
	}
	// Missing intermediate root data must not be mistaken for a different fork.
	delete(chain.roots, roots[2].Hash())
	parent := &types.Header{
		Number: big.NewInt(1),
		Extra:  (&types.QKCExtra{PrevRootBlock: roots[1].Hash(), Cursor: pos(4, 0, 0)}).Encode(),
	}
	if _, err := inbox.Deposits(parent, roots[3].Hash()); !errors.Is(err, ErrInconsistentClusterState) {
		t.Fatalf("missing root ancestry: error %v, want %v", err, ErrInconsistentClusterState)
	}
}

func TestXShardInboxMissingRoot(t *testing.T) {
	t.Run("at start", func(t *testing.T) {
		roots, _, _, _, _, _, chain := testRootChain()
		delete(chain.roots, roots[2].Hash())
		inbox := NewXShardInbox(params.NewTestQuarkChainConfig(10002), chain)
		if _, err := inbox.Deposits(cursorAt(1, 0, 0), roots[3].Hash()); !errors.Is(err, ErrInconsistentClusterState) {
			t.Errorf("deposits error %v, want %v", err, ErrInconsistentClusterState)
		}
	})
	t.Run("while walking", func(t *testing.T) {
		roots, _, _, _, _, _, chain := testRootChain()
		inbox := NewXShardInbox(params.NewTestQuarkChainConfig(10002), chain)
		deposits, err := inbox.Deposits(cursorAt(1, 3, 0), roots[3].Hash())
		if err != nil {
			t.Fatal(err)
		}
		delete(chain.roots, roots[2].Hash())
		if _, err := deposits.Next(); !errors.Is(err, ErrInconsistentClusterState) {
			t.Errorf("next deposit error %v, want %v", err, ErrInconsistentClusterState)
		}
	})
}
