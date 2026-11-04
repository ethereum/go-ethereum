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

package wire

import (
	"bytes"
	"math/big"
	"reflect"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/params"
	qkctypes "github.com/ethereum/go-ethereum/qkc/types"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/holiman/uint256"
)

// ---- opcode → message registry ----

func TestNewClusterMessageRegistry(t *testing.T) {
	cases := []struct {
		op   ClusterOp
		want interface{}
	}{
		{ClusterOpPing, &PingRequest{}},
		{ClusterOpPong, &PongResponse{}},
		{ClusterOpConnectToSlavesRequest, &ConnectToSlavesRequest{}},
		{ClusterOpConnectToSlavesResponse, &ConnectToSlavesResponse{}},
		{ClusterOpAddRootBlockRequest, &AddRootBlockRequest{}},
		{ClusterOpAddRootBlockResponse, &AddRootBlockResponse{}},
		{ClusterOpAddMinorBlockHeaderRequest, &AddMinorBlockHeaderRequest{}},
		{ClusterOpAddMinorBlockHeaderResponse, &AddMinorBlockHeaderResponse{}},
		{ClusterOpAddXshardTxListRequest, &AddXshardTxListRequest{}},
		{ClusterOpAddXshardTxListResponse, &AddXshardTxListResponse{}},
		{ClusterOpGetRootChainStakesRequest, &GetRootChainStakesRequest{}},
		{ClusterOpGetRootChainStakesResponse, &GetRootChainStakesResponse{}},
	}
	for _, c := range cases {
		got, err := NewClusterMessage(c.op)
		if err != nil {
			t.Fatalf("NewClusterMessage(0x%02x): %v", byte(c.op), err)
		}
		if reflect.TypeOf(got) != reflect.TypeOf(c.want) {
			t.Fatalf("NewClusterMessage(0x%02x) = %T, want %T", byte(c.op), got, c.want)
		}
	}

	if _, err := NewClusterMessage(ClusterOp(0x99)); err == nil {
		t.Fatal("NewClusterMessage: unknown opcode must be rejected")
	}
}

func TestNewCommandMessageRegistry(t *testing.T) {
	msg, err := NewCommandMessage(CommandOpHello)
	if err != nil {
		t.Fatalf("NewCommandMessage(Hello): %v", err)
	}
	if _, ok := msg.(*HelloCommand); !ok {
		t.Fatalf("NewCommandMessage(Hello) = %T, want *HelloCommand", msg)
	}
}

// ---- SSZ round trips ----

// roundTripSSZ marshals msg, unmarshals into a fresh value of the same type
// via the registry-independent path, and re-marshals: canonical encoding
// must reproduce the input bytes exactly. This avoids false mismatches from
// nil vs empty slices on dynamic fields.
func roundTripSSZ(t *testing.T, msg SSZMessage) {
	t.Helper()
	enc, err := msg.MarshalSSZ()
	if err != nil {
		t.Fatalf("MarshalSSZ: %v", err)
	}
	if size := msg.SizeSSZ(); size != len(enc) {
		t.Fatalf("SizeSSZ() = %d, but MarshalSSZ produced %d bytes", size, len(enc))
	}

	fresh := reflect.New(reflect.TypeOf(msg).Elem()).Interface().(SSZMessage)
	if err := fresh.UnmarshalSSZ(enc); err != nil {
		t.Fatalf("UnmarshalSSZ: %v", err)
	}
	dec, err := fresh.MarshalSSZ()
	if err != nil {
		t.Fatalf("re-MarshalSSZ: %v", err)
	}
	if !bytes.Equal(enc, dec) {
		t.Fatalf("round trip mismatch:\n  got  %x\n  want %x", dec, enc)
	}
}

