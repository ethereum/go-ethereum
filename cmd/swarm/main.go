// Copyright 2016 The go-ethereum Authors
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

package main

import (
	"crypto/ecdsa"
	"fmt"
	"io/ioutil"
	"os"
	"os/signal"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"syscall"

	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/ethereum/go-ethereum/cmd/utils"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/console"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/internal/debug"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/p2p/discover"
	"github.com/ethereum/go-ethereum/swarm"
	bzzapi "github.com/ethereum/go-ethereum/swarm/api"
	swarmmetrics "github.com/ethereum/go-ethereum/swarm/metrics"
	"github.com/ethereum/go-ethereum/swarm/tracing"
	sv "github.com/ethereum/go-ethereum/swarm/version"

	"gopkg.in/urfave/cli.v1"
)

const clientIdentifier = "swarm"
const helpTemplate = `NAME:
{{.HelpName}} - {{.Usage}}

USAGE:
{{if .UsageText}}{{.UsageText}}{{else}}{{.HelpName}}{{if .VisibleFlags}} [command options]{{end}} {{if .ArgsUsage}}{{.ArgsUsage}}{{else}}[arguments...]{{end}}{{end}}{{if .Category}}

CATEGORY:
{{.Category}}{{end}}{{if .Description}}

DESCRIPTION:
{{.Description}}{{end}}{{if .VisibleFlags}}

OPTIONS:
{{range .VisibleFlags}}{{.}}
{{end}}{{end}}
`

var (
	gitCommit        string // Git SHA1 commit hash of the release (set via linker flags)
	testbetBootNodes = []string{
		"enode://ec8ae764f7cb0417bdfb009b9d0f18ab3818a3a4e8e7c67dd5f18971a93510a2e6f43cd0b69a27e439a9629457ea804104f37c85e41eed057d3faabbf7744cdf@13.74.157.139:30429",
		"enode://c2e1fceb3bf3be19dff71eec6cccf19f2dbf7567ee017d130240c670be8594bc9163353ca55dd8df7a4f161dd94b36d0615c17418b5a3cdcbb4e9d99dfa4de37@13.74.157.139:30430",
		"enode://fe29b82319b734ce1ec68b84657d57145fee237387e63273989d354486731e59f78858e452ef800a020559da22dcca759536e6aa5517c53930d29ce0b1029286@13.74.157.139:30431",
		"enode://1d7187e7bde45cf0bee489ce9852dd6d1a0d9aa67a33a6b8e6db8a4fbc6fcfa6f0f1a5419343671521b863b187d1c73bad3603bae66421d157ffef357669ddb8@13.74.157.139:30432",
		"enode://0e4cba800f7b1ee73673afa6a4acead4018f0149d2e3216be3f133318fd165b324cd71b81fbe1e80deac8dbf56e57a49db7be67f8b9bc81bd2b7ee496434fb5d@13.74.157.139:30433",
	}
)

