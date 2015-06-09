package api

import "github.com/ethereum/go-ethereum/rpc/shared"

// combines multiple API's
type MergedApi struct {
	apis    []string
	methods map[string]EthereumApi
}

// create new merged api instance
func newMergedApi(apis ...EthereumApi) *MergedApi {
	mergedApi := new(MergedApi)
	mergedApi.apis = make([]string, len(apis))
	mergedApi.methods = make(map[string]EthereumApi)

	for i, api := range apis {
		mergedApi.apis[i] = api.Name()
		for _, method := range api.Methods() {
			mergedApi.methods[method] = api
		}
	}
	return mergedApi
}

// Supported RPC methods
func (self *MergedApi) Methods() []string {
	all := make([]string, len(self.methods))
	for method, _ := range self.methods {
		all = append(all, method)
	}
	return all
}

// Call the correct API's Execute method for the given request
func (self *MergedApi) Execute(req *shared.Request) (interface{}, error) {
	if res, _ := self.handle(req); res != nil {
		return res, nil
	}
	if api, found := self.methods[req.Method]; found {
		return api.Execute(req)
	}
	return nil, shared.NewNotImplementedError(req.Method)
}

func (self *MergedApi) Name() string {
	return MergedApiName
}

func (self *MergedApi) handle(req *shared.Request) (interface{}, error) {
	if req.Method == "support_apis" { // provided API's
		return self.apis, nil
	}

	return nil, nil
}
