// Copyright 2026-2027, QuarkChain.

package wire

import (
	"bytes"
	"encoding/hex"
	"math/big"
	"reflect"
	"testing"

	"github.com/ethereum/go-ethereum/qkc/account"
	"github.com/ethereum/go-ethereum/qkc/serialize"
)

// =============================================================================
// §1 Custom wire types
// =============================================================================

func TestPrependedSizeBytes4(t *testing.T) {
	cases := []struct {
		name    string
		data    PrependedSizeBytes4
		wantHex string
	}{
		{"empty", nil, "00000000"},
		{"two_bytes", PrependedSizeBytes4{0xAA, 0xBB}, "00000002aabb"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var buf []byte
			if err := tc.data.Serialize(&buf); err != nil {
				t.Fatalf("Serialize: %v", err)
			}
			if got := hex.EncodeToString(buf); got != tc.wantHex {
				t.Errorf("wire: got %s, want %s", got, tc.wantHex)
			}

			bb := serialize.NewByteBuffer(buf)
			var got PrependedSizeBytes4
			if err := got.Deserialize(bb); err != nil {
				t.Fatalf("Deserialize: %v", err)
			}
			if !bytes.Equal(got, tc.data) {
				t.Errorf("round-trip mismatch")
			}
		})
	}
}

func TestPrependedSizeHashList4_RoundTrip(t *testing.T) {
	cases := []PrependedSizeHashList4{
		nil,
		{makeHash(1)},
		{makeHash(1), makeHash(2)},
	}
	for _, hashes := range cases {
		var buf []byte
		if err := hashes.Serialize(&buf); err != nil {
			t.Fatalf("Serialize: %v", err)
		}
		bb := serialize.NewByteBuffer(buf)
		var got PrependedSizeHashList4
		if err := got.Deserialize(bb); err != nil {
			t.Fatalf("Deserialize: %v", err)
		}
		if len(got) != len(hashes) {
			t.Fatalf("length mismatch: got %d, want %d", len(got), len(hashes))
		}
		for j := range got {
			if got[j] != hashes[j] {
				t.Errorf("hash[%d] mismatch", j)
			}
		}
	}
}

// =============================================================================
// §2 Message round-trips (representative samples)
// =============================================================================

func TestMessageRoundTrip(t *testing.T) {
	tests := []struct {
		name string
		msg  any
	}{
		{"PingRequest_no_RawBytes", PingRequest{
			ID:              []byte("slave1"),
			FullShardIDList: []uint32{0x00010001, 0x00020002},
		}},
		{"GenTxRequest_RawBytes_last", GenTxRequest{
			NumTxPerShard: 10,
			XShardPercent: 30,
			Tx:            &RawBytes{0x01, 0x02, 0x03},
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

			var buf2 []byte
			if err := serialize.Serialize(&buf2, reflect.ValueOf(got).Elem().Interface()); err != nil {
				t.Fatalf("Re-serialize: %v", err)
			}
			if !bytes.Equal(buf, buf2) {
				t.Errorf("round-trip mismatch")
			}
		})
	}
}

// =============================================================================
// §3 Python/Go protocol compatibility
// =============================================================================
//
// Golden vectors are derived from pyquarkchain's FIELDS definitions.
// They verify Go serialization produces byte-identical output to Python.

// --- Primitives ---

func TestPythonCompat_Address(t *testing.T) {
	tests := []struct {
		name      string
		addr      account.Address
		pythonHex string
	}{
		{
			"simple",
			account.Address{
				Recipient:    account.Recipient{0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01},
				FullShardKey: 0x00010001,
			},
			"010101010101010101010101010101010101010100010001",
		},
		{
			"empty_recipient",
			account.Address{
				Recipient:    account.Recipient{},
				FullShardKey: 0x00010001,
			},
			"000000000000000000000000000000000000000000010001",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			goBytes, err := serialize.SerializeToBytes(&tc.addr)
			if err != nil {
				t.Fatalf("Serialize: %v", err)
			}
			assertPythonMatch(t, tc.pythonHex, goBytes)
		})
	}
}

