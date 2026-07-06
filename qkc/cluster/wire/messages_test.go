// Copyright 2026-2027, QuarkChain.

package wire

import (
	"bytes"
	"math/big"
	"reflect"
	"testing"

	"github.com/ethereum/go-ethereum/qkc/serialize"
)

// =============================================================================
// §1 Custom wire types
// =============================================================================

func TestPrependedSizeBytes4_RoundTrip(t *testing.T) {
	cases := [][]byte{
		nil,
		{},
		{0x00},
		{0xAA, 0xBB},
		bytes.Repeat([]byte{0xFF}, 100),
	}
	for i, data := range cases {
		t.Run("", func(t *testing.T) {
			p := PrependedSizeBytes4(data)
			var buf []byte
			if err := p.Serialize(&buf); err != nil {
				t.Fatalf("case %d: Serialize: %v", i, err)
			}

			bb := serialize.NewByteBuffer(buf)
			var got PrependedSizeBytes4
			if err := got.Deserialize(bb); err != nil {
				t.Fatalf("case %d: Deserialize: %v", i, err)
			}

			if !bytes.Equal(got, data) {
				t.Errorf("case %d: mismatch\n  got  %x\n  want %x", i, got, data)
			}
		})
	}
}

func TestPrependedSizeBytes4_WireFormat(t *testing.T) {
	wantHex := "00000002aabb"

	p := PrependedSizeBytes4{0xAA, 0xBB}
	var buf []byte
	if err := p.Serialize(&buf); err != nil {
		t.Fatalf("Serialize: %v", err)
	}

	gotHex := hexEncode(buf)
	if gotHex != wantHex {
		t.Errorf("wire format mismatch:\n  got  %s\n  want %s", gotHex, wantHex)
	}

	bb := serialize.NewByteBuffer(buf)
	var got PrependedSizeBytes4
	if err := got.Deserialize(bb); err != nil {
		t.Fatalf("Deserialize: %v", err)
	}
	if !bytes.Equal(got, p) {
		t.Errorf("round-trip mismatch")
	}
}

func TestPrependedSizeBytes4_Deserialize_InvalidLength(t *testing.T) {
	buf := []byte{0xFF, 0xFF, 0xFF, 0xFF}
	bb := serialize.NewByteBuffer(buf)

	var p PrependedSizeBytes4
	err := p.Deserialize(bb)
	if err == nil {
		t.Error("expected error for length exceeding remaining buffer")
	}
}

func TestPrependedSizeHashList4_RoundTrip(t *testing.T) {
	cases := [][][HashLength]byte{
		nil,
		{},
		{makeHash(1)},
		{makeHash(1), makeHash(2)},
	}
	for i, hashes := range cases {
		t.Run("", func(t *testing.T) {
			p := PrependedSizeHashList4(hashes)
			var buf []byte
			if err := p.Serialize(&buf); err != nil {
				t.Fatalf("case %d: Serialize: %v", i, err)
			}

			bb := serialize.NewByteBuffer(buf)
			var got PrependedSizeHashList4
			if err := got.Deserialize(bb); err != nil {
				t.Fatalf("case %d: Deserialize: %v", i, err)
			}

			if len(got) != len(hashes) {
				t.Fatalf("case %d: length mismatch: got %d, want %d", i, len(got), len(hashes))
			}
			for j := range got {
				if got[j] != hashes[j] {
					t.Errorf("case %d: hash[%d] mismatch", i, j)
				}
			}
		})
	}
}

func TestPrependedSizeHashList4_WireFormat(t *testing.T) {
	p := PrependedSizeHashList4{makeHash(0x11), makeHash(0x22)}
	var buf []byte
	if err := p.Serialize(&buf); err != nil {
		t.Fatalf("Serialize: %v", err)
	}

	if len(buf) != 4+2*HashLength {
		t.Errorf("wire length mismatch: got %d, want %d", len(buf), 4+2*HashLength)
	}

	// count prefix is already checked by Deserialize; this is supplementary.
	bb := serialize.NewByteBuffer(buf)
	var got PrependedSizeHashList4
	if err := got.Deserialize(bb); err != nil {
		t.Fatalf("Deserialize: %v", err)
	}
	if len(got) != 2 {
		t.Errorf("round-trip length mismatch")
	}
}

