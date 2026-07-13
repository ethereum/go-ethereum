// Copyright 2026-2027, QuarkChain.

package slave

import (
	"context"
	"fmt"
	"io"
	"net"

	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/qkc/cluster/wire"
	"github.com/ethereum/go-ethereum/qkc/serialize"
)

// MasterConn represents the slave-side TCP connection to the cluster master.
// It corresponds to Python's quarkchain.cluster.slave.MasterConnection and uses
// 12-byte ClusterMetadata framing.
//
// Architecture:
//
//	MasterConn embeds *rpcConn
//
// All master→slave ClusterOp handlers are registered during construction.
// Business handlers that depend on unported components (Shard, StateDB, etc.)
// are implemented as protocol-compatible stubs that return valid responses.
type MasterConn struct {
	*rpcConn

	localID              []byte
	localFullShardIDList []uint32
}

// NewMasterConn dials the master at addr and returns a MasterConn.
// maxPayloadSize controls frame payload size limit; 0 disables the limit.
// localID and localFullShardIDList identify this slave and are used in PONG.
func NewMasterConn(addr string, maxPayloadSize uint32, localID []byte, localFullShardIDList []uint32, logger log.Logger) (*MasterConn, error) {
	conn, err := net.DialTimeout("tcp", addr, defaultDialTimeout)
	if err != nil {
		return nil, fmt.Errorf("dial master %s: %w", addr, err)
	}
	return newMasterConn(conn, maxPayloadSize, localID, localFullShardIDList, logger), nil
}

// NewMasterConnFromConn wraps an accepted net.Conn as a MasterConn.
// maxPayloadSize controls frame payload size limit; 0 disables the limit.
func NewMasterConnFromConn(conn net.Conn, maxPayloadSize uint32, localID []byte, localFullShardIDList []uint32, logger log.Logger) *MasterConn {
	return newMasterConn(conn, maxPayloadSize, localID, localFullShardIDList, logger)
}

func newMasterConn(conn net.Conn, maxPayloadSize uint32, localID []byte, localFullShardIDList []uint32, logger log.Logger) *MasterConn {
	readFrame := func(r io.Reader) (*wire.Frame, error) {
		return wire.ReadFrame(r, maxPayloadSize)
	}
	mc := &MasterConn{
		rpcConn:              newRPCConn(conn, readFrame, wire.WriteFrame, logger),
		localID:              append([]byte(nil), localID...),
		localFullShardIDList: append([]uint32(nil), localFullShardIDList...),
	}

	mc.registerOpSerializers()
	mc.registerHandlers()

	return mc
}

