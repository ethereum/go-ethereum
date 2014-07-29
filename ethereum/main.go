package main

import (
	"runtime"

	"github.com/ethereum/eth-go/ethlog"
	"github.com/ethereum/eth-go/ethutil"
	"github.com/ethereum/go-ethereum/utils"
)

const (
	ClientIdentifier = "Ethereum(G)"
	Version          = "0.6.1"
)

var logger = ethlog.NewLogger("CLI")

func main() {
	runtime.GOMAXPROCS(runtime.NumCPU())

	utils.HandleInterrupt()

	// precedence: code-internal flag default < config file < environment variables < command line
	Init() // parsing command line

	// If the difftool option is selected ignore all other log output
	if DiffTool {
		LogLevel = 0
	}

	utils.InitConfig(ConfigFile, Datadir, "ETH")
	ethutil.Config.Diff = DiffTool
	ethutil.Config.DiffType = DiffType

	utils.InitDataDir(Datadir)

	utils.InitLogging(Datadir, LogFile, LogLevel, DebugFile)

	db := utils.NewDatabase()

	keyManager := utils.NewKeyManager(KeyStore, Datadir, db)

	// create, import, export keys
	utils.KeyTasks(keyManager, KeyRing, GenAddr, SecretFile, ExportDir, NonInteractive)

	clientIdentity := utils.NewClientIdentity(ClientIdentifier, Version, Identifier)

	ethereum := utils.NewEthereum(db, clientIdentity, keyManager, UseUPnP, OutboundPort, MaxPeer)

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
