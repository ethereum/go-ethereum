package live

import (
	"errors"

	"github.com/ethereum/go-ethereum/core"
)

type ctorFunc func() (core.BlockchainLogger, error)

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
func (d *directory) New(name string) (core.BlockchainLogger, error) {
	if f, ok := d.elems[name]; ok {
		return f()
	}
	return nil, errors.New("not found")
}