// registerOpSerializers registers serializers for every opcode in Python's
// CLUSTER_OP_SERIALIZER_MAP. This covers master→slave, slave→master and
// slave→slave opcodes so outbound RPC responses can be deserialized if needed.
func (mc *MasterConn) registerOpSerializers() {
	mc.rpcConn.RegisterOpSerializers(map[byte]*OpSerializer{
		// §1 Cluster initialisation
		byte(wire.ClusterOpPing):                          OpSerializerFor[wire.PingRequest, wire.PongResponse](),
		byte(wire.ClusterOpPong):                          OpSerializerFor[wire.PongResponse, wire.PingRequest](),
		byte(wire.ClusterOpConnectToSlavesRequest):        OpSerializerFor[wire.ConnectToSlavesRequest, wire.ConnectToSlavesResponse](),
		byte(wire.ClusterOpConnectToSlavesResponse):       OpSerializerFor[wire.ConnectToSlavesResponse, wire.ConnectToSlavesRequest](),
		byte(wire.ClusterOpAddRootBlockRequest):           OpSerializerFor[wire.AddRootBlockRequest, wire.AddRootBlockResponse](),
		byte(wire.ClusterOpAddRootBlockResponse):          OpSerializerFor[wire.AddRootBlockResponse, wire.AddRootBlockRequest](),
		byte(wire.ClusterOpGetEcoInfoListRequest):         OpSerializerFor[wire.GetEcoInfoListRequest, wire.GetEcoInfoListResponse](),
		byte(wire.ClusterOpGetEcoInfoListResponse):        OpSerializerFor[wire.GetEcoInfoListResponse, wire.GetEcoInfoListRequest](),
		byte(wire.ClusterOpGetNextBlockToMineRequest):     OpSerializerFor[wire.GetNextBlockToMineRequest, wire.GetNextBlockToMineResponse](),
		byte(wire.ClusterOpGetNextBlockToMineResponse):    OpSerializerFor[wire.GetNextBlockToMineResponse, wire.GetNextBlockToMineRequest](),
		byte(wire.ClusterOpGetUnconfirmedHeadersRequest):  OpSerializerFor[wire.GetUnconfirmedHeadersRequest, wire.GetUnconfirmedHeadersResponse](),
		byte(wire.ClusterOpGetUnconfirmedHeadersResponse): OpSerializerFor[wire.GetUnconfirmedHeadersResponse, wire.GetUnconfirmedHeadersRequest](),
		byte(wire.ClusterOpGetAccountDataRequest):         OpSerializerFor[wire.GetAccountDataRequest, wire.GetAccountDataResponse](),
		byte(wire.ClusterOpGetAccountDataResponse):        OpSerializerFor[wire.GetAccountDataResponse, wire.GetAccountDataRequest](),
		byte(wire.ClusterOpAddTransactionRequest):         OpSerializerFor[wire.AddTransactionRequest, wire.AddTransactionResponse](),
		byte(wire.ClusterOpAddTransactionResponse):        OpSerializerFor[wire.AddTransactionResponse, wire.AddTransactionRequest](),

		// §2 Slave → Master (mining)
		byte(wire.ClusterOpAddMinorBlockHeaderRequest):  OpSerializerFor[wire.AddMinorBlockHeaderRequest, wire.AddMinorBlockHeaderResponse](),
		byte(wire.ClusterOpAddMinorBlockHeaderResponse): OpSerializerFor[wire.AddMinorBlockHeaderResponse, wire.AddMinorBlockHeaderRequest](),

		// §3 Slave ↔ Slave (xshard direct)
		byte(wire.ClusterOpAddXshardTxListRequest):  OpSerializerFor[wire.AddXshardTxListRequest, wire.AddXshardTxListResponse](),
		byte(wire.ClusterOpAddXshardTxListResponse): OpSerializerFor[wire.AddXshardTxListResponse, wire.AddXshardTxListRequest](),

		// §4 Master → Slave (sync / virtual conns)
		byte(wire.ClusterOpSyncMinorBlockListRequest):           OpSerializerFor[wire.SyncMinorBlockListRequest, wire.SyncMinorBlockListResponse](),
		byte(wire.ClusterOpSyncMinorBlockListResponse):          OpSerializerFor[wire.SyncMinorBlockListResponse, wire.SyncMinorBlockListRequest](),
		byte(wire.ClusterOpAddMinorBlockRequest):                OpSerializerFor[wire.AddMinorBlockRequest, wire.AddMinorBlockResponse](),
		byte(wire.ClusterOpAddMinorBlockResponse):               OpSerializerFor[wire.AddMinorBlockResponse, wire.AddMinorBlockRequest](),
		byte(wire.ClusterOpCreateClusterPeerConnectionRequest):  OpSerializerFor[wire.CreateClusterPeerConnectionRequest, wire.CreateClusterPeerConnectionResponse](),
		byte(wire.ClusterOpCreateClusterPeerConnectionResponse): OpSerializerFor[wire.CreateClusterPeerConnectionResponse, wire.CreateClusterPeerConnectionRequest](),
		byte(wire.ClusterOpDestroyClusterPeerConnectionCommand): OpSerializerFor[wire.DestroyClusterPeerConnectionCommand, wire.DestroyClusterPeerConnectionCommand](),
		byte(wire.ClusterOpGetMinorBlockRequest):                OpSerializerFor[wire.GetMinorBlockRequest, wire.GetMinorBlockResponse](),
		byte(wire.ClusterOpGetMinorBlockResponse):               OpSerializerFor[wire.GetMinorBlockResponse, wire.GetMinorBlockRequest](),
		byte(wire.ClusterOpGetTransactionRequest):               OpSerializerFor[wire.GetTransactionRequest, wire.GetTransactionResponse](),
		byte(wire.ClusterOpGetTransactionResponse):              OpSerializerFor[wire.GetTransactionResponse, wire.GetTransactionRequest](),

		// §5 Slave ↔ Slave (xshard batch)
		byte(wire.ClusterOpBatchAddXshardTxListRequest):  OpSerializerFor[wire.BatchAddXshardTxListRequest, wire.BatchAddXshardTxListResponse](),
		byte(wire.ClusterOpBatchAddXshardTxListResponse): OpSerializerFor[wire.BatchAddXshardTxListResponse, wire.BatchAddXshardTxListRequest](),

		// §6 Master → Slave (JSON-RPC-like)
		byte(wire.ClusterOpExecuteTransactionRequest):           OpSerializerFor[wire.ExecuteTransactionRequest, wire.ExecuteTransactionResponse](),
		byte(wire.ClusterOpExecuteTransactionResponse):          OpSerializerFor[wire.ExecuteTransactionResponse, wire.ExecuteTransactionRequest](),
		byte(wire.ClusterOpGetTransactionReceiptRequest):        OpSerializerFor[wire.GetTransactionReceiptRequest, wire.GetTransactionReceiptResponse](),
		byte(wire.ClusterOpGetTransactionReceiptResponse):       OpSerializerFor[wire.GetTransactionReceiptResponse, wire.GetTransactionReceiptRequest](),
		byte(wire.ClusterOpMineRequest):                         OpSerializerFor[wire.MineRequest, wire.MineResponse](),
		byte(wire.ClusterOpMineResponse):                        OpSerializerFor[wire.MineResponse, wire.MineRequest](),
		byte(wire.ClusterOpGenTxRequest):                        OpSerializerFor[wire.GenTxRequest, wire.GenTxResponse](),
		byte(wire.ClusterOpGenTxResponse):                       OpSerializerFor[wire.GenTxResponse, wire.GenTxRequest](),
		byte(wire.ClusterOpGetTransactionListByAddressRequest):  OpSerializerFor[wire.GetTransactionListByAddressRequest, wire.GetTransactionListByAddressResponse](),
		byte(wire.ClusterOpGetTransactionListByAddressResponse): OpSerializerFor[wire.GetTransactionListByAddressResponse, wire.GetTransactionListByAddressRequest](),
		byte(wire.ClusterOpGetLogRequest):                       OpSerializerFor[wire.GetLogRequest, wire.GetLogResponse](),
		byte(wire.ClusterOpGetLogResponse):                      OpSerializerFor[wire.GetLogResponse, wire.GetLogRequest](),
		byte(wire.ClusterOpEstimateGasRequest):                  OpSerializerFor[wire.EstimateGasRequest, wire.EstimateGasResponse](),
		byte(wire.ClusterOpEstimateGasResponse):                 OpSerializerFor[wire.EstimateGasResponse, wire.EstimateGasRequest](),
		byte(wire.ClusterOpGetStorageRequest):                   OpSerializerFor[wire.GetStorageRequest, wire.GetStorageResponse](),
		byte(wire.ClusterOpGetStorageResponse):                  OpSerializerFor[wire.GetStorageResponse, wire.GetStorageRequest](),
		byte(wire.ClusterOpGetCodeRequest):                      OpSerializerFor[wire.GetCodeRequest, wire.GetCodeResponse](),
		byte(wire.ClusterOpGetCodeResponse):                     OpSerializerFor[wire.GetCodeResponse, wire.GetCodeRequest](),
		byte(wire.ClusterOpGasPriceRequest):                     OpSerializerFor[wire.GasPriceRequest, wire.GasPriceResponse](),
		byte(wire.ClusterOpGasPriceResponse):                    OpSerializerFor[wire.GasPriceResponse, wire.GasPriceRequest](),
		byte(wire.ClusterOpGetWorkRequest):                      OpSerializerFor[wire.GetWorkRequest, wire.GetWorkResponse](),
		byte(wire.ClusterOpGetWorkResponse):                     OpSerializerFor[wire.GetWorkResponse, wire.GetWorkRequest](),
		byte(wire.ClusterOpSubmitWorkRequest):                   OpSerializerFor[wire.SubmitWorkRequest, wire.SubmitWorkResponse](),
		byte(wire.ClusterOpSubmitWorkResponse):                  OpSerializerFor[wire.SubmitWorkResponse, wire.SubmitWorkRequest](),

		// §7 Slave → Master (block list)
		byte(wire.ClusterOpAddMinorBlockHeaderListRequest):  OpSerializerFor[wire.AddMinorBlockHeaderListRequest, wire.AddMinorBlockHeaderListResponse](),
		byte(wire.ClusterOpAddMinorBlockHeaderListResponse): OpSerializerFor[wire.AddMinorBlockHeaderListResponse, wire.AddMinorBlockHeaderListRequest](),

		// §8 Master → Slave (JRPC & staking)
		byte(wire.ClusterOpCheckMinorBlockRequest):     OpSerializerFor[wire.CheckMinorBlockRequest, wire.CheckMinorBlockResponse](),
		byte(wire.ClusterOpCheckMinorBlockResponse):    OpSerializerFor[wire.CheckMinorBlockResponse, wire.CheckMinorBlockRequest](),
		byte(wire.ClusterOpGetAllTransactionsRequest):  OpSerializerFor[wire.GetAllTransactionsRequest, wire.GetAllTransactionsResponse](),
		byte(wire.ClusterOpGetAllTransactionsResponse): OpSerializerFor[wire.GetAllTransactionsResponse, wire.GetAllTransactionsRequest](),
		byte(wire.ClusterOpGetRootChainStakesRequest):  OpSerializerFor[wire.GetRootChainStakesRequest, wire.GetRootChainStakesResponse](),
		byte(wire.ClusterOpGetRootChainStakesResponse): OpSerializerFor[wire.GetRootChainStakesResponse, wire.GetRootChainStakesRequest](),
		byte(wire.ClusterOpGetTotalBalanceRequest):     OpSerializerFor[wire.GetTotalBalanceRequest, wire.GetTotalBalanceResponse](),
		byte(wire.ClusterOpGetTotalBalanceResponse):    OpSerializerFor[wire.GetTotalBalanceResponse, wire.GetTotalBalanceRequest](),
	})
}

