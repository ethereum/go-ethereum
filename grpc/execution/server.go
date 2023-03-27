// Package execution provides the gRPC server for the execution layer.
//
// Its procedures will be called from the conductor. It is responsible
// for immediately executing lists of ordered transactions that come from the shared sequencer.
package execution

import (
	"context"

	"github.com/ethereum/go-ethereum/core/types"
	executionv1 "github.com/ethereum/go-ethereum/grpc/gen/proto/execution/v1"
	"github.com/ethereum/go-ethereum/internal/ethapi"
	"github.com/ethereum/go-ethereum/log"
)

// executionServiceServer is the implementation of the ExecutionServiceServer interface.
type ExecutionServiceServer struct {
	// NOTE - from the generated code:
	// All implementations must embed UnimplementedExecutionServiceServer
	// for forward compatibility
	executionv1.UnimplementedExecutionServiceServer

	// TODO - will need access to the consensus api to call functions for building a block
	// e.g. getPayload, newPayload, forkchoiceUpdated

	Backend ethapi.Backend

	// TODO - will need access to forkchoice on first run.
	// this will probably be passed in when calling NewServer
}

// FIXME - how do we know which hash to start with? will probably need another api function like
// GetHeadHash() to get the head hash of the forkchoice

func (s *ExecutionServiceServer) DoBlock(ctx context.Context, req *executionv1.DoBlockRequest) (*executionv1.DoBlockResponse, error) {
	log.Info("DoBlock called request", "request", req)

	// NOTE - Request.Header.ParentHash needs to match forkchoice head hash
	// ParentHash should be the forkchoice head of the last block

	// TODO - need to call consensus api to build a block

	// txs := bytesToTransactions(req.Transactions)
	// for _, tx := range txs {
	// 	s.Backend.SendTx(ctx, tx)
	// }

	res := &executionv1.DoBlockResponse{
		// TODO - get state root from last block
		StateRoot: []byte{0x00},
	}
	return res, nil
}

// convert bytes to transactions
func bytesToTransactions(b [][]byte) []*types.Transaction {
	txs := []*types.Transaction{}
	for _, txBytes := range b {
		tx := &types.Transaction{}
		tx.UnmarshalBinary(txBytes)
		txs = append(txs, tx)
	}
	return txs
}
