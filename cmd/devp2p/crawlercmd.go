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
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/forkid"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/p2p/enode"
	"github.com/ethereum/go-ethereum/p2p/rlpx"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/rpc"
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
			utils.MainnetFlag,
			utils.RopstenFlag,
			utils.RinkebyFlag,
			utils.GoerliFlag,
			utils.NetworkIdFlag,
			crawlTimeoutFlag,
			nodeURLFlag,
			influxDBFlag,
			influxDBURLFlag,
			influxDBBucketFlag,
			influxDBOrgFlag,
			influxDBTokenFlag,
		},
	}
	nodeURLFlag = cli.StringFlag{
		Name:  "nodeURL",
		Usage: "URL of the node you want to connect to",
		Value: "http://localhost:8545",
	}
	influxDBFlag = cli.BoolFlag{
		Name:  "influxdb",
		Usage: "Store the crawled data in a influxdb",
	}
	influxDBURLFlag = cli.StringFlag{
		Name:  "influxdb.url",
		Usage: "URL where the influxdb is reachable",
	}
	influxDBBucketFlag = cli.StringFlag{
		Name:  "influxdb.bucket",
		Usage: "Bucket where the data should be stored",
	}
	influxDBOrgFlag = cli.StringFlag{
		Name:  "influxdb.org",
		Usage: "InfluxDB organization where data should be stored",
	}
	influxDBTokenFlag = cli.StringFlag{
		Name:  "influxdb.token",
		Usage: "Token used to authenticate with influxdb",
	}
	status           *ethtest.Status
	lastStatusUpdate time.Time
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
	if ctx.IsSet(influxDBFlag.Name) {
		url := ctx.String(influxDBURLFlag.Name)
		bucket := ctx.String(influxDBBucketFlag.Name)
		org := ctx.String(influxDBOrgFlag.Name)
		token := ctx.String(influxDBTokenFlag.Name)
		var err error
		if influxdb, err = NewInflux(url, bucket, org, token); err != nil {
			exit(err)
		}
		log.Info("Connected to influxdb")
	}

	for {
		inputSet = crawlRound(ctx, inputSet, nodesFile, influxdb, 10*time.Second)
	}
	return nil
}

func discv5(ctx *cli.Context, inputSet nodeSet, timeout time.Duration) nodeSet {
	disc := startV5(ctx)
	defer disc.Close()
	// Crawl the DHT for some time
	c := newCrawler(inputSet, disc, disc.RandomNodes())
	c.revalidateInterval = 10 * time.Minute
	return c.run(timeout)
}

func discv4(ctx *cli.Context, inputSet nodeSet, timeout time.Duration) nodeSet {
	disc := startV4(ctx)
	defer disc.Close()
	// Crawl the DHT for some time
	c := newCrawler(inputSet, disc, disc.RandomNodes())
	c.revalidateInterval = 10 * time.Minute
	return c.run(timeout)
}

// makeGenesis is the pendant to utils.MakeGenesis
// with local flags instead of global flags.
func makeGenesis(ctx *cli.Context) *core.Genesis {
	switch {
	case ctx.Bool(utils.RopstenFlag.Name):
		return core.DefaultRopstenGenesisBlock()
	case ctx.Bool(utils.RinkebyFlag.Name):
		return core.DefaultRinkebyGenesisBlock()
	case ctx.Bool(utils.GoerliFlag.Name):
		return core.DefaultGoerliGenesisBlock()
	case ctx.Bool(utils.YoloV3Flag.Name):
		return core.DefaultYoloV3GenesisBlock()
	default:
		return core.DefaultGenesisBlock()
	}
}

func crawlRound(ctx *cli.Context, inputSet nodeSet, outputFile string, influxdb *influx, timeout time.Duration) nodeSet {
	var nodes []crawledNode
	output := make(nodeSet)
	log.Info("DiscV5")
	v5 := discv5(ctx, nodeSet{}, timeout)
	output.add(v5.nodes()...)
	log.Info("DiscV4")
	v4 := discv4(ctx, nodeSet{}, timeout)
	output.add(v4.nodes()...)

	genesis := makeGenesis(ctx)
	if genesis == nil {
		genesis = core.DefaultGenesisBlock()
	}
	networkID := ctx.Uint64(utils.NetworkIdFlag.Name)
	nodeURL := ctx.String(nodeURLFlag.Name)
	// Try to connect and get the status of all nodes
	for _, node := range output {
		info, err := getClientInfo(genesis, networkID, nodeURL, node.N)
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

func getClientInfo(genesis *core.Genesis, networkID uint64, nodeURL string, n *enode.Node) (*clientInfo, error) {
	var info clientInfo
	conn, sk, err := dial(n)
	if err != nil {
		return nil, err
	}
	defer conn.Close()
	conn.SetDeadline(time.Now().Add(5 * time.Second))

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
		fmt.Printf("Caps: %v \n", msg.Caps)
	case *ethtest.Disconnect:
		return nil, fmt.Errorf("bad hello handshake: %v", msg.Reason.Error())
	case *ethtest.Error:
		return nil, fmt.Errorf("bad hello handshake: %v", msg.Error())
	default:
		return nil, fmt.Errorf("bad hello handshake: %v", msg.Code())
	}
	highestEthVersion := uint32(negotiateEthProtocol(ourHandshake.Caps, info.Capabilities))
	// If node provides no eth version, we can skip it.
	if highestEthVersion == 0 {
		return &info, nil
	}
	conn.SetDeadline(time.Now().Add(15 * time.Second))
	// write status message, if we have a backing node
	if len(nodeURL) > 0 {
		// write status message
		if status, err := getStatus(genesis.Config, genesis.ToBlock(nil).Hash(), networkID, nodeURL); err != nil {
			log.Error("Local node failed to respond", "err", err)
		} else {
			status.ProtocolVersion = highestEthVersion
			if err := conn.Write(status); err != nil {
				return nil, err
			}
		}
	}

	// Regardless of whether we wrote a status message or not, the remote side
	// might still send us one.

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

func getStatus(config *params.ChainConfig, genesis common.Hash, network uint64, nodeURL string) (*ethtest.Status, error) {
	if status == nil {
		status = &ethtest.Status{
			ProtocolVersion: 66,
			NetworkID:       network,
			TD:              big.NewInt(0),
			Head:            common.Hash{},
			Genesis:         genesis,
			ForkID:          forkid.NewID(config, genesis, 0),
		}
		lastStatusUpdate = time.Time{}
	}

	if time.Since(lastStatusUpdate) > 15*time.Second {
		header, td, err := getBCState(nodeURL)
		if err != nil {
			return nil, err
		}
		status.Head = header.Hash()
		status.TD = td
		status.ForkID = forkid.NewID(config, genesis, header.Number.Uint64())
	}
	return status, nil
}

func getBCState(nodeURL string) (*types.Header, *big.Int, error) {
	raw, err := rpc.Dial(nodeURL)
	if err != nil {
		return nil, nil, err
	}

	// Retrieve total difficulty
	var result types.Block
	if err := raw.CallContext(context.Background(), &result, "eth_getBlockByNumber", "latest", false); err != nil {
		return nil, nil, err
	}

	return result.Header(), result.DeprecatedTd(), nil
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
