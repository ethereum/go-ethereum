// Copyright 2015 The go-ethereum Authors
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

package utils

import (
	"crypto/ecdsa"
	"fmt"
	"io/ioutil"
	"log"
	"math"
	"math/big"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"

	"github.com/codegangsta/cli"
	"github.com/ethereum/ethash"
	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/eth"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/event"
	"github.com/ethereum/go-ethereum/logger"
	"github.com/ethereum/go-ethereum/logger/glog"
	"github.com/ethereum/go-ethereum/metrics"
	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/p2p/discover"
	"github.com/ethereum/go-ethereum/p2p/nat"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/rpc/api"
	"github.com/ethereum/go-ethereum/rpc/codec"
	"github.com/ethereum/go-ethereum/rpc/comms"
	"github.com/ethereum/go-ethereum/rpc/shared"
	"github.com/ethereum/go-ethereum/rpc/useragent"
	"github.com/ethereum/go-ethereum/whisper"
	"github.com/ethereum/go-ethereum/xeth"
)

func init() {
	cli.AppHelpTemplate = `{{.Name}} {{if .Flags}}[global options] {{end}}command{{if .Flags}} [command options]{{end}} [arguments...]

VERSION:
   {{.Version}}

COMMANDS:
   {{range .Commands}}{{.Name}}{{with .ShortName}}, {{.}}{{end}}{{ "\t" }}{{.Usage}}
   {{end}}{{if .Flags}}
GLOBAL OPTIONS:
   {{range .Flags}}{{.}}
   {{end}}{{end}}
`

	cli.CommandHelpTemplate = `{{.Name}}{{if .Subcommands}} command{{end}}{{if .Flags}} [command options]{{end}} [arguments...]
{{if .Description}}{{.Description}}
{{end}}{{if .Subcommands}}
SUBCOMMANDS:
	{{range .Subcommands}}{{.Name}}{{with .ShortName}}, {{.}}{{end}}{{ "\t" }}{{.Usage}}
	{{end}}{{end}}{{if .Flags}}
OPTIONS:
	{{range .Flags}}{{.}}
	{{end}}{{end}}
`
}

// NewApp creates an app with sane defaults.
func NewApp(version, usage string) *cli.App {
	app := cli.NewApp()
	app.Name = filepath.Base(os.Args[0])
	app.Author = ""
	//app.Authors = nil
	app.Email = ""
	app.Version = version
	app.Usage = usage
	return app
}

// These are all the command line flags we support.
// If you add to this list, please remember to include the
// flag in the appropriate command definition.
//
// The flags are defined here so their names and help texts
// are the same for all commands.

