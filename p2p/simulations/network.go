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
	"sync"

	"github.com/ethereum/go-ethereum/event"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/p2p/discover"
	"github.com/ethereum/go-ethereum/p2p/simulations/adapters"
)

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
func (self *Network) Events() *event.Feed {
	return &self.events
}

// NewNode adds a new node to the network with a random ID
func (self *Network) NewNode() (*Node, error) {
	conf := adapters.RandomNodeConfig()
	conf.Services = []string{self.DefaultService}
	return self.NewNodeWithConfig(conf)
}

// NewNodeWithConfig adds a new node to the network with the given config,
// returning an error if a node with the same ID or name already exists
func (self *Network) NewNodeWithConfig(conf *adapters.NodeConfig) (*Node, error) {
	self.lock.Lock()
	defer self.lock.Unlock()

	// create a random ID and PrivateKey if not set
	if conf.ID == (discover.NodeID{}) {
		c := adapters.RandomNodeConfig()
		conf.ID = c.ID
		conf.PrivateKey = c.PrivateKey
	}
	id := conf.ID

	// assign a name to the node if not set
	if conf.Name == "" {
		conf.Name = fmt.Sprintf("node%02d", len(self.Nodes)+1)
	}

	// check the node doesn't already exist
	if node := self.getNode(id); node != nil {
		return nil, fmt.Errorf("node with ID %q already exists", id)
	}
	if node := self.getNodeByName(conf.Name); node != nil {
		return nil, fmt.Errorf("node with name %q already exists", conf.Name)
	}

	// if no services are configured, use the default service
	if len(conf.Services) == 0 {
		conf.Services = []string{self.DefaultService}
	}

	// use the NodeAdapter to create the node
	adapterNode, err := self.nodeAdapter.NewNode(conf)
	if err != nil {
		return nil, err
	}
	node := &Node{
		Node:   adapterNode,
		Config: conf,
	}
	log.Trace(fmt.Sprintf("node %v created", id))
	self.nodeMap[id] = len(self.Nodes)
	self.Nodes = append(self.Nodes, node)

	// emit a "control" event
	self.events.Send(ControlEvent(node))

	return node, nil
}

// Config returns the network configuration
func (self *Network) Config() *NetworkConfig {
	return &self.NetworkConfig
}

// StartAll starts all nodes in the network
func (self *Network) StartAll() error {
	for _, node := range self.Nodes {
		if node.Up {
			continue
		}
		if err := self.Start(node.ID()); err != nil {
			return err
		}
	}
	return nil
}

// StopAll stops all nodes in the network
func (self *Network) StopAll() error {
	for _, node := range self.Nodes {
		if !node.Up {
			continue
		}
		if err := self.Stop(node.ID()); err != nil {
			return err
		}
	}
	return nil
}

// Start starts the node with the given ID
func (self *Network) Start(id discover.NodeID) error {
	return self.startWithSnapshots(id, nil)
}

// startWithSnapshots starts the node with the given ID using the give
// snapshots
func (self *Network) startWithSnapshots(id discover.NodeID, snapshots map[string][]byte) error {
	node := self.GetNode(id)
	if node == nil {
		return fmt.Errorf("node %v does not exist", id)
	}
	if node.Up {
		return fmt.Errorf("node %v already up", id)
	}
	log.Trace(fmt.Sprintf("starting node %v: %v using %v", id, node.Up, self.nodeAdapter.Name()))
	if err := node.Start(snapshots); err != nil {
		log.Warn(fmt.Sprintf("start up failed: %v", err))
		return err
	}
	node.Up = true
	log.Info(fmt.Sprintf("started node %v: %v", id, node.Up))

	self.events.Send(NewEvent(node))

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
	go self.watchPeerEvents(id, events, sub)
	return nil
}

