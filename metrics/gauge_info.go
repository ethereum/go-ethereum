package metrics

import (
	"sync"
)

// GaugeInfos hold a GaugeInfoValue value that can be set arbitrarily.
type GaugeInfo interface {
	Snapshot() GaugeInfo
	Update(GaugeInfoValue)
	Value() GaugeInfoValue
	ValueJsonString() string
}

type GaugeInfoEntry struct {
	Key string
	Val string
}

type GaugeInfoValue []GaugeInfoEntry

func NewGaugeInfoEntry(key string, val string) GaugeInfoEntry {
	return GaugeInfoEntry{key, val}
}

// GetOrRegisterGaugeInfo returns an existing GaugeInfo or constructs and registers a
// new StandardGaugeInfo.
func GetOrRegisterGaugeInfo(name string, r Registry) GaugeInfo {
	if nil == r {
		r = DefaultRegistry
	}
	return r.GetOrRegister(name, NewGaugeInfo()).(GaugeInfo)
}

// NewGaugeInfo constructs a new StandardGaugeInfo.
func NewGaugeInfo() GaugeInfo {
	if !Enabled {
		return NilGaugeInfo{}
	}
	return &StandardGaugeInfo{
		value: GaugeInfoValue{},
	}
}

// NewRegisteredGaugeInfo constructs and registers a new StandardGaugeInfo.
func NewRegisteredGaugeInfo(name string, r Registry) GaugeInfo {
	c := NewGaugeInfo()
	if nil == r {
		r = DefaultRegistry
	}
	r.Register(name, c)
	return c
}

// NewFunctionalGauge constructs a new FunctionalGauge.
func NewFunctionalGaugeInfo(f func() GaugeInfoValue) GaugeInfo {
	if !Enabled {
		return NilGaugeInfo{}
	}
	return &FunctionalGaugeInfo{value: f}
}

// NewRegisteredFunctionalGauge constructs and registers a new StandardGauge.
func NewRegisteredFunctionalGaugeInfo(name string, r Registry, f func() GaugeInfoValue) GaugeInfo {
	c := NewFunctionalGaugeInfo(f)
	if nil == r {
		r = DefaultRegistry
	}
	r.Register(name, c)
	return c
}

// GaugeInfoSnapshot is a read-only copy of another GaugeInfo.
type GaugeInfoSnapshot GaugeInfoValue

// Snapshot returns the snapshot.
func (g GaugeInfoSnapshot) Snapshot() GaugeInfo { return g }

// Update panics.
func (GaugeInfoSnapshot) Update(GaugeInfoValue) {
	panic("Update called on a GaugeInfoSnapshot")
}

// Value returns the value at the time the snapshot was taken.
func (g GaugeInfoSnapshot) Value() GaugeInfoValue { return GaugeInfoValue(g) }

// Value returns the value at the time the snapshot was taken in JSON string format.
func (g GaugeInfoSnapshot) ValueJsonString() string {
	return gaugeInfoValueToJsonString(g.Value())
}

// NilGauge is a no-op Gauge.
type NilGaugeInfo struct{}

// Snapshot is a no-op.
func (NilGaugeInfo) Snapshot() GaugeInfo { return NilGaugeInfo{} }

// Update is a no-op.
func (NilGaugeInfo) Update(v GaugeInfoValue) {}

// Value is a no-op.
func (NilGaugeInfo) Value() GaugeInfoValue { return GaugeInfoValue{} }

// Value is a no-op.
func (NilGaugeInfo) ValueJsonString() string { return gaugeInfoValueToJsonString(GaugeInfoValue{}) }

// StandardGaugeInfo is the standard implementation of a GaugeInfo and uses
// sync.Mutex to manage a single string value.
type StandardGaugeInfo struct {
	mutex sync.Mutex
	value GaugeInfoValue
}

// Snapshot returns a read-only copy of the gauge.
func (g *StandardGaugeInfo) Snapshot() GaugeInfo {
	return GaugeInfoSnapshot(g.Value())
}

// Update updates the gauge's value.
func (g *StandardGaugeInfo) Update(v GaugeInfoValue) {
	g.mutex.Lock()
	defer g.mutex.Unlock()
	g.value = v
}

// Value returns the gauge's current value.
func (g *StandardGaugeInfo) Value() GaugeInfoValue {
	g.mutex.Lock()
	defer g.mutex.Unlock()
	return g.value
}

// Value returns the gauge's current value in JSON string format.
func (g *StandardGaugeInfo) ValueJsonString() string {
	g.mutex.Lock()
	defer g.mutex.Unlock()
	return gaugeInfoValueToJsonString(g.value)
}

// FunctionalGaugeInfo returns value from given function
type FunctionalGaugeInfo struct {
	value func() GaugeInfoValue
}

// Value returns the gauge's current value.
func (g FunctionalGaugeInfo) Value() GaugeInfoValue {
	return g.value()
}

// Value returns the gauge's current value in JSON string format
func (g FunctionalGaugeInfo) ValueJsonString() string {
	return gaugeInfoValueToJsonString(g.value())
}

// Snapshot returns the snapshot.
func (g FunctionalGaugeInfo) Snapshot() GaugeInfo { return GaugeInfoSnapshot(g.Value()) }

// Update panics.
func (FunctionalGaugeInfo) Update(GaugeInfoValue) {
	panic("Update called on a FunctionalGaugeInfo")
}

// Custom conversion to Json to avoid printing "Key" and "Val"
func gaugeInfoValueToJsonString(g GaugeInfoValue) string {
	lastIdx := len(g) - 1
	v := "{"
	for idx, entry := range g {
		v += "\"" + entry.Key + "\":\"" + entry.Val + "\""
		if idx != lastIdx {
			v += ","
		}
	}
	v += "}"
	return v
}
