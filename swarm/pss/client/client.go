package client

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"

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
type PssClientConfig struct {
	SelfHost   string
	RemoteHost string
	RemotePort int
	Secure     bool
}

func NewPssClientConfig() *PssClientConfig {
	return &PssClientConfig{
		SelfHost:   "localhost",
		RemoteHost: "localhost",
		RemotePort: 8546,
	}
}

type PssClient struct {
	localuri     string
	remoteuri    string
	ctx          context.Context
	cancel       func()
	subscription *rpc.ClientSubscription
	topicsC      chan []byte
	msgC         chan pss.PssAPIMsg
	quitC        chan struct{}
	quitting     uint32
	ws           *rpc.Client
	lock         sync.Mutex
	peerPool     map[pss.PssTopic]map[pot.Address]*pssRPCRW
	protos       map[pss.PssTopic]*p2p.Protocol
}

type pssRPCRW struct {
	*PssClient
	topic *pss.PssTopic
	msgC  chan []byte
	addr  pot.Address
}

func (self *PssClient) newpssRPCRW(addr pot.Address, topic *pss.PssTopic) *pssRPCRW {
	return &pssRPCRW{
		PssClient: self,
		topic:     topic,
		msgC:      make(chan []byte),
		addr:      addr,
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
	pmsg, err := rlp.EncodeToBytes(pss.PssProtocolMsg{
		Code:    msg.Code,
		Size:    msg.Size,
		Payload: rlpdata,
	})
	if err != nil {
		return err
	}
	return rw.PssClient.ws.CallContext(rw.PssClient.ctx, nil, "pss_sendPss", rw.topic, pss.PssAPIMsg{
		Addr: rw.addr.Bytes(),
		Msg:  pmsg,
	})
}

func NewPssClient(ctx context.Context, cancel func(), config *PssClientConfig) *PssClient {
	prefix := "ws"

	if ctx == nil {
		ctx = context.Background()
	}
	if cancel == nil {
		cancel = func() { return }
	}

	pssc := &PssClient{
		msgC:     make(chan pss.PssAPIMsg),
		quitC:    make(chan struct{}),
		peerPool: make(map[pss.PssTopic]map[pot.Address]*pssRPCRW),
		protos:   make(map[pss.PssTopic]*p2p.Protocol),
		ctx:      ctx,
		cancel:   cancel,
	}

	if config.RemoteHost == "" {
		config.RemoteHost = "localhost"
	}

	if config.RemotePort == 0 {
		config.RemotePort = defaultWSHost
	}

	if config.SelfHost == "" {
		config.SelfHost = "localhost"
	}

	if config.Secure {
		prefix = "wss"
	}

	pssc.remoteuri = fmt.Sprintf("%s://%s:%d", prefix, config.RemoteHost, config.RemotePort)
	pssc.localuri = fmt.Sprintf("%s://%s", prefix, config.SelfHost)

	return pssc
}

func NewPssClientWithRPC(ctx context.Context, client *rpc.Client) *PssClient {
	return &PssClient{
		msgC:     make(chan pss.PssAPIMsg),
		quitC:    make(chan struct{}),
		peerPool: make(map[pss.PssTopic]map[pot.Address]*pssRPCRW),
		protos:   make(map[pss.PssTopic]*p2p.Protocol),
		ws:       client,
		ctx:      ctx,
	}
}

func (self *PssClient) shutdown() {
	atomic.StoreUint32(&self.quitting, 1)
	self.cancel()
}

func (self *PssClient) Start() error {
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

func (self *PssClient) RunProtocol(proto *p2p.Protocol) error {
	topic := pss.NewTopic(proto.Name, int(proto.Version))
	msgC := make(chan pss.PssAPIMsg)
	self.peerPool[topic] = make(map[pot.Address]*pssRPCRW)
	sub, err := self.ws.Subscribe(self.ctx, "pss", msgC, "receivePss", topic)
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

func (self *PssClient) Stop() error {
	self.cancel()
	return nil
}

func (self *PssClient) AddPssPeer(addr pot.Address, spec *protocols.Spec) {
	topic := pss.NewTopic(spec.Name, int(spec.Version))
	if self.peerPool[topic][addr] == nil {
		self.peerPool[topic][addr] = self.newpssRPCRW(addr, &topic)
		nid, _ := discover.HexID("0x00")
		p := p2p.NewPeer(nid, fmt.Sprintf("%v", addr), []p2p.Cap{})
		go self.protos[topic].Run(p, self.peerPool[topic][addr])
	}
}

func (self *PssClient) RemovePssPeer(addr pot.Address, spec *protocols.Spec) {
	topic := pss.NewTopic(spec.Name, int(spec.Version))
	delete(self.peerPool[topic], addr)
}

func (self *PssClient) SubscribeEvents(ch chan *p2p.PeerEvent) event.Subscription {
	log.Error("PSS client handles events internally, use the read functions instead")
	return nil
}

func (self *PssClient) PeerCount() int {
	return len(self.peerPool)
}

func (self *PssClient) NodeInfo() *p2p.NodeInfo {
	return nil
}

func (self *PssClient) PeersInfo() []*p2p.PeerInfo {
	return nil
}
func (self *PssClient) AddPeer(node *discover.Node) {
	log.Error("Cannot add peer in PSS with discover.Node, need swarm overlay address")
}

func (self *PssClient) RemovePeer(node *discover.Node) {
	log.Error("Cannot remove peer in PSS with discover.Node, need swarm overlay address")
}