var (
	// General settings
	DataDirFlag = DirectoryFlag{
		Name:  "datadir",
		Usage: "Data directory for the databases and keystore",
		Value: DirectoryString{common.DefaultDataDir()},
	}
	NetworkIdFlag = cli.IntFlag{
		Name:  "networkid",
		Usage: "Network identifier (integer, 0=Olympic, 1=Frontier, 2=Morden)",
		Value: eth.NetworkId,
	}
	OlympicFlag = cli.BoolFlag{
		Name:  "olympic",
		Usage: "Olympic network: pre-configured pre-release test network",
	}
	TestNetFlag = cli.BoolFlag{
		Name:  "testnet",
		Usage: "Morden network: pre-configured test network with modified starting nonces (replay protection)",
	}
	DevModeFlag = cli.BoolFlag{
		Name:  "dev",
		Usage: "Developer mode: pre-configured private network with several debugging flags",
	}
	GenesisFileFlag = cli.StringFlag{
		Name:  "genesis",
		Usage: "Insert/overwrite the genesis block (JSON format)",
	}
	IdentityFlag = cli.StringFlag{
		Name:  "identity",
		Usage: "Custom node name",
	}
	NatspecEnabledFlag = cli.BoolFlag{
		Name:  "natspec",
		Usage: "Enable NatSpec confirmation notice",
	}
	DocRootFlag = DirectoryFlag{
		Name:  "docroot",
		Usage: "Document Root for HTTPClient file scheme",
		Value: DirectoryString{common.HomeDir()},
	}
	CacheFlag = cli.IntFlag{
		Name:  "cache",
		Usage: "Megabytes of memory allocated to internal caching (min 16MB / database forced)",
		Value: 0,
	}
	BlockchainVersionFlag = cli.IntFlag{
		Name:  "blockchainversion",
		Usage: "Blockchain version (integer)",
		Value: core.BlockChainVersion,
	}
	FastSyncFlag = cli.BoolFlag{
		Name:  "fast",
		Usage: "Enable fast syncing through state downloads",
	}
	LightKDFFlag = cli.BoolFlag{
		Name:  "lightkdf",
		Usage: "Reduce key-derivation RAM & CPU usage at some expense of KDF strength",
	}
	// Miner settings
	// TODO: refactor CPU vs GPU mining flags
	MiningEnabledFlag = cli.BoolFlag{
		Name:  "mine",
		Usage: "Enable mining",
	}
	MinerThreadsFlag = cli.IntFlag{
		Name:  "minerthreads",
		Usage: "Number of CPU threads to use for mining",
		Value: runtime.NumCPU(),
	}
	MiningGPUFlag = cli.StringFlag{
		Name:  "minergpus",
		Usage: "List of GPUs to use for mining (e.g. '0,1' will use the first two GPUs found)",
	}
	AutoDAGFlag = cli.BoolFlag{
		Name:  "autodag",
		Usage: "Enable automatic DAG pregeneration",
	}
	EtherbaseFlag = cli.StringFlag{
		Name:  "etherbase",
		Usage: "Public address for block mining rewards (default = first account created)",
		Value: "0",
	}
	GasPriceFlag = cli.StringFlag{
		Name:  "gasprice",
		Usage: "Minimal gas price to accept for mining a transactions",
		Value: new(big.Int).Mul(big.NewInt(50), common.Shannon).String(),
	}
	ExtraDataFlag = cli.StringFlag{
		Name:  "extradata",
		Usage: "Block extra data set by the miner (default = client version)",
	}
	// Account settings
	UnlockedAccountFlag = cli.StringFlag{
		Name:  "unlock",
		Usage: "Comma separated list of accounts to unlock",
		Value: "",
	}
	PasswordFileFlag = cli.StringFlag{
		Name:  "password",
		Usage: "Password file to use for non-inteactive password input",
		Value: "",
	}

	// vm flags
	VMDebugFlag = cli.BoolFlag{
		Name:  "vmdebug",
		Usage: "Virtual Machine debug output",
	}
	VMForceJitFlag = cli.BoolFlag{
		Name:  "forcejit",
		Usage: "Force the JIT VM to take precedence",
	}
	VMJitCacheFlag = cli.IntFlag{
		Name:  "jitcache",
		Usage: "Amount of cached JIT VM programs",
		Value: 64,
	}
	VMEnableJitFlag = cli.BoolFlag{
		Name:  "jitvm",
		Usage: "Enable the JIT VM",
	}

	// logging and debug settings
	VerbosityFlag = cli.IntFlag{
		Name:  "verbosity",
		Usage: "Logging verbosity: 0-6 (0=silent, 1=error, 2=warn, 3=info, 4=core, 5=debug, 6=debug detail)",
		Value: int(logger.InfoLevel),
	}
	LogFileFlag = cli.StringFlag{
		Name:  "logfile",
		Usage: "Log output file within the data dir (default = no log file generated)",
		Value: "",
	}
	LogVModuleFlag = cli.GenericFlag{
		Name:  "vmodule",
		Usage: "Per-module verbosity: comma-separated list of <module>=<level>, where <module> is file literal or a glog pattern",
		Value: glog.GetVModule(),
	}
	BacktraceAtFlag = cli.GenericFlag{
		Name:  "backtrace",
		Usage: "Request a stack trace at a specific logging statement (e.g. \"block.go:271\")",
		Value: glog.GetTraceLocation(),
	}
	PProfEanbledFlag = cli.BoolFlag{
		Name:  "pprof",
		Usage: "Enable the profiling server on localhost",
	}
	PProfPortFlag = cli.IntFlag{
		Name:  "pprofport",
		Usage: "Profile server listening port",
		Value: 6060,
	}
	MetricsEnabledFlag = cli.BoolFlag{
		Name:  metrics.MetricsEnabledFlag,
		Usage: "Enable metrics collection and reporting",
	}

	// RPC settings
	RPCEnabledFlag = cli.BoolFlag{
		Name:  "rpc",
		Usage: "Enable the HTTP-RPC server",
	}
	RPCListenAddrFlag = cli.StringFlag{
		Name:  "rpcaddr",
		Usage: "HTTP-RPC server listening interface",
		Value: "127.0.0.1",
	}
	RPCPortFlag = cli.IntFlag{
		Name:  "rpcport",
		Usage: "HTTP-RPC server listening port",
		Value: 8545,
	}
	RPCCORSDomainFlag = cli.StringFlag{
		Name:  "rpccorsdomain",
		Usage: "Domains from which to accept cross origin requests (browser enforced)",
		Value: "",
	}
	RpcApiFlag = cli.StringFlag{
		Name:  "rpcapi",
		Usage: "API's offered over the HTTP-RPC interface",
		Value: comms.DefaultHttpRpcApis,
	}
	IPCDisabledFlag = cli.BoolFlag{
		Name:  "ipcdisable",
		Usage: "Disable the IPC-RPC server",
	}
	IPCApiFlag = cli.StringFlag{
		Name:  "ipcapi",
		Usage: "API's offered over the IPC-RPC interface",
		Value: comms.DefaultIpcApis,
	}
	IPCPathFlag = DirectoryFlag{
		Name:  "ipcpath",
		Usage: "Filename for IPC socket/pipe",
		Value: DirectoryString{common.DefaultIpcPath()},
	}
	ExecFlag = cli.StringFlag{
		Name:  "exec",
		Usage: "Execute JavaScript statement (only in combination with console/attach)",
	}
	// Network Settings
	MaxPeersFlag = cli.IntFlag{
		Name:  "maxpeers",
		Usage: "Maximum number of network peers (network disabled if set to 0)",
		Value: 25,
	}
	MaxPendingPeersFlag = cli.IntFlag{
		Name:  "maxpendpeers",
		Usage: "Maximum number of pending connection attempts (defaults used if set to 0)",
		Value: 0,
	}
	ListenPortFlag = cli.IntFlag{
		Name:  "port",
		Usage: "Network listening port",
		Value: 30303,
	}
	BootnodesFlag = cli.StringFlag{
		Name:  "bootnodes",
		Usage: "Comma separated enode URLs for P2P discovery bootstrap",
		Value: "",
	}
	NodeKeyFileFlag = cli.StringFlag{
		Name:  "nodekey",
		Usage: "P2P node key file",
	}
	NodeKeyHexFlag = cli.StringFlag{
		Name:  "nodekeyhex",
		Usage: "P2P node key as hex (for testing)",
	}
	NATFlag = cli.StringFlag{
		Name:  "nat",
		Usage: "NAT port mapping mechanism (any|none|upnp|pmp|extip:<IP>)",
		Value: "any",
	}
	NoDiscoverFlag = cli.BoolFlag{
		Name:  "nodiscover",
		Usage: "Disables the peer discovery mechanism (manual peer addition)",
	}
	WhisperEnabledFlag = cli.BoolFlag{
		Name:  "shh",
		Usage: "Enable Whisper",
	}
	// ATM the url is left to the user and deployment to
	JSpathFlag = cli.StringFlag{
		Name:  "jspath",
		Usage: "JavaSript root path for `loadScript` and document root for `admin.httpGet`",
		Value: ".",
	}
	SolcPathFlag = cli.StringFlag{
		Name:  "solc",
		Usage: "Solidity compiler command to be used",
		Value: "solc",
	}

	// Gas price oracle settings
	GpoMinGasPriceFlag = cli.StringFlag{
		Name:  "gpomin",
		Usage: "Minimum suggested gas price",
		Value: new(big.Int).Mul(big.NewInt(50), common.Shannon).String(),
	}
	GpoMaxGasPriceFlag = cli.StringFlag{
		Name:  "gpomax",
		Usage: "Maximum suggested gas price",
		Value: new(big.Int).Mul(big.NewInt(500), common.Shannon).String(),
	}
	GpoFullBlockRatioFlag = cli.IntFlag{
		Name:  "gpofull",
		Usage: "Full block threshold for gas price calculation (%)",
		Value: 80,
	}
	GpobaseStepDownFlag = cli.IntFlag{
		Name:  "gpobasedown",
		Usage: "Suggested gas price base step down ratio (1/1000)",
		Value: 10,
	}
	GpobaseStepUpFlag = cli.IntFlag{
		Name:  "gpobaseup",
		Usage: "Suggested gas price base step up ratio (1/1000)",
		Value: 100,
	}
	GpobaseCorrectionFactorFlag = cli.IntFlag{
		Name:  "gpobasecf",
		Usage: "Suggested gas price base correction factor (%)",
		Value: 110,
	}
)

