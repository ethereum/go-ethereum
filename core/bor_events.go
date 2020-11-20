package core

import (
	"github.com/ethereum/go-ethereum/core/types"
)

// StateSyncEvent represents state sync events
type StateSyncEvent struct {
	Data *types.StateSyncData
}
