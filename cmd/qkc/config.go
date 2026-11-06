// Copyright 2026-2027 The quarkchain Authors
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
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/cmd/utils"
	"github.com/ethereum/go-ethereum/node"
	qkcconfig "github.com/ethereum/go-ethereum/qkc/config"
	"github.com/urfave/cli/v2"
)

var (
	configFileFlag = &cli.StringFlag{
		Name:  "config",
		Usage: "QKC deployment TOML file",
	}
	qkcRoleFlag = &cli.StringFlag{
		Name:  "qkc.role",
		Usage: "Process role: master or slave",
	}
	qkcChainFlag = &cli.StringFlag{
		Name:  "qkc.chain",
		Usage: "Chain ID to select from the deployment (required for slave role)",
	}
)

var qkcConfigFlags = []cli.Flag{
	configFileFlag, qkcRoleFlag, qkcChainFlag, utils.NetworkIdFlag,
	utils.DataDirFlag, utils.DBEngineFlag, utils.SyncModeFlag, utils.StateSchemeFlag,
	utils.ListenPortFlag, utils.MaxPeersFlag, utils.NoDiscoverFlag,
	utils.HTTPEnabledFlag, utils.HTTPListenAddrFlag, utils.HTTPPortFlag, utils.HTTPApiFlag,
	utils.WSEnabledFlag, utils.WSListenAddrFlag, utils.WSPortFlag, utils.WSApiFlag,
	utils.IPCDisabledFlag, utils.IPCPathFlag,
	utils.RPCGlobalGasCapFlag, utils.RPCGlobalEVMTimeoutFlag, utils.RPCGlobalTxFeeCapFlag,
}

var checkQKCConfigCommand = &cli.Command{
	Name:   "checkconfig",
	Usage:  "Validate QKC deployment settings without reading genesis files or starting services",
	Flags:  qkcConfigFlags,
	Action: checkQKCConfig,
}

