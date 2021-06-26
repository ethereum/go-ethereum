package tracers

import (
  "github.com/ethereum/go-ethereum/plugins"
  "github.com/ethereum/go-ethereum/core/vm"
  "github.com/ethereum/go-ethereum/core/state"
  "github.com/ethereum/go-ethereum/log"
)


type TracerResult interface {
	vm.Tracer
	GetResult() (interface{}, error)
}

func GetPluginTracer(pl *plugins.PluginLoader, name string) (func(*state.StateDB)TracerResult, bool) {
  tracers := pl.Lookup("Tracers", func(item interface{}) bool {
    _, ok := item.(map[string]func(*state.StateDB)TracerResult)
    return ok
  })
  for _, tmap := range tracers {
    if tracerMap, ok := tmap.(map[string]func(*state.StateDB)TracerResult); ok {
      if tracer, ok := tracerMap[name]; ok {
        return tracer, true
      }
    }
  }
  return nil, false
}

func getPluginTracer(name string) (func(*state.StateDB)TracerResult, bool) {
  if plugins.DefaultPluginLoader == nil {
		log.Warn("Attempting GetPluginTracer, but default PluginLoader has not been initialized")
    return nil, false
  }
  return GetPluginTracer(plugins.DefaultPluginLoader, name)
}
