package backends

import (
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/event"
)

// SubscribeStateSyncEvent subscribes to state sync events
func (fb *filterBackend) SubscribeStateSyncEvent(ch chan<- core.StateSyncEvent) event.Subscription {
	return fb.bc.SubscribeStateSyncEvent(ch)
}
