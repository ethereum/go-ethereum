// Copyright 2017 The go-ethereum Authors
// This file is part of go-ethereum.
//
// go-ethereum is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// go-ethereum is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with go-ethereum. If not, see <http://www.gnu.org/licenses/>.

package main

import (
	"context"
	"crypto/ecdsa"
	"flag"
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"sync"
	"syscall"
	"testing"
	"time"

	"github.com/docker/docker/pkg/reexec"
	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/ethereum/go-ethereum/internal/cmdtest"
	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/ethereum/go-ethereum/swarm"
	"github.com/ethereum/go-ethereum/swarm/api"
	swarmhttp "github.com/ethereum/go-ethereum/swarm/api/http"
)

var loglevel = flag.Int("loglevel", 3, "verbosity of logs")

func init() {
	// Run the app if we've been exec'd as "swarm-test" in runSwarm.
	reexec.Register("swarm-test", func() {
		if err := app.Run(os.Args); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		os.Exit(0)
	})
}

const clusterSize = 3

var clusteronce sync.Once
var cluster *testCluster

func initCluster(t *testing.T) {
	clusteronce.Do(func() {
		cluster = newTestCluster(t, clusterSize)
	})
}

func serverFunc(api *api.API) swarmhttp.TestServer {
	return swarmhttp.NewServer(api, "")
}
func TestMain(m *testing.M) {
	// check if we have been reexec'd
	if reexec.Init() {
		return
	}
	os.Exit(m.Run())
}

func runSwarm(t *testing.T, args ...string) *cmdtest.TestCmd {
	tt := cmdtest.NewTestCmd(t, nil)

	found := false
	for _, v := range args {
		if v == "--bootnodes" {
			found = true
			break
		}
	}

	if !found {
		args = append([]string{"--bootnodes", ""}, args...)
	}

	// Boot "swarm". This actually runs the test binary but the TestMain
	// function will prevent any tests from running.
	tt.Run("swarm-test", args...)

	return tt
}

type testCluster struct {
	Nodes  []*testNode
	TmpDir string
}

// newTestCluster starts a test swarm cluster of the given size.
//
// A temporary directory is created and each node gets a data directory inside
// it.
//
// Each node listens on 127.0.0.1 with random ports for both the HTTP and p2p
// ports (assigned by first listening on 127.0.0.1:0 and then passing the ports
// as flags).
//
// When starting more than one node, they are connected together using the
// admin SetPeer RPC method.

func newTestCluster(t *testing.T, size int) *testCluster {
	cluster := &testCluster{}
	defer func() {
		if t.Failed() {
			cluster.Shutdown()
		}
	}()

	tmpdir, err := ioutil.TempDir("", "swarm-test")
	if err != nil {
		t.Fatal(err)
	}
	cluster.TmpDir = tmpdir

	// start the nodes
	cluster.StartNewNodes(t, size)

	if size == 1 {
		return cluster
	}

	// connect the nodes together
	for _, node := range cluster.Nodes {
		if err := node.Client.Call(nil, "admin_addPeer", cluster.Nodes[0].Enode); err != nil {
			t.Fatal(err)
		}
	}

	// wait until all nodes have the correct number of peers
outer:
	for _, node := range cluster.Nodes {
		var peers []*p2p.PeerInfo
		for start := time.Now(); time.Since(start) < time.Minute; time.Sleep(50 * time.Millisecond) {
			if err := node.Client.Call(&peers, "admin_peers"); err != nil {
				t.Fatal(err)
			}
			if len(peers) == len(cluster.Nodes)-1 {
				continue outer
			}
		}
		t.Fatalf("%s only has %d / %d peers", node.Name, len(peers), len(cluster.Nodes)-1)
	}

	return cluster
}

func (c *testCluster) Shutdown() {
	for _, node := range c.Nodes {
		node.Shutdown()
	}
	os.RemoveAll(c.TmpDir)
}

func (c *testCluster) Stop() {
	for _, node := range c.Nodes {
		node.Shutdown()
	}
}

func (c *testCluster) StartNewNodes(t *testing.T, size int) {
	c.Nodes = make([]*testNode, 0, size)
	for i := 0; i < size; i++ {
		dir := filepath.Join(c.TmpDir, fmt.Sprintf("swarm%02d", i))
		if err := os.Mkdir(dir, 0700); err != nil {
			t.Fatal(err)
		}

		node := newTestNode(t, dir)
		node.Name = fmt.Sprintf("swarm%02d", i)

		c.Nodes = append(c.Nodes, node)
	}
}