func TestPrependedSizeHashList4_Deserialize_InvalidLength(t *testing.T) {
	buf := []byte{0xFF, 0xFF, 0xFF, 0xFF}
	bb := serialize.NewByteBuffer(buf)

	var p PrependedSizeHashList4
	err := p.Deserialize(bb)
	if err == nil {
		t.Error("expected error for count exceeding buffer capacity")
	}
}

// =============================================================================
// §2 Message struct round-trips
// =============================================================================
//
// Only structs with *RawBytes as the LAST field (or no RawBytes at all) can
// safely round-trip.  Structs with non-last RawBytes are only verified via the
// factory completeness test (§6) — they serialize correctly but cannot be
// deserialized until the real Go type replaces RawBytes.

func TestMessageRoundTrip(t *testing.T) {
	toAddr := makeAddress(0x00010001, 2)

	tests := []struct {
		name string
		msg  any
	}{
		// --- no RawBytes ---
		{"PingRequest_without_root_tip", PingRequest{
			ID:              []byte("slave1"),
			FullShardIDList: []uint32{0x00010001, 0x00020002},
			RootTip:         nil,
		}},
		{"PongResponse", PongResponse{
			ID:              []byte("master"),
			FullShardIDList: []uint32{0x00010001},
		}},
		{"SlaveInfo", SlaveInfo{
			ID:              []byte("s1"),
			Host:            []byte("127.0.0.1"),
			Port:            38391,
			FullShardIDList: []uint32{0x00010001, 0x00010002},
		}},
		{"ConnectToSlavesRequest", ConnectToSlavesRequest{
			SlaveInfoList: []SlaveInfo{
				{ID: []byte("s1"), Host: []byte("10.0.0.1"), Port: 38391, FullShardIDList: []uint32{0x00010001}},
				{ID: []byte("s2"), Host: []byte("10.0.0.2"), Port: 38392, FullShardIDList: []uint32{0x00020001}},
			},
		}},
		{"ConnectToSlavesResponse", ConnectToSlavesResponse{
			ResultList: []PrependedSizeBytes4{{0xAA, 0xBB}, {0xCC, 0xDD, 0xEE}},
		}},
		{"ArtificialTxConfig", ArtificialTxConfig{60, 10}},
		{"MineRequest", MineRequest{
			ArtificialTxConfig: ArtificialTxConfig{TargetRootBlockTime: 60, TargetMinorBlockTime: 10},
			Mining:             true,
		}},
		{"EcoInfo", EcoInfo{
			Branch:                           0x00010001,
			Height:                           12345,
			CoinbaseAmount:                   big.NewInt(1000),
			Difficulty:                       big.NewInt(1000000),
			UnconfirmedHeadersCoinbaseAmount: big.NewInt(500),
		}},
		{"GetEcoInfoListRequest", GetEcoInfoListRequest{}},
		{"GetEcoInfoListResponse", GetEcoInfoListResponse{
			ErrorCode: 0,
			EcoInfoList: []EcoInfo{
				{Branch: 0x00010001, Height: 1, CoinbaseAmount: big.NewInt(1), Difficulty: big.NewInt(1), UnconfirmedHeadersCoinbaseAmount: big.NewInt(1)},
			},
		}},
		{"GetNextBlockToMineRequest", GetNextBlockToMineRequest{
			Branch:             0x00010001,
			Address:            makeAddress(0x00010001, 1),
			ArtificialTxConfig: ArtificialTxConfig{TargetRootBlockTime: 60, TargetMinorBlockTime: 10},
		}},
		{"GetAccountDataRequest", GetAccountDataRequest{
			Address:     makeAddress(0x00010001, 1),
			BlockHeight: nil,
		}},
		{"GetLogRequest", GetLogRequest{
			Branch:    0x00010001,
			Addresses: [][AddressLength]byte{makeAddress(0x00010001, 1)},
			Topics: []PrependedSizeHashList4{
				{makeHash(0xAA)},
				{makeHash(0xBB), makeHash(0xCC)},
			},
			StartBlock: 100,
			EndBlock:   200,
		}},
		{"PingPongCommand", PingPongCommand{makeHash(42)}},
		{"PeerInfo", PeerInfo{IP: makeUint128(0x0102030405060708), Port: 38391}},
		{"TransactionDetail", TransactionDetail{
			TxHash:          makeHash(1),
			Nonce:           10,
			FromAddress:     makeAddress(0x00010001, 1),
			ToAddress:       &toAddr,
			Value:           big.NewInt(100),
			BlockHeight:     50,
			Timestamp:       1600000000,
			Success:         true,
			GasTokenID:      1,
			TransferTokenID: 1,
			IsFromRootChain: false,
		}},

		// --- RawBytes as last field (safe to round-trip) ---
		{"GenTxRequest", GenTxRequest{
			NumTxPerShard: 10,
			XShardPercent: 30,
			Tx:            &RawBytes{0x01, 0x02, 0x03},
		}},
		{"GetNextBlockToMineResponse", GetNextBlockToMineResponse{
			ErrorCode: 0,
			Block:     &RawBytes{0xAA, 0xBB},
		}},
		{"AddTransactionRequest", AddTransactionRequest{
			Tx: &RawBytes{0x01, 0x02},
		}},
		{"CheckMinorBlockRequest", CheckMinorBlockRequest{
			MinorBlockHeader: &RawBytes{0x01, 0x02},
		}},
		{"GetLogResponse", GetLogResponse{
			ErrorCode: 0,
			Logs:      []*RawBytes{{0x01, 0x02}}, // single element only — multi-element []*RawBytes cannot round-trip
		}},
		{"BatchAddXshardTxListRequest", BatchAddXshardTxListRequest{
			AddXshardTxListRequestList: []AddXshardTxListRequest{
				{Branch: 0x00010001, MinorBlockHash: makeHash(1), TxList: &RawBytes{0x01, 0x02}},
			},
		}},
		{"AddXshardTxListRequest", AddXshardTxListRequest{
			Branch:         0x00010001,
			MinorBlockHash: makeHash(1),
			TxList:         &RawBytes{0x01, 0x02},
		}},
		{"NewTransactionListCommand", NewTransactionListCommand{
			TransactionList: []*RawBytes{{0x01, 0x02}}, // single element only
		}},
		{"NewBlockMinorCommand", NewBlockMinorCommand{
			Block: &RawBytes{0x01, 0x02},
		}},
		{"NewRootBlockCommand", NewRootBlockCommand{
			Block: &RawBytes{0x01, 0x02},
		}},
		{"GetRootBlockListResponse", GetRootBlockListResponse{
			RootBlockList: []*RawBytes{{0x01, 0x02}}, // single element only
		}},
		{"GetMinorBlockListResponse", GetMinorBlockListResponse{
			MinorBlockList: []*RawBytes{{0x01, 0x02}}, // single element only
		}},
		{"GetUnconfirmedHeadersResponse", GetUnconfirmedHeadersResponse{
			ErrorCode: 0,
			HeadersInfoList: []HeadersInfo{
				{Branch: 0x00010001, HeaderList: []*RawBytes{{0x01, 0x02}}}, // single element only
			},
		}},
		{"PingRequest_with_root_tip", PingRequest{
			ID:              []byte("test"),
			FullShardIDList: []uint32{1, 2},
			RootTip:         &RawBytes{0xAA, 0xBB, 0xCC},
		}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var buf []byte
			if err := serialize.Serialize(&buf, tc.msg); err != nil {
				t.Fatalf("Serialize: %v", err)
			}

			bb := serialize.NewByteBuffer(buf)
			got := reflect.New(reflect.TypeOf(tc.msg)).Interface()
			if err := serialize.Deserialize(bb, got); err != nil {
				t.Fatalf("Deserialize: %v", err)
			}

			gotVal := reflect.ValueOf(got).Elem().Interface()

			// Re-serialize and compare bytes.  This sidesteps reflect.DeepEqual
			// pointer-identity problems with *RawBytes while still verifying
			// that the wire format is preserved through round-trip.
			var buf2 []byte
			if err := serialize.Serialize(&buf2, gotVal); err != nil {
				t.Fatalf("Re-serialize: %v", err)
			}
			if !bytes.Equal(buf, buf2) {
				t.Errorf("round-trip mismatch\n  got  %x\n  want %x", buf2, buf)
			}
		})
	}
}