var (
	ChequebookAddrFlag = cli.StringFlag{
		Name:   "chequebook",
		Usage:  "chequebook contract address",
		EnvVar: SWARM_ENV_CHEQUEBOOK_ADDR,
	}
	SwarmAccountFlag = cli.StringFlag{
		Name:   "bzzaccount",
		Usage:  "Swarm account key file",
		EnvVar: SWARM_ENV_ACCOUNT,
	}
	SwarmListenAddrFlag = cli.StringFlag{
		Name:   "httpaddr",
		Usage:  "Swarm HTTP API listening interface",
		EnvVar: SWARM_ENV_LISTEN_ADDR,
	}
	SwarmPortFlag = cli.StringFlag{
		Name:   "bzzport",
		Usage:  "Swarm local http api port",
		EnvVar: SWARM_ENV_PORT,
	}
	SwarmNetworkIdFlag = cli.IntFlag{
		Name:   "bzznetworkid",
		Usage:  "Network identifier (integer, default 3=swarm testnet)",
		EnvVar: SWARM_ENV_NETWORK_ID,
	}
	SwarmSwapEnabledFlag = cli.BoolFlag{
		Name:   "swap",
		Usage:  "Swarm SWAP enabled (default false)",
		EnvVar: SWARM_ENV_SWAP_ENABLE,
	}
	SwarmSwapAPIFlag = cli.StringFlag{
		Name:   "swap-api",
		Usage:  "URL of the Ethereum API provider to use to settle SWAP payments",
		EnvVar: SWARM_ENV_SWAP_API,
	}
	SwarmSyncDisabledFlag = cli.BoolTFlag{
		Name:   "nosync",
		Usage:  "Disable swarm syncing",
		EnvVar: SWARM_ENV_SYNC_DISABLE,
	}
	SwarmSyncUpdateDelay = cli.DurationFlag{
		Name:   "sync-update-delay",
		Usage:  "Duration for sync subscriptions update after no new peers are added (default 15s)",
		EnvVar: SWARM_ENV_SYNC_UPDATE_DELAY,
	}
	SwarmLightNodeEnabled = cli.BoolFlag{
		Name:   "lightnode",
		Usage:  "Enable Swarm LightNode (default false)",
		EnvVar: SWARM_ENV_LIGHT_NODE_ENABLE,
	}
	SwarmDeliverySkipCheckFlag = cli.BoolFlag{
		Name:   "delivery-skip-check",
		Usage:  "Skip chunk delivery check (default false)",
		EnvVar: SWARM_ENV_DELIVERY_SKIP_CHECK,
	}
	EnsAPIFlag = cli.StringSliceFlag{
		Name:   "ens-api",
		Usage:  "ENS API endpoint for a TLD and with contract address, can be repeated, format [tld:][contract-addr@]url",
		EnvVar: SWARM_ENV_ENS_API,
	}
	SwarmApiFlag = cli.StringFlag{
		Name:  "bzzapi",
		Usage: "Swarm HTTP endpoint",
		Value: "http://127.0.0.1:8500",
	}
	SwarmRecursiveFlag = cli.BoolFlag{
		Name:  "recursive",
		Usage: "Upload directories recursively",
	}
	SwarmWantManifestFlag = cli.BoolTFlag{
		Name:  "manifest",
		Usage: "Automatic manifest upload (default true)",
	}
	SwarmUploadDefaultPath = cli.StringFlag{
		Name:  "defaultpath",
		Usage: "path to file served for empty url path (none)",
	}
	SwarmAccessGrantKeyFlag = cli.StringFlag{
		Name:  "grant-key",
		Usage: "grants a given public key access to an ACT",
	}
	SwarmAccessGrantKeysFlag = cli.StringFlag{
		Name:  "grant-keys",
		Usage: "grants a given list of public keys in the following file (separated by line breaks) access to an ACT",
	}
	SwarmUpFromStdinFlag = cli.BoolFlag{
		Name:  "stdin",
		Usage: "reads data to be uploaded from stdin",
	}
	SwarmUploadMimeType = cli.StringFlag{
		Name:  "mime",
		Usage: "Manually specify MIME type",
	}
	SwarmEncryptedFlag = cli.BoolFlag{
		Name:  "encrypt",
		Usage: "use encrypted upload",
	}
	SwarmAccessPasswordFlag = cli.StringFlag{
		Name:   "password",
		Usage:  "Password",
		EnvVar: SWARM_ACCESS_PASSWORD,
	}
	SwarmDryRunFlag = cli.BoolFlag{
		Name:  "dry-run",
		Usage: "dry-run",
	}
	CorsStringFlag = cli.StringFlag{
		Name:   "corsdomain",
		Usage:  "Domain on which to send Access-Control-Allow-Origin header (multiple domains can be supplied separated by a ',')",
		EnvVar: SWARM_ENV_CORS,
	}
	SwarmStorePath = cli.StringFlag{
		Name:   "store.path",
		Usage:  "Path to leveldb chunk DB (default <$GETH_ENV_DIR>/swarm/bzz-<$BZZ_KEY>/chunks)",
		EnvVar: SWARM_ENV_STORE_PATH,
	}
	SwarmStoreCapacity = cli.Uint64Flag{
		Name:   "store.size",
		Usage:  "Number of chunks (5M is roughly 20-25GB) (default 5000000)",
		EnvVar: SWARM_ENV_STORE_CAPACITY,
	}
	SwarmStoreCacheCapacity = cli.UintFlag{
		Name:   "store.cache.size",
		Usage:  "Number of recent chunks cached in memory (default 5000)",
		EnvVar: SWARM_ENV_STORE_CACHE_CAPACITY,
	}
	SwarmResourceMultihashFlag = cli.BoolFlag{
		Name:  "multihash",
		Usage: "Determines how to interpret data for a resource update. If not present, data will be interpreted as raw, literal data that will be included in the resource",
	}
	SwarmResourceNameFlag = cli.StringFlag{
		Name:  "name",
		Usage: "User-defined name for the new resource",
	}
	SwarmResourceDataOnCreateFlag = cli.StringFlag{
		Name:  "data",
		Usage: "Initializes the resource with the given hex-encoded data. Data must be prefixed by 0x",
	}
)

