package flow

import (
	"sync"
)

// MeterRegistry is a registry for named meters.
type MeterRegistry struct {
	meters sync.Map
}

// Get gets (or creates) a meter by name.
func (r *MeterRegistry) Get(name string) *Meter {
	if m, ok := r.meters.Load(name); ok {
		return m.(*Meter)
	}
	m, _ := r.meters.LoadOrStore(name, new(Meter))
	return m.(*Meter)
}

// Remove removes the named meter from the registry.
//
// Note: The only reason to do this is to save a bit of memory. Unused meters
// don't consume any CPU (after they go idle).
func (r *MeterRegistry) Remove(name string) {
	r.meters.Delete(name)
}

// ForEach calls the passed function for each registered meter.
func (r *MeterRegistry) ForEach(iterFunc func(string, *Meter)) {
	r.meters.Range(func(k, v interface{}) bool {
		iterFunc(k.(string), v.(*Meter))
		return true
	})
}