// MustMakeDataDir retrieves the currently requested data directory, terminating
// if none (or the empty string) is specified. If the node is starting a testnet,
// the a subdirectory of the specified datadir will be used.
func MustMakeDataDir(ctx *cli.Context) string {
	if path := ctx.GlobalString(DataDirFlag.Name); path != "" {
		if ctx.GlobalBool(TestNetFlag.Name) {
			return filepath.Join(path, "/testnet")
		}
		return path
	}
	Fatalf("Cannot determine default data directory, please set manually (--datadir)")
	return ""
}

// MakeNodeKey creates a node key from set command line flags, either loading it
// from a file or as a specified hex value. If neither flags were provided, this
// method returns nil and an emphemeral key is to be generated.
func MakeNodeKey(ctx *cli.Context) *ecdsa.PrivateKey {
	var (
		hex  = ctx.GlobalString(NodeKeyHexFlag.Name)
		file = ctx.GlobalString(NodeKeyFileFlag.Name)

		key *ecdsa.PrivateKey
		err error
	)
	switch {
	case file != "" && hex != "":
		Fatalf("Options %q and %q are mutually exclusive", NodeKeyFileFlag.Name, NodeKeyHexFlag.Name)

	case file != "":
		if key, err = crypto.LoadECDSA(file); err != nil {
			Fatalf("Option %q: %v", NodeKeyFileFlag.Name, err)
		}

	case hex != "":
		if key, err = crypto.HexToECDSA(hex); err != nil {
			Fatalf("Option %q: %v", NodeKeyHexFlag.Name, err)
		}
	}
	return key
}

