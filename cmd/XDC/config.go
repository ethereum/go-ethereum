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
	"strings"
	"unicode"

	"gopkg.in/urfave/cli.v1"

	"github.com/XinFinOrg/XDPoSChain/XDCx"
	"github.com/XinFinOrg/XDPoSChain/cmd/utils"
	"github.com/XinFinOrg/XDPoSChain/common"
	"github.com/XinFinOrg/XDPoSChain/eth"
	"github.com/XinFinOrg/XDPoSChain/internal/debug"
	"github.com/XinFinOrg/XDPoSChain/log"
	"github.com/XinFinOrg/XDPoSChain/node"
	"github.com/XinFinOrg/XDPoSChain/params"
	whisper "github.com/XinFinOrg/XDPoSChain/whisper/whisperv6"
	"github.com/naoina/toml"
)

var (
	dumpConfigCommand = cli.Command{
		Action:      utils.MigrateFlags(dumpConfig),
		Name:        "dumpconfig",
		Usage:       "Show configuration values",
		ArgsUsage:   "",
		Flags:       append(append(nodeFlags, rpcFlags...), whisperFlags...),
		Category:    "MISCELLANEOUS COMMANDS",
		Description: `The dumpconfig command shows configuration values.`,
	}

	configFileFlag = cli.StringFlag{
		Name:  "config",
		Usage: "TOML configuration file",
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
	Eth         eth.Config
	Shh         whisper.Config
	Node        node.Config
	Ethstats    ethstatsConfig
	XDCX        XDCx.Config
	Account     account
	StakeEnable bool
	Bootnodes   Bootnodes
	Verbosity   int
	NAT         string
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
	cfg.HTTPModules = append(cfg.HTTPModules, "eth", "shh")
	cfg.WSModules = append(cfg.WSModules, "eth", "shh")
	cfg.IPCPath = "XDC.ipc"
	return cfg
}

func makeConfigNode(ctx *cli.Context) (*node.Node, XDCConfig) {
	// Load defaults.
	cfg := XDCConfig{
		Eth:         eth.DefaultConfig,
		Shh:         whisper.DefaultConfig,
		XDCX:        XDCx.DefaultConfig,
		Node:        defaultNodeConfig(),
		StakeEnable: true,
		Verbosity:   3,
		NAT:         "",
	}
	// Load config file.
	if file := ctx.GlobalString(configFileFlag.Name); file != "" {
		if err := loadConfig(file, &cfg); err != nil {
			utils.Fatalf("%v", err)
		}
	}
	if ctx.GlobalIsSet(utils.StakingEnabledFlag.Name) {
		cfg.StakeEnable = ctx.GlobalBool(utils.StakingEnabledFlag.Name)
	}
	if !ctx.GlobalIsSet(debug.VerbosityFlag.Name) {
		debug.Glogger.Verbosity(log.Lvl(cfg.Verbosity))
	}

	if !ctx.GlobalIsSet(utils.NATFlag.Name) && cfg.NAT != "" {
		ctx.Set(utils.NATFlag.Name, cfg.NAT)
	}

	// Check testnet is enable.
	if ctx.GlobalBool(utils.XDCTestnetFlag.Name) {
		common.IsTestnet = true
		common.TRC21IssuerSMC = common.TRC21IssuerSMCTestNet
		cfg.Eth.NetworkId = 51
		common.RelayerRegistrationSMC = common.RelayerRegistrationSMCTestnet
		common.TIPTRC21Fee = common.TIPXDCXTestnet
		common.TIPTRC21Fee = common.TIPTRC21FeeTestnet
		common.TIPXDCXCancellationFee = common.TIPXDCXCancellationFeeTestnet
	}

	if ctx.GlobalBool(utils.Enable0xPrefixFlag.Name) {
		common.Enable0xPrefix = true;
	}
	
	// Rewound
	if rewound := ctx.GlobalInt(utils.RewoundFlag.Name); rewound != 0 {
		common.Rewound = uint64(rewound)
	}

	// Check rollback hash exist.
	if rollbackHash := ctx.GlobalString(utils.RollbackFlag.Name); rollbackHash != "" {
		common.RollbackHash = common.HexToHash(rollbackHash)
	}

	// Check GasPrice
	common.MinGasPrice = big.NewInt(common.DefaultMinGasPrice)
	if ctx.GlobalIsSet(utils.GasPriceFlag.Name) {
		if gasPrice := int64(ctx.GlobalInt(utils.GasPriceFlag.Name)); gasPrice > common.DefaultMinGasPrice {
			common.MinGasPrice = big.NewInt(gasPrice)
		}
	}

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
	utils.SetEthConfig(ctx, stack, &cfg.Eth)
	if ctx.GlobalIsSet(utils.EthStatsURLFlag.Name) {
		cfg.Ethstats.URL = ctx.GlobalString(utils.EthStatsURLFlag.Name)
	}

	utils.SetShhConfig(ctx, stack, &cfg.Shh)
	utils.SetXDCXConfig(ctx, &cfg.XDCX, cfg.Node.DataDir)
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

// enableWhisper returns true in case one of the whisper flags is set.
func enableWhisper(ctx *cli.Context) bool {
	for _, flag := range whisperFlags {
		if ctx.GlobalIsSet(flag.GetName()) {
			return true
		}
	}
	return false
}

func makeFullNode(ctx *cli.Context) (*node.Node, XDCConfig) {
	stack, cfg := makeConfigNode(ctx)

	// Register XDCX's OrderBook service if requested.
	// enable in default
	utils.RegisterXDCXService(stack, &cfg.XDCX)
	utils.RegisterEthService(stack, &cfg.Eth)

	// Whisper must be explicitly enabled by specifying at least 1 whisper flag or in dev mode
	shhEnabled := enableWhisper(ctx)
	shhAutoEnabled := !ctx.GlobalIsSet(utils.WhisperEnabledFlag.Name) && ctx.GlobalIsSet(utils.DeveloperFlag.Name)
	if shhEnabled || shhAutoEnabled {
		if ctx.GlobalIsSet(utils.WhisperMaxMessageSizeFlag.Name) {
			cfg.Shh.MaxMessageSize = uint32(ctx.Int(utils.WhisperMaxMessageSizeFlag.Name))
		}
		if ctx.GlobalIsSet(utils.WhisperMinPOWFlag.Name) {
			cfg.Shh.MinimumAcceptedPOW = ctx.Float64(utils.WhisperMinPOWFlag.Name)
		}
		utils.RegisterShhService(stack, &cfg.Shh)
	}

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
