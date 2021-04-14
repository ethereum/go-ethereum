// Copyright 2021 The go-ethereum Authors
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
	"context"
	"crypto/ecdsa"
	"fmt"
	"math/big"
	"net"
	"time"

	"github.com/ethereum/go-ethereum/cmd/devp2p/internal/ethtest"
	"github.com/ethereum/go-ethereum/cmd/utils"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/forkid"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/p2p/enode"
	"github.com/ethereum/go-ethereum/p2p/rlpx"
	"github.com/ethereum/go-ethereum/params"
	"gopkg.in/urfave/cli.v1"
)

var (
	crawlerCommand = cli.Command{
		Name:      "crawl",
		Usage:     "Crawl the ethereum network",
		ArgsUsage: "<nodefile>",
		Action:    crawlNodes,
		Flags: []cli.Flag{
			bootnodesFlag,
			crawlTimeoutFlag,
			utils.MetricsEnableInfluxDBFlag,
			utils.MetricsInfluxDBEndpointFlag,
			utils.MetricsInfluxDBDatabaseFlag,
			utils.MetricsInfluxDBUsernameFlag,
			utils.MetricsInfluxDBPasswordFlag,
		},
	}
)

type crawledNode struct {
	node nodeJSON
	info *clientInfo
}

type clientInfo struct {
	ClientType      string
	SoftwareVersion uint64
	Capabilities    []p2p.Cap
	NetworkID       uint64
	ForkID          forkid.ID
	Blockheight     string
	TotalDifficulty *big.Int
	HeadHash        common.Hash
}

func crawlNodes(ctx *cli.Context) error {
	var inputSet nodeSet
	if ctx.NArg() < 1 {
		return fmt.Errorf("need nodes file as argument")
	}
	nodesFile := ctx.Args().First()
	if common.FileExist(nodesFile) {
		inputSet = loadNodesJSON(nodesFile)
	}

	var influxdb *influx
	if ctx.GlobalIsSet(utils.MetricsEnableInfluxDBFlag.Name) {
		influxdb = &influx{}
		url := ctx.GlobalString(utils.MetricsInfluxDBEndpointFlag.Name)
		database := ctx.GlobalString(utils.MetricsInfluxDBDatabaseFlag.Name)
		username := ctx.GlobalString(utils.MetricsInfluxDBUsernameFlag.Name)
		password := ctx.GlobalString(utils.MetricsInfluxDBPasswordFlag.Name)
		if err := influxdb.connect(url, database, username, password); err != nil {
			exit(err)
		}
		log.Info("Connected to influxdb")
	}

	for {
		inputSet = crawlRound(ctx, inputSet, nodesFile, influxdb, 1*time.Minute)
	}
	return nil
}

func crawlRound(ctx *cli.Context, inputSet nodeSet, outputFile string, influxdb *influx, timeout time.Duration) nodeSet {
	var nodes []crawledNode
	disc := startV5(ctx)
	defer disc.Close()
	// Crawl the DHT for some time
	c := newCrawler(inputSet, disc, disc.RandomNodes())
	c.revalidateInterval = 10 * time.Minute
	output := c.run(timeout)
	// Try to connect and get the status of all nodes
	for _, node := range output {
		info, err := getClientInfo(node.N)
		if err != nil {
			log.Warn("GetClientInfo failed", "error", err, "nodeID", node.N.ID())
		} else {
			log.Info("GetClientInfo succeeded")
		}
		nodes = append(nodes, crawledNode{node: node, info: info})
	}
	// Write the enodes to a file
	writeNodesJSON(outputFile, output)
	// Write the node info to influx
	if influxdb != nil {
		if err := influxdb.updateNodes(nodes); err != nil {
			exit(err)
		}
	}
	return output
}

