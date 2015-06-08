package api

import "github.com/ethereum/go-ethereum/rpc/shared"

const (
	// List with all API's which are offered over the IPC interface by default
	DefaultIpcApis = "eth,web3"

	EthApiName = "eth"
	MergedApiName = "merged"
	Web3ApiName = "web3"
)

// Ethereum RPC API interface
type EthereumApi interface {
	// API identifier
	Name() string

	// Execute the given request and returns the response or an error
	Execute(*shared.Request) (interface{}, error)

	// List of supported RCP methods this API provides
	Methods() []string
}

// Merge multiple API's to a single API instance
func Merge(apis ...EthereumApi) EthereumApi {
	return newMergedApi(apis...)
}
