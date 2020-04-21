// Copyright 2020 The go-ethereum Authors
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

package utils

import (
	"fmt"
	"github.com/ethereum/go-ethereum/eth"
	"github.com/ethereum/go-ethereum/node"
	"gopkg.in/urfave/cli.v1"
	"strings"
)

var (
	ShowDeprecated = cli.Command{
		Action: showDeprecated,
		Name: "show-deprecated-flags",
		Usage: "Show flags that have been deprecated",
		Flags: []cli.Flag{
			LegacyTestnetFlag,
			LightLegacyServFlag,
			LightLegacyPeersFlag,
			MinerLegacyThreadsFlag,
			MinerLegacyGasTargetFlag,
			MinerLegacyGasPriceFlag,
			MinerLegacyEtherbaseFlag,
			MinerLegacyExtraDataFlag,
		},
		Description: "Show flags that have been deprecated and will soon be removed",
	}
	LegacyTestnetFlag = cli.BoolFlag{ // TODO(q9f): Remove after Ropsten is discontinued.
		Name:  "testnet",
		Usage: "Pre-configured test network (Deprecated: Please choose one of --goerli, --rinkeby, or --ropsten.)",
	}
	LightLegacyServFlag = cli.IntFlag{ // Deprecated in favor of light.serve, remove in 2021
		Name:  "lightserv",
		Usage: "Maximum percentage of time allowed for serving LES requests (deprecated, use --light.serve)",
		Value: eth.DefaultConfig.LightServ,
	}
	LightLegacyPeersFlag = cli.IntFlag{ // Deprecated in favor of light.maxpeers, remove in 2021
		Name:  "lightpeers",
		Usage: "Maximum number of light clients to serve, or light servers to attach to  (deprecated, use --light.maxpeers)",
		Value: eth.DefaultConfig.LightPeers,
	}
	MinerLegacyThreadsFlag = cli.IntFlag{
		Name:  "minerthreads",
		Usage: "Number of CPU threads to use for mining (deprecated, use --miner.threads)",
		Value: 0,
	}
	MinerLegacyGasTargetFlag = cli.Uint64Flag{
		Name:  "targetgaslimit",
		Usage: "Target gas floor for mined blocks (deprecated, use --miner.gastarget)",
		Value: eth.DefaultConfig.Miner.GasFloor,
	}
	MinerLegacyGasPriceFlag = BigFlag{
		Name:  "gasprice",
		Usage: "Minimum gas price for mining a transaction (deprecated, use --miner.gasprice)",
		Value: eth.DefaultConfig.Miner.GasPrice,
	}
	MinerLegacyEtherbaseFlag = cli.StringFlag{
		Name:  "etherbase",
		Usage: "Public address for block mining rewards (default = first account, deprecated, use --miner.etherbase)",
		Value: "0",
	}
	MinerLegacyExtraDataFlag = cli.StringFlag{
		Name:  "extradata",
		Usage: "Block extra data set by the miner (default = client version, deprecated, use --miner.extradata)",
	}
	RPCLegacyEnabledFlag = cli.BoolFlag{
		Name:  "rpc",
		Usage: "Enable the HTTP-RPC server (deprecated, use --http)",
	}
	RPCLegacyListenAddrFlag = cli.StringFlag{
		Name:  "rpcaddr",
		Usage: "HTTP-RPC server listening interface (deprecated, use --http.addr)",
		Value: node.DefaultHTTPHost,
	}
	RPCLegacyPortFlag = cli.IntFlag{
		Name:  "rpcport",
		Usage: "HTTP-RPC server listening port (deprecated, use --http.port)",
		Value: node.DefaultHTTPPort,
	}
	RPCLegacyCORSDomainFlag = cli.StringFlag{
		Name:  "rpccorsdomain",
		Usage: "Comma separated list of domains from which to accept cross origin requests (browser enforced) (deprecated, use --http.corsdomain)",
		Value: "",
	}
	RPCLegacyVirtualHostsFlag = cli.StringFlag{
		Name:  "rpcvhosts",
		Usage: "Comma separated list of virtual hostnames from which to accept requests (server enforced). Accepts '*' wildcard. (deprecated, use --http.vhosts)",
		Value: strings.Join(node.DefaultConfig.HTTPVirtualHosts, ","),
	}
	RPCLegacyApiFlag = cli.StringFlag{
		Name:  "rpcapi",
		Usage: "API's offered over the HTTP-RPC interface (deprecated, use --http.api)",
		Value: "",
	}
	WSLegacyListenAddrFlag = cli.StringFlag{
		Name:  "wsaddr",
		Usage: "WS-RPC server listening interface (deprecated, use --ws.addr)",
		Value: node.DefaultWSHost,
	}
	WSLegacyPortFlag = cli.IntFlag{
		Name:  "wsport",
		Usage: "WS-RPC server listening port (deprecated, use --ws.port)",
		Value: node.DefaultWSPort,
	}
	WSLegacyApiFlag = cli.StringFlag{
		Name:  "wsapi",
		Usage: "API's offered over the WS-RPC interface (deprecated, use --ws.api)",
		Value: "",
	}
	WSLegacyAllowedOriginsFlag = cli.StringFlag{
		Name:  "wsorigins",
		Usage: "Origins from which to accept websockets requests (deprecated, use --ws.origins)",
		Value: "",
	}
	GpoLegacyBlocksFlag = cli.IntFlag{
		Name:  "gpoblocks",
		Usage: "Number of recent blocks to check for gas prices (deprecated, use --gpo.blocks)",
		Value: eth.DefaultConfig.GPO.Blocks,
	}
	GpoLegacyPercentileFlag = cli.IntFlag{
		Name:  "gpopercentile",
		Usage: "Suggested gas price is the given percentile of a set of recent transaction gas prices (deprecated, use --gpo.percentile)",
		Value: eth.DefaultConfig.GPO.Percentile,
	}
)

func showDeprecated(c *cli.Context) {
	fmt.Println("--------------------------------------------------------------------")
	fmt.Println("The following flags are deprecated and will be removed in the future!")
	fmt.Println("--------------------------------------------------------------------")
	fmt.Println()

	for _, flag := range c.Command.Flags {
		fmt.Println(flag.String())
	}
}
