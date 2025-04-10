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

package adapters

import (
	"bufio"
	"context"
	"crypto/ecdsa"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/XinFinOrg/XDPoSChain/log"
	"github.com/XinFinOrg/XDPoSChain/node"
	"github.com/XinFinOrg/XDPoSChain/p2p"
	"github.com/XinFinOrg/XDPoSChain/p2p/discover"
	"github.com/XinFinOrg/XDPoSChain/rpc"
	"github.com/docker/docker/pkg/reexec"
	"github.com/gorilla/websocket"
)

// ExecAdapter is a NodeAdapter which runs simulation nodes by executing the
// current binary as a child process.
//
// An init hook is used so that the child process executes the node services
// (rather than whataver the main() function would normally do), see the
// execP2PNode function for more information.
type ExecAdapter struct {
	// BaseDir is the directory under which the data directories for each
	// simulation node are created.
	BaseDir string

	nodes map[discover.NodeID]*ExecNode
}

// NewExecAdapter returns an ExecAdapter which stores node data in
// subdirectories of the given base directory
func NewExecAdapter(baseDir string) *ExecAdapter {
	return &ExecAdapter{
		BaseDir: baseDir,
		nodes:   make(map[discover.NodeID]*ExecNode),
	}
}

// Name returns the name of the adapter for logging purposes
func (e *ExecAdapter) Name() string {
	return "exec-adapter"
}

// NewNode returns a new ExecNode using the given config
func (e *ExecAdapter) NewNode(config *NodeConfig) (Node, error) {
	if len(config.Lifecycles) == 0 {
		return nil, errors.New("node must have at least one service lifecycle")
	}
	for _, service := range config.Lifecycles {
		if _, exists := lifecycleConstructorFuncs[service]; !exists {
			return nil, fmt.Errorf("unknown node service %q", service)
		}
	}

	// create the node directory using the first 12 characters of the ID
	// as Unix socket paths cannot be longer than 256 characters
	dir := filepath.Join(e.BaseDir, config.ID.String()[:12])
	if err := os.Mkdir(dir, 0755); err != nil {
		return nil, fmt.Errorf("error creating node directory: %s", err)
	}

	// generate the config
	conf := &execNodeConfig{
		Stack: node.DefaultConfig,
		Node:  config,
	}
	conf.Stack.DataDir = filepath.Join(dir, "data")
	conf.Stack.WSHost = "127.0.0.1"
	conf.Stack.WSPort = 0
	conf.Stack.WSOrigins = []string{"*"}
	conf.Stack.WSExposeAll = true
	conf.Stack.P2P.EnableMsgEvents = false
	conf.Stack.P2P.NoDiscovery = true
	conf.Stack.P2P.NAT = nil

	// listen on a random localhost port (we'll get the actual port after
	// starting the node through the RPC admin.nodeInfo method)
	conf.Stack.P2P.ListenAddr = "127.0.0.1:0"

	node := &ExecNode{
		ID:      config.ID,
		Dir:     dir,
		Config:  conf,
		adapter: e,
	}
	node.newCmd = node.execCommand
	e.nodes[node.ID] = node
	return node, nil
}

// ExecNode starts a simulation node by exec'ing the current binary and
// running the configured services
type ExecNode struct {
	ID     discover.NodeID
	Dir    string
	Config *execNodeConfig
	Cmd    *exec.Cmd
	Info   *p2p.NodeInfo

	adapter *ExecAdapter
	client  *rpc.Client
	wsAddr  string
	newCmd  func() *exec.Cmd
	key     *ecdsa.PrivateKey
}

// Addr returns the node's enode URL
func (n *ExecNode) Addr() []byte {
	if n.Info == nil {
		return nil
	}
	return []byte(n.Info.Enode)
}

// Client returns an rpc.Client which can be used to communicate with the
// underlying services (it is set once the node has started)
func (n *ExecNode) Client() (*rpc.Client, error) {
	return n.client, nil
}

// wsAddrPattern is a regex used to read the WebSocket address from the node's
// log
var wsAddrPattern = regexp.MustCompile(`ws://[\d.:]+`)

