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
	//"admin_startMiner":          (*admin).StartMiner,
	}
)

// admin callback handler
type adminhandler func(*admin, *shared.Request) (interface{}, error)

// admin api provider
type admin struct {
	ethereum ethereum.Ethereum
	methods  map[string]adminhandler
	codec    codec.ApiCoder
}

// create a new admin api instance
func NewAdmin(ethereum ethereum.Ethereum, coder codec.Codec) *admin {
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
