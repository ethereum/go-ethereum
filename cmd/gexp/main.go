// Copyright 2014 The go-ethereum Authors && Copyright 2015 go-expanse Authors
// This file is part of go-expanse.
//
// go-expanse is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// go-expanse is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with go-expanse. If not, see <http://www.gnu.org/licenses/>.

// gexp is the official command-line client for Expanse.
package main

import (
	"encoding/hex"
	"fmt"
	"io/ioutil"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/codegangsta/cli"
	"github.com/expanse-project/ethash"
	"github.com/expanse-project/go-expanse/cmd/utils"
	"github.com/expanse-project/go-expanse/common"
	"github.com/expanse-project/go-expanse/core"
	"github.com/expanse-project/go-expanse/exp"
	"github.com/expanse-project/go-expanse/ethdb"
	"github.com/expanse-project/go-expanse/internal/debug"
	"github.com/expanse-project/go-expanse/logger"
	"github.com/expanse-project/go-expanse/logger/glog"
	"github.com/expanse-project/go-expanse/metrics"
	"github.com/expanse-project/go-expanse/node"
	"github.com/expanse-project/go-expanse/params"
	"github.com/expanse-project/go-expanse/release"
	"github.com/expanse-project/go-expanse/rlp"
)

const (
	clientIdentifier = "Gexp"   // Client identifier to advertise over the network
	versionMajor     = 1        // Major version component of the current release
	versionMinor     = 4        // Minor version component of the current release
	versionPatch     = 5        // Patch version component of the current release
	versionMeta      = "stable" // Version metadata to append to the version string
	versionOracle = "0x926d69cc3bbf81d52cba6886d788df007a15a3cd" // Expanse address of the Gexp release oracle
)

var (
	gitCommit string         // Git SHA1 commit hash of the release (set via linker flags)
	verString string         // Combined textual representation of all the version components
	relConfig release.Config // Structured version information and release oracle config
	app       *cli.App
)

