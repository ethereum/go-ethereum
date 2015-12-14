package network

import (
	"fmt"
	"math/rand"
	"path/filepath"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/logger"
	"github.com/ethereum/go-ethereum/logger/glog"
	"github.com/ethereum/go-ethereum/p2p/discover"
	"github.com/ethereum/go-ethereum/swarm/network/kademlia"
	"github.com/ethereum/go-ethereum/swarm/storage"
)

// Hive is the logistic manager of the swarm
// it uses a generic kademlia nodetable to find best peer list
// for any target
// this is used by the netstore to search for content in the swarm
// the bzz protocol peersMsgData exchange is relayed to Kademlia
// for db storage and filtering
// connections and disconnections are reported and relayed
// to keep the nodetable uptodate

type Hive struct {
	listenAddr   func() string
	callInterval uint64
	id           discover.NodeID
	addr         kademlia.Address
	kad          *kademlia.Kademlia
	path         string
	toggle       chan bool
	more         chan bool

	// for testing only
	swapEnabled bool
	syncEnabled bool
	blockRead   bool
	blockWrite  bool
}

const (
	callInterval = 3000000000
	// bucketSize   = 3
	// maxProx      = 8
	// proxBinSize  = 4
)

type HiveParams struct {
	CallInterval uint64
	KadDbPath    string
	*kademlia.KadParams
}

func NewHiveParams(path string) *HiveParams {
	kad := kademlia.NewKadParams()
	// kad.BucketSize = bucketSize
	// kad.MaxProx = maxProx
	// kad.ProxBinSize = proxBinSize

	return &HiveParams{
		CallInterval: callInterval,
		KadDbPath:    filepath.Join(path, "bzz-peers.json"),
		KadParams:    kad,
	}
}

func NewHive(addr common.Hash, params *HiveParams, swapEnabled, syncEnabled bool) *Hive {
	kad := kademlia.New(kademlia.Address(addr), params.KadParams)
	return &Hive{
		callInterval: params.CallInterval,
		kad:          kad,
		addr:         kad.Addr(),
		path:         params.KadDbPath,
		swapEnabled:  swapEnabled,
		syncEnabled:  syncEnabled,
	}
}

func (self *Hive) SyncEnabled(on bool) {
	self.syncEnabled = on
}

func (self *Hive) SwapEnabled(on bool) {
	self.swapEnabled = on
}

func (self *Hive) BlockNetworkRead(on bool) {
	self.blockRead = on
}

func (self *Hive) BlockNetworkWrite(on bool) {
	self.blockWrite = on
}

// public accessor to the hive base address
func (self *Hive) Addr() kademlia.Address {
	return self.addr
}

// Start receives network info only at startup
// listedAddr is a function to retrieve listening address to advertise to peers
// connectPeer is a function to connect to a peer based on its NodeID or enode URL
// there are called on the p2p.Server which runs on the node
func (self *Hive) Start(id discover.NodeID, listenAddr func() string, connectPeer func(string) error) (err error) {
	self.toggle = make(chan bool)
	self.more = make(chan bool)
	self.id = id
	self.listenAddr = listenAddr
	err = self.kad.Load(self.path, nil)
	if err != nil {
		glog.V(logger.Warn).Infof("[BZZ] KΛÐΞMLIΛ Warning: error reading kaddb '%s' (skipping): %v", self.path, err)
		err = nil
	}
	// this loop is doing bootstrapping and maintains a healthy table
	go self.keepAlive()
	go func() {
		// whenever toggled ask kademlia about most preferred peer
		for alive := range self.more {
			if !alive {
				// receiving false closes the loop while allowing parallel routines
				// to attempt to write to more (remove Peer when shutting down)
				return
			}
			node, proxLimit := self.kad.FindBest()
			if node != nil && len(node.Url) > 0 {
				glog.V(logger.Detail).Infof("[BZZ] KΛÐΞMLIΛ hive: call for bee %v", node.Url)
				// enode or any lower level connection address is unnecessary in future
				// discovery table is used to look it up.
				connectPeer(node.Url)
			} else if proxLimit > -1 {
				// a random peer is taken from the table
				peers := self.kad.FindClosest(kademlia.RandomAddressAt(self.addr, rand.Intn(self.kad.MaxProx)), 1)
				if len(peers) > 0 {
					// a random address at prox bin 0 is sent for lookup
					randAddr := kademlia.RandomAddressAt(self.addr, proxLimit)
					req := &retrieveRequestMsgData{
						Key: storage.Key(randAddr[:]),
					}
					glog.V(logger.Detail).Infof("[BZZ] KΛÐΞMLIΛ hive: call any bee in area %v messenger bee %v", randAddr, peers[0])
					peers[0].(*peer).retrieve(req)
				}
				self.toggle <- true
				glog.V(logger.Detail).Infof("[BZZ] KΛÐΞMLIΛ hive: buzz kept alive")
			} else {
				self.toggle <- false
			}
			glog.V(logger.Detail).Infof("[BZZ] KΛÐΞMLIΛ hive: queen's address: %v, population: %d (%d)", self.addr, self.kad.Count(), self.kad.DBCount())
		}
	}()
	return
}

