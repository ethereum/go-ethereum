package tracers

import (
	"encoding/json"
	"github.com/ethereum/go-ethereum/eth/tracers/blocknative"
)

// Register a tracer lookup function that creates and returns blocknative
// tracers. We put the registration here instead of in the blocknative package
// so that it doesn't have to import this package, which imports geth/core, and
// in turn allows us to use the blocknative tracers from inside the geth/core
// package without causing circular dependency issues.
func init() {
	for name, constructor := range blocknative.Tracers {
		DefaultDirectory.Register(name, func(_ *Context, m json.RawMessage) (Tracer, error) {
			return constructor(m)
		}, false)
	}
	// RegisterLookup(false, func(name string, _ *Context, cfg json.RawMessage) (Tracer, error) {
	// 	if constructor, ok := blocknative.Tracers[name]; ok {
	// 		return constructor(cfg)
	// 	}
	// 	return nil, errors.New("no blocknative tracer found")
	// })
}
