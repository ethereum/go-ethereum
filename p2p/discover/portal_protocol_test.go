package discover

import (
	"fmt"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/internal/testlog"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/p2p/discover/portalwire"
	"github.com/ethereum/go-ethereum/p2p/enode"
	"github.com/stretchr/testify/assert"
	"golang.org/x/exp/slices"
)

func setupLocalPortalNode(addr string, bootNodes []*enode.Node) (*PortalProtocol, error) {
	conf := DefaultPortalProtocolConfig()
	if addr != "" {
		conf.ListenAddr = addr
	}
	if bootNodes != nil {
		conf.BootstrapNodes = bootNodes
	}
	portalProtocol, err := NewPortalProtocol(conf, portalwire.HistoryNetwork, newkey())
	if err != nil {
		return nil, err
	}

	return portalProtocol, nil
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
}
