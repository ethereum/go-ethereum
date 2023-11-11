package discover

import (
	"crypto/rand"
	"fmt"
	"github.com/optimism-java/utp-go"
	"sync"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/internal/testlog"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/p2p/discover/portalwire"
	"github.com/ethereum/go-ethereum/p2p/enode"
	"github.com/stretchr/testify/assert"
	"golang.org/x/exp/slices"
)

type MockStorage struct {
	db map[string][]byte
}

func (m *MockStorage) ContentId(contentKey []byte) []byte {
	return crypto.Keccak256(contentKey)
}

func (m *MockStorage) Get(contentKey []byte, contentId []byte) ([]byte, error) {
	if content, ok := m.db[string(contentId)]; ok {
		return content, nil
	}
	return nil, ContentNotFound
}

func (m *MockStorage) Put(contentKey []byte, content []byte) error {
	m.db[string(m.ContentId(contentKey))] = content
	return nil
}

func setupLocalPortalNode(addr string, bootNodes []*enode.Node) (*PortalProtocol, error) {
	conf := DefaultPortalProtocolConfig()
	if addr != "" {
		conf.ListenAddr = addr
	}
	if bootNodes != nil {
		conf.BootstrapNodes = bootNodes
	}
	portalProtocol, err := NewPortalProtocol(conf, portalwire.HistoryNetwork, newkey(), &MockStorage{db: make(map[string][]byte)})
	if err != nil {
		return nil, err
	}

	return portalProtocol, nil
}

func TestPortalWireProtocolUdp(t *testing.T) {
	node1, err := setupLocalPortalNode(":7777", nil)
	assert.NoError(t, err)
	node1.log = testlog.Logger(t, log.LvlTrace)
	err = node1.Start()
	assert.NoError(t, err)

	node2, err := setupLocalPortalNode(":7778", []*enode.Node{node1.localNode.Node()})
	assert.NoError(t, err)
	node2.log = testlog.Logger(t, log.LvlTrace)
	err = node2.Start()
	assert.NoError(t, err)

	node3, err := setupLocalPortalNode(":7779", []*enode.Node{node1.localNode.Node()})
	assert.NoError(t, err)
	node3.log = testlog.Logger(t, log.LvlTrace)
	err = node3.Start()
	assert.NoError(t, err)
	time.Sleep(10 * time.Second)

	assert.Equal(t, 2, len(node1.table.Nodes()))
	assert.Equal(t, 2, len(node2.table.Nodes()))
	assert.Equal(t, 2, len(node3.table.Nodes()))

	rAddr, _ := utp.ResolveUTPAddr("utp", "127.0.0.1:7777")
	lAddr, _ := utp.ResolveUTPAddr("utp", "127.0.0.1:7778")

	var wg sync.WaitGroup
	wg.Add(4)

	cid := uint32(12)
	cliSendMsgWithCid := "there are connection id : 12!"
	cliSendMsgWithRandomCid := "there are connection id: random!"

	serverEchoWithCid := "accept connection sends back msg: echo"
	serverEchoWithRandomCid := "ccept connection with random cid sends msg: echo"
	go func() {
		var acceptConn *utp.Conn
		defer func() {
			wg.Done()
			_ = acceptConn.Close()
		}()
		acceptConn, err := node1.utp.AcceptUTPWithConnId(cid)
		if err != nil {
			panic(err)
		}
		buf := make([]byte, 100)
		n, err := acceptConn.Read(buf)
		if err != nil {
			panic(err)
		}
		assert.Equal(t, cliSendMsgWithCid, string(buf[:n]))
		acceptConn.Write([]byte(serverEchoWithCid))
	}()
	go func() {
		defer wg.Done()
		randomConnIdConn, err := node1.utp.Accept()
		if err != nil {
			panic(err)
		}
		buf := make([]byte, 100)
		n, err := randomConnIdConn.Read(buf)
		if err != nil {
			panic(err)
		}
		assert.Equal(t, cliSendMsgWithRandomCid, string(buf[:n]))
		randomConnIdConn.Write([]byte(serverEchoWithRandomCid))
	}()

	go func() {
		defer wg.Done()
		connWithConnId, err := utp.DialUTPOptions("utp", lAddr, rAddr, utp.WithConnId(cid), utp.WithSocketManager(node2.utpSm))
		if err != nil {
			panic(err)
		}
		_, err = connWithConnId.Write([]byte("there are connection id : 12!"))
		if err != nil {
			panic(err)
		}
		buf := make([]byte, 100)
		n, err := connWithConnId.Read(buf)
		if err != nil {
			panic(err)
		}
		assert.Equal(t, serverEchoWithCid, string(buf[:n]))
	}()
	go func() {
		defer wg.Done()
		randomConnIdConn, err := utp.DialUTPOptions("utp", lAddr, rAddr, utp.WithSocketManager(node2.utpSm))
		if err != nil {
			panic(err)
		}
		_, err = randomConnIdConn.Write([]byte(cliSendMsgWithRandomCid))
		if err != nil {
			panic(err)
		}
		buf := make([]byte, 100)
		n, err := randomConnIdConn.Read(buf)
		if err != nil {
			panic(err)
		}
		assert.Equal(t, serverEchoWithRandomCid, string(buf[:n]))
	}()
	wg.Wait()
}