// MakeNodeName creates a node name from a base set and the command line flags.
func MakeNodeName(client, version string, ctx *cli.Context) string {
	name := common.MakeName(client, version)
	if identity := ctx.GlobalString(IdentityFlag.Name); len(identity) > 0 {
		name += "/" + identity
	}
	if ctx.GlobalBool(VMEnableJitFlag.Name) {
		name += "/JIT"
	}
	return name
}

// MakeBootstrapNodes creates a list of bootstrap nodes from the command line
// flags, reverting to pre-configured ones if none have been specified.
func MakeBootstrapNodes(ctx *cli.Context) []*discover.Node {
	// Return pre-configured nodes if none were manually requested
	if !ctx.GlobalIsSet(BootnodesFlag.Name) {
		if ctx.GlobalBool(TestNetFlag.Name) {
			return TestNetBootNodes
		}
		return FrontierBootNodes
	}
	// Otherwise parse and use the CLI bootstrap nodes
	bootnodes := []*discover.Node{}

	for _, url := range strings.Split(ctx.GlobalString(BootnodesFlag.Name), ",") {
		node, err := discover.ParseNode(url)
		if err != nil {
			glog.V(logger.Error).Infof("Bootstrap URL %s: %v\n", url, err)
			continue
		}
		bootnodes = append(bootnodes, node)
	}
	return bootnodes
}