// registerHandlers registers all master→slave RPC handlers and marks the
// fire-and-forget opcodes as non-RPC.
func (mc *MasterConn) registerHandlers() {
	mc.rpcConn.RegisterTypedHandlers(map[byte]TypedHandler{
		byte(wire.ClusterOpPing):                                mc.handlePing,
		byte(wire.ClusterOpConnectToSlavesRequest):              mc.handleConnectToSlaves,
		byte(wire.ClusterOpMineRequest):                         mc.handleMine,
		byte(wire.ClusterOpGenTxRequest):                        mc.handleGenTx,
		byte(wire.ClusterOpAddRootBlockRequest):                 mc.handleAddRootBlock,
		byte(wire.ClusterOpGetEcoInfoListRequest):               mc.handleGetEcoInfoList,
		byte(wire.ClusterOpGetNextBlockToMineRequest):           mc.handleGetNextBlockToMine,
		byte(wire.ClusterOpAddMinorBlockRequest):                mc.handleAddMinorBlock,
		byte(wire.ClusterOpGetUnconfirmedHeadersRequest):        mc.handleGetUnconfirmedHeaders,
		byte(wire.ClusterOpGetAccountDataRequest):               mc.handleGetAccountData,
		byte(wire.ClusterOpAddTransactionRequest):               mc.handleAddTransaction,
		byte(wire.ClusterOpCreateClusterPeerConnectionRequest):  mc.handleCreateClusterPeerConnection,
		byte(wire.ClusterOpDestroyClusterPeerConnectionCommand): mc.handleDestroyClusterPeerConnection,
		byte(wire.ClusterOpGetMinorBlockRequest):                mc.handleGetMinorBlock,
		byte(wire.ClusterOpGetTransactionRequest):               mc.handleGetTransaction,
		byte(wire.ClusterOpSyncMinorBlockListRequest):           mc.handleSyncMinorBlockList,
		byte(wire.ClusterOpExecuteTransactionRequest):           mc.handleExecuteTransaction,
		byte(wire.ClusterOpGetTransactionReceiptRequest):        mc.handleGetTransactionReceipt,
		byte(wire.ClusterOpGetTransactionListByAddressRequest):  mc.handleGetTransactionListByAddress,
		byte(wire.ClusterOpGetLogRequest):                       mc.handleGetLogs,
		byte(wire.ClusterOpEstimateGasRequest):                  mc.handleEstimateGas,
		byte(wire.ClusterOpGetStorageRequest):                   mc.handleGetStorageAt,
		byte(wire.ClusterOpGetCodeRequest):                      mc.handleGetCode,
		byte(wire.ClusterOpGasPriceRequest):                     mc.handleGasPrice,
		byte(wire.ClusterOpGetWorkRequest):                      mc.handleGetWork,
		byte(wire.ClusterOpSubmitWorkRequest):                   mc.handleSubmitWork,
		byte(wire.ClusterOpCheckMinorBlockRequest):              mc.handleCheckMinorBlock,
		byte(wire.ClusterOpGetAllTransactionsRequest):           mc.handleGetAllTransactions,
		byte(wire.ClusterOpGetRootChainStakesRequest):           mc.handleGetRootChainStakes,
		byte(wire.ClusterOpGetTotalBalanceRequest):              mc.handleGetTotalBalance,
	})

	mc.rpcConn.RegisterNonRPCOps([]byte{
		byte(wire.ClusterOpDestroyClusterPeerConnectionCommand),
	})
}

