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
	"bufio"
	"fmt"
	"os"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/codegangsta/cli"
	"github.com/ethereum/ethash"
	"github.com/ethereum/go-ethereum/cmd/utils"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/eth"
	"github.com/ethereum/go-ethereum/logger"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/peterh/liner"
)

const (
	ClientIdentifier = "Ethereum(G)"
	Version          = "0.9.2"
)

var (
	clilogger = logger.NewLogger("CLI")
	app       = utils.NewApp(Version, "the go-ethereum command line interface")
)

func init() {
	app.Action = run
	app.HideVersion = true // we have a command to print the version
	app.Commands = []cli.Command{
		blocktestCmd,
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
			Action: accountList,
			Name:   "account",
			Usage:  "manage accounts",
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
				},
			},
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
			Action: console,
			Name:   "console",
			Usage:  `Ethereum Console: interactive JavaScript environment`,
			Description: `
Console is an interactive shell for the Ethereum JavaScript runtime environment which exposes a node admin interface as well as the DAPP JavaScript API.
See https://github.com/ethereum/go-ethereum/wiki/Frontier-Console
`,
		},
		{
			Action: execJSFiles,
			Name:   "js",
			Usage:  `executes the given JavaScript files in the Ethereum Frontier JavaScript VM`,
			Description: `
The Ethereum JavaScript VM exposes a node admin interface as well as the DAPP JavaScript API. See https://github.com/ethereum/go-ethereum/wiki/Frontier-Console
`,
		},
		{
			Action: importchain,
			Name:   "import",
			Usage:  `import a blockchain file`,
		},
		{
			Action: exportchain,
			Name:   "export",
			Usage:  `export blockchain into file`,
		},
	}
	app.Flags = []cli.Flag{
		utils.UnlockedAccountFlag,
		utils.BootnodesFlag,
		utils.DataDirFlag,
		utils.JSpathFlag,
		utils.ListenPortFlag,
		utils.LogFileFlag,
		utils.LogJSONFlag,
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
		utils.UnencryptedKeysFlag,
		utils.VMDebugFlag,
		utils.ProtocolVersionFlag,
		utils.NetworkIdFlag,
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
	if err := app.Run(os.Args); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(ctx *cli.Context) {
	fmt.Printf("Welcome to the FRONTIER\n")
	utils.HandleInterrupt()
	cfg := utils.MakeEthConfig(ClientIdentifier, Version, ctx)
	ethereum, err := eth.New(cfg)
	if err != nil {
		utils.Fatalf("%v", err)
	}

	startEth(ctx, ethereum)
	// this blocks the thread
	ethereum.WaitForShutdown()
}

func console(ctx *cli.Context) {
	cfg := utils.MakeEthConfig(ClientIdentifier, Version, ctx)
	ethereum, err := eth.New(cfg)
	if err != nil {
		utils.Fatalf("%v", err)
	}

	startEth(ctx, ethereum)
	repl := newJSRE(ethereum, ctx.String(utils.JSpathFlag.Name))
	repl.interactive()

	ethereum.Stop()
	ethereum.WaitForShutdown()
}

func execJSFiles(ctx *cli.Context) {
	cfg := utils.MakeEthConfig(ClientIdentifier, Version, ctx)
	ethereum, err := eth.New(cfg)
	if err != nil {
		utils.Fatalf("%v", err)
	}

	startEth(ctx, ethereum)
	repl := newJSRE(ethereum, ctx.String(utils.JSpathFlag.Name))
	for _, file := range ctx.Args() {
		repl.exec(file)
	}

	ethereum.Stop()
	ethereum.WaitForShutdown()
}

func startEth(ctx *cli.Context, eth *eth.Ethereum) {
	utils.StartEthereum(eth)

	// Load startup keys. XXX we are going to need a different format
	account := ctx.GlobalString(utils.UnlockedAccountFlag.Name)
	if len(account) > 0 {
		split := strings.Split(account, ":")
		if len(split) != 2 {
			utils.Fatalf("Illegal 'unlock' format (address:password)")
		}
		am := eth.AccountManager()
		// Attempt to unlock the account
		err := am.Unlock(common.FromHex(split[0]), split[1])
		if err != nil {
			utils.Fatalf("Unlock account failed '%v'", err)
		}
	}
	// Start auxiliary services if enabled.
	if ctx.GlobalBool(utils.RPCEnabledFlag.Name) {
		utils.StartRPC(eth, ctx)
	}
	if ctx.GlobalBool(utils.MiningEnabledFlag.Name) {
		eth.StartMining()
	}
}

func accountList(ctx *cli.Context) {
	am := utils.GetAccountManager(ctx)
	accts, err := am.Accounts()
	if err != nil {
		utils.Fatalf("Could not list accounts: %v", err)
	}
	for _, acct := range accts {
		fmt.Printf("Address: %x\n", acct)
	}
}

func accountCreate(ctx *cli.Context) {
	am := utils.GetAccountManager(ctx)
	passphrase := ""
	if !ctx.GlobalBool(utils.UnencryptedKeysFlag.Name) {
		fmt.Println("The new account will be encrypted with a passphrase.")
		fmt.Println("Please enter a passphrase now.")
		auth, err := readPassword("Passphrase: ", true)
		if err != nil {
			utils.Fatalf("%v", err)
		}
		confirm, err := readPassword("Repeat Passphrase: ", false)
		if err != nil {
			utils.Fatalf("%v", err)
		}
		if auth != confirm {
			utils.Fatalf("Passphrases did not match.")
		}
		passphrase = auth
	}
	acct, err := am.NewAccount(passphrase)
	if err != nil {
		utils.Fatalf("Could not create the account: %v", err)
	}
	fmt.Printf("Address: %x\n", acct.Address)
}

func importchain(ctx *cli.Context) {
	if len(ctx.Args()) != 1 {
		utils.Fatalf("This command requires an argument.")
	}
	chainmgr, _, _ := utils.GetChain(ctx)
	start := time.Now()
	err := utils.ImportChain(chainmgr, ctx.Args().First())
	if err != nil {
		utils.Fatalf("Import error: %v\n", err)
	}
	fmt.Printf("Import done in %v", time.Since(start))
	return
}

func exportchain(ctx *cli.Context) {
	if len(ctx.Args()) != 1 {
		utils.Fatalf("This command requires an argument.")
	}
	chainmgr, _, _ := utils.GetChain(ctx)
	start := time.Now()
	err := utils.ExportChain(chainmgr, ctx.Args().First())
	if err != nil {
		utils.Fatalf("Export error: %v\n", err)
	}
	fmt.Printf("Export done in %v", time.Since(start))
	return
}

func dump(ctx *cli.Context) {
	chainmgr, _, stateDb := utils.GetChain(ctx)
	for _, arg := range ctx.Args() {
		var block *types.Block
		if hashish(arg) {
			block = chainmgr.GetBlock(common.HexToHash(arg))
		} else {
			num, _ := strconv.Atoi(arg)
			block = chainmgr.GetBlockByNumber(uint64(num))
		}
		if block == nil {
			fmt.Println("{}")
			utils.Fatalf("block not found")
		} else {
			statedb := state.New(block.Root(), stateDb)
			fmt.Printf("%s\n", statedb.Dump())
			// fmt.Println(block)
		}
	}
}

func makedag(ctx *cli.Context) {
	chain, _, _ := utils.GetChain(ctx)
	pow := ethash.New(chain)
	fmt.Println("making cache")
	pow.UpdateCache(true)
	fmt.Println("making DAG")
	pow.UpdateDAG()
}

func version(c *cli.Context) {
	fmt.Printf(`%v
Version: %v
Protocol Version: %d
Network Id: %d
GO: %s
OS: %s
GOPATH=%s
GOROOT=%s
`, ClientIdentifier, Version, c.GlobalInt(utils.ProtocolVersionFlag.Name), c.GlobalInt(utils.NetworkIdFlag.Name), runtime.Version(), runtime.GOOS, os.Getenv("GOPATH"), runtime.GOROOT())
}

// hashish returns true for strings that look like hashes.
func hashish(x string) bool {
	_, err := strconv.Atoi(x)
	return err != nil
}

func readPassword(prompt string, warnTerm bool) (string, error) {
	if liner.TerminalSupported() {
		lr := liner.NewLiner()
		defer lr.Close()
		return lr.PasswordPrompt(prompt)
	}
	if warnTerm {
		fmt.Println("!! Unsupported terminal, password will be echoed.")
	}
	fmt.Print(prompt)
	input, err := bufio.NewReader(os.Stdin).ReadString('\n')
	fmt.Println()
	return input, err
}
