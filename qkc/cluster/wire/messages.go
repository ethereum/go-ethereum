// Copyright 2026-2027, QuarkChain.

// Package wire defines the Go-side wire-compatible message structs for all
// Cluster RPC and P2P opcodes.
//
// These structs are a strict binary-compatible representation of the Python
// QuarkChain Serializable definitions in:
//   - quarkchain/cluster/rpc.py
//   - quarkchain/cluster/p2p_commands.py
//
// -----------------------------------------------------------------------------
// Protocol Contract
// -----------------------------------------------------------------------------
//
// This package defines a BYTE-LEVEL WIRE CONTRACT.
//
// The following invariants MUST always hold:
//   - Struct field order MUST match Python FIELDS order exactly
//   - Encoding MUST be byte-identical to Python Serializable output
//   - Optional fields MUST preserve presence markers
//   - Slice encoding MUST use 4-byte big-endian length prefixes
//
// Any deviation from these rules is considered a protocol-breaking change.
//
// -----------------------------------------------------------------------------
// Serialization Tags (qkc/serialize)
// -----------------------------------------------------------------------------
//
// The wire format is enforced via struct tags:
//
//	bytesizeofslicelen:"4"
//	  - 4-byte big-endian length prefix for slices
//	  - Compatible with Python PrependedSizeBytesSerializer(4)
//
//	ser:"nil"
//	  - Nullable pointer field with 1-byte presence marker
//	  - Compatible with Python Optional(T)
//
//	ser:"-"
//	  - Field is excluded from serialization
//
// -----------------------------------------------------------------------------
// Primitive Type Mapping (Python → Go)
// -----------------------------------------------------------------------------
//
//	uint8        → uint8              (1 byte)
//	uint16       → uint16             (2 bytes BE)
//	uint32       → uint32             (4 bytes BE)
//	uint64       → uint64             (8 bytes BE)
//	uint128      → [16]byte           (16 bytes BE)
//	uint256      → serialize.Uint256   (32 bytes big-endian)
//	biguint      → serialize.BigUint   (1-byte length + big-endian bytes)
//	hash256      → [32]byte           (32 bytes)
//	Branch       → uint32             (4 bytes)
//	Address      → account.Address     (24 bytes: 20B recipient + 4B full_shard_key)
//	signature65  → [65]byte           (65 bytes)
//	boolean      → bool               (0x00 / 0x01)
//
// -----------------------------------------------------------------------------
// Layout Organization
// -----------------------------------------------------------------------------
//
// Structs are grouped by opcode domain (see opcode.go):
//
//	§1  Cluster initialization
//	§2  Virtual connection management
//	§3  Block updates
//	§4  Block queries
//	§5  Account / staking
//	§6  Cross-shard communication
//	§7  P2P commands
//	§8  P2P queries
//
// This grouping is purely organizational and does NOT affect wire format.
//
// -----------------------------------------------------------------------------
// Design Principle
// -----------------------------------------------------------------------------
//
// This package is the SINGLE SOURCE OF TRUTH for wire-level compatibility
// between Go and Python implementations.
//
// It is NOT allowed to:
//   - introduce semantic deviations from Python FIELDS
//   - change serialization rules locally per struct
//   - diverge from opcode mapping defined in protocol.go
package wire

import (
	"github.com/ethereum/go-ethereum/qkc/account"
	"github.com/ethereum/go-ethereum/qkc/serialize"
)

// =============================================================================
// Wire-level constants
// =============================================================================
//
// These match quarkchain/core.py:Constant and Python built-in type sizes.

// HashLength is the byte length of a hash256.
const HashLength = 32

// SignatureLength is the byte length of a signature65.
const SignatureLength = 65

// UInt128Length is the byte length of a uint128.
const UInt128Length = 16

// =============================================================================
// §1  Cluster initialisation
// =============================================================================

// PingRequest (ClusterOp.PING, 0x81) — sent by master to initialise a slave.
//
//	FIELDS = [
//	    ("id", PrependedSizeBytesSerializer(4)),
//	    ("full_shard_id_list", PrependedSizeListSerializer(4, uint32)),
//	    ("root_tip", Optional(RootBlock)),
//	]
type PingRequest struct {
	ID              []byte    `bytesizeofslicelen:"4"`
	FullShardIDList []uint32  `bytesizeofslicelen:"4"`
	RootTip         *RawBytes `ser:"nil"` // TODO: Replace with *RootBlock once core.RootBlock is ported
}