// rawBytes is a helper that returns a non-nil *wire.RawBytes pointer.
func rawBytes(b []byte) *wire.RawBytes {
	rb := wire.RawBytes(b)
	return &rb
}

// emptyRawBytes returns a non-nil *wire.RawBytes pointing to an empty slice.
func emptyRawBytes() *wire.RawBytes {
	return rawBytes([]byte{})
}

// LocalID returns this slave's ID used in PONG responses.
func (mc *MasterConn) LocalID() []byte {
	return append([]byte(nil), mc.localID...)
}

// LocalFullShardIDList returns this slave's full shard ID list used in PONG responses.
func (mc *MasterConn) LocalFullShardIDList() []uint32 {
	return append([]uint32(nil), mc.localFullShardIDList...)
}

// handlePing responds to the master's PING with this slave's identity.
// Python: MasterConnection.handle_ping -> Pong(self.slave_server.id, ...).
func (mc *MasterConn) handlePing(req any) (any, error) {
	// TODO: when core.RootBlock is ported, use ping.root_tip to drive shard creation.
	_ = req.(*wire.PingRequest)

	return &wire.PongResponse{
		ID:              append([]byte(nil), mc.localID...),
		FullShardIDList: append([]uint32(nil), mc.localFullShardIDList...),
	}, nil
}