func TestChainHeaderMessagesSSZ(t *testing.T) {
	withdrawals := common.HexToHash("0x11")
	header := &types.Header{
		ParentHash:      common.HexToHash("0x12"),
		Coinbase:        common.HexToAddress("0x13"),
		Difficulty:      big.NewInt(0),
		Number:          big.NewInt(42),
		GasLimit:        1000000,
		GasUsed:         21000,
		Time:            12345,
		Extra:           (&types.QKCExtra{ShardID: 1}).Encode(),
		MixDigest:       common.HexToHash("0x0304"),
		BaseFee:         big.NewInt(100),
		WithdrawalsHash: &withdrawals,
	}
	header.Bloom[0] = 0x25
	headerData, err := rlp.EncodeToBytes(header)
	if err != nil {
		t.Fatal(err)
	}
	msg := &AddMinorBlockHeaderRequest{
		MinorBlockHeader: Data(headerData),
	}
	if _, err := uint256.NewInt(0x0102).MarshalSSZAppend(msg.CoinbaseAmount[:0]); err != nil {
		t.Fatal(err)
	}
	encoded, err := msg.MarshalSSZ()
	if err != nil {
		t.Fatal(err)
	}
	if len(encoded) != msg.SizeSSZ() {
		t.Fatalf("header message size: got %d, want %d", len(encoded), msg.SizeSSZ())
	}
	// Pin the numeric byte order on the wire: reward 0x0102 must be
	// little-endian, unlike the header's MixDigest bytes.
	if want := append([]byte{0x02, 0x01}, make([]byte, 30)...); !bytes.Equal(encoded[4:36], want) {
		t.Fatalf("reward is not little-endian uint256: %x", encoded[4:36])
	}
	var decoded AddMinorBlockHeaderRequest
	if err := decoded.UnmarshalSSZ(encoded); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(decoded.MinorBlockHeader, headerData) {
		t.Fatal("SSZ changed the encoded header")
	}
	if decoded.CoinbaseAmount != msg.CoinbaseAmount {
		t.Fatal("SSZ changed the reward")
	}
	var decodedHeader types.Header
	if err := rlp.DecodeBytes(decoded.MinorBlockHeader, &decodedHeader); err != nil {
		t.Fatal(err)
	}
	if decodedHeader.Hash() != header.Hash() {
		t.Fatalf("header hash changed in round trip: got %s, want %s", decodedHeader.Hash(), header.Hash())
	}
	if _, err := msg.HashTreeRoot(); err != nil {
		t.Fatalf("header hash tree root: %v", err)
	}
}

func TestCrossChainDepositsSSZ(t *testing.T) {
	to := common.HexToAddress("0x1234")
	deposit := &types.DepositTx{
		Version:         types.DepositTxVersion,
		SourceBlockHash: common.HexToHash("0x56"),
		SourceLogIndex:  9,
		From:            common.HexToAddress("0x78"),
		To:              &to,
		Value:           uint256.NewInt(1000),
		GasPrice:        uint256.NewInt(2),
		Gas:             params.XShardSendGas,
		IsMove:          true,
		Data:            []byte{},
	}
	depositData, err := rlp.EncodeToBytes(deposit)
	if err != nil {
		t.Fatal(err)
	}
	msg := &AddXshardTxListRequest{
		ChainID:        2,
		MinorBlockHash: deposit.SourceBlockHash,
		DepositTxList:  []Data{Data(depositData)},
	}
	encoded, err := msg.MarshalSSZ()
	if err != nil {
		t.Fatal(err)
	}
	if len(encoded) != msg.SizeSSZ() {
		t.Fatalf("deposit message size: got %d, want %d", len(encoded), msg.SizeSSZ())
	}
	var decoded AddXshardTxListRequest
	if err := decoded.UnmarshalSSZ(encoded); err != nil {
		t.Fatal(err)
	}
	if len(decoded.DepositTxList) != 1 || !bytes.Equal(decoded.DepositTxList[0], depositData) {
		t.Fatal("SSZ changed the encoded deposit")
	}
	var decodedDeposit types.DepositTx
	if err := rlp.DecodeBytes(decoded.DepositTxList[0], &decodedDeposit); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(&decodedDeposit, deposit) {
		t.Fatalf("deposit changed in round trip: got %#v, want %#v", &decodedDeposit, deposit)
	}
	if decodedDeposit.SourceBlockHash != decoded.MinorBlockHash {
		t.Fatal("deposit source does not match the enclosing list")
	}
	if _, err := msg.HashTreeRoot(); err != nil {
		t.Fatalf("deposit hash tree root: %v", err)
	}

	empty := &AddXshardTxListRequest{ChainID: 2, MinorBlockHash: msg.MinorBlockHash}
	var decodedEmpty AddXshardTxListRequest
	encoded, err = empty.MarshalSSZ()
	if err != nil {
		t.Fatal(err)
	}
	if err := decodedEmpty.UnmarshalSSZ(encoded); err != nil {
		t.Fatal(err)
	}
	if decodedEmpty.DepositTxList == nil || len(decodedEmpty.DepositTxList) != 0 {
		t.Fatal("empty deposit list must decode as a present empty list")
	}
}

