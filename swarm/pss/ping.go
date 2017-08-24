package pss

import (
	"crypto/ecdsa"
	"fmt"
	"io/ioutil"
	"os"
	"time"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/p2p/discover"
	"github.com/ethereum/go-ethereum/p2p/protocols"
	"github.com/ethereum/go-ethereum/swarm/network"
	"github.com/ethereum/go-ethereum/swarm/storage"
	whisper "github.com/ethereum/go-ethereum/whisper/whisperv5"
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

// Sample protocol used for tests
var PingProtocol = &protocols.Spec{
	Name:       "psstest",
	Version:    1,
	MaxMsgSize: 10 * 1024 * 1024,
	Messages: []interface{}{
		PingMsg{},
	},
}

var PingTopic = whisper.BytesToTopic([]byte(fmt.Sprintf("%s:%d", PingProtocol.Name, PingProtocol.Version)))

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

func NewTestPss(privkey *ecdsa.PrivateKey, ppextra *PssParams) *Pss {

	var nid discover.NodeID
	copy(nid[:], crypto.FromECDSAPub(&privkey.PublicKey))
	addr := network.NewAddrFromNodeID(nid)

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
	pp := NewPssParams(privkey)
	if ppextra != nil {
		pp.SymKeyCacheCapacity = ppextra.SymKeyCacheCapacity
	}

	overlay := network.NewKademlia(addr.Over(), kp)
	ps := NewPss(overlay, dpa, pp)

	return ps
}
