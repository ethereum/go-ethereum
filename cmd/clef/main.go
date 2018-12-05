// Copyright 2018 The go-ethereum Authors
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

// signer is a utility that can be used so sign transactions and
// arbitrary data.
package main

import (
	"bufio"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/signal"
	"os/user"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/ethereum/go-ethereum/cmd/utils"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/console"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/ethereum/go-ethereum/signer/core"
	"github.com/ethereum/go-ethereum/signer/rules"
	"github.com/ethereum/go-ethereum/signer/storage"
	"gopkg.in/urfave/cli.v1"
)

// ExternalAPIVersion -- see extapi_changelog.md
const ExternalAPIVersion = "4.0.0"

// InternalAPIVersion -- see intapi_changelog.md
const InternalAPIVersion = "3.0.0"

const legalWarning = `
WARNING! 

Clef is alpha software, and not yet publically released. This software has _not_ been audited, and there
are no guarantees about the workings of this software. It may contain severe flaws. You should not use this software
unless you agree to take full responsibility for doing so, and know what you are doing. 

TLDR; THIS IS NOT PRODUCTION-READY SOFTWARE! 

`

var (
	logLevelFlag = cli.IntFlag{
		Name:  "loglevel",
		Value: 4,
		Usage: "log level to emit to the screen",
	}
	advancedMode = cli.BoolFlag{
		Name:  "advanced",
		Usage: "If enabled, issues warnings instead of rejections for suspicious requests. Default off",
	}
	keystoreFlag = cli.StringFlag{
		Name:  "keystore",
		Value: filepath.Join(node.DefaultDataDir(), "keystore"),
		Usage: "Directory for the keystore",
	}
	configdirFlag = cli.StringFlag{
		Name:  "configdir",
		Value: DefaultConfigDir(),
		Usage: "Directory for Clef configuration",
	}
	rpcPortFlag = cli.IntFlag{
		Name:  "rpcport",
		Usage: "HTTP-RPC server listening port",
		Value: node.DefaultHTTPPort + 5,
	}
	signerSecretFlag = cli.StringFlag{
		Name:  "signersecret",
		Usage: "A file containing the (encrypted) master seed to encrypt Clef data, e.g. keystore credentials and ruleset hash",
	}
	dBFlag = cli.StringFlag{
		Name:  "4bytedb",
		Usage: "File containing 4byte-identifiers",
		Value: "./4byte.json",
	}
	customDBFlag = cli.StringFlag{
		Name:  "4bytedb-custom",
		Usage: "File used for writing new 4byte-identifiers submitted via API",
		Value: "./4byte-custom.json",
	}
	auditLogFlag = cli.StringFlag{
		Name:  "auditlog",
		Usage: "File used to emit audit logs. Set to \"\" to disable",
		Value: "audit.log",
	}
	ruleFlag = cli.StringFlag{
		Name:  "rules",
		Usage: "Enable rule-engine",
		Value: "rules.json",
	}
	stdiouiFlag = cli.BoolFlag{
		Name: "stdio-ui",
		Usage: "Use STDIN/STDOUT as a channel for an external UI. " +
			"This means that an STDIN/STDOUT is used for RPC-communication with a e.g. a graphical user " +
			"interface, and can be used when Clef is started by an external process.",
	}
	testFlag = cli.BoolFlag{
		Name:  "stdio-ui-test",
		Usage: "Mechanism to test interface between Clef and UI. Requires 'stdio-ui'.",
	}
	app         = cli.NewApp()
	initCommand = cli.Command{
		Action:    utils.MigrateFlags(initializeSecrets),
		Name:      "init",
		Usage:     "Initialize the signer, generate secret storage",
		ArgsUsage: "",
		Flags: []cli.Flag{
			logLevelFlag,
			configdirFlag,
		},
		Description: `
The init command generates a master seed which Clef can use to store credentials and data needed for 
the rule-engine to work.`,
	}
	attestCommand = cli.Command{
		Action:    utils.MigrateFlags(attestFile),
		Name:      "attest",
		Usage:     "Attest that a js-file is to be used",
		ArgsUsage: "<sha256sum>",
		Flags: []cli.Flag{
			logLevelFlag,
			configdirFlag,
			signerSecretFlag,
		},
		Description: `
The attest command stores the sha256 of the rule.js-file that you want to use for automatic processing of 
incoming requests. 

Whenever you make an edit to the rule file, you need to use attestation to tell 
Clef that the file is 'safe' to execute.`,
	}

	setCredentialCommand = cli.Command{
		Action:    utils.MigrateFlags(setCredential),
		Name:      "setpw",
		Usage:     "Store a credential for a keystore file",
		ArgsUsage: "<address>",
		Flags: []cli.Flag{
			logLevelFlag,
			configdirFlag,
			signerSecretFlag,
		},
		Description: `
		The setpw command stores a password for a given address (keyfile). If you enter a blank passphrase, it will 
remove any stored credential for that address (keyfile)
`,
	}
)

