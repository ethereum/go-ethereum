package eth

import (
  "github.com/ethereum/go-ethereum/plugins"
  "github.com/ethereum/go-ethereum/node"
  "github.com/ethereum/go-ethereum/params"
  "github.com/ethereum/go-ethereum/core/vm"
  "github.com/ethereum/go-ethereum/consensus"
  "github.com/ethereum/go-ethereum/consensus/ethash"
  "github.com/ethereum/go-ethereum/eth/ethconfig"
  "github.com/ethereum/go-ethereum/ethdb"
)


  func pluginCreateConsensusEngine(stack *node.Node, chainConfig *params.ChainConfig, config *ethash.Config, notify []string, noverify bool, db ethdb.Database) consensus.Engine {
  fnList := plugins.Lookup("CreateConsensusEngine", func(item interface{}) bool {
    _, ok := item.(func(*node.Node, *params.ChainConfig, *ethash.Config, []string, bool, ethdb.Database) consensus.Engine)
    return ok
  })
  for _, fni := range fnList {
    if fn, ok := fni.(func(*node.Node, *params.ChainConfig, *ethash.Config, []string, bool, ethdb.Database) consensus.Engine); ok {
      return fn(stack, chainConfig, config, notify, noverify, db)
    }
  }
  return ethconfig.CreateConsensusEngine(stack, chainConfig, config, notify, noverify, db)
}
func pluginUpdateBlockchainVMConfig(cfg *vm.Config) {
  fnList := plugins.Lookup("UpdateBlockchainVMConfig", func(item interface{}) bool {
    _, ok := item.(func(*vm.Config))
    return ok
  })
  for _, fni := range fnList {
    if fn, ok := fni.(func(*vm.Config)); ok {
      fn(cfg)
      return
    }
  }
}

// ce, err := plug.Lookup("CreateConsensusEngine")
// if err == nil {
// 	cce, ok := ce.(func (stack *node.Node, chainConfig *params.ChainConfig, config *ethash.Config, notify []string, noverify bool, db ethdb.Database) consensus.Engine)
// 	if !ok {
// 		log.Warn("Could not cast plugin.CreateConsensusEngine to appropriate function", "file", fpath)
// 	} else {
// 		if setConsensus {
// 			log.Warn("CreateConsensusEngine redeclared", "file", fpath)
// 		}
// 		pl.CreateConsensusEngine = cce
// 		setConsensus = true
// 	}
// }
