package main

import (
	"fmt"
	"github.com/ethereum/go-ethereum/ethereal/ui"
	"github.com/ethereum/go-ethereum/utils"
	"github.com/go-qml/qml"
	"runtime"
)

const Debug = true

func main() {
	qml.Init(nil)

	runtime.GOMAXPROCS(runtime.NumCPU())

	// precedence: code-internal flag default < config file < environment variables < command line
	Init() // parsing command line
	utils.InitConfig(ConfigFile, Datadir, Identifier, "ETH")

	utils.InitDataDir(Datadir)

	utils.InitLogging(Datadir, LogFile, LogLevel, DebugFile)

	ethereum := utils.NewEthereum(UseUPnP, OutboundPort, MaxPeer)

	// create, import, export keys
  utils.KeyTasks(GenAddr, ImportKey, ExportKey, NonInteractive)

  if ShowGenesis {
    utils.ShowGenesis(ethereum)
  }

	if StartRpc {
		utils.StartRpc(ethereum, RpcPort)
	}

	utils.StartEthereum(ethereum, UseSeed)

	gui := ethui.New(ethereum, logLevel)
	gui.Start(AssetPath)
}
