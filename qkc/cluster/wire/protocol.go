// Copyright 2026-2027, QuarkChain.

// Package wire: opcode → message factory and protocol-level enumeration
// types.  This file maps each ClusterOp / CommandOp to the concrete message
// struct that decodes a frame payload of that opcode.
package wire

import (
	"fmt"
)

// Direction matches the P2P sync Direction enum in
// quarkchain/cluster/p2p_commands.py.
//
//	DIRECTIONS = [GENESIS, TIP]
//	class Direction(IntEnum):
//	    GENESIS = 0
//	    TIP     = 1
type Direction uint8

const (
	DirectionGenesis Direction = 0
	DirectionTip     Direction = 1
)

func (d Direction) String() string {
	switch d {
	case DirectionGenesis:
		return "GENESIS"
	case DirectionTip:
		return "TIP"
	default:
		return fmt.Sprintf("Direction(%d)", uint8(d))
	}
}

// NewClusterMessage returns a new empty message struct pointer corresponding
// to the given ClusterOp.  Used by slave/master_conn.go to allocate the
// concrete type before deserialising a frame payload.
func NewClusterMessage(op ClusterOp) (interface{}, error) {
	switch op {
	case ClusterOpPing:
		return &PingRequest{}, nil
	case ClusterOpPong:
		return &PongResponse{}, nil
	case ClusterOpConnectToSlavesRequest:
		return &ConnectToSlavesRequest{}, nil
	case ClusterOpConnectToSlavesResponse:
		return &ConnectToSlavesResponse{}, nil
	case ClusterOpAddRootBlockRequest:
		return &AddRootBlockRequest{}, nil
	case ClusterOpAddRootBlockResponse:
		return &AddRootBlockResponse{}, nil
	case ClusterOpGetEcoInfoListRequest:
		return &GetEcoInfoListRequest{}, nil
	case ClusterOpGetEcoInfoListResponse:
		return &GetEcoInfoListResponse{}, nil
	case ClusterOpGetNextBlockToMineRequest:
		return &GetNextBlockToMineRequest{}, nil
	case ClusterOpGetNextBlockToMineResponse:
		return &GetNextBlockToMineResponse{}, nil
	case ClusterOpGetUnconfirmedHeadersRequest:
		return &GetUnconfirmedHeadersRequest{}, nil
	case ClusterOpGetUnconfirmedHeadersResponse:
		return &GetUnconfirmedHeadersResponse{}, nil
	case ClusterOpGetAccountDataRequest:
		return &GetAccountDataRequest{}, nil
	case ClusterOpGetAccountDataResponse:
		return &GetAccountDataResponse{}, nil
	case ClusterOpAddTransactionRequest:
		return &AddTransactionRequest{}, nil
	case ClusterOpAddTransactionResponse:
		return &AddTransactionResponse{}, nil
	case ClusterOpAddMinorBlockHeaderRequest:
		return &AddMinorBlockHeaderRequest{}, nil
	case ClusterOpAddMinorBlockHeaderResponse:
		return &AddMinorBlockHeaderResponse{}, nil
	case ClusterOpAddXshardTxListRequest:
		return &AddXshardTxListRequest{}, nil
	case ClusterOpAddXshardTxListResponse:
		return &AddXshardTxListResponse{}, nil
	case ClusterOpSyncMinorBlockListRequest:
		return &SyncMinorBlockListRequest{}, nil
	case ClusterOpSyncMinorBlockListResponse:
		return &SyncMinorBlockListResponse{}, nil
	case ClusterOpAddMinorBlockRequest:
		return &AddMinorBlockRequest{}, nil
	case ClusterOpAddMinorBlockResponse:
		return &AddMinorBlockResponse{}, nil
	case ClusterOpCreateClusterPeerConnectionRequest:
		return &CreateClusterPeerConnectionRequest{}, nil
	case ClusterOpCreateClusterPeerConnectionResponse:
		return &CreateClusterPeerConnectionResponse{}, nil
	case ClusterOpDestroyClusterPeerConnectionCommand:
		return &DestroyClusterPeerConnectionCommand{}, nil
	case ClusterOpGetMinorBlockRequest:
		return &GetMinorBlockRequest{}, nil
	case ClusterOpGetMinorBlockResponse:
		return &GetMinorBlockResponse{}, nil
	case ClusterOpGetTransactionRequest:
		return &GetTransactionRequest{}, nil
	case ClusterOpGetTransactionResponse:
		return &GetTransactionResponse{}, nil
	case ClusterOpBatchAddXshardTxListRequest:
		return &BatchAddXshardTxListRequest{}, nil
	case ClusterOpBatchAddXshardTxListResponse:
		return &BatchAddXshardTxListResponse{}, nil
	case ClusterOpExecuteTransactionRequest:
		return &ExecuteTransactionRequest{}, nil
	case ClusterOpExecuteTransactionResponse:
		return &ExecuteTransactionResponse{}, nil
	case ClusterOpGetTransactionReceiptRequest:
		return &GetTransactionReceiptRequest{}, nil
	case ClusterOpGetTransactionReceiptResponse:
		return &GetTransactionReceiptResponse{}, nil
	case ClusterOpMineRequest:
		return &MineRequest{}, nil
	case ClusterOpMineResponse:
		return &MineResponse{}, nil
	case ClusterOpGenTxRequest:
		return &GenTxRequest{}, nil
	case ClusterOpGenTxResponse:
		return &GenTxResponse{}, nil
	case ClusterOpGetTransactionListByAddressRequest:
		return &GetTransactionListByAddressRequest{}, nil
	case ClusterOpGetTransactionListByAddressResponse:
		return &GetTransactionListByAddressResponse{}, nil
	case ClusterOpGetLogRequest:
		return &GetLogRequest{}, nil
	case ClusterOpGetLogResponse:
		return &GetLogResponse{}, nil
	case ClusterOpEstimateGasRequest:
		return &EstimateGasRequest{}, nil
	case ClusterOpEstimateGasResponse:
		return &EstimateGasResponse{}, nil
	case ClusterOpGetStorageRequest:
		return &GetStorageRequest{}, nil
	case ClusterOpGetStorageResponse:
		return &GetStorageResponse{}, nil
	case ClusterOpGetCodeRequest:
		return &GetCodeRequest{}, nil
	case ClusterOpGetCodeResponse:
		return &GetCodeResponse{}, nil
	case ClusterOpGasPriceRequest:
		return &GasPriceRequest{}, nil
	case ClusterOpGasPriceResponse:
		return &GasPriceResponse{}, nil
	case ClusterOpGetWorkRequest:
		return &GetWorkRequest{}, nil
	case ClusterOpGetWorkResponse:
		return &GetWorkResponse{}, nil
	case ClusterOpSubmitWorkRequest:
		return &SubmitWorkRequest{}, nil
	case ClusterOpSubmitWorkResponse:
		return &SubmitWorkResponse{}, nil
	case ClusterOpAddMinorBlockHeaderListRequest:
		return &AddMinorBlockHeaderListRequest{}, nil
	case ClusterOpAddMinorBlockHeaderListResponse:
		return &AddMinorBlockHeaderListResponse{}, nil
	case ClusterOpCheckMinorBlockRequest:
		return &CheckMinorBlockRequest{}, nil
	case ClusterOpCheckMinorBlockResponse:
		return &CheckMinorBlockResponse{}, nil
	case ClusterOpGetAllTransactionsRequest:
		return &GetAllTransactionsRequest{}, nil
	case ClusterOpGetAllTransactionsResponse:
		return &GetAllTransactionsResponse{}, nil
	case ClusterOpGetRootChainStakesRequest:
		return &GetRootChainStakesRequest{}, nil
	case ClusterOpGetRootChainStakesResponse:
		return &GetRootChainStakesResponse{}, nil
	case ClusterOpGetTotalBalanceRequest:
		return &GetTotalBalanceRequest{}, nil
	case ClusterOpGetTotalBalanceResponse:
		return &GetTotalBalanceResponse{}, nil
	default:
		return nil, fmt.Errorf("unknown ClusterOp: 0x%x", op)
	}
}

