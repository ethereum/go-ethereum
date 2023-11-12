// Package execution provides the gRPC server for the execution layer.
//
// Its procedures will be called from the conductor. It is responsible
// for immediately executing lists of ordered transactions that come from the shared sequencer.
package execution

import (
	"context"
	"fmt"

	"github.com/ethereum/go-ethereum/beacon/engine"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/eth"
	"github.com/ethereum/go-ethereum/eth/catalyst"
	executionv1a1 "github.com/ethereum/go-ethereum/grpc/gen/astria/execution/v1alpha1"
	executionv1a2 "github.com/ethereum/go-ethereum/grpc/gen/astria/execution/v1alpha2"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/miner"
	codes "google.golang.org/grpc/codes"
	status "google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// executionServiceServer is the implementation of the ExecutionServiceServerV1Alpha1 interface.
type ExecutionServiceServerV1Alpha1 struct {
	// NOTE - from the generated code:
	// All implementations must embed UnimplementedExecutionServiceServer
	// for forward compatibility
	executionv1a1.UnimplementedExecutionServiceServer

	consensus *catalyst.ConsensusAPI
	eth       *eth.Ethereum

	bc *core.BlockChain
}

func NewExecutionServiceServerV1Alpha1(eth *eth.Ethereum) *ExecutionServiceServerV1Alpha1 {
	consensus := catalyst.NewConsensusAPI(eth)

	bc := eth.BlockChain()

	return &ExecutionServiceServerV1Alpha1{
		eth:       eth,
		consensus: consensus,
		bc:        bc,
	}
}

func (s *ExecutionServiceServerV1Alpha1) DoBlock(ctx context.Context, req *executionv1a1.DoBlockRequest) (*executionv1a1.DoBlockResponse, error) {
	log.Info("DoBlock called request", "request", req)
	prevHeadHash := common.BytesToHash(req.PrevBlockHash)

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

	// TODO: we should probably just execute + store the block directly instead of using the engine api.
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

	res := &executionv1a1.DoBlockResponse{
		BlockHash: fcEndResp.PayloadStatus.LatestValidHash.Bytes(),
	}
	return res, nil
}

func (s *ExecutionServiceServerV1Alpha1) FinalizeBlock(ctx context.Context, req *executionv1a1.FinalizeBlockRequest) (*executionv1a1.FinalizeBlockResponse, error) {
	header := s.bc.GetHeaderByHash(common.BytesToHash(req.BlockHash))
	if header == nil {
		return nil, fmt.Errorf("failed to get header for block hash 0x%x", req.BlockHash)
	}

	s.bc.SetFinalized(header)
	return &executionv1a1.FinalizeBlockResponse{}, nil
}

func (s *ExecutionServiceServerV1Alpha1) InitState(ctx context.Context, req *executionv1a1.InitStateRequest) (*executionv1a1.InitStateResponse, error) {
	currHead := s.bc.CurrentHeader()
	res := &executionv1a1.InitStateResponse{
		BlockHash: currHead.Hash().Bytes(),
	}

	return res, nil
}

// ExecutionServiceServerV1Alpha2 is the implementation of the
// ExecutionServiceServer interface.
type ExecutionServiceServerV1Alpha2 struct {
	// NOTE - from the generated code: All implementations must embed
	// UnimplementedExecutionServiceServer for forward compatibility
	executionv1a2.UnimplementedExecutionServiceServer

	eth *eth.Ethereum
	bc  *core.BlockChain
}

func NewExecutionServiceServerV1Alpha2(eth *eth.Ethereum) *ExecutionServiceServerV1Alpha2 {
	bc := eth.BlockChain()

	if merger := eth.Merger(); !merger.PoSFinalized() {
		merger.FinalizePoS()
	}

	return &ExecutionServiceServerV1Alpha2{
		eth: eth,
		bc:  bc,
	}
}

// GetBlock will return a block given an identifier.
func (s *ExecutionServiceServerV1Alpha2) GetBlock(ctx context.Context, req *executionv1a2.GetBlockRequest) (*executionv1a2.Block, error) {
	log.Info("GetBlock called", "request", req)

	res, err := s.getBlockFromIdentifier(req.GetIdentifier())
	if err != nil {
		log.Error("failed finding block", err)
		return nil, err
	}

	log.Info("GetBlock completed", "request", req, "response", res)
	return res, nil
}

// BatchGetBlocks will return an array of Blocks given an array of block
// identifiers.
func (s *ExecutionServiceServerV1Alpha2) BatchGetBlocks(ctx context.Context, req *executionv1a2.BatchGetBlocksRequest) (*executionv1a2.BatchGetBlocksResponse, error) {
	log.Info("BatchGetBlocks called", "request", req)
	var blocks []*executionv1a2.Block

	ids := req.GetIdentifiers()
	for _, id := range ids {
		block, err := s.getBlockFromIdentifier(id)
		if err != nil {
			log.Error("failed finding block with id", id, "error", err)
			return nil, err
		}

		blocks = append(blocks, block)
	}

	res := &executionv1a2.BatchGetBlocksResponse{
		Blocks: blocks,
	}

	log.Info("BatchGetBlocks completed", "request", req, "response", res)
	return res, nil
}

// ExecuteBlock drives deterministic derivation of a rollup block from sequencer
// block data
func (s *ExecutionServiceServerV1Alpha2) ExecuteBlock(ctx context.Context, req *executionv1a2.ExecuteBlockRequest) (*executionv1a2.Block, error) {
	log.Info("ExecuteBlock called", "request", req)

	// Validate block being created has valid previous hash
	prevHeadHash := common.BytesToHash(req.PrevBlockHash)
	softHash := s.bc.CurrentSafeBlock().Hash()
	if prevHeadHash != softHash {
		return nil, status.Error(codes.FailedPrecondition, "Block can only be created on top of soft block.")
	}

	// This set of ordered TXs on the TxPool is has been configured to be used by
	// the Miner when building a payload.
	s.eth.TxPool().SetAstriaOrdered(req.Transactions)

	// Build a payload to add to the chain
	payloadAttributes := &miner.BuildPayloadArgs{
		Parent:       prevHeadHash,
		Timestamp:    uint64(req.GetTimestamp().GetSeconds()),
		Random:       common.Hash{},
		FeeRecipient: common.Address{},
	}
	payload, err := s.eth.Miner().BuildPayload(payloadAttributes)
	if err != nil {
		log.Error("failed to build payload", "err", err)
		return nil, status.Error(codes.InvalidArgument, "Could not build block with provided txs")
	}

	// call blockchain.InsertChain to actually execute and write the blocks to
	// state
	block, err := engine.ExecutableDataToBlock(*payload.Resolve().ExecutionPayload)
	if err != nil {
		log.Error("failed to convert executable data to block", err)
		return nil, status.Error(codes.Internal, "failed to execute block")
	}
	blocks := types.Blocks{
		block,
	}
	n, err := s.bc.InsertChain(blocks)
	if err != nil {
		log.Error("failed to insert block to chain", err, "index", n)
		return nil, status.Error(codes.Internal, "failed to insert block to chain")
	}

	// remove txs from original mempool
	for _, tx := range block.Transactions() {
		s.eth.TxPool().RemoveTx(tx.Hash())
	}

	res := &executionv1a2.Block{
		Number: uint32(block.NumberU64()),
		Hash:   block.Hash().Bytes(),
		Timestamp: &timestamppb.Timestamp{
			Seconds: int64(block.Time()),
		},
	}

	log.Info("ExecuteBlock completed", "request", req, "response", res)
	return res, nil
}

// GetCommitmentState fetches the current CommitmentState of the chain.
func (s *ExecutionServiceServerV1Alpha2) GetCommitmentState(ctx context.Context, req *executionv1a2.GetCommitmentStateRequest) (*executionv1a2.CommitmentState, error) {
	log.Info("GetCommitmentState called", "request", req)

	softBlock, err := ethHeaderToExecutionBlock(s.bc.CurrentSafeBlock())
	if err != nil {
		log.Error("error finding safe block", err)
		return nil, status.Error(codes.Internal, "could not locate soft block")
	}
	firmBlock, err := ethHeaderToExecutionBlock(s.bc.CurrentFinalBlock())
	if err != nil {
		log.Error("error finding final block", err)
		return nil, status.Error(codes.Internal, "could not locate firm block")
	}

	res := &executionv1a2.CommitmentState{
		Soft: softBlock,
		Firm: firmBlock,
	}

	log.Info("GetCommitmentState completed", "request", req, "response", res)
	return res, nil
}

// UpdateCommitmentState replaces the whole CommitmentState with a new
// CommitmentState.
func (s *ExecutionServiceServerV1Alpha2) UpdateCommitmentState(ctx context.Context, req *executionv1a2.UpdateCommitmentStateRequest) (*executionv1a2.CommitmentState, error) {
	log.Info("UpdateCommitmentState called", "request", req)
	
	softEthHash := common.BytesToHash(req.CommitmentState.Soft.Hash)
	firmEthHash := common.BytesToHash(req.CommitmentState.Firm.Hash)

	// Validate that the firm and soft blocks exist before going further
	softBlock := s.bc.GetBlockByHash(softEthHash)
	if softBlock == nil {
		return nil, status.Error(codes.InvalidArgument, "Soft block specified does not exist")
	}
	firmBlock := s.bc.GetBlockByHash(firmEthHash)
	if firmBlock == nil {
		return nil, status.Error(codes.InvalidArgument, "Firm block specified does not exist")
	}

	currentHead := s.bc.CurrentBlock().Hash()

	// Update the canonical chain to soft block. We must do this before last
	// validation step since there is no way to check if firm block descends from
	// anything but the canonical chain
	if currentHead != softEthHash {
		if _, err := s.bc.SetCanonical(softBlock); err != nil {
			log.Error("failed updating canonical chain to soft block", err)
			return nil, status.Error(codes.Internal, "Could not update head to safe hash")
		}
	}

	// Once head is updated validate that firm belongs to chain
	if s.bc.GetCanonicalHash(firmBlock.NumberU64()) != firmEthHash {
		log.Error("firm block not found in canonical chain defined by soft block, rolling back")

		// We don't want partial commitments, rolling back.
		rollbackBlock := s.bc.GetBlockByHash(currentHead)
		if _, err := s.bc.SetCanonical(rollbackBlock); err != nil {
			panic("rollback to previous head after failed validation failed")
		}

		return nil, status.Error(codes.InvalidArgument, "soft block in request is not a descendant of the current firmly committed block")
	}

	// Updating the safe and final after everything validated
	currentSafe := s.bc.CurrentSafeBlock().Hash()
	if currentSafe != softEthHash {
		s.bc.SetSafe(softBlock.Header())
	}
	currentFirm := s.bc.CurrentFinalBlock().Hash()
	if currentFirm != firmEthHash {
		s.bc.SetFinalized(firmBlock.Header())
	}

	log.Info("UpdateCommitmentState completed", "request", req)
	return req.CommitmentState, nil
}

func (s *ExecutionServiceServerV1Alpha2) getBlockFromIdentifier(identifier *executionv1a2.BlockIdentifier) (*executionv1a2.Block, error) {
	var header *types.Header

	// Grab the header based on the identifier provided
	switch idType := identifier.Identifier.(type) {
	case *executionv1a2.BlockIdentifier_BlockNumber:
		header = s.bc.GetHeaderByNumber(uint64(identifier.GetBlockNumber()))
	case *executionv1a2.BlockIdentifier_BlockHash:
		header = s.bc.GetHeaderByHash(common.BytesToHash(identifier.GetBlockHash()))
	default:
		return nil, status.Errorf(codes.InvalidArgument, "identifier has unexpected type %T", idType)
	}

	if header == nil {
		return nil, status.Errorf(codes.NotFound, "Couldn't locate block with identifier %s", identifier.Identifier)
	}

	res, err := ethHeaderToExecutionBlock(header)
	if err != nil {
		// This should never happen since we validate header exists above.
		return nil, status.Error(codes.Internal, "internal error")
	}

	return res, nil
}

func ethHeaderToExecutionBlock(header *types.Header) (*executionv1a2.Block, error) {
	if header == nil {
		return nil, fmt.Errorf("cannot convert nil header to execution block")
	}

	return &executionv1a2.Block{
		Number:          uint32(header.Number.Int64()),
		Hash:            header.Hash().Bytes(),
		ParentBlockHash: header.ParentHash.Bytes(),
	}, nil
}
