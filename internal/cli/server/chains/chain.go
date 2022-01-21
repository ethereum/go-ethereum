package chains

import (
	"encoding/json"
	"io/ioutil"

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
	chain, err := ImportFromFile(name)
	if err != nil {
		chain, ok := chains[name]
		return chain, ok
	}
	return chain, true
}

func ImportFromFile(filename string) (*Chain, error) {
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	return importChain(data)
}

func importChain(content []byte) (*Chain, error) {
	var chain *Chain

	if err := json.Unmarshal(content, &chain); err != nil {
		return nil, err
	}

	return chain, nil
}