// handleConnectToSlaves accepts a list of slaves to connect to.
// Python: returns ConnectToSlavesResponse with one empty bytes result per slave.
func (mc *MasterConn) handleConnectToSlaves(req any) (any, error) {
	r := req.(*wire.ConnectToSlavesRequest)

	// TODO: delegate to SlaveServer.slave_connection_manager.connect_to_slave.
	resultList := make([]wire.PrependedSizeBytes4, len(r.SlaveInfoList))
	for i := range resultList {
		resultList[i] = wire.PrependedSizeBytes4{}
	}
	return &wire.ConnectToSlavesResponse{ResultList: resultList}, nil
}

// handleMine starts or stops mining.
// Python: MineResponse(error_code=0).
func (mc *MasterConn) handleMine(req any) (any, error) {
	_ = req.(*wire.MineRequest)
	// TODO: delegate to SlaveServer.start_mining / stop_mining.
	return &wire.MineResponse{ErrorCode: 0}, nil
}

// handleGenTx generates transactions.
// Python: GenTxResponse(error_code=0).
func (mc *MasterConn) handleGenTx(req any) (any, error) {
	_ = req.(*wire.GenTxRequest)
	// TODO: delegate to SlaveServer.create_transactions.
	return &wire.GenTxResponse{ErrorCode: 0}, nil
}

// handleAddRootBlock processes a root block from the master.
// Python: returns AddRootBlockResponse(error_code=0, switched=False) on success.
func (mc *MasterConn) handleAddRootBlock(req any) (any, error) {
	_ = req.(*wire.AddRootBlockRequest)
	// TODO: delegate to shard.add_root_block and SlaveServer.create_shards.
	return &wire.AddRootBlockResponse{ErrorCode: 0, Switched: false}, nil
}

// handleGetEcoInfoList returns economic info for all initialized shards.
// Python: returns empty list when no shards are initialized.
func (mc *MasterConn) handleGetEcoInfoList(req any) (any, error) {
	_ = req.(*wire.GetEcoInfoListRequest)
	// TODO: collect real EcoInfo from shard states.
	return &wire.GetEcoInfoListResponse{ErrorCode: 0, EcoInfoList: []wire.EcoInfo{}}, nil
}

// handleGetNextBlockToMine returns a block template for the requested branch.
// Python requires the shard to exist; without shard runtime we return not-found.
func (mc *MasterConn) handleGetNextBlockToMine(req any) (any, error) {
	_ = req.(*wire.GetNextBlockToMineRequest)
	// TODO: delegate to shard.state.create_block_to_mine.
	return &wire.GetNextBlockToMineResponse{ErrorCode: 1, Block: emptyRawBytes()}, nil
}