// Start exec's the node passing the ID and service as command line arguments
// and the node config encoded as JSON in an environment variable.
func (n *ExecNode) Start(snapshots map[string][]byte) (err error) {
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
	confCopy.Snapshots = snapshots
	confCopy.PeerAddrs = make(map[string]string)
	for id, node := range n.adapter.nodes {
		confCopy.PeerAddrs[id.String()] = node.wsAddr
	}
	confData, err := json.Marshal(confCopy)
	if err != nil {
		return fmt.Errorf("error generating node config: %s", err)
	}

	// use a pipe for stderr so we can both copy the node's stderr to
	// os.Stderr and read the WebSocket address from the logs
	stderrR, stderrW := io.Pipe()
	stderr := io.MultiWriter(os.Stderr, stderrW)

	// start the node
	cmd := n.newCmd()
	cmd.Stdout = os.Stdout
	cmd.Stderr = stderr
	cmd.Env = append(os.Environ(), envNodeConfig+"="+string(confData))
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("error starting node: %s", err)
	}
	n.Cmd = cmd

	// read the WebSocket address from the stderr logs
	var wsAddr string
	wsAddrC := make(chan string)
	go func() {
		s := bufio.NewScanner(stderrR)
		for s.Scan() {
			if strings.Contains(s.Text(), "WebSocket endpoint opened:") {
				wsAddrC <- wsAddrPattern.FindString(s.Text())
			}
		}
	}()
	select {
	case wsAddr = <-wsAddrC:
		if wsAddr == "" {
			return errors.New("failed to read WebSocket address from stderr")
		}
	case <-time.After(10 * time.Second):
		return errors.New("timed out waiting for WebSocket address on stderr")
	}

	// create the RPC client and load the node info
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	client, err := rpc.DialWebsocket(ctx, wsAddr, "")
	if err != nil {
		return fmt.Errorf("error dialing rpc websocket: %s", err)
	}
	var info p2p.NodeInfo
	if err := client.CallContext(ctx, &info, "admin_nodeInfo"); err != nil {
		return fmt.Errorf("error getting node info: %s", err)
	}
	n.client = client
	n.wsAddr = wsAddr
	n.Info = &info

	return nil
}

