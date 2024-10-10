package tracers

import (
	"encoding/json"
	"errors"

	"github.com/ethereum/go-ethereum/core/tracing"
)

// LiveDirectory is the collection of tracers which can be used
// during normal block import operations.
var LiveDirectory = liveDirectory{elems: make(map[string]tracing.LiveConstructor)}

type liveDirectory struct {
	elems map[string]tracing.LiveConstructor
}

// Register registers a tracer constructor by name.
func (d *liveDirectory) Register(name string, f tracing.LiveConstructor) {
	d.elems[name] = f
}

// New instantiates a tracer by name.
func (d *liveDirectory) New(name string, config json.RawMessage, stack tracing.Node, backend tracing.Backend) (*tracing.Hooks, error) {
	if f, ok := d.elems[name]; ok {
		return f(config, stack, backend)
	}
	return nil, errors.New("not found")
}
