package network

import (
	"fmt"

	"github.com/ethereum/go-ethereum/logger"
	"github.com/ethereum/go-ethereum/logger/glog"
	// "github.com/ethereum/go-ethereum/p2p/discover"
)

// discovery bzz overlay extension doing peer relaying

// messages related to peer discovery
var DiscoveryMsgs = []interface{}{
	&getPeersMsg{},
	&peersMsg{},
	&subPeersMsg{},
}

type discPeer struct {
	Peer
	overlay Overlay
	peers   map[string]bool
	// peers     map[discover.NodeID]bool
	proxLimit uint8 // the proximity radius advertised by remote to subscribe to peers
	sentPeers bool  // set to true  when the peer is first notifed of peers close to them
}

// discovery peer contructor
// registers the handlers for discovery messages
func NewDiscovery(p Peer, o Overlay) *discPeer {
	self := &discPeer{
		overlay: o,
		Peer:    p,
		peers:   make(map[string]bool),
	}
	self.seen(self)

	p.Register(&peersMsg{}, self.handlePeersMsg)
	p.Register(&getPeersMsg{}, self.handleGetPeersMsg)
	p.Register(&subPeersMsg{}, self.handleSubPeersMsg)

	return self
}

// NotifyPeer notifies the receiver remote end of a peer p or PO po.
// callback for overlay driver
func (self *discPeer) NotifyPeer(p Peer, po uint8) error {
	glog.V(logger.Warn).Infof("peers %v", self.peers)
	if po < self.proxLimit || self.seen(p) {
		return nil
	}
	glog.V(logger.Warn).Infof("notification about %x", p.OverlayAddr())

	resp := &peersMsg{
		Peers: []*peerAddr{&peerAddr{OAddr: p.OverlayAddr(), UAddr: p.UnderlayAddr()}}, // perhaps the PeerAddr interface is unnecessary generalization
	}
	return self.Send(resp)
}

// NotifyProx sends a subPeers Msg to the receiver notifying them about
// a change in the prox limit (radius of the set including the nearest X peers
// or first empty row)
// callback for overlay driver
func (self *discPeer) NotifyProx(po uint8) error {
	return self.Send(&subPeersMsg{ProxLimit: po})
}

/*
peersMsg is the message to pass peer information
It is always a response to a peersRequestMsg

The encoding of a peer is identical to that in the devp2p base protocol peers
messages: [IP, Port, NodeID]
note that a node's DPA address is not the NodeID but the hash of the NodeID.

TODO:
To mitigate against spurious peers messages, requests should be remembered
and correctness of responses should be checked

If the proxBin of peers in the response is incorrect the sender should be
disconnected
*/

// peersMsg encapsulates an array of peer addresses
// used for communicating about known peers
// relevant for bootstrapping connectivity and updating peersets
type peersMsg struct {
	Peers []*peerAddr
}

func (self peersMsg) String() string {
	return fmt.Sprintf("%T: %v", self, self.Peers)
}

// getPeersMsg is sent to (random) peers to request (Max) peers of a specific order
type getPeersMsg struct {
	Order uint8
	Max   uint8
}

func (self getPeersMsg) String() string {
	return fmt.Sprintf("%T: accept max %v peers of PO%03d", self, self.Max, self.Order)
}

// subPeers msg is communicating the depth/sharpness/focus  of the overlay table of a peer
type subPeersMsg struct {
	ProxLimit uint8
}

func (self subPeersMsg) String() string {
	return fmt.Sprintf("%T: request peers > PO%02d. ", self, self.ProxLimit)
}

func (self *discPeer) handleSubPeersMsg(msg interface{}) error {
	spm := msg.(*subPeersMsg)
	self.proxLimit = spm.ProxLimit
	if !self.sentPeers {
		var peers []*peerAddr
		self.overlay.EachLivePeer(self.OverlayAddr(), 255, func(p Peer, po int) bool {
			if uint8(po) < self.proxLimit {
				return false
			}
			self.seen(p)
			peers = append(peers, &peerAddr{p.OverlayAddr(), p.UnderlayAddr()})
			return true
		})
		glog.V(logger.Warn).Infof("found initial %v peers not farther than %v", len(peers), self.proxLimit)
		if len(peers) > 0 {
			self.Send(&peersMsg{Peers: peers})
		}
	}
	self.sentPeers = true
	return nil
}

// handlePeersMsg called by the protocol when receiving peerset (for target address)
// list of nodes ([]PeerAddr in peersMsg) is added to the overlay db using the
// Register interface method
func (self *discPeer) handlePeersMsg(msg interface{}) error {
	// register all addresses
	var nas []PeerAddr
	for _, na := range msg.(*peersMsg).Peers {
		addr := PeerAddr(na)
		nas = append(nas, addr)
		self.seen(addr)
	}

	if len(nas) == 0 {
		glog.V(logger.Debug).Infof("whoops, no peers in incoming peersMsg from %v", self)
		return nil
	}
	glog.V(logger.Debug).Infof("got peer addresses from %x, %v (%v)", self.OverlayAddr(), nas, len(nas))
	return self.overlay.Register(nas...)
}

// handleGetPeersMsg is called by the protocol when receiving a
// peerset (for target address) request
// peers suggestions are retrieved from the overlay topology driver
// using the EachLivePeer interface iterator method
// peers sent are remembered throughout a session and not sent twice
func (self *discPeer) handleGetPeersMsg(msg interface{}) error {
	var peers []*peerAddr
	req := msg.(*getPeersMsg)
	i := 0
	self.overlay.EachLivePeer(self.OverlayAddr(), int(req.Order), func(n Peer, po int) bool {
		i++
		// only send peers we have not sent before in this session
		if self.seen(n) {
			peers = append(peers, &peerAddr{n.OverlayAddr(), n.UnderlayAddr()})
		}
		return len(peers) < int(req.Max)
	})
	if len(peers) == 0 {
		glog.V(logger.Debug).Infof("no peers found for %v", self)
		return nil
	}
	glog.V(logger.Debug).Infof("%v peers sent to %v", len(peers), self)
	resp := &peersMsg{
		Peers: peers,
	}
	return self.Send(resp)
}

func RequestOrder(k Overlay, order, broadcastSize, maxPeers uint8) {
	req := &getPeersMsg{
		Order: uint8(order),
		Max:   maxPeers,
	}
	var i uint8
	var err error
	k.EachLivePeer(nil, 255, func(n Peer, po int) bool {
		glog.V(logger.Detail).Infof("%T sent to %v", req, n.ID())
		err = n.Send(req)
		if err == nil {
			i++
			if i >= broadcastSize {
				return false
			}
		}
		return true
	})
	glog.V(logger.Info).Infof("requesting bees of PO%03d from %v/%v (each max %v)", order, i, broadcastSize, maxPeers)
}

func (self *discPeer) seen(p PeerAddr) bool {
	k := NodeId(p).NodeID.String()
	if self.peers[k] {
		return true
	}
	self.peers[k] = true
	return false
}
