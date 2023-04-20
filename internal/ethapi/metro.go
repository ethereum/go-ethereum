package ethapi

import (
	metrotx "github.com/astriaorg/metro-transactions/tx"
	"github.com/ethereum/go-ethereum/log"
)

const secondaryChainID = "ethereum"

type MetroAPI struct {
	endpoint string
}

func NewMetroAPI(endpoint string) *MetroAPI {
	log.Info("NewMetroAPI", "endpoint", endpoint)
	return &MetroAPI{endpoint: endpoint}
}

func (api *MetroAPI) SubmitTransaction(tx []byte) error {
	return metrotx.BuildAndSendSecondaryTransaction(api.endpoint, secondaryChainID, tx)
}
