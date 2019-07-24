package simulation

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"reflect"
	"strings"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/ethersphere/swarm/log"
	"github.com/ethersphere/swarm/network"
	"golang.org/x/sync/errgroup"
)

type nodeMap struct {
	sync.RWMutex
	internal map[NodeID]Node
}

func newNodeMap() *nodeMap {
	return &nodeMap{
		internal: make(map[NodeID]Node),
	}
}

func (nm *nodeMap) Load(key NodeID) (value Node, ok bool) {
	nm.RLock()
	result, ok := nm.internal[key]
	nm.RUnlock()
	return result, ok
}

func (nm *nodeMap) LoadAll() []Node {
	nm.RLock()
	v := []Node{}
	for _, node := range nm.internal {
		v = append(v, node)
	}
	nm.RUnlock()
	return v
}

func (nm *nodeMap) Store(key NodeID, value Node) {
	nm.Lock()
	nm.internal[key] = value
	nm.Unlock()
}

// Simulation is used to simulate a network of nodes
type Simulation struct {
	adapter Adapter
	nodes   *nodeMap
}

// NewSimulation creates a new simulation given an adapter
func NewSimulation(adapter Adapter) *Simulation {
	sim := &Simulation{
		adapter: adapter,
		nodes:   newNodeMap(),
	}
	return sim
}

func getAdapterFromSnapshotConfig(snapshot *AdapterSnapshot) (Adapter, error) {
	if snapshot == nil {
		return nil, errors.New("snapshot can't be nil")
	}
	var adapter Adapter
	var err error
	switch t := snapshot.Type; t {
	case "exec":
		adapter, err = NewExecAdapter(snapshot.Config.(ExecAdapterConfig))
	case "docker":
		adapter, err = NewDockerAdapter(snapshot.Config.(DockerAdapterConfig))
	case "kubernetes":
		adapter, err = NewKubernetesAdapter(snapshot.Config.(KubernetesAdapterConfig))
	default:
		return nil, fmt.Errorf("unknown adapter type: %s", t)
	}
	if err != nil {
		return nil, fmt.Errorf("could not initialize %s adapter: %v", snapshot.Type, err)
	}
	return adapter, nil
}

// NewSimulationFromSnapshot creates a simulation from a snapshot
func NewSimulationFromSnapshot(snapshot *Snapshot) (*Simulation, error) {
	// Create adapter
	adapter, err := getAdapterFromSnapshotConfig(snapshot.DefaultAdapter)
	if err != nil {
		return nil, err
	}
	sim := &Simulation{
		adapter: adapter,
		nodes:   newNodeMap(),
	}

	// Loop over nodes and add them
	for _, n := range snapshot.Nodes {
		if n.Adapter == nil {
			if err := sim.Init(n.Config); err != nil {
				return sim, fmt.Errorf("failed to initialize node %v", err)
			}
		} else {
			adapter, err := getAdapterFromSnapshotConfig(n.Adapter)
			if err != nil {
				return sim, fmt.Errorf("could not read adapter configureation for node %s: %v", n.Config.ID, err)
			}
			if err := sim.InitWithAdapter(n.Config, adapter); err != nil {
				return sim, fmt.Errorf("failed to initialize node %s: %v", n.Config.ID, err)
			}
		}
	}

	// Start all nodes
	err = sim.StartAll()
	if err != nil {
		return sim, err
	}

	// Establish connections
	m := make(map[string]Node)
	for _, n := range sim.GetAll() {
		enode := removeNetworkAddressFromEnode(n.Info().Enode)
		m[enode] = n
	}

	for _, con := range snapshot.Connections {
		from, ok := m[con.From]
		if !ok {
			return sim, fmt.Errorf("no node found with enode: %s", con.From)
		}
		to, ok := m[con.To]
		if !ok {
			return sim, fmt.Errorf("no node found with enode: %s", con.To)
		}

		client, err := sim.RPCClient(from.Info().ID)
		if err != nil {
			return sim, err
		}
		defer client.Close()

		if err := client.Call(nil, "admin_addPeer", to.Info().Enode); err != nil {
			return sim, err
		}
	}
	return sim, nil
}

