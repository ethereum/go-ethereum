package api

import (
	ethereum "github.com/ethereum/go-ethereum/eth"
	"github.com/ethereum/go-ethereum/rpc/codec"
	"github.com/ethereum/go-ethereum/rpc/shared"
)

const (
	AdminVersion = "1.0.0"
)

var (
	// mapping between methods and handlers
	AdminMapping = map[string]adminhandler{
//		"admin_startRPC":          (*admin).StartRPC,
//		"admin_startRPC":          (*admin).StopRPC,
//		"admin_startRPC":          (*admin).NodeInfo,
//		"admin_startRPC":          (*admin).StartRPC,
//		"admin_startRPC":          (*admin).StartRPC,
//		"admin_startRPC":          (*admin).StartRPC,
//		"admin_startRPC":          (*admin).StartRPC,
//		"admin_startRPC":          (*admin).StartRPC,
//		"admin_startRPC":          (*admin).StartRPC,
//		"admin_startRPC":          (*admin).StartRPC,
//		"admin_startRPC":          (*admin).StartRPC,
//		"admin_startRPC":          (*admin).StartRPC,
	}
)

/*
addPeer: [Function],
  startRPC: [Function],
  stopRPC: [Function],
  nodeInfo: [Function],
  peers: [Function],
  newAccount: [Function],
  unlock: [Function],
  import: [Function],
  export: [Function],
  verbosity: [Function],
  progress: [Function],
  setSolc: [Function],
  contractInfo: {
    start: [Function],
    stop: [Function],
    newRegistry: [Function],
    get: [Function],
    register: [Function],
    registerUrl: [Function]
  },
 */

// admin callback handler
type adminhandler func(*admin, *shared.Request) (interface{}, error)

// admin api provider
type admin struct {
	ethereum *ethereum.Ethereum
	methods  map[string]adminhandler
	codec    codec.ApiCoder
}

// create a new admin api instance
func NewAdmin(ethereum *ethereum.Ethereum, coder codec.Codec) *admin {
	return &admin{
		ethereum: ethereum,
		methods:  AdminMapping,
		codec:    coder.New(nil),
	}
}

// collection with supported methods
func (self *admin) Methods() []string {
	methods := make([]string, len(self.methods))
	i := 0
	for k := range self.methods {
		methods[i] = k
		i++
	}
	return methods
}

// Execute given request
func (self *admin) Execute(req *shared.Request) (interface{}, error) {
	if callback, ok := self.methods[req.Method]; ok {
		return callback(self, req)
	}

	return nil, &shared.NotImplementedError{req.Method}
}

func (self *admin) Id() string {
	return "admin"
}
