package portalwire

import (
	"context"
	"crypto/ecdsa"
	"crypto/rand"
	"errors"
	"fmt"
	"io"
	"net"
	"sync"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/p2p/discover"
	"github.com/ethereum/go-ethereum/portalnetwork/storage"
	"github.com/optimism-java/utp-go"
	"github.com/optimism-java/utp-go/libutp"
	"github.com/prysmaticlabs/go-bitfield"
	"golang.org/x/exp/slices"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/internal/testlog"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/p2p/enode"
	assert "github.com/stretchr/testify/require"
)

func setupLocalPortalNode(addr string, bootNodes []*enode.Node) (*PortalProtocol, error) {
	conf := DefaultPortalProtocolConfig()
	conf.NAT = nil
	if addr != "" {
		conf.ListenAddr = addr
	}
	if bootNodes != nil {
		conf.BootstrapNodes = bootNodes
	}

	addr1, err := net.ResolveUDPAddr("udp", conf.ListenAddr)
	if err != nil {
		return nil, err
	}
	conn, err := net.ListenUDP("udp", addr1)
	if err != nil {
		return nil, err
	}

	privKey := newkey()

	discCfg := discover.Config{
		PrivateKey:  privKey,
		NetRestrict: conf.NetRestrict,
		Bootnodes:   conf.BootstrapNodes,
	}

	nodeDB, err := enode.OpenDB(conf.NodeDBPath)
	if err != nil {
		return nil, err
	}

	localNode := enode.NewLocalNode(nodeDB, privKey)
	localNode.SetFallbackIP(net.IP{127, 0, 0, 1})
	localNode.Set(Tag)

	if conf.NAT == nil {
		var addrs []net.Addr
		addrs, err = net.InterfaceAddrs()

		if err != nil {
			return nil, err
		}

		for _, address := range addrs {
			// check ip addr is loopback addr
			if ipnet, ok := address.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
				if ipnet.IP.To4() != nil {
					localNode.SetStaticIP(ipnet.IP)
					break
				}
			}
		}
	}

	discV5, err := discover.ListenV5(conn, localNode, discCfg)
	if err != nil {
		return nil, err
	}
	utpSocket := NewPortalUtp(context.Background(), conf, discV5, conn)

	contentQueue := make(chan *ContentElement, 50)
	portalProtocol, err := NewPortalProtocol(
		conf,
		History,
		privKey,
		conn,
		localNode,
		discV5,
		utpSocket,
		&storage.MockStorage{Db: make(map[string][]byte)},
		contentQueue)
	if err != nil {
		return nil, err
	}

	return portalProtocol, nil
}

