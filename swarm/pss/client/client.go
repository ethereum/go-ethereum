// simple abstraction for implementing pss functionality
//
// the pss client library aims to simplify usage of the p2p.protocols package over pss
//
// IO is performed using the ordinary p2p.MsgReadWriter interface, which transparently communicates with a pss node via RPC using websockets as transport layer, using methods in the PssAPI class in the swarm/pss package
//
//
// Minimal-ish usage example (requires a running pss node with websocket RPC):
//
//
//   import (
//  	"context"
//  	"fmt"
//  	"os"
//  	pss "github.com/ethereum/go-ethereum/swarm/pss/client"
//  	"github.com/ethereum/go-ethereum/p2p/protocols"
//  	"github.com/ethereum/go-ethereum/p2p"
//  	"github.com/ethereum/go-ethereum/pot"
//  	"github.com/ethereum/go-ethereum/log"
//  )
//
//  type FooMsg struct {
//  	Bar int
//  }
//
//
//  func fooHandler (msg interface{}) error {
//  	foomsg, ok := msg.(*FooMsg)
//  	if ok {
//  		log.Debug("Yay, just got a message", "msg", foomsg)
//  	}
//  	return fmt.Errorf("Unknown message")
//  }
//
//  spec := &protocols.Spec{
//  	Name: "foo",
//  	Version: 1,
//  	MaxMsgSize: 1024,
//  	Messages: []interface{}{
//  		FooMsg{},
//  	},
//  }
//
//  proto := &p2p.Protocol{
//  	Name: spec.Name,
//  	Version: spec.Version,
//  	Length: uint64(len(spec.Messages)),
//  	Run: func(p *p2p.Peer, rw p2p.MsgReadWriter) error {
//  		pp := protocols.NewPeer(p, rw, spec)
//  		return pp.Run(fooHandler)
//  	},
//  }
//
//  func implementation() {
//      cfg := pss.NewClientConfig()
//      psc := pss.NewClient(context.Background(), nil, cfg)
//      err := psc.Start()
//      if err != nil {
//      	log.Crit("can't start pss client")
//      	os.Exit(1)
//      }
//
//	log.Debug("connected to pss node", "bzz addr", psc.BaseAddr)
//
//      err = psc.RunProtocol(proto)
//      if err != nil {
//      	log.Crit("can't start protocol on pss websocket")
//      	os.Exit(1)
//      }
//
//      addr := pot.RandomAddress() // should be a real address, of course
//      psc.AddPssPeer(addr, spec)
//
//      // use the protocol for something
//
//      psc.Stop()
//  }
//
// BUG(test): TestIncoming test times out due to deadlock issues in the swarm hive
package client

import (
	"context"
	"fmt"
	"sync"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/node"
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
// SelfHost: local if host to connect from
// Secure: whether or not to use secure connection (not currently in use)
type ClientConfig struct {
	RemoteHost string
	RemotePort int
	SelfHost   string
	Secure     bool
}

// Generates a pss client configuration with default values
func NewClientConfig() *ClientConfig {
	return &ClientConfig{
		SelfHost:   node.DefaultWSHost,
		RemoteHost: node.DefaultWSHost,
		RemotePort: node.DefaultWSPort,
	}
}

// After a successful connection with Client.Start, BaseAddr contains the swarm overlay address of the pss node
type Client struct {
	BaseAddr     []byte
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

// implements p2p.MsgReadWriter
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

// Constructor for production-environment clients
// Performs sanity checks on configuration paramters and gets everything ready to connect to pss node
func NewClient(ctx context.Context, cancel func(), config *ClientConfig) (*Client, error) {
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

	return pssc, nil
}

// Constructor for test implementations
// The 'rpcclient' parameter allows passing a in-memory rpc client to act as the remote websocket RPC.
func NewClientWithRPC(ctx context.Context, rpcclient *rpc.Client) (*Client, error) {
	var oaddr []byte
	err := rpcclient.CallContext(ctx, &oaddr, "pss_baseAddr")
	if err != nil {
		return nil, fmt.Errorf("cannot get pss node baseaddress: %v", err)
	}
	return &Client{
		msgC:     make(chan pss.APIMsg),
		quitC:    make(chan struct{}),
		peerPool: make(map[pss.Topic]map[pot.Address]*pssRPCRW),
		protos:   make(map[pss.Topic]*p2p.Protocol),
		ws:       rpcclient,
		ctx:      ctx,
		BaseAddr: oaddr,
	}, nil
}

func (self *Client) shutdown() {
	self.cancel()
}

// Connects to the websockets RPC
// Retrieves the swarm overlay address from the pss node
func (self *Client) Start() error {
	if self.ws != nil {
		return nil
	}
	log.Debug("Dialing ws", "src", self.localuri, "dst", self.remoteuri)
	ws, err := rpc.DialWebsocket(self.ctx, self.remoteuri, self.localuri)
	if err != nil {
		return fmt.Errorf("Couldnt dial pss websocket: %v", err)
	}

	var oaddr []byte
	err = ws.CallContext(self.ctx, &oaddr, "pss_baseAddr")
	if err != nil {
		return err
	}

	self.ws = ws
	self.BaseAddr = oaddr

	return nil
}

// Mounts a new devp2p protcool on the pss connection
// the protocol is aliased as a "pss topic"
// uses normal devp2p Send and incoming message handler routines from the p2p/protocols package
//
// when an incoming message is received from a peer that is not yet known to the client, this peer object is instantiated, and the protocol is run on it.
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

// Always call this to ensure that we exit cleanly
func (self *Client) Stop() error {
	self.cancel()
	return nil
}

// Preemptively add a remote pss peer
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

// Remove a remote pss peer
//
// Note this doesn't actually currently drop the peer, but only remmoves the reference from the client's peer lookup table
func (self *Client) RemovePssPeer(addr pot.Address, spec *protocols.Spec) {
	topic := pss.NewTopic(spec.Name, int(spec.Version))
	delete(self.peerPool[topic], addr)
}