// PongResponse (ClusterOp.PONG, 0x82) — slave's reply to PING.
//
//	FIELDS = [
//	    ("id", PrependedSizeBytesSerializer(4)),
//	    ("full_shard_id_list", PrependedSizeListSerializer(4, uint32)),
//	]
type PongResponse struct {
	ID              []byte   `bytesizeofslicelen:"4"`
	FullShardIDList []uint32 `bytesizeofslicelen:"4"`
}

// SlaveInfo (used by ConnectToSlavesRequest) — describes a remote slave.
//
//	FIELDS = [
//	    ("id", PrependedSizeBytesSerializer(4)),
//	    ("host", PrependedSizeBytesSerializer(4)),
//	    ("port", uint16),
//	    ("full_shard_id_list", PrependedSizeListSerializer(4, uint32)),
//	]
type SlaveInfo struct {
	ID              []byte `bytesizeofslicelen:"4"`
	Host            []byte `bytesizeofslicelen:"4"`
	Port            uint16
	FullShardIDList []uint32 `bytesizeofslicelen:"4"`
}

// ConnectToSlavesRequest (ClusterOp.CONNECT_TO_SLAVES_REQUEST, 0x83).
//
//	FIELDS = [("slave_info_list", PrependedSizeListSerializer(4, SlaveInfo))]
type ConnectToSlavesRequest struct {
	SlaveInfoList []SlaveInfo `bytesizeofslicelen:"4"`
}

// ConnectToSlavesResponse (ClusterOp.CONNECT_TO_SLAVES_RESPONSE, 0x84).
//
// result_list has the same size as slave_info_list; empty result = success,
// otherwise the bytes are a serialised error message.
//
//	FIELDS = [
//	    ("result_list", PrependedSizeListSerializer(4, PrependedSizeBytesSerializer(4)))
//	]
type ConnectToSlavesResponse struct {
	ResultList []PrependedSizeBytes4 `bytesizeofslicelen:"4"`
}

// ArtificialTxConfig — used by MineRequest / GetNextBlockToMineRequest /
// AddMinorBlockHeaderResponse.
//
//	FIELDS = [("target_root_block_time", uint32), ("target_minor_block_time", uint32)]
type ArtificialTxConfig struct {
	TargetRootBlockTime  uint32
	TargetMinorBlockTime uint32
}

// MineRequest (ClusterOp.MINE_REQUEST, 0xA7) — start/stop mining on slaves.
//
//	FIELDS = [("artificial_tx_config", ArtificialTxConfig), ("mining", boolean)]
type MineRequest struct {
	ArtificialTxConfig ArtificialTxConfig
	Mining             bool
}

// MineResponse (ClusterOp.MINE_RESPONSE, 0xA8).
type MineResponse struct {
	ErrorCode uint32
}

// GenTxRequest (ClusterOp.GEN_TX_REQUEST, 0xA9) — generate transactions.
//
//	FIELDS = [
//	    ("num_tx_per_shard", uint32),
//	    ("x_shard_percent", uint32),     # [0, 100]
//	    ("tx", TypedTransaction),
//	]
type GenTxRequest struct {
	NumTxPerShard uint32
	XShardPercent uint32
	Tx            *RawBytes // TODO: Replace with *TypedTransaction once core.TypedTransaction is ported
}

// GenTxResponse (ClusterOp.GEN_TX_RESPONSE, 0xAA).
type GenTxResponse struct {
	ErrorCode uint32
}

// =============================================================================
// §2  Virtual connection management (mode 3 — Peer→Master→Slave)
// =============================================================================

// CreateClusterPeerConnectionRequest (ClusterOp.CREATE_CLUSTER_PEER_CONNECTION_REQUEST, 0x99).
type CreateClusterPeerConnectionRequest struct {
	ClusterPeerID uint64
}

// CreateClusterPeerConnectionResponse (ClusterOp.CREATE_CLUSTER_PEER_CONNECTION_RESPONSE, 0x9A).
type CreateClusterPeerConnectionResponse struct {
	ErrorCode uint32
}

// DestroyClusterPeerConnectionCommand (ClusterOp.DESTROY_CLUSTER_PEER_CONNECTION_COMMAND, 0x9B)
// — fire-and-forget, no response.
type DestroyClusterPeerConnectionCommand struct {
	ClusterPeerID uint64
}

// =============================================================================
// §3  Block updates
// =============================================================================

// AddRootBlockRequest (ClusterOp.ADD_ROOT_BLOCK_REQUEST, 0x85).
//
//	FIELDS = [("root_block", RootBlock), ("expect_switch", boolean)]
type AddRootBlockRequest struct {
	// TODO: Replace with *RootBlock once core.RootBlock is ported.
	RootBlock    *RawBytes
	ExpectSwitch bool
}