func TestPythonCompat_OptionalAddress(t *testing.T) {
	tests := []struct {
		name      string
		addr      *account.Address
		pythonHex string
	}{
		{"none", nil, "00"},
		{
			"present",
			&account.Address{
				Recipient:    account.Recipient{0x02, 0x02, 0x02, 0x02, 0x02, 0x02, 0x02, 0x02, 0x02, 0x02, 0x02, 0x02, 0x02, 0x02, 0x02, 0x02, 0x02, 0x02, 0x02, 0x02},
				FullShardKey: 0x00010001,
			},
			"01020202020202020202020202020202020202020200010001",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var goBytes []byte
			if tc.addr == nil {
				goBytes = []byte{0x00}
			} else {
				addrBytes, err := serialize.SerializeToBytes(tc.addr)
				if err != nil {
					t.Fatalf("Serialize: %v", err)
				}
				goBytes = append([]byte{0x01}, addrBytes...)
			}
			assertPythonMatch(t, tc.pythonHex, goBytes)
		})
	}
}

func TestPythonCompat_Uint256(t *testing.T) {
	tests := []struct {
		name      string
		value     *big.Int
		pythonHex string
	}{
		{"zero", big.NewInt(0), "0000000000000000000000000000000000000000000000000000000000000000"},
		{"small_1000", big.NewInt(1000), "00000000000000000000000000000000000000000000000000000000000003e8"},
		{"max_uint256", new(big.Int).Sub(new(big.Int).Lsh(big.NewInt(1), 256), big.NewInt(1)), "ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ui := serialize.Uint256{Value: tc.value}
			goBytes, err := serialize.SerializeToBytes(&ui)
			if err != nil {
				t.Fatalf("Serialize: %v", err)
			}
			assertPythonMatch(t, tc.pythonHex, goBytes)
		})
	}
}

func TestPythonCompat_BigUint(t *testing.T) {
	tests := []struct {
		name      string
		value     *big.Int
		pythonHex string
	}{
		{"zero", big.NewInt(0), "00"},
		{"small_1000000", big.NewInt(1000000), "030f4240"},
		{"power_of_256", new(big.Int).Exp(big.NewInt(256), big.NewInt(10), nil), "0b0100000000000000000000"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			bu := serialize.BigUint{Value: tc.value}
			goBytes, err := serialize.SerializeToBytes(&bu)
			if err != nil {
				t.Fatalf("Serialize: %v", err)
			}
			assertPythonMatch(t, tc.pythonHex, goBytes)
		})
	}
}

// --- Messages ---
//
// Coverage matrix:
//   PingRequest          → simple fields (bytes, []uint32, Optional)
//   SlaveInfo            → bytes, []uint32, uint16
//   GetAccountDataRequest → Address + Optional(uint64)

func TestPythonCompat_PingRequest(t *testing.T) {
	// ID="test", FullShardIDList=[1,2], RootTip=None
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
	assertPythonMatch(t, wantHex, buf)
}

func TestPythonCompat_SlaveInfo(t *testing.T) {
	// ID="s1", Host="localhost", Port=38391, FullShardIDList=[0x00010001]
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
	assertPythonMatch(t, wantHex, buf)
}

func TestPythonCompat_GetAccountDataRequest(t *testing.T) {
	// Address(recipient=0x0101..01, full_shard_key=0x00010001), BlockHeight=None
	wantHex := "010101010101010101010101010101010101010100010001" + "00"

	req := GetAccountDataRequest{
		Address: account.Address{
			Recipient:    account.Recipient{0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01},
			FullShardKey: 0x00010001,
		},
		BlockHeight: nil,
	}
	var buf []byte
	if err := serialize.Serialize(&buf, &req); err != nil {
		t.Fatalf("Serialize: %v", err)
	}
	assertPythonMatch(t, wantHex, buf)
}

