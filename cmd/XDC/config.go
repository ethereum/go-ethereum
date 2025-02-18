// Copyright 2017 The go-ethereum Authors
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
	"bufio"
	"errors"
	"fmt"
	"io"
	"math/big"
	"os"
	"reflect"
	"slices"
	"strings"
	"unicode"

	"github.com/XinFinOrg/XDPoSChain/XDCx"
	"github.com/XinFinOrg/XDPoSChain/accounts/keystore"
	"github.com/XinFinOrg/XDPoSChain/accounts/scwallet"
	"github.com/XinFinOrg/XDPoSChain/accounts/usbwallet"
	"github.com/XinFinOrg/XDPoSChain/cmd/utils"
	"github.com/XinFinOrg/XDPoSChain/common"
	"github.com/XinFinOrg/XDPoSChain/eth/ethconfig"
	"github.com/XinFinOrg/XDPoSChain/internal/flags"
	"github.com/XinFinOrg/XDPoSChain/log"
	"github.com/XinFinOrg/XDPoSChain/metrics"
	"github.com/XinFinOrg/XDPoSChain/node"
	"github.com/XinFinOrg/XDPoSChain/params"
	"github.com/naoina/toml"
	"github.com/urfave/cli/v2"
)

var (
	dumpConfigCommand = &cli.Command{
		Action:      dumpConfig,
		Name:        "dumpconfig",
		Usage:       "Show configuration values",
		ArgsUsage:   "",
		Flags:       slices.Concat(nodeFlags, rpcFlags),
		Description: `The dumpconfig command shows configuration values.`,
	}

	configFileFlag = &cli.StringFlag{
		Name:     "config",
		Usage:    "TOML configuration file",
		Category: flags.EthCategory,
	}
)

// These settings ensure that TOML keys use the same names as Go struct fields.
var tomlSettings = toml.Config{
	NormFieldName: func(rt reflect.Type, key string) string {
		return key
	},
	FieldToKey: func(rt reflect.Type, field string) string {
		return field
	},
	MissingField: func(rt reflect.Type, field string) error {
		link := ""
		if unicode.IsUpper(rune(rt.Name()[0])) && rt.PkgPath() != "main" {
			link = fmt.Sprintf(", see https://godoc.org/%s#%s for available fields", rt.PkgPath(), rt.Name())
		}
		return fmt.Errorf("field '%s' is not defined in %s%s", field, rt.String(), link)
	},
}

type ethstatsConfig struct {
	URL string
}

type account struct {
	Unlocks   []string
	Passwords []string
}

type Bootnodes struct {
	Mainnet []string
	Testnet []string
}

type XDCConfig struct {
	Eth       ethconfig.Config
	Node      node.Config
	Ethstats  ethstatsConfig
	Metrics   metrics.Config
	XDCX      XDCx.Config
	Account   account
	Bootnodes Bootnodes
	Verbosity int
	NAT       string
}

func loadConfig(file string, cfg *XDCConfig) error {
	f, err := os.Open(file)
	if err != nil {
		return err
	}
	defer f.Close()
	err = tomlSettings.NewDecoder(bufio.NewReader(f)).Decode(cfg)
	// Add file name to errors that have a line number.
	if _, ok := err.(*toml.LineError); ok {
		err = errors.New(file + ", " + err.Error())
	}
	return err
}

func defaultNodeConfig() node.Config {
	cfg := node.DefaultConfig
	cfg.Name = clientIdentifier
	cfg.Version = params.VersionWithCommit(gitCommit)
	cfg.HTTPModules = append(cfg.HTTPModules, "eth")
	cfg.WSModules = append(cfg.WSModules, "eth")
	cfg.IPCPath = "XDC.ipc"
	return cfg
}

