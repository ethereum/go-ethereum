package qml

import (
	"sync"
)

var stats *Statistics
var statsMutex sync.Mutex

func Stats() (snapshot Statistics) {
	statsMutex.Lock()
	snapshot = *stats
	statsMutex.Unlock()
	return
}

func CollectStats(enabled bool) {
	statsMutex.Lock()
	if enabled {
		if stats == nil {
			stats = &Statistics{}
		}
	} else {
		stats = nil
	}
	statsMutex.Unlock()
}

func ResetStats() {
	statsMutex.Lock()
	old := stats
	stats = &Statistics{}
	// These are absolute values:
	stats.EnginesAlive = old.EnginesAlive
	stats.ValuesAlive = old.ValuesAlive
	statsMutex.Unlock()
	return
}

type Statistics struct {
	EnginesAlive     int
	ValuesAlive      int
	ConnectionsAlive int
}

func (stats *Statistics) enginesAlive(delta int) {
	if stats != nil {
		statsMutex.Lock()
		stats.EnginesAlive += delta
		statsMutex.Unlock()
	}
}

func (stats *Statistics) valuesAlive(delta int) {
	if stats != nil {
		statsMutex.Lock()
		stats.ValuesAlive += delta
		statsMutex.Unlock()
	}
}

func (stats *Statistics) connectionsAlive(delta int) {
	if stats != nil {
		statsMutex.Lock()
		stats.ConnectionsAlive += delta
		statsMutex.Unlock()
	}
}
