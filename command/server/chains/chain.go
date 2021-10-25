package chains

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
)

type Chain struct {
	Hash      common.Hash
	Genesis   *core.Genesis
	Bootnodes []string
	NetworkId uint64
	DNS       []string
}

var chains = map[string]*Chain{
	"mainnet": mainnetBor,
	"mumbai":  mumbaiTestnet,
}

func GetChain(name string) (*Chain, bool) {
	chain, ok := chains[name]
	return chain, ok
}
