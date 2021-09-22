package eth

import (
	"fmt"
	"plugin"

	"github.com/ethereum/go-ethereum/rpc"
)

// Plugin describes an RPC endpoint that can be added to the node as a plugin.
type Plugin interface {
	Namespace() string
	Version() string
	Service(*Ethereum) interface{}
}

// loadRPCPlugins loads a set of plugins and creates their rpc.API objects
func loadRPCPlugins(plugins []string, eth *Ethereum) ([]rpc.API, error) {
	apis := make([]rpc.API, 0)

	for _, path := range plugins {
		p, err := plugin.Open(path)
		if err != nil {
			return nil, fmt.Errorf("could not open plugin: %s, err: %s", path, err)
		}

		v, err := p.Lookup("Register")
		if err != nil {
			return nil, fmt.Errorf("symbol `Register` not found in plugin: %s", path)
		}

		api, ok := v.(Plugin)
		if !ok {
			return nil, fmt.Errorf("invalid plugin: %s", path)
		}

		apis = append(apis, rpc.API{
			Namespace: api.Namespace(),
			Version:   api.Version(),
			Service:   api.Service(eth),
			Public:    true,
		})
	}

	return apis, nil
}