func TestMasterMessagesSSZ(t *testing.T) {
	root := &AddRootBlockRequest{RootBlock: Data{0x01, 0x02}}
	roundTripSSZ(t, root)

	request := &GetRootChainStakesRequest{
		Address:        common.HexToAddress("0x1234"),
		MinorBlockHash: common.HexToHash("0x5678"),
	}
	roundTripSSZ(t, request)

	response := &GetRootChainStakesResponse{
		ErrorCode: 1,
		Signer:    common.HexToAddress("0x9abc"),
	}
	if _, err := uint256.NewInt(0x0102).MarshalSSZAppend(response.Stakes[:0]); err != nil {
		t.Fatal(err)
	}
	roundTripSSZ(t, response)
}

func TestRootBlockPayloadsSSZ(t *testing.T) {
	header := &qkctypes.RootBlockHeader{Version: 1, Number: 2, Extra: []byte("root")}
	headerData, err := header.MarshalSSZ()
	if err != nil {
		t.Fatal(err)
	}
	hello := &HelloCommand{RootBlockHeader: Data(headerData)}
	var decodedHello HelloCommand
	if err := decodedHello.UnmarshalSSZ(mustMarshalSSZ(t, hello)); err != nil {
		t.Fatal(err)
	}
	var decodedHeader qkctypes.RootBlockHeader
	if err := decodedHeader.UnmarshalSSZ(decodedHello.RootBlockHeader); err != nil {
		t.Fatal(err)
	}
	if decodedHeader.Hash() != header.Hash() {
		t.Fatal("root block header changed in SSZ message")
	}

	block := qkctypes.NewRootBlock(header, nil, nil)
	blockData, err := block.MarshalSSZ()
	if err != nil {
		t.Fatal(err)
	}
	ping := &PingRequest{HasRootTip: true, RootTip: Data(blockData)}
	var decodedPing PingRequest
	if err := decodedPing.UnmarshalSSZ(mustMarshalSSZ(t, ping)); err != nil {
		t.Fatal(err)
	}
	var decodedBlock qkctypes.RootBlock
	if err := decodedBlock.UnmarshalSSZ(decodedPing.RootTip); err != nil {
		t.Fatal(err)
	}
	if decodedBlock.Hash() != block.Hash() {
		t.Fatal("root block changed in SSZ message")
	}
}

func TestPingPongSSZRoundTrip(t *testing.T) {
	ping := &PingRequest{
		ChainID:    0x10002,
		HasRootTip: true,
		RootTip:    []byte{0xaa, 0xbb},
	}
	roundTripSSZ(t, ping)

	// Explicit deep-equality on the non-empty case.
	var got PingRequest
	if err := got.UnmarshalSSZ(mustMarshalSSZ(t, ping)); err != nil {
		t.Fatalf("UnmarshalSSZ: %v", err)
	}
	if !reflect.DeepEqual(&got, ping) {
		t.Fatalf("deep mismatch: got %+v, want %+v", &got, ping)
	}

	pong := &PongResponse{ChainID: 2}
	roundTripSSZ(t, pong)

	// Boundary: all dynamic fields empty.
	roundTripSSZ(t, &PingRequest{})
	roundTripSSZ(t, &PongResponse{})
}

