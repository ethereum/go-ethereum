package main

import (
	"net"
	"testing"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/p2p/enode"
	"github.com/ethereum/go-ethereum/p2p/nat"
	"github.com/stretchr/testify/assert"
)

func newLocalNodeForTesting() (*enode.LocalNode, *enode.DB) {
	db, _ := enode.OpenDB("")
	key, _ := crypto.GenerateKey()
	return enode.NewLocalNode(db, key), db
}

func TestDoPortMapping(t *testing.T) {
	nat := nat.ExtIP{33, 44, 55, 66}
	localNode, _ := newLocalNodeForTesting()
	listenerAddr := &net.UDPAddr{IP: net.IP{127, 0, 0, 1}, Port: 1234}

	doPortMapping(nat, localNode, listenerAddr)

	assert.Equal(t, localNode.Seq(), uint64(1))
	assert.Equal(t, localNode.Node().IP(), net.IP{33, 44, 55, 66})
	assert.Equal(t, localNode.Node().UDP(), 1234)
	assert.Equal(t, localNode.Node().TCP(), 0)

	_ = localNode.Node().UDP()
	assert.Equal(t, localNode.Seq(), uint64(2))
}
