package core

import (
  "github.com/ethereum/go-ethereum/core/types"
  "github.com/ethereum/go-ethereum/plugins"
)

func pluginPreProcessBlock(block *types.Block) {
  fnList := plugins.Lookup("ProcessBlock", func(item interface{}) bool {
    _, ok := item.(func(*types.Block))
    return ok
  })
  for _, fni := range fnList {
    if fn, ok := fni.(func(*types.Block)); ok {
      fn(block)
    }
  }
}
func pluginPreProcessTransaction(tx *types.Transaction, block *types.Block, i int) {
  fnList := plugins.Lookup("ProcessTransaction", func(item interface{}) bool {
    _, ok := item.(func(*types.Transaction, *types.Block, int))
    return ok
  })
  for _, fni := range fnList {
    if fn, ok := fni.(func(*types.Transaction, *types.Block, int)); ok {
      fn(tx, block, i)
    }
  }
}
func pluginBlockProcessingError(tx *types.Transaction, block *types.Block, err error) {
  fnList := plugins.Lookup("ProcessingError", func(item interface{}) bool {
    _, ok := item.(func(*types.Transaction, *types.Block, error))
    return ok
  })
  for _, fni := range fnList {
    if fn, ok := fni.(func(*types.Transaction, *types.Block, error)); ok {
      fn(tx, block, err)
    }
  }
}
func pluginPostProcessTransaction(tx *types.Transaction, block *types.Block, i int, receipt *types.Receipt) {
  fnList := plugins.Lookup("ProcessTransaction", func(item interface{}) bool {
    _, ok := item.(func(*types.Transaction, *types.Block, int, *types.Receipt))
    return ok
  })
  for _, fni := range fnList {
    if fn, ok := fni.(func(*types.Transaction, *types.Block, int, *types.Receipt)); ok {
      fn(tx, block, i, receipt)
    }
  }
}
func pluginPostProcessBlock(block *types.Block) {
  fnList := plugins.Lookup("ProcessBlock", func(item interface{}) bool {
    _, ok := item.(func(*types.Block))
    return ok
  })
  for _, fni := range fnList {
    if fn, ok := fni.(func(*types.Block)); ok {
      fn(block)
    }
  }
}
