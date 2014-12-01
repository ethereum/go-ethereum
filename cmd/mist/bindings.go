// Copyright (c) 2013-2014, Jeffrey Wilcke. All rights reserved.
//
// This library is free software; you can redistribute it and/or
// modify it under the terms of the GNU General Public
// License as published by the Free Software Foundation; either
// version 2.1 of the License, or (at your option) any later version.
//
// This library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the GNU
// General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with this library; if not, write to the Free Software
// Foundation, Inc., 51 Franklin Street, Fifth Floor, Boston,
// MA 02110-1301  USA

package main

import (
	"encoding/json"
	"os"
	"strconv"

	"github.com/ethereum/go-ethereum/chain"
	"github.com/ethereum/go-ethereum/cmd/utils"
	"github.com/ethereum/go-ethereum/ethutil"
	"github.com/ethereum/go-ethereum/logger"
)

type plugin struct {
	Name string `json:"name"`
	Path string `json:"path"`
}

// LogPrint writes to the GUI log.
func (gui *Gui) LogPrint(level logger.LogLevel, msg string) {
	/*
		str := strings.TrimRight(s, "\n")
		lines := strings.Split(str, "\n")

		view := gui.getObjectByName("infoView")
		for _, line := range lines {
			view.Call("addLog", line)
		}
	*/
}
func (gui *Gui) Transact(recipient, value, gas, gasPrice, d string) (string, error) {
	var data string
	if len(recipient) == 0 {
		code, err := ethutil.Compile(d, false)
		if err != nil {
			return "", err
		}
		data = ethutil.Bytes2Hex(code)
	} else {
		data = ethutil.Bytes2Hex(utils.FormatTransactionData(d))
	}

	return gui.pipe.Transact(gui.privateKey(), recipient, value, gas, gasPrice, data)
}

func (gui *Gui) SetCustomIdentifier(customIdentifier string) {
	gui.clientIdentity.SetCustomIdentifier(customIdentifier)
	gui.config.Save("id", customIdentifier)
}

func (gui *Gui) GetCustomIdentifier() string {
	return gui.clientIdentity.GetCustomIdentifier()
}

// functions that allow Gui to implement interface guilogger.LogSystem
func (gui *Gui) SetLogLevel(level logger.LogLevel) {
	gui.logLevel = level
	gui.stdLog.SetLogLevel(level)
	gui.config.Save("loglevel", level)
}

func (gui *Gui) GetLogLevel() logger.LogLevel {
	return gui.logLevel
}

func (self *Gui) AddPlugin(pluginPath string) {
	self.plugins[pluginPath] = plugin{Name: pluginPath, Path: pluginPath}

	json, _ := json.MarshalIndent(self.plugins, "", "    ")
	ethutil.WriteFile(ethutil.Config.ExecPath+"/plugins.json", json)
}

func (self *Gui) RemovePlugin(pluginPath string) {
	delete(self.plugins, pluginPath)

	json, _ := json.MarshalIndent(self.plugins, "", "    ")
	ethutil.WriteFile(ethutil.Config.ExecPath+"/plugins.json", json)
}

// this extra function needed to give int typecast value to gui widget
// that sets initial loglevel to default
func (gui *Gui) GetLogLevelInt() int {
	return int(gui.logLevel)
}
func (self *Gui) DumpState(hash, path string) {
	var stateDump []byte

	if len(hash) == 0 {
		stateDump = self.eth.BlockManager().CurrentState().Dump()
	} else {
		var block *chain.Block
		if hash[0] == '#' {
			i, _ := strconv.Atoi(hash[1:])
			block = self.eth.ChainManager().GetBlockByNumber(uint64(i))
		} else {
			block = self.eth.ChainManager().GetBlock(ethutil.Hex2Bytes(hash))
		}

		if block == nil {
			guilogger.Infof("block err: not found %s\n", hash)
			return
		}

		stateDump = block.State().Dump()
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
