package server

import (
	"context"
	"errors"
	"fmt"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rpc"
	protobor "github.com/maticnetwork/polyproto/bor"
	protoutil "github.com/maticnetwork/polyproto/utils"
)

func (s *Server) GetRootHash(ctx context.Context, req *protobor.GetRootHashRequest) (*protobor.GetRootHashResponse, error) {
	fmt.Printf(">>>>> GetRootHash: req %v\n", req)
	rootHash, err := s.backend.APIBackend.GetRootHash(ctx, req.StartBlockNumber, req.EndBlockNumber)
	if err != nil {
		fmt.Printf(">>>>> GetRootHash: err %v\n", err)
		return nil, err
	}

	fmt.Printf(">>>>> GetRootHash: returning\n")
	return &protobor.GetRootHashResponse{RootHash: rootHash}, nil
}

func (s *Server) GetVoteOnHash(ctx context.Context, req *protobor.GetVoteOnHashRequest) (*protobor.GetVoteOnHashResponse, error) {
	fmt.Printf(">>>>> GetVoteOnHash: req %v\n", req)
	vote, err := s.backend.APIBackend.GetVoteOnHash(ctx, req.StartBlockNumber, req.EndBlockNumber, req.Hash, req.MilestoneId)
	if err != nil {
		fmt.Printf(">>>>> GetVoteOnHash: err %v\n", err)
		return nil, err
	}

	fmt.Printf(">>>>> GetVoteOnHash: returning\n")
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
	fmt.Printf(">>>>> HeaderByNumber: req %v\n", req)
	header, err := s.backend.APIBackend.HeaderByNumber(ctx, rpc.BlockNumber(req.Number))
	if err != nil {
		fmt.Printf(">>>>> HeaderByNumber: err %v\n", err)
		return nil, err
	}

	fmt.Printf(">>>>> HeaderByNumber: returning\n")
	return &protobor.GetHeaderByNumberResponse{Header: headerToProtoborHeader(header)}, nil
}

func (s *Server) BlockByNumber(ctx context.Context, req *protobor.GetBlockByNumberRequest) (*protobor.GetBlockByNumberResponse, error) {
	fmt.Printf(">>>>> BlockByNumber: req %v\n", req)
	block, err := s.backend.APIBackend.BlockByNumber(ctx, rpc.BlockNumber(req.Number))
	if err != nil {
		fmt.Printf(">>>>> BlockByNumber: err %v\n", err)
		return nil, err
	}

	fmt.Printf(">>>>> BlockByNumber: returning\n")
	return &protobor.GetBlockByNumberResponse{Block: blockToProtoBlock(block)}, nil
}

func blockToProtoBlock(h *types.Block) *protobor.Block {
	return &protobor.Block{
		Header: headerToProtoborHeader(h.Header()),
	}
}

func (s *Server) TransactionReceipt(ctx context.Context, req *protobor.ReceiptRequest) (*protobor.ReceiptResponse, error) {
	fmt.Printf(">>>>> TransactionReceipt: req %v\n", req)
	_, blockHash, _, txnIndex, err := s.backend.APIBackend.GetTransaction(ctx, protoutil.ConvertH256ToHash(req.Hash))
	if err != nil {
		fmt.Printf(">>>>> TransactionReceipt: err1 %v\n", err)
		return nil, err
	}

	receipts, err := s.backend.APIBackend.GetReceipts(ctx, blockHash)
	if err != nil {
		fmt.Printf(">>>>> TransactionReceipt: err2 %v\n", err)
		return nil, err
	}

	if receipts == nil {
		fmt.Printf(">>>>> TransactionReceipt: err3 %v\n", errors.New("no receipts found"))
		return nil, errors.New("no receipts found")
	}

	if len(receipts) <= int(txnIndex) {
		fmt.Printf(">>>>> TransactionReceipt: err4 %v\n", errors.New("transaction index out of bounds"))
		return nil, errors.New("transaction index out of bounds")
	}

	fmt.Printf(">>>>> TransactionReceipt: returning\n")
	return &protobor.ReceiptResponse{Receipt: ConvertReceiptToProtoReceipt(receipts[txnIndex])}, nil
}

func (s *Server) BorBlockReceipt(ctx context.Context, req *protobor.ReceiptRequest) (*protobor.ReceiptResponse, error) {
	fmt.Printf(">>>>> BorBlockReceipt: req %v\n", req)
	receipt, err := s.backend.APIBackend.GetBorBlockReceipt(ctx, protoutil.ConvertH256ToHash(req.Hash))
	if err != nil {
		fmt.Printf(">>>>> BorBlockReceipt: err %v\n", err)
		return nil, err
	}

	fmt.Printf(">>>>> BorBlockReceipt: returning\n")
	return &protobor.ReceiptResponse{Receipt: ConvertReceiptToProtoReceipt(receipt)}, nil
}
