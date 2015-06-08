package api

import "github.com/ethereum/go-ethereum/rpc/shared"

// Ethereum RPC API interface
type EthereumApi interface {
	// Execute the given request and returns the response or an error
	Execute(*shared.Request) (interface{}, error)

	// List of supported RCP methods this API provides
	Methods() []string
}