func init() {
	app.Name = "Clef"
	app.Usage = "Manage Ethereum account operations"
	app.Flags = []cli.Flag{
		logLevelFlag,
		keystoreFlag,
		configdirFlag,
		utils.NetworkIdFlag,
		utils.LightKDFFlag,
		utils.NoUSBFlag,
		utils.RPCListenAddrFlag,
		utils.RPCVirtualHostsFlag,
		utils.IPCDisabledFlag,
		utils.IPCPathFlag,
		utils.RPCEnabledFlag,
		rpcPortFlag,
		signerSecretFlag,
		dBFlag,
		customDBFlag,
		auditLogFlag,
		ruleFlag,
		stdiouiFlag,
		testFlag,
		advancedMode,
	}
	app.Action = signer
	app.Commands = []cli.Command{initCommand, attestCommand, setCredentialCommand}

}
func main() {
	if err := app.Run(os.Args); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func initializeSecrets(c *cli.Context) error {
	if err := initialize(c); err != nil {
		return err
	}
	configDir := c.GlobalString(configdirFlag.Name)

	masterSeed := make([]byte, 256)
	num, err := io.ReadFull(rand.Reader, masterSeed)
	if err != nil {
		return err
	}
	if num != len(masterSeed) {
		return fmt.Errorf("failed to read enough random")
	}

	n, p := keystore.StandardScryptN, keystore.StandardScryptP
	if c.GlobalBool(utils.LightKDFFlag.Name) {
		n, p = keystore.LightScryptN, keystore.LightScryptP
	}
	text := "The master seed of clef is locked with a password. Please give a password. Do not forget this password."
	var password string
	for {
		password = getPassPhrase(text, true)
		if err := core.ValidatePasswordFormat(password); err != nil {
			fmt.Printf("invalid password: %v\n", err)
		} else {
			break
		}
	}
	cipherSeed, err := encryptSeed(masterSeed, []byte(password), n, p)
	if err != nil {
		return fmt.Errorf("failed to encrypt master seed: %v", err)
	}

	err = os.Mkdir(configDir, 0700)
	if err != nil && !os.IsExist(err) {
		return err
	}
	location := filepath.Join(configDir, "masterseed.json")
	if _, err := os.Stat(location); err == nil {
		return fmt.Errorf("file %v already exists, will not overwrite", location)
	}
	err = ioutil.WriteFile(location, cipherSeed, 0400)
	if err != nil {
		return err
	}
	fmt.Printf("A master seed has been generated into %s\n", location)
	fmt.Printf(`
This is required to be able to store credentials, such as : 
* Passwords for keystores (used by rule engine)
* Storage for javascript rules
* Hash of rule-file

You should treat that file with utmost secrecy, and make a backup of it. 
NOTE: This file does not contain your accounts. Those need to be backed up separately!

`)
	return nil
}
func attestFile(ctx *cli.Context) error {
	if len(ctx.Args()) < 1 {
		utils.Fatalf("This command requires an argument.")
	}
	if err := initialize(ctx); err != nil {
		return err
	}

	stretchedKey, err := readMasterKey(ctx, nil)
	if err != nil {
		utils.Fatalf(err.Error())
	}
	configDir := ctx.GlobalString(configdirFlag.Name)
	vaultLocation := filepath.Join(configDir, common.Bytes2Hex(crypto.Keccak256([]byte("vault"), stretchedKey)[:10]))
	confKey := crypto.Keccak256([]byte("config"), stretchedKey)

	// Initialize the encrypted storages
	configStorage := storage.NewAESEncryptedStorage(filepath.Join(vaultLocation, "config.json"), confKey)
	val := ctx.Args().First()
	configStorage.Put("ruleset_sha256", val)
	log.Info("Ruleset attestation updated", "sha256", val)
	return nil
}

func setCredential(ctx *cli.Context) error {
	if len(ctx.Args()) < 1 {
		utils.Fatalf("This command requires an address to be passed as an argument.")
	}
	if err := initialize(ctx); err != nil {
		return err
	}

	address := ctx.Args().First()
	password := getPassPhrase("Enter a passphrase to store with this address.", true)

	stretchedKey, err := readMasterKey(ctx, nil)
	if err != nil {
		utils.Fatalf(err.Error())
	}
	configDir := ctx.GlobalString(configdirFlag.Name)
	vaultLocation := filepath.Join(configDir, common.Bytes2Hex(crypto.Keccak256([]byte("vault"), stretchedKey)[:10]))
	pwkey := crypto.Keccak256([]byte("credentials"), stretchedKey)

	// Initialize the encrypted storages
	pwStorage := storage.NewAESEncryptedStorage(filepath.Join(vaultLocation, "credentials.json"), pwkey)
	pwStorage.Put(address, password)
	log.Info("Credential store updated", "key", address)
	return nil
}

func initialize(c *cli.Context) error {
	// Set up the logger to print everything
	logOutput := os.Stdout
	if c.GlobalBool(stdiouiFlag.Name) {
		logOutput = os.Stderr
		// If using the stdioui, we can't do the 'confirm'-flow
		fmt.Fprintf(logOutput, legalWarning)
	} else {
		if !confirm(legalWarning) {
			return fmt.Errorf("aborted by user")
		}
	}

	log.Root().SetHandler(log.LvlFilterHandler(log.Lvl(c.Int(logLevelFlag.Name)), log.StreamHandler(logOutput, log.TerminalFormat(true))))
	return nil
}

func signer(c *cli.Context) error {
	if err := initialize(c); err != nil {
		return err
	}
	var (
		ui core.SignerUI
	)
	if c.GlobalBool(stdiouiFlag.Name) {
		log.Info("Using stdin/stdout as UI-channel")
		ui = core.NewStdIOUI()
	} else {
		log.Info("Using CLI as UI-channel")
		ui = core.NewCommandlineUI()
	}
	fourByteDb := c.GlobalString(dBFlag.Name)
	fourByteLocal := c.GlobalString(customDBFlag.Name)
	db, err := core.NewAbiDBFromFiles(fourByteDb, fourByteLocal)
	if err != nil {
		utils.Fatalf(err.Error())
	}
	log.Info("Loaded 4byte db", "signatures", db.Size(), "file", fourByteDb, "local", fourByteLocal)

	var (
		api core.ExternalAPI
	)

	configDir := c.GlobalString(configdirFlag.Name)
	if stretchedKey, err := readMasterKey(c, ui); err != nil {
		log.Info("No master seed provided, rules disabled", "error", err)
	} else {

		if err != nil {
			utils.Fatalf(err.Error())
		}
		vaultLocation := filepath.Join(configDir, common.Bytes2Hex(crypto.Keccak256([]byte("vault"), stretchedKey)[:10]))

		// Generate domain specific keys
		pwkey := crypto.Keccak256([]byte("credentials"), stretchedKey)
		jskey := crypto.Keccak256([]byte("jsstorage"), stretchedKey)
		confkey := crypto.Keccak256([]byte("config"), stretchedKey)

		// Initialize the encrypted storages
		pwStorage := storage.NewAESEncryptedStorage(filepath.Join(vaultLocation, "credentials.json"), pwkey)
		jsStorage := storage.NewAESEncryptedStorage(filepath.Join(vaultLocation, "jsstorage.json"), jskey)
		configStorage := storage.NewAESEncryptedStorage(filepath.Join(vaultLocation, "config.json"), confkey)

		//Do we have a rule-file?
		ruleJS, err := ioutil.ReadFile(c.GlobalString(ruleFlag.Name))
		if err != nil {
			log.Info("Could not load rulefile, rules not enabled", "file", "rulefile")
		} else {
			hasher := sha256.New()
			hasher.Write(ruleJS)
			shasum := hasher.Sum(nil)
			storedShasum := configStorage.Get("ruleset_sha256")
			if storedShasum != hex.EncodeToString(shasum) {
				log.Info("Could not validate ruleset hash, rules not enabled", "got", hex.EncodeToString(shasum), "expected", storedShasum)
			} else {
				// Initialize rules
				ruleEngine, err := rules.NewRuleEvaluator(ui, jsStorage, pwStorage)
				if err != nil {
					utils.Fatalf(err.Error())
				}
				ruleEngine.Init(string(ruleJS))
				ui = ruleEngine
				log.Info("Rule engine configured", "file", c.String(ruleFlag.Name))
			}
		}
	}

	apiImpl := core.NewSignerAPI(
		c.GlobalInt64(utils.NetworkIdFlag.Name),
		c.GlobalString(keystoreFlag.Name),
		c.GlobalBool(utils.NoUSBFlag.Name),
		ui, db,
		c.GlobalBool(utils.LightKDFFlag.Name),
		c.GlobalBool(advancedMode.Name))
	api = apiImpl
	// Audit logging
	if logfile := c.GlobalString(auditLogFlag.Name); logfile != "" {
		api, err = core.NewAuditLogger(logfile, api)
		if err != nil {
			utils.Fatalf(err.Error())
		}
		log.Info("Audit logs configured", "file", logfile)
	}
	// register signer API with server
	var (
		extapiURL = "n/a"
		ipcapiURL = "n/a"
	)
	rpcAPI := []rpc.API{
		{
			Namespace: "account",
			Public:    true,
			Service:   api,
			Version:   "1.0"},
	}
	if c.GlobalBool(utils.RPCEnabledFlag.Name) {

		vhosts := splitAndTrim(c.GlobalString(utils.RPCVirtualHostsFlag.Name))
		cors := splitAndTrim(c.GlobalString(utils.RPCCORSDomainFlag.Name))

		// start http server
		httpEndpoint := fmt.Sprintf("%s:%d", c.GlobalString(utils.RPCListenAddrFlag.Name), c.Int(rpcPortFlag.Name))
		listener, _, err := rpc.StartHTTPEndpoint(httpEndpoint, rpcAPI, []string{"account"}, cors, vhosts, rpc.DefaultHTTPTimeouts)
		if err != nil {
			utils.Fatalf("Could not start RPC api: %v", err)
		}
		extapiURL = fmt.Sprintf("http://%s", httpEndpoint)
		log.Info("HTTP endpoint opened", "url", extapiURL)

		defer func() {
			listener.Close()
			log.Info("HTTP endpoint closed", "url", httpEndpoint)
		}()

	}
	if !c.GlobalBool(utils.IPCDisabledFlag.Name) {
		if c.IsSet(utils.IPCPathFlag.Name) {
			ipcapiURL = c.GlobalString(utils.IPCPathFlag.Name)
		} else {
			ipcapiURL = filepath.Join(configDir, "clef.ipc")
		}

		listener, _, err := rpc.StartIPCEndpoint(ipcapiURL, rpcAPI)
		if err != nil {
			utils.Fatalf("Could not start IPC api: %v", err)
		}
		log.Info("IPC endpoint opened", "url", ipcapiURL)
		defer func() {
			listener.Close()
			log.Info("IPC endpoint closed", "url", ipcapiURL)
		}()

	}

	if c.GlobalBool(testFlag.Name) {
		log.Info("Performing UI test")
		go testExternalUI(apiImpl)
	}
	ui.OnSignerStartup(core.StartupInfo{
		Info: map[string]interface{}{
			"extapi_version": ExternalAPIVersion,
			"intapi_version": InternalAPIVersion,
			"extapi_http":    extapiURL,
			"extapi_ipc":     ipcapiURL,
		},
	})

	abortChan := make(chan os.Signal)
	signal.Notify(abortChan, os.Interrupt)

	sig := <-abortChan
	log.Info("Exiting...", "signal", sig)

	return nil
}

// splitAndTrim splits input separated by a comma
// and trims excessive white space from the substrings.
func splitAndTrim(input string) []string {
	result := strings.Split(input, ",")
	for i, r := range result {
		result[i] = strings.TrimSpace(r)
	}
	return result
}

// DefaultConfigDir is the default config directory to use for the vaults and other
// persistence requirements.
func DefaultConfigDir() string {
	// Try to place the data folder in the user's home dir
	home := homeDir()
	if home != "" {
		if runtime.GOOS == "darwin" {
			return filepath.Join(home, "Library", "Signer")
		} else if runtime.GOOS == "windows" {
			return filepath.Join(home, "AppData", "Roaming", "Signer")
		} else {
			return filepath.Join(home, ".clef")
		}
	}
	// As we cannot guess a stable location, return empty and handle later
	return ""
}

func homeDir() string {
	if home := os.Getenv("HOME"); home != "" {
		return home
	}
	if usr, err := user.Current(); err == nil {
		return usr.HomeDir
	}
	return ""
}
func readMasterKey(ctx *cli.Context, ui core.SignerUI) ([]byte, error) {
	var (
		file      string
		configDir = ctx.GlobalString(configdirFlag.Name)
	)
	if ctx.GlobalIsSet(signerSecretFlag.Name) {
		file = ctx.GlobalString(signerSecretFlag.Name)
	} else {
		file = filepath.Join(configDir, "masterseed.json")
	}
	if err := checkFile(file); err != nil {
		return nil, err
	}
	cipherKey, err := ioutil.ReadFile(file)
	if err != nil {
		return nil, err
	}
	var password string
	// If ui is not nil, get the password from ui.
	if ui != nil {
		resp, err := ui.OnInputRequired(core.UserInputRequest{
			Title:      "Master Password",
			Prompt:     "Please enter the password to decrypt the master seed",
			IsPassword: true})
		if err != nil {
			return nil, err
		}
		password = resp.Text
	} else {
		password = getPassPhrase("Decrypt master seed of clef", false)
	}
	masterSeed, err := decryptSeed(cipherKey, password)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt the master seed of clef")
	}
	if len(masterSeed) < 256 {
		return nil, fmt.Errorf("master seed of insufficient length, expected >255 bytes, got %d", len(masterSeed))
	}

	// Create vault location
	vaultLocation := filepath.Join(configDir, common.Bytes2Hex(crypto.Keccak256([]byte("vault"), masterSeed)[:10]))
	err = os.Mkdir(vaultLocation, 0700)
	if err != nil && !os.IsExist(err) {
		return nil, err
	}
	return masterSeed, nil
}