func TestChainToChainPingCarriesChainIDWithoutRootTip(t *testing.T) {
	for _, chainID := range []uint32{0, 1} {
		ping := &PingRequest{ChainID: chainID, HasRootTip: false}
		encoded := mustMarshalSSZ(t, ping)

		var got PingRequest
		if err := got.UnmarshalSSZ(encoded); err != nil {
			t.Fatalf("UnmarshalSSZ for chain ID %d: %v", chainID, err)
		}
		if got.ChainID != chainID || got.HasRootTip || len(got.RootTip) != 0 {
			t.Fatalf("decoded ping = %+v, want ChainID=%d and no root tip", got, chainID)
		}
	}
}

func TestConnectToSlavesSSZRoundTrip(t *testing.T) {
	req := &ConnectToSlavesRequest{
		SlaveInfoList: []SlaveInfo{
			{Host: []byte("10.0.0.1"), Port: 38391, ChainID: 1},
			{Host: []byte("10.0.0.2"), Port: 38392, ChainID: 2},
		},
	}
	roundTripSSZ(t, req)

	// Results mirror the request order; empty bytes mean success.
	resp := &ConnectToSlavesResponse{
		ResultList: [][]byte{
			nil,                    // success
			[]byte("dial timeout"), // failure
		},
	}
	roundTripSSZ(t, resp)

	// Boundary: empty lists.
	roundTripSSZ(t, &ConnectToSlavesRequest{})
	roundTripSSZ(t, &ConnectToSlavesResponse{})
}

func TestHelloCommandSSZRoundTrip(t *testing.T) {
	msg := &HelloCommand{
		Version:              1,
		NetworkID:            2,
		PeerID:               common.Hash{1, 2, 3},
		PeerIP:               [UInt128Length]byte{127, 0, 0, 1},
		PeerPort:             38291,
		RootBlockHeader:      []byte{0xaa, 0xbb},
		GenesisRootBlockHash: common.Hash{4, 5, 6},
	}
	roundTripSSZ(t, msg)
}

func TestSSZRejectsInvalidOffset(t *testing.T) {
	// PingRequest's RootTip offset cannot point inside its 9-byte fixed section.
	bad := []byte{0x01, 0x00, 0x00, 0x00, 0x00, 0x08, 0x00, 0x00, 0x00}
	if err := new(PingRequest).UnmarshalSSZ(bad); err == nil {
		t.Fatal("UnmarshalSSZ accepted an offset inside the fixed section")
	}
}

