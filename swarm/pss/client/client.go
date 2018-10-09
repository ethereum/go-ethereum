// Copyright 2018 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

// +build !noclient,!noprotocol

package client

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/p2p/enode"
	"github.com/ethereum/go-ethereum/p2p/protocols"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/ethereum/go-ethereum/swarm/log"
	"github.com/ethereum/go-ethereum/swarm/pss"
)

const (
	handshakeRetryTimeout = 1000
	handshakeRetryCount   = 3
)

// The pss client provides devp2p emulation over pss RPC API,
// giving access to pss methods from a different process
type Client struct {
	BaseAddrHex string

	// peers
	peerPool map[pss.Topic]map[string]*pssRPCRW
	protos   map[pss.Topic]*p2p.Protocol

	// rpc connections
	rpc  *rpc.Client
	subs []*rpc.ClientSubscription

	// channels
	topicsC chan []byte
	quitC   chan struct{}

	poolMu sync.Mutex
}

// implements p2p.MsgReadWriter
type pssRPCRW struct {
	*Client
	topic    string
	msgC     chan []byte
	addr     pss.PssAddress
	pubKeyId string
	lastSeen time.Time
	closed   bool
}

func (c *Client) newpssRPCRW(pubkeyid string, addr pss.PssAddress, topicobj pss.Topic) (*pssRPCRW, error) {
	topic := topicobj.String()
	err := c.rpc.Call(nil, "pss_setPeerPublicKey", pubkeyid, topic, hexutil.Encode(addr[:]))
	if err != nil {
		return nil, fmt.Errorf("setpeer %s %s: %v", topic, pubkeyid, err)
	}
	return &pssRPCRW{
		Client:   c,
		topic:    topic,
		msgC:     make(chan []byte),
		addr:     addr,
		pubKeyId: pubkeyid,
	}, nil
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

// If only one message slot left
// then new is requested through handshake
// if buffer is empty, handshake request blocks until return
// after which pointer is changed to first new key in buffer
// will fail if:
// - any api calls fail
// - handshake retries are exhausted without reply,
// - send fails
func (rw *pssRPCRW) WriteMsg(msg p2p.Msg) error {
	log.Trace("got writemsg pssclient", "msg", msg)
	if rw.closed {
		return fmt.Errorf("connection closed")
	}
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

	// Get the keys we have
	var symkeyids []string
	err = rw.Client.rpc.Call(&symkeyids, "pss_getHandshakeKeys", rw.pubKeyId, rw.topic, false, true)
	if err != nil {
		return err
	}

	// Check the capacity of the first key
	var symkeycap uint16
	if len(symkeyids) > 0 {
		err = rw.Client.rpc.Call(&symkeycap, "pss_getHandshakeKeyCapacity", symkeyids[0])
		if err != nil {
			return err
		}
	}

	err = rw.Client.rpc.Call(nil, "pss_sendSym", symkeyids[0], rw.topic, hexutil.Encode(pmsg))
	if err != nil {
		return err
	}

	// If this is the last message it is valid for, initiate new handshake
	if symkeycap == 1 {
		var retries int
		var sync bool
		// if it's the only remaining key, make sure we don't continue until we have new ones for further writes
		if len(symkeyids) == 1 {
			sync = true
		}
		// initiate handshake
		_, err := rw.handshake(retries, sync, false)
		if err != nil {
			log.Warn("failing", "err", err)
			return err
		}
	}
	return nil
}

// retry and synchronicity wrapper for handshake api call
// returns first new symkeyid upon successful execution
func (rw *pssRPCRW) handshake(retries int, sync bool, flush bool) (string, error) {

	var symkeyids []string
	var i int
	// request new keys
	// if the key buffer was depleted, make this as a blocking call and try several times before giving up
	for i = 0; i < 1+retries; i++ {
		log.Debug("handshake attempt pssrpcrw", "pubkeyid", rw.pubKeyId, "topic", rw.topic, "sync", sync)
		err := rw.Client.rpc.Call(&symkeyids, "pss_handshake", rw.pubKeyId, rw.topic, sync, flush)
		if err == nil {
			var keyid string
			if sync {
				keyid = symkeyids[0]
			}
			return keyid, nil
		}
		if i-1+retries > 1 {
			time.Sleep(time.Millisecond * handshakeRetryTimeout)
		}
	}

	return "", fmt.Errorf("handshake failed after %d attempts", i)
}

// Custom constructor
//
// Provides direct access to the rpc object
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

// Main constructor
//
// The 'rpcclient' parameter allows passing a in-memory rpc client to act as the remote websocket RPC.
func NewClientWithRPC(rpcclient *rpc.Client) (*Client, error) {
	client := newClient()
	client.rpc = rpcclient
	err := client.rpc.Call(&client.BaseAddrHex, "pss_baseAddr")
	if err != nil {
		return nil, fmt.Errorf("cannot get pss node baseaddress: %v", err)
	}
	return client, nil
}

func newClient() (client *Client) {
	client = &Client{
		quitC:    make(chan struct{}),
		peerPool: make(map[pss.Topic]map[string]*pssRPCRW),
		protos:   make(map[pss.Topic]*p2p.Protocol),
	}
	return
}

// Mounts a new devp2p protcool on the pss connection
//
// the protocol is aliased as a "pss topic"
// uses normal devp2p send and incoming message handler routines from the p2p/protocols package
//
// when an incoming message is received from a peer that is not yet known to the client,
// this peer object is instantiated, and the protocol is run on it.
func (c *Client) RunProtocol(ctx context.Context, proto *p2p.Protocol) error {
	topicobj := pss.BytesToTopic([]byte(fmt.Sprintf("%s:%d", proto.Name, proto.Version)))
	topichex := topicobj.String()
	msgC := make(chan pss.APIMsg)
	c.peerPool[topicobj] = make(map[string]*pssRPCRW)
	sub, err := c.rpc.Subscribe(ctx, "pss", msgC, "receive", topichex)
	if err != nil {
		return fmt.Errorf("pss event subscription failed: %v", err)
	}
	c.subs = append(c.subs, sub)
	err = c.rpc.Call(nil, "pss_addHandshake", topichex)
	if err != nil {
		return fmt.Errorf("pss handshake activation failed: %v", err)
	}

	// dispatch incoming messages
	go func() {
		for {
			select {
			case msg := <-msgC:
				// we only allow sym msgs here
				if msg.Asymmetric {
					continue
				}
				// we get passed the symkeyid
				// need the symkey itself to resolve to peer's pubkey
				var pubkeyid string
				err = c.rpc.Call(&pubkeyid, "pss_getHandshakePublicKey", msg.Key)
				if err != nil || pubkeyid == "" {
					log.Trace("proto err or no pubkey", "err", err, "symkeyid", msg.Key)
					continue
				}
				// if we don't have the peer on this protocol already, create it
				// this is more or less the same as AddPssPeer, less the handshake initiation
				if c.peerPool[topicobj][pubkeyid] == nil {
					var addrhex string
					err := c.rpc.Call(&addrhex, "pss_getAddress", topichex, false, msg.Key)
					if err != nil {
						log.Trace(err.Error())
						continue
					}
					addrbytes, err := hexutil.Decode(addrhex)
					if err != nil {
						log.Trace(err.Error())
						break
					}
					addr := pss.PssAddress(addrbytes)
					rw, err := c.newpssRPCRW(pubkeyid, addr, topicobj)
					if err != nil {
						break
					}
					c.peerPool[topicobj][pubkeyid] = rw
					p := p2p.NewPeer(enode.ID{}, fmt.Sprintf("%v", addr), []p2p.Cap{})
					go proto.Run(p, c.peerPool[topicobj][pubkeyid])
				}
				go func() {
					c.peerPool[topicobj][pubkeyid].msgC <- msg.Msg
				}()
			case <-c.quitC:
				return
			}
		}
	}()

	c.protos[topicobj] = proto
	return nil
}

// Always call this to ensure that we exit cleanly
func (c *Client) Close() error {
	for _, s := range c.subs {
		s.Unsubscribe()
	}
	return nil
}

// Add a pss peer (public key) and run the protocol on it
//
// client.RunProtocol with matching topic must have been
// run prior to adding the peer, or this method will
// return an error.
//
// The key must exist in the key store of the pss node
// before the peer is added. The method will return an error
// if it is not.
func (c *Client) AddPssPeer(pubkeyid string, addr []byte, spec *protocols.Spec) error {
	topic := pss.ProtocolTopic(spec)
	if c.peerPool[topic] == nil {
		return errors.New("addpeer on unset topic")
	}
	if c.peerPool[topic][pubkeyid] == nil {
		rw, err := c.newpssRPCRW(pubkeyid, addr, topic)
		if err != nil {
			return err
		}
		_, err = rw.handshake(handshakeRetryCount, true, true)
		if err != nil {
			return err
		}
		c.poolMu.Lock()
		c.peerPool[topic][pubkeyid] = rw
		c.poolMu.Unlock()
		p := p2p.NewPeer(enode.ID{}, fmt.Sprintf("%v", addr), []p2p.Cap{})
		go c.protos[topic].Run(p, c.peerPool[topic][pubkeyid])
	}
	return nil
}

// Remove a pss peer
//
// TODO: underlying cleanup
func (c *Client) RemovePssPeer(pubkeyid string, spec *protocols.Spec) {
	log.Debug("closing pss client peer", "pubkey", pubkeyid, "protoname", spec.Name, "protoversion", spec.Version)
	c.poolMu.Lock()
	defer c.poolMu.Unlock()
	topic := pss.ProtocolTopic(spec)
	c.peerPool[topic][pubkeyid].closed = true
	delete(c.peerPool[topic], pubkeyid)
}
