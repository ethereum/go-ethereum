package api

import (
	"github.com/ethereum/go-ethereum/eth"
	"github.com/ethereum/go-ethereum/rpc/codec"
	"github.com/ethereum/go-ethereum/rpc/shared"
	"github.com/ethereum/go-ethereum/xeth"
)

var (
	// mapping between methods and handlers
	netMapping = map[string]nethandler{
		"net_id":        (*net).NetworkVersion,
		"net_peerCount": (*net).PeerCount,
		"net_listening": (*net).IsListening,
		"net_peers":     (*net).Peers,
	}
)

// net callback handler
type nethandler func(*net, *shared.Request) (interface{}, error)

// net api provider
type net struct {
	xeth     *xeth.XEth
	ethereum *eth.Ethereum
	methods  map[string]nethandler
	codec    codec.ApiCoder
}

// create a new net api instance
func NewNetApi(xeth *xeth.XEth, eth *eth.Ethereum, coder codec.Codec) *net {
	return &net{
		xeth:     xeth,
		ethereum: eth,
		methods:  netMapping,
		codec:    coder.New(nil),
	}
}

// collection with supported methods
func (self *net) Methods() []string {
	methods := make([]string, len(self.methods))
	i := 0
	for k := range self.methods {
		methods[i] = k
		i++
	}
	return methods
}

// Execute given request
func (self *net) Execute(req *shared.Request) (interface{}, error) {
	if callback, ok := self.methods[req.Method]; ok {
		return callback(self, req)
	}

	return nil, shared.NewNotImplementedError(req.Method)
}

func (self *net) Name() string {
	return NetApiName
}

// Network version
func (self *net) NetworkVersion(req *shared.Request) (interface{}, error) {
	return self.xeth.NetworkVersion(), nil
}

// Number of connected peers
func (self *net) PeerCount(req *shared.Request) (interface{}, error) {
	return self.xeth.PeerCount(), nil
}

func (self *net) IsListening(req *shared.Request) (interface{}, error) {
	return self.xeth.IsListening(), nil
}

func (self *net) Peers(req *shared.Request) (interface{}, error) {
	return self.ethereum.PeersInfo(), nil
}
