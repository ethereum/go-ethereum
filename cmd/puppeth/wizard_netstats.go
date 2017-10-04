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
	"fmt"
	"os"
	"strings"

	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/log"
	"github.com/olekukonko/tablewriter"
)

// networkStats verifies the status of network components and generates a protip
// configuration set to give users hints on how to do various tasks.
func (w *wizard) networkStats(tips bool) {
	if len(w.servers) == 0 {
		log.Error("No remote machines to gather stats from")
		return
	}
	protips := new(protips)

	// Iterate over all the specified hosts and check their status
	stats := tablewriter.NewWriter(os.Stdout)
	stats.SetHeader([]string{"Server", "IP", "Status", "Service", "Details"})
	stats.SetColWidth(100)

	for server, pubkey := range w.conf.Servers {
		client := w.servers[server]
		logger := log.New("server", server)
		logger.Info("Starting remote server health-check")

		// If the server is not connected, try to connect again
		if client == nil {
			conn, err := dial(server, pubkey)
			if err != nil {
				logger.Error("Failed to establish remote connection", "err", err)
				stats.Append([]string{server, "", err.Error(), "", ""})
				continue
			}
			client = conn
		}
		// Client connected one way or another, run health-checks
		services := make(map[string]string)
		logger.Debug("Checking for nginx availability")
		if infos, err := checkNginx(client, w.network); err != nil {
			if err != ErrServiceUnknown {
				services["nginx"] = err.Error()
			}
		} else {
			services["nginx"] = infos.String()
		}
		logger.Debug("Checking for ethstats availability")
		if infos, err := checkEthstats(client, w.network); err != nil {
			if err != ErrServiceUnknown {
				services["ethstats"] = err.Error()
			}
		} else {
			services["ethstats"] = infos.String()
			protips.ethstats = infos.config
		}
		logger.Debug("Checking for bootnode availability")
		if infos, err := checkNode(client, w.network, true); err != nil {
			if err != ErrServiceUnknown {
				services["bootnode"] = err.Error()
			}
		} else {
			services["bootnode"] = infos.String()

			protips.genesis = string(infos.genesis)
			protips.bootFull = append(protips.bootFull, infos.enodeFull)
			if infos.enodeLight != "" {
				protips.bootLight = append(protips.bootLight, infos.enodeLight)
			}
		}
		logger.Debug("Checking for sealnode availability")
		if infos, err := checkNode(client, w.network, false); err != nil {
			if err != ErrServiceUnknown {
				services["sealnode"] = err.Error()
			}
		} else {
			services["sealnode"] = infos.String()
			protips.genesis = string(infos.genesis)
		}
		logger.Debug("Checking for faucet availability")
		if infos, err := checkFaucet(client, w.network); err != nil {
			if err != ErrServiceUnknown {
				services["faucet"] = err.Error()
			}
		} else {
			services["faucet"] = infos.String()
		}
		logger.Debug("Checking for dashboard availability")
		if infos, err := checkDashboard(client, w.network); err != nil {
			if err != ErrServiceUnknown {
				services["dashboard"] = err.Error()
			}
		} else {
			services["dashboard"] = infos.String()
		}
		// All status checks complete, report and check next server
		delete(w.services, server)
		for service := range services {
			w.services[server] = append(w.services[server], service)
		}
		server, address := client.server, client.address
		for service, status := range services {
			stats.Append([]string{server, address, "online", service, status})
			server, address = "", ""
		}
		if len(services) == 0 {
			stats.Append([]string{server, address, "online", "", ""})
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
	if !tips {
		stats.Render()
	} else {
		protips.print(w.network)
	}
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

// print analyzes the network information available and prints a collection of
// pro tips for the user's consideration.
func (p *protips) print(network string) {
	// If a known genesis block is available, display it and prepend an init command
	fullinit, lightinit := "", ""
	if p.genesis != "" {
		fullinit = fmt.Sprintf("geth --datadir=$HOME/.%s init %s.json && ", network, network)
		lightinit = fmt.Sprintf("geth --datadir=$HOME/.%s --light init %s.json && ", network, network)
	}
	// If an ethstats server is available, add the ethstats flag
	statsflag := ""
	if p.ethstats != "" {
		if strings.Contains(p.ethstats, " ") {
			statsflag = fmt.Sprintf(` --ethstats="yournode:%s"`, p.ethstats)
		} else {
			statsflag = fmt.Sprintf(` --ethstats=yournode:%s`, p.ethstats)
		}
	}
	// If bootnodes have been specified, add the bootnode flag
	bootflagFull := ""
	if len(p.bootFull) > 0 {
		bootflagFull = fmt.Sprintf(` --bootnodes %s`, strings.Join(p.bootFull, ","))
	}
	bootflagLight := ""
	if len(p.bootLight) > 0 {
		bootflagLight = fmt.Sprintf(` --bootnodes %s`, strings.Join(p.bootLight, ","))
	}
	// Assemble all the known pro-tips
	var tasks, tips []string

	tasks = append(tasks, "Run an archive node with historical data")
	tips = append(tips, fmt.Sprintf("%sgeth --networkid=%d --datadir=$HOME/.%s --cache=1024%s%s", fullinit, p.network, network, statsflag, bootflagFull))

	tasks = append(tasks, "Run a full node with recent data only")
	tips = append(tips, fmt.Sprintf("%sgeth --networkid=%d --datadir=$HOME/.%s --cache=512 --fast%s%s", fullinit, p.network, network, statsflag, bootflagFull))

	tasks = append(tasks, "Run a light node with on demand retrievals")
	tips = append(tips, fmt.Sprintf("%sgeth --networkid=%d --datadir=$HOME/.%s --light%s%s", lightinit, p.network, network, statsflag, bootflagLight))

	tasks = append(tasks, "Run an embedded node with constrained memory")
	tips = append(tips, fmt.Sprintf("%sgeth --networkid=%d --datadir=$HOME/.%s --cache=32 --light%s%s", lightinit, p.network, network, statsflag, bootflagLight))

	// If the tips are short, display in a table
	short := true
	for _, tip := range tips {
		if len(tip) > 100 {
			short = false
			break
		}
	}
	fmt.Println()
	if short {
		howto := tablewriter.NewWriter(os.Stdout)
		howto.SetHeader([]string{"Fun tasks for you", "Tips on how to"})
		howto.SetColWidth(100)

		for i := 0; i < len(tasks); i++ {
			howto.Append([]string{tasks[i], tips[i]})
		}
		howto.Render()
		return
	}
	// Meh, tips got ugly, split into many lines
	for i := 0; i < len(tasks); i++ {
		fmt.Println(tasks[i])
		fmt.Println(strings.Repeat("-", len(tasks[i])))
		fmt.Println(tips[i])
		fmt.Println()
		fmt.Println()
	}
}
