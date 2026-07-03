// Copyright 2026-2027, QuarkChain.

package wire

// ClusterOpBase is the offset added to all cluster op values on wire.
// Values below ClusterOpBase belong to CommandOp (P2P).
const ClusterOpBase ClusterOp = 128

// ClusterOp  —  cluster RPC opcodes (master ↔ slave, slave ↔ slave)
type ClusterOp byte

const (
	// ── §1  Cluster initialisation ───────────────────────────────────────────
	ClusterOpPing                          ClusterOp = 1 + ClusterOpBase  // 0x81
	ClusterOpPong                          ClusterOp = 2 + ClusterOpBase  // 0x82
	ClusterOpConnectToSlavesRequest        ClusterOp = 3 + ClusterOpBase  // 0x83
	ClusterOpConnectToSlavesResponse       ClusterOp = 4 + ClusterOpBase  // 0x84
	ClusterOpAddRootBlockRequest           ClusterOp = 5 + ClusterOpBase  // 0x85
	ClusterOpAddRootBlockResponse          ClusterOp = 6 + ClusterOpBase  // 0x86
	ClusterOpGetEcoInfoListRequest         ClusterOp = 7 + ClusterOpBase  // 0x87
	ClusterOpGetEcoInfoListResponse        ClusterOp = 8 + ClusterOpBase  // 0x88
	ClusterOpGetNextBlockToMineRequest     ClusterOp = 9 + ClusterOpBase  // 0x89
	ClusterOpGetNextBlockToMineResponse    ClusterOp = 10 + ClusterOpBase // 0x8A
	ClusterOpGetUnconfirmedHeadersRequest  ClusterOp = 11 + ClusterOpBase // 0x8B
	ClusterOpGetUnconfirmedHeadersResponse ClusterOp = 12 + ClusterOpBase // 0x8C
	ClusterOpGetAccountDataRequest         ClusterOp = 13 + ClusterOpBase // 0x8D
	ClusterOpGetAccountDataResponse        ClusterOp = 14 + ClusterOpBase // 0x8E
	ClusterOpAddTransactionRequest         ClusterOp = 15 + ClusterOpBase // 0x8F
	ClusterOpAddTransactionResponse        ClusterOp = 16 + ClusterOpBase // 0x90

	// ── §2  Slave → Master (mining) ──────────────────────────────────────────
	ClusterOpAddMinorBlockHeaderRequest  ClusterOp = 17 + ClusterOpBase // 0x91
	ClusterOpAddMinorBlockHeaderResponse ClusterOp = 18 + ClusterOpBase // 0x92

	// ── §3  Slave ↔ Slave (xshard direct) ────────────────────────────────────
	ClusterOpAddXshardTxListRequest  ClusterOp = 19 + ClusterOpBase // 0x93
	ClusterOpAddXshardTxListResponse ClusterOp = 20 + ClusterOpBase // 0x94

	// ── §4  Master → Slave (sync / virtual conns) ────────────────────────────
	ClusterOpSyncMinorBlockListRequest           ClusterOp = 21 + ClusterOpBase // 0x95
	ClusterOpSyncMinorBlockListResponse          ClusterOp = 22 + ClusterOpBase // 0x96
	ClusterOpAddMinorBlockRequest                ClusterOp = 23 + ClusterOpBase // 0x97
	ClusterOpAddMinorBlockResponse               ClusterOp = 24 + ClusterOpBase // 0x98
	ClusterOpCreateClusterPeerConnectionRequest  ClusterOp = 25 + ClusterOpBase // 0x99
	ClusterOpCreateClusterPeerConnectionResponse ClusterOp = 26 + ClusterOpBase // 0x9A
	ClusterOpDestroyClusterPeerConnectionCommand ClusterOp = 27 + ClusterOpBase // 0x9B (non-RPC)

	// 28 is skipped in Python.  Wire value 0x9C is intentionally unused.

	ClusterOpGetMinorBlockRequest   ClusterOp = 29 + ClusterOpBase // 0x9D
	ClusterOpGetMinorBlockResponse  ClusterOp = 30 + ClusterOpBase // 0x9E
	ClusterOpGetTransactionRequest  ClusterOp = 31 + ClusterOpBase // 0x9F
	ClusterOpGetTransactionResponse ClusterOp = 32 + ClusterOpBase // 0xA0

	// ── §5  Slave ↔ Slave (xshard batch) ─────────────────────────────────────
	ClusterOpBatchAddXshardTxListRequest  ClusterOp = 33 + ClusterOpBase // 0xA1
	ClusterOpBatchAddXshardTxListResponse ClusterOp = 34 + ClusterOpBase // 0xA2

	// ── §6  Master → Slave (JSON-RPC-like) ───────────────────────────────────
	ClusterOpExecuteTransactionRequest           ClusterOp = 35 + ClusterOpBase // 0xA3
	ClusterOpExecuteTransactionResponse          ClusterOp = 36 + ClusterOpBase // 0xA4
	ClusterOpGetTransactionReceiptRequest        ClusterOp = 37 + ClusterOpBase // 0xA5
	ClusterOpGetTransactionReceiptResponse       ClusterOp = 38 + ClusterOpBase // 0xA6
	ClusterOpMineRequest                         ClusterOp = 39 + ClusterOpBase // 0xA7
	ClusterOpMineResponse                        ClusterOp = 40 + ClusterOpBase // 0xA8
	ClusterOpGenTxRequest                        ClusterOp = 41 + ClusterOpBase // 0xA9
	ClusterOpGenTxResponse                       ClusterOp = 42 + ClusterOpBase // 0xAA
	ClusterOpGetTransactionListByAddressRequest  ClusterOp = 43 + ClusterOpBase // 0xAB
	ClusterOpGetTransactionListByAddressResponse ClusterOp = 44 + ClusterOpBase // 0xAC
	ClusterOpGetLogRequest                       ClusterOp = 45 + ClusterOpBase // 0xAD
	ClusterOpGetLogResponse                      ClusterOp = 46 + ClusterOpBase // 0xAE
	ClusterOpEstimateGasRequest                  ClusterOp = 47 + ClusterOpBase // 0xAF
	ClusterOpEstimateGasResponse                 ClusterOp = 48 + ClusterOpBase // 0xB0
	ClusterOpGetStorageRequest                   ClusterOp = 49 + ClusterOpBase // 0xB1
	ClusterOpGetStorageResponse                  ClusterOp = 50 + ClusterOpBase // 0xB2
	ClusterOpGetCodeRequest                      ClusterOp = 51 + ClusterOpBase // 0xB3
	ClusterOpGetCodeResponse                     ClusterOp = 52 + ClusterOpBase // 0xB4
	ClusterOpGasPriceRequest                     ClusterOp = 53 + ClusterOpBase // 0xB5
	ClusterOpGasPriceResponse                    ClusterOp = 54 + ClusterOpBase // 0xB6
	ClusterOpGetWorkRequest                      ClusterOp = 55 + ClusterOpBase // 0xB7
	ClusterOpGetWorkResponse                     ClusterOp = 56 + ClusterOpBase // 0xB8
	ClusterOpSubmitWorkRequest                   ClusterOp = 57 + ClusterOpBase // 0xB9
	ClusterOpSubmitWorkResponse                  ClusterOp = 58 + ClusterOpBase // 0xBA

	// ── §7  Slave → Master (block list) ──────────────────────────────────────
	ClusterOpAddMinorBlockHeaderListRequest  ClusterOp = 59 + ClusterOpBase // 0xBB
	ClusterOpAddMinorBlockHeaderListResponse ClusterOp = 60 + ClusterOpBase // 0xBC

	// ── §8  Master → Slave (JRPC & staking) ──────────────────────────────────
	ClusterOpCheckMinorBlockRequest     ClusterOp = 61 + ClusterOpBase // 0xBD
	ClusterOpCheckMinorBlockResponse    ClusterOp = 62 + ClusterOpBase // 0xBE
	ClusterOpGetAllTransactionsRequest  ClusterOp = 63 + ClusterOpBase // 0xBF
	ClusterOpGetAllTransactionsResponse ClusterOp = 64 + ClusterOpBase // 0xC0
	ClusterOpGetRootChainStakesRequest  ClusterOp = 65 + ClusterOpBase // 0xC1
	ClusterOpGetRootChainStakesResponse ClusterOp = 66 + ClusterOpBase // 0xC2
	ClusterOpGetTotalBalanceRequest     ClusterOp = 67 + ClusterOpBase // 0xC3
	ClusterOpGetTotalBalanceResponse    ClusterOp = 68 + ClusterOpBase // 0xC4
)

