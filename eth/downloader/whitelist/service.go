package whitelist

import (
	"errors"
	"fmt"
	"sync"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
)

// Checkpoint whitelist
type Service struct {
	m                   sync.Mutex
	checkpointWhitelist map[uint64]common.Hash // Checkpoint whitelist, populated by reaching out to heimdall
	checkpointOrder     []uint64               // Checkpoint order, populated by reaching out to heimdall
	maxCapacity         uint
}

func NewService(maxCapacity uint) *Service {
	return &Service{
		checkpointWhitelist: make(map[uint64]common.Hash),
		checkpointOrder:     []uint64{},
		maxCapacity:         maxCapacity,
	}
}

var (
	ErrCheckpointMismatch = errors.New("checkpoint mismatch")
	ErrNoRemoteCheckoint  = errors.New("remote peer doesn't have a checkoint")
)

// IsValidChain checks if the chain we're about to receive from this peer is valid or not
// in terms of reorgs. We won't reorg beyond the last bor checkpoint submitted to mainchain.
func (w *Service) IsValidChain(remoteHeader *types.Header, fetchHeadersByNumber func(number uint64, amount int, skip int, reverse bool) ([]*types.Header, []common.Hash, error)) (bool, error) {
	// We want to validate the chain by comparing the last checkpointed block
	// we're storing in `checkpointWhitelist` with the peer's block.
	//
	// Check for availaibility of the last checkpointed block.
	// This can be also be empty if our heimdall is not responding
	// or we're running without it.
	if len(w.checkpointWhitelist) == 0 {
		// worst case, we don't have the checkpoints in memory
		return true, nil
	}

	// Fetch the last checkpoint entry
	lastCheckpointBlockNum := w.checkpointOrder[len(w.checkpointOrder)-1]
	lastCheckpointBlockHash := w.checkpointWhitelist[lastCheckpointBlockNum]

	// todo: we can extract this as an interface and mock as well or just test IsValidChain in isolation from downloader passing fake fetchHeadersByNumber functions
	headers, hashes, err := fetchHeadersByNumber(lastCheckpointBlockNum, 1, 0, false)
	if err != nil {
		return false, fmt.Errorf("%w: last checkpoint %d, err %v", ErrNoRemoteCheckoint, lastCheckpointBlockNum, err)
	}

	if len(headers) == 0 {
		return false, fmt.Errorf("%w: last checkpoint %d", ErrNoRemoteCheckoint, lastCheckpointBlockNum)
	}

	reqBlockNum := headers[0].Number.Uint64()
	reqBlockHash := hashes[0]

	// Check against the checkpointed blocks
	if reqBlockNum == lastCheckpointBlockNum && reqBlockHash == lastCheckpointBlockHash {
		return true, nil
	}

	return false, ErrCheckpointMismatch
}

func (w *Service) ProcessCheckpoint(endBlockNum uint64, endBlockHash common.Hash) {
	w.m.Lock()
	defer w.m.Unlock()

	w.enqueueCheckpointWhitelist(endBlockNum, endBlockHash)
	// If size of checkpoint whitelist map is greater than 10, remove the oldest entry.

	if w.length() > int(w.maxCapacity) {
		w.dequeueCheckpointWhitelist()
	}
}

// GetCheckpointWhitelist returns the existing whitelisted
// entries of checkpoint of the form block number -> block hash.
func (w *Service) GetCheckpointWhitelist() map[uint64]common.Hash {
	w.m.Lock()
	defer w.m.Unlock()

	return w.checkpointWhitelist
}

// PurgeCheckpointWhitelist purges data from checkpoint whitelist map
func (w *Service) PurgeCheckpointWhitelist() {
	w.m.Lock()
	defer w.m.Unlock()

	w.checkpointWhitelist = make(map[uint64]common.Hash)
	w.checkpointOrder = make([]uint64, 0)
}

// EnqueueWhitelistBlock enqueues blockNumber, blockHash to the checkpoint whitelist map
func (w *Service) enqueueCheckpointWhitelist(key uint64, val common.Hash) {
	if _, ok := w.checkpointWhitelist[key]; !ok {
		log.Debug("Enqueing new checkpoint whitelist", "block number", key, "block hash", val)

		w.checkpointWhitelist[key] = val
		w.checkpointOrder = append(w.checkpointOrder, key)
	}
}

// DequeueWhitelistBlock dequeues block, blockhash from the checkpoint whitelist map
func (w *Service) dequeueCheckpointWhitelist() {
	if len(w.checkpointOrder) > 0 {
		log.Debug("Dequeing checkpoint whitelist", "block number", w.checkpointOrder[0], "block hash", w.checkpointWhitelist[w.checkpointOrder[0]])

		delete(w.checkpointWhitelist, w.checkpointOrder[0])
		w.checkpointOrder = w.checkpointOrder[1:]
	}
}

// length returns the len of the whitelist.
func (w *Service) length() int {
	return len(w.checkpointWhitelist)
}
