// Package simulations simulates p2p networks.
//
// Network
//  - has nodes
//  - has connections
//  - has triggers (input eventer, triggers things like start and stop of nodes, connecting them)
//  - has output eventer, where stuff that happens during simulation is sent
//  - the adapter of new nodes is assigned by the Node Adapter Function.
//
// Sources of Trigger events
// - UI (click of button)
// - Journal (replay captured events)
// - Mocker (generate random events)
//
// Adapters
// - each node has an adapter
// - contains methods to connect to another node using the same adapter type
// - models communication too (sending and receiving messages)
//
// REST API
// - Session Controller: handles Networks
// - Network Controller
//   - handles one Network
//   - has sub controller for triggering events
//   - get output events
//
package simulations

import (
	"bytes"
	"context"
	"fmt"
	"sync"

	"github.com/ethereum/go-ethereum/event"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/p2p/discover"
	"github.com/ethereum/go-ethereum/p2p/simulations/adapters"
	"github.com/ethereum/go-ethereum/pot"
	"github.com/ethereum/go-ethereum/rpc"
)

type NetworkConfig struct {
	ID             string `json:"id"`
	DefaultService string `json:"default_service,omitempty"`
}

// Network models a p2p network
// the actual logic of bringing nodes and connections up and down and
// messaging is implemented in the particular NodeAdapter interface
type Network struct {
	NetworkConfig

	Nodes []*Node `json:"nodes"`
	Conns []*Conn `json:"conns"`

	nodeAdapter adapters.NodeAdapter

	// input trigger events and other events
	events  event.Feed // generated events a journal can subsribe to
	lock    sync.RWMutex
	nodeMap map[discover.NodeID]int
	connMap map[string]int
	quitc   chan bool
}

func NewNetwork(nodeAdapter adapters.NodeAdapter, conf *NetworkConfig) *Network {
	return &Network{
		NetworkConfig: *conf,
		nodeAdapter:   nodeAdapter,
		nodeMap:       make(map[discover.NodeID]int),
		connMap:       make(map[string]int),
		quitc:         make(chan bool),
	}
}

// Subscribe reads control events from a channel and executes them
func (self *Network) Subscribe(events chan *Event) {
	log.Info("subscribe")
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

// Events returns the output eventer of the Network.
func (self *Network) Events() *event.Feed {
	return &self.events
}

type Node struct {
	adapters.Node `json:"-"`

	Config *adapters.NodeConfig `json:"config"`
	Up     bool                 `json:"up"`

	controlFired bool
}

func (self *Node) ID() discover.NodeID {
	return self.Config.ID
}

func (self *Node) String() string {
	return fmt.Sprintf("Node %v", self.ID().TerminalString())
}

func (self *Node) NodeInfo() *p2p.NodeInfo {
	info := self.Node.NodeInfo()
	info.Name = self.Config.Name
	return info
}

// active connections are represented by the Node entry object so that
// you journal updates could filter if passive knowledge about peers is
// irrelevant
type Conn struct {
	One        discover.NodeID `json:"one"`
	Other      discover.NodeID `json:"other"`
	one, other *Node
	// connection down by default
	Up bool `json:"up"`
	// reverse is false by default (One dialled/dropped the Other)
	Reverse bool `json:"reverse"`
	// A scalar distance value denoting how "far" Other is from One (Kademlia table)
	Distance int `json:"distance"`
	// indicates if a ControlEvent has already been fired for this connection
	controlFired bool
}

func (self *Conn) String() string {
	return fmt.Sprintf("Conn %v->%v", self.One.TerminalString(), self.Other.TerminalString())
}

type Msg struct {
	One          discover.NodeID `json:"one"`
	Other        discover.NodeID `json:"other"`
	Protocol     string          `json:"protocol"`
	Code         uint64          `json:"code"`
	Received     bool            `json:"received"`
	controlFired bool
}

func (self *Msg) String() string {
	return fmt.Sprintf("Msg(%d) %v->%v", self.Code, self.One.TerminalString(), self.Other.TerminalString())
}

// NewNode adds a new node to the network with a random ID
func (self *Network) NewNode() (*Node, error) {
	conf := adapters.RandomNodeConfig()
	conf.Services = []string{self.DefaultService}
	return self.NewNodeWithConfig(conf)
}

// NewNodeWithConfig adds a new node to the network with the given config
// errors if a node by the same id already exist
func (self *Network) NewNodeWithConfig(conf *adapters.NodeConfig) (*Node, error) {
	self.lock.Lock()
	defer self.lock.Unlock()
	if conf.ID == (discover.NodeID{}) {
		c := adapters.RandomNodeConfig()
		conf.ID = c.ID
		conf.PrivateKey = c.PrivateKey
	}
	id := conf.ID
	if node := self.getNode(id); node != nil {
		return nil, fmt.Errorf("node already exists: %q", id)
	}
	if conf.Name == "" {
		conf.Name = fmt.Sprintf("node%02d", len(self.Nodes)+1)
	}
	if node := self.getNodeByName(conf.Name); node != nil {
		return nil, fmt.Errorf("node already exists: %q", conf.Name)
	}
	if len(conf.Services) == 0 {
		conf.Services = []string{self.DefaultService}
	}

	_, found := self.nodeMap[id]
	if found {
		return nil, fmt.Errorf("node %v already added", id)
	}
	self.nodeMap[id] = len(self.Nodes)

	adapterNode, err := self.nodeAdapter.NewNode(conf)
	if err != nil {
		return nil, err
	}
	node := &Node{
		Node:   adapterNode,
		Config: conf,
	}
	self.Nodes = append(self.Nodes, node)
	log.Trace(fmt.Sprintf("node %v created", id))
	self.events.Send(ControlEvent(node))
	return node, nil
}

func (self *Network) Config() *NetworkConfig {
	return &self.NetworkConfig
}

// newConn adds a new connection to the network
// it errors if the respective nodes do not exist
func (self *Network) newConn(oneID, otherID discover.NodeID) (*Conn, error) {
	one := self.getNode(oneID)
	if one == nil {
		return nil, fmt.Errorf("one %v does not exist", one)
	}
	other := self.getNode(otherID)
	if other == nil {
		return nil, fmt.Errorf("other %v does not exist", other)
	}
	distance, _ := pot.DefaultPof(256)(one.Addr(), other.Addr(), 0)
	return &Conn{
		One:      oneID,
		Other:    otherID,
		one:      one,
		other:    other,
		Distance: distance,
	}, nil
}

func (self *Conn) nodesUp() error {
	if !self.one.Up {
		return fmt.Errorf("one %v is not up", self.One)
	}
	if !self.other.Up {
		return fmt.Errorf("other %v is not up", self.Other)
	}
	return nil
}

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

// Start(id) starts up the node (relevant only for instance with own p2p or remote)
func (self *Network) Start(id discover.NodeID) error {
	return self.startWithSnapshots(id, nil)
}

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
				self.DidSend(id, peer, *event.MsgCode, event.Protocol)
			case p2p.PeerEventTypeMsgRecv:
				self.DidReceive(peer, id, *event.MsgCode)
			}
		case err := <-sub.Err():
			if err != nil {
				log.Error(fmt.Sprintf("error getting peer events for node %v", id), "err", err)
			}
			return
		}
	}
}

