// Copyright 2014 The go-ethereum Authors
// This file is part of go-ethereum.
//
// go-ethereum is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// go-ethereum is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with go-ethereum. If not, see <http://www.gnu.org/licenses/>.

// geth is the official command-line client for Ethereum.
package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/codegangsta/cli"
	"github.com/ethereum/ethash"
	"github.com/ethereum/go-ethereum/cmd/utils"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/eth"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/internal/debug"
	"github.com/ethereum/go-ethereum/logger"
	"github.com/ethereum/go-ethereum/logger/glog"
	"github.com/ethereum/go-ethereum/metrics"
	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/rlp"
)

const (
	ClientIdentifier = "Geth"
	Version          = "1.5.0-unstable"
	VersionMajor     = 1
	VersionMinor     = 5
	VersionPatch     = 0
)

var (
	gitCommit       string // set via linker flagg
	nodeNameVersion string
	app             *cli.App
)

func init() {
	if gitCommit == "" {
		nodeNameVersion = Version
	} else {
		nodeNameVersion = Version + "-" + gitCommit[:8]
	}

	app = utils.NewApp(Version, "the go-ethereum command line interface")
	app.Action = geth
	app.HideVersion = true // we have a command to print the version
	app.Commands = []cli.Command{
		importCommand,
		exportCommand,
		upgradedbCommand,
		removedbCommand,
		dumpCommand,
		monitorCommand,
		accountCommand,
		walletCommand,
		{
			Action: makedag,
			Name:   "makedag",
			Usage:  "generate ethash dag (for testing)",
			Description: `
The makedag command generates an ethash DAG in /tmp/dag.

This command exists to support the system testing project.
Regular users do not need to execute it.
`,
		},
		{
			Action: gpuinfo,
			Name:   "gpuinfo",
			Usage:  "gpuinfo",
			Description: `
Prints OpenCL device info for all found GPUs.
`,
		},
		{
			Action: gpubench,
			Name:   "gpubench",
			Usage:  "benchmark GPU",
			Description: `
Runs quick benchmark on first GPU found.
`,
		},
		{
			Action: version,
			Name:   "version",
			Usage:  "print ethereum version numbers",
			Description: `
The output of this command is supposed to be machine-readable.
`,
		},
		{
			Action: initGenesis,
			Name:   "init",
			Usage:  "bootstraps and initialises a new genesis block (JSON)",
			Description: `
The init command initialises a new genesis block and definition for the network.
This is a destructive action and changes the network in which you will be
participating.
`,
		},
		{
			Action: console,
			Name:   "console",
			Usage:  `Geth Console: interactive JavaScript environment`,
			Description: `
The Geth console is an interactive shell for the JavaScript runtime environment
which exposes a node admin interface as well as the Ðapp JavaScript API.
See https://github.com/ethereum/go-ethereum/wiki/Javascipt-Console
`,
		},
		{
			Action: attach,
			Name:   "attach",
			Usage:  `Geth Console: interactive JavaScript environment (connect to node)`,
			Description: `
		The Geth console is an interactive shell for the JavaScript runtime environment
		which exposes a node admin interface as well as the Ðapp JavaScript API.
		See https://github.com/ethereum/go-ethereum/wiki/Javascipt-Console.
		This command allows to open a console on a running geth node.
		`,
		},
		{
			Action: execScripts,
			Name:   "js",
			Usage:  `executes the given JavaScript files in the Geth JavaScript VM`,
			Description: `
The JavaScript VM exposes a node admin interface as well as the Ðapp
JavaScript API. See https://github.com/ethereum/go-ethereum/wiki/Javascipt-Console
`,
		},
	}

	app.Flags = []cli.Flag{
		utils.IdentityFlag,
		utils.UnlockedAccountFlag,
		utils.PasswordFileFlag,
		utils.GenesisFileFlag,
		utils.BootnodesFlag,
		utils.DataDirFlag,
		utils.KeyStoreDirFlag,
		utils.BlockchainVersionFlag,
		utils.OlympicFlag,
		utils.FastSyncFlag,
		utils.CacheFlag,
		utils.LightKDFFlag,
		utils.JSpathFlag,
		utils.ListenPortFlag,
		utils.MaxPeersFlag,
		utils.MaxPendingPeersFlag,
		utils.EtherbaseFlag,
		utils.GasPriceFlag,
		utils.MinerThreadsFlag,
		utils.MiningEnabledFlag,
		utils.MiningGPUFlag,
		utils.AutoDAGFlag,
		utils.TargetGasLimitFlag,
		utils.NATFlag,
		utils.NatspecEnabledFlag,
		utils.NoDiscoverFlag,
		utils.NodeKeyFileFlag,
		utils.NodeKeyHexFlag,
		utils.RPCEnabledFlag,
		utils.RPCListenAddrFlag,
		utils.RPCPortFlag,
		utils.RPCApiFlag,
		utils.WSEnabledFlag,
		utils.WSListenAddrFlag,
		utils.WSPortFlag,
		utils.WSApiFlag,
		utils.WSAllowedOriginsFlag,
		utils.IPCDisabledFlag,
		utils.IPCApiFlag,
		utils.IPCPathFlag,
		utils.ExecFlag,
		utils.PreLoadJSFlag,
		utils.WhisperEnabledFlag,
		utils.DevModeFlag,
		utils.TestNetFlag,
		utils.VMForceJitFlag,
		utils.VMJitCacheFlag,
		utils.VMEnableJitFlag,
		utils.NetworkIdFlag,
		utils.RPCCORSDomainFlag,
		utils.MetricsEnabledFlag,
		utils.FakePoWFlag,
		utils.SolcPathFlag,
		utils.GpoMinGasPriceFlag,
		utils.GpoMaxGasPriceFlag,
		utils.GpoFullBlockRatioFlag,
		utils.GpobaseStepDownFlag,
		utils.GpobaseStepUpFlag,
		utils.GpobaseCorrectionFactorFlag,
		utils.ExtraDataFlag,
	}
	app.Flags = append(app.Flags, debug.Flags...)

	app.Before = func(ctx *cli.Context) error {
		runtime.GOMAXPROCS(runtime.NumCPU())
		if err := debug.Setup(ctx); err != nil {
			return err
		}
		// Start system runtime metrics collection
		go metrics.CollectProcessMetrics(3 * time.Second)

		utils.SetupNetwork(ctx)

		// Deprecation warning.
		if ctx.GlobalIsSet(utils.GenesisFileFlag.Name) {
			common.PrintDepricationWarning("--genesis is deprecated. Switch to use 'geth init /path/to/file'")
		}

		return nil
	}

	app.After = func(ctx *cli.Context) error {
		logger.Flush()
		debug.Exit()
		utils.Stdin.Close() // Resets terminal mode.
		return nil
	}
}

