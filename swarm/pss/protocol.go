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

// +build !nopssprotocol

package pss

import (
	"bytes"
	"fmt"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/p2p/protocols"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/ethereum/go-ethereum/swarm/log"
)

const (
	IsActiveProtocol = true
)

// Convenience wrapper for devp2p protocol messages for transport over pss
type ProtocolMsg struct {
	Code       uint64
	Size       uint32
	Payload    []byte
	ReceivedAt time.Time
}

// Creates a ProtocolMsg
func NewProtocolMsg(code uint64, msg interface{}) ([]byte, error) {

	rlpdata, err := rlp.EncodeToBytes(msg)
	if err != nil {
		return nil, err
	}

	// TODO verify that nested structs cannot be used in rlp
	smsg := &ProtocolMsg{
		Code:    code,
		Size:    uint32(len(rlpdata)),
		Payload: rlpdata,
	}

	return rlp.EncodeToBytes(smsg)
}

// Protocol options to be passed to a new Protocol instance
//
// The parameters specify which encryption schemes to allow
type ProtocolParams struct {
	Asymmetric bool
	Symmetric  bool
}

// PssReadWriter bridges pss send/receive with devp2p protocol send/receive
//
// Implements p2p.MsgReadWriter
type PssReadWriter struct {
	*Pss
	LastActive time.Time
	rw         chan p2p.Msg
	spec       *protocols.Spec
	topic      *Topic
	sendFunc   func(string, Topic, []byte) error
	key        string
	closed     bool
}

// Implements p2p.MsgReader
func (prw *PssReadWriter) ReadMsg() (p2p.Msg, error) {
	msg := <-prw.rw
	log.Trace(fmt.Sprintf("pssrw readmsg: %v", msg))
	return msg, nil
}

// Implements p2p.MsgWriter
func (prw *PssReadWriter) WriteMsg(msg p2p.Msg) error {
	log.Trace("pssrw writemsg", "msg", msg)
	if prw.closed {
		return fmt.Errorf("connection closed")
	}
	rlpdata := make([]byte, msg.Size)
	msg.Payload.Read(rlpdata)
	pmsg, err := rlp.EncodeToBytes(ProtocolMsg{
		Code:    msg.Code,
		Size:    msg.Size,
		Payload: rlpdata,
	})
	if err != nil {
		return err
	}
	return prw.sendFunc(prw.key, *prw.topic, pmsg)
}

// Injects a p2p.Msg into the MsgReadWriter, so that it appears on the associated p2p.MsgReader
func (prw *PssReadWriter) injectMsg(msg p2p.Msg) error {
	log.Trace(fmt.Sprintf("pssrw injectmsg: %v", msg))
	prw.rw <- msg
	return nil
}

// Convenience object for emulation devp2p over pss
type Protocol struct {
	*Pss
	proto        *p2p.Protocol
	topic        *Topic
	spec         *protocols.Spec
	pubKeyRWPool map[string]p2p.MsgReadWriter
	symKeyRWPool map[string]p2p.MsgReadWriter
	Asymmetric   bool
	Symmetric    bool
	RWPoolMu     sync.Mutex
}

// Activates devp2p emulation over a specific pss topic
//
// One or both encryption schemes must be specified. If
// only one is specified, the protocol will not be valid
// for the other, and will make the message handler
// return errors
func RegisterProtocol(ps *Pss, topic *Topic, spec *protocols.Spec, targetprotocol *p2p.Protocol, options *ProtocolParams) (*Protocol, error) {
	if !options.Asymmetric && !options.Symmetric {
		return nil, fmt.Errorf("specify at least one of asymmetric or symmetric messaging mode")
	}
	pp := &Protocol{
		Pss:          ps,
		proto:        targetprotocol,
		topic:        topic,
		spec:         spec,
		pubKeyRWPool: make(map[string]p2p.MsgReadWriter),
		symKeyRWPool: make(map[string]p2p.MsgReadWriter),
		Asymmetric:   options.Asymmetric,
		Symmetric:    options.Symmetric,
	}
	return pp, nil
}

