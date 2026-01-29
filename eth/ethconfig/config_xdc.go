// Copyright 2021 XDC Network
// This file is part of the XDC library.

package ethconfig

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
)

// XDCConfig holds XDC-specific configuration
type XDCConfig struct {
	// XDPoS configuration
	XDPoSRewards           bool
	XDPoSSlashing          bool
	XDPoSGap               uint64
	XDPoSEpoch             uint64
	XDPoSRewardFraction    int
	XDPoSFoundationWallet  common.Address

	// XDCx configuration
	XDCxEnabled            bool
	XDCxDataDir            string
	XDCxReplicasEnabled    bool

	// XDCxLending configuration
	XDCxLendingEnabled     bool
	XDCxLendingDataDir     string

	// Sync configuration
	XDCSnapSync            bool
	XDCSnapShotBlock       uint64
	XDCCheckpointInterval  uint64

	// Masternode configuration
	MasternodeEnabled      bool
	MasternodeKey          string
	MasternodeCoinbase     common.Address
}

// DefaultXDCConfig returns the default XDC configuration
func DefaultXDCConfig() *XDCConfig {
	return &XDCConfig{
		XDPoSRewards:          true,
		XDPoSSlashing:         true,
		XDPoSGap:              450,
		XDPoSEpoch:            900,
		XDPoSRewardFraction:   88,
		XDPoSFoundationWallet: common.HexToAddress("0x0000000000000000000000000000000000000068"),
		XDCxEnabled:           false,
		XDCxDataDir:           "",
		XDCxReplicasEnabled:   false,
		XDCxLendingEnabled:    false,
		XDCxLendingDataDir:    "",
		XDCSnapSync:           false,
		XDCSnapShotBlock:      0,
		XDCCheckpointInterval: 900,
		MasternodeEnabled:     false,
	}
}

// XDCRewardConfig holds reward configuration
type XDCRewardConfig struct {
	// Block reward in wei
	BlockReward *big.Int

	// Foundation reward percentage (out of 100)
	FoundationPercent int

	// Masternode reward percentage (out of 100)
	MasternodePercent int

	// Voter reward percentage (out of 100)
	VoterPercent int
}

// DefaultRewardConfig returns the default reward configuration
func DefaultRewardConfig() *XDCRewardConfig {
	return &XDCRewardConfig{
		BlockReward:       big.NewInt(0), // XDC doesn't have traditional block rewards
		FoundationPercent: 12,
		MasternodePercent: 60,
		VoterPercent:      28,
	}
}

// XDCSyncConfig holds sync-specific configuration
type XDCSyncConfig struct {
	// Snapshot URL for initial sync
	SnapshotURL string

	// Checkpoint block numbers and hashes
	Checkpoints map[uint64]common.Hash

	// Whether to verify checkpoints
	VerifyCheckpoints bool

	// Fast sync pivot point
	PivotBlock uint64
}

// DefaultSyncConfig returns the default sync configuration
func DefaultSyncConfig() *XDCSyncConfig {
	return &XDCSyncConfig{
		SnapshotURL:       "",
		Checkpoints:       make(map[uint64]common.Hash),
		VerifyCheckpoints: true,
		PivotBlock:        0,
	}
}

// XDCNetworkConfig holds network-specific configuration
type XDCNetworkConfig struct {
	// Network ID
	NetworkID uint64

	// Network name
	NetworkName string

	// Chain ID
	ChainID *big.Int

	// Genesis hash
	GenesisHash common.Hash

	// Bootnodes
	Bootnodes []string

	// Static nodes
	StaticNodes []string
}

// MainnetConfig returns mainnet configuration
func MainnetConfig() *XDCNetworkConfig {
	return &XDCNetworkConfig{
		NetworkID:   50,
		NetworkName: "xdc-mainnet",
		ChainID:     big.NewInt(50),
	}
}

// TestnetConfig returns Apothem testnet configuration
func TestnetConfig() *XDCNetworkConfig {
	return &XDCNetworkConfig{
		NetworkID:   51,
		NetworkName: "xdc-apothem",
		ChainID:     big.NewInt(51),
	}
}

// DevnetConfig returns devnet configuration
func DevnetConfig() *XDCNetworkConfig {
	return &XDCNetworkConfig{
		NetworkID:   551,
		NetworkName: "xdc-devnet",
		ChainID:     big.NewInt(551),
	}
}

// ValidateXDCConfig validates XDC configuration
func ValidateXDCConfig(cfg *XDCConfig) error {
	if cfg.XDPoSEpoch == 0 {
		return &configError{"XDPoS epoch cannot be zero"}
	}

	if cfg.XDPoSGap == 0 {
		return &configError{"XDPoS gap cannot be zero"}
	}

	if cfg.XDPoSGap >= cfg.XDPoSEpoch {
		return &configError{"XDPoS gap must be less than epoch"}
	}

	if cfg.MasternodeEnabled && cfg.MasternodeCoinbase == (common.Address{}) {
		return &configError{"Masternode coinbase required when masternode is enabled"}
	}

	return nil
}

// configError represents a configuration error
type configError struct {
	message string
}

func (e *configError) Error() string {
	return e.message
}

// IsXDCNetworkID checks if the network ID is an XDC network
func IsXDCNetworkID(networkID uint64) bool {
	return networkID == 50 || networkID == 51 || networkID == 551
}

// GetXDCNetworkName returns the network name for a network ID
func GetXDCNetworkName(networkID uint64) string {
	switch networkID {
	case 50:
		return "mainnet"
	case 51:
		return "apothem"
	case 551:
		return "devnet"
	default:
		return "unknown"
	}
}
