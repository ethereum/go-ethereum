package heimdallapp

import (
	"context"
	"math/big"

	"github.com/cosmos/cosmos-sdk/types"

	"github.com/ethereum/go-ethereum/consensus/bor/heimdall/checkpoint"
	"github.com/ethereum/go-ethereum/log"

	hmTypes "github.com/maticnetwork/heimdall/types"
	abci "github.com/tendermint/tendermint/abci/types"
)

func (h *HeimdallAppClient) FetchCheckpointCount(_ context.Context) (int64, error) {
	log.Info("Fetching checkpoint count")

	res := h.hApp.CheckpointKeeper.GetACKCount(h.NewContext())

	log.Info("Fetched checkpoint count")

	return int64(res), nil
}

func (h *HeimdallAppClient) FetchCheckpoint(_ context.Context, number int64) (*checkpoint.Checkpoint, error) {
	log.Info("Fetching checkpoint", "number", number)

	res, err := h.hApp.CheckpointKeeper.GetCheckpointByNumber(h.NewContext(), uint64(number))
	if err != nil {
		return nil, err
	}

	log.Info("Fetched checkpoint", "number", number)

	return toBorCheckpoint(res), nil
}

func (h *HeimdallAppClient) NewContext() types.Context {
	return h.hApp.NewContext(true, abci.Header{Height: h.hApp.LastBlockHeight()})
}

func toBorCheckpoint(hdCheckpoint hmTypes.Checkpoint) *checkpoint.Checkpoint {
	return &checkpoint.Checkpoint{
		Proposer:   hdCheckpoint.Proposer.EthAddress(),
		StartBlock: big.NewInt(int64(hdCheckpoint.StartBlock)),
		EndBlock:   big.NewInt(int64(hdCheckpoint.EndBlock)),
		RootHash:   hdCheckpoint.RootHash.EthHash(),
		BorChainID: hdCheckpoint.BorChainID,
		Timestamp:  hdCheckpoint.TimeStamp,
	}
}
