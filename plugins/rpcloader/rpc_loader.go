package rpcloader

import (
  "github.com/ethereum/go-ethereum/node"
  "github.com/ethereum/go-ethereum/plugins"
)

type APILoader func(*node.Node, Backend) []rpc.API

func GetRPCPluginsFromLoader(pl *plugins.PluginLoader) []APILoader {
  result := []APILoader{}
  for _, plug := range pl.Plugins {
    fn, err := plug.Lookup("GetAPIs")
    if err == nil {
      apiLoader, ok := fn.(func(*node.Node, Backend) []rpc.API)
      if !ok {
        log.Warn("Could not cast plugin.GetAPIs to APILoader", "file", fpath)
        } else {
          result = append(result, APILoader(apiLoader))
        }
      } else { log.Debug("Error retrieving GetAPIs for plugin", "file", fpath, "error", err.Error()) }
  }
}

func GetRPCPlugins() []APILoader {
  if plugins.DefaultPluginLoader == nil {
		log.Warn("Attempting GetRPCPlugins, but default PluginLoader has not been initialized")
		return []APILoader{}
	 }
  return GetRPCPluginsFromLoader(plugins.DefaultPluginLoader)

}
func GetAPIsFromLoader(pl *plugins.PluginLoader, stack *node.Node, backend Backend) []rpc.API {
	apis := []rpc.API{}
	for _, apiLoader := range pl.RPCPlugins {
		apis = append(apis, apiLoader(stack, backend)...)
	}
	return apis
}

func GetAPIs(stack *node.Node, backend Backend) []rpc.API {
	if plugins.DefaultPluginLoader == nil {
		log.Warn("Attempting GetAPIs, but default PluginLoader has not been initialized")
		return []rpc.API{}
	 }
	return GetAPIsFromLoader(plugins.DefaultPluginLoader, stack, backend)
}
