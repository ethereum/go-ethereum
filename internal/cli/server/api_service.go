package server

import (
	"context"
	"errors"
	"math"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rpc"

	protobor "github.com/maticnetwork/polyproto/bor"
	protoutil "github.com/maticnetwork/polyproto/utils"
)

func (s *Server) GetRootHash(ctx context.Context, req *protobor.GetRootHashRequest) (*protobor.GetRootHashResponse, error) {
	rootHash, err := s.backend.APIBackend.GetRootHash(ctx, req.StartBlockNumber, req.EndBlockNumber)
	if err != nil {
		return nil, err
	}

	return &protobor.GetRootHashResponse{RootHash: rootHash}, nil
}

func (s *Server) GetVoteOnHash(ctx context.Context, req *protobor.GetVoteOnHashRequest) (*protobor.GetVoteOnHashResponse, error) {
	vote, err := s.backend.APIBackend.GetVoteOnHash(ctx, req.StartBlockNumber, req.EndBlockNumber, req.Hash, req.MilestoneId)
	if err != nil {
		return nil, err
	}

	return &protobor.GetVoteOnHashResponse{Response: vote}, nil
}

func headerToProtoborHeader(h *types.Header) *protobor.Header {
	return &protobor.Header{
		Number:     h.Number.Uint64(),
		ParentHash: protoutil.ConvertHashToH256(h.ParentHash),
		Time:       h.Time,
	}
}

func (s *Server) HeaderByNumber(ctx context.Context, req *protobor.GetHeaderByNumberRequest) (*protobor.GetHeaderByNumberResponse, error) {
	bN, err := getRpcBlockNumberFromString(req.Number)
	if err != nil {
		return nil, err
	}
	header, err := s.backend.APIBackend.HeaderByNumber(ctx, bN)
	if err != nil {
		return nil, err
	}

	if header == nil {
		return nil, errors.New("header not found")
	}

	return &protobor.GetHeaderByNumberResponse{Header: headerToProtoborHeader(header)}, nil
}

func (s *Server) BlockByNumber(ctx context.Context, req *protobor.GetBlockByNumberRequest) (*protobor.GetBlockByNumberResponse, error) {
	bN, err := getRpcBlockNumberFromString(req.Number)
	if err != nil {
		return nil, err
	}
	block, err := s.backend.APIBackend.BlockByNumber(ctx, bN)
	if err != nil {
		return nil, err
	}

	if block == nil {
		return nil, errors.New("block not found")
	}

	return &protobor.GetBlockByNumberResponse{Block: blockToProtoBlock(block)}, nil
}

func blockToProtoBlock(h *types.Block) *protobor.Block {
	return &protobor.Block{
		Header: headerToProtoborHeader(h.Header()),
	}
}

func (s *Server) TransactionReceipt(ctx context.Context, req *protobor.ReceiptRequest) (*protobor.ReceiptResponse, error) {
	_, _, blockHash, _, txnIndex := s.backend.APIBackend.GetTransaction(protoutil.ConvertH256ToHash(req.Hash))

	receipts, err := s.backend.APIBackend.GetReceipts(ctx, blockHash)
	if err != nil {
		return nil, err
	}

	if receipts == nil {
		return nil, errors.New("no receipts found")
	}

	if len(receipts) <= int(txnIndex) {
		return nil, errors.New("transaction index out of bounds")
	}

	return &protobor.ReceiptResponse{Receipt: ConvertReceiptToProtoReceipt(receipts[txnIndex])}, nil
}

func (s *Server) BorBlockReceipt(ctx context.Context, req *protobor.ReceiptRequest) (*protobor.ReceiptResponse, error) {
	receipt, err := s.backend.APIBackend.GetBorBlockReceipt(ctx, protoutil.ConvertH256ToHash(req.Hash))
	if err != nil {
		return nil, err
	}

	return &protobor.ReceiptResponse{Receipt: ConvertReceiptToProtoReceipt(receipt)}, nil
}

func getRpcBlockNumberFromString(blockNumber string) (rpc.BlockNumber, error) {
	switch blockNumber {
	case "latest":
		return rpc.LatestBlockNumber, nil
	case "earliest":
		return rpc.EarliestBlockNumber, nil
	case "pending":
		return rpc.PendingBlockNumber, nil
	case "finalized":
		return rpc.FinalizedBlockNumber, nil
	case "safe":
		return rpc.SafeBlockNumber, nil
	default:
		blckNum, err := hexutil.DecodeUint64(blockNumber)
		if err != nil {
			return rpc.BlockNumber(0), errors.New("invalid block number")
		}
		if blckNum > math.MaxInt64 {
			return rpc.BlockNumber(0), errors.New("block number out of range")
		}
		return rpc.BlockNumber(blckNum), nil
	}
}
