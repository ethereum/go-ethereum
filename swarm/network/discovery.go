package network

import (
	"fmt"
	"sync"

	"github.com/ethereum/go-ethereum/log"
)

// discovery bzz overlay extension doing peer relaying

type discPeer struct {
	*bzzPeer
	overlay   Overlay
	mtx       sync.Mutex
	peers     map[string]bool
	depth     uint8 // the proximity radius advertised by remote to subscribe to peers
	sentPeers bool  // set to true  when the peer is first notifed of peers close to them
}

// NewDiscovery discovery peer contructor
func NewDiscovery(p *bzzPeer, o Overlay) *discPeer {
	self := &discPeer{
		overlay: o,
		bzzPeer: p,
		peers:   make(map[string]bool),
	}
	self.seen(self)
	return self
}

func (self *discPeer) HandleMsg(msg interface{}) error {
	switch msg := msg.(type) {

	case *peersMsg:
		return self.handlePeersMsg(msg)

	case *getPeersMsg:
		return self.handleGetPeersMsg(msg)

	case *subPeersMsg:
		return self.handleSubPeersMsg(msg)

	default:
		return fmt.Errorf("unknown message type: %T", msg)
	}
}

// NotifyPeer notifies the receiver remote end of a peer p or PO po.
// callback for overlay driver
func (self *discPeer) NotifyPeer(a OverlayAddr, po uint8) error {
	if po < self.depth || self.seen(a) {
		return nil
	}
	log.Warn(fmt.Sprintf("notification about %x", a.Address()))

	resp := &peersMsg{
		Peers: []*bzzAddr{ToAddr(a)}, // perhaps the PeerAddr interface is unnecessary generalization
	}
	return self.Send(resp)
}

// NotifyDepth sends a subPeers Msg to the receiver notifying them about
// a change in the prox limit (radius of the set including the nearest X peers
// or first empty row)
// callback for overlay driver
func (self *discPeer) NotifyDepth(po uint8) error {
	return self.Send(&subPeersMsg{Depth: po})
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
	Peers []*bzzAddr
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
	Depth uint8
}

func (self subPeersMsg) String() string {
	return fmt.Sprintf("%T: request peers > PO%02d. ", self, self.Depth)
}

func (self *discPeer) handleSubPeersMsg(msg *subPeersMsg) error {
	self.depth = msg.Depth
	if !self.sentPeers {
		var peers []*bzzAddr
		self.overlay.EachConn(self.Over(), 255, func(p OverlayConn, po int, isproxbin bool) bool {
			if uint8(po) < self.depth {
				return false
			}
			if !self.seen(p) {
				peers = append(peers, ToAddr(p.Off()))
			}
			return true
		})
		log.Warn(fmt.Sprintf("found initial %v peers not farther than %v", len(peers), self.depth))
		if len(peers) > 0 {
			if err := self.Send(&peersMsg{Peers: peers}); err != nil {
				return err
			}
		}
	}
	self.sentPeers = true
	return nil
}

// handlePeersMsg called by the protocol when receiving peerset (for target address)
// list of nodes ([]PeerAddr in peersMsg) is added to the overlay db using the
// Register interface method
func (self *discPeer) handlePeersMsg(msg *peersMsg) error {
	// register all addresses
	if len(msg.Peers) == 0 {
		log.Debug(fmt.Sprintf("whoops, no peers in incoming peersMsg from %v", self))
		return nil
	}

	c := make(chan OverlayAddr)
	go func() {
		defer close(c)
		for _, a := range msg.Peers {
			self.seen(a)
			c <- a
		}
	}()
	log.Info("discovery overlay register")
	return self.overlay.Register(c)
}

// handleGetPeersMsg is called by the protocol when receiving a
// peerset (for target address) request
// peers suggestions are retrieved from the overlay topology driver
// using the EachConn interface iterator method
// peers sent are remembered throughout a session and not sent twice
func (self *discPeer) handleGetPeersMsg(msg *getPeersMsg) error {
	var peers []*bzzAddr
	i := 0
	self.overlay.EachConn(self.Over(), int(msg.Order), func(p OverlayConn, po int, isproxbin bool) bool {
		i++
		// only send peers we have not sent before in this session
		a := ToAddr(p.Off())
		if self.seen(a) {
			peers = append(peers, a)
		}
		return len(peers) < int(msg.Max)
	})
	if len(peers) == 0 {
		log.Debug(fmt.Sprintf("no peers found for %v", self))
		return nil
	}
	log.Debug(fmt.Sprintf("%v peers sent to %v", len(peers), self))
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
	//var err error
	k.EachConn(nil, 255, func(p OverlayConn, po int, isproxbin bool) bool {
		if err := p.(Conn).Send(req); err == nil {
			i++
			if i >= broadcastSize {
				return false
			}
		}
		return true
	})
	log.Info(fmt.Sprintf("requesting bees of PO%03d from %v/%v (each max %v)", order, i, broadcastSize, maxPeers))
}

func (self *discPeer) seen(p OverlayPeer) bool {
	self.mtx.Lock()
	defer self.mtx.Unlock()
	k := string(p.Address())
	if self.peers[k] {
		return true
	}
	self.peers[k] = true
	return false
}
