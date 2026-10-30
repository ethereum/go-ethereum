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

package eip7807

import (
	"encoding/json"
	"math/big"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/types/bal"
	"github.com/ethereum/go-ethereum/internal/ethapi"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/ethereum/go-ethereum/trie"
)

// fixture is the erigontech/erigon PR 23854 payload fixture as a Geth block:
// two unsigned transactions, two receipts, one withdrawal, one withdrawal
// request, an empty block access list, and a blob schedule of
// {target 1, max 2, update fraction 10000}. Its root is the one payload root
// another client has published for the current EIP text. mutate, if given,
// edits the header before the block is built.
func fixture(mutate func(*types.Header)) (*params.ChainConfig, *types.Block, types.Receipts, [][]byte) {
	var (
		to            = common.HexToAddress("0x1111111111111111111111111111111111111111")
		requests      = erigonRequests()
		requestsHash  = types.CalcRequestsHash(requests)
		accessList    = &bal.BlockAccessList{}
		accessHash    = accessList.Hash()
		blobGasUsed   = uint64(131072)
		excessBlobGas = uint64(4096)
		beaconRoot    = common.HexToHash("0x4444444444444444444444444444444444444444444444444444444444444444")
		slot          = uint64(12345)
	)
	txs := []*types.Transaction{
		types.NewTx(&types.LegacyTx{Nonce: 1, GasPrice: big.NewInt(3), Gas: 21_000, To: &to, Value: big.NewInt(2), Data: []byte{0xaa}}),
		types.NewTx(&types.DynamicFeeTx{ChainID: big.NewInt(1), Nonce: 2, GasTipCap: big.NewInt(5), GasFeeCap: big.NewInt(6), Gas: 50_000, To: &to, Value: big.NewInt(4), Data: []byte{0xbb, 0xcc}}),
	}
	receipts := types.Receipts{
		{Type: types.LegacyTxType, Status: types.ReceiptStatusSuccessful, CumulativeGasUsed: 21_000},
		{Type: types.DynamicFeeTxType, Status: types.ReceiptStatusFailed, CumulativeGasUsed: 42_000},
	}
	withdrawals := []*types.Withdrawal{{Index: 1, Validator: 2, Address: to, Amount: 3}}
	header := &types.Header{
		ParentHash:          common.HexToHash("0x1111111111111111111111111111111111111111111111111111111111111111"),
		Coinbase:            common.HexToAddress("0x2222222222222222222222222222222222222222"),
		Root:                common.HexToHash("0x3333333333333333333333333333333333333333333333333333333333333333"),
		Number:              big.NewInt(99),
		GasLimit:            30_000_000,
		GasUsed:             42_000,
		Time:                1,
		Extra:               common.FromHex("deadbeef"),
		MixDigest:           common.HexToHash("0x5555555555555555555555555555555555555555555555555555555555555555"),
		BaseFee:             big.NewInt(1_000_000_000),
		BlobGasUsed:         &blobGasUsed,
		ExcessBlobGas:       &excessBlobGas,
		ParentBeaconRoot:    &beaconRoot,
		RequestsHash:        &requestsHash,
		BlockAccessListHash: &accessHash,
		SlotNumber:          &slot,
	}
	if mutate != nil {
		mutate(header)
	}
	block := types.NewBlock(header, &types.Body{Transactions: txs, Withdrawals: withdrawals}, receipts, trie.NewStackTrie(nil)).WithAccessList(accessList)
	config := &params.ChainConfig{
		ChainID:     big.NewInt(1),
		LondonBlock: big.NewInt(0),
		CancunTime:  new(uint64),
		BlobScheduleConfig: &params.BlobScheduleConfig{
			Cancun: &params.BlobConfig{Target: 1, Max: 2, UpdateFraction: 10_000},
		},
	}
	return config, block, receipts, requests
}

