// Copyright 2020 The go-ethereum Authors
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

package eth

import (
	"bytes"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rlp"
)

// Tests that the custom union field encoder and decoder works correctly.
func TestGetBlockHeadersDataEncodeDecode(t *testing.T) {
	// Create a "random" hash for testing
	var hash common.Hash
	for i := range hash {
		hash[i] = byte(i)
	}
	// Assemble some table driven tests
	tests := []struct {
		packet *GetBlockHeadersPacket
		fail   bool
	}{
		// Providing the origin as either a hash or a number should both work
		{fail: false, packet: &GetBlockHeadersPacket{Origin: HashOrNumber{Number: 314}}},
		{fail: false, packet: &GetBlockHeadersPacket{Origin: HashOrNumber{Hash: hash}}},

		// Providing arbitrary query field should also work
		{fail: false, packet: &GetBlockHeadersPacket{Origin: HashOrNumber{Number: 314}, Amount: 314, Skip: 1, Reverse: true}},
		{fail: false, packet: &GetBlockHeadersPacket{Origin: HashOrNumber{Hash: hash}, Amount: 314, Skip: 1, Reverse: true}},

		// Providing both the origin hash and origin number must fail
		{fail: true, packet: &GetBlockHeadersPacket{Origin: HashOrNumber{Hash: hash, Number: 314}}},
	}
	// Iterate over each of the tests and try to encode and then decode
	for i, tt := range tests {
		bytes, err := rlp.EncodeToBytes(tt.packet)
		if err != nil && !tt.fail {
			t.Fatalf("test %d: failed to encode packet: %v", i, err)
		} else if err == nil && tt.fail {
			t.Fatalf("test %d: encode should have failed", i)
		}
		if !tt.fail {
			packet := new(GetBlockHeadersPacket)
			if err := rlp.DecodeBytes(bytes, packet); err != nil {
				t.Fatalf("test %d: failed to decode packet: %v", i, err)
			}
			if packet.Origin.Hash != tt.packet.Origin.Hash || packet.Origin.Number != tt.packet.Origin.Number || packet.Amount != tt.packet.Amount ||
				packet.Skip != tt.packet.Skip || packet.Reverse != tt.packet.Reverse {
				t.Fatalf("test %d: encode decode mismatch: have %+v, want %+v", i, packet, tt.packet)
			}
		}
	}
}

// TestEth66EmptyMessages tests encoding of empty eth66 messages
func TestEth66EmptyMessages(t *testing.T) {
	// All empty messages encodes to the same format
	want := common.FromHex("c4820457c0")

	for i, msg := range []interface{}{
		// Headers
		GetBlockHeadersPacket66{1111, nil},
		BlockHeadersPacket66{1111, nil},
		// Bodies
		GetBlockBodiesPacket66{1111, nil},
		BlockBodiesPacket66{1111, nil},
		BlockBodiesRLPPacket66{1111, nil},
		// Node data
		GetNodeDataPacket66{1111, nil},
		NodeDataPacket66{1111, nil},
		// Receipts
		GetReceiptsPacket66{1111, nil},
		ReceiptsPacket66{1111, nil},
		// Transactions
		GetPooledTransactionsPacket66{1111, nil},
		PooledTransactionsPacket66{1111, nil},
		PooledTransactionsRLPPacket66{1111, nil},

		// Headers
		BlockHeadersPacket66{1111, BlockHeadersPacket([]*types.Header{})},
		// Bodies
		GetBlockBodiesPacket66{1111, GetBlockBodiesPacket([]common.Hash{})},
		BlockBodiesPacket66{1111, BlockBodiesPacket([]*BlockBody{})},
		BlockBodiesRLPPacket66{1111, BlockBodiesRLPPacket([]rlp.RawValue{})},
		// Node data
		GetNodeDataPacket66{1111, GetNodeDataPacket([]common.Hash{})},
		NodeDataPacket66{1111, NodeDataPacket([][]byte{})},
		// Receipts
		GetReceiptsPacket66{1111, GetReceiptsPacket([]common.Hash{})},
		ReceiptsPacket66{1111, ReceiptsPacket([][]*types.Receipt{})},
		// Transactions
		GetPooledTransactionsPacket66{1111, GetPooledTransactionsPacket([]common.Hash{})},
		PooledTransactionsPacket66{1111, PooledTransactionsPacket([]*types.Transaction{})},
		PooledTransactionsRLPPacket66{1111, PooledTransactionsRLPPacket([]rlp.RawValue{})},
	} {
		if have, _ := rlp.EncodeToBytes(msg); !bytes.Equal(have, want) {
			t.Errorf("test %d, type %T, have\n\t%x\nwant\n\t%x", i, msg, have, want)
		}
	}
}

