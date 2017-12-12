// Copyright 2015 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

// Package release contains the node service that tracks client releases.
package release

//go:generate abigen --sol ./contract.sol --pkg release --out ./contract.go

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/eth"
	"github.com/ethereum/go-ethereum/internal/ethapi"
	"github.com/ethereum/go-ethereum/les"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/rpc"
)

// Interval to check for new releases
const releaseRecheckInterval = time.Hour

// Config contains the configurations of the release service.
type Config struct {
	Oracle common.Address // Ethereum address of the release oracle
	Major  uint32         // Major version component of the release
	Minor  uint32         // Minor version component of the release
	Patch  uint32         // Patch version component of the release
	Commit [20]byte       // Git SHA1 commit hash of the release
}

// ReleaseService is a node service that periodically checks the blockchain for
// newly released versions of the client being run and issues a warning to the
// user about it.
type ReleaseService struct {
	config Config          // Current version to check releases against
	oracle *ReleaseOracle  // Native binding to the release oracle contract
	quit   chan chan error // Quit channel to terminate the version checker
}

// NewReleaseService creates a new service to periodically check for new client
// releases and notify the user of such.
func NewReleaseService(ctx *node.ServiceContext, config Config) (node.Service, error) {
	// Retrieve the Ethereum service dependency to access the blockchain
	var apiBackend ethapi.Backend
	var ethereum *eth.Ethereum
	if err := ctx.Service(&ethereum); err == nil {
		apiBackend = ethereum.ApiBackend
	} else {
		var ethereum *les.LightEthereum
		if err := ctx.Service(&ethereum); err == nil {
			apiBackend = ethereum.ApiBackend
		} else {
			return nil, err
		}
	}
	// Construct the release service
	contract, err := NewReleaseOracle(config.Oracle, eth.NewContractBackend(apiBackend))
	if err != nil {
		return nil, err
	}
	return &ReleaseService{
		config: config,
		oracle: contract,
		quit:   make(chan chan error),
	}, nil
}

// Protocols returns an empty list of P2P protocols as the release service does
// not have a networking component.
func (r *ReleaseService) Protocols() []p2p.Protocol { return nil }

// APIs returns an empty list of RPC descriptors as the release service does not
// expose any functioanlity to the outside world.
func (r *ReleaseService) APIs() []rpc.API { return nil }

// Start spawns the periodic version checker goroutine
func (r *ReleaseService) Start(server *p2p.Server) error {
	go r.checker()
	return nil
}

// Stop terminates all goroutines belonging to the service, blocking until they
// are all terminated.
func (r *ReleaseService) Stop() error {
	errc := make(chan error)
	r.quit <- errc
	return <-errc
}

// checker runs indefinitely in the background, periodically checking for new
// client releases.
func (r *ReleaseService) checker() {
	// Set up the timers to periodically check for releases
	timer := time.NewTimer(0) // Immediately fire a version check
	defer timer.Stop()

	for {
		select {
		case <-timer.C:
			// Rechedule the timer before continuing
			timer.Reset(releaseRecheckInterval)
			r.checkVersion()
		case errc := <-r.quit:
			errc <- nil
			return
		}
	}
}

func (r *ReleaseService) checkVersion() {
	// Retrieve the current version, and handle missing contracts gracefully
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	opts := &bind.CallOpts{Context: ctx}
	defer cancel()

	version, err := r.oracle.CurrentVersion(opts)
	if err != nil {
		if err == bind.ErrNoCode {
			log.Debug("Release oracle not found", "contract", r.config.Oracle)
		} else {
			log.Error("Failed to retrieve current release", "err", err)
		}
		return
	}
	// Version was successfully retrieved, notify if newer than ours
	if version.Major > r.config.Major ||
		(version.Major == r.config.Major && version.Minor > r.config.Minor) ||
		(version.Major == r.config.Major && version.Minor == r.config.Minor && version.Patch > r.config.Patch) {

		warning := fmt.Sprintf("Client v%d.%d.%d-%x seems older than the latest upstream release v%d.%d.%d-%x",
			r.config.Major, r.config.Minor, r.config.Patch, r.config.Commit[:4], version.Major, version.Minor, version.Patch, version.Commit[:4])
		howtofix := fmt.Sprintf("Please check https://github.com/ethereum/go-ethereum/releases for new releases")
		separator := strings.Repeat("-", len(warning))

		log.Warn(separator)
		log.Warn(warning)
		log.Warn(howtofix)
		log.Warn(separator)
	} else {
		log.Debug("Client seems up to date with upstream",
			"local", fmt.Sprintf("v%d.%d.%d-%x", r.config.Major, r.config.Minor, r.config.Patch, r.config.Commit[:4]),
			"upstream", fmt.Sprintf("v%d.%d.%d-%x", version.Major, version.Minor, version.Patch, version.Commit[:4]))
	}
}