func TestPythonCompat_GetAccountDataRequest_NonNilBlockHeight(t *testing.T) {
	// Address(recipient=0x0101..01, full_shard_key=0x00010001), BlockHeight=100
	// Covers: Optional(*uint64) non-nil → presence marker (0x01) + 8B uint64
	wantHex := "010101010101010101010101010101010101010100010001" +
		"01" +
		"0000000000000064"

	blockHeight := uint64(100)
	req := GetAccountDataRequest{
		Address: account.Address{
			Recipient:    account.Recipient{0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01},
			FullShardKey: 0x00010001,
		},
		BlockHeight: &blockHeight,
	}
	var buf []byte
	if err := serialize.Serialize(&buf, &req); err != nil {
		t.Fatalf("Serialize: %v", err)
	}
	assertPythonMatch(t, wantHex, buf)
}

func TestPythonCompat_TransactionDetail(t *testing.T) {
	// Covers: Uint256 in message context, Optional(Address) non-nil in message context,
	// hash256, uint64, bool, and nested struct composition.
	//
	// Python FIELDS:
	//   ("tx_hash", hash256),              # [32]byte
	//   ("nonce", uint64),                 # uint64
	//   ("from_address", Address),         # account.Address (24B)
	//   ("to_address", Optional(Address)), # *account.Address + ser:"nil"
	//   ("value", uint256),                # serialize.Uint256 (32B fixed)
	//   ("block_height", uint64),          # uint64
	//   ("timestamp", uint64),             # uint64
	//   ("success", boolean),              # bool
	//   ("gas_token_id", uint64),          # uint64
	//   ("transfer_token_id", uint64),     # uint64
	//   ("is_from_root_chain", boolean),   # bool
	//
	// TxHash = makeHash(0x01) = {0x01, 0x02, ..., 0x20}
	// FromAddress.Recipient = {0x02, 0x00, ..., 0x00} (20 bytes)
	// ToAddress.Recipient = {0x03, 0x00, ..., 0x00} (20 bytes)
	wantHex := "0102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1f20" + // tx_hash (32B)
		"0000000000000005" + // nonce (8B)
		"020000000000000000000000000000000000000000010001" + // from_address (24B)
		"01" + // to_address present marker
		"030000000000000000000000000000000000000000020002" + // to_address (24B)
		"00000000000000000000000000000000000000000000000000000000000003e8" + // value uint256 (32B)
		"0000000000000064" + // block_height (8B)
		"00000000499602d2" + // timestamp (8B)
		"01" + // success (1B)
		"0000000000000001" + // gas_token_id (8B)
		"0000000000000002" + // transfer_token_id (8B)
		"00" // is_from_root_chain (1B)

	detail := TransactionDetail{
		TxHash:          makeHash(0x01),
		Nonce:           5,
		FromAddress:     makeAddress(0x00010001, 0x02),
		ToAddress:       &account.Address{Recipient: account.Recipient{0x03}, FullShardKey: 0x00020002},
		Value:           serialize.Uint256{Value: big.NewInt(1000)},
		BlockHeight:     100,
		Timestamp:       1234567890,
		Success:         true,
		GasTokenID:      1,
		TransferTokenID: 2,
		IsFromRootChain: false,
	}
	var buf []byte
	if err := serialize.Serialize(&buf, &detail); err != nil {
		t.Fatalf("Serialize: %v", err)
	}
	assertPythonMatch(t, wantHex, buf)
}