//declare a few constant error messages, useful for later error check comparisons in test
var (
	SWARM_ERR_NO_BZZACCOUNT   = "bzzaccount option is required but not set; check your config file, command line or environment variables"
	SWARM_ERR_SWAP_SET_NO_API = "SWAP is enabled but --swap-api is not set"
)

// this help command gets added to any subcommand that does not define it explicitly
var defaultSubcommandHelp = cli.Command{
	Action:             func(ctx *cli.Context) { cli.ShowCommandHelpAndExit(ctx, "", 1) },
	CustomHelpTemplate: helpTemplate,
	Name:               "help",
	Usage:              "shows this help",
	Hidden:             true,
}

var defaultNodeConfig = node.DefaultConfig

// This init function sets defaults so cmd/swarm can run alongside geth.
func init() {
	defaultNodeConfig.Name = clientIdentifier
	defaultNodeConfig.Version = sv.VersionWithCommit(gitCommit)
	defaultNodeConfig.P2P.ListenAddr = ":30399"
	defaultNodeConfig.IPCPath = "bzzd.ipc"
	// Set flag defaults for --help display.
	utils.ListenPortFlag.Value = 30399
}

var app = utils.NewApp(gitCommit, "Ethereum Swarm")

// This init function creates the cli.App.
func init() {
	app.Action = bzzd
	app.HideVersion = true // we have a command to print the version
	app.Copyright = "Copyright 2013-2016 The go-ethereum Authors"
	app.Commands = []cli.Command{
		{
			Action:             version,
			CustomHelpTemplate: helpTemplate,
			Name:               "version",
			Usage:              "Print version numbers",
			Description:        "The output of this command is supposed to be machine-readable",
		},
		{
			Action:             upload,
			CustomHelpTemplate: helpTemplate,
			Name:               "up",
			Usage:              "uploads a file or directory to swarm using the HTTP API",
			ArgsUsage:          "<file>",
			Flags:              []cli.Flag{SwarmEncryptedFlag},
			Description:        "uploads a file or directory to swarm using the HTTP API and prints the root hash",
		},
		{
			CustomHelpTemplate: helpTemplate,
			Name:               "access",
			Usage:              "encrypts a reference and embeds it into a root manifest",
			ArgsUsage:          "<ref>",
			Description:        "encrypts a reference and embeds it into a root manifest",
			Subcommands: []cli.Command{
				{
					CustomHelpTemplate: helpTemplate,
					Name:               "new",
					Usage:              "encrypts a reference and embeds it into a root manifest",
					ArgsUsage:          "<ref>",
					Description:        "encrypts a reference and embeds it into a root access manifest and prints the resulting manifest",
					Subcommands: []cli.Command{
						{
							Action:             accessNewPass,
							CustomHelpTemplate: helpTemplate,
							Flags: []cli.Flag{
								utils.PasswordFileFlag,
								SwarmDryRunFlag,
							},
							Name:        "pass",
							Usage:       "encrypts a reference with a password and embeds it into a root manifest",
							ArgsUsage:   "<ref>",
							Description: "encrypts a reference and embeds it into a root access manifest and prints the resulting manifest",
						},
						{
							Action:             accessNewPK,
							CustomHelpTemplate: helpTemplate,
							Flags: []cli.Flag{
								utils.PasswordFileFlag,
								SwarmDryRunFlag,
								SwarmAccessGrantKeyFlag,
							},
							Name:        "pk",
							Usage:       "encrypts a reference with the node's private key and a given grantee's public key and embeds it into a root manifest",
							ArgsUsage:   "<ref>",
							Description: "encrypts a reference and embeds it into a root access manifest and prints the resulting manifest",
						},
						{
							Action:             accessNewACT,
							CustomHelpTemplate: helpTemplate,
							Flags: []cli.Flag{
								SwarmAccessGrantKeysFlag,
								SwarmDryRunFlag,
							},
							Name:        "act",
							Usage:       "encrypts a reference with the node's private key and a given grantee's public key and embeds it into a root manifest",
							ArgsUsage:   "<ref>",
							Description: "encrypts a reference and embeds it into a root access manifest and prints the resulting manifest",
						},
					},
				},
			},
		},
		{
			CustomHelpTemplate: helpTemplate,
			Name:               "resource",
			Usage:              "(Advanced) Create and update Mutable Resources",
			ArgsUsage:          "<create|update|info>",
			Description:        "Works with Mutable Resource Updates",
			Subcommands: []cli.Command{
				{
					Action:             resourceCreate,
					CustomHelpTemplate: helpTemplate,
					Name:               "create",
					Usage:              "creates a new Mutable Resource",
					ArgsUsage:          "<frequency>",
					Description:        "creates a new Mutable Resource",
					Flags:              []cli.Flag{SwarmResourceNameFlag, SwarmResourceDataOnCreateFlag, SwarmResourceMultihashFlag},
				},
				{
					Action:             resourceUpdate,
					CustomHelpTemplate: helpTemplate,
					Name:               "update",
					Usage:              "updates the content of an existing Mutable Resource",
					ArgsUsage:          "<Manifest Address or ENS domain> <0x Hex data>",
					Description:        "updates the content of an existing Mutable Resource",
					Flags:              []cli.Flag{SwarmResourceMultihashFlag},
				},
				{
					Action:             resourceInfo,
					CustomHelpTemplate: helpTemplate,
					Name:               "info",
					Usage:              "obtains information about an existing Mutable Resource",
					ArgsUsage:          "<Manifest Address or ENS domain>",
					Description:        "obtains information about an existing Mutable Resource",
				},
			},
		},
		{
			Action:             list,
			CustomHelpTemplate: helpTemplate,
			Name:               "ls",
			Usage:              "list files and directories contained in a manifest",
			ArgsUsage:          "<manifest> [<prefix>]",
			Description:        "Lists files and directories contained in a manifest",
		},
		{
			Action:             hash,
			CustomHelpTemplate: helpTemplate,
			Name:               "hash",
			Usage:              "print the swarm hash of a file or directory",
			ArgsUsage:          "<file>",
			Description:        "Prints the swarm hash of file or directory",
		},
		{
			Action:      download,
			Name:        "down",
			Flags:       []cli.Flag{SwarmRecursiveFlag, SwarmAccessPasswordFlag},
			Usage:       "downloads a swarm manifest or a file inside a manifest",
			ArgsUsage:   " <uri> [<dir>]",
			Description: `Downloads a swarm bzz uri to the given dir. When no dir is provided, working directory is assumed. --recursive flag is expected when downloading a manifest with multiple entries.`,
		},
		{
			Name:               "manifest",
			CustomHelpTemplate: helpTemplate,
			Usage:              "perform operations on swarm manifests",
			ArgsUsage:          "COMMAND",
			Description:        "Updates a MANIFEST by adding/removing/updating the hash of a path.\nCOMMAND could be: add, update, remove",
			Subcommands: []cli.Command{
				{
					Action:             manifestAdd,
					CustomHelpTemplate: helpTemplate,
					Name:               "add",
					Usage:              "add a new path to the manifest",
					ArgsUsage:          "<MANIFEST> <path> <hash>",
					Description:        "Adds a new path to the manifest",
				},
				{
					Action:             manifestUpdate,
					CustomHelpTemplate: helpTemplate,
					Name:               "update",
					Usage:              "update the hash for an already existing path in the manifest",
					ArgsUsage:          "<MANIFEST> <path> <newhash>",
					Description:        "Update the hash for an already existing path in the manifest",
				},
				{
					Action:             manifestRemove,
					CustomHelpTemplate: helpTemplate,
					Name:               "remove",
					Usage:              "removes a path from the manifest",
					ArgsUsage:          "<MANIFEST> <path>",
					Description:        "Removes a path from the manifest",
				},
			},
		},
		{
			Name:               "fs",
			CustomHelpTemplate: helpTemplate,
			Usage:              "perform FUSE operations",
			ArgsUsage:          "fs COMMAND",
			Description:        "Performs FUSE operations by mounting/unmounting/listing mount points. This assumes you already have a Swarm node running locally. For all operation you must reference the correct path to bzzd.ipc in order to communicate with the node",
			Subcommands: []cli.Command{
				{
					Action:             mount,
					CustomHelpTemplate: helpTemplate,
					Name:               "mount",
					Flags:              []cli.Flag{utils.IPCPathFlag},
					Usage:              "mount a swarm hash to a mount point",
					ArgsUsage:          "swarm fs mount --ipcpath <path to bzzd.ipc> <manifest hash> <mount point>",
					Description:        "Mounts a Swarm manifest hash to a given mount point. This assumes you already have a Swarm node running locally. You must reference the correct path to your bzzd.ipc file",
				},
				{
					Action:             unmount,
					CustomHelpTemplate: helpTemplate,
					Name:               "unmount",
					Flags:              []cli.Flag{utils.IPCPathFlag},
					Usage:              "unmount a swarmfs mount",
					ArgsUsage:          "swarm fs unmount --ipcpath <path to bzzd.ipc> <mount point>",
					Description:        "Unmounts a swarmfs mount residing at <mount point>. This assumes you already have a Swarm node running locally. You must reference the correct path to your bzzd.ipc file",
				},
				{
					Action:             listMounts,
					CustomHelpTemplate: helpTemplate,
					Name:               "list",
					Flags:              []cli.Flag{utils.IPCPathFlag},
					Usage:              "list swarmfs mounts",
					ArgsUsage:          "swarm fs list --ipcpath <path to bzzd.ipc>",
					Description:        "Lists all mounted swarmfs volumes. This assumes you already have a Swarm node running locally. You must reference the correct path to your bzzd.ipc file",
				},
			},
		},
		{
			Name:               "db",
			CustomHelpTemplate: helpTemplate,
			Usage:              "manage the local chunk database",
			ArgsUsage:          "db COMMAND",
			Description:        "Manage the local chunk database",
			Subcommands: []cli.Command{
				{
					Action:             dbExport,
					CustomHelpTemplate: helpTemplate,
					Name:               "export",
					Usage:              "export a local chunk database as a tar archive (use - to send to stdout)",
					ArgsUsage:          "<chunkdb> <file>",
					Description: `
Export a local chunk database as a tar archive (use - to send to stdout).

    swarm db export ~/.ethereum/swarm/bzz-KEY/chunks chunks.tar

The export may be quite large, consider piping the output through the Unix
pv(1) tool to get a progress bar:

    swarm db export ~/.ethereum/swarm/bzz-KEY/chunks - | pv > chunks.tar
`,
				},
				{
					Action:             dbImport,
					CustomHelpTemplate: helpTemplate,
					Name:               "import",
					Usage:              "import chunks from a tar archive into a local chunk database (use - to read from stdin)",
					ArgsUsage:          "<chunkdb> <file>",
					Description: `Import chunks from a tar archive into a local chunk database (use - to read from stdin).

    swarm db import ~/.ethereum/swarm/bzz-KEY/chunks chunks.tar

The import may be quite large, consider piping the input through the Unix
pv(1) tool to get a progress bar:

    pv chunks.tar | swarm db import ~/.ethereum/swarm/bzz-KEY/chunks -`,
				},
				{
					Action:             dbClean,
					CustomHelpTemplate: helpTemplate,
					Name:               "clean",
					Usage:              "remove corrupt entries from a local chunk database",
					ArgsUsage:          "<chunkdb>",
					Description:        "Remove corrupt entries from a local chunk database",
				},
			},
		},

		// See config.go
		DumpConfigCommand,
	}

	// append a hidden help subcommand to all commands that have subcommands
	// if a help command was already defined above, that one will take precedence.
	addDefaultHelpSubcommands(app.Commands)

	sort.Sort(cli.CommandsByName(app.Commands))

	app.Flags = []cli.Flag{
		utils.IdentityFlag,
		utils.DataDirFlag,
		utils.BootnodesFlag,
		utils.KeyStoreDirFlag,
		utils.ListenPortFlag,
		utils.NoDiscoverFlag,
		utils.DiscoveryV5Flag,
		utils.NetrestrictFlag,
		utils.NodeKeyFileFlag,
		utils.NodeKeyHexFlag,
		utils.MaxPeersFlag,
		utils.NATFlag,
		utils.IPCDisabledFlag,
		utils.IPCPathFlag,
		utils.PasswordFileFlag,
		// bzzd-specific flags
		CorsStringFlag,
		EnsAPIFlag,
		SwarmTomlConfigPathFlag,
		SwarmSwapEnabledFlag,
		SwarmSwapAPIFlag,
		SwarmSyncDisabledFlag,
		SwarmSyncUpdateDelay,
		SwarmLightNodeEnabled,
		SwarmDeliverySkipCheckFlag,
		SwarmListenAddrFlag,
		SwarmPortFlag,
		SwarmAccountFlag,
		SwarmNetworkIdFlag,
		ChequebookAddrFlag,
		// upload flags
		SwarmApiFlag,
		SwarmRecursiveFlag,
		SwarmWantManifestFlag,
		SwarmUploadDefaultPath,
		SwarmUpFromStdinFlag,
		SwarmUploadMimeType,
		// storage flags
		SwarmStorePath,
		SwarmStoreCapacity,
		SwarmStoreCacheCapacity,
	}
	rpcFlags := []cli.Flag{
		utils.WSEnabledFlag,
		utils.WSListenAddrFlag,
		utils.WSPortFlag,
		utils.WSApiFlag,
		utils.WSAllowedOriginsFlag,
	}
	app.Flags = append(app.Flags, rpcFlags...)
	app.Flags = append(app.Flags, debug.Flags...)
	app.Flags = append(app.Flags, swarmmetrics.Flags...)
	app.Flags = append(app.Flags, tracing.Flags...)
	app.Before = func(ctx *cli.Context) error {
		runtime.GOMAXPROCS(runtime.NumCPU())
		if err := debug.Setup(ctx, ""); err != nil {
			return err
		}
		swarmmetrics.Setup(ctx)
		tracing.Setup(ctx)
		return nil
	}
	app.After = func(ctx *cli.Context) error {
		debug.Exit()
		return nil
	}
}

