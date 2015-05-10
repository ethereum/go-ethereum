/*
	This file is part of go-ethereum

	go-ethereum is free software: you can redistribute it and/or modify
	it under the terms of the GNU General Public License as published by
	the Free Software Foundation, either version 3 of the License, or
	(at your option) any later version.

	go-ethereum is distributed in the hope that it will be useful,
	but WITHOUT ANY WARRANTY; without even the implied warranty of
	MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
	GNU General Public License for more details.

	You should have received a copy of the GNU General Public License
	along with go-ethereum.  If not, see <http://www.gnu.org/licenses/>.
*/
/**
 * @authors
 * 	Jeffrey Wilcke <i@jev.io>
 */
package main

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"strconv"

	"github.com/ethereum/go-ethereum/cmd/utils"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
)

type plugin struct {
	Name string `json:"name"`
	Path string `json:"path"`
}

func (gui *Gui) Transact(from, recipient, value, gas, gasPrice, d string) (string, error) {
	d = common.Bytes2Hex(utils.FormatTransactionData(d))

	return gui.xeth.Transact(from, recipient, "", value, gas, gasPrice, d)
}

func (self *Gui) AddPlugin(pluginPath string) {
	self.plugins[pluginPath] = plugin{Name: pluginPath, Path: pluginPath}

	json, _ := json.MarshalIndent(self.plugins, "", "    ")
	ioutil.WriteFile(self.eth.DataDir+"/plugins.json", json, os.ModePerm)
}

func (self *Gui) RemovePlugin(pluginPath string) {
	delete(self.plugins, pluginPath)

	json, _ := json.MarshalIndent(self.plugins, "", "    ")
	ioutil.WriteFile(self.eth.DataDir+"/plugins.json", json, os.ModePerm)
}

func (self *Gui) DumpState(hash, path string) {
	var stateDump []byte

	if len(hash) == 0 {
		stateDump = self.eth.ChainManager().State().Dump()
	} else {
		var block *types.Block
		if hash[0] == '#' {
			i, _ := strconv.Atoi(hash[1:])
			block = self.eth.ChainManager().GetBlockByNumber(uint64(i))
		} else {
			block = self.eth.ChainManager().GetBlock(common.HexToHash(hash))
		}

		if block == nil {
			guilogger.Infof("block err: not found %s\n", hash)
			return
		}

		stateDump = state.New(block.Root(), self.eth.StateDb()).Dump()
	}

	file, err := os.OpenFile(path[7:], os.O_CREATE|os.O_RDWR, os.ModePerm)
	if err != nil {
		guilogger.Infoln("dump err: ", err)
		return
	}
	defer file.Close()

	guilogger.Infof("dumped state (%s) to %s\n", hash, path)

	file.Write(stateDump)
}