func TestPortalWireProtocolUdp(t *testing.T) {
	node1, err := setupLocalPortalNode(":8777", nil)
	assert.NoError(t, err)
	node1.Log = testlog.Logger(t, log.LvlTrace)
	err = node1.Start()
	assert.NoError(t, err)

	node2, err := setupLocalPortalNode(":8778", []*enode.Node{node1.localNode.Node()})
	assert.NoError(t, err)
	node2.Log = testlog.Logger(t, log.LvlTrace)
	err = node2.Start()
	assert.NoError(t, err)
	time.Sleep(12 * time.Second)

	node3, err := setupLocalPortalNode(":8779", []*enode.Node{node1.localNode.Node()})
	assert.NoError(t, err)
	node3.Log = testlog.Logger(t, log.LvlTrace)
	err = node3.Start()
	assert.NoError(t, err)
	time.Sleep(12 * time.Second)

	cid1 := libutp.ReceConnId(12)
	cid2 := libutp.ReceConnId(116)
	cliSendMsgWithCid1 := "there are connection id : 12!"
	cliSendMsgWithCid2 := "there are connection id: 116!"

	serverEchoWithCid := "accept connection sends back msg: echo"

	largeTestContent := make([]byte, 1199)
	_, err = rand.Read(largeTestContent)
	assert.NoError(t, err)

	var workGroup sync.WaitGroup
	var acceptGroup sync.WaitGroup
	workGroup.Add(4)
	acceptGroup.Add(1)
	go func() {
		var acceptConn *utp.Conn
		defer func() {
			workGroup.Done()
			_ = acceptConn.Close()
		}()
		acceptConn, err := node1.Utp.AcceptWithCid(context.Background(), node2.localNode.ID(), cid1)
		if err != nil {
			panic(err)
		}
		acceptGroup.Done()
		buf := make([]byte, 100)
		n, err := acceptConn.Read(buf)
		if err != nil && err != io.EOF {
			panic(err)
		}
		assert.Equal(t, cliSendMsgWithCid1, string(buf[:n]))
		_, err = acceptConn.Write([]byte(serverEchoWithCid))
		if err != nil {
			panic(err)
		}
	}()
	go func() {
		var connId2Conn net.Conn
		defer func() {
			workGroup.Done()
			_ = connId2Conn.Close()
		}()
		connId2Conn, err := node1.Utp.AcceptWithCid(context.Background(), node2.localNode.ID(), cid2)
		if err != nil {
			panic(err)
		}
		buf := make([]byte, 100)
		n, err := connId2Conn.Read(buf)
		if err != nil && err != io.EOF {
			panic(err)
		}
		assert.Equal(t, cliSendMsgWithCid2, string(buf[:n]))

		_, err = connId2Conn.Write(largeTestContent)
		if err != nil {
			panic(err)
		}
	}()

	go func() {
		var connWithConnId net.Conn
		defer func() {
			workGroup.Done()
			if connWithConnId != nil {
				_ = connWithConnId.Close()
			}
		}()
		connWithConnId, err = node2.Utp.DialWithCid(context.Background(), node1.localNode.Node(), cid1.SendId())
		if err != nil {
			panic(err)
		}
		_, err = connWithConnId.Write([]byte(cliSendMsgWithCid1))
		if err != nil && err != io.EOF {
			panic(err)
		}
		buf := make([]byte, 100)
		n, err := connWithConnId.Read(buf)
		if err != nil && err != io.EOF {
			panic(err)
		}
		assert.Equal(t, serverEchoWithCid, string(buf[:n]))
	}()
	go func() {
		var ConnId2Conn net.Conn
		defer func() {
			workGroup.Done()
			if ConnId2Conn != nil {
				_ = ConnId2Conn.Close()
			}
		}()
		ConnId2Conn, err = node2.Utp.DialWithCid(context.Background(), node1.localNode.Node(), cid2.SendId())
		if err != nil && err != io.EOF {
			panic(err)
		}
		_, err = ConnId2Conn.Write([]byte(cliSendMsgWithCid2))
		if err != nil {
			panic(err)
		}

		data := make([]byte, 0)
		buf := make([]byte, 1024)
		for {
			var n int
			n, err = ConnId2Conn.Read(buf)
			if err != nil {
				if errors.Is(err, io.EOF) {
					break
				}
			}
			data = append(data, buf[:n]...)
		}
		assert.Equal(t, largeTestContent, data)
	}()
	workGroup.Wait()
	node1.Stop()
	node2.Stop()
	node3.Stop()
}

