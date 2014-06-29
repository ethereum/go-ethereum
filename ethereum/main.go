package main

import (
	"github.com/ethereum/eth-go/ethlog"
	"github.com/ethereum/go-ethereum/utils"
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

	db := utils.NewDatabase()

	keyManager := utils.NewKeyManager(KeyStore, Datadir, db)

	// create, import, export keys
	utils.KeyTasks(keyManager, KeyRing, GenAddr, SecretFile, ExportDir, NonInteractive)

	ethereum := utils.NewEthereum(db, keyManager, UseUPnP, OutboundPort, MaxPeer)

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

	// this blocks the thread
	ethereum.WaitForShutdown()
	ethlog.Flush()
}
