// Copyright 2021 XDC Network
// This file is part of the XDC library.

package utils

import (
	"github.com/ethereum/go-ethereum/eth/ethconfig"
	"github.com/ethereum/go-ethereum/node"
	"github.com/urfave/cli/v2"
)

// XDC-specific command line flags
var (
	// XDPoS flags
	XDPoSRewardFlag = &cli.BoolFlag{
		Name:     "xdpos.rewards",
		Usage:    "Enable block rewards for validators",
		Value:    true,
		Category: "XDPoS",
	}
	XDPoSSlashingFlag = &cli.BoolFlag{
		Name:     "xdpos.slashing",
		Usage:    "Enable slashing for misbehaving validators",
		Value:    true,
		Category: "XDPoS",
	}
	XDPoSValidatorFlag = &cli.StringFlag{
		Name:     "xdpos.validator",
		Usage:    "Public address for block validation (coinbase)",
		Category: "XDPoS",
	}
	XDPoSGapFlag = &cli.Uint64Flag{
		Name:     "xdpos.gap",
		Usage:    "Gap for snapshot creation in XDPoS",
		Value:    450,
		Category: "XDPoS",
	}
	XDPoSEpochFlag = &cli.Uint64Flag{
		Name:     "xdpos.epoch",
		Usage:    "Epoch length for validator set updates",
		Value:    900,
		Category: "XDPoS",
	}

	// XDCx flags
	XDCxEnableFlag = &cli.BoolFlag{
		Name:     "xdcx",
		Usage:    "Enable XDC decentralized exchange",
		Value:    false,
		Category: "XDCx",
	}
	XDCxDataDirFlag = &cli.StringFlag{
		Name:     "xdcx.datadir",
		Usage:    "Data directory for XDCx",
		Category: "XDCx",
	}

	// XDCxLending flags
	XDCxLendingEnableFlag = &cli.BoolFlag{
		Name:     "xdcxlending",
		Usage:    "Enable XDC lending protocol",
		Value:    false,
		Category: "XDCxLending",
	}
	XDCxLendingDataDirFlag = &cli.StringFlag{
		Name:     "xdcxlending.datadir",
		Usage:    "Data directory for XDCxLending",
		Category: "XDCxLending",
	}

	// Network flags
	XDCMainnetFlag = &cli.BoolFlag{
		Name:     "xdc.mainnet",
		Usage:    "Connect to XDC mainnet",
		Category: "XDC Network",
	}
	XDCTestnetFlag = &cli.BoolFlag{
		Name:     "xdc.testnet",
		Usage:    "Connect to XDC Apothem testnet",
		Category: "XDC Network",
	}
	XDCDevnetFlag = &cli.BoolFlag{
		Name:     "xdc.devnet",
		Usage:    "Connect to XDC devnet",
		Category: "XDC Network",
	}

	// Sync flags
	XDCSnapSyncFlag = &cli.BoolFlag{
		Name:     "xdc.snapsync",
		Usage:    "Enable XDC snapshot sync",
		Value:    false,
		Category: "XDC Sync",
	}
	XDCSnapShotBlockFlag = &cli.Uint64Flag{
		Name:     "xdc.snapshot.block",
		Usage:    "Snapshot block number for sync",
		Category: "XDC Sync",
	}
	XDCCheckpointIntervalFlag = &cli.Uint64Flag{
		Name:     "xdc.checkpoint.interval",
		Usage:    "Checkpoint interval in blocks",
		Value:    900,
		Category: "XDC Sync",
	}

	// Masternode flags
	MasternodeFlag = &cli.BoolFlag{
		Name:     "masternode",
		Usage:    "Run as a masternode",
		Value:    false,
		Category: "Masternode",
	}
	MasternodeKeyFlag = &cli.StringFlag{
		Name:     "masternode.key",
		Usage:    "Masternode private key for signing",
		Category: "Masternode",
	}
	MasternodeCoinbaseFlag = &cli.StringFlag{
		Name:     "masternode.coinbase",
		Usage:    "Masternode coinbase address",
		Category: "Masternode",
	}
)

// XDCFlags contains all XDC-specific flags
var XDCFlags = []cli.Flag{
	XDPoSRewardFlag,
	XDPoSSlashingFlag,
	XDPoSValidatorFlag,
	XDPoSGapFlag,
	XDPoSEpochFlag,
	XDCxEnableFlag,
	XDCxDataDirFlag,
	XDCxLendingEnableFlag,
	XDCxLendingDataDirFlag,
	XDCMainnetFlag,
	XDCTestnetFlag,
	XDCDevnetFlag,
	XDCSnapSyncFlag,
	XDCSnapShotBlockFlag,
	XDCCheckpointIntervalFlag,
	MasternodeFlag,
	MasternodeKeyFlag,
	MasternodeCoinbaseFlag,
}

