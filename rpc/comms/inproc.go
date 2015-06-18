package comms

import (
	"fmt"

	"github.com/ethereum/go-ethereum/eth"
	"github.com/ethereum/go-ethereum/rpc/api"
	"github.com/ethereum/go-ethereum/rpc/codec"
	"github.com/ethereum/go-ethereum/rpc/shared"
	"github.com/ethereum/go-ethereum/xeth"
)

type InProcClient struct {
	api         api.EthereumApi
	codec       codec.Codec
	lastId      interface{}
	lastJsonrpc string
	lastErr     error
	lastRes     interface{}
}

// Create a new in process client
func NewInProcClient(codec codec.Codec) *InProcClient {
	return &InProcClient{
		codec: codec,
	}
}

func (self *InProcClient) Close() {
	// do nothing
}

// Need to setup api support
func (self *InProcClient) Initialize(xeth *xeth.XEth, eth *eth.Ethereum) {
	if apis, err := api.ParseApiString(api.AllApis, self.codec, xeth, eth); err == nil {
		self.api = api.Merge(apis...)
	}
}

func (self *InProcClient) Send(req interface{}) error {
	if r, ok := req.(*shared.Request); ok {
		self.lastId = r.Id
		self.lastJsonrpc = r.Jsonrpc
		self.lastRes, self.lastErr = self.api.Execute(r)
		return self.lastErr
	}

	return fmt.Errorf("Invalid request (%T)", req)
}

func (self *InProcClient) Recv() (interface{}, error) {
	return self.lastRes, self.lastErr
}

func (self *InProcClient) SupportedModules() (map[string]string, error) {
	req := shared.Request{
		Id:      1,
		Jsonrpc: "2.0",
		Method:  "modules",
	}

	if res, err := self.api.Execute(&req); err == nil {
		if result, ok := res.(map[string]string); ok {
			return result, nil
		}
	} else {
		return nil, err
	}

	return nil, fmt.Errorf("Invalid response")
}
