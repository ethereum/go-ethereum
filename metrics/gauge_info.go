package metrics

import (
	"encoding/json"
	"sync"
)

// GaugeInfoValue is a mapping of keys to values
type GaugeInfoValue map[string]string

func (val GaugeInfoValue) String() string {
	data, _ := json.Marshal(val)
	return string(data)
}

// GetOrRegisterGaugeInfo returns an existing GaugeInfo or constructs and registers a
// new GaugeInfo.
func GetOrRegisterGaugeInfo(name string, r Registry) *GaugeInfo {
	return getOrRegister(name, NewGaugeInfo, r)
}

// NewGaugeInfo constructs a new GaugeInfo.
func NewGaugeInfo() *GaugeInfo {
	return &GaugeInfo{
		value: GaugeInfoValue{},
	}
}

// NewRegisteredGaugeInfo constructs and registers a new GaugeInfo.
func NewRegisteredGaugeInfo(name string, r Registry) *GaugeInfo {
	c := NewGaugeInfo()
	if nil == r {
		r = DefaultRegistry
	}
	r.Register(name, c)
	return c
}

// GaugeInfoSnapshot is a read-only copy of another GaugeInfo.
type GaugeInfoSnapshot GaugeInfoValue

// Value returns the value at the time the snapshot was taken.
func (g GaugeInfoSnapshot) Value() GaugeInfoValue { return GaugeInfoValue(g) }

// GaugeInfo maintains a set of key/value mappings.
type GaugeInfo struct {
	mutex sync.Mutex
	value GaugeInfoValue
}

// Snapshot returns a read-only copy of the gauge.
func (g *GaugeInfo) Snapshot() GaugeInfoSnapshot {
	return GaugeInfoSnapshot(g.value)
}

// Update updates the gauge's value.
func (g *GaugeInfo) Update(v GaugeInfoValue) {
	g.mutex.Lock()
	defer g.mutex.Unlock()
	g.value = v
}
