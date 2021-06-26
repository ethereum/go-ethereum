package main

import (
  "github.com/ethereum/go-ethereum/node"
  "github.com/ethereum/go-ethereum/plugins"
  "github.com/ethereum/go-ethereum/plugins/interfaces"
  "github.com/ethereum/go-ethereum/rpc"
  "github.com/ethereum/go-ethereum/log"
)

type APILoader func(*node.Node, interfaces.Backend) []rpc.API

func GetAPIsFromLoader(pl *plugins.PluginLoader, stack *node.Node, backend interfaces.Backend) []rpc.API {
  result := []rpc.API{}
  fnList := pl.Lookup("GetAPIs", func(item interface{}) bool {
    _, ok := item.(func(*node.Node, interfaces.Backend) []rpc.API)
      return ok
  })
  for _, fni := range fnList {
    if fn, ok := fni.(func(*node.Node, interfaces.Backend) []rpc.API); ok {
      result = append(result, fn(stack, backend)...)
    }
  }
  return result
}

func pluginGetAPIs(stack *node.Node, backend interfaces.Backend) []rpc.API {
	if plugins.DefaultPluginLoader == nil {
		log.Warn("Attempting GetAPIs, but default PluginLoader has not been initialized")
		return []rpc.API{}
	 }
	return GetAPIsFromLoader(plugins.DefaultPluginLoader, stack, backend)
}
