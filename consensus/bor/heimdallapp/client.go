package heimdallapp

import (
	"github.com/ethereum/go-ethereum/log"

	"github.com/maticnetwork/heimdall/app"
	"github.com/maticnetwork/heimdall/cmd/heimdalld/service"
)

const (
	stateFetchLimit = 50
)

type HeimdallAppClient struct {
	hApp *app.HeimdallApp
}

func NewHeimdallAppClient() *HeimdallAppClient {
	return &HeimdallAppClient{
		hApp: service.GetHeimdallApp(),
	}
}

func (h *HeimdallAppClient) Close() {
	// Nothing to close as of now
	log.Warn("Shutdown detected, Closing Heimdall App conn")
}
