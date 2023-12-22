package toolkit

import (
	log "github.com/sirupsen/logrus"

	"github.com/ethereum/go-ethereum/internal/ethapi"
	"github.com/ethereum/go-ethereum/node"
)

type PluginToolkit struct {
	Node    *node.Node
	Backend ethapi.Backend
	Logger  *log.Logger
}
