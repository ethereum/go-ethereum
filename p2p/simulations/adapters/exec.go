package adapters

import (
	"context"
	"crypto/ecdsa"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/docker/docker/pkg/reexec"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/rpc"
)

// ExecAdapter is a NodeAdapter which runs nodes by executing the current
// binary as a child process.
//
// An init hook is used so that the child process executes the node service
// (rather than whataver the main() function would normally do), see the
// execP2PNode function for more information.
type ExecAdapter struct {
	BaseDir string
}

// NewExecAdapter returns an ExecAdapter which stores node data in
// subdirectories of the given base directory
func NewExecAdapter(baseDir string) *ExecAdapter {
	return &ExecAdapter{BaseDir: baseDir}
}

// Name returns the name of the adapter for logging purpoeses
func (e *ExecAdapter) Name() string {
	return "exec-adapter"
}

// NewNode returns a new ExecNode using the given config
func (e *ExecAdapter) NewNode(config *NodeConfig) (Node, error) {
	for _, name := range config.Services {
		if _, exists := serviceFuncs[name]; !exists {
			return nil, fmt.Errorf("unknown node service %q", name)
		}
	}

	// create the node directory using the first 12 characters of the ID
	// as Unix socket paths cannot be longer than 256 characters
	dir := filepath.Join(e.BaseDir, config.Id.String()[:12])
	if err := os.Mkdir(dir, 0755); err != nil {
		return nil, fmt.Errorf("error creating node directory: %s", err)
	}

	// generate the config
	conf := &execNodeConfig{
		Stack: node.DefaultConfig,
		Node:  config,
	}
	conf.Stack.DataDir = filepath.Join(dir, "data")
	conf.Stack.P2P.EnableMsgEvents = false
	conf.Stack.P2P.NoDiscovery = true
	conf.Stack.P2P.NAT = nil

	// listen on a random localhost port (we'll get the actual port after
	// starting the node through the RPC admin.nodeInfo method)
	conf.Stack.P2P.ListenAddr = "127.0.0.1:0"

	node := &ExecNode{
		ID:     config.Id,
		Dir:    dir,
		Config: conf,
		Services: config.Services,
	}
	node.newCmd = node.execCommand
	return node, nil
}

// ExecNode is a NodeAdapter which starts the node by exec'ing the current
// binary and running a registered ServiceFunc.
//
// Communication with the node is performed using RPC over stdin / stdout
// so that we don't need access to either the node's filesystem or TCP stack
// (so for example we can run the node in a remote Docker container and
// still communicate with it).
type ExecNode struct {
	ID     *NodeId
	Dir    string
	Config *execNodeConfig
	Cmd    *exec.Cmd
	Info   *p2p.NodeInfo
	Services []string

	client *rpc.Client
	rpcMux *rpcMux
	newCmd func() *exec.Cmd
	key    *ecdsa.PrivateKey
}

// Addr returns the node's enode URL
func (n *ExecNode) Addr() []byte {
	if n.Info == nil {
		return nil
	}
	return []byte(n.Info.Enode)
}

// Client returns an rpc.Client which can be used to communicate with the
// underlying service (it is set once the node has started)
func (n *ExecNode) Client() (*rpc.Client, error) {
	return n.client, nil
}

// Start exec's the node passing the ID and service as command line arguments
// and the node config encoded as JSON in the _P2P_NODE_CONFIG environment
// variable
func (n *ExecNode) Start(snapshot []byte) (err error) {
	if n.Cmd != nil {
		return errors.New("already started")
	}
	defer func() {
		if err != nil {
			log.Error("node failed to start", "err", err)
			n.Stop()
		}
	}()

	// encode a copy of the config containing the snapshot
	confCopy := *n.Config
	confCopy.Snapshot = snapshot
	confData, err := json.Marshal(confCopy)
	if err != nil {
		return fmt.Errorf("error generating node config: %s", err)
	}

	// create a net.Pipe for RPC communication over stdin / stdout
	pipe1, pipe2 := net.Pipe()

	// start the node
	cmd := n.newCmd()
	cmd.Stdin = pipe1
	cmd.Stdout = pipe1
	cmd.Stderr = os.Stderr
	cmd.Env = append(os.Environ(), fmt.Sprintf("_P2P_NODE_CONFIG=%s", confData))
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("error starting node: %s", err)
	}
	n.Cmd = cmd

	// create the RPC client and load the node info
	n.rpcMux = newRPCMux(pipe2)
	n.client = n.rpcMux.Client()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	var info p2p.NodeInfo
	if err := n.client.CallContext(ctx, &info, "admin_nodeInfo"); err != nil {
		return fmt.Errorf("error getting node info: %s", err)
	}
	n.Info = &info

	return nil
}


func (n *ExecNode) GetService(name string) node.Service {
	return nil
}

// execCommand returns a command which runs the node locally by exec'ing
// the current binary but setting argv[0] to "p2p-node" so that the child
// runs execP2PNode
func (n *ExecNode) execCommand() *exec.Cmd {
	return &exec.Cmd{
		Path: reexec.Self(),
		Args: []string{"p2p-node", n.Services[0], n.ID.String()},
	}
}

// Stop stops the node by first sending SIGTERM and then SIGKILL if the node
// doesn't stop within 5s
func (n *ExecNode) Stop() error {
	if n.Cmd == nil {
		return nil
	}
	defer func() {
		n.Cmd = nil
	}()

	if n.client != nil {
		n.client.Close()
		n.client = nil
		n.Info = nil
	}

	if err := n.Cmd.Process.Signal(syscall.SIGTERM); err != nil {
		return n.Cmd.Process.Kill()
	}
	waitErr := make(chan error)
	go func() {
		waitErr <- n.Cmd.Wait()
	}()
	select {
	case err := <-waitErr:
		return err
	case <-time.After(5 * time.Second):
		return n.Cmd.Process.Kill()
	}
}

