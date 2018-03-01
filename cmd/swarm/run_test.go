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
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"path/filepath"
	"runtime"
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
)

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

func TestMain(m *testing.M) {
	// check if we have been reexec'd
	if reexec.Init() {
		return
	}
	os.Exit(m.Run())
}

func runSwarm(t *testing.T, args ...string) *cmdtest.TestCmd {
	tt := cmdtest.NewTestCmd(t, nil)

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
	cluster.Nodes = make([]*testNode, 0, size)
	for i := 0; i < size; i++ {
		dir := filepath.Join(cluster.TmpDir, fmt.Sprintf("swarm%02d", i))
		if err := os.Mkdir(dir, 0700); err != nil {
			t.Fatal(err)
		}

		node := newTestNode(t, dir)
		node.Name = fmt.Sprintf("swarm%02d", i)

		cluster.Nodes = append(cluster.Nodes, node)
	}

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

type testNode struct {
	Name   string
	Addr   string
	URL    string
	Enode  string
	Dir    string
	Client *rpc.Client
	Cmd    *cmdtest.TestCmd
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

func newTestNode(t *testing.T, dir string) *testNode {

	conf, account := getTestAccount(t, dir)
	node := &testNode{Dir: dir}

	// assign ports
	httpPort, err := assignTCPPort()
	if err != nil {
		t.Fatal(err)
	}
	p2pPort, err := assignTCPPort()
	if err != nil {
		t.Fatal(err)
	}

	// start the node
	node.Cmd = runSwarm(t,
		"--port", p2pPort,
		"--nodiscover",
		"--datadir", dir,
		"--ipcpath", conf.IPCPath,
		"--ens-api", "",
		"--bzzaccount", account.Address.String(),
		"--bzznetworkid", "321",
		"--bzzport", httpPort,
		"--verbosity", "6",
	)
	node.Cmd.InputLine(testPassphrase)
	defer func() {
		if t.Failed() {
			node.Shutdown()
		}
	}()

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
	node.Enode = fmt.Sprintf("enode://%s@127.0.0.1:%s", nodeInfo.ID, p2pPort)

	return node
}

func (n *testNode) Shutdown() {
	if n.Cmd != nil {
		n.Cmd.Kill()
	}
}

func assignTCPPort() (string, error) {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return "", err
	}
	l.Close()
	_, port, err := net.SplitHostPort(l.Addr().String())
	if err != nil {
		return "", err
	}
	return port, nil
}