func TestPortalWireProtocol(t *testing.T) {
	node1, err := setupLocalPortalNode(":7777", nil)
	assert.NoError(t, err)
	node1.Log = testlog.Logger(t, log.LevelDebug)
	err = node1.Start()
	assert.NoError(t, err)

	node2, err := setupLocalPortalNode(":7778", []*enode.Node{node1.localNode.Node()})
	assert.NoError(t, err)
	node2.Log = testlog.Logger(t, log.LevelDebug)
	err = node2.Start()
	assert.NoError(t, err)

	time.Sleep(12 * time.Second)

	node3, err := setupLocalPortalNode(":7779", []*enode.Node{node1.localNode.Node()})
	assert.NoError(t, err)
	node3.Log = testlog.Logger(t, log.LevelDebug)
	err = node3.Start()
	assert.NoError(t, err)

	time.Sleep(12 * time.Second)

	slices.ContainsFunc(node1.table.NodeList(), func(n *enode.Node) bool {
		return n.ID() == node2.localNode.Node().ID()
	})
	slices.ContainsFunc(node1.table.NodeList(), func(n *enode.Node) bool {
		return n.ID() == node3.localNode.Node().ID()
	})

	slices.ContainsFunc(node2.table.NodeList(), func(n *enode.Node) bool {
		return n.ID() == node1.localNode.Node().ID()
	})
	slices.ContainsFunc(node2.table.NodeList(), func(n *enode.Node) bool {
		return n.ID() == node3.localNode.Node().ID()
	})

	slices.ContainsFunc(node3.table.NodeList(), func(n *enode.Node) bool {
		return n.ID() == node1.localNode.Node().ID()
	})
	slices.ContainsFunc(node3.table.NodeList(), func(n *enode.Node) bool {
		return n.ID() == node2.localNode.Node().ID()
	})

	err = node1.storage.Put(nil, node1.toContentId([]byte("test_key")), []byte("test_value"))
	assert.NoError(t, err)

	flag, content, err := node2.findContent(node1.localNode.Node(), []byte("test_key"))
	assert.NoError(t, err)
	assert.Equal(t, ContentRawSelector, flag)
	assert.Equal(t, []byte("test_value"), content)

	flag, content, err = node2.findContent(node3.localNode.Node(), []byte("test_key"))
	assert.NoError(t, err)
	assert.Equal(t, ContentEnrsSelector, flag)
	assert.Equal(t, 1, len(content.([]*enode.Node)))
	assert.Equal(t, node1.localNode.Node().ID(), content.([]*enode.Node)[0].ID())

	// create a byte slice of length 1199 and fill it with random data
	// this will be used as a test content
	largeTestContent := make([]byte, 2000)
	_, err = rand.Read(largeTestContent)
	assert.NoError(t, err)

	err = node1.storage.Put(nil, node1.toContentId([]byte("large_test_key")), largeTestContent)
	assert.NoError(t, err)

	flag, content, err = node2.findContent(node1.localNode.Node(), []byte("large_test_key"))
	assert.NoError(t, err)
	assert.Equal(t, largeTestContent, content)
	assert.Equal(t, ContentConnIdSelector, flag)

	testEntry1 := &ContentEntry{
		ContentKey: []byte("test_entry1"),
		Content:    []byte("test_entry1_content"),
	}

	testEntry2 := &ContentEntry{
		ContentKey: []byte("test_entry2"),
		Content:    []byte("test_entry2_content"),
	}

	testTransientOfferRequest := &TransientOfferRequest{
		Contents: []*ContentEntry{testEntry1, testEntry2},
	}

	offerRequest := &OfferRequest{
		Kind:    TransientOfferRequestKind,
		Request: testTransientOfferRequest,
	}

	contentKeys, err := node1.offer(node3.localNode.Node(), offerRequest)
	assert.Equal(t, uint64(2), bitfield.Bitlist(contentKeys).Count())
	assert.NoError(t, err)

	contentElement := <-node3.contentQueue
	assert.Equal(t, node1.localNode.Node().ID(), contentElement.Node)
	assert.Equal(t, testEntry1.ContentKey, contentElement.ContentKeys[0])
	assert.Equal(t, testEntry1.Content, contentElement.Contents[0])
	assert.Equal(t, testEntry2.ContentKey, contentElement.ContentKeys[1])
	assert.Equal(t, testEntry2.Content, contentElement.Contents[1])

	testGossipContentKeys := [][]byte{[]byte("test_gossip_content_keys"), []byte("test_gossip_content_keys2")}
	testGossipContent := [][]byte{[]byte("test_gossip_content"), []byte("test_gossip_content2")}
	id := node1.Self().ID()
	gossip, err := node1.Gossip(&id, testGossipContentKeys, testGossipContent)
	assert.NoError(t, err)
	assert.Equal(t, 2, gossip)

	contentElement = <-node2.contentQueue
	assert.Equal(t, node1.localNode.Node().ID(), contentElement.Node)
	assert.Equal(t, testGossipContentKeys[0], contentElement.ContentKeys[0])
	assert.Equal(t, testGossipContent[0], contentElement.Contents[0])
	assert.Equal(t, testGossipContentKeys[1], contentElement.ContentKeys[1])
	assert.Equal(t, testGossipContent[1], contentElement.Contents[1])

	contentElement = <-node3.contentQueue
	assert.Equal(t, node1.localNode.Node().ID(), contentElement.Node)
	assert.Equal(t, testGossipContentKeys[0], contentElement.ContentKeys[0])
	assert.Equal(t, testGossipContent[0], contentElement.Contents[0])
	assert.Equal(t, testGossipContentKeys[1], contentElement.ContentKeys[1])
	assert.Equal(t, testGossipContent[1], contentElement.Contents[1])

	node1.Stop()
	node2.Stop()
	node3.Stop()
}

func TestCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	go func(ctx context.Context) {
		defer func() {
			t.Log("goroutine cancel")
		}()

		time.Sleep(time.Second * 5)
	}(ctx)

	cancel()
	t.Log("after main cancel")

	time.Sleep(time.Second * 3)
}