// AddRootBlockResponse (ClusterOp.ADD_ROOT_BLOCK_RESPONSE, 0x86).
type AddRootBlockResponse struct {
	ErrorCode uint32
	Switched  bool
}

// EcoInfo — used by GetEcoInfoListResponse.
//
//	FIELDS = [
//	    ("branch", Branch),
//	    ("height", uint64),
//	    ("coinbase_amount", uint256),
//	    ("difficulty", biguint),
//	    ("unconfirmed_headers_coinbase_amount", uint256),
//	]
type EcoInfo struct {
	Branch                           uint32
	Height                           uint64
	CoinbaseAmount                   serialize.Uint256
	Difficulty                       serialize.BigUint
	UnconfirmedHeadersCoinbaseAmount serialize.Uint256
}

// GetEcoInfoListRequest (ClusterOp.GET_ECO_INFO_LIST_REQUEST, 0x87) — empty body.
type GetEcoInfoListRequest struct{}

// GetEcoInfoListResponse (ClusterOp.GET_ECO_INFO_LIST_RESPONSE, 0x88).
type GetEcoInfoListResponse struct {
	ErrorCode   uint32
	EcoInfoList []EcoInfo `bytesizeofslicelen:"4"`
}

// GetNextBlockToMineRequest (ClusterOp.GET_NEXT_BLOCK_TO_MINE_REQUEST, 0x89).
//
//	FIELDS = [
//	    ("branch", Branch),
//	    ("address", Address),
//	    ("artificial_tx_config", ArtificialTxConfig),
//	]
type GetNextBlockToMineRequest struct {
	Branch             uint32
	Address            account.Address
	ArtificialTxConfig ArtificialTxConfig
}

// GetNextBlockToMineResponse (ClusterOp.GET_NEXT_BLOCK_TO_MINE_RESPONSE, 0x8A).
type GetNextBlockToMineResponse struct {
	ErrorCode uint32
	Block     *RawBytes // TODO: Replace with *MinorBlock once core.MinorBlock is ported
}

// AddMinorBlockRequest (ClusterOp.ADD_MINOR_BLOCK_REQUEST, 0x97) — JRPC-mined blocks.
//
//	FIELDS = [("minor_block_data", PrependedSizeBytesSerializer(4))]
type AddMinorBlockRequest struct {
	MinorBlockData []byte `bytesizeofslicelen:"4"`
}

// AddMinorBlockResponse (ClusterOp.ADD_MINOR_BLOCK_RESPONSE, 0x98).
type AddMinorBlockResponse struct {
	ErrorCode uint32
}

// CheckMinorBlockRequest (ClusterOp.CHECK_MINOR_BLOCK_REQUEST, 0xBD).
type CheckMinorBlockRequest struct {
	MinorBlockHeader *RawBytes // TODO: Replace with *MinorBlockHeader once core.MinorBlockHeader is ported
}

// CheckMinorBlockResponse (ClusterOp.CHECK_MINOR_BLOCK_RESPONSE, 0xBE).
type CheckMinorBlockResponse struct {
	ErrorCode uint32
}

// HeadersInfo — used by GetUnconfirmedHeadersResponse.
type HeadersInfo struct {
	Branch     uint32
	HeaderList []*RawBytes `bytesizeofslicelen:"4"` // TODO: Replace with []*MinorBlockHeader once core.MinorBlockHeader is ported
}

// GetUnconfirmedHeadersRequest (ClusterOp.GET_UNCONFIRMED_HEADERS_REQUEST, 0x8B) — empty body.
type GetUnconfirmedHeadersRequest struct{}

// GetUnconfirmedHeadersResponse (ClusterOp.GET_UNCONFIRMED_HEADERS_RESPONSE, 0x8C).
type GetUnconfirmedHeadersResponse struct {
	ErrorCode       uint32
	HeadersInfoList []HeadersInfo `bytesizeofslicelen:"4"`
}

// AccountBranchData — used by GetAccountDataResponse.
//
//	FIELDS = [
//	    ("branch", Branch),
//	    ("transaction_count", uint256),
//	    ("token_balances", TokenBalanceMap),
//	    ("is_contract", boolean),
//	    ("posw_mineable_blocks", uint16),
//	    ("mined_blocks", uint16),
//	]
type AccountBranchData struct {
	Branch           uint32
	TransactionCount serialize.Uint256
	// TODO: Replace with *TokenBalanceMap once core.TokenBalanceMap is ported.
	TokenBalances      *RawBytes
	IsContract         bool
	PoswMineableBlocks uint16
	MinedBlocks        uint16
}