// TestEth66Messages tests the encoding of all redefined eth66 messages
func TestEth66Messages(t *testing.T) {
	// Some basic structs used during testing
	var (
		header       *types.Header
		blockBody    *BlockBody
		blockBodyRlp rlp.RawValue
		txs          []*types.Transaction
		txRlps       []rlp.RawValue
		hashes       []common.Hash
		receipts     []*types.Receipt
		receiptsRlp  rlp.RawValue

		err error
	)
	header = &types.Header{
		Difficulty: big.NewInt(2222),
		Number:     big.NewInt(3333),
		GasLimit:   4444,
		GasUsed:    5555,
		Time:       6666,
		Extra:      []byte{0x77, 0x88},
	}
	// Init the transactions, taken from a different test
	{
		for _, hexrlp := range []string{
			"f867088504a817c8088302e2489435353535353535353535353535353535353535358202008025a064b1702d9298fee62dfeccc57d322a463ad55ca201256d01f62b45b2e1c21c12a064b1702d9298fee62dfeccc57d322a463ad55ca201256d01f62b45b2e1c21c10",
			"f867098504a817c809830334509435353535353535353535353535353535353535358202d98025a052f8f61201b2b11a78d6e866abc9c3db2ae8631fa656bfe5cb53668255367afba052f8f61201b2b11a78d6e866abc9c3db2ae8631fa656bfe5cb53668255367afb",
		} {
			var tx *types.Transaction
			rlpdata := common.FromHex(hexrlp)
			if err := rlp.DecodeBytes(rlpdata, &tx); err != nil {
				t.Fatal(err)
			}
			txs = append(txs, tx)
			txRlps = append(txRlps, rlpdata)
		}
	}
	// init the block body data, both object and rlp form
	blockBody = &BlockBody{
		Transactions: txs,
		Uncles:       []*types.Header{header},
	}
	blockBodyRlp, err = rlp.EncodeToBytes(blockBody)
	if err != nil {
		t.Fatal(err)
	}

	hashes = []common.Hash{
		common.HexToHash("deadc0de"),
		common.HexToHash("feedbeef"),
	}
	byteSlices := [][]byte{
		common.FromHex("deadc0de"),
		common.FromHex("feedbeef"),
	}
	// init the receipts
	{
		receipts = []*types.Receipt{
			{
				Status:            types.ReceiptStatusFailed,
				CumulativeGasUsed: 1,
				Logs: []*types.Log{
					{
						Address: common.BytesToAddress([]byte{0x11}),
						Topics:  []common.Hash{common.HexToHash("dead"), common.HexToHash("beef")},
						Data:    []byte{0x01, 0x00, 0xff},
					},
				},
				TxHash:          hashes[0],
				ContractAddress: common.BytesToAddress([]byte{0x01, 0x11, 0x11}),
				GasUsed:         111111,
			},
		}
		rlpData, err := rlp.EncodeToBytes(receipts)
		if err != nil {
			t.Fatal(err)
		}
		receiptsRlp = rlpData
	}

	for i, tc := range []struct {
		message interface{}
		want    []byte
	}{
		{
			GetBlockHeadersPacket66{1111, &GetBlockHeadersPacket{HashOrNumber{hashes[0], 0}, 5, 5, false}},
			common.FromHex("e8820457e4a000000000000000000000000000000000000000000000000000000000deadc0de050580"),
		},
		{
			GetBlockHeadersPacket66{1111, &GetBlockHeadersPacket{HashOrNumber{common.Hash{}, 9999}, 5, 5, false}},
			common.FromHex("ca820457c682270f050580"),
		},
		{
			BlockHeadersPacket66{1111, BlockHeadersPacket{header}},
			common.FromHex("f90202820457f901fcf901f9a00000000000000000000000000000000000000000000000000000000000000000a00000000000000000000000000000000000000000000000000000000000000000940000000000000000000000000000000000000000a00000000000000000000000000000000000000000000000000000000000000000a00000000000000000000000000000000000000000000000000000000000000000a00000000000000000000000000000000000000000000000000000000000000000b90100000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000008208ae820d0582115c8215b3821a0a827788a00000000000000000000000000000000000000000000000000000000000000000880000000000000000"),
		},
		{
			GetBlockBodiesPacket66{1111, GetBlockBodiesPacket(hashes)},
			common.FromHex("f847820457f842a000000000000000000000000000000000000000000000000000000000deadc0dea000000000000000000000000000000000000000000000000000000000feedbeef"),
		},
		{
			BlockBodiesPacket66{1111, BlockBodiesPacket([]*BlockBody{blockBody})},
			common.FromHex("f902dc820457f902d6f902d3f8d2f867088504a817c8088302e2489435353535353535353535353535353535353535358202008025a064b1702d9298fee62dfeccc57d322a463ad55ca201256d01f62b45b2e1c21c12a064b1702d9298fee62dfeccc57d322a463ad55ca201256d01f62b45b2e1c21c10f867098504a817c809830334509435353535353535353535353535353535353535358202d98025a052f8f61201b2b11a78d6e866abc9c3db2ae8631fa656bfe5cb53668255367afba052f8f61201b2b11a78d6e866abc9c3db2ae8631fa656bfe5cb53668255367afbf901fcf901f9a00000000000000000000000000000000000000000000000000000000000000000a00000000000000000000000000000000000000000000000000000000000000000940000000000000000000000000000000000000000a00000000000000000000000000000000000000000000000000000000000000000a00000000000000000000000000000000000000000000000000000000000000000a00000000000000000000000000000000000000000000000000000000000000000b90100000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000008208ae820d0582115c8215b3821a0a827788a00000000000000000000000000000000000000000000000000000000000000000880000000000000000"),
		},
		{ // Identical to non-rlp-shortcut version
			BlockBodiesRLPPacket66{1111, BlockBodiesRLPPacket([]rlp.RawValue{blockBodyRlp})},
			common.FromHex("f902dc820457f902d6f902d3f8d2f867088504a817c8088302e2489435353535353535353535353535353535353535358202008025a064b1702d9298fee62dfeccc57d322a463ad55ca201256d01f62b45b2e1c21c12a064b1702d9298fee62dfeccc57d322a463ad55ca201256d01f62b45b2e1c21c10f867098504a817c809830334509435353535353535353535353535353535353535358202d98025a052f8f61201b2b11a78d6e866abc9c3db2ae8631fa656bfe5cb53668255367afba052f8f61201b2b11a78d6e866abc9c3db2ae8631fa656bfe5cb53668255367afbf901fcf901f9a00000000000000000000000000000000000000000000000000000000000000000a00000000000000000000000000000000000000000000000000000000000000000940000000000000000000000000000000000000000a00000000000000000000000000000000000000000000000000000000000000000a00000000000000000000000000000000000000000000000000000000000000000a00000000000000000000000000000000000000000000000000000000000000000b90100000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000008208ae820d0582115c8215b3821a0a827788a00000000000000000000000000000000000000000000000000000000000000000880000000000000000"),
		},
		{
			GetNodeDataPacket66{1111, GetNodeDataPacket(hashes)},
			common.FromHex("f847820457f842a000000000000000000000000000000000000000000000000000000000deadc0dea000000000000000000000000000000000000000000000000000000000feedbeef"),
		},
		{
			NodeDataPacket66{1111, NodeDataPacket(byteSlices)},
			common.FromHex("ce820457ca84deadc0de84feedbeef"),
		},
		{
			GetReceiptsPacket66{1111, GetReceiptsPacket(hashes)},
			common.FromHex("f847820457f842a000000000000000000000000000000000000000000000000000000000deadc0dea000000000000000000000000000000000000000000000000000000000feedbeef"),
		},
		{
			ReceiptsPacket66{1111, ReceiptsPacket([][]*types.Receipt{receipts})},
			common.FromHex("f90172820457f9016cf90169f901668001b9010000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000f85ff85d940000000000000000000000000000000000000011f842a0000000000000000000000000000000000000000000000000000000000000deada0000000000000000000000000000000000000000000000000000000000000beef830100ff"),
		},
		{
			ReceiptsRLPPacket66{1111, ReceiptsRLPPacket([]rlp.RawValue{receiptsRlp})},
			common.FromHex("f90172820457f9016cf90169f901668001b9010000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000f85ff85d940000000000000000000000000000000000000011f842a0000000000000000000000000000000000000000000000000000000000000deada0000000000000000000000000000000000000000000000000000000000000beef830100ff"),
		},
		{
			GetPooledTransactionsPacket66{1111, GetPooledTransactionsPacket(hashes)},
			common.FromHex("f847820457f842a000000000000000000000000000000000000000000000000000000000deadc0dea000000000000000000000000000000000000000000000000000000000feedbeef"),
		},
		{
			PooledTransactionsPacket66{1111, PooledTransactionsPacket(txs)},
			common.FromHex("f8d7820457f8d2f867088504a817c8088302e2489435353535353535353535353535353535353535358202008025a064b1702d9298fee62dfeccc57d322a463ad55ca201256d01f62b45b2e1c21c12a064b1702d9298fee62dfeccc57d322a463ad55ca201256d01f62b45b2e1c21c10f867098504a817c809830334509435353535353535353535353535353535353535358202d98025a052f8f61201b2b11a78d6e866abc9c3db2ae8631fa656bfe5cb53668255367afba052f8f61201b2b11a78d6e866abc9c3db2ae8631fa656bfe5cb53668255367afb"),
		},
		{
			PooledTransactionsRLPPacket66{1111, PooledTransactionsRLPPacket(txRlps)},
			common.FromHex("f8d7820457f8d2f867088504a817c8088302e2489435353535353535353535353535353535353535358202008025a064b1702d9298fee62dfeccc57d322a463ad55ca201256d01f62b45b2e1c21c12a064b1702d9298fee62dfeccc57d322a463ad55ca201256d01f62b45b2e1c21c10f867098504a817c809830334509435353535353535353535353535353535353535358202d98025a052f8f61201b2b11a78d6e866abc9c3db2ae8631fa656bfe5cb53668255367afba052f8f61201b2b11a78d6e866abc9c3db2ae8631fa656bfe5cb53668255367afb"),
		},
	} {
		if have, _ := rlp.EncodeToBytes(tc.message); !bytes.Equal(have, tc.want) {
			t.Errorf("test %d, type %T, have\n\t%x\nwant\n\t%x", i, tc.message, have, tc.want)
		}
	}
}
