package heimdallgrpc

import (
	"context"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/consensus/bor/heimdall"
	"github.com/ethereum/go-ethereum/consensus/bor/heimdall/milestone"
	"github.com/ethereum/go-ethereum/log"

	proto "github.com/maticnetwork/polyproto/heimdall"
	protoutils "github.com/maticnetwork/polyproto/utils"
)

func (h *HeimdallGRPCClient) FetchMilestoneCount(ctx context.Context) (int64, error) {
	log.Info("Fetching milestone count")

	res, err := h.client.FetchMilestoneCount(ctx, nil)
	if err != nil {
		return 0, err
	}

	log.Info("Fetched milestone count")

	return res.Result.Count, nil
}

func (h *HeimdallGRPCClient) FetchMilestone(ctx context.Context) (*milestone.Milestone, error) {
	log.Info("Fetching milestone")

	res, err := h.client.FetchMilestone(ctx, nil)
	if err != nil {
		return nil, err
	}

	log.Info("Fetched milestone")

	milestone := &milestone.Milestone{
		StartBlock: new(big.Int).SetUint64(res.Result.StartBlock),
		EndBlock:   new(big.Int).SetUint64(res.Result.EndBlock),
		Hash:       protoutils.ConvertH256ToHash(res.Result.RootHash),
		Proposer:   protoutils.ConvertH160toAddress(res.Result.Proposer),
		BorChainID: res.Result.BorChainID,
		Timestamp:  uint64(res.Result.Timestamp.GetSeconds()),
	}

	return milestone, nil
}

func (h *HeimdallGRPCClient) FetchLastNoAckMilestone(ctx context.Context) (string, error) {
	log.Debug("Fetching latest no ack milestone Id")

	res, err := h.client.FetchLastNoAckMilestone(ctx, nil)
	if err != nil {
		return "", err
	}

	log.Debug("Fetched last no-ack milestone", "res", res.Result.Result)

	return res.Result.Result, nil
}

func (h *HeimdallGRPCClient) FetchNoAckMilestone(ctx context.Context, milestoneID string) error {
	req := &proto.FetchMilestoneNoAckRequest{
		MilestoneID: milestoneID,
	}

	log.Debug("Fetching no ack milestone", "milestoneID", milestoneID)

	res, err := h.client.FetchNoAckMilestone(ctx, req)
	if err != nil {
		return err
	}

	if !res.Result.Result {
		return fmt.Errorf("%w: milestoneID %q", heimdall.ErrNotInRejectedList, milestoneID)
	}

	log.Debug("Fetched no ack milestone", "milestoneID", milestoneID)

	return nil
}

func (h *HeimdallGRPCClient) FetchMilestoneID(ctx context.Context, milestoneID string) error {
	req := &proto.FetchMilestoneIDRequest{
		MilestoneID: milestoneID,
	}

	log.Debug("Fetching milestone id", "milestoneID", milestoneID)

	res, err := h.client.FetchMilestoneID(ctx, req)
	if err != nil {
		return err
	}

	if !res.Result.Result {
		return fmt.Errorf("%w: milestoneID %q", heimdall.ErrNotInMilestoneList, milestoneID)
	}

	log.Debug("Fetched milestone id", "milestoneID", milestoneID)

	return nil
}
