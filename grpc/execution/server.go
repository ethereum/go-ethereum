// Package execution provides the gRPC server for the execution layer.
//
// Its procedures will be called from the conductor. It is responsible
// for immediately executing lists of ordered transactions that come from the shared sequencer.
package execution

import (
	"context"
	"fmt"
	"time"

	"github.com/ethereum/go-ethereum/beacon/engine"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/eth"
	"github.com/ethereum/go-ethereum/eth/catalyst"
	executionv1 "github.com/ethereum/go-ethereum/grpc/gen/astria/execution/v1"
	"github.com/ethereum/go-ethereum/log"
)

// executionServiceServer is the implementation of the ExecutionServiceServer interface.
type ExecutionServiceServer struct {
	// NOTE - from the generated code:
	// All implementations must embed UnimplementedExecutionServiceServer
	// for forward compatibility
	executionv1.UnimplementedExecutionServiceServer

	consensus *catalyst.ConsensusAPI
	eth       *eth.Ethereum

	bc *core.BlockChain
}

func NewExecutionServiceServer(eth *eth.Ethereum) *ExecutionServiceServer {
	consensus := catalyst.NewConsensusAPI(eth)

	bc := eth.BlockChain()

	return &ExecutionServiceServer{
		eth:       eth,
		consensus: consensus,
		bc:        bc,
	}
}

func (s *ExecutionServiceServer) DoBlock(ctx context.Context, req *executionv1.DoBlockRequest) (*executionv1.DoBlockResponse, error) {
	log.Info("DoBlock called request", "request", req)
	prevHeadHash := common.BytesToHash(req.PrevStateRoot)

	// The Engine API has been modified to use transactions from this mempool and abide by it's ordering.
	s.eth.TxPool().SetAstriaOrdered(req.Transactions)

	// Do the whole Engine API in a single loop
	startForkChoice := &engine.ForkchoiceStateV1{
		HeadBlockHash:      prevHeadHash,
		SafeBlockHash:      prevHeadHash,
		FinalizedBlockHash: prevHeadHash,
	}
	payloadAttributes := &engine.PayloadAttributes{
		Timestamp:             uint64(req.GetTimestamp().GetSeconds()),
		Random:                common.Hash{},
		SuggestedFeeRecipient: common.Address{},
	}
	fcStartResp, err := s.consensus.ForkchoiceUpdatedV1(*startForkChoice, payloadAttributes)
	if err != nil {
		return nil, err
	}

	// super janky but this is what the payload builder requires :/ (miner.worker.buildPayload())
	// we should probably just execute + store the block directly instead of using the engine api.
	time.Sleep(time.Second)
	payloadResp, err := s.consensus.GetPayloadV1(*fcStartResp.PayloadID)
	if err != nil {
		log.Error("failed to call GetPayloadV1", "err", err)
		return nil, err
	}

	// call blockchain.InsertChain to actually execute and write the blocks to state
	block, err := engine.ExecutableDataToBlock(*payloadResp)
	if err != nil {
		return nil, err
	}
	blocks := types.Blocks{
		block,
	}
	n, err := s.bc.InsertChain(blocks)
	if err != nil {
		return nil, err
	}
	if n != 1 {
		return nil, fmt.Errorf("failed to insert block into blockchain (n=%d)", n)
	}

	// remove txs from original mempool
	for _, tx := range block.Transactions() {
		s.eth.TxPool().RemoveTx(tx.Hash())
	}

	finalizedBlock := s.bc.CurrentFinalBlock()
	newForkChoice := &engine.ForkchoiceStateV1{
		HeadBlockHash:      block.Hash(),
		SafeBlockHash:      block.Hash(),
		FinalizedBlockHash: finalizedBlock.Hash(),
	}
	fcEndResp, err := s.consensus.ForkchoiceUpdatedV1(*newForkChoice, nil)
	if err != nil {
		log.Error("failed to call ForkchoiceUpdatedV1", "err", err)
		return nil, err
	}

	res := &executionv1.DoBlockResponse{
		BlockHash: fcEndResp.PayloadStatus.LatestValidHash.Bytes(),
	}
	return res, nil
}

func (s *ExecutionServiceServer) FinalizeBlock(ctx context.Context, req *executionv1.FinalizeBlockRequest) (*executionv1.FinalizeBlockResponse, error) {
	header := s.bc.GetHeaderByHash(common.BytesToHash(req.BlockHash))
	if header == nil {
		return nil, fmt.Errorf("failed to get header for block hash 0x%x", req.BlockHash)
	}

	s.bc.SetFinalized(header)
	return &executionv1.FinalizeBlockResponse{}, nil
}

func (s *ExecutionServiceServer) InitState(ctx context.Context, req *executionv1.InitStateRequest) (*executionv1.InitStateResponse, error) {
	currHead := s.eth.BlockChain().CurrentHeader()
	res := &executionv1.InitStateResponse{
		BlockHash: currHead.Hash().Bytes(),
	}

	return res, nil
}