func getClientInfo(n *enode.Node) (*clientInfo, error) {
	var info clientInfo
	conn, sk, err := dial(n)
	if err != nil {
		return nil, err
	}
	conn.SetDeadline(time.Now().Add(15 * time.Second))

	// write hello to client
	pub0 := crypto.FromECDSAPub(&sk.PublicKey)[1:]
	ourHandshake := &ethtest.Hello{
		Version: 5,
		Caps: []p2p.Cap{
			{Name: "eth", Version: 64},
			{Name: "eth", Version: 65},
			{Name: "eth", Version: 66},
		},
		ID: pub0,
	}
	if err := conn.Write(ourHandshake); err != nil {
		return nil, err
	}

	// read hello from client
	switch msg := conn.Read().(type) {
	case *ethtest.Hello:
		// set snappy if version is at least 5
		if msg.Version >= 5 {
			conn.SetSnappy(true)
		}
		info.Capabilities = msg.Caps
		info.SoftwareVersion = msg.Version
		info.ClientType = msg.Name
	case *ethtest.Disconnect:
		return nil, fmt.Errorf("bad hello handshake: %v", msg.Reason.Error())
	case *ethtest.Error:
		return nil, fmt.Errorf("bad hello handshake: %v", msg.Error())
	default:
		return nil, fmt.Errorf("bad hello handshake: %v", msg.Code())
	}
	conn.SetDeadline(time.Now().Add(15 * time.Second))
	// write status message
	status, err := getCurrentStatus()
	if err != nil {
		return nil, err
	}
	status.ProtocolVersion = uint32(negotiateEthProtocol(ourHandshake.Caps, info.Capabilities))
	if err := conn.Write(status); err != nil {
		return nil, err
	}

	// read status message from client
	switch msg := conn.Read().(type) {
	case *ethtest.Status:
		info.ForkID = msg.ForkID
		info.HeadHash = msg.Head
		info.NetworkID = msg.NetworkID
		// m.ProtocolVersion
		info.TotalDifficulty = msg.TD
	case *ethtest.Disconnect:
		return nil, fmt.Errorf("bad status handshake: %v", msg.Reason.Error())
	case *ethtest.Error:
		return nil, fmt.Errorf("bad status handshake: %v", msg.Error())
	default:
		return nil, fmt.Errorf("bad status handshake: %v", msg.Code())
	}

	// Disconnect from client
	conn.Write(ethtest.Disconnect{Reason: p2p.DiscQuitting})

	return &info, nil
}

// dial attempts to dial the given node and perform a handshake,
func dial(n *enode.Node) (*ethtest.Conn, *ecdsa.PrivateKey, error) {
	var conn ethtest.Conn
	// dial
	fd, err := net.Dial("tcp", fmt.Sprintf("%v:%d", n.IP(), n.TCP()))
	if err != nil {
		return nil, nil, err
	}
	conn.Conn = rlpx.NewConn(fd, n.Pubkey())
	// do encHandshake
	ourKey, _ := crypto.GenerateKey()
	_, err = conn.Handshake(ourKey)
	if err != nil {
		return nil, nil, err
	}
	return &conn, ourKey, nil
}

func getCurrentStatus() (*ethtest.Status, error) {
	cl, err := ethclient.Dial("wss://mainnet.infura.io/ws/v3/2e5a3920f039435d9bb23729c7b65186")
	if err != nil {
		return nil, err
	}

	nwid, err := cl.NetworkID(context.Background())
	if err != nil {
		return nil, err
	}

	genesis, err := cl.HeaderByNumber(context.Background(), big.NewInt(0))
	if err != nil {
		return nil, err
	}
	header, err := cl.HeaderByNumber(context.Background(), nil)
	if err != nil {
		return nil, err
	}

	status := &ethtest.Status{
		ProtocolVersion: 66,
		NetworkID:       nwid.Uint64(),
		TD:              header.Difficulty,
		Head:            header.Hash(),
		Genesis:         genesis.Hash(),
		ForkID:          forkid.NewID(params.MainnetChainConfig, genesis.Hash(), header.Number.Uint64()),
	}
	return status, nil
}

// negotiateEthProtocol sets the Conn's eth protocol version
// to highest advertised capability from peer
func negotiateEthProtocol(caps, peer []p2p.Cap) uint {
	var highestEthVersion uint
	for _, capability := range peer {
		if capability.Name != "eth" {
			continue
		}
		if capability.Version > highestEthVersion && capability.Version <= caps[len(caps)-1].Version {
			highestEthVersion = capability.Version
		}
	}
	return highestEthVersion
}