func (c *testCluster) StartExistingNodes(t *testing.T, size int, bzzaccount string) {
	c.Nodes = make([]*testNode, 0, size)
	for i := 0; i < size; i++ {
		dir := filepath.Join(c.TmpDir, fmt.Sprintf("swarm%02d", i))
		node := existingTestNode(t, dir, bzzaccount)
		node.Name = fmt.Sprintf("swarm%02d", i)

		c.Nodes = append(c.Nodes, node)
	}
}

func (c *testCluster) Cleanup() {
	os.RemoveAll(c.TmpDir)
}

type testNode struct {
	Name       string
	Addr       string
	URL        string
	Enode      string
	Dir        string
	IpcPath    string
	PrivateKey *ecdsa.PrivateKey
	Client     *rpc.Client
	Cmd        *cmdtest.TestCmd
}

const testPassphrase = "swarm-test-passphrase"

func getTestAccount(t *testing.T, dir string) (conf *node.Config, account accounts.Account) {
	// create key
	conf = &node.Config{
		DataDir: dir,
		IPCPath: "bzzd.ipc",
		NoUSB:   true,
	}
	n, err := node.New(conf)
	if err != nil {
		t.Fatal(err)
	}
	account, err = n.AccountManager().Backends(keystore.KeyStoreType)[0].(*keystore.KeyStore).NewAccount(testPassphrase)
	if err != nil {
		t.Fatal(err)
	}

	// use a unique IPCPath when running tests on Windows
	if runtime.GOOS == "windows" {
		conf.IPCPath = fmt.Sprintf("bzzd-%s.ipc", account.Address.String())
	}

	return conf, account
}

