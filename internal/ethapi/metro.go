package ethapi

import (
	metrotx "github.com/astriaorg/metro-transactions/tx"
)

const secondaryChainID = "ethereum"

func submitMetroTransaction(tx []byte) error {
	return metrotx.BuildAndSendSecondaryTransaction(metrotx.DefaultGRPCEndpoint, secondaryChainID, tx)
}