// makeConfigNode loads geth configuration and creates a blank node instance.
func makeConfigNode(ctx *cli.Context) (*node.Node, XDCConfig) {
	// Load defaults.
	cfg := XDCConfig{
		Eth:       ethconfig.Defaults,
		XDCX:      XDCx.DefaultConfig,
		Node:      defaultNodeConfig(),
		Metrics:   metrics.DefaultConfig,
		Verbosity: 3,
		NAT:       "",
	}
	// Load config file.
	if file := ctx.String(configFileFlag.Name); file != "" {
		if err := loadConfig(file, &cfg); err != nil {
			utils.Fatalf("%v", err)
		}
	}
	if ctx.IsSet(utils.MiningEnabledFlag.Name) {
		log.Warn("The flag --mine is deprecated and will be removed")
	}
	// if !ctx.IsSet(debug.VerbosityFlag.Name) {
	// 	debug.Verbosity(log.Lvl(cfg.Verbosity))
	// }

	if !ctx.IsSet(utils.NATFlag.Name) && cfg.NAT != "" {
		ctx.Set(utils.NATFlag.Name, cfg.NAT)
	}

	// Check testnet is enable.
	if ctx.Bool(utils.XDCTestnetFlag.Name) {
		common.IsTestnet = true
		common.TRC21IssuerSMC = common.TRC21IssuerSMCTestNet
		cfg.Eth.NetworkId = 51
		common.RelayerRegistrationSMC = common.RelayerRegistrationSMCTestnet
		common.TIPTRC21Fee = common.TIPTRC21FeeTestnet
		common.TIPXDCXCancellationFee = common.TIPXDCXCancellationFeeTestnet
	}

	if ctx.Bool(utils.EnableXDCPrefixFlag.Name) {
		common.Enable0xPrefix = false
	}

	// Rewound
	if rewound := ctx.Int(utils.RewoundFlag.Name); rewound != 0 {
		common.Rewound = uint64(rewound)
	}

	// Check rollback hash exist.
	if rollbackHash := ctx.String(utils.RollbackFlag.Name); rollbackHash != "" {
		common.RollbackHash = common.HexToHash(rollbackHash)
	}

	// Check GasPrice
	common.MinGasPrice = big.NewInt(common.DefaultMinGasPrice)
	if ctx.IsSet(utils.MinerGasPriceFlag.Name) {
		if gasPrice := int64(ctx.Int(utils.MinerGasPriceFlag.Name)); gasPrice > common.DefaultMinGasPrice {
			common.MinGasPrice = big.NewInt(gasPrice)
		}
	}
	common.MinGasPrice50x = common.MinGasPrice50x.Mul(common.MinGasPrice, big.NewInt(50))

	// read passwords from environment
	passwords := []string{}
	for _, env := range cfg.Account.Passwords {
		if trimmed := strings.TrimSpace(env); trimmed != "" {
			value := os.Getenv(trimmed)
			for _, info := range strings.Split(value, ",") {
				if trimmed2 := strings.TrimSpace(info); trimmed2 != "" {
					passwords = append(passwords, trimmed2)
				}
			}
		}
	}
	cfg.Account.Passwords = passwords

	// Apply flags.
	utils.SetNodeConfig(ctx, &cfg.Node)
	stack, err := node.New(&cfg.Node)
	if err != nil {
		utils.Fatalf("Failed to create the protocol stack: %v", err)
	}
	// Node doesn't by default populate account manager backends
	if err := setAccountManagerBackends(stack); err != nil {
		utils.Fatalf("Failed to set account manager backends: %v", err)
	}

	utils.SetEthConfig(ctx, stack, &cfg.Eth)
	if ctx.IsSet(utils.EthStatsURLFlag.Name) {
		cfg.Ethstats.URL = ctx.String(utils.EthStatsURLFlag.Name)
	}

	utils.SetXDCXConfig(ctx, &cfg.XDCX, cfg.Node.DataDir)

	applyMetricConfig(ctx, &cfg)

	return stack, cfg
}

func applyValues(values []string, params *[]string) {
	data := []string{}
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			data = append(data, trimmed)
		}
	}
	if len(data) > 0 {
		*params = data
	}

}