// loadQKCConfig selects process settings and applies CLI overrides without reading genesis files.
func loadQKCConfig(ctx *cli.Context) (*qkcconfig.NodeConfig, *qkcconfig.ClusterConfig, error) {
	if ctx.NArg() != 0 {
		return nil, nil, fmt.Errorf("checkconfig takes no positional arguments")
	}
	cluster, err := qkcconfig.LoadCluster(ctx.String(configFileFlag.Name))
	if err != nil {
		return nil, nil, err
	}
	var chainID *big.Int
	if ctx.IsSet(qkcChainFlag.Name) {
		id, ok := new(big.Int).SetString(ctx.String(qkcChainFlag.Name), 10)
		if !ok || id.Sign() <= 0 || id.BitLen() > 256 {
			return nil, nil, fmt.Errorf("--qkc.chain must be a positive uint256 chain ID")
		}
		chainID = id
	}
	cfg, err := cluster.SelectNode(ctx.String(qkcRoleFlag.Name), chainID)
	if err != nil {
		return nil, nil, err
	}
	if cfg.Role == qkcconfig.RoleMaster {
		for _, flag := range []cli.Flag{utils.SyncModeFlag, utils.StateSchemeFlag, utils.RPCGlobalGasCapFlag, utils.RPCGlobalEVMTimeoutFlag, utils.RPCGlobalTxFeeCapFlag} {
			if ctx.IsSet(flag.Names()[0]) {
				return nil, nil, fmt.Errorf("--%s requires slave role", flag.Names()[0])
			}
		}
	}
	if ctx.IsSet(utils.NetworkIdFlag.Name) {
		if cfg.Eth != nil {
			cfg.Eth.NetworkId = ctx.Uint64(utils.NetworkIdFlag.Name)
		}
	}
	if ctx.IsSet(utils.DataDirFlag.Name) {
		cfg.Node.DataDir = ctx.String(utils.DataDirFlag.Name)
	}
	if ctx.IsSet(utils.DBEngineFlag.Name) {
		cfg.Node.DBEngine = ctx.String(utils.DBEngineFlag.Name)
	}
	if ctx.IsSet(utils.SyncModeFlag.Name) {
		if err := cfg.Eth.SyncMode.UnmarshalText([]byte(ctx.String(utils.SyncModeFlag.Name))); err != nil {
			return nil, nil, err
		}
	}
	if ctx.IsSet(utils.StateSchemeFlag.Name) {
		cfg.Eth.StateScheme = ctx.String(utils.StateSchemeFlag.Name)
	}
	if ctx.IsSet(utils.ListenPortFlag.Name) {
		cfg.Node.P2P.ListenAddr = fmt.Sprintf(":%d", ctx.Int(utils.ListenPortFlag.Name))
	}
	if ctx.IsSet(utils.MaxPeersFlag.Name) {
		cfg.Node.P2P.MaxPeers = ctx.Int(utils.MaxPeersFlag.Name)
	}
	if ctx.IsSet(utils.NoDiscoverFlag.Name) {
		cfg.Node.P2P.NoDiscovery = ctx.Bool(utils.NoDiscoverFlag.Name)
	}
	if ctx.IsSet(utils.HTTPPortFlag.Name) {
		cfg.Node.HTTPPort = ctx.Int(utils.HTTPPortFlag.Name)
	}
	if ctx.IsSet(utils.HTTPApiFlag.Name) {
		cfg.Node.HTTPModules = utils.SplitAndTrim(ctx.String(utils.HTTPApiFlag.Name))
	}
	if ctx.IsSet(utils.WSPortFlag.Name) {
		cfg.Node.WSPort = ctx.Int(utils.WSPortFlag.Name)
	}
	if ctx.IsSet(utils.WSApiFlag.Name) {
		cfg.Node.WSModules = utils.SplitAndTrim(ctx.String(utils.WSApiFlag.Name))
	}
	applyQKCEndpoint(ctx, utils.HTTPEnabledFlag.Name, utils.HTTPListenAddrFlag.Name, &cfg.Node.HTTPHost, node.DefaultHTTPHost)
	applyQKCEndpoint(ctx, utils.WSEnabledFlag.Name, utils.WSListenAddrFlag.Name, &cfg.Node.WSHost, node.DefaultWSHost)
	if ctx.IsSet(utils.IPCDisabledFlag.Name) && ctx.IsSet(utils.IPCPathFlag.Name) {
		return nil, nil, fmt.Errorf("--ipcdisable and --ipcpath are mutually exclusive")
	}
	if ctx.Bool(utils.IPCDisabledFlag.Name) {
		cfg.Node.IPCPath = ""
	}
	if ctx.IsSet(utils.IPCPathFlag.Name) {
		cfg.Node.IPCPath = ctx.String(utils.IPCPathFlag.Name)
	}
	if ctx.IsSet(utils.RPCGlobalGasCapFlag.Name) {
		cfg.Eth.RPCGasCap = ctx.Uint64(utils.RPCGlobalGasCapFlag.Name)
	}
	if ctx.IsSet(utils.RPCGlobalEVMTimeoutFlag.Name) {
		cfg.Eth.RPCEVMTimeout = ctx.Duration(utils.RPCGlobalEVMTimeoutFlag.Name)
	}
	if ctx.IsSet(utils.RPCGlobalTxFeeCapFlag.Name) {
		cfg.Eth.RPCTxFeeCap = ctx.Float64(utils.RPCGlobalTxFeeCapFlag.Name)
	}
	if ctx.IsSet(utils.NetworkIdFlag.Name) && ctx.Uint64(utils.NetworkIdFlag.Name) != cluster.NetworkID {
		return nil, nil, fmt.Errorf("--networkid must match the positive NetworkID %d in the deployment", cluster.NetworkID)
	}
	return cfg, cluster, nil
}

// Explicit false disables an endpoint configured in TOML; an address alone
// changes the address without enabling an endpoint that was disabled.
func applyQKCEndpoint(ctx *cli.Context, enabled, address string, host *string, defaultHost string) {
	if ctx.IsSet(address) && *host != "" {
		*host = ctx.String(address)
	}
	if ctx.IsSet(enabled) {
		if !ctx.Bool(enabled) {
			*host = ""
		} else if ctx.IsSet(address) {
			*host = ctx.String(address)
		} else if *host == "" {
			*host = defaultHost
		}
	}
}

func checkQKCConfig(ctx *cli.Context) error {
	cfg, cluster, err := loadQKCConfig(ctx)
	if err != nil {
		return err
	}
	if err := cluster.Validate(); err != nil {
		return err
	}
	identity := cfg.Role
	if cfg.ChainID != nil {
		identity = fmt.Sprintf("%s, chain ID %v", cfg.Role, cfg.ChainID)
	}
	_, err = fmt.Fprintf(ctx.App.Writer, "QuarkChain deployment configuration is valid (%s, network ID %d)\n", identity, cluster.NetworkID)
	return err
}