// execCommand returns a command which runs the node locally by exec'ing
// the current binary but setting argv[0] to "p2p-node" so that the child
// runs execP2PNode
func (n *ExecNode) execCommand() *exec.Cmd {
	return &exec.Cmd{
		Path: reexec.Self(),
		Args: []string{"p2p-node", strings.Join(n.Config.Node.Lifecycles, ","), n.ID.String()},
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
		n.wsAddr = ""
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

// ServeRPC serves RPC requests over the given connection by dialling the
// node's WebSocket address and joining the two connections
func (n *ExecNode) ServeRPC(clientConn *websocket.Conn) error {
	conn, _, err := websocket.DefaultDialer.Dial(n.wsAddr, nil)
	if err != nil {
		return err
	}
	var wg sync.WaitGroup
	wg.Add(2)
	go wsCopy(&wg, conn, clientConn)
	go wsCopy(&wg, clientConn, conn)
	wg.Wait()
	conn.Close()
	return nil
}

func wsCopy(wg *sync.WaitGroup, src, dst *websocket.Conn) {
	defer wg.Done()
	for {
		msgType, r, err := src.NextReader()
		if err != nil {
			return
		}
		w, err := dst.NextWriter(msgType)
		if err != nil {
			return
		}
		if _, err = io.Copy(w, r); err != nil {
			return
		}
	}
}

// Snapshots creates snapshots of the services by calling the
// simulation_snapshot RPC method
func (n *ExecNode) Snapshots() (map[string][]byte, error) {
	if n.client == nil {
		return nil, errors.New("RPC not started")
	}
	var snapshots map[string][]byte
	return snapshots, n.client.Call(&snapshots, "simulation_snapshot")
}

func init() {
	// register a reexec function to start a devp2p node when the current
	// binary is executed as "p2p-node"
	reexec.Register("p2p-node", execP2PNode)
}

// execNodeConfig is used to serialize the node configuration so it can be
// passed to the child process as a JSON encoded environment variable
type execNodeConfig struct {
	Stack     node.Config       `json:"stack"`
	Node      *NodeConfig       `json:"node"`
	Snapshots map[string][]byte `json:"snapshots,omitempty"`
	PeerAddrs map[string]string `json:"peer_addrs,omitempty"`
}

func initLogging() {
	// Initialize the logging by default first.
	var innerHandler slog.Handler
	innerHandler = slog.NewTextHandler(os.Stderr, nil)
	glogger := log.NewGlogHandler(innerHandler)
	glogger.Verbosity(log.LevelInfo)
	log.SetDefault(log.NewLogger(glogger))

	confEnv := os.Getenv(envNodeConfig)
	if confEnv == "" {
		return
	}
	var conf execNodeConfig
	if err := json.Unmarshal([]byte(confEnv), &conf); err != nil {
		return
	}
	var writer = os.Stderr
	if conf.Node.LogFile != "" {
		logWriter, err := os.Create(conf.Node.LogFile)
		if err != nil {
			return
		}
		writer = logWriter
	}
	var verbosity = log.LevelInfo
	if conf.Node.LogVerbosity <= log.LevelTrace && conf.Node.LogVerbosity >= log.LevelCrit {
		verbosity = log.FromLegacyLevel(int(conf.Node.LogVerbosity))
	}
	// Reinitialize the logger
	innerHandler = log.NewTerminalHandler(writer, true)
	glogger = log.NewGlogHandler(innerHandler)
	glogger.Verbosity(verbosity)
	log.SetDefault(log.NewLogger(glogger))
}

// execP2PNode starts a simulation node when the current binary is executed with
// argv[0] being "p2p-node", reading the service / ID from argv[1] / argv[2]
// and the node config from an environment variable.
func execP2PNode() {
	initLogging()

	// read the services from argv
	serviceNames := strings.Split(os.Args[1], ",")

	// decode the config
	confEnv := os.Getenv(envNodeConfig)
	if confEnv == "" {
		log.Crit("missing " + envNodeConfig)
	}
	var conf execNodeConfig
	if err := json.Unmarshal([]byte(confEnv), &conf); err != nil {
		log.Crit("error decoding "+envNodeConfig, "err", err)
	}
	conf.Stack.P2P.PrivateKey = conf.Node.PrivateKey
	conf.Stack.Logger = log.New("node.id", conf.Node.ID.String())

	// use explicit IP address in ListenAddr so that Enode URL is usable
	externalIP := func() string {
		addrs, err := net.InterfaceAddrs()
		if err != nil {
			log.Crit("error getting IP address", "err", err)
		}
		for _, addr := range addrs {
			if ip, ok := addr.(*net.IPNet); ok && !ip.IP.IsLoopback() {
				return ip.IP.String()
			}
		}
		log.Crit("unable to determine explicit IP address")
		return ""
	}
	if strings.HasPrefix(conf.Stack.P2P.ListenAddr, ":") {
		conf.Stack.P2P.ListenAddr = externalIP() + conf.Stack.P2P.ListenAddr
	}
	if conf.Stack.WSHost == "0.0.0.0" {
		conf.Stack.WSHost = externalIP()
	}

	// initialize the devp2p stack
	stack, err := node.New(&conf.Stack)
	if err != nil {
		log.Crit("error creating node stack", "err", err)
	}

	// Register the services, collecting them into a map so they can
	// be accessed by the snapshot API.
	services := make(map[string]node.Lifecycle, len(serviceNames))
	for _, name := range serviceNames {
		lifecycleFunc, exists := lifecycleConstructorFuncs[name]
		if !exists {
			log.Crit("unknown node service", "name", name)
		}
		ctx := &ServiceContext{
			RPCDialer: &wsRPCDialer{addrs: conf.PeerAddrs},
			Config:    conf.Node,
		}
		if conf.Snapshots != nil {
			ctx.Snapshot = conf.Snapshots[name]
		}
		service, err := lifecycleFunc(ctx, stack)
		if err != nil {
			log.Crit("error starting service", "name", name, "err", err)
		}
		services[name] = service
		stack.RegisterLifecycle(service)
	}

	// Add the snapshot API.
	stack.RegisterAPIs([]rpc.API{{
		Namespace: "simulation",
		Version:   "1.0",
		Service:   SnapshotAPI{services},
	}})

	if err := stack.Start(); err != nil {
		log.Crit("error stating node stack", "err", err)
	}

	// stop the stack if we get a SIGTERM signal
	go func() {
		sigc := make(chan os.Signal, 1)
		signal.Notify(sigc, syscall.SIGTERM)
		defer signal.Stop(sigc)
		<-sigc
		log.Info("Received SIGTERM, shutting down...")
		stack.Close()
	}()

	// wait for the stack to exit
	stack.Wait()
}

const (
	envNodeConfig = "_P2P_NODE_CONFIG"
)

// SnapshotAPI provides an RPC method to create snapshots of services
type SnapshotAPI struct {
	services map[string]node.Lifecycle
}

func (api SnapshotAPI) Snapshot() (map[string][]byte, error) {
	snapshots := make(map[string][]byte)
	for name, service := range api.services {
		if s, ok := service.(interface {
			Snapshot() ([]byte, error)
		}); ok {
			snap, err := s.Snapshot()
			if err != nil {
				return nil, err
			}
			snapshots[name] = snap
		}
	}
	return snapshots, nil
}

type wsRPCDialer struct {
	addrs map[string]string
}

// DialRPC implements the RPCDialer interface by creating a WebSocket RPC
// client of the given node
func (w *wsRPCDialer) DialRPC(id discover.NodeID) (*rpc.Client, error) {
	addr, ok := w.addrs[id.String()]
	if !ok {
		return nil, fmt.Errorf("unknown node: %s", id)
	}
	return rpc.DialWebsocket(context.Background(), addr, "http://localhost")
}