func TestPythonCompat_SyncMinorBlockListResponse_NonNilShardStats(t *testing.T) {
	// Covers: BigUint in message context (via ShardStats.Difficulty),
	// Optional(struct) non-nil (ShardStats), and nested struct composition.
	//
	// Python FIELDS:
	//   ("error_code", uint32),
	//   ("block_coinbase_map", PrependedSizeMapSerializer(4, hash256, TokenBalanceMap)),
	//   ("shard_stats", Optional(ShardStats)),
	//
	// ShardStats FIELDS:
	//   ("branch", Branch),              # uint32
	//   ("height", uint64),              # uint64
	//   ("difficulty", biguint),         # BigUint (1B len + bytes)
	//   ("coinbase_address", Address),   # account.Address (24B)
	//   ("timestamp", uint64),           # uint64
	//   ("tx_count60s", uint32),         # uint32
	//   ("pending_tx_count", uint32),    # uint32
	//   ("total_tx_count", uint32),      # uint32
	//   ("block_count60s", uint32),      # uint32
	//   ("stale_block_count60s", uint32),# uint32
	//   ("last_block_time", uint32),     # uint32
	wantHex := "00000000" + // error_code (4B)
		"02aabb" + // block_coinbase_map: 1B len=2 + 2B data (3B)
		"01" + // shard_stats present marker (1B)
		"00000001" + // branch (4B)
		"0000000000000064" + // height (8B)
		"030f4240" + // difficulty: 1B len=3 + 3B data (4B)
		"040000000000000000000000000000000000000000010001" + // coinbase_address (24B)
		"00000000499602d2" + // timestamp (8B)
		"0000000a" + // tx_count60s (4B)
		"00000005" + // pending_tx_count (4B)
		"00000064" + // total_tx_count (4B)
		"00000002" + // block_count60s (4B)
		"00000001" + // stale_block_count60s (4B)
		"77359400" // last_block_time (4B)

	resp := SyncMinorBlockListResponse{
		ErrorCode:        0,
		BlockCoinbaseMap: &RawBytes{0xAA, 0xBB},
		ShardStats: &ShardStats{
			Branch:             1,
			Height:             100,
			Difficulty:         serialize.BigUint{Value: big.NewInt(1000000)},
			CoinbaseAddress:    makeAddress(0x00010001, 0x04),
			Timestamp:          1234567890,
			TxCount60s:         10,
			PendingTxCount:     5,
			TotalTxCount:       100,
			BlockCount60s:      2,
			StaleBlockCount60s: 1,
			LastBlockTime:      2000000000,
		},
	}
	var buf []byte
	if err := serialize.Serialize(&buf, &resp); err != nil {
		t.Fatalf("Serialize: %v", err)
	}
	assertPythonMatch(t, wantHex, buf)
}

// assertPythonMatch compares Go serialized bytes against Python golden hex.
func assertPythonMatch(t *testing.T, pythonHex string, goBytes []byte) {
	t.Helper()
	pythonBytes, err := hex.DecodeString(pythonHex)
	if err != nil {
		t.Fatalf("invalid python hex: %v", err)
	}
	if bytes.Equal(pythonBytes, goBytes) {
		return
	}
	t.Errorf("python/go wire mismatch")
	t.Errorf("  python (%d bytes): %s", len(pythonBytes), pythonHex)
	t.Errorf("  go     (%d bytes): %s", len(goBytes), hex.EncodeToString(goBytes))
	minLen := len(pythonBytes)
	if len(goBytes) < minLen {
		minLen = len(goBytes)
	}
	for i := 0; i < minLen; i++ {
		if pythonBytes[i] != goBytes[i] {
			start := i - 4
			if start < 0 {
				start = 0
			}
			end := i + 5
			if end > minLen {
				end = minLen
			}
			t.Errorf("  first diff at byte %d: python=%x go=%x", i, pythonBytes[start:end], goBytes[start:end])
			return
		}
	}
	if len(pythonBytes) != len(goBytes) {
		t.Errorf("  length mismatch: python=%d go=%d", len(pythonBytes), len(goBytes))
	}
}

// =============================================================================
// §4 Field order verification
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
// §5 Factory completeness
// =============================================================================

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

