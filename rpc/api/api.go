package api

import (
	"github.com/ethereum/go-ethereum/rpc/shared"
)

// Merge multiple API's to a single API instance
func Merge(apis ...shared.EthereumApi) shared.EthereumApi {
	return newMergedApi(apis...)
}