func (s *AdapterSnapshot) detectConfigurationType() error {
	adapterconfig, err := json.Marshal(s.Config)
	if err != nil {
		return err
	}
	switch t := s.Type; t {
	case "exec":
		var config ExecAdapterConfig
		err := json.Unmarshal(adapterconfig, &config)
		if err != nil {
			return err
		}
		s.Config = config
	case "docker":
		var config DockerAdapterConfig
		err := json.Unmarshal(adapterconfig, &config)
		if err != nil {
			return err
		}
		s.Config = config
	case "kubernetes":
		var config KubernetesAdapterConfig
		err := json.Unmarshal(adapterconfig, &config)
		if err != nil {
			return err
		}
		s.Config = config
	default:
		return fmt.Errorf("unknown adapter type: %s", t)
	}
	return nil
}

func unmarshalSnapshot(data []byte, snapshot *Snapshot) error {
	err := json.Unmarshal(data, snapshot)
	if err != nil {
		return err
	}

	// snapshot.Adapter.Config will be of type map[string]interface{}
	// we have to unmarshal it to the correct adapter configuration struct
	if err := snapshot.DefaultAdapter.detectConfigurationType(); err != nil {
		return err
	}
	for _, n := range snapshot.Nodes {
		if n.Adapter != nil {
			if err := n.Adapter.detectConfigurationType(); err != nil {
				return err
			}
		}
	}
	return nil
}

// LoadSnapshotFromFile loads a snapshot from a given JSON file
func LoadSnapshotFromFile(filePath string) (*Snapshot, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	bytes, err := ioutil.ReadAll(f)
	if err != nil {
		return nil, err
	}

	var snapshot Snapshot
	err = unmarshalSnapshot(bytes, &snapshot)
	if err != nil {
		return nil, err
	}
	return &snapshot, nil
}

// Get returns a node by ID
func (s *Simulation) Get(id NodeID) (Node, error) {
	node, ok := s.nodes.Load(id)
	if !ok {
		return nil, fmt.Errorf("a node with id %s does not exist", id)
	}
	return node, nil
}

// GetAll returns all nodes
func (s *Simulation) GetAll() []Node {
	return s.nodes.LoadAll()
}

// DefaultAdapter returns the default adapter that the simulation was initialized with
func (s *Simulation) DefaultAdapter() Adapter {
	return s.adapter
}

// Init initializes a node with the NodeConfig with the default Adapter
func (s *Simulation) Init(config NodeConfig) error {
	return s.InitWithAdapter(config, s.DefaultAdapter())
}

// InitWithAdapter initializes a node with the NodeConfig and the given Adapter
func (s *Simulation) InitWithAdapter(config NodeConfig, adapter Adapter) error {
	if _, ok := s.nodes.Load(config.ID); ok {
		return fmt.Errorf("a node with id %s already exists", config.ID)
	}
	node := adapter.NewNode(config)
	s.nodes.Store(config.ID, node)
	return nil
}

// Start starts a node by ID
func (s *Simulation) Start(id NodeID) error {
	node, ok := s.nodes.Load(id)
	if !ok {
		return fmt.Errorf("a node with id %s does not exist", id)
	}

	if err := node.Start(); err != nil {
		return fmt.Errorf("could not start node: %v", err)
	}
	return nil
}

// Stop stops a node by ID
func (s *Simulation) Stop(id NodeID) error {
	node, ok := s.nodes.Load(id)
	if !ok {
		return fmt.Errorf("a node with id %s does not exist", id)
	}

	if err := node.Stop(); err != nil {
		return fmt.Errorf("could not stop node: %v", err)
	}
	return nil
}

// StartAll starts all nodes
func (s *Simulation) StartAll() error {
	g, _ := errgroup.WithContext(context.Background())
	for _, node := range s.nodes.LoadAll() {
		g.Go(node.Start)
	}
	return g.Wait()
}

// StopAll stops all nodes
func (s *Simulation) StopAll() error {
	g, _ := errgroup.WithContext(context.Background())
	for _, node := range s.nodes.LoadAll() {
		g.Go(node.Stop)
	}
	return g.Wait()
}