func existingTestNode(t *testing.T, dir string, bzzaccount string) *testNode {
	conf, _ := getTestAccount(t, dir)
	node := &testNode{Dir: dir}

	// use a unique IPCPath when running tests on Windows
	if runtime.GOOS == "windows" {
		conf.IPCPath = fmt.Sprintf("bzzd-%s.ipc", bzzaccount)
	}

	// assign ports
	ports, err := getAvailableTCPPorts(2)
	if err != nil {
		t.Fatal(err)
	}
	p2pPort := ports[0]
	httpPort := ports[1]

	// start the node
	node.Cmd = runSwarm(t,
		"--bootnodes", "",
		"--port", p2pPort,
		"--nat", "extip:127.0.0.1",
		"--datadir", dir,
		"--ipcpath", conf.IPCPath,
		"--ens-api", "",
		"--bzzaccount", bzzaccount,
		"--bzznetworkid", "321",
		"--bzzport", httpPort,
		"--verbosity", fmt.Sprint(*loglevel),
	)
	node.Cmd.InputLine(testPassphrase)
	defer func() {
		if t.Failed() {
			node.Shutdown()
		}
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// ensure that all ports have active listeners
	// so that the next node will not get the same
	// when calling getAvailableTCPPorts
	err = waitTCPPorts(ctx, ports...)
	if err != nil {
		t.Fatal(err)
	}

	// wait for the node to start
	for start := time.Now(); time.Since(start) < 10*time.Second; time.Sleep(50 * time.Millisecond) {
		node.Client, err = rpc.Dial(conf.IPCEndpoint())
		if err == nil {
			break
		}
	}
	if node.Client == nil {
		t.Fatal(err)
	}

	// load info
	var info swarm.Info
	if err := node.Client.Call(&info, "bzz_info"); err != nil {
		t.Fatal(err)
	}
	node.Addr = net.JoinHostPort("127.0.0.1", info.Port)
	node.URL = "http://" + node.Addr

	var nodeInfo p2p.NodeInfo
	if err := node.Client.Call(&nodeInfo, "admin_nodeInfo"); err != nil {
		t.Fatal(err)
	}
	node.Enode = nodeInfo.Enode
	node.IpcPath = conf.IPCPath
	return node
}

func newTestNode(t *testing.T, dir string) *testNode {

	conf, account := getTestAccount(t, dir)
	ks := keystore.NewKeyStore(path.Join(dir, "keystore"), 1<<18, 1)

	pk := decryptStoreAccount(ks, account.Address.Hex(), []string{testPassphrase})

	node := &testNode{Dir: dir, PrivateKey: pk}

	// assign ports
	ports, err := getAvailableTCPPorts(2)
	if err != nil {
		t.Fatal(err)
	}
	p2pPort := ports[0]
	httpPort := ports[1]

	// start the node
	node.Cmd = runSwarm(t,
		"--bootnodes", "",
		"--port", p2pPort,
		"--nat", "extip:127.0.0.1",
		"--datadir", dir,
		"--ipcpath", conf.IPCPath,
		"--ens-api", "",
		"--bzzaccount", account.Address.String(),
		"--bzznetworkid", "321",
		"--bzzport", httpPort,
		"--verbosity", fmt.Sprint(*loglevel),
	)
	node.Cmd.InputLine(testPassphrase)
	defer func() {
		if t.Failed() {
			node.Shutdown()
		}
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// ensure that all ports have active listeners
	// so that the next node will not get the same
	// when calling getAvailableTCPPorts
	err = waitTCPPorts(ctx, ports...)
	if err != nil {
		t.Fatal(err)
	}

	// wait for the node to start
	for start := time.Now(); time.Since(start) < 10*time.Second; time.Sleep(50 * time.Millisecond) {
		node.Client, err = rpc.Dial(conf.IPCEndpoint())
		if err == nil {
			break
		}
	}
	if node.Client == nil {
		t.Fatal(err)
	}

	// load info
	var info swarm.Info
	if err := node.Client.Call(&info, "bzz_info"); err != nil {
		t.Fatal(err)
	}
	node.Addr = net.JoinHostPort("127.0.0.1", info.Port)
	node.URL = "http://" + node.Addr

	var nodeInfo p2p.NodeInfo
	if err := node.Client.Call(&nodeInfo, "admin_nodeInfo"); err != nil {
		t.Fatal(err)
	}
	node.Enode = nodeInfo.Enode
	node.IpcPath = conf.IPCPath
	return node
}

func (n *testNode) Shutdown() {
	if n.Cmd != nil {
		n.Cmd.Kill()
	}
}

// getAvailableTCPPorts returns a set of ports that
// nothing is listening on at the time.
//
// Function assignTCPPort cannot be called in sequence
// and guardantee that the same port will be returned in
// different calls as the listener is closed within the function,
// not after all listeners are started and selected unique
// available ports.
func getAvailableTCPPorts(count int) (ports []string, err error) {
	for i := 0; i < count; i++ {
		l, err := net.Listen("tcp", "127.0.0.1:0")
		if err != nil {
			return nil, err
		}
		// defer close in the loop to be sure the same port will not
		// be selected in the next iteration
		defer l.Close()

		_, port, err := net.SplitHostPort(l.Addr().String())
		if err != nil {
			return nil, err
		}
		ports = append(ports, port)
	}
	return ports, nil
}

// waitTCPPorts blocks until tcp connections can be
// established on all provided ports. It runs all
// ports dialers in parallel, and returns the first
// encountered error.
// See waitTCPPort also.
func waitTCPPorts(ctx context.Context, ports ...string) error {
	var err error
	// mu locks err variable that is assigned in
	// other goroutines
	var mu sync.Mutex

	// cancel is canceling all goroutines
	// when the firs error is returned
	// to prevent unnecessary waiting
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	var wg sync.WaitGroup
	for _, port := range ports {
		wg.Add(1)
		go func(port string) {
			defer wg.Done()

			e := waitTCPPort(ctx, port)

			mu.Lock()
			defer mu.Unlock()
			if e != nil && err == nil {
				err = e
				cancel()
			}
		}(port)
	}
	wg.Wait()

	return err
}

// waitTCPPort blocks until tcp connection can be established
// ona provided port. It has a 3 minute timeout as maximum,
// to prevent long waiting, but it can be shortened with
// a provided context instance. Dialer has a 10 second timeout
// in every iteration, and connection refused error will be
// retried in 100 milliseconds periods.
func waitTCPPort(ctx context.Context, port string) error {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Minute)
	defer cancel()

	for {
		c, err := (&net.Dialer{Timeout: 10 * time.Second}).DialContext(ctx, "tcp", "127.0.0.1:"+port)
		if err != nil {
			if operr, ok := err.(*net.OpError); ok {
				if syserr, ok := operr.Err.(*os.SyscallError); ok && syserr.Err == syscall.ECONNREFUSED {
					time.Sleep(100 * time.Millisecond)
					continue
				}
			}
			return err
		}
		return c.Close()
	}
}
