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
	kad  *kademlia.Kademlia
	path string
}

func newHive(address common.Hash, hivepath string) *hive {
	return &hive{
		path: hivepath,
		kad:  kademlia.New(kademlia.Address(address)),
	}
}

func (self *hive) start() (err error) {
	self.kad.Start()
	err = self.kad.Load(self.path)
	if err != nil {
		dpaLogger.Warnf("Warning: error reading kademlia node db (skipping): %v", err)
		err = nil
	}
	// go func() {
	// 	for {
	// 		select {
	// 		case <-timer:
	// 		case <-subscr:
	// 		}
	// 		maxpeers := 4
	// 		self.getPeerEntries(maxpeers)
	// 	}
	// }()
	return
}

func (self *hive) addPeer(p peer) {
	self.kad.AddNode(p)
}

func (self *hive) removePeer(p peer) {
	self.kad.RemoveNode(p)
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

// called to ask periodically for preferences
// Kademlia ideally maintains a queue of prioritized nodes
func (self *hive) getPeerEntries(max int) (resp *peersMsgData, err error) {
	nrs, err := self.kad.GetNodeRecords(max)
	for _, n := range nrs {
		_ = n
		// resp // build response from kademlia noderecords
	}
	return
}
