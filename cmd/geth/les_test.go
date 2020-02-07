package main

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/rpc"
)

/*
func (g *gethNode) waitSynced() error {
	ch := make(chan interface{})
	sub, err := g.rpc.Subscribe(context.Background(), "eth", ch, "syncing")
	if err != nil {
		return fmt.Errorf("syncing: %v", err)
	}
	defer sub.Unsubscribe()
	timeout := time.After(40 * time.Second)
	for {
		select {
		case ev := <-ch:
			syncing, ok := ev.(bool)
			if ok && !syncing {
				return nil
			}
		case err := <-sub.Err():
			return fmt.Errorf("notification: %v", err)
		case <-timeout:
			return fmt.Errorf("timeout syncing")
		}
	}
}
*/

type gethrpc struct {
	name string
	rpc  *rpc.Client
	geth *testgeth
	test *testing.T
}

func (g *gethrpc) killAndWait() {
	g.geth.Kill()
	g.geth.WaitExit()
}

func (g *gethrpc) callRPC(result interface{}, method string, args ...interface{}) {
	if err := g.rpc.Call(&result, method, args...); err != nil {
		g.test.Fatalf("callRPC %v: %v", method, err)
	}
}

func (g *gethrpc) addPeer(enode string) {
	g.test.Log("adding peer:", enode)
	peerCh := make(chan *p2p.PeerEvent)
	sub, err := g.rpc.Subscribe(context.Background(), "admin", peerCh, "peerEvents")
	if err != nil {
		g.test.Fatalf("subscribe %v: %v", g.name, err)
	}
	defer sub.Unsubscribe()
	g.callRPC(nil, "admin_addPeer", enode)
	select {
	case ev := <-peerCh:
		g.test.Logf("%v received event: %v", g.name, ev)
	case err := <-sub.Err():
		g.test.Fatalf("%v sub error: %v", g.name, err)
	}
}

func (g *gethrpc) waitSynced() {
	// Check if it's synced now
	var result interface{}
	g.callRPC(&result, "eth_syncing")
	syncing, ok := result.(bool)
	if ok && !syncing {
		g.test.Logf("%v already synced", g.name)
		return
	}

	// Actually wait, subscribe to the event
	ch := make(chan interface{})
	sub, err := g.rpc.Subscribe(context.Background(), "eth", ch, "syncing")
	if err != nil {
		g.test.Fatalf("%v syncing: %v", g.name, err)
	}
	defer sub.Unsubscribe()
	g.test.Log("subscribed")
	timeout := time.After(4 * time.Second)
	for {
		select {
		case ev := <-ch:
			g.test.Log("'syncing' event", ev)
			syncing, ok := ev.(bool)
			if ok && !syncing {
				return
			}
			g.test.Log("Other 'syncing' event", ev)
		case err := <-sub.Err():
			g.test.Fatalf("%v notification: %v", g.name, err)
			return
		case <-timeout:
			g.test.Fatalf("%v timeout syncing", g.name)
			return
		}
	}
}

func startGethWithRpc(t *testing.T, name string, ipcpath string, args ...string) *gethrpc {
	g := &gethrpc{test: t, name: name}
	args = append([]string{"--verbosity=5"}, args...)
	g.geth = runGeth(t, args...)
	// wait before we can attach to it. TODO: probe for it properly
	time.Sleep(1 * time.Second)
	var err error
	g.rpc, err = rpc.Dial(ipcpath)
	if err != nil {
		t.Fatalf("%v rpc connect: %v", name, err)
	}
	t.Logf("%v rpc dial done", name)
	return g
}

