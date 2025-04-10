// Copyright 2017 The go-ethereum Authors
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

package simulations

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/XinFinOrg/XDPoSChain/event"
	"github.com/XinFinOrg/XDPoSChain/log"
	"github.com/XinFinOrg/XDPoSChain/p2p"
	"github.com/XinFinOrg/XDPoSChain/p2p/discover"
	"github.com/XinFinOrg/XDPoSChain/p2p/simulations/adapters"
)

var dialBanTimeout = 200 * time.Millisecond

// NetworkConfig defines configuration options for starting a Network
type NetworkConfig struct {
	ID             string `json:"id"`
	DefaultService string `json:"default_service,omitempty"`
}

// Network models a p2p simulation network which consists of a collection of
// simulated nodes and the connections which exist between them.
//
// The Network has a single NodeAdapter which is responsible for actually
// starting nodes and connecting them together.
//
// The Network emits events when nodes are started and stopped, when they are
// connected and disconnected, and also when messages are sent between nodes.
type Network struct {
	NetworkConfig

	Nodes   []*Node `json:"nodes"`
	nodeMap map[discover.NodeID]int

	Conns   []*Conn `json:"conns"`
	connMap map[string]int

	nodeAdapter adapters.NodeAdapter
	events      event.Feed
	lock        sync.RWMutex
	quitc       chan struct{}
}

// NewNetwork returns a Network which uses the given NodeAdapter and NetworkConfig
func NewNetwork(nodeAdapter adapters.NodeAdapter, conf *NetworkConfig) *Network {
	return &Network{
		NetworkConfig: *conf,
		nodeAdapter:   nodeAdapter,
		nodeMap:       make(map[discover.NodeID]int),
		connMap:       make(map[string]int),
		quitc:         make(chan struct{}),
	}
}

// Events returns the output event feed of the Network.
func (net *Network) Events() *event.Feed {
	return &net.events
}

// NewNode adds a new node to the network with a random ID
func (net *Network) NewNode() (*Node, error) {
	conf := adapters.RandomNodeConfig()
	conf.Lifecycles = []string{net.DefaultService}
	return net.NewNodeWithConfig(conf)
}

// NewNodeWithConfig adds a new node to the network with the given config,
// returning an error if a node with the same ID or name already exists
func (net *Network) NewNodeWithConfig(conf *adapters.NodeConfig) (*Node, error) {
	net.lock.Lock()
	defer net.lock.Unlock()

	// create a random ID and PrivateKey if not set
	if conf.ID == (discover.NodeID{}) {
		c := adapters.RandomNodeConfig()
		conf.ID = c.ID
		conf.PrivateKey = c.PrivateKey
	}
	id := conf.ID
	if conf.Reachable == nil {
		conf.Reachable = func(otherID discover.NodeID) bool {
			_, err := net.InitConn(conf.ID, otherID)
			return err == nil
		}
	}

	// assign a name to the node if not set
	if conf.Name == "" {
		conf.Name = fmt.Sprintf("node%02d", len(net.Nodes)+1)
	}

	// check the node doesn't already exist
	if node := net.getNode(id); node != nil {
		return nil, fmt.Errorf("node with ID %q already exists", id)
	}
	if node := net.getNodeByName(conf.Name); node != nil {
		return nil, fmt.Errorf("node with name %q already exists", conf.Name)
	}

	// if no services are configured, use the default service
	if len(conf.Lifecycles) == 0 {
		conf.Lifecycles = []string{net.DefaultService}
	}

	// use the NodeAdapter to create the node
	adapterNode, err := net.nodeAdapter.NewNode(conf)
	if err != nil {
		return nil, err
	}
	node := &Node{
		Node:   adapterNode,
		Config: conf,
	}
	log.Trace(fmt.Sprintf("node %v created", id))
	net.nodeMap[id] = len(net.Nodes)
	net.Nodes = append(net.Nodes, node)

	// emit a "control" event
	net.events.Send(ControlEvent(node))

	return node, nil
}

// Config returns the network configuration
func (net *Network) Config() *NetworkConfig {
	return &net.NetworkConfig
}

// StartAll starts all nodes in the network
func (net *Network) StartAll() error {
	for _, node := range net.Nodes {
		if node.Up {
			continue
		}
		if err := net.Start(node.ID()); err != nil {
			return err
		}
	}
	return nil
}

