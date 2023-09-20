package heimdallapp

import (
	"context"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/consensus/bor/heimdall/milestone"

	"github.com/ethereum/go-ethereum/log"

	chTypes "github.com/maticnetwork/heimdall/checkpoint/types"
	hmTypes "github.com/maticnetwork/heimdall/types"
)

func (h *HeimdallAppClient) FetchMilestoneCount(_ context.Context) (int64, error) {
	log.Info("Fetching milestone count")

	res := h.hApp.CheckpointKeeper.GetMilestoneCount(h.NewContext())

	log.Info("Fetched Milestone Count")

	return int64(res), nil
}

func (h *HeimdallAppClient) FetchMilestone(_ context.Context) (*milestone.Milestone, error) {
	log.Info("Fetching Latest Milestone")

	res, err := h.hApp.CheckpointKeeper.GetLastMilestone(h.NewContext())
	if err != nil {
		return nil, err
	}

	log.Info("Fetched Latest Milestone")

	return toBorMilestone(res), nil
}

func (h *HeimdallAppClient) FetchNoAckMilestone(_ context.Context, milestoneID string) error {
	log.Info("Fetching No Ack Milestone By MilestoneID", "MilestoneID", milestoneID)

	res := h.hApp.CheckpointKeeper.GetNoAckMilestone(h.NewContext(), milestoneID)
	if res {
		log.Info("Fetched No Ack By MilestoneID", "MilestoneID", milestoneID)
		return nil
	}

	return fmt.Errorf("Still No Ack Milestone exist corresponding to MilestoneId:%v", milestoneID)
}

func (h *HeimdallAppClient) FetchLastNoAckMilestone(_ context.Context) (string, error) {
	log.Info("Fetching Latest No Ack Milestone ID")

	res := h.hApp.CheckpointKeeper.GetLastNoAckMilestone(h.NewContext())

	log.Info("Fetched Latest No Ack Milestone ID")

	return res, nil
}

func (h *HeimdallAppClient) FetchMilestoneID(_ context.Context, milestoneID string) error {
	log.Info("Fetching Milestone ID ", "MilestoneID", milestoneID)

	res := chTypes.GetMilestoneID()

	if res == milestoneID {
		return nil
	}

	return fmt.Errorf("Milestone corresponding to Milestone ID:%v doesn't exist in Heimdall", milestoneID)
}

func toBorMilestone(hdMilestone *hmTypes.Milestone) *milestone.Milestone {
	return &milestone.Milestone{
		Proposer:   hdMilestone.Proposer.EthAddress(),
		StartBlock: big.NewInt(int64(hdMilestone.StartBlock)),
		EndBlock:   big.NewInt(int64(hdMilestone.EndBlock)),
		Hash:       hdMilestone.Hash.EthHash(),
		BorChainID: hdMilestone.BorChainID,
		Timestamp:  hdMilestone.TimeStamp,
	}
}
