package tracers

import (
	"encoding/json"
	"errors"
	"github.com/ethereum/go-ethereum/eth/tracers/blocknative"
)

// Register a tracer lookup function that creates and returns blocknative
// tracers. We put the registration here instead of in the blocknative package
// so that it doesn't have to import this package, which imports geth/core, and
// in turn allows us to use the blocknative tracers from inside the geth/core
// package without causing circular dependency issues.
func init() {
	RegisterLookup(false, func(name string, _ *Context, _ json.RawMessage) (Tracer, error) {
		if constructor, ok := blocknative.Tracers[name]; ok {
			return constructor()
		}
		return nil, errors.New("no blocknative tracer found")
	})
}
