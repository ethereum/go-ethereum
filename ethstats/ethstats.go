// Copyright 2016 The go-ethereum Authors
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

// Package ethstats implements the network stats reporting service.
package ethstats

import (
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"regexp"
	"runtime"
	"strconv"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/eth"
	"github.com/ethereum/go-ethereum/event"
	"github.com/ethereum/go-ethereum/les"
	"github.com/ethereum/go-ethereum/logger"
	"github.com/ethereum/go-ethereum/logger/glog"
	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/rpc"
	"golang.org/x/net/websocket"
)

// Service implements an Ethereum netstats reporting daemon that pushes local
// chain statistics up to a monitoring server.
type Service struct {
	stack *node.Node // Temporary workaround, remove when API finalized

	server *p2p.Server        // Peer-to-peer server to retrieve networking infos
	eth    *eth.Ethereum      // Full Ethereum service if monitoring a full node
	les    *les.LightEthereum // Light Ethereum service if monitoring a light node

	node string // Name of the node to display on the monitoring page
	pass string // Password to authorize access to the monitoring page
	host string // Remote address of the monitoring service
}

// New returns a monitoring service ready for stats reporting.
func New(url string, ethServ *eth.Ethereum, lesServ *les.LightEthereum) (*Service, error) {
	// Parse the netstats connection url
	re := regexp.MustCompile("([^:@]*)(:([^@]*))?@(.+)")
	parts := re.FindStringSubmatch(url)
	if len(parts) != 5 {
		return nil, fmt.Errorf("invalid netstats url: \"%s\", should be nodename:secret@host:port", url)
	}
	// Assemble and return the stats service
	return &Service{
		eth:  ethServ,
		les:  lesServ,
		node: parts[1],
		pass: parts[3],
		host: parts[4],
	}, nil
}

// Protocols implements node.Service, returning the P2P network protocols used
// by the stats service (nil as it doesn't use the devp2p overlay network).
func (s *Service) Protocols() []p2p.Protocol { return nil }

// APIs implements node.Service, returning the RPC API endpoints provided by the
// stats service (nil as it doesn't provide any user callable APIs).
func (s *Service) APIs() []rpc.API { return nil }

// Start implements node.Service, starting up the monitoring and reporting daemon.
func (s *Service) Start(server *p2p.Server) error {
	s.server = server
	go s.loop()

	glog.V(logger.Info).Infoln("Stats daemon started")
	return nil
}

// Stop implements node.Service, terminating the monitoring and reporting daemon.
func (s *Service) Stop() error {
	glog.V(logger.Info).Infoln("Stats daemon stopped")
	return nil
}

// loop keeps trying to connect to the netstats server, reporting chain events
// until termination.
func (s *Service) loop() {
	// Subscribe tso chain events to execute updates on
	var emux *event.TypeMux
	if s.eth != nil {
		emux = s.eth.EventMux()
	} else {
		emux = s.les.EventMux()
	}
	headSub := emux.Subscribe(core.ChainHeadEvent{})
	defer headSub.Unsubscribe()

	txSub := emux.Subscribe(core.TxPreEvent{})
	defer txSub.Unsubscribe()

	// Loop reporting until termination
	for {
		// Establish a websocket connection to the server and authenticate the node
		conn, err := websocket.Dial(fmt.Sprintf("wss://%s/api", s.host), "", "http://localhost/")
		if err != nil {
			glog.V(logger.Warn).Infof("Stats server unreachable: %v", err)
			time.Sleep(10 * time.Second)
			continue
		}
		in := json.NewDecoder(conn)
		out := json.NewEncoder(conn)

		if err = s.login(in, out); err != nil {
			glog.V(logger.Warn).Infof("Stats login failed: %v", err)
			conn.Close()
			time.Sleep(10 * time.Second)
			continue
		}
		if err = s.report(in, out); err != nil {
			glog.V(logger.Warn).Infof("Initial stats report failed: %v", err)
			conn.Close()
			continue
		}
		// Keep sending status updates until the connection breaks
		fullReport := time.NewTicker(15 * time.Second)

		for err == nil {
			select {
			case <-fullReport.C:
				if err = s.report(in, out); err != nil {
					glog.V(logger.Warn).Infof("Full stats report failed: %v", err)
				}
			case head := <-headSub.Chan():
				if err = s.reportBlock(out, head.Data.(core.ChainHeadEvent).Block); err != nil {
					glog.V(logger.Warn).Infof("Block stats report failed: %v", err)
				}
				if err = s.reportPending(out); err != nil {
					glog.V(logger.Warn).Infof("Post-block transaction stats report failed: %v", err)
				}
			case <-txSub.Chan():
				// Exhaust events to avoid reporting too frequently
				for exhausted := false; !exhausted; {
					select {
					case <-headSub.Chan():
					default:
						exhausted = true
					}
				}
				if err = s.reportPending(out); err != nil {
					glog.V(logger.Warn).Infof("Transaction stats report failed: %v", err)
				}
			}
		}
		// Make sure the connection is closed
		conn.Close()
	}
}

