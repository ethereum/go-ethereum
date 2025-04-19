// Copyright 2023 The go-ethereum Authors
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
	"os/signal"
	"sort"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/cmd/utils"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/urfave/cli/v2"
)

var (
	netstatCommand = &cli.Command{
		Action:    netstat,
		Name:      "netstat",
		Usage:     "Network status monitor and diagnostics",
		ArgsUsage: " ",
		Flags: []cli.Flag{
			utils.HTTPEnabledFlag,
			utils.HTTPListenAddrFlag,
			utils.HTTPPortFlag,
			utils.HTTPApiFlag,
		},
		Category: "MONITOR COMMANDS",
		Description: `
The netstat command displays detailed information about the network status
of the running Ethereum node. It shows connected peers, their network addresses,
supported protocols, and other relevant information for network diagnostics.

This tool is useful for:
- Verifying connectivity to the Ethereum network
- Debugging peer connection issues
- Monitoring active network protocols
- Checking network quality and health
`,
	}

	// Flag for live monitoring mode
	monitorFlag = &cli.BoolFlag{
		Name:  "monitor",
		Usage: "Enable continuous monitoring of network status",
		Value: false,
	}
)

// netstat displays detailed network information about the Ethereum node.
func netstat(ctx *cli.Context) error {
	// Attach to the node specified by the user
	node := makeFullNode(ctx)
	defer node.Close()

	// Attach to a running geth instance
	client := node.Attach()
	if client == nil {
		// If we can't attach to a node, try to start one
		utils.StartNode(ctx, node, false)

		// Try to attach again after starting
		client = node.Attach()
		if client == nil {
			return fmt.Errorf("failed to attach to the Ethereum client")
		}
	}
	defer client.Close()

	// Get node information
	var nodeInfo p2p.NodeInfo
	if err := client.Call(&nodeInfo, "admin_nodeInfo"); err != nil {
		return fmt.Errorf("failed to retrieve node info: %v", err)
	}

	// Get peer information
	var peers []*p2p.PeerInfo
	if err := client.Call(&peers, "admin_peers"); err != nil {
		return fmt.Errorf("failed to retrieve peer info: %v", err)
	}

	// Display node information
	fmt.Println()
	fmt.Println("==== NODE INFORMATION ====")
	fmt.Println()
	fmt.Printf("Node ID:       %s\n", nodeInfo.ID)
	fmt.Printf("Name:          %s\n", nodeInfo.Name)
	fmt.Printf("Enode URL:     %s\n", nodeInfo.Enode)
	fmt.Printf("IP Address:    %s\n", nodeInfo.IP)
	fmt.Printf("Listen Addr:   %s\n", nodeInfo.ListenAddr)
	fmt.Printf("Discovery:     %d\n", nodeInfo.Ports.Discovery)
	fmt.Printf("Listener:      %d\n", nodeInfo.Ports.Listener)

	// Display protocols
	fmt.Println()
	fmt.Println("Supported Protocols:")
	for name, proto := range nodeInfo.Protocols {
		jsonBytes, _ := json.MarshalIndent(proto, "    ", "  ")
		fmt.Printf("  %s: %s\n", name, string(jsonBytes))
	}

	// Display peer information
	fmt.Println()
	fmt.Println("==== CONNECTED PEERS ====")
	fmt.Println()

	// Format and sort peers
	if len(peers) == 0 {
		fmt.Println("No peers connected")
	} else {
		// Sort peers by connection type (inbound/outbound) and then by name
		sort.Slice(peers, func(i, j int) bool {
			if peers[i].Network.Inbound != peers[j].Network.Inbound {
				return !peers[i].Network.Inbound // Outbound first
			}
			return peers[i].Name < peers[j].Name
		})

		// Print peer information
		for i, peer := range peers {
			fmt.Printf("Peer #%d:\n", i+1)
			fmt.Printf("  ID:         %s\n", peer.ID)
			fmt.Printf("  Name:       %s\n", peer.Name)
			fmt.Printf("  Direction:  %s\n", connectionDirection(peer.Network.Inbound))
			fmt.Printf("  Remote:     %s\n", peer.Network.RemoteAddress)
			fmt.Printf("  Local:      %s\n", peer.Network.LocalAddress)
			if peer.Network.Static {
				fmt.Printf("  Type:       Static\n")
			} else if peer.Network.Trusted {
				fmt.Printf("  Type:       Trusted\n")
			} else {
				fmt.Printf("  Type:       Dynamic\n")
			}

			// Display capabilities
			fmt.Printf("  Caps:       %s\n", strings.Join(peer.Caps, ", "))

			// Show protocol-specific details
			fmt.Println("  Protocols:")
			for proto, details := range peer.Protocols {
				jsonBytes, _ := json.MarshalIndent(details, "    ", "  ")
				fmt.Printf("    %s: %s\n", proto, string(jsonBytes))
			}
			fmt.Println()
		}
	}

	// Live monitoring mode if the flag is set
	if ctx.Bool(monitorFlag.Name) {
		return monitorNetwork(client)
	}

	return nil
}

