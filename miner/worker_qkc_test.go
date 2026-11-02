// Copyright 2026 The quarkchain Authors
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

package miner

import (
	"context"
	"errors"
	"maps"
	"math/big"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/consensus"
	"github.com/ethereum/go-ethereum/consensus/beacon"
	"github.com/ethereum/go-ethereum/consensus/ethash"
	"github.com/ethereum/go-ethereum/consensus/posw"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/txpool"
	"github.com/ethereum/go-ethereum/core/txpool/legacypool"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/holiman/uint256"
)

// Header sealing is outside block construction; execute and validate the body normally.
type qkcTestEngine struct{ consensus.Engine }

func (qkcTestEngine) VerifyHeader(consensus.ChainHeaderReader, *types.Header) error { return nil }
func (qkcTestEngine) VerifyHeaders(_ consensus.ChainHeaderReader, headers []*types.Header) (chan<- struct{}, <-chan error) {
	results := make(chan error, len(headers))
	for range headers {
		results <- nil
	}
	return make(chan struct{}), results
}

type miningInbox struct {
	deposits  []*types.DepositTx
	failAfter int
	err       error
}

func (in *miningInbox) Deposits(*types.Header, common.Hash) (core.XShardDeposits, error) {
	return &miningDeposits{inbox: in}, nil
}

type miningDeposits struct {
	inbox    *miningInbox
	consumed int
	eof      bool
}

func (d *miningDeposits) Next() (*types.DepositTx, error) {
	if d.inbox.err != nil && d.consumed == d.inbox.failAfter {
		return nil, d.inbox.err
	}
	if d.consumed == len(d.inbox.deposits) {
		d.eof = true
		return nil, nil
	}
	tx := d.inbox.deposits[d.consumed]
	d.consumed++
	return tx, nil
}

func (d *miningDeposits) Cursor() types.XShardCursor {
	if d.eof {
		return types.XShardCursor{RootBlockHeight: 2}
	}
	return types.XShardCursor{RootBlockHeight: 1, XShardDepositIndex: uint64(d.consumed)}
}

func TestTotalFeesWithDeposits(t *testing.T) {
	txs := []*types.Transaction{
		types.NewTx(&types.DepositTx{GasPrice: uint256.NewInt(10), Gas: 9000}),
		types.NewTx(&types.DepositTx{GasPrice: uint256.NewInt(2), Gas: 9000}),
		types.NewTx(&types.DepositTx{GasPrice: new(uint256.Int), Gas: 9000}),
		types.NewTx(&types.LegacyTx{GasPrice: big.NewInt(15), Gas: 21000}),
		types.NewTx(&types.DynamicFeeTx{GasFeeCap: big.NewInt(10), GasTipCap: big.NewInt(5), Gas: 22000}),
		types.NewTx(&types.DynamicFeeTx{Nonce: 1, GasFeeCap: big.NewInt(10), GasTipCap: big.NewInt(5), Gas: 22000}),
	}
	block := types.NewBlockWithHeader(&types.Header{BaseFee: big.NewInt(7)}).WithBody(types.Body{Transactions: txs})
	receipts := types.Receipts{
		{GasUsed: 9000}, {GasUsed: 9000}, {GasUsed: 9000}, {GasUsed: 21000}, {GasUsed: 21001}, {GasUsed: 21001},
	}
	// Deposits pay 10, 2 and 0 per gas; ordinary transactions pay tips of 8 and 3.
	want := big.NewInt(9000*(10+2) + 21000*8 + 2*63003)
	config := *params.MergedTestChainConfig
	if fees := totalFees(&config, block, receipts); fees.Cmp(want) != 0 {
		t.Fatalf("total fees %v, want %v", fees, want)
	}
	// Tax each transaction at 50%. The two odd fees each retain 31501;
	// taxing their combined amount would overstate the miner's income by one.
	config = *params.NewTestQuarkChainConfig(10001)
	want = big.NewInt(9000*(5+1) + 21000*4 + 2*31501)
	if fees := totalFees(&config, block, receipts); fees.Cmp(want) != 0 {
		t.Fatalf("after-tax total fees %v, want %v", fees, want)
	}
}