// GetAccountDataRequest (ClusterOp.GET_ACCOUNT_DATA_REQUEST, 0x8D).
type GetAccountDataRequest struct {
	Address     account.Address
	BlockHeight *uint64 `ser:"nil"` // Optional uint64
}

// GetAccountDataResponse (ClusterOp.GET_ACCOUNT_DATA_RESPONSE, 0x8E).
type GetAccountDataResponse struct {
	ErrorCode             uint32
	AccountBranchDataList []AccountBranchData `bytesizeofslicelen:"4"`
}

// AddTransactionRequest (ClusterOp.ADD_TRANSACTION_REQUEST, 0x8F).
type AddTransactionRequest struct {
	Tx *RawBytes // TODO: Replace with *TypedTransaction once core.TypedTransaction is ported
}

// AddTransactionResponse (ClusterOp.ADD_TRANSACTION_RESPONSE, 0x90).
type AddTransactionResponse struct {
	ErrorCode uint32
}

// ShardStats — used by AddMinorBlockHeaderRequest / SyncMinorBlockListResponse.
//
//	FIELDS = [
//	    ("branch", Branch),
//	    ("height", uint64),
//	    ("difficulty", biguint),
//	    ("coinbase_address", Address),
//	    ("timestamp", uint64),
//	    ("tx_count60s", uint32),
//	    ("pending_tx_count", uint32),
//	    ("total_tx_count", uint32),
//	    ("block_count60s", uint32),
//	    ("stale_block_count60s", uint32),
//	    ("last_block_time", uint32),
//	]
type ShardStats struct {
	Branch             uint32
	Height             uint64
	Difficulty         serialize.BigUint
	CoinbaseAddress    account.Address
	Timestamp          uint64
	TxCount60s         uint32
	PendingTxCount     uint32
	TotalTxCount       uint32
	BlockCount60s      uint32
	StaleBlockCount60s uint32
	LastBlockTime      uint32
}

// AddMinorBlockHeaderRequest (ClusterOp.ADD_MINOR_BLOCK_HEADER_REQUEST, 0x91) — slave→master.
//
//	FIELDS = [
//	    ("minor_block_header", MinorBlockHeader),
//	    ("tx_count", uint32),
//	    ("x_shard_tx_count", uint32),
//	    ("coinbase_amount_map", TokenBalanceMap),
//	    ("shard_stats", ShardStats),
//	]
type AddMinorBlockHeaderRequest struct {
	// TODO: Replace with *MinorBlockHeader once core.MinorBlockHeader is ported.
	MinorBlockHeader *RawBytes
	TxCount          uint32
	XShardTxCount    uint32
	// TODO: Replace with *TokenBalanceMap once core.TokenBalanceMap is ported.
	CoinbaseAmountMap *RawBytes
	ShardStats        ShardStats
}

// AddMinorBlockHeaderResponse (ClusterOp.ADD_MINOR_BLOCK_HEADER_RESPONSE, 0x92).
type AddMinorBlockHeaderResponse struct {
	ErrorCode          uint32
	ArtificialTxConfig ArtificialTxConfig
}

// AddMinorBlockHeaderListRequest (ClusterOp.ADD_MINOR_BLOCK_HEADER_LIST_REQUEST, 0xBB) — slave→master.
type AddMinorBlockHeaderListRequest struct {
	MinorBlockHeaderList  []*RawBytes `bytesizeofslicelen:"4"` // TODO: Replace with []*MinorBlockHeader once core.MinorBlockHeader is ported
	CoinbaseAmountMapList []*RawBytes `bytesizeofslicelen:"4"` // TODO: Replace with []*TokenBalanceMap once core.TokenBalanceMap is ported
}

// AddMinorBlockHeaderListResponse (ClusterOp.ADD_MINOR_BLOCK_HEADER_LIST_RESPONSE, 0xBC).
type AddMinorBlockHeaderListResponse struct {
	ErrorCode uint32
}

// SyncMinorBlockListRequest (ClusterOp.SYNC_MINOR_BLOCK_LIST_REQUEST, 0x95).
type SyncMinorBlockListRequest struct {
	MinorBlockHashList [][HashLength]byte `bytesizeofslicelen:"4"`
	Branch             uint32
	ClusterPeerID      uint64
}

