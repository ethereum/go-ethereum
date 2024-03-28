package tracers

import (
	"encoding/json"
	"errors"

	"github.com/ethereum/go-ethereum/core/tracing"
)

type ctorFunc func(config json.RawMessage) (*tracing.Hooks, error)

// LiveDirectory is the collection of tracers which can be used
// during normal block import operations.
var LiveDirectory = liveDirectory{elems: make(map[string]ctorFunc)}

type liveDirectory struct {
	elems map[string]ctorFunc
}

// Register registers a tracer constructor by name.
func (d *liveDirectory) Register(name string, f ctorFunc) {
	d.elems[name] = f
}

// New instantiates a tracer by name.
func (d *liveDirectory) New(name string, config json.RawMessage) (*tracing.Hooks, error) {
	if f, ok := d.elems[name]; ok {
		return f(config)
	}
	return nil, errors.New("not found")
}
