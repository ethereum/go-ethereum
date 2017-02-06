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
	"fmt"
	"reflect"
	"sync"

	"github.com/ethereum/go-ethereum/event"
	"github.com/ethereum/go-ethereum/logger/glog"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/p2p/adapters"
	"github.com/ethereum/go-ethereum/p2p/discover"
)

type NetworkQuery struct {
	Type string
}

type NetworkConfig struct {
	// Type   NetworkType
	// Config json.RawMessage // type-specific configs
	// type
	// Events []string
	Id string
}

// event types related to connectivity, i.e., nodes coming on dropping off
// and connections established and dropped
var ConnectivityEvents = []interface{}{&NodeEvent{}, &ConnEvent{}, &MsgEvent{}}

// NewNetworkController creates a ResourceController responding to GET and DELETE methods
// it embeds a mockers controller, a journal player, node and connection contollers.
//
// Events from the eventer go into the provided journal. The content of the journal can be
// accessed through the HTTP API.
func NewNetworkController(conf *NetworkConfig, eventer *event.TypeMux, journal *Journal) Controller {

	self := NewResourceContoller(
		&ResourceHandlers{
			// GET /<networkId>/
			Retrieve: &ResourceHandler{
				Handle: func(msg interface{}, parent *ResourceController) (interface{}, error) {
					glog.V(6).Infof("msg: %v", msg)
					cyConfig, ok := msg.(*CyConfig)
					if ok {
						return UpdateCy(cyConfig, journal)
					}
					snapshotConfig, ok := msg.(*SnapshotConfig)
					if ok {
						return Snapshot(snapshotConfig, journal)
					}
					return nil, fmt.Errorf("invalId json body: must be CyConfig or SnapshotConfig")
				},
				Type: reflect.TypeOf(&CyConfig{}),
			},
			// DELETE /<networkId>/
			Destroy: &ResourceHandler{
				Handle: func(msg interface{}, parent *ResourceController) (interface{}, error) {
					parent.DeleteResource(conf.Id)
					return nil, nil
				},
			},
		},
	)
	// subscribe to all event entries (generated)
	journal.Subscribe(eventer, ConnectivityEvents...)
	// self.SetResource("nodes", NewNodesController(eventer))
	// self.SetResource("connections", NewConnectionsController(eventer))
	self.SetResource("mockevents", NewMockersController(eventer))
	self.SetResource("journals", NewJournalPlayersController(eventer))
	return Controller(self)
}

// Network models a p2p network
// the actual logic of bringing nodes and connections up and down and
// messaging is implemented in the particular NodeAdapter interface
type Network struct {
	// input trigger events and other events
	triggers *event.TypeMux // event triggers
	events   *event.TypeMux // events
	lock     sync.RWMutex
	nodeMap  map[discover.NodeID]int
	connMap  map[string]int
	Nodes    []*Node `json:"nodes"`
	Conns    []*Conn `json:"conns"`
	messenger	func(p2p.MsgReadWriter) adapters.Messenger
	//
	// adapters.Messenger
	// node adapter function that creates the node model for
	// the particular type of network from a config
	naf func(*NodeConfig) adapters.NodeAdapter
}

func NewNetwork(triggers, events *event.TypeMux) *Network {
	return &Network{
		triggers: triggers,
		events:   events,
		nodeMap:  make(map[discover.NodeID]int),
		connMap:  make(map[string]int),
		messenger: adapters.NewSimPipe,
	}
}

func (self *Network) SetNaf(naf func(*NodeConfig) adapters.NodeAdapter) {
	self.naf = naf
}

// Events returns the output eventer of the Network.
func (self *Network) Events() *event.TypeMux {
	return self.events
}

type Node struct {
	Id     *adapters.NodeId `json:"id"`
	Up     bool
	config *NodeConfig
	na     adapters.NodeAdapter
}

func (self *Node) Adapter() adapters.NodeAdapter {
	return self.na
}

func (self *Node) String() string {
	return fmt.Sprintf("Node %v", self.Id.Label())
}

type NodeEvent struct {
	Action string
	Type   string
	node   *Node
}

type ConnEvent struct {
	Action string
	Type   string
	conn   *Conn
}

type MsgEvent struct {
	Action string
	Type   string
	msg    *Msg
}

func (self *ConnEvent) String() string {
	return fmt.Sprintf("<Action: %v, Type: %v, Data: %v>\n", self.Action, self.Type, self.conn)
}

func (self *NodeEvent) String() string {
	return fmt.Sprintf("<Action: %v, Type: %v, Data: %v>\n", self.Action, self.Type, self.node)
}

func (self *MsgEvent) String() string {
	return fmt.Sprintf("<Action: %v, Type: %v, Data: %v>\n", self.Action, self.Type, self.msg)
}

func (self *Node) event(up bool) *NodeEvent {
	var action string
	if up {
		action = "up"
	} else {
		action = "down"
	}
	return &NodeEvent{
		Action: action,
		Type:   "node",
		node:   self,
	}
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
}

func (self *Conn) String() string {
	return fmt.Sprintf("Conn %v->%v", self.One.Label(), self.Other.Label())
}

func (self *Conn) event(up, rev bool) *ConnEvent {
	var action string
	if up {
		action = "up"
	} else {
		action = "down"
	}
	return &ConnEvent{
		Action: action,
		Type:   "conn",
		conn:   self,
	}
}

type Msg struct {
	One   *adapters.NodeId `json:"one"`
	Other *adapters.NodeId `json:"other"`
	Code  uint64		   `json:"conn"`
}

func (self *Msg) String() string {
	return fmt.Sprintf("Msg(%d) %v->%v", self.Code, self.One.Label(), self.Other.Label())
}

