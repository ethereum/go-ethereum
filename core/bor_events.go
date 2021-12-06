package core

import (
	"github.com/ethereum/go-ethereum/core/types"
)

// StateSyncEvent represents state sync events
type StateSyncEvent struct {
	Data *types.StateSyncData
}

var (
	Chain2HeadReorgEvent     = "reorg"
	Chain2HeadCanonicalEvent = "head"
	Chain2HeadForkEvent      = "fork"
)

// For tracking reorgs related information
type Chain2HeadEvent struct {
	NewChain []*types.Block
	OldChain []*types.Block
	Type     string
}