// SyncMinorBlockListResponse (ClusterOp.SYNC_MINOR_BLOCK_LIST_RESPONSE, 0x96).
//
// block_coinbase_map: PrependedSizeMapSerializer(4, hash256, TokenBalanceMap)
//
//	FIELDS = [
//	    ("error_code", uint32),
//	    ("block_coinbase_map", PrependedSizeMapSerializer(4, hash256, TokenBalanceMap)),
//	    ("shard_stats", Optional(ShardStats)),
//	]
type SyncMinorBlockListResponse struct {
	ErrorCode uint32
	// TODO: Replace with real block_coinbase_map once core.TokenBalanceMap is ported.
	BlockCoinbaseMap *RawBytes
	ShardStats       *ShardStats `ser:"nil"`
}

// =============================================================================
// §4  Block queries
// =============================================================================

// MinorBlockExtraInfo — used by GetMinorBlockResponse.
type MinorBlockExtraInfo struct {
	EffectiveDifficulty serialize.BigUint
	PoswMineableBlocks  uint16
	PoswMinedBlocks     uint16
}

// GetMinorBlockRequest (ClusterOp.GET_MINOR_BLOCK_REQUEST, 0x9D).
type GetMinorBlockRequest struct {
	Branch         uint32
	MinorBlockHash [HashLength]byte
	Height         uint64
	NeedExtraInfo  bool
}

// GetMinorBlockResponse (ClusterOp.GET_MINOR_BLOCK_RESPONSE, 0x9E).
type GetMinorBlockResponse struct {
	ErrorCode uint32
	// TODO: Replace with *MinorBlock once core.MinorBlock is ported.
	MinorBlock *RawBytes
	ExtraInfo  *MinorBlockExtraInfo `ser:"nil"`
}

// GetTransactionRequest (ClusterOp.GET_TRANSACTION_REQUEST, 0x9F).
type GetTransactionRequest struct {
	TxHash [HashLength]byte
	Branch uint32
}

// GetTransactionResponse (ClusterOp.GET_TRANSACTION_RESPONSE, 0xA0).
type GetTransactionResponse struct {
	ErrorCode uint32
	// TODO: Replace with *MinorBlock once core.MinorBlock is ported.
	MinorBlock *RawBytes
	Index      uint32
}

// ExecuteTransactionRequest (ClusterOp.EXECUTE_TRANSACTION_REQUEST, 0xA3).
type ExecuteTransactionRequest struct {
	// TODO: Replace with *TypedTransaction once core.TypedTransaction is ported.
	Tx          *RawBytes
	FromAddress account.Address
	BlockHeight *uint64 `ser:"nil"`
}

// ExecuteTransactionResponse (ClusterOp.EXECUTE_TRANSACTION_RESPONSE, 0xA4).
type ExecuteTransactionResponse struct {
	ErrorCode uint32
	Result    []byte `bytesizeofslicelen:"4"`
}

// GetTransactionReceiptRequest (ClusterOp.GET_TRANSACTION_RECEIPT_REQUEST, 0xA5).
type GetTransactionReceiptRequest struct {
	TxHash [HashLength]byte
	Branch uint32
}

// GetTransactionReceiptResponse (ClusterOp.GET_TRANSACTION_RECEIPT_RESPONSE, 0xA6).
type GetTransactionReceiptResponse struct {
	ErrorCode uint32
	// TODO: Replace with *MinorBlock once core.MinorBlock is ported.
	MinorBlock *RawBytes
	Index      uint32
	// TODO: Replace with *TransactionReceipt once core.TransactionReceipt is ported.
	Receipt *RawBytes
}

// TransactionDetail — used by GetTransactionListByAddressResponse and
// GetAllTransactionsResponse.
type TransactionDetail struct {
	TxHash          [HashLength]byte
	Nonce           uint64
	FromAddress     account.Address
	ToAddress       *account.Address `ser:"nil"` // Optional Address
	Value           serialize.Uint256
	BlockHeight     uint64
	Timestamp       uint64
	Success         bool
	GasTokenID      uint64
	TransferTokenID uint64
	IsFromRootChain bool
}

// GetTransactionListByAddressRequest (ClusterOp.GET_TRANSACTION_LIST_BY_ADDRESS_REQUEST, 0xAB).
type GetTransactionListByAddressRequest struct {
	Address         account.Address
	TransferTokenID *uint64 `ser:"nil"`
	Start           []byte  `bytesizeofslicelen:"4"`
	Limit           uint32
}

// GetTransactionListByAddressResponse (ClusterOp.GET_TRANSACTION_LIST_BY_ADDRESS_RESPONSE, 0xAC).
type GetTransactionListByAddressResponse struct {
	ErrorCode uint32
	TxList    []TransactionDetail `bytesizeofslicelen:"4"`
	Next      []byte              `bytesizeofslicelen:"4"`
}