func startLightServer(t *testing.T) *gethrpc {
	// Create a temporary data directory to use
	datadir := tmpdir(t)
	defer os.RemoveAll(datadir)
	ipcpath := filepath.Join(datadir, "geth.ipc")

	t.Log("server datadir", datadir)
	runGeth(t, "--datadir", datadir, "init", "./testdata/genesis.json").WaitExit()
	t.Log("init done")
	runGeth(t, "--datadir", datadir, "--gcmode=archive", "import", "./testdata/blockchain.blocks").WaitExit()
	t.Log("import done")
	g := startGethWithRpc(t, "server", ipcpath, "--datadir", datadir, "--networkid=42", "--port=0", "--rpcport=0", "--rpc", "--rpcapi=admin,eth,les", "--light.serve=100", "--light.maxpeers=1", "--nodiscover", "--nat=extip:127.0.0.1")
	return g
}

func startClient(t *testing.T) *gethrpc {
	// Create a temporary data directory to use
	datadir := tmpdir(t)
	defer os.RemoveAll(datadir)
	ipcpath := filepath.Join(datadir, "geth.ipc")

	runGeth(t, "--datadir", datadir, "init", "./testdata/genesis.json").WaitExit()
	g := startGethWithRpc(t, "client", ipcpath, "--datadir", datadir, "--networkid=42", "--port=0", "--rpcport=0", "--rpc", "--rpcapi=admin,eth,les", "--nodiscover", "--syncmode=light")
	return g
}

func TestPriorityClient(t *testing.T) {
	// Init and start server
	server := startLightServer(t)
	defer server.killAndWait()
	nodeInfo := make(map[string]interface{})
	server.callRPC(&nodeInfo, "admin_nodeInfo")
	enode := nodeInfo["enode"].(string)
	server.waitSynced()

	client := startClient(t)
	defer client.killAndWait()
	client.addPeer(enode)
	var peers []interface{}
	client.callRPC(&peers, "admin_peers")
	if len(peers) != 1 {
		t.Logf("Expected: # of client peers == 1, actual: %v", len(peers))
		t.Fail()
	}

	/*

		// Client
		clientdir := "/tmp/client"
		if err := runGeth(clientdir, false, "init", "./initdata/testGenesis.json"); err != nil {
			t.Fatal("init client", err)
		}
		client, err := startGeth(clientdir, true, "--networkid=42", "--syncmode=light", "--nodiscover")
		defer client.kill()
		if err != nil {
			t.Fatal("start client", err)
		}
		if err := client.addPeer(enode); err != nil {
			t.Fatal("addPeer", err)
		}
		var peers []interface{}
		if err := client.rpc.Call(&peers, "admin_peers"); err != nil {
			t.Fatal("peers", err)
		}
		if len(peers) != 1 {
			t.Log("Expected: # of client peers == 1")
			t.Fail()
		}

		// Priority client
		priodir := "/tmp/prio"
		if err := runGeth(priodir, false, "init", "./initdata/testGenesis.json"); err != nil {
			t.Fatal("init prio", err)
		}
		prio, err := startGeth(priodir, true, "--networkid=42", "--syncmode=light", "--nodiscover")
		defer prio.kill()
		if err != nil {
			t.Fatal(prio.cmd.String(), err)
		}
		prioNodeInfo := make(map[string]interface{})
		if err := prio.rpc.Call(&prioNodeInfo, "admin_nodeInfo"); err != nil {
			t.Fatal("prio nodeInfo:", err)
		}
		nodeID := prioNodeInfo["id"].(string)
		t.Log("nodeID", nodeID)
		tokens := 3_000_000_000
		if err := server.rpc.Call(nil, "les_addBalance", nodeID, tokens, "foobar"); err != nil {
			t.Fatal("addBalance:", err)
		}
		if err := prio.addPeer(enode); err != nil {
			t.Fatal("addPeer", err)
		}

		// Check if priority client is actually syncing and the regular client got kicked out
		if err := prio.rpc.Call(&peers, "admin_peers"); err != nil {
			t.Fatal("prio peers", err)
		}
		if len(peers) != 1 {
			t.Fatal("Expected: # of prio peers == 1")
		}

		if err := client.rpc.Call(&peers, "admin_peers"); err != nil {
			t.Fatal("client peers", err)
		}
		if len(peers) > 0 {
			t.Fatal("Expected: # of client peers == 0")
		}
	*/
}