func TestPortalWireProtocol(t *testing.T) {
	node1, err := setupLocalPortalNode(":7777", nil)
	assert.NoError(t, err)
	node1.log = testlog.Logger(t, log.LvlTrace)
	err = node1.Start()
	assert.NoError(t, err)
	fmt.Println(node1.localNode.Node().String())

	node2, err := setupLocalPortalNode(":7778", []*enode.Node{node1.localNode.Node()})
	assert.NoError(t, err)
	node2.log = testlog.Logger(t, log.LvlTrace)
	err = node2.Start()
	assert.NoError(t, err)
	fmt.Println(node2.localNode.Node().String())

	node3, err := setupLocalPortalNode(":7779", []*enode.Node{node1.localNode.Node()})
	assert.NoError(t, err)
	node3.log = testlog.Logger(t, log.LvlTrace)
	err = node3.Start()
	assert.NoError(t, err)
	fmt.Println(node3.localNode.Node().String())
	time.Sleep(10 * time.Second)

	assert.Equal(t, 2, len(node1.table.Nodes()))
	assert.Equal(t, 2, len(node2.table.Nodes()))
	assert.Equal(t, 2, len(node3.table.Nodes()))

	slices.ContainsFunc(node1.table.Nodes(), func(n *enode.Node) bool {
		return n.ID() == node2.localNode.Node().ID()
	})
	slices.ContainsFunc(node1.table.Nodes(), func(n *enode.Node) bool {
		return n.ID() == node3.localNode.Node().ID()
	})

	slices.ContainsFunc(node2.table.Nodes(), func(n *enode.Node) bool {
		return n.ID() == node1.localNode.Node().ID()
	})
	slices.ContainsFunc(node2.table.Nodes(), func(n *enode.Node) bool {
		return n.ID() == node3.localNode.Node().ID()
	})

	slices.ContainsFunc(node3.table.Nodes(), func(n *enode.Node) bool {
		return n.ID() == node1.localNode.Node().ID()
	})
	slices.ContainsFunc(node3.table.Nodes(), func(n *enode.Node) bool {
		return n.ID() == node2.localNode.Node().ID()
	})

	err = node1.storage.Put([]byte("test_key"), []byte("test_value"))
	assert.NoError(t, err)

	flag, content, err := node2.findContent(node1.localNode.Node(), []byte("test_key"))
	assert.NoError(t, err)
	assert.Equal(t, portalwire.ContentRawSelector, flag)
	assert.Equal(t, []byte("test_value"), content)

	flag, content, err = node2.findContent(node3.localNode.Node(), []byte("test_key"))
	assert.NoError(t, err)
	assert.Equal(t, portalwire.ContentEnrsSelector, flag)
	assert.Equal(t, 1, len(content.([]*enode.Node)))
	assert.Equal(t, node1.localNode.Node().ID(), content.([]*enode.Node)[0].ID())

	// create a byte slice of length 1199 and fill it with random data
	// this will be used as a test content
	largeTestContent := make([]byte, 1199)
	_, err = rand.Read(largeTestContent)
	assert.NoError(t, err)

	err = node1.storage.Put([]byte("large_test_key"), largeTestContent)
	assert.NoError(t, err)

	flag, content, err = node2.findContent(node1.localNode.Node(), []byte("large_test_key"))
	assert.NoError(t, err)
	assert.Equal(t, portalwire.ContentConnIdSelector, flag)
	assert.Equal(t, largeTestContent, content)
}