// keepAlive is a forever loop
// in its awake state it periodically triggers connection attempts
// by writing to self.more until Kademlia Table is saturated
// wake state is toggled by writing to self.toggle
// it restarts if the table becomes non-full again due to disconnections
func (self *Hive) keepAlive() {
	alarm := time.NewTicker(time.Duration(self.callInterval)).C
	for {
		select {
		case <-alarm:
			if self.kad.DBCount() > 0 {
				select {
				case self.more <- true:
				default:
				}
			}
		case need, alive := <-self.toggle:
			if !alive {
				self.more <- false
				return
			}
			if alarm == nil && need {
				alarm = time.NewTicker(time.Duration(self.callInterval)).C
			}
			if alarm != nil && !need {
				alarm = nil

			}
		}
	}
}

func (self *Hive) Stop() error {
	// closing toggle channel quits the updateloop
	close(self.toggle)
	return self.kad.Save(self.path, saveSync)
}

// called at the end of a successful protocol handshake
func (self *Hive) addPeer(p *peer) {
	glog.V(logger.Detail).Infof("[BZZ] KΛÐΞMLIΛ hive: hi new bee %v", p)
	self.kad.On(p, loadSync)
	// self lookup (can be encoded as nil/zero key since peers addr known) + no id ()
	// the most common way of saying hi in bzz is initiation of gossip
	// let me know about anyone new from my hood , here is the storageradius
	// to send the 6 byte self lookup
	// we do not record as request or forward it, just reply with peers
	p.retrieve(&retrieveRequestMsgData{})
	glog.V(logger.Detail).Infof("[BZZ] KΛÐΞMLIΛ hive: 'whatsup wheresdaparty' sent to %v", p)
	select {
	case self.more <- true:
	default:
	}
}

// called after peer disconnected
func (self *Hive) removePeer(p *peer) {
	glog.V(logger.Debug).Infof("[BZZ] KΛÐΞMLIΛ hive: bee %v removed", p)
	self.kad.Off(p, saveSync)
	select {
	case self.more <- true:
	default:
	}
	if self.kad.Count() == 0 {
		glog.V(logger.Debug).Infof("[BZZ] KΛÐΞMLIΛ hive: empty, all bees gone")
	}
}

// Retrieve a list of live peers that are closer to target than us
func (self *Hive) getPeers(target storage.Key, max int) (peers []*peer) {
	var addr kademlia.Address
	copy(addr[:], target[:])
	for _, node := range self.kad.FindClosest(addr, max) {
		peers = append(peers, node.(*peer))
	}
	return
}

// disconnects all the peers
func (self *Hive) DropAll() {
	glog.V(logger.Info).Infof("[BZZ] KΛÐΞMLIΛ hive: dropping all bees")
	for _, node := range self.kad.FindClosest(kademlia.Address{}, 0) {
		node.Drop()
	}
}

