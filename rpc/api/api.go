package api

import (
	"fmt"

	"github.com/ethereum/go-ethereum/rpc/shared"
	e "github.com/ethereum/go-ethereum/eth"
	"strings"
	"github.com/ethereum/go-ethereum/rpc/codec"
	"github.com/ethereum/go-ethereum/xeth"
)

const (
	DefaultIpcApis = "admin,debug,eth,miner,net,web3"
	DefaultHttpApiS = "web3,eth"
)

// Descriptor for all API implementations
type Ethereum interface {
	// API identifier, e.g. admin or web3
	Id() string
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

func (self *mergedEthereumApi) Id() string {
	return "mergedEthereumApi"
}

func Merge(apis ...Ethereum) Ethereum {
	return newMergedEthereumApi(apis...)
}

// Parses the given apiString (comma separated list with api identifiers) to a collection of API's
func ParseApiString(apiString string, codec codec.Codec, eth *e.Ethereum) ([]Ethereum, error) {
	xeth := xeth.New(eth, nil)
	apiNames := strings.Split(apiString, ",")
	apis := make([]Ethereum, len(apiNames))

	for i, name := range apiNames {
		switch name {
		case "admin":
			apis[i] = NewAdmin(eth, codec)
		case "eth":
			apis[i] = NewEth(xeth, codec)
		case "debug":
			apis[i] = NewDebug(xeth, eth, codec)
		case "miner":
			apis[i] = NewMiner(eth, codec)
		case "net":
			apis[i] = NewNet(xeth, eth, codec)
		case "web3":
			apis[i] = NewWeb3(xeth, codec)
		default:
			return nil, fmt.Errorf("Api '%s' isn't supported", name)
		}
	}

	return apis, nil
}
