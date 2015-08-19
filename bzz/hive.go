package bzz

import (
	"math/rand"
	"time"

	"github.com/ethereum/go-ethereum/common/kademlia"
	"github.com/ethereum/go-ethereum/logger"
	"github.com/ethereum/go-ethereum/logger/glog"
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
	more chan bool
}

func newHive() (*hive, error) {
	kad := kademlia.New()
	kad.BucketSize = 3
	kad.MaxProx = 10
	kad.ProxBinSize = 8
	return &hive{
		kad: kad,
	}, nil
}

func (self *hive) start(baseAddr *peerAddr, hivepath string, connectPeer func(string) error) (err error) {
	self.ping = make(chan bool)
	self.more = make(chan bool)
	self.path = hivepath

	self.addr = kademlia.Address(baseAddr.hash)
	self.kad.Start(self.addr)
	err = self.kad.Load(self.path)
	if err != nil {
		glog.V(logger.Warn).Infof("[BZZ] KΛÐΞMLIΛ Warning: error reading kaddb '%s' (skipping): %v", self.path, err)
		err = nil
	}
	/* this loop is doing the actual table maintenance
	including bootstrapping and maintaining a healthy table
	Note: At the moment, this does not have any timer/timeout . That means if your
	peers do not reply to launch the game into movement , it will stay stuck
	add or remove a peer to wake up
	*/
	go self.pinger()
	go func() {
		// whenever pinged ask kademlia about most preferred peer
		for _ = range self.ping {
			node, proxLimit := self.kad.GetNodeRecord()
			if node != nil && len(node.Url) > 0 {
				glog.V(logger.Detail).Infof("[BZZ] KΛÐΞMLIΛ hive: call for bee %v", node)
				// enode or any lower level connection address is unnecessary in future
				// discovery table is used to look it up.
				connectPeer(node.Url)
			} else if proxLimit > -1 {
				// a random peer is taken from the table
				peers := self.kad.GetNodes(kademlia.RandomAddressAt(self.addr, rand.Intn(self.kad.MaxProx)), 1)
				if len(peers) > 0 {
					// a random address at prox bin 0 is sent for lookup
					randAddr := kademlia.RandomAddressAt(self.addr, proxLimit)
					req := &retrieveRequestMsgData{
						Key: Key(randAddr[:]),
					}
					glog.V(logger.Detail).Infof("[BZZ] KΛÐΞMLIΛ hive: call any bee in area %x messenger bee %v", randAddr[:4], peers[0])
					peers[0].(peer).retrieve(req)
				}
				if self.more == nil {
					glog.V(logger.Detail).Infof("[BZZ] KΛÐΞMLIΛ hive: buzz buzz need more bees")
					self.more = make(chan bool)
					go self.pinger()
				}
				self.more <- true
				glog.V(logger.Detail).Infof("[BZZ] KΛÐΞMLIΛ hive: buzz kept alive")
			} else {
				if self.more != nil {
					close(self.more)
					self.more = nil
				}
			}
			glog.V(logger.Detail).Infof("[BZZ] KΛÐΞMLIΛ hive: queen's address: %x, population: %d (%d)\n%v", self.addr[:4], self.kad.Count(), self.kad.DBCount(), self.kad)
		}
	}()
	return
}

func (self *hive) pinger() {
	clock := time.NewTicker(1 * time.Second)
	for {
		select {
		case <-clock.C:
			if self.kad.DBCount() > 0 {
				select {
				case self.ping <- true:
				default:
				}
			}
		case _, more := <-self.more:
			if !more {
				return
			}
		}
	}
}

func (self *hive) stop() error {
	// closing ping channel quits the updateloop
	close(self.ping)
	if self.more != nil {
		close(self.more)
		self.more = nil
	}
	return self.kad.Stop(self.path)
}

func (self *hive) addPeer(p peer) {

	glog.V(logger.Detail).Infof("[BZZ] KΛÐΞMLIΛ hive: hi new bee %v", p)
	self.kad.AddNode(p)
	// self lookup (can be encoded as nil/zero key since peers addr known) + no id ()
	// the most common way of saying hi in bzz is initiation of gossip
	// let me know about anyone new from my hood , here is the storageradius
	// to send the 6 byte self lookup
	// we do not record as request or forward it, just reply with peers
	p.retrieve(&retrieveRequestMsgData{})
	glog.V(logger.Detail).Infof("[BZZ] KΛÐΞMLIΛ hive: 'whatsup wheresdaparty' sent to %v", p)
	self.ping <- true
}

func (self *hive) removePeer(p peer) {
	glog.V(logger.Detail).Infof("[BZZ] KΛÐΞMLIΛ hive: bee %v gone offline", p)
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
	addr.new()
	now := time.Now()
	return &kademlia.NodeRecord{
		Addr:   kademlia.Address(addr.hash),
		Active: time.Now().Unix(),
		Url:    addr.enode,
		After:  now.Unix(),
	}
}

// called by the protocol when receiving peerset (for target address)
// peersMsgData is converted to a slice of NodeRecords for Kademlia
// this is to store all thats needed
func (self *hive) addPeerEntries(req *peersMsgData) {
	var nrs []*kademlia.NodeRecord
	for _, p := range req.Peers {
		nrs = append(nrs, newNodeRecord(p))
	}
	self.kad.AddNodeRecords(nrs)
}