// Generic handler for incoming messages over devp2p emulation
//
// To be passed to pss.Register()
//
// Will run the protocol on a new incoming peer, provided that
// the encryption key of the message has a match in the internal
// pss keypool
//
// Fails if protocol is not valid for the message encryption scheme,
// if adding a new peer fails, or if the message is not a serialized
// p2p.Msg (which it always will be if it is sent from this object).
func (p *Protocol) Handle(msg []byte, peer *p2p.Peer, asymmetric bool, keyid string) error {
	var vrw *PssReadWriter
	if p.Asymmetric != asymmetric && p.Symmetric == !asymmetric {
		return fmt.Errorf("invalid protocol encryption")
	} else if (!p.isActiveSymKey(keyid, *p.topic) && !asymmetric) ||
		(!p.isActiveAsymKey(keyid, *p.topic) && asymmetric) {

		rw, err := p.AddPeer(peer, *p.topic, asymmetric, keyid)
		if err != nil {
			return err
		} else if rw == nil {
			return fmt.Errorf("handle called on nil MsgReadWriter for new key " + keyid)
		}
		vrw = rw.(*PssReadWriter)
	}

	pmsg, err := ToP2pMsg(msg)
	if err != nil {
		return fmt.Errorf("could not decode pssmsg")
	}
	if asymmetric {
		if p.pubKeyRWPool[keyid] == nil {
			return fmt.Errorf("handle called on nil MsgReadWriter for key " + keyid)
		}
		vrw = p.pubKeyRWPool[keyid].(*PssReadWriter)
	} else {
		if p.symKeyRWPool[keyid] == nil {
			return fmt.Errorf("handle called on nil MsgReadWriter for key " + keyid)
		}
		vrw = p.symKeyRWPool[keyid].(*PssReadWriter)
	}
	vrw.injectMsg(pmsg)
	return nil
}

// check if (peer) symmetric key is currently registered with this topic
func (p *Protocol) isActiveSymKey(key string, topic Topic) bool {
	return p.symKeyRWPool[key] != nil
}

// check if (peer) asymmetric key is currently registered with this topic
func (p *Protocol) isActiveAsymKey(key string, topic Topic) bool {
	return p.pubKeyRWPool[key] != nil
}

// Creates a serialized (non-buffered) version of a p2p.Msg, used in the specialized internal p2p.MsgReadwriter implementations
func ToP2pMsg(msg []byte) (p2p.Msg, error) {
	payload := &ProtocolMsg{}
	if err := rlp.DecodeBytes(msg, payload); err != nil {
		return p2p.Msg{}, fmt.Errorf("pss protocol handler unable to decode payload as p2p message: %v", err)
	}

	return p2p.Msg{
		Code:       payload.Code,
		Size:       uint32(len(payload.Payload)),
		ReceivedAt: time.Now(),
		Payload:    bytes.NewBuffer(payload.Payload),
	}, nil
}

// Runs an emulated pss Protocol on the specified peer,
// linked to a specific topic
// `key` and `asymmetric` specifies what encryption key
// to link the peer to.
// The key must exist in the pss store prior to adding the peer.
func (p *Protocol) AddPeer(peer *p2p.Peer, topic Topic, asymmetric bool, key string) (p2p.MsgReadWriter, error) {
	rw := &PssReadWriter{
		Pss:   p.Pss,
		rw:    make(chan p2p.Msg),
		spec:  p.spec,
		topic: p.topic,
		key:   key,
	}
	if asymmetric {
		rw.sendFunc = p.Pss.SendAsym
	} else {
		rw.sendFunc = p.Pss.SendSym
	}
	if asymmetric {
		if !p.Pss.isPubKeyStored(key) {
			return nil, fmt.Errorf("asym key does not exist: %s", key)
		}
		p.RWPoolMu.Lock()
		p.pubKeyRWPool[key] = rw
		p.RWPoolMu.Unlock()
	} else {
		if !p.Pss.isSymKeyStored(key) {
			return nil, fmt.Errorf("symkey does not exist: %s", key)
		}
		p.RWPoolMu.Lock()
		p.symKeyRWPool[key] = rw
		p.RWPoolMu.Unlock()
	}
	go func() {
		err := p.proto.Run(peer, rw)
		log.Warn(fmt.Sprintf("pss vprotocol quit on %v topic %v: %v", peer, topic, err))
	}()
	return rw, nil
}

func (p *Protocol) RemovePeer(asymmetric bool, key string) {
	log.Debug("closing pss peer", "asym", asymmetric, "key", key)
	p.RWPoolMu.Lock()
	defer p.RWPoolMu.Unlock()
	if asymmetric {
		rw := p.pubKeyRWPool[key].(*PssReadWriter)
		rw.closed = true
		delete(p.pubKeyRWPool, key)
	} else {
		rw := p.symKeyRWPool[key].(*PssReadWriter)
		rw.closed = true
		delete(p.symKeyRWPool, key)
	}
}

// Uniform translation of protocol specifiers to topic
func ProtocolTopic(spec *protocols.Spec) Topic {
	return BytesToTopic([]byte(fmt.Sprintf("%s:%d", spec.Name, spec.Version)))
}
