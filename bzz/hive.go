package bzz

import (
	// "fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/kademlia"
)

type peer struct {
	*bzzProtocol
}

// peer not necessary here
// bzz protocol could implement kademlia.Node interface with
// Addr(), LastActive() and Drop()

// Hive is the logistic manager of the swarm
// it uses a generic kademlia nodetable to find best peer list
// for any target
// this is used by the netstore to search for content in the swarm
// the bzz protocol peersMsgData exchange is relayed to Kademlia
// for db storage and filtering
// connections and disconnections are reported and relayed
// to keep the nodetable uptodate

type hive struct {
	addr kademlia.Address
	kad  *kademlia.Kademlia
	path string
	ping chan bool
}

func newHive(hivepath string) *hive {
	return &hive{
		path: hivepath,
		kad:  kademlia.New(),
	}
}

func (self *hive) start(address kademlia.Address, connectPeer func(string) error) (err error) {
	self.ping = make(chan bool)
	self.addr = address
	self.kad.Start(address)
	err = self.kad.Load(self.path)
	if err != nil {
		dpaLogger.Warnf("Warning: error reading kademlia node db (skipping): %v", err)
		err = nil
	}
	go func() {
		// whenever pinged ask kademlia about most preferred peer
		for _ = range self.ping {
			node, full := self.kad.GetNodeRecord()
			if node != nil {
				// if Url known, connect to peer
				if len(node.Url) > 0 {
					dpaLogger.Debugf("hive: attempt to connect kaddb node %v", node)
					connectPeer(node.Url)
				} else if !full {
					// a random peer is taken from the table
					peers := self.kad.GetNodes(kademlia.RandomAddress(), 1)
					if len(peers) > 0 {
						// a random address at prox bin 0 is sent for lookup
						req := &retrieveRequestMsgData{
							Key: Key(common.Hash(kademlia.RandomAddressAt(self.addr, 0)).Bytes()),
						}
						dpaLogger.Debugf("hive: look up random address with prox order 0 from peer %v", peers[0])
						peers[0].(peer).retrieve(req)
					}
				}
			}
		}
	}()
	return
}

func (self *hive) stop() error {
	// closing ping channel quits the updateloop
	close(self.ping)
	return self.kad.Stop(self.path)
}

func (self *hive) addPeer(p peer) {
	dpaLogger.Debugf("hive: add peer %v", p)
	self.kad.AddNode(p)
	// self lookup
	req := &retrieveRequestMsgData{
		Key: Key(common.Hash(self.addr).Bytes()),
	}
	p.retrieve(req)
	self.ping <- true
}

func (self *hive) removePeer(p peer) {
	dpaLogger.Debugf("hive: remove peer %v", p)
	self.kad.RemoveNode(p)
	self.ping <- false
}

// Retrieve a list of live peers that are closer to target than us
func (self *hive) getPeers(target Key, max int) (peers []peer) {
	var addr kademlia.Address
	copy(addr[:], target[:])
	for _, node := range self.kad.GetNodes(addr, max) {
		peers = append(peers, node.(peer))
	}
	return
}

func newNodeRecord(addr *peerAddr) *kademlia.NodeRecord {
	return &kademlia.NodeRecord{
		Address: addr.addr(),
		Active:  0,
		Url:     addr.url(),
	}
}

// called by the protocol upon receiving peerset (for target address)
// peersMsgData is converted to a slice of NodeRecords for Kademlia
// this is to store all thats needed
func (self *hive) addPeerEntries(req *peersMsgData) {
	var nrs []*kademlia.NodeRecord
	for _, p := range req.Peers {
		nrs = append(nrs, newNodeRecord(p))
	}
	self.kad.AddNodeRecords(nrs)
}
