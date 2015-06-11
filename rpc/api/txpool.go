package api

import (
	"github.com/ethereum/go-ethereum/eth"
	"github.com/ethereum/go-ethereum/rpc/codec"
	"github.com/ethereum/go-ethereum/rpc/shared"
	"github.com/ethereum/go-ethereum/xeth"
)

const (
	TxPoolApiVersion = "1.0"
)

var (
	// mapping between methods and handlers
	txpoolMapping = map[string]txpoolhandler{
		"txpool_status": (*txPoolApi).Status,
	}
)

// net callback handler
type txpoolhandler func(*txPoolApi, *shared.Request) (interface{}, error)

// txpool api provider
type txPoolApi struct {
	xeth     *xeth.XEth
	ethereum *eth.Ethereum
	methods  map[string]txpoolhandler
	codec    codec.ApiCoder
}

// create a new txpool api instance
func NewTxPoolApi(xeth *xeth.XEth, eth *eth.Ethereum, coder codec.Codec) *txPoolApi {
	return &txPoolApi{
		xeth:     xeth,
		ethereum: eth,
		methods:  txpoolMapping,
		codec:    coder.New(nil),
	}
}

// collection with supported methods
func (self *txPoolApi) Methods() []string {
	methods := make([]string, len(self.methods))
	i := 0
	for k := range self.methods {
		methods[i] = k
		i++
	}
	return methods
}

// Execute given request
func (self *txPoolApi) Execute(req *shared.Request) (interface{}, error) {
	if callback, ok := self.methods[req.Method]; ok {
		return callback(self, req)
	}

	return nil, shared.NewNotImplementedError(req.Method)
}

func (self *txPoolApi) Name() string {
	return TxPoolApiName
}

func (self *txPoolApi) ApiVersion() string {
	return TxPoolApiVersion
}

func (self *txPoolApi) Status(req *shared.Request) (interface{}, error) {
	return map[string]int{
		"pending": self.ethereum.TxPool().GetTransactions().Len(),
		"queued":  self.ethereum.TxPool().GetQueuedTransactions().Len(),
	}, nil
}