// GetAllTransactionsRequest (ClusterOp.GET_ALL_TRANSACTIONS_REQUEST, 0xBF).
type GetAllTransactionsRequest struct {
	Branch uint32
	Start  []byte `bytesizeofslicelen:"4"`
	Limit  uint32
}

// GetAllTransactionsResponse (ClusterOp.GET_ALL_TRANSACTIONS_RESPONSE, 0xC0).
type GetAllTransactionsResponse struct {
	ErrorCode uint32
	TxList    []TransactionDetail `bytesizeofslicelen:"4"`
	Next      []byte              `bytesizeofslicelen:"4"`
}

// GetLogRequest (ClusterOp.GET_LOG_REQUEST, 0xAD).
//
//	FIELDS = [
//	    ("branch", Branch),
//	    ("addresses", PrependedSizeListSerializer(4, Address)),
//	    ("topics", PrependedSizeListSerializer(4, PrependedSizeListSerializer(4, hash256))),
//	    ("start_block", uint64),
//	    ("end_block", uint64),
//	]
type GetLogRequest struct {
	Branch     uint32
	Addresses  []account.Address        `bytesizeofslicelen:"4"`
	Topics     []PrependedSizeHashList4 `bytesizeofslicelen:"4"`
	StartBlock uint64
	EndBlock   uint64
}

// GetLogResponse (ClusterOp.GET_LOG_RESPONSE, 0xAE).
type GetLogResponse struct {
	ErrorCode uint32
	Logs      []*RawBytes `bytesizeofslicelen:"4"` // TODO: Replace with []*Log once core.Log is ported
}

// EstimateGasRequest (ClusterOp.ESTIMATE_GAS_REQUEST, 0xAF).
type EstimateGasRequest struct {
	// TODO: Replace with *TypedTransaction once core.TypedTransaction is ported.
	Tx          *RawBytes
	FromAddress account.Address
}

// EstimateGasResponse (ClusterOp.ESTIMATE_GAS_RESPONSE, 0xB0).
type EstimateGasResponse struct {
	ErrorCode uint32
	Result    uint32
}

// GetStorageRequest (ClusterOp.GET_STORAGE_REQUEST, 0xB1).
type GetStorageRequest struct {
	Address     account.Address
	Key         serialize.Uint256
	BlockHeight *uint64 `ser:"nil"`
}

// GetStorageResponse (ClusterOp.GET_STORAGE_RESPONSE, 0xB2).
type GetStorageResponse struct {
	ErrorCode uint32
	Result    [HashLength]byte
}

// GetCodeRequest (ClusterOp.GET_CODE_REQUEST, 0xB3).
type GetCodeRequest struct {
	Address     account.Address
	BlockHeight *uint64 `ser:"nil"`
}

// GetCodeResponse (ClusterOp.GET_CODE_RESPONSE, 0xB4).
type GetCodeResponse struct {
	ErrorCode uint32
	Result    []byte `bytesizeofslicelen:"4"`
}

// GasPriceRequest (ClusterOp.GAS_PRICE_REQUEST, 0xB5).
type GasPriceRequest struct {
	Branch  uint32
	TokenID uint64
}

// GasPriceResponse (ClusterOp.GAS_PRICE_RESPONSE, 0xB6).
type GasPriceResponse struct {
	ErrorCode uint32
	Result    uint64
}

// GetWorkRequest (ClusterOp.GET_WORK_REQUEST, 0xB7).
type GetWorkRequest struct {
	Branch       uint32
	CoinbaseAddr *account.Address `ser:"nil"` // Optional Address
}

// GetWorkResponse (ClusterOp.GET_WORK_RESPONSE, 0xB8).
type GetWorkResponse struct {
	ErrorCode  uint32
	HeaderHash [HashLength]byte
	Height     uint64
	Difficulty serialize.BigUint
}

// SubmitWorkRequest (ClusterOp.SUBMIT_WORK_REQUEST, 0xB9).
type SubmitWorkRequest struct {
	Branch     uint32
	HeaderHash [HashLength]byte
	Nonce      uint64
	Mixhash    [HashLength]byte
	Signature  *[SignatureLength]byte `ser:"nil"` // Optional signature65
}

// SubmitWorkResponse (ClusterOp.SUBMIT_WORK_RESPONSE, 0xBA).
type SubmitWorkResponse struct {
	ErrorCode uint32
	Success   bool
}

// =============================================================================
// §5  Account / staking
// =============================================================================

