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

type PingMsg struct {
	Created time.Time
}

type Ping struct {
	C chan struct{}
}

func (self *Ping) PingHandler(msg interface{}) error {
	log.Warn("got ping", "msg", msg)
	self.C <- struct{}{}
	return nil
}

var PingProtocol = &protocols.Spec{
	Name:       "psstest",
	Version:    1,
	MaxMsgSize: 10 * 1024 * 1024,
	Messages: []interface{}{
		PingMsg{},
	},
}

var PingTopic = NewTopic(PingProtocol.Name, int(PingProtocol.Version))

func NewPingMsg(to []byte, spec *protocols.Spec, topic Topic, senderaddr []byte) PssMsg {
	data := PingMsg{
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
		Payload: NewEnvelope(senderaddr, topic, rlpbundle),
	}

	return pssmsg
}

func NewPingProtocol(handler func(interface{}) error) *p2p.Protocol {
	return &p2p.Protocol{
		Name:    PingProtocol.Name,
		Version: PingProtocol.Version,
		Length:  uint64(PingProtocol.MaxMsgSize),
		Run: func(p *p2p.Peer, rw p2p.MsgReadWriter) error {
			pp := protocols.NewPeer(p, rw, PingProtocol)
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