// checkFile is a convenience function to check if a file
// * exists
// * is mode 0400
func checkFile(filename string) error {
	info, err := os.Stat(filename)
	if err != nil {
		return fmt.Errorf("failed stat on %s: %v", filename, err)
	}
	// Check the unix permission bits
	if info.Mode().Perm()&0377 != 0 {
		return fmt.Errorf("file (%v) has insecure file permissions (%v)", filename, info.Mode().String())
	}
	return nil
}

// confirm displays a text and asks for user confirmation
func confirm(text string) bool {
	fmt.Printf(text)
	fmt.Printf("\nEnter 'ok' to proceed:\n>")

	text, err := bufio.NewReader(os.Stdin).ReadString('\n')
	if err != nil {
		log.Crit("Failed to read user input", "err", err)
	}

	if text := strings.TrimSpace(text); text == "ok" {
		return true
	}
	return false
}

func testExternalUI(api *core.SignerAPI) {

	ctx := context.WithValue(context.Background(), "remote", "clef binary")
	ctx = context.WithValue(ctx, "scheme", "in-proc")
	ctx = context.WithValue(ctx, "local", "main")

	errs := make([]string, 0)

	api.UI.ShowInfo("Testing 'ShowInfo'")
	api.UI.ShowError("Testing 'ShowError'")

	checkErr := func(method string, err error) {
		if err != nil && err != core.ErrRequestDenied {
			errs = append(errs, fmt.Sprintf("%v: %v", method, err.Error()))
		}
	}
	var err error

	_, err = api.SignTransaction(ctx, core.SendTxArgs{From: common.MixedcaseAddress{}}, nil)
	checkErr("SignTransaction", err)
	_, err = api.Sign(ctx, common.MixedcaseAddress{}, common.Hex2Bytes("01020304"))
	checkErr("Sign", err)
	_, err = api.List(ctx)
	checkErr("List", err)
	_, err = api.New(ctx)
	checkErr("New", err)
	_, err = api.Export(ctx, common.Address{})
	checkErr("Export", err)
	_, err = api.Import(ctx, json.RawMessage{})
	checkErr("Import", err)

	api.UI.ShowInfo("Tests completed")

	if len(errs) > 0 {
		log.Error("Got errors")
		for _, e := range errs {
			log.Error(e)
		}
	} else {
		log.Info("No errors")
	}

}

