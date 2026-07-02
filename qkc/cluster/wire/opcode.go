// Copyright 2026-2027, QuarkChain.

package wire

import (
	"errors"
	"fmt"
)

// CLUSTER_OP_BASE is the offset added to all cluster op values on wire.
// Values below CLUSTER_OP_BASE belong to CommandOp (P2P).
const CLUSTER_OP_BASE = 128

// =============================================================================
// ClusterOp  —  cluster RPC opcodes (master ↔ slave, slave ↔ slave)
// =============================================================================

const (
	// ── §1  Cluster initialisation ───────────────────────────────────────────
	OP_PING                             = 1 + CLUSTER_OP_BASE  // 0x81
	OP_PONG                             = 2 + CLUSTER_OP_BASE  // 0x82
	OP_CONNECT_TO_SLAVES_REQUEST        = 3 + CLUSTER_OP_BASE  // 0x83
	OP_CONNECT_TO_SLAVES_RESPONSE       = 4 + CLUSTER_OP_BASE  // 0x84
	OP_ADD_ROOT_BLOCK_REQUEST           = 5 + CLUSTER_OP_BASE  // 0x85
	OP_ADD_ROOT_BLOCK_RESPONSE          = 6 + CLUSTER_OP_BASE  // 0x86
	OP_GET_ECO_INFO_LIST_REQUEST        = 7 + CLUSTER_OP_BASE  // 0x87
	OP_GET_ECO_INFO_LIST_RESPONSE       = 8 + CLUSTER_OP_BASE  // 0x88
	OP_GET_NEXT_BLOCK_TO_MINE_REQUEST   = 9 + CLUSTER_OP_BASE  // 0x89
	OP_GET_NEXT_BLOCK_TO_MINE_RESPONSE  = 10 + CLUSTER_OP_BASE // 0x8A
	OP_GET_UNCONFIRMED_HEADERS_REQUEST  = 11 + CLUSTER_OP_BASE // 0x8B
	OP_GET_UNCONFIRMED_HEADERS_RESPONSE = 12 + CLUSTER_OP_BASE // 0x8C
	OP_GET_ACCOUNT_DATA_REQUEST         = 13 + CLUSTER_OP_BASE // 0x8D
	OP_GET_ACCOUNT_DATA_RESPONSE        = 14 + CLUSTER_OP_BASE // 0x8E
	OP_ADD_TRANSACTION_REQUEST          = 15 + CLUSTER_OP_BASE // 0x8F
	OP_ADD_TRANSACTION_RESPONSE         = 16 + CLUSTER_OP_BASE // 0x90

	// ── §2  Slave → Master (mining) ──────────────────────────────────────────
	OP_ADD_MINOR_BLOCK_HEADER_REQUEST  = 17 + CLUSTER_OP_BASE // 0x91
	OP_ADD_MINOR_BLOCK_HEADER_RESPONSE = 18 + CLUSTER_OP_BASE // 0x92

	// ── §3  Slave ↔ Slave (xshard direct) ────────────────────────────────────
	OP_ADD_XSHARD_TX_LIST_REQUEST  = 19 + CLUSTER_OP_BASE // 0x93
	OP_ADD_XSHARD_TX_LIST_RESPONSE = 20 + CLUSTER_OP_BASE // 0x94

	// ── §4  Master → Slave (sync / virtual conns) ────────────────────────────
	OP_SYNC_MINOR_BLOCK_LIST_REQUEST           = 21 + CLUSTER_OP_BASE // 0x95
	OP_SYNC_MINOR_BLOCK_LIST_RESPONSE          = 22 + CLUSTER_OP_BASE // 0x96
	OP_ADD_MINOR_BLOCK_REQUEST                 = 23 + CLUSTER_OP_BASE // 0x97
	OP_ADD_MINOR_BLOCK_RESPONSE                = 24 + CLUSTER_OP_BASE // 0x98
	OP_CREATE_CLUSTER_PEER_CONNECTION_REQUEST  = 25 + CLUSTER_OP_BASE // 0x99
	OP_CREATE_CLUSTER_PEER_CONNECTION_RESPONSE = 26 + CLUSTER_OP_BASE // 0x9A
	OP_DESTROY_CLUSTER_PEER_CONNECTION_COMMAND = 27 + CLUSTER_OP_BASE // 0x9B (non-RPC)

	// 28 is skipped in Python.  Wire value 0x9C is intentionally unused.

	OP_GET_MINOR_BLOCK_REQUEST  = 29 + CLUSTER_OP_BASE // 0x9D
	OP_GET_MINOR_BLOCK_RESPONSE = 30 + CLUSTER_OP_BASE // 0x9E
	OP_GET_TRANSACTION_REQUEST  = 31 + CLUSTER_OP_BASE // 0x9F
	OP_GET_TRANSACTION_RESPONSE = 32 + CLUSTER_OP_BASE // 0xA0

	// ── §5  Slave ↔ Slave (xshard batch) ─────────────────────────────────────
	OP_BATCH_ADD_XSHARD_TX_LIST_REQUEST  = 33 + CLUSTER_OP_BASE // 0xA1
	OP_BATCH_ADD_XSHARD_TX_LIST_RESPONSE = 34 + CLUSTER_OP_BASE // 0xA2

	// ── §6  Master → Slave (JSON-RPC-like) ───────────────────────────────────
	OP_EXECUTE_TRANSACTION_REQUEST              = 35 + CLUSTER_OP_BASE // 0xA3
	OP_EXECUTE_TRANSACTION_RESPONSE             = 36 + CLUSTER_OP_BASE // 0xA4
	OP_GET_TRANSACTION_RECEIPT_REQUEST          = 37 + CLUSTER_OP_BASE // 0xA5
	OP_GET_TRANSACTION_RECEIPT_RESPONSE         = 38 + CLUSTER_OP_BASE // 0xA6
	OP_MINE_REQUEST                             = 39 + CLUSTER_OP_BASE // 0xA7
	OP_MINE_RESPONSE                            = 40 + CLUSTER_OP_BASE // 0xA8
	OP_GEN_TX_REQUEST                           = 41 + CLUSTER_OP_BASE // 0xA9
	OP_GEN_TX_RESPONSE                          = 42 + CLUSTER_OP_BASE // 0xAA
	OP_GET_TRANSACTION_LIST_BY_ADDRESS_REQUEST  = 43 + CLUSTER_OP_BASE // 0xAB
	OP_GET_TRANSACTION_LIST_BY_ADDRESS_RESPONSE = 44 + CLUSTER_OP_BASE // 0xAC
	OP_GET_LOG_REQUEST                          = 45 + CLUSTER_OP_BASE // 0xAD
	OP_GET_LOG_RESPONSE                         = 46 + CLUSTER_OP_BASE // 0xAE
	OP_ESTIMATE_GAS_REQUEST                     = 47 + CLUSTER_OP_BASE // 0xAF
	OP_ESTIMATE_GAS_RESPONSE                    = 48 + CLUSTER_OP_BASE // 0xB0
	OP_GET_STORAGE_REQUEST                      = 49 + CLUSTER_OP_BASE // 0xB1
	OP_GET_STORAGE_RESPONSE                     = 50 + CLUSTER_OP_BASE // 0xB2
	OP_GET_CODE_REQUEST                         = 51 + CLUSTER_OP_BASE // 0xB3
	OP_GET_CODE_RESPONSE                        = 52 + CLUSTER_OP_BASE // 0xB4
	OP_GAS_PRICE_REQUEST                        = 53 + CLUSTER_OP_BASE // 0xB5
	OP_GAS_PRICE_RESPONSE                       = 54 + CLUSTER_OP_BASE // 0xB6
	OP_GET_WORK_REQUEST                         = 55 + CLUSTER_OP_BASE // 0xB7
	OP_GET_WORK_RESPONSE                        = 56 + CLUSTER_OP_BASE // 0xB8
	OP_SUBMIT_WORK_REQUEST                      = 57 + CLUSTER_OP_BASE // 0xB9
	OP_SUBMIT_WORK_RESPONSE                     = 58 + CLUSTER_OP_BASE // 0xBA

	// ── §7  Slave → Master (block list) ──────────────────────────────────────
	OP_ADD_MINOR_BLOCK_HEADER_LIST_REQUEST  = 59 + CLUSTER_OP_BASE // 0xBB
	OP_ADD_MINOR_BLOCK_HEADER_LIST_RESPONSE = 60 + CLUSTER_OP_BASE // 0xBC

	// ── §8  Master → Slave (JRPC & staking) ──────────────────────────────────
	OP_CHECK_MINOR_BLOCK_REQUEST      = 61 + CLUSTER_OP_BASE // 0xBD
	OP_CHECK_MINOR_BLOCK_RESPONSE     = 62 + CLUSTER_OP_BASE // 0xBE
	OP_GET_ALL_TRANSACTIONS_REQUEST   = 63 + CLUSTER_OP_BASE // 0xBF
	OP_GET_ALL_TRANSACTIONS_RESPONSE  = 64 + CLUSTER_OP_BASE // 0xC0
	OP_GET_ROOT_CHAIN_STAKES_REQUEST  = 65 + CLUSTER_OP_BASE // 0xC1
	OP_GET_ROOT_CHAIN_STAKES_RESPONSE = 66 + CLUSTER_OP_BASE // 0xC2
	OP_GET_TOTAL_BALANCE_REQUEST      = 67 + CLUSTER_OP_BASE // 0xC3
	OP_GET_TOTAL_BALANCE_RESPONSE     = 68 + CLUSTER_OP_BASE // 0xC4
)

