package main

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/rpc"
)

type gethrpc struct {
	name     string
	rpc      *rpc.Client
	geth     *testgeth
	nodeInfo *p2p.NodeInfo
}

func (g *gethrpc) killAndWait() {
	g.geth.Kill()
	g.geth.WaitExit()
}

func (g *gethrpc) callRPC(result interface{}, method string, args ...interface{}) {
	if err := g.rpc.Call(&result, method, args...); err != nil {
		g.geth.Fatalf("callRPC %v: %v", method, err)
	}
}

func (g *gethrpc) addPeer(peer *gethrpc) {
	g.geth.Logf("%v.addPeer(%v)", g.name, peer.name)
	enode := peer.getNodeInfo().Enode
	peerCh := make(chan *p2p.PeerEvent)
	sub, err := g.rpc.Subscribe(context.Background(), "admin", peerCh, "peerEvents")
	if err != nil {
		g.geth.Fatalf("subscribe %v: %v", g.name, err)
	}
	defer sub.Unsubscribe()
	g.callRPC(nil, "admin_addPeer", enode)
	dur := 14 * time.Second
	timeout := time.After(dur)
	select {
	case ev := <-peerCh:
		g.geth.Logf("%v received event: type=%v, peer=%v", g.name, ev.Type, ev.Peer)
	case err := <-sub.Err():
		g.geth.Fatalf("%v sub error: %v", g.name, err)
	case <-timeout:
		g.geth.Error("timeout adding peer after", dur)
	}
}

// Use this function instead of `g.nodeInfo` directly
func (g *gethrpc) getNodeInfo() *p2p.NodeInfo {
	if g.nodeInfo != nil {
		return g.nodeInfo
	}
	g.nodeInfo = &p2p.NodeInfo{}
	g.callRPC(&g.nodeInfo, "admin_nodeInfo")
	return g.nodeInfo
}

func (g *gethrpc) waitSynced() {
	// Check if it's synced now
	var result interface{}
	g.callRPC(&result, "eth_syncing")
	syncing, ok := result.(bool)
	if ok && !syncing {
		g.geth.Logf("%v already synced", g.name)
		return
	}

	// Actually wait, subscribe to the event
	ch := make(chan interface{})
	sub, err := g.rpc.Subscribe(context.Background(), "eth", ch, "syncing")
	if err != nil {
		g.geth.Fatalf("%v syncing: %v", g.name, err)
	}
	defer sub.Unsubscribe()
	timeout := time.After(4 * time.Second)
	select {
	case ev := <-ch:
		g.geth.Log("'syncing' event", ev)
		syncing, ok := ev.(bool)
		if ok && !syncing {
			break
		}
		g.geth.Log("Other 'syncing' event", ev)
	case err := <-sub.Err():
		g.geth.Fatalf("%v notification: %v", g.name, err)
		break
	case <-timeout:
		g.geth.Fatalf("%v timeout syncing", g.name)
		break
	}
}

func startGethWithIpc(t *testing.T, name string, args ...string) *gethrpc {
	g := &gethrpc{name: name}
	args = append([]string{"--networkid=42", "--port=0", "--nousb"}, args...)
	t.Logf("Starting %v with rpc: %v", name, args)
	g.geth = runGeth(t, args...)
	// wait before we can attach to it. TODO: probe for it properly
	time.Sleep(1 * time.Second)
	var err error
	ipcpath := filepath.Join(g.geth.Datadir, "geth.ipc")
	g.rpc, err = rpc.Dial(ipcpath)
	if err != nil {
		t.Fatalf("%v rpc connect: %v", name, err)
	}
	return g
}

func initGeth(t *testing.T) string {
	g := runGeth(t, "--nousb", "--networkid=42", "init", "./testdata/clique.json")
	datadir := g.Datadir
	g.WaitExit()
	return datadir
}

func startLightServer(t *testing.T) *gethrpc {
	datadir := initGeth(t)
	runGeth(t, "--nousb", "--datadir", datadir, "--password", "./testdata/password.txt", "account", "import", "./testdata/key.prv").WaitExit()
	account := "0x02f0d131f1f97aef08aec6e3291b957d9efe7105"
	server := startGethWithIpc(t, "lightserver", "--allow-insecure-unlock", "--datadir", datadir, "--password", "./testdata/password.txt", "--unlock", account, "--mine", "--light.serve=100", "--light.maxpeers=1", "--nodiscover", "--nat=extip:127.0.0.1")
	return server
}

func startClient(t *testing.T, name string) *gethrpc {
	datadir := initGeth(t)
	return startGethWithIpc(t, name, "--datadir", datadir, "--nodiscover", "--syncmode=light", "--nat=extip:127.0.0.1")
}

func TestPriorityClient(t *testing.T) {
	lightServer := startLightServer(t)
	defer lightServer.killAndWait()

	// Start client and add lightServer as peer
	freeCli := startClient(t, "freeCli")
	defer freeCli.killAndWait()
	freeCli.addPeer(lightServer)

	var peers []*p2p.PeerInfo
	freeCli.callRPC(&peers, "admin_peers")
	if len(peers) != 1 {
		t.Errorf("Expected: # of client peers == 1, actual: %v", len(peers))
		return
	}

	// Set up priority client, get its nodeID, increase its balance on the lightServer
	prioCli := startClient(t, "prioCli")
	defer prioCli.killAndWait()
	// 3_000_000_000 once we move to Go 1.13
	tokens := 3000000000
	lightServer.callRPC(nil, "les_addBalance", prioCli.getNodeInfo().ID, tokens)
	prioCli.addPeer(lightServer)

	// Check if priority client is actually syncing and the regular client got kicked out
	prioCli.callRPC(&peers, "admin_peers")
	if len(peers) != 1 {
		t.Errorf("Expected: # of prio peers == 1, actual: %v", len(peers))
	}

	nodes := map[string]*gethrpc{
		lightServer.getNodeInfo().ID: lightServer,
		freeCli.getNodeInfo().ID:     freeCli,
		prioCli.getNodeInfo().ID:     prioCli,
	}
	time.Sleep(1 * time.Second)
	lightServer.callRPC(&peers, "admin_peers")
	peersWithNames := make(map[string]string)
	for _, p := range peers {
		peersWithNames[nodes[p.ID].name] = p.ID
	}
	if _, freeClientFound := peersWithNames[freeCli.name]; freeClientFound {
		t.Error("client is still a peer of lightServer", peersWithNames)
	}
	if _, prioClientFound := peersWithNames[prioCli.name]; !prioClientFound {
		t.Error("prio client is not among lightServer peers", peersWithNames)
	}
}
