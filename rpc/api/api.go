package api

import (
	"strings"

	"github.com/ethereum/go-ethereum/rpc/shared"
)

const (
	AdminApiName    = "admin"
	EthApiName      = "eth"
	DebugApiName    = "debug"
	MergedApiName   = "merged"
	MinerApiName    = "miner"
	NetApiName      = "net"
	ShhApiName      = "shh"
	TxPoolApiName   = "txpool"
	PersonalApiName = "personal"
	Web3ApiName     = "web3"
)

var (
	// List with all API's which are offered over the IPC interface by default
	DefaultIpcApis = strings.Join([]string{
		AdminApiName, EthApiName, DebugApiName, MinerApiName, NetApiName,
		ShhApiName, TxPoolApiName, PersonalApiName, Web3ApiName,
	}, ",")
)

const (
	// List with all API's which are offered over the IPC interface by default
	DefaultIpcApis = "eth"
)

// Ethereum RPC API interface
type EthereumApi interface {
	// API identifier
	Name() string

	// API version
	ApiVersion() string

	// Execute the given request and returns the response or an error
	Execute(*shared.Request) (interface{}, error)

	// List of supported RCP methods this API provides
	Methods() []string
}

// Merge multiple API's to a single API instance
func Merge(apis ...EthereumApi) EthereumApi {
	return newMergedApi(apis...)
}
