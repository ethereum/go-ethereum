package core

import (
	"github.com/ethereum/go-ethereum/core/types"
)

// StateSyncEvent represents state sync events
type StateSyncEvent struct {
	Data *types.StateSyncData
}

// For tracking reorgs related information
type Chain2HeadEvent struct {
	NewChain []*types.Block
	OldChain []*types.Block
	Type     string
}