func TestBuildXShardDeposits(t *testing.T) {
	missing := errors.New("missing deposit list")
	for _, amsterdam := range []bool{false, true} {
		fork := "osaka"
		if amsterdam {
			fork = "amsterdam"
		}
		t.Run(fork, func(t *testing.T) {
			for _, tt := range []struct {
				name                       string
				limit                      uint64
				available, included, users int
				eof, noTxs, missingInbox   bool
				failAfter                  int
				iteratorErr, wantErr       error
				edit                       func([]*types.DepositTx)
			}{
				{name: "empty inbox", limit: 100000, eof: true},
				{name: "empty below move gas", limit: 17000, eof: true},
				{name: "first does not fit", limit: 17000, available: 1},
				{name: "odd gas limit", limit: 100001, available: 6, included: 5},
				{name: "exact with backlog", limit: 90000, available: 6, included: 5},
				{name: "exact at EOF", limit: 90000, available: 5, included: 5, eof: true},
				{name: "unused deposit gas goes to users", limit: 100000, available: 2, included: 2, users: 3, eof: true},
				{name: "deposits precede users", limit: 100000, available: 6, included: 5, users: 2},
				{name: "noTxs keeps deposits", limit: 100000, available: 2, included: 2, users: 3, noTxs: true, eof: true},
				{name: "zero price reward", limit: 100000, available: 1, included: 1, eof: true,
					edit: func(ds []*types.DepositTx) { ds[0].GasPrice.Clear() }},
				{name: "declared gas blocks next deposit", limit: 100000, available: 2, included: 1,
					edit: func(ds []*types.DepositTx) { ds[1].Gas = 41001 }},
				{name: "non move rejected", limit: 100000, available: 1, wantErr: core.ErrDepositNotMove,
					edit: func(ds []*types.DepositTx) { ds[0].IsMove = false }},
				{name: "missing initial list", limit: 100000, iteratorErr: missing, wantErr: missing},
				{name: "missing list at exact exhaustion", limit: 90000, available: 5, failAfter: 5, iteratorErr: missing, wantErr: missing},
				{name: "missing inbox", limit: 100000, missingInbox: true},
			} {
				t.Run(tt.name, func(t *testing.T) {
					config := *params.NewTestQuarkChainConfig(10002, 10001, 10002)
					if amsterdam {
						config.AmsterdamTime = new(uint64)
					}
					alloc := core.SystemContractAllocs()
					maps.Copy(alloc, core.PoSWStakingContractAllocs(&config))
					alloc[testBankAddress] = types.Account{Balance: testBankFunds}
					// Deposits to a contract must remain moves, even if its code reverts.
					alloc[testUserAddress] = types.Account{Code: common.FromHex("0x60006000fd"), Balance: new(big.Int)}
					genesis := &core.Genesis{
						Config: &config, GasLimit: tt.limit,
						ExtraData: (&types.QKCExtra{ChainID: big.NewInt(10002), Cursor: types.XShardCursor{RootBlockHeight: 1}}).Encode(),
						Alloc:     alloc,
					}
					engine := qkcTestEngine{beacon.New(ethash.NewFaker())}
					chain, err := core.NewBlockChain(rawdb.NewMemoryDatabase(), genesis, engine, nil)
					if err != nil {
						t.Fatal(err)
					}
					defer chain.Stop()
					pool, err := txpool.New(testTxPoolConfig.PriceLimit, chain, []txpool.SubPool{legacypool.New(testTxPoolConfig, chain)})
					if err != nil {
						t.Fatal(err)
					}
					defer pool.Close()
					inbox := &miningInbox{failAfter: tt.failAfter, err: tt.iteratorErr}
					for i := range tt.available {
						inbox.deposits = append(inbox.deposits, &types.DepositTx{
							Version: types.DepositTxVersion, SourceBlockHash: common.Hash{1}, SourceLogIndex: uint64(i),
							From: testBankAddress, To: &testUserAddress, Value: uint256.NewInt(1),
							Gas: params.XShardSendGas, GasPrice: uint256.NewInt(params.GWei), IsMove: true,
						})
					}
					if tt.edit != nil {
						tt.edit(inbox.deposits)
					}
					if !tt.missingInbox {
						chain.SetXShardInbox(inbox)
					}
					for i := range tt.users {
						tx := types.MustSignNewTx(testBankKey, types.LatestSigner(&config), &types.DynamicFeeTx{
							ChainID: config.ChainID, Nonce: uint64(i), Gas: params.TxGas,
							To: &common.Address{0xee}, Value: big.NewInt(1),
							GasFeeCap: big.NewInt(2 * params.GWei), GasTipCap: big.NewInt(params.GWei),
						})
						if err := pool.Add([]*types.Transaction{tx}, true)[0]; err != nil {
							t.Fatal(err)
						}
					}
					mc := DefaultConfig
					mc.GasCeil = tt.limit
					miner := New(NewMockBackend(chain, pool), mc, engine)
					args := &generateParams{
						timestamp: 1, forceTime: true, parentHash: chain.Genesis().Hash(),
						coinbase: common.Address{0xcc},
						qkc:      &types.QKCExtra{ChainID: big.NewInt(10002), PrevRootBlock: common.Hash{0x77}}, noTxs: tt.noTxs,
					}
					result := miner.generateWork(context.Background(), args, false)
					if tt.missingInbox {
						if result.err == nil {
							t.Fatal("built without an inbox")
						}
						return
					}
					if tt.wantErr != nil {
						if !errors.Is(result.err, tt.wantErr) {
							t.Fatalf("error %v, want %v", result.err, tt.wantErr)
						}
						return
					}
					if result.err != nil {
						t.Fatal(result.err)
					}
					header := result.block.Header()
					if header.ParentBeaconRoot == nil || *header.ParentBeaconRoot != (common.Hash{}) {
						t.Fatal("QKC beacon root is not zero")
					}
					if amsterdam && (header.SlotNumber == nil || *header.SlotNumber != 0) {
						t.Fatal("QKC slot number is not zero")
					}
					if got := result.stateDB.GetState(params.BeaconRootsAddress, common.BigToHash(big.NewInt(1))); got != (common.Hash{}) {
						t.Fatalf("beacon timestamp written by QKC builder: %x", got)
					}
					extra, err := result.block.Header().QKCExtra()
					if err != nil {
						t.Fatal(err)
					}
					cursor := types.XShardCursor{RootBlockHeight: 1, XShardDepositIndex: uint64(tt.included)}
					if tt.eof {
						cursor = types.XShardCursor{RootBlockHeight: 2}
					}
					if extra.Cursor != cursor || extra.XShardGasLimit != tt.limit/2 || extra.XShardGasUsed != uint64(tt.included)*params.XShardSendGas {
						t.Fatalf("extra %+v, want cursor %+v and %d deposits", extra, cursor, tt.included)
					}
					users := tt.users
					if tt.noTxs {
						users = 0
					}
					txs := result.block.Transactions()
					if len(txs) != tt.included+users {
						t.Fatalf("transactions %d, want %d", len(txs), tt.included+users)
					}
					for i := range tt.included {
						if txs[i].Hash() != types.NewTx(inbox.deposits[i]).Hash() {
							t.Fatalf("deposit %d differs", i)
						}
					}
					if balance := result.stateDB.GetBalance(testUserAddress); !balance.Eq(uint256.NewInt(uint64(tt.included))) {
						t.Fatalf("recipient balance %v, want %d", balance, tt.included)
					}
					if result.stateDB.GetNonce(testBankAddress) != uint64(users) {
						t.Fatal("deposits changed the sender nonce")
					}
					if got := result.stateDB.GetBalance(args.coinbase).ToBig(); got.Cmp(result.fees) != 0 {
						t.Fatalf("coinbase fees %v, reported %v", got, result.fees)
					}
					// Exercise the wire encoding before re-execution and deposit validation.
					encoded, err := rlp.EncodeToBytes(result.block)
					if err != nil {
						t.Fatal(err)
					}
					var decoded types.Block
					if err := rlp.DecodeBytes(encoded, &decoded); err != nil {
						t.Fatal(err)
					}
					if decoded.Hash() != result.block.Hash() {
						t.Fatal("RLP round trip changed block hash")
					}
					_, err = chain.InsertChain([]*types.Block{&decoded})
					if amsterdam && tt.limit < 2*params.XShardSendGas {
						// Like geth, construction permits this candidate; import rejects its oversized BAL.
						if err == nil || !strings.Contains(err.Error(), "block access list exceeds size constraint") {
							t.Fatalf("expected access-list size error on import, got %v", err)
						}
						return
					}
					if err != nil {
						t.Fatalf("import built block: %v", err)
					}
				})
			}
		})
	}
}