// nodeInfo is the collection of metainformation about a node that is displayed
// on the monitoring page.
type nodeInfo struct {
	Name     string `json:"name"`
	Node     string `json:"node"`
	Port     int    `json:"port"`
	Network  string `json:"net"`
	Protocol string `json:"protocol"`
	API      string `json:"api"`
	Os       string `json:"os"`
	OsVer    string `json:"os_v"`
	Client   string `json:"client"`
}

// authMsg is the authentication infos needed to login to a monitoring server.
type authMsg struct {
	Id     string   `json:"id"`
	Info   nodeInfo `json:"info"`
	Secret string   `json:"secret"`
}

// login tries to authorize the client at the remote server.
func (s *Service) login(in *json.Decoder, out *json.Encoder) error {
	// Construct and send the login authentication
	infos := s.server.NodeInfo()

	var network, protocol string
	if info := infos.Protocols["eth"]; info != nil {
		network = strconv.Itoa(info.(*eth.EthNodeInfo).Network)
		protocol = fmt.Sprintf("eth/%d", eth.ProtocolVersions[0])
	} else {
		network = strconv.Itoa(infos.Protocols["les"].(*eth.EthNodeInfo).Network)
		protocol = fmt.Sprintf("les/%d", les.ProtocolVersions[0])
	}
	auth := &authMsg{
		Id: s.node,
		Info: nodeInfo{
			Name:     s.node,
			Node:     infos.Name,
			Port:     infos.Ports.Listener,
			Network:  network,
			Protocol: protocol,
			API:      "No",
			Os:       runtime.GOOS,
			OsVer:    runtime.GOARCH,
			Client:   "0.1.1",
		},
		Secret: s.pass,
	}
	login := map[string][]interface{}{
		"emit": []interface{}{"hello", auth},
	}
	if err := out.Encode(login); err != nil {
		return err
	}
	// Retrieve the remote ack or connection termination
	var ack map[string][]string
	if err := in.Decode(&ack); err != nil || len(ack["emit"]) != 1 || ack["emit"][0] != "ready" {
		return errors.New("unauthorized")
	}
	return nil
}

// report collects all possible data to report and send it to the stats server.
// This should only be used on reconnects or rarely to avoid overloading the
// server. Use the individual methods for reporting subscribed events.
func (s *Service) report(in *json.Decoder, out *json.Encoder) error {
	if err := s.reportLatency(in, out); err != nil {
		return err
	}
	if err := s.reportBlock(out, nil); err != nil {
		return err
	}
	if err := s.reportPending(out); err != nil {
		return err
	}
	if err := s.reportStats(out); err != nil {
		return err
	}
	return nil
}

// reportLatency sends a ping request to the server, measures the RTT time and
// finally sends a latency update.
func (s *Service) reportLatency(in *json.Decoder, out *json.Encoder) error {
	// Send the current time to the ethstats server
	start := time.Now()

	ping := map[string][]interface{}{
		"emit": []interface{}{"node-ping", map[string]string{
			"id":         s.node,
			"clientTime": start.String(),
		}},
	}
	if err := out.Encode(ping); err != nil {
		return err
	}
	// Wait for the pong request to arrive back
	var pong map[string][]interface{}
	if err := in.Decode(&pong); err != nil || len(pong["emit"]) != 2 || pong["emit"][0].(string) != "node-pong" {
		return errors.New("unexpected ping reply")
	}
	// Send back the measured latency
	latency := map[string][]interface{}{
		"emit": []interface{}{"latency", map[string]string{
			"id":      s.node,
			"latency": strconv.Itoa(int((time.Since(start) / time.Duration(2)).Nanoseconds() / 1000000)),
		}},
	}
	if err := out.Encode(latency); err != nil {
		return err
	}
	return nil
}

// blockStats is the information to report about individual blocks.
type blockStats struct {
	Number    *big.Int       `json:"number"`
	Hash      common.Hash    `json:"hash"`
	Miner     common.Address `json:"miner"`
	GasUsed   *big.Int       `json:"gasUsed"`
	GasLimit  *big.Int       `json:"gasLimit"`
	Diff      string         `json:"difficulty"`
	TotalDiff string         `json:"totalDifficulty"`
	Txs       txStats        `json:"transactions"`
	Uncles    uncleStats     `json:"uncles"`
}

// txStats is a custom wrapper around a transaction array to force serializing
// empty arrays instead of returning null for them.
type txStats []*types.Transaction