// MakeListenAddress creates a TCP listening address string from set command
// line flags.
func MakeListenAddress(ctx *cli.Context) string {
	return fmt.Sprintf(":%d", ctx.GlobalInt(ListenPortFlag.Name))
}

// MakeNAT creates a port mapper from set command line flags.
func MakeNAT(ctx *cli.Context) nat.Interface {
	natif, err := nat.Parse(ctx.GlobalString(NATFlag.Name))
	if err != nil {
		Fatalf("Option %s: %v", NATFlag.Name, err)
	}
	return natif
}

// MakeGenesisBlock loads up a genesis block from an input file specified in the
// command line, or returns the empty string if none set.
func MakeGenesisBlock(ctx *cli.Context) string {
	genesis := ctx.GlobalString(GenesisFileFlag.Name)
	if genesis == "" {
		return ""
	}
	data, err := ioutil.ReadFile(genesis)
	if err != nil {
		Fatalf("Failed to load custom genesis file: %v", err)
	}
	return string(data)
}

// MakeAccountManager creates an account manager from set command line flags.
func MakeAccountManager(ctx *cli.Context) *accounts.Manager {
	// Create the keystore crypto primitive, light if requested
	scryptN := crypto.StandardScryptN
	scryptP := crypto.StandardScryptP

	if ctx.GlobalBool(LightKDFFlag.Name) {
		scryptN = crypto.LightScryptN
		scryptP = crypto.LightScryptP
	}
	// Assemble an account manager using the configured datadir
	var (
		datadir  = MustMakeDataDir(ctx)
		keystore = crypto.NewKeyStorePassphrase(filepath.Join(datadir, "keystore"), scryptN, scryptP)
	)
	return accounts.NewManager(keystore)
}

// MakeAddress converts an account specified directly as a hex encoded string or
// a key index in the key store to an internal account representation.
func MakeAddress(accman *accounts.Manager, account string) (a common.Address, err error) {
	// If the specified account is a valid address, return it
	if common.IsHexAddress(account) {
		return common.HexToAddress(account), nil
	}
	// Otherwise try to interpret the account as a keystore index
	index, err := strconv.Atoi(account)
	if err != nil {
		return a, fmt.Errorf("invalid account address or index %q", account)
	}
	hex, err := accman.AddressByIndex(index)
	if err != nil {
		return a, fmt.Errorf("can't get account #%d (%v)", index, err)
	}
	return common.HexToAddress(hex), nil
}

// MakeEtherbase retrieves the etherbase either from the directly specified
// command line flags or from the keystore if CLI indexed.
func MakeEtherbase(accman *accounts.Manager, ctx *cli.Context) common.Address {
	accounts, _ := accman.Accounts()
	if !ctx.GlobalIsSet(EtherbaseFlag.Name) && len(accounts) == 0 {
		glog.V(logger.Error).Infoln("WARNING: No etherbase set and no accounts found as default")
		return common.Address{}
	}
	etherbase := ctx.GlobalString(EtherbaseFlag.Name)
	if etherbase == "" {
		return common.Address{}
	}
	// If the specified etherbase is a valid address, return it
	addr, err := MakeAddress(accman, etherbase)
	if err != nil {
		Fatalf("Option %q: %v", EtherbaseFlag.Name, err)
	}
	return addr
}