func TestBuildQKCBlockDifficulty(t *testing.T) {
	config := *params.NewTestQuarkChainConfig(10002, 10001, 10002)
	// The contract stores PREVRANDAO, so import fails unless re-execution sees
	// the difficulty the builder used.
	contract := common.Address{0xdd}
	alloc := core.SystemContractAllocs()
	maps.Copy(alloc, core.PoSWStakingContractAllocs(&config))
	alloc[testBankAddress] = types.Account{Balance: testBankFunds}
	alloc[contract] = types.Account{Code: common.FromHex("0x4460005500"), Balance: new(big.Int)}
	genesis := &core.Genesis{
		Config: &config, GasLimit: 1_000_000, Mixhash: common.BigToHash(big.NewInt(1_000_000)),
		ExtraData: (&types.QKCExtra{ChainID: big.NewInt(10002), Cursor: types.XShardCursor{RootBlockHeight: 1}}).Encode(),
		Alloc:     alloc,
	}
	// The PoSW engine prepares the difficulty, before the builder executes transactions.
	pow, err := posw.New(&config, genesis.ToBlock().Header())
	if err != nil {
		t.Fatal(err)
	}
	engine := qkcTestEngine{pow}
	chain, err := core.NewBlockChain(rawdb.NewMemoryDatabase(), genesis, engine, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer chain.Stop()
	chain.SetXShardInbox(&miningInbox{})
	pool, err := txpool.New(testTxPoolConfig.PriceLimit, chain, []txpool.SubPool{legacypool.New(testTxPoolConfig, chain)})
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	tx := types.MustSignNewTx(testBankKey, types.LatestSigner(&config), &types.DynamicFeeTx{
		ChainID: config.ChainID, Gas: 100_000, To: &contract,
		GasFeeCap: big.NewInt(2 * params.GWei), GasTipCap: big.NewInt(params.GWei),
	})
	if err := pool.Add([]*types.Transaction{tx}, true)[0]; err != nil {
		t.Fatal(err)
	}
	miner := New(NewMockBackend(chain, pool), DefaultConfig, engine)
	parent := chain.Genesis().Header()
	// A fast block raises the difficulty and a slow one lowers it.
	for i, timestamp := range []uint64{5, 25} {
		block, err := miner.BuildQKCBlock(context.Background(), QKCBlockArgs{
			Parent: parent.Hash(), PrevRootBlock: common.Hash{0x77}, Timestamp: timestamp, NoTxs: i > 0,
		})
		if err != nil {
			t.Fatal(err)
		}
		want := posw.CalcDifficulty(config.QuarkChain.PoSW, timestamp, parent.Time, parent.QKCDifficulty())
		if got := block.Header().QKCDifficulty(); got.Cmp(want) != 0 || got.Cmp(parent.QKCDifficulty()) == 0 {
			t.Fatalf("block %d difficulty %v, want %v", i+1, got, want)
		}
		if _, err := chain.InsertChain([]*types.Block{block}); err != nil {
			t.Fatalf("import block %d: %v", i+1, err)
		}
		if i == 0 {
			if len(block.Transactions()) != 1 {
				t.Fatal("missing the PREVRANDAO transaction")
			}
			state, err := chain.StateAt(block.Root(), block.Number(), block.Time())
			if err != nil {
				t.Fatal(err)
			}
			if got := state.GetState(contract, common.Hash{}); got != block.MixDigest() {
				t.Fatalf("PREVRANDAO %x, want difficulty %x", got, block.MixDigest())
			}
		}
		parent = block.Header()
	}
}
