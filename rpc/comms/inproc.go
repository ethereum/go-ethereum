package comms

import (
	"github.com/ethereum/go-ethereum/rpc/api"
	"github.com/ethereum/go-ethereum/rpc/shared"
	"fmt"
	"github.com/ethereum/go-ethereum/rpc/codec"
	"github.com/ethereum/go-ethereum/xeth"
	"github.com/ethereum/go-ethereum/eth"
)

type InProcClient struct {
	api api.EthereumApi
	codec codec.Codec
	lastId interface{}
	lastJsonrpc string
	lastErr error
	lastRes interface{}
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
	//return *shared.NewRpcResponse(self.lastId, self.lastJsonrpc, self.lastRes, self.lastErr), nil
}