// =============================================================================
// §3 ser:"nil" behavior
// =============================================================================

func TestOptionalMarker_PresentAndAbsent(t *testing.T) {
	absent := PingRequest{ID: []byte("x"), RootTip: nil}
	var buf []byte
	if err := serialize.Serialize(&buf, &absent); err != nil {
		t.Fatalf("Serialize absent: %v", err)
	}
	if buf[len(buf)-1] != 0x00 {
		t.Errorf("absent optional should end with 0x00, got %x", buf[len(buf)-1])
	}

	present := PingRequest{ID: []byte("x"), RootTip: &RawBytes{0xAA}}
	buf = nil
	if err := serialize.Serialize(&buf, &present); err != nil {
		t.Fatalf("Serialize present: %v", err)
	}
	if buf[len(buf)-2] != 0x01 || buf[len(buf)-1] != 0xAA {
		t.Errorf("present optional should write marker 0x01 then data, got %x", buf[len(buf)-2:])
	}
}

func TestNonOptionalRawBytes_NoMarker(t *testing.T) {
	req := AddRootBlockRequest{RootBlock: &RawBytes{0xAA}, ExpectSwitch: true}
	var buf []byte
	if err := serialize.Serialize(&buf, &req); err != nil {
		t.Fatalf("Serialize: %v", err)
	}
	// RawBytes now uses 4-byte length prefix, so first bytes should be 00000001 (length=1)
	// followed by AA (actual data), then 01 (ExpectSwitch=true)
	want := []byte{0x00, 0x00, 0x00, 0x01, 0xAA, 0x01}
	if len(buf) < len(want) {
		t.Fatalf("buffer too short: got %d bytes, want at least %d", len(buf), len(want))
	}
	for i, b := range want {
		if buf[i] != b {
			t.Errorf("byte %d: got %x, want %x\nfull buf: %x", i, buf[i], b, buf)
			break
		}
	}
}

