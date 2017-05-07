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
	"context"
	"fmt"
	"sync"

	"github.com/ethereum/go-ethereum/event"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/p2p/discover"
	"github.com/ethereum/go-ethereum/p2p/simulations/adapters"
	"github.com/ethereum/go-ethereum/rpc"
)

type NetworkConfig struct {
	// Type   NetworkType
	// Config json.RawMessage // type-specific configs
	// type
	// Events []string
	Id                  string
	DefaultMockerConfig *MockerConfig
	Backend             bool
	DefaultService      string
}

type NetworkControl interface {
	Events() *event.TypeMux
	Config() *NetworkConfig
	Subscribe(*event.TypeMux, ...interface{})
}

// Network models a p2p network
// the actual logic of bringing nodes and connections up and down and
// messaging is implemented in the particular NodeAdapter interface
type Network struct {
	nodeAdapter adapters.NodeAdapter

	// input trigger events and other events
	events  event.Feed // generated events a journal can subsribe to
	lock    sync.RWMutex
	nodeMap map[discover.NodeID]int
	connMap map[string]int
	Nodes   []*Node `json:"nodes"`
	Conns   []*Conn `json:"conns"`
	quitc   chan bool
	conf    *NetworkConfig
}

func NewNetwork(nodeAdapter adapters.NodeAdapter, conf *NetworkConfig) *Network {
	return &Network{
		nodeAdapter: nodeAdapter,
		conf:        conf,
		nodeMap:     make(map[discover.NodeID]int),
		connMap:     make(map[string]int),
		quitc:       make(chan bool),
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

	if err := self.NewNodeWithConfig(e.Node.Config); err != nil {
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

func (self *Node) ID() *adapters.NodeId {
	return self.Config.Id
}

func (self *Node) String() string {
	return fmt.Sprintf("Node %v", self.ID().Label())
}

// active connections are represented by the Node entry object so that
// you journal updates could filter if passive knowledge about peers is
// irrelevant
type Conn struct {
	One        *adapters.NodeId `json:"one"`
	Other      *adapters.NodeId `json:"other"`
	one, other *Node
	// connection down by default
	Up bool `json:"up"`
	// reverse is false by default (One dialled/dropped the Other)
	Reverse bool `json:"reverse"`
	// Info
	// average throughput, recent average throughput etc
	controlFired bool
}

func (self *Conn) String() string {
	return fmt.Sprintf("Conn %v->%v", self.One.Label(), self.Other.Label())
}

type Msg struct {
	One          *adapters.NodeId `json:"one"`
	Other        *adapters.NodeId `json:"other"`
	Code         uint64           `json:"code"`
	Received     bool             `json:"received"`
	controlFired bool
}

func (self *Msg) String() string {
	return fmt.Sprintf("Msg(%d) %v->%v", self.Code, self.One.Label(), self.Other.Label())
}

// NewNode adds a new node to the network with a random ID
func (self *Network) NewNode() (*adapters.NodeConfig, error) {
	conf := adapters.RandomNodeConfig()
	conf.Service = self.conf.DefaultService
	if err := self.NewNodeWithConfig(conf); err != nil {
		return nil, err
	}
	return conf, nil
}

// NewNodeWithConfig adds a new node to the network with the given config
// errors if a node by the same id already exist
func (self *Network) NewNodeWithConfig(conf *adapters.NodeConfig) error {
	self.lock.Lock()
	defer self.lock.Unlock()
	id := conf.Id
	if conf.Service == "" {
		conf.Service = self.conf.DefaultService
	}

	_, found := self.nodeMap[id.NodeID]
	if found {
		return fmt.Errorf("node %v already added", id)
	}
	self.nodeMap[id.NodeID] = len(self.Nodes)

	adapterNode, err := self.nodeAdapter.NewNode(conf)
	if err != nil {
		return err
	}
	node := &Node{
		Node:   adapterNode,
		Config: conf,
	}
	self.Nodes = append(self.Nodes, node)
	log.Trace(fmt.Sprintf("node %v created", id))
	self.events.Send(ControlEvent(node))
	return nil
}

func (self *Network) Config() *NetworkConfig {
	return self.conf
}

// newConn adds a new connection to the network
// it errors if the respective nodes do not exist
func (self *Network) newConn(oneId, otherId *adapters.NodeId) (*Conn, error) {
	one := self.getNode(oneId)
	if one == nil {
		return nil, fmt.Errorf("one %v does not exist", one)
	}
	other := self.getNode(otherId)
	if other == nil {
		return nil, fmt.Errorf("other %v does not exist", other)
	}
	return &Conn{
		One:   oneId,
		Other: otherId,
		one:   one,
		other: other,
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

// Start(id) starts up the node (relevant only for instance with own p2p or remote)
func (self *Network) Start(id *adapters.NodeId) error {
	node := self.GetNode(id)
	if node == nil {
		return fmt.Errorf("node %v does not exist", id)
	}
	if node.Up {
		return fmt.Errorf("node %v already up", id)
	}
	log.Trace(fmt.Sprintf("starting node %v: %v using %v", id, node.Up, self.nodeAdapter.Name()))
	if err := node.Start(); err != nil {
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

func (self *Network) watchPeerEvents(id *adapters.NodeId, events chan *p2p.PeerEvent, sub event.Subscription) {
	defer sub.Unsubscribe()
	for {
		select {
		case event, ok := <-events:
			if !ok {
				return
			}
			peer := &adapters.NodeId{NodeID: event.Peer}
			switch event.Type {
			case p2p.PeerEventTypeAdd:
				if err := self.DidConnect(id, peer); err != nil {
					log.Error(fmt.Sprintf("error generating connection up event %s => %s", id.Label(), peer.Label()), "err", err)
				}
			case p2p.PeerEventTypeDrop:
				if err := self.DidDisconnect(id, peer); err != nil {
					log.Error(fmt.Sprintf("error generating connection down event %s => %s", id.Label(), peer.Label()), "err", err)
				}
			case p2p.PeerEventTypeMsgSend:
				if err := self.DidSend(id, peer, *event.MsgCode); err != nil {
					log.Error(fmt.Sprintf("error generating msg send event %s => %s", id.Label(), peer.Label()), "err", err)
				}
			case p2p.PeerEventTypeMsgRecv:
				if err := self.DidReceive(peer, id, *event.MsgCode); err != nil {
					log.Error(fmt.Sprintf("error generating msg receive event %s => %s", peer.Label(), id.Label()), "err", err)
				}
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
func (self *Network) Stop(id *adapters.NodeId) error {
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

// Connect(i, j) attempts to connect nodes i and j (args given as nodeId)
// calling the node's nodadapters Connect method
// connection is established (as if) the first node dials out to the other
func (self *Network) Connect(oneId, otherId *adapters.NodeId) error {
	log.Debug(fmt.Sprintf("connecting %s to %s", oneId, otherId))
	conn, err := self.GetOrCreateConn(oneId, otherId)
	if err != nil {
		return err
	}
	if conn.Up {
		return fmt.Errorf("%v and %v already connected", oneId, otherId)
	}
	err = conn.nodesUp()
	if err != nil {
		return err
	}
	var rev bool
	if conn.One.NodeID != oneId.NodeID {
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

// Disconnect(i, j) attempts to disconnect nodes i and j (args given as nodeId)
// calling the node's nodadapters Disconnect method
// sets the Conn model to Down
// the disconnect will be initiated (the connection is dropped by) the first node
// it errors if either of the nodes is down (or does not exist)
func (self *Network) Disconnect(oneId, otherId *adapters.NodeId) error {
	conn := self.GetConn(oneId, otherId)
	if conn == nil {
		return fmt.Errorf("connection between %v and %v does not exist", oneId, otherId)
	}
	if !conn.Up {
		return fmt.Errorf("%v and %v already disconnected", oneId, otherId)
	}
	var rev bool
	if conn.One.NodeID != oneId.NodeID {
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

func (self *Network) DidConnect(one, other *adapters.NodeId) error {
	conn, err := self.GetOrCreateConn(one, other)
	if err != nil {
		return fmt.Errorf("connection between %v and %v does not exist", one, other)
	}
	if conn.Up {
		return fmt.Errorf("%v and %v already connected", one, other)
	}
	conn.Reverse = conn.One.NodeID != one.NodeID
	conn.Up = true
	// connection event posted
	self.events.Send(NewEvent(conn))
	return nil
}

func (self *Network) DidDisconnect(one, other *adapters.NodeId) error {
	conn, err := self.GetOrCreateConn(one, other)
	if err != nil {
		return fmt.Errorf("connection between %v and %v does not exist", one, other)
	}
	if !conn.Up {
		return fmt.Errorf("%v and %v already disconnected", one, other)
	}
	conn.Reverse = conn.One.NodeID != one.NodeID
	conn.Up = false
	self.events.Send(NewEvent(conn))
	return nil
}

// Send(senderid, receiverid) sends a message from one node to another
func (self *Network) Send(senderid, receiverid *adapters.NodeId, msgcode uint64, protomsg interface{}) {
	msg := &Msg{
		One:   senderid,
		Other: receiverid,
		Code:  msgcode,
	}
	//self.GetNode(senderid).na.(*adapters.SimNode).GetPeer(receiverid).SendMsg(msgcode, protomsg) // phew!
	self.events.Send(ControlEvent(msg))
}

func (self *Network) DidSend(sender, receiver *adapters.NodeId, msgcode uint64) error {
	msg := &Msg{
		One:      sender,
		Other:    receiver,
		Code:     msgcode,
		Received: false,
	}
	self.events.Send(NewEvent(msg))
	return nil
}

func (self *Network) DidReceive(sender, receiver *adapters.NodeId, msgcode uint64) error {
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
func (self *Network) GetNode(id *adapters.NodeId) *Node {
	self.lock.Lock()
	defer self.lock.Unlock()
	return self.getNode(id)
}

func (self *Network) GetNodes() []*Node {
	self.lock.Lock()
	defer self.lock.Unlock()
	return self.Nodes
}

func (self *Network) getNode(id *adapters.NodeId) *Node {
	i, found := self.nodeMap[id.NodeID]
	if !found {
		return nil
	}
	return self.Nodes[i]
}

// GetConn(i, j) retrieves the connectiton model for the connection between
// the order of nodes does not matter, i.e., GetConn(i,j) == GetConn(j, i)
// returns nil if the node does not exist
func (self *Network) GetConn(oneId, otherId *adapters.NodeId) *Conn {
	self.lock.Lock()
	defer self.lock.Unlock()
	return self.getConn(oneId, otherId)
}

// GetConn(i, j) retrieves the connectiton model for the connection between
// i and j, or creates a new one if it does not exist
// the order of nodes does not matter, i.e., GetConn(i,j) == GetConn(j, i)
func (self *Network) GetOrCreateConn(oneId, otherId *adapters.NodeId) (*Conn, error) {
	self.lock.Lock()
	defer self.lock.Unlock()
	conn := self.getConn(oneId, otherId)
	if conn != nil {
		return conn, nil
	}
	conn, err := self.newConn(oneId, otherId)
	if err != nil {
		return nil, err
	}
	label := ConnLabel(oneId, otherId)
	self.connMap[label] = len(self.Conns)
	self.Conns = append(self.Conns, conn)
	return conn, nil
}

func (self *Network) getConn(oneId, otherId *adapters.NodeId) *Conn {
	label := ConnLabel(oneId, otherId)
	i, found := self.connMap[label]
	if !found {
		return nil
	}
	return self.Conns[i]
}

func (self *Network) Shutdown() {
	// disconnect all nodes
	for _, conn := range self.Conns {
		log.Debug(fmt.Sprintf("disconnecting %s from %s", conn.One.Label(), conn.Other.Label()))
		if err := self.Disconnect(conn.One, conn.Other); err != nil {
			log.Warn(fmt.Sprintf("error disconnecting %s from %s", conn.One.Label(), conn.Other.Label()), "err", err)
		}
	}

	// stop all nodes
	for _, node := range self.Nodes {
		log.Debug(fmt.Sprintf("stopping node %s", node.ID().Label()))
		if err := node.Stop(); err != nil {
			log.Warn(fmt.Sprintf("error stopping node %s", node.ID().Label()), "err", err)
		}
	}
}