func makeFullNode(ctx *cli.Context) (*node.Node, XDCConfig) {
	stack, cfg := makeConfigNode(ctx)

	// Start metrics export if enabled
	utils.SetupMetrics(&cfg.Metrics)

	// Register XDCX's OrderBook service if requested.
	// enable in default
	utils.RegisterXDCXService(stack, &cfg.XDCX)
	utils.RegisterEthService(stack, &cfg.Eth, cfg.Node.Version)

	// Add the Ethereum Stats daemon if requested.
	if cfg.Ethstats.URL != "" {
		utils.RegisterEthStatsService(stack, cfg.Ethstats.URL)
	}

	return stack, cfg
}

// dumpConfig is the dumpconfig command.
func dumpConfig(ctx *cli.Context) error {
	_, cfg := makeConfigNode(ctx)
	comment := ""

	if cfg.Eth.Genesis != nil {
		cfg.Eth.Genesis = nil
		comment += "# Note: this config doesn't contain the genesis block.\n\n"
	}

	out, err := tomlSettings.Marshal(&cfg)
	if err != nil {
		return err
	}
	io.WriteString(os.Stdout, comment)
	os.Stdout.Write(out)
	return nil
}

func applyMetricConfig(ctx *cli.Context, cfg *XDCConfig) {
	if ctx.IsSet(utils.MetricsEnabledFlag.Name) {
		cfg.Metrics.Enabled = ctx.Bool(utils.MetricsEnabledFlag.Name)
	}
	if ctx.IsSet(utils.MetricsEnabledExpensiveFlag.Name) {
		log.Warn("Expensive metrics are collected by default, please remove this flag", "flag", utils.MetricsEnabledExpensiveFlag.Name)
	}
	if ctx.IsSet(utils.MetricsHTTPFlag.Name) {
		cfg.Metrics.HTTP = ctx.String(utils.MetricsHTTPFlag.Name)
	}
	if ctx.IsSet(utils.MetricsPortFlag.Name) {
		cfg.Metrics.Port = ctx.Int(utils.MetricsPortFlag.Name)
	}
	if ctx.IsSet(utils.MetricsEnableInfluxDBFlag.Name) {
		cfg.Metrics.EnableInfluxDB = ctx.Bool(utils.MetricsEnableInfluxDBFlag.Name)
	}
	if ctx.IsSet(utils.MetricsInfluxDBEndpointFlag.Name) {
		cfg.Metrics.InfluxDBEndpoint = ctx.String(utils.MetricsInfluxDBEndpointFlag.Name)
	}
	if ctx.IsSet(utils.MetricsInfluxDBDatabaseFlag.Name) {
		cfg.Metrics.InfluxDBDatabase = ctx.String(utils.MetricsInfluxDBDatabaseFlag.Name)
	}
	if ctx.IsSet(utils.MetricsInfluxDBUsernameFlag.Name) {
		cfg.Metrics.InfluxDBUsername = ctx.String(utils.MetricsInfluxDBUsernameFlag.Name)
	}
	if ctx.IsSet(utils.MetricsInfluxDBPasswordFlag.Name) {
		cfg.Metrics.InfluxDBPassword = ctx.String(utils.MetricsInfluxDBPasswordFlag.Name)
	}
	if ctx.IsSet(utils.MetricsInfluxDBTagsFlag.Name) {
		cfg.Metrics.InfluxDBTags = ctx.String(utils.MetricsInfluxDBTagsFlag.Name)
	}
	if ctx.IsSet(utils.MetricsEnableInfluxDBV2Flag.Name) {
		cfg.Metrics.EnableInfluxDBV2 = ctx.Bool(utils.MetricsEnableInfluxDBV2Flag.Name)
	}
	if ctx.IsSet(utils.MetricsInfluxDBTokenFlag.Name) {
		cfg.Metrics.InfluxDBToken = ctx.String(utils.MetricsInfluxDBTokenFlag.Name)
	}
	if ctx.IsSet(utils.MetricsInfluxDBBucketFlag.Name) {
		cfg.Metrics.InfluxDBBucket = ctx.String(utils.MetricsInfluxDBBucketFlag.Name)
	}
	if ctx.IsSet(utils.MetricsInfluxDBOrganizationFlag.Name) {
		cfg.Metrics.InfluxDBOrganization = ctx.String(utils.MetricsInfluxDBOrganizationFlag.Name)
	}
	// Sanity-check the commandline flags. It is fine if some unused fields is part
	// of the toml-config, but we expect the commandline to only contain relevant
	// arguments, otherwise it indicates an error.
	var (
		enableExport   = ctx.Bool(utils.MetricsEnableInfluxDBFlag.Name)
		enableExportV2 = ctx.Bool(utils.MetricsEnableInfluxDBV2Flag.Name)
	)
	if enableExport || enableExportV2 {
		v1FlagIsSet := ctx.IsSet(utils.MetricsInfluxDBUsernameFlag.Name) ||
			ctx.IsSet(utils.MetricsInfluxDBPasswordFlag.Name)

		v2FlagIsSet := ctx.IsSet(utils.MetricsInfluxDBTokenFlag.Name) ||
			ctx.IsSet(utils.MetricsInfluxDBOrganizationFlag.Name) ||
			ctx.IsSet(utils.MetricsInfluxDBBucketFlag.Name)

		if enableExport && v2FlagIsSet {
			utils.Fatalf("Flags --influxdb-metrics.organization, --influxdb-metrics.token, --influxdb-metrics.bucket are only available for influxdb-v2")
		} else if enableExportV2 && v1FlagIsSet {
			utils.Fatalf("Flags --influxdb-metrics.username, --influxdb-metrics.password are only available for influxdb-v1")
		}
	}
}