func main() {
	if err := app.Run(os.Args); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func makeDefaultExtra() []byte {
	var clientInfo = struct {
		Version   uint
		Name      string
		GoVersion string
		Os        string
	}{uint(VersionMajor<<16 | VersionMinor<<8 | VersionPatch), ClientIdentifier, runtime.Version(), runtime.GOOS}
	extra, err := rlp.EncodeToBytes(clientInfo)
	if err != nil {
		glog.V(logger.Warn).Infoln("error setting canonical miner information:", err)
	}

	if uint64(len(extra)) > params.MaximumExtraDataSize.Uint64() {
		glog.V(logger.Warn).Infoln("error setting canonical miner information: extra exceeds", params.MaximumExtraDataSize)
		glog.V(logger.Debug).Infof("extra: %x\n", extra)
		return nil
	}
	return extra
}

// geth is the main entry point into the system if no special subcommand is ran.
// It creates a default node based on the command line arguments and runs it in
// blocking mode, waiting for it to be shut down.
func geth(ctx *cli.Context) {
	node := utils.MakeSystemNode(ClientIdentifier, nodeNameVersion, makeDefaultExtra(), ctx)
	startNode(ctx, node)
	node.Wait()
}

// attach will connect to a running geth instance attaching a JavaScript console and to it.
func attach(ctx *cli.Context) {
	// attach to a running geth instance
	client, err := utils.NewRemoteRPCClient(ctx)
	if err != nil {
		utils.Fatalf("Unable to attach to geth: %v", err)
	}

	repl := newLightweightJSRE(
		ctx.GlobalString(utils.JSpathFlag.Name),
		client,
		ctx.GlobalString(utils.DataDirFlag.Name),
		true,
	)

	// preload user defined JS files into the console
	err = repl.preloadJSFiles(ctx)
	if err != nil {
		utils.Fatalf("unable to preload JS file %v", err)
	}

	// in case the exec flag holds a JS statement execute it and return
	if ctx.GlobalString(utils.ExecFlag.Name) != "" {
		repl.batch(ctx.GlobalString(utils.ExecFlag.Name))
	} else {
		repl.welcome()
		repl.interactive()
	}
}

// initGenesis will initialise the given JSON format genesis file and writes it as
// the zero'd block (i.e. genesis) or will fail hard if it can't succeed.
func initGenesis(ctx *cli.Context) {
	genesisPath := ctx.Args().First()
	if len(genesisPath) == 0 {
		utils.Fatalf("must supply path to genesis JSON file")
	}

	chainDb, err := ethdb.NewLDBDatabase(filepath.Join(utils.MustMakeDataDir(ctx), "chaindata"), 0, 0)
	if err != nil {
		utils.Fatalf("could not open database: %v", err)
	}

	genesisFile, err := os.Open(genesisPath)
	if err != nil {
		utils.Fatalf("failed to read genesis file: %v", err)
	}

	block, err := core.WriteGenesisBlock(chainDb, genesisFile)
	if err != nil {
		utils.Fatalf("failed to write genesis block: %v", err)
	}
	glog.V(logger.Info).Infof("successfully wrote genesis block and/or chain rule set: %x", block.Hash())
}

// console starts a new geth node, attaching a JavaScript console to it at the
// same time.
func console(ctx *cli.Context) {
	// Create and start the node based on the CLI flags
	node := utils.MakeSystemNode(ClientIdentifier, nodeNameVersion, makeDefaultExtra(), ctx)
	startNode(ctx, node)

	// Attach to the newly started node, and either execute script or become interactive
	client, err := node.Attach()
	if err != nil {
		utils.Fatalf("Failed to attach to the inproc geth: %v", err)
	}
	repl := newJSRE(node,
		ctx.GlobalString(utils.JSpathFlag.Name),
		ctx.GlobalString(utils.RPCCORSDomainFlag.Name),
		client, true)

	// preload user defined JS files into the console
	err = repl.preloadJSFiles(ctx)
	if err != nil {
		utils.Fatalf("unable to preload JS file %v", err)
	}

	// in case the exec flag holds a JS statement execute it and return
	if script := ctx.GlobalString(utils.ExecFlag.Name); script != "" {
		repl.batch(script)
	} else {
		repl.welcome()
		repl.interactive()
	}
	node.Stop()
}

// execScripts starts a new geth node based on the CLI flags, and executes each
// of the JavaScript files specified as command arguments.
func execScripts(ctx *cli.Context) {
	// Create and start the node based on the CLI flags
	node := utils.MakeSystemNode(ClientIdentifier, nodeNameVersion, makeDefaultExtra(), ctx)
	startNode(ctx, node)

	// Attach to the newly started node and execute the given scripts
	client, err := node.Attach()
	if err != nil {
		utils.Fatalf("Failed to attach to the inproc geth: %v", err)
	}
	repl := newJSRE(node,
		ctx.GlobalString(utils.JSpathFlag.Name),
		ctx.GlobalString(utils.RPCCORSDomainFlag.Name),
		client, false)

	for _, file := range ctx.Args() {
		repl.exec(file)
	}
	node.Stop()
}

// startNode boots up the system node and all registered protocols, after which
// it unlocks any requested accounts, and starts the RPC/IPC interfaces and the
// miner.
func startNode(ctx *cli.Context, stack *node.Node) {
	// Start up the node itself
	utils.StartNode(stack)

	// Unlock any account specifically requested
	var ethereum *eth.Ethereum
	if err := stack.Service(&ethereum); err != nil {
		utils.Fatalf("ethereum service not running: %v", err)
	}
	accman := ethereum.AccountManager()
	passwords := utils.MakePasswordList(ctx)

	accounts := strings.Split(ctx.GlobalString(utils.UnlockedAccountFlag.Name), ",")
	for i, account := range accounts {
		if trimmed := strings.TrimSpace(account); trimmed != "" {
			unlockAccount(ctx, accman, trimmed, i, passwords)
		}
	}
	// Start auxiliary services if enabled
	if ctx.GlobalBool(utils.MiningEnabledFlag.Name) {
		if err := ethereum.StartMining(ctx.GlobalInt(utils.MinerThreadsFlag.Name), ctx.GlobalString(utils.MiningGPUFlag.Name)); err != nil {
			utils.Fatalf("Failed to start mining: %v", err)
		}
	}
}

func makedag(ctx *cli.Context) {
	args := ctx.Args()
	wrongArgs := func() {
		utils.Fatalf(`Usage: geth makedag <block number> <outputdir>`)
	}
	switch {
	case len(args) == 2:
		blockNum, err := strconv.ParseUint(args[0], 0, 64)
		dir := args[1]
		if err != nil {
			wrongArgs()
		} else {
			dir = filepath.Clean(dir)
			// seems to require a trailing slash
			if !strings.HasSuffix(dir, "/") {
				dir = dir + "/"
			}
			_, err = ioutil.ReadDir(dir)
			if err != nil {
				utils.Fatalf("Can't find dir")
			}
			fmt.Println("making DAG, this could take awhile...")
			ethash.MakeDAG(blockNum, dir)
		}
	default:
		wrongArgs()
	}
}

func gpuinfo(ctx *cli.Context) {
	eth.PrintOpenCLDevices()
}

func gpubench(ctx *cli.Context) {
	args := ctx.Args()
	wrongArgs := func() {
		utils.Fatalf(`Usage: geth gpubench <gpu number>`)
	}
	switch {
	case len(args) == 1:
		n, err := strconv.ParseUint(args[0], 0, 64)
		if err != nil {
			wrongArgs()
		}
		eth.GPUBench(n)
	case len(args) == 0:
		eth.GPUBench(0)
	default:
		wrongArgs()
	}
}

func version(c *cli.Context) {
	fmt.Println(ClientIdentifier)
	fmt.Println("Version:", Version)
	if gitCommit != "" {
		fmt.Println("Git Commit:", gitCommit)
	}
	fmt.Println("Protocol Versions:", eth.ProtocolVersions)
	fmt.Println("Network Id:", c.GlobalInt(utils.NetworkIdFlag.Name))
	fmt.Println("Go Version:", runtime.Version())
	fmt.Println("OS:", runtime.GOOS)
	fmt.Printf("GOPATH=%s\n", os.Getenv("GOPATH"))
	fmt.Printf("GOROOT=%s\n", runtime.GOROOT())
}
