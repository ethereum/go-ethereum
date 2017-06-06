package pss

import (
	"fmt"
	"io/ioutil"
	"os"
	"time"

	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/p2p/protocols"
	"github.com/ethereum/go-ethereum/swarm/network"
	"github.com/ethereum/go-ethereum/swarm/storage"
)

type PssPingMsg struct {
	Created time.Time
}

type PssPing struct {
	QuitC chan struct{}
}

func (self *PssPing) PssPingHandler(msg interface{}) error {
	log.Warn("got ping", "msg", msg)
	self.QuitC <- struct{}{}
	return nil
}

var PssPingProtocol = &protocols.Spec{
	Name:       "psstest",
	Version:    1,
	MaxMsgSize: 10 * 1024 * 1024,
	Messages: []interface{}{
		PssPingMsg{},
	},
}

var PssPingTopic = NewTopic(PssPingProtocol.Name, int(PssPingProtocol.Version))

func NewPssPingMsg(to []byte, spec *protocols.Spec, topic PssTopic, senderaddr []byte) PssMsg {
	data := PssPingMsg{
		Created: time.Now(),
	}
	code, found := spec.GetCode(&data)
	if !found {
		return PssMsg{}
	}

	rlpbundle, err := NewProtocolMsg(code, data)
	if err != nil {
		return PssMsg{}
	}

	pssmsg := PssMsg{
		To:      to,
		Payload: NewPssEnvelope(senderaddr, topic, rlpbundle),
	}

	return pssmsg
}

func NewPssPingProtocol(handler func(interface{}) error) *p2p.Protocol {
	return &p2p.Protocol{
		Name:    PssPingProtocol.Name,
		Version: PssPingProtocol.Version,
		Length:  uint64(PssPingProtocol.MaxMsgSize),
		Run: func(p *p2p.Peer, rw p2p.MsgReadWriter) error {
			pp := protocols.NewPeer(p, rw, PssPingProtocol)
			log.Trace(fmt.Sprintf("running pss vprotocol on peer %v", p))
			err := pp.Run(handler)
			return err
		},
	}
}

func NewTestPss(addr []byte) *Pss {
	if addr == nil {
		addr = network.RandomAddr().OAddr
	}

	// set up storage
	cachedir, err := ioutil.TempDir("", "pss-cache")
	if err != nil {
		log.Error("create pss cache tmpdir failed", "error", err)
		os.Exit(1)
	}
	dpa, err := storage.NewLocalDPA(cachedir)
	if err != nil {
		log.Error("local dpa creation failed", "error", err)
		os.Exit(1)
	}

	// set up routing
	kp := network.NewKadParams()
	kp.MinProxBinSize = 3

	// create pss
	pp := NewPssParams(true)

	overlay := network.NewKademlia(addr, kp)
	ps := NewPss(overlay, dpa, pp)

	return ps
}