func setAccountManagerBackends(stack *node.Node) error {
	conf := stack.Config()
	am := stack.AccountManager()
	keydir := stack.KeyStoreDir()
	scryptN := keystore.StandardScryptN
	scryptP := keystore.StandardScryptP
	if conf.UseLightweightKDF {
		scryptN = keystore.LightScryptN
		scryptP = keystore.LightScryptP
	}

	// For now, we're using EITHER external signer OR local signers.
	// If/when we implement some form of lockfile for USB and keystore wallets,
	// we can have both, but it's very confusing for the user to see the same
	// accounts in both externally and locally, plus very racey.
	am.AddBackend(keystore.NewKeyStore(keydir, scryptN, scryptP))
	if conf.USB {
		// Start a USB hub for Ledger hardware wallets
		if ledgerhub, err := usbwallet.NewLedgerHub(); err != nil {
			log.Warn(fmt.Sprintf("Failed to start Ledger hub, disabling: %v", err))
		} else {
			am.AddBackend(ledgerhub)
		}
		// Start a USB hub for Trezor hardware wallets (HID version)
		if trezorhub, err := usbwallet.NewTrezorHubWithHID(); err != nil {
			log.Warn(fmt.Sprintf("Failed to start HID Trezor hub, disabling: %v", err))
		} else {
			am.AddBackend(trezorhub)
		}
		// Start a USB hub for Trezor hardware wallets (WebUSB version)
		if trezorhub, err := usbwallet.NewTrezorHubWithWebUSB(); err != nil {
			log.Warn(fmt.Sprintf("Failed to start WebUSB Trezor hub, disabling: %v", err))
		} else {
			am.AddBackend(trezorhub)
		}
	}
	if len(conf.SmartCardDaemonPath) > 0 {
		// Start a smart card hub
		if schub, err := scwallet.NewHub(conf.SmartCardDaemonPath, scwallet.Scheme, keydir); err != nil {
			log.Warn(fmt.Sprintf("Failed to start smart card hub, disabling: %v", err))
		} else {
			am.AddBackend(schub)
		}
	}

	return nil
}
