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
	"fmt"
	"os"
	"runtime"
	"strconv"
	"time"

	"github.com/codegangsta/cli"
	"github.com/ethereum/go-ethereum/cmd/ethereum/repl"
	"github.com/ethereum/go-ethereum/cmd/utils"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/eth"
	"github.com/ethereum/go-ethereum/ethutil"
	"github.com/ethereum/go-ethereum/logger"
	"github.com/ethereum/go-ethereum/state"
)

const (
	ClientIdentifier = "Ethereum(G)"
	Version          = "0.8.6"
)

var (
	clilogger = logger.NewLogger("CLI")
	app       = cli.NewApp()
)

func init() {
	app.Version = Version
	app.Usage = "the go-ethereum command-line client"
	app.Action = run
	app.HideVersion = true // we have a command to print the version
	app.Commands = []cli.Command{
		{
			Action: version,
			Name:   "version",
			Usage:  "print ethereum version numbers",
			Description: `
The output of this command is supposed to be machine-readable.
`,
		},
		{
			Action: dump,
			Name:   "dump",
			Usage:  `dump a specific block from storage`,
			Description: `
The arguments are interpreted as block numbers or hashes.
Use "ethereum dump 0" to dump the genesis block.
`,
		},
		{
			Action: runjs,
			Name:   "js",
			Usage:  `interactive JavaScript console`,
			Description: `
In the console, you can use the eth object to interact
with the running ethereum stack. The API does not match
ethereum.js.

A JavaScript file can be provided as the argument. The
runtime will execute the file and exit.
`,
		},
		{
			Action: importchain,
			Name:   "import",
			Usage:  `import a blockchain file`,
		},
	}
	app.Author = ""
	app.Email = ""
	app.Flags = []cli.Flag{
		utils.BootnodesFlag,
		utils.DataDirFlag,
		utils.KeyRingFlag,
		utils.KeyStoreFlag,
		utils.ListenPortFlag,
		utils.LogFileFlag,
		utils.LogFormatFlag,
		utils.LogLevelFlag,
		utils.MaxPeersFlag,
		utils.MinerThreadsFlag,
		utils.MiningEnabledFlag,
		utils.NATFlag,
		utils.NodeKeyFileFlag,
		utils.NodeKeyHexFlag,
		utils.RPCEnabledFlag,
		utils.RPCListenAddrFlag,
		utils.RPCPortFlag,
		utils.VMTypeFlag,
	}

	// missing:
	// flag.StringVar(&ConfigFile, "conf", defaultConfigFile, "config file")
	// flag.BoolVar(&DiffTool, "difftool", false, "creates output for diff'ing. Sets LogLevel=0")
	// flag.StringVar(&DiffType, "diff", "all", "sets the level of diff output [vm, all]. Has no effect if difftool=false")

	// potential subcommands:
	// flag.StringVar(&SecretFile, "import", "", "imports the file given (hex or mnemonic formats)")
	// flag.StringVar(&ExportDir, "export", "", "exports the session keyring to files in the directory given")
	// flag.BoolVar(&GenAddr, "genaddr", false, "create a new priv/pub key")
}

func main() {
	runtime.GOMAXPROCS(runtime.NumCPU())
	defer logger.Flush()
	utils.HandleInterrupt()
	if err := app.Run(os.Args); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(ctx *cli.Context) {
	fmt.Printf("Welcome to the FRONTIER\n")
	eth := utils.GetEthereum(ClientIdentifier, Version, ctx)
	startEth(ctx, eth)
	// this blocks the thread
	eth.WaitForShutdown()
}

func runjs(ctx *cli.Context) {
	eth := utils.GetEthereum(ClientIdentifier, Version, ctx)
	startEth(ctx, eth)
	if len(ctx.Args()) == 0 {
		repl := ethrepl.NewJSRepl(eth)
		repl.Start()
		utils.RegisterInterrupt(func(os.Signal) { repl.Stop() })
		eth.WaitForShutdown()
	} else if len(ctx.Args()) == 1 {
		execJsFile(eth, ctx.Args()[0])
	} else {
		utils.Fatalf("This command can handle at most one argument.")
	}
}

func startEth(ctx *cli.Context, eth *eth.Ethereum) {
	utils.StartEthereum(eth)
	if ctx.GlobalBool(utils.RPCEnabledFlag.Name) {
		addr := ctx.GlobalString(utils.RPCListenAddrFlag.Name)
		port := ctx.GlobalInt(utils.RPCPortFlag.Name)
		utils.StartRpc(eth, addr, port)
	}
	if ctx.GlobalBool(utils.MiningEnabledFlag.Name) {
		eth.Miner().Start()
	}
}

func importchain(ctx *cli.Context) {
	if len(ctx.Args()) != 1 {
		utils.Fatalf("This command requires an argument.")
	}
	chain, _ := utils.GetChain(ctx)
	start := time.Now()
	err := utils.ImportChain(chain, ctx.Args().First())
	if err != nil {
		utils.Fatalf("Import error: %v\n", err)
	}
	fmt.Printf("Import done in", time.Since(start))
	return
}

func dump(ctx *cli.Context) {
	chain, db := utils.GetChain(ctx)
	for _, arg := range ctx.Args() {
		var block *types.Block
		if hashish(arg) {
			block = chain.GetBlock(ethutil.Hex2Bytes(arg))
		} else {
			num, _ := strconv.Atoi(arg)
			block = chain.GetBlockByNumber(uint64(num))
		}
		if block == nil {
			fmt.Println("{}")
			utils.Fatalf("block not found")
		} else {
			statedb := state.New(block.Root(), db)
			fmt.Printf("%s\n", statedb.Dump())
			// fmt.Println(block)
		}
	}
}

// hashish returns true for strings that look like hashes.
func hashish(x string) bool {
	_, err := strconv.Atoi(x)
	return err != nil
}

func version(c *cli.Context) {
	fmt.Printf(`%v %v
PV=%d
GOOS=%s
GO=%s
GOPATH=%s
GOROOT=%s
`, ClientIdentifier, Version, eth.ProtocolVersion, runtime.GOOS, runtime.Version(), os.Getenv("GOPATH"), runtime.GOROOT())
}
