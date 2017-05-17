package pss

import (
	"io/ioutil"
	"os"
	"time"

	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/p2p/protocols"
	"github.com/ethereum/go-ethereum/swarm/network"
	"github.com/ethereum/go-ethereum/swarm/storage"
)	

type pssPingMsg struct {
	Created time.Time
}

type pssPing struct {
	quitC chan struct{}
}

func (self *pssPing) pssPingHandler(msg interface{}) error {
	log.Warn("got ping", "msg", msg)
	self.quitC <- struct{}{}
	return nil
}

var pssPingProtocol = &protocols.Spec{
	Name:       "psstest",
	Version:    1,
	MaxMsgSize: 10 * 1024 * 1024,
	Messages: []interface{}{
		pssPingMsg{},
	},
}

var pssPingTopic = NewTopic(pssPingProtocol.Name, int(pssPingProtocol.Version))

func newTestPss(addr []byte) *Pss {	
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
	pp := NewPssParams()

	overlay := network.NewKademlia(addr, kp)
	ps := NewPss(overlay, dpa, pp)

	return ps
}

func newPssPingMsg(ps *Pss, spec *protocols.Spec, topic PssTopic, senderaddr []byte) PssMsg {
	data := pssPingMsg{
		Created: time.Now(),
	}
	code, found := spec.GetCode(&data)
	if !found {
		return PssMsg{}
	}

	rlpbundle, err := newProtocolMsg(code, data)
	if err != nil {
		return PssMsg{}
	}

	pssmsg := PssMsg{
		To: ps.Overlay.BaseAddr(),
		Payload: NewPssEnvelope(senderaddr, topic, rlpbundle),
	}

	return pssmsg
}