// getPassPhrase retrieves the password associated with clef, either fetched
// from a list of preloaded passphrases, or requested interactively from the user.
// TODO: there are many `getPassPhrase` functions, it will be better to abstract them into one.
func getPassPhrase(prompt string, confirmation bool) string {
	fmt.Println(prompt)
	password, err := console.Stdin.PromptPassword("Passphrase: ")
	if err != nil {
		utils.Fatalf("Failed to read passphrase: %v", err)
	}
	if confirmation {
		confirm, err := console.Stdin.PromptPassword("Repeat passphrase: ")
		if err != nil {
			utils.Fatalf("Failed to read passphrase confirmation: %v", err)
		}
		if password != confirm {
			utils.Fatalf("Passphrases do not match")
		}
	}
	return password
}

type encryptedSeedStorage struct {
	Description string              `json:"description"`
	Version     int                 `json:"version"`
	Params      keystore.CryptoJSON `json:"params"`
}

// encryptSeed uses a similar scheme as the keystore uses, but with a different wrapping,
// to encrypt the master seed
func encryptSeed(seed []byte, auth []byte, scryptN, scryptP int) ([]byte, error) {
	cryptoStruct, err := keystore.EncryptDataV3(seed, auth, scryptN, scryptP)
	if err != nil {
		return nil, err
	}
	return json.Marshal(&encryptedSeedStorage{"Clef seed", 1, cryptoStruct})
}

