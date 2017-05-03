package adapters

import (
	"context"
	"crypto/ecdsa"
	"encoding/hex"
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
	"github.com/ethereum/go-ethereum/crypto"
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
	if _, exists := serviceFuncs[config.Service]; !exists {
		return nil, fmt.Errorf("unknown node service %q", config.Service)
	}

	// create the node directory using the first 12 characters of the ID
	// as Unix socket paths cannot be longer than 256 characters
	dir := filepath.Join(e.BaseDir, config.Id.String()[:12])
	if err := os.Mkdir(dir, 0755); err != nil {
		return nil, fmt.Errorf("error creating node directory: %s", err)
	}

	// generate the config
	conf := node.DefaultConfig
	conf.DataDir = filepath.Join(dir, "data")
	conf.P2P.EnableMsgEvents = true
	conf.P2P.NoDiscovery = true
	conf.P2P.NAT = nil

	// listen on a random localhost port (we'll get the actual port after
	// starting the node through the RPC admin.nodeInfo method)
	conf.P2P.ListenAddr = "127.0.0.1:0"

	node := &ExecNode{
		ID:      config.Id,
		Service: config.Service,
		Dir:     dir,
		Config:  &conf,
		key:     config.PrivateKey,
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
	ID      *NodeId
	Service string
	Dir     string
	Config  *node.Config
	Cmd     *exec.Cmd
	Info    *p2p.NodeInfo

	client *rpc.Client
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

// Start exec's the node passing the ID and service as command line arguments,
// the node config encoded as JSON in the _P2P_NODE_CONFIG environment
// variable and the node's private key hex-endoded in the _P2P_NODE_KEY
// environment variable
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

	// encode the private key
	key := hex.EncodeToString(crypto.FromECDSA(n.key))

	// create a net.Pipe for RPC communication over stdin / stdout
	pipe1, pipe2 := net.Pipe()

	// start the node
	cmd := n.newCmd()
	cmd.Stdin = pipe1
	cmd.Stdout = pipe1
	cmd.Stderr = os.Stderr
	cmd.Env = append(os.Environ(),
		fmt.Sprintf("_P2P_NODE_CONFIG=%s", conf),
		fmt.Sprintf("_P2P_NODE_KEY=%s", key),
	)
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("error starting node: %s", err)
	}
	n.Cmd = cmd

	// create the RPC client and load the node info
	n.client = rpc.NewClientWithConn(pipe2)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	var info p2p.NodeInfo
	if err := n.client.CallContext(ctx, &info, "admin_nodeInfo"); err != nil {
		return fmt.Errorf("error getting node info: %s", err)
	}
	n.Info = &info

	return nil
}

// execCommand returns a command which runs the node locally by exec'ing
// the current binary but setting argv[0] to "p2p-node" so that the child
// runs execP2PNode
func (n *ExecNode) execCommand() *exec.Cmd {
	return &exec.Cmd{
		Path: reexec.Self(),
		Args: []string{"p2p-node", n.Service, n.ID.String()},
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
	if n.client == nil {
		return n.Info
	}
	info := &p2p.NodeInfo{}
	if err := n.client.Call(&info, "admin_nodeInfo"); err != nil {
		return n.Info
	}
	return info
}

func init() {
	// register a reexec function to start a devp2p node when the current
	// binary is executed as "p2p-node"
	reexec.Register("p2p-node", execP2PNode)
}

// execP2PNode starts a devp2p node when the current binary is executed with
// argv[0] being "p2p-node", reading the service / ID from argv[1] / argv[2],
// the node config from the _P2P_NODE_CONFIG environment variable and the
// private key from the _P2P_NODE_KEY environment variable
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

	// decode the private key
	keyEnv := os.Getenv("_P2P_NODE_KEY")
	if keyEnv == "" {
		log.Crit("missing _P2P_NODE_KEY")
	}
	key, err := hex.DecodeString(keyEnv)
	if err != nil {
		log.Crit("error decoding _P2P_NODE_KEY", "err", err)
	}
	conf.P2P.PrivateKey = crypto.ToECDSA(key)

	// initialize the service
	serviceFunc, exists := serviceFuncs[serviceName]
	if !exists {
		log.Crit(fmt.Sprintf("unknown node service %q", serviceName))
	}
	service := serviceFunc(id)

	// use explicit IP address in ListenAddr so that Enode URL is usable
	if strings.HasPrefix(conf.P2P.ListenAddr, ":") {
		addrs, err := net.InterfaceAddrs()
		if err != nil {
			log.Crit("error getting IP address", "err", err)
		}
		for _, addr := range addrs {
			if ip, ok := addr.(*net.IPNet); ok && !ip.IP.IsLoopback() {
				conf.P2P.ListenAddr = ip.IP.String() + conf.P2P.ListenAddr
				break
			}
		}
	}

	// start the devp2p stack
	stack, err := startP2PNode(&conf, service)
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

	constructor := func(s node.Service) node.ServiceConstructor {
		return func(ctx *node.ServiceContext) (node.Service, error) {
			return s, nil
		}
	}

	// register the peer events API
	//
	// TODO: move this to node.PrivateAdminAPI once the following is merged:
	//       https://github.com/ethereum/go-ethereum/pull/13885
	if err := stack.Register(constructor(&PeerAPI{stack.Server})); err != nil {
		return nil, err
	}

	if err := stack.Register(constructor(service)); err != nil {
		return nil, err
	}
	if err := stack.Start(); err != nil {
		return nil, err
	}
	return stack, nil
}

// PeerAPI is used to expose peer events under the "eth" RPC namespace.
//
// TODO: move this to node.PrivateAdminAPI and expose under the "admin"
//       namespace once the following is merged:
//       https://github.com/ethereum/go-ethereum/pull/13885
type PeerAPI struct {
	server func() p2p.Server
}

func (p *PeerAPI) Protocols() []p2p.Protocol {
	return nil
}

func (p *PeerAPI) APIs() []rpc.API {
	return []rpc.API{{
		Namespace: "eth",
		Version:   "1.0",
		Service:   p,
	}}
}

func (p *PeerAPI) Start(p2p.Server) error {
	return nil
}

func (p *PeerAPI) Stop() error {
	return nil
}

// PeerEvents creates an RPC sunscription which receives peer events from the
// underlying p2p.Server
func (p *PeerAPI) PeerEvents(ctx context.Context) (*rpc.Subscription, error) {
	notifier, supported := rpc.NotifierFromContext(ctx)
	if !supported {
		return &rpc.Subscription{}, rpc.ErrNotificationsUnsupported
	}

	rpcSub := notifier.CreateSubscription()

	go func() {
		events := make(chan *p2p.PeerEvent)
		sub := p.server().SubscribeEvents(events)
		defer sub.Unsubscribe()

		for {
			select {
			case event := <-events:
				notifier.Notify(rpcSub.ID, event)
			case <-rpcSub.Err():
				return
			case <-notifier.Closed():
				return
			}
		}
	}()

	return rpcSub, nil
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