// =============================================================================
// §4 Wire compatibility (hand-computed vectors)
// =============================================================================
//
// These test vectors are hand-computed from the Python FIELDS definitions to
// verify that Go serialization produces identical bytes. They are NOT produced
// by running pyquarkchain directly.

func TestWireCompat_PingRequest(t *testing.T) {
	wantHex := "0000000474657374000000020000000100000002" + "00"

	ping := PingRequest{
		ID:              []byte("test"),
		FullShardIDList: []uint32{1, 2},
		RootTip:         nil,
	}
	var buf []byte
	if err := serialize.Serialize(&buf, &ping); err != nil {
		t.Fatalf("Serialize: %v", err)
	}
	gotHex := hexEncode(buf)
	if gotHex != wantHex {
		t.Errorf("wire mismatch:\n  got  %s\n  want %s", gotHex, wantHex)
	}
}

func TestWireCompat_PongResponse(t *testing.T) {
	wantHex := "000000026f6b0000000100000003"

	pong := PongResponse{
		ID:              []byte("ok"),
		FullShardIDList: []uint32{3},
	}
	var buf []byte
	if err := serialize.Serialize(&buf, &pong); err != nil {
		t.Fatalf("Serialize: %v", err)
	}
	gotHex := hexEncode(buf)
	if gotHex != wantHex {
		t.Errorf("wire mismatch:\n  got  %s\n  want %s", gotHex, wantHex)
	}
}

func TestWireCompat_SlaveInfo(t *testing.T) {
	wantHex := "000000027331" +
		"000000096c6f63616c686f7374" +
		"95f7" +
		"0000000100010001"

	slave := SlaveInfo{
		ID:              []byte("s1"),
		Host:            []byte("localhost"),
		Port:            38391,
		FullShardIDList: []uint32{0x00010001},
	}
	var buf []byte
	if err := serialize.Serialize(&buf, &slave); err != nil {
		t.Fatalf("Serialize: %v", err)
	}
	gotHex := hexEncode(buf)
	if gotHex != wantHex {
		t.Errorf("wire mismatch:\n  got  %s\n  want %s", gotHex, wantHex)
	}
}

// =============================================================================
// §5 Field order verification
// =============================================================================

