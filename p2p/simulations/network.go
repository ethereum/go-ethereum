package simulations

import (
	"fmt"
	"reflect"
	"sync"

	"github.com/ethereum/go-ethereum/event"
	"github.com/ethereum/go-ethereum/logger/glog"
	"github.com/ethereum/go-ethereum/p2p/adapters"
	"github.com/ethereum/go-ethereum/p2p/discover"
)

const lablen = 4

type NetworkConfig struct {
	// Type   NetworkType
	// Config json.RawMessage // type-specific configs
	// type
	// Events []string
	Id string
}

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
					return nil, fmt.Errorf("invalid json body: must be CyConfig or SnapshotConfig")
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
	journal.Subscribe(eventer, &Entry{})
	// self.SetResource("nodes", NewNodesController(eventer))
	// self.SetResource("connections", NewConnectionsController(eventer))
	self.SetResource("mockevents", NewMockersController(eventer))
	self.SetResource("journals", NewJournalPlayersController(eventer))
	return Controller(self)
}

// Network
// this can be the hook for uptime
type Network struct {
	adapters.Messenger
	lock    sync.RWMutex
	NodeMap map[discover.NodeID]int
	Nodes   []*SimNode
}

func NewNetwork(m adapters.Messenger) *Network {
	return &Network{
		Messenger: m,
		NodeMap:   make(map[discover.NodeID]int),
	}
}

type SimNode struct {
	ID         *discover.NodeID
	config     *NodeConfig
	NetAdapter adapters.NetAdapter
}

func (self *SimNode) String() string {
	return fmt.Sprintf("SimNode %v", self.ID.String()[0:lablen])
}

func (self *SimConn) String() string {
	return fmt.Sprintf("SimConn %v->%v", self.Caller.String()[0:lablen], self.Callee.String()[0:lablen])
}

// active connections are represented by the SimNode entry object so that
// you journal updates could filter if passive knowledge about peers is
// irrelevant
type SimConn struct {
	Caller         *discover.NodeID `json:"caller"`
	Callee         *discover.NodeID `json:"callee"`
	caller, callee *SimNode
	// Info
	// active connection
	// average throughput, recent average throughput
}

type NodeConfig struct {
	ID  *discover.NodeID
	Run func(adapters.NetAdapter, adapters.Messenger) adapters.ProtoCall
}

func (self *Network) Protocol(id *discover.NodeID) adapters.ProtoCall {
	self.lock.Lock()
	defer self.lock.Unlock()
	node := self.getNode(id)
	if node == nil {
		return nil
	}
	na := node.NetAdapter.(*adapters.SimNet)
	return na.Run
}

// TODO: ignored for now
type QueryConfig struct {
	Format string // "cy.update", "journal",
}

type Know struct {
	Subject *discover.NodeID `json:"subject"`
	Object  *discover.NodeID `json:"object"`
	// Into
	// number of attempted connections
	// time of attempted connections
	// number of active connections during the session
	// number of active connections since records began
	// swap balance
}

func (self *Network) StartNode(conf *NodeConfig) error {
	self.lock.Lock()
	defer self.lock.Unlock()
	id := conf.ID

	_, found := self.NodeMap[*id]
	if found {
		return fmt.Errorf("node %v already running", id)
	}
	simnet := adapters.NewSimNet(id, self, self)
	if conf.Run != nil {
		simnet.Run = conf.Run(simnet, self.Messenger)
	}
	self.NodeMap[*id] = len(self.Nodes)
	self.Nodes = append(self.Nodes, &SimNode{id, conf, simnet})
	return nil
}

func (self *Network) GetNode(id *discover.NodeID) *SimNode {
	self.lock.Lock()
	defer self.lock.Unlock()
	return self.getNode(id)
}

func (self *Network) getNode(id *discover.NodeID) *SimNode {
	i, found := self.NodeMap[*id]
	if !found {
		return nil
	}
	return self.Nodes[i]
}