// NewCommandMessage returns a new empty message struct pointer corresponding
// to the given CommandOp.  Note: both PING and PONG share PingPongCommand —
// this matches the Python implementation where the same class is registered
// for both opcodes (see p2p_commands.py: REGISTER_OP_TO_SERIALIZER).
func NewCommandMessage(op CommandOp) (interface{}, error) {
	switch op {
	case CommandOpHello:
		return &HelloCommand{}, nil
	case CommandOpNewMinorBlockHeaderList:
		return &NewMinorBlockHeaderListCommand{}, nil
	case CommandOpNewTransactionList:
		return &NewTransactionListCommand{}, nil
	case CommandOpGetPeerListRequest:
		return &GetPeerListRequest{}, nil
	case CommandOpGetPeerListResponse:
		return &GetPeerListResponse{}, nil
	case CommandOpGetRootBlockHeaderListRequest:
		return &GetRootBlockHeaderListRequest{}, nil
	case CommandOpGetRootBlockHeaderListResponse:
		return &GetRootBlockHeaderListResponse{}, nil
	case CommandOpGetRootBlockListRequest:
		return &GetRootBlockListRequest{}, nil
	case CommandOpGetRootBlockListResponse:
		return &GetRootBlockListResponse{}, nil
	case CommandOpGetMinorBlockListRequest:
		return &GetMinorBlockListRequest{}, nil
	case CommandOpGetMinorBlockListResponse:
		return &GetMinorBlockListResponse{}, nil
	case CommandOpGetMinorBlockHeaderListRequest:
		return &GetMinorBlockHeaderListRequest{}, nil
	case CommandOpGetMinorBlockHeaderListResponse:
		return &GetMinorBlockHeaderListResponse{}, nil
	case CommandOpNewBlockMinor:
		return &NewBlockMinorCommand{}, nil
	case CommandOpPing, CommandOpPong:
		return &PingPongCommand{}, nil
	case CommandOpGetRootBlockHeaderListWithSkipRequest:
		return &GetRootBlockHeaderListWithSkipRequest{}, nil
	case CommandOpGetRootBlockHeaderListWithSkipResponse:
		// Shares GetRootBlockHeaderListResponse with CommandOpGetRootBlockHeaderListResponse (0x06).
		// Python's REGISTER_OP_TO_SERIALIZER also maps both opcodes (0x06, 0x11) to the same class.
		// Verified: p2p_commands.py line 357.
		return &GetRootBlockHeaderListResponse{}, nil
	case CommandOpNewRootBlock:
		return &NewRootBlockCommand{}, nil
	case CommandOpGetMinorBlockHeaderListWithSkipRequest:
		return &GetMinorBlockHeaderListWithSkipRequest{}, nil
	case CommandOpGetMinorBlockHeaderListWithSkipResponse:
		// Shares GetMinorBlockHeaderListResponse with CommandOpGetMinorBlockHeaderListResponse (0x0C).
		// Python's REGISTER_OP_TO_SERIALIZER also maps both opcodes (0x0C, 0x14) to the same class.
		// Verified: p2p_commands.py line 360.
		return &GetMinorBlockHeaderListResponse{}, nil
	default:
		return nil, fmt.Errorf("unknown CommandOp: 0x%x", op)
	}
}
