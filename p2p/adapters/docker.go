package adapters

import (
	"fmt"
	"net"
	// "net/http"
	// "time"
)

func NewRemoteNode(id *NodeId, n Network) *RemoteNode {
	return &RemoteNode{
		ID:      id,
		Network: n,
	}
}

// RemoteNode is the network adapter that
type RemoteNode struct {
	ID   *NodeId
	addr net.Addr
	Network
}

func Name(id []byte) string {
	return fmt.Sprintf("test-%08x", id)
}

// inject(s) sends an RPC command remotely via ssh to the particular dockernode
func (self *RemoteNode) inject(string) error {
	return nil
}

func (self *RemoteNode) LocalAddr() []byte {
	return []byte(self.addr.String())
}

func (self *RemoteNode) ParseAddr(p []byte, s string) ([]byte, error) {
	return p, nil
}

func (self *RemoteNode) Disconnect(rid []byte) error {
	// ssh+ipc -> drop
	// assumes the remote node is running the p2p module as part of the protocol
	cmd := fmt.Sprintf(`p2p.Drop("%v")`, string(rid))
	return self.inject(cmd)
}

func (self *RemoteNode) Connect(rid []byte) error {
	// ssh+ipc -> connect
	//
	cmd := fmt.Sprintf(`admin.addPeer("%v")`, string(rid))
	return self.inject(cmd)
}