// monitorNetwork continuously displays network information with periodic updates
func monitorNetwork(client *rpc.Client) error {
	fmt.Println("Entering network monitoring mode. Press Ctrl+C to exit.")

	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	// Setup to capture Ctrl+C
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)

	for {
		select {
		case <-ticker.C:
			// Clear the screen and refresh data
			fmt.Print("\033[H\033[2J") // ANSI escape sequence to clear the screen

			// Get peer information
			var peers []*p2p.PeerInfo
			if err := client.Call(&peers, "admin_peers"); err != nil {
				fmt.Printf("Error retrieving peer info: %v\n", err)
				continue
			}

			// Display peer count
			fmt.Println()
			fmt.Printf("==== NETWORK STATUS ==== (Updated: %s)\n", time.Now().Format("15:04:05"))
			fmt.Printf("Total Peers: %d\n\n", len(peers))

			// Display peer summary
			var inbound, outbound, trusted, static int
			for _, p := range peers {
				if p.Network.Inbound {
					inbound++
				} else {
					outbound++
				}
				if p.Network.Trusted {
					trusted++
				}
				if p.Network.Static {
					static++
				}
			}

			fmt.Printf("Inbound:  %d\n", inbound)
			fmt.Printf("Outbound: %d\n", outbound)
			fmt.Printf("Trusted:  %d\n", trusted)
			fmt.Printf("Static:   %d\n", static)

			// Display peer list
			if len(peers) > 0 {
				fmt.Println("\nPeer List:")
				fmt.Printf("%-4s %-8s %-42s %-30s %-20s\n",
					"", "Type", "Node ID", "Name", "Address")
				fmt.Println(strings.Repeat("-", 110))

				// Sort peers as before
				sort.Slice(peers, func(i, j int) bool {
					if peers[i].Network.Inbound != peers[j].Network.Inbound {
						return !peers[i].Network.Inbound
					}
					return peers[i].Name < peers[j].Name
				})

				for i, p := range peers {
					id := p.ID
					if len(id) > 10 {
						id = id[:8] + "..." + id[len(id)-8:]
					}
					name := p.Name
					if len(name) > 30 {
						name = name[:27] + "..."
					}

					peerType := "Out"
					if p.Network.Inbound {
						peerType = "In"
					}
					if p.Network.Trusted {
						peerType += "+T"
					}
					if p.Network.Static {
						peerType += "+S"
					}

					fmt.Printf("%-4d %-8s %-42s %-30s %-20s\n",
						i+1, peerType, id, name, p.Network.RemoteAddress)
				}
			}

		case <-interrupt:
			return nil
		}
	}
}

// connectionDirection returns a human-readable string describing the connection direction
func connectionDirection(inbound bool) string {
	if inbound {
		return "Inbound (remote dialed us)"
	}
	return "Outbound (we dialed remote)"
}