// StopAll stops all nodes in the network
func (net *Network) StopAll() error {
	for _, node := range net.Nodes {
		if !node.Up {
			continue
		}
		if err := net.Stop(node.ID()); err != nil {
			return err
		}
	}
	return nil
}

// Start starts the node with the given ID
func (net *Network) Start(id discover.NodeID) error {
	return net.startWithSnapshots(id, nil)
}

// startWithSnapshots starts the node with the given ID using the give
// snapshots
func (net *Network) startWithSnapshots(id discover.NodeID, snapshots map[string][]byte) error {
	node := net.GetNode(id)
	if node == nil {
		return fmt.Errorf("node %v does not exist", id)
	}
	if node.Up {
		return fmt.Errorf("node %v already up", id)
	}
	log.Trace(fmt.Sprintf("starting node %v: %v using %v", id, node.Up, net.nodeAdapter.Name()))
	if err := node.Start(snapshots); err != nil {
		log.Warn(fmt.Sprintf("start up failed: %v", err))
		return err
	}
	node.Up = true
	log.Info(fmt.Sprintf("started node %v: %v", id, node.Up))

	net.events.Send(NewEvent(node))

	// subscribe to peer events
	client, err := node.Client()
	if err != nil {
		return fmt.Errorf("error getting rpc client  for node %v: %s", id, err)
	}
	events := make(chan *p2p.PeerEvent)
	sub, err := client.Subscribe(context.Background(), "admin", events, "peerEvents")
	if err != nil {
		return fmt.Errorf("error getting peer events for node %v: %s", id, err)
	}
	go net.watchPeerEvents(id, events, sub)
	return nil
}

// watchPeerEvents reads peer events from the given channel and emits
// corresponding network events
func (net *Network) watchPeerEvents(id discover.NodeID, events chan *p2p.PeerEvent, sub event.Subscription) {
	defer func() {
		sub.Unsubscribe()

		// assume the node is now down
		net.lock.Lock()
		node := net.getNode(id)
		node.Up = false
		net.lock.Unlock()
		net.events.Send(NewEvent(node))
	}()
	for {
		select {
		case event, ok := <-events:
			if !ok {
				return
			}
			peer := event.Peer
			switch event.Type {

			case p2p.PeerEventTypeAdd:
				net.DidConnect(id, peer)

			case p2p.PeerEventTypeDrop:
				net.DidDisconnect(id, peer)

			case p2p.PeerEventTypeMsgSend:
				net.DidSend(id, peer, event.Protocol, *event.MsgCode)

			case p2p.PeerEventTypeMsgRecv:
				net.DidReceive(peer, id, event.Protocol, *event.MsgCode)

			}

		case err := <-sub.Err():
			if err != nil {
				log.Error(fmt.Sprintf("error getting peer events for node %v", id), "err", err)
			}
			return
		}
	}
}

// Stop stops the node with the given ID
func (net *Network) Stop(id discover.NodeID) error {
	node := net.GetNode(id)
	if node == nil {
		return fmt.Errorf("node %v does not exist", id)
	}
	if !node.Up {
		return fmt.Errorf("node %v already down", id)
	}
	if err := node.Stop(); err != nil {
		return err
	}
	node.Up = false
	log.Info(fmt.Sprintf("stop node %v: %v", id, node.Up))

	net.events.Send(ControlEvent(node))
	return nil
}

// Connect connects two nodes together by calling the "admin_addPeer" RPC
// method on the "one" node so that it connects to the "other" node
func (net *Network) Connect(oneID, otherID discover.NodeID) error {
	log.Debug(fmt.Sprintf("connecting %s to %s", oneID, otherID))
	conn, err := net.InitConn(oneID, otherID)
	if err != nil {
		return err
	}
	client, err := conn.one.Client()
	if err != nil {
		return err
	}
	net.events.Send(ControlEvent(conn))
	return client.Call(nil, "admin_addPeer", string(conn.other.Addr()))
}

// Disconnect disconnects two nodes by calling the "admin_removePeer" RPC
// method on the "one" node so that it disconnects from the "other" node
func (net *Network) Disconnect(oneID, otherID discover.NodeID) error {
	conn := net.GetConn(oneID, otherID)
	if conn == nil {
		return fmt.Errorf("connection between %v and %v does not exist", oneID, otherID)
	}
	if !conn.Up {
		return fmt.Errorf("%v and %v already disconnected", oneID, otherID)
	}
	client, err := conn.one.Client()
	if err != nil {
		return err
	}
	net.events.Send(ControlEvent(conn))
	return client.Call(nil, "admin_removePeer", string(conn.other.Addr()))
}