// watchPeerEvents reads peer events from the given channel and emits
// corresponding network events
func (self *Network) watchPeerEvents(id discover.NodeID, events chan *p2p.PeerEvent, sub event.Subscription) {
	defer func() {
		sub.Unsubscribe()

		// assume the node is now down
		self.lock.Lock()
		node := self.getNode(id)
		node.Up = false
		self.lock.Unlock()
		self.events.Send(NewEvent(node))
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
				self.DidConnect(id, peer)

			case p2p.PeerEventTypeDrop:
				self.DidDisconnect(id, peer)

			case p2p.PeerEventTypeMsgSend:
				self.DidSend(id, peer, event.Protocol, *event.MsgCode)

			case p2p.PeerEventTypeMsgRecv:
				self.DidReceive(peer, id, event.Protocol, *event.MsgCode)

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
func (self *Network) Stop(id discover.NodeID) error {
	node := self.GetNode(id)
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

	self.events.Send(ControlEvent(node))
	return nil
}

// Connect connects two nodes together by calling the "admin_addPeer" RPC
// method on the "one" node so that it connects to the "other" node
func (self *Network) Connect(oneID, otherID discover.NodeID) error {
	log.Debug(fmt.Sprintf("connecting %s to %s", oneID, otherID))
	conn, err := self.GetOrCreateConn(oneID, otherID)
	if err != nil {
		return err
	}
	if conn.Up {
		return fmt.Errorf("%v and %v already connected", oneID, otherID)
	}
	if err := conn.nodesUp(); err != nil {
		return err
	}
	client, err := conn.one.Client()
	if err != nil {
		return err
	}
	self.events.Send(ControlEvent(conn))
	return client.Call(nil, "admin_addPeer", string(conn.other.Addr()))
}

// Disconnect disconnects two nodes by calling the "admin_removePeer" RPC
// method on the "one" node so that it disconnects from the "other" node
func (self *Network) Disconnect(oneID, otherID discover.NodeID) error {
	conn := self.GetConn(oneID, otherID)
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
	self.events.Send(ControlEvent(conn))
	return client.Call(nil, "admin_removePeer", string(conn.other.Addr()))
}

// DidConnect tracks the fact that the "one" node connected to the "other" node
func (self *Network) DidConnect(one, other discover.NodeID) error {
	conn, err := self.GetOrCreateConn(one, other)
	if err != nil {
		return fmt.Errorf("connection between %v and %v does not exist", one, other)
	}
	if conn.Up {
		return fmt.Errorf("%v and %v already connected", one, other)
	}
	conn.Up = true
	self.events.Send(NewEvent(conn))
	return nil
}

// DidDisconnect tracks the fact that the "one" node disconnected from the
// "other" node
func (self *Network) DidDisconnect(one, other discover.NodeID) error {
	conn, err := self.GetOrCreateConn(one, other)
	if err != nil {
		return fmt.Errorf("connection between %v and %v does not exist", one, other)
	}
	if !conn.Up {
		return fmt.Errorf("%v and %v already disconnected", one, other)
	}
	conn.Up = false
	self.events.Send(NewEvent(conn))
	return nil
}

// DidSend tracks the fact that "sender" sent a message to "receiver"
func (self *Network) DidSend(sender, receiver discover.NodeID, proto string, code uint64) error {
	msg := &Msg{
		One:      sender,
		Other:    receiver,
		Protocol: proto,
		Code:     code,
		Received: false,
	}
	self.events.Send(NewEvent(msg))
	return nil
}

// DidReceive tracks the fact that "receiver" received a message from "sender"
func (self *Network) DidReceive(sender, receiver discover.NodeID, proto string, code uint64) error {
	msg := &Msg{
		One:      sender,
		Other:    receiver,
		Protocol: proto,
		Code:     code,
		Received: true,
	}
	self.events.Send(NewEvent(msg))
	return nil
}

// GetNode gets the node with the given ID, returning nil if the node does not
// exist
func (self *Network) GetNode(id discover.NodeID) *Node {
	self.lock.Lock()
	defer self.lock.Unlock()
	return self.getNode(id)
}

// GetNode gets the node with the given name, returning nil if the node does
// not exist
func (self *Network) GetNodeByName(name string) *Node {
	self.lock.Lock()
	defer self.lock.Unlock()
	return self.getNodeByName(name)
}

func (self *Network) getNode(id discover.NodeID) *Node {
	i, found := self.nodeMap[id]
	if !found {
		return nil
	}
	return self.Nodes[i]
}

func (self *Network) getNodeByName(name string) *Node {
	for _, node := range self.Nodes {
		if node.Config.Name == name {
			return node
		}
	}
	return nil
}

// GetNodes returns the existing nodes
func (self *Network) GetNodes() []*Node {
	self.lock.Lock()
	defer self.lock.Unlock()
	return self.Nodes
}

// GetConn returns the connection which exists between "one" and "other"
// regardless of which node initiated the connection
func (self *Network) GetConn(oneID, otherID discover.NodeID) *Conn {
	self.lock.Lock()
	defer self.lock.Unlock()
	return self.getConn(oneID, otherID)
}

// GetOrCreateConn is like GetConn but creates the connection if it doesn't
// already exist
func (self *Network) GetOrCreateConn(oneID, otherID discover.NodeID) (*Conn, error) {
	self.lock.Lock()
	defer self.lock.Unlock()
	if conn := self.getConn(oneID, otherID); conn != nil {
		return conn, nil
	}

	one := self.getNode(oneID)
	if one == nil {
		return nil, fmt.Errorf("node %v does not exist", oneID)
	}
	other := self.getNode(otherID)
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
	self.connMap[label] = len(self.Conns)
	self.Conns = append(self.Conns, conn)
	return conn, nil
}

func (self *Network) getConn(oneID, otherID discover.NodeID) *Conn {
	label := ConnLabel(oneID, otherID)
	i, found := self.connMap[label]
	if !found {
		return nil
	}
	return self.Conns[i]
}

// Shutdown stops all nodes in the network and closes the quit channel
func (self *Network) Shutdown() {
	for _, node := range self.Nodes {
		log.Debug(fmt.Sprintf("stopping node %s", node.ID().TerminalString()))
		if err := node.Stop(); err != nil {
			log.Warn(fmt.Sprintf("error stopping node %s", node.ID().TerminalString()), "err", err)
		}
	}
	close(self.quitc)
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
func (self *Node) ID() discover.NodeID {
	return self.Config.ID
}

// String returns a log-friendly string
func (self *Node) String() string {
	return fmt.Sprintf("Node %v", self.ID().TerminalString())
}

// NodeInfo returns information about the node
func (self *Node) NodeInfo() *p2p.NodeInfo {
	// avoid a panic if the node is not started yet
	if self.Node == nil {
		return nil
	}
	info := self.Node.NodeInfo()
	info.Name = self.Config.Name
	return info
}

// MarshalJSON implements the json.Marshaler interface so that the encoded
// JSON includes the NodeInfo
func (self *Node) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Info   *p2p.NodeInfo        `json:"info,omitempty"`
		Config *adapters.NodeConfig `json:"config,omitempty"`
		Up     bool                 `json:"up"`
	}{
		Info:   self.NodeInfo(),
		Config: self.Config,
		Up:     self.Up,
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

	one   *Node
	other *Node
}

// nodesUp returns whether both nodes are currently up
func (self *Conn) nodesUp() error {
	if !self.one.Up {
		return fmt.Errorf("one %v is not up", self.One)
	}
	if !self.other.Up {
		return fmt.Errorf("other %v is not up", self.Other)
	}
	return nil
}

// String returns a log-friendly string
func (self *Conn) String() string {
	return fmt.Sprintf("Conn %v->%v", self.One.TerminalString(), self.Other.TerminalString())
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
func (self *Msg) String() string {
	return fmt.Sprintf("Msg(%d) %v->%v", self.Code, self.One.TerminalString(), self.Other.TerminalString())
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
func (self *Network) Snapshot() (*Snapshot, error) {
	self.lock.Lock()
	defer self.lock.Unlock()
	snap := &Snapshot{
		Nodes: make([]NodeSnapshot, len(self.Nodes)),
		Conns: make([]Conn, len(self.Conns)),
	}
	for i, node := range self.Nodes {
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
	for i, conn := range self.Conns {
		snap.Conns[i] = *conn
	}
	return snap, nil
}

// Load loads a network snapshot
func (self *Network) Load(snap *Snapshot) error {
	for _, n := range snap.Nodes {
		if _, err := self.NewNodeWithConfig(n.Node.Config); err != nil {
			return err
		}
		if !n.Node.Up {
			continue
		}
		if err := self.startWithSnapshots(n.Node.Config.ID, n.Snapshots); err != nil {
			return err
		}
	}
	for _, conn := range snap.Conns {
		if err := self.Connect(conn.One, conn.Other); err != nil {
			return err
		}
	}
	return nil
}

// Subscribe reads control events from a channel and executes them
func (self *Network) Subscribe(events chan *Event) {
	for {
		select {
		case event, ok := <-events:
			if !ok {
				return
			}
			if event.Control {
				self.executeControlEvent(event)
			}
		case <-self.quitc:
			return
		}
	}
}

func (self *Network) executeControlEvent(event *Event) {
	log.Trace("execute control event", "type", event.Type, "event", event)
	switch event.Type {
	case EventTypeNode:
		if err := self.executeNodeEvent(event); err != nil {
			log.Error("error executing node event", "event", event, "err", err)
		}
	case EventTypeConn:
		if err := self.executeConnEvent(event); err != nil {
			log.Error("error executing conn event", "event", event, "err", err)
		}
	case EventTypeMsg:
		log.Warn("ignoring control msg event")
	}
}

func (self *Network) executeNodeEvent(e *Event) error {
	if !e.Node.Up {
		return self.Stop(e.Node.ID())
	}

	if _, err := self.NewNodeWithConfig(e.Node.Config); err != nil {
		return err
	}
	return self.Start(e.Node.ID())
}

func (self *Network) executeConnEvent(e *Event) error {
	if e.Conn.Up {
		return self.Connect(e.Conn.One, e.Conn.Other)
	} else {
		return self.Disconnect(e.Conn.One, e.Conn.Other)
	}
}
