package network

import (
	"bytes"
	"fmt"

	"github.com/ethereum/go-ethereum/log"
)

type Pss struct {
	Overlay
	LocalAddr []byte
	C         chan []byte
}

func NewPss(k Overlay, addr []byte) *Pss {
	return &Pss{
		Overlay:   k,
		LocalAddr: addr,
		C:         make(chan []byte),
	}
}

type PssMsg struct {
	To   []byte
	Data []byte
}

func (pm *PssMsg) String() string {
	return fmt.Sprintf("PssMsg: Recipient: %v", pm.To)
}

func (ps *Pss) HandlePssMsg(msg interface{}) error {
	pssmsg := msg.(*PssMsg)
	to := pssmsg.To
	if bytes.Equal(to, ps.LocalAddr) {
		log.Trace(fmt.Sprintf("Pss to us, yay! %v", to))
		ps.C <- pssmsg.Data
		return nil
	}

	ps.EachLivePeer(to, 255, func(p Peer, po int) bool {
		err := p.Send(pssmsg)
		if err != nil {
			return true
		}
		return false
	})

	return nil
}