func TestOpcodeStructTypeMapping(t *testing.T) {
	// Verifies that each opcode maps to the correct struct type, not just non-nil.
	// Prevents opcode swap errors that would cause silent protocol corruption.
	clusterCases := []struct {
		op       ClusterOp
		expected reflect.Type
	}{
		{ClusterOpPing, reflect.TypeFor[PingRequest]()},
		{ClusterOpPong, reflect.TypeFor[PongResponse]()},
		{ClusterOpConnectToSlavesRequest, reflect.TypeFor[ConnectToSlavesRequest]()},
		{ClusterOpConnectToSlavesResponse, reflect.TypeFor[ConnectToSlavesResponse]()},
		{ClusterOpAddRootBlockRequest, reflect.TypeFor[AddRootBlockRequest]()},
		{ClusterOpAddRootBlockResponse, reflect.TypeFor[AddRootBlockResponse]()},
		{ClusterOpGetEcoInfoListRequest, reflect.TypeFor[GetEcoInfoListRequest]()},
		{ClusterOpGetEcoInfoListResponse, reflect.TypeFor[GetEcoInfoListResponse]()},
		{ClusterOpGetNextBlockToMineRequest, reflect.TypeFor[GetNextBlockToMineRequest]()},
		{ClusterOpGetNextBlockToMineResponse, reflect.TypeFor[GetNextBlockToMineResponse]()},
		{ClusterOpGetUnconfirmedHeadersRequest, reflect.TypeFor[GetUnconfirmedHeadersRequest]()},
		{ClusterOpGetUnconfirmedHeadersResponse, reflect.TypeFor[GetUnconfirmedHeadersResponse]()},
		{ClusterOpGetAccountDataRequest, reflect.TypeFor[GetAccountDataRequest]()},
		{ClusterOpGetAccountDataResponse, reflect.TypeFor[GetAccountDataResponse]()},
		{ClusterOpAddTransactionRequest, reflect.TypeFor[AddTransactionRequest]()},
		{ClusterOpAddTransactionResponse, reflect.TypeFor[AddTransactionResponse]()},
		{ClusterOpAddMinorBlockHeaderRequest, reflect.TypeFor[AddMinorBlockHeaderRequest]()},
		{ClusterOpAddMinorBlockHeaderResponse, reflect.TypeFor[AddMinorBlockHeaderResponse]()},
		{ClusterOpAddXshardTxListRequest, reflect.TypeFor[AddXshardTxListRequest]()},
		{ClusterOpAddXshardTxListResponse, reflect.TypeFor[AddXshardTxListResponse]()},
		{ClusterOpSyncMinorBlockListRequest, reflect.TypeFor[SyncMinorBlockListRequest]()},
		{ClusterOpSyncMinorBlockListResponse, reflect.TypeFor[SyncMinorBlockListResponse]()},
		{ClusterOpAddMinorBlockRequest, reflect.TypeFor[AddMinorBlockRequest]()},
		{ClusterOpAddMinorBlockResponse, reflect.TypeFor[AddMinorBlockResponse]()},
		{ClusterOpCreateClusterPeerConnectionRequest, reflect.TypeFor[CreateClusterPeerConnectionRequest]()},
		{ClusterOpCreateClusterPeerConnectionResponse, reflect.TypeFor[CreateClusterPeerConnectionResponse]()},
		{ClusterOpDestroyClusterPeerConnectionCommand, reflect.TypeFor[DestroyClusterPeerConnectionCommand]()},
		{ClusterOpGetMinorBlockRequest, reflect.TypeFor[GetMinorBlockRequest]()},
		{ClusterOpGetMinorBlockResponse, reflect.TypeFor[GetMinorBlockResponse]()},
		{ClusterOpGetTransactionRequest, reflect.TypeFor[GetTransactionRequest]()},
		{ClusterOpGetTransactionResponse, reflect.TypeFor[GetTransactionResponse]()},
		{ClusterOpBatchAddXshardTxListRequest, reflect.TypeFor[BatchAddXshardTxListRequest]()},
		{ClusterOpBatchAddXshardTxListResponse, reflect.TypeFor[BatchAddXshardTxListResponse]()},
		{ClusterOpExecuteTransactionRequest, reflect.TypeFor[ExecuteTransactionRequest]()},
		{ClusterOpExecuteTransactionResponse, reflect.TypeFor[ExecuteTransactionResponse]()},
		{ClusterOpGetTransactionReceiptRequest, reflect.TypeFor[GetTransactionReceiptRequest]()},
		{ClusterOpGetTransactionReceiptResponse, reflect.TypeFor[GetTransactionReceiptResponse]()},
		{ClusterOpMineRequest, reflect.TypeFor[MineRequest]()},
		{ClusterOpMineResponse, reflect.TypeFor[MineResponse]()},
		{ClusterOpGenTxRequest, reflect.TypeFor[GenTxRequest]()},
		{ClusterOpGenTxResponse, reflect.TypeFor[GenTxResponse]()},
		{ClusterOpGetTransactionListByAddressRequest, reflect.TypeFor[GetTransactionListByAddressRequest]()},
		{ClusterOpGetTransactionListByAddressResponse, reflect.TypeFor[GetTransactionListByAddressResponse]()},
		{ClusterOpGetLogRequest, reflect.TypeFor[GetLogRequest]()},
		{ClusterOpGetLogResponse, reflect.TypeFor[GetLogResponse]()},
		{ClusterOpEstimateGasRequest, reflect.TypeFor[EstimateGasRequest]()},
		{ClusterOpEstimateGasResponse, reflect.TypeFor[EstimateGasResponse]()},
		{ClusterOpGetStorageRequest, reflect.TypeFor[GetStorageRequest]()},
		{ClusterOpGetStorageResponse, reflect.TypeFor[GetStorageResponse]()},
		{ClusterOpGetCodeRequest, reflect.TypeFor[GetCodeRequest]()},
		{ClusterOpGetCodeResponse, reflect.TypeFor[GetCodeResponse]()},
		{ClusterOpGasPriceRequest, reflect.TypeFor[GasPriceRequest]()},
		{ClusterOpGasPriceResponse, reflect.TypeFor[GasPriceResponse]()},
		{ClusterOpGetWorkRequest, reflect.TypeFor[GetWorkRequest]()},
		{ClusterOpGetWorkResponse, reflect.TypeFor[GetWorkResponse]()},
		{ClusterOpSubmitWorkRequest, reflect.TypeFor[SubmitWorkRequest]()},
		{ClusterOpSubmitWorkResponse, reflect.TypeFor[SubmitWorkResponse]()},
		{ClusterOpAddMinorBlockHeaderListRequest, reflect.TypeFor[AddMinorBlockHeaderListRequest]()},
		{ClusterOpAddMinorBlockHeaderListResponse, reflect.TypeFor[AddMinorBlockHeaderListResponse]()},
		{ClusterOpCheckMinorBlockRequest, reflect.TypeFor[CheckMinorBlockRequest]()},
		{ClusterOpCheckMinorBlockResponse, reflect.TypeFor[CheckMinorBlockResponse]()},
		{ClusterOpGetAllTransactionsRequest, reflect.TypeFor[GetAllTransactionsRequest]()},
		{ClusterOpGetAllTransactionsResponse, reflect.TypeFor[GetAllTransactionsResponse]()},
		{ClusterOpGetRootChainStakesRequest, reflect.TypeFor[GetRootChainStakesRequest]()},
		{ClusterOpGetRootChainStakesResponse, reflect.TypeFor[GetRootChainStakesResponse]()},
		{ClusterOpGetTotalBalanceRequest, reflect.TypeFor[GetTotalBalanceRequest]()},
		{ClusterOpGetTotalBalanceResponse, reflect.TypeFor[GetTotalBalanceResponse]()},
	}
	for _, tc := range clusterCases {
		t.Run("", func(t *testing.T) {
			msg, err := NewClusterMessage(tc.op)
			if err != nil {
				t.Fatalf("NewClusterMessage(0x%x): %v", tc.op, err)
			}
			got := reflect.TypeOf(msg)
			want := reflect.PointerTo(tc.expected)
			if got != want {
				t.Errorf("opcode 0x%x: got type %v, want %v", tc.op, got, want)
			}
		})
	}

	commandCases := []struct {
		op       CommandOp
		expected reflect.Type
	}{
		{CommandOpHello, reflect.TypeFor[HelloCommand]()},
		{CommandOpNewMinorBlockHeaderList, reflect.TypeFor[NewMinorBlockHeaderListCommand]()},
		{CommandOpNewTransactionList, reflect.TypeFor[NewTransactionListCommand]()},
		{CommandOpGetPeerListRequest, reflect.TypeFor[GetPeerListRequest]()},
		{CommandOpGetPeerListResponse, reflect.TypeFor[GetPeerListResponse]()},
		{CommandOpGetRootBlockHeaderListRequest, reflect.TypeFor[GetRootBlockHeaderListRequest]()},
		{CommandOpGetRootBlockHeaderListResponse, reflect.TypeFor[GetRootBlockHeaderListResponse]()},
		{CommandOpGetRootBlockListRequest, reflect.TypeFor[GetRootBlockListRequest]()},
		{CommandOpGetRootBlockListResponse, reflect.TypeFor[GetRootBlockListResponse]()},
		{CommandOpGetMinorBlockListRequest, reflect.TypeFor[GetMinorBlockListRequest]()},
		{CommandOpGetMinorBlockListResponse, reflect.TypeFor[GetMinorBlockListResponse]()},
		{CommandOpGetMinorBlockHeaderListRequest, reflect.TypeFor[GetMinorBlockHeaderListRequest]()},
		{CommandOpGetMinorBlockHeaderListResponse, reflect.TypeFor[GetMinorBlockHeaderListResponse]()},
		{CommandOpNewBlockMinor, reflect.TypeFor[NewBlockMinorCommand]()},
		{CommandOpPing, reflect.TypeFor[PingPongCommand]()},
		{CommandOpPong, reflect.TypeFor[PingPongCommand]()},
		{CommandOpGetRootBlockHeaderListWithSkipRequest, reflect.TypeFor[GetRootBlockHeaderListWithSkipRequest]()},
		{CommandOpGetRootBlockHeaderListWithSkipResponse, reflect.TypeFor[GetRootBlockHeaderListResponse]()},
		{CommandOpNewRootBlock, reflect.TypeFor[NewRootBlockCommand]()},
		{CommandOpGetMinorBlockHeaderListWithSkipRequest, reflect.TypeFor[GetMinorBlockHeaderListWithSkipRequest]()},
		{CommandOpGetMinorBlockHeaderListWithSkipResponse, reflect.TypeFor[GetMinorBlockHeaderListResponse]()},
	}
	for _, tc := range commandCases {
		t.Run("", func(t *testing.T) {
			msg, err := NewCommandMessage(tc.op)
			if err != nil {
				t.Fatalf("NewCommandMessage(0x%x): %v", tc.op, err)
			}
			got := reflect.TypeOf(msg)
			want := reflect.PointerTo(tc.expected)
			if got != want {
				t.Errorf("opcode 0x%x: got type %v, want %v", tc.op, got, want)
			}
		})
	}
}

// =============================================================================
// Helpers
// =============================================================================

func makeAddress(fullShardID uint32, recipient byte) account.Address {
	return account.Address{
		Recipient:    account.BytesToIdentityRecipient([]byte{recipient, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}),
		FullShardKey: fullShardID,
	}
}

func makeHash(seed byte) [HashLength]byte {
	var h [HashLength]byte
	for i := range h {
		h[i] = seed + byte(i)
	}
	return h
}
