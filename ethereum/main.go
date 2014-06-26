package main

import (
	"github.com/ethereum/go-ethereum/utils"
	"github.com/ethereum/eth-go/ethlog"
	"runtime"
)

var logger = ethlog.NewLogger("CLI")

func main() {
	runtime.GOMAXPROCS(runtime.NumCPU())

	utils.HandleInterrupt()

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

	if StartMining {
		utils.StartMining(ethereum)
	}

	// better reworked as cases
	if StartJsConsole {
		InitJsConsole(ethereum)
	} else if len(InputFile) > 0 {
		ExecJsFile(ethereum, InputFile)
	}

	if StartRpc {
		utils.StartRpc(ethereum, RpcPort)
	}

	utils.StartEthereum(ethereum, UseSeed)
}