// DidConnect tracks the fact that the "one" node connected to the "other" node
func (net *Network) DidConnect(one, other discover.NodeID) error {
	conn, err := net.GetOrCreateConn(one, other)
	if err != nil {
		return fmt.Errorf("connection between %v and %v does not exist", one, other)
	}
	if conn.Up {
		return fmt.Errorf("%v and %v already connected", one, other)
	}
	conn.Up = true
	net.events.Send(NewEvent(conn))
	return nil
}

// DidDisconnect tracks the fact that the "one" node disconnected from the
// "other" node
func (net *Network) DidDisconnect(one, other discover.NodeID) error {
	conn := net.GetConn(one, other)
	if conn == nil {
		return fmt.Errorf("connection between %v and %v does not exist", one, other)
	}
	if !conn.Up {
		return fmt.Errorf("%v and %v already disconnected", one, other)
	}
	conn.Up = false
	conn.initiated = time.Now().Add(-dialBanTimeout)
	net.events.Send(NewEvent(conn))
	return nil
}

// DidSend tracks the fact that "sender" sent a message to "receiver"
func (net *Network) DidSend(sender, receiver discover.NodeID, proto string, code uint64) error {
	msg := &Msg{
		One:      sender,
		Other:    receiver,
		Protocol: proto,
		Code:     code,
		Received: false,
	}
	net.events.Send(NewEvent(msg))
	return nil
}

// DidReceive tracks the fact that "receiver" received a message from "sender"
func (net *Network) DidReceive(sender, receiver discover.NodeID, proto string, code uint64) error {
	msg := &Msg{
		One:      sender,
		Other:    receiver,
		Protocol: proto,
		Code:     code,
		Received: true,
	}
	net.events.Send(NewEvent(msg))
	return nil
}

// GetNode gets the node with the given ID, returning nil if the node does not
// exist
func (net *Network) GetNode(id discover.NodeID) *Node {
	net.lock.Lock()
	defer net.lock.Unlock()
	return net.getNode(id)
}

// GetNode gets the node with the given name, returning nil if the node does
// not exist
func (net *Network) GetNodeByName(name string) *Node {
	net.lock.Lock()
	defer net.lock.Unlock()
	return net.getNodeByName(name)
}

func (net *Network) getNode(id discover.NodeID) *Node {
	i, found := net.nodeMap[id]
	if !found {
		return nil
	}
	return net.Nodes[i]
}

func (net *Network) getNodeByName(name string) *Node {
	for _, node := range net.Nodes {
		if node.Config.Name == name {
			return node
		}
	}
	return nil
}

// GetNodes returns the existing nodes
func (net *Network) GetNodes() (nodes []*Node) {
	net.lock.Lock()
	defer net.lock.Unlock()

	nodes = append(nodes, net.Nodes...)
	return nodes
}

// GetConn returns the connection which exists between "one" and "other"
// regardless of which node initiated the connection
func (net *Network) GetConn(oneID, otherID discover.NodeID) *Conn {
	net.lock.Lock()
	defer net.lock.Unlock()
	return net.getConn(oneID, otherID)
}

// GetOrCreateConn is like GetConn but creates the connection if it doesn't
// already exist
func (net *Network) GetOrCreateConn(oneID, otherID discover.NodeID) (*Conn, error) {
	net.lock.Lock()
	defer net.lock.Unlock()
	return net.getOrCreateConn(oneID, otherID)
}

func (net *Network) getOrCreateConn(oneID, otherID discover.NodeID) (*Conn, error) {
	if conn := net.getConn(oneID, otherID); conn != nil {
		return conn, nil
	}

	one := net.getNode(oneID)
	if one == nil {
		return nil, fmt.Errorf("node %v does not exist", oneID)
	}
	other := net.getNode(otherID)
	if other == nil {
		return nil, fmt.Errorf("node %v does not exist", otherID)
	}
	conn := &Conn{
		One:   oneID,
		Other: otherID,
		one:   one,
		other: other,
	}
	label := ConnLabel(oneID, otherID)
	net.connMap[label] = len(net.Conns)
	net.Conns = append(net.Conns, conn)
	return conn, nil
}

func (net *Network) getConn(oneID, otherID discover.NodeID) *Conn {
	label := ConnLabel(oneID, otherID)
	i, found := net.connMap[label]
	if !found {
		return nil
	}
	return net.Conns[i]
}