func (self *Msg) event() *MsgEvent {
	return &MsgEvent{
		Action: "up",
		//Type:   fmt.Sprintf("%d", self.Code),
		Type:   "msg",
		msg:    self,
	}
}

type NodeConfig struct {
	Id *adapters.NodeId `json:"Id"`
}

// TODO: ignored for now
type QueryConfig struct {
	Format string // "cy.update", "journal",
}

type Know struct {
	Subject *adapters.NodeId `json:"subject"`
	Object  *adapters.NodeId `json:"object"`
	// Into
	// number of attempted connections
	// time of attempted connections
	// number of active connections during the session
	// number of active connections since records began
	// swap balance
}

// NewNode adds a new node to the network
// errors if a node by the same id already exist
func (self *Network) NewNode(conf *NodeConfig) error {
	self.lock.Lock()
	defer self.lock.Unlock()
	id := conf.Id

	_, found := self.nodeMap[id.NodeID]
	if found {
		return fmt.Errorf("node %v already added", id)
	}
	self.nodeMap[id.NodeID] = len(self.Nodes)
	na := self.naf(conf)
	node := &Node{
		Id:     conf.Id,
		config: conf,
		na:     na,
	}
	self.Nodes = append(self.Nodes, node)
	glog.V(6).Infof("node %v created", id)
	return nil
}

func (self *Network) NewGenericSimNode(conf *NodeConfig) adapters.NodeAdapter {
	id := conf.Id
	na := adapters.NewSimNode(id, self, self.messenger)
	return na
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

// sa := node.Adapter()
// err := sa.Stop()
// if err != nil {
// 	return err
// }

// Start(id) starts up the node (relevant only for instance with own p2p or remote)
func (self *Network) Start(id *adapters.NodeId) error {
	node := self.GetNode(id)
	if node == nil {
		return fmt.Errorf("node %v does not exist", id)
	}
	if node.Up {
		return fmt.Errorf("node %v already up", id)
	}
	glog.V(6).Infof("starting node %v: %v adapter %v", id, node.Up, node.Adapter())
	sa, ok := node.Adapter().(adapters.StartAdapter)
	if ok {
		err := sa.Start()
		if err != nil {
			return err
		}
	}
	node.Up = true
	glog.V(6).Infof("started node %v: %v", id, node.Up)

	self.events.Post(&NodeEvent{
		Action: "up",
		Type:   "node",
		node:   node,
	})
	return nil
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
	sa, ok := node.Adapter().(adapters.StartAdapter)
	if ok {
		err := sa.Stop()
		if err != nil {
			return err
		}
	}
	node.Up = false
	self.events.Post(&NodeEvent{
		Action: "down",
		Type:   "node",
		node:   node,
	})
	return nil
}

// Connect(i, j) attempts to connect nodes i and j (args given as nodeId)
// calling the node's nodadapters Connect method
// connection is established (as if) the first node dials out to the other
func (self *Network) Connect(oneId, otherId *adapters.NodeId) error {
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
	if rev {
		err = conn.other.na.Connect(oneId.Bytes())
	} else {
		err = conn.one.na.Connect(otherId.Bytes())
	}
	if err != nil {
		return err
	}
	return nil
	// return self.DidConnect(oneId, otherId)
}

// Disconnect(i, j) attempts to disconnect nodes i and j (args given as nodeId)
// calling the node's nodadapters Disconnect method
// sets the Conn model to Down
// the disconnect will be initiated (the connection is dropped by) the first node
// it errors if either of the nodes is down (or does not exist)
func (self *Network) Disconnect(oneId, otherId *adapters.NodeId, disconnect bool) error {
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
	// if Disconnect is externally triggered one needs to call the actual
	// adapter's disconnect method
	if disconnect {
		var err error
		if rev {
			err = conn.other.na.Disconnect(oneId.Bytes())
		} else {
			err = conn.one.na.Disconnect(otherId.Bytes())
		}
		if err != nil {
			return err
		}
	}
	return nil
	// return self.DidDisconnect(oneId, otherId)
}

func (self *Network) DidConnect(one, other *adapters.NodeId) error {
	conn := self.GetConn(one, other)
	if conn == nil {
		return fmt.Errorf("connection between %v and %v does not exist", one, other)
	}
	if conn.Up {
		return fmt.Errorf("%v and %v already connected", one, other)
	}
	conn.Reverse = conn.One.NodeID != one.NodeID
	conn.Up = true
	// connection event posted
	self.events.Post(conn.event(true, conn.Reverse))
	return nil
}

func (self *Network) DidDisconnect(one, other *adapters.NodeId) error {
	conn := self.GetConn(one, other)
	if conn == nil {
		return fmt.Errorf("connection between %v and %v does not exist", one, other)
	}
	if !conn.Up {
		return fmt.Errorf("%v and %v already disconnected", one, other)
	}
	conn.Reverse = conn.One.NodeID != one.NodeID
	conn.Up = false
	self.events.Post(conn.event(false, conn.Reverse))
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
	self.events.Post(msg.event())                                                                // should also include send status maybe
}

// GetNodeAdapter(id) returns the NodeAdapter for node with id
// returns nil if node does not exist
func (self *Network) GetNodeAdapter(id *adapters.NodeId) adapters.NodeAdapter {
	self.lock.Lock()
	defer self.lock.Unlock()
	node := self.getNode(id)
	if node == nil {
		return nil
	}
	return node.na
}

// GetNode retrieves the node model for the id given as arg
// returns nil if the node does not exist
func (self *Network) GetNode(id *adapters.NodeId) *Node {
	self.lock.Lock()
	defer self.lock.Unlock()
	return self.getNode(id)
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