// MakeMinerExtra resolves extradata for the miner from the set command line flags
// or returns a default one composed on the client, runtime and OS metadata.
func MakeMinerExtra(extra []byte, ctx *cli.Context) []byte {
	if ctx.GlobalIsSet(ExtraDataFlag.Name) {
		return []byte(ctx.GlobalString(ExtraDataFlag.Name))
	}
	return extra
}

// MakePasswordList loads up a list of password from a file specified by the
// command line flags.
func MakePasswordList(ctx *cli.Context) []string {
	if path := ctx.GlobalString(PasswordFileFlag.Name); path != "" {
		blob, err := ioutil.ReadFile(path)
		if err != nil {
			Fatalf("Failed to read password file: %v", err)
		}
		return strings.Split(string(blob), "\n")
	}
	return nil
}

// MakeSystemNode sets up a local node, configures the services to launch and
// assembles the P2P protocol stack.
func MakeSystemNode(name, version string, extra []byte, ctx *cli.Context) *node.Node {
	// Avoid conflicting network flags
	networks, netFlags := 0, []cli.BoolFlag{DevModeFlag, TestNetFlag, OlympicFlag}
	for _, flag := range netFlags {
		if ctx.GlobalBool(flag.Name) {
			networks++
		}
	}
	if networks > 1 {
		Fatalf("The %v flags are mutually exclusive", netFlags)
	}
	// Configure the node's service container
	stackConf := &node.Config{
		DataDir:         MustMakeDataDir(ctx),
		PrivateKey:      MakeNodeKey(ctx),
		Name:            MakeNodeName(name, version, ctx),
		NoDiscovery:     ctx.GlobalBool(NoDiscoverFlag.Name),
		BootstrapNodes:  MakeBootstrapNodes(ctx),
		ListenAddr:      MakeListenAddress(ctx),
		NAT:             MakeNAT(ctx),
		MaxPeers:        ctx.GlobalInt(MaxPeersFlag.Name),
		MaxPendingPeers: ctx.GlobalInt(MaxPendingPeersFlag.Name),
	}
	// Configure the Ethereum service
	accman := MakeAccountManager(ctx)

	ethConf := &eth.Config{
		Genesis:                 MakeGenesisBlock(ctx),
		FastSync:                ctx.GlobalBool(FastSyncFlag.Name),
		BlockChainVersion:       ctx.GlobalInt(BlockchainVersionFlag.Name),
		DatabaseCache:           ctx.GlobalInt(CacheFlag.Name),
		NetworkId:               ctx.GlobalInt(NetworkIdFlag.Name),
		AccountManager:          accman,
		Etherbase:               MakeEtherbase(accman, ctx),
		MinerThreads:            ctx.GlobalInt(MinerThreadsFlag.Name),
		ExtraData:               MakeMinerExtra(extra, ctx),
		NatSpec:                 ctx.GlobalBool(NatspecEnabledFlag.Name),
		DocRoot:                 ctx.GlobalString(DocRootFlag.Name),
		GasPrice:                common.String2Big(ctx.GlobalString(GasPriceFlag.Name)),
		GpoMinGasPrice:          common.String2Big(ctx.GlobalString(GpoMinGasPriceFlag.Name)),
		GpoMaxGasPrice:          common.String2Big(ctx.GlobalString(GpoMaxGasPriceFlag.Name)),
		GpoFullBlockRatio:       ctx.GlobalInt(GpoFullBlockRatioFlag.Name),
		GpobaseStepDown:         ctx.GlobalInt(GpobaseStepDownFlag.Name),
		GpobaseStepUp:           ctx.GlobalInt(GpobaseStepUpFlag.Name),
		GpobaseCorrectionFactor: ctx.GlobalInt(GpobaseCorrectionFactorFlag.Name),
		SolcPath:                ctx.GlobalString(SolcPathFlag.Name),
		AutoDAG:                 ctx.GlobalBool(AutoDAGFlag.Name) || ctx.GlobalBool(MiningEnabledFlag.Name),
	}
	// Configure the Whisper service
	shhEnable := ctx.GlobalBool(WhisperEnabledFlag.Name)

	// Override any default configs in dev mode or the test net
	switch {
	case ctx.GlobalBool(OlympicFlag.Name):
		if !ctx.GlobalIsSet(NetworkIdFlag.Name) {
			ethConf.NetworkId = 1
		}
		if !ctx.GlobalIsSet(GenesisFileFlag.Name) {
			ethConf.Genesis = core.OlympicGenesisBlock()
		}

	case ctx.GlobalBool(TestNetFlag.Name):
		if !ctx.GlobalIsSet(NetworkIdFlag.Name) {
			ethConf.NetworkId = 2
		}
		if !ctx.GlobalIsSet(GenesisFileFlag.Name) {
			ethConf.Genesis = core.TestNetGenesisBlock()
		}
		state.StartingNonce = 1048576 // (2**20)

	case ctx.GlobalBool(DevModeFlag.Name):
		// Override the base network stack configs
		if !ctx.GlobalIsSet(DataDirFlag.Name) {
			stackConf.DataDir = filepath.Join(os.TempDir(), "/ethereum_dev_mode")
		}
		if !ctx.GlobalIsSet(MaxPeersFlag.Name) {
			stackConf.MaxPeers = 0
		}
		if !ctx.GlobalIsSet(ListenPortFlag.Name) {
			stackConf.ListenAddr = ":0"
		}
		// Override the Ethereum protocol configs
		if !ctx.GlobalIsSet(GenesisFileFlag.Name) {
			ethConf.Genesis = core.OlympicGenesisBlock()
		}
		if !ctx.GlobalIsSet(GasPriceFlag.Name) {
			ethConf.GasPrice = new(big.Int)
		}
		if !ctx.GlobalIsSet(WhisperEnabledFlag.Name) {
			shhEnable = true
		}
		if !ctx.GlobalIsSet(VMDebugFlag.Name) {
			vm.Debug = true
		}
		ethConf.PowTest = true
	}
	// Assemble and return the protocol stack
	stack, err := node.New(stackConf)
	if err != nil {
		Fatalf("Failed to create the protocol stack: %v", err)
	}
	if err := stack.Register(func(ctx *node.ServiceContext) (node.Service, error) {
		return eth.New(ctx, ethConf)
	}); err != nil {
		Fatalf("Failed to register the Ethereum service: %v", err)
	}
	if shhEnable {
		if err := stack.Register(func(*node.ServiceContext) (node.Service, error) { return whisper.New(), nil }); err != nil {
			Fatalf("Failed to register the Whisper service: %v", err)
		}
	}
	return stack
}

