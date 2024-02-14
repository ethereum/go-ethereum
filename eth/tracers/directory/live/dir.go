package live

import (
	"encoding/json"
	"errors"

	"github.com/ethereum/go-ethereum/core"
)

type ctorFunc func(config json.RawMessage) (core.BlockchainLogger, error)

// Directory is the collection of tracers which can be used
// during normal block import operations.
var Directory = directory{elems: make(map[string]ctorFunc)}

type directory struct {
	elems map[string]ctorFunc
}

// Register registers a tracer constructor by name.
func (d *directory) Register(name string, f ctorFunc) {
	d.elems[name] = f
}

// New instantiates a tracer by name.
func (d *directory) New(name string, config json.RawMessage) (core.BlockchainLogger, error) {
	if f, ok := d.elems[name]; ok {
		return f(config)
	}
	return nil, errors.New("not found")
}
