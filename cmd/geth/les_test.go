package main

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/rpc"
)

/*
var rpcArgs = []string{"--rpc", "--rpcapi=admin,eth,les"}

var port int = 30303
var rpcPort int = 8545

type gethNode struct {
	datadir string
	args    []string
	cmd     *exec.Cmd
	rpc     *rpc.Client
}

func startGeth(datadir string, keepDatadir bool, args ...string) (*gethNode, error) {
	g := &gethNode{datadir, args, nil, nil}
	if !keepDatadir {
		err := os.RemoveAll(datadir)
		if err != nil {
			return g, err
		}
	}
	allArgs := []string{"--datadir", datadir, "--port", fmt.Sprintf("%d", port), "--rpcport", fmt.Sprintf("%d", rpcPort)}
	port += 1
	rpcPort += 1
	allArgs = append(allArgs, rpcArgs...)
	allArgs = append(allArgs, args...)
	cmd := exec.Command("geth", allArgs...)
	log.Println("to start", cmd.String())
	var err error
	cmd.Stderr = os.Stderr
	err = cmd.Start()
	if err != nil {
		return nil, err
	}
	g.cmd = cmd
	time.Sleep(1 * time.Second) // wait before we can attach to it
	// TODO: probe for it properly
	g.rpc, err = rpc.Dial(g.ipcpath())
	if err != nil {
		return nil, err
	}
	return g, nil
}


// Start and wait for it to finish
func runGeth(datadir string, keepDatadir bool, args ...string) error {
	if !keepDatadir {
		if err := os.RemoveAll(datadir); err != nil {
			return err
		}
	}
	allArgs := []string{"--datadir", datadir}
	allArgs = append(allArgs, args...)
	cmd := exec.Command("geth", allArgs...)
	log.Println("to run", cmd.String())
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("starting but %s", err)
	}
	if err := cmd.Wait(); err != nil {
		return fmt.Errorf("waiting but %s", err)
	}
	return nil
}

func (g *gethNode) ipcpath() string {
	return filepath.Join(g.datadir, "geth.ipc")
}

func (g *gethNode) kill() error {
	err := g.cmd.Process.Kill()
	if err != nil {
		return err
	}
	_, err = g.cmd.Process.Wait()
	return err
}

func (g *gethNode) addPeer(enode string) error {
	peerCh := make(chan *p2p.PeerEvent)
	sub, err := g.rpc.Subscribe(context.Background(), "admin", peerCh, "peerEvents")
	if err != nil {
		return fmt.Errorf("subscribe: %v", err)
	}
	defer sub.Unsubscribe()
	if err := g.rpc.Call(nil, "admin_addPeer", enode); err != nil {
		return fmt.Errorf("admin_addPeer: %v", err)
	}
	select {
	case ev := <-peerCh:
		fmt.Print("event", ev)

	case err := <-sub.Err():
		return fmt.Errorf("notification: %v", err)
	}
	return nil
}

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

func startLightServer(t *testing.T) *rpc.Client {
	// Create a temporary data directory to use
	datadir := tmpdir(t)
	defer os.RemoveAll(datadir)
	ipcpath := filepath.Join(datadir, "geth.ipc")

	t.Log("server datadir", datadir)
	runGeth(t, "--datadir", datadir, "init", "./testdata/genesis.json").WaitExit()
	t.Log("init done")
	runGeth(t, "--datadir", datadir, "--gcmode=archive", "import", "./testdata/blockchain.blocks").WaitExit()
	t.Log("import done")
	geth := runGeth(t, "--datadir", datadir, "--networkid=42", "--port=0", "--rpcport=0", "--rpc", "--rpcapi=admin,eth,les", "--light.serve=100", "--light.maxpeers=1", "--nodiscover", "--nat=extip:127.0.0.1")
	defer geth.WaitExit()
	defer geth.Kill()
	t.Log("started lightserver")

	// wait before we can attach to it. TODO: probe for it properly
	time.Sleep(1 * time.Second)
	rpc, err := rpc.Dial(ipcpath)
	if err != nil {
		t.Fatalf("rpc connect: %v", err)
	}
	return rpc
}

func TestPriorityClient(t *testing.T) {
	// Init and start server
	server := startLightServer(t)
	nodeInfo := make(map[string]interface{})
	if err := server.Call(&nodeInfo, "admin_nodeInfo"); err != nil {
		t.Fatal("nodeInfo:", err)
	}
	enode := nodeInfo["enode"].(string)
	t.Log("enode", enode)
	/*
		if err := runGeth(datadir, true, "--gcmode=archive", "import", "./initdata/testBlockchain.blocks"); err != nil {
			t.Fatal("import", err)
		}
		t.Log("import done")
		server, err := startGeth(datadir, true, "--networkid=42", "--light.serve=100", "--light.maxpeers=1", "--nodiscover", "--nat=extip:127.0.0.1")
		defer server.kill()
		if err != nil {
			t.Fatal("start server", err)
		}
		nodeInfo := make(map[string]interface{})
		if err := server.rpc.Call(&nodeInfo, "admin_nodeInfo"); err != nil {
			t.Fatal("nodeInfo:", err)
		}
		enode := nodeInfo["enode"].(string)
		t.Log("enode", enode)

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