// GetRootChainStakesRequest (ClusterOp.GET_ROOT_CHAIN_STAKES_REQUEST, 0xC1).
type GetRootChainStakesRequest struct {
	Address        account.Address
	MinorBlockHash [HashLength]byte
}

// GetRootChainStakesResponse (ClusterOp.GET_ROOT_CHAIN_STAKES_RESPONSE, 0xC2).
type GetRootChainStakesResponse struct {
	ErrorCode uint32
	Stakes    serialize.BigUint
	Signer    [20]byte
}

// GetTotalBalanceRequest (ClusterOp.GET_TOTAL_BALANCE_REQUEST, 0xC3).
type GetTotalBalanceRequest struct {
	Branch         uint32
	Start          *[HashLength]byte `ser:"nil"` // Optional hash256
	TokenID        uint64
	Limit          uint32
	MinorBlockHash [HashLength]byte
	RootBlockHash  *[HashLength]byte `ser:"nil"` // Optional hash256
}

// GetTotalBalanceResponse (ClusterOp.GET_TOTAL_BALANCE_RESPONSE, 0xC4).
type GetTotalBalanceResponse struct {
	ErrorCode    uint32
	TotalBalance serialize.BigUint
	Next         []byte `bytesizeofslicelen:"4"`
}

// =============================================================================
// §6  Cross-shard (Slave↔Slave, direct TCP, no metadata)
// =============================================================================

// AddXshardTxListRequest (ClusterOp.ADD_XSHARD_TX_LIST_REQUEST, 0x93).
type AddXshardTxListRequest struct {
	Branch         uint32
	MinorBlockHash [HashLength]byte
	TxList         *RawBytes // TODO: Replace with *CrossShardTransactionList once core.CrossShardTransactionList is ported
}

// AddXshardTxListResponse (ClusterOp.ADD_XSHARD_TX_LIST_RESPONSE, 0x94).
type AddXshardTxListResponse struct {
	ErrorCode uint32
}

// BatchAddXshardTxListRequest (ClusterOp.BATCH_ADD_XSHARD_TX_LIST_REQUEST, 0xA1).
type BatchAddXshardTxListRequest struct {
	AddXshardTxListRequestList []AddXshardTxListRequest `bytesizeofslicelen:"4"`
}

// BatchAddXshardTxListResponse (ClusterOp.BATCH_ADD_XSHARD_TX_LIST_RESPONSE, 0xA2).
type BatchAddXshardTxListResponse struct {
	ErrorCode uint32
}

// =============================================================================
// §7  P2P commands (CommandOp, cluster_peer_id != 0)
// =============================================================================

// HelloCommand (CommandOp.HELLO, 0x00) — initial inter-cluster handshake.
//
//	FIELDS = [
//	    ("version", uint32),
//	    ("network_id", uint32),
//	    ("peer_id", hash256),
//	    ("peer_ip", uint128),
//	    ("peer_port", uint16),
//	    ("chain_mask_list", PrependedSizeListSerializer(4, uint32)),
//	    ("root_block_header", RootBlockHeader),
//	    ("genesis_root_block_hash", hash256),
//	]
type HelloCommand struct {
	Version       uint32
	NetworkID     uint32
	PeerID        [HashLength]byte
	PeerIP        [UInt128Length]byte
	PeerPort      uint16
	ChainMaskList []uint32 `bytesizeofslicelen:"4"`
	// TODO: Replace with *RootBlockHeader once core.RootBlockHeader is ported.
	RootBlockHeader      *RawBytes
	GenesisRootBlockHash [HashLength]byte
}

// NewMinorBlockHeaderListCommand (CommandOp.NEW_MINOR_BLOCK_HEADER_LIST, 0x01).
type NewMinorBlockHeaderListCommand struct {
	// TODO: Replace with *RootBlockHeader once core.RootBlockHeader is ported.
	RootBlockHeader      *RawBytes
	MinorBlockHeaderList []*RawBytes `bytesizeofslicelen:"4"` // TODO: Replace with []*MinorBlockHeader once core.MinorBlockHeader is ported
}

// NewTransactionListCommand (CommandOp.NEW_TRANSACTION_LIST, 0x02).
type NewTransactionListCommand struct {
	TransactionList []*RawBytes `bytesizeofslicelen:"4"` // TODO: Replace with []*TypedTransaction once core.TypedTransaction is ported
}

// NewBlockMinorCommand (CommandOp.NEW_BLOCK_MINOR, 0x0D).
type NewBlockMinorCommand struct {
	Block *RawBytes // TODO: Replace with *MinorBlock once core.MinorBlock is ported
}