// decryptSeed decrypts the master seed
func decryptSeed(keyjson []byte, auth string) ([]byte, error) {
	var encSeed encryptedSeedStorage
	if err := json.Unmarshal(keyjson, &encSeed); err != nil {
		return nil, err
	}
	if encSeed.Version != 1 {
		log.Warn(fmt.Sprintf("unsupported encryption format of seed: %d, operation will likely fail", encSeed.Version))
	}
	seed, err := keystore.DecryptDataV3(encSeed.Params, auth)
	if err != nil {
		return nil, err
	}
	return seed, err
}

/**
//Create Account

curl -H "Content-Type: application/json" -X POST --data '{"jsonrpc":"2.0","method":"account_new","params":["test"],"id":67}' localhost:8550

// List accounts

curl -i -H "Content-Type: application/json" -X POST --data '{"jsonrpc":"2.0","method":"account_list","params":[""],"id":67}' http://localhost:8550/

// Make Transaction
// safeSend(0x12)
// 4401a6e40000000000000000000000000000000000000000000000000000000000000012

// supplied abi
curl -i -H "Content-Type: application/json" -X POST --data '{"jsonrpc":"2.0","method":"account_signTransaction","params":[{"from":"0x82A2A876D39022B3019932D30Cd9c97ad5616813","gas":"0x333","gasPrice":"0x123","nonce":"0x0","to":"0x07a565b7ed7d7a678680a4c162885bedbb695fe0", "value":"0x10", "data":"0x4401a6e40000000000000000000000000000000000000000000000000000000000000012"},"test"],"id":67}' http://localhost:8550/

// Not supplied
curl -i -H "Content-Type: application/json" -X POST --data '{"jsonrpc":"2.0","method":"account_signTransaction","params":[{"from":"0x82A2A876D39022B3019932D30Cd9c97ad5616813","gas":"0x333","gasPrice":"0x123","nonce":"0x0","to":"0x07a565b7ed7d7a678680a4c162885bedbb695fe0", "value":"0x10", "data":"0x4401a6e40000000000000000000000000000000000000000000000000000000000000012"}],"id":67}' http://localhost:8550/

// Sign data

curl -i -H "Content-Type: application/json" -X POST --data '{"jsonrpc":"2.0","method":"account_sign","params":["0x694267f14675d7e1b9494fd8d72fefe1755710fa","bazonk gaz baz"],"id":67}' http://localhost:8550/


**/
