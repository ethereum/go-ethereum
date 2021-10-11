package native

import (
	"encoding/json"

	"github.com/ethereum/go-ethereum/core/vm"
)

type Tracer interface {
	vm.Tracer
	GetResult() (json.RawMessage, error)
}

type Constructor func() Tracer

var tracers map[string]Constructor = make(map[string]Constructor)

func Register(name string, fn Constructor) {
	tracers[name] = fn
}

func New(name string) (Tracer, bool) {
	if fn, ok := tracers[name]; ok {
		return fn(), true
	}
	return nil, false
}
