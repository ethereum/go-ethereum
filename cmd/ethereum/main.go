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
	"bytes"
	"fmt"
	"os"
	"runtime"

	"github.com/ethereum/go-ethereum/cmd/utils"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethutil"
	"github.com/ethereum/go-ethereum/logger"
	"github.com/ethereum/go-ethereum/rlp"
)

const (
	ClientIdentifier = "Ethereum(G)"
	Version          = "0.7.10"
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

		// block.GetRoot() does not exist
		//fmt.Printf("RLP: %x\nstate: %x\nhash: %x\n", ethutil.Rlp(block), block.GetRoot(), block.Hash())

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
		clilogger.Infof("importing chain '%s'\n", ImportChain)
		c, err := ethutil.ReadAllFile(ImportChain)
		if err != nil {
			clilogger.Infoln(err)
			return
		}
		var chain types.Blocks
		if err := rlp.Decode(bytes.NewReader([]byte(c)), &chain); err != nil {
			clilogger.Infoln(err)
			return
		}

		ethereum.ChainManager().Reset()
		if err := ethereum.ChainManager().InsertChain(chain); err != nil {
			clilogger.Infoln(err)
			return
		}
		clilogger.Infof("imported %d blocks\n", len(chain))
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
