// Copyright 2023 The XDC Network Authors
// This file is part of the XDC Network library.
//
// The XDC Network library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.

package node

import (
	"crypto/ecdsa"
	"os"
	"path/filepath"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
)

// XDCConfig contains XDPoS-specific node configuration
type XDCConfig struct {
	// EnableMasternode enables masternode mode
	EnableMasternode bool

	// Masternode account address
	MasternodeAddr common.Address

	// Masternode signing key
	MasternodeKey *ecdsa.PrivateKey

	// StakingAddress is the address that holds masternode stake
	StakingAddress common.Address

	// RewardAddress is the address to receive rewards
	RewardAddress common.Address

	// Enable XDCx DEX
	EnableXDCx bool

	// XDCx database path
	XDCxDataDir string

	// Enable lending
	EnableLending bool

	// Lending database path
	LendingDataDir string

	// Announce masternode to network
	Announce bool

	// SlashingEnabled enables slashing for misbehavior
	SlashingEnabled bool

	// GapBlockNum is the gap block before epoch switch
	GapBlockNum uint64

	// LogLevel for XDPoS consensus
	LogLevel int
}

// DefaultXDCConfig returns default XDC configuration
func DefaultXDCConfig() *XDCConfig {
	return &XDCConfig{
		EnableMasternode: false,
		EnableXDCx:       false,
		EnableLending:    false,
		Announce:         true,
		SlashingEnabled:  true,
		GapBlockNum:      50,
		LogLevel:         3,
	}
}

// ResolveXDCxDataDir resolves the XDCx data directory
func (c *XDCConfig) ResolveXDCxDataDir(baseDir string) string {
	if c.XDCxDataDir != "" {
		return c.XDCxDataDir
	}
	return filepath.Join(baseDir, "XDCx")
}

// ResolveLendingDataDir resolves the lending data directory
func (c *XDCConfig) ResolveLendingDataDir(baseDir string) string {
	if c.LendingDataDir != "" {
		return c.LendingDataDir
	}
	return filepath.Join(baseDir, "lending")
}

// Validate validates the XDC configuration
func (c *XDCConfig) Validate() error {
	if c.EnableMasternode {
		if c.MasternodeAddr == (common.Address{}) {
			return ErrMasternodeAddressRequired
		}
		if c.MasternodeKey == nil {
			return ErrMasternodeKeyRequired
		}
	}
	return nil
}

// Error types
type ConfigError struct {
	message string
}

func (e *ConfigError) Error() string {
	return e.message
}

var (
	ErrMasternodeAddressRequired = &ConfigError{"masternode address required when masternode mode enabled"}
	ErrMasternodeKeyRequired     = &ConfigError{"masternode key required when masternode mode enabled"}
)

// XDCNodeConfig extends the base node config with XDC options
type XDCNodeConfig struct {
	// Embed base Config
	Config

	// XDC specific options
	XDC *XDCConfig
}

// SetXDCDefaults sets default values for XDC configuration
func (c *XDCNodeConfig) SetXDCDefaults() {
	if c.XDC == nil {
		c.XDC = DefaultXDCConfig()
	}
}

// EnsureXDCDirectories creates necessary XDC directories
func (c *XDCNodeConfig) EnsureXDCDirectories() error {
	if c.XDC == nil {
		return nil
	}

	baseDir := c.DataDir

	// Create XDCx directory if enabled
	if c.XDC.EnableXDCx {
		xdcxDir := c.XDC.ResolveXDCxDataDir(baseDir)
		if err := os.MkdirAll(xdcxDir, 0755); err != nil {
			return err
		}
		log.Info("XDCx data directory", "path", xdcxDir)
	}

	// Create lending directory if enabled
	if c.XDC.EnableLending {
		lendingDir := c.XDC.ResolveLendingDataDir(baseDir)
		if err := os.MkdirAll(lendingDir, 0755); err != nil {
			return err
		}
		log.Info("Lending data directory", "path", lendingDir)
	}

	return nil
}

// XDCStartupInfo logs XDC-specific startup information
func (c *XDCNodeConfig) XDCStartupInfo() {
	if c.XDC == nil {
		return
	}

	log.Info("XDC Network Configuration",
		"masternode", c.XDC.EnableMasternode,
		"xdcx", c.XDC.EnableXDCx,
		"lending", c.XDC.EnableLending,
	)

	if c.XDC.EnableMasternode {
		log.Info("Masternode Configuration",
			"address", c.XDC.MasternodeAddr.Hex(),
			"announce", c.XDC.Announce,
		)
	}
}

// ApplyXDCFlags applies XDC-specific command line flags
func ApplyXDCFlags(cfg *XDCConfig, ctx interface{}) {
	// This would be integrated with the command line flag parsing
	// to apply flags like --masternode, --xdcx, etc.
}

// XDCChainType represents the type of XDC chain
type XDCChainType string

const (
	XDCMainnet  XDCChainType = "mainnet"
	XDCApothem  XDCChainType = "apothem"  // Testnet
	XDCDevnet   XDCChainType = "devnet"
	XDCPrivate  XDCChainType = "private"
)

// GetXDCChainType returns the chain type based on network ID
func GetXDCChainType(networkID uint64) XDCChainType {
	switch networkID {
	case 50:
		return XDCMainnet
	case 51:
		return XDCApothem
	case 551:
		return XDCDevnet
	default:
		return XDCPrivate
	}
}

// XDCNetworkInfo returns network information for the chain type
func XDCNetworkInfo(chainType XDCChainType) (networkID uint64, chainID uint64) {
	switch chainType {
	case XDCMainnet:
		return 50, 50
	case XDCApothem:
		return 51, 51
	case XDCDevnet:
		return 551, 551
	default:
		return 0, 0
	}
}
