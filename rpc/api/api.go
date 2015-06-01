package api

import (
	"github.com/ethereum/go-ethereum/rpc/shared"
)

// Descriptor for all API implementations
type Ethereum interface {
	// Execute single API request
	Execute(*shared.Request) (interface{}, error)
	// List with supported methods
	Methods() []string
}

type mergedEthereumApi struct {
	apis map[string]Ethereum
}

func newMergedEthereumApi(apis ...Ethereum) *mergedEthereumApi {
	mergedApi := new(mergedEthereumApi)
	mergedApi.apis = make(map[string]Ethereum)

	for _, api := range apis {
		for _, method := range api.Methods() {
			mergedApi.apis[method] = api
		}
	}
	return mergedApi
}

func (self *mergedEthereumApi) Methods() []string {
	return nil
}

func (self *mergedEthereumApi) Execute(req *shared.Request) (interface{}, error) {
	if api, found := self.apis[req.Method]; found {
		return api.Execute(req)
	}
	return nil, shared.NewNotImplementedError(req.Method)
}

func Merge(apis ...Ethereum) Ethereum {
	return newMergedEthereumApi(apis...)
}