func (s txStats) MarshalJSON() ([]byte, error) {
	if txs := ([]*types.Transaction)(s); len(txs) > 0 {
		return json.Marshal(txs)
	}
	return []byte("[]"), nil
}

// uncleStats is a custom wrapper around an uncle array to force serializing
// empty arrays instead of returning null for them.
type uncleStats []*types.Header

func (s uncleStats) MarshalJSON() ([]byte, error) {
	if uncles := ([]*types.Header)(s); len(uncles) > 0 {
		return json.Marshal(uncles)
	}
	return []byte("[]"), nil
}

// reportBlock retrieves the current chain head and repors it to the stats server.
func (s *Service) reportBlock(out *json.Encoder, block *types.Block) error {
	// Gather the head block infos from the local blockchain
	var (
		head   *types.Header
		td     *big.Int
		txs    []*types.Transaction
		uncles []*types.Header
	)
	if s.eth != nil {
		// Full nodes have all needed information available
		if block == nil {
			block = s.eth.BlockChain().CurrentBlock()
		}
		head = block.Header()
		td = s.eth.BlockChain().GetTd(head.Hash(), head.Number.Uint64())

		txs = block.Transactions()
		uncles = block.Uncles()
	} else {
		// Light nodes would need on-demand lookups for transactions/uncles, skip
		if block != nil {
			head = block.Header()
		} else {
			head = s.les.BlockChain().CurrentHeader()
		}
		td = s.les.BlockChain().GetTd(head.Hash(), head.Number.Uint64())
	}
	// Assemble the block stats report and send it to the server
	stats := map[string]interface{}{
		"id": s.node,
		"block": &blockStats{
			Number:    head.Number,
			Hash:      head.Hash(),
			Miner:     head.Coinbase,
			GasUsed:   new(big.Int).Set(head.GasUsed),
			GasLimit:  new(big.Int).Set(head.GasLimit),
			Diff:      head.Difficulty.String(),
			TotalDiff: td.String(),
			Txs:       txs,
			Uncles:    uncles,
		},
	}
	report := map[string][]interface{}{
		"emit": []interface{}{"block", stats},
	}
	if err := out.Encode(report); err != nil {
		return err
	}
	return nil
}

// pendStats is the information to report about pending transactions.
type pendStats struct {
	Pending int `json:"pending"`
}

// reportPending retrieves the current number of pending transactions and reports
// it to the stats server.
func (s *Service) reportPending(out *json.Encoder) error {
	// Retrieve the pending count from the local blockchain
	var pending int
	if s.eth != nil {
		pending, _ = s.eth.TxPool().Stats()
	} else {
		pending = s.les.TxPool().Stats()
	}
	// Assemble the transaction stats and send it to the server
	stats := map[string]interface{}{
		"id": s.node,
		"stats": &pendStats{
			Pending: pending,
		},
	}
	report := map[string][]interface{}{
		"emit": []interface{}{"pending", stats},
	}
	if err := out.Encode(report); err != nil {
		return err
	}
	return nil
}

// blockStats is the information to report about the local node.
type nodeStats struct {
	Active   bool `json:"active"`
	Syncing  bool `json:"syncing"`
	Mining   bool `json:"mining"`
	Hashrate int  `json:"hashrate"`
	Peers    int  `json:"peers"`
	GasPrice int  `json:"gasPrice"`
	Uptime   int  `json:"uptime"`
}

// reportPending retrieves various stats about the node at the networking and
// mining layer and reports it to the stats server.
func (s *Service) reportStats(out *json.Encoder) error {
	// Gather the syncing and mining infos from the local miner instance
	var (
		mining   bool
		hashrate int
		syncing  bool
		gasprice int
	)
	if s.eth != nil {
		mining = s.eth.Miner().Mining()
		hashrate = int(s.eth.Miner().HashRate())

		sync := s.eth.Downloader().Progress()
		syncing = s.eth.BlockChain().CurrentHeader().Number.Uint64() >= sync.HighestBlock

		gasprice = int(s.eth.Miner().GasPrice().Uint64())
	} else {
		sync := s.les.Downloader().Progress()
		syncing = s.les.BlockChain().CurrentHeader().Number.Uint64() >= sync.HighestBlock
	}
	stats := map[string]interface{}{
		"id": s.node,
		"stats": &nodeStats{
			Active:   true,
			Mining:   mining,
			Hashrate: hashrate,
			Peers:    s.server.PeerCount(),
			GasPrice: gasprice,
			Syncing:  syncing,
			Uptime:   100,
		},
	}
	report := map[string][]interface{}{
		"emit": []interface{}{"stats", stats},
	}
	if err := out.Encode(report); err != nil {
		return err
	}
	return nil
}