// NodeInfo returns information about the node
func (n *ExecNode) NodeInfo() *p2p.NodeInfo {
	info := &p2p.NodeInfo{
		ID: n.ID.String(),
	}
	if n.client != nil {
		n.client.Call(&info, "admin_nodeInfo")
	}
	return info
}

// ServeRPC serves RPC requests over the given connection using the node's
// RPC multiplexer
func (n *ExecNode) ServeRPC(conn net.Conn) error {
	if n.rpcMux == nil {
		return errors.New("RPC not started")
	}
	n.rpcMux.Serve(conn)
	return nil
}

// Snapshot creates a snapshot of the service state by calling the
// simulation_snapshot RPC method
func (n *ExecNode) Snapshot() ([]byte, error) {
	if n.client == nil {
		return nil, errors.New("RPC not started")
	}
	var snapshot []byte
	return snapshot, n.client.Call(&snapshot, "simulation_snapshot")
}

func init() {
	// register a reexec function to start a devp2p node when the current
	// binary is executed as "p2p-node"
	reexec.Register("p2p-node", execP2PNode)
}

// execNodeConfig is used to serialize the node configuration so it can be
// passed to the child process as a JSON encoded environment variable
type execNodeConfig struct {
	Stack    node.Config `json:"stack"`
	Node     *NodeConfig `json:"node"`
	Snapshot []byte      `json:"snapshot,omitempty"`
}

// execP2PNode starts a devp2p node when the current binary is executed with
// argv[0] being "p2p-node", reading the service / ID from argv[1] / argv[2]
// and the node config from the _P2P_NODE_CONFIG environment variable
func execP2PNode() {
	glogger := log.NewGlogHandler(log.StreamHandler(os.Stderr, log.LogfmtFormat()))
	glogger.Verbosity(log.LvlInfo)
	log.Root().SetHandler(glogger)

	// read the service and ID from argv
	serviceName := os.Args[1]
	id := NewNodeIdFromHex(os.Args[2])

	// decode the config
	confEnv := os.Getenv("_P2P_NODE_CONFIG")
	if confEnv == "" {
		log.Crit("missing _P2P_NODE_CONFIG")
	}
	var conf execNodeConfig
	if err := json.Unmarshal([]byte(confEnv), &conf); err != nil {
		log.Crit("error decoding _P2P_NODE_CONFIG", "err", err)
	}
	conf.Stack.P2P.PrivateKey = conf.Node.PrivateKey

	// initialize the service
	serviceFunc, exists := serviceFuncs[serviceName]
	if !exists {
		log.Crit(fmt.Sprintf("unknown node service %q", serviceName))
	}
	service := serviceFunc(id, conf.Snapshot)

	// use explicit IP address in ListenAddr so that Enode URL is usable
	if strings.HasPrefix(conf.Stack.P2P.ListenAddr, ":") {
		addrs, err := net.InterfaceAddrs()
		if err != nil {
			log.Crit("error getting IP address", "err", err)
		}
		for _, addr := range addrs {
			if ip, ok := addr.(*net.IPNet); ok && !ip.IP.IsLoopback() {
				conf.Stack.P2P.ListenAddr = ip.IP.String() + conf.Stack.P2P.ListenAddr
				break
			}
		}
	}

	// start the devp2p stack
	stack, err := startP2PNode(&conf.Stack, service)
	if err != nil {
		log.Crit("error starting p2p node", "err", err)
	}

	// use stdin / stdout for RPC to avoid the parent needing to access
	// either the local filesystem or TCP stack (useful when running in
	// Docker)
	handler, err := stack.RPCHandler()
	if err != nil {
		log.Crit("error getting RPC server", "err", err)
	}
	go handler.ServeCodec(rpc.NewJSONCodec(&stdioConn{os.Stdin, os.Stdout}), rpc.OptionMethodInvocation|rpc.OptionSubscriptions)

	go func() {
		sigc := make(chan os.Signal, 1)
		signal.Notify(sigc, syscall.SIGTERM)
		defer signal.Stop(sigc)
		<-sigc
		log.Info("Received SIGTERM, shutting down...")
		stack.Stop()
	}()

	stack.Wait()
}

func startP2PNode(conf *node.Config, service node.Service) (*node.Node, error) {
	stack, err := node.New(conf)
	if err != nil {
		return nil, err
	}
	constructor := func(ctx *node.ServiceContext) (node.Service, error) {
		return &snapshotService{service}, nil
	}
	if err := stack.Register(constructor); err != nil {
		return nil, err
	}
	if err := stack.Start(); err != nil {
		return nil, err
	}
	return stack, nil
}

// snapshotService wraps a node.Service and injects a snapshot API into the
// list of RPC APIs
type snapshotService struct {
	node.Service
}

func (s *snapshotService) APIs() []rpc.API {
	return append([]rpc.API{{
		Namespace: "simulation",
		Version:   "1.0",
		Service:   SnapshotAPI{s.Service},
	}}, s.Service.APIs()...)
}

// SnapshotAPI provides an RPC method to create a snapshot of a node.Service
type SnapshotAPI struct {
	service node.Service
}

func (api SnapshotAPI) Snapshot() ([]byte, error) {
	if s, ok := api.service.(interface {
		Snapshot() ([]byte, error)
	}); ok {
		return s.Snapshot()
	}
	return nil, nil
}

// stdioConn wraps os.Stdin / os.Stdout with a no-op Close method so we can
// use stdio for RPC messages
type stdioConn struct {
	io.Reader
	io.Writer
}

func (r *stdioConn) Close() error {
	return nil
}
