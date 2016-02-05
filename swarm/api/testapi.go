package api

import (
	// "fmt"

	"github.com/ethereum/go-ethereum/swarm/network"
)

type Control struct {
	api  *Api
	hive *network.Hive
}

func NewControl(api *Api, hive *network.Hive) *Control {
	return &Control{api, hive}
}

func (self *Control) BlockNetworkRead(on bool) {
	self.hive.BlockNetworkRead(on)
}

func (self *Control) SyncEnabled(on bool) {
	self.hive.SyncEnabled(on)
}

func (self *Control) SwapEnabled(on bool) {
	self.hive.SwapEnabled(on)
}
