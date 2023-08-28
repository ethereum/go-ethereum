package whitelist

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/metrics"
)

type checkpoint struct {
	finality[*rawdb.Checkpoint]
}

type checkpointService interface {
	finalityService
}

var (
	//Metrics for collecting the whitelisted milestone number
	whitelistedCheckpointNumberMeter = metrics.NewRegisteredGauge("chain/checkpoint/latest", nil)

	//Metrics for collecting the number of invalid chains received
	CheckpointChainMeter = metrics.NewRegisteredMeter("chain/checkpoint/isvalidchain", nil)

	//Metrics for collecting the number of valid peers received
	CheckpointPeerMeter = metrics.NewRegisteredMeter("chain/checkpoint/isvalidpeer", nil)
)

// IsValidChain checks the validity of chain by comparing it
// against the local checkpoint entry
func (w *checkpoint) IsValidChain(currentHeader *types.Header, chain []*types.Header) (bool, error) {
	w.finality.RLock()
	defer w.finality.RUnlock()

	res, err := w.finality.IsValidChain(currentHeader, chain)

	if res {
		CheckpointChainMeter.Mark(int64(1))
	} else {
		CheckpointPeerMeter.Mark(int64(-1))
	}

	return res, err
}

// IsValidPeer checks if the chain we're about to receive from a peer is valid or not
// in terms of reorgs. We won't reorg beyond the last bor finality submitted to mainchain.
func (w *checkpoint) IsValidPeer(fetchHeadersByNumber func(number uint64, amount int, skip int, reverse bool) ([]*types.Header, []common.Hash, error)) (bool, error) {
	res, err := w.finality.IsValidPeer(fetchHeadersByNumber)

	if res {
		CheckpointPeerMeter.Mark(int64(1))
	} else {
		CheckpointPeerMeter.Mark(int64(-1))
	}

	return res, err
}

func (w *checkpoint) Process(block uint64, hash common.Hash) {
	w.finality.Lock()
	defer w.finality.Unlock()

	w.finality.Process(block, hash)

	whitelistedCheckpointNumberMeter.Update(int64(block))
}