// SetupLogger configures glog from the logging-related command line flags.
func SetupLogger(ctx *cli.Context) {
	glog.SetV(ctx.GlobalInt(VerbosityFlag.Name))
	glog.CopyStandardLogTo("INFO")
	glog.SetToStderr(true)
	if ctx.GlobalIsSet(LogFileFlag.Name) {
		logger.New("", ctx.GlobalString(LogFileFlag.Name), ctx.GlobalInt(VerbosityFlag.Name))
	}
	if ctx.GlobalIsSet(VMDebugFlag.Name) {
		vm.Debug = ctx.GlobalBool(VMDebugFlag.Name)
	}
}

// SetupNetwork configures the system for either the main net or some test network.
func SetupNetwork(ctx *cli.Context) {
	switch {
	case ctx.GlobalBool(OlympicFlag.Name):
		params.DurationLimit = big.NewInt(8)
		params.GenesisGasLimit = big.NewInt(3141592)
		params.MinGasLimit = big.NewInt(125000)
		params.MaximumExtraDataSize = big.NewInt(1024)
		NetworkIdFlag.Value = 0
		core.BlockReward = big.NewInt(1.5e+18)
		core.ExpDiffPeriod = big.NewInt(math.MaxInt64)
	}
}

// SetupVM configured the VM package's global settings
func SetupVM(ctx *cli.Context) {
	vm.EnableJit = ctx.GlobalBool(VMEnableJitFlag.Name)
	vm.ForceJit = ctx.GlobalBool(VMForceJitFlag.Name)
	vm.SetJITCacheSize(ctx.GlobalInt(VMJitCacheFlag.Name))
}

