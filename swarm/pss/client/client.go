package client

import (
	"context"
	"fmt"
	"sync"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/event"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/p2p/discover"
	"github.com/ethereum/go-ethereum/p2p/protocols"
	"github.com/ethereum/go-ethereum/pot"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/ethereum/go-ethereum/swarm/pss"
)

const (
	inboxCapacity  = 3000
	outboxCapacity = 100
	defaultWSHost  = 8546
	addrLen        = common.HashLength
)

// RemoteHost: hostname of node running websockets proxy to pss (default localhost)
// RemotePort: port of node running websockets proxy to pss (0 = go-ethereum node default)
// Secure: whether or not to use secure connection
// SelfHost: local if host to connect from
type ClientConfig struct {
	SelfHost   string
	RemoteHost string
	RemotePort int
	Secure     bool
}

func NewClientConfig() *ClientConfig {
	return &ClientConfig{
		SelfHost:   "localhost",
		RemoteHost: "localhost",
		RemotePort: 8546,
	}
}

type Client struct {
	localuri     string
	remoteuri    string
	ctx          context.Context
	cancel       func()
	subscription *rpc.ClientSubscription
	topicsC      chan []byte
	msgC         chan pss.APIMsg
	quitC        chan struct{}
	ws           *rpc.Client
	lock         sync.Mutex
	peerPool     map[pss.Topic]map[pot.Address]*pssRPCRW
	protos       map[pss.Topic]*p2p.Protocol
}

type pssRPCRW struct {
	*Client
	topic *pss.Topic
	msgC  chan []byte
	addr  pot.Address
}

func (self *Client) newpssRPCRW(addr pot.Address, topic *pss.Topic) *pssRPCRW {
	return &pssRPCRW{
		Client: self,
		topic:  topic,
		msgC:   make(chan []byte),
		addr:   addr,
	}
}

func (rw *pssRPCRW) ReadMsg() (p2p.Msg, error) {
	msg := <-rw.msgC
	log.Trace("pssrpcrw read", "msg", msg)
	pmsg, err := pss.ToP2pMsg(msg)
	if err != nil {
		return p2p.Msg{}, err
	}

	return pmsg, nil
}

func (rw *pssRPCRW) WriteMsg(msg p2p.Msg) error {
	log.Trace("got writemsg pssclient", "msg", msg)
	rlpdata := make([]byte, msg.Size)
	msg.Payload.Read(rlpdata)
	pmsg, err := rlp.EncodeToBytes(pss.ProtocolMsg{
		Code:    msg.Code,
		Size:    msg.Size,
		Payload: rlpdata,
	})
	if err != nil {
		return err
	}
	return rw.Client.ws.CallContext(rw.Client.ctx, nil, "pss_send", rw.topic, pss.APIMsg{
		Addr: rw.addr.Bytes(),
		Msg:  pmsg,
	})
}

func NewClient(ctx context.Context, cancel func(), config *ClientConfig) *Client {
	prefix := "ws"

	if ctx == nil {
		ctx = context.Background()
	}
	if cancel == nil {
		cancel = func() { return }
	}

	pssc := &Client{
		msgC:     make(chan pss.APIMsg),
		quitC:    make(chan struct{}),
		peerPool: make(map[pss.Topic]map[pot.Address]*pssRPCRW),
		protos:   make(map[pss.Topic]*p2p.Protocol),
		ctx:      ctx,
		cancel:   cancel,
	}

	if config.Secure {
		prefix = "wss"
	}

	pssc.remoteuri = fmt.Sprintf("%s://%s:%d", prefix, config.RemoteHost, config.RemotePort)
	pssc.localuri = fmt.Sprintf("%s://%s", prefix, config.SelfHost)

	return pssc
}

func NewClientWithRPC(ctx context.Context, client *rpc.Client) *Client {
	return &Client{
		msgC:     make(chan pss.APIMsg),
		quitC:    make(chan struct{}),
		peerPool: make(map[pss.Topic]map[pot.Address]*pssRPCRW),
		protos:   make(map[pss.Topic]*p2p.Protocol),
		ws:       client,
		ctx:      ctx,
	}
}

func (self *Client) shutdown() {
	self.cancel()
}

func (self *Client) Start() error {
	if self.ws != nil {
		return nil
	}
	log.Debug("Dialing ws", "src", self.localuri, "dst", self.remoteuri)
	ws, err := rpc.DialWebsocket(self.ctx, self.remoteuri, self.localuri)
	if err != nil {
		return fmt.Errorf("Couldnt dial pss websocket: %v", err)
	}

	self.ws = ws

	return nil
}

func (self *Client) RunProtocol(proto *p2p.Protocol) error {
	topic := pss.NewTopic(proto.Name, int(proto.Version))
	msgC := make(chan pss.APIMsg)
	self.peerPool[topic] = make(map[pot.Address]*pssRPCRW)
	sub, err := self.ws.Subscribe(self.ctx, "pss", msgC, "receive", topic)
	if err != nil {
		return fmt.Errorf("pss event subscription failed: %v", err)
	}

	self.subscription = sub

	// dispatch incoming messages
	go func() {
		for {
			select {
			case msg := <-msgC:
				var addr pot.Address
				copy(addr[:], msg.Addr)
				if self.peerPool[topic][addr] == nil {
					self.peerPool[topic][addr] = self.newpssRPCRW(addr, &topic)
					nid, _ := discover.HexID("0x00")
					p := p2p.NewPeer(nid, fmt.Sprintf("%v", addr), []p2p.Cap{})
					go proto.Run(p, self.peerPool[topic][addr])
				}
				go func() {
					self.peerPool[topic][addr].msgC <- msg.Msg
				}()
			case <-self.quitC:
				self.shutdown()
				return
			}
		}
	}()

	self.protos[topic] = proto
	return nil
}

func (self *Client) Stop() error {
	self.cancel()
	return nil
}

func (self *Client) AddPssPeer(addr pot.Address, spec *protocols.Spec) {
	topic := pss.NewTopic(spec.Name, int(spec.Version))
	if self.peerPool[topic] == nil {
		log.Error("addpeer on unset topic")
		return
	}
	if self.peerPool[topic][addr] == nil {
		self.peerPool[topic][addr] = self.newpssRPCRW(addr, &topic)
		nid, _ := discover.HexID("0x00")
		p := p2p.NewPeer(nid, fmt.Sprintf("%v", addr), []p2p.Cap{})
		go self.protos[topic].Run(p, self.peerPool[topic][addr])
	}
}

func (self *Client) RemovePssPeer(addr pot.Address, spec *protocols.Spec) {
	topic := pss.NewTopic(spec.Name, int(spec.Version))
	delete(self.peerPool[topic], addr)
}

func (self *Client) SubscribeEvents(ch chan *p2p.PeerEvent) event.Subscription {
	log.Error("PSS client handles events internally, use the read functions instead")
	return nil
}

func (self *Client) PeerCount() int {
	return len(self.peerPool)
}

func (self *Client) NodeInfo() *p2p.NodeInfo {
	return nil
}

func (self *Client) PeersInfo() []*p2p.PeerInfo {
	return nil
}
func (self *Client) AddPeer(node *discover.Node) {
	log.Error("Cannot add peer in PSS with discover.Node, need swarm overlay address")
}

func (self *Client) RemovePeer(node *discover.Node) {
	log.Error("Cannot remove peer in PSS with discover.Node, need swarm overlay address")
}