// InitConn(one, other) retrieves the connectiton model for the connection between
// peers one and other, or creates a new one if it does not exist
// the order of nodes does not matter, i.e., Conn(i,j) == Conn(j, i)
// it checks if the connection is already up, and if the nodes are running
// NOTE:
// it also checks whether there has been recent attempt to connect the peers
// this is cheating as the simulation is used as an oracle and know about
// remote peers attempt to connect to a node which will then not initiate the connection
func (net *Network) InitConn(oneID, otherID discover.NodeID) (*Conn, error) {
	net.lock.Lock()
	defer net.lock.Unlock()
	if oneID == otherID {
		return nil, fmt.Errorf("refusing to connect to self %v", oneID)
	}
	conn, err := net.getOrCreateConn(oneID, otherID)
	if err != nil {
		return nil, err
	}
	if time.Since(conn.initiated) < dialBanTimeout {
		return nil, fmt.Errorf("connection between %v and %v recently attempted", oneID, otherID)
	}
	if conn.Up {
		return nil, fmt.Errorf("%v and %v already connected", oneID, otherID)
	}
	err = conn.nodesUp()
	if err != nil {
		return nil, fmt.Errorf("nodes not up: %v", err)
	}
	conn.initiated = time.Now()
	return conn, nil
}

// Shutdown stops all nodes in the network and closes the quit channel
func (net *Network) Shutdown() {
	for _, node := range net.Nodes {
		log.Debug(fmt.Sprintf("stopping node %s", node.ID().TerminalString()))
		if err := node.Stop(); err != nil {
			log.Warn(fmt.Sprintf("error stopping node %s", node.ID().TerminalString()), "err", err)
		}
		// If the node has the close method, call it.
		if closer, ok := node.Node.(io.Closer); ok {
			if err := closer.Close(); err != nil {
				log.Warn("Can't close node", "id", node.ID(), "err", err)
			}
		}
	}
	close(net.quitc)
}

// Reset resets all network properties:
// emtpies the nodes and the connection list
func (net *Network) Reset() {
	net.lock.Lock()
	defer net.lock.Unlock()

	//re-initialize the maps
	net.connMap = make(map[string]int)
	net.nodeMap = make(map[discover.NodeID]int)

	net.Nodes = nil
	net.Conns = nil
}

// Node is a wrapper around adapters.Node which is used to track the status
// of a node in the network
type Node struct {
	adapters.Node `json:"-"`

	// Config if the config used to created the node
	Config *adapters.NodeConfig `json:"config"`

	// Up tracks whether or not the node is running
	Up bool `json:"up"`
}

// ID returns the ID of the node
func (n *Node) ID() discover.NodeID {
	return n.Config.ID
}

// String returns a log-friendly string
func (n *Node) String() string {
	return fmt.Sprintf("Node %v", n.ID().TerminalString())
}

// NodeInfo returns information about the node
func (n *Node) NodeInfo() *p2p.NodeInfo {
	// avoid a panic if the node is not started yet
	if n.Node == nil {
		return nil
	}
	info := n.Node.NodeInfo()
	if info == nil {
		return nil
	}

	info.Name = n.Config.Name
	return info
}

// MarshalJSON implements the json.Marshaler interface so that the encoded
// JSON includes the NodeInfo
func (n *Node) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Info   *p2p.NodeInfo        `json:"info,omitempty"`
		Config *adapters.NodeConfig `json:"config,omitempty"`
		Up     bool                 `json:"up"`
	}{
		Info:   n.NodeInfo(),
		Config: n.Config,
		Up:     n.Up,
	})
}

// Conn represents a connection between two nodes in the network
type Conn struct {
	// One is the node which initiated the connection
	One discover.NodeID `json:"one"`

	// Other is the node which the connection was made to
	Other discover.NodeID `json:"other"`

	// Up tracks whether or not the connection is active
	Up bool `json:"up"`
	// Registers when the connection was grabbed to dial
	initiated time.Time

	one   *Node
	other *Node
}

// nodesUp returns whether both nodes are currently up
func (c *Conn) nodesUp() error {
	if !c.one.Up {
		return fmt.Errorf("one %v is not up", c.One)
	}
	if !c.other.Up {
		return fmt.Errorf("other %v is not up", c.Other)
	}
	return nil
}