// SetXDCConfig applies XDC-specific configuration
func SetXDCConfig(ctx *cli.Context, cfg *ethconfig.Config) {
	// XDPoS configuration
	if ctx.IsSet(XDPoSRewardFlag.Name) {
		cfg.XDPoSRewards = ctx.Bool(XDPoSRewardFlag.Name)
	}
	if ctx.IsSet(XDPoSSlashingFlag.Name) {
		cfg.XDPoSSlashing = ctx.Bool(XDPoSSlashingFlag.Name)
	}
	if ctx.IsSet(XDPoSGapFlag.Name) {
		cfg.XDPoSGap = ctx.Uint64(XDPoSGapFlag.Name)
	}
	if ctx.IsSet(XDPoSEpochFlag.Name) {
		cfg.XDPoSEpoch = ctx.Uint64(XDPoSEpochFlag.Name)
	}

	// XDCx configuration
	if ctx.IsSet(XDCxEnableFlag.Name) {
		cfg.XDCxEnabled = ctx.Bool(XDCxEnableFlag.Name)
	}

	// XDCxLending configuration
	if ctx.IsSet(XDCxLendingEnableFlag.Name) {
		cfg.XDCxLendingEnabled = ctx.Bool(XDCxLendingEnableFlag.Name)
	}

	// Sync configuration
	if ctx.IsSet(XDCSnapSyncFlag.Name) {
		cfg.XDCSnapSync = ctx.Bool(XDCSnapSyncFlag.Name)
	}
	if ctx.IsSet(XDCCheckpointIntervalFlag.Name) {
		cfg.XDCCheckpointInterval = ctx.Uint64(XDCCheckpointIntervalFlag.Name)
	}
}

// SetXDCNodeConfig applies XDC-specific node configuration
func SetXDCNodeConfig(ctx *cli.Context, cfg *node.Config) {
	// Set XDC-specific data directories
	if ctx.IsSet(XDCxDataDirFlag.Name) {
		// Configure XDCx data directory
	}
	if ctx.IsSet(XDCxLendingDataDirFlag.Name) {
		// Configure XDCxLending data directory
	}
}

// SetXDCNetworkConfig configures the network for XDC
func SetXDCNetworkConfig(ctx *cli.Context, cfg *ethconfig.Config) {
	if ctx.Bool(XDCMainnetFlag.Name) {
		// Configure for mainnet
		cfg.NetworkId = 50
	} else if ctx.Bool(XDCTestnetFlag.Name) {
		// Configure for Apothem testnet
		cfg.NetworkId = 51
	} else if ctx.Bool(XDCDevnetFlag.Name) {
		// Configure for devnet
		cfg.NetworkId = 551
	}
}

// MasternodeConfig holds masternode configuration
type MasternodeConfig struct {
	Enable   bool
	Key      string
	Coinbase string
}

// SetMasternodeConfig sets masternode configuration
func SetMasternodeConfig(ctx *cli.Context) *MasternodeConfig {
	config := &MasternodeConfig{
		Enable: ctx.Bool(MasternodeFlag.Name),
	}

	if config.Enable {
		if ctx.IsSet(MasternodeKeyFlag.Name) {
			config.Key = ctx.String(MasternodeKeyFlag.Name)
		}
		if ctx.IsSet(MasternodeCoinbaseFlag.Name) {
			config.Coinbase = ctx.String(MasternodeCoinbaseFlag.Name)
		}
	}

	return config
}

// ValidateXDCFlags validates XDC-specific flags
func ValidateXDCFlags(ctx *cli.Context) error {
	// Validate that mainnet, testnet, and devnet are mutually exclusive
	networks := 0
	if ctx.Bool(XDCMainnetFlag.Name) {
		networks++
	}
	if ctx.Bool(XDCTestnetFlag.Name) {
		networks++
	}
	if ctx.Bool(XDCDevnetFlag.Name) {
		networks++
	}
	if networks > 1 {
		return cli.Exit("Cannot specify multiple XDC networks", 1)
	}

	// Validate masternode configuration
	if ctx.Bool(MasternodeFlag.Name) {
		if !ctx.IsSet(MasternodeCoinbaseFlag.Name) {
			return cli.Exit("Masternode coinbase address is required", 1)
		}
	}

	return nil
}