// handleAddMinorBlock adds a JRPC-mined minor block.
// Python: returns AddMinorBlockResponse(error_code=0) on success.
func (mc *MasterConn) handleAddMinorBlock(req any) (any, error) {
	_ = req.(*wire.AddMinorBlockRequest)
	// TODO: deserialize MinorBlock and delegate to shard.add_block.
	return &wire.AddMinorBlockResponse{ErrorCode: 0}, nil
}

// handleGetUnconfirmedHeaders returns unconfirmed headers per shard.
// Python: returns empty list when no shards are initialized.
func (mc *MasterConn) handleGetUnconfirmedHeaders(req any) (any, error) {
	_ = req.(*wire.GetUnconfirmedHeadersRequest)
	// TODO: collect real HeadersInfo from shard states.
	return &wire.GetUnconfirmedHeadersResponse{ErrorCode: 0, HeadersInfoList: []wire.HeadersInfo{}}, nil
}

// handleGetAccountData returns account data across shards.
// Python: returns empty list when there are no shards for the address.
func (mc *MasterConn) handleGetAccountData(req any) (any, error) {
	_ = req.(*wire.GetAccountDataRequest)
	// TODO: delegate to SlaveServer.get_account_data.
	return &wire.GetAccountDataResponse{ErrorCode: 0, AccountBranchDataList: []wire.AccountBranchData{}}, nil
}

// handleAddTransaction adds a transaction to the tx pool.
// Python: returns AddTransactionResponse(error_code=0) on success.
func (mc *MasterConn) handleAddTransaction(req any) (any, error) {
	_ = req.(*wire.AddTransactionRequest)
	// TODO: delegate to SlaveServer.add_tx.
	return &wire.AddTransactionResponse{ErrorCode: 0}, nil
}

// handleCreateClusterPeerConnection creates virtual peer connections for all shards.
// Python: returns CreateClusterPeerConnectionResponse(error_code=0) on success.
func (mc *MasterConn) handleCreateClusterPeerConnection(req any) (any, error) {
	_ = req.(*wire.CreateClusterPeerConnectionRequest)
	// TODO: create PeerShardConnection instances and wire with the dispatcher (PR6).
	return &wire.CreateClusterPeerConnectionResponse{ErrorCode: 0}, nil
}

// handleDestroyClusterPeerConnection is a fire-and-forget command to tear down
// a virtual peer connection. No response is sent.
func (mc *MasterConn) handleDestroyClusterPeerConnection(req any) (any, error) {
	_ = req.(*wire.DestroyClusterPeerConnectionCommand)
	// TODO: notify dispatcher / close peer shard connections (PR6).
	return nil, nil
}

// handleGetMinorBlock fetches a minor block by hash or height.
// Python returns error_code=1 with an empty block when not found.
func (mc *MasterConn) handleGetMinorBlock(req any) (any, error) {
	_ = req.(*wire.GetMinorBlockRequest)
	// TODO: delegate to SlaveServer.get_minor_block_by_hash / by_height.
	return &wire.GetMinorBlockResponse{
		ErrorCode:  1,
		MinorBlock: emptyRawBytes(),
		ExtraInfo:  nil,
	}, nil
}

// handleGetTransaction fetches a transaction by hash.
// Python returns error_code=1 with an empty block when not found.
func (mc *MasterConn) handleGetTransaction(req any) (any, error) {
	_ = req.(*wire.GetTransactionRequest)
	// TODO: delegate to SlaveServer.get_transaction_by_hash.
	return &wire.GetTransactionResponse{
		ErrorCode:  1,
		MinorBlock: emptyRawBytes(),
		Index:      0,
	}, nil
}

// handleSyncMinorBlockList downloads and applies a list of minor blocks.
// Python returns error_code=0 with empty data when the input list is empty.
func (mc *MasterConn) handleSyncMinorBlockList(req any) (any, error) {
	r := req.(*wire.SyncMinorBlockListRequest)
	_ = r
	// TODO: delegate to SlaveServer.add_block_list_for_sync.
	return &wire.SyncMinorBlockListResponse{
		ErrorCode:        0,
		BlockCoinbaseMap: emptyRawBytes(),
		ShardStats:       nil,
	}, nil
}

// handleExecuteTransaction executes a transaction and returns the result.
// Python returns error_code=1 when execution fails (e.g. shard missing).
func (mc *MasterConn) handleExecuteTransaction(req any) (any, error) {
	_ = req.(*wire.ExecuteTransactionRequest)
	// TODO: delegate to SlaveServer.execute_tx.
	return &wire.ExecuteTransactionResponse{ErrorCode: 1, Result: []byte{}}, nil
}