// String returns a log-friendly string
func (c *Conn) String() string {
	return fmt.Sprintf("Conn %v->%v", c.One.TerminalString(), c.Other.TerminalString())
}

// Msg represents a p2p message sent between two nodes in the network
type Msg struct {
	One      discover.NodeID `json:"one"`
	Other    discover.NodeID `json:"other"`
	Protocol string          `json:"protocol"`
	Code     uint64          `json:"code"`
	Received bool            `json:"received"`
}

// String returns a log-friendly string
func (m *Msg) String() string {
	return fmt.Sprintf("Msg(%d) %v->%v", m.Code, m.One.TerminalString(), m.Other.TerminalString())
}

// ConnLabel generates a deterministic string which represents a connection
// between two nodes, used to compare if two connections are between the same
// nodes
func ConnLabel(source, target discover.NodeID) string {
	var first, second discover.NodeID
	if bytes.Compare(source.Bytes(), target.Bytes()) > 0 {
		first = target
		second = source
	} else {
		first = source
		second = target
	}
	return fmt.Sprintf("%v-%v", first, second)
}

// Snapshot represents the state of a network at a single point in time and can
// be used to restore the state of a network
type Snapshot struct {
	Nodes []NodeSnapshot `json:"nodes,omitempty"`
	Conns []Conn         `json:"conns,omitempty"`
}

// NodeSnapshot represents the state of a node in the network
type NodeSnapshot struct {
	Node Node `json:"node,omitempty"`

	// Snapshots is arbitrary data gathered from calling node.Snapshots()
	Snapshots map[string][]byte `json:"snapshots,omitempty"`
}

// Snapshot creates a network snapshot
func (net *Network) Snapshot() (*Snapshot, error) {
	net.lock.Lock()
	defer net.lock.Unlock()
	snap := &Snapshot{
		Nodes: make([]NodeSnapshot, len(net.Nodes)),
		Conns: make([]Conn, len(net.Conns)),
	}
	for i, node := range net.Nodes {
		snap.Nodes[i] = NodeSnapshot{Node: *node}
		if !node.Up {
			continue
		}
		snapshots, err := node.Snapshots()
		if err != nil {
			return nil, err
		}
		snap.Nodes[i].Snapshots = snapshots
	}
	for i, conn := range net.Conns {
		snap.Conns[i] = *conn
	}
	return snap, nil
}

// Load loads a network snapshot
func (net *Network) Load(snap *Snapshot) error {
	for _, n := range snap.Nodes {
		if _, err := net.NewNodeWithConfig(n.Node.Config); err != nil {
			return err
		}
		if !n.Node.Up {
			continue
		}
		if err := net.startWithSnapshots(n.Node.Config.ID, n.Snapshots); err != nil {
			return err
		}
	}
	for _, conn := range snap.Conns {

		if !net.GetNode(conn.One).Up || !net.GetNode(conn.Other).Up {
			//in this case, at least one of the nodes of a connection is not up,
			//so it would result in the snapshot `Load` to fail
			continue
		}
		if err := net.Connect(conn.One, conn.Other); err != nil {
			return err
		}
	}
	return nil
}

// Subscribe reads control events from a channel and executes them
func (net *Network) Subscribe(events chan *Event) {
	for {
		select {
		case event, ok := <-events:
			if !ok {
				return
			}
			if event.Control {
				net.executeControlEvent(event)
			}
		case <-net.quitc:
			return
		}
	}
}

func (net *Network) executeControlEvent(event *Event) {
	log.Trace("execute control event", "type", event.Type, "event", event)
	switch event.Type {
	case EventTypeNode:
		if err := net.executeNodeEvent(event); err != nil {
			log.Error("error executing node event", "event", event, "err", err)
		}
	case EventTypeConn:
		if err := net.executeConnEvent(event); err != nil {
			log.Error("error executing conn event", "event", event, "err", err)
		}
	case EventTypeMsg:
		log.Warn("ignoring control msg event")
	}
}

func (net *Network) executeNodeEvent(e *Event) error {
	if !e.Node.Up {
		return net.Stop(e.Node.ID())
	}

	if _, err := net.NewNodeWithConfig(e.Node.Config); err != nil {
		return err
	}
	return net.Start(e.Node.ID())
}

func (net *Network) executeConnEvent(e *Event) error {
	if e.Conn.Up {
		return net.Connect(e.Conn.One, e.Conn.Other)
	} else {
		return net.Disconnect(e.Conn.One, e.Conn.Other)
	}
}