func TestFieldOrder(t *testing.T) {
	cases := []struct {
		name     string
		typ      reflect.Type
		expected []string
	}{
		{"PingRequest", reflect.TypeFor[PingRequest](), []string{"ID", "FullShardIDList", "RootTip"}},
		{"PongResponse", reflect.TypeFor[PongResponse](), []string{"ID", "FullShardIDList"}},
		{"SlaveInfo", reflect.TypeFor[SlaveInfo](), []string{"ID", "Host", "Port", "FullShardIDList"}},
		{"EcoInfo", reflect.TypeFor[EcoInfo](), []string{"Branch", "Height", "CoinbaseAmount", "Difficulty", "UnconfirmedHeadersCoinbaseAmount"}},
		{"ShardStats", reflect.TypeFor[ShardStats](), []string{"Branch", "Height", "Difficulty", "CoinbaseAddress", "Timestamp",
			"TxCount60s", "PendingTxCount", "TotalTxCount", "BlockCount60s", "StaleBlockCount60s", "LastBlockTime"}},
		{"TransactionDetail", reflect.TypeFor[TransactionDetail](), []string{"TxHash", "Nonce", "FromAddress", "ToAddress", "Value",
			"BlockHeight", "Timestamp", "Success", "GasTokenID", "TransferTokenID", "IsFromRootChain"}},
		{"HelloCommand", reflect.TypeFor[HelloCommand](), []string{"Version", "NetworkID", "PeerID", "PeerIP", "PeerPort",
			"ChainMaskList", "RootBlockHeader", "GenesisRootBlockHash"}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.typ.NumField() != len(tc.expected) {
				t.Fatalf("field count mismatch: got %d, want %d", tc.typ.NumField(), len(tc.expected))
			}
			for i, want := range tc.expected {
				if tc.typ.Field(i).Name != want {
					t.Errorf("field %d: got %s, want %s", i, tc.typ.Field(i).Name, want)
				}
			}
		})
	}
}

// =============================================================================
// §6 Factory completeness
// =============================================================================

func TestNewClusterMessage_Completeness(t *testing.T) {
	ops := []ClusterOp{
		ClusterOpPing,
		ClusterOpPong,
		ClusterOpConnectToSlavesRequest,
		ClusterOpConnectToSlavesResponse,
		ClusterOpAddRootBlockRequest,
		ClusterOpAddRootBlockResponse,
		ClusterOpGetEcoInfoListRequest,
		ClusterOpGetEcoInfoListResponse,
		ClusterOpGetNextBlockToMineRequest,
		ClusterOpGetNextBlockToMineResponse,
		ClusterOpGetUnconfirmedHeadersRequest,
		ClusterOpGetUnconfirmedHeadersResponse,
		ClusterOpGetAccountDataRequest,
		ClusterOpGetAccountDataResponse,
		ClusterOpAddTransactionRequest,
		ClusterOpAddTransactionResponse,
		ClusterOpAddMinorBlockHeaderRequest,
		ClusterOpAddMinorBlockHeaderResponse,
		ClusterOpAddXshardTxListRequest,
		ClusterOpAddXshardTxListResponse,
		ClusterOpSyncMinorBlockListRequest,
		ClusterOpSyncMinorBlockListResponse,
		ClusterOpAddMinorBlockRequest,
		ClusterOpAddMinorBlockResponse,
		ClusterOpCreateClusterPeerConnectionRequest,
		ClusterOpCreateClusterPeerConnectionResponse,
		ClusterOpDestroyClusterPeerConnectionCommand,
		ClusterOpGetMinorBlockRequest,
		ClusterOpGetMinorBlockResponse,
		ClusterOpGetTransactionRequest,
		ClusterOpGetTransactionResponse,
		ClusterOpBatchAddXshardTxListRequest,
		ClusterOpBatchAddXshardTxListResponse,
		ClusterOpExecuteTransactionRequest,
		ClusterOpExecuteTransactionResponse,
		ClusterOpGetTransactionReceiptRequest,
		ClusterOpGetTransactionReceiptResponse,
		ClusterOpMineRequest,
		ClusterOpMineResponse,
		ClusterOpGenTxRequest,
		ClusterOpGenTxResponse,
		ClusterOpGetTransactionListByAddressRequest,
		ClusterOpGetTransactionListByAddressResponse,
		ClusterOpGetLogRequest,
		ClusterOpGetLogResponse,
		ClusterOpEstimateGasRequest,
		ClusterOpEstimateGasResponse,
		ClusterOpGetStorageRequest,
		ClusterOpGetStorageResponse,
		ClusterOpGetCodeRequest,
		ClusterOpGetCodeResponse,
		ClusterOpGasPriceRequest,
		ClusterOpGasPriceResponse,
		ClusterOpGetWorkRequest,
		ClusterOpGetWorkResponse,
		ClusterOpSubmitWorkRequest,
		ClusterOpSubmitWorkResponse,
		ClusterOpAddMinorBlockHeaderListRequest,
		ClusterOpAddMinorBlockHeaderListResponse,
		ClusterOpCheckMinorBlockRequest,
		ClusterOpCheckMinorBlockResponse,
		ClusterOpGetAllTransactionsRequest,
		ClusterOpGetAllTransactionsResponse,
		ClusterOpGetRootChainStakesRequest,
		ClusterOpGetRootChainStakesResponse,
		ClusterOpGetTotalBalanceRequest,
		ClusterOpGetTotalBalanceResponse,
	}
	for _, op := range ops {
		t.Run("ClusterOp(0x"+string("0123456789ABCDEF"[byte(op)>>4])+string("0123456789ABCDEF"[byte(op)&0x0F])+")", func(t *testing.T) {
			msg, err := NewClusterMessage(op)
			if err != nil {
				t.Fatalf("NewClusterMessage(0x%x): %v", op, err)
			}
			if msg == nil {
				t.Fatalf("NewClusterMessage(0x%x) returned nil", op)
			}
			typ := reflect.TypeOf(msg)
			if typ.Kind() != reflect.Pointer || typ.Elem().Kind() != reflect.Struct {
				t.Errorf("expected *struct, got %T", msg)
			}
		})
	}
}