// MakeChain creates a chain manager from set command line flags.
func MakeChain(ctx *cli.Context) (chain *core.BlockChain, chainDb ethdb.Database) {
	datadir := MustMakeDataDir(ctx)
	cache := ctx.GlobalInt(CacheFlag.Name)

	var err error
	if chainDb, err = ethdb.NewLDBDatabase(filepath.Join(datadir, "chaindata"), cache); err != nil {
		Fatalf("Could not open database: %v", err)
	}
	if ctx.GlobalBool(OlympicFlag.Name) {
		_, err := core.WriteTestNetGenesisBlock(chainDb)
		if err != nil {
			glog.Fatalln(err)
		}
	}

	eventMux := new(event.TypeMux)
	pow := ethash.New()
	//genesis := core.GenesisBlock(uint64(ctx.GlobalInt(GenesisNonceFlag.Name)), blockDB)
	chain, err = core.NewBlockChain(chainDb, pow, eventMux)
	if err != nil {
		Fatalf("Could not start chainmanager: %v", err)
	}

	return chain, chainDb
}

func IpcSocketPath(ctx *cli.Context) (ipcpath string) {
	if runtime.GOOS == "windows" {
		ipcpath = common.DefaultIpcPath()
		if ctx.GlobalIsSet(IPCPathFlag.Name) {
			ipcpath = ctx.GlobalString(IPCPathFlag.Name)
		}
	} else {
		ipcpath = common.DefaultIpcPath()
		if ctx.GlobalIsSet(DataDirFlag.Name) {
			ipcpath = filepath.Join(ctx.GlobalString(DataDirFlag.Name), "geth.ipc")
		}
		if ctx.GlobalIsSet(IPCPathFlag.Name) {
			ipcpath = ctx.GlobalString(IPCPathFlag.Name)
		}
	}

	return
}

// StartIPC starts a IPC JSON-RPC API server.
func StartIPC(stack *node.Node, ctx *cli.Context) error {
	config := comms.IpcConfig{
		Endpoint: IpcSocketPath(ctx),
	}

	initializer := func(conn net.Conn) (comms.Stopper, shared.EthereumApi, error) {
		var ethereum *eth.Ethereum
		if err := stack.Service(&ethereum); err != nil {
			return nil, nil, err
		}
		fe := useragent.NewRemoteFrontend(conn, ethereum.AccountManager())
		xeth := xeth.New(stack, fe)
		apis, err := api.ParseApiString(ctx.GlobalString(IPCApiFlag.Name), codec.JSON, xeth, stack)
		if err != nil {
			return nil, nil, err
		}
		return xeth, api.Merge(apis...), nil
	}
	return comms.StartIpc(config, codec.JSON, initializer)
}

// StartRPC starts a HTTP JSON-RPC API server.
func StartRPC(stack *node.Node, ctx *cli.Context) error {
	config := comms.HttpConfig{
		ListenAddress: ctx.GlobalString(RPCListenAddrFlag.Name),
		ListenPort:    uint(ctx.GlobalInt(RPCPortFlag.Name)),
		CorsDomain:    ctx.GlobalString(RPCCORSDomainFlag.Name),
	}

	xeth := xeth.New(stack, nil)
	codec := codec.JSON

	apis, err := api.ParseApiString(ctx.GlobalString(RpcApiFlag.Name), codec, xeth, stack)
	if err != nil {
		return err
	}
	return comms.StartHttp(config, codec, api.Merge(apis...))
}

func StartPProf(ctx *cli.Context) {
	address := fmt.Sprintf("localhost:%d", ctx.GlobalInt(PProfPortFlag.Name))
	go func() {
		log.Println(http.ListenAndServe(address, nil))
	}()
}
