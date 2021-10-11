package native

import "github.com/ethereum/go-ethereum/core/vm"

type Constructor func() vm.Tracer

var tracers map[string]Constructor = make(map[string]Constructor)

func Register(name string, fn Constructor) {
	tracers[name] = fn
}

func New(name string) (vm.Tracer, bool) {
	if fn, ok := tracers[name]; ok {
		return fn(), true
	}
	return nil, false
}