func TestContentLookup(t *testing.T) {
	node1, err := setupLocalPortalNode(":17777", nil)
	assert.NoError(t, err)
	node1.Log = testlog.Logger(t, log.LvlTrace)
	err = node1.Start()
	assert.NoError(t, err)

	node2, err := setupLocalPortalNode(":17778", []*enode.Node{node1.localNode.Node()})
	assert.NoError(t, err)
	node2.Log = testlog.Logger(t, log.LvlTrace)
	err = node2.Start()
	assert.NoError(t, err)
	fmt.Println(node2.localNode.Node().String())

	node3, err := setupLocalPortalNode(":17779", []*enode.Node{node1.localNode.Node()})
	assert.NoError(t, err)
	node3.Log = testlog.Logger(t, log.LvlTrace)
	err = node3.Start()
	assert.NoError(t, err)

	defer func() {
		node1.Stop()
		node2.Stop()
		node3.Stop()
	}()

	contentKey := []byte{0x3, 0x4}
	content := []byte{0x1, 0x2}
	contentId := node1.toContentId(contentKey)

	err = node3.storage.Put(nil, contentId, content)
	assert.NoError(t, err)

	res, _, err := node1.ContentLookup(contentKey, contentId)
	assert.NoError(t, err)
	assert.Equal(t, res, content)

	nonExist := []byte{0x2, 0x4}
	res, _, err = node1.ContentLookup(nonExist, node1.toContentId(nonExist))
	assert.Equal(t, ContentNotFound, err)
	assert.Nil(t, res)
}

func TestTraceContentLookup(t *testing.T) {
	node1, err := setupLocalPortalNode(":17787", nil)
	assert.NoError(t, err)
	node1.Log = testlog.Logger(t, log.LvlTrace)
	err = node1.Start()
	assert.NoError(t, err)

	node2, err := setupLocalPortalNode(":17788", []*enode.Node{node1.localNode.Node()})
	assert.NoError(t, err)
	node2.Log = testlog.Logger(t, log.LvlTrace)
	err = node2.Start()
	assert.NoError(t, err)

	node3, err := setupLocalPortalNode(":17789", []*enode.Node{node2.localNode.Node()})
	assert.NoError(t, err)
	node3.Log = testlog.Logger(t, log.LvlTrace)
	err = node3.Start()
	assert.NoError(t, err)

	defer node1.Stop()
	defer node2.Stop()
	defer node3.Stop()

	contentKey := []byte{0x3, 0x4}
	content := []byte{0x1, 0x2}
	contentId := node1.toContentId(contentKey)

	err = node1.storage.Put(nil, contentId, content)
	assert.NoError(t, err)

	node1Id := hexutil.Encode(node1.Self().ID().Bytes())
	node2Id := hexutil.Encode(node2.Self().ID().Bytes())
	node3Id := hexutil.Encode(node3.Self().ID().Bytes())

	res, err := node3.TraceContentLookup(contentKey, contentId)
	assert.NoError(t, err)
	assert.Equal(t, res.Content, hexutil.Encode(content))
	assert.Equal(t, res.UtpTransfer, false)
	assert.Equal(t, res.Trace.Origin, node3Id)
	assert.Equal(t, res.Trace.TargetId, hexutil.Encode(contentId))
	assert.Equal(t, res.Trace.ReceivedFrom, node1Id)

	// check nodeMeta
	node1Meta := res.Trace.Metadata[node1Id]
	assert.Equal(t, node1Meta.Enr, node1.Self().String())
	dis := node1.Distance(node1.Self().ID(), enode.ID(contentId))
	assert.Equal(t, node1Meta.Distance, hexutil.Encode(dis[:]))

	node2Meta := res.Trace.Metadata[node2Id]
	assert.Equal(t, node2Meta.Enr, node2.Self().String())
	dis = node2.Distance(node2.Self().ID(), enode.ID(contentId))
	assert.Equal(t, node2Meta.Distance, hexutil.Encode(dis[:]))

	node3Meta := res.Trace.Metadata[node3Id]
	assert.Equal(t, node3Meta.Enr, node3.Self().String())
	dis = node3.Distance(node3.Self().ID(), enode.ID(contentId))
	assert.Equal(t, node3Meta.Distance, hexutil.Encode(dis[:]))

	// check response
	node3Response := res.Trace.Responses[node3Id]
	assert.Equal(t, node3Response.RespondedWith, []string{node2Id})

	node2Response := res.Trace.Responses[node2Id]
	assert.Equal(t, node2Response.RespondedWith, []string{node1Id})

	node1Response := res.Trace.Responses[node1Id]
	assert.Equal(t, node1Response.RespondedWith, ([]string)(nil))
}

func newkey() *ecdsa.PrivateKey {
	key, err := crypto.GenerateKey()
	if err != nil {
		panic("couldn't generate key: " + err.Error())
	}
	return key
}
