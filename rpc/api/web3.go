package api

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/rpc/codec"
	"github.com/ethereum/go-ethereum/rpc/shared"
	"github.com/ethereum/go-ethereum/xeth"
)

const (
	Web3Version = "1.0.0"
)

var (
// mapping between methods and handlers
	Web3Mapping = map[string]web3handler{
		"web3_sha3":          (*web3).Sha3,
		"web3_clientVersion": (*web3).ClientVersion,
	}
)

// web3 callback handler
type web3handler func(*web3, *shared.Request) (interface{}, error)

// web3 api provider
type web3 struct {
	xeth    *xeth.XEth
	methods map[string]web3handler
	codec   codec.ApiCoder
}

// create a new web3 api instance
func NewWeb3(xeth *xeth.XEth, coder codec.Codec) *web3 {
	return &web3{
		xeth:    xeth,
		methods: Web3Mapping,
		codec:   coder.New(nil),
	}
}

// collection with supported methods
func (self *web3) Methods() []string {
	methods := make([]string, len(self.methods))
	i := 0
	for k := range self.methods {
		methods[i] = k
		i++
	}
	return methods
}

// Execute given request
func (self *web3) Execute(req *shared.Request) (interface{}, error) {
	if callback, ok := self.methods[req.Method]; ok {
		return callback(self, req)
	}

	return nil, &shared.NotImplementedError{req.Method}
}

func (self *web3) Name() string {
	return Web3ApiName
}

// Version of the API this instance provides
func (self *web3) Version() string {
	return Web3Version
}

// Calculates the sha3 over req.Params.Data
func (self *web3) Sha3(req *shared.Request) (interface{}, error) {
	args := new(Sha3Args)
	if err := self.codec.Decode(req.Params, &args); err != nil {
		return nil, err
	}

	return common.ToHex(crypto.Sha3(common.FromHex(args.Data))), nil
}

// returns the xeth client vrsion
func (self *web3) ClientVersion(req *shared.Request) (interface{}, error) {
	return self.xeth.ClientVersion(), nil
}
