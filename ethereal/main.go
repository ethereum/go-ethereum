package main

import (
	"github.com/ethereum/go-ethereum/ethereal/ui"
	"github.com/ethereum/go-ethereum/utils"
	"github.com/go-qml/qml"
	"runtime"
)

func main() {
	runtime.GOMAXPROCS(runtime.NumCPU())

	utils.HandleInterrupt()

	qml.Init(nil)

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

	gui := ethui.New(ethereum, LogLevel)
	gui.Start(AssetPath)

	utils.StartEthereum(ethereum, UseSeed)


}