// =============================================================================
// CommandOp  —  P2P command opcodes (peer ↔ peer, cluster_peer_id != 0)
// =============================================================================

const (
	// Master-only.
	OP_HELLO = 0x00

	// Master → Slave (NON-RPC).
	OP_NEW_MINOR_BLOCK_HEADER_LIST = 0x01
	OP_NEW_TRANSACTION_LIST        = 0x02

	// Master-only.
	OP_GET_PEER_LIST_REQUEST               = 0x03
	OP_GET_PEER_LIST_RESPONSE              = 0x04
	OP_GET_ROOT_BLOCK_HEADER_LIST_REQUEST  = 0x05
	OP_GET_ROOT_BLOCK_HEADER_LIST_RESPONSE = 0x06
	OP_GET_ROOT_BLOCK_LIST_REQUEST         = 0x07
	OP_GET_ROOT_BLOCK_LIST_RESPONSE        = 0x08

	// Master → Slave (RPC).
	OP_GET_MINOR_BLOCK_LIST_REQUEST         = 0x09
	OP_GET_MINOR_BLOCK_LIST_RESPONSE        = 0x0A
	OP_GET_MINOR_BLOCK_HEADER_LIST_REQUEST  = 0x0B
	OP_GET_MINOR_BLOCK_HEADER_LIST_RESPONSE = 0x0C

	// Master → Slave (NON-RPC).
	OP_NEW_BLOCK_MINOR = 0x0D

	// Master-only.
	OP_PING_P2P                                      = 0x0E
	OP_PONG_P2P                                      = 0x0F
	OP_GET_ROOT_BLOCK_HEADER_LIST_WITH_SKIP_REQUEST  = 0x10
	OP_GET_ROOT_BLOCK_HEADER_LIST_WITH_SKIP_RESPONSE = 0x11
	OP_NEW_ROOT_BLOCK                                = 0x12 // NON-RPC

	// Master → Slave (RPC).
	OP_GET_MINOR_BLOCK_HEADER_LIST_WITH_SKIP_REQUEST  = 0x13
	OP_GET_MINOR_BLOCK_HEADER_LIST_WITH_SKIP_RESPONSE = 0x14
)