// handleGetTransactionReceipt fetches a transaction receipt.
// Python returns error_code=1 with empty block/receipt when not found.
func (mc *MasterConn) handleGetTransactionReceipt(req any) (any, error) {
	_ = req.(*wire.GetTransactionReceiptRequest)
	// TODO: delegate to SlaveServer.get_transaction_receipt.
	return &wire.GetTransactionReceiptResponse{
		ErrorCode:  1,
		MinorBlock: emptyRawBytes(),
		Index:      0,
		Receipt:    emptyRawBytes(),
	}, nil
}

// handleGetTransactionListByAddress returns transactions for an address.
// Python returns error_code=1 with empty lists when the shard is missing.
func (mc *MasterConn) handleGetTransactionListByAddress(req any) (any, error) {
	_ = req.(*wire.GetTransactionListByAddressRequest)
	// TODO: delegate to SlaveServer.get_transaction_list_by_address.
	return &wire.GetTransactionListByAddressResponse{
		ErrorCode: 1,
		TxList:    []wire.TransactionDetail{},
		Next:      []byte{},
	}, nil
}

// handleGetLogs returns logs matching the filter.
// Python returns error_code=1 with empty logs when the shard is missing.
func (mc *MasterConn) handleGetLogs(req any) (any, error) {
	_ = req.(*wire.GetLogRequest)
	// TODO: delegate to SlaveServer.get_logs.
	return &wire.GetLogResponse{ErrorCode: 1, Logs: []*wire.RawBytes{}}, nil
}

// handleEstimateGas estimates gas for a transaction.
// Python returns error_code=1 when estimation fails (e.g. shard missing).
func (mc *MasterConn) handleEstimateGas(req any) (any, error) {
	_ = req.(*wire.EstimateGasRequest)
	// TODO: delegate to SlaveServer.estimate_gas.
	return &wire.EstimateGasResponse{ErrorCode: 1, Result: 0}, nil
}

// handleGetStorageAt reads storage at the given address/key.
// Python returns error_code=1 with a zero result when the shard is missing.
func (mc *MasterConn) handleGetStorageAt(req any) (any, error) {
	_ = req.(*wire.GetStorageRequest)
	// TODO: delegate to SlaveServer.get_storage_at.
	return &wire.GetStorageResponse{ErrorCode: 1, Result: [wire.HashLength]byte{}}, nil
}

// handleGetCode reads code at the given address.
// Python returns error_code=1 with empty bytes when the shard is missing.
func (mc *MasterConn) handleGetCode(req any) (any, error) {
	_ = req.(*wire.GetCodeRequest)
	// TODO: delegate to SlaveServer.get_code.
	return &wire.GetCodeResponse{ErrorCode: 1, Result: []byte{}}, nil
}

// handleGasPrice returns the gas price for a token on a branch.
// Python returns error_code=1 with result 0 when the shard is missing.
func (mc *MasterConn) handleGasPrice(req any) (any, error) {
	_ = req.(*wire.GasPriceRequest)
	// TODO: delegate to SlaveServer.gas_price.
	return &wire.GasPriceResponse{ErrorCode: 1, Result: 0}, nil
}

// handleGetWork returns mining work.
// Python returns error_code=1 when work cannot be produced.
func (mc *MasterConn) handleGetWork(req any) (any, error) {
	_ = req.(*wire.GetWorkRequest)
	// TODO: delegate to SlaveServer.get_work.
	return &wire.GetWorkResponse{ErrorCode: 1}, nil
}

// handleSubmitWork submits mining work.
// Python returns error_code=1, success=False when submission fails.
func (mc *MasterConn) handleSubmitWork(req any) (any, error) {
	_ = req.(*wire.SubmitWorkRequest)
	// TODO: delegate to SlaveServer.submit_work.
	return &wire.SubmitWorkResponse{ErrorCode: 1, Success: false}, nil
}

// handleCheckMinorBlock validates a minor block header.
// Python returns CheckMinorBlockResponse(error_code=0) when the block is valid,
// and error_code=errno.EBADMSG when the shard is missing or validation fails.
// This stub returns ErrorCode=1 to signal "not implemented / cannot validate".
func (mc *MasterConn) handleCheckMinorBlock(req any) (any, error) {
	_ = req.(*wire.CheckMinorBlockRequest)
	// TODO: delegate to shard.check_minor_block_by_header.
	return &wire.CheckMinorBlockResponse{ErrorCode: 1}, nil
}