// PingPongCommand (CommandOp.PING 0x0E, PONG 0x0F).
type PingPongCommand struct {
	Message [HashLength]byte
}

// NewRootBlockCommand (CommandOp.NEW_ROOT_BLOCK, 0x12).
type NewRootBlockCommand struct {
	Block *RawBytes // TODO: Replace with *RootBlock once core.RootBlock is ported
}

// =============================================================================
// §8  P2P queries (CommandOp)
// =============================================================================

// PeerInfo — used by GetPeerListResponse.
type PeerInfo struct {
	IP   [UInt128Length]byte
	Port uint16
}

// GetPeerListRequest (CommandOp.GET_PEER_LIST_REQUEST, 0x03).
type GetPeerListRequest struct {
	MaxPeers uint32
}

// GetPeerListResponse (CommandOp.GET_PEER_LIST_RESPONSE, 0x04).
type GetPeerListResponse struct {
	PeerInfoList []PeerInfo `bytesizeofslicelen:"4"`
}

// GetRootBlockHeaderListRequest (CommandOp.GET_ROOT_BLOCK_HEADER_LIST_REQUEST, 0x05).
type GetRootBlockHeaderListRequest struct {
	BlockHash [HashLength]byte
	Limit     uint32
	Direction Direction
}

// GetRootBlockHeaderListResponse (CommandOp.GET_ROOT_BLOCK_HEADER_LIST_RESPONSE, 0x06).
type GetRootBlockHeaderListResponse struct {
	// TODO: Replace with *RootBlockHeader once core.RootBlockHeader is ported.
	RootTip         *RawBytes
	BlockHeaderList []*RawBytes `bytesizeofslicelen:"4"` // TODO: Replace with []*RootBlockHeader once core.RootBlockHeader is ported
}

// GetRootBlockHeaderListWithSkipRequest (CommandOp.GET_ROOT_BLOCK_HEADER_LIST_WITH_SKIP_REQUEST, 0x10).
type GetRootBlockHeaderListWithSkipRequest struct {
	Type      uint8
	Data      [HashLength]byte
	Limit     uint32
	Skip      uint32
	Direction Direction
}

// GetRootBlockListRequest (CommandOp.GET_ROOT_BLOCK_LIST_REQUEST, 0x07).
type GetRootBlockListRequest struct {
	RootBlockHashList [][HashLength]byte `bytesizeofslicelen:"4"`
}

// GetRootBlockListResponse (CommandOp.GET_ROOT_BLOCK_LIST_RESPONSE, 0x08).
type GetRootBlockListResponse struct {
	RootBlockList []*RawBytes `bytesizeofslicelen:"4"` // TODO: Replace with []*RootBlock once core.RootBlock is ported
}

// GetMinorBlockListRequest (CommandOp.GET_MINOR_BLOCK_LIST_REQUEST, 0x09).
type GetMinorBlockListRequest struct {
	MinorBlockHashList [][HashLength]byte `bytesizeofslicelen:"4"`
}

// GetMinorBlockListResponse (CommandOp.GET_MINOR_BLOCK_LIST_RESPONSE, 0x0A).
type GetMinorBlockListResponse struct {
	MinorBlockList []*RawBytes `bytesizeofslicelen:"4"` // TODO: Replace with []*MinorBlock once core.MinorBlock is ported
}

// GetMinorBlockHeaderListRequest (CommandOp.GET_MINOR_BLOCK_HEADER_LIST_REQUEST, 0x0B).
type GetMinorBlockHeaderListRequest struct {
	BlockHash [HashLength]byte
	Branch    uint32
	Limit     uint32
	Direction Direction
}

// GetMinorBlockHeaderListResponse (CommandOp.GET_MINOR_BLOCK_HEADER_LIST_RESPONSE, 0x0C).
type GetMinorBlockHeaderListResponse struct {
	// TODO: Replace with *RootBlockHeader once core.RootBlockHeader is ported.
	RootTip *RawBytes
	// TODO: Replace with *MinorBlockHeader once core.MinorBlockHeader is ported.
	ShardTip        *RawBytes
	BlockHeaderList []*RawBytes `bytesizeofslicelen:"4"` // TODO: Replace with []*MinorBlockHeader once core.MinorBlockHeader is ported
}

// GetMinorBlockHeaderListWithSkipRequest (CommandOp.GET_MINOR_BLOCK_HEADER_LIST_WITH_SKIP_REQUEST, 0x13).
type GetMinorBlockHeaderListWithSkipRequest struct {
	Type      uint8
	Data      [HashLength]byte
	Branch    uint32
	Limit     uint32
	Skip      uint32
	Direction Direction
}
