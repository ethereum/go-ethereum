package state

import (
  "github.com/ethereum/go-ethereum/common"
  "github.com/ethereum/go-ethereum/plugins"
  "github.com/ethereum/go-ethereum/log"
)

func PluginStateUpdate(pl *plugins.PluginLoader, blockRoot, parentRoot common.Hash, destructs map[common.Hash]struct{}, accounts map[common.Hash][]byte, storage map[common.Hash]map[common.Hash][]byte) {
  fnList := pl.Lookup("StateUpdate", func(item interface{}) bool  {
    _, ok := item.(func(common.Hash, common.Hash, map[common.Hash]struct{}, map[common.Hash][]byte, map[common.Hash]map[common.Hash][]byte))
      return ok
  })
  for _, fni := range fnList {
    if fn, ok := fni.(func(common.Hash, common.Hash, map[common.Hash]struct{}, map[common.Hash][]byte, map[common.Hash]map[common.Hash][]byte)); ok {
      fn(blockRoot, parentRoot, destructs, accounts, storage)
    }
  }
}

func pluginStateUpdate(blockRoot, parentRoot common.Hash, destructs map[common.Hash]struct{}, accounts map[common.Hash][]byte, storage map[common.Hash]map[common.Hash][]byte) {
	if plugins.DefaultPluginLoader == nil {
		log.Warn("Attempting StateUpdate, but default PluginLoader has not been initialized")
    return
	 }
	PluginStateUpdate(plugins.DefaultPluginLoader, blockRoot, parentRoot, destructs, accounts, storage)
}
