package native

import (
	"encoding/json"

	"github.com/ethereum/go-ethereum/core/vm"
)

// Tracer interface extends vm.Tracer and additionally
// allows collecting the tracing result.
type Tracer interface {
	vm.Tracer
	GetResult() (json.RawMessage, error)
}

// constructor creates a new instance of a Tracer.
type constructor func() Tracer

var tracers map[string]constructor = make(map[string]constructor)

// register makes native tracers in this directory which adhere
// to the `Tracer` interface available to the rest of the codebase.
// It is typically invoked in the `init()` function.
func register(name string, fn constructor) {
	tracers[name] = fn
}

// New returns a new instance of a tracer, if one was
// registered under the given name.
func New(name string) (Tracer, bool) {
	if fn, ok := tracers[name]; ok {
		return fn(), true
	}
	return nil, false
}
