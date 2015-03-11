package utils

import (
	"crypto/ecdsa"
	"fmt"
	"net"
	"net/http"
	"os"
	"path"
	"runtime"

	"github.com/codegangsta/cli"
	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/eth"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/ethutil"
	"github.com/ethereum/go-ethereum/event"
	"github.com/ethereum/go-ethereum/logger"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/p2p/nat"
	"github.com/ethereum/go-ethereum/rpc"
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
	app.Name = path.Base(os.Args[0])
	app.Author = ""
	app.Authors = nil
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
	/*
		VMTypeFlag = cli.IntFlag{
			Name:  "vm",
			Usage: "Virtual Machine type: 0 is standard VM, 1 is debug VM",
		}
	*/
	UnlockedAccountFlag = cli.StringFlag{
		Name:  "unlock",
		Usage: "Unlock a given account untill this programs exits (address:password)",
	}
	VMDebugFlag = cli.BoolFlag{
		Name:  "vmdebug",
		Usage: "Virtual Machine debug output",
	}
	DataDirFlag = cli.StringFlag{
		Name:  "datadir",
		Usage: "Data directory to be used",
		Value: ethutil.DefaultDataDir(),
	}
	MinerThreadsFlag = cli.IntFlag{
		Name:  "minerthreads",
		Usage: "Number of miner threads",
		Value: runtime.NumCPU(),
	}
	MiningEnabledFlag = cli.BoolFlag{
		Name:  "mine",
		Usage: "Enable mining",
	}
	UnencryptedKeysFlag = cli.BoolFlag{
		Name:  "unencrypted-keys",
		Usage: "disable private key disk encryption (for testing)",
	}

	LogFileFlag = cli.StringFlag{
		Name:  "logfile",
		Usage: "Send log output to a file",
	}
	LogLevelFlag = cli.IntFlag{
		Name:  "loglevel",
		Usage: "0-5 (silent, error, warn, info, debug, debug detail)",
		Value: int(logger.InfoLevel),
	}
	LogFormatFlag = cli.StringFlag{
		Name:  "logformat",
		Usage: `"std" or "raw"`,
		Value: "std",
	}

	// RPC settings
	RPCEnabledFlag = cli.BoolFlag{
		Name:  "rpc",
		Usage: "Whether RPC server is enabled",
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

	// Network Settings
	MaxPeersFlag = cli.IntFlag{
		Name:  "maxpeers",
		Usage: "Maximum number of network peers",
		Value: 16,
	}
	ListenPortFlag = cli.IntFlag{
		Name:  "port",
		Usage: "Network listening port",
		Value: 30303,
	}
	BootnodesFlag = cli.StringFlag{
		Name:  "bootnodes",
		Usage: "Space-separated enode URLs for discovery bootstrap",
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
		Usage: "Port mapping mechanism (any|none|upnp|pmp|extip:<IP>)",
		Value: "any",
	}
)

func GetNAT(ctx *cli.Context) nat.Interface {
	natif, err := nat.Parse(ctx.GlobalString(NATFlag.Name))
	if err != nil {
		Fatalf("Option %s: %v", NATFlag.Name, err)
	}
	return natif
}

func GetNodeKey(ctx *cli.Context) (key *ecdsa.PrivateKey) {
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

func GetEthereum(clientID, version string, ctx *cli.Context) (*eth.Ethereum, error) {
	return eth.New(&eth.Config{
		Name:           p2p.MakeName(clientID, version),
		DataDir:        ctx.GlobalString(DataDirFlag.Name),
		LogFile:        ctx.GlobalString(LogFileFlag.Name),
		LogLevel:       ctx.GlobalInt(LogLevelFlag.Name),
		LogFormat:      ctx.GlobalString(LogFormatFlag.Name),
		MinerThreads:   ctx.GlobalInt(MinerThreadsFlag.Name),
		AccountManager: GetAccountManager(ctx),
		VmDebug:        ctx.GlobalBool(VMDebugFlag.Name),
		MaxPeers:       ctx.GlobalInt(MaxPeersFlag.Name),
		Port:           ctx.GlobalString(ListenPortFlag.Name),
		NAT:            GetNAT(ctx),
		NodeKey:        GetNodeKey(ctx),
		Shh:            true,
		Dial:           true,
		BootNodes:      ctx.GlobalString(BootnodesFlag.Name),
	})
}

func GetChain(ctx *cli.Context) (*core.ChainManager, ethutil.Database, ethutil.Database) {
	dataDir := ctx.GlobalString(DataDirFlag.Name)
	blockDb, err := ethdb.NewLDBDatabase(path.Join(dataDir, "blockchain"))
	if err != nil {
		Fatalf("Could not open database: %v", err)
	}

	stateDb, err := ethdb.NewLDBDatabase(path.Join(dataDir, "state"))
	if err != nil {
		Fatalf("Could not open database: %v", err)
	}
	return core.NewChainManager(blockDb, stateDb, new(event.TypeMux)), blockDb, stateDb
}

func GetAccountManager(ctx *cli.Context) *accounts.Manager {
	dataDir := ctx.GlobalString(DataDirFlag.Name)
	var ks crypto.KeyStore2
	if ctx.GlobalBool(UnencryptedKeysFlag.Name) {
		ks = crypto.NewKeyStorePlain(path.Join(dataDir, "plainkeys"))
	} else {
		ks = crypto.NewKeyStorePassphrase(path.Join(dataDir, "keys"))
	}
	return accounts.NewManager(ks)
}

func StartRPC(eth *eth.Ethereum, ctx *cli.Context) {
	addr := ctx.GlobalString(RPCListenAddrFlag.Name)
	port := ctx.GlobalInt(RPCPortFlag.Name)
	dataDir := ctx.GlobalString(DataDirFlag.Name)

	l, err := net.Listen("tcp", fmt.Sprintf("%s:%d", addr, port))
	if err != nil {
		Fatalf("Can't listen on %s:%d: %v", addr, port, err)
	}
	go http.Serve(l, rpc.JSONRPC(xeth.New(eth, nil), dataDir))
}