func init() {
	// Construct the textual version string from the individual components
	verString = fmt.Sprintf("%d.%d.%d", versionMajor, versionMinor, versionPatch)
	if versionMeta != "" {
		verString += "-" + versionMeta
	}
	if gitCommit != "" {
		verString += "-" + gitCommit[:8]
	}
	// Construct the version release oracle configuration
	relConfig.Oracle = common.HexToAddress(versionOracle)

	relConfig.Major = uint32(versionMajor)
	relConfig.Minor = uint32(versionMinor)
	relConfig.Patch = uint32(versionPatch)

	commit, _ := hex.DecodeString(gitCommit)
	copy(relConfig.Commit[:], commit)

	// Initialize the CLI app and start Gexp
	app = utils.NewApp(verString, "the go-expanse command line interface")
	app.Action = gexp
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
			Usage:  "print expanse version numbers",
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
			Usage:  `Gexp Console: interactive JavaScript environment`,
			Description: `
The Gexp console is an interactive shell for the JavaScript runtime environment
which exposes a node admin interface as well as the Ðapp JavaScript API.
See https://github.com/expanse-project/go-expanse/wiki/Javascipt-Console
`,
		},
		{
			Action: attach,
			Name:   "attach",
			Usage:  `Gexp Console: interactive JavaScript environment (connect to node)`,
			Description: `
		The Gexp console is an interactive shell for the JavaScript runtime environment
		which exposes a node admin interface as well as the Ðapp JavaScript API.
		See https://github.com/expanse-project/go-expanse/wiki/Javascipt-Console.
		This command allows to open a console on a running gexp node.
		`,
		},
		{
			Action: execScripts,
			Name:   "js",
			Usage:  `executes the given JavaScript files in the Gexp JavaScript VM`,
			Description: `
The JavaScript VM exposes a node admin interface as well as the Ðapp
JavaScript API. See https://github.com/expanse-project/go-expanse/wiki/Javascipt-Console
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
			common.PrintDepricationWarning("--genesis is deprecated. Switch to use 'gexp init /path/to/file'")
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
	}{uint(versionMajor<<16 | versionMinor<<8 | versionPatch), clientIdentifier, runtime.Version(), runtime.GOOS}
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

// gexp is the main entry point into the system if no special subcommand is ran.
// It creates a default node based on the command line arguments and runs it in
// blocking mode, waiting for it to be shut down.
func gexp(ctx *cli.Context) {
	node := utils.MakeSystemNode(clientIdentifier, verString, relConfig, makeDefaultExtra(), ctx)
	startNode(ctx, node)
	node.Wait()
}

// attach will connect to a running gexp instance attaching a JavaScript console and to it.
func attach(ctx *cli.Context) {
	// attach to a running gexp instance
	client, err := utils.NewRemoteRPCClient(ctx)
	if err != nil {
		utils.Fatalf("Unable to attach to gexp: %v", err)
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

// console starts a new gexp node, attaching a JavaScript console to it at the
// same time.
func console(ctx *cli.Context) {
	// Create and start the node based on the CLI flags
	node := utils.MakeSystemNode(clientIdentifier, verString, relConfig, makeDefaultExtra(), ctx)
	startNode(ctx, node)

	// Attach to the newly started node, and either execute script or become interactive
	client, err := node.Attach()
	if err != nil {
		utils.Fatalf("Failed to attach to the inproc gexp: %v", err)
	}
	repl := newJSRE(node,
		ctx.GlobalString(utils.JSpathFlag.Name),
		ctx.GlobalString(utils.RPCCORSDomainFlag.Name),
		client, true)

	// preload user defined JS files into the console
	err = repl.preloadJSFiles(ctx)
	if err != nil {
		utils.Fatalf("%v", err)
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

// execScripts starts a new gexp node based on the CLI flags, and executes each
// of the JavaScript files specified as command arguments.
func execScripts(ctx *cli.Context) {
	// Create and start the node based on the CLI flags
	node := utils.MakeSystemNode(clientIdentifier, verString, relConfig, makeDefaultExtra(), ctx)
	startNode(ctx, node)
	defer node.Stop()

	// Attach to the newly started node and execute the given scripts
	client, err := node.Attach()
	if err != nil {
		utils.Fatalf("Failed to attach to the inproc gexp: %v", err)
	}
	repl := newJSRE(node,
		ctx.GlobalString(utils.JSpathFlag.Name),
		ctx.GlobalString(utils.RPCCORSDomainFlag.Name),
		client, false)

	// Run all given files.
	for _, file := range ctx.Args() {
		if err = repl.re.Exec(file); err != nil {
			break
		}
	}
	if err != nil {
		utils.Fatalf("JavaScript Error: %v", jsErrorString(err))
	}
	// JS files loaded successfully.
	// Wait for pending callbacks, but stop for Ctrl-C.
	abort := make(chan os.Signal, 1)
	signal.Notify(abort, os.Interrupt)
	go func() {
		<-abort
		repl.re.Stop(false)
	}()
	repl.re.Stop(true)
}

// startNode boots up the system node and all registered protocols, after which
// it unlocks any requested accounts, and starts the RPC/IPC interfaces and the
// miner.
func startNode(ctx *cli.Context, stack *node.Node) {
	// Start up the node itself
	utils.StartNode(stack)

	// Unlock any account specifically requested
	var expanse *exp.Expanse
	if err := stack.Service(&expanse); err != nil {
		utils.Fatalf("ethereum service not running: %v", err)
	}
	accman := expanse.AccountManager()
	passwords := utils.MakePasswordList(ctx)

	accounts := strings.Split(ctx.GlobalString(utils.UnlockedAccountFlag.Name), ",")
	for i, account := range accounts {
		if trimmed := strings.TrimSpace(account); trimmed != "" {
			unlockAccount(ctx, accman, trimmed, i, passwords)
		}
	}
	// Start auxiliary services if enabled
	if ctx.GlobalBool(utils.MiningEnabledFlag.Name) {
		if err := expanse.StartMining(ctx.GlobalInt(utils.MinerThreadsFlag.Name), ctx.GlobalString(utils.MiningGPUFlag.Name)); err != nil {
			utils.Fatalf("Failed to start mining: %v", err)
		}
	}
}

func makedag(ctx *cli.Context) {
	args := ctx.Args()
	wrongArgs := func() {
		utils.Fatalf(`Usage: gexp makedag <block number> <outputdir>`)
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
	exp.PrintOpenCLDevices()
}

func gpubench(ctx *cli.Context) {
	args := ctx.Args()
	wrongArgs := func() {
		utils.Fatalf(`Usage: gexp gpubench <gpu number>`)
	}
	switch {
	case len(args) == 1:
		n, err := strconv.ParseUint(args[0], 0, 64)
		if err != nil {
			wrongArgs()
		}
		exp.GPUBench(n)
	case len(args) == 0:
		exp.GPUBench(0)
	default:
		wrongArgs()
	}
}

func version(c *cli.Context) {
	fmt.Println(clientIdentifier)
	fmt.Println("Version:", verString)
	fmt.Println("Protocol Versions:", exp.ProtocolVersions)
	fmt.Println("Network Id:", c.GlobalInt(utils.NetworkIdFlag.Name))
	fmt.Println("Go Version:", runtime.Version())
	fmt.Println("OS:", runtime.GOOS)
	fmt.Printf("GOPATH=%s\n", os.Getenv("GOPATH"))
	fmt.Printf("GOROOT=%s\n", runtime.GOROOT())
}
