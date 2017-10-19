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
	"encoding/json"
	"os"
	"sort"
	"strings"

	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/log"
	"github.com/olekukonko/tablewriter"
)

// networkStats verifies the status of network components and generates a protip
// configuration set to give users hints on how to do various tasks.
func (w *wizard) networkStats() {
	if len(w.servers) == 0 {
		log.Error("No remote machines to gather stats from")
		return
	}
	protips := new(protips)

	// Iterate over all the specified hosts and check their status
	stats := make(serverStats)

	for server, pubkey := range w.conf.Servers {
		client := w.servers[server]
		logger := log.New("server", server)
		logger.Info("Starting remote server health-check")

		stat := &serverStat{
			address:  client.address,
			services: make(map[string]map[string]string),
		}
		stats[client.server] = stat

		if client == nil {
			conn, err := dial(server, pubkey)
			if err != nil {
				logger.Error("Failed to establish remote connection", "err", err)
				stat.failure = err.Error()
				continue
			}
			client = conn
		}
		// Client connected one way or another, run health-checks
		logger.Debug("Checking for nginx availability")
		if infos, err := checkNginx(client, w.network); err != nil {
			if err != ErrServiceUnknown {
				stat.services["nginx"] = map[string]string{"offline": err.Error()}
			}
		} else {
			stat.services["nginx"] = infos.Report()
		}
		logger.Debug("Checking for ethstats availability")
		if infos, err := checkEthstats(client, w.network); err != nil {
			if err != ErrServiceUnknown {
				stat.services["ethstats"] = map[string]string{"offline": err.Error()}
			}
		} else {
			stat.services["ethstats"] = infos.Report()
			protips.ethstats = infos.config
		}
		logger.Debug("Checking for bootnode availability")
		if infos, err := checkNode(client, w.network, true); err != nil {
			if err != ErrServiceUnknown {
				stat.services["bootnode"] = map[string]string{"offline": err.Error()}
			}
		} else {
			stat.services["bootnode"] = infos.Report()

			protips.genesis = string(infos.genesis)
			protips.bootFull = append(protips.bootFull, infos.enodeFull)
			if infos.enodeLight != "" {
				protips.bootLight = append(protips.bootLight, infos.enodeLight)
			}
		}
		logger.Debug("Checking for sealnode availability")
		if infos, err := checkNode(client, w.network, false); err != nil {
			if err != ErrServiceUnknown {
				stat.services["sealnode"] = map[string]string{"offline": err.Error()}
			}
		} else {
			stat.services["sealnode"] = infos.Report()
			protips.genesis = string(infos.genesis)
		}
		logger.Debug("Checking for faucet availability")
		if infos, err := checkFaucet(client, w.network); err != nil {
			if err != ErrServiceUnknown {
				stat.services["faucet"] = map[string]string{"offline": err.Error()}
			}
		} else {
			stat.services["faucet"] = infos.Report()
		}
		logger.Debug("Checking for dashboard availability")
		if infos, err := checkDashboard(client, w.network); err != nil {
			if err != ErrServiceUnknown {
				stat.services["dashboard"] = map[string]string{"offline": err.Error()}
			}
		} else {
			stat.services["dashboard"] = infos.Report()
		}
		// All status checks complete, report and check next server
		delete(w.services, server)
		for service := range stat.services {
			w.services[server] = append(w.services[server], service)
		}
	}
	// If a genesis block was found, load it into our configs
	if protips.genesis != "" && w.conf.genesis == nil {
		genesis := new(core.Genesis)
		if err := json.Unmarshal([]byte(protips.genesis), genesis); err != nil {
			log.Error("Failed to parse remote genesis", "err", err)
		} else {
			w.conf.genesis = genesis
			protips.network = genesis.Config.ChainId.Int64()
		}
	}
	if protips.ethstats != "" {
		w.conf.ethstats = protips.ethstats
	}
	w.conf.bootFull = protips.bootFull
	w.conf.bootLight = protips.bootLight

	// Print any collected stats and return
	stats.render()
}

// serverStat is a collection of service configuration parameters and health
// check reports to print to the user.
type serverStat struct {
	address  string
	failure  string
	services map[string]map[string]string
}

// serverStats is a collection of server stats for multiple hosts.
type serverStats map[string]*serverStat

// render converts the gathered statistics into a user friendly tabular report
// and prints it to the standard output.
func (stats serverStats) render() {
	// Start gathering service statistics and config parameters
	table := tablewriter.NewWriter(os.Stdout)

	table.SetHeader([]string{"Server", "Address", "Service", "Config", "Value"})
	table.SetAlignment(tablewriter.ALIGN_LEFT)
	table.SetColWidth(100)

	// Find the longest lines for all columns for the hacked separator
	separator := make([]string, 5)
	for server, stat := range stats {
		if len(server) > len(separator[0]) {
			separator[0] = strings.Repeat("-", len(server))
		}
		if len(stat.address) > len(separator[1]) {
			separator[1] = strings.Repeat("-", len(stat.address))
		}
		for service, configs := range stat.services {
			if len(service) > len(separator[2]) {
				separator[2] = strings.Repeat("-", len(service))
			}
			for config, value := range configs {
				if len(config) > len(separator[3]) {
					separator[3] = strings.Repeat("-", len(config))
				}
				if len(value) > len(separator[4]) {
					separator[4] = strings.Repeat("-", len(value))
				}
			}
		}
	}
	// Fill up the server report in alphabetical order
	servers := make([]string, 0, len(stats))
	for server := range stats {
		servers = append(servers, server)
	}
	sort.Strings(servers)

	for i, server := range servers {
		// Add a separator between all servers
		if i > 0 {
			table.Append(separator)
		}
		// Fill up the service report in alphabetical order
		services := make([]string, 0, len(stats[server].services))
		for service := range stats[server].services {
			services = append(services, service)
		}
		sort.Strings(services)

		for j, service := range services {
			// Add an empty line between all services
			if j > 0 {
				table.Append([]string{"", "", "", separator[3], separator[4]})
			}
			// Fill up the config report in alphabetical order
			configs := make([]string, 0, len(stats[server].services[service]))
			for service := range stats[server].services[service] {
				configs = append(configs, service)
			}
			sort.Strings(configs)

			for k, config := range configs {
				switch {
				case j == 0 && k == 0:
					table.Append([]string{server, stats[server].address, service, config, stats[server].services[service][config]})
				case k == 0:
					table.Append([]string{"", "", service, config, stats[server].services[service][config]})
				default:
					table.Append([]string{"", "", "", config, stats[server].services[service][config]})
				}
			}
		}
	}
	table.Render()
}

// protips contains a collection of network infos to report pro-tips
// based on.
type protips struct {
	genesis   string
	network   int64
	bootFull  []string
	bootLight []string
	ethstats  string
}