// Stop(id) shuts down the node (relevant only for instance with own p2p or remote)
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

	self.events.Send(NewEvent(node))
	return nil
}

// Connect(i, j) attempts to connect nodes i and j (args given as nodeID)
// calling the node's nodadapters Connect method
// connection is established (as if) the first node dials out to the other
func (self *Network) Connect(oneID, otherID discover.NodeID) error {
	log.Debug(fmt.Sprintf("connecting %s to %s", oneID, otherID))
	conn, err := self.GetOrCreateConn(oneID, otherID)
	if err != nil {
		return err
	}
	if conn.Up {
		return fmt.Errorf("%v and %v already connected", oneID, otherID)
	}
	err = conn.nodesUp()
	if err != nil {
		return err
	}
	var rev bool
	if conn.One != oneID {
		rev = true
	}
	// if Connect is called because of external trigger, it needs to call
	// the actual adaptor's connect method
	// any other way of connection (like peerpool) will need to call back
	// to this method with connect = false to avoid infinite recursion
	// this is not relevant for nodes starting up (which can only be externally triggered)
	var addr []byte
	var client *rpc.Client
	if rev {
		addr = conn.one.Addr()
		client, err = conn.other.Client()
	} else {
		addr = conn.other.Addr()
		client, err = conn.one.Client()
	}
	if err != nil {
		return err
	}
	self.events.Send(ControlEvent(conn))
	return client.Call(nil, "admin_addPeer", string(addr))
}

// Disconnect(i, j) attempts to disconnect nodes i and j (args given as nodeID)
// calling the node's nodadapters Disconnect method
// sets the Conn model to Down
// the disconnect will be initiated (the connection is dropped by) the first node
// it errors if either of the nodes is down (or does not exist)
func (self *Network) Disconnect(oneID, otherID discover.NodeID) error {
	conn := self.GetConn(oneID, otherID)
	if conn == nil {
		return fmt.Errorf("connection between %v and %v does not exist", oneID, otherID)
	}
	if !conn.Up {
		return fmt.Errorf("%v and %v already disconnected", oneID, otherID)
	}
	var rev bool
	if conn.One != oneID {
		rev = true
	}
	var addr []byte
	var client *rpc.Client
	var err error
	if rev {
		addr = conn.one.Addr()
		client, err = conn.other.Client()
	} else {
		addr = conn.other.Addr()
		client, err = conn.one.Client()
	}
	if err != nil {
		return err
	}
	self.events.Send(ControlEvent(conn))
	return client.Call(nil, "admin_removePeer", string(addr))
}