func main() {
	if err := app.Run(os.Args); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func version(ctx *cli.Context) error {
	fmt.Println(strings.Title(clientIdentifier))
	fmt.Println("Version:", sv.VersionWithMeta)
	if gitCommit != "" {
		fmt.Println("Git Commit:", gitCommit)
	}
	fmt.Println("Go Version:", runtime.Version())
	fmt.Println("OS:", runtime.GOOS)
	return nil
}

func bzzd(ctx *cli.Context) error {
	//build a valid bzzapi.Config from all available sources:
	//default config, file config, command line and env vars

	bzzconfig, err := buildConfig(ctx)
	if err != nil {
		utils.Fatalf("unable to configure swarm: %v", err)
	}

	cfg := defaultNodeConfig

	//pss operates on ws
	cfg.WSModules = append(cfg.WSModules, "pss")

	//geth only supports --datadir via command line
	//in order to be consistent within swarm, if we pass --datadir via environment variable
	//or via config file, we get the same directory for geth and swarm
	if _, err := os.Stat(bzzconfig.Path); err == nil {
		cfg.DataDir = bzzconfig.Path
	}
	//setup the ethereum node
	utils.SetNodeConfig(ctx, &cfg)
	stack, err := node.New(&cfg)
	if err != nil {
		utils.Fatalf("can't create node: %v", err)
	}

	//a few steps need to be done after the config phase is completed,
	//due to overriding behavior
	initSwarmNode(bzzconfig, stack, ctx)
	//register BZZ as node.Service in the ethereum node
	registerBzzService(bzzconfig, stack)
	//start the node
	utils.StartNode(stack)

	go func() {
		sigc := make(chan os.Signal, 1)
		signal.Notify(sigc, syscall.SIGTERM)
		defer signal.Stop(sigc)
		<-sigc
		log.Info("Got sigterm, shutting swarm down...")
		stack.Stop()
	}()

	// Add bootnodes as initial peers.
	if bzzconfig.BootNodes != "" {
		bootnodes := strings.Split(bzzconfig.BootNodes, ",")
		injectBootnodes(stack.Server(), bootnodes)
	} else {
		if bzzconfig.NetworkID == 3 {
			injectBootnodes(stack.Server(), testbetBootNodes)
		}
	}

	stack.Wait()
	return nil
}

func registerBzzService(bzzconfig *bzzapi.Config, stack *node.Node) {
	//define the swarm service boot function
	boot := func(_ *node.ServiceContext) (node.Service, error) {
		// In production, mockStore must be always nil.
		return swarm.NewSwarm(bzzconfig, nil)
	}
	//register within the ethereum node
	if err := stack.Register(boot); err != nil {
		utils.Fatalf("Failed to register the Swarm service: %v", err)
	}
}

func getAccount(bzzaccount string, ctx *cli.Context, stack *node.Node) *ecdsa.PrivateKey {
	//an account is mandatory
	if bzzaccount == "" {
		utils.Fatalf(SWARM_ERR_NO_BZZACCOUNT)
	}
	// Try to load the arg as a hex key file.
	if key, err := crypto.LoadECDSA(bzzaccount); err == nil {
		log.Info("Swarm account key loaded", "address", crypto.PubkeyToAddress(key.PublicKey))
		return key
	}
	// Otherwise try getting it from the keystore.
	am := stack.AccountManager()
	ks := am.Backends(keystore.KeyStoreType)[0].(*keystore.KeyStore)

	return decryptStoreAccount(ks, bzzaccount, utils.MakePasswordList(ctx))
}

// getPrivKey returns the private key of the specified bzzaccount
// Used only by client commands, such as `resource`
func getPrivKey(ctx *cli.Context) *ecdsa.PrivateKey {
	// booting up the swarm node just as we do in bzzd action
	bzzconfig, err := buildConfig(ctx)
	if err != nil {
		utils.Fatalf("unable to configure swarm: %v", err)
	}
	cfg := defaultNodeConfig
	if _, err := os.Stat(bzzconfig.Path); err == nil {
		cfg.DataDir = bzzconfig.Path
	}
	utils.SetNodeConfig(ctx, &cfg)
	stack, err := node.New(&cfg)
	if err != nil {
		utils.Fatalf("can't create node: %v", err)
	}
	return getAccount(bzzconfig.BzzAccount, ctx, stack)
}

func decryptStoreAccount(ks *keystore.KeyStore, account string, passwords []string) *ecdsa.PrivateKey {
	var a accounts.Account
	var err error
	if common.IsHexAddress(account) {
		a, err = ks.Find(accounts.Account{Address: common.HexToAddress(account)})
	} else if ix, ixerr := strconv.Atoi(account); ixerr == nil && ix > 0 {
		if accounts := ks.Accounts(); len(accounts) > ix {
			a = accounts[ix]
		} else {
			err = fmt.Errorf("index %d higher than number of accounts %d", ix, len(accounts))
		}
	} else {
		utils.Fatalf("Can't find swarm account key %s", account)
	}
	if err != nil {
		utils.Fatalf("Can't find swarm account key: %v - Is the provided bzzaccount(%s) from the right datadir/Path?", err, account)
	}
	keyjson, err := ioutil.ReadFile(a.URL.Path)
	if err != nil {
		utils.Fatalf("Can't load swarm account key: %v", err)
	}
	for i := 0; i < 3; i++ {
		password := getPassPhrase(fmt.Sprintf("Unlocking swarm account %s [%d/3]", a.Address.Hex(), i+1), i, passwords)
		key, err := keystore.DecryptKey(keyjson, password)
		if err == nil {
			return key.PrivateKey
		}
	}
	utils.Fatalf("Can't decrypt swarm account key")
	return nil
}

// getPassPhrase retrieves the password associated with bzz account, either by fetching
// from a list of pre-loaded passwords, or by requesting it interactively from user.
func getPassPhrase(prompt string, i int, passwords []string) string {
	// non-interactive
	if len(passwords) > 0 {
		if i < len(passwords) {
			return passwords[i]
		}
		return passwords[len(passwords)-1]
	}

	// fallback to interactive mode
	if prompt != "" {
		fmt.Println(prompt)
	}
	password, err := console.Stdin.PromptPassword("Passphrase: ")
	if err != nil {
		utils.Fatalf("Failed to read passphrase: %v", err)
	}
	return password
}

func injectBootnodes(srv *p2p.Server, nodes []string) {
	for _, url := range nodes {
		n, err := discover.ParseNode(url)
		if err != nil {
			log.Error("Invalid swarm bootnode", "err", err)
			continue
		}
		srv.AddPeer(n)
	}
}

// addDefaultHelpSubcommand scans through defined CLI commands and adds
// a basic help subcommand to each
// if a help command is already defined, it will take precedence over the default.
func addDefaultHelpSubcommands(commands []cli.Command) {
	for i := range commands {
		cmd := &commands[i]
		if cmd.Subcommands != nil {
			cmd.Subcommands = append(cmd.Subcommands, defaultSubcommandHelp)
			addDefaultHelpSubcommands(cmd.Subcommands)
		}
	}
}