// =============================================================================
// Opcode classification
// =============================================================================

// InCommandOpRange reports whether op lies in the P2P command range (< CLUSTER_OP_BASE).
// This only classifies the range, not validity. Callers must reject unrecognised
// values, e.g. via a default case in opcode switch statements.
func InCommandOpRange(op byte) bool { return op < CLUSTER_OP_BASE }

// InClusterOpRange reports whether op lies in the cluster range (≥ CLUSTER_OP_BASE).
// This only classifies the range, not validity. Callers must reject unrecognised
// values, e.g. via a default case in opcode switch statements.
func InClusterOpRange(op byte) bool { return op >= CLUSTER_OP_BASE }

// =============================================================================
// Errors
// =============================================================================

type ClusterError struct {
	Code    uint32
	Message string
}

func (e *ClusterError) Error() string {
	if e.Message != "" {
		return fmt.Sprintf("[%d] %s", e.Code, e.Message)
	}
	return fmt.Sprintf("cluster error [%d]", e.Code)
}

func NewClusterError(code uint32, msg string) *ClusterError {
	return &ClusterError{Code: code, Message: msg}
}

var (
	ErrConnectionClosed = errors.New("wire: connection closed")
	ErrUnsupportedOp    = errors.New("wire: unsupported op")
	ErrNotImplemented   = errors.New("wire: not implemented")
)
