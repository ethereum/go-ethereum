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
	"database/sql"
	"fmt"
	"math/big"
	"net"
	"os"
	"time"

	_ "github.com/mattn/go-sqlite3"

	"github.com/ethereum/go-ethereum/cmd/devp2p/internal/ethtest"
	"github.com/ethereum/go-ethereum/cmd/utils"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/forkid"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/p2p/enode"
	"github.com/ethereum/go-ethereum/p2p/enr"
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
			utils.MainnetFlag,
			utils.RopstenFlag,
			utils.RinkebyFlag,
			utils.GoerliFlag,
			utils.NetworkIdFlag,
			crawlTimeoutFlag,
			nodeURLFlag,
			tableNameFlag,
		},
	}
	nodeURLFlag = cli.StringFlag{
		Name:  "nodeURL",
		Usage: "URL of the node you want to connect to",
		Value: "http://localhost:8545",
	}
	nodeFileFlag = cli.StringFlag{
		Name:  "nodefile",
		Usage: "Path to a node file containing nodes to be crawled",
	}
	timeoutFlag = cli.DurationFlag{
		Name:  "timeout",
		Usage: "Timeout for the crawling in a round",
		Value: 1 * time.Minute,
	}
	tableNameFlag = cli.StringFlag{
		Name:  "table",
		Usage: "Name of the sqlite table",
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

	if nodesFile := ctx.String(nodeFileFlag.Name); nodesFile != "" {
		if common.FileExist(nodesFile) {
			inputSet = loadNodesJSON(nodesFile)
		}
	}

	var db *sql.DB
	if ctx.IsSet(tableNameFlag.Name) {
		name := ctx.String(tableNameFlag.Name)
		shouldInit := false
		if _, err := os.Stat(name); os.IsNotExist(err) {
			shouldInit = true
		}
		var err error
		if db, err = sql.Open("sqlite3", name); err != nil {
			exit(err)
		}
		log.Info("Connected to db")
		if shouldInit {
			log.Info("DB did not exist, init")
			if err := createDB(db); err != nil {
				exit(err)
			}
		}

	}
	timeout := ctx.Duration(timeoutFlag.Name)

	for {
		inputSet = crawlRound(ctx, inputSet, db, timeout)
	}
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
	default:
		return core.DefaultGenesisBlock()
	}
}

func crawlRound(ctx *cli.Context, inputSet nodeSet, db *sql.DB, timeout time.Duration) nodeSet {
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

	reqChan := make(chan nodeJSON, len(output))
	respChan := make(chan crawledNode, 10)
	getNodeLoop := func(in <-chan nodeJSON, out chan<- crawledNode) {
		for {
			node := <-in
			info, err := getClientInfo(genesis, networkID, nodeURL, node.N)
			if err != nil {
				log.Warn("GetClientInfo failed", "error", err, "nodeID", node.N.ID())
			} else {
				log.Info("GetClientInfo succeeded")
			}
			out <- crawledNode{node: node, info: info}
		}
	}
	// Schedule 10 workers
	for i := 0; i < 10; i++ {
		go getNodeLoop(reqChan, respChan)
	}

	// Try to connect and get the status of all nodes
	for _, node := range output {
		reqChan <- node
	}
	var nodes []crawledNode
	for i := 0; i < len(output); i++ {
		node := <-respChan
		nodes = append(nodes, node)
	}
	// Write the node info to influx
	if db != nil {
		if err := updateNodes(db, nodes); err != nil {
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
		// Set correct TD if received TD is higher
		if msg.TD.Cmp(status.TD) > 0 {
			status.TD = msg.TD
		}
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
		header, err := getBCState(nodeURL)
		if err != nil {
			return nil, err
		}
		status.Head = header.Hash()
		status.ForkID = forkid.NewID(config, genesis, header.Number.Uint64())
	}
	return status, nil
}

func getBCState(nodeURL string) (*types.Header, error) {
	cl, err := ethclient.Dial(nodeURL)
	if err != nil {
		return nil, err
	}

	return cl.HeaderByNumber(context.Background(), nil)
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

func updateNodes(db *sql.DB, nodes []crawledNode) error {
	log.Info("Writing nodes to db", "nodes", len(nodes))
	now := time.Now()
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	stmt, err := tx.Prepare(
		`insert into nodes(ID, 
			Now,
			ClientType,
			PK,
			SoftwareVersion,
			Capabilities,
			NetworkID,
			ForkID,
			Blockheight,
			TotalDifficulty,
			HeadHash,
			IP,
			FirstSeen,
			LastSeen,
			Seq,
			Score,
			ConnType) 
			values(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`)
	if err != nil {
		return err
	}
	defer stmt.Close()
	for _, node := range nodes {
		n := node.node
		info := &clientInfo{}
		if node.info != nil {
			info = node.info
		}
		connType := ""
		var portUDP enr.UDP
		if n.N.Load(&portUDP) == nil {
			connType = "UDP"
		}
		var portTCP enr.TCP
		if n.N.Load(&portTCP) == nil {
			connType = "TCP"
		}
		var caps string
		for _, c := range info.Capabilities {
			caps = fmt.Sprintf("%v, %v", caps, c.String())
		}
		var pk string
		if n.N.Pubkey() != nil {
			pk = fmt.Sprintf("X: %v, Y: %v", n.N.Pubkey().X.String(), n.N.Pubkey().Y.String())
		}
		fid := fmt.Sprintf("Hash: %v, Next %v", info.ForkID.Hash, info.ForkID.Next)

		_, err = stmt.Exec(
			n.N.ID().String(),
			now.String(),
			info.ClientType,
			pk,
			info.SoftwareVersion,
			caps,
			info.NetworkID,
			fid,
			info.Blockheight,
			info.TotalDifficulty.String(),
			info.HeadHash.String(),
			n.N.IP().String(),
			n.FirstResponse.String(),
			n.LastResponse.String(),
			n.Seq,
			n.Score,
			connType,
		)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

func createDB(db *sql.DB) error {
	sqlStmt := `
	CREATE TABLE nodes (
		ID text not null, 
		Now text not null,
		ClientType text,
		PK text,
		SoftwareVersion text,
		Capabilities text,
		NetworkID number,
		ForkID text,
		Blockheight text,
		TotalDifficulty text,
		HeadHash text,
		IP text,
		FirstSeen text,
		LastSeen text,
		Seq number,
		Score number,
		ConnType text,
		PRIMARY KEY (ID, Now)
	);
	delete from nodes;
	`
	_, err := db.Exec(sqlStmt)
	return err
}
