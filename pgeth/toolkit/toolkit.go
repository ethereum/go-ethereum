package toolkit

import (
	"github.com/ethereum/go-ethereum/internal/ethapi"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/node"
)

type PluginToolkit struct {
	Node    *node.Node
	Backend ethapi.Backend
	Logger  log.Logger
}
