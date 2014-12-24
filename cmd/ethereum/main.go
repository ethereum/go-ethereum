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
	"fmt"
	"os"
	"runtime"
	"time"

	"github.com/ethereum/go-ethereum/cmd/utils"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethutil"
	"github.com/ethereum/go-ethereum/logger"
)

const (
	ClientIdentifier = "Ethereum(G)"
	Version          = "0.7.11"
)

var clilogger = logger.NewLogger("CLI")

func main() {
	runtime.GOMAXPROCS(runtime.NumCPU())

	defer func() {
		logger.Flush()
	}()

	utils.HandleInterrupt()

	// precedence: code-internal flag default < config file < environment variables < command line
	Init() // parsing command line

	// If the difftool option is selected ignore all other log output
	if DiffTool || Dump {
		LogLevel = 0
	}

	utils.InitConfig(VmType, ConfigFile, Datadir, "ETH")
	ethutil.Config.Diff = DiffTool
	ethutil.Config.DiffType = DiffType

	utils.InitDataDir(Datadir)

	utils.InitLogging(Datadir, LogFile, LogLevel, DebugFile)

	db := utils.NewDatabase()
	err := utils.DBSanityCheck(db)
	if err != nil {
		fmt.Println(err)

		os.Exit(1)
	}

	keyManager := utils.NewKeyManager(KeyStore, Datadir, db)

	// create, import, export keys
	utils.KeyTasks(keyManager, KeyRing, GenAddr, SecretFile, ExportDir, NonInteractive)

	clientIdentity := utils.NewClientIdentity(ClientIdentifier, Version, Identifier, string(keyManager.PublicKey()))

	ethereum := utils.NewEthereum(db, clientIdentity, keyManager, utils.NatType(NatType, PMPGateway), OutboundPort, MaxPeer)

	if Dump {
		var block *types.Block

		if len(DumpHash) == 0 && DumpNumber == -1 {
			block = ethereum.ChainManager().CurrentBlock()
		} else if len(DumpHash) > 0 {
			block = ethereum.ChainManager().GetBlock(ethutil.Hex2Bytes(DumpHash))
		} else {
			block = ethereum.ChainManager().GetBlockByNumber(uint64(DumpNumber))
		}

		if block == nil {
			fmt.Fprintln(os.Stderr, "block not found")

			// We want to output valid JSON
			fmt.Println("{}")

			os.Exit(1)
		}

		// Leave the Println. This needs clean output for piping
		fmt.Printf("%s\n", block.State().Dump())

		fmt.Println(block)

		os.Exit(0)
	}

	if ShowGenesis {
		utils.ShowGenesis(ethereum)
	}

	if StartMining {
		utils.StartMining(ethereum)
	}

	if len(ImportChain) > 0 {
		start := time.Now()
		err := utils.ImportChain(ethereum, ImportChain)
		if err != nil {
			clilogger.Infoln(err)
		}
		clilogger.Infoln("export done in", time.Since(start))
		return
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

	if StartWebSockets {
		utils.StartWebSockets(ethereum)
	}

	utils.StartEthereum(ethereum, UseSeed)

	// this blocks the thread
	ethereum.WaitForShutdown()
}