// TestPayloadFromBlock checks every field the mapping fills against the
// fixture's sources and the roots Erigon pins.
func TestPayloadFromBlock(t *testing.T) {
	config, block, receipts, requests := fixture(nil)
	p, err := PayloadFromBlock(config, block, receipts, requests)
	if err != nil {
		t.Fatal(err)
	}
	header := block.Header()
	checks := []struct {
		name      string
		got, want any
	}{
		{"parent hash", p.ParentHash, header.ParentHash},
		{"miner", p.Miner, header.Coinbase},
		{"state root", p.StateRoot, header.Root},
		{"transaction count", len(p.Transactions), 2},
		{"transaction 1 type", p.Transactions[1][0], byte(types.DynamicFeeTxType)},
		{"number", p.Number, uint64(99)},
		{"gas limits", p.GasLimits, types.GasAmounts{Regular: 30_000_000, Blob: 2 * params.BlobTxBlobGasPerBlob}},
		{"gas used", p.GasUsed, types.GasAmounts{Regular: 42_000, Blob: 131072}},
		{"timestamp", p.Timestamp, uint64(1)},
		{"extra data", common.Bytes2Hex(p.ExtraData), "deadbeef"},
		{"mix hash", p.MixHash, header.MixDigest},
		{"base fee", p.BaseFeesPerGas.Regular.Uint64(), uint64(1_000_000_000)},
		{"blob base fee", p.BaseFeesPerGas.Blob.Uint64(), uint64(1)},
		{"withdrawal count", len(p.Withdrawals), 1},
		{"excess gas", p.ExcessGas, types.GasAmounts{Regular: 0, Blob: 4096}},
		{"parent beacon block root", p.ParentBeaconBlockRoot, *header.ParentBeaconRoot},
		{"block access list", common.Bytes2Hex(p.BlockAccessList), "c0"},
		{"slot number", p.SlotNumber, uint64(12345)},
		{"receipts root", p.ReceiptsRoot, common.HexToHash("0x4e93e5e6c10dd15a002b4447369b75d15ead60ee8caf9186ae9850f07a9c4088")},
		{"requests hash", p.RequestsHash, common.HexToHash("0xb1c4c413201f39ef0e4528edbd42e595a5258578a22e8634a9dee79684fdf245")},
	}
	for _, c := range checks {
		if c.got != c.want {
			t.Errorf("%s: %v, want %v", c.name, c.got, c.want)
		}
	}
	// The root of the mapped payload, computed by the SSZ executable
	// reference (eth-ssz-specs 0.1.0) from the same values.
	root, err := p.Root()
	if err != nil {
		t.Fatal(err)
	}
	if want := common.HexToHash("0x854226253e114b2ccdae85d48c9f4b6eb8de49f062c12e48b10985db314f5ce4"); root != want {
		t.Errorf("payload root %x, want %x", root, want)
	}
	if root == block.Hash() {
		t.Error("payload root equals the keccak block hash")
	}
	// Erigon's adapter takes excess_gas.regular as an input, 7 in its fixture,
	// where the header has no source for it and this mapping sets 0. With that
	// one value substituted, the mapping reproduces the root Erigon pins, so
	// every other field agrees with its implementation.
	p.ExcessGas.Regular = 7
	root, err = p.Root()
	if err != nil {
		t.Fatal(err)
	}
	if want := common.HexToHash("0x51951516710af90e8c20092e086f37232c69b558d6ba0035333982e115e1f627"); root != want {
		t.Errorf("payload root with Erigon's regular excess gas %x, want %x", root, want)
	}
}

// referenceDump renders a block as JSON-RPC shaped data, with the requests,
// the block access list and the blob schedule the RPC does not carry, for the
// fellowship's Python reference. That script re-encodes every element from
// the data and recomputes the root with the SSZ executable reference, so
// agreement checks the mapping's choices, not only the hashing.
func referenceDump(config *params.ChainConfig, block *types.Block, receipts types.Receipts, requests [][]byte, root common.Hash) ([]byte, error) {
	flat := make([]hexutil.Bytes, len(requests))
	for i, r := range requests {
		flat[i] = r
	}
	accessList, err := rlp.EncodeToBytes(block.AccessList())
	if err != nil {
		return nil, err
	}
	return json.MarshalIndent(map[string]any{
		"block":             ethapi.RPCMarshalBlock(block, true, true, config),
		"receipts":          receipts,
		"executionRequests": flat,
		"blockAccessList":   hexutil.Bytes(accessList),
		"blobSchedule":      config.BlobConfig(config.LatestFork(block.Time())),
		"root":              root,
	}, "", " ")
}

