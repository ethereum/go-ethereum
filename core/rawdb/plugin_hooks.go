package rawdb


import (
  "github.com/ethereum/go-ethereum/plugins"
  "github.com/ethereum/go-ethereum/log"
)

func PluginAppendAncient(pl *plugins.PluginLoader, number uint64, hash, header, body, receipts, td []byte) {
  fnList := pl.Lookup("AppendAncient", func(item interface{}) bool {
    _, ok := item.(func(number uint64, hash, header, body, receipts, td []byte))
    return ok
  })
  for _, fni := range fnList {
    if fn, ok := fni.(func(number uint64, hash, header, body, receipts, td []byte)); ok {
      fn(number, hash, header, body, receipts, td)
    }
  }
}
func pluginAppendAncient(number uint64, hash, header, body, receipts, td []byte) {
  if plugins.DefaultPluginLoader == nil {
		log.Warn("Attempting AppendAncient, but default PluginLoader has not been initialized")
    return
  }
  PluginAppendAncient(plugins.DefaultPluginLoader, number, hash, header, body, receipts, td)
}
