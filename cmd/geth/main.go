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
	"io"
	"io/ioutil"
	_ "net/http/pprof"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"

	"github.com/codegangsta/cli"
	"github.com/ethereum/ethash"
	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/cmd/utils"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/eth"
	"github.com/ethereum/go-ethereum/logger"
	"github.com/mattn/go-colorable"
	"github.com/mattn/go-isatty"
)

const (
	ClientIdentifier = "Geth"
	Version          = "0.9.29"
)

var (
	gitCommit       string // set via linker flag
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
	app.Action = run
	app.HideVersion = true // we have a command to print the version
	app.Commands = []cli.Command{
		blocktestCommand,
		importCommand,
		exportCommand,
		upgradedbCommand,
		removedbCommand,
		dumpCommand,
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
			Action: version,
			Name:   "version",
			Usage:  "print ethereum version numbers",
			Description: `
The output of this command is supposed to be machine-readable.
`,
		},

		{
			Name:  "wallet",
			Usage: "ethereum presale wallet",
			Subcommands: []cli.Command{
				{
					Action: importWallet,
					Name:   "import",
					Usage:  "import ethereum presale wallet",
				},
			},
			Description: `

    get wallet import /path/to/my/presale.wallet

will prompt for your password and imports your ether presale account.
It can be used non-interactively with the --password option taking a
passwordfile as argument containing the wallet password in plaintext.

`},
		{
			Action: accountList,
			Name:   "account",
			Usage:  "manage accounts",
			Description: `

Manage accounts lets you create new accounts, list all existing accounts,
import a private key into a new account.

'            help' shows a list of subcommands or help for one subcommand.

It supports interactive mode, when you are prompted for password as well as
non-interactive mode where passwords are supplied via a given password file.
Non-interactive mode is only meant for scripted use on test networks or known
safe environments.

Make sure you remember the password you gave when creating a new account (with
either new or import). Without it you are not able to unlock your account.

Note that exporting your key in unencrypted format is NOT supported.

Keys are stored under <DATADIR>/keys.
It is safe to transfer the entire directory or the individual keys therein
between ethereum nodes.
Make sure you backup your keys regularly.

And finally. DO NOT FORGET YOUR PASSWORD.
`,
			Subcommands: []cli.Command{
				{
					Action: accountList,
					Name:   "list",
					Usage:  "print account addresses",
				},
				{
					Action: accountCreate,
					Name:   "new",
					Usage:  "create a new account",
					Description: `

    ethereum account new

Creates a new account. Prints the address.

The account is saved in encrypted format, you are prompted for a passphrase.

You must remember this passphrase to unlock your account in the future.

For non-interactive use the passphrase can be specified with the --password flag:

    ethereum --password <passwordfile> account new

Note, this is meant to be used for testing only, it is a bad idea to save your
password to file or expose in any other way.
					`,
				},
				{
					Action: accountImport,
					Name:   "import",
					Usage:  "import a private key into a new account",
					Description: `

    ethereum account import <keyfile>

Imports an unencrypted private key from <keyfile> and creates a new account.
Prints the address.

The keyfile is assumed to contain an unencrypted private key in hexadecimal format.

The account is saved in encrypted format, you are prompted for a passphrase.

You must remember this passphrase to unlock your account in the future.

For non-interactive use the passphrase can be specified with the -password flag:

    ethereum --password <passwordfile> account import <keyfile>

Note:
As you can directly copy your encrypted accounts to another ethereum instance,
this import mechanism is not needed when you transfer an account between
nodes.
					`,
				},
			},
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
			Action: execJSFiles,
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
		utils.GenesisNonceFlag,
		utils.BootnodesFlag,
		utils.DataDirFlag,
		utils.BlockchainVersionFlag,
		utils.JSpathFlag,
		utils.ListenPortFlag,
		utils.MaxPeersFlag,
		utils.MaxPendingPeersFlag,
		utils.EtherbaseFlag,
		utils.GasPriceFlag,
		utils.MinerThreadsFlag,
		utils.MiningEnabledFlag,
		utils.AutoDAGFlag,
		utils.NATFlag,
		utils.NatspecEnabledFlag,
		utils.NoDiscoverFlag,
		utils.NodeKeyFileFlag,
		utils.NodeKeyHexFlag,
		utils.RPCEnabledFlag,
		utils.RPCListenAddrFlag,
		utils.RPCPortFlag,
		utils.IPCDisabledFlag,
		utils.IPCApiFlag,
		utils.IPCPathFlag,
		utils.WhisperEnabledFlag,
		utils.VMDebugFlag,
		utils.ProtocolVersionFlag,
		utils.NetworkIdFlag,
		utils.RPCCORSDomainFlag,
		utils.VerbosityFlag,
		utils.BacktraceAtFlag,
		utils.LogToStdErrFlag,
		utils.LogVModuleFlag,
		utils.LogFileFlag,
		utils.LogJSONFlag,
		utils.PProfEanbledFlag,
		utils.PProfPortFlag,
		utils.SolcPathFlag,
	}
	app.Before = func(ctx *cli.Context) error {
		utils.SetupLogger(ctx)
		if ctx.GlobalBool(utils.PProfEanbledFlag.Name) {
			utils.StartPProf(ctx)
		}
		return nil
	}
}