// contructor for kademlia.NodeRecord based on peer address alone
// TODO: should go away and only addr passed to kademlia
func newNodeRecord(addr *peerAddr) *kademlia.NodeRecord {
	now := kademlia.Time(time.Now())
	return &kademlia.NodeRecord{
		Addr:  addr.Addr,
		Url:   addr.String(),
		Seen:  now,
		After: now,
	}
}

// called by the protocol when receiving peerset (for target address)
// peersMsgData is converted to a slice of NodeRecords for Kademlia
// this is to store all thats needed
func (self *Hive) HandlePeersMsg(req *peersMsgData, from *peer) {
	var nrs []*kademlia.NodeRecord
	for _, p := range req.Peers {
		nrs = append(nrs, newNodeRecord(p))
	}
	self.kad.Add(nrs)
}

// peer wraps the protocol instance to represent a connected peer
// it implements kademlia.Node interface
type peer struct {
	*bzz // protocol instance running on peer connection
}

// protocol instance implements kademlia.Node interface (embedded peer)
func (self *peer) Addr() kademlia.Address {
	return self.remoteAddr.Addr
}

func (self *peer) Url() string {
	return self.remoteAddr.String()
}

// TODO take into account traffic
func (self *peer) LastActive() time.Time {
	return time.Now()
}

// reads the serialised form of sync state persisted as the 'Meta' attribute
// and sets the decoded syncState on the online node
func loadSync(record *kademlia.NodeRecord, node kademlia.Node) error {
	p, ok := node.(*peer)
	if !ok {
		return fmt.Errorf("invalid type")
	}
	if record.Meta == nil {
		glog.V(logger.Debug).Infof("no sync state for node record %v setting default", record)
		p.syncState = &syncState{DbSyncState: &storage.DbSyncState{}}
		return nil
	}
	state, err := decodeSync(record.Meta)
	if err != nil {
		return fmt.Errorf("error decoding kddb record meta info into a sync state: %v", err)
	}
	glog.V(logger.Detail).Infof("sync state for node record %v read from Meta: %s", record, string(*(record.Meta)))
	p.syncState = state
	return err
}

// callback when saving a sync state
func saveSync(record *kademlia.NodeRecord, node kademlia.Node) {
	if p, ok := node.(*peer); ok {
		meta, err := encodeSync(p.syncState)
		if err != nil {
			glog.V(logger.Warn).Infof("error saving sync state for %v: %v", node, err)
			return
		}
		glog.V(logger.Detail).Infof("saved sync state for %v: %s", node, string(*meta))
		record.Meta = meta
	}
}

// the immediate response to a retrieve request,
// sends relevant peer data given by the kademlia hive to the requester
// TODO: remember peers sent for duration of the session, only new peers sent
func (self *Hive) peers(req *retrieveRequestMsgData) {
	// FIXME: should check req.MaxPeers but then should not default to zero or make sure we set it when sending retrieveRequests
	// we might need chunk.req to cache relevant peers response,
	// hive change would expire it
	if req != nil && req.MaxPeers >= 0 {
		var addrs []*peerAddr
		if req.timeout == nil || time.Now().Before(*(req.timeout)) {
			key := req.Key
			// self lookup from remote peer
			if storage.IsZeroKey(key) {
				addr := req.from.Addr()
				key = storage.Key(addr[:])
				req.Key = nil
			}
			// get peer addresses from hive
			for _, peer := range self.getPeers(key, int(req.MaxPeers)) {
				addrs = append(addrs, peer.remoteAddr)
			}
			glog.V(logger.Detail).Infof("[BZZ] Hive sending %d peer addresses to %v. req.Id: %v, req.Key: %x", len(addrs), req.from, req.Id, req.Key.Log())

			peersData := &peersMsgData{
				Peers: addrs,
				Key:   req.Key,
				Id:    req.Id,
			}
			peersData.setTimeout(req.timeout)
			req.from.peers(peersData)
		}
	}
}

func (self *Hive) String() string {
	return self.kad.String()
}
