package live

import (
	"context"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/rpc"
)

type filterAPI struct {
	filter *filter
}

type txTraceResult struct {
	TxHash common.Hash `json:"txHash"`           // transaction hash
	Result interface{} `json:"result,omitempty"` // Trace results produced by the tracer
	Error  string      `json:"error,omitempty"`  // Trace failure produced by the tracer
}

func (api *filterAPI) Block(ctx context.Context, blockNr rpc.BlockNumber) ([]*txTraceResult, error) {
	return nil, nil
}