// RPCClient returns an RPC Client for a given node
func (s *Simulation) RPCClient(id NodeID) (*rpc.Client, error) {
	node, ok := s.nodes.Load(id)
	if !ok {
		return nil, fmt.Errorf("a node with id %s does not exist", id)
	}

	info := node.Info()

	var client *rpc.Client
	var err error
	for start := time.Now(); time.Since(start) < 10*time.Second; time.Sleep(50 * time.Millisecond) {
		client, err = rpc.Dial(info.RPCListen)
		if err == nil {
			break
		}
	}
	if client == nil {
		return nil, fmt.Errorf("could not establish rpc connection: %v", err)
	}

	return client, nil
}

// HTTPBaseAddr returns the address for the HTTP API
func (s *Simulation) HTTPBaseAddr(id NodeID) (string, error) {
	node, ok := s.nodes.Load(id)
	if !ok {
		return "", fmt.Errorf("a node with id %s does not exist", id)
	}
	info := node.Info()
	return info.HTTPListen, nil
}

// Snapshot returns a snapshot of the simulation
func (s *Simulation) Snapshot() (*Snapshot, error) {
	snap := Snapshot{}

	// Default adapter snapshot
	asnap := s.DefaultAdapter().Snapshot()
	snap.DefaultAdapter = &asnap

	// Nodes snapshot
	nodes := s.GetAll()
	snap.Nodes = make([]NodeSnapshot, len(nodes))

	snap.Connections = []ConnectionSnapshot{}

	for idx, n := range nodes {
		ns, err := n.Snapshot()
		if err != nil {
			return nil, fmt.Errorf("failed to get nodes snapshot %s: %v", n.Info().ID, err)
		}

		// Don't need to specify the node's adapter snapshot if it's
		// the same as the default adapters snapshot
		if reflect.DeepEqual(asnap, *ns.Adapter) {
			ns.Adapter = nil
		}
		snap.Nodes[idx] = ns

		// Get connections
		client, err := s.RPCClient(n.Info().ID)
		if err != nil {
			return nil, err
		}
		defer client.Close()
		var peers []*p2p.PeerInfo
		err = client.Call(&peers, "admin_peers")
		if err != nil {
			return nil, err
		}
		for _, p := range peers {
			// Only care about outbound connections
			if !p.Network.Inbound {
				snap.Connections = append(snap.Connections, ConnectionSnapshot{
					// we need to remove network addresses from enodes
					// because they will change between simulations
					From: removeNetworkAddressFromEnode(n.Info().Enode),
					To:   removeNetworkAddressFromEnode(p.Enode),
				})
			}
		}
	}

	return &snap, nil
}

// AddBootnode adds and starts a bootnode with the given id and arguments
func (s *Simulation) AddBootnode(id NodeID, args []string) (Node, error) {
	a := []string{
		"--bootnode-mode",
		"--bootnodes", "",
	}
	a = append(a, args...)
	return s.AddNode(id, a)
}

// AddNode adds and starts a node with the given id and arguments
func (s *Simulation) AddNode(id NodeID, args []string) (Node, error) {
	bzzkey, err := randomHexKey()
	if err != nil {
		return nil, err
	}
	nodekey, err := randomHexKey()
	if err != nil {
		return nil, err
	}
	a := []string{
		"--bzzkeyhex", bzzkey,
		"--nodekeyhex", nodekey,
	}
	a = append(a, args...)
	cfg := NodeConfig{
		ID:   id,
		Args: a,
		// TODO: Figure out how to handle logs when using AddNode(...)
		Stdout: ioutil.Discard,
		Stderr: ioutil.Discard,
	}
	err = s.Init(cfg)
	if err != nil {
		return nil, err
	}

	err = s.Start(id)
	if err != nil {
		return nil, err
	}
	node, err := s.Get(id)
	if err != nil {
		return nil, err
	}
	return node, nil
}