func main() {
	runtime.GOMAXPROCS(runtime.NumCPU())
	defer logger.Flush()
	if err := app.Run(os.Args); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(ctx *cli.Context) {
	utils.HandleInterrupt()
	cfg := utils.MakeEthConfig(ClientIdentifier, nodeNameVersion, ctx)
	ethereum, err := eth.New(cfg)
	if err != nil {
		utils.Fatalf("%v", err)
	}

	startEth(ctx, ethereum)
	// this blocks the thread
	ethereum.WaitForShutdown()
}

func console(ctx *cli.Context) {
	// Wrap the standard output with a colorified stream (windows)
	if isatty.IsTerminal(os.Stdout.Fd()) {
		if pr, pw, err := os.Pipe(); err == nil {
			go io.Copy(colorable.NewColorableStdout(), pr)
			os.Stdout = pw
		}
	}

	cfg := utils.MakeEthConfig(ClientIdentifier, nodeNameVersion, ctx)
	ethereum, err := eth.New(cfg)
	if err != nil {
		utils.Fatalf("%v", err)
	}

	startEth(ctx, ethereum)
	repl := newJSRE(
		ethereum,
		ctx.String(utils.JSpathFlag.Name),
		ctx.GlobalString(utils.RPCCORSDomainFlag.Name),
		ctx.GlobalString(utils.IPCPathFlag.Name),
		true,
		nil,
	)
	repl.interactive()

	ethereum.Stop()
	ethereum.WaitForShutdown()
}

func execJSFiles(ctx *cli.Context) {
	cfg := utils.MakeEthConfig(ClientIdentifier, nodeNameVersion, ctx)
	ethereum, err := eth.New(cfg)
	if err != nil {
		utils.Fatalf("%v", err)
	}

	startEth(ctx, ethereum)
	repl := newJSRE(
		ethereum,
		ctx.String(utils.JSpathFlag.Name),
		ctx.GlobalString(utils.RPCCORSDomainFlag.Name),
		ctx.GlobalString(utils.IPCPathFlag.Name),
		false,
		nil,
	)
	for _, file := range ctx.Args() {
		repl.exec(file)
	}

	ethereum.Stop()
	ethereum.WaitForShutdown()
}

func unlockAccount(ctx *cli.Context, am *accounts.Manager, account string) (passphrase string) {
	var err error
	// Load startup keys. XXX we are going to need a different format

	if !((len(account) == 40) || (len(account) == 42)) { // with or without 0x
		utils.Fatalf("Invalid account address '%s'", account)
	}
	// Attempt to unlock the account 3 times
	attempts := 3
	for tries := 0; tries < attempts; tries++ {
		msg := fmt.Sprintf("Unlocking account %s | Attempt %d/%d", account, tries+1, attempts)
		passphrase = getPassPhrase(ctx, msg, false)
		err = am.Unlock(common.HexToAddress(account), passphrase)
		if err == nil {
			break
		}
	}
	if err != nil {
		utils.Fatalf("Unlock account failed '%v'", err)
	}
	fmt.Printf("Account '%s' unlocked.\n", account)
	return
}

func startEth(ctx *cli.Context, eth *eth.Ethereum) {
	// Start Ethereum itself

	utils.StartEthereum(eth)
	am := eth.AccountManager()

	account := ctx.GlobalString(utils.UnlockedAccountFlag.Name)
	accounts := strings.Split(account, " ")
	for _, account := range accounts {
		if len(account) > 0 {
			if account == "primary" {
				primaryAcc, err := am.Primary()
				if err != nil {
					utils.Fatalf("no primary account: %v", err)
				}
				account = primaryAcc.Hex()
			}
			unlockAccount(ctx, am, account)
		}
	}
	// Start auxiliary services if enabled.
	if !ctx.GlobalBool(utils.IPCDisabledFlag.Name) {
		if err := utils.StartIPC(eth, ctx); err != nil {
			utils.Fatalf("Error string IPC: %v", err)
		}
	}
	if ctx.GlobalBool(utils.RPCEnabledFlag.Name) {
		if err := utils.StartRPC(eth, ctx); err != nil {
			utils.Fatalf("Error starting RPC: %v", err)
		}
	}
	if ctx.GlobalBool(utils.MiningEnabledFlag.Name) {
		if err := eth.StartMining(ctx.GlobalInt(utils.MinerThreadsFlag.Name)); err != nil {
			utils.Fatalf("%v", err)
		}
	}
}

func accountList(ctx *cli.Context) {
	am := utils.MakeAccountManager(ctx)
	accts, err := am.Accounts()
	if err != nil {
		utils.Fatalf("Could not list accounts: %v", err)
	}
	name := "Primary"
	for i, acct := range accts {
		fmt.Printf("%s #%d: %x\n", name, i, acct)
		name = "Account"
	}
}

func getPassPhrase(ctx *cli.Context, desc string, confirmation bool) (passphrase string) {
	passfile := ctx.GlobalString(utils.PasswordFileFlag.Name)
	if len(passfile) == 0 {
		fmt.Println(desc)
		auth, err := utils.PromptPassword("Passphrase: ", true)
		if err != nil {
			utils.Fatalf("%v", err)
		}
		if confirmation {
			confirm, err := utils.PromptPassword("Repeat Passphrase: ", false)
			if err != nil {
				utils.Fatalf("%v", err)
			}
			if auth != confirm {
				utils.Fatalf("Passphrases did not match.")
			}
		}
		passphrase = auth

	} else {
		passbytes, err := ioutil.ReadFile(passfile)
		if err != nil {
			utils.Fatalf("Unable to read password file '%s': %v", passfile, err)
		}
		passphrase = string(passbytes)
	}
	return
}

func accountCreate(ctx *cli.Context) {
	am := utils.MakeAccountManager(ctx)
	passphrase := getPassPhrase(ctx, "Your new account is locked with a password. Please give a password. Do not forget this password.", true)
	acct, err := am.NewAccount(passphrase)
	if err != nil {
		utils.Fatalf("Could not create the account: %v", err)
	}
	fmt.Printf("Address: %x\n", acct)
}

func importWallet(ctx *cli.Context) {
	keyfile := ctx.Args().First()
	if len(keyfile) == 0 {
		utils.Fatalf("keyfile must be given as argument")
	}
	keyJson, err := ioutil.ReadFile(keyfile)
	if err != nil {
		utils.Fatalf("Could not read wallet file: %v", err)
	}

	am := utils.MakeAccountManager(ctx)
	passphrase := getPassPhrase(ctx, "", false)

	acct, err := am.ImportPreSaleKey(keyJson, passphrase)
	if err != nil {
		utils.Fatalf("Could not create the account: %v", err)
	}
	fmt.Printf("Address: %x\n", acct)
}

func accountImport(ctx *cli.Context) {
	keyfile := ctx.Args().First()
	if len(keyfile) == 0 {
		utils.Fatalf("keyfile must be given as argument")
	}
	am := utils.MakeAccountManager(ctx)
	passphrase := getPassPhrase(ctx, "Your new account is locked with a password. Please give a password. Do not forget this password.", true)
	acct, err := am.Import(keyfile, passphrase)
	if err != nil {
		utils.Fatalf("Could not create the account: %v", err)
	}
	fmt.Printf("Address: %x\n", acct)
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

func version(c *cli.Context) {
	fmt.Println(ClientIdentifier)
	fmt.Println("Version:", Version)
	if gitCommit != "" {
		fmt.Println("Git Commit:", gitCommit)
	}
	fmt.Println("Protocol Version:", c.GlobalInt(utils.ProtocolVersionFlag.Name))
	fmt.Println("Network Id:", c.GlobalInt(utils.NetworkIdFlag.Name))
	fmt.Println("Go Version:", runtime.Version())
	fmt.Println("OS:", runtime.GOOS)
	fmt.Printf("GOPATH=%s\n", os.Getenv("GOPATH"))
	fmt.Printf("GOROOT=%s\n", runtime.GOROOT())
}