func TestGeneratedCodecsRejectMalformedPayloads(t *testing.T) {
	cases := []struct {
		name     string
		offsetAt int
		new      func() SSZMessage
	}{
		{"PingRequest", 5, func() SSZMessage { return new(PingRequest) }},
		{"PongResponse", -1, func() SSZMessage { return new(PongResponse) }},
		{"SlaveInfo", 0, func() SSZMessage { return new(SlaveInfo) }},
		{"ConnectToSlavesRequest", 0, func() SSZMessage { return new(ConnectToSlavesRequest) }},
		{"ConnectToSlavesResponse", 0, func() SSZMessage { return new(ConnectToSlavesResponse) }},
		{"AddRootBlockRequest", 0, func() SSZMessage { return new(AddRootBlockRequest) }},
		{"AddRootBlockResponse", -1, func() SSZMessage { return new(AddRootBlockResponse) }},
		{"HelloCommand", 58, func() SSZMessage { return new(HelloCommand) }},
		{"AddMinorBlockHeaderRequest", 0, func() SSZMessage { return new(AddMinorBlockHeaderRequest) }},
		{"AddMinorBlockHeaderResponse", -1, func() SSZMessage { return new(AddMinorBlockHeaderResponse) }},
		{"AddXshardTxListRequest", 36, func() SSZMessage { return new(AddXshardTxListRequest) }},
		{"AddXshardTxListResponse", -1, func() SSZMessage { return new(AddXshardTxListResponse) }},
		{"GetRootChainStakesRequest", -1, func() SSZMessage { return new(GetRootChainStakesRequest) }},
		{"GetRootChainStakesResponse", -1, func() SSZMessage { return new(GetRootChainStakesResponse) }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			encoded, err := tc.new().MarshalSSZ()
			if err != nil {
				t.Fatalf("MarshalSSZ empty value: %v", err)
			}
			if len(encoded) < tc.offsetAt+4 {
				t.Fatalf("empty encoding too short for offset mutation: %x", encoded)
			}

			if tc.offsetAt >= 0 {
				badOffset := append([]byte(nil), encoded...)
				for i := 0; i < 4; i++ {
					badOffset[tc.offsetAt+i] = 0
				}
				if err := tc.new().UnmarshalSSZ(badOffset); err == nil {
					t.Fatal("UnmarshalSSZ accepted a zero first offset")
				}
			}

			if err := tc.new().UnmarshalSSZ(encoded[:len(encoded)-1]); err == nil {
				t.Fatal("UnmarshalSSZ accepted a truncated fixed section")
			}
		})
	}
}

func TestResultListGoldenEncoding(t *testing.T) {
	msg := &ConnectToSlavesResponse{ResultList: [][]byte{nil, []byte("x")}}
	got, err := msg.MarshalSSZ()
	if err != nil {
		t.Fatalf("MarshalSSZ: %v", err)
	}
	// The container points to ResultList at offset 4. ResultList's two
	// variable-size elements are located by offsets 8 and 8; the first element
	// is empty and the second contains "x".
	want := []byte{
		0x04, 0x00, 0x00, 0x00,
		0x08, 0x00, 0x00, 0x00,
		0x08, 0x00, 0x00, 0x00,
		0x78,
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("result list encoding mismatch:\n  got  %x\n  want %x", got, want)
	}
}

// ---- golden encoding ----

// TestSSZGoldenEncoding pins the encoding of a PingRequest with a fixed-width
// ChainID, HasRootTip=false, and an empty RootTip.
//
//	PingRequest{ChainID: 1, HasRootTip: false}
//	  ChainID  = 1 -> 01 00 00 00
//	  HasRootTip   -> 00
//	  offset(RootTip) = 9 -> 09 00 00 00
func TestSSZGoldenEncoding(t *testing.T) {
	ping := &PingRequest{ChainID: 1}
	got, err := ping.MarshalSSZ()
	if err != nil {
		t.Fatalf("MarshalSSZ: %v", err)
	}
	want := []byte{0x01, 0x00, 0x00, 0x00, 0x00, 0x09, 0x00, 0x00, 0x00}
	if !bytes.Equal(got, want) {
		t.Fatalf("golden mismatch:\n  got  %x\n  want %x", got, want)
	}

	// Empty RootTip: its offset points to the end of the fixed section.
	empty, err := (&PingRequest{}).MarshalSSZ()
	if err != nil {
		t.Fatalf("MarshalSSZ: %v", err)
	}
	wantEmpty := []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x09, 0x00, 0x00, 0x00}
	if !bytes.Equal(empty, wantEmpty) {
		t.Fatalf("empty golden mismatch:\n  got  %x\n  want %x", empty, wantEmpty)
	}
}

// ---- helpers ----

func mustMarshalSSZ(t *testing.T, msg SSZMessage) []byte {
	t.Helper()
	enc, err := msg.MarshalSSZ()
	if err != nil {
		t.Fatalf("MarshalSSZ: %v", err)
	}
	return enc
}
