package whitelist

import (
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"gotest.tools/assert"
)

// NewMockService creates a new mock whitelist service
func NewMockService(maxCapacity uint) *Service {
	return &Service{
		checkpointWhitelist: make(map[uint64]common.Hash),
		checkpointOrder:     []uint64{},
		maxCapacity:         maxCapacity,
	}
}

// TestWhitelistCheckpoint checks the checkpoint whitelist map queue mechanism
func TestWhitelistCheckpoint(t *testing.T) {
	t.Parallel()

	s := NewMockService(10)
	for i := 0; i < 10; i++ {
		s.EnqueueCheckpointWhitelist(uint64(i), common.Hash{})
	}
	assert.Equal(t, len(s.GetCheckpointWhitelist()), 10, "expected 10 items in whitelist")

	s.EnqueueCheckpointWhitelist(11, common.Hash{})
	s.DequeueCheckpointWhitelist()
	assert.Equal(t, len(s.GetCheckpointWhitelist()), 10, "expected 10 items in whitelist")
}
