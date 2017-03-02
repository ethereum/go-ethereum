package network

import (
	"fmt"
)

// discovery bzz hive extension for efficient peer relaying
// this will be triggered by p2p/protocol already

type discPeer struct {
	Peer
	hive  Hive
	sub   overlaySubscription
	peers map[discover.NodeID]bool
}

type overlaySubscription interface {
	Subscribe(proxLimit uint) chan interface{}
	SubscribeProxChange(proxLimit uint) chan interface{}
}

// new discovery contructor
func NewDiscovery(p Peer, h Hive) error {
	overlay, ok := h.Overlay.(overlaySubscription)
	if !ok {
		return fmt.Errorf("overlay does not support subscription")
	}
	self := &discPeer{
		sub:   overlay,
		hive:  h,
		Peer:  p,
		peers: make(map[discover.NodeID]Peer),
	}

	c := sub.SubscribeProxChange()
	go func() {
		for {
			select {
			case <-h.quit:
				return
			case p := <-c:
				resp := &peersMsg{
					Peers: []*peerAddr{p.PeerAddr.(*peerAddr)},
				}
				p.Send(resp)
			}
		}
	}()
	p.Register(&subPeersMsg{}, self.handleSubPeersMsg)
	p.Register(&peersMsg{}, self.handlePeersMsg)
	p.Register(&getPeersMsg{}, self.handleGetPeersMsg)

	return nil
}

/*
peersMsg is the message to pass peer information
It is always a response to a peersRequestMsg

The encoding of a peer is identical to that in the devp2p base protocol peers
messages: [IP, Port, NodeID]
note that a node's DPA address is not the NodeID but the hash of the NodeID.

To mitigate against spurious peers messages, requests should be remembered
and correctness of responses should be checked

If the proxBin of peers in the response is incorrect the sender should be
disconnected
*/

// peersMsg encapsulates an array of peer addresses
// used for communicating about known peers
// relevvant for bootstrapping connectivity and updating peersets
type peersMsg struct {
	Peers []*peerAddr
}

func (self peersMsg) String() string {
	return fmt.Sprintf("%T: %v", self, self.Peers)
}

// getPeersMsg is sent to (random) peers to request (Max) peers of a specific order
type getPeersMsg struct {
	Order uint
	Max   uint
}

func (self getPeersMsg) String() string {
	return fmt.Sprintf("%T: accept max %v peers of PO%03d", self, self.Max, self.Order)
}

// subPeers msg is communicating the depth/sharpness/focus  of the overlay table of a peer
type subPeersMsg struct {
	MinProxBinSize uint
	ProxLimit      uint
	// Offset  uint
	// Batch   uint
}

func (self subPeersMsg) String() string {
	return fmt.Sprintf("request peers > PO%02d. ProxLimit: %02d", self.Request, self.ProxLimit)
}

func (self *discPeer) handleSubPeersMsg(msg interface{}) error {
	spm := msg.(*subPeersMsg)
	self.sub.Subscribe(spm.ProxLimit)
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
		p.peers[NodeId(addr)] = true
	}
	return p.hive.Register(nas...)
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
	p.Overlay.EachLivePeer(p.OverlayAddr(), int(req.Order), func(n Peer, po int) bool {
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
	err := p.Send(resp)
	if err != nil {
		return err
	}
	return nil
}