func TestNewCommandMessage_Completeness(t *testing.T) {
	ops := []CommandOp{
		CommandOpHello,
		CommandOpNewMinorBlockHeaderList,
		CommandOpNewTransactionList,
		CommandOpGetPeerListRequest,
		CommandOpGetPeerListResponse,
		CommandOpGetRootBlockHeaderListRequest,
		CommandOpGetRootBlockHeaderListResponse,
		CommandOpGetRootBlockListRequest,
		CommandOpGetRootBlockListResponse,
		CommandOpGetMinorBlockListRequest,
		CommandOpGetMinorBlockListResponse,
		CommandOpGetMinorBlockHeaderListRequest,
		CommandOpGetMinorBlockHeaderListResponse,
		CommandOpNewBlockMinor,
		CommandOpPing,
		CommandOpPong,
		CommandOpGetRootBlockHeaderListWithSkipRequest,
		CommandOpGetRootBlockHeaderListWithSkipResponse,
		CommandOpNewRootBlock,
		CommandOpGetMinorBlockHeaderListWithSkipRequest,
		CommandOpGetMinorBlockHeaderListWithSkipResponse,
	}
	for _, op := range ops {
		t.Run("", func(t *testing.T) {
			msg, err := NewCommandMessage(op)
			if err != nil {
				t.Fatalf("NewCommandMessage(0x%x): %v", op, err)
			}
			if msg == nil {
				t.Fatalf("NewCommandMessage(0x%x) returned nil", op)
			}
		})
	}
}

func TestNewClusterMessage_UnknownOpcode(t *testing.T) {
	_, err := NewClusterMessage(ClusterOp(0xFF))
	if err == nil {
		t.Error("expected error for unknown ClusterOp")
	}
}

func TestNewCommandMessage_UnknownOpcode(t *testing.T) {
	_, err := NewCommandMessage(CommandOp(0xEE))
	if err == nil {
		t.Error("expected error for unknown CommandOp")
	}
}

// =============================================================================
// Helpers
// =============================================================================

func makeAddress(fullShardID uint32, recipient byte) [AddressLength]byte {
	var addr [AddressLength]byte
	addr[16] = byte(fullShardID >> 24)
	addr[17] = byte(fullShardID >> 16)
	addr[18] = byte(fullShardID >> 8)
	addr[19] = byte(fullShardID)
	addr[0] = recipient
	return addr
}

func makeHash(seed byte) [HashLength]byte {
	var h [HashLength]byte
	for i := range h {
		h[i] = seed + byte(i)
	}
	return h
}

func makeUint128(seed uint64) [UInt128Length]byte {
	var u [UInt128Length]byte
	for i := range 8 {
		u[i] = byte(seed >> (56 - 8*i))
	}
	return u
}

func hexEncode(b []byte) string {
	const hexChars = "0123456789abcdef"
	s := make([]byte, len(b)*2)
	for i, v := range b {
		s[i*2] = hexChars[v>>4]
		s[i*2+1] = hexChars[v&0x0f]
	}
	return string(s)
}
