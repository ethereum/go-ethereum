package catalyst

import (
	"github.com/ethereum/go-ethereum/eth"
	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/rpc"
)

type Service struct {
	api *consensusAPI
}

// New creates a catalyst service and registers it with the node.
func New(stack *node.Node, backend *eth.Ethereum) *Service {
	c := &Service{api: newConsensusAPI(backend)}
	stack.RegisterAPIs([]rpc.API{
		{
			Namespace: "consensus",
			Version:   "1.0",
			Service:   c.api,
			Public:    true,
		},
	})
	return c
}
