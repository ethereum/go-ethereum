package adapters

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/docker/docker/pkg/reexec"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/rpc"
)

// serviceFunc returns a node.ServiceConstructor which can be used to boot
// devp2p nodes
type serviceFunc func(id *NodeId) node.ServiceConstructor

// serviceFuncs is a map of registered services which are used to boot devp2p
// nodes
var serviceFuncs = make(map[string]serviceFunc)

// RegisterService registers the given serviceFunc which can then be used to
// start a devp2p node with the given name
func RegisterService(name string, f serviceFunc) {
	if _, exists := serviceFuncs[name]; exists {
		panic(fmt.Sprintf("node service already exists: %q", name))
	}
	serviceFuncs[name] = f
}

// ExecNode is a NodeAdapter which starts the node by exec'ing the current
// binary and running a registered serviceFunc
type ExecNode struct {
	ID      *NodeId
	Service string
	Dir     string
	Config  *node.Config
	Cmd     *exec.Cmd
	Client  *rpc.Client
	Info    *p2p.NodeInfo
}

// NewExecNode creates a new ExecNode which will run the given service using a
// sub-directory of the given baseDir
func NewExecNode(id *NodeId, service, baseDir string) (*ExecNode, error) {
	if _, exists := serviceFuncs[service]; !exists {
		return nil, fmt.Errorf("unknown node service %q", service)
	}

	// create the node directory using the first 12 characters of the ID
	dir := filepath.Join(baseDir, id.String()[0:12])
	if err := os.Mkdir(dir, 0755); err != nil {
		return nil, fmt.Errorf("error creating node directory: %s", err)
	}

	// generate the config
	conf := node.DefaultConfig
	conf.DataDir = filepath.Join(dir, "data")
	conf.IPCPath = filepath.Join(dir, "ipc.sock")
	conf.P2P.ListenAddr = "127.0.0.1:0"
	conf.P2P.NoDiscovery = true
	conf.P2P.NAT = nil

	return &ExecNode{
		ID:      id,
		Service: service,
		Dir:     dir,
		Config:  &conf,
	}, nil
}

// Addr returns the node's enode URL
func (n *ExecNode) Addr() []byte {
	if n.Info == nil {
		return nil
	}
	return []byte(n.Info.Enode)
}

// Start exec's the node passing the ID and service as command line arguments
// and the node config encoded as JSON in the _P2P_NODE_CONFIG environment
// variable
func (n *ExecNode) Start() (err error) {
	if n.Cmd != nil {
		return errors.New("already started")
	}
	defer func() {
		if err != nil {
			log.Error("node failed to start", "err", err)
			n.Stop()
		}
	}()

	// encode the config
	conf, err := json.Marshal(n.Config)
	if err != nil {
		return fmt.Errorf("error generating node config: %s", err)
	}

	// start the node
	cmd := &exec.Cmd{
		Path:   reexec.Self(),
		Args:   []string{"p2p-node", n.Service, n.ID.String()},
		Stdout: os.Stdout,
		Stderr: os.Stderr,
		Env:    append(os.Environ(), fmt.Sprintf("_P2P_NODE_CONFIG=%s", conf)),
	}
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("error starting node: %s", err)
	}
	n.Cmd = cmd

	// create the RPC client
	for start := time.Now(); time.Since(start) < 10*time.Second; time.Sleep(50 * time.Millisecond) {
		n.Client, err = rpc.Dial(n.Config.IPCPath)
		if err == nil {
			break
		}
	}
	if n.Client == nil {
		return fmt.Errorf("error creating IPC client: %s", err)
	}

	// load info
	var info p2p.NodeInfo
	if err := n.Client.Call(&info, "admin_nodeInfo"); err != nil {
		return fmt.Errorf("error getting node info: %s", err)
	}
	n.Info = &info

	return nil
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

	if n.Client != nil {
		n.Client.Close()
		n.Client = nil
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

// Connect connects the node to the given addr by calling the Admin.AddPeer
// IPC method
func (n *ExecNode) Connect(addr []byte) error {
	if n.Client == nil {
		return errors.New("node not started")
	}
	return n.Client.Call(nil, "admin_addPeer", string(addr))
}

// Disconnect disconnects the node from the given addr by calling the
// Admin.RemovePeer IPC method
func (n *ExecNode) Disconnect(addr []byte) error {
	if n.Client == nil {
		return errors.New("node not started")
	}
	return n.Client.Call(nil, "admin_removePeer", string(addr))
}

func init() {
	// register a reexec function to start a devp2p node when the current
	// binary is executed as "p2p-node"
	reexec.Register("p2p-node", execP2PNode)
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
	var conf node.Config
	if err := json.Unmarshal([]byte(confEnv), &conf); err != nil {
		log.Crit("error decoding _P2P_NODE_CONFIG", "err", err)
	}

	// lookup the service constructor
	service, exists := serviceFuncs[serviceName]
	if !exists {
		log.Crit(fmt.Sprintf("unknown node service %q", serviceName))
	}

	// start the devp2p stack
	stack, err := node.New(&conf)
	if err != nil {
		log.Crit("error creating node", "err", err)
	}
	if err := stack.Register(service(id)); err != nil {
		log.Crit("error registering service", "err", err)
	}
	if err := stack.Start(); err != nil {
		log.Crit("error starting node", "err", err)
	}

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
