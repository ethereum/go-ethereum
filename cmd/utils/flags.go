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
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with go-ethereum.  If not, see <http://www.gnu.org/licenses/>.

package utils

import (
	"crypto/ecdsa"
	"fmt"
	"log"
	"math/big"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strconv"

	"github.com/ethereum/go-ethereum/metrics"

	"github.com/codegangsta/cli"
	"github.com/ethereum/ethash"
	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/eth"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/event"
	"github.com/ethereum/go-ethereum/logger"
	"github.com/ethereum/go-ethereum/logger/glog"
	"github.com/ethereum/go-ethereum/p2p/nat"
	"github.com/ethereum/go-ethereum/rpc/api"
	"github.com/ethereum/go-ethereum/rpc/codec"
	"github.com/ethereum/go-ethereum/rpc/comms"
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
		Usage: "Data directory to be used",
		Value: DirectoryString{common.DefaultDataDir()},
	}
	NetworkIdFlag = cli.IntFlag{
		Name:  "networkid",
		Usage: "Network Id (integer)",
		Value: eth.NetworkId,
	}
	BlockchainVersionFlag = cli.IntFlag{
		Name:  "blockchainversion",
		Usage: "Blockchain version (integer)",
		Value: core.BlockChainVersion,
	}
	GenesisNonceFlag = cli.IntFlag{
		Name:  "genesisnonce",
		Usage: "Sets the genesis nonce",
		Value: 42,
	}
	IdentityFlag = cli.StringFlag{
		Name:  "identity",
		Usage: "Custom node name",
	}
	NatspecEnabledFlag = cli.BoolFlag{
		Name:  "natspec",
		Usage: "Enable NatSpec confirmation notice",
	}

	// miner settings
	MinerThreadsFlag = cli.IntFlag{
		Name:  "minerthreads",
		Usage: "Number of miner threads",
		Value: runtime.NumCPU(),
	}
	MiningEnabledFlag = cli.BoolFlag{
		Name:  "mine",
		Usage: "Enable mining",
	}
	AutoDAGFlag = cli.BoolFlag{
		Name:  "autodag",
		Usage: "Enable automatic DAG pregeneration",
	}
	EtherbaseFlag = cli.StringFlag{
		Name:  "etherbase",
		Usage: "Public address for block mining rewards. By default the address first created is used",
		Value: "0",
	}
	GasPriceFlag = cli.StringFlag{
		Name:  "gasprice",
		Usage: "Sets the minimal gasprice when mining transactions",
		Value: new(big.Int).Mul(big.NewInt(1), common.Szabo).String(),
	}

	UnlockedAccountFlag = cli.StringFlag{
		Name:  "unlock",
		Usage: "Unlock the account given until this program exits (prompts for password). '--unlock n' unlocks the n-th account in order or creation.",
		Value: "",
	}
	PasswordFileFlag = cli.StringFlag{
		Name:  "password",
		Usage: "Path to password file to use with options and subcommands needing a password",
		Value: "",
	}

	// logging and debug settings
	LogFileFlag = cli.StringFlag{
		Name:  "logfile",
		Usage: "Send log output to a file",
	}
	VerbosityFlag = cli.IntFlag{
		Name:  "verbosity",
		Usage: "Logging verbosity: 0-6 (0=silent, 1=error, 2=warn, 3=info, 4=core, 5=debug, 6=debug detail)",
		Value: int(logger.InfoLevel),
	}
	LogJSONFlag = cli.StringFlag{
		Name:  "logjson",
		Usage: "Send json structured log output to a file or '-' for standard output (default: no json output)",
		Value: "",
	}
	LogToStdErrFlag = cli.BoolFlag{
		Name:  "logtostderr",
		Usage: "Logs are written to standard error instead of to files.",
	}
	LogVModuleFlag = cli.GenericFlag{
		Name:  "vmodule",
		Usage: "The syntax of the argument is a comma-separated list of pattern=N, where pattern is a literal file name (minus the \".go\" suffix) or \"glob\" pattern and N is a log verbosity level.",
		Value: glog.GetVModule(),
	}
	VMDebugFlag = cli.BoolFlag{
		Name:  "vmdebug",
		Usage: "Virtual Machine debug output",
	}
	BacktraceAtFlag = cli.GenericFlag{
		Name:  "backtrace_at",
		Usage: "If set to a file and line number (e.g., \"block.go:271\") holding a logging statement, a stack trace will be logged",
		Value: glog.GetTraceLocation(),
	}
	PProfEanbledFlag = cli.BoolFlag{
		Name:  "pprof",
		Usage: "Enable the profiling server on localhost",
	}
	PProfPortFlag = cli.IntFlag{
		Name:  "pprofport",
		Usage: "Port on which the profiler should listen",
		Value: 6060,
	}
	MetricsEnabledFlag = cli.BoolFlag{
		Name:  metrics.MetricsEnabledFlag,
		Usage: "Enables metrics collection and reporting",
	}

	// RPC settings
	RPCEnabledFlag = cli.BoolFlag{
		Name:  "rpc",
		Usage: "Enable the JSON-RPC server",
	}
	RPCListenAddrFlag = cli.StringFlag{
		Name:  "rpcaddr",
		Usage: "Listening address for the JSON-RPC server",
		Value: "127.0.0.1",
	}
	RPCPortFlag = cli.IntFlag{
		Name:  "rpcport",
		Usage: "Port on which the JSON-RPC server should listen",
		Value: 8545,
	}
	RPCCORSDomainFlag = cli.StringFlag{
		Name:  "rpccorsdomain",
		Usage: "Domain on which to send Access-Control-Allow-Origin header",
		Value: "",
	}
	RpcApiFlag = cli.StringFlag{
		Name:  "rpcapi",
		Usage: "Specify the API's which are offered over the HTTP RPC interface",
		Value: comms.DefaultHttpRpcApis,
	}
	IPCDisabledFlag = cli.BoolFlag{
		Name:  "ipcdisable",
		Usage: "Disable the IPC-RPC server",
	}
	IPCApiFlag = cli.StringFlag{
		Name:  "ipcapi",
		Usage: "Specify the API's which are offered over the IPC interface",
		Value: comms.DefaultIpcApis,
	}
	IPCPathFlag = DirectoryFlag{
		Name:  "ipcpath",
		Usage: "Filename for IPC socket/pipe",
		Value: DirectoryString{common.DefaultIpcPath()},
	}
	ExecFlag = cli.StringFlag{
		Name:  "exec",
		Usage: "Execute javascript statement (only in combination with console/attach)",
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
		Usage: "Space-separated enode URLs for p2p discovery bootstrap",
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
		Usage: "Enable whisper",
	}
	// ATM the url is left to the user and deployment to
	JSpathFlag = cli.StringFlag{
		Name:  "jspath",
		Usage: "JS library path to be used with console and js subcommands",
		Value: ".",
	}
	SolcPathFlag = cli.StringFlag{
		Name:  "solc",
		Usage: "solidity compiler to be used",
		Value: "solc",
	}
	GpoMinGasPriceFlag = cli.StringFlag{
		Name:  "gpomin",
		Usage: "Minimum suggested gas price",
		Value: new(big.Int).Mul(big.NewInt(1), common.Szabo).String(),
	}
	GpoMaxGasPriceFlag = cli.StringFlag{
		Name:  "gpomax",
		Usage: "Maximum suggested gas price",
		Value: new(big.Int).Mul(big.NewInt(100), common.Szabo).String(),
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

// MakeNAT creates a port mapper from set command line flags.
func MakeNAT(ctx *cli.Context) nat.Interface {
	natif, err := nat.Parse(ctx.GlobalString(NATFlag.Name))
	if err != nil {
		Fatalf("Option %s: %v", NATFlag.Name, err)
	}
	return natif
}

// MakeNodeKey creates a node key from set command line flags.
func MakeNodeKey(ctx *cli.Context) (key *ecdsa.PrivateKey) {
	hex, file := ctx.GlobalString(NodeKeyHexFlag.Name), ctx.GlobalString(NodeKeyFileFlag.Name)
	var err error
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

// MakeEthConfig creates ethereum options from set command line flags.
func MakeEthConfig(clientID, version string, ctx *cli.Context) *eth.Config {
	customName := ctx.GlobalString(IdentityFlag.Name)
	if len(customName) > 0 {
		clientID += "/" + customName
	}
	am := MakeAccountManager(ctx)
	etherbase, err := ParamToAddress(ctx.GlobalString(EtherbaseFlag.Name), am)
	if err != nil {
		glog.V(logger.Error).Infoln("WARNING: No etherbase set and no accounts found as default")
	}

	return &eth.Config{
		Name:                    common.MakeName(clientID, version),
		DataDir:                 ctx.GlobalString(DataDirFlag.Name),
		GenesisNonce:            ctx.GlobalInt(GenesisNonceFlag.Name),
		BlockChainVersion:       ctx.GlobalInt(BlockchainVersionFlag.Name),
		SkipBcVersionCheck:      false,
		NetworkId:               ctx.GlobalInt(NetworkIdFlag.Name),
		LogFile:                 ctx.GlobalString(LogFileFlag.Name),
		Verbosity:               ctx.GlobalInt(VerbosityFlag.Name),
		LogJSON:                 ctx.GlobalString(LogJSONFlag.Name),
		Etherbase:               common.HexToAddress(etherbase),
		MinerThreads:            ctx.GlobalInt(MinerThreadsFlag.Name),
		AccountManager:          am,
		VmDebug:                 ctx.GlobalBool(VMDebugFlag.Name),
		MaxPeers:                ctx.GlobalInt(MaxPeersFlag.Name),
		MaxPendingPeers:         ctx.GlobalInt(MaxPendingPeersFlag.Name),
		Port:                    ctx.GlobalString(ListenPortFlag.Name),
		NAT:                     MakeNAT(ctx),
		NatSpec:                 ctx.GlobalBool(NatspecEnabledFlag.Name),
		Discovery:               !ctx.GlobalBool(NoDiscoverFlag.Name),
		NodeKey:                 MakeNodeKey(ctx),
		Shh:                     ctx.GlobalBool(WhisperEnabledFlag.Name),
		Dial:                    true,
		BootNodes:               ctx.GlobalString(BootnodesFlag.Name),
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
}

// SetupLogger configures glog from the logging-related command line flags.
func SetupLogger(ctx *cli.Context) {
	glog.SetV(ctx.GlobalInt(VerbosityFlag.Name))
	glog.CopyStandardLogTo("INFO")
	glog.SetToStderr(true)
	glog.SetLogDir(ctx.GlobalString(LogFileFlag.Name))
}

// MakeChain creates a chain manager from set command line flags.
func MakeChain(ctx *cli.Context) (chain *core.ChainManager, blockDB, stateDB, extraDB common.Database) {
	dd := ctx.GlobalString(DataDirFlag.Name)
	var err error
	if blockDB, err = ethdb.NewLDBDatabase(filepath.Join(dd, "blockchain")); err != nil {
		Fatalf("Could not open database: %v", err)
	}
	if stateDB, err = ethdb.NewLDBDatabase(filepath.Join(dd, "state")); err != nil {
		Fatalf("Could not open database: %v", err)
	}
	if extraDB, err = ethdb.NewLDBDatabase(filepath.Join(dd, "extra")); err != nil {
		Fatalf("Could not open database: %v", err)
	}

	eventMux := new(event.TypeMux)
	pow := ethash.New()
	genesis := core.GenesisBlock(uint64(ctx.GlobalInt(GenesisNonceFlag.Name)), blockDB)
	chain, err = core.NewChainManager(genesis, blockDB, stateDB, extraDB, pow, eventMux)
	if err != nil {
		Fatalf("Could not start chainmanager: %v", err)
	}

	proc := core.NewBlockProcessor(stateDB, extraDB, pow, chain, eventMux)
	chain.SetProcessor(proc)
	return chain, blockDB, stateDB, extraDB
}

// MakeChain creates an account manager from set command line flags.
func MakeAccountManager(ctx *cli.Context) *accounts.Manager {
	dataDir := ctx.GlobalString(DataDirFlag.Name)
	ks := crypto.NewKeyStorePassphrase(filepath.Join(dataDir, "keystore"))
	return accounts.NewManager(ks)
}

func IpcSocketPath(ctx *cli.Context) (ipcpath string) {
	if common.IsWindows() {
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

func StartIPC(eth *eth.Ethereum, ctx *cli.Context) error {
	config := comms.IpcConfig{
		Endpoint: IpcSocketPath(ctx),
	}

	xeth := xeth.New(eth, nil)
	codec := codec.JSON

	apis, err := api.ParseApiString(ctx.GlobalString(IPCApiFlag.Name), codec, xeth, eth)
	if err != nil {
		return err
	}

	return comms.StartIpc(config, codec, api.Merge(apis...))
}

func StartRPC(eth *eth.Ethereum, ctx *cli.Context) error {
	config := comms.HttpConfig{
		ListenAddress: ctx.GlobalString(RPCListenAddrFlag.Name),
		ListenPort:    uint(ctx.GlobalInt(RPCPortFlag.Name)),
		CorsDomain:    ctx.GlobalString(RPCCORSDomainFlag.Name),
	}

	xeth := xeth.New(eth, nil)
	codec := codec.JSON

	apis, err := api.ParseApiString(ctx.GlobalString(RpcApiFlag.Name), codec, xeth, eth)
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

func ParamToAddress(addr string, am *accounts.Manager) (addrHex string, err error) {
	if !((len(addr) == 40) || (len(addr) == 42)) { // with or without 0x
		index, err := strconv.Atoi(addr)
		if err != nil {
			Fatalf("Invalid account address '%s'", addr)
		}

		addrHex, err = am.AddressByIndex(index)
		if err != nil {
			return "", err
		}
	} else {
		addrHex = addr
	}
	return
}