func (self *Network) DidConnect(one, other discover.NodeID) error {
	conn, err := self.GetOrCreateConn(one, other)
	if err != nil {
		return fmt.Errorf("connection between %v and %v does not exist", one, other)
	}
	if conn.Up {
		return fmt.Errorf("%v and %v already connected", one, other)
	}
	conn.Reverse = conn.One != one
	conn.Up = true
	// connection event posted
	self.events.Send(NewEvent(conn))
	return nil
}

func (self *Network) DidDisconnect(one, other discover.NodeID) error {
	conn, err := self.GetOrCreateConn(one, other)
	if err != nil {
		return fmt.Errorf("connection between %v and %v does not exist", one, other)
	}
	if !conn.Up {
		return fmt.Errorf("%v and %v already disconnected", one, other)
	}
	conn.Reverse = conn.One != one
	conn.Up = false
	self.events.Send(NewEvent(conn))
	return nil
}

// Send(senderid, receiverid) sends a message from one node to another
func (self *Network) Send(senderid, receiverid discover.NodeID, msgcode uint64, protomsg interface{}) {
	msg := &Msg{
		One:   senderid,
		Other: receiverid,
		Code:  msgcode,
	}
	//self.GetNode(senderid).na.(*adapters.SimNode).GetPeer(receiverid).SendMsg(msgcode, protomsg) // phew!
	self.events.Send(ControlEvent(msg))
}

func (self *Network) DidSend(sender, receiver discover.NodeID, msgcode uint64, msgProtocol string) error {
	msg := &Msg{
		One:      sender,
		Other:    receiver,
		Code:     msgcode,
    Protocol: msgProtocol,
		Received: false,
	}
	self.events.Send(NewEvent(msg))
	return nil
}

func (self *Network) DidReceive(sender, receiver discover.NodeID, msgcode uint64) error {
	msg := &Msg{
		One:      sender,
		Other:    receiver,
		Code:     msgcode,
		Received: true,
	}
	self.events.Send(NewEvent(msg))
	return nil
}

// GetNode retrieves the node model for the id given as arg
// returns nil if the node does not exist
func (self *Network) GetNode(id discover.NodeID) *Node {
	self.lock.Lock()
	defer self.lock.Unlock()
	return self.getNode(id)
}

func (self *Network) GetNodeByName(name string) *Node {
	self.lock.Lock()
	defer self.lock.Unlock()
	return self.getNodeByName(name)
}

func (self *Network) getNodeByName(name string) *Node {
	for _, node := range self.Nodes {
		if node.Config.Name == name {
			return node
		}
	}
	return nil
}

func (self *Network) GetNodes() []*Node {
	self.lock.Lock()
	defer self.lock.Unlock()
	return self.Nodes
}

func (self *Network) getNode(id discover.NodeID) *Node {
	i, found := self.nodeMap[id]
	if !found {
		return nil
	}
	return self.Nodes[i]
}

// GetConn(i, j) retrieves the connectiton model for the connection between
// the order of nodes does not matter, i.e., GetConn(i,j) == GetConn(j, i)
// returns nil if the node does not exist
func (self *Network) GetConn(oneID, otherID discover.NodeID) *Conn {
	self.lock.Lock()
	defer self.lock.Unlock()
	return self.getConn(oneID, otherID)
}

// GetConn(i, j) retrieves the connectiton model for the connection between
// i and j, or creates a new one if it does not exist
// the order of nodes does not matter, i.e., GetConn(i,j) == GetConn(j, i)
func (self *Network) GetOrCreateConn(oneID, otherID discover.NodeID) (*Conn, error) {
	self.lock.Lock()
	defer self.lock.Unlock()
	conn := self.getConn(oneID, otherID)
	if conn != nil {
		return conn, nil
	}
	conn, err := self.newConn(oneID, otherID)
	if err != nil {
		return nil, err
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

func (self *Network) Shutdown() {
	// stop all nodes
	for _, node := range self.Nodes {
		log.Debug(fmt.Sprintf("stopping node %s", node.ID().TerminalString()))
		if err := node.Stop(); err != nil {
			log.Warn(fmt.Sprintf("error stopping node %s", node.ID().TerminalString()), "err", err)
		}
	}
}

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
	Node

	// Snapshot is arbitrary data gathered from calling node.Snapshots()
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

func (self *Network) Load(snap *Snapshot) error {
	for _, node := range snap.Nodes {
		if _, err := self.NewNodeWithConfig(node.Config); err != nil {
			return err
		}
		if !node.Up {
			continue
		}
		if err := self.startWithSnapshots(node.Config.ID, node.Snapshots); err != nil {
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
