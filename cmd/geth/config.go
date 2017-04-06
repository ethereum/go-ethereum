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

package main

import (
	"bufio"
	"encoding/hex"
	"errors"
	"os"

	cli "gopkg.in/urfave/cli.v1"

	"github.com/ethereum/go-ethereum/cmd/utils"
	"github.com/ethereum/go-ethereum/contracts/release"
	"github.com/ethereum/go-ethereum/eth"
	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/params"
	"github.com/naoina/toml"
)

var (
	dumpConfigCommand = cli.Command{
		Action:      dumpConfig,
		Name:        "dumpconfig",
		Usage:       "Show configuration values",
		ArgsUsage:   "",
		Category:    "MISCELLANEOUS COMMANDS",
		Description: `The dumpconfig command shows configuration values.`,
	}

	configFileFlag = cli.StringFlag{
		Name:  "config",
		Usage: "TOML configuration file",
	}
)

var defaultNodeConfig = node.Config{
	Name:    clientIdentifier,
	Version: params.VersionWithCommit(gitCommit),
	DataDir: node.DefaultDataDir(),
	P2P: p2p.Config{
		MaxPeers:  25,
		Discovery: true,
	},
	HTTPModules: []string{"eth", "net", "web3"},
	WSModules:   []string{"eth", "net", "web3"},
}

type gethConfig struct {
	Eth  eth.Config
	Node node.Config
}

func loadConfig(file string, cfg *gethConfig) error {
	f, err := os.Open(file)
	if err != nil {
		return err
	}
	defer f.Close()
	err = toml.NewDecoder(bufio.NewReader(f)).Decode(cfg)
	// Add file name to errors that have a line number.
	if _, ok := err.(*toml.LineError); ok {
		err = errors.New(file + ", " + err.Error())
	}
	return err
}

func makeConfigNode(ctx *cli.Context) (*node.Node, gethConfig) {
	// Load defaults.
	cfg := gethConfig{
		Eth:  eth.DefaultConfig,
		Node: defaultNodeConfig,
	}
	// Load config file.
	if file := ctx.GlobalString(configFileFlag.Name); file != "" {
		if err := loadConfig(file, &cfg); err != nil {
			utils.Fatalf("%v", err)
		}
	}
	// Apply flags.
	utils.SetNodeConfig(ctx, clientIdentifier, gitCommit, &cfg.Node)
	stack, err := node.New(&cfg.Node)
	if err != nil {
		utils.Fatalf("Failed to create the protocol stack: %v", err)
	}
	utils.SetEthConfig(ctx, stack, &cfg.Eth)
	return stack, cfg
}

func makeFullNode(ctx *cli.Context) *node.Node {
	stack, cfg := makeConfigNode(ctx)

	utils.RegisterEthService(stack, &cfg.Eth)

	// Whisper must be explicitly enabled, but is auto-enabled in --dev mode.
	shhEnabled := ctx.GlobalBool(utils.WhisperEnabledFlag.Name)
	shhAutoEnabled := !ctx.GlobalIsSet(utils.WhisperEnabledFlag.Name) && ctx.GlobalIsSet(utils.DevModeFlag.Name)
	if shhEnabled || shhAutoEnabled {
		utils.RegisterShhService(stack)
	}
	// Add the Ethereum Stats daemon if requested
	if url := ctx.GlobalString(utils.EthStatsURLFlag.Name); url != "" {
		utils.RegisterEthStatsService(stack, url)
	}
	// Add the release oracle service so it boots along with node.
	if err := stack.Register(func(ctx *node.ServiceContext) (node.Service, error) {
		config := release.Config{
			Oracle: relOracle,
			Major:  uint32(params.VersionMajor),
			Minor:  uint32(params.VersionMinor),
			Patch:  uint32(params.VersionPatch),
		}
		commit, _ := hex.DecodeString(gitCommit)
		copy(config.Commit[:], commit)
		return release.NewReleaseService(ctx, config)
	}); err != nil {
		utils.Fatalf("Failed to register the Geth release oracle service: %v", err)
	}
	return stack
}

// dumpConfig is the dumpconfig command.
func dumpConfig(ctx *cli.Context) error {
	_, cfg := makeConfigNode(ctx)
	out, err := toml.Marshal(&cfg)
	if err != nil {
		return err
	}
	os.Stdout.Write(out)
	return nil
}
