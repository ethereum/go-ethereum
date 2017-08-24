package client

import (
	"context"
	"fmt"
	"sync"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/p2p/discover"
	"github.com/ethereum/go-ethereum/p2p/protocols"
	"github.com/ethereum/go-ethereum/pot"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/ethereum/go-ethereum/swarm/pss"
	whisper "github.com/ethereum/go-ethereum/whisper/whisperv5"
)

const (
	inboxCapacity  = 3000
	outboxCapacity = 100
	addrLen        = common.HashLength
)

// After a successful connection with Client.Start, BaseAddr contains the swarm overlay address of the pss node
type Client struct {
	BaseAddr []byte

	// peers
	peerPool map[whisper.TopicType]map[pot.Address]*pssRPCRW
	protos   map[whisper.TopicType]*p2p.Protocol

	// rpc connections
	rpc *rpc.Client
	sub *rpc.ClientSubscription

	// channels
	topicsC chan []byte
	msgC    chan pss.APIMsg
	quitC   chan struct{}

	lock sync.Mutex
}

// implements p2p.MsgReadWriter
type pssRPCRW struct {
	*Client
	topic *whisper.TopicType
	msgC  chan []byte
	addr  pot.Address
}

func (self *Client) newpssRPCRW(addr pot.Address, topic *whisper.TopicType) *pssRPCRW {
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

	return rw.Client.rpc.Call(nil, "pss_send", rw.topic, pss.APIMsg{
		Addr: rw.addr.Bytes(),
		Msg:  pmsg,
	})

}

func NewClient(rpcurl string) (*Client, error) {
	rpcclient, err := rpc.Dial(rpcurl)
	if err != nil {
		return nil, err
	}

	client, err := NewClientWithRPC(rpcclient)
	if err != nil {
		return nil, err
	}
	return client, nil
}

// Constructor for test implementations
// The 'rpcclient' parameter allows passing a in-memory rpc client to act as the remote websocket RPC.
func NewClientWithRPC(rpcclient *rpc.Client) (*Client, error) {
	client := newClient()
	client.rpc = rpcclient
	err := client.rpc.Call(&client.BaseAddr, "pss_baseAddr")
	if err != nil {
		return nil, fmt.Errorf("cannot get pss node baseaddress: %v", err)
	}
	return client, nil
}

func newClient() (client *Client) {
	client = &Client{
		msgC:     make(chan pss.APIMsg),
		quitC:    make(chan struct{}),
		peerPool: make(map[whisper.TopicType]map[pot.Address]*pssRPCRW),
		protos:   make(map[whisper.TopicType]*p2p.Protocol),
	}
	return
}

// Mounts a new devp2p protcool on the pss connection
// the protocol is aliased as a "pss topic"
// uses normal devp2p Send and incoming message handler routines from the p2p/protocols package
//
// when an incoming message is received from a peer that is not yet known to the client, this peer object is instantiated, and the protocol is run on it.
func (self *Client) RunProtocol(ctx context.Context, proto *p2p.Protocol) error {
	topic := whisper.BytesToTopic([]byte(fmt.Sprintf("%s:%d", proto.Name, proto.Version)))
	msgC := make(chan pss.APIMsg)
	self.peerPool[topic] = make(map[pot.Address]*pssRPCRW)
	sub, err := self.rpc.Subscribe(ctx, "pss", msgC, "receive", topic)
	if err != nil {
		return fmt.Errorf("pss event subscription failed: %v", err)
	}
	self.sub = sub

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
				return
			}
		}
	}()

	self.protos[topic] = proto
	return nil
}

// Always call this to ensure that we exit cleanly
func (self *Client) Stop() error {
	return nil
}

// Preemptively add a remote pss peer
func (self *Client) AddPssPeer(addr pot.Address, spec *protocols.Spec) {
	topic := whisper.BytesToTopic([]byte(fmt.Sprintf("%s:%d", spec.Name, spec.Version)))
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
	topic := whisper.BytesToTopic([]byte(fmt.Sprintf("%s:%d", spec.Name, spec.Version)))
	delete(self.peerPool[topic], addr)
}
