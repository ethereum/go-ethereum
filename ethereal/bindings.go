package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"

	"github.com/ethereum/eth-go/ethchain"
	"github.com/ethereum/eth-go/ethlog"
	"github.com/ethereum/eth-go/ethpipe"
	"github.com/ethereum/eth-go/ethutil"
	"github.com/ethereum/go-ethereum/utils"
)

type plugin struct {
	Name string `json:"name"`
	Path string `json:"path"`
}

func (gui *Gui) Println(v ...interface{}) {
	gui.printLog(fmt.Sprintln(v...))
}

func (gui *Gui) Printf(format string, v ...interface{}) {
	gui.printLog(fmt.Sprintf(format, v...))
}

// Print function that logs directly to the GUI
func (gui *Gui) printLog(s string) {
	/*
		str := strings.TrimRight(s, "\n")
		lines := strings.Split(str, "\n")

		view := gui.getObjectByName("infoView")
		for _, line := range lines {
			view.Call("addLog", line)
		}
	*/
}
func (gui *Gui) Transact(recipient, value, gas, gasPrice, d string) (*ethpipe.JSReceipt, error) {
	var data string
	if len(recipient) == 0 {
		code, err := ethutil.Compile(d, false)
		if err != nil {
			return nil, err
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

func (gui *Gui) ToggleTurboMining() {
	gui.miner.ToggleTurbo()
}

// functions that allow Gui to implement interface ethlog.LogSystem
func (gui *Gui) SetLogLevel(level ethlog.LogLevel) {
	gui.logLevel = level
	gui.stdLog.SetLogLevel(level)
	gui.config.Save("loglevel", level)
}

func (gui *Gui) GetLogLevel() ethlog.LogLevel {
	return gui.logLevel
}

func (self *Gui) AddPlugin(pluginPath string) {
	self.plugins[pluginPath] = plugin{Name: "SomeName", Path: pluginPath}

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
		stateDump = self.eth.StateManager().CurrentState().Dump()
	} else {
		var block *ethchain.Block
		if hash[0] == '#' {
			i, _ := strconv.Atoi(hash[1:])
			block = self.eth.BlockChain().GetBlockByNumber(uint64(i))
		} else {
			block = self.eth.BlockChain().GetBlock(ethutil.Hex2Bytes(hash))
		}

		if block == nil {
			logger.Infof("block err: not found %s\n", hash)
			return
		}

		stateDump = block.State().Dump()
	}

	file, err := os.OpenFile(path[7:], os.O_CREATE|os.O_RDWR, os.ModePerm)
	if err != nil {
		logger.Infoln("dump err: ", err)
		return
	}
	defer file.Close()

	logger.Infof("dumped state (%s) to %s\n", hash, path)

	file.Write(stateDump)
}
func (gui *Gui) ToggleMining() {
	var txt string
	if gui.eth.Mining {
		utils.StopMining(gui.eth)
		txt = "Start mining"

		gui.getObjectByName("miningLabel").Set("visible", false)
	} else {
		utils.StartMining(gui.eth)
		gui.miner = utils.GetMiner()
		txt = "Stop mining"

		gui.getObjectByName("miningLabel").Set("visible", true)
	}

	gui.win.Root().Set("miningButtonText", txt)
}
