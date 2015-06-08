package api

import "github.com/ethereum/go-ethereum/rpc/shared"

const (
	// List with all API's which are offered over the IPC interface by default
	DefaultIpcApis = "eth"
)

// Ethereum RPC API interface
type EthereumApi interface {
	// Execute the given request and returns the response or an error
	Execute(*shared.Request) (interface{}, error)

	// List of supported RCP methods this API provides
	Methods() []string
}