// CommandOp  —  P2P command opcodes (peer ↔ peer, cluster_peer_id != 0)
type CommandOp byte

const (
	// Master-only.
	CommandOpHello CommandOp = 0x00

	// Master → Slave (NON-RPC).
	CommandOpNewMinorBlockHeaderList CommandOp = 0x01
	CommandOpNewTransactionList      CommandOp = 0x02

	// Master-only.
	CommandOpGetPeerListRequest             CommandOp = 0x03
	CommandOpGetPeerListResponse            CommandOp = 0x04
	CommandOpGetRootBlockHeaderListRequest  CommandOp = 0x05
	CommandOpGetRootBlockHeaderListResponse CommandOp = 0x06
	CommandOpGetRootBlockListRequest        CommandOp = 0x07
	CommandOpGetRootBlockListResponse       CommandOp = 0x08

	// Master → Slave (RPC).
	CommandOpGetMinorBlockListRequest        CommandOp = 0x09
	CommandOpGetMinorBlockListResponse       CommandOp = 0x0A
	CommandOpGetMinorBlockHeaderListRequest  CommandOp = 0x0B
	CommandOpGetMinorBlockHeaderListResponse CommandOp = 0x0C

	// Master → Slave (NON-RPC).
	CommandOpNewBlockMinor CommandOp = 0x0D

	// Master-only.
	CommandOpPing                                   CommandOp = 0x0E
	CommandOpPong                                   CommandOp = 0x0F
	CommandOpGetRootBlockHeaderListWithSkipRequest  CommandOp = 0x10
	CommandOpGetRootBlockHeaderListWithSkipResponse CommandOp = 0x11
	CommandOpNewRootBlock                           CommandOp = 0x12 // NON-RPC

	// Master → Slave (RPC).
	CommandOpGetMinorBlockHeaderListWithSkipRequest  CommandOp = 0x13
	CommandOpGetMinorBlockHeaderListWithSkipResponse CommandOp = 0x14
)

// =============================================================================
// Opcode classification
// =============================================================================

// InCommandOpRange reports whether op lies in the P2P command range (< ClusterOpBase).
// This only classifies the range, not validity. Callers must reject unrecognised
// values, e.g. via a default case in opcode switch statements.
func InCommandOpRange(op byte) bool { return op < byte(ClusterOpBase) }

// InClusterOpRange reports whether op lies in the cluster range (≥ ClusterOpBase).
// This only classifies the range, not validity. Callers must reject unrecognised
// values, e.g. via a default case in opcode switch statements.
func InClusterOpRange(op byte) bool { return op >= byte(ClusterOpBase) }
