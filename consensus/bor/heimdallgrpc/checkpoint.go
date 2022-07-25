package heimdallgrpc

import (
	"context"
	"math/big"

	"github.com/ethereum/go-ethereum/consensus/bor/heimdall/checkpoint"

	proto "github.com/maticnetwork/polyproto/heimdall"
)

func (h *HeimdallGRPCClient) FetchCheckpointCount(ctx context.Context) (int64, error) {
	res, err := h.client.FetchCheckpointCount(context.Background(), nil)
	if err != nil {
		return 0, err
	}

	return res.Result.Result, nil
}

func (h *HeimdallGRPCClient) FetchCheckpoint(ctx context.Context, number int64) (*checkpoint.Checkpoint, error) {
	req := &proto.FetchCheckpointRequest{
		ID: number,
	}

	res, err := h.client.FetchCheckpoint(context.Background(), req)
	if err != nil {
		return nil, err
	}

	checkpoint := &checkpoint.Checkpoint{
		StartBlock: new(big.Int).SetUint64(res.Result.StartBlock),
		EndBlock:   new(big.Int).SetUint64(res.Result.EndBlock),
		RootHash:   ConvertH256ToHash(res.Result.RootHash),
		Proposer:   ConvertH160toAddress(res.Result.Proposer),
		BorChainID: res.Result.BorChainID,
		Timestamp:  uint64(res.Result.Timestamp.GetSeconds()),
	}

	return checkpoint, nil
}
