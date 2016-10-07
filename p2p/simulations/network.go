package simulations

import (
	"bytes"
	"fmt"
	"math/rand"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/logger/glog"
	"github.com/ethereum/go-ethereum/p2p/adapters"
	"github.com/ethereum/go-ethereum/p2p/discover"
)

const lablen = 4

func NewNetworkController(n *Network, parent *ResourceController) Controller {
	self := NewResourceContoller(
		&ResourceHandlers{
			// Destroy: n.Shutdown, nil
			// Create: n.StartNode, NodeConfig
			// Update: n.Setup, NodeConfig
			// Retrieve: n.Retrieve,
			Retrieve: &ResourceHandler{
				Handle: n.Query,
			},
		},
	)
	if parent != nil {
		parent.SetResource(fmt.Sprintf("%d", parent.id), self)
	}
	// self.SetResource("nodes", NewNodesController())
	return Controller(self)
}

// Network
// this can be the hook for uptime
type Network struct {
	adapters.Messenger
	lock    sync.RWMutex
	NodeMap map[discover.NodeID]int
	Nodes   []*SimNode
	Journal []*Entry
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

type NodeConfig struct {
	ID  *discover.NodeID
	Run func(adapters.NetAdapter, adapters.Messenger) adapters.ProtoCall
}

func Key(id []byte) string {
	return string(id)
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

type CyData struct {
	Id     string `json:"id"`
	Source string `json:"source"`
	Target string `json:"target"`
	On     bool   `json:"on"`
}

type CyElement struct {
	Data    *CyData `json:"data"`
	Classes string  `json:"classes"`
	Group   string  `json:"group"`
	// selected: false, // whether the element is selected (default false)
	// selectable: true, // whether the selection state is mutable (default true)
	// locked: false, // when locked a node's position is immutable (default false)
	// grabbable: true, // whether the node can be grabbed and moved by the user
}

type Entry struct {
	Action string      `json:"action"`
	Type   string      `json:"type"`
	Object interface{} `json:"object"`
}

func (self *Entry) Stirng() string {
	return fmt.Sprintf("<Action: %v, Type: %v, Data: %v>\n", self.Action, self.Type, self.Object)
}

func (n *Network) AppendEntries(entries ...*Entry) {
	n.lock.Lock()
	n.Journal = append(n.Journal, entries...)
	n.lock.Unlock()
}

type Know struct {
	Subject *discover.NodeID `json:"subject"`
	Object  *discover.NodeID `json:"objectr"`
	// Into
	// number of attempted connections
	// time of attempted connections
	// number of active connections during the session
	// number of active connections since records began
	// swap balance
}

// active connections are represented by the SimNode entry object so that
// you journal updates could filter if passive knowledge about peers is
// irrelevant
type SimConn struct {
	Caller *discover.NodeID `json:"caller"`
	Callee *discover.NodeID `json:"callee"`
	// Info
	// active connection
	// average throughput, recent average throughput
}

func (self *Network) CyUpdate() *CyUpdate {
	self.lock.Lock()
	defer self.lock.Unlock()
	added := []*CyElement{}
	removed := []string{}
	var el *CyElement
	for _, entry := range self.Journal {
		glog.V(6).Infof("journal entry: %v", entry)
		switch entry.Type {
		case "Node":
			el = &CyElement{Group: "nodes", Data: &CyData{Id: entry.Object.(*SimNode).ID.String()[0:lablen]}}
		case "Conn":
			// mutually exclusive directed edge (caller -> callee)
			source := entry.Object.(*SimConn).Caller.String()[0:lablen]
			target := entry.Object.(*SimConn).Callee.String()[0:lablen]
			first := source
			second := target
			if bytes.Compare([]byte(first), []byte(second)) > 1 {
				first = target
				second = source
			}
			id := fmt.Sprintf("%v-%v", first, second)
			el = &CyElement{Group: "edges", Data: &CyData{Id: id, Source: source, Target: target}}
		case "Know":
			// independent directed edge (peer0 registers peer1)
			source := entry.Object.(*Know).Subject.String()[0:lablen]
			target := entry.Object.(*Know).Object.String()[0:lablen]
			id := fmt.Sprintf("%v-%v-%v", source, target, "know")
			el = &CyElement{Group: "edges", Data: &CyData{Id: id, Source: source, Target: target}}
		}
		switch entry.Action {
		case "Add":
			added = append(added, el)
		case "Remove":
			removed = append(removed, el.Data.Id)
		case "On":
			el.Data.On = true
			added = append(added, el)
		case "Off":
			el.Data.On = false
			removed = append(removed, el.Data.Id)
		}
	}
	self.Journal = nil
	return &CyUpdate{
		Add:    added,
		Remove: removed,
	}
}

type CyUpdate struct {
	Add    []*CyElement `json:"add"`
	Remove []string     `json:"remove"`
}

func (self *Network) Query(conf interface{}, c *ResourceController) (interface{}, error) {
	glog.V(6).Infof("query: GET handler ")
	// config := conf.(*QueryConfig)
	return interface{}(self.CyUpdate()), nil
}

// deltas: changes in the number of cumulative actions: non-negative integers.
// base unit is the fixed minimal interval  between two measurements (time quantum)
// acceleration : to slow down you just set the base unit higher.
// to speed up: skip x number of base units
// frequency: given as the (constant or average) number of base units between measurements
// if resolution is expressed as the inverse of frequency  = preserved information
// setting the acceleration
// beginning of the record (lifespan) of the network is index 0
// acceleration means that snapshots are rarer so the same span can be generated by the journal
// then update logs can be compressed (toonly one state transition per affected node)
// epoch, epochcount

type Delta struct {
	On  int
	Off int
}

func oneOutOf(n int) int {
	t := rand.Intn(n)
	if t == 0 {
		return 1
	}
	return 0
}

func deltas(i int) (d []*Delta) {
	if i == 0 {
		return []*Delta{
			&Delta{10, 0},
			&Delta{20, 0},
		}
	}
	return []*Delta{
		&Delta{oneOutOf(10), oneOutOf(10)},
		&Delta{oneOutOf(2), oneOutOf(2)},
	}
}

func mockJournalTest(nw *Network, ticker *<-chan time.Time) {

	ids := RandomNodeIDs(100)
	action := "Off"
	for n := 0; ; n++ {
		select {
		case <-*ticker:
			var entries []*Entry

			if n == 0 {

				entries = []*Entry{
					&Entry{
						Type:   "Node",
						Action: "On",
						Object: &SimNode{ID: ids[0]},
					},
					&Entry{
						Type:   "Node",
						Action: "On",
						Object: &SimNode{ID: ids[1]},
					},
				}
			} else {
				sc := &SimConn{
					Caller: ids[0],
					Callee: ids[1],
				}
				if n%3 == 0 {
					if action == "On" {
						action = "Off"
					} else {
						action = "On"
					}
					entries = append(entries, &Entry{
						Type:   "Conn",
						Action: action,
						Object: sc,
					})
				}
			}

			glog.V(6).Info("entries: %v", entries)
			nw.AppendEntries(entries...)
		}
	}
}

func mockJournal(nw *Network, ticker *<-chan time.Time) {
	ids := RandomNodeIDs(100)
	var onNodes []*SimNode
	offNodes := ids
	var onConns []*SimConn

	for n := 0; ; n++ {
		select {
		case <-*ticker:
			var entries []*Entry
			ds := deltas(n)
			for i := 0; len(offNodes) > 0 && i < ds[0].On; i++ {
				c := rand.Intn(len(offNodes))
				sn := &SimNode{ID: offNodes[c]}
				entries = append(entries, &Entry{
					Type:   "Node",
					Action: "On",
					Object: sn,
				})
				onNodes = append(onNodes, sn)
				offNodes = append(offNodes[0:c], offNodes[c+1:]...)
			}
			for i := 0; len(onNodes) > 0 && i < ds[0].Off; i++ {
				c := rand.Intn(len(onNodes))
				sn := onNodes[c]
				entries = append(entries, &Entry{
					Type:   "Node",
					Action: "Off",
					Object: sn,
				})
				onNodes = append(onNodes[0:c], onNodes[c+1:]...)
				offNodes = append(offNodes, sn.ID)
			}
			for i := 0; len(onNodes) > 1 && i < ds[1].On; i++ {
				caller := onNodes[rand.Intn(len(onNodes))].ID
				callee := onNodes[rand.Intn(len(onNodes))].ID
				if caller == callee {
					i--
					continue
				}
				sc := &SimConn{
					Caller: caller,
					Callee: callee,
				}
				entries = append(entries, &Entry{
					Type:   "Conn",
					Action: "On",
					Object: sc,
				})
				onConns = append(onConns, sc)
			}
			for i := 0; len(onConns) > 0 && i < ds[1].Off; i++ {
				c := rand.Intn(len(onConns))
				entries = append(entries, &Entry{
					Type:   "Conn",
					Action: "Off",
					Object: onConns[c],
				})
				onConns = append(onConns[0:c], onConns[c+1:]...)
			}
			glog.V(6).Info("entries: %v", entries)
			nw.AppendEntries(entries...)
		}
	}

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
	self.Nodes = append(self.Nodes, &SimNode{id, conf, adapters.NetAdapter(simnet)})
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
