package api

import (
	"strings"

	"fmt"

	"github.com/ethereum/go-ethereum/eth"
	"github.com/ethereum/go-ethereum/rpc/codec"
	"github.com/ethereum/go-ethereum/xeth"
	"github.com/ethereum/go-ethereum/rpc/shared"
)

const (
	EthApiName = "eth"
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
		case EthApiName:
			apis[i] = NewEthApi(xeth, codec)
		default:
			return nil, fmt.Errorf("Unknown API '%s'", name)
		}
	}

	return apis, nil
}

// combines multiple API's
type mergedApi struct {
	apis map[string]EthereumApi
}

// create new merged api instance
func newMergedApi(apis ...EthereumApi) *mergedApi {
	mergedApi := new(mergedApi)
	mergedApi.apis = make(map[string]EthereumApi)

	for _, api := range apis {
		for _, method := range api.Methods() {
			mergedApi.apis[method] = api
		}
	}
	return mergedApi
}

// Supported RPC methods
func (self *mergedApi) Methods() []string {
	all := make([]string, len(self.apis))
	for method, _ := range self.apis {
		all = append(all, method)
	}
	return all
}

// Call the correct API's Execute method for the given request
func (self *mergedApi) Execute(req *shared.Request) (interface{}, error) {
	if api, found := self.apis[req.Method]; found {
		return api.Execute(req)
	}
	return nil, shared.NewNotImplementedError(req.Method)
}

// Merge multiple API's to a single API instance
func Merge(apis ...EthereumApi) EthereumApi {
	return newMergedApi(apis...)
}
