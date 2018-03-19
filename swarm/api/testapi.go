// Copyright 2016 The go-etherinc Authors
// This file is part of the go-etherinc library.
//
// The go-etherinc library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-etherinc library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-etherinc library. If not, see <http://www.gnu.org/licenses/>.

package api

import (
	"github.com/etherinc/go-etherinc/swarm/network"
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

func (self *Control) Hive() string {
	return self.hive.String()
}