// AddNodes adds and starts 'count' nodes with a given ID prefix, arguments.
// If the idPrefix is "node" and count is 3 then the following nodes will be
// created: node-0, node-1, node-2
func (s *Simulation) AddNodes(idPrefix string, count int, args []string) ([]Node, error) {
	g, _ := errgroup.WithContext(context.Background())

	idFormat := "%s-%d"

	for i := 0; i < count; i++ {
		id := NodeID(fmt.Sprintf(idFormat, idPrefix, i))
		g.Go(func() error {
			node, err := s.AddNode(id, args)
			if err != nil {
				log.Warn("Failed to add node", "id", id, "err", err.Error())
			} else {
				log.Info("Added node", "id", id, "enode", node.Info().Enode)
			}
			return err
		})
	}
	err := g.Wait()
	if err != nil {
		return nil, err
	}

	nodes := make([]Node, count)
	for i := 0; i < count; i++ {
		id := NodeID(fmt.Sprintf(idFormat, idPrefix, i))
		nodes[i], err = s.Get(id)
		if err != nil {
			return nil, err
		}
	}
	return nodes, nil
}

// CreateClusterWithBootnode adds and starts a bootnode. Afterwards it will add and start 'count' nodes that connect
// to the bootnode. All nodes can be provided by custom arguments.
// If the idPrefix is "node" and count is 3 then you will have the following nodes created:
//  node-bootnode, node-0, node-1, node-2.
// The bootnode will be the first node on the returned Node slice.
func (s *Simulation) CreateClusterWithBootnode(idPrefix string, count int, args []string) ([]Node, error) {
	bootnode, err := s.AddBootnode(NodeID(fmt.Sprintf("%s-bootnode", idPrefix)), args)
	if err != nil {
		return nil, err
	}

	nodeArgs := []string{
		"--bootnodes", bootnode.Info().Enode,
	}
	nodeArgs = append(nodeArgs, args...)

	n, err := s.AddNodes(idPrefix, count, nodeArgs)
	if err != nil {
		return nil, err
	}
	nodes := []Node{bootnode}
	nodes = append(nodes, n...)
	return nodes, nil
}

// WaitForHealthyNetwork will block until all the nodes are considered
// to have a healthy kademlia table
func (s *Simulation) WaitForHealthyNetwork() error {
	nodes := s.GetAll()

	// Generate RPC clients
	var clients struct {
		RPC []*rpc.Client
		mu  sync.Mutex
	}
	clients.RPC = make([]*rpc.Client, len(nodes))

	g, _ := errgroup.WithContext(context.Background())

	for idx, node := range nodes {
		node := node
		idx := idx
		g.Go(func() error {
			id := node.Info().ID
			client, err := s.RPCClient(id)
			if err != nil {
				return err
			}
			clients.mu.Lock()
			clients.RPC[idx] = client
			clients.mu.Unlock()
			return nil
		})
	}
	if err := g.Wait(); err != nil {
		return err
	}
	for _, c := range clients.RPC {
		defer c.Close()
	}

	// Generate addresses for PotMap
	addrs := [][]byte{}
	for _, node := range nodes {
		byteaddr, err := hexutil.Decode(node.Info().BzzAddr)
		if err != nil {
			return err
		}
		addrs = append(addrs, byteaddr)
	}

	ppmap := network.NewPeerPotMap(network.NewKadParams().NeighbourhoodSize, addrs)

	log.Info("Waiting for healthy kademlia...")

	// Check for healthInfo on all nodes
	for {
		g, _ = errgroup.WithContext(context.Background())
		for i := 0; i < len(nodes)-1; i++ {
			i := i
			g.Go(func() error {
				log.Debug("Checking hive_getHealthInfo", "node", nodes[i].Info().ID)
				healthy := &network.Health{}
				if err := clients.RPC[i].Call(&healthy, "hive_getHealthInfo", ppmap[nodes[i].Info().BzzAddr[2:]]); err != nil {
					return err
				}
				if !healthy.Healthy() {
					return fmt.Errorf("node %s is not healthy", nodes[i].Info().ID)
				}
				return nil
			})
		}
		err := g.Wait()
		if err == nil {
			break
		}
		log.Info("Not healthy yet...", "msg", err.Error())
		time.Sleep(500 * time.Millisecond)
	}

	log.Info("Healthy kademlia on all nodes")
	return nil
}

func randomHexKey() (string, error) {
	key, err := crypto.GenerateKey()
	if err != nil {
		return "", err
	}
	keyhex := hex.EncodeToString(crypto.FromECDSA(key))
	return keyhex, nil
}

func removeNetworkAddressFromEnode(enode string) string {
	if idx := strings.Index(enode, "@"); idx != -1 {
		return enode[:idx]
	}
	return enode
}