// TestReferenceDump writes the fixture for the Python reference. It runs only
// when EIP7807_REFERENCE_DUMP names the directory to write into.
func TestReferenceDump(t *testing.T) {
	dir := os.Getenv("EIP7807_REFERENCE_DUMP")
	if dir == "" {
		t.Skip("EIP7807_REFERENCE_DUMP not set")
	}
	config, block, receipts, requests := fixture(nil)
	p, err := PayloadFromBlock(config, block, receipts, requests)
	if err != nil {
		t.Fatal(err)
	}
	root, err := p.Root()
	if err != nil {
		t.Fatal(err)
	}
	dump, err := referenceDump(config, block, receipts, requests, root)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "fixture.json"), dump, 0o644); err != nil {
		t.Fatal(err)
	}
}

// TestPayloadFromBlockRejects covers the blocks and inputs the mapping
// refuses: a header from before a fork that added a field, and inputs the
// header does not commit to.
func TestPayloadFromBlockRejects(t *testing.T) {
	for _, c := range []struct {
		name   string
		header func(*types.Header)
		err    string
	}{
		{"before London", func(h *types.Header) { h.BaseFee = nil }, "no base fee"},
		{"before Cancun", func(h *types.Header) { h.BlobGasUsed = nil }, "no blob gas"},
		{"no beacon root", func(h *types.Header) { h.ParentBeaconRoot = nil }, "no parent beacon block root"},
		{"before Prague", func(h *types.Header) { h.RequestsHash = nil }, "no requests hash"},
		{"before Amsterdam", func(h *types.Header) { h.BlockAccessListHash = nil }, "no block access list hash"},
		{"before Glamsterdam", func(h *types.Header) { h.SlotNumber = nil }, "no slot number"},
		{"wrong requests hash", func(h *types.Header) { h.RequestsHash = &common.Hash{} }, "requests hash"},
		{"wrong access list hash", func(h *types.Header) { h.BlockAccessListHash = &common.Hash{} }, "block access list hash"},
		{"base fee overflow", func(h *types.Header) { h.BaseFee = new(big.Int).Lsh(big.NewInt(1), 256) }, "base fee does not fit"},
		{"long extra data", func(h *types.Header) { h.Extra = make([]byte, 33) }, "extra data of 33 bytes"},
		{"number past uint64", func(h *types.Header) { h.Number = new(big.Int).Lsh(big.NewInt(1), 64) }, "does not fit uint64"},
	} {
		config, block, receipts, requests := fixture(c.header)
		if _, err := PayloadFromBlock(config, block, receipts, requests); err == nil || !strings.Contains(err.Error(), c.err) {
			t.Errorf("%s: error %v, want %q", c.name, err, c.err)
		}
	}

	config, block, receipts, requests := fixture(nil)
	if _, err := PayloadFromBlock(config, block, receipts[:1], requests); err == nil || !strings.Contains(err.Error(), "1 receipts for 2") {
		t.Errorf("receipt count: %v", err)
	}
	other := types.Receipts{receipts[1], receipts[0]}
	if _, err := PayloadFromBlock(config, block, other, requests); err == nil || !strings.Contains(err.Error(), "receipts root") {
		t.Errorf("receipts mismatch: %v", err)
	}
	if _, err := PayloadFromBlock(config, block, receipts, nil); err == nil || !strings.Contains(err.Error(), "requests hash") {
		t.Errorf("requests mismatch: %v", err)
	}
	resealed := block.Header()
	resealed.TxHash = common.Hash{}
	if _, err := PayloadFromBlock(config, block.WithSeal(resealed), receipts, requests); err == nil || !strings.Contains(err.Error(), "transactions root") {
		t.Errorf("body mismatch: %v", err)
	}
	resealed = block.Header()
	resealed.WithdrawalsHash = &common.Hash{}
	if _, err := PayloadFromBlock(config, block.WithSeal(resealed), receipts, requests); err == nil || !strings.Contains(err.Error(), "withdrawals root") {
		t.Errorf("withdrawals mismatch: %v", err)
	}
	withoutAccessList := types.NewBlock(block.Header(), block.Body(), receipts, trie.NewStackTrie(nil))
	if _, err := PayloadFromBlock(config, withoutAccessList, receipts, requests); err == nil || !strings.Contains(err.Error(), "no access list") {
		t.Errorf("missing access list: %v", err)
	}
	if _, err := PayloadFromBlock(&params.ChainConfig{LondonBlock: big.NewInt(0)}, block, receipts, requests); err == nil || !strings.Contains(err.Error(), "no blob schedule") {
		t.Errorf("no blob schedule: %v", err)
	}
}
