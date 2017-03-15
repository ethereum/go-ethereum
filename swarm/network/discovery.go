package network

import (
	"bytes"
	"fmt"

	"github.com/ethereum/go-ethereum/logger"
	"github.com/ethereum/go-ethereum/logger/glog"
	"github.com/ethereum/go-ethereum/p2p/discover"
)

// discovery bzz overlay extension doing peer relaying
// can be switched off

type discPeer struct {
	Peer
	overlay   Overlay
	proxLimit uint8
	peers     map[discover.NodeID]bool
}

// NotifyPeer notifies the receiver remote end of a peer p or PO po.
// callback for overlay driver
func (self *discPeer) NotifyPeer(p Peer, po uint8) error {
	if po < self.proxLimit || self.peers[p.ID()] {
		return nil
	}
	resp := &peersMsg{
		Peers: []*peerAddr{p.(*discPeer).Peer.(*bzzPeer).peerAddr},
	}
	return p.Send(resp)
}

// NotifyProx sends a subPeers Msg to the receiver notifying them about
// a change in the prox limit (radius of the set including the nearest X peers
// or first empty row)
// callback for overlay driver
func (self *discPeer) NotifyProx(po uint8) error {
	return self.Send(&SubPeersMsg{ProxLimit: po, MinProxBinSize: 8})
}

// new discovery contructor
func NewDiscovery(p Peer, o Overlay) *discPeer {
	self := &discPeer{
		overlay: o,
		Peer:    p,
		peers:   make(map[discover.NodeID]bool),
	}

	p.Register(&peersMsg{}, self.handlePeersMsg)
	p.Register(&getPeersMsg{}, self.handleGetPeersMsg)
	p.Register(&SubPeersMsg{}, self.handleSubPeersMsg)

	return self
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
type SubPeersMsg struct {
	MinProxBinSize uint8
	ProxLimit      uint8
}

func (self SubPeersMsg) String() string {
	return fmt.Sprintf("%T: request peers > PO%02d. ", self, self.ProxLimit)
}

func (self *discPeer) handleSubPeersMsg(msg interface{}) error {
	spm := msg.(*SubPeersMsg)
	self.proxLimit = spm.ProxLimit
	return nil
}

// handlePeersMsg called by the protocol when receiving peerset (for target address)
// list of nodes ([]PeerAddr in peersMsg) is added to the overlay db using the
// Register interface method
func (p *discPeer) handlePeersMsg(msg interface{}) error {
	// register all addresses
	var nas []PeerAddr
	for _, na := range msg.(*peersMsg).Peers {
		addr := PeerAddr(na)
		nas = append(nas, addr)
		p.peers[NodeId(addr).NodeID] = true
	}
	return p.overlay.Register(nas...)
}

// handleGetPeersMsg is called by the protocol when receiving a
// peerset (for target address) request
// peers suggestions are retrieved from the overlay topology driver
// using the EachLivePeer interface iterator method
// peers sent are remembered throughout a session and not sent twice
func (p *discPeer) handleGetPeersMsg(msg interface{}) error {
	req := msg.(*getPeersMsg)
	var peers []*peerAddr
	alreadySent := p.peers
	i := 0
	p.overlay.EachLivePeer(p.OverlayAddr(), int(req.Order), func(n Peer, po int) bool {
		i++
		if bytes.Compare(n.OverlayAddr(), p.OverlayAddr()) != 0 &&
			// only send peers we have not sent before in this session
			!alreadySent[n.ID()] {
			alreadySent[n.ID()] = true
			peers = append(peers, &peerAddr{n.OverlayAddr(), n.UnderlayAddr()})
		}
		// return int(req.Order) == po && len(peers) < int(req.Max)
		return len(peers) < int(req.Max)
	})
	if len(peers) == 0 {
		glog.V(logger.Debug).Infof("no peers found for %v", p)
		return nil
	}
	glog.V(logger.Debug).Infof("%v peers sent to %v", len(peers), p)
	resp := &peersMsg{
		Peers: peers,
	}
	return p.Send(resp)
}