// handleGetAllTransactions returns all transactions in the mempool.
// Python returns error_code=1 with empty lists when the shard is missing.
func (mc *MasterConn) handleGetAllTransactions(req any) (any, error) {
	_ = req.(*wire.GetAllTransactionsRequest)
	// TODO: delegate to SlaveServer.get_all_transactions.
	return &wire.GetAllTransactionsResponse{
		ErrorCode: 1,
		TxList:    []wire.TransactionDetail{},
		Next:      []byte{},
	}, nil
}

// handleGetRootChainStakes reads root-chain stake info.
// Python returns GetRootChainStakesResponse(0, stakes, signer).
func (mc *MasterConn) handleGetRootChainStakes(req any) (any, error) {
	_ = req.(*wire.GetRootChainStakesRequest)
	// TODO: delegate to SlaveServer.get_root_chain_stakes.
	return &wire.GetRootChainStakesResponse{
		ErrorCode: 0,
		Stakes:    serialize.BigUint{},
		Signer:    [20]byte{},
	}, nil
}

// handleGetTotalBalance returns the total token balance across accounts.
// Python catches exceptions and returns GetTotalBalanceResponse(1, 0, b"").
func (mc *MasterConn) handleGetTotalBalance(req any) (any, error) {
	_ = req.(*wire.GetTotalBalanceRequest)
	// TODO: delegate to SlaveServer.get_total_balance.
	return &wire.GetTotalBalanceResponse{
		ErrorCode:    1,
		TotalBalance: serialize.BigUint{},
		Next:         []byte{},
	}, nil
}

// SetForwarder installs a raw-frame forwarder hook for peer traffic
// (cluster_peer_id != 0). This is used by the dispatcher in PR6.
func (mc *MasterConn) SetForwarder(f func(*wire.Frame) bool) {
	mc.rpcConn.SetForwarder(f)
}

// SendRPCMeta sends a request with ClusterMetadata and waits for the response.
// It is the primitive used by all typed outbound methods.
func (mc *MasterConn) SendRPCMeta(ctx context.Context, opcode byte, payload []byte, meta wire.ClusterMetadata) (*wire.Frame, error) {
	return mc.rpcConn.SendRPCMeta(ctx, opcode, payload, meta)
}

// SendAddMinorBlockHeader sends AddMinorBlockHeaderRequest to the master and
// returns the parsed response.
func (mc *MasterConn) SendAddMinorBlockHeader(ctx context.Context, req *wire.AddMinorBlockHeaderRequest) (*wire.AddMinorBlockHeaderResponse, error) {
	payload, err := serializeBytes(req)
	if err != nil {
		return nil, fmt.Errorf("serialize AddMinorBlockHeaderRequest: %w", err)
	}
	frame, err := mc.SendRPCMeta(ctx, byte(wire.ClusterOpAddMinorBlockHeaderRequest), payload, wire.ClusterMetadata{})
	if err != nil {
		return nil, err
	}
	var resp wire.AddMinorBlockHeaderResponse
	if err := deserializeBytes(frame.Payload, &resp); err != nil {
		return nil, fmt.Errorf("deserialize AddMinorBlockHeaderResponse: %w", err)
	}
	return &resp, nil
}

// SendAddMinorBlockHeaderList sends AddMinorBlockHeaderListRequest to the master
// and returns the parsed response.
func (mc *MasterConn) SendAddMinorBlockHeaderList(ctx context.Context, req *wire.AddMinorBlockHeaderListRequest) (*wire.AddMinorBlockHeaderListResponse, error) {
	payload, err := serializeBytes(req)
	if err != nil {
		return nil, fmt.Errorf("serialize AddMinorBlockHeaderListRequest: %w", err)
	}
	frame, err := mc.SendRPCMeta(ctx, byte(wire.ClusterOpAddMinorBlockHeaderListRequest), payload, wire.ClusterMetadata{})
	if err != nil {
		return nil, err
	}
	var resp wire.AddMinorBlockHeaderListResponse
	if err := deserializeBytes(frame.Payload, &resp); err != nil {
		return nil, fmt.Errorf("deserialize AddMinorBlockHeaderListResponse: %w", err)
	}
	return &resp, nil
}
