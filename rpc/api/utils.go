package api

import (
	"strings"

	"fmt"

	"github.com/ethereum/go-ethereum/eth"
	"github.com/ethereum/go-ethereum/rpc/codec"
	"github.com/ethereum/go-ethereum/xeth"
)

// Parse a comma separated API string to individual api's
func ParseApiString(apistr string, codec codec.Codec, xeth *xeth.XEth, eth *eth.Ethereum) ([]EthereumApi, error) {
	if len(strings.TrimSpace(apistr)) == 0 {
		return nil, fmt.Errorf("Empty apistr provided")
	}

	names := strings.Split(apistr, ",")
	apis := make([]EthereumApi, len(names))

	for i, name := range names {
		switch strings.ToLower(strings.TrimSpace(name)) {
		case DebugApiName:
			apis[i] = NewDebugApi(xeth, eth, codec)
		case EthApiName:
			apis[i] = NewEthApi(xeth, codec)
		case MinerApiName:
			apis[i] = NewMinerApi(eth, codec)
		case NetApiName:
			apis[i] = NewNetApi(xeth, eth, codec)
		case Web3ApiName:
			apis[i] = NewWeb3(xeth, codec)
		default:
			return nil, fmt.Errorf("Unknown API '%s'", name)
		}
	}

	return apis, nil
}

func Javascript(name string) string {
	switch strings.ToLower(strings.TrimSpace(name)) {
	case DebugApiName:
		return Debug_JS
	case MinerApiName:
		return Miner_JS
	case NetApiName:
		return Net_JS
	}

	return ""
}
